// Byline: Codex · GPT-6.1 · 2026-10-05. Optional sync registration on the existing worker; no runtime or scheduling changes.
package profferworker

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

// configureToolkitSync admits optional B2 sync using the already configured validator's artifacts, extractor and signing key.
// Inputs: context, validator and TOOLKIT_LIBRARY_SYNC_* environment/files; outputs: optional fifteen-Activity group or safe error.
// Effects: bounded configuration reads/client construction only. Choose after configureToolkitValidation; no schedule, dispatch or transfer is started.
func configureToolkitSync(ctx context.Context, validation *activities.ToolkitLibraryValidationActivities) (*activities.ToolkitLibrarySyncActivities, error) {
	return configureToolkitSyncWithFactory(ctx, os.Getenv(librarysync.EnvBackendURL), validation, librarysync.NewServiceFromEnv)
}

// configureToolkitSyncWithFactory checks explicit opt-in and dependency admission before invoking the sync constructor.
// Inputs: context, optional HTTPS origin, configured validator and injected factory; outputs: optional group or sanitized startup error.
// Effects: constructor only when admitted; canceled/configuration failures cannot silently fall back or expose credentials.
// Choose for focused tests without B2, database, Temporal or Python runtime calls.
func configureToolkitSyncWithFactory(ctx context.Context, endpoint string, validation *activities.ToolkitLibraryValidationActivities, factory func(libraryvalidation.Artifacts, libraryvalidation.Extractor, []byte) (*librarysync.Service, error)) (*activities.ToolkitLibrarySyncActivities, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("proffer worker: TOOLKIT_LIBRARY_SYNC_BACKEND_URL requires a fixed HTTPS origin without credentials, path, query or fragment")
	}
	if validation == nil || validation.Service == nil || validation.Service.Artifacts == nil || validation.Service.Extractor == nil || len(validation.Service.SigningKey) < 32 {
		return nil, errors.New("proffer worker: toolkit sync requires configured toolkit validation artifacts, extractor and signing key")
	}
	if factory == nil {
		return nil, errors.New("proffer worker: toolkit sync constructor missing")
	}
	v := validation.Service
	service, err := factory(v.Artifacts, v.Extractor, v.SigningKey)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if err != nil || service == nil {
		return nil, errors.New("proffer worker: toolkit sync service configuration failed")
	}
	return &activities.ToolkitLibrarySyncActivities{Service: service}, nil
}
