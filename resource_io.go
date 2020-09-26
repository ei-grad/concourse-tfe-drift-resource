package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
)

type (
	version struct {
		Ref string `json:"ref"`
	}
	sourceJSON struct {
		Workspace     string `json:"workspace"`
		Organization  string `json:"organization"`
		Token         string `json:"token"`
		Address       string `json:"address""`
	}
	inputJSON struct {
		Params  paramsJSON `json:"params"`
		Source  sourceJSON `json:"source"`
		Version version    `json:"version"`
	}
	versionMetadata struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	checkOutputJSON []version
	inOutputJSON    struct {
		Version  version           `json:"version"`
		Metadata []versionMetadata `json:"metadata"`
	}
	outOutputJSON inOutputJSON
	paramsJSON    struct {
		Vars          map[string]variableJSON `json:"vars"`
		EnvVars       map[string]variableJSON `json:"env_vars"`
		Message       string                  `json:"message"`
		Confirm       bool                    `json:"confirm"`
		PollingPeriod int                     `json:"polling_period"`
		Sensitive     bool                    `json:"sensitive"`
	}
	variableJSON struct {
		File        string `json:"file"`
		Value       string `json:"value"`
		Description string `json:"description"`
		Sensitive   bool   `json:"sensitive"`
		Hcl         bool   `json:"hcl"`
	}
)

func getInputs(in io.Reader) (inputJSON, error) {
	input := inputJSON{}
	input.Source = sourceJSON{
		Address:       "https://app.terraform.io",
	}
	input.Params = paramsJSON{
		Message: fmt.Sprintf("Queued by %s/%s (%s)",
			os.Getenv("BUILD_PIPELINE_NAME"),
			os.Getenv("BUILD_JOB_NAME"),
			os.Getenv("BUILD_NAME")),
		PollingPeriod: 5,
		Sensitive: false,
	}

	decoder := json.NewDecoder(in)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&input)
	if err != nil {
		return input, formatError(err, "parsing input")
	}

	// a few sanity checks
	if _, err := url.ParseRequestURI(input.Source.Address); err != nil {
		return input, formatError(err,"parsing source address")
	}
	if input.Source.Workspace == "" || input.Source.Organization == "" || input.Source.Token == "" {
		return input, fmt.Errorf("error parsing input: workspace, organization, and token fields must be set")
	}
	if input.Params.PollingPeriod < 1 {
		return input, fmt.Errorf("error parsing input: polling_period must be at least 1 second")
	}
	return input, nil
}

func formatError(err error, context string) error {
	return fmt.Errorf("error %s: %s", context, err)
}
