package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/runtimeapi"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
)

// mountContextSourceRoutes adds context-first paths beside all existing starter routes.
// Inputs: existing and context handlers. Outputs: combined handler or error.
// Effects: route registration only; choose for new source starts and actor-bound status.
func mountContextSourceRoutes(existing, contextRoutes http.Handler) (http.Handler, error) {
	if existing == nil || contextRoutes == nil {
		return nil, errors.New("context source routes require existing and context handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("/reference-import/context/", contextRoutes)
	mux.Handle("/", existing)
	return mux, nil
}

// contextSourceHandler binds context-first Proffer starts to the shared client and token.
// Inputs: existing Temporal client, task queue and service token path. Outputs: guarded routes.
// Effects: validates seams; starts Go workflow only on a request, never a Python workflow.
func contextSourceHandler(temporalClient client.Client, taskQueue, serviceTokenFile string) (http.Handler, error) {
	starter, err := platformtemporal.NewContextStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewContextSourceHTTPHandler(starter, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
