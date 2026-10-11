// Byline: Codex · GPT-6 · 2026-10-11
package profferworker

import (
	"log/slog"

	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/Cursedpotential/probata/engine/discoveryconfig"
	"github.com/Cursedpotential/probata/engine/discoveryflow"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
)

// registerOptionalDiscovery installs standalone discovery without changing intake registration.
// Inputs: existing registrar/database and explicit discovery settings. Outputs: optional registrations.
// Effects: validates settings and prepares derivative storage; configuration failure disables only discovery.
// Choose alongside existing worker registrations, independently of ingestion and projection.
func registerOptionalDiscovery(registrar ExtractionRegistrar, db platformpostgres.DB) {
	cfg, err := discoveryconfig.FromEnvironment()
	if err != nil {
		slog.Error("adaptive discovery disabled: configuration unavailable", "error", err)
		return
	}
	if !cfg.Enabled {
		return
	}
	store, err := platformpostgres.NewDiscoveryStore(db, cfg.Datasets)
	if err != nil {
		slog.Error("adaptive discovery disabled: persistence unavailable")
		return
	}
	var surreal discovery.SurrealReader
	if len(cfg.Surreal.Tables) > 0 {
		surreal, err = discovery.NewSurrealHTTPReader(cfg.Surreal)
		if err != nil {
			slog.Error("adaptive discovery disabled: selected projection unavailable")
			return
		}
	}
	reader, err := platformpostgres.NewDiscoverySourceReader(db, cfg.Datasets, surreal)
	if err != nil {
		slog.Error("adaptive discovery disabled: source configuration unavailable")
		return
	}
	files, err := discovery.NewFiles(cfg.Root)
	if err != nil {
		slog.Error("adaptive discovery disabled: shared derivative storage unavailable")
		return
	}
	discoveryflow.Register(registrar, discovery.Activities{Store: store, Reader: reader, Files: files, ModelConfig: cfg.ModelConfig})
	slog.Info("adaptive discovery registered", "dataset_count", len(cfg.Datasets))
}
