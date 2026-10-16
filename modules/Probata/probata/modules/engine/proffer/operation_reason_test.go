package proffer

import (
	"errors"
	"testing"
)

// Regression (owner UX test 2026-10-15): a failed run surfaced the last stage's
// not_applicable note instead of the error that ended it.
func TestLastFailedStageReasonSkipsNotApplicableAndSuccess(t *testing.T) {
	results := []StageResult{
		{Status: StatusFailed, Reason: "earlier failure"},
		{Status: StatusSuccess},
		{Status: StatusNotApplicable, Reason: "source has no applicable source-level metadata"},
	}
	if got := lastFailedStageReason(results); got != "earlier failure" {
		t.Fatalf("got %q", got)
	}
	if got := lastFailedStageReason([]StageResult{{Status: StatusNotApplicable, Reason: "note"}}); got != "" {
		t.Fatalf("a not_applicable note stood in for a failure: %q", got)
	}
}

func TestRecordFailureReasonUsesTheEndingErrorOnlyForFailedRuns(t *testing.T) {
	failed := &run{operation: OperationState{Lifecycle: OperationFailed, Reason: "source has no applicable source-level metadata"}}
	failed.recordFailureReason(errors.New(`resolve registered handler for detected signature "text": no parser adapter declares format "text"`))
	if failed.operation.Reason != `resolve registered handler for detected signature "text": no parser adapter declares format "text"` {
		t.Fatalf("reason = %q", failed.operation.Reason)
	}
	for _, lifecycle := range []OperationLifecycle{OperationCancelled, OperationRerunRequired, OperationCompleted} {
		kept := &run{operation: OperationState{Lifecycle: lifecycle, Reason: "own reason"}}
		kept.recordFailureReason(errors.New("boom"))
		if kept.operation.Reason != "own reason" {
			t.Fatalf("%s reason overwritten: %q", lifecycle, kept.operation.Reason)
		}
	}
	(&run{operation: OperationState{Lifecycle: OperationFailed}}).recordFailureReason(nil)
}
