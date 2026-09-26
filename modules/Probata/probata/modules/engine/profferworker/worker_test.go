package profferworker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

type registrationRecorder struct {
	workflowCount int
	workflowNames []string
	names         []string
}

func (r *registrationRecorder) RegisterWorkflow(interface{}) { r.workflowCount++ }

func (r *registrationRecorder) RegisterWorkflowWithOptions(_ interface{}, options workflow.RegisterOptions) {
	r.workflowCount++
	r.workflowNames = append(r.workflowNames, options.Name)
}

func (r *registrationRecorder) RegisterActivityWithOptions(_ interface{}, options activity.RegisterOptions) {
	r.names = append(r.names, options.Name)
}

func TestRegisterAllRegistersCanonicalStagesAndReplayAliasesExactlyOnce(t *testing.T) {
	recorder := &registrationRecorder{}
	RegisterAll(recorder, Registrations{HandlerSelection: HandlerSelectionActivities{
		Recommend: func(context.Context, proffer.StageRequest) (proffer.HandlerRecommendationResult, error) {
			return proffer.HandlerRecommendationResult{}, nil
		},
		Validate: func(context.Context, proffer.StageRequest) (proffer.HandlerSelectionValidationResult, error) {
			return proffer.HandlerSelectionValidationResult{}, nil
		},
	}})
	// Three workflows: the per-source ProfferWorkflow, the batch-by-folder
	// workflow that starts one child run per object, and the repair plan
	// workflow (Byline: Claude Code · Opus 5.5 · 2026-09-25).
	// Byline: Claude Code · Opus 5 · 2026-09-21
	if recorder.workflowCount != 3 {
		t.Fatalf("workflow registration count = %d, want 3", recorder.workflowCount)
	}
	if len(recorder.workflowNames) != 2 || recorder.workflowNames[0] != proffer.BatchWorkflowName || recorder.workflowNames[1] != repairplan.WorkflowName {
		t.Fatalf("named workflow registrations = %v, want %q and %q", recorder.workflowNames, proffer.BatchWorkflowName, repairplan.WorkflowName)
	}
	const replayAliasCount = 3
	// 6 = 2 structured-ELT + derive_sms_threads + 3 handler/flow standalones,
	// plus the 4 batch-by-folder Activities and the 5 repair-plan Activities.
	const standaloneActivityCount = 6
	const batchActivityCount = 4
	repairActivityCount := len(stagegraph.RepairPlanActivities)
	if len(recorder.names) != len(stagegraph.Stages)+replayAliasCount+standaloneActivityCount+batchActivityCount+repairActivityCount || len(stagegraph.Stages) != 26 || repairActivityCount != 5 {
		t.Fatalf("activity registration count = %d, want 26 canonical + 3 replay aliases + 6 standalone + 4 batch + 5 repair-plan activities", len(recorder.names))
	}
	for _, descriptor := range stagegraph.RepairPlanActivities {
		found := 0
		for _, registered := range recorder.names {
			if registered == string(descriptor.ID) {
				found++
			}
		}
		if found != 1 {
			t.Errorf("repair-plan activity %q registered %d times", descriptor.ID, found)
		}
	}
	for _, name := range []string{
		activities.ListBatchFolderActivityName, activities.BindImportOperationActivityName,
		activities.ReadImportOperationActivityName, activities.FindImportBindingsActivityName,
	} {
		found := 0
		for _, registered := range recorder.names {
			if registered == name {
				found++
			}
		}
		if found != 1 {
			t.Errorf("batch activity %q registered %d times", name, found)
		}
	}
	registered := make(map[string]int, len(recorder.names))
	for _, name := range recorder.names {
		registered[name]++
	}
	for _, descriptor := range stagegraph.Stages {
		if registered[string(descriptor.ID)] != 1 {
			t.Errorf("canonical stage %q registered %d times", descriptor.ID, registered[string(descriptor.ID)])
		}
	}
	for _, alias := range []string{"hash_source_activity", "hash_raw_records_activity", "hash_raw_generation_activity"} {
		if registered[alias] != 1 {
			t.Errorf("replay alias %q registered %d times", alias, registered[alias])
		}
	}
	if registered[activities.ExecuteStructuredELTActivityName] != 1 {
		t.Errorf("standalone structured ELT activity %q registered %d times", activities.ExecuteStructuredELTActivityName, registered[activities.ExecuteStructuredELTActivityName])
	}
	if registered[activities.SelectStructuredELTActivityName] != 1 {
		t.Errorf("standalone structured ELT activity %q registered %d times", activities.SelectStructuredELTActivityName, registered[activities.SelectStructuredELTActivityName])
	}
	// The derive route's Activity is a standalone optional-stage body: a new
	// capability gets its own Activity, never a widened existing one.
	// Byline: Claude Code · Opus 5 · 2026-09-20
	if registered[string(stagegraph.DeriveSMSThreads)] != 1 {
		t.Errorf("derive activity %q registered %d times", stagegraph.DeriveSMSThreads, registered[string(stagegraph.DeriveSMSThreads)])
	}
	if registered[proffer.RecommendHandlerActivityName] != 1 || registered[proffer.ValidateHandlerSelectionActivityName] != 1 {
		t.Errorf("handler recommendation/validation activities were not registered exactly once: %#v", registered)
	}
	if registered["run_n8n_flow_activity"] != 1 {
		t.Errorf("generic n8n flow activity registered %d times", registered["run_n8n_flow_activity"])
	}
	for name, count := range registered {
		if count != 1 {
			t.Errorf("activity name %q registered %d times", name, count)
		}
	}
}

func TestLoadConfiguredFlowBindingsAllowsNoExtraFlows(t *testing.T) {
	registry, err := loadConfiguredFlowBindings("")
	if err != nil {
		t.Fatalf("loadConfiguredFlowBindings() error = %v", err)
	}
	if registry.Count() != 0 {
		t.Fatalf("binding count = %d, want 0", registry.Count())
	}
}

func TestLoadConfiguredFlowBindingsFailsClosedWhenConfiguredFileIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	_, err := loadConfiguredFlowBindings(path)
	if err == nil || !strings.Contains(err.Error(), "configured but unavailable") {
		t.Fatalf("loadConfiguredFlowBindings() error = %v, want unavailable file rejection", err)
	}
}

func TestLoadConfiguredFlowBindingsValidatesConfiguredFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bindings.json")
	if err := os.WriteFile(path, []byte(`{"bindings":[{"name":"ocr_page","webhook_path":"proffer/ocr-page"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := loadConfiguredFlowBindings(path)
	if err != nil {
		t.Fatalf("loadConfiguredFlowBindings() error = %v", err)
	}
	if registry.Count() != 1 {
		t.Fatalf("binding count = %d, want 1", registry.Count())
	}

	if err := os.WriteFile(path, []byte(`{"bindings":[{"name":"bad/name","webhook_path":"proffer/ocr-page"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfiguredFlowBindings(path); err == nil || !strings.Contains(err.Error(), "invalid N8N_FLOW_BINDINGS_FILE") {
		t.Fatalf("loadConfiguredFlowBindings() error = %v, want invalid binding rejection", err)
	}
}
