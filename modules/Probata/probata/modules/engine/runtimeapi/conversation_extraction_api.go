// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Conversation extraction and Surreal send routes on the Proffer starter. Same boundary as
// the other starter routes: tailnet peer + mounted service token, Authentik actor headers
// for every owner act, an Idempotency-Key on every write.
//
//	GET  /reference-import/extractors                         every selectable extractor
//	POST /reference-import/conversations/extract              start extraction_request_workflow
//	POST /reference-import/conversations/send-to-surreal      start send_to_surreal_workflow
//	GET  /reference-import/conversations/workflows/{id}       progress of either workflow
package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// ConversationWorkflows starts and reads the conversation-level workflows.
type ConversationWorkflows interface {
	StartConversationExtraction(context.Context, flow.RequestInput) (flow.Started, error)
	StartSend(context.Context, flow.SendInput) (flow.Started, error)
	Status(context.Context, string) (flow.Progress, error)
}

// ConversationPreflightStore reads exact target admission and workload metadata.
// Inputs: matter/ref/selection. Output: preflight; effects: reads only.
type ConversationPreflightStore interface {
	PreflightConversations(context.Context, string, []flow.ConversationRef, []string) (flow.ExtractionPreflight, error)
}

// ConversationExtractionHTTPHandler serves the conversation extraction and send routes.
type ConversationExtractionHTTPHandler struct {
	workflows        ConversationWorkflows
	serviceTokenPath string
	clock            func() time.Time
	preflightStore   ConversationPreflightStore
}

// SetPreflightStore connects the existing canonical store before serving routes.
// Input: read-only admission seam. Output: none; effects: composition only.
func (h *ConversationExtractionHTTPHandler) SetPreflightStore(store ConversationPreflightStore) {
	h.preflightStore = store
}

// NewConversationExtractionHTTPHandler validates its seams and the service token.
func NewConversationExtractionHTTPHandler(workflows ConversationWorkflows, serviceTokenPath string) (*ConversationExtractionHTTPHandler, error) {
	if workflows == nil {
		return nil, errors.New("conversation extraction handler requires a workflow client")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &ConversationExtractionHTTPHandler{workflows: workflows, serviceTokenPath: serviceTokenPath, clock: time.Now}, nil
}

// Routes returns the conversation extraction mux.
func (h *ConversationExtractionHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /reference-import/extractors", h.auth(h.extractors))
	mux.HandleFunc("POST /reference-import/conversations/extract", h.auth(h.extract))
	mux.HandleFunc("POST /reference-import/conversations/preflight", h.auth(h.preflight))
	mux.HandleFunc("GET /reference-import/conversations/history", h.auth(h.history))
	mux.HandleFunc("POST /reference-import/conversations/send-to-surreal", h.auth(h.send))
	mux.HandleFunc("GET /reference-import/conversations/workflows/{workflow_id}", h.auth(h.status))
	return mux
}

func (h *ConversationExtractionHTTPHandler) auth(next http.HandlerFunc) http.HandlerFunc {
	return tailnetServiceAuth(h.serviceTokenPath, "proffer conversation extraction tailnet authorization required", next)
}

// extractors lists every selectable extractor and names the default.
func (h *ConversationExtractionHTTPHandler) extractors(w http.ResponseWriter, _ *http.Request) {
	previewJSON(w, http.StatusOK, map[string]any{"extractors": flow.Registry, "default": flow.DefaultExtractorID})
}

type conversationRequest struct {
	PreflightToken string                 `json:"preflight_token,omitempty"`
	CourtCaseID    string                 `json:"court_case_id,omitempty"`
	OperatingMode  string                 `json:"operating_mode,omitempty"`
	MatterID       string                 `json:"matter_id"`
	Conversations  []flow.ConversationRef `json:"conversations"`
	Extractors     []string               `json:"extractors"`
	// IncludeExtractions applies to the send only; the default is true.
	IncludeExtractions *bool `json:"include_extractions,omitempty"`
}

func (h *ConversationExtractionHTTPHandler) decode(w http.ResponseWriter, r *http.Request) (conversationRequest, bool) {
	var body conversationRequest
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return body, false
	}
	if _, err := uuid.Parse(strings.TrimSpace(body.MatterID)); err != nil {
		previewError(w, http.StatusUnprocessableEntity, errors.New("matter_id must be a uuid"))
		return body, false
	}
	mode, err := caseidentity.ParseMode(body.OperatingMode)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return body, false
	}
	if err := caseidentity.RequireCanonicalWrite(mode); err != nil {
		previewError(w, http.StatusConflict, err)
		return body, false
	}
	// Legacy callers used the existing fixed court scope; new callers forward the full pair explicitly.
	if body.CourtCaseID == "" {
		body.CourtCaseID = caseidentity.AuthoritativeCourtCaseID
	}
	if !caseidentity.AdmittedIdentity(body.MatterID, body.CourtCaseID) {
		previewError(w, http.StatusUnprocessableEntity, errors.New("conversation request requires the approved matter"))
		return body, false
	}
	body.OperatingMode = string(mode)
	return body, true
}

// extract starts extraction_request_workflow for the chosen conversations and extractors.
func (h *ConversationExtractionHTTPHandler) extract(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decode(w, r)
	if !ok {
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	input := flow.RequestInput{
		OperatingMode: body.OperatingMode, CourtCaseID: body.CourtCaseID,
		RequestID: flow.DeterministicID("conversation_extraction_request", key, actor.SubjectUID, body.MatterID),
		MatterID:  body.MatterID, Conversations: body.Conversations, Extractors: body.Extractors,
		Actor: actor, RequestedAt: h.clock().UTC(),
	}
	if err := input.Validate(); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if h.preflightStore == nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("Extraction admission is unavailable; no workflow was started"))
		return
	}
	check, err := h.preflightStore.PreflightConversations(r.Context(), body.MatterID, body.Conversations, body.Extractors)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("Extraction target admission could not be read; no workflow was started"))
		return
	}
	if !check.Eligible {
		previewJSON(w, http.StatusConflict, map[string]any{"error": "Extraction prerequisite is not satisfied; no workflow was started", "preflight": check})
		return
	}
	if body.PreflightToken == "" || body.PreflightToken != flow.PreflightToken(check) {
		previewError(w, http.StatusConflict, errors.New("Extraction selection or workload changed, or preflight confirmation is missing. Refresh preflight and confirm the exact target again; no workflow was started"))
		return
	}
	// Dispatch only the exact generation pins checked above. Conversation coordinates
	// remain metadata, so the workflow cannot silently resolve a newer generation.
	input.RequestedConversations = body.Conversations
	input.Conversations = nil
	seenRuns := map[string]bool{}
	for _, target := range check.Targets {
		for _, run := range target.Runs {
			key := run.SourceVersionID + ":" + run.GenerationID
			if !seenRuns[key] {
				input.Runs = append(input.Runs, run)
				seenRuns[key] = true
			}
		}
	}
	if len(input.Runs) == 0 {
		previewError(w, http.StatusConflict, errors.New("Extraction admission returned no pinned runs; no workflow was started"))
		return
	}
	started, err := h.workflows.StartConversationExtraction(r.Context(), input)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	specs, _ := flow.NormalizeExtractors(body.Extractors)
	chosen := make([]string, len(specs))
	for i, spec := range specs {
		chosen[i] = spec.ID
	}
	previewJSON(w, http.StatusAccepted, map[string]any{
		"workflow_id": started.WorkflowID, "run_id": started.RunID, "kind": "extraction",
		"conversations": len(body.Conversations), "extractors": chosen,
	})
}

// send starts send_to_surreal_workflow for the chosen conversations.
func (h *ConversationExtractionHTTPHandler) send(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decode(w, r)
	if !ok {
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	include := body.IncludeExtractions == nil || *body.IncludeExtractions
	input := flow.SendInput{
		OperatingMode: body.OperatingMode, CourtCaseID: body.CourtCaseID,
		RequestID: flow.DeterministicID("surreal_send_request", key, actor.SubjectUID, body.MatterID),
		MatterID:  body.MatterID, Conversations: body.Conversations, IncludeExtractions: include,
		Actor: actor, RequestedAt: h.clock().UTC(),
	}
	if err := input.Validate(); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	started, err := h.workflows.StartSend(r.Context(), input)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	previewJSON(w, http.StatusAccepted, map[string]any{
		"workflow_id": started.WorkflowID, "run_id": started.RunID, "kind": "send_to_surreal",
		"conversations": len(body.Conversations), "include_extractions": include,
	})
}

// status returns the progress of an extraction or a send workflow.
func (h *ConversationExtractionHTTPHandler) status(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("workflow_id")
	kind := ""
	switch {
	case strings.HasPrefix(id, flow.ConversationExtractionIDPrefix), strings.HasPrefix(id, flow.AutoExtractionWorkflowIDPrefix):
		kind = "extraction"
	case strings.HasPrefix(id, flow.SendToSurrealIDPrefix):
		kind = "send_to_surreal"
	}
	if kind == "" || len(id) > 512 {
		previewError(w, http.StatusNotFound, errors.New("unknown workflow"))
		return
	}
	progress, err := h.workflows.Status(r.Context(), id)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	steps := progress.Steps
	if steps == nil {
		steps = []flow.StepResult{}
	}
	receipts := progress.Receipts
	if receipts == nil {
		receipts = []flow.SendReceipt{}
	}
	previewJSON(w, http.StatusOK, map[string]any{"workflow_id": id, "run_id": progress.RunID, "kind": kind, "outcome": progress.Outcome, "steps": steps, "receipts": receipts})
}

// preflight reads selected workload and durable admission without starting work.
// Inputs: existing authenticated conversation request. Output: source-pinned bounds.
// Effects: SELECT only; pick before Extract and repeat admission at dispatch.
func (h *ConversationExtractionHTTPHandler) preflight(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decode(w, r)
	if !ok {
		return
	}
	input := flow.RequestInput{RequestID: "preflight", MatterID: body.MatterID, Conversations: body.Conversations, Extractors: body.Extractors}
	if err := input.Validate(); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if h.preflightStore == nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("Extraction admission reader is unavailable"))
		return
	}
	out, err := h.preflightStore.PreflightConversations(r.Context(), body.MatterID, body.Conversations, body.Extractors)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("Extraction admission could not be read"))
		return
	}
	out.Token = flow.PreflightToken(out)
	previewJSON(w, http.StatusOK, out)
}

// history reads retained extraction parents for one exact existing conversation.
// Inputs: approved matter and source/conversation pair. Output: bounded matching
// statuses and retention limits. Effects: Temporal reads only, never retries work.
func (h *ConversationExtractionHTTPHandler) history(w http.ResponseWriter, r *http.Request) {
	matter := r.URL.Query().Get("matter_id")
	ref := flow.ConversationRef{ExportKey: r.URL.Query().Get("export_key"), Conv: r.URL.Query().Get("conv")}
	if matter != caseidentity.AuthoritativeMatterID || ref.ExportKey == "" || len(ref.ExportKey) > 2048 || ref.Conv == "" || len(ref.Conv) > 2048 {
		previewError(w, http.StatusUnprocessableEntity, errors.New("exact approved matter and conversation required"))
		return
	}
	reader, ok := h.workflows.(interface {
		ConversationHistory(context.Context, string, flow.ConversationRef) (flow.ConversationHistory, error)
	})
	if !ok {
		previewError(w, http.StatusServiceUnavailable, errors.New("Durable extraction history reader is unavailable"))
		return
	}
	out, err := reader.ConversationHistory(r.Context(), matter, ref)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("Durable extraction history is unavailable; absence of attempts is not established"))
		return
	}
	previewJSON(w, http.StatusOK, out)
}
