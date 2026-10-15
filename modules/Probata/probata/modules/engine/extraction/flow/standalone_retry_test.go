package flow

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/mock"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"testing"
)

// TestStandaloneEntityModelRetryPreservesRulesAndReconcile verifies real Temporal Activity retries without rerunning successful siblings.
// Inputs: admitted standalone Entity workflow with transient or exhausted model failure. Output: three actual attempts and preserved rule IDs.
// Effects: SDK environment only; choose to prove standalone execution and composed shared Activity semantics.
func TestStandaloneEntityModelRetryPreservesRulesAndReconcile(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		t.Run(map[bool]string{false: "transient", true: "exhausted"}[exhausted], func(t *testing.T) {
			env := extractionEnv(t)
			attempts, rules, reconcile := 0, 0, 0
			in := ExtractionRequest{ExtractionID: "existing-click", Run: RunRef{MatterMode: "LIVE", PreviewHandle: "actual-handle", SourceVersionID: "actual-version", GenerationID: "actual-generation"}, UseModel: true}
			env.OnActivity(ProposeRulesActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, got ExtractionRequest) (ProposeResult, error) {
				rules++
				if got != in {
					t.Fatal("rules pins changed")
				}
				return ProposeResult{ExtractionRunID: "successful-rules"}, nil
			})
			env.OnActivity(ExtractModelActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, got ExtractionRequest) (ProposeResult, error) {
				attempts++
				if got != in {
					t.Fatal("retry changed idempotency/source")
				}
				if exhausted || attempts < 3 {
					return ProposeResult{}, errors.New("transient model failure")
				}
				return ProposeResult{ExtractionRunID: "successful-model"}, nil
			})
			env.OnActivity(ReconcileActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, got ReconcileRequest) (ReconcileResult, error) {
				reconcile++
				if got.RunIDs[0] != "successful-rules" {
					t.Fatal("successful rules discarded")
				}
				return ReconcileResult{}, nil
			})
			env.ExecuteWorkflow(ExtractionWorkflowName, in)
			var out Progress
			if err := env.GetWorkflowResult(&out); err != nil {
				t.Fatal(err)
			}
			if attempts != 3 || rules != 1 || reconcile != 1 {
				t.Fatalf("model%d rules%d reconcile%d", attempts, rules, reconcile)
			}
			want := OutcomeCompleted
			if exhausted {
				want = OutcomeCompletedFlagged
			}
			if out.Outcome != want {
				t.Fatalf("outcome%s want%s", out.Outcome, want)
			}
			if exhausted && out.Steps[1].Status != StepFailed {
				t.Fatal("model exhaustion hidden")
			}
		})
	}
}

// TestStandaloneExternalRetryAndExhaustion verifies the external Activity's actual bounded attempts and no duplicate staging.
// Inputs: standalone Semantica workflow and transient or exhausted failure. Output: two attempts, actual failure and one-or-zero staging writes.
// Effects: SDK environment only; choose for independent external retry regression.
func TestStandaloneExternalRetryAndExhaustion(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		t.Run(map[bool]string{false: "transient", true: "exhausted"}[exhausted], func(t *testing.T) {
			env, _ := requestEnv()
			attempts, stages := 0, 0
			env.OnActivity(BeginExternalRunActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, in ExternalRunStart) (ExternalRunStart, error) { return in, nil })
			env.OnActivity(SemanticaExtractActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, in ExternalPageRequest) (json.RawMessage, error) {
				attempts++
				if in.SourceVersionID != "actual-version" || in.GenerationID != "actual-generation" {
					t.Fatal("retry lost source pin")
				}
				if exhausted || attempts < 2 {
					return nil, errors.New("transient external failure")
				}
				return json.RawMessage(`{}`), nil
			})
			env.OnActivity(StageExternalPageActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, _ StageExternalPage) (StagePageResult, error) {
				stages++
				return StagePageResult{Done: true}, nil
			})
			env.OnActivity(FinishExternalRunActivity, mock.Anything, mock.Anything).Return(nil)
			env.ExecuteWorkflow(ExternalWorkflowName, ExternalRunInput{ExtractionID: "existing-click", Run: RunRef{MatterMode: "LIVE", PreviewHandle: "actual-handle", SourceVersionID: "actual-version", GenerationID: "actual-generation"}, Extractor: "semantica"})
			var out Progress
			if exhausted {
				if env.GetWorkflowError() == nil {
					t.Fatal("fatal external workflow completed successfully")
				}
				var ok bool
				out, ok = extractionFailureProgress(env.GetWorkflowError())
				if !ok {
					t.Fatal("fatal external workflow lost Progress details")
				}
			} else if err := env.GetWorkflowResult(&out); err != nil {
				t.Fatal(err)
			}
			want, stageWant := OutcomeCompleted, 1
			if exhausted {
				want, stageWant = OutcomeFailed, 0
			}
			if out.Outcome != want || attempts != 2 || stages != stageWant {
				t.Fatalf("outcome%s attempts%d stages%d", out.Outcome, attempts, stages)
			}
		})
	}
}

// TestActualPendingAttemptAndFailureDetailsReachStatus verifies Describe metadata rather than inferred policy counts.
// Inputs: actual-shaped pending Activity and typed failure. Output: retrying/attempt/error and complete Progress preservation.
// Effects: none; choose for standalone and child status adapter regressions.
func TestActualPendingAttemptAndFailureDetailsReachStatus(t *testing.T) {
	progress := Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "model", Status: StepRunning}}}
	out := decoratePendingExtractionActivities(progress, []*workflowpb.PendingActivityInfo{{ActivityId: "actual-activity", ActivityType: &commonpb.ActivityType{Name: ExtractModelActivity}, Attempt: 2, LastFailure: &failurepb.Failure{Message: "actual upstream timeout"}}})
	if out.Steps[0].Status != "retrying" || out.Steps[0].Attempt != 2 || out.Steps[0].LastFailure != "actual upstream timeout" {
		t.Fatalf("retry status %+v", out)
	}
	failed := Progress{Outcome: OutcomeFailed, Steps: []StepResult{{Step: "extract", Status: StepFailed, Detail: "actual failure"}}}
	got, ok := extractionFailureProgress(temporal.NewNonRetryableApplicationError("all failed", "AllExtractionChildrenFailed", nil, failed))
	if !ok || got.Steps[0].Detail != "actual failure" || got.Outcome != OutcomeFailed {
		t.Fatalf("lost failure %+v", got)
	}
}

// TestStandaloneFatalEntityExhaustion verifies a fatal rules stage fails Temporal after bounded retries.
// Inputs: admitted independent Entity request. Outputs: three attempts and typed Progress containing the actual failed stage.
// Effects: SDK test only; pick to distinguish fatal failure from a preserved partial model result.
func TestStandaloneFatalEntityExhaustion(t *testing.T) {
	env := extractionEnv(t)
	attempts := 0
	env.OnActivity(ProposeRulesActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, _ ExtractionRequest) (ProposeResult, error) {
		attempts++
		return ProposeResult{}, errors.New("rules unavailable")
	})
	env.ExecuteWorkflow(ExtractionWorkflowName, ExtractionRequest{Run: RunRef{MatterMode: "LIVE"}})
	out, ok := extractionFailureProgress(env.GetWorkflowError())
	if env.GetWorkflowError() == nil || !ok || out.Outcome != OutcomeFailed || attempts != 3 || out.Steps[0].Status != StepFailed {
		t.Fatalf("attempts%d progress%+v error%v", attempts, out, env.GetWorkflowError())
	}
}

// TestStandaloneBindingFailureRetainsProgress verifies missing admission fails before Activities with durable details.
// Inputs: each independent extractor with missing mode. Outputs: Temporal failure and binding-stage Progress.
// Effects: SDK tests only; choose to guard against fabricated LIVE or retrying invalid authority.
func TestStandaloneBindingFailureRetainsProgress(t *testing.T) {
	for _, external := range []bool{false, true} {
		env := extractionEnv(t)
		if external {
			env.ExecuteWorkflow(ExternalExtractionWorkflow, ExternalRunInput{Extractor: "semantica"})
		} else {
			env.ExecuteWorkflow(ExtractionWorkflowName, ExtractionRequest{})
		}
		out, ok := extractionFailureProgress(env.GetWorkflowError())
		if !ok || out.Outcome != OutcomeFailed || len(out.Steps) != 1 || out.Steps[0].Step != "binding" {
			t.Fatalf("external%v progress%+v error%v", external, out, env.GetWorkflowError())
		}
	}
}

// TestStandaloneFatalResultReplayCompatibility preserves the historical completed business-failure shape.
// Inputs: old version marker and exhausted standalone external extractor. Outputs: legacy nil workflow error with failed Progress.
// Effects: SDK test only; choose to verify old histories retain their terminal semantics.
func TestStandaloneFatalResultReplayCompatibility(t *testing.T) {
	env, _ := requestEnv()
	env.OnGetVersion("standalone-extraction-fatal-result-v1", workflow.DefaultVersion, 1).Return(workflow.DefaultVersion)
	env.ExecuteWorkflow(ExternalWorkflowName, ExternalRunInput{Run: RunRef{MatterMode: "LIVE"}, Extractor: "unknown"})
	var out Progress
	if err := env.GetWorkflowResult(&out); err != nil || out.Outcome != OutcomeFailed {
		t.Fatalf("progress%+v error%v", out, err)
	}
}
