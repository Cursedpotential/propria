package runtimeapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// ContextSourceWorkflows is the shared Go Proffer start/status seam for context-first imports.
// Inputs: actor-bound workflow input or ID. Outputs: real Temporal IDs and progress.
// Effects: delegated Temporal calls; choose over the legacy preview route for context-v1.
type ContextSourceWorkflows interface {
	Start(context.Context, proffer.WorkflowInput) (proffer.ContextStarted, error)
	Status(context.Context, string) (proffer.ContextProgress, error)
}

// ContextSourceHTTPHandler serves source registration and actor-bound progress.
// Inputs: shared workflow client and service token. Outputs: guarded HTTP routes.
// Effects: starts/polls Go Proffer workflows; choose for new context-first intake.
type ContextSourceHTTPHandler struct {
	workflows        ContextSourceWorkflows
	serviceTokenPath string
}

// NewContextSourceHTTPHandler validates the workflow seam and existing token file.
// Inputs: shared starter and token path. Outputs: guarded handler or error.
// Effects: reads token for validation; choose at starter composition, not as a new service.
func NewContextSourceHTTPHandler(workflows ContextSourceWorkflows, serviceTokenPath string) (*ContextSourceHTTPHandler, error) {
	if workflows == nil {
		return nil, errors.New("context source handler requires a workflow client")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &ContextSourceHTTPHandler{workflows, serviceTokenPath}, nil
}

// Routes mounts context-first starts and actor-bound status on the existing tailnet guard.
// Inputs: authenticated HTTP requests. Outputs: 202 start and reference-only status.
// Effects: route registration; choose beside the established Proffer endpoints.
func (h *ContextSourceHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reference-import/context/sources", tailnetServiceAuth(h.serviceTokenPath, "context source tailnet authorization required", h.start))
	mux.HandleFunc("GET /reference-import/context/workflows/{workflow_id}", tailnetServiceAuth(h.serviceTokenPath, "context source tailnet authorization required", h.status))
	return mux
}

type contextSourceBody struct {
	proffer.ContextResourceBounds

	ContractVersion   string `json:"contract_version"`
	SourceRef         string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id"`
	PackageRef        string `json:"package_ref"`
	SourceKind        string `json:"source_kind"`
	DeclaredFormat    string `json:"declared_format"`
	MatterID          string `json:"matter_id"`
	CourtCaseID       string `json:"court_case_id"`
}

// validate checks source coordinates and existing operation limits without creating policy gates.
// Inputs: decoded request. Output: error or nil. Effects: none.
// Choose before scheduling the source registration Activity.
func (b contextSourceBody) validate() error {
	if err := b.ContextResourceBounds.Validate(); err != nil {
		return err
	}
	if b.ContractVersion != proffer.ContextContractVersion || strings.TrimSpace(b.SourceRef) == "" ||
		len(b.SourceRef) > 2048 || len(b.ProviderVersionID) > 512 || len(b.PackageRef) > 2048 ||
		len(b.SourceKind) > 128 || len(b.DeclaredFormat) > 128 ||
		len(b.MatterID) > 512 || len(b.CourtCaseID) > 512 ||
		strings.ContainsAny(b.SourceRef+b.ProviderVersionID+b.PackageRef+b.SourceKind+b.DeclaredFormat+b.MatterID+b.CourtCaseID, "\x00\r\n") {
		return errors.New("context source requires bounded contract and native source coordinates")
	}
	return nil
}

// start binds a real source pointer to an actor and idempotent Go workflow.
// Inputs: context-v1 JSON, actor headers and Idempotency-Key. Outputs: real Temporal IDs.
// Effects: starts/joins ProfferWorkflow; choose before any parser or custody route.
func (h *ContextSourceHTTPHandler) start(w http.ResponseWriter, r *http.Request) {
	var body contextSourceBody
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	if err := body.validate(); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if body.SourceKind == "" {
		body.SourceKind = "unknown"
	}
	if body.DeclaredFormat == "" {
		body.DeclaredFormat = "unknown"
	}
	in := proffer.WorkflowInput{
		ContextContract: body.ContractVersion, ActorSubjectUID: actor.SubjectUID,
		ContextResourceBounds: body.ContextResourceBounds,
		RequestID: proffer.ContextWorkflowIDPrefix + flow.DeterministicID("context_source_ingest", key, actor.SubjectUID,
			body.SourceRef, body.ProviderVersionID, body.PackageRef, body.SourceKind, body.DeclaredFormat, body.MatterID, body.CourtCaseID,
			fmt.Sprintf("%d:%d:%d:%d:%d", body.MaxSourceBytes, body.MaxRecords, body.MaxTextBytes, body.MaxChunks, body.MaxModelCalls)),
		SourceRef: proffer.Ref(body.SourceRef), ProviderVersionID: body.ProviderVersionID,
		PackageRef: proffer.Ref(body.PackageRef), SourceKind: body.SourceKind,
		DeclaredFormat: body.DeclaredFormat, MatterID: body.MatterID, CourtCaseID: body.CourtCaseID,
	}
	started, err := h.workflows.Start(r.Context(), in)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	previewJSON(w, http.StatusAccepted, started)
}

// status releases compact context progress only to the actor that started the run.
// Inputs: context workflow ID and authenticated actor header. Outputs: status and references.
// Effects: one Temporal read; choose for polling without fetching source content.
func (h *ContextSourceHTTPHandler) status(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("workflow_id")
	if !strings.HasPrefix(id, proffer.ContextWorkflowIDPrefix) || len(id) > 160 {
		previewError(w, http.StatusNotFound, errors.New("unknown context workflow"))
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
		previewError(w, http.StatusNotFound, errors.New("unknown context workflow"))
		return
	}
	previewJSON(w, http.StatusOK, struct {
		WorkflowID string `json:"workflow_id"`
		proffer.ContextProgress
	}{WorkflowID: id, ContextProgress: progress})
}
