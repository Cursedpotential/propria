// Byline: Codex · GPT-6 · 2026-10-11
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDiscoveryMountPreservesExistingRoutes verifies optional analysis does not gate intake.
func TestDiscoveryMountPreservesExistingRoutes(t *testing.T) {
	existing := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
	optional := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, enabled := range []bool{false, true} {
		var addon http.Handler
		if enabled {
			addon = optional
		}
		mux, err := mountDiscoveryRoutes(existing, addon)
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest("POST", "/intake/start", nil))
		if recorder.Code != 202 {
			t.Fatalf("discovery enabled=%v changed existing route: %d", enabled, recorder.Code)
		}
		if enabled {
			recorder = httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest("GET", "/discovery/datasets", nil))
			if recorder.Code != 204 {
				t.Fatalf("discovery route was not composed: %d", recorder.Code)
			}
		}
	}
}
