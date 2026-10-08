package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// ApprovedGraphQueryWorkflows is the existing Temporal starter seam for approved context reads.
type ApprovedGraphQueryWorkflows interface {
	Start(context.Context, approvedgraphqueryflow.Request) (approvedgraphqueryflow.Started, error)
	Status(context.Context, string) (approvedgraphqueryflow.Progress, error)
}

// ApprovedGraphQueryHTTPHandler serves actor-bound, revision-pinned query starts and status.
type ApprovedGraphQueryHTTPHandler struct {
	workflows        ApprovedGraphQueryWorkflows
	serviceTokenPath string
}

// NewApprovedGraphQueryHTTPHandler checks the shared workflow seam and service credential.
// Inputs: existing Temporal workflow starter and mounted token path. Output: guarded handler.
// Effects: reads the token for validation; choose this for approved graph reads, not a new query store.
func NewApprovedGraphQueryHTTPHandler(workflows ApprovedGraphQueryWorkflows, serviceTokenPath string) (*ApprovedGraphQueryHTTPHandler, error) {
	if workflows == nil {
		return nil, errors.New("approved graph query handler requires a workflow client")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &ApprovedGraphQueryHTTPHandler{workflows, serviceTokenPath}, nil
}

// Routes mounts the approved query start and actor-bound status paths.
// Inputs: requests bearing the shared service token. Output: HTTP responses.
// Effects: route registration; choose beside the existing Proffer starter routes.
func (h *ApprovedGraphQueryHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reference-import/analysis/queries", tailnetServiceAuth(h.serviceTokenPath, "approved graph query tailnet authorization required", h.start))
	mux.HandleFunc("GET /reference-import/analysis/workflows/{workflow_id}", tailnetServiceAuth(h.serviceTokenPath, "approved graph query tailnet authorization required", h.status))
	return mux
}

type approvedGraphQueryBody struct {
	OperatingMode string                         `json:"operating_mode"`
	Scope         surrealsink.ApprovedQueryScope `json:"scope"`
}

// start binds the authenticated actor and idempotency key to an exact query scope.
// Inputs: bounded operating mode and scope JSON, actor headers and Idempotency-Key.
// Output: Temporal workflow IDs. Effects: starts or joins one query; choose for approved context reads.
func (h *ApprovedGraphQueryHTTPHandler) start(w http.ResponseWriter, r *http.Request) {
	var body approvedGraphQueryBody
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	mode, err := caseidentity.ParseMode(body.OperatingMode)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	in := approvedgraphqueryflow.Request{
		OperatingMode: string(mode), Actor: actor,
		RequestID: flow.DeterministicID("approved_context_graph_query", key, actor.SubjectUID, body.Scope.MatterID),
		Scope:     body.Scope,
	}
	if err := in.Validate(); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	started, err := h.workflows.Start(r.Context(), in)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	previewJSON(w, http.StatusAccepted, started)
}

// status releases real Temporal progress only to the actor that started the query.
// Inputs: bounded query workflow ID and authenticated actor headers. Output: status metadata.
// Effects: one Temporal status read; choose for workflow polling, never direct claim retrieval.
func (h *ApprovedGraphQueryHTTPHandler) status(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("workflow_id")
	if !strings.HasPrefix(id, approvedgraphqueryflow.WorkflowIDPrefix) || len(id) > 160 {
		previewError(w, http.StatusNotFound, errors.New("unknown approved graph query workflow"))
		return
	}
	actor, _, err := authenticatedActor(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	progress, err := h.workflows.Status(r.Context(), id)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	if progress.ActorSubjectUID == "" || progress.ActorSubjectUID != actor {
		previewError(w, http.StatusNotFound, errors.New("unknown approved graph query workflow"))
		return
	}
	previewJSON(w, http.StatusOK, map[string]any{"workflow_id": id, "outcome": progress.Outcome, "result": progress.Result, "error": progress.Error})
}
