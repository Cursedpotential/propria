// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

// Command cf-jobs-worker runs the Temporal worker for the Case Bible's Cloudflare bulk-read jobs: format sniffing, ZIP member
// listing and the B2 hash backfill (engine/cfjobs). It polls its own task queue, never the proffer worker's.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cursedpotential/probata/engine/cfjobs"
)

func main() {
	cfg, err := cfjobs.LoadConfig()
	if err != nil {
		slog.Error("cf-jobs worker configuration invalid", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cfjobs.Run(ctx, cfg); err != nil {
		slog.Error("cf-jobs worker stopped", "error", err)
		os.Exit(1)
	}
}
