// Byline: Claude Code · Opus 5.5 · 2026-09-25

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEntityExtractionRoutesMountBesideExistingOnes(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	extraction := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	routes, err := mountEntityExtractionRoutes(existing, extraction)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/reference-import/entities/proposals":       http.StatusAccepted,
		"/reference-import/entities/commits/x":       http.StatusAccepted,
		"/reference-import/previews/handle/operator": http.StatusTeapot,
		"/reference-import/start":                    http.StatusTeapot,
		"/reference-import/repair/tools":             http.StatusTeapot,
	} {
		recorder := httptest.NewRecorder()
		routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != want {
			t.Errorf("%s -> %d, want %d", target, recorder.Code, want)
		}
	}
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/reference-import/events/from-record", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("from-record -> %d", recorder.Code)
	}
	if _, err := mountEntityExtractionRoutes(nil, extraction); err == nil {
		t.Fatal("a nil existing handler must fail")
	}
}
