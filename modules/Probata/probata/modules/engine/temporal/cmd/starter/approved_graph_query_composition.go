package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountApprovedGraphQueryRoutes adds the approved query paths beside existing starter routes.
// Inputs: existing routes and guarded query routes. Output: combined handler.
// Effects: route registration only; choose this for approved context start and status.
func mountApprovedGraphQueryRoutes(existing, queries http.Handler) (http.Handler, error) {
	if existing == nil || queries == nil {
		return nil, errors.New("approved graph query routes require existing and query handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("/reference-import/analysis/", queries)
	mux.Handle("/", existing)
	return mux, nil
}

// approvedGraphQueryHandler binds approved reads to the shared Temporal client and token.
// Inputs: existing client, task queue and service token path. Output: guarded routes.
// Effects: validates the seams, then starts workflows only on requests; choose over a second scheduler.
func approvedGraphQueryHandler(temporalClient client.Client, taskQueue, serviceTokenFile string) (http.Handler, error) {
	starter, err := approvedgraphqueryflow.NewStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewApprovedGraphQueryHTTPHandler(starter, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
