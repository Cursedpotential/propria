package temporal

import (
	"fmt"
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"

	"github.com/Cursedpotential/probata/engine/proffer"
)

func TestAlreadyStartedRunIDRecoversExistingExecution(t *testing.T) {
	runID, ok := alreadyStartedRunID(fmt.Errorf("wrapped: %w", &serviceerror.WorkflowExecutionAlreadyStarted{RunId: "run-existing"}))
	if !ok || runID != "run-existing" {
		t.Fatalf("runID=%q ok=%v", runID, ok)
	}
}

// A terminated run's query still reports its last recorded stage as running;
// Temporal's execution status decides. Byline: Claude Code · Opus 5.5 · 2026-10-02
func TestClosedExecutionStateFollowsTemporalStatus(t *testing.T) {
	running := proffer.OperationState{Lifecycle: proffer.OperationRunning, Wait: proffer.OperationWaitRepairDecision}
	cases := []struct {
		status enumspb.WorkflowExecutionStatus
		want   proffer.OperationLifecycle
		term   bool
	}{
		{enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, proffer.OperationRunning, false},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, proffer.OperationCancelled, true},
		{enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, proffer.OperationCancelled, true},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, proffer.OperationFailed, true},
		{enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, proffer.OperationFailed, true},
		{enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, proffer.OperationCompleted, true},
	}
	for _, c := range cases {
		got := closedExecutionState(running, c.status)
		if got.Lifecycle != c.want || got.Terminal != c.term {
			t.Fatalf("%v: lifecycle %q terminal %v", c.status, got.Lifecycle, got.Terminal)
		}
		if c.term && (got.Wait != "" || got.Reason == "") {
			t.Fatalf("%v: a closed run keeps wait %q / reason %q", c.status, got.Wait, got.Reason)
		}
	}
	if got := closedExecutionState(running, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING); got.Wait != proffer.OperationWaitRepairDecision {
		t.Fatal("an open run's state changed")
	}
}
