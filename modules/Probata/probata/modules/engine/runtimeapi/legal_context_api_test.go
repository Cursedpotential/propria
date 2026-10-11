// Byline: Codex · GPT-5 · 2026-10-05 (canonical mode fixtures)
package runtimeapi

import (
	"context"
	"errors"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

type legalContextStub struct {
	calls int
	query platformpostgres.LegalContextQuery
	err   error
}

func (s *legalContextStub) ReadLegalContext(_ context.Context, q platformpostgres.LegalContextQuery) (platformpostgres.LegalContextResult, error) {
	s.calls++
	s.query = q
	return platformpostgres.LegalContextResult{Available: true, Mode: q.Mode, Records: []platformpostgres.LegalContextRecord{}}, s.err
}

func TestLegalContextAuthValidationAndScopedRead(t *testing.T) {
	store := &legalContextStub{}
	handler, err := NewLegalContextHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, err)
	routes := handler.Routes()
	outside := newPreviewRequest(http.MethodGet, "/legal-context/records?mode=REAL&kind=event", nil)
	outside.RemoteAddr = "203.0.113.9:4444"
	require.Equal(t, 401, servePreviewRequest(routes, outside).Code)
	wrongToken := newPreviewRequest(http.MethodGet, "/legal-context/records?mode=REAL&kind=event", nil)
	wrongToken.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
	require.Equal(t, 401, servePreviewRequest(routes, wrongToken).Code)
	for _, query := range []string{"mode=other&kind=event", "mode=REAL&kind=assertion", "mode=REAL&kind=event&limit=0", "mode=REAL&kind=event&limit=101", "mode=REAL&kind=event&record_id=not-an-id", "mode=REAL&kind=event&q=" + strings.Repeat("a", 201)} {
		require.Equal(t, 422, servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/legal-context/records?"+query, nil)).Code, query)
	}
	require.Zero(t, store.calls)
	response := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/legal-context/records?mode=TEST&kind=event&q=exchange&limit=5", nil))
	require.Equal(t, 200, response.Code, response.Body.String())
	require.JSONEq(t, `{"available":true,"mode":"DEV","matter_id":"","court_case_id":"","records":[],"truncated":false}`, response.Body.String())
	require.Equal(t, "DEV", string(store.query.Mode))
	require.Equal(t, 5, store.query.Limit)
	require.Equal(t, "exchange", store.query.Search)
	defaultResponse := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/legal-context/records?kind=event", nil))
	require.Equal(t, 200, defaultResponse.Code)
	require.Equal(t, "LIVE", string(store.query.Mode))
	store.err = errors.New("sensitive database error")
	failed := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/legal-context/records?mode=REAL&kind=entity", nil))
	require.Equal(t, 503, failed.Code)
	require.NotContains(t, failed.Body.String(), "sensitive")
}
