// Byline: Codex · GPT-6 · 2026-10-07.
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAtomicToolRoutesMountBesideExistingOnes keeps the new path scoped.
// Inputs: two marker handlers. Output: route assertions. Effects: none.
// Choose this when the starter's composition changes.
func TestAtomicToolRoutesMountBesideExistingOnes(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	actions := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	routes, err := mountAtomicToolRoutes(existing, actions)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/reference-import/atomic-tools/actions":     http.StatusAccepted,
		"/reference-import/atomic-tools/workflows/x": http.StatusAccepted,
		"/reference-import/conversations/workflows/x": http.StatusTeapot,
		"/reference-import/start":                     http.StatusTeapot,
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != want {
			t.Errorf("%s -> %d, want %d", target, recorder.Code, want)
		}
	}
	if _, err := mountAtomicToolRoutes(nil, actions); err == nil {
		t.Fatal("a nil existing handler must fail")
	}
	if _, err := mountAtomicToolRoutes(existing, nil); err == nil {
		t.Fatal("a nil action handler must fail")
	}
}
