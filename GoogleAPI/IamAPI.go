package GoogleAPI

import (
	"context"
	"fmt"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"strings"
	"time"
)

// IamAPI is a wrapper for the Firestore client
type IamAPI struct {
	Service   *iam.Service
	SleepTime int
	MaxTries  int
}

// NewIamAPI returns a new IamAPI
func NewIamAPI(client *http.Client, sleepTime int, ctx context.Context) *IamAPI {
	// Create a new IamAPI
	newAPI := &IamAPI{}

	// Create a Firestore client
	service, err := iam.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// Set the Firestore client
	newAPI.Service = service
	newAPI.SleepTime = sleepTime
	newAPI.MaxTries = 10
	return newAPI
}

// GCPServiceAccount is a struct for a GCP service account
type GCPServiceAccount struct {
	Email          string `json:"email"`
	Oauth2ClientId string `json:"oauth_2_client_id"`
	ProjectId      string `json:"project_id"`
	UniqueId       string `json:"unique_id"`
}

// GetProjectServiceAccounts returns a list of service accounts for a project
func (receiver *IamAPI) GetProjectServiceAccounts(projectNumber string) ([]*GCPServiceAccount, error) {
	// Add the project id to the path
	projectNumber = "projects/" + projectNumber

	// Create a slice of GCPServiceAccount
	var gcpServiceAccounts []*GCPServiceAccount

	// Get the number of tries
	tryCounter := 0
	// Get the list of service accounts
	for {
		//Requesting the following scopes:
		//Billing Account Administrator
		//Folder Creator
		//Organization Administrator
		//Organization Role Administrator
		//Owner

		// Highly unlikely pagination will be needed as the number of service accounts is limited to 100
		res, err := receiver.Service.Projects.ServiceAccounts.List(projectNumber).Fields("*").Do()
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

		// Check if there are any service accounts
		if res.Accounts != nil {
			// Return the list of service accounts
			for _, account := range res.Accounts {
				// Add the service account to the slice
				gcpServiceAccounts = append(gcpServiceAccounts, &GCPServiceAccount{
					Email:          account.Email,
					Oauth2ClientId: account.Oauth2ClientId,
					ProjectId:      account.ProjectId,
					UniqueId:       account.UniqueId,
				})
			}
		}

		break
	}

	// Return the list of service accounts
	return gcpServiceAccounts, nil

}
