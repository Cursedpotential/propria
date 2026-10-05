// Byline: Codex · GPT-5 · 2026-10-05
package runtimeapi

import (
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestEveryCasePostRejectsDevBeforeStoreInvocation(t *testing.T) {
	for _, pattern := range CaseIdentityRoutePatterns {
		if !strings.HasPrefix(pattern, "POST ") {
			continue
		}
		t.Run(pattern, func(t *testing.T) {
			store, routes := newCaseIdentityHandler(t)
			path := strings.ReplaceAll(strings.TrimPrefix(pattern, "POST "), "{alias_id}", caseTestPerson)
			path = strings.ReplaceAll(path, "{person_id}", caseTestPerson)
			req := newPreviewRequest(http.MethodPost, path+"?mode=DEV", []byte("{}"))
			req.Header.Set("Idempotency-Key", "guard-proof")
			response := servePreviewRequest(routes, req)
			require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
			require.Zero(t, store.actor)
			require.Zero(t, store.readCalls)
		})
	}
}

func TestDevOrUnapprovedStartHasZeroDispatchAndBindings(t *testing.T) {
	for _, mode := range []string{"DEV", "TEST", "invalid"} {
		handler, store, workflow := previewTestHandler(t)
		payload := map[string]string{"request_id": "guard-proof", "matter_id": caseidentity.AuthoritativeMatterID, "court_case_id": caseidentity.AuthoritativeCourtCaseID, "source_ref": "upload://" + strings.Repeat("a", 64), "declared_format": "sms_xml", "parser_options_ref": "options-1", "operating_mode": mode}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", body)
		require.GreaterOrEqual(t, response.Code, 400)
		require.Empty(t, workflow.started.RequestID)
		require.Empty(t, store.entries)
	}
	handler, store, workflow := previewTestHandler(t)
	payload := `{"request_id":"guard-proof","matter_id":"deadbeef-dead-beef-dead-beefdeadbeef","court_case_id":"cafebabe-cafe-babe-cafe-babecafebabe","source_ref":"upload://` + strings.Repeat("a", 64) + `","declared_format":"sms_xml","parser_options_ref":"options-1","operating_mode":"LIVE"}`
	require.Equal(t, http.StatusUnprocessableEntity, servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(payload)).Code)
	require.Empty(t, workflow.started.RequestID)
	require.Empty(t, store.entries)
}

func TestRecoveryPreservesModeAndUnknownHandleCannotMutate(t *testing.T) {
	handler, store, workflow := previewTestHandler(t)
	handle := startPreview(t, handler)
	response := servePreview(handler.Routes(), http.MethodGet, "/reference-import/operations/"+handle, nil)
	var detail OperationDetail
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &detail))
	require.Equal(t, "LIVE", detail.OperatingMode)
	require.Equal(t, "LIVE", workflow.started.OperatingMode)
	for _, mode := range []string{"", "DEV", "invalid"} {
		store.mu.Lock()
		store.entries[handle].binding.OperatingMode = mode
		store.mu.Unlock()
		for _, suffix := range []string{"decision", "repair-decision", "handler-selection", "cancel"} {
			denied := servePreview(handler.Routes(), http.MethodPost, "/reference-import/previews/"+handle+"/"+suffix, []byte("{}"))
			require.Equal(t, http.StatusConflict, denied.Code, denied.Body.String())
		}
	}
	require.Empty(t, workflow.cancelled)
	require.Empty(t, workflow.decision.Decider)
}
