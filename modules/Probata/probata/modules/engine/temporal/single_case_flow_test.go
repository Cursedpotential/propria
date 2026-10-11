// Byline: Codex · GPT-5 · 2026-10-05.
package temporal

import (
	"context"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRegisteredCaseFlowCannotOmitItsOperatingContext(t *testing.T) {
	registry, err := NewFlowRegistry([]FlowBinding{{Name: "case_repair", WebhookPath: "repair/case"}, {Name: "independent", WebhookPath: "maintenance/read"}})
	require.NoError(t, err)
	require.NoError(t, registry.AdmitCaseConnected([]string{"case_repair"}))
	classified, err := registry.Lookup("case_repair")
	require.NoError(t, err)
	require.True(t, classified.RequireCaseScope)
	untouched, err := registry.Lookup("independent")
	require.NoError(t, err)
	require.False(t, untouched.RequireCaseScope)
	acts := FlowActivities{Client: &N8NClient{}, Registry: registry}
	for _, request := range []FlowRequest{
		{Flow: "case_repair", RequestID: "missing-all"},
		{Flow: "case_repair", RequestID: "dev", OperatingMode: "DEV", MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID},
		{Flow: "case_repair", RequestID: "foreign", OperatingMode: "LIVE", MatterID: "11111111-1111-1111-1111-111111111111", CourtCaseID: caseidentity.AuthoritativeCourtCaseID},
	} {
		_, err := acts.RunFlow(context.Background(), request)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "HTTP")
	}
	require.Error(t, registry.AdmitCaseConnected([]string{"undeclared"}))
}
