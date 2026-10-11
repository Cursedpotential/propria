// Byline: Codex · GPT-5 · 2026-10-05
package proffer

import (
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

func TestDirectWorkflowRejectsUnknownDevOrWrongIdentityBeforeActivities(t *testing.T) {
	for _, mode := range []string{"", "DEV", "invalid", "LIVE"} {
		t.Run("single-"+mode, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			input := testInput()
			input.OperatingMode = mode
			if mode == "LIVE" {
				input.MatterID = "11111111-1111-1111-1111-111111111111"
			}
			// No Activities are registered; any dispatch would fail this proof.
			env.ExecuteWorkflow(ProfferWorkflow, input)
			require.Error(t, env.GetWorkflowError())
			require.NotContains(t, env.GetWorkflowError().Error(), "unable to find")
		})
		t.Run("batch-"+mode, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			input := batchInput()
			input.OperatingMode = mode
			if mode == "LIVE" {
				input.CourtCaseID = "cafebabe-cafe-babe-cafe-babecafebabe"
			}
			env.ExecuteWorkflow(BatchWorkflow, input)
			require.Error(t, env.GetWorkflowError())
			require.NotContains(t, env.GetWorkflowError().Error(), "unable to find")
		})
	}
}
