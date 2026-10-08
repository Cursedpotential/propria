// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Entity and event extraction routes on the Proffer starter. Same boundary as
// the other starter routes: tailnet peer + mounted service token, Authentik
// actor headers for every owner act, an Idempotency-Key on every write.
//
//	POST /reference-import/entities/extract             start EntityExtractionWorkflow
//	GET  /reference-import/entities/extractions/{id}    its progress
//	GET  /reference-import/entities/proposals           current entity + event proposals
//	POST /reference-import/entities/corrections         one owner edit (entity or event)
//	POST /reference-import/events/from-record           "event worth recalling" on a record
//	GET  /reference-import/entities/records/{id}        one record of the run (click a mention)
//	GET  /reference-import/entities/registry            search committed entities (merge target)
//	POST /reference-import/entities/validate            pass/fail list before Run
//	POST /reference-import/entities/commit              start ExtractionCommitWorkflow (422 unless valid)
//	GET  /reference-import/entities/commits/{id}        its progress
package runtimeapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// ExtractionWorkflows starts and reads the extraction workflows.
type ExtractionWorkflows interface {
	StartExtraction(context.Context, flow.ExtractionRequest) (flow.Started, error)
	StartCommit(context.Context, flow.CommitRequest) (flow.Started, error)
	Status(context.Context, string) (flow.Progress, error)
}

// EntityExtractionHTTPHandler serves the extraction routes.
type EntityExtractionHTTPHandler struct {
	store            service.Store
	workflows        ExtractionWorkflows
	serviceTokenPath string
	clock            func() time.Time
}

// NewEntityExtractionHTTPHandler validates its seams and the service token.
func NewEntityExtractionHTTPHandler(store service.Store, workflows ExtractionWorkflows, serviceTokenPath string) (*EntityExtractionHTTPHandler, error) {
	if store == nil || workflows == nil {
		return nil, errors.New("entity extraction handler requires a store and a workflow client")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &EntityExtractionHTTPHandler{store: store, workflows: workflows, serviceTokenPath: serviceTokenPath, clock: time.Now}, nil
}

// Routes returns the extraction mux.
func (h *EntityExtractionHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reference-import/entities/extract", h.auth(h.extract))
	mux.HandleFunc("GET /reference-import/entities/extractions/{workflow_id}", h.auth(h.extractionStatus))
	mux.HandleFunc("GET /reference-import/entities/proposals", h.auth(h.proposals))
	mux.HandleFunc("POST /reference-import/entities/corrections", h.auth(h.correct))
	mux.HandleFunc("POST /reference-import/events/from-record", h.auth(h.markEvent))
	mux.HandleFunc("GET /reference-import/entities/records/{record_id}", h.auth(h.record))
	mux.HandleFunc("GET /reference-import/entities/registry", h.auth(h.registry))
	mux.HandleFunc("POST /reference-import/entities/validate", h.auth(h.validate))
	mux.HandleFunc("POST /reference-import/entities/commit", h.auth(h.commit))
	mux.HandleFunc("GET /reference-import/entities/commits/{workflow_id}", h.auth(h.commitStatus))
	mux.HandleFunc("POST /reference-import/ai-candidates/stage", h.auth(h.stageAICandidates))
	mux.HandleFunc("GET /reference-import/ai-candidates", h.auth(h.aiCandidates))
	mux.HandleFunc("POST /reference-import/ai-candidates/decision", h.auth(h.decideAICandidate))
	return mux
}

type aiStageRequest struct {
	runRequest
	Candidates []service.AICandidate `json:"candidates"`
}

type aiDecisionRequest struct {
	runRequest
	service.AISourcePin
	CandidateID           string `json:"candidate_id"`
	ExpectedContentSHA256 string `json:"expected_content_sha256"`
	Decision              string `json:"decision"`
}

// aiReviewStore exposes the retained-source branch only when the configured store supports it.
// Inputs: none. Outputs: store or an unavailable error. Effects: none.
// Choose for AI candidates, never the SMS generation route.
func (h *EntityExtractionHTTPHandler) aiReviewStore() (service.AIReviewStore, error) {
	store, ok := h.store.(service.AIReviewStore)
	if !ok {
		return nil, errors.New("retained-source AI review store is unavailable")
	}
	return store, nil
}

// stageAICandidates stages bounded retained-source proposals after authenticated actor and durable scope checks.
// Inputs: preview, candidates, Idempotency-Key and Authentik actor. Outputs: stable IDs.
// Effects: pending working rows only. Choose for a verified AI candidate bundle bridge.
func (h *EntityExtractionHTTPHandler) stageAICandidates(w http.ResponseWriter, r *http.Request) {
	var body aiStageRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	if !previewHandlePattern.MatchString(body.PreviewHandle) {
		previewError(w, http.StatusUnprocessableEntity, errors.New("preview_handle is invalid"))
		return
	}
	mode, err := caseidentity.ParseMode(body.MatterMode)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := caseidentity.RequireCanonicalWrite(mode); err != nil {
		previewError(w, http.StatusConflict, err)
		return
	}
	store, err := h.aiReviewStore()
	if err != nil {
		h.fail(w, err)
		return
	}
	if len(body.Candidates) == 0 {
		previewError(w, http.StatusUnprocessableEntity, errors.New("candidates are required"))
		return
	}
	verified, err := store.VerifyAISource(r.Context(), body.PreviewHandle, body.Candidates[0].AISourcePin)
	if err != nil {
		h.fail(w, err)
		return
	}
	if verified != body.MatterMode {
		previewError(w, http.StatusConflict, errors.New("request mode disagrees with durable source receipt"))
		return
	}
	digest := requestDigest(key, actor, body)
	runID := flow.DeterministicID("ai_candidate_run", body.Candidates[0].SourceVersionID, digest)
	ids, err := store.StageAICandidates(r.Context(), body.PreviewHandle, runID, digest, body.Candidates)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusAccepted, map[string]any{"source_version_id": body.Candidates[0].SourceVersionID, "run_id": runID, "candidate_ids": ids, "matter_mode": verified})
}

// aiCandidates reads pending and decided AI candidates under the exact retained source pin.
// Inputs: preview handle and custody pin in query parameters. Outputs: bounded candidate views.
// Effects: read-only. Choose for the owner AI review surface.
func (h *EntityExtractionHTTPHandler) aiCandidates(w http.ResponseWriter, r *http.Request) {
	store, err := h.aiReviewStore()
	if err != nil {
		h.fail(w, err)
		return
	}
	pin := service.AISourcePin{SourceVersionID: r.URL.Query().Get("source_version_id"), SourceObjectID: r.URL.Query().Get("source_object_id"), SourceSHA256: r.URL.Query().Get("source_sha256")}
	if version := r.URL.Query().Get("version_id"); version != "" {
		pin.VersionID = &version
	}
	mode, err := store.VerifyAISource(r.Context(), r.URL.Query().Get("preview_handle"), pin)
	if err != nil {
		h.fail(w, err)
		return
	}
	if mode != r.URL.Query().Get("matter_mode") {
		previewError(w, http.StatusConflict, errors.New("request mode disagrees with durable source receipt"))
		return
	}
	const pageSize = 200
	rows, err := store.ListAICandidates(r.Context(), pin.SourceVersionID, r.URL.Query().Get("after_id"), pageSize+1)
	if err != nil {
		h.fail(w, err)
		return
	}
	nextCursor := ""
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		nextCursor = rows[len(rows)-1].ID
	}
	previewJSON(w, http.StatusOK, map[string]any{"source_version_id": pin.SourceVersionID, "candidates": rows, "next_cursor": nextCursor, "matter_mode": mode})
}

// decideAICandidate records one explicit owner approval, rejection or request for more information.
// Inputs: retained pin, candidate ID, decision, actor and Idempotency-Key. Outputs: receipt digest.
// Effects: commits review state, then attempts approved graph dispatch; enqueue failure retains the receipt.
// Choose after inspecting a candidate; pending projection can retry the identical decision request.
func (h *EntityExtractionHTTPHandler) decideAICandidate(w http.ResponseWriter, r *http.Request) {
	var body aiDecisionRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	mode, err := caseidentity.ParseMode(body.MatterMode)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := caseidentity.RequireCanonicalWrite(mode); err != nil {
		previewError(w, http.StatusConflict, err)
		return
	}
	store, err := h.aiReviewStore()
	if err != nil {
		h.fail(w, err)
		return
	}
	verified, err := store.VerifyAISource(r.Context(), body.PreviewHandle, body.AISourcePin)
	if err != nil {
		h.fail(w, err)
		return
	}
	if verified != body.MatterMode {
		previewError(w, http.StatusConflict, errors.New("request mode disagrees with durable source receipt"))
		return
	}
	digest := requestDigest(key, actor, body)
	decisionID, err := store.DecideAICandidate(r.Context(), body.AISourcePin, body.CandidateID, body.ExpectedContentSHA256, body.Decision, actor, digest, h.clock().UTC())
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, map[string]any{
		"candidate_id": body.CandidateID, "decision": body.Decision, "decision_id": decisionID,
		"request_digest": digest, "matter_mode": verified, "decision_committed": true,
		"projection": h.dispatchApprovedAIDecisionProjection(r.Context(), body.Decision, body.SourceVersionID, body.CandidateID, decisionID, digest),
	})
}

func (h *EntityExtractionHTTPHandler) auth(next http.HandlerFunc) http.HandlerFunc {
	return tailnetServiceAuth(h.serviceTokenPath, "proffer entity extraction tailnet authorization required", next)
}

// tailnetServiceAuth preserves extraction callers while applying shared Proffer service admission.
// Inputs: mounted token, denial text and handler. Outputs: authenticated handler.
// Effects: token reload and canonical-write fencing. Choose for entity and conversation extraction.
func tailnetServiceAuth(serviceTokenPath, denied string, next http.HandlerFunc) http.HandlerFunc {
	return profferServiceAuth(serviceTokenPath, denied, next)
}

type runRequest struct {
	PreviewHandle string `json:"preview_handle"`
	MatterMode    string `json:"matter_mode"`
}

func (h *EntityExtractionHTTPHandler) resolveRun(ctx context.Context, handle, mode string, write bool) (flow.RunRef, int, error) {
	if !previewHandlePattern.MatchString(handle) {
		return flow.RunRef{}, http.StatusUnprocessableEntity, errors.New("preview_handle is invalid")
	}
	requested, modeErr := caseidentity.ParseMode(mode)
	if modeErr != nil {
		return flow.RunRef{}, http.StatusUnprocessableEntity, modeErr
	}
	if modeErr := caseidentity.RequireCanonicalWrite(requested); write && modeErr != nil {
		return flow.RunRef{}, http.StatusConflict, modeErr
	}
	run, err := h.store.ResolveRun(ctx, handle)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, service.ErrNotFound) {
			status = http.StatusNotFound
		}
		return flow.RunRef{}, status, err
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(run.MatterMode)); write && err != nil {
		return flow.RunRef{}, http.StatusConflict, err
	}
	if write && run.MatterMode != string(requested) {
		return flow.RunRef{}, http.StatusConflict, errors.New("request mode disagrees with the durable operating_mode receipt")
	}
	return run, 0, nil
}

func actorAndKey(r *http.Request) (entities.Actor, string, error) {
	uid, username, err := authenticatedActor(r)
	if err != nil {
		return entities.Actor{}, "", err
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 512 {
		return entities.Actor{}, "", errors.New("a bounded Idempotency-Key is required")
	}
	return entities.Actor{SubjectUID: uid, Username: username}, key, nil
}

// requestDigest binds a write to its key, its actor and its exact body.
func requestDigest(key string, actor entities.Actor, body any) string {
	canonical, _ := json.Marshal(body)
	sum := sha256.Sum256([]byte(key + "\x00" + actor.SubjectUID + "\x00" + string(canonical)))
	return hex.EncodeToString(sum[:])
}

func (h *EntityExtractionHTTPHandler) fail(w http.ResponseWriter, err error) {
	var invalid service.ErrInvalid
	switch {
	case errors.As(err, &invalid):
		previewError(w, http.StatusUnprocessableEntity, err)
	case errors.Is(err, entities.ErrConflict):
		previewError(w, http.StatusConflict, err)
	case errors.Is(err, service.ErrNotFound):
		previewError(w, http.StatusNotFound, err)
	default:
		previewError(w, http.StatusServiceUnavailable, err)
	}
}

type extractRequest struct {
	runRequest
	UseModel *bool `json:"use_model,omitempty"`
}

func (h *EntityExtractionHTTPHandler) extract(w http.ResponseWriter, r *http.Request) {
	var body extractRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	run, status, err := h.resolveRun(r.Context(), body.PreviewHandle, body.MatterMode, true)
	if err != nil {
		previewError(w, status, err)
		return
	}
	useModel := body.UseModel == nil || *body.UseModel
	request := flow.ExtractionRequest{
		ExtractionID: flow.DeterministicID("extraction", run.PreviewHandle, key, actor.SubjectUID),
		Run:          run, Actor: actor, UseModel: useModel, RequestedAt: h.clock().UTC(),
	}
	started, err := h.workflows.StartExtraction(r.Context(), request)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusAccepted, map[string]any{
		"workflow_id": started.WorkflowID, "run_id": started.RunID, "extraction_id": request.ExtractionID,
		"normalized_generation_id": run.GenerationID, "matter_mode": run.MatterMode, "use_model": useModel,
	})
}

func (h *EntityExtractionHTTPHandler) status(w http.ResponseWriter, r *http.Request, prefix string) {
	handle := r.URL.Query().Get("preview_handle")
	workflowID := r.PathValue("workflow_id")
	if !previewHandlePattern.MatchString(handle) || !strings.HasPrefix(workflowID, prefix+handle+":") || len(workflowID) > 512 {
		previewError(w, http.StatusNotFound, errors.New("workflow does not belong to this run"))
		return
	}
	progress, err := h.workflows.Status(r.Context(), workflowID)
	if err != nil {
		h.fail(w, err)
		return
	}
	steps := progress.Steps
	if steps == nil {
		// A workflow that is not queryable yet has no steps; the contract is a list, never null.
		steps = []flow.StepResult{}
	}
	previewJSON(w, http.StatusOK, map[string]any{"workflow_id": workflowID, "outcome": progress.Outcome, "steps": steps})
}

func (h *EntityExtractionHTTPHandler) extractionStatus(w http.ResponseWriter, r *http.Request) {
	h.status(w, r, "entity-extraction:")
}

func (h *EntityExtractionHTTPHandler) commitStatus(w http.ResponseWriter, r *http.Request) {
	h.status(w, r, "extraction-commit:")
}

// maxMentionsShown bounds each proposal's model mentions in API responses;
// the full set stays in staging and is what a commit writes.
const maxMentionsShown = 25

func (h *EntityExtractionHTTPHandler) proposals(w http.ResponseWriter, r *http.Request) {
	run, status, err := h.resolveRun(r.Context(), r.URL.Query().Get("preview_handle"), r.URL.Query().Get("matter_mode"), false)
	if err != nil {
		previewError(w, status, err)
		return
	}
	current, err := h.store.CurrentEntities(r.Context(), run.GenerationID)
	if err != nil {
		h.fail(w, err)
		return
	}
	currentEvents, err := h.store.CurrentEvents(r.Context(), run.GenerationID)
	if err != nil {
		h.fail(w, err)
		return
	}
	runs, err := h.store.ExtractionRuns(r.Context(), run.GenerationID)
	if err != nil {
		h.fail(w, err)
		return
	}
	if runs == nil {
		// A run nobody has extracted yet: an empty list, never null.
		runs = []service.RunSummary{}
	}
	type entityView struct {
		entities.Proposal
		ModelMentionCount int      `json:"model_mention_count"`
		Keys              []string `json:"keys"`
	}
	entityViews := make([]entityView, 0, len(current))
	for _, proposal := range current {
		count := len(proposal.ModelMentions)
		if count > maxMentionsShown {
			proposal.ModelMentions = proposal.ModelMentions[:maxMentionsShown]
		}
		keys := proposal.Keys()
		proposal.Aliases, proposal.Extractors, keys = nonNil(proposal.Aliases), nonNil(proposal.Extractors), nonNil(keys)
		entityViews = append(entityViews, entityView{Proposal: proposal, ModelMentionCount: count, Keys: keys})
	}
	type eventView struct {
		events.Proposal
		SourceAvailableFrom *time.Time `json:"source_available_from,omitempty"`
		ResolvedEntities    []string   `json:"resolved_entity_candidate_ids"`
		UnresolvedKeys      []string   `json:"unresolved_entity_keys,omitempty"`
	}
	eventViews := make([]eventView, 0, len(currentEvents))
	for _, event := range currentEvents {
		resolved, unresolved, ambiguous := events.ResolveEntityKeys(event, current)
		ids := make([]string, 0, len(resolved))
		for _, id := range resolved {
			ids = append(ids, id)
		}
		event.SourceRecords, event.Extractors = nonNil(event.SourceRecords), nonNil(event.Extractors)
		eventViews = append(eventViews, eventView{Proposal: event, SourceAvailableFrom: event.SourceAvailableFrom(), ResolvedEntities: uniqueSorted(ids), UnresolvedKeys: append(unresolved, ambiguous...)})
	}
	var aliasKinds []string
	for _, kind := range []entities.AliasKind{entities.AliasNickname, entities.AliasLegal, entities.AliasMaiden, entities.AliasMisspelling, entities.AliasPhonetic, entities.AliasInitials, entities.AliasOther} {
		aliasKinds = append(aliasKinds, string(kind))
	}
	previewJSON(w, http.StatusOK, map[string]any{
		"preview_handle": run.PreviewHandle, "normalized_generation_id": run.GenerationID, "matter_mode": run.MatterMode,
		"entities": entityViews, "events": eventViews, "extractions": runs,
		"entity_types": entities.RegistryTypes(), "event_types": events.EventTypes, "alias_kinds": aliasKinds,
	})
}

// nonNil keeps list fields lists on the wire: the panel maps over them.
func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

type correctionRequest struct {
	runRequest
	service.CorrectionEnvelope
}

func (h *EntityExtractionHTTPHandler) correct(w http.ResponseWriter, r *http.Request) {
	var body correctionRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	run, status, err := h.resolveRun(r.Context(), body.PreviewHandle, body.MatterMode, true)
	if err != nil {
		previewError(w, status, err)
		return
	}
	digest := requestDigest(key, actor, body)
	if err := service.Correct(r.Context(), h.store, run, body.CorrectionEnvelope, actor, h.clock().UTC(), digest); err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, map[string]any{"ok": true, "normalized_generation_id": run.GenerationID})
}

type markRequest struct {
	runRequest
	events.MarkRequest
}

func (h *EntityExtractionHTTPHandler) markEvent(w http.ResponseWriter, r *http.Request) {
	var body markRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	run, status, err := h.resolveRun(r.Context(), body.PreviewHandle, body.MatterMode, true)
	if err != nil {
		previewError(w, status, err)
		return
	}
	event, err := service.MarkEvent(r.Context(), h.store, run, body.MarkRequest, actor, h.clock().UTC(), requestDigest(key, actor, body))
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusCreated, map[string]any{"event": event, "source_available_from": event.SourceAvailableFrom()})
}

func (h *EntityExtractionHTTPHandler) record(w http.ResponseWriter, r *http.Request) {
	run, status, err := h.resolveRun(r.Context(), r.URL.Query().Get("preview_handle"), "", false)
	if err != nil {
		previewError(w, status, err)
		return
	}
	message, err := h.store.MessageByID(r.Context(), run.GenerationID, r.PathValue("record_id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, message)
}

func (h *EntityExtractionHTTPHandler) registry(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 200 {
		previewError(w, http.StatusUnprocessableEntity, errors.New("q is at most 200 characters"))
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	found, err := h.store.SearchRegistry(r.Context(), query, limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	if found == nil {
		found = []entities.RegistryEntity{}
	}
	previewJSON(w, http.StatusOK, map[string]any{"entities": found})
}

func (h *EntityExtractionHTTPHandler) validate(w http.ResponseWriter, r *http.Request) {
	var body runRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	run, status, err := h.resolveRun(r.Context(), body.PreviewHandle, body.MatterMode, true)
	if err != nil {
		previewError(w, status, err)
		return
	}
	if run.MatterMode == "" {
		previewError(w, http.StatusUnprocessableEntity, errors.New("matter_mode is required"))
		return
	}
	report, err := service.Validate(r.Context(), h.store, run)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, report)
}

type commitRequest struct {
	runRequest
	Digest string `json:"digest"`
}

func (h *EntityExtractionHTTPHandler) commit(w http.ResponseWriter, r *http.Request) {
	var body commitRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	run, status, err := h.resolveRun(r.Context(), body.PreviewHandle, body.MatterMode, true)
	if err != nil {
		previewError(w, status, err)
		return
	}
	if run.MatterMode == "" {
		previewError(w, http.StatusUnprocessableEntity, errors.New("matter_mode is required"))
		return
	}
	report, err := service.Validate(r.Context(), h.store, run)
	if err != nil {
		h.fail(w, err)
		return
	}
	if !report.OK || report.Digest != body.Digest {
		reason := "validation failed; fix the failing checks and validate again"
		if report.OK {
			reason = "the proposals changed since they were validated; validate again"
		}
		previewJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": reason, "report": report})
		return
	}
	request := flow.CommitRequest{
		CommitID: flow.DeterministicID("commit", run.PreviewHandle, report.Digest, key),
		Run:      run, Actor: actor, Digest: report.Digest, RequestedAt: h.clock().UTC(),
		CollectionSlug: flow.DefaultCollectionSlug, ProjectionTaskQueue: flow.DefaultProjectionTaskQueue,
	}
	started, err := h.workflows.StartCommit(r.Context(), request)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusAccepted, map[string]any{"workflow_id": started.WorkflowID, "run_id": started.RunID, "commit_id": request.CommitID, "counts": report.Counts})
}

// Report type re-exported for the starter's contract tests.
type Report = commitcheck.Report
