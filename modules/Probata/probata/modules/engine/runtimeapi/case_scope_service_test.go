// Byline: Codex · GPT-6 · 2026-10-06.
package runtimeapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/stretchr/testify/require"
)

// scopeServiceStore returns only the authoritative ID pair for authentication tests.
// Inputs: synthetic request. Outputs: bounded case scope. Effects: in-memory call count.
// Choose to verify the service-read seam independently from Case page mutation routes.
type scopeServiceStore struct {
	caseIdentityStoreStub
	scopeCalls int
}

// ReadScope returns a valid synthetic scope after the handler authorizes its peer.
// Inputs: requested mode. Outputs: fixed approved IDs. Effects: increments test counter.
// Choose for network/token tests; it does not replace the live registry verification.
func (s *scopeServiceStore) ReadScope(_ context.Context, mode caseidentity.Mode) (caseidentity.ScopeView, error) {
	s.scopeCalls++
	return caseidentity.ScopeView{Mode: mode, Matter: caseidentity.ScopeMatter{ID: caseidentity.AuthoritativeMatterID}, CourtCase: caseidentity.ScopeCourtCase{ID: caseidentity.AuthoritativeCourtCaseID, MatterID: caseidentity.AuthoritativeMatterID}}, nil
}

// TestCaseScopeServiceAuthorizationIsReadOnlyAndPeerBound checks the narrow service permission.
// Inputs: real-peer/token/path combinations including forwarded spoofing. Outputs: status/call assertions.
// Effects: memory only. Choose to prevent widening Case page or mutation access while enabling workers.
func TestCaseScopeServiceAuthorizationIsReadOnlyAndPeerBound(t *testing.T) {
	t.Setenv("PROFFER_CASE_SCOPE_SERVICE_CIDRS", "172.25.0.0/16")
	for _, tc := range []struct {
		name, peer, method, path, auth string
		want                           int
	}{
		{"worker", "172.25.0.27:4444", "GET", "/case-identity/scope?mode=LIVE", "valid", 200},
		{"tailnet", "100.91.190.107:4444", "GET", "/case-identity/scope?mode=LIVE", "valid", 200},
		{"missing-token", "172.25.0.27:4444", "GET", "/case-identity/scope?mode=LIVE", "missing", 401},
		{"wrong-token", "172.25.0.27:4444", "GET", "/case-identity/scope?mode=LIVE", "wrong", 401},
		{"other-private", "172.26.0.27:4444", "GET", "/case-identity/scope?mode=LIVE", "valid", 401},
		{"public-forwarded-spoof", "203.0.113.9:4444", "GET", "/case-identity/scope?mode=LIVE", "valid", 401},
		{"case-page-still-tailnet", "172.25.0.27:4444", "GET", "/case-identity?mode=LIVE", "valid", 401},
		{"mutations-still-tailnet", "172.25.0.27:4444", "POST", "/case-identity/identifiers?mode=LIVE", "valid", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &scopeServiceStore{}
			h, err := NewCaseIdentityHTTPHandler(s, serviceTokenPath(t))
			require.NoError(t, err)
			r := newPreviewRequest(tc.method, tc.path, nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-For", "100.91.190.107")
			if tc.auth == "missing" {
				r.Header.Del("Authorization")
			}
			if tc.auth == "wrong" {
				r.Header.Set("Authorization", "Bearer invalid")
			}
			w := servePreviewRequest(h.Routes(), r)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.want == http.StatusOK {
				require.Equal(t, 1, s.scopeCalls)
			} else {
				require.Zero(t, s.scopeCalls)
			}
			require.Zero(t, s.readCalls)
		})
	}
}

// TestCaseScopeNetworkConfigurationFailsClosed rejects public or overbroad network configuration.
// Inputs: invalid service CIDRs. Outputs: constructor errors. Effects: none beyond test environment.
// Choose to enforce the existing private-network validator for this additional read-only route.
func TestCaseScopeNetworkConfigurationFailsClosed(t *testing.T) {
	for _, value := range []string{"0.0.0.0/0", "172.0.0.0/8", "203.0.113.0/24", "bad"} {
		t.Setenv("PROFFER_CASE_SCOPE_SERVICE_CIDRS", value)
		_, err := NewCaseIdentityHTTPHandler(&scopeServiceStore{}, serviceTokenPath(t))
		require.Error(t, err)
	}
}
