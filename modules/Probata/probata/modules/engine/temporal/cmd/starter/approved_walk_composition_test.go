package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestApprovedWalkRoutesPreserveGraphQueries checks composition through the actual existing query mount.
// Inputs: distinct route markers for existing, query and walk handlers. Outputs: exact routing/nil-seam assertions.
// Effects: in-memory HTTP only; choose to catch a walk subtree shadowing graph queries or other starter routes.
func TestApprovedWalkRoutesPreserveGraphQueries(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	queries := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	walks := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	routes, err := mountApprovedGraphQueryRoutes(existing, queries)
	if err != nil {
		t.Fatal(err)
	}
	routes, err = mountApprovedWalkRoutes(routes, walks)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/reference-import/analysis/walks", http.StatusAccepted},
		{http.MethodGet, "/reference-import/analysis/walk-workflows/approved-context-walk:one", http.StatusAccepted},
		{http.MethodPost, "/reference-import/analysis/queries", http.StatusCreated},
		{http.MethodGet, "/reference-import/analysis/workflows/approved-context-query:one", http.StatusCreated},
		{http.MethodGet, "/reference-import/atomic-tools/actions", http.StatusTeapot},
		{http.MethodPost, "/reference-import/context/sources", http.StatusTeapot},
		{http.MethodGet, "/reference-import/start", http.StatusTeapot},
	} {
		response := httptest.NewRecorder()
		routes.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
		if response.Code != tc.want {
			t.Errorf("%s %s -> %d, want %d", tc.method, tc.path, response.Code, tc.want)
		}
	}
	if _, err := mountApprovedWalkRoutes(nil, walks); err == nil {
		t.Fatal("nil existing handler accepted")
	}
	if _, err := mountApprovedWalkRoutes(existing, nil); err == nil {
		t.Fatal("nil walk handler accepted")
	}
	if _, err := approvedWalkHandler(nil, "proffer-v1", "unused"); err == nil {
		t.Fatal("missing shared Temporal client accepted")
	}
}
