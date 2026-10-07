// Byline: Codex · GPT-5 · 2026-10-07 (shared internal read/write admission proof).
package runtimeapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProfferServiceNetworks validates the complete explicit internal-network policy.
// Inputs: empty, valid and invalid configuration. Outputs: parsed prefixes or errors.
// Effects: none. Choose to prevent public, noncanonical, IPv6 and RFC1918 blanket admission.
func TestProfferServiceNetworks(t *testing.T) {
	for _, value := range []string{"", " \t", "172.25.0.0/16", "10.8.0.0/24, 192.168.4.0/24", "172.25.0.27/32"} {
		networks, err := profferServiceNetworks(value)
		require.NoError(t, err, value)
		require.True(t, networks.Allows("100.91.190.107:4444"))
	}
	for _, value := range []string{
		"bad", "0.0.0.0/0", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "172.24.0.0/15",
		"203.0.113.0/24", "127.0.0.0/24", "169.254.0.0/16", "100.64.0.0/16", "fc00::/64",
		"::ffff:172.25.0.0/112", "172.25.0.27/16", "172.25.0.0/016", "172.25.0.0/16,bad",
		"172.25.0.0/16,", ",172.25.0.0/16", "172.25.0.0/16,,10.8.0.0/24",
	} {
		networks, err := profferServiceNetworks(value)
		require.Error(t, err, value)
		require.False(t, networks.Allows("100.91.190.107:4444"), "invalid config must not retain even tailnet admission: %s", value)
	}
}

// TestProfferSharedGateVariants enforces identical admission across every former duplicate gate.
// Inputs: direct peer, network configuration, token and method combinations. Outputs: status/effect counts.
// Effects: test counter only. Choose to cover reads and writes without real stores or workflow dispatch.
func TestProfferSharedGateVariants(t *testing.T) {
	for _, variant := range []string{"overlay", "extraction", "preview", "source-context"} {
		t.Run(variant, func(t *testing.T) {
			for _, tc := range []struct {
				name, networks, peer, auth string
				want                       int
			}{
				{"tailnet-default", "", "100.91.190.107:4444", "valid", 204},
				{"internal-default-denied", "", "172.25.0.27:4444", "valid", 401},
				{"internal-explicit", "172.25.0.0/16,10.8.0.0/24", "172.25.0.27:4444", "valid", 204},
				{"second-internal", "172.25.0.0/16,10.8.0.0/24", "10.8.0.7:4444", "valid", 204},
				{"outside-private", "172.25.0.0/16", "172.26.0.27:4444", "valid", 401},
				{"public-spoof", "172.25.0.0/16", "203.0.113.9:4444", "valid", 401},
				{"missing-token", "172.25.0.0/16", "172.25.0.27:4444", "missing", 401},
				{"bad-token", "172.25.0.0/16", "172.25.0.27:4444", "bad", 401},
				{"bad-scheme", "172.25.0.0/16", "172.25.0.27:4444", "scheme", 401},
				{"malformed-peer", "172.25.0.0/16", "172.25.0.27", "valid", 401},
				{"ipv6-peer", "172.25.0.0/16", "[fd00::1]:4444", "valid", 401},
				{"invalid-config-tailnet", "172.25.0.0/16,bad", "100.91.190.107:4444", "valid", 401},
				{"invalid-config-internal", "172.25.0.0/16,bad", "172.25.0.27:4444", "valid", 401},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Setenv(profferInternalServiceCIDRsEnv, tc.networks)
					path := serviceTokenPath(t)
					calls := 0
					for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
						next := func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(http.StatusNoContent) }
						var gate http.HandlerFunc
						switch variant {
						case "overlay":
							gate = overlayAuth(path, "test", next)
						case "extraction":
							gate = tailnetServiceAuth(path, "denied", next)
						case "preview":
							gate = (&PreviewHTTPHandler{serviceTokenPath: path}).auth(next)
						case "source-context":
							gate = (&SourceContextHTTPHandler{serviceTokenPath: path}).auth(next)
						}
						r := newPreviewRequest(method, "/test?mode=LIVE", nil)
						r.RemoteAddr = tc.peer
						r.Header.Set("X-Forwarded-For", "100.91.190.107")
						r.Header.Set("X-Real-IP", "172.25.0.27")
						r.Header.Set("Forwarded", "for=100.91.190.107")
						switch tc.auth {
						case "missing":
							r.Header.Del("Authorization")
						case "bad":
							r.Header.Set("Authorization", "Bearer invalid")
						case "scheme":
							r.Header.Set("Authorization", "Basic "+strings.Repeat("s", 32))
						}
						w := servePreviewRequest(gate, r)
						require.Equal(t, tc.want, w.Code, method+": "+w.Body.String())
					}
					if tc.want == 204 {
						require.Equal(t, 5, calls)
					} else {
						require.Zero(t, calls)
					}
				})
			}
		})
	}
}

// TestProfferInternalCaseWritesRetainSafetyGates proves admitted internal POSTs reach only valid writes.
// Inputs: actor, idempotency, mode and body combinations. Outputs: status and stub write invocation assertions.
// Effects: synthetic memory only. Choose to prove network admission does not bypass downstream write safety.
func TestProfferInternalCaseWritesRetainSafetyGates(t *testing.T) {
	t.Setenv(profferInternalServiceCIDRsEnv, "172.25.0.0/16")
	for _, tc := range []struct {
		name, mode, remove string
		want               int
	}{
		{"valid-live", "LIVE", "", 201}, {"missing-actor", "LIVE", "X-authentik-uid", 401},
		{"missing-idempotency", "LIVE", "Idempotency-Key", 401}, {"dev-denied", "DEV", "", 409},
		{"unknown-mode", "invalid", "", 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, routes := newCaseIdentityHandler(t)
			r := newPreviewRequest(http.MethodPost, "/case-identity/identifiers?mode="+tc.mode,
				[]byte(`{"entity_id":"`+caseTestPerson+`","raw_value":"8105550142","kind":"phone","status":"confirmed","basis":"synthetic confirmation","change_reason":"test"}`))
			r.RemoteAddr = "172.25.0.27:4444"
			r.Header.Set("Idempotency-Key", "internal-write-1")
			if tc.remove != "" {
				r.Header.Del(tc.remove)
			}
			w := servePreviewRequest(routes, r)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			if tc.want == 201 {
				require.Equal(t, "8105550142", store.identifier.RawValue)
				require.Equal(t, "internal-write-1", store.actor.IdempotencyKey)
				require.Equal(t, "authentik-user-1", store.actor.SubjectUID)
			} else {
				require.Empty(t, store.identifier)
				require.Empty(t, store.actor)
			}
		})
	}
}

// TestProfferServiceCredentialReload keeps mounted-token rotation live without handler recreation.
// Inputs: one handler and successive valid/malformed credential files. Outputs: admission statuses.
// Effects: writes synthetic test credentials only. Choose to protect credential reload and fail-closed behavior.
func TestProfferServiceCredentialReload(t *testing.T) {
	t.Setenv(profferInternalServiceCIDRsEnv, "172.25.0.0/16")
	path := serviceTokenPath(t)
	calls := 0
	gate := overlayAuth(path, "reload", func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(204) })
	request := func(token string) int {
		r := newPreviewRequest("GET", "/test", nil)
		r.RemoteAddr = "172.25.0.27:4444"
		r.Header.Set("Authorization", "Bearer "+token)
		return servePreviewRequest(gate, r).Code
	}
	require.Equal(t, 204, request(strings.Repeat("s", 32)))
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("r", 32)), 0600))
	require.Equal(t, 401, request(strings.Repeat("s", 32)))
	require.Equal(t, 204, request(strings.Repeat("r", 32)))
	require.NoError(t, os.WriteFile(path, []byte("invalid"), 0600))
	require.Equal(t, 401, request("invalid"))
	require.Equal(t, 2, calls)
	missing := profferServiceAuth(path+"-missing", "denied", func(http.ResponseWriter, *http.Request) { t.Fatal("missing token admitted") })
	w := httptest.NewRecorder()
	r := newPreviewRequest("GET", "/test", nil)
	r.RemoteAddr = "172.25.0.27:4444"
	missing(w, r)
	require.Equal(t, 401, w.Code)
}

// TestInternalStartProbeRejectsBeforeEffects validates the side-effect-free deployment admission probe.
// Inputs: internal peer, valid mounted token and incomplete JSON object on the real start route.
// Outputs: 400 without workflow start or preview binding. Effects: synthetic fixtures only.
// Choose before live admission verification; never submit valid synthetic case work to production.
func TestInternalStartProbeRejectsBeforeEffects(t *testing.T) {
	t.Setenv(profferInternalServiceCIDRsEnv, "172.25.0.0/16")
	handler, store, workflow := previewTestHandler(t)
	r := newPreviewRequest(http.MethodPost, "/reference-import/start?mode=LIVE", []byte(`{}`))
	r.RemoteAddr = "172.25.0.27:4444"
	w := servePreviewRequest(handler.Routes(), r)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "start request is incomplete")
	require.Empty(t, workflow.started)
	bindings, err := store.ListBindings(t.Context(), nil, 10)
	require.NoError(t, err)
	require.Empty(t, bindings.Bindings)
}
