package runtimeapi

import (
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/investigation"
	"net/http"
)

const InvestigationCreateRoutePattern = "POST /legal-context/investigations"
const InvestigationReadRoutePattern = "GET /legal-context/investigations/{request_id}"

type InvestigationHTTPHandler struct {
	store            investigation.Store
	serviceTokenPath string
}

func NewInvestigationHTTPHandler(store investigation.Store, path string) (*InvestigationHTTPHandler, error) {
	if store == nil {
		return nil, errors.New("investigation handler requires a store")
	}
	if _, e := loadServiceToken(path); e != nil {
		return nil, e
	}
	return &InvestigationHTTPHandler{store, path}, nil
}
func (h *InvestigationHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(InvestigationCreateRoutePattern, overlayAuth(h.serviceTokenPath, "investigation", h.create))
	mux.HandleFunc(InvestigationReadRoutePattern, overlayAuth(h.serviceTokenPath, "investigation", h.read))
	return mux
}
func (h *InvestigationHTTPHandler) fail(w http.ResponseWriter, e error) {
	code := 503
	public := errors.New("Probata investigation requests are unavailable")
	switch {
	case errors.Is(e, investigation.ErrConflict), errors.Is(e, investigation.ErrScope):
		code = 409
		public = e
	case errors.Is(e, investigation.ErrSource):
		code = 422
		public = e
	case errors.Is(e, investigation.ErrNotFound):
		code = 404
		public = e
	}
	previewError(w, code, public)
}
func (h *InvestigationHTTPHandler) create(w http.ResponseWriter, r *http.Request) {
	uid, name, e := authenticatedActor(r)
	if e != nil {
		previewError(w, 401, errors.New("authenticated actor required"))
		return
	}
	a := investigation.Actor{UID: uid, Username: name, Key: r.Header.Get("Idempotency-Key")}
	if e = investigation.ValidateActor(a); e != nil {
		previewError(w, 422, e)
		return
	}
	var request investigation.Request
	if e = decodeOverlayJSON(w, r, &request); e != nil {
		previewError(w, 422, errors.New("invalid investigation request JSON"))
		return
	}
	if e = investigation.Validate(request); e != nil {
		previewError(w, 422, e)
		return
	}
	receipt, e := h.store.Create(r.Context(), request, a)
	if e != nil {
		h.fail(w, e)
		return
	}
	previewJSON(w, 200, receipt)
}
func (h *InvestigationHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	if _, _, e := authenticatedActor(r); e != nil {
		previewError(w, 401, errors.New("authenticated actor required"))
		return
	}
	scope := investigation.Scope{Mode: caseidentity.Mode(r.URL.Query().Get("mode")), MatterID: r.URL.Query().Get("matter_id"), CourtCaseID: r.URL.Query().Get("court_case_id")}
	if e := investigation.ValidateScope(scope); e != nil {
		previewError(w, 422, e)
		return
	}
	if !investigation.ValidID(r.PathValue("request_id")) {
		previewError(w, 422, errors.New("request_id requires a canonical non-nil UUID"))
		return
	}
	receipt, e := h.store.Read(r.Context(), r.PathValue("request_id"), scope)
	if e != nil {
		h.fail(w, e)
		return
	}
	previewJSON(w, 200, receipt)
}
