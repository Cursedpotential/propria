// Byline: Codex · GPT-6 · 2026-10-08.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

type unavailableApprovedProjectionLister struct{ reason error }

// ListApprovedProjections reports a missing analytical configuration through the guarded route.
// Inputs: any scoped request. Outputs: visible service error. Effects: none.
// Pick while the starter is available but its separately configured analytical reader is not.
func (l unavailableApprovedProjectionLister) ListApprovedProjections(context.Context, surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error) {
	return surrealsink.ApprovedProjectionPage{}, l.reason
}

// approvedGraphProjectionsHandler binds the existing analysis identity and shared service token.
// Inputs: mounted token path and existing ANALYSIS_SURREAL_* plus graph policy/service settings.
// Outputs: guarded listing handler. Effects: mounted credential reads only; no graph request until GET.
// Pick beside the query handler so missing analytical configuration returns 503 on this route.
func approvedGraphProjectionsHandler(serviceTokenFile string) (http.Handler, error) {
	var lister runtimeapi.ApprovedProjectionLister
	cfg, err := surrealsink.AnalysisConfigFromEnv()
	if err == nil {
		var client *surrealsink.Client
		client, err = surrealsink.NewAnalysis(cfg)
		if err == nil {
			policy := strings.TrimSpace(os.Getenv("ANALYSIS_GRAPH_ACCESS_POLICY_ID"))
			service := strings.TrimSpace(os.Getenv("ANALYSIS_GRAPH_CREATED_BY_SERVICE"))
			if policy == "" || service == "" {
				err = errors.New("analytical graph policy and service configuration are required")
			} else {
				lister = &surrealsink.ApprovedClaimsSink{Client: client, AccessPolicyID: policy, CreatedByService: service}
			}
		}
	}
	if err != nil {
		lister = unavailableApprovedProjectionLister{reason: fmt.Errorf("approved graph projections: %w", err)}
	}
	handler, err := runtimeapi.NewApprovedGraphProjectionsHTTPHandler(lister, serviceTokenFile)
	if err != nil {
		return nil, err
	}
	return handler.Routes(), nil
}

// mountApprovedGraphProjectionsRoutes adds only the exact GET path ahead of existing routes.
// Inputs: existing combined starter routes and guarded projection handler. Outputs: combined handler.
// Effects: route registration only; choose after query routes so their subtree remains reachable.
func mountApprovedGraphProjectionsRoutes(existing, projections http.Handler) (http.Handler, error) {
	if existing == nil || projections == nil {
		return nil, errors.New("approved graph projection routes require existing and projection handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("GET /reference-import/analysis/projections", projections)
	mux.Handle("/", existing)
	return mux, nil
}
