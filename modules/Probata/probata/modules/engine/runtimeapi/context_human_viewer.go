package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/Cursedpotential/probata/engine/postgres"
	"github.com/google/uuid"
)

// HumanSourceViewStore resolves native source and stage metadata without normalized-generation dependencies.
type HumanSourceViewStore interface {
	View(context.Context, string) (postgres.HumanSourceView, error)
}

// HumanContextViewerHandler serves current native human bundles to the SBV-derived Workbench viewer.
// Inputs: existing tailnet service token path, native metadata repository and read-only shared bundle root.
// Outputs: reference-only snapshot plus bounded native content pages; effects: read-only metadata/files.
type HumanContextViewerHandler struct {
	Metadata         HumanSourceViewStore
	Bundles          *humancontext.Bundles
	ServiceTokenPath string
}

// NewHumanContextViewerHandler validates the native metadata/bundle seam and existing service token.
// Inputs: metadata reader, read-only mounted bundle root and existing token path. Outputs: additive handler.
// Effects: existing token validation only; choose beside ContextSourceHTTPHandler without a second service.
func NewHumanContextViewerHandler(metadata HumanSourceViewStore, bundles *humancontext.Bundles, tokenPath string) (*HumanContextViewerHandler, error) {
	if metadata == nil || bundles == nil || strings.TrimSpace(bundles.Root) == "" {
		return nil, errors.New("human viewer dependencies unavailable")
	}
	if _, err := loadServiceToken(tokenPath); err != nil {
		return nil, err
	}
	return &HumanContextViewerHandler{metadata, bundles, tokenPath}, nil
}

// Routes mounts native human snapshots and pages beside existing context start/status routes.
// Inputs: real human source-version handles. Outputs: genuine native bundle projections without fake legacy receipts.
// Effects: registration only; existing tailnet service authentication is reused without new groups or approvals.
func (h *HumanContextViewerHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /reference-import/context/previews/{handle}", tailnetServiceAuth(h.ServiceTokenPath, "context source tailnet authorization required", h.snapshot))
	mux.HandleFunc("GET /reference-import/context/previews/{handle}/content", tailnetServiceAuth(h.ServiceTokenPath, "context source tailnet authorization required", h.content))
	return mux
}
func (h *HumanContextViewerHandler) view(r *http.Request) (postgres.HumanSourceView, error) {
	if h.Metadata == nil || h.Bundles == nil {
		return postgres.HumanSourceView{}, errors.New("human viewer unavailable")
	}
	id := strings.TrimPrefix(r.PathValue("handle"), "human-")
	if _, err := uuid.Parse(id); err != nil {
		return postgres.HumanSourceView{}, errors.New("invalid human source handle")
	}
	return h.Metadata.View(r.Context(), id)
}

// snapshot returns durable native stage references without fetching source bodies.
// Inputs: actual source handle. Outputs: native snapshot; effects: metadata reads only.
func (h *HumanContextViewerHandler) snapshot(w http.ResponseWriter, r *http.Request) {
	view, err := h.view(r)
	if err != nil {
		previewError(w, http.StatusNotFound, err)
		return
	}
	previewJSON(w, http.StatusOK, view)
}

// content reads one bounded native record page from the latest completed extraction/index bundle.
// Inputs: source handle, integer offset and bounded limit. Outputs: records and continuation, no attachment payload copies.
// Effects: read-only metadata and shared bundle files; unsupported/malformed records retain their actual status.
func (h *HumanContextViewerHandler) content(w http.ResponseWriter, r *http.Request) {
	view, err := h.view(r)
	if err != nil {
		previewError(w, http.StatusNotFound, err)
		return
	}
	if view.ManifestRef == "" {
		previewError(w, http.StatusConflict, errors.New("human extraction is not ready"))
		return
	}
	manifest, err := h.Bundles.ReadManifest(view.ManifestRef)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if manifest.Source.SourceVersionRef != view.SourceVersionRef || manifest.Source.SourceRef != view.SourceRef || manifest.WorkflowID != view.WorkflowID {
		previewError(w, http.StatusConflict, errors.New("human viewer source pins differ"))
		return
	}
	offset := int64(0)
	limit := 25
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			previewError(w, 400, err)
			return
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			previewError(w, 400, err)
			return
		}
	}
	if r.URL.Query().Get("section") == "chunks" {
		page, err := h.Bundles.ReadChunkPage(r.Context(), view.ManifestRef, offset, limit)
		if err != nil {
			previewError(w, http.StatusUnprocessableEntity, err)
			return
		}
		previewJSON(w, http.StatusOK, page)
		return
	}
	page, err := h.Bundles.ReadPage(r.Context(), view.ManifestRef, offset, limit)
	if err != nil {
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	}
	previewJSON(w, http.StatusOK, page)
}
