// Byline: Codex · GPT-5 · 2026-10-05 (approved LIVE fixture)
package temporal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// This file proves this package's real N8NActivities correctly plug into
// the real engine/proffer.ProfferWorkflow end to end: the other 21
// stages are mocked (exactly like engine/proffer's own test suite does — this
// package does not own their bodies), but select_parser_activity and
// execute_parser_activity are this package's actual production code,
// exercised against real httptest servers standing in for n8n. That is what
// distinguishes this from engine/proffer/workflow_test.go's own hold tests
// (which mock every stage, including the two n8n-backed ones) and from this
// package's n8n_client_test.go/activities_test.go (which exercise the real
// HTTP client but never inside the real workflow).

// placeholderStageActivity exists only so the TestWorkflowEnvironment has a
// function signature to register the 21 non-parser canon stage names
// against; every test below mocks it via OnActivity before executing the
// workflow (mirrors engine/proffer/workflow_test.go's own placeholderActivity
// pattern, reproduced locally since it is unexported there).
func placeholderStageActivity(_ context.Context, _ proffer.StageRequest) (proffer.StageResult, error) {
	return proffer.StageResult{Status: proffer.StatusFailed, Reason: "integration test: placeholder activity ran unmocked", ReceiptRef: "placeholder-receipt"}, nil
}

func placeholderRecommendHandler(_ context.Context, _ proffer.StageRequest) (proffer.HandlerRecommendationResult, error) {
	return proffer.HandlerRecommendationResult{}, nil
}

func placeholderValidateHandler(_ context.Context, _ proffer.StageRequest) (proffer.HandlerSelectionValidationResult, error) {
	return proffer.HandlerSelectionValidationResult{}, nil
}

func stageStub(id stagegraph.StageID) proffer.StageResult {
	return proffer.StageResult{Status: proffer.StatusSuccess, Ref: proffer.Ref(string(id) + "-ref"), ReceiptRef: proffer.Ref(string(id) + "-receipt")}
}

// registerRealActivities registers this package's real N8NActivities (Client
// pointed at the given fake n8n base URL) under select_parser_activity and
// execute_parser_activity, and every other canon stage against the
// unmocked-is-an-error placeholder, then stubs every non-parser stage to
// succeed via OnActivity.
func registerRealActivities(t *testing.T, env *testsuite.TestWorkflowEnvironment, n8nBaseURL string) {
	t.Helper()
	cfg := testConfig(t, n8nBaseURL)
	cfg.SelectHTTPTimeout = 5 * time.Second
	cfg.ExecuteHTTPTimeout = 5 * time.Second
	client, err := NewN8NClient(cfg)
	if err != nil {
		t.Fatalf("NewN8NClient() error = %v", err)
	}
	acts := N8NActivities{Client: client}
	env.RegisterActivityWithOptions(acts.SelectParser, activity.RegisterOptions{Name: string(stagegraph.SelectParser)})
	env.RegisterActivityWithOptions(acts.ExecuteParser, activity.RegisterOptions{Name: string(stagegraph.ExecuteParser)})
	env.RegisterActivityWithOptions(placeholderRecommendHandler, activity.RegisterOptions{Name: proffer.RecommendHandlerActivityName})
	env.RegisterActivityWithOptions(placeholderValidateHandler, activity.RegisterOptions{Name: proffer.ValidateHandlerSelectionActivityName})
	for _, d := range stagegraph.Stages {
		if d.ID == stagegraph.SelectParser || d.ID == stagegraph.ExecuteParser {
			continue
		}
		env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(d.ID)})
	}
	// The Weaviate-first stage runs on every new history.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(stagegraph.PublishContextSearch)})
	// The first-party context propose stage also runs on every new history; with
	// no messages it is not applicable and confirm/commit are skipped.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(stagegraph.ProposeFirstPartyContext)})
	env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(stagegraph.ResolveContextParticipants)})
	// The conversation-chunk Activities live on the Python worker; a run with nothing to chunk gets empty answers.
	// commit_call_log is likewise optional and has no call record to commit here.
	// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	env.RegisterActivityWithOptions(func(context.Context, proffer.ChunkThreadsRequest) (proffer.ChunkThreadsResult, error) {
		return proffer.ChunkThreadsResult{}, nil
	}, activity.RegisterOptions{Name: proffer.ChunkContextThreadsActivityName})
	env.RegisterActivityWithOptions(func(context.Context, proffer.PublishChunksRequest) (proffer.PublishChunksResult, error) {
		return proffer.PublishChunksResult{}, nil
	}, activity.RegisterOptions{Name: proffer.PublishContextChunksActivityName})
	env.RegisterActivityWithOptions(func(context.Context, proffer.PublishCallLogFilesRequest) (proffer.PublishCallLogFilesResult, error) {
		return proffer.PublishCallLogFilesResult{}, nil
	}, activity.RegisterOptions{Name: proffer.PublishCallLogFilesActivityName})
	env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(stagegraph.CommitCallLog)})
	env.RegisterActivityWithOptions(placeholderStageActivity, activity.RegisterOptions{Name: string(stagegraph.MatchMessageOccurrences)})
	candidate := proffer.HandlerCandidate{HandlerID: "sbv", HandlerVersion: "test", ExecutionPath: proffer.HandlerPathDecoder, CompatibilityRef: "compatibility-ref", Reason: "integration decoder"}
	env.OnActivity(proffer.RecommendHandlerActivityName, mock.Anything, mock.Anything).Return(proffer.HandlerRecommendationResult{
		RecommendationRef: "recommendation-ref", ReceiptRef: "recommendation-receipt", DetectedFormat: "whatsapp_export_json",
		DetectedFormatRef: "format-ref", SignatureRef: "signature-ref", Recommended: candidate,
	}, nil).Once()
	env.OnActivity(proffer.ValidateHandlerSelectionActivityName, mock.Anything, mock.Anything).Return(proffer.HandlerSelectionValidationResult{
		DecisionRef: "handler-decision-ref", ActorRef: "operator-1", ValidationReceipt: "validation-receipt",
		RecommendationRef: "recommendation-ref", DetectedFormat: "whatsapp_export_json", DetectedFormatRef: "format-ref", SignatureRef: "signature-ref", Chosen: candidate,
	}, nil).Once()
	for _, d := range stagegraph.Stages {
		if d.ID == stagegraph.SelectParser || d.ID == stagegraph.ExecuteParser {
			continue
		}
		env.OnActivity(string(d.ID), mock.Anything, mock.Anything).Return(stageStub(d.ID), nil).Once()
	}
	env.OnActivity(string(stagegraph.PublishContextSearch), mock.Anything, mock.Anything).Return(stageStub(stagegraph.PublishContextSearch), nil).Maybe()
	env.OnActivity(string(stagegraph.CommitCallLog), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "call-log-receipt", Reason: "no call records",
	}, nil).Maybe()
	env.OnActivity(string(stagegraph.MatchMessageOccurrences), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "message-match-receipt", Reason: "no message records",
	}, nil).Maybe()
	env.OnActivity(string(stagegraph.ResolveContextParticipants), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "resolve-participants-receipt", Reason: "no message records",
	}, nil).Maybe()
	env.OnActivity(string(stagegraph.ProposeFirstPartyContext), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "propose-first-party-receipt", Reason: "no message records",
	}, nil).Maybe()
	env.OnActivity(string(stagegraph.MatchMessageOccurrences), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "match-up-receipt", Reason: "no message records",
	}, nil).Maybe()
	env.OnActivity(string(stagegraph.CommitCallLog), mock.Anything, mock.Anything).Return(proffer.StageResult{
		Status: proffer.StatusNotApplicable, ReceiptRef: "call-log-receipt", Reason: "no call records",
	}, nil).Maybe()
}

func integrationInput() proffer.WorkflowInput {
	return proffer.WorkflowInput{
		RequestID: "req-1", SourceRef: "acquisition-ref",
		OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		DeclaredFormat: "whatsapp_export_json", ParserOptionsRef: "parser-options-ref",
	}
}

// fakeN8N stands in for the two n8n webhooks this package's Activities call.
// selectCalls/executeCalls count real invocations so tests can assert the
// parser HTTP call never happened on rejection.
type fakeN8N struct {
	server       *httptest.Server
	selectCalls  int32
	executeCalls int32
}

func newFakeN8N(t *testing.T) *fakeN8N {
	t.Helper()
	f := &fakeN8N{}
	mux := http.NewServeMux()
	mux.HandleFunc("/proffer/select-parser-activity", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.selectCalls, 1)
		writeFakeStageResult(w, "select_parser_activity", "selection-ref", "selection-receipt")
	})
	mux.HandleFunc("/proffer/execute-parser-activity", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.executeCalls, 1)
		writeFakeStageResult(w, "execute_parser_activity", "execute-ref", "execute-receipt")
	})
	f.server = httptest.NewServer(mux)
	return f
}

func writeFakeStageResult(w http.ResponseWriter, stage, ref, receiptRef string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"stage": stage, "status": "success", "ref": ref, "receipt_ref": receiptRef,
	})
}

func TestIntegrationApprovedRunsAllStagesAndCallsRealParserHTTP(t *testing.T) {
	n8n := newFakeN8N(t)
	defer n8n.server.Close()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(proffer.ProfferWorkflow)
	registerRealActivities(t, env, n8n.server.URL)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(proffer.RepairDecisionSignalName, proffer.RepairDecision{DecisionRef: "repair-decision-ref"})
		env.SignalWorkflow(proffer.HandlerSelectionDecisionSignalName, proffer.HandlerSelectionDecision{DecisionRef: "handler-decision-ref"})
		env.SignalWorkflow(proffer.PreviewDecisionSignalName, proffer.PreviewDecision{Approved: true, Decider: "operator-1"})
	}, time.Millisecond)

	env.ExecuteWorkflow(proffer.ProfferWorkflow, integrationInput())

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow returned error on the approved path: %v", err)
	}
	var result proffer.WorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("GetWorkflowResult failed: %v", err)
	}
	if result.Status != proffer.StatusSuccess {
		t.Errorf("result.Status = %q, want %q", result.Status, proffer.StatusSuccess)
	}
	if len(result.Stages) != len(stagegraph.Stages)+4 {
		t.Errorf("result.Stages has %d entries, want %d (every stage exactly once, plus resolve_context_participants, publish_context_search, propose_first_party_context and commit_call_log)", len(result.Stages), len(stagegraph.Stages)+4)
	}
	if got := atomic.LoadInt32(&n8n.selectCalls); got != 1 {
		t.Errorf("fake n8n select endpoint called %d times, want exactly 1", got)
	}
	if got := atomic.LoadInt32(&n8n.executeCalls); got != 1 {
		t.Errorf("fake n8n execute endpoint called %d times, want exactly 1 before normalized preview approval", got)
	}
}

func TestIntegrationRejectedNeverCallsRealParserHTTP(t *testing.T) {
	n8n := newFakeN8N(t)
	defer n8n.server.Close()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(proffer.ProfferWorkflow)
	registerRealActivities(t, env, n8n.server.URL)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(proffer.RepairDecisionSignalName, proffer.RepairDecision{DecisionRef: "repair-decision-ref"})
		env.SignalWorkflow(proffer.HandlerSelectionDecisionSignalName, proffer.HandlerSelectionDecision{DecisionRef: "handler-decision-ref"})
		env.SignalWorkflow(proffer.PreviewDecisionSignalName, proffer.PreviewDecision{Approved: false, Reason: "wrong format", Decider: "operator-1"})
	}, time.Millisecond)

	env.ExecuteWorkflow(proffer.ProfferWorkflow, integrationInput())

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("workflow returned nil error after a rejected preview decision; fail-closed requires an error")
	}
	if got := atomic.LoadInt32(&n8n.selectCalls); got != 1 {
		t.Errorf("fake n8n select endpoint called %d times, want exactly 1 (selection must still run before the hold)", got)
	}
	if got := atomic.LoadInt32(&n8n.executeCalls); got != 1 {
		t.Errorf("fake n8n execute endpoint called %d times, want 1 — normalized preview must exist before rejection", got)
	}
}
