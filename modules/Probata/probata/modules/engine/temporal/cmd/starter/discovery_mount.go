// Byline: Codex · GPT-6 · 2026-10-11
package main

import (
	"errors"
	"net/http"
)

// mountDiscoveryRoutes adds the explicitly configured discovery surface.
// Inputs: existing starter routes and an optional authenticated discovery handler.
// Outputs: a composed HTTP handler; a nil discovery handler preserves existing routes.
// Effects: registration only. Choose for optional analysis without changing intake.
func mountDiscoveryRoutes(existing, discovery http.Handler) (http.Handler, error) {
	if existing == nil {
		return nil, errors.New("discovery routes require existing starter routes")
	}
	if discovery == nil {
		return existing, nil
	}
	mux := http.NewServeMux()
	mux.Handle("/discovery/", discovery)
	mux.Handle("/", existing)
	return mux, nil
}
