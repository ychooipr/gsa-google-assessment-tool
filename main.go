package main

import (
	"archive/zip"
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/MetaPhase-Consulting/gsa-google-assessment-tool/GoogleAPI"
	"golang.org/x/oauth2"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/drive/v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed client_secret.json
var clientSecretData []byte

var CTX = context.Background()
var Scopes = []string{
	drive.DriveFileScope,
	drive.DriveReadonlyScope,
	cloudresourcemanager.CloudPlatformScope,
	strings.Join(GoogleAPI.AdminScopes, " "),
}
var VERSION = "2023.6.14_ScriptsAudit"
var DelegationKeyPath = ""
var CustomerID = "my_customer"
var ReportsPath = "output_" + time.Now().Format(time.RFC3339)
var DriveReportsPath = "root"

// GoogleClientAuthenticationFlowHandler This is a function that returns the Google API client with the correct scopes
func GoogleClientAuthenticationFlowHandler(clientSecretData []byte, scopes []string, ctx context.Context) *http.Client {
	// PrototypeOauth2Token This is a prototype token that is used to generate a new token
	var PrototypeOauth2Token = oauth2.Token{
		TokenType: "Bearer",
		Expiry:    time.Time{}.Add(time.Hour * 24 * 365)}

	// Get token data from flags
	flag.StringVar(&PrototypeOauth2Token.AccessToken, "access_token", "", "string: Access token")
	flag.StringVar(&PrototypeOauth2Token.RefreshToken, "refresh_token", "", "string: Refresh token")
	flag.StringVar(&DelegationKeyPath, "key_path", "svcKey.json", "string: Delegation Key Path")
	flag.StringVar(&CustomerID, "customer_id", "my_customer", "string: Customer ID")
	flag.Parse()
	// Parse flags
	flag.Parse()

	// Marshal token data
	tokenData, err := json.Marshal(PrototypeOauth2Token)
	if err != nil {
		log.Printf("Unable to marshal token data: %v", err)
		panic(err)
	}

	// Get Return the Google API client
	return GoogleAPI.GetOAuth2GoogleClient(clientSecretData, tokenData, scopes, ctx)
}

// generateNewToken This function generates a new token
func generateNewToken() {
	token, _ := GoogleAPI.GenerateOAuth2Token(clientSecretData, Scopes, CTX)
	log.Printf("Copy the following command and use this to execute the application:\n\n"+
		"********************************* Copy Text Below *******************************************\n%s "+
		"-access_token %s "+
		"-refresh_token %s\n"+
		"********************************* Copy Text Above *******************************************\n\n"+
		"Please restart the application after copying the above command.",
		os.Args[0], token.AccessToken, token.RefreshToken)
	os.Exit(0)
}

// GoogleCloudProject This is a struct that contains all the information for a Google Cloud Project
type GoogleCloudProject struct {
	Id              string                         `json:"id"`
	Number          int                            `json:"number"`
	Name            string                         `json:"name"`
	ServiceAccounts []*GoogleAPI.GCPServiceAccount `json:"service_accounts"`
	Notes           string                         `json:"notes"`
}

// GetAllGoogleCloudProjects This function gets all Google Cloud Projects and service accounts for each project
func GetAllGoogleCloudProjects(crmAPI *GoogleAPI.CloudResourceManagerAPI, iAmAPI *GoogleAPI.IamAPI) ([]*GoogleCloudProject, error) {

	// Get all projects
	allProjects, err := crmAPI.GetAllProjects()
	if err != nil {
		log.Fatalf("Unable to get all projects: %v", err)
		return nil, err
	}

	// Create a gcpProjectList to store all projects
	var gcpProjectList []*GoogleCloudProject

	log.Println("Getting all service accounts for all projects")

	totalJobs := len(allProjects)
	maxExecutes := 100
	totalBatches := (totalJobs / maxExecutes) + 1
	batchCounter := 1
	for {
		log.Printf("<----- Get ServiceAccounts Batch [%directory] of [%directory] ----->\n", batchCounter, totalBatches)
		if len(allProjects) < maxExecutes {
			maxExecutes = len(allProjects)
		}

		wg := &sync.WaitGroup{}
		wg.Add(maxExecutes)

		for _, job := range allProjects[:maxExecutes] {
			go func(project *cloudresourcemanager.Project) {
				defer wg.Done()
				newGCP := &GoogleCloudProject{Id: project.ProjectId,
					Number: int(project.ProjectNumber),
					Name:   project.Name}
				serviceAccounts, err := iAmAPI.GetProjectServiceAccounts(project.ProjectId)
				if err != nil { // If there is an error, set the notes to the error message
					newGCP.Notes = fmt.Sprintf(err.Error())
				} else { // If there is no error, set the service accounts
					newGCP.ServiceAccounts = serviceAccounts
				}
				// Append the newGCP to the gcpProjectList
				gcpProjectList = append(gcpProjectList, newGCP)
			}(job)
		}
		wg.Wait()

		allProjects = allProjects[maxExecutes:]
		if len(allProjects) == 0 {
			break
		}
		batchCounter++
	}
	// Return the gcpProjectList
	return gcpProjectList, nil
}

// PrintProgressBar function that prints a progress bar to the console
func PrintProgressBar(current, total int, title string) {
	barLength := 30
	percent := float64(current) / float64(total)
	bar := strings.Repeat("=", int(percent*float64(barLength)))
	fmt.Printf("\r[%s] %.0f%% (%directory/%directory) %s", bar+strings.Repeat(" ", barLength-int(percent*float64(barLength))), percent*100, current, total, title)
	if current == total {
		fmt.Println()
	}
}

// ZipDirectory This function zips a directory
func ZipDirectory(sourceDir, destinationFile string) *os.File {

	// Create a new zip file
	zipFile, err := os.Create(destinationFile)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	// Close the zip file
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk the directory tree recursively and add files to the archive
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}

		// Skip directories and hidden files
		if info.IsDir() || filepath.Base(path)[0] == '.' {
			log.Println("Skipping file: " + path)
			return nil
		}

		// Create a new file in the archive
		zipFile, err := zipWriter.Create(path)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}

		// Open the source file
		file, err := os.Open(path)
		log.Println("Zipping: " + path)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		defer file.Close()

		// Copy the file contents to the archive
		_, err = io.Copy(zipFile, file)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}

		return nil
	})

	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	fmt.Println("Archive created successfully")
	return zipFile
}

// Start: main  ########################################################################################################
func main() {
	mainTimer := time.Now()
	//Get Arguments --------------------------------------------------------------------------------------------
	// Check if we need to generate a token
	// This uses an argument rather than a flag as it is only used once
	if len(os.Args) < 2 {
		generateNewToken()
	}

	//Create logs --------------------------------------------------------------------------------------------
	// Create log directory if it doesn't exist
	if _, err := os.Stat(ReportsPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(ReportsPath, os.ModePerm)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
	}
	// Create the log file name
	logFileName := fmt.Sprintf("%s", filepath.Base(strings.ReplaceAll(os.Args[0], ".exe", ""))+".log")

	// Start Create log file and set it as the output ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	logFile, err := os.OpenFile(ReportsPath+string(os.PathSeparator)+logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Println(err.Error())
		return
	}
	// Defer closing the log file
	defer func() {
		err := logFile.Close()
		if err != nil {
			log.Printf(err.Error())
			panic(err)
		}
	}()
	// Set the output to the console and the log file
	mw := io.MultiWriter(os.Stdout, logFile)
	// Set the output
	log.SetOutput(mw)
	// End: Create log file and set it as the output +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

	log.Printf("Version: %s", VERSION)
	// Get the required APIs
	googleClient := GoogleClientAuthenticationFlowHandler(clientSecretData, Scopes, CTX)

	// Execution function

	// Get all the projects and service accounts
	log.Printf("Time to run %s: %s", os.Args[0], time.Since(mainTimer).String())
}

// End: main  ##########################################################################################################

// inventory function that gets all the projects and service accounts
func inventory(googleClient *http.Client) {
	directoryAPI := GoogleAPI.NewDirectoryAPI(googleClient, 2, CTX)
	crmAPI := GoogleAPI.NewCloudResourceManagerAPI(googleClient, 2, CTX)
	iamAPI := GoogleAPI.NewIamAPI(googleClient, 60, CTX)

	// Start a wait group to wait for all the goroutines to finish $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$
	wg := &sync.WaitGroup{}
	// Add 3 to the wait group
	wg.Add(3)

	// Start the goroutine to get all the projects ---------------------------------------------------------------------
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		timer := time.Now()

		csvFile, _ := os.Create(ReportsPath + string(os.PathSeparator) + "projects.csv")
		var records [][]string
		defer csvFile.Close()
		csvWriter := csv.NewWriter(csvFile)
		csvWriter.Write([]string{"project_id", "project_number", "project_name", "service_accounts"})
		csvWriter.Flush()

		// Get all the projects
		log.Printf("Getting all projects...")
		projectList, err := GetAllGoogleCloudProjects(crmAPI, iamAPI)
		if err != nil {
			log.Printf("Error getting all projects: %s", err.Error())
			return
		}
		log.Println("Time to get all projects: " + time.Since(timer).String())

		// Get all the service accounts for each project
		timer = time.Now()
		for i := range projectList {
			data, _ := json.Marshal(projectList[i].ServiceAccounts)
			records = append(records, []string{projectList[i].Id, fmt.Sprintf("%v", projectList[i].Number), projectList[i].Name, string(data)})
		}
		log.Printf("Time to write projectList to a csv: %s", time.Since(timer).String())
		timer = time.Now()
		log.Printf("Writing projects to csv...")
		err = csvWriter.WriteAll(records)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		csvWriter.Flush()
		log.Printf("Time to write projects to a csv: %s", time.Since(timer).String())
	}(wg)

	// Start the goroutine to get all the users ------------------------------------------------------------------------
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		timer := time.Now()
		log.Printf("Getting all users...")
		csvFile, _ := os.Create(ReportsPath + string(os.PathSeparator) + "users.csv")
		defer csvFile.Close()
		csvWriter := csv.NewWriter(csvFile)
		var records [][]string
		err := csvWriter.Write([]string{"user_id", "primary_email", "archived", "is_admin", "is_delegated_admin", "is_suspended", "last_login_time", "is_mailbox_setup", "notes"})
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		csvWriter.Flush()

		// Get all the users
		users, err := directoryAPI.GetUsersAndToken("")
		if err != nil {
			log.Printf("Error getting users: %s", err.Error())
			return
		}

		log.Printf("Time to get users: %s", time.Since(timer).String())
		timer = time.Now()
		log.Println("Writing users to csv...")
		for i := range users {
			user := users[i]
			records = append(records, []string{
				user.Id,
				user.PrimaryEmail,
				strconv.FormatBool(user.Archived),
				strconv.FormatBool(user.IsAdmin),
				strconv.FormatBool(user.IsDelegatedAdmin),
				strconv.FormatBool(user.Suspended),
				user.LastLoginTime,
				strconv.FormatBool(user.IsMailboxSetup),
				user.Tokens.(string)})
		}
		csvWriter.WriteAll(records)
		csvWriter.Flush()
		log.Printf("Time to write users to a csv: %s", time.Since(timer).String())
	}(wg)

	// Start the goroutine to get all the groups -----------------------------------------------------------------------
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		timer := time.Now()
		log.Printf("Getting all groups...")
		csvFile, _ := os.Create(ReportsPath + string(os.PathSeparator) + "groups.csv")
		defer csvFile.Close()
		csvWriter := csv.NewWriter(csvFile)
		var records [][]string
		csvWriter.Write([]string{"email", "name", "member_count", "admin_created"})
		csvWriter.Flush()
		groups, err := directoryAPI.QueryGroups("")
		if err != nil {
			log.Println("Error getting groups: " + err.Error())
			return
		}

		totalJobs := len(groups)
		maxExecutes := 1000
		totalBatches := (totalJobs / maxExecutes) + 1
		batchCounter := 1
		for {
			log.Printf("<----- Groups Batch [%directory] of [%directory] ----->\n", batchCounter, totalBatches)
			if len(groups) < maxExecutes {
				maxExecutes = len(groups)
			}

			wg := &sync.WaitGroup{}
			wg.Add(maxExecutes)

			for _, job := range groups[:maxExecutes] {
				go func(group *directory.Group) {
					defer wg.Done()
					//firestoreAPI.Client.Collection("groups").Doc(group.Email).Set(CTX, group)
					records = append(records, []string{
						group.Email,
						group.Name,
						strconv.Itoa(int(group.DirectMembersCount)),
						strconv.FormatBool(group.AdminCreated)})

				}(job)
			}
			wg.Wait()

			groups = groups[maxExecutes:]
			if len(groups) == 0 {
				break
			}
			batchCounter++
		}
		log.Printf("Time to get groups: %s", time.Since(timer).String())

		timer = time.Now()
		log.Println("Writing groups to csv...")
		csvWriter.WriteAll(records)
		csvWriter.Flush()
		log.Printf("Time to write groups to a csv: %s", time.Since(timer).String())
	}(wg)
	// Wait for all the goroutines to finish
	wg.Wait()
	// Wait for all the goroutines to finish $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$

	// Start: Upload the reports to Google
	uploadReport(ReportsPath, googleClient)
	// End: Upload the reports to Google ^^^^

}

// groupsAudit will get all the groups and their members
func groupsAudit(googleClient *http.Client) {
	timer := time.Now()

	// Create the directory API
	directoryAPI := GoogleAPI.NewDirectoryAPI(googleClient, 2, CTX)

	//Get all the groups of the domain
	log.Println("Getting all groups...")
	allGroups, err := directoryAPI.QueryGroups("")
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	log.Printf("Total of %d groups found...", len(allGroups))

	// Sort the groups by member count
	log.Println("Sorting groups by member count...")
	for i := range allGroups {
		for j := range allGroups {
			if allGroups[i].DirectMembersCount > allGroups[j].DirectMembersCount {
				temp := allGroups[i]
				allGroups[i] = allGroups[j]
				allGroups[j] = temp
			}
		}
	}
	log.Println("Sorted all groups")

	// Create the groups csv
	var csvRows [][]string

	// S
	totalJobs := len(allGroups)
	maxExecutes := 100
	totalBatches := (totalJobs / maxExecutes) + 1
	batchCounter := 1
	// Iterate over the groups and get the owners using goroutines
	for {
		batchTimer := time.Now()
		if len(allGroups) < maxExecutes {
			maxExecutes = len(allGroups)
		}

		wg := &sync.WaitGroup{}
		wg.Add(maxExecutes)

		for _, job := range allGroups[:maxExecutes] {
			go func(group *directory.Group) {
				defer wg.Done()
				// Get the owners of the group
				owners, err := directoryAPI.GetGroupMembers(group, "OWNER")
				if err != nil {
					log.Println("Attempted to get \"OWNERS\" from group:"+group.Email, err.Error())
					panic(err)
				}
				// Create a slice of the owner emails
				var ownersList []string
				// iterate over the owners and get the email
				for _, owner := range owners {
					// Add the email to the slice
					ownersList = append(ownersList, owner.Email)
				}

				// Get the managers of the group
				managers, err := directoryAPI.GetGroupMembers(group, "MANAGER")
				if err != nil {
					log.Println("Attempted to get \"MANAGERS\" from group:"+group.Email, err.Error())
					panic(err)
				}
				// Create a slice of the manager emails
				var managersList []string
				// iterate over the managers and get the email
				for _, manager := range managers {
					// Add the email to the slice
					managersList = append(managersList, manager.Email)
				}

				// Get the members of the group
				subscriptions, err := directoryAPI.GetSubscriptions(group.Email)
				if err != nil {
					log.Println("Attempted to get subscriptions from group:"+group.Email, err.Error())
					panic(err)
				}
				// Create a slice of the group emails
				var subscriptionEmails []string
				// iterate over the subscriptions and get the email
				for _, sub := range subscriptions {
					// Add the email to the slice
					subscriptionEmails = append(subscriptionEmails, sub.Email)
				}

				// Write the group to the csv
				csvRows = append(csvRows, []string{group.Email, // Group email
					fmt.Sprint(group.DirectMembersCount),  // Members count
					fmt.Sprint(len(ownersList)),           // Owners count
					strings.Join(ownersList, ","),         // Owners
					fmt.Sprint(len(managersList)),         // Managers count
					strings.Join(managersList, ","),       // Managers
					fmt.Sprint(len(subscriptionEmails)),   // Subscriptions count
					strings.Join(subscriptionEmails, ","), // Subscriptions
				})
			}(job)
		}
		wg.Wait()
		log.Printf("<----- Batch [%d] of [%d] completed in %s ----->\n", batchCounter, totalBatches, time.Since(batchTimer))

		allGroups = allGroups[maxExecutes:]
		if len(allGroups) == 0 {
			break
		}
		batchCounter++
	}

	// Create the csv file
	csvFile, _ := os.Create(ReportsPath + string(os.PathSeparator) + "groupsMap.csv")
	defer csvFile.Close()

	// Create the csv writer
	csvWriter := csv.NewWriter(csvFile)
	// Write the header
	csvWriter.Write([]string{"GROUP_EMAIL", "MEMBERS_COUNT", "OWNER_COUNT", "OWNERS", "MANAGER_COUNT", "MANAGERS", "SUB_COUNT", "SUBSCRIPTIONS"})
	// Flush the writer to the file so the header is written
	csvWriter.Flush()

	// Write the csvRows to the csv
	for i, row := range csvRows {
		PrintProgressBar(i+1, len(csvRows), "Writing groups to csv")
		err = csvWriter.Write(row)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		csvWriter.Flush()
	}

	// Start: Upload the reports to Google
	uploadReport(ReportsPath, googleClient)
	// End: Upload the reports to Google ^^^^

	log.Printf("Group memberships completed in %s", time.Since(timer).String())
}

// uploadReport This function uploads a folder to Google Drive
func uploadReport(outputPath string, googleClient *http.Client) {
	reportsTimer := time.Now()
	// Start: Zip the reports folder ***********************************************************************************
	zipFile := ZipDirectory(outputPath, outputPath+".zip")
	zipped, err := os.ReadFile(zipFile.Name())
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	// End: Zip the reports folder *************************************************************************************

	// Start: Upload the zipped file to Google Drive ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
	log.Println("Uploading zipped file to Google Drive...")
	uploadedFile, err := GoogleAPI.NewDriveAPI(googleClient, 3, CTX).
		UploadFile(
			zipped,
			zipFile.Name(), DriveReportsPath)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	log.Printf("Uploaded file...\nfile_id: %s\nweb_view_link: %s\nfile_size: %dMB in %s",
		uploadedFile.Name, uploadedFile.WebViewLink, uploadedFile.Size/1024/1024, time.Since(reportsTimer).String())
	// End: Upload the zipped file to Google Drive ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
}

func sharedDrivesAudit(googleClient *http.Client) {
	timer := time.Now()
	driveAPI := GoogleAPI.NewDriveAPI(googleClient, 3, CTX)

	// Get all the shared drives
	allDrives := driveAPI.GetAllDrives()
	log.Printf("Found %d shared drives", len(allDrives))

	// Create the groups csv
	var csvRows [][]string

	totalJobs := len(allDrives)
	maxExecutes := 100
	totalBatches := (totalJobs / maxExecutes) + 1
	batchCounter := 1
	for {
		log.Printf("<----- Batch [%d] of [%d] ----->\n", batchCounter, totalBatches)
		if len(allDrives) < maxExecutes {
			maxExecutes = len(allDrives)
		}

		wg := &sync.WaitGroup{}
		wg.Add(maxExecutes)

		for _, job := range allDrives[:maxExecutes] {
			go func(worker *GoogleAPI.SharedDrive) {
				defer wg.Done()

				ownerCount := 0
				organizerCount := 0
				fileOrganizerCount := 0
				writerCount := 0
				commenterCount := 0
				readerCount := 0
				for _, permission := range worker.Permissions {
					switch permission.Role {
					case "owner":
						ownerCount++
						break
					case "organizer":
						organizerCount++
						break
					case "fileOrganizer":
						fileOrganizerCount++
						break
					case "writer":
						writerCount++
						break
					case "commenter":
						commenterCount++
						break
					case "reader":
						readerCount++
						break
					}
				}

				m := make(map[string]string)
				for _, group := range worker.Groups {
					m[group.EmailAddress] = group.Role
				}

				groups := ""
				if len(m) > 0 {
					mapJson, err := json.Marshal(m)
					if err != nil {
						log.Println(err.Error())
						panic(err)
					}
					groups = string(mapJson)
				}

				csvRows = append(csvRows, []string{
					worker.MetaData.Id,
					worker.MetaData.Name,
					strconv.Itoa(ownerCount),
					strconv.Itoa(organizerCount),
					strconv.Itoa(fileOrganizerCount),
					strconv.Itoa(writerCount),
					strconv.Itoa(commenterCount),
					strconv.Itoa(readerCount),
					groups})
			}(job)
		}
		wg.Wait()

		allDrives = allDrives[maxExecutes:]
		if len(allDrives) == 0 {
			break
		}
		batchCounter++
	}

	// Create the csv file
	csvFile, _ := os.Create(ReportsPath + string(os.PathSeparator) + "sharedDrivesMap.csv")
	defer csvFile.Close()

	// Create the csv writer
	csvWriter := csv.NewWriter(csvFile)
	// Write the header
	csvWriter.Write([]string{"DRIVE_ID", "DRIVE_NAME", "OWNER_COUNT", "ORGANIZER_COUNT", "FILE_ORGANIZER_COUNT", "WRITER_COUNT", "COMMENTER_COUNT", "READER_COUNT", "GROUPS"})
	// Flush the writer to the file so the header is written
	csvWriter.Flush()

	// Write the csvRows to the csv
	for i, row := range csvRows {
		PrintProgressBar(i+1, len(csvRows), "Writing shared drives to csv")
		err := csvWriter.Write(row)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		csvWriter.Flush()
	}

	// Start: Upload the reports to Google
	uploadReport(ReportsPath, googleClient)
	// End: Upload the reports to Google ^^^^

	log.Printf("Shared Drives completed in %s", time.Since(timer).String())
}

// googleAppsScriptAudit audits all the Google Apps Scripts in the domain
func googleAppsScriptAudit(googleClient *http.Client) {
	log.Printf("Starting Google Apps Script audit...")
	timer := time.Now()
	// Get the delegation key data
	log.Println("Getting delegation key data...")
	keyData, err := os.ReadFile(DelegationKeyPath)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	// Initialize the Google Directory API
	log.Printf("Initializing Google Directory API...")
	directoryAPI := GoogleAPI.NewDirectoryAPI(googleClient, 3, CTX)

	// Pull all the users from the domain
	log.Printf("Pulling all users from the domain...")
	allUsers, err := directoryAPI.QueryUsers("")
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	// Create rows for the csv
	var csvRows [][]string

	// Loop through all the users
	log.Printf("Looping through %d users...", len(allUsers))
	for i, user := range allUsers {
		// Set the user's primary email as the subject for the JWT
		log.Printf("[%d] of [%d] Scanning user: %s", i+1, len(allUsers), user.PrimaryEmail)
		jwt := GoogleAPI.GetJWTClient(user.PrimaryEmail, keyData, []string{drive.DriveReadonlyScope}, CTX)
		// Initialize the Google Drive API
		driveAPI := GoogleAPI.NewDriveAPI(jwt, 3, CTX)
		// Get all the Google Apps Scripts owned by the user
		files, err := driveAPI.GetFiles("mimeType='application/vnd.google-apps.script' AND 'me' in owners")
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		for _, file := range files {
			log.Println(file.Name)
			csvRows = append(csvRows, []string{
				file.Owners[0].EmailAddress,
				file.Id,
				file.Name,
				file.CreatedTime,
				file.ViewedByMeTime,
				strconv.FormatBool(file.Shared),
				file.TeamDriveId})
		}
	}
	headers := []string{"OWNER", "FILE_ID", "FILE_NAME", "CREATED", "LAST_VIEWED", "SHARED", "TEAM_DRIVE_ID"}
	writeCSV("userOwnedGoogleAppsScripts.csv", headers, csvRows)
	log.Printf("Google Apps Scripts completed in %s", time.Since(timer).String())
}

func writeCSV(filename string, headers []string, csvRows [][]string) {
	timer := time.Now()
	log.Printf("Writing %d rows to %s", len(csvRows)+1, filename)
	// Create the csv file
	csvFile, err := os.Create(ReportsPath + string(os.PathSeparator) + filename)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	defer csvFile.Close()

	// Create the csv writer
	csvWriter := csv.NewWriter(csvFile)
	// Write the header
	csvWriter.Write(headers)
	// Flush the writer to the file so the header is written
	csvWriter.Flush()

	// Write the csvRows to the csv
	for i, row := range csvRows {
		PrintProgressBar(i+1, len(csvRows), "Writing to "+filename)
		err := csvWriter.Write(row)
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		csvWriter.Flush()
	}
	log.Printf("Finished writing %s in %s", filename, time.Since(timer))
}
