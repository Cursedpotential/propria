// Byline: Claude Code · Opus 5.5 · 2026-09-26
package main

import (
	"errors"
	"net/http"

	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// reviewOverlayHandlers builds the metadata screen and context review
// handlers over the platform pool. Neither needs the overlay tables at
// startup: until the overlay SQL is applied, their routes answer 503 with a
// "not installed" reason and every other starter route keeps working.
func reviewOverlayHandlers(pool platformpostgres.DB, serviceTokenFile string) (metadata, contextReview http.Handler, err error) {
	metadataStore, err := platformpostgres.NewSourceMetadataStore(pool)
	if err != nil {
		return nil, nil, err
	}
	metadataHandler, err := runtimeapi.NewSourceMetadataHTTPHandler(metadataStore, serviceTokenFile)
	if err != nil {
		return nil, nil, err
	}
	reviewStore, err := platformpostgres.NewContextReviewStore(pool)
	if err != nil {
		return nil, nil, err
	}
	reviewHandler, err := runtimeapi.NewContextReviewHTTPHandler(reviewStore, serviceTokenFile)
	if err != nil {
		return nil, nil, err
	}
	return metadataHandler.Routes(), reviewHandler.Routes(), nil
}

// mountReviewOverlayRoutes gives the metadata screen and the context review
// overlays their exact routes under /reference-import/previews/ and leaves
// every other path to the existing starter routes.
func mountReviewOverlayRoutes(existing, metadata, contextReview http.Handler) (http.Handler, error) {
	if existing == nil || metadata == nil || contextReview == nil {
		return nil, errors.New("review overlay routes require existing, metadata and context review handlers")
	}
	const preview = "/reference-import/previews/{preview_handle}"
	mux := http.NewServeMux()
	mux.Handle("GET "+preview+"/metadata", metadata)
	mux.Handle("POST "+preview+"/metadata/corrections", metadata)
	mux.Handle("GET "+preview+"/messages/{message_id}/context-review", contextReview)
	mux.Handle("POST "+preview+"/messages/{message_id}/context-review", contextReview)
	mux.Handle("POST "+preview+"/messages/{message_id}/foreshadowing", contextReview)
	mux.Handle("/", existing)
	return mux, nil
}
