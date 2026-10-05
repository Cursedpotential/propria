// Byline: Codex · GPT-6.1 · 2026-10-04. Optional proposal validation on the existing worker; no new service.
package profferworker

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

// configureToolkitValidation optionally admits the validator using the existing B2 credential format and bounded parser runtime.
// Inputs: context and TOOLKIT_VALIDATION_* environment/files. Outputs: configured four-Activity group, nil when disabled, or visible failure.
// Effects: bounded credential reads, client construction and service parser preflight; no network requests, record writes or source transfers.
// Choose in the existing worker Run beside configureToolkitCatalog; unset Case MCP URL leaves validation unregistered.
func configureToolkitValidation(ctx context.Context) (*activities.ToolkitLibraryValidationActivities, error) {
	return configureToolkitValidationWithFactories(ctx, os.Getenv(libraryvalidation.EnvCaseMCPURL), func() (libraryvalidation.VersionStore, error) {
		cfg, err := acquisition.LoadObjectStorageConfigFile(strings.TrimSpace(os.Getenv(libraryvalidation.EnvB2ConfigFile)))
		if err != nil {
			return nil, err
		}
		if err := libraryvalidation.ValidateB2StorageEndpoint(cfg.Endpoint); err != nil {
			return nil, err
		}
		client, err := acquisition.NewS3Client(cfg)
		if err != nil {
			return nil, err
		}
		return smsthreads.S3Store{Client: client}, nil
	}, libraryvalidation.NewScopedServiceFromEnv)
}

// configureToolkitValidationWithFactories checks opt-in configuration and propagates cancellation around bounded service admission.
// Inputs: context, optional Case MCP URL and injected store/service constructors. Outputs: optional Activity group or sanitized startup failure.
// Effects: constructor calls only when enabled; no fallback to conversation credentials or an alternative model/runtime.
// Choose for focused configuration tests without secrets, Python processing, live services or database access.
func configureToolkitValidationWithFactories(ctx context.Context, endpoint string, storeFactory func() (libraryvalidation.VersionStore, error), serviceFactory func(libraryvalidation.VersionStore) (*libraryvalidation.Service, error)) (*activities.ToolkitLibraryValidationActivities, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("proffer worker: TOOLKIT_VALIDATION_CASE_MCP_URL must be an absolute HTTP service URL without credentials or fragment")
	}
	if storeFactory == nil || serviceFactory == nil {
		return nil, errors.New("proffer worker: toolkit validation constructors missing")
	}
	store, err := storeFactory()
	if err != nil || store == nil {
		return nil, errors.New("proffer worker: toolkit validation B2 configuration failed")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	service, err := serviceFactory(store)
	if err != nil || service == nil {
		return nil, errors.New("proffer worker: toolkit validation service/runtime configuration failed")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return &activities.ToolkitLibraryValidationActivities{Service: service}, nil
}
