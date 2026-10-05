package profferworker

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/contacts"
	"github.com/Cursedpotential/probata/engine/dedupe"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/Cursedpotential/probata/engine/superindex"
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
	// +1: the call-log back-fill workflow. Byline: Claude Code · Opus 5.5 · 2026-10-02
	// +2: the conversation-chunk re-chunk and per-message removal workflows. Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	// +1: the contacts import workflow (Claude Code · Sonnet · 2026-10-02).
	// +1: the same-device message dedupe workflow. Byline: Claude Code · Opus 5.5 · 2026-10-02
	// +1: isolated toolkit inventory workflow (Codex, 2026-10-04).
	// +1: selected-text snapshot workflow (Codex, 2026-10-04).
	// +1: ledger comparison workflow (Codex, 2026-10-04).
	// +1: separate recovered-archive preservation workflow (Codex, 2026-10-04).
	// +1: independent synthetic provider probe (Codex, 2026-10-04).
	if recorder.workflowCount != 14 {
		t.Fatalf("workflow registration count = %d, want 14", recorder.workflowCount)
	}
	wantNamed := []string{
		proffer.BatchWorkflowName, proffer.CallLogBackfillWorkflowName, proffer.ConversationChunksBackfillWorkflowName,
		proffer.ConversationChunksRemovalWorkflowName, dedupe.WorkflowName, superindex.WorkflowName, repairplan.WorkflowName, contacts.WorkflowName, activities.ToolkitPackageInventoryWorkflowName, activities.ToolkitSelectedTextWorkflowName, activities.ToolkitLedgerComparisonWorkflowName,
		activities.ToolkitPackagePreservationWorkflowName,
		activities.ToolkitPackageConditionalWriteProbeWorkflowName,
	}
	if !reflect.DeepEqual(recorder.workflowNames, wantNamed) {
		t.Fatalf("named workflow registrations = %v, want %v", recorder.workflowNames, wantNamed)
	}
	const replayAliasCount = 3
	// 7 = 2 structured-ELT + derive_sms_threads + publish_context_search + 3
	// handler/flow standalones, plus the 4 batch-by-folder Activities and the 5
	// repair-plan Activities. (publish_context_search: Claude Code · Opus 5.5 · 2026-10-01)
	// +4: the first-party context stages (D04). Byline: Claude Code · Opus 5.5 · 2026-10-01
	// +1: resolve_context_participants (2026-10-02).
	// +1: record_auto_approval (Claude Code · Opus 5.5 · 2026-10-02).
	// +1: commit_call_log (Claude Code · Opus 5.5 · 2026-10-02).
	// +1: match_message_occurrences (Claude Code · Opus 5.5 · 2026-10-02).
	// +6: the contacts import Activities (Claude Code · Sonnet · 2026-10-02).
	// +7: the message dedupe plan and its six steps (Claude Code · Opus 5.5 · 2026-10-02).
	// +1: native toolkit inventory Activity (Codex, 2026-10-04).
	// +1: selected-text snapshot Activity (Codex, 2026-10-04).
	// +1: ledger comparison Activity (Codex, 2026-10-04).
	// +2: preservation copy and independent verification (Codex, 2026-10-04).
	// +1: synthetic provider probe (Codex, 2026-10-04).
	const standaloneActivityCount = 35
	const batchActivityCount = 4
	repairActivityCount := len(stagegraph.RepairPlanActivities)
	if len(recorder.names) != len(stagegraph.Stages)+replayAliasCount+standaloneActivityCount+batchActivityCount+repairActivityCount || len(stagegraph.Stages) != 26 || repairActivityCount != 5 {
		t.Fatalf("activity registration count = %d, want 26 canonical + 3 replay aliases + 35 standalone + 4 batch + 5 repair-plan activities", len(recorder.names))
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
	if registered[activities.ToolkitPackageInventoryActivityName] != 1 {
		t.Fatal("native toolkit inventory Activity must be registered exactly once")
	}
	if registered[activities.ToolkitSelectedTextActivityName] != 1 {
		t.Fatal("selected-text snapshot Activity must be registered exactly once")
	}
	if registered[activities.ToolkitLedgerComparisonActivityName] != 1 {
		t.Fatal("ledger comparison Activity must be registered exactly once")
	}
	for _, name := range []string{activities.ToolkitPackagePreservationCopyActivityName, activities.ToolkitPackagePreservationVerifyActivityName, activities.ToolkitPackageConditionalWriteProbeActivityName} {
		if registered[name] != 1 {
			t.Fatalf("preservation Activity %q must be registered exactly once", name)
		}
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
	if registered[string(stagegraph.PublishContextSearch)] != 1 {
		t.Errorf("weaviate-first activity %q registered %d times", stagegraph.PublishContextSearch, registered[string(stagegraph.PublishContextSearch)])
	}
	// The four first-party context stages (D04). Byline: Claude Code · Opus 5.5 · 2026-10-01
	for _, id := range []stagegraph.StageID{
		stagegraph.ResolveContextParticipants,
		stagegraph.ProposeFirstPartyContext, stagegraph.ConfirmFirstPartyContext,
		stagegraph.CommitFirstPartyMessages, stagegraph.CommitFirstPartyContextThreads,
	} {
		if registered[string(id)] != 1 {
			t.Errorf("first-party context activity %q registered %d times", id, registered[string(id)])
		}
	}
	if registered[proffer.RecommendHandlerActivityName] != 1 || registered[proffer.ValidateHandlerSelectionActivityName] != 1 {
		t.Errorf("handler recommendation/validation activities were not registered exactly once: %#v", registered)
	}
	if registered["run_n8n_flow_activity"] != 1 {
		t.Errorf("generic n8n flow activity registered %d times", registered["run_n8n_flow_activity"])
	}
	// The conversation-chunk Activities run on the Python worker's queue (server/temporal/worker.py); the Go worker
	// registers none of them, so the total above is unchanged and the Python worker alone polls them.
	// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	for _, name := range []string{
		proffer.ChunkContextThreadsActivityName, proffer.PublishContextChunksActivityName, proffer.PublishCallLogFilesActivityName,
		proffer.ListContextThreadsActivityName, proffer.EstimateContextChunksActivityName,
		proffer.VerifyChunkCoverageActivityName, proffer.RemovePerMessageObjectsActivityName,
	} {
		if registered[name] != 0 {
			t.Errorf("python-queue activity %q is registered on the Go worker %d times; it belongs to the Python worker", name, registered[name])
		}
	}
	for name, count := range registered {
		if count != 1 {
			t.Errorf("activity name %q registered %d times", name, count)
		}
	}
}

// TestAIWorkproductRegistry verifies the optional placement group has one workflow and three unique Activities.
// Inputs: disabled/enabled worker registrations. Outputs: exact registry assertions. Effects: no I/O or source writes.
// Choose for integration coverage; byte and storage behavior belongs to the Activity tests.
// Byline: Codex · GPT-6 · 2026-10-05.
func TestAIWorkproductRegistry(t *testing.T) {
	base := &registrationRecorder{}
	RegisterAll(base, Registrations{})
	enabled := &registrationRecorder{}
	RegisterAll(enabled, Registrations{AIWorkproductPlacement: activities.NewAIWorkproductPlacementActivities("/data/proffer/derive-scratch", nil)})
	if enabled.workflowCount != base.workflowCount+1 || len(enabled.names) != len(base.names)+3 {
		t.Fatalf("AI placement registration changed unexpected counts: workflows %d/%d, Activities %d/%d", enabled.workflowCount, base.workflowCount, len(enabled.names), len(base.names))
	}
	for _, name := range []string{activities.AIWorkproductInspectActivityName, activities.AIWorkproductCopyActivityName, activities.AIWorkproductReadbackActivityName} {
		count := 0
		for _, registered := range enabled.names {
			if registered == name {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("AI placement Activity %q registered %d times", name, count)
		}
		for _, registered := range base.names {
			if registered == name {
				t.Fatalf("disabled group registered %q", name)
			}
		}
	}
	count := 0
	for _, name := range enabled.workflowNames {
		if name == activities.AIWorkproductPlacementWorkflowName {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("AI placement workflow registered %d times", count)
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
