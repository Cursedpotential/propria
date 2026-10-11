// Byline: Codex · GPT-6 · 2026-10-07.
package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/atomictool"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountAtomicToolRoutes attaches operator-started source tools beside existing routes.
// Inputs: existing starter routes and the guarded tool handler. Output: combined routes.
// Effects: route registration only; choose this for the atomic tool start/status API.
func mountAtomicToolRoutes(existing, actions http.Handler) (http.Handler, error) {
	if existing == nil || actions == nil {
		return nil, errors.New("atomic tool routes require existing and action handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("/reference-import/atomic-tools/", actions)
	mux.Handle("/", existing)
	return mux, nil
}

// atomicToolHandler binds source-pinned actions to the existing Temporal client.
// Inputs: shared client, Proffer task queue and service token file. Output: guarded routes.
// Effects: none before invocation; choose instead of an independent scheduler or tool call.
func atomicToolHandler(temporalClient client.Client, taskQueue, serviceTokenFile string) (http.Handler, error) {
	starter, err := atomictool.NewStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewAtomicToolHTTPHandler(starter, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
