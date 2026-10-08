// Byline: Claude Code · Opus 5.5 · 2026-09-25

package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountEntityExtractionRoutes adds the entity/event extraction surface
// without changing any existing route's ownership.
func mountEntityExtractionRoutes(existing, extraction http.Handler) (http.Handler, error) {
	if existing == nil || extraction == nil {
		return nil, errors.New("entity extraction routes require existing and extraction handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("/reference-import/entities/", extraction)
	mux.Handle("POST /reference-import/events/from-record", extraction)
	mux.Handle("/", existing)
	return mux, nil
}

// entityExtractionHandler builds source-aware review routes on the existing Temporal starter.
// Inputs: database, Temporal client, task queue and existing service token file. Outputs: HTTP routes.
// Effects: constructs adapters only; request handlers verify retained originals before pending AI writes.
// Choose for both normalized-message extraction and native AI review using the same owner decision surface.
func entityExtractionHandler(db platformpostgres.DB, temporalClient client.Client, taskQueue, serviceTokenFile string) (http.Handler, error) {
	openOriginal, err := runtimeapi.NewRetainedObjectOpener(db)
	if err != nil {
		return nil, err
	}
	store, err := platformpostgres.NewEntityExtractionStoreWithAIOriginal(db, openOriginal)
	if err != nil {
		return nil, err
	}
	starter, err := flow.NewStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewEntityExtractionHTTPHandler(store, starter, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
