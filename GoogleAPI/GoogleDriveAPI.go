package GoogleAPI

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DriveAPI This is the struct that is used to interact with the Google Drive API
type DriveAPI struct {
	Service   *drive.Service
	Subject   string
	SleepTime int
	Jobs      *sync.WaitGroup
	MaxTries  int
}

// NewDriveAPI This method is used to create a new DriveAPI client
func NewDriveAPI(client *http.Client, sleepTime int, ctx context.Context) *DriveAPI {
	newDriveAPI := &DriveAPI{}
	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	newDriveAPI.Service = service
	newDriveAPI.Jobs = &sync.WaitGroup{}
	newDriveAPI.SleepTime = sleepTime
	newDriveAPI.MaxTries = 10
	return newDriveAPI
}

// UploadFile This method is used to change the owner of a file
func (receiver *DriveAPI) UploadFile(fileData []byte, fileName, parentFolderId string) (*drive.File, error) {
	// Count the number of bytes in the file
	byteCount := func(b int64) string {
		const unit = 1000
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB",
			float64(b)/float64(div), "kMGTPE"[exp])
	}

	// Get the file data and create a reader
	reader := bytes.NewReader(fileData)
	// Create the file metadata
	var metaData = &drive.File{Name: fileName}
	if parentFolderId != "" {
		var parents []string
		parents = append(parents, parentFolderId)
		metaData.Parents = parents
	}
	// Create the progress updater
	progressUpdater := googleapi.ProgressUpdater(func(now, size int64) {
		log.Println("CurrentFile:",
			"["+byteCount(now), "of", byteCount(reader.Size())+"]")
	})

	// Try to upload the file
	tryCounter := 0
	// Loop until we get a result
	for {
		// Upload the file
		result, err := receiver.Service.Files.Create(metaData).Media(reader).ProgressUpdater(progressUpdater).Fields("*").Do()
		if err != nil {
			if strings.Contains(err.Error(), "quota") && tryCounter < receiver.MaxTries {
				fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), time.Duration(receiver.SleepTime))
				time.Sleep(time.Duration(receiver.SleepTime) * time.Second)
				continue
			} else if strings.Contains(err.Error(), "500") && tryCounter < receiver.MaxTries {
				fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), 60)
				time.Sleep(time.Second * 60)
				tryCounter++
				continue
			} else {
				return nil, err
			}
		}
		return result, err
	}

}

// GetAllDrives This method is used to get all of the shared drives
func (receiver *DriveAPI) GetAllDrives() []*SharedDrive {
	drivesListCall := receiver.Service.Drives.List().PageSize(100).UseDomainAdminAccess(true).Fields("*")
	var sharedDrives []*SharedDrive
	for {
		drivesList, err := drivesListCall.Do()
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}

		totalJobs := len(drivesList.Drives)
		maxExecutes := 10
		totalBatches := (totalJobs / maxExecutes)
		batchCounter := 1
		for {
			log.Printf("<----- Batch [%d] of [%d] ----->\n", batchCounter, totalBatches)
			if len(drivesList.Drives) < maxExecutes {
				maxExecutes = len(drivesList.Drives)
			}

			wg := &sync.WaitGroup{}
			wg.Add(maxExecutes)

			for _, job := range drivesList.Drives[:maxExecutes] {
				go func(worker *drive.Drive) {
					defer wg.Done()
					sd := &SharedDrive{
						MetaData:    worker,
						Permissions: receiver.GetFilePermissions(worker.Id)}
					for _, permission := range sd.Permissions {
						switch permission.Type {
						case "user":
							sd.Users = append(sd.Users, permission)
						case "group":
							sd.Groups = append(sd.Groups, permission)
						case "domain":
							sd.Domain = permission.Domain
						}
					}
					sharedDrives = append(sharedDrives, sd)
				}(job)
			}
			wg.Wait()

			drivesList.Drives = drivesList.Drives[maxExecutes:]
			if len(drivesList.Drives) == 0 {
				break
			}
			batchCounter++
		}

		log.Println("Shared Drive thus far:", len(sharedDrives))
		if drivesList.NextPageToken == "" {
			break
		}
		drivesListCall.PageToken(drivesList.NextPageToken)
	}
	return sharedDrives
}

// GetFilePermissions This method is used to get the permissions for a file
func (receiver *DriveAPI) GetFilePermissions(fileId string) []*drive.Permission {
	msg := fmt.Sprintf("Getting permissions for[%s]", fileId)
	defer func() { log.Println(msg) }()
	request := receiver.Service.Permissions.List(fileId).Fields("*").SupportsAllDrives(true).
		SupportsTeamDrives(true).UseDomainAdminAccess(true).PageSize(100)
	var permissions []*drive.Permission
	for {
		permissionList, err := request.Do()
		if err != nil {
			log.Println(err.Error())
			panic(err)
		}
		permissions = append(permissions, permissionList.Permissions...)
		if permissionList.NextPageToken == "" {
			break
		}
		request.PageToken(permissionList.NextPageToken)
	}
	msg += fmt.Sprintf(" - %d permissions found", len(permissions))
	return permissions
}

// SharedDrive is a struct that contains the metadata and permissions for a shared drive
type SharedDrive struct {
	MetaData    *drive.Drive
	Permissions []*drive.Permission
	Groups      []*drive.Permission
	Users       []*drive.Permission
	Domain      string
}

// GetFiles returns a list of files in the user's drive
func (receiver *DriveAPI) GetFiles(q string) ([]*drive.File, error) {
	// Create a list to store the files
	var files []*drive.File

	// Get the files in the user's drive and add them to the list
	for pt := ""; ; {
		// Get the files
		res, err := receiver.Service.Files.List().Q(q).PageSize(1000).PageToken(pt).Fields("*").Do()
		if err != nil {
			if strings.Contains(err.Error(), "quota") {
				fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), time.Duration(receiver.SleepTime))
				time.Sleep(time.Duration(receiver.SleepTime) * time.Second)
				continue
			} else if strings.Contains(err.Error(), "500") {
				fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), 60)
				time.Sleep(time.Second * 60)
				continue
			} else {
				return nil, err
			}
		}

		// Add the files to the list
		files = append(files, res.Files...)
		log.Printf("User %s files thus far: %d", receiver.Subject, len(files))

		// If there is no next page, break the loop
		pt = res.NextPageToken
		if pt == "" {
			break
		}
	}
	return files, nil
}
