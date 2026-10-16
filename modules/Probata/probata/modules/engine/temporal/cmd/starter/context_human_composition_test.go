package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNativeHumanRoutesPreservePrefixOwnership checks viewer routing beside context starts and existing imports.
// Inputs: marker handlers and real route shapes. Outputs: independently asserted dispatch ownership.
// Effects: in-memory HTTP requests only; choose to catch the shared-prefix regression before releasing the starter.
func TestNativeHumanRoutesPreservePrefixOwnership(t *testing.T) {
	marker := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Owner", name)
			w.WriteHeader(http.StatusNoContent)
		})
	}
	handler, err := mountNativeHumanContextRoutes(marker("existing"), marker("sources"), marker("viewer"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ method, path, owner string }{
		{"GET", "/reference-import/context/previews/human-actual-version", "viewer"},
		{"GET", "/reference-import/context/previews/human-actual-version/content?offset=25", "viewer"},
		{"POST", "/reference-import/context/sources", "sources"},
		{"GET", "/reference-import/context/workflows/context-source:actual", "sources"},
		{"POST", "/reference-import/source-preview", "existing"},
		{"GET", "/reference-import/previews/actual", "existing"},
		{"GET", "/healthz", "existing"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != http.StatusNoContent || response.Header().Get("X-Owner") != test.owner {
				t.Fatalf("route owner = %q status = %d, want %q", response.Header().Get("X-Owner"), response.Code, test.owner)
			}
		})
	}
	if _, err := mountNativeHumanContextRoutes(nil, marker("sources"), marker("viewer")); err == nil {
		t.Fatal("missing existing handler accepted")
	}
}
