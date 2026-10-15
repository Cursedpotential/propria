package proffer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestCanonicalBatchUnitRootsRejectsOverlapAndSubstitution proves exact package selection before dispatch.
// Inputs: explicit URI variants. Outputs: sorted distinct safe roots or refusal. Effects: none.
func TestCanonicalBatchUnitRootsRejectsOverlapAndSubstitution(t *testing.T) {
	got, err := CanonicalBatchUnitRoots("b2", "fixture", "exports/", []string{"b2://fixture/exports/b/", "b2://fixture/exports/a/"})
	require.NoError(t, err)
	require.Equal(t, []string{"b2://fixture/exports/a/", "b2://fixture/exports/b/"}, got)
	for _, roots := range [][]string{{"b2://fixture/exports/a/", "b2://fixture/exports/a/child/"}, {"b2://fixture/exports/a/", "b2://fixture/exports/%61/"}, {"b2://other/exports/a/"}, {"b2://fixture/elsewhere/a/"}, {"b2://fixture/exports/../"}, {"b2://fixture/exports/a/?versionId=1"}} {
		_, err := CanonicalBatchUnitRoots("b2", "fixture", "exports/", roots)
		require.Error(t, err)
	}
}

// TestBatchUnitsStartOnePinnedChildAndContinueAfterAmbiguity proves package-level counts and attachment nonfanout.
// Inputs: three selected units, one failed freeze and two valid four-member inventories. Outputs: two child jobs and all three outcomes.
// Effects: Temporal SDK simulation only; no files, database or provider calls.
func TestBatchUnitsStartOnePinnedChildAndContinueAfterAmbiguity(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	registerBatchActivityStubs(env)
	env.RegisterActivityWithOptions(func(context.Context, BatchUnitRequest) (BatchUnitResult, error) { return BatchUnitResult{}, nil }, activity.RegisterOptions{Name: FreezeBatchUnitActivityName})
	in := batchInput()
	in.Prefix = "exports/"
	in.DeclaredFormat = "unknown_binary"
	in.UnitRoots = []string{"b2://bucket/exports/a/", "b2://bucket/exports/b/", "b2://bucket/exports/c/"}
	env.OnActivity(FreezeBatchUnitActivityName, mock.Anything, mock.Anything).Return(func(_ context.Context, req BatchUnitRequest) (BatchUnitResult, error) {
		if req.UnitRoot == in.UnitRoots[1] {
			return BatchUnitResult{}, temporal.NewNonRetryableApplicationError("ambiguous entrypoint", "inventory", nil)
		}
		return BatchUnitResult{UnitRoot: req.UnitRoot, UnitManifestRef: Ref("file:///fixture/inventory.json"), ManifestSHA256: "frozen-sha", MemberCount: 4, PrimaryRef: Ref(req.UnitRoot + "index.html?versionId=immutable"), SelectedFormat: "imessage_html"}, nil
	})
	mockEmptyPriorImports(env)
	mockBindings(env)
	started := []WorkflowInput{}
	env.OnWorkflow(ProfferWorkflow, mock.Anything, mock.Anything).Return(func(_ workflow.Context, child WorkflowInput) (WorkflowResult, error) {
		started = append(started, child)
		return WorkflowResult{SourceVersionRef: "v", Status: StatusSuccess}, nil
	})
	status := runBatch(t, env, in)
	require.Equal(t, 3, status.Counts.Total)
	require.Equal(t, 2, status.Counts.Done)
	require.Equal(t, 1, status.Counts.Failed)
	require.Len(t, started, 2)
	require.Nil(t, status.Items[1].MemberCount)
	for _, child := range started {
		require.Equal(t, "imessage_html", child.DeclaredFormat)
		require.Contains(t, string(child.SourceRef), "versionId=immutable")
		require.Equal(t, 4, child.UnitMemberCount)
		require.NotEmpty(t, child.UnitManifestRef)
		require.Empty(t, child.PackageRef)
	}
}
