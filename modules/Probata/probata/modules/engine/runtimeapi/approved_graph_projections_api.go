// Byline: Codex · GPT-6 · 2026-10-08.
package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// ApprovedProjectionLister is the bounded completed-checkpoint read seam.
type ApprovedProjectionLister interface {
	ListApprovedProjections(context.Context, surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error)
}

// ApprovedGraphProjectionsHTTPHandler serves source-pinned revision descriptors for the fixed service policy.
type ApprovedGraphProjectionsHTTPHandler struct {
	lister           ApprovedProjectionLister
	serviceTokenPath string
}

// NewApprovedGraphProjectionsHTTPHandler validates the existing service credential and listing seam.
// Inputs: configured analytical lister and mounted service token path. Outputs: guarded handler.
// Effects: reads token for startup validation; choose for completed projection discovery.
func NewApprovedGraphProjectionsHTTPHandler(lister ApprovedProjectionLister, serviceTokenPath string) (*ApprovedGraphProjectionsHTTPHandler, error) {
	if lister == nil {
		return nil, errors.New("approved graph projections require a configured analytical lister")
	}
	if _, err := loadServiceToken(serviceTokenPath); err != nil {
		return nil, err
	}
	return &ApprovedGraphProjectionsHTTPHandler{lister: lister, serviceTokenPath: serviceTokenPath}, nil
}

// Routes mounts only the completed projection GET path beside existing query routes.
// Inputs: requests with existing service token and actor headers. Outputs: guarded HTTP route.
// Effects: route registration; choose in the shared starter without replacing its analysis subtree.
func (h *ApprovedGraphProjectionsHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /reference-import/analysis/projections", tailnetServiceAuth(h.serviceTokenPath, "approved graph projection tailnet authorization required", h.list))
	return mux
}

// list validates one actor and canonical case before returning bounded source-pinned descriptors.
// Inputs: matter_id, court_case_id, optional limit and cursor. Outputs: page JSON or HTTP error.
// Effects: read-only analytical listing; choose before an exact revision-scoped query action.
func (h *ApprovedGraphProjectionsHTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	if _, _, err := authenticatedActor(r); err != nil {
		previewError(w, http.StatusUnauthorized, err)
		return
	}
	query := r.URL.Query()
	for key, values := range query {
		if (key != "matter_id" && key != "court_case_id" && key != "limit" && key != "cursor") || len(values) != 1 {
			previewError(w, http.StatusBadRequest, errors.New("approved graph projection query parameters are invalid"))
			return
		}
	}
	scope := surrealsink.ApprovedProjectionScope{MatterID: strings.TrimSpace(query.Get("matter_id")), CourtCaseID: strings.TrimSpace(query.Get("court_case_id")), Cursor: query.Get("cursor")}
	if !caseidentity.AdmittedIdentity(scope.MatterID, scope.CourtCaseID) || scope.MatterID == "" || scope.CourtCaseID == "" {
		previewError(w, http.StatusUnprocessableEntity, errors.New("canonical matter and court case are required"))
		return
	}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 50 {
			previewError(w, http.StatusUnprocessableEntity, errors.New("projection limit must be 1 through 50"))
			return
		}
		scope.Limit = limit
	}
	if len(scope.Cursor) > 1024 {
		previewError(w, http.StatusUnprocessableEntity, errors.New("projection cursor exceeds bound"))
		return
	}
	page, err := h.lister.ListApprovedProjections(r.Context(), scope)
	if err != nil {
		if errors.Is(err, surrealsink.ErrApprovedProjectionInput) {
			previewError(w, http.StatusUnprocessableEntity, err)
			return
		}
		previewError(w, http.StatusServiceUnavailable, err)
		return
	}
	if page.Items == nil {
		page.Items = []surrealsink.ApprovedProjectionDescriptor{}
	}
	previewJSON(w, http.StatusOK, page)
}
