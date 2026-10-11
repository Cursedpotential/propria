package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

type approvedQueryRecorder struct {
	requests  []approvedgraphqueryflow.Request
	progress  approvedgraphqueryflow.Progress
	startErr  error
	statusErr error
}

func (f *approvedQueryRecorder) Start(_ context.Context, in approvedgraphqueryflow.Request) (approvedgraphqueryflow.Started, error) {
	f.requests = append(f.requests, in)
	if f.startErr != nil {
		return approvedgraphqueryflow.Started{}, f.startErr
	}
	return approvedgraphqueryflow.Started{WorkflowID: approvedgraphqueryflow.WorkflowIDPrefix + in.RequestID, RunID: "synthetic-run"}, nil
}

func (f *approvedQueryRecorder) Status(context.Context, string) (approvedgraphqueryflow.Progress, error) {
	return f.progress, f.statusErr
}

func approvedQueryTestHandler(t *testing.T) (*ApprovedGraphQueryHTTPHandler, *approvedQueryRecorder) {
	t.Helper()
	recorder := &approvedQueryRecorder{}
	handler, err := NewApprovedGraphQueryHTTPHandler(recorder, serviceTokenPath(t))
	require.NoError(t, err)
	return handler, recorder
}

func approvedQueryScope() surrealsink.ApprovedQueryScope {
	return surrealsink.ApprovedQueryScope{
		MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID,
		AccessPolicyID: "policy-1", ApprovedRevisionID: "revision-2",
		ApprovalDigest: strings.Repeat("a", 64), ProjectionGenerationID: "generation-3",
		ProjectionHash: strings.Repeat("b", 64), Perspective: "as_lived",
		Horizon: time.Date(2025, 9, 12, 14, 30, 0, 0, time.UTC), Limit: 2, Cursor: "next-page",
	}
}

// TestApprovedGraphQueryStartForwardsExactScope proves one actor-bound click preserves every query pin.
// Inputs: synthetic as-lived scope and headers. Output: started workflow and forwarded request.
// Effects: fake workflow call only; choose when the API's query handoff changes.
func TestApprovedGraphQueryStartForwardsExactScope(t *testing.T) {
	handler, workflows := approvedQueryTestHandler(t)
	scope := approvedQueryScope()
	body := map[string]any{"operating_mode": "LIVE", "scope": scope}
	first := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/analysis/queries", body, "query-click-1"))
	again := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/analysis/queries", body, "query-click-1"))
	require.Equal(t, http.StatusAccepted, first.Code, first.Body.String())
	require.Equal(t, http.StatusAccepted, again.Code)
	require.Len(t, workflows.requests, 2)
	require.Equal(t, scope, workflows.requests[0].Scope)
	require.Equal(t, "authentik-user-1", workflows.requests[0].Actor.SubjectUID)
	require.Equal(t, "LIVE", workflows.requests[0].OperatingMode)
	require.Equal(t, flow.DeterministicID("approved_context_graph_query", "query-click-1", "authentik-user-1", scope.MatterID), workflows.requests[0].RequestID)
	require.Equal(t, workflows.requests[0].RequestID, workflows.requests[1].RequestID)
	var started approvedgraphqueryflow.Started
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &started))
	require.Equal(t, approvedgraphqueryflow.WorkflowIDPrefix+workflows.requests[0].RequestID, started.WorkflowID)
}

// TestApprovedGraphQueryStatusHidesAnotherActor denies a valid workflow to a different user.
// Inputs: one real-form workflow ID and two actor headers. Output: own status or 404.
// Effects: fake status read only; choose when status authorization changes.
func TestApprovedGraphQueryStatusHidesAnotherActor(t *testing.T) {
	handler, workflows := approvedQueryTestHandler(t)
	id := approvedgraphqueryflow.WorkflowIDPrefix + "synthetic-workflow"
	workflows.progress = approvedgraphqueryflow.Progress{ActorSubjectUID: "authentik-user-1", Outcome: "completed"}
	path := "/reference-import/analysis/workflows/" + id
	require.Equal(t, http.StatusOK, servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, path, nil)).Code)
	other := newPreviewRequest(http.MethodGet, path, nil)
	other.Header.Set("X-authentik-uid", "another-user")
	require.Equal(t, http.StatusNotFound, servePreviewRequest(handler.Routes(), other).Code)
	require.Equal(t, http.StatusNotFound, servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, "/reference-import/analysis/workflows/atomic-tool:wrong-family", nil)).Code)
}

// TestApprovedGraphQueryRejectsInvalidRequestsAndUnavailableWorkflow checks the HTTP boundary.
// Inputs: malformed JSON, absent key, invalid pin and workflow errors. Output: bounded status codes.
// Effects: fake workflow calls only; choose when validation or failure mapping changes.
func TestApprovedGraphQueryRejectsInvalidRequestsAndUnavailableWorkflow(t *testing.T) {
	handler, workflows := approvedQueryTestHandler(t)
	path := "/reference-import/analysis/queries"
	badJSON := newPreviewRequest(http.MethodPost, path, []byte(`{"operating_mode":"LIVE","scope":{},"unknown":true}`))
	badJSON.Header.Set("Idempotency-Key", "bad-json")
	require.Equal(t, http.StatusBadRequest, servePreviewRequest(handler.Routes(), badJSON).Code)
	body := map[string]any{"operating_mode": "LIVE", "scope": approvedQueryScope()}
	require.Equal(t, http.StatusUnauthorized, servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, path, body, "")).Code)
	invalid := approvedQueryScope()
	invalid.ProjectionHash = "not-a-digest"
	require.Equal(t, http.StatusUnprocessableEntity, servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, path, map[string]any{"operating_mode": "LIVE", "scope": invalid}, "bad-pin")).Code)
	require.Empty(t, workflows.requests)
	workflows.startErr = errors.New("synthetic Temporal outage")
	require.Equal(t, http.StatusServiceUnavailable, servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, path, body, "outage")).Code)
}
