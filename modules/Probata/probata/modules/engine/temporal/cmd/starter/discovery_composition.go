// Byline: Codex · GPT-6 · 2026-10-11
package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/Cursedpotential/probata/engine/discoveryconfig"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
	"go.temporal.io/sdk/client"
)

// optionalDiscoveryHandler composes authenticated discovery only under explicit configuration.
// Inputs: shared database/client/queue/service credential. Outputs: optional HTTP surface and restart recovery.
// Effects: reads discovery configuration; malformed configuration leaves existing routes operational.
// Choose at the existing starter composition; recovery resumes only committed analysis dispatches.
func optionalDiscoveryHandler(db platformpostgres.DB, c client.Client, queue, tokenFile string) (http.Handler, func(context.Context)) {
	cfg, err := discoveryconfig.FromEnvironment()
	if err != nil {
		slog.Error("adaptive discovery route unavailable", "error", err)
		return nil, nil
	}
	if !cfg.Enabled {
		return nil, nil
	}
	store, err := platformpostgres.NewDiscoveryStore(db, cfg.Datasets)
	if err != nil {
		slog.Error("adaptive discovery route unavailable: persistence configuration")
		return nil, nil
	}
	starter, err := platformtemporal.NewDiscoveryStarter(c, queue)
	if err != nil {
		slog.Error("adaptive discovery route unavailable: orchestration configuration")
		return nil, nil
	}
	handler, err := runtimeapi.NewDiscoveryHTTPHandler(store, starter, tokenFile)
	if err != nil {
		slog.Error("adaptive discovery route unavailable: authentication configuration")
		return nil, nil
	}
	return handler.Routes(), func(ctx context.Context) { reconcileDiscovery(ctx, store, starter) }
}

// reconcileDiscovery retries durable pending dispatches after outages or process restarts.
// Inputs: lifecycle context and committed run store/Temporal starter. Outputs: none.
// Effects: idempotently starts workflows and records dispatch coordinates in analysis tables.
// Choose only for enabled discovery; SQL remains the authority and revision races are retried next pass.
func reconcileDiscovery(ctx context.Context, store discovery.Store, starter *platformtemporal.DiscoveryStarter) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		runs, err := store.PendingDispatch(attemptCtx, 20)
		if err == nil {
			for _, run := range runs {
				workflowID, temporalRunID, startErr := starter.Start(attemptCtx, run)
				if startErr == nil {
					_ = store.RecordDispatch(attemptCtx, run.Request.RunID, run.Request.Scope, run.FeedbackRevision, workflowID, temporalRunID)
				}
			}
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
