// Byline: Claude Code · Opus 5.5 · 2026-10-02
package proffer

import (
	"context"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

func TestCallLogBackfillRunsOnlyTheCallLogCommitWithTheRunsOwnRefs(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var got StageRequest
	env.RegisterActivityWithOptions(func(_ context.Context, req StageRequest) (StageResult, error) {
		got = req
		return StageResult{Stage: stagegraph.CommitCallLog, Status: StatusSuccess, Ref: "r", ReceiptRef: "r"}, nil
	}, activity.RegisterOptions{Name: string(stagegraph.CommitCallLog)})
	env.ExecuteWorkflow(CallLogBackfillWorkflow, CallLogBackfillInput{
		RequestID: "req-1", MatterID: "m", CourtCaseID: "c", SourceVersionRef: "sv", NormalizedGenerationRef: "ng",
		PreviewHandle: "ph", ParticipantResolutionRef: "pr", OwnerPersonID: "owner", PerspectivePersonID: "matt",
		DeclaredFormat: "xml",
	})
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("workflow error = %v", env.GetWorkflowError())
	}
	if got.RequestID != "req-1" || got.SourceVersionRef != "sv" || got.Refs["normalized_generation"] != "ng" ||
		got.Refs["preview_handle"] != "ph" || got.Refs["participant_resolution"] != "pr" || got.Refs["perspective_person"] != "matt" {
		t.Fatalf("request = %+v", got)
	}
	env2 := suite.NewTestWorkflowEnvironment()
	env2.ExecuteWorkflow(CallLogBackfillWorkflow, CallLogBackfillInput{RequestID: "req-1"})
	if env2.GetWorkflowError() == nil {
		t.Fatal("a back-fill without its run's references must fail")
	}
}
