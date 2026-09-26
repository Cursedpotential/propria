package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMountPreviewRoutesOwnsOpaqueStartAndPreservesLegacyFallback(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("legacy")) })
	preview := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"preview_handle":"opaque"}`)) })
	routes, err := mountPreviewRoutes(existing, preview)
	if err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	routes.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/reference-import/start", nil))
	if !strings.Contains(start.Body.String(), "preview_handle") {
		t.Fatalf("opaque start response = %q", start.Body.String())
	}
	legacy := httptest.NewRecorder()
	routes.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if legacy.Body.String() != "legacy" {
		t.Fatalf("legacy fallback response = %q", legacy.Body.String())
	}
}

// Every route PreviewHTTPHandler.Routes() registers must be forwarded by the outer mux;
// the operations routes were not, and 404'd in production (found live 2026-09-20).
func TestMountPreviewRoutesForwardsEveryPreviewRoute(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("legacy")) })
	preview := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("preview")) })
	routes, err := mountPreviewRoutes(existing, preview)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/reference-import/start"},
		{http.MethodGet, "/reference-import/operations"},
		{http.MethodGet, "/reference-import/operations?limit=50"},
		{http.MethodGet, "/reference-import/operations/handle"},
		{http.MethodGet, "/reference-import/previews/handle"},
		{http.MethodGet, "/reference-import/previews/handle/messages"},
		{http.MethodGet, "/reference-import/previews/handle/content"},
		{http.MethodGet, "/reference-import/previews/handle/events"},
		{http.MethodPost, "/reference-import/previews/handle/decision"},
		{http.MethodPost, "/reference-import/previews/handle/repair-decision"},
		{http.MethodPost, "/reference-import/previews/handle/handler-selection"},
		{http.MethodGet, "/reference-import/previews/handle/source-context"},
		// Batch routes were missing until 2026-09-25 and 404'd.
		// Byline: Claude Code · Opus 5.5 · 2026-09-25
		{http.MethodPost, "/reference-import/start-batch"},
		{http.MethodGet, "/reference-import/batches/batch-id"},
		// Repair workflow builder.
		{http.MethodGet, "/reference-import/repair/tools"},
		{http.MethodPost, "/reference-import/repair/propose"},
		{http.MethodPost, "/reference-import/repair/validate"},
		{http.MethodPost, "/reference-import/repair/run"},
		{http.MethodGet, "/reference-import/repair/runs/repair-plan-x-0123456789ab"},
	} {
		got := httptest.NewRecorder()
		routes.ServeHTTP(got, httptest.NewRequest(tc.method, tc.path, nil))
		if got.Body.String() != "preview" {
			t.Errorf("%s %s reached %q, want the preview handler", tc.method, tc.path, got.Body.String())
		}
	}
	legacy := httptest.NewRecorder()
	routes.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/reference-import/wf-1/preview", nil))
	if legacy.Body.String() != "legacy" {
		t.Errorf("legacy workflow preview reached %q, want the legacy handler", legacy.Body.String())
	}
}

func TestPreviewCursorKeyFailsClosedAndAcceptsBoundedSecret(t *testing.T) {
	shortPath := filepath.Join(t.TempDir(), "short")
	if err := os.WriteFile(shortPath, []byte("short"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(previewCursorKeyFileEnv, shortPath)
	if _, err := previewCursorKey(); err == nil {
		t.Fatal("short cursor key was accepted")
	}
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyPath, []byte(strings.Repeat("k", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(previewCursorKeyFileEnv, keyPath)
	key, err := previewCursorKey()
	if err != nil || len(key) < 32 {
		t.Fatalf("cursor key length=%d err=%v", len(key), err)
	}
}
