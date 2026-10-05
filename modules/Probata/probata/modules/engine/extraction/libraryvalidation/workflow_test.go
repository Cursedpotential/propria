// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestTemporalFullOperationCarriesOnlyBoundedReferences(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(ctx context.Context, in LibraryValidationInput) (Plan, error) {
		return s.Prepare(ctx, in.ProposalID)
	}, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(s.SourceSnapshot, activity.RegisterOptions{Name: SnapshotActivity})
	env.RegisterActivityWithOptions(s.VerifyClaim, activity.RegisterOptions{Name: VerifyActivity})
	env.RegisterActivityWithOptions(s.Finish, activity.RegisterOptions{Name: ReceiptActivity})
	env.ExecuteWorkflow(ToolkitLibraryValidationWorkflow, LibraryValidationInput{ProposalID: repo.proposal.ID})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var result Result
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, Verified, result.Status)
	require.True(t, result.DatabaseCommitted)
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), fixtureQuote)
	require.NotContains(t, string(raw), "personal_note")
	query, err := env.QueryWorkflow(ProgressQuery)
	require.NoError(t, err)
	var progress Progress
	require.NoError(t, query.Get(&progress))
	require.Equal(t, "complete", progress.Phase)
	require.Equal(t, 1, progress.Completed)
}

func TestTemporalFailedClaimStillCommitsExplicitPartialReceipt(t *testing.T) {
	s, repo, _ := serviceFixture(t)
	repo.proposal.Citations = append(repo.proposal.Citations, repo.proposal.Citations[0])
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(ctx context.Context, in LibraryValidationInput) (Plan, error) {
		return s.Prepare(ctx, in.ProposalID)
	}, activity.RegisterOptions{Name: PrepareActivity})
	env.RegisterActivityWithOptions(func(ctx context.Context, in ClaimInput) (StepResult, error) {
		if in.Index == 1 {
			return StepResult{}, temporal.NewNonRetryableApplicationError("fixture blocked source", "FixtureFetchFailure", nil)
		}
		return s.SourceSnapshot(ctx, in)
	}, activity.RegisterOptions{Name: SnapshotActivity})
	env.RegisterActivityWithOptions(s.VerifyClaim, activity.RegisterOptions{Name: VerifyActivity})
	env.RegisterActivityWithOptions(s.Finish, activity.RegisterOptions{Name: ReceiptActivity})
	env.ExecuteWorkflow(ToolkitLibraryValidationWorkflow, LibraryValidationInput{ProposalID: repo.proposal.ID})
	var blocked *temporal.ApplicationError
	require.ErrorAs(t, env.GetWorkflowError(), &blocked)
	require.Equal(t, "LibraryValidationBlocked", blocked.Type())
	var result Result
	require.NoError(t, blocked.Details(&result))
	require.Equal(t, Partial, result.Status)
	require.Equal(t, "CLAIM_UNPROCESSED", repo.receipt.ClaimChecks[1].FailureCode)
	query, err := env.QueryWorkflow(ProgressQuery)
	require.NoError(t, err)
	var progress Progress
	require.NoError(t, query.Get(&progress))
	require.Equal(t, "validation_blocked", progress.Phase)
}

func TestTemporalInvalidInputAndPreparationFailuresVisible(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.ExecuteWorkflow(ToolkitLibraryValidationWorkflow, LibraryValidationInput{ProposalID: "inline-personal-body"})
	require.Error(t, env.GetWorkflowError())
	env = suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, LibraryValidationInput) (Plan, error) {
		return Plan{}, errors.New("validation configuration missing")
	}, activity.RegisterOptions{Name: PrepareActivity})
	env.ExecuteWorkflow(ToolkitLibraryValidationWorkflow, LibraryValidationInput{ProposalID: "library_proposal:11111111-2222-3333-4444-555555555555"})
	require.ErrorContains(t, env.GetWorkflowError(), "configuration missing")
}
