package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

// check performs the check operation.
//
// See https://concourse-ci.org/implementing-resource-types.html#resource-check
func check(input inputJSON) ([]byte, error) {

	// XXX: input.Version.Ref should be the latest "applied" run ID, do we need
	// to check that it is? implement some configurable behavior?

	latestRun, err := getLatestRun()
	if err != nil {
		return nil, fmt.Errorf("can't get latest run timestamp: %w", err)
	}

	// NOTE: Should we check if `latestRun` is stale to proceed with drift
	// detection?
	//
	// Probably not, because if the latest run is in planned status, then other
	// created runs anyway would be in "pending" status, this means we are not
	// able to continue to perform drift detection before the current run is
	// finished.

	// TODO: Maybe this should be configurable, to allow to cancel the latest
	// run and proceed with drift detection if it is stale.

	log.Printf("latest run: %s (%s)", latestRun.ID, latestRun.Status)
	// XXX: planned run always has changes? add an assertion?
	if latestRun.Status == "planned" && latestRun.HasChanges {
		// The latest run is in planned status and has changes, this is a drift
		// that we arewaiting for.
		return json.Marshal(checkOutputJSON{
			{Ref: latestRun.ID},
		})
	}

	// return input.Version.Ref if latestRun is in a non-final state
	if !statusIsFinal(latestRun.Status) {
		log.Printf("yield for run %s to be in a final state", latestRun.ID)
		return json.Marshal(checkOutputJSON{
			{Ref: input.Version.Ref},
		})
	}

	// if Params.PollingPeriod didn't pass from the latest run has been
	// planned, return its ID
	log.Printf("latest run created %s ago",
		time.Since(latestRun.CreatedAt).Truncate(time.Second))
	if Duration(time.Since(latestRun.CreatedAt)) < input.Params.PollingPeriod {
		return json.Marshal(checkOutputJSON{
			{Ref: input.Version.Ref},
		})
	}

	// PollingPeriod passed, lets make a new Plan&Apply run
	log.Printf("polling_period passed, creating a new run")
	var message string = "Triggered by concourse-tfe-drift-resource"
	newRun, err := client.Runs.Create(
		context.Background(),
		tfe.RunCreateOptions{
			Workspace: workspace,
			Message:   &message,
		},
	)
	log.Printf("new run created: %s", newRun.ID)

	return json.Marshal(checkOutputJSON{
		{Ref: input.Version.Ref},
	})
}

// getLatestRun returns the latest run for the commit-sha of the current
// configuration version of the workspace.
func getLatestRun() (latestRun *tfe.Run, err error) {

	currentConfigurationVersion := workspace.CurrentConfigurationVersion
	if currentConfigurationVersion == nil {
		err = fmt.Errorf("workspace has no current configuration version")
		return
	}
	ingressAttributes := currentConfigurationVersion.IngressAttributes
	if ingressAttributes == nil {
		err = fmt.Errorf("latest workspace configuration version %s "+
			"has no ingress attributes",
			currentConfigurationVersion.ID,
		)
		return
	}
	commitSha := ingressAttributes.CommitSHA
	if commitSha == "" {
		err = fmt.Errorf("latest workspace configuration version %s "+
			"ingress attributes has empty commit-sha",
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
			// XXX: do we need to exclude "canceled"?
			//Status: "pending,fetching,fetching_completed,pre_plan_running,pre_plan_completed,queuing,plan_queued,planning,planned,cost_estimating,cost_estimated,policy_checking,policy_override,policy_soft_failed,policy_checked,confirmed,post_plan_running,post_plan_completed,planned_and_finished,planned_and_saved,apply_queued,applying,applied,discarded,errored",
		},
	)

	// check for unlikely situation that Terraform Cloud API returned no runs
	// filtered by commit of latest run
	if len(runs.Items) == 0 {
		err = fmt.Errorf(
			"Terraform Cloud API returned no runs for the commit \"%s\" "+
				"of the current configuration version \"%s\" "+
				"(this should never happen)",
			commitSha, currentConfigurationVersion.ID,
		)
		return
	}

	return runs.Items[0], nil
}

// statusIsFinal returns true if the status is final, false otherwise.
//
// See https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run#run-states
func statusIsFinal(status tfe.RunStatus) bool {
	switch status {
	case
		"policy_soft_failed",
		"planned_and_finished",
		"applied",
		"discarded",
		"errored",
		"cancelled":
		return true
	default:
		return false
	}
}
