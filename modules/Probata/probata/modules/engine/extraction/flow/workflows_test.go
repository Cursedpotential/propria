// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package flow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
)

func placeholderCommitStep(context.Context, CommitRequest) (CommitStepResult, error) {
	return CommitStepResult{}, errors.New("placeholder ran unmocked")
}

func placeholderFinalize(context.Context, FinalizeRequest) (CommitStepResult, error) {
	return CommitStepResult{}, errors.New("placeholder ran unmocked")
}

func placeholderProjection(context.Context, ProjectionRequest) (ProjectionResult, error) {
	return ProjectionResult{}, errors.New("placeholder ran unmocked")
}

func placeholderPropose(context.Context, ExtractionRequest) (ProposeResult, error) {
	return ProposeResult{}, errors.New("placeholder ran unmocked")
}

func placeholderReconcile(context.Context, ReconcileRequest) (ReconcileResult, error) {
	return ReconcileResult{}, errors.New("placeholder ran unmocked")
}

func commitEnv(t *testing.T) (*testsuite.TestWorkflowEnvironment, *[]string) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(ExtractionCommitWorkflow, workflowOptions(CommitWorkflowName))
	for _, name := range []string{ValidateCommitActivity, CommitEntitiesActivity, CommitAliasesActivity, CommitMentionsActivity, CommitEventsActivity, CommitMembersActivity} {
		env.RegisterActivityWithOptions(placeholderCommitStep, activity.RegisterOptions{Name: name})
	}
	env.RegisterActivityWithOptions(placeholderFinalize, activity.RegisterOptions{Name: FinalizeCommitActivity})
	env.RegisterActivityWithOptions(placeholderProjection, activity.RegisterOptions{Name: BuildProjectionActivity})
	order := &[]string{}
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		*order = append(*order, info.ActivityType.Name)
	})
	return env, order
}

func okReport(digest string) CommitStepResult {
	return CommitStepResult{Report: &commitcheck.Report{OK: true, Digest: digest, Checks: []commitcheck.Check{{Rule: "live_mode", Status: commitcheck.Pass}}}}
}

func commitRequest() CommitRequest {
	return CommitRequest{CommitID: "c-1", Digest: "digest-1", Run: RunRef{PreviewHandle: "h", GenerationID: "g", MatterMode: "LIVE"}, CollectionSlug: "primary"}
}

func TestCommitWorkflowRunsEveryStepInOrderThenProjects(t *testing.T) {
	env, order := commitEnv(t)
	env.OnActivity(ValidateCommitActivity, mock.Anything, mock.Anything).Return(okReport("digest-1"), nil).Once()
	for _, name := range []string{CommitEntitiesActivity, CommitAliasesActivity, CommitMentionsActivity, CommitEventsActivity, CommitMembersActivity} {
		env.OnActivity(name, mock.Anything, mock.Anything).Return(CommitStepResult{Written: 2}, nil).Once()
	}
	var finalize FinalizeRequest
	env.OnActivity(FinalizeCommitActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, request FinalizeRequest) (CommitStepResult, error) {
		finalize = request
		return CommitStepResult{Detail: "receipt"}, nil
	}).Once()
	env.OnActivity(BuildProjectionActivity, mock.Anything, mock.Anything).Return(ProjectionResult{GenerationID: "gen-7", Sequence: 7, MemberCount: 2, Created: true}, nil).Once()
	env.ExecuteWorkflow(CommitWorkflowName, commitRequest())
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("workflow error: %v", env.GetWorkflowError())
	}
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	want := []string{ValidateCommitActivity, CommitEntitiesActivity, CommitAliasesActivity, CommitMentionsActivity, CommitEventsActivity, CommitMembersActivity, FinalizeCommitActivity, BuildProjectionActivity}
	if len(*order) != len(want) {
		t.Fatalf("order = %v", *order)
	}
	for i := range want {
		if (*order)[i] != want[i] {
			t.Fatalf("step %d = %s, want %s (all: %v)", i, (*order)[i], want[i], *order)
		}
	}
	if progress.Outcome != OutcomeCommitted || !finalize.Promote || finalize.Outcome != OutcomeCommitted {
		t.Fatalf("outcome=%s finalize=%+v", progress.Outcome, finalize)
	}
	for _, step := range progress.Steps {
		if step.Status != StepCompleted {
			t.Fatalf("step %s is %s", step.Step, step.Status)
		}
	}
}

func TestCommitWorkflowStopsBeforeWritingWhenValidationFails(t *testing.T) {
	env, order := commitEnv(t)
	failed := CommitStepResult{Report: &commitcheck.Report{OK: false, Digest: "digest-1", Checks: []commitcheck.Check{{Rule: "live_mode", Status: commitcheck.Fail, Reason: "TEST run"}}}}
	env.OnActivity(ValidateCommitActivity, mock.Anything, mock.Anything).Return(failed, nil).Once()
	var finalize FinalizeRequest
	env.OnActivity(FinalizeCommitActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, request FinalizeRequest) (CommitStepResult, error) {
		finalize = request
		return CommitStepResult{}, nil
	}).Once()
	env.ExecuteWorkflow(CommitWorkflowName, commitRequest())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeValidationFailed || finalize.Promote || finalize.Outcome != OutcomeValidationFailed {
		t.Fatalf("outcome=%s finalize=%+v", progress.Outcome, finalize)
	}
	for _, name := range *order {
		if name != ValidateCommitActivity && name != FinalizeCommitActivity {
			t.Fatalf("no write may run after failed validation, ran %s", name)
		}
	}
}

func TestCommitWorkflowRefusesAChangedProposalSet(t *testing.T) {
	env, order := commitEnv(t)
	env.OnActivity(ValidateCommitActivity, mock.Anything, mock.Anything).Return(okReport("someone-edited-it"), nil).Once()
	env.OnActivity(FinalizeCommitActivity, mock.Anything, mock.Anything).Return(CommitStepResult{}, nil).Once()
	env.ExecuteWorkflow(CommitWorkflowName, commitRequest())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeValidationFailed || len(*order) != 2 {
		t.Fatalf("a commit must run exactly the validated set: outcome=%s order=%v", progress.Outcome, *order)
	}
}

func TestCommitWorkflowFailedStepWritesAFailedReceiptAndStops(t *testing.T) {
	env, order := commitEnv(t)
	env.OnActivity(ValidateCommitActivity, mock.Anything, mock.Anything).Return(okReport("digest-1"), nil).Once()
	env.OnActivity(CommitEntitiesActivity, mock.Anything, mock.Anything).Return(CommitStepResult{Written: 3}, nil).Once()
	env.OnActivity(CommitAliasesActivity, mock.Anything, mock.Anything).Return(CommitStepResult{}, errors.New("permission denied for table entity_alias")).Times(3)
	var finalize FinalizeRequest
	env.OnActivity(FinalizeCommitActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, request FinalizeRequest) (CommitStepResult, error) {
		finalize = request
		return CommitStepResult{}, nil
	}).Once()
	env.ExecuteWorkflow(CommitWorkflowName, commitRequest())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeFailed || finalize.FailedAt != "aliases" || finalize.Promote {
		t.Fatalf("outcome=%s finalize=%+v", progress.Outcome, finalize)
	}
	for _, name := range *order {
		if name == CommitMentionsActivity || name == BuildProjectionActivity {
			t.Fatalf("nothing after the failed step may run: %v", *order)
		}
	}
}

func TestProjectionFailureDoesNotUndoTheCommit(t *testing.T) {
	env, _ := commitEnv(t)
	env.OnActivity(ValidateCommitActivity, mock.Anything, mock.Anything).Return(okReport("digest-1"), nil).Once()
	for _, name := range []string{CommitEntitiesActivity, CommitAliasesActivity, CommitMentionsActivity, CommitEventsActivity, CommitMembersActivity} {
		env.OnActivity(name, mock.Anything, mock.Anything).Return(CommitStepResult{Written: 1}, nil).Once()
	}
	env.OnActivity(FinalizeCommitActivity, mock.Anything, mock.Anything).Return(CommitStepResult{}, nil).Once()
	env.OnActivity(BuildProjectionActivity, mock.Anything, mock.Anything).Return(ProjectionResult{}, errors.New("no worker polling evidence-pipeline")).Times(2)
	env.ExecuteWorkflow(CommitWorkflowName, commitRequest())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCommitted {
		t.Fatalf("the commit stands even when the projection cannot run: %s", progress.Outcome)
	}
	last := progress.Steps[len(progress.Steps)-1]
	if last.Step != "projection" || last.Status != StepFailed {
		t.Fatalf("projection must be reported failed: %+v", last)
	}
}

func extractionEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(EntityExtractionWorkflow, workflowOptions(ExtractionWorkflowName))
	env.RegisterActivityWithOptions(placeholderPropose, activity.RegisterOptions{Name: ProposeRulesActivity})
	env.RegisterActivityWithOptions(placeholderPropose, activity.RegisterOptions{Name: ExtractModelActivity})
	env.RegisterActivityWithOptions(placeholderReconcile, activity.RegisterOptions{Name: ReconcileActivity})
	return env
}

func TestExtractionFlagsInvalidModelBatchesButStillReconciles(t *testing.T) {
	env := extractionEnv(t)
	env.OnActivity(ProposeRulesActivity, mock.Anything, mock.Anything).Return(ProposeResult{ExtractionRunID: "run-rules", Proposals: 2, Messages: 927}, nil).Once()
	env.OnActivity(ExtractModelActivity, mock.Anything, mock.Anything).Return(ProposeResult{
		ExtractionRunID: "run-model", Proposals: 9, Events: 4, Batches: 3,
		InvalidBatches: []InvalidBatch{{Index: 1, FirstOrdinal: 40, LastOrdinal: 79, Reason: "empty reply"}},
	}, nil).Once()
	var reconcile ReconcileRequest
	env.OnActivity(ReconcileActivity, mock.Anything, mock.Anything).Return(func(_ context.Context, request ReconcileRequest) (ReconcileResult, error) {
		reconcile = request
		return ReconcileResult{Proposals: 7}, nil
	}).Once()
	env.ExecuteWorkflow(ExtractionWorkflowName, ExtractionRequest{ExtractionID: "x", UseModel: true})
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCompletedFlagged {
		t.Fatalf("an invalid batch is a flag, not a failure: %s", progress.Outcome)
	}
	if len(reconcile.RunIDs) != 2 || reconcile.RunIDs[1] != "run-model" {
		t.Fatalf("reconcile must see both runs: %v", reconcile.RunIDs)
	}
	var flagged bool
	for _, step := range progress.Steps {
		if step.Step == "model" && len(step.Flags) == 1 {
			flagged = true
		}
	}
	if !flagged {
		t.Fatalf("the invalid batch must be visible: %+v", progress.Steps)
	}
}

func TestExtractionWithoutModelSkipsIt(t *testing.T) {
	env := extractionEnv(t)
	env.OnActivity(ProposeRulesActivity, mock.Anything, mock.Anything).Return(ProposeResult{ExtractionRunID: "run-rules"}, nil).Once()
	env.OnActivity(ReconcileActivity, mock.Anything, mock.Anything).Return(ReconcileResult{}, nil).Once()
	env.ExecuteWorkflow(ExtractionWorkflowName, ExtractionRequest{ExtractionID: "x", UseModel: false})
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCompleted || progress.Steps[1].Status != StepSkipped {
		t.Fatalf("progress = %+v", progress)
	}
}

func TestDeterministicIDsAreStable(t *testing.T) {
	if DeterministicID("a", "b") != DeterministicID("a", "b") || DeterministicID("a", "b") == DeterministicID("a", "c") {
		t.Fatal("deterministic ids must be stable and distinct")
	}
	if CommitWorkflowID("h", "0123456789abcdef0123456789abcdef") != "extraction-commit:h:0123456789abcdef01234567" {
		t.Fatalf("commit workflow id = %s", CommitWorkflowID("h", "0123456789abcdef0123456789abcdef"))
	}
}
