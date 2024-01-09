package main

import (
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-tfe"
	"go.uber.org/mock/gomock"
)

var (
	ctrl          *gomock.Controller
	mockClient    tfe.Client
	runs          *MockRuns
	workspaces    *MockWorkspaces
	variables     *MockVariables
	stateVersions *MockStateVersions
	test          *testing.T
)

func setup(t *testing.T) tfe.Run {
	test = t
	ctrl = gomock.NewController(t)

	mockClient = tfe.Client{}
	client = &mockClient
	runs = NewMockRuns(ctrl)
	client.Runs = runs
	workspaces = NewMockWorkspaces(ctrl)
	client.Workspaces = workspaces
	variables = NewMockVariables(ctrl)
	client.Variables = variables
	stateVersions = NewMockStateVersions(ctrl)
	client.StateVersions = stateVersions

	workspace = &tfe.Workspace{
		ID:           "foo",
		Organization: &tfe.Organization{CostEstimationEnabled: false},
	}

	return tfe.Run{
		ID:        "bar",
		Status:    tfe.RunPending,
		Message:   "test run",
		CreatedAt: time.Now(),
		CostEstimate: &tfe.CostEstimate{
			DeltaMonthlyCost:    "+a billion dollars",
			ProposedMonthlyCost: "a few cents",
		},
		Actions:              &tfe.RunActions{IsConfirmable: true},
		ConfigurationVersion: &tfe.ConfigurationVersion{Source: tfe.ConfigurationSourceGithub},
		HasChanges:           true,
	}
}

func didntErrorWithSubstr(err error, expected string) bool {
	return err == nil || !strings.Contains(err.Error(), expected)
}
