// Byline: Codex · GPT-5 · 2026-10-07.
package temporal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cursedpotential/probata/engine/servicepeer"
)

// TestInternalStarterPeerAdmission preserves network-only start/read access and existing dev bypass.
// Inputs: direct peer, explicit network and bypass combinations. Outputs: statuses/effect counters.
// Effects: synthetic callback only, never Temporal or database dispatch. Choose for HTTP admission regression proof.
func TestInternalStarterPeerAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, config, peer, bypass string
		want                       int
	}{
		{"internal", "172.25.0.0/16", "172.25.0.27:4444", "false", 204},
		{"tailnet", "", "100.91.190.107:4444", "false", 204},
		{"default-denied", "", "172.25.0.27:4444", "false", 401},
		{"outside-private", "172.25.0.0/16", "172.26.0.27:4444", "false", 401},
		{"spoof", "172.25.0.0/16", "203.0.113.9:4444", "false", 401},
		{"preserved-dev-bypass", "", "203.0.113.9:4444", "true", 204},
		{"invalid-with-bypass", "172.25.0.0/16,bad", "172.25.0.27:4444", "true", 401},
		{"invalid-tailnet", "172.25.0.0/16,bad", "100.91.190.107:4444", "false", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(servicepeer.InternalCIDRsEnv, tc.config)
			t.Setenv(platformDevAuthBypassEnv, tc.bypass)
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				calls := 0
				gate := (&StarterHTTPHandler{}).withAuth(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(204) })
				r := httptest.NewRequest(method, "/reference-import/test", nil)
				r.RemoteAddr = tc.peer
				r.Header.Set("X-Forwarded-For", "172.25.0.27")
				r.Header.Set("Forwarded", "for=100.91.190.107")
				w := httptest.NewRecorder()
				gate(w, r)
				if w.Code != tc.want {
					t.Fatalf("%s: got %d, want %d: %s", method, w.Code, tc.want, w.Body.String())
				}
				if tc.want == 401 && calls != 0 {
					t.Fatal("rejected request invoked downstream effects")
				}
				if tc.want == 204 && calls != 1 {
					t.Fatal("admitted request did not invoke handler")
				}
			}
		})
	}
}
