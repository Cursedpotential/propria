package runtimeapi

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

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
