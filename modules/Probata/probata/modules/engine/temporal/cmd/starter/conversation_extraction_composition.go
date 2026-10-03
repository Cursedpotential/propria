// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package main

import (
	"errors"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountConversationExtractionRoutes adds the conversation extraction and Surreal send surface
// without changing any existing route's ownership.
func mountConversationExtractionRoutes(existing, conversation http.Handler) (http.Handler, error) {
	if existing == nil || conversation == nil {
		return nil, errors.New("conversation extraction routes require existing and conversation handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("GET /reference-import/extractors", conversation)
	mux.Handle("/reference-import/conversations/", conversation)
	mux.Handle("/", existing)
	return mux, nil
}

// conversationExtractionHandler builds the routes over the starter's Temporal client (same task queue as the worker).
func conversationExtractionHandler(temporalClient client.Client, taskQueue, serviceTokenFile string) (http.Handler, error) {
	starter, err := flow.NewStarter(temporalClient, taskQueue)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewConversationExtractionHTTPHandler(starter, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}
