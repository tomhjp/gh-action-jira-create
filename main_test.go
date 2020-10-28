package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructPayload(t *testing.T) {
	const expectedJSON = `{"fields":{"custom_field":[{"name":"foo"}],"description":"The description with some {{code}}","foo":"bar","issuetype":{"name":"Bug"},"project":{"key":"FOO"},"summary":"The summary"}}`
	reqBody, err := json.Marshal(constructPayload("FOO", "Bug", "The summary", "The description with some {{code}}", map[string]interface{}{
		"foo":          "bar",
		"custom_field": []map[string]string{{"name": "foo"}},
	}))
	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(reqBody))
}
