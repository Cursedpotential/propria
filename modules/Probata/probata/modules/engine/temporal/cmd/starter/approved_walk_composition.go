package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/approvedwalkflow"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountApprovedWalkRoutes adds the two walk paths while preserving the existing analysis query subtree.
// Inputs: existing routes and guarded walk routes. Outputs: combined HTTP handler.
// Effects: route registration only; choose beside approved graph query composition, without another auth seam.
func mountApprovedWalkRoutes(existing, walks http.Handler) (http.Handler, error) {
	if existing == nil || walks == nil {
		return nil, errors.New("approved walk routes require existing and walk handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("POST /reference-import/analysis/walks", walks)
	mux.Handle("GET /reference-import/analysis/walk-workflows/{workflow_id}", walks)
	mux.Handle("/", existing)
	return mux, nil
}

// approvedWalkHandler binds walk starts and status to the existing Temporal client, queue and service token.
// Inputs: configured client, task queue and mounted token path. Outputs: guarded walk routes or error.
// Effects: credential validation only until requests arrive; choose over a separate scheduler or login.
func approvedWalkHandler(temporalClient client.Client, taskQueue, serviceTokenFile string, databases ...platformpostgres.DB) (http.Handler, error) {
	starter, err := approvedwalkflow.NewStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	var checker runtimeapi.ApprovedProjectionFreshness
	if len(databases) == 1 {
		checker, err = platformpostgres.NewApprovedDecisionChecker(databases[0])
		if err != nil {
			return nil, err
		}
	}
	handler, err := runtimeapi.NewApprovedWalkHTTPHandler(starter, serviceTokenFile, checker)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
