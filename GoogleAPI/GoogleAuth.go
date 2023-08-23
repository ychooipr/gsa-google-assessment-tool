package GoogleAPI

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"strings"
)

// GetJWTClient returns a new client with the delegated credentials
func GetJWTClient(subjectEmail string, delegationKey []byte, scopes []string, ctx context.Context) *http.Client {
	// Create a new client with the delegated credentials
	jwt, err := google.JWTConfigFromJSON(delegationKey)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	// Set the subject to the user's email address
	jwt.Subject = subjectEmail

	// Set the scopes for the client
	jwt.Scopes = scopes

	// Return the client
	return jwt.Client(ctx)
}

// GetOAuth2GoogleClient returns a new client with the delegated credentials
func GetOAuth2GoogleClient(clientSecret, oauth2Token []byte, scopes []string, ctx context.Context) *http.Client {

	// Get the config from the client secret data
	config, err := google.ConfigFromJSON(clientSecret, scopes...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Unmarshal the token
	token := &oauth2.Token{}

	// Unmarshal the token data
	err = json.Unmarshal(oauth2Token, token)
	if err != nil {
		log.Fatalf("Unable to unmarshal token data %v", err)
	}

	// Return the client
	return config.Client(ctx, token)
}

// GenerateOAuth2Token  returns a new client with the delegated credentials
func GenerateOAuth2Token(clientSecretData []byte, scopes []string, ctx context.Context) (*oauth2.Token, []byte) {
	// Get the config from the client secret data
	config, err := google.ConfigFromJSON(clientSecretData, scopes...)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Get the token from the user
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser, authorize the app and then copy the "+
		"response url and paste it below: \n\n%v\n\n\nPaste response url here:", authURL)

	// Use fmt.Scan() to read from stdin instead of bufio.NewReader(os.Stdin).ReadString('\n')
	var authCode string
	// Read the auth code from the command-lin

	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	// Remove the code= and everything before from the auth code
	authCode = strings.Split(authCode, "code=")[1]
	// Remove the &scope and everything after from the auth code
	authCode = strings.Split(authCode, "&")[0]

	// Exchange the auth code for a token
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}

	// Marshal the token to JSON
	tokenData, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Unable to marshal token to JSON %v", err)
	}

	// Return the token and the token data
	return token, tokenData
}
