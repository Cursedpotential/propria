// Byline: Codex · GPT-5 · 2026-10-05 (canonical admission fixtures)
package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/investigation"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

type investigationStub struct {
	calls int
	err   error
}

func (s *investigationStub) Create(_ context.Context, r investigation.Request, a investigation.Actor) (investigation.Receipt, error) {
	s.calls++
	return investigation.Receipt{Request: r, RequestID: a.Key, Status: "received", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Results: []investigation.Result{}}, s.err
}
func (s *investigationStub) Read(_ context.Context, id string, scope investigation.Scope) (investigation.Receipt, error) {
	s.calls++
	return investigation.Receipt{RequestID: id, Request: investigation.Request{Scope: scope}, Status: "received", Results: []investigation.Result{}}, s.err
}
func TestInvestigationHTTPAdmissionAndErrors(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	store := &investigationStub{}
	handler, e := NewInvestigationHTTPHandler(store, serviceTokenPath(t))
	require.NoError(t, e)
	routes := handler.Routes()
	body, _ := json.Marshal(investigation.Request{Scope: investigation.Scope{Mode: caseidentity.ModeLive, MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID}, LegalMatterID: id, ClaimID: id, FollowupID: id, Question: "Find missing context", Sources: []investigation.Source{}})
	request := func(text string) *http.Request {
		r := newPreviewRequest("POST", "/legal-context/investigations", []byte(text))
		r.Header.Set("X-authentik-uid", "actor")
		r.Header.Set("X-authentik-username", "service")
		r.Header.Set("Idempotency-Key", id)
		return r
	}
	missing := request(string(body))
	missing.Header.Del("X-authentik-uid")
	require.Equal(t, 401, servePreviewRequest(routes, missing).Code)
	wrong := request(string(body))
	wrong.RemoteAddr = "192.0.2.1:9"
	require.Equal(t, 401, servePreviewRequest(routes, wrong).Code)
	missingKey := request(string(body))
	missingKey.Header.Del("Idempotency-Key")
	require.Equal(t, 422, servePreviewRequest(routes, missingKey).Code)
	for _, raw := range []string{"null", "{}", string(body) + " {}", `{"actor":"forged"}`, strings.Repeat(" ", int(maxOverlayRequestBytes)) + string(body)} {
		require.Equal(t, 422, servePreviewRequest(routes, request(raw)).Code, raw[:min(len(raw), 100)])
	}
	require.Zero(t, store.calls)
	response := servePreviewRequest(routes, request(string(body)))
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), `"status":"received"`)
	require.Contains(t, response.Body.String(), `"results":[]`)
	for _, tc := range []struct {
		err  error
		code int
	}{{investigation.ErrScope, 409}, {investigation.ErrConflict, 409}, {investigation.ErrSource, 422}, {investigation.ErrNotFound, 404}, {errors.New("password database detail"), 503}} {
		store.err = tc.err
		response = servePreviewRequest(routes, request(string(body)))
		require.Equal(t, tc.code, response.Code)
		require.NotContains(t, response.Body.String(), "password")
	}
	store.err = nil
	get := newPreviewRequest("GET", "/legal-context/investigations/"+id+"?mode=TEST&matter_id="+id+"&court_case_id="+id, nil)
	get.Header.Set("X-authentik-uid", "actor")
	get.Header.Set("X-authentik-username", "service")
	require.Equal(t, 200, servePreviewRequest(routes, get).Code)
	get.Header.Del("X-authentik-username")
	require.Equal(t, 401, servePreviewRequest(routes, get).Code)
}
