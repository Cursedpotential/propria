// Byline: Codex · GPT-6 · 2026-10-05. Private starter sync and original boundary proofs.
package runtimeapi

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
)

type toolkitSyncFixture struct{ starts, reads int }

// StartLibrarySync records a synthetic operation dispatch without Temporal or private records.
// Inputs: operation ID. Outputs: fixed identities. Effects: fixture counter; choose for HTTP admission tests.
func (f *toolkitSyncFixture) StartLibrarySync(context.Context, string) (string, string, error) {
	f.starts++
	return "fixture-sync", "fixture-run", nil
}

// ReadOriginal supplies synthetic verified bytes without storage access.
// Inputs: binding/version. Outputs: small fixture PDF. Effects: fixture counter; choose for proxy authorization tests.
func (f *toolkitSyncFixture) ReadOriginal(context.Context, string, string) (librarysync.OriginalBytes, error) {
	f.reads++
	return librarysync.OriginalBytes{Bytes: []byte("%PDF-fixture"), ContentType: "application/pdf", Filename: "fixture.pdf"}, nil
}

// TestToolkitLibrarySyncPrivateBoundary rejects ordinary headers, peers and editable-body injection before service calls.
// Inputs: temporary dedicated credential and synthetic requests. Outputs: exact response and call-count assertions.
// Effects: isolated fixture calls only; choose before deploying private sync/original routes.
func TestToolkitLibrarySyncPrivateBoundary(t *testing.T) {
	t.Setenv("TOOLKIT_LIBRARY_SYNC_SERVICE_CIDRS", "")
	token := "synthetic-dedicated-sync-credential-32bytes"
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	f := &toolkitSyncFixture{}
	h, err := NewToolkitLibrarySyncHandler(f, f, file)
	if err != nil {
		t.Fatal(err)
	}
	op := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	original := "/toolkit/library/files/library_file:" + strings.Repeat("a", 64) + "/original?version_id=pinned-version"
	for _, tc := range []struct {
		name, method, route, peer, token, body string
		status                                 int
	}{
		{"dispatch", "POST", "/toolkit/library/sync", "100.72.169.40:1234", token, `{"operation_id":"` + op + `"}`, 202},
		{"body injection", "POST", "/toolkit/library/sync", "100.72.169.40:1234", token, `{"operation_id":"` + op + `","key":"elsewhere"}`, 400},
		{"missing credential", "POST", "/toolkit/library/sync", "100.72.169.40:1234", "", `invalid`, 401},
		{"untrusted peer", "POST", "/toolkit/library/sync", "172.18.0.2:1234", token, `invalid`, 401},
		{"original", "GET", original, "100.72.169.40:1234", token, "", 200},
		{"original unauthorized", "GET", original, "100.72.169.40:1234", "", "", 401},
		{"ambiguous pin", "GET", original + "&version_id=other", "100.72.169.40:1234", token, "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.route, strings.NewReader(tc.body))
			r.RemoteAddr = tc.peer
			r.Header.Set("Authorization", "Bearer "+tc.token)
			r.Header.Set("Tailscale-User-Login", "owner@example.invalid")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d want %d", w.Code, tc.status)
			}
			if w.Code == 200 && w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("private original caching")
			}
		})
	}
	if f.starts != 1 || f.reads != 1 {
		t.Fatalf("unexpected service calls starts=%d reads=%d", f.starts, f.reads)
	}
}
