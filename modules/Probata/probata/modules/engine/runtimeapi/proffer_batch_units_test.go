package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

type unitBatchClient struct{ started []proffer.BatchInput }

func (s *unitBatchClient) StartBatch(_ context.Context, in proffer.BatchInput) (string, string, error) {
	s.started = append(s.started, in)
	return in.BatchID, "run", nil
}
func (s *unitBatchClient) BatchStatus(context.Context, string) (proffer.BatchStatus, error) {
	return proffer.BatchStatus{}, nil
}

// TestStartBatchForwardsExplicitUnitsAndRefusesOverlapBeforeDispatch protects the HTTP package contract.
// Inputs: configured source root and explicit package selections. Outputs: sorted roots or HTTP 422.
// Effects: memory-only starter calls; no tokens, files or live workflows.
func TestStartBatchForwardsExplicitUnitsAndRefusesOverlapBeforeDispatch(t *testing.T) {
	t.Setenv(objectstores.RootsEnv, `[{"id":"fixture","label":"Fixture","url":"b2://fixture/exports/"}]`)
	client := &unitBatchClient{}
	handler := &PreviewHTTPHandler{batch: client}
	req := batchStartRequest{OperatingMode: "LIVE", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c", BatchID: "batch-0000000000000000000000000000000001", FolderRef: "b2://fixture/exports/", DeclaredFormat: "unknown_binary", ParserOptionsRef: "options", UnitRoots: []string{"b2://fixture/exports/b/", "b2://fixture/exports/a/"}}
	invoke := func() int {
		body, err := json.Marshal(req)
		require.NoError(t, err)
		response := httptest.NewRecorder()
		handler.startBatch(response, httptest.NewRequest(http.MethodPost, "/reference-import/start-batch", bytes.NewReader(body)))
		return response.Code
	}
	require.Equal(t, http.StatusCreated, invoke())
	require.Equal(t, []string{"b2://fixture/exports/a/", "b2://fixture/exports/b/"}, client.started[0].UnitRoots)
	req.UnitRoots = []string{"b2://fixture/exports/a/", "b2://fixture/exports/a/child/"}
	require.Equal(t, http.StatusUnprocessableEntity, invoke())
	require.Len(t, client.started, 1)
}
