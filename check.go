package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

func check(input inputJSON) ([]byte, error) {

	latestRun, err := getLatestRun(
		client, input.Source.Organization, input.Source.Workspace,
	)
	if err != nil {
		return nil, fmt.Errorf("can't get latest run timestamp: %w", err)
	}
	// XXX: DEBUG
	fmt.Printf("latest run details: %+v\n", latestRun)

	// get latest run finished timestamp
	runFinishedAt, err := getRunFinishedAt(latestRun)
	if err != nil {
		return nil, fmt.Errorf("can't get latest run finished timestamp: %w", err)
	}

	// if Params.PollingPeriod didn't pass from the latest run has been planned, return its ID
	if time.Since(runFinishedAt) < input.Params.PollingPeriod {
		return json.Marshal(checkOutputJSON{
			{Ref: latestRun.ID},
		})
	}

	// PollingPeriod passed, lets make a new Plan&Apply run
	newRun, err = client.Runs.Create(
		context.Background(),
		&tfe.RunCreateOptions{
			Message: &input.Params.Message,
		},
	)

	// TODO: wait for it to finish
	// TODO: if it has changes, return its ID

	return json.Marshal(checkOutputJSON{
		{Ref: latestRun.ID},
	})
}

func getRunFinishedAt(run *tfe.Run) (finishedAt time.Time, err error) {
	// there is no dedicated field for finished timestamp in tfe.Run.StatusTimestamps
	// so we need to get it from undocumented run-events API
	runEvents, err := client.RunEvents.List(
		context.Background(),
		run.ID,
		&tfe.RunEventListOptions{},
	)
	if err != nil {
		err = fmt.Errorf("can't get run events: %w", err)
		return
	}
	for _, runEvent := range runEvents.Items {
		if runEvent.Action == "finished" {
			return runEvent.CreatedAt, nil
		}
	}
	err = fmt.Errorf("can't find \"finished\" run event")
	return
}

func getLatestRun(
	client *tfe.Client,
	organization, workspaceID string,
) (latestRun *tfe.Run, err error) {

	// get commit-sha from workspace's current configuration version ingress
	// attributes
	workspace, err := client.Workspaces.ReadWithOptions(
		context.Background(),
		organization,
		workspaceID,
		&tfe.WorkspaceReadOptions{
			Include: []tfe.WSIncludeOpt{
				tfe.WSCurrentrunConfigVerIngress,
			},
		},
	)
	if err != nil {
		err = fmt.Errorf("can't read workspace \"%s/%s\": %w",
			organization, workspaceID, err)
		return
	}
	currentConfigurationVersion := workspace.CurrentConfigurationVersion
	if currentConfigurationVersion == nil {
		err = fmt.Errorf("workspace \"%s/%s\" "+
			"has no current configuration version",
			organization, workspaceID,
		)
		return
	}
	ingressAttributes := currentConfigurationVersion.IngressAttributes
	if ingressAttributes == nil {
		err = fmt.Errorf("workspace \"%s/%s\" "+
			"latest configuration version %s "+
			"has no ingress attributes",
			organization, workspaceID,
			currentConfigurationVersion.ID,
		)
		return
	}
	commitSha := ingressAttributes.CommitSHA
	if commitSha == "" {
		err = fmt.Errorf("workspace \"%s/%s\" "+
			"latest configuration version %s ingress attributes "+
			"has empty commit-sha",
			organization, workspaceID,
			currentConfigurationVersion.ID,
		)
		return
	}

	// get latest run for this commit-sha
	runs, err := client.Runs.List(
		context.Background(),
		workspace.ID,
		&tfe.RunListOptions{
			ListOptions: tfe.ListOptions{PageSize: 1},
			Commit:      commitSha,
		},
	)

	// check for unlikely situation that Terraform Cloud API returned no runs
	// filtered by commit of latest run
	if len(runs.Items) == 0 {
		err = fmt.Errorf(
			"Terraform Cloud API returned no runs for the commit \"%s\" "+
				"of the current configuration version \"%s\" "+
				"in the workspace \"%s/%s\" "+
				"(this should never happen)",
			commitSha, currentConfigurationVersion.ID,
			organization, workspaceID,
		)
		return
	}

	return runs.Items[0], nil
}
