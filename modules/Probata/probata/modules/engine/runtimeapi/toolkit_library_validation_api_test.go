// Byline: Codex, GPT-6, 2026-10-04. Retained proposal starter boundary tests.
package runtimeapi

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type toolkitValidatorFixture struct {
	calls    int
	proposal string
}

// StartLibraryValidation captures synthetic dispatch without running a workflow or touching case data.
// Inputs: synthetic proposal id. Outputs: fixed workflow/run ids. Effects: fixture counter only.
// Choose for route boundary tests; production uses Temporal.
func (f *toolkitValidatorFixture) StartLibraryValidation(_ context.Context, proposal string) (string, string, error) {
	f.calls++
	f.proposal = proposal
	return "library-validation-fixture", "fixture-run", nil
}

// TestToolkitLibraryValidationBoundary proves peer/token checks and proposal-only strict decoding.
// Inputs: synthetic HTTP requests and temporary test credential. Outputs: assertions.
// Effects: test fixture calls only; no production requests. Choose before deploying the starter route.
func TestToolkitLibraryValidationBoundary(t *testing.T) {
	t.Setenv("TOOLKIT_VALIDATION_SERVICE_CIDRS", "")
	credential := filepath.Join(t.TempDir(), "fixture-token")
	if err := os.WriteFile(credential, []byte("synthetic-test-credential-32bytes-or-more"), 0600); err != nil {
		t.Fatal(err)
	}
	starter := &toolkitValidatorFixture{}
	handler, err := NewToolkitLibraryValidationHandler(starter, credential)
	if err != nil {
		t.Fatal(err)
	}
	proposal := "library_proposal:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	for _, tc := range []struct {
		name, peer, token, body string
		status                  int
	}{
		{"valid", "100.72.169.40:1234", "synthetic-test-credential-32bytes-or-more", `{"proposal_id":"` + proposal + `"}`, 202},
		{"untrusted peer", "172.18.0.2:1234", "synthetic-test-credential-32bytes-or-more", `{"proposal_id":"` + proposal + `"}`, 401},
		{"wrong token", "100.72.169.40:1234", "wrong", `{"proposal_id":"` + proposal + `"}`, 401},
		{"body injection", "100.72.169.40:1234", "synthetic-test-credential-32bytes-or-more", `{"proposal_id":"` + proposal + `","status":"VERIFIED_PRIMARY"}`, 400},
		{"invalid id", "100.72.169.40:1234", "synthetic-test-credential-32bytes-or-more", `{"proposal_id":"source:primary"}`, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/toolkit/library/validate", strings.NewReader(tc.body))
			req.RemoteAddr = tc.peer
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d, expected %d", rec.Code, tc.status)
			}
		})
	}
	if starter.calls != 1 || starter.proposal != proposal {
		t.Fatalf("unexpected dispatch count %d", starter.calls)
	}
}

// TestToolkitLibraryValidationPrivateNetworks restricts the container exception to explicit private networks.
// Inputs: private, public and overly broad CIDRs. Outputs: configuration assertions.
// Effects: none. Choose to verify the validation-only service configuration before deployment.
func TestToolkitLibraryValidationPrivateNetworks(t *testing.T) {
	for _, cidr := range []string{"0.0.0.0/0", "8.8.8.0/24", "10.0.0.0/8", "not-a-network"} {
		if _, err := toolkitValidationServiceNetworks(cidr); err == nil {
			t.Fatalf("accepted unsafe service network %s", cidr)
		}
	}
	networks, err := toolkitValidationServiceNetworks("172.25.0.0/16")
	if err != nil || len(networks) != 2 {
		t.Fatal("explicit service network was not admitted")
	}
}
