// Byline: Codex · GPT-5 · 2026-10-07 (dedicated toolkit contract preservation).
package runtimeapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestToolkitConfiguredNetworksPreserveDedicatedCredentials covers inventory/multi-network toolkit admission.
// Inputs: explicit owned network union, actual socket peers and distinct synthetic credentials.
// Outputs: status and dispatch assertions. Effects: isolated callbacks only.
// Choose to verify configurable network rollout without merging toolkit and Proffer token contracts.
func TestToolkitConfiguredNetworksPreserveDedicatedCredentials(t *testing.T) {
	var cidrs []string
	for subnet := 0; subnet <= 23; subnet++ {
		cidrs = append(cidrs, fmt.Sprintf("10.200.%d.0/24", subnet))
	}
	cidrs = append(cidrs, "10.201.0.0/29", "10.201.8.0/29", "10.250.72.0/24")
	for subnet := 17; subnet <= 31; subnet++ {
		cidrs = append(cidrs, fmt.Sprintf("172.%d.0.0/16", subnet))
	}
	for _, subnet := range []int{0, 16, 32, 48, 64, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240} {
		cidrs = append(cidrs, fmt.Sprintf("192.168.%d.0/20", subnet))
	}
	networks, err := toolkitValidationServiceNetworks(strings.Join(cidrs, ","))
	require.NoError(t, err)
	require.Len(t, networks, 58, "57 explicitly configured networks plus tailnet")
	for _, kind := range []string{"validation", "sync"} {
		t.Run(kind, func(t *testing.T) {
			token := "synthetic-dedicated-" + kind + "-credential-32bytes"
			file := filepath.Join(t.TempDir(), "token")
			require.NoError(t, os.WriteFile(file, []byte(token), 0600))
			calls := 0
			gate := toolkitValidationServiceAuth(file, networks, func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(204) })
			for _, peer := range []string{"172.25.0.27:4444", "10.200.23.4:4444", "10.201.8.3:4444", "192.168.244.8:4444"} {
				for _, supplied := range []string{token, "", strings.Repeat("s", 32)} {
					r := httptest.NewRequest("POST", "/toolkit/library/"+kind, nil)
					r.RemoteAddr = peer
					if supplied != "" {
						r.Header.Set("Authorization", "Bearer "+supplied)
					}
					w := httptest.NewRecorder()
					gate(w, r)
					if supplied == token {
						require.Equal(t, 204, w.Code)
					} else {
						require.Equal(t, 401, w.Code)
					}
				}
			}
			require.Equal(t, 4, calls, "only dedicated credentials may invoke toolkit effects")
		})
	}
}
