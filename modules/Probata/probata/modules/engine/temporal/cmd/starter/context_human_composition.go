package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/humancontext"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
)

// mountNativeHumanContextRoutes gives native viewer routes precedence over context starts and legacy imports.
// Inputs: existing, context source and native viewer handlers. Outputs: combined HTTP handler or configuration error.
// Effects: route registration only; choose when both context handlers share the reference-import/context prefix.
func mountNativeHumanContextRoutes(existing, sources, viewer http.Handler) (http.Handler, error) {
	if existing == nil || sources == nil || viewer == nil {
		return nil, errors.New("native human context routes require existing, source and viewer handlers")
	}
	mux := http.NewServeMux()
	mux.Handle("POST /reference-import/context/packages/{source_version_ref}/review", existing)
	mux.Handle("/reference-import/context/previews/", viewer)
	mux.Handle("/reference-import/context/", sources)
	mux.Handle("/", existing)
	return mux, nil
}

// humanContextBundleRoot resolves the read-only native bundle mount used by this starter.
// Inputs: optional HUMAN_CONTEXT_ROOT environment setting. Outputs: absolute non-root bundle path or error.
// Effects: none; choose for native record viewing, while extraction remains in the separate Go worker.
func humanContextBundleRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("HUMAN_CONTEXT_ROOT"))
	if root == "" {
		root = "/data/proffer/derive-scratch/human-context"
	}
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) || root == string(filepath.Separator) {
		return "", errors.New("native human bundle root must be an absolute non-root directory")
	}
	return root, nil
}

// nativeHumanContextRoutes composes actor-bound starts and read-only native viewer routes on the existing starter.
// Inputs: existing routes, platform pool, shared Temporal client/queue and service-token path. Outputs: guarded routes.
// Effects: validates existing authentication and mount configuration; no source reads, schema calls or workflow starts.
// Choose to expose the native human-source contract beside the established raw preview and legacy import surfaces.
func nativeHumanContextRoutes(existing http.Handler, pool *pgxpool.Pool, temporalClient client.Client, queue, tokenPath string) (http.Handler, error) {
	if pool == nil {
		return nil, errors.New("native human context routes require the existing platform pool")
	}
	root, err := humanContextBundleRoot()
	if err != nil {
		return nil, err
	}
	sources, err := contextSourceHandler(temporalClient, queue, tokenPath)
	if err != nil {
		return nil, err
	}
	viewer, err := runtimeapi.NewHumanContextViewerHandler(&platformpostgres.HumanSourceMetadata{DB: pool}, &humancontext.Bundles{Root: root}, tokenPath)
	if err != nil {
		return nil, err
	}
	return mountNativeHumanContextRoutes(existing, sources, viewer.Routes())
}
