package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// analyzeUsers takes the users.csv file and analyzes the users
func main() {
	outputFile := "data/output.csv"
	csvFile, err := os.Create(outputFile)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	csvWriter.Write([]string{"userEmail", "archived", "isAdmin", "isDelegatedAdmin", "suspended", "lastLoginTime", "isMailboxSetup", "clientID", "displayText", "kind", "scopes"})
	csvWriter.Flush()

	fileData, err := os.ReadFile("data/users.csv")
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	records, err := csv.NewReader(bytes.NewReader(fileData)).ReadAll()
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	records = append(records[:0], records[1:]...)
	for _, r := range records {
		//PrintProgressBar(i, len(records), "Analyzing users...")
		userEmail := r[1]
		archived := r[2]
		isAdmin := r[3]
		isDelegatedAdmin := r[4]
		suspended := r[5]
		lastLoginTime := r[6]
		isMailboxSetup := r[7]

		// Define a slice of maps to store the JSON tokens
		var tokens []map[string]interface{}

		// Get the JSON tokens from the string
		b := []byte(r[8])

		// Unmarshal the JSON tokens into the slice of maps
		err := json.Unmarshal(b, &tokens)
		if err != nil {
			fmt.Println("Error:", err)
			panic(err)
		}

		// Print the JSON tokens in a formatted way to the console and to the csv file as well
		for _, token := range tokens {
			clientId := token["clientId"].(string)
			displayText := token["displayText"].(string)
			kind := token["kind"].(string)
			scopes := token["scopes"].([]any)
			csvWriter.Write(
				[]string{userEmail,
					archived,
					isAdmin,
					isDelegatedAdmin,
					suspended,
					lastLoginTime,
					isMailboxSetup,
					clientId,
					displayText,
					kind,
					fmt.Sprint(scopes)})
			csvWriter.Flush()
		}
	}

}
