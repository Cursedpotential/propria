package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

// findingStore is the in-memory store plus the optional duplicate lookup.
type findingStore struct {
	*MemoryPreviewStore
	found []previewmodel.ImportedContent
	err   error
	asked string
}

func (s *findingStore) FindImportedContent(_ context.Context, _ uuid.UUID, sourceRef string) ([]previewmodel.ImportedContent, error) {
	s.asked = sourceRef
	return s.found, s.err
}

func duplicateGuardHandler(t *testing.T, found []previewmodel.ImportedContent, findErr error) (*PreviewHTTPHandler, *findingStore, *previewWorkflowStub) {
	t.Helper()
	store := &findingStore{MemoryPreviewStore: NewMemoryPreviewStore(&countingEntropy{next: 1}), found: found, err: findErr}
	workflow := &previewWorkflowStub{}
	handler, err := NewPreviewHTTPHandler(workflow, store, store, store, bytes.Repeat([]byte("k"), 32), serviceTokenPath(t), sourceContextValidatorStub{})
	require.NoError(t, err)
	return handler, store, workflow
}

const guardStartBody = `{"request_id":"request-dup","matter_id":"01a0f751-e07b-75cc-9ad5-63ad9449a8ba","court_case_id":"01a0f751-e07b-76a1-a738-eb3e3aa3e68c","source_ref":"upload://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","declared_format":"sms_xml","parser_options_ref":"options-1"}`

func TestStartBlocksContentThatIsAlreadyImported(t *testing.T) {
	at := time.Date(2026, 10, 3, 2, 32, 0, 0, time.UTC)
	handler, store, workflow := duplicateGuardHandler(t, []previewmodel.ImportedContent{{WorkflowID: "earlier", SourceKey: "b2://bucket/message_3.json", ImportedAt: at, Committed: true}}, nil)
	response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(guardStartBody))
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	var body struct {
		Detail          string `json:"detail"`
		AlreadyImported struct {
			Status     string    `json:"status"`
			SourceKey  string    `json:"source_key"`
			ImportedAt time.Time `json:"imported_at"`
		} `json:"already_imported"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, "imported", body.AlreadyImported.Status)
	require.Equal(t, "b2://bucket/message_3.json", body.AlreadyImported.SourceKey)
	require.True(t, at.Equal(body.AlreadyImported.ImportedAt))
	require.Contains(t, body.Detail, "already been imported")
	require.Empty(t, workflow.started.RequestID, "no second processing run may start")
	require.Equal(t, "upload://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", store.asked)
}

func TestStartBlocksWhileAnImportOfTheContentIsRunning(t *testing.T) {
	handler, _, workflow := duplicateGuardHandler(t, []previewmodel.ImportedContent{{WorkflowID: "running", SourceKey: "upload://a", Committed: false}}, nil)
	workflow.operation = proffer.OperationState{Lifecycle: proffer.OperationRunning}
	response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(guardStartBody))
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"status":"in_progress"`)
	require.Empty(t, workflow.started.RequestID)
}

func TestStartAllowsARetryWhenEarlierImportsFailedOrWerePurged(t *testing.T) {
	for name, opErr := range map[string]error{"finished": nil, "history purged": serviceerror.NewNotFound("workflow not found")} {
		t.Run(name, func(t *testing.T) {
			handler, _, workflow := duplicateGuardHandler(t, []previewmodel.ImportedContent{{WorkflowID: "earlier", Committed: false}}, nil)
			workflow.operation = proffer.OperationState{Lifecycle: proffer.OperationFailed, Terminal: true}
			workflow.operationErr = opErr
			response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(guardStartBody))
			require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
			require.Equal(t, "request-dup", workflow.started.RequestID)
		})
	}
}

func TestStartFailsClosedWhenDuplicateStatusCannotBeVerified(t *testing.T) {
	handler, _, workflow := duplicateGuardHandler(t, nil, errors.New("database unavailable"))
	response := servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(guardStartBody))
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Empty(t, workflow.started.RequestID)

	handler, _, workflow = duplicateGuardHandler(t, []previewmodel.ImportedContent{{WorkflowID: "earlier"}}, nil)
	workflow.operationErr = errors.New("temporal unreachable")
	response = servePreview(handler.Routes(), http.MethodPost, "/reference-import/start", []byte(guardStartBody))
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Empty(t, workflow.started.RequestID)
}

func TestStartWithoutTheCapabilityIsUnchanged(t *testing.T) {
	handler, _, workflow := previewTestHandler(t)
	startPreview(t, handler)
	require.Equal(t, "request-1", workflow.started.RequestID)
}
