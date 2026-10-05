// Byline: Codex · GPT-6 · 2026-10-05. Durable sync workflow retry and running-operation join proof.
package main

import (
	"context"
	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"testing"
)

// syncDispatchClient intercepts only workflow starts without opening Temporal connections.
// Inputs: injected callback; outputs: synthetic run/error. Effects: none; choose for starter policy proof.
type syncDispatchClient struct {
	client.Client
	execute func(client.StartWorkflowOptions, interface{}, []interface{}) (client.WorkflowRun, error)
}

// ExecuteWorkflow captures dispatch options and payload without executing any workflow.
// Inputs: context/options/type/payload; outputs: callback result. Effects: none; choose for the focused dispatch fixture.
func (c syncDispatchClient) ExecuteWorkflow(_ context.Context, o client.StartWorkflowOptions, w interface{}, args ...interface{}) (client.WorkflowRun, error) {
	return c.execute(o, w, args)
}

// syncDispatchRun supplies one synthetic run ID; no workflow result is read.
// Inputs: run ID; outputs: GetRunID; effects: none; choose for dispatch-only proof.
type syncDispatchRun struct {
	client.WorkflowRun
	id string
}

// GetRunID returns the retained synthetic run identity without a network request.
// Inputs: none; outputs: run ID; effects: none; choose for the workflow-start fixture.
func (r syncDispatchRun) GetRunID() string { return r.id }

// TestLibrarySyncClosedRetryStartsNewRun proves closed retryable outcomes can reconcile the same consumed intent.
// Inputs: synthetic Temporal client; outputs: same workflow ID and new run ID. Effects: no Temporal or B2 writes.
// Choose to catch the completed-write_unknown trap that FAILED_ONLY would leave unrecoverable.
func TestLibrarySyncClosedRetryStartsNewRun(t *testing.T) {
	op := "11111111-1111-4111-8111-111111111111"
	called := false
	c := syncDispatchClient{execute: func(o client.StartWorkflowOptions, w interface{}, args []interface{}) (client.WorkflowRun, error) {
		called = true
		if o.ID != "library-sync-write-"+op || o.TaskQueue != "synthetic" || o.WorkflowIDReusePolicy != enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE || w != librarysync.WriteWorkflowName || len(args) != 1 || args[0] != (librarysync.WriteInput{OperationID: op}) {
			t.Fatal("closed sync retry lost stable identity or retry policy")
		}
		return syncDispatchRun{id: "22222222-2222-4222-8222-222222222222"}, nil
	}}
	id, rid, err := (toolkitValidationStarter{temporal: c, taskQueue: "synthetic"}).StartLibrarySync(context.Background(), op)
	if err != nil || id != "library-sync-write-"+op || rid != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("retry dispatch failed: %s %s %v", id, rid, err)
	}
	if !called {
		t.Fatal("workflow was not dispatched")
	}
}

// TestLibrarySyncRunningOperationJoinsExistingRun prevents a second in-flight run for the same outbox operation.
// Inputs: already-started response; outputs: existing workflow/run IDs. Effects: none; choose alongside closed retry proof.
func TestLibrarySyncRunningOperationJoinsExistingRun(t *testing.T) {
	c := syncDispatchClient{execute: func(client.StartWorkflowOptions, interface{}, []interface{}) (client.WorkflowRun, error) {
		return nil, &serviceerror.WorkflowExecutionAlreadyStarted{RunId: "existing-run"}
	}}
	id, rid, err := (toolkitValidationStarter{temporal: c, taskQueue: "synthetic"}).StartLibrarySync(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil || id != "library-sync-write-11111111-1111-4111-8111-111111111111" || rid != "existing-run" {
		t.Fatalf("running join failed: %s %s %v", id, rid, err)
	}
}
