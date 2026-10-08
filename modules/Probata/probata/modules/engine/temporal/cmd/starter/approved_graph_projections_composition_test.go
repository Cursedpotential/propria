package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMountApprovedGraphProjectionsKeepsQuerySubtree verifies the exact-route composition.
// Inputs: existing query marker and projection marker handlers. Outputs: routed status codes.
// Effects: none; choose when the starter's analytical routes change.
func TestMountApprovedGraphProjectionsKeepsQuerySubtree(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	projections := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	routes, err := mountApprovedGraphProjectionsRoutes(existing, projections)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/reference-import/analysis/projections":       http.StatusOK,
		"/reference-import/analysis/queries":           http.StatusAccepted,
		"/reference-import/analysis/workflows/query-1": http.StatusAccepted,
		"/reference-import/start":                      http.StatusAccepted,
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != want {
			t.Errorf("%s -> %d, want %d", target, recorder.Code, want)
		}
	}
	if _, err := mountApprovedGraphProjectionsRoutes(nil, projections); err == nil {
		t.Fatal("nil existing routes were accepted")
	}
	if _, err := mountApprovedGraphProjectionsRoutes(existing, nil); err == nil {
		t.Fatal("nil projection routes were accepted")
	}
}
