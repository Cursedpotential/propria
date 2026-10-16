package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

// legacyAIBatchStub counts workflow starts without performing external work.
// Inputs: batch request. Outputs: empty test identifiers. Effects: test counter only; choose for admission regressions.
type legacyAIBatchStub struct{ starts int }

// StartBatch records dispatch attempts in the admission fixture without external effects.
// Inputs: ignored request. Outputs: empty IDs. Effects: counter increment; choose only for start refusal tests.
func (s *legacyAIBatchStub) StartBatch(context.Context, proffer.BatchInput) (string, string, error) {
	s.starts++
	return "", "", nil
}
// BatchStatus satisfies the fixture boundary with no runtime lookup.
// Inputs: ignored ID. Outputs: empty status. Effects: none; choose only for admission fixtures.
func (s *legacyAIBatchStub) BatchStatus(context.Context, string) (proffer.BatchStatus, error) {
	return proffer.BatchStatus{}, nil
}

// TestLegacyStartsRejectKnownAIBeforeWorkflowOrBinding protects the native AI route at both legacy starts.
// Inputs: recognized AI declarations. Outputs: 409 with the supported route. Effects: no workflow/binding writes.
func TestLegacyStartsRejectKnownAIBeforeWorkflowOrBinding(t *testing.T) {
	formats := []string{"chatgpt", "claude", "gemini_markdown", "chatgpt_official_json", "chatgpt_json_array", "chatgpt_conversations_json", "claude_conversations_json", "claude_ai_export_json", "gemini_activity_json", "ai_markdown_transcript", "ai_generic_json", "ai_conversations_json", "ai_chat_file", "chatgpt_markdown", "claude_markdown"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			handler, store, workflow := previewTestHandler(t)
			batch := &legacyAIBatchStub{}
			require.NoError(t, handler.UseBatchWorkflow(batch))
			body, err := json.Marshal(map[string]string{"request_id": "ai-source", "source_ref": "upload://" + strings.Repeat("a", 64), "declared_format": format, "parser_options_ref": "options"})
			require.NoError(t, err)
			response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", body)
			require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
			require.Contains(t, response.Body.String(), "/reference-import/context/sources")
			require.Empty(t, workflow.started.RequestID)
			require.Empty(t, store.entries)
			body, err = json.Marshal(map[string]string{"batch_id": strings.Repeat("b", 32), "folder_ref": "b2://source/folder", "declared_format": format, "parser_options_ref": "options"})
			require.NoError(t, err)
			response = servePreview(handler.Routes(), http.MethodPost, "/reference-import/start-batch", body)
			require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
			require.Zero(t, batch.starts)
		})
	}
	for _, format := range []string{"json", "message_export_json", "markdown", "ndjson"} {
		require.False(t, nativeAIFormat(format), "generic format must not infer AI")
	}
}

func TestTerminalStaleGatesNeverPersistDecisions(t *testing.T) {
	for _, lifecycle := range []proffer.OperationLifecycle{proffer.OperationCancelled, proffer.OperationFailed, proffer.OperationCompleted, proffer.OperationRerunRequired} {
		for _, route := range []struct {
			path, body string
			phase      proffer.PreviewPhase
		}{
			{"handler-selection", `{"compatibility_ref":"55555555-5555-5555-5555-555555555555"}`, proffer.PhaseAwaitingHandlerSelection},
			{"repair-decision", `{"approved":true,"apply_repair":false}`, proffer.PhaseAwaitingRepairDecision},
			{"decision", `{"approved":true}`, proffer.PhaseAwaitingDecision},
		} {
			t.Run(string(lifecycle)+"/"+route.path, func(t *testing.T) {
				handler, store, workflow := previewTestHandler(t)
				handle := startPreview(t, handler)
				putValidProjection(t, store, handle)
				workflow.state.Phase = route.phase
				workflow.operation = proffer.OperationState{Lifecycle: lifecycle, Terminal: false}
				order := []string{}
				repairs := &repairWriterStub{order: &order}
				handlers := &handlerDecisionWriterStub{order: &order}
				guarded, err := NewPreviewHTTPHandler(workflow, store, repairs, handlers, bytes.Repeat([]byte("k"), 32), serviceTokenPath(t))
				require.NoError(t, err)
				before := len(store.entries[handle].projections)
				response := servePreview(guarded.Routes(), http.MethodPost, "/reference-import/previews/"+handle+"/"+route.path, []byte(route.body))
				require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
				require.Empty(t, order)
				require.Equal(t, before, len(store.entries[handle].projections))
				require.Empty(t, workflow.handler.DecisionRef)
				require.Empty(t, workflow.repair.DecisionRef)
				require.Empty(t, workflow.decision.Decider)
				visible := servePreview(guarded.Routes(), http.MethodGet, "/reference-import/previews/"+handle, nil)
				require.Equal(t, http.StatusOK, visible.Code)
				require.Contains(t, visible.Body.String(), `"phase":"`+string(lifecycle)+`"`)
				require.NotContains(t, visible.Body.String(), `"phase":"`+string(route.phase)+`"`)
			})
		}
	}
}

func TestUnavailableOperationNeverPersistsAStaleGate(t *testing.T) {
	handler, _, workflow := previewTestHandler(t)
	handle := startPreview(t, handler)
	workflow.operation = proffer.OperationState{Lifecycle: proffer.OperationUnavailable}
	response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/previews/"+handle+"/handler-selection", []byte(`{}`))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.True(t, strings.Contains(response.Body.String(), "no decision was recorded"))
}
