// Byline: Claude Code · Opus 5.5 · 2026-09-25

package profferworker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
)

func TestRegisterExtractionInstallsBothWorkflowsAndEveryActivityOnce(t *testing.T) {
	recorder := &registrationRecorder{}
	RegisterExtraction(recorder, activities.EntityExtractionActivities{})
	if strings.Join(recorder.workflowNames, ",") != flow.ExtractionWorkflowName+","+flow.CommitWorkflowName {
		t.Fatalf("workflows = %v", recorder.workflowNames)
	}
	want := []string{
		flow.ProposeRulesActivity, flow.ExtractModelActivity, flow.ReconcileActivity, flow.ValidateCommitActivity,
		flow.CommitEntitiesActivity, flow.CommitAliasesActivity, flow.CommitMentionsActivity, flow.CommitEventsActivity,
		flow.CommitMembersActivity, flow.FinalizeCommitActivity,
	}
	if strings.Join(recorder.names, ",") != strings.Join(want, ",") {
		t.Fatalf("activities = %v", recorder.names)
	}
	// build_timeline_generation_activity belongs to the Python worker's queue.
	for _, name := range recorder.names {
		if name == flow.BuildProjectionActivity {
			t.Fatal("the projection activity must not be registered on the proffer worker")
		}
	}
}

func TestBuildExtractionModelConfiguration(t *testing.T) {
	t.Setenv(model.EnvAPIKeyFile, "")
	acts, err := buildExtraction(nil)
	if err == nil {
		t.Fatalf("a nil database must fail, got %+v", acts)
	}
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "nvidia-api-key")
	if err := os.WriteFile(keyFile, []byte("nvapi-test-key-value-000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(model.EnvAPIKeyFile, keyFile)
	t.Setenv(model.EnvModelID, "z-ai/glm-5.1")
	if _, err := model.ConfigFromEnv(); err == nil || !strings.Contains(err.Error(), "GLM") {
		t.Fatalf("a GLM model must fail the worker's boot: %v", err)
	}
	t.Setenv(model.EnvModelID, "")
	cfg, err := model.ConfigFromEnv()
	if err != nil || cfg.ModelID != model.DefaultModelID || cfg.BaseURL != model.DefaultBaseURL {
		t.Fatalf("defaults: %+v %v", cfg, err)
	}
	t.Setenv(model.EnvAPIKeyFile, filepath.Join(dir, "absent"))
	if _, err := model.ConfigFromEnv(); err != model.ErrDisabled {
		t.Fatalf("an unmounted key disables the model instead of failing: %v", err)
	}
}
