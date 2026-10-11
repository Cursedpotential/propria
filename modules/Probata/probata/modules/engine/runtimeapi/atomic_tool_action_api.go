package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/atomictool"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// AtomicToolWorkflows is the existing Proffer starter's Temporal action seam.
type AtomicToolWorkflows interface {
	Start(context.Context, atomictool.Request) (atomictool.Started, error)
	Status(context.Context, string) (atomictool.Progress, error)
}

// AtomicToolHTTPHandler serves actor-scoped source-pinned actions and status.
// It uses the same tailnet service token as the other Proffer starter routes.
type AtomicToolHTTPHandler struct {
	workflows        AtomicToolWorkflows
	serviceTokenPath string
}

// NewAtomicToolHTTPHandler validates the existing workflow client and token.
func NewAtomicToolHTTPHandler(workflows AtomicToolWorkflows, serviceTokenPath string) (*AtomicToolHTTPHandler, error) {
	if workflows == nil {
		return nil, errors.New("atomic tool handler requires a workflow client")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &AtomicToolHTTPHandler{workflows, serviceTokenPath}, nil
}

// Routes mounts one action start and one actor-bound status read, without a
// direct tool-execution route or a second scheduler.
func (h *AtomicToolHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reference-import/atomic-tools/actions", tailnetServiceAuth(h.serviceTokenPath, "atomic tool tailnet authorization required", h.start))
	mux.HandleFunc("GET /reference-import/atomic-tools/workflows/{workflow_id}", tailnetServiceAuth(h.serviceTokenPath, "atomic tool tailnet authorization required", h.status))
	return mux
}

type atomicToolBody struct {
	OperatingMode string         `json:"operating_mode"`
	MatterID      string         `json:"matter_id"`
	CourtCaseID   string         `json:"court_case_id"`
	ToolID        string         `json:"tool_id"`
	SourceRef     string         `json:"source_ref"`
	SourceSHA256  string         `json:"source_sha256"`
	Args          map[string]any `json:"args"`
}

// start admits only the approved case and authenticated actor, then starts
// or joins one Temporal action identified by the caller's Idempotency-Key.
func (h *AtomicToolHTTPHandler) start(w http.ResponseWriter, r *http.Request) {
	var body atomicToolBody
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	mode, err := caseidentity.ParseMode(body.OperatingMode)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	in := atomictool.Request{
		OperatingMode: string(mode), CourtCaseID: body.CourtCaseID,
		MatterID: body.MatterID, Actor: actor,
		RequestID: flow.DeterministicID("atomic_tool_action", key, actor.SubjectUID, body.MatterID),
		ToolID:    body.ToolID, SourceRef: body.SourceRef, SourceSHA256: body.SourceSHA256, Args: body.Args,
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

// status reads real Temporal state and releases it only to the starting actor.
// It returns bounded outcome, ref and audit metadata, never tool output.
func (h *AtomicToolHTTPHandler) status(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("workflow_id")
	if !strings.HasPrefix(id, atomictool.WorkflowIDPrefix) || len(id) > 160 {
		previewError(w, http.StatusNotFound, errors.New("unknown atomic tool workflow"))
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
		previewError(w, http.StatusNotFound, errors.New("unknown atomic tool workflow"))
		return
	}
	previewJSON(w, http.StatusOK, map[string]any{"workflow_id": id, "outcome": progress.Outcome, "result": progress.Result, "error": progress.Error})
}
