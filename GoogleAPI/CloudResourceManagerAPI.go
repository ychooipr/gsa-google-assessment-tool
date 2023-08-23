package GoogleAPI

import (
	"context"
	"fmt"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"strings"
	"time"
)

// CloudResourceManagerAPI This is the struct that is used to interact with the Cloud Resource Manager API
type CloudResourceManagerAPI struct {
	Client    *cloudresourcemanager.Service
	SleepTime int
	MaxTries  int
}

// NewCloudResourceManagerAPI returns a new CloudResourceManagerAPI
func NewCloudResourceManagerAPI(client *http.Client, sleepTimer int, ctx context.Context) *CloudResourceManagerAPI {
	// Create a new CloudResourceManagerAPI
	newAPI := &CloudResourceManagerAPI{}

	// Create a Firestore client
	firestoreClient, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// Set the Firestore client
	newAPI.Client = firestoreClient
	newAPI.SleepTime = sleepTimer
	newAPI.MaxTries = 10
	return newAPI
}

// GetAllProjects returns all projects
func (receiver *CloudResourceManagerAPI) GetAllProjects() ([]*cloudresourcemanager.Project, error) {
	// create a slice to hold all projects
	var allProjects []*cloudresourcemanager.Project
	// create a page token to hold the next page token
	var pageToken string

	// Set the number of tries
	tryCounter := 0
	// loop through all pages
	for {
		// get the next page of projects
		response, err := receiver.Client.Projects.List().
			Fields("*").
			PageToken(pageToken).Do()
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

		// append the projects to the slice
		allProjects = append(allProjects, response.Projects...)
		log.Printf("Projects thus far: %d", len(allProjects))

		// if there is no next page, break
		if response.NextPageToken == "" {
			break
		}

		// set the page token to the next page token
		pageToken = response.NextPageToken
	}

	// return all projects
	return allProjects, nil
}
