// Byline: Codex · GPT-5 · 2026-10-07 (internal upload/read proof using isolated fixtures).
package acquisition

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Cursedpotential/probata/engine/servicepeer"
	"github.com/stretchr/testify/require"
)

// TestInternalUploadAndObjectAdmission proves internal upload effects and sealed-object reads.
// Inputs: explicit network, direct peers and invalid configurations. Outputs: statuses and bytes.
// Effects: synthetic sealed objects beneath t.TempDir only. Choose to preserve existing network-only contracts.
func TestInternalUploadAndObjectAdmission(t *testing.T) {
	t.Setenv(servicepeer.InternalCIDRsEnv, "172.25.0.0/16")
	ingress, err := NewUploadIngress(UploadIngressConfig{Root: t.TempDir(), MaxBytes: 1024})
	require.NoError(t, err)
	objects, err := ingress.ObjectHandler()
	require.NoError(t, err)
	content := []byte("synthetic internal upload fixture")
	r := httptest.NewRequest(http.MethodPost, "/acquisition/upload", bytes.NewReader(content))
	r.RemoteAddr = "172.25.0.27:4444"
	w := httptest.NewRecorder()
	ingress.ServeHTTP(w, r)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var accepted uploadAcceptedResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accepted))
	for _, tc := range []struct {
		name, config, peer string
		want               int
	}{
		{"internal", "172.25.0.0/16", "172.25.0.27:4444", 200},
		{"tailnet", "", "100.91.190.107:4444", 200},
		{"default-denied", "", "172.25.0.27:4444", 401},
		{"outside-private", "172.25.0.0/16", "172.26.0.27:4444", 401},
		{"spoof", "172.25.0.0/16", "203.0.113.9:4444", 401},
		{"invalid-tailnet", "172.25.0.0/16,bad", "100.91.190.107:4444", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(servicepeer.InternalCIDRsEnv, tc.config)
			r := httptest.NewRequest("GET", "/acquisition/upload/"+accepted.SHA256, nil)
			r.SetPathValue("sha256", accepted.SHA256)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-For", "172.25.0.27")
			w := httptest.NewRecorder()
			objects.ServeHTTP(w, r)
			require.Equal(t, tc.want, w.Code)
			if tc.want == 200 {
				require.Equal(t, content, w.Body.Bytes())
			} else {
				r := httptest.NewRequest("POST", "/acquisition/upload", bytes.NewBufferString("must not seal"))
				r.RemoteAddr = tc.peer
				r.Header.Set("X-Forwarded-For", "172.25.0.27")
				w := httptest.NewRecorder()
				ingress.ServeHTTP(w, r)
				require.Equal(t, http.StatusUnauthorized, w.Code)
			}
		})
	}
}
