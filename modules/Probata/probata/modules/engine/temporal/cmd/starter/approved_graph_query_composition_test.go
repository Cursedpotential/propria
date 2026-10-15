package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestApprovedGraphQueryRoutesMountBesideExistingOnes checks the query paths without losing earlier routes.
// Inputs: existing and query marker handlers. Output: route and nil-handler assertions.
// Effects: none; choose when the starter's approved query composition changes.
func TestApprovedGraphQueryRoutesMountBesideExistingOnes(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	queries := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	routes, err := mountApprovedGraphQueryRoutes(existing, queries)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/reference-import/analysis/queries":           http.StatusAccepted,
		"/reference-import/analysis/workflows/query-1": http.StatusAccepted,
		"/reference-import/atomic-tools/actions":       http.StatusTeapot,
		"/reference-import/conversations/workflows/x":  http.StatusTeapot,
		"/reference-import/start":                      http.StatusTeapot,
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != want {
			t.Errorf("%s -> %d, want %d", target, recorder.Code, want)
		}
	}
	if _, err := mountApprovedGraphQueryRoutes(nil, queries); err == nil {
		t.Fatal("a nil existing handler must fail")
	}
	if _, err := mountApprovedGraphQueryRoutes(existing, nil); err == nil {
		t.Fatal("a nil query handler must fail")
	}
}
