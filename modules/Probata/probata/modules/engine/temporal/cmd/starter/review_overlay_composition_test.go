// Byline: Claude Code · Opus 5.5 · 2026-09-26
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMountReviewOverlayRoutesOwnsOnlyItsExactPaths(t *testing.T) {
	marker := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("X-Handler", name) })
	}
	routes, err := mountReviewOverlayRoutes(marker("existing"), marker("metadata"), marker("context-review"))
	if err != nil {
		t.Fatal(err)
	}
	const preview = "/reference-import/previews/overlayhandle_0123456789abcdefABCDEF"
	for _, test := range []struct{ method, path, want string }{
		{http.MethodGet, preview + "/metadata", "metadata"},
		{http.MethodPost, preview + "/metadata/corrections", "metadata"},
		{http.MethodGet, preview + "/messages/0190a000-0000-7000-8000-0000000000e1/context-review", "context-review"},
		{http.MethodPost, preview + "/messages/0190a000-0000-7000-8000-0000000000e1/context-review", "context-review"},
		{http.MethodPost, preview + "/messages/0190a000-0000-7000-8000-0000000000e1/foreshadowing", "context-review"},
		{http.MethodGet, preview, "existing"},
		{http.MethodGet, preview + "/messages", "existing"},
		{http.MethodGet, preview + "/content", "existing"},
		{http.MethodPost, preview + "/decision", "existing"},
		{http.MethodPost, "/reference-import/start", "existing"},
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
		if got := recorder.Header().Get("X-Handler"); got != test.want {
			t.Errorf("%s %s went to %q, want %q", test.method, test.path, got, test.want)
		}
	}
	if _, err := mountReviewOverlayRoutes(nil, marker("m"), marker("c")); err == nil {
		t.Fatal("nil existing handler accepted")
	}
}
