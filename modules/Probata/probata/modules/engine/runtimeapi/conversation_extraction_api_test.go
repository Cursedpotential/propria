// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package runtimeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

const conversationMatter = "11111111-1111-4111-8111-111111111111"

type conversationRecorder struct {
	extractions []flow.RequestInput
	sends       []flow.SendInput
}

func (c *conversationRecorder) StartConversationExtraction(_ context.Context, in flow.RequestInput) (flow.Started, error) {
	c.extractions = append(c.extractions, in)
	return flow.Started{WorkflowID: flow.ConversationExtractionWorkflowID(in.RequestID), RunID: "run"}, nil
}

func (c *conversationRecorder) StartSend(_ context.Context, in flow.SendInput) (flow.Started, error) {
	c.sends = append(c.sends, in)
	return flow.Started{WorkflowID: flow.SendWorkflowID(in.RequestID), RunID: "run"}, nil
}

func (c *conversationRecorder) Status(context.Context, string) (flow.Progress, error) {
	return flow.Progress{Outcome: flow.OutcomeRunning}, nil
}

func conversationHandler(t *testing.T) (*ConversationExtractionHTTPHandler, *conversationRecorder) {
	t.Helper()
	recorder := &conversationRecorder{}
	handler, err := NewConversationExtractionHTTPHandler(recorder, serviceTokenPath(t))
	require.NoError(t, err)
	return handler, recorder
}

func conversationBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"matter_id":     conversationMatter,
		"conversations": []map[string]string{{"export_key": "exports/sms-001.xml", "conv": "8105550101"}},
	}
	for key, value := range extra {
		body[key] = value
	}
	return body
}

// TestConversationRoutesRequireTheServiceToken proves the tailnet boundary.
func TestConversationRoutesRequireTheServiceToken(t *testing.T) {
	handler, _ := conversationHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/reference-import/extractors", nil)
	req.RemoteAddr = "100.64.1.9:3456"
	recorder := httptest.NewRecorder()
	handler.Routes().ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// TestExtractorsListsEveryRegisteredExtractorAndTheDefault proves the picker's source.
func TestExtractorsListsEveryRegisteredExtractorAndTheDefault(t *testing.T) {
	handler, _ := conversationHandler(t)
	response := servePreview(handler.Routes(), http.MethodGet, "/reference-import/extractors", nil)
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Extractors []struct {
			ID      string `json:"id"`
			Default bool   `json:"default"`
			Kind    string `json:"kind"`
		} `json:"extractors"`
		Default string `json:"default"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, flow.DefaultExtractorID, body.Default)
	require.Len(t, body.Extractors, len(flow.Registry))
	ids := []string{}
	for _, extractor := range body.Extractors {
		ids = append(ids, extractor.ID)
	}
	require.Contains(t, ids, "semantica")
	require.Contains(t, ids, "langextract")
}

// TestExtractStartsOneWorkflowPerClickWithTheChosenExtractors proves the idempotency key and the selection reach the workflow.
func TestExtractStartsOneWorkflowPerClickWithTheChosenExtractors(t *testing.T) {
	handler, workflows := conversationHandler(t)
	body := conversationBody(map[string]any{"extractors": []string{flow.DefaultExtractorID, "semantica"}})
	first := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/extract", body, "click-1"))
	again := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/extract", body, "click-1"))
	require.Equal(t, http.StatusAccepted, first.Code, first.Body.String())
	require.Equal(t, http.StatusAccepted, again.Code)
	require.Len(t, workflows.extractions, 2)
	require.Equal(t, workflows.extractions[0].RequestID, workflows.extractions[1].RequestID, "a retried click joins the same workflow")
	require.Equal(t, []string{flow.DefaultExtractorID, "semantica"}, workflows.extractions[0].Extractors)
	require.Equal(t, conversationMatter, workflows.extractions[0].MatterID)
	require.Equal(t, "operator", workflows.extractions[0].Actor.Username)
	other := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/extract", body, "click-2"))
	require.Equal(t, http.StatusAccepted, other.Code)
	require.NotEqual(t, workflows.extractions[0].RequestID, workflows.extractions[2].RequestID)
}

// TestExtractRefusesWhatCannotRun proves bad requests start nothing.
func TestExtractRefusesWhatCannotRun(t *testing.T) {
	handler, workflows := conversationHandler(t)
	post := func(body map[string]any, key string) *httptest.ResponseRecorder {
		return servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/extract", body, key))
	}
	require.Equal(t, http.StatusUnauthorized, post(conversationBody(nil), "").Code, "an Idempotency-Key is required")
	require.Equal(t, http.StatusUnprocessableEntity, post(conversationBody(map[string]any{"extractors": []string{"nope"}}), "k1").Code)
	require.Equal(t, http.StatusUnprocessableEntity, post(conversationBody(map[string]any{"matter_id": "not-a-uuid"}), "k2").Code)
	require.Equal(t, http.StatusUnprocessableEntity, post(conversationBody(map[string]any{"conversations": []map[string]string{}}), "k3").Code)
	require.Empty(t, workflows.extractions)
}

// TestSendToSurrealStartsAWorkflowAndIncludesExtractionsByDefault proves the send route.
func TestSendToSurrealStartsAWorkflowAndIncludesExtractionsByDefault(t *testing.T) {
	handler, workflows := conversationHandler(t)
	response := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/send-to-surreal", conversationBody(nil), "send-1"))
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	require.Len(t, workflows.sends, 1)
	require.True(t, workflows.sends[0].IncludeExtractions)
	off := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/conversations/send-to-surreal", conversationBody(map[string]any{"include_extractions": false}), "send-2"))
	require.Equal(t, http.StatusAccepted, off.Code)
	require.False(t, workflows.sends[1].IncludeExtractions)
}

// TestConversationStatusIsScopedToConversationWorkflows proves the status route reads only the workflows it starts.
func TestConversationStatusIsScopedToConversationWorkflows(t *testing.T) {
	handler, _ := conversationHandler(t)
	for _, id := range []string{"conversation-extraction:abc", "send-to-surreal:abc", "auto-extraction:handle"} {
		require.Equal(t, http.StatusOK, servePreview(handler.Routes(), http.MethodGet, "/reference-import/conversations/workflows/"+id, nil).Code, id)
	}
	require.Equal(t, http.StatusNotFound, servePreview(handler.Routes(), http.MethodGet, "/reference-import/conversations/workflows/proffer-workflow-123", nil).Code)
}
