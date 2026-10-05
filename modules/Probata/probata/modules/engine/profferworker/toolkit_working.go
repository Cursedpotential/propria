// Byline: Codex · GPT-6 · 2026-10-05. Optional dedicated working catalog setup; parent owns worker.go integration.
package profferworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Cursedpotential/probata/engine/activities"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// toolkitWorkingRepository owns the dedicated working catalog connection and its shutdown lifecycle.
// Inputs: authenticated working metadata operations. Outputs: scoped registration/readback and Close.
// Effects: delegated metadata operations only; choose over the separate recovery and read-only catalog repositories.
type toolkitWorkingRepository interface {
	activities.ToolkitWorkingCatalogRepository
	Close()
}

// toolkitWorkingOpener injects explicit mounted-file connection admission for optional worker setup.
// Inputs: context and absolute DSN-file path. Outputs: admitted repository or error.
// Effects: connection/file reads only; choose for configuration tests without credentials, schema application or catalog execution.
type toolkitWorkingOpener func(context.Context, string) (toolkitWorkingRepository, error)

// ConfigureToolkitWorkingCatalog optionally constructs working catalog Activities using their own mounted DSN file.
// Inputs: context, CASEBIBLE_WORKING_DATABASE_URL_FILE value and existing absolute TOOLKIT_INVENTORY_ROOT.
// Outputs: optional Activity group and once-only cleanup, or explicit startup failure.
// Effects: mounted-root metadata and dedicated connection admission reads only; no DDL, catalog writes or B2 operations.
// Choose beside configureToolkitCatalog after parent worker integration; an unset file leaves the group disabled.
func ConfigureToolkitWorkingCatalog(ctx context.Context, file, root string) (*activities.ToolkitWorkingCatalogActivities, func(), error) {
	return configureToolkitWorkingWithOpener(ctx, file, root, func(ctx context.Context, path string) (toolkitWorkingRepository, error) {
		return platformpostgres.OpenToolkitWorkingCatalogFromFile(ctx, path)
	})
}

// configureToolkitWorkingWithOpener checks opt-in configuration and guarantees ownership of any admitted connection.
// Inputs: context, optional file, mounted metadata root and injected opener. Outputs: group/cleanup or sanitized failure.
// Effects: root stat and opener only when enabled; choose for lifecycle/configuration tests rather than a live worker run.
func configureToolkitWorkingWithOpener(ctx context.Context, file, root string, open toolkitWorkingOpener) (*activities.ToolkitWorkingCatalogActivities, func(), error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !filepath.IsAbs(file) {
		return nil, nil, errors.New("proffer worker: CASEBIBLE_WORKING_DATABASE_URL_FILE must be absolute")
	}
	root = strings.TrimSpace(root)
	if !filepath.IsAbs(root) {
		return nil, nil, errors.New("proffer worker: working catalog requires the existing absolute TOOLKIT_INVENTORY_ROOT")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, nil, errors.New("proffer worker: working catalog inventory root unavailable")
	}
	if open == nil {
		return nil, nil, errors.New("proffer worker: working catalog opener missing")
	}
	repo, err := open(ctx, file)
	if err != nil {
		if repo != nil {
			repo.Close()
		}
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, errors.New("proffer worker: dedicated Case Bible working configuration/admission failed")
	}
	if repo == nil {
		return nil, nil, errors.New("proffer worker: dedicated working repository missing")
	}
	if err = ctx.Err(); err != nil {
		repo.Close()
		return nil, nil, err
	}
	group := activities.NewToolkitWorkingCatalogActivities(root, repo)
	var once sync.Once
	return &group, func() { once.Do(repo.Close) }, nil
}

// ToolkitWorkingRegistrar is the worker registry subset needed for the working catalog's three named registrations.
// Inputs: workflow/Activity functions and Temporal options. Outputs: named worker registry entries.
// Effects: in-memory registration only; choose with the existing worker rather than creating a separate worker or task queue.
type ToolkitWorkingRegistrar interface {
	activities.ActivityRegistrar
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}

// RegisterToolkitWorkingCatalog installs one workflow and two independent Activities only when configured.
// Inputs: existing worker registrar and admitted optional group. Outputs: registration error or three entries.
// Effects: registry mutation only; no connection opening, database writes, workflow start or deployment.
// Choose in parent RegisterAll after Turing's worker.go lane finishes; nil group is an intentional no-op.
func RegisterToolkitWorkingCatalog(registrar ToolkitWorkingRegistrar, group *activities.ToolkitWorkingCatalogActivities) error {
	if group == nil {
		return nil
	}
	if registrar == nil || group.Catalog == nil || !filepath.IsAbs(group.AllowedRoot) || group.Heartbeat == nil {
		return errors.New("proffer worker: working catalog registration requires admitted group and registrar")
	}
	registrar.RegisterWorkflowWithOptions(activities.ToolkitWorkingCatalogWorkflow, workflow.RegisterOptions{Name: activities.ToolkitWorkingCatalogWorkflowName})
	registrar.RegisterActivityWithOptions(group.RegisterToolkitWorkingCatalog, activity.RegisterOptions{Name: activities.ToolkitWorkingCatalogRegisterActivityName})
	registrar.RegisterActivityWithOptions(group.ReadbackToolkitWorkingCatalog, activity.RegisterOptions{Name: activities.ToolkitWorkingCatalogReadbackActivityName})
	return nil
}
