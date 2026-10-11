// Byline: Codex · GPT-6.1-sol · 2026-10-06
package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/stretchr/testify/require"
)

type caseScopeStoreStub struct {
	caseIdentityStoreStub
	scopeCalls int
	deadline   time.Time
	view       caseidentity.ScopeView
	scopeErr   error
}

func (s *caseScopeStoreStub) ReadScope(ctx context.Context, mode caseidentity.Mode) (caseidentity.ScopeView, error) {
	s.scopeCalls++
	s.deadline, _ = ctx.Deadline()
	view := s.view
	view.Mode = mode
	return view, s.scopeErr
}

func newScopeHandler(t *testing.T) (*caseScopeStoreStub, http.Handler) {
	t.Helper()
	s := &caseScopeStoreStub{view: caseidentity.ScopeView{
		Matter:    caseidentity.ScopeMatter{ID: caseidentity.AuthoritativeMatterID},
		CourtCase: caseidentity.ScopeCourtCase{ID: caseidentity.AuthoritativeCourtCaseID, MatterID: caseidentity.AuthoritativeMatterID},
	}}
	h, err := NewCaseIdentityHTTPHandler(s, serviceTokenPath(t))
	require.NoError(t, err)
	return s, h.Routes()
}

func TestCaseScopeReadFreshModesReturnOnlyBoundedIdentity(t *testing.T) {
	s, routes := newScopeHandler(t)
	for _, test := range []struct{ query, mode string }{{"", "LIVE"}, {"?mode=DEV", "DEV"}, {"?mode=LIVE", "LIVE"}, {"?mode=TEST", "DEV"}, {"?mode=REAL", "LIVE"}} {
		before := time.Now()
		r := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity/scope"+test.query, nil))
		require.Equal(t, http.StatusOK, r.Code, r.Body.String())
		require.JSONEq(t, `{"mode":"`+test.mode+`","matter":{"id":"`+caseidentity.AuthoritativeMatterID+`"},"court_case":{"id":"`+caseidentity.AuthoritativeCourtCaseID+`","matter_id":"`+caseidentity.AuthoritativeMatterID+`"}}`, r.Body.String())
		require.Less(t, r.Body.Len(), 512)
		require.WithinDuration(t, before.Add(caseidentity.ScopeReadTimeout), s.deadline, 100*time.Millisecond)
	}
	require.Equal(t, 5, s.scopeCalls) // Every request reads again; no cached approval.
	require.Zero(t, s.readCalls)
}

func TestCaseScopeReadPreservesExactTailnetAndServiceTokenBoundary(t *testing.T) {
	s, routes := newScopeHandler(t)
	for _, failure := range []string{"outside", "missing-token", "wrong-token"} {
		r := newPreviewRequest(http.MethodGet, "/case-identity/scope", nil)
		switch failure {
		case "outside":
			r.RemoteAddr = "203.0.113.9:4444"
		case "missing-token":
			r.Header.Del("Authorization")
		case "wrong-token":
			r.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 32))
		}
		require.Equal(t, http.StatusUnauthorized, servePreviewRequest(routes, r).Code, failure)
	}
	require.Zero(t, s.scopeCalls)
	require.Zero(t, s.readCalls)
}

func TestCaseScopeUnknownModeAndUnsupportedProviderNeverReadWholePage(t *testing.T) {
	s, routes := newScopeHandler(t)
	r := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity/scope?mode=wrong", nil))
	require.Equal(t, http.StatusUnprocessableEntity, r.Code)
	require.Zero(t, s.scopeCalls)
	fullStore, fullRoutes := newCaseIdentityHandler(t)
	r = servePreviewRequest(fullRoutes, newPreviewRequest(http.MethodGet, "/case-identity/scope", nil))
	require.Equal(t, http.StatusServiceUnavailable, r.Code)
	require.Zero(t, fullStore.readCalls)
	// The existing full-page sibling still works with Store-only providers.
	require.Equal(t, http.StatusOK, servePreviewRequest(fullRoutes, newPreviewRequest(http.MethodGet, "/case-identity", nil)).Code)
	require.Equal(t, 1, fullStore.readCalls)
}

func TestCaseScopeMissingForeignAndUnavailableApprovalFailClosed(t *testing.T) {
	for _, failure := range []string{"missing", "foreign-matter", "foreign-court", "foreign-parent", "padded-provider", "not-found", "unavailable"} {
		t.Run(failure, func(t *testing.T) {
			s, routes := newScopeHandler(t)
			want := http.StatusNotFound
			switch failure {
			case "missing":
				s.view.Matter.ID = ""
			case "foreign-matter":
				s.view.Matter.ID = "11111111-1111-1111-1111-111111111111"
			case "foreign-court":
				s.view.CourtCase.ID = "11111111-1111-1111-1111-111111111111"
			case "foreign-parent":
				s.view.CourtCase.MatterID = "11111111-1111-1111-1111-111111111111"
			case "padded-provider":
				s.view.Matter.ID = strings.Repeat(" ", 65536) + caseidentity.AuthoritativeMatterID
				s.view.CourtCase.MatterID = s.view.Matter.ID
			case "not-found":
				s.scopeErr = caseidentity.ErrNotFound
			case "unavailable":
				s.scopeErr = errors.New("database unavailable")
				want = http.StatusServiceUnavailable
			}
			r := servePreviewRequest(routes, newPreviewRequest(http.MethodGet, "/case-identity/scope", nil))
			require.Equal(t, want, r.Code)
			require.NotContains(t, r.Body.String(), `"matter"`)
			require.Zero(t, s.readCalls)
		})
	}
}

func TestCaseScopeCanceledRequestCannotReturnApproval(t *testing.T) {
	s, routes := newScopeHandler(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := newPreviewRequest(http.MethodGet, "/case-identity/scope", nil).WithContext(ctx)
	require.Equal(t, http.StatusServiceUnavailable, servePreviewRequest(routes, req).Code)
	require.Zero(t, s.readCalls)
}
