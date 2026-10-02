package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/google/uuid"
)

const LegalContextRoutePattern = "GET /legal-context/records"

type LegalContextReader interface {
	ReadLegalContext(context.Context, platformpostgres.LegalContextQuery) (platformpostgres.LegalContextResult, error)
}
type LegalContextHTTPHandler struct {
	store            LegalContextReader
	serviceTokenPath string
}

func NewLegalContextHTTPHandler(store LegalContextReader, tokenPath string) (*LegalContextHTTPHandler, error) {
	if store == nil {
		return nil, errors.New("legal context requires a reader")
	}
	if _, err := loadServiceToken(tokenPath); err != nil {
		return nil, err
	}
	return &LegalContextHTTPHandler{store, tokenPath}, nil
}

func (h *LegalContextHTTPHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(LegalContextRoutePattern, overlayAuth(h.serviceTokenPath, "legal context", h.read))
	return mux
}

func (h *LegalContextHTTPHandler) read(w http.ResponseWriter, r *http.Request) {
	mode, err := caseidentity.ParseMode(r.URL.Query().Get("mode"))
	if err != nil {
		previewError(w, 422, err)
		return
	}
	query := platformpostgres.LegalContextQuery{Mode: mode, Kind: r.URL.Query().Get("kind"), Search: strings.TrimSpace(r.URL.Query().Get("q")), RecordID: r.URL.Query().Get("record_id"), Limit: 30}
	if query.Kind != "entity" && query.Kind != "event" {
		previewError(w, 422, errors.New("kind must be entity or event"))
		return
	}
	if len(query.Search) > 200 {
		previewError(w, 422, errors.New("q is at most 200 bytes"))
		return
	}
	if query.RecordID != "" {
		if id, err := uuid.Parse(query.RecordID); err != nil || id.String() != query.RecordID || id == uuid.Nil {
			previewError(w, 422, errors.New("record_id must be a canonical non-nil UUID"))
			return
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		query.Limit, err = strconv.Atoi(raw)
		if err != nil || query.Limit < 1 || query.Limit > 100 {
			previewError(w, 422, errors.New("limit must be 1 through 100"))
			return
		}
	}
	result, err := h.store.ReadLegalContext(r.Context(), query)
	if err != nil {
		previewError(w, 503, errors.New("Probata context records are unavailable"))
		return
	}
	previewJSON(w, 200, result)
}
