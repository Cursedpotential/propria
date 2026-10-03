// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// MessageDedupeWorkflow in the Temporal test environment with recorded step
// Activities. A unit test of the orchestration only — not evidence that any
// row was removed; the live dry run against probata-db is that evidence.

package dedupe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type stepLog struct{ requests []StepRequest }

func (l *stepLog) register(env *testsuite.TestWorkflowEnvironment, refuse Step) {
	for _, step := range append([]Step{StepPlan}, Steps...) {
		step := step
		env.RegisterActivityWithOptions(func(_ context.Context, request StepRequest) (Receipt, error) {
			l.requests = append(l.requests, request)
			if step == refuse {
				return Receipt{}, temporal.NewNonRetryableApplicationError("refused", RefusalType, Refusal{Reason: "test"})
			}
			counts := map[string]int64{}
			if step == StepPlan {
				counts["copies"] = 17082
			}
			return Receipt{ReceiptID: "r-" + string(step), Step: step, DryRun: request.DryRun, Counts: counts}, nil
		}, activity.RegisterOptions{Name: ActivityName(step)})
	}
}

func TestEveryStepRunsInOrderAsItsOwnActivity(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		log := &stepLog{}
		log.register(env, "")
		env.ExecuteWorkflow(MessageDedupeWorkflow, Input{DedupeID: "same-device-20261002", DryRun: dryRun, ExpectedCopies: 17082})
		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
		var result Result
		require.NoError(t, env.GetWorkflowResult(&result))
		require.Equal(t, int64(17082), result.Copies)
		require.Len(t, result.Receipts, 1+len(Steps))
		want := append([]Step{StepPlan}, Steps...)
		for index, request := range log.requests {
			require.Equal(t, want[index], request.Step)
			require.Equal(t, dryRun, request.DryRun)
			require.Equal(t, DefaultBatchSize, request.BatchSize)
			require.Equal(t, int64(17082), request.ExpectedCopies)
			require.Equal(t, "same-device-20261002", request.DedupeID)
		}
	}
}

func TestARefusedStepStopsTheRunWithoutRetrying(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	log := &stepLog{}
	log.register(env, StepRemoveThreadMemberships)
	env.ExecuteWorkflow(MessageDedupeWorkflow, Input{DedupeID: "same-device-20261002"})
	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
	steps := []Step{}
	for _, request := range log.requests {
		steps = append(steps, request.Step)
	}
	require.Equal(t, []Step{StepPlan, StepRepointOccurrences, StepRemoveThreadMemberships}, steps,
		"the refused step runs once and nothing after it runs")
}

func TestABadInputIsRefusedBeforeAnyStep(t *testing.T) {
	for _, in := range []Input{{DedupeID: "short"}, {DedupeID: "same-device-20261002", BatchSize: MaxBatchSize + 1}, {DedupeID: "same-device-20261002", ExpectedCopies: -1}} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		log := &stepLog{}
		log.register(env, "")
		env.ExecuteWorkflow(MessageDedupeWorkflow, in)
		require.Error(t, env.GetWorkflowError(), "%+v", in)
		require.Empty(t, log.requests)
	}
}
