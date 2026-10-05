// Byline: Codex · GPT-6-Sol · 2026-10-05.
package proffer

import (
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestAINeutralRoutingUsesRecordedApplicabilityAndKeepsLegacyCommands checks versioned workflow scheduling.
// Inputs: Activity-produced applicability, conflicting incoming labels and historical marker versions. Outputs: assertions.
// Effects: Temporal test history only. Choose alongside Activity/store tests; these synthetic flags are not source proof.
func TestAINeutralRoutingUsesRecordedApplicabilityAndKeepsLegacyCommands(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		verifiedAI, legacy, wantChunks bool
		declared                       string
	}{
		{"verified-AI-request-human", true, false, false, "smsbackuprestore_xml"},
		{"legacy-AI-commands", true, true, true, "chatgpt_official_json"},
		{"request-AI-no-verified-AI", false, false, true, "chatgpt_official_json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(ProfferWorkflow)
			mockAllStagesSucceed(env)
			env.OnActivity(string(stagegraph.ResolveContextParticipants), mock.Anything, mock.Anything).Return(
				StageResult{Stage: stagegraph.ResolveContextParticipants, Status: StatusNotApplicable, ReceiptRef: "AI-resolution-receipt", Reason: "AI roles remain source labels", AIChatSource: tc.verifiedAI}, nil).Once()
			if tc.legacy {
				env.OnGetVersion(aiNeutralRoutingChangeID, workflow.DefaultVersion, aiNeutralRoutingVersion).Return(workflow.DefaultVersion).Once()
			}
			order := newOrderRecorder(env)
			approveHold(env)
			in := testInput()
			in.DeclaredFormat = tc.declared
			env.ExecuteWorkflow(ProfferWorkflow, in)
			if err := env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			for _, activity := range []string{ChunkContextThreadsActivityName, PublishCallLogFilesActivityName} {
				if order.contains(activity) != tc.wantChunks {
					t.Fatalf("%s commands=%v wantChunks=%v", activity, order.snapshot(), tc.wantChunks)
				}
			}
			if !order.contains(string(stagegraph.PublishContextSearch)) || !order.contains(string(stagegraph.PublishPreview)) || order.contains(string(stagegraph.MatchMessageOccurrences)) || order.contains(string(stagegraph.ConfirmFirstPartyContext)) || order.contains(string(stagegraph.CommitFirstPartyMessages)) {
				t.Fatalf("AI flow commands=%v", order.snapshot())
			}
		})
	}
}
