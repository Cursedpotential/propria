// Byline: Claude Code · Opus 5.5 · 2026-10-01
package main

import (
	"errors"
	"net/http"

	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// caseIdentityHandler builds the Workbench Case page routes over the platform
// pool. The registry objects are not needed at startup: until
// scripts/2026-10-01-case-identity-registry.sql is applied the routes answer
// 503 "not installed" and every other starter route keeps working.
func caseIdentityHandler(pool platformpostgres.DB, serviceTokenFile string) (http.Handler, error) {
	store, err := platformpostgres.NewCaseIdentityStore(pool)
	if err != nil {
		return nil, err
	}
	handler, err := runtimeapi.NewCaseIdentityHTTPHandler(store, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}

// mountCaseIdentityRoutes gives the case identity routes their exact patterns
// and leaves every other path to the existing starter routes.
func mountCaseIdentityRoutes(existing, caseIdentity http.Handler) (http.Handler, error) {
	if existing == nil || caseIdentity == nil {
		return nil, errors.New("case identity routes require existing and case identity handlers")
	}
	mux := http.NewServeMux()
	for _, pattern := range runtimeapi.CaseIdentityRoutePatterns {
		mux.Handle(pattern, caseIdentity)
	}
	mux.Handle("/", existing)
	return mux, nil
}
