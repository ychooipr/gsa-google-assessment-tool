package GoogleAPI

import (
	"context"
	"encoding/json"
	"fmt"
	directory "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AdminScopes This is the list of scopes for the Admin Directory API
var AdminScopes = []string{
	directory.AdminDirectoryUserReadonlyScope,
	directory.AdminDirectoryGroupReadonlyScope,
	directory.AdminDirectoryGroupMemberReadonlyScope,
	directory.AdminDirectoryResourceCalendarReadonlyScope,
	directory.AdminDirectoryUserSecurityScope,
}

// DirectoryAPI This is the struct that is used to interact with the Admin SDK
type DirectoryAPI struct {
	DirectoryService *directory.Service
	Customer         string
	SleepTime        int
	Jobs             *sync.WaitGroup
	MaxTries         int
}

// NewDirectoryAPI  This method is used to create a new DirectoryAPI
func NewDirectoryAPI(client *http.Client, sleepTime int, ctx context.Context) *DirectoryAPI {
	// Create a new Directory API
	newAdminAPI := &DirectoryAPI{}

	// Create a new directory service
	adminService, err := directory.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	// Set the adminService
	newAdminAPI.DirectoryService = adminService

	// Create a new WaitGroup
	newAdminAPI.Jobs = &sync.WaitGroup{}

	// Set the sleep time
	newAdminAPI.Customer = "my_customer"

	// Set the sleep time
	newAdminAPI.SleepTime = sleepTime

	// Set the max tries
	newAdminAPI.MaxTries = 10

	// Return the new DriveAPI
	return newAdminAPI
}

// QueryUsers This method is used to query and return a list of users
func (receiver *DirectoryAPI) QueryUsers(q string) ([]*directory.User, error) {
	// Set the page token to an empty string
	pt := ""

	// Create a new userList
	var userList []*directory.User

	// Create a new request
	request := receiver.
		DirectoryService.
		Users.
		List().
		Fields("*").
		PageToken(pt).
		Customer(receiver.Customer)

	// If there is a query add it to the request
	if q != "" {
		request.Query(q)
	}

	// A counter for the number of tries
	tryCounter := 0
	// Loop through all pages
	for {
		response, err := receiver.
			DirectoryService.
			Users.
			List().
			Query(q).
			Fields("*").
			PageToken(pt).
			Customer(receiver.Customer).
			Do()

		if err != nil {
			if strings.Contains(err.Error(), "quota") {
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

		// Pass the current users to the userList
		userList = append(userList, response.Users...)
		log.Printf("Users thus far: %d", len(userList))

		// Check if there is a next page and add it to the page token
		pt = response.NextPageToken

		if pt == "" {
			break
		}
	}

	// Return the userList
	return userList, nil
}

// QueryGroups This method is used to get a list of groups
func (receiver *DirectoryAPI) QueryGroups(q string) ([]*directory.Group, error) {
	// Set the page token to an empty string
	pt := ""

	// Create a new groupsList
	var groupsList []*directory.Group

	// Create a new page
	page := receiver.DirectoryService.Groups.
		List().
		Customer(receiver.Customer).
		SortOrder("ASCENDING").
		Fields("*")

	// If there is a query add it to the page
	if q != "" {
		page.Query(q)
	}

	// Loop through all pages
	for {
		// Get the groups
		response, err := page.PageToken(pt).Do()
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

		// Pass the current groups to the groupsList
		groupsList = append(groupsList, response.Groups...)

		log.Printf("Groups thus far: %d", len(groupsList))

		// Check if there is a next page and add it to the page token
		pt = response.NextPageToken
		// if there is no next page break
		if pt == "" {
			break
		}
	}

	// Return the groupsList
	return groupsList, nil
}

// GetGroupMembers This method pulls all the members of a group
func (receiver *DirectoryAPI) GetGroupMembers(groupEmail *directory.Group, role string) ([]*directory.Member, error) {
	request := receiver.
		DirectoryService.
		Members.
		List(groupEmail.Email).
		Fields("*")

	if role != "" {
		request.Roles(role)
	}

	pt := ""

	var memberList []*directory.Member
	for {
		response, err := request.PageToken(pt).Do()
		if err != nil {
			if strings.Contains(err.Error(), "quota") {
				//fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), time.Duration(receiver.SleepTime))
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

		memberList = append(memberList, response.Members...)
		//log.Printf("Pulled %d of %d members from %s", len(memberList), groupEmail.DirectMembersCount, groupEmail.Email)
		// Check if there is a next page and add it to the page token
		pt = response.NextPageToken
		// if there is no next page break
		if pt == "" {
			break
		}
	}

	return memberList, nil
}

// GetSubscriptions GetGroupMembers This method gets all the groups a member is a part of
func (receiver *DirectoryAPI) GetSubscriptions(memberEmail string) ([]*directory.Group, error) {
	var parents []*directory.Group
	pageToken := ""
	// Loop through all pages
	for {
		// Get the groups
		page, err := receiver.DirectoryService.Groups.
			List().
			UserKey(memberEmail).Fields("*").
			PageToken(pageToken).Do()

		// Check for errors
		if err != nil {
			if strings.Contains(err.Error(), "quota") {
				//fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), 2)
				time.Sleep(2 * time.Second)
				continue
			} else if strings.Contains(err.Error(), "500") {
				fmt.Printf("%s, sleeping for %d seconds ...", err.Error(), 60)
				time.Sleep(time.Second * 60)
				continue
			} else {
				return nil, err
			}
		}

		// append the groups to the parents
		parents = append(parents, page.Groups...)

		// Check if there is a next page and add it to the page token
		pageToken = page.NextPageToken
		// if there is no next page break
		if pageToken == "" {
			break
		}
	}
	return parents, nil
}

// GetUserTokens This method is used to get a list of tokens for a user
func (receiver *DirectoryAPI) GetUserTokens(userEmail string) ([]*directory.Token, error) {
	// A counter for the number of tries
	tryCounter := 0
	for {
		res, err := receiver.DirectoryService.Tokens.List(userEmail).Fields("*").Do()
		if err != nil {
			if strings.Contains(err.Error(), "quota") {
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
		return res.Items, nil
	}
}

// GoogleUser This struct is used to hold a user and their tokens
type GoogleUser struct {
	Id               string `json:"id"`
	PrimaryEmail     string `json:"primary_email"`
	Archived         bool   `json:"archived"`
	IsAdmin          bool   `json:"is_admin"`
	IsDelegatedAdmin bool   `json:"is_delegated_admin"`
	Suspended        bool   `json:"is_suspended"`
	IsMailboxSetup   bool   `json:"is_mailbox_setup"`
	LastLoginTime    string `json:"last_login_time"`
	Tokens           any    `json:"tokens"`
}

// GetUsersAndToken This method is used to get a list of users and their tokens
func (receiver *DirectoryAPI) GetUsersAndToken(q string) ([]*GoogleUser, error) {
	userList, err := receiver.QueryUsers(q)
	if err != nil {
		log.Printf("Error getting users: %s", q)
		return nil, err
	}
	var users []*GoogleUser

	// Print the users to a firestore collection----------------------------------------------------------
	totalJobs := len(userList)
	maxExecutes := 100
	totalBatches := (totalJobs / maxExecutes) + 1
	batchCounter := 1
	for {
		log.Printf("<----- Users Tokens Batch [%d] of [%d] ----->\n", batchCounter, totalBatches)
		if len(userList) < maxExecutes {
			maxExecutes = len(userList)
		}

		wg := &sync.WaitGroup{}
		wg.Add(maxExecutes)

		for _, job := range userList[:maxExecutes] {
			go func(user *directory.User) {
				defer wg.Done()
				// Get the user tokens
				tokens, err := receiver.GetUserTokens(user.PrimaryEmail)
				// Check for errors
				if err != nil {
					fmt.Printf("Error getting tokens for user: %s", user.PrimaryEmail)
					return
				} else if tokens == nil {
					log.Printf("No tokens for user: %s", user.PrimaryEmail)
					return
				}
				// Serialize the tokens
				data, _ := json.Marshal(tokens)
				// Pass the tokens to the user
				u := &GoogleUser{
					Id:               user.Id,
					PrimaryEmail:     user.PrimaryEmail,
					Archived:         user.Archived,
					IsAdmin:          user.IsAdmin,
					IsDelegatedAdmin: user.IsDelegatedAdmin,
					LastLoginTime:    user.LastLoginTime,
					Suspended:        user.Suspended,
					IsMailboxSetup:   user.IsMailboxSetup,
					Tokens:           string(data),
				}
				users = append(users, u)
			}(job)
		}
		wg.Wait()

		userList = userList[maxExecutes:]
		if len(userList) == 0 {
			break
		}
		batchCounter++
	}

	return users, nil
}
