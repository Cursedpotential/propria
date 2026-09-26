// Byline: Claude Code · Opus 5.5 · 2026-09-26
//
// Metadata screen routes on the Proffer starter (same boundary as the context
// review routes: tailnet peer + service token, Authentik actor on writes, an
// Idempotency-Key on every write).
//
//	GET  /reference-import/previews/{preview_handle}/metadata?subject_sha256=
//	POST /reference-import/previews/{preview_handle}/metadata/corrections
//
// The read returns what the platform durably recorded for one file of the run;
// it extracts nothing. A correction is an append-only overlay; the observed
// value is never written.
package runtimeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Cursedpotential/probata/engine/sourcemeta"
)

// SourceMetadataHTTPHandler serves the metadata screen routes.
type SourceMetadataHTTPHandler struct {
	store            sourcemeta.Store
	serviceTokenPath string
}

// NewSourceMetadataHTTPHandler validates its store and the service token.
func NewSourceMetadataHTTPHandler(store sourcemeta.Store, serviceTokenPath string) (*SourceMetadataHTTPHandler, error) {
	if store == nil {
		return nil, errors.New("source metadata handler requires a store")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &SourceMetadataHTTPHandler{store: store, serviceTokenPath: serviceTokenPath}, nil
}

// Routes returns the metadata mux.
func (h *SourceMetadataHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	const base = "/reference-import/previews/{preview_handle}/metadata"
	mux.HandleFunc("GET "+base, overlayAuth(h.serviceTokenPath, "metadata", h.read))
	mux.HandleFunc("POST "+base+"/corrections", overlayAuth(h.serviceTokenPath, "metadata", h.correct))
	return mux
}

func (h *SourceMetadataHTTPHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sourcemeta.ErrNotFound), errors.Is(err, sourcemeta.ErrSubjectNotInRun):
		previewError(w, http.StatusNotFound, err)
	case errors.Is(err, sourcemeta.ErrScopeMissing), errors.Is(err, sourcemeta.ErrStaleRevision),
		errors.Is(err, sourcemeta.ErrIdempotencyConflict):
		previewError(w, http.StatusConflict, err)
	default:
		previewError(w, http.StatusServiceUnavailable, err)
	}
}

func (h *SourceMetadataHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("preview_handle")
	subject := strings.TrimSpace(r.URL.Query().Get("subject_sha256"))
	if err := sourcemeta.ValidatePreviewHandle(handle); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err := sourcemeta.ValidateSHA256(subject, false); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	view, err := h.store.Read(r.Context(), handle, subject)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusOK, view)
}

type metadataCorrectionRequest struct {
	SubjectSHA256  string          `json:"subject_sha256"`
	FieldKey       string          `json:"field_key"`
	SupersedesRef  string          `json:"supersedes_ref"`
	Action         string          `json:"action"`
	SourceValue    json.RawMessage `json:"source_value"`
	CorrectedValue json.RawMessage `json:"corrected_value"`
	ChangeReason   string          `json:"change_reason"`
}

func (h *SourceMetadataHTTPHandler) correct(w http.ResponseWriter, r *http.Request) {
	var body metadataCorrectionRequest
	if err := decodeOverlayJSON(w, r, &body); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	uid, username, key, err := overlayActor(r)
	if err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	spec := sourcemeta.CorrectionSpec{
		PreviewHandle: r.PathValue("preview_handle"), SubjectSHA256: strings.TrimSpace(body.SubjectSHA256),
		FieldKey: body.FieldKey, SupersedesRef: strings.TrimSpace(body.SupersedesRef), Action: body.Action,
		SourceValue: body.SourceValue, CorrectedValue: body.CorrectedValue, ChangeReason: body.ChangeReason,
		ActorSubjectUID: uid, ActorUsername: username, IdempotencyKey: key,
	}
	if err := sourcemeta.ValidateCorrection(spec); err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	spec.ContentDigest = sourcemeta.CorrectionDigest(spec)
	receipt, err := h.store.PersistCorrection(r.Context(), spec)
	if err != nil {
		h.fail(w, err)
		return
	}
	previewJSON(w, http.StatusCreated, receipt)
}
