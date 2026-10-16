// Byline: Codex · GPT-6 · 2026-10-11
package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/google/uuid"
)

// DiscoveryDispatcher starts one persisted feedback revision in Temporal.
// Inputs: a saved canonical run. Outputs: execution coordinates or a retryable error.
// Effects: workflow dispatch only. Choose after PG acknowledgement, never before it.
type DiscoveryDispatcher interface {
	Start(context.Context, discovery.Run) (string, string, error)
}

// DiscoveryHTTPHandler exposes optional discovery independently of intake and approval.
// Inputs: PG store, dispatcher and mounted service token. Outputs: authenticated routes.
// Effects: dedicated analysis records and durable dispatch; never evidence publication.
type DiscoveryHTTPHandler struct {
	store      discovery.Store
	dispatcher DiscoveryDispatcher
	tokenPath  string
}

// NewDiscoveryHTTPHandler validates the configured analysis transport dependencies.
// Inputs: canonical store, Temporal dispatcher and credential path. Outputs: handler.
// Effects: reads the mounted credential. Choose for the optional discovery endpoint.
func NewDiscoveryHTTPHandler(store discovery.Store, dispatcher DiscoveryDispatcher, tokenPath string) (*DiscoveryHTTPHandler, error) {
	if store == nil || dispatcher == nil {
		return nil, errors.New("discovery requires canonical persistence and dispatch")
	}
	if _, err := loadServiceToken(tokenPath); err != nil {
		return nil, err
	}
	return &DiscoveryHTTPHandler{store: store, dispatcher: dispatcher, tokenPath: tokenPath}, nil
}

// Routes registers scoped reads and attributed analysis corrections.
// Inputs: configured handler. Outputs: route mux. Effects: registration only.
// Choose independently of existing Review decisions and Surreal publication.
func (h *DiscoveryHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /discovery/datasets", overlayAuth(h.tokenPath, "discovery", h.datasets))
	mux.HandleFunc("POST /discovery/runs", overlayAuth(h.tokenPath, "discovery", h.create))
	mux.HandleFunc("GET /discovery/runs/{run_id}", overlayAuth(h.tokenPath, "discovery", h.read))
	mux.HandleFunc("GET /discovery/runs/{run_id}/candidates", overlayAuth(h.tokenPath, "discovery", h.candidates))
	mux.HandleFunc("GET /discovery/runs/{run_id}/context/{candidate_id}", overlayAuth(h.tokenPath, "discovery", h.sourceContext))
	mux.HandleFunc("POST /discovery/runs/{run_id}/feedback", overlayAuth(h.tokenPath, "discovery", h.feedback))
	return mux
}

// discoveryScope parses the exact HTTP case coordinates without choosing an identity.
// Inputs: query coordinates. Outputs: a validated scope. Effects: none.
// Choose for bounded discovery reads; persistence also verifies admitted case identity.
func discoveryScope(r *http.Request) (discovery.Scope, error) {
	mode, err := caseidentity.ParseMode(r.URL.Query().Get("mode"))
	if err != nil {
		return discovery.Scope{}, err
	}
	scope := discovery.Scope{Mode: string(mode), MatterID: r.URL.Query().Get("matter_id"), CourtCaseID: r.URL.Query().Get("court_case_id")}
	if !canonicalDiscoveryID(scope.MatterID) || !canonicalDiscoveryID(scope.CourtCaseID) {
		return discovery.Scope{}, discovery.ErrInvalid
	}
	return scope, nil
}

func canonicalDiscoveryID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

func (h *DiscoveryHTTPHandler) actor(w http.ResponseWriter, r *http.Request) (discovery.Actor, bool) {
	uid, name, err := authenticatedActor(r)
	if err != nil {
		previewError(w, 401, errors.New("authenticated actor required"))
		return discovery.Actor{}, false
	}
	return discovery.Actor{SubjectUID: uid, Username: name}, true
}

func (h *DiscoveryHTTPHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, discovery.ErrInvalid):
		previewError(w, 422, errors.New("invalid discovery request"))
	case errors.Is(err, discovery.ErrNotFound):
		previewError(w, 404, errors.New("discovery record not found"))
	case errors.Is(err, discovery.ErrConflict), errors.Is(err, discovery.ErrScope):
		previewError(w, 409, errors.New("discovery scope or revision changed; refresh the run"))
	case errors.Is(err, caseidentity.ErrDevWrite), errors.Is(err, caseidentity.ErrOperatingModeUnknown):
		previewError(w, 409, err)
	default:
		previewError(w, 503, errors.New("adaptive discovery is unavailable"))
	}
}

func (h *DiscoveryHTTPHandler) readScope(w http.ResponseWriter, r *http.Request) (discovery.Scope, bool) {
	if _, ok := h.actor(w, r); !ok {
		return discovery.Scope{}, false
	}
	scope, err := discoveryScope(r)
	if err != nil {
		h.fail(w, discovery.ErrInvalid)
		return discovery.Scope{}, false
	}
	if runID := r.PathValue("run_id"); runID != "" && !canonicalDiscoveryID(runID) {
		h.fail(w, discovery.ErrInvalid)
		return discovery.Scope{}, false
	}
	return scope, true
}

// datasets lists only server-bound selections admitted for this case.
// Inputs: authenticated scope. Outputs: bounded dataset definitions. Effects: reads.
// Choose before run creation so clients never supply database URLs or SQL.
func (h *DiscoveryHTTPHandler) datasets(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.readScope(w, r)
	if !ok {
		return
	}
	items, err := h.store.Datasets(r.Context(), scope)
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []discovery.Dataset{}
	}
	previewJSON(w, 200, map[string]any{"items": items})
}

// create saves the selected analysis before starting its independently retryable workflow.
// Inputs: request without actor and matching UUID Idempotency-Key. Outputs: saved run.
// Effects: PG analysis control and dispatch; a failed dispatch remains pending in PG.
func (h *DiscoveryHTTPHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var request discovery.RunRequest
	if err := decodeOverlayJSON(w, r, &request); err != nil {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	if request.Actor.SubjectUID != "" || request.Actor.Username != "" || !canonicalDiscoveryID(request.RunID) || r.Header.Get("Idempotency-Key") != request.RunID {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	mode, err := caseidentity.ParseMode(request.Scope.Mode)
	if err != nil {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	request.Scope.Mode = string(mode)
	for _, raw := range []string{r.URL.Query().Get("mode"), r.Header.Get("X-Propria-Operating-Mode")} {
		if raw == "" {
			continue
		}
		boundaryMode, err := caseidentity.ParseMode(raw)
		if err != nil || boundaryMode != mode {
			h.fail(w, discovery.ErrScope)
			return
		}
	}
	if err := caseidentity.RequireCanonicalWrite(mode); err != nil {
		h.fail(w, err)
		return
	}
	request.Actor = actor
	run, err := h.store.CreateRun(r.Context(), request)
	if err != nil {
		h.fail(w, err)
		return
	}
	run = h.dispatch(r.Context(), run)
	previewJSON(w, http.StatusAccepted, run)
}

// dispatch records a successful start after the run is already saved.
// Inputs: pending PG run. Outputs: latest saved state. Effects: Temporal and PG receipt.
// Choose for initial submission; worker reconciliation also retries pending starts.
func (h *DiscoveryHTTPHandler) dispatch(ctx context.Context, run discovery.Run) discovery.Run {
	if !run.DispatchPending {
		return run
	}
	workflowID, temporalRunID, err := h.dispatcher.Start(ctx, run)
	if err != nil {
		return run
	}
	if err = h.store.RecordDispatch(ctx, run.Request.RunID, run.Request.Scope, run.FeedbackRevision, workflowID, temporalRunID); err != nil {
		return run
	}
	updated, err := h.store.ReadRun(ctx, run.Request.RunID, run.Request.Scope)
	if err != nil {
		return run
	}
	return updated
}

// read returns the PG run independently of the selected input store's availability.
// Inputs: authenticated run ID and case. Outputs: persisted analysis state. Effects: reads.
func (h *DiscoveryHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.readScope(w, r)
	if !ok {
		return
	}
	run, err := h.store.ReadRun(r.Context(), r.PathValue("run_id"), scope)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, 200, run)
}

// candidates reads one bounded queue with its current correction revision.
// Inputs: run, case, cursor and limit. Outputs: ranked source-linked candidates. Effects: reads.
func (h *DiscoveryHTTPHandler) candidates(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.readScope(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > discovery.MaxCandidates {
			h.fail(w, discovery.ErrInvalid)
			return
		}
		limit = parsed
	}
	page, err := h.store.ListCandidates(r.Context(), r.PathValue("run_id"), scope, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, 200, page)
}

// sourceContext returns bounded complete episode records and genuine root-source locators.
// Inputs: run/candidate IDs and scope. Outputs: chronology for human review. Effects: reads.
// Choose over abbreviated classifier excerpts when assessing attribution or context.
func (h *DiscoveryHTTPHandler) sourceContext(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.readScope(w, r)
	if !ok {
		return
	}
	if !canonicalDiscoveryID(r.PathValue("candidate_id")) {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	value, err := h.store.ReadContext(r.Context(), r.PathValue("run_id"), scope, r.PathValue("candidate_id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, 200, value)
}

// feedback acknowledges an attributed correction only after its PG commit.
// Inputs: scoped run, revision, correction and matching UUID idempotency header.
// Outputs: durable feedback receipt with pending or dispatched rescore state.
// Effects: append-only analysis feedback; no evidence promotion or Surreal publication.
func (h *DiscoveryHTTPHandler) feedback(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.readScope(w, r)
	if !ok {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var request discovery.FeedbackRequest
	if err := decodeOverlayJSON(w, r, &request); err != nil {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	if !canonicalDiscoveryID(request.FeedbackID) || r.Header.Get("Idempotency-Key") != request.FeedbackID {
		h.fail(w, discovery.ErrInvalid)
		return
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(scope.Mode)); err != nil {
		h.fail(w, err)
		return
	}
	receipt, err := h.store.SubmitFeedback(r.Context(), r.PathValue("run_id"), scope, request, actor)
	if err != nil {
		h.fail(w, err)
		return
	}
	receipt.Run = h.dispatch(r.Context(), receipt.Run)
	previewJSON(w, 200, receipt)
}
