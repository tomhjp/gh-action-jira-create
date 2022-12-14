package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/tomhjp/gh-action-jira/config"
	"github.com/tomhjp/gh-action-jira/format"
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
	description := os.Getenv("INPUT_DESCRIPTION")
	extraFieldsString := os.Getenv("INPUT_EXTRA_FIELDS")
	extraFields := map[string]interface{}{}
	err := json.Unmarshal([]byte(extraFieldsString), &extraFields)
	if err != nil {
		return fmt.Errorf("failed to deserialise extraFields: %w", err)
	}

	config, err := config.ReadConfig()
	if err != nil {
		return err
	}

	description, err = format.GitHubToJira(description)
	if err != nil {
		return fmt.Errorf("failed to convert GitHub markdown to Jira: %w", err)
	}

	key, err := createIssue(config, project, issueType, summary, description, extraFields)
	if err != nil {
		return err
	}

	fmt.Printf("Created issue %s\n", key)

	// Special format log line to set output for the action.
	// The `set-output` command is deprecated and will be disabled soon. Please upgrade to using Environment Files. 
	// For more information see: https://github.blog/changelog/2022-10-11-github-actions-deprecating-save-state-and-set-output-commands/
	fmt.Printf("echo \"key=%s\" >> $GITHUB_OUTPUT\n", key)

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
			"description": description,
		},
	}
	for key, value := range extraFields {
		payload.Fields[key] = value
	}

	return payload
}
