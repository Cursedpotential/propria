// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-26
//
// Context review routes on the Proffer starter. Same boundary as every other
// starter route: tailnet peer + mounted service token, Authentik actor headers
// on every owner act, an Idempotency-Key on every write.
//
//	GET  /reference-import/previews/{preview_handle}/messages/{message_id}/context-review?horizon=
//	POST /reference-import/previews/{preview_handle}/messages/{message_id}/context-review
//	POST /reference-import/previews/{preview_handle}/messages/{message_id}/foreshadowing
//
// KNOWLEDGE HORIZON: the read defaults to as_lived. The as-lived answer never
// carries the foreshadowing key, and the store's as-lived read plan never
// queries the flag table. Only an explicit horizon=hindsight (the owner's own
// Review surface) reads it.
package runtimeapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/contextreview"
)

// maxOverlayRequestBytes bounds one overlay body: 2 x 32 names of at most 200
// runes (4 bytes each) plus a 4000-byte reason fit comfortably.
const maxOverlayRequestBytes int64 = 96 << 10

// ContextReviewHTTPHandler serves the context review overlay routes.
type ContextReviewHTTPHandler struct {
	store            contextreview.Store
	serviceTokenPath string
}

// NewContextReviewHTTPHandler validates its store and the service token.
func NewContextReviewHTTPHandler(store contextreview.Store, serviceTokenPath string) (*ContextReviewHTTPHandler, error) {
	if store == nil {
		return nil, errors.New("context review handler requires a store")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &ContextReviewHTTPHandler{store: store, serviceTokenPath: serviceTokenPath}, nil
}

// Routes returns the context review mux.
func (h *ContextReviewHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	const base = "/reference-import/previews/{preview_handle}/messages/{message_id}"
	mux.HandleFunc("GET "+base+"/context-review", overlayAuth(h.serviceTokenPath, "context review", h.read))
	mux.HandleFunc("POST "+base+"/context-review", overlayAuth(h.serviceTokenPath, "context review", h.writeReview))
	mux.HandleFunc("POST "+base+"/foreshadowing", overlayAuth(h.serviceTokenPath, "context review", h.writeForeshadowing))
	return mux
}

// overlayAuth applies shared direct-peer/service-token admission and canonical-write fencing.
// Inputs: mounted token, route label, next handler. Outputs: authenticated handler.
// Effects: per-request credential reads; downstream actor/idempotency checks remain unchanged.
// Choose for Review, metadata, investigation, legal context and Case routes.
func overlayAuth(serviceTokenPath, label string, next http.HandlerFunc) http.HandlerFunc {
	return profferServiceAuth(serviceTokenPath, "proffer "+label+" service authorization required", next)
}

// decodeOverlayJSON reads exactly one bounded JSON object with no unknown fields.
func decodeOverlayJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxOverlayRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain exactly one JSON object")
		}
		return err
	}
	return nil
}

// overlayActor returns the Authentik actor and the Idempotency-Key.
func overlayActor(r *http.Request) (uid, username, key string, err error) {
	uid, username, err = authenticatedActor(r)
	if err != nil {
		return "", "", "", err
	}
	key = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if err := contextreview.ValidateActor(uid, username, key); err != nil {
		return "", "", "", err
	}
	return uid, username, key, nil
}

func (h *ContextReviewHTTPHandler) subject(r *http.Request) (contextreview.Subject, error) {
	subject := contextreview.Subject{PreviewHandle: r.PathValue("preview_handle"), MessageID: r.PathValue("message_id")}
	return subject, contextreview.ValidateSubject(subject)
}

func (h *ContextReviewHTTPHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, contextreview.ErrSubjectNotFound):
		previewError(w, http.StatusNotFound, err)
	case errors.Is(err, contextreview.ErrScopeMissing), errors.Is(err, contextreview.ErrStaleRevision),
		errors.Is(err, contextreview.ErrIdempotencyConflict):
		previewError(w, http.StatusConflict, err)
	default:
		previewError(w, http.StatusServiceUnavailable, err)
	}
}

func (h *ContextReviewHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	subject, err := h.subject(r)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	horizon, err := contextreview.ParseHorizon(r.URL.Query().Get("horizon"))
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	view, err := h.store.Read(r.Context(), subject, horizon)
	if err != nil {
		h.fail(w, err)
		return
	}
	// Defence in depth: whatever a store returns, an as-lived answer never
	// carries a foreshadowing member.
	if horizon != contextreview.HorizonHindsight {
		view.Foreshadowing = nil
		view.Horizon = contextreview.HorizonAsLived
	}
	if view.Reviews == nil {
		view.Reviews = []contextreview.ReviewRevision{}
	}
	previewJSON(w, http.StatusOK, view)
}

type contextReviewRequest struct {
	SupersedesRef string                `json:"supersedes_ref"`
	AddressedTo   []contextreview.Party `json:"addressed_to"`
	About         []contextreview.Party `json:"about"`
	AboutChild    *string               `json:"about_child"`
	Relevant      *bool                 `json:"relevant"`
	ChangeReason  string                `json:"change_reason"`
}

func (h *ContextReviewHTTPHandler) writeReview(w http.ResponseWriter, r *http.Request) {
	subject, err := h.subject(r)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	var body contextReviewRequest
	if err := decodeOverlayJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	uid, username, key, err := overlayActor(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	spec := contextreview.ReviewSpec{
		Subject: subject, SupersedesRef: strings.TrimSpace(body.SupersedesRef),
		Assertions: contextreview.Assertions{
			AddressedTo: body.AddressedTo, About: body.About, AboutChild: body.AboutChild, Relevant: body.Relevant,
		}.Normalized(),
		ChangeReason: body.ChangeReason, ActorSubjectUID: uid, ActorUsername: username, IdempotencyKey: key,
	}
	for _, check := range []error{
		contextreview.ValidateSupersedes(spec.SupersedesRef),
		contextreview.ValidateAssertions(spec.Assertions),
		contextreview.ValidateChange(spec.ChangeReason, ""),
	} {
		if check != nil {
			previewError(w, http.StatusUnprocessableEntity, check)
			return
		}
	}
	spec.ContentDigest = contextreview.ReviewDigest(spec)
	receipt, err := h.store.PersistReview(r.Context(), spec)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusCreated, receipt)
}

type foreshadowingRequest struct {
	SupersedesRef string `json:"supersedes_ref"`
	Foreshadowing *bool  `json:"foreshadowing"`
	Note          string `json:"note"`
	ChangeReason  string `json:"change_reason"`
}

func (h *ContextReviewHTTPHandler) writeForeshadowing(w http.ResponseWriter, r *http.Request) {
	subject, err := h.subject(r)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	var body foreshadowingRequest
	if err := decodeOverlayJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	uid, username, key, err := overlayActor(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	if body.Foreshadowing == nil {
		previewError(w, http.StatusUnprocessableEntity, errors.New("foreshadowing must be true or false"))
		return
	}
	spec := contextreview.ForeshadowingSpec{
		Subject: subject, SupersedesRef: strings.TrimSpace(body.SupersedesRef), Foreshadowing: *body.Foreshadowing,
		Note: body.Note, ChangeReason: body.ChangeReason, ActorSubjectUID: uid, ActorUsername: username, IdempotencyKey: key,
	}
	for _, check := range []error{
		contextreview.ValidateSupersedes(spec.SupersedesRef),
		contextreview.ValidateChange(spec.ChangeReason, spec.Note),
	} {
		if check != nil {
			previewError(w, http.StatusUnprocessableEntity, check)
			return
		}
	}
	spec.ContentDigest = contextreview.ForeshadowingDigest(spec)
	receipt, err := h.store.PersistForeshadowing(r.Context(), spec)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusCreated, receipt)
}
