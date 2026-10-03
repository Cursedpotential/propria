// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestConversationExtractionRoutesMountBesideExistingOnes proves the new routes do not take over any existing route.
func TestConversationExtractionRoutesMountBesideExistingOnes(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	conversation := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	routes, err := mountConversationExtractionRoutes(existing, conversation)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/reference-import/extractors":                    http.StatusAccepted,
		"/reference-import/conversations/workflows/x":     http.StatusAccepted,
		"/reference-import/conversations/send-to-surreal": http.StatusAccepted,
		"/reference-import/entities/proposals":            http.StatusTeapot,
		"/reference-import/previews/handle/operator":      http.StatusTeapot,
		"/reference-import/start":                         http.StatusTeapot,
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != want {
			t.Errorf("%s -> %d, want %d", target, recorder.Code, want)
		}
	}
	if _, err := mountConversationExtractionRoutes(nil, conversation); err == nil {
		t.Fatal("a nil existing handler must fail")
	}
}
