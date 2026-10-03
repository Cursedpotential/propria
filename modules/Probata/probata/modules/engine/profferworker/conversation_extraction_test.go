// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package profferworker

import (
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// TestRegisterConversationExtractionInstallsThreeWorkflowsAndEveryActivityOnce proves the names the starter and the workflows use are the ones registered.
func TestRegisterConversationExtractionInstallsThreeWorkflowsAndEveryActivityOnce(t *testing.T) {
	recorder := &registrationRecorder{}
	RegisterConversationExtraction(recorder, activities.ConversationActivities{})
	if strings.Join(recorder.workflowNames, ",") != flow.RequestWorkflowName+","+flow.ExternalWorkflowName+","+flow.SendWorkflowName {
		t.Fatalf("workflows = %v", recorder.workflowNames)
	}
	want := []string{
		flow.ResolveConversationsActivity, flow.BeginExternalRunActivity, flow.StageExternalPageActivity, flow.FinishExternalRunActivity,
		flow.PlanSurrealSendActivity, flow.UpsertConversationActivity, flow.UpsertExtractionsActivity, flow.VerifySurrealSendActivity,
	}
	if strings.Join(recorder.names, ",") != strings.Join(want, ",") {
		t.Fatalf("activities = %v", recorder.names)
	}
	// The external extractors are Python Activities on the evidence-pipeline queue.
	for _, name := range recorder.names {
		if name == flow.SemanticaExtractActivity || name == flow.LangExtractExtractActivity {
			t.Fatalf("%s belongs to the Python worker's queue, not the proffer worker", name)
		}
	}
}

// TestBuildConversationActivitiesWithoutSurrealDisablesOnlyTheSend proves a worker with no Surreal connection still boots.
func TestBuildConversationActivitiesWithoutSurrealDisablesOnlyTheSend(t *testing.T) {
	t.Setenv(surrealsink.EnvURL, "")
	acts, err := buildConversationActivities(nil, nil)
	if err == nil {
		t.Fatalf("a nil database must fail, got %+v", acts)
	}
}
