package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"github.com/kalafut/m2j"

	"github.com/tomhjp/gh-action-jira/config"
	"github.com/tomhjp/gh-action-jira/jira"
)

func main() {
	err := create()
	if err != nil {
		log.Fatal(err)
	}
}

func create() error {
	project := os.Getenv("INPUT_PROJECT")
	if project == "" {
		return errors.New("no project provided as input")
	}
	issueType := os.Getenv("INPUT_ISSUE_TYPE")
	summary := os.Getenv("INPUT_SUMMARY")
	descriptionString := os.Getenv("INPUT_DESCRIPTION")
	descriptionFile := os.Getenv("INPUT_DESCRIPTION_FILE")
	if descriptionString != "" && descriptionFile != "" {
		return errors.New("cannot provide both `description` and `description_file`")
	}
	var description string
	if descriptionString != "" {
		description = descriptionString
	} else {
		fmt.Printf("Reading description contents from file %q", descriptionFile)
		contents, err := ioutil.ReadFile(descriptionFile)
		if err != nil {
			return err
		}
		description = string(contents)
	}
	extraFieldsString := os.Getenv("INPUT_EXTRA_FIELDS")
	extraFields := map[string]interface{}{}
	err := json.Unmarshal([]byte(extraFieldsString), &extraFields)
	if err != nil {
		return fmt.Errorf("failed to deserialise extraFields: %s", err)
	}

	config, err := config.ReadConfig()
	if err != nil {
		return err
	}

	key, err := createIssue(config, project, issueType, summary, description, extraFields)
	if err != nil {
		return err
	}

	fmt.Printf("Created issue %s\n", key)

	// Special format log line to set output for the action.
	// See https://docs.github.com/en/actions/creating-actions/metadata-syntax-for-github-actions#outputs-for-composite-run-steps-actions.
	fmt.Printf("::set-output name=key::%s\n", key)

	return nil
}

type createIssuePayload struct {
	Fields map[string]interface{} `json:"fields"`
}

type createIssueResponse struct {
	Key string `json:"key"`
}

func createIssue(config config.JiraConfig, project, issueType, summary, description string, extraFields map[string]interface{}) (string, error) {
	payload := constructPayload(project, issueType, summary, description, extraFields)
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Use the REST API v2 because it has a much simpler schema for description.
	respBody, err := jira.DoRequest(config, "POST", "/rest/api/2/issue", url.Values{}, bytes.NewReader(reqBody))
	if err != nil {
		indentedBody, marshalErr := json.MarshalIndent(payload, "", "  ")
		if marshalErr != nil {
			// We made a best effort, oh well, just print it ugly.
			indentedBody = reqBody
		}
		fmt.Println("Request body:")
		fmt.Printf("%s\n", string(indentedBody))
		return "", err
	}

	var response createIssueResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return "", err
	}
	return response.Key, nil
}

// An abomination of type unsafety to allow us to handle arbitrary JSON values for extraFields
func constructPayload(project, issueType, summary, description string, extraFields map[string]interface{}) createIssuePayload {
	payload := createIssuePayload{
		Fields: map[string]interface{}{
			"project": struct {
				Key string `json:"key"`
			}{
				project,
			},
			"issuetype": struct {
				Name string `json:"name"`
			}{
				issueType,
			},
			"summary":     summary,
			"description": m2j.MDToJira(description),
		},
	}
	for key, value := range extraFields {
		payload.Fields[key] = value
	}

	return payload
}
