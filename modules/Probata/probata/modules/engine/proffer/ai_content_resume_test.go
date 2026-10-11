// Byline: Codex / 2026-10-06.
package proffer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// aiResumeFixture supplies synthetic source pins with the existing admitted test case.
// Inputs: none. Outputs: resume request. Effects: none; choose for wire/admission tests.
func aiResumeFixture() AIContentResumeInput {
	in := testInput()
	return AIContentResumeInput{RequestID: "original-ai-import", OperatingMode: in.OperatingMode,
		MatterID: in.MatterID, CourtCaseID: in.CourtCaseID,
		SourceVersionID:        "11111111-1111-4111-8111-111111111111",
		NormalizedGenerationID: "22222222-2222-4222-8222-222222222222",
		VerificationID:         "33333333-3333-4333-8333-333333333333"}
}

// TestAIExtractionCooldownRetriesPreserveLegacyPolicy checks waits do not exhaust the old three-attempt limit.
// Inputs: synthetic provider deferrals under old and new version markers. Outputs: bounded attempt assertions.
// Effects: virtual Temporal history only; choose to protect replay and cooldown continuation without model calls.
// Byline: Codex · GPT-6 · 2026-10-07.
func TestAIExtractionCooldownRetriesPreserveLegacyPolicy(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint("legacy-", legacy), func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			if legacy {
				env.OnGetVersion(aiContentProviderRetryChangeID, workflow.DefaultVersion, 1).Return(workflow.DefaultVersion).Once()
			}
			attempts := 0
			for _, step := range []struct{ name, stage string }{
				{AIPrepareContentActivityName, "prepared"}, {AIExtractWorkProductsActivityName, "work_products"},
				{AIExtractCandidatesActivityName, "candidates"}, {AIEmbedContentActivityName, "embedded"},
				{AIPublishContentActivityName, "published"}, {AIVerifyContentPublicationActivityName, "verified"},
			} {
				stage := step.stage
				env.RegisterActivityWithOptions(func(_ context.Context, req AIContentRequest) (AIContentResult, error) {
					if stage == "candidates" {
						attempts++
						if attempts <= 4 {
							return AIContentResult{}, temporal.NewApplicationError("synthetic provider cooldown; no request dispatched", "AIProviderDeferred")
						}
					}
					return AIContentResult{RequestID: req.RequestID, OperatingMode: req.OperatingMode, MatterID: req.MatterID, CourtCaseID: req.CourtCaseID,
						SourceVersionID: req.SourceVersionID, NormalizedGenerationID: req.NormalizedGenerationID, VerificationID: req.VerificationID,
						Stage: stage, BundleRef: Ref("file:///retained/" + stage + ".json"), Records: 1, Conversations: 1,
						Chunks: 1, ObjectsWritten: 1, ObjectsVerified: 1}, nil
				}, activity.RegisterOptions{Name: step.name})
			}
			env.ExecuteWorkflow(AIContentResumeWorkflow, aiResumeFixture())
			if legacy {
				if attempts != 3 || env.GetWorkflowError() == nil {
					t.Fatalf("legacy retry policy changed: attempts=%d error=%v", attempts, env.GetWorkflowError())
				}
			} else if attempts != 5 || env.GetWorkflowError() != nil {
				t.Fatalf("provider waits exhausted scheduler retries: attempts=%d error=%v", attempts, env.GetWorkflowError())
			}
		})
	}
	options := aiExtractionActivityOptions()
	if options.ScheduleToCloseTimeout != 24*time.Hour || options.StartToCloseTimeout != 2*time.Hour || options.RetryPolicy.MaximumAttempts != 0 || options.RetryPolicy.MaximumInterval != 15*time.Minute {
		t.Fatalf("unbounded or altered extraction deadline: %+v", options)
	}
}

// TestAIContentResumeReusesVerifiedPinsAndVersionedBounds verifies six independent stages without reacquisition.
// Inputs: synthetic receipts at both supported limit versions. Outputs: final pinned summary.
// Effects: test workflow history only; choose to protect resumed source identity and legacy replay limits.
func TestAIContentResumeReusesVerifiedPinsAndVersionedBounds(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint("legacy-", legacy), func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(AIContentResumeWorkflow)
			in := aiResumeFixture()
			records, chunks := 612, 159
			if legacy {
				records, chunks = 132, 24
				env.OnGetVersion(aiContentBoundsChangeID, workflow.DefaultVersion, 1).Return(workflow.DefaultVersion).Once()
			}
			var order []string
			for _, step := range []struct{ name, stage string }{
				{AIPrepareContentActivityName, "prepared"}, {AIExtractWorkProductsActivityName, "work_products"},
				{AIExtractCandidatesActivityName, "candidates"}, {AIEmbedContentActivityName, "embedded"},
				{AIPublishContentActivityName, "published"}, {AIVerifyContentPublicationActivityName, "verified"},
			} {
				stage := step.stage
				env.RegisterActivityWithOptions(func(_ context.Context, req AIContentRequest) (AIContentResult, error) {
					if req.RequestID != in.RequestID || req.SourceVersionID != in.SourceVersionID || req.NormalizedGenerationID != in.NormalizedGenerationID || req.VerificationID != in.VerificationID {
						return AIContentResult{}, fmt.Errorf("resume lost original verified pins")
					}
					if (!legacy && (req.MaxRecords != 1024 || req.MaxChunks != 256 || req.MaxModelCalls != 512)) ||
						(legacy && (req.MaxRecords != 256 || req.MaxChunks != 128 || req.MaxModelCalls != 128)) || req.MaxTextBytes != 2097152 {
						return AIContentResult{}, fmt.Errorf("wrong versioned resource limits")
					}
					order = append(order, stage)
					return AIContentResult{RequestID: req.RequestID, OperatingMode: req.OperatingMode, MatterID: req.MatterID, CourtCaseID: req.CourtCaseID,
						SourceVersionID: req.SourceVersionID, NormalizedGenerationID: req.NormalizedGenerationID, VerificationID: req.VerificationID,
						Stage: stage, BundleRef: Ref("file:///retained/" + stage + ".json"), Records: records, Conversations: 23,
						Chunks: chunks, ObjectsWritten: chunks, ObjectsVerified: chunks, WorkProducts: 2, Candidates: 5}, nil
				}, activity.RegisterOptions{Name: step.name})
			}
			env.ExecuteWorkflow(AIContentResumeWorkflow, in)
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			if fmt.Sprint(order) != "[prepared work_products candidates embedded published verified]" {
				t.Fatalf("unexpected content sequence %v", order)
			}
			var out AIContentSummary
			if err := env.GetWorkflowResult(&out); err != nil || out.Chunks != chunks || out.VerificationRef == "" {
				t.Fatalf("missing verified resume result: %+v %v", out, err)
			}
			state := queryOperation(t, env)
			if state.OperatingMode != in.OperatingMode || !state.Terminal || state.Lifecycle != OperationCompleted || state.CurrentStage != "" || state.Wait != "" || len(state.ActiveStages) != 0 || state.CompletedStageCount != 6 {
				t.Fatalf("invalid completed resume state: %+v", state)
			}
		})
	}
}

// TestAIContentResumeFailureIsTerminal verifies a failed Activity leaves a truthful terminal query.
// Inputs: a nonretryable synthetic preparation failure. Outputs: failed state and one failed stage.
// Effects: test history only; choose to prevent a stopped resume from appearing active.
func TestAIContentResumeFailureIsTerminal(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, AIContentRequest) (AIContentResult, error) {
		return AIContentResult{}, temporal.NewNonRetryableApplicationError("fixture preparation denied", "FixtureDenied", nil)
	}, activity.RegisterOptions{Name: AIPrepareContentActivityName})
	env.ExecuteWorkflow(AIContentResumeWorkflow, aiResumeFixture())
	state := queryOperation(t, env)
	if env.GetWorkflowError() == nil || state.OperatingMode != "LIVE" || !state.Terminal || state.Lifecycle != OperationFailed || state.CurrentStage != "" || state.Wait != "" || len(state.ActiveStages) != 0 || state.CompletedStageCount != 1 {
		t.Fatalf("invalid failed resume state: %+v %v", state, env.GetWorkflowError())
	}
}

// TestAIContentResumeCancellationIsTerminal preserves cancellation rather than labeling it a stage failure.
// Inputs: an operator cancellation while preparation is pending. Outputs: a LIVE cancelled terminal query.
// Effects: synthetic workflow history only; choose to keep the operation surface consistent with Temporal.
func TestAIContentResumeCancellationIsTerminal(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, AIContentRequest) (AIContentResult, error) {
		return AIContentResult{}, nil
	}, activity.RegisterOptions{Name: AIPrepareContentActivityName})
	env.OnActivity(AIPrepareContentActivityName, mock.Anything, mock.Anything).After(time.Hour).Return(AIContentResult{}, nil)
	env.RegisterDelayedCallback(func() { env.CancelWorkflow() }, time.Second)
	env.ExecuteWorkflow(AIContentResumeWorkflow, aiResumeFixture())
	state := queryOperation(t, env)
	if env.GetWorkflowError() == nil || state.OperatingMode != "LIVE" || !state.Terminal || state.Lifecycle != OperationCancelled || len(state.ActiveStages) != 0 {
		t.Fatalf("invalid cancelled resume state: %+v %v", state, env.GetWorkflowError())
	}
}

// TestAIContentResumeRejectsUnadmittedInputs checks that invalid case/source identities schedule no stage.
// Inputs: foreign case, DEV mode, missing request and invalid UUIDs. Outputs: admission failures.
// Effects: test-only history; choose to keep resume from becoming new-source admission.
func TestAIContentResumeRejectsUnadmittedInputs(t *testing.T) {
	for _, field := range []string{"mode", "case", "request", "source", "generation", "verification"} {
		t.Run(field, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			in := aiResumeFixture()
			switch field {
			case "mode":
				in.OperatingMode = "DEV"
			case "case":
				in.CourtCaseID = "44444444-4444-4444-8444-444444444444"
			case "request":
				in.RequestID = ""
			case "source":
				in.SourceVersionID = ""
			case "generation":
				in.NormalizedGenerationID = "00000000-0000-0000-0000-000000000000"
			case "verification":
				in.VerificationID = "not-a-uuid"
			}
			order := newOrderRecorder(env)
			env.ExecuteWorkflow(AIContentResumeWorkflow, in)
			if env.GetWorkflowError() == nil || len(order.snapshot()) != 0 {
				t.Fatalf("invalid admission scheduled content: %v %v", env.GetWorkflowError(), order.snapshot())
			}
		})
	}
}
