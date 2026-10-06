// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package proffer

// Byline: Codex · 2026-10-04. SDK history scheduling proof with reference-only synthetic input.
import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

// TestSourceIntegrityWorkflowSchedulesOnlyIntegrityWithActualOperationID proves reference-only independent scheduling.
// Inputs: SDK workflow fixture. Outputs: exact Activity/operation assertions. Effects: memory only;
// choose to prove the base Proffer pipeline is never invoked.
func TestSourceIntegrityWorkflowSchedulesOnlyIntegrityWithActualOperationID(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	calls := 0
	env.RegisterActivityWithOptions(func(ctx context.Context, r StageRequest) (StageResult, error) {
		calls++
		info := activity.GetInfo(ctx)
		if r.RequestID != "origin" || r.SourceVersionRef != "version" || r.Refs["original"] != "object" || r.Refs["integrity_operation"] != Ref(info.WorkflowExecution.ID) || len(r.Refs) != 2 {
			t.Errorf("wrong reference request: %+v", r)
		}
		return StageResult{Stage: stagegraph.AssessSourceIntegrity, Status: StatusSuccess, Ref: "receipt", ReceiptRef: "receipt"}, nil
	}, activity.RegisterOptions{Name: string(stagegraph.AssessSourceIntegrity)})
	env.ExecuteWorkflow(SourceIntegrityWorkflow, SourceIntegrityInput{RequestID: "origin", SourceVersionRef: "version", OriginalRef: "object"})
	var got StageResult
	if err := env.GetWorkflowResult(&got); err != nil || env.GetWorkflowError() != nil || calls != 1 || got.ReceiptRef != "receipt" {
		t.Fatalf("calls=%d result=%+v error=%v", calls, got, env.GetWorkflowError())
	}
	opts := optionsFor(stagegraph.AssessSourceIntegrity)
	if opts.RetryPolicy.MaximumAttempts != 1 || opts.HeartbeatTimeout <= 0 {
		t.Fatal("integrity requires heartbeat and one attempt")
	}
}

// TestSourceIntegrityWorkflowFailureKeepsReceiptAndDoesNotRetry preserves terminal failure/cancellation references.
// Inputs: synthetic Activity outcomes. Outputs: one-attempt and error-detail assertions.
// Effects: memory only; choose instead of a live failure workflow for scheduling proof.
func TestSourceIntegrityWorkflowFailureKeepsReceiptAndDoesNotRetry(t *testing.T) {
	for _, kind := range []string{"incomplete", "unavailable", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			calls := 0
			env.RegisterActivityWithOptions(func(context.Context, StageRequest) (StageResult, error) {
				calls++
				r := StageResult{Stage: stagegraph.AssessSourceIntegrity, Status: StatusFailed, ReceiptRef: "receipt", Reason: "stream_read_failed"}
				switch kind {
				case "unavailable":
					return StageResult{}, errors.New("database unavailable")
				case "cancel":
					return r, temporal.NewCanceledError(r)
				}
				return r, nil
			}, activity.RegisterOptions{Name: string(stagegraph.AssessSourceIntegrity)})
			env.ExecuteWorkflow(SourceIntegrityWorkflow, SourceIntegrityInput{RequestID: "origin", SourceVersionRef: "version", OriginalRef: "object"})
			err := env.GetWorkflowError()
			if err == nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if kind == "incomplete" {
				var app *temporal.ApplicationError
				if !errors.As(err, &app) || app.Type() != "SourceIntegrityIncomplete" {
					t.Fatal(err)
				}
				var r StageResult
				if app.Details(&r) != nil || r.ReceiptRef != "receipt" {
					t.Fatal("failure history lost receipt")
				}
			}
			if kind == "cancel" && !temporal.IsCanceledError(err) {
				t.Fatal(err)
			}
		})
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(SourceIntegrityWorkflow, SourceIntegrityInput{RequestID: "origin"})
	if env.GetWorkflowError() == nil {
		t.Fatal("missing references must fail before scheduling")
	}
}
