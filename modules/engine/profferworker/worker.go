package profferworker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/activities"
	sbvadapter "github.com/Cursedpotential/probata/engine/adapters/sbv"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/normalize"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/parser"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Registrations contains the bounded Activity groups that collectively
// implement all 26 Proffer stages. Filesystem and embedded observation bodies are
// separate values so extractor provenance cannot cross activity boundaries.
type Registrations struct {
	Lifecycle             activities.SourceLifecycleActivities
	FilesystemObservation activities.SourceObservationActivities
	InventoryObservation  activities.SourceObservationActivities
	EmbeddedObservation   activities.SourceObservationActivities
	N8N                   platformtemporal.N8NActivities
	N8NFlows              platformtemporal.FlowActivities
	Hash                  activities.HashActivities
	StructuredELT         activities.StructuredELTActivities
	DeriveSMSThreads      activities.DeriveSMSThreadsActivities
	HandlerSelection      HandlerSelectionActivities
	Raw                   activities.RawPipelineActivities
	Normalized            activities.NormalizedPipelineActivities
	Repair                activities.RepairActivities
	Preview               activities.PreviewProjectionActivity
	// BatchImport serves the batch-by-folder workflow. Its fields are nil in a
	// worker built without a Temporal client (RegisterAll still registers the
	// Activities; they fail closed when called unwired).
	// Byline: Claude Code · Opus 5 · 2026-09-21
	BatchImport activities.BatchImportActivities
	// RepairPlan serves RepairPlanWorkflow (the repair workflow builder).
	// Unwired fields fail closed when called.
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	RepairPlan activities.RepairPlanActivities
	// Extraction serves the entity/event extraction workflows (extraction.go).
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	Extraction activities.EntityExtractionActivities
}

// HandlerSelectionActivities is the production integration seam for the
// content-backed recommendation and actor-bound decision-validation stores.
// Their implementations belong with runtime persistence, not workflow
// orchestration. Both must be supplied together before new-version workflows
// are enabled in a deployed worker.
type HandlerSelectionActivities struct {
	Recover   func(context.Context, proffer.HandlerRecoveryRequest) (proffer.HandlerRecommendationResult, error)
	Recommend func(context.Context, proffer.StageRequest) (proffer.HandlerRecommendationResult, error)
	Validate  func(context.Context, proffer.StageRequest) (proffer.HandlerSelectionValidationResult, error)
}

// RegisterAll installs the one workflow plus every exact stagegraph name on
// one worker. A partial worker must never poll this task queue.
func RegisterAll(registrar interface {
	activities.ActivityRegistrar
	RegisterWorkflow(interface{})
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}, registrations Registrations) {
	registrar.RegisterWorkflow(proffer.ProfferWorkflow)
	registrar.RegisterWorkflowWithOptions(proffer.BatchWorkflow, workflow.RegisterOptions{Name: proffer.BatchWorkflowName})
	activities.RegisterBatchImportActivities(registrar, registrations.BatchImport)
	registrar.RegisterWorkflowWithOptions(repairplan.RepairPlanWorkflow, workflow.RegisterOptions{Name: repairplan.WorkflowName})
	activities.RegisterRepairPlanActivities(registrar, registrations.RepairPlan)
	activities.RegisterSourceLifecycleActivities(registrar, registrations.Lifecycle)
	activities.RegisterFilesystemMetadataActivity(registrar, registrations.FilesystemObservation)
	activities.RegisterHashActivities(registrar, registrations.Hash)
	activities.RegisterInventoryContainerActivity(registrar, registrations.InventoryObservation)
	activities.RegisterEmbeddedMetadataActivity(registrar, registrations.EmbeddedObservation)
	registrar.RegisterActivityWithOptions(registrations.N8N.SelectParser, activity.RegisterOptions{Name: string(stagegraph.SelectParser)})
	registrar.RegisterActivityWithOptions(registrations.N8N.ExecuteParser, activity.RegisterOptions{Name: string(stagegraph.ExecuteParser)})
	activities.RegisterStructuredELTActivities(registrar, registrations.StructuredELT)
	activities.RegisterDeriveSMSThreadsActivity(registrar, registrations.DeriveSMSThreads)
	if registrations.HandlerSelection.Recommend != nil || registrations.HandlerSelection.Validate != nil {
		if registrations.HandlerSelection.Recommend == nil || registrations.HandlerSelection.Validate == nil {
			panic("proffer worker: handler recommendation and validation activities must be registered together")
		}
		registrar.RegisterActivityWithOptions(registrations.HandlerSelection.Recommend, activity.RegisterOptions{Name: proffer.RecommendHandlerActivityName})
		registrar.RegisterActivityWithOptions(registrations.HandlerSelection.Validate, activity.RegisterOptions{Name: proffer.ValidateHandlerSelectionActivityName})
		if registrations.HandlerSelection.Recover != nil {
			registrar.RegisterActivityWithOptions(registrations.HandlerSelection.Recover, activity.RegisterOptions{Name: proffer.RecoverHandlerActivityName})
		}
	}
	registrar.RegisterActivityWithOptions(registrations.N8NFlows.RunFlow, activity.RegisterOptions{Name: platformtemporal.RunFlowActivityName})
	activities.RegisterRawPipelineActivities(registrar, registrations.Raw)
	activities.RegisterNormalizedPipelineActivities(registrar, registrations.Normalized)
	activities.RegisterRepairActivities(registrar, registrations.Repair)
	activities.RegisterPreviewProjectionActivity(registrar, registrations.Preview)
}

// Run constructs concrete production adapters, verifies PostgreSQL and shared
// storage before polling, and serves the dedicated Proffer queue until shutdown.
func Run(ctx context.Context, cfg Config) error {
	if stringsTrim(cfg.TemporalTaskQueue) == "" {
		return errors.New("proffer worker: TEMPORAL_TASK_QUEUE is required")
	}
	if cfg.TemporalTaskQueue == legacyEvidenceTaskQueue {
		return errors.New("proffer worker: refusing legacy evidence-pipeline task queue")
	}
	if err := validateSharedPaths(cfg); err != nil {
		return err
	}
	flowRegistry, err := loadConfiguredFlowBindings(cfg.N8NFlowBindingsFile)
	if err != nil {
		return err
	}
	if err := prepareSharedPaths(cfg); err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("proffer worker: configure platform database pool: invalid configuration")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return errors.New("proffer worker: connect to platform database: unavailable")
	}
	if err := platformpostgres.ProbeProfferSchema(ctx, pool); err != nil {
		return err
	}

	// The Temporal client is dialed BEFORE the registrations are built: the
	// batch-by-folder Activities read a run's durable lifecycle through it.
	// Byline: Claude Code · Opus 5 · 2026-09-21
	temporalClient, err := client.Dial(client.Options{HostPort: cfg.TemporalHostPort, Namespace: cfg.TemporalNamespace})
	if err != nil {
		return fmt.Errorf("proffer worker: connect to Temporal: %w", err)
	}
	defer temporalClient.Close()

	// The Case Bible catalog is optional and read-only; only
	// repair.find_other_version reads it. The pool connects on first use, so
	// an unreachable catalog never stops the worker starting.
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	var catalog activities.CatalogVersionFinder
	if cfg.Catalog.Enabled {
		catalogPool, err := platformpostgres.OpenCatalogPool(ctx, platformpostgres.CatalogConnection{
			Host: cfg.Catalog.Host, Port: cfg.Catalog.Port, Database: cfg.Catalog.Database,
			User: cfg.Catalog.User, Password: cfg.Catalog.Password,
		})
		if err != nil {
			return fmt.Errorf("proffer worker: %w", err)
		}
		defer catalogPool.Close()
		store, err := platformpostgres.NewCatalogVersionStore(catalogPool)
		if err != nil {
			return err
		}
		catalog = store
	}

	registrations, err := buildRegistrations(pool, cfg, flowRegistry, temporalClient, catalog)
	if err != nil {
		return err
	}

	temporalWorker := worker.New(temporalClient, cfg.TemporalTaskQueue, workerOptions(cfg))
	RegisterAll(temporalWorker, registrations)
	RegisterExtraction(temporalWorker, registrations.Extraction)
	if err := temporalWorker.Start(); err != nil {
		return fmt.Errorf("proffer worker: start Temporal worker: %w", err)
	}
	slog.Info(
		"universal import worker started",
		"task_queue", cfg.TemporalTaskQueue,
		"max_concurrent_activities", cfg.MaxConcurrentActivities,
		"namespace", cfg.TemporalNamespace,
		"activity_count", len(stagegraph.Stages),
		"n8n_flow_activity", platformtemporal.RunFlowActivityName,
		"n8n_flow_binding_count", flowRegistry.Count(),
	)
	<-ctx.Done()
	temporalWorker.Stop()
	return nil
}

// DERIVE_SCRATCH_DIR is read by LoadConfig (config.go) into
// Config.DeriveScratchDir, validated as an absolute path there and created by
// prepareSharedPaths, so the worker no longer reads the variable here.

// derivationStores opens one S3 client per configured object-store scheme, on
// first use, from the same OBJECT_STORES_JSON the acquisition resolvers read.
func derivationStores(stores objectstores.Stores) func(string) (smsthreads.ObjectStore, error) {
	var mu sync.Mutex
	opened := map[string]smsthreads.ObjectStore{}
	return func(scheme string) (smsthreads.ObjectStore, error) {
		mu.Lock()
		defer mu.Unlock()
		if store, ok := opened[scheme]; ok {
			return store, nil
		}
		credentialFile, ok := stores[scheme]
		if !ok {
			return nil, fmt.Errorf("scheme %q is not a configured object store (configured: %v)", scheme, stores.Schemes())
		}
		cfg, err := acquisition.LoadObjectStorageConfigFile(credentialFile)
		if err != nil {
			return nil, err
		}
		client, err := acquisition.NewS3Client(cfg)
		if err != nil {
			return nil, err
		}
		opened[scheme] = smsthreads.S3Store{Client: client}
		return opened[scheme], nil
	}
}

// workerOptions bounds how much of the queue one worker takes at once.
func workerOptions(cfg Config) worker.Options {
	return worker.Options{MaxConcurrentActivityExecutionSize: cfg.MaxConcurrentActivities}
}

// batchFolderLister adapts one configured object store per scheme into the
// narrow listing seam the batch Activity takes.
func batchFolderLister(stores objectstores.Stores) activities.BatchFolderLister {
	var mu sync.Mutex
	opened := map[string]acquisition.ObjectLister{}
	return func(ctx context.Context, scheme, bucket, prefix, cursor string, limit int32) ([]string, string, error) {
		mu.Lock()
		lister, ok := opened[scheme]
		if !ok {
			credentialFile, configured := stores[scheme]
			if !configured {
				mu.Unlock()
				return nil, "", fmt.Errorf("scheme %q is not a configured object store (configured: %v)", scheme, stores.Schemes())
			}
			storeCfg, err := acquisition.LoadObjectStorageConfigFile(credentialFile)
			if err != nil {
				mu.Unlock()
				return nil, "", err
			}
			s3client, err := acquisition.NewS3Client(storeCfg)
			if err != nil {
				mu.Unlock()
				return nil, "", err
			}
			lister = acquisition.S3Lister{Client: s3client}
			opened[scheme] = lister
		}
		mu.Unlock()
		page, err := lister.ListObjectsPage(ctx, bucket, prefix, cursor, limit)
		if err != nil {
			return nil, "", err
		}
		return page.Keys, page.NextCursor, nil
	}
}

func buildRegistrations(pool *pgxpool.Pool, cfg Config, flowRegistry *platformtemporal.FlowRegistry, temporalClient client.Client, catalog activities.CatalogVersionFinder) (Registrations, error) {
	openObject, err := runtimeapi.NewRetainedObjectOpener(pool)
	if err != nil {
		return Registrations{}, err
	}
	// The API boundary (runtimeapi/source_ref.go) admits only upload:// and
	// r2:// source refs, so a worker wired with the file:// resolver alone can
	// never resolve anything a caller is allowed to send: every run died in
	// retain_original_activity with "acquisition reference must be a file://
	// URI" (live rehearsal 2026-09-02, docs/reviews/2026-09-02-proffer-rehearsal-
	// acquisition-seam.md). acquisition.NewSchemeRouter is the seam that closes
	// this, and its own doc comment names exactly this wiring.
	//
	// file:// stays registered for internal callers that mint sealed refs
	// (acquisition/seal.go returns file:// URIs); it is not reachable from the
	// HTTP boundary.
	//
	// ~~r2:// is deliberately NOT registered yet: the Go worker has no R2
	// credential plumbing~~ CORRECTED 2026-09-05 (live rehearsal
	// req-rehearsal-20260905-r2-1788608263 died in retain_original_activity with
	// `no acquisition resolver registered for scheme "r2"`): the worker compose
	// has mounted /run/secrets/casebible-r2.json and set CASEBIBLE_R2_CONFIG_PATH
	// since 2026-08-29, and the API boundary admits r2:// refs, so r2:// is now
	// registered exactly as the tool gateway does it (cmd/tool-gateway/main.go
	// buildResolver), sealing into the same SOURCE_OBJECT_DIR. Cross-host source
	// bytes travel via object storage (D-132). Absent the config path, r2://
	// stays unregistered and fails closed as before.
	filesystemResolver, err := runtimeapi.NewFilesystemImmutableAcquisitionResolver(cfg.SourceObjectDir)
	if err != nil {
		return Registrations{}, err
	}
	uploadResolver, err := acquisition.NewUploadIngressResolver(cfg.SourceObjectDir)
	if err != nil {
		return Registrations{}, err
	}
	resolvers := map[string]platformpostgres.ImmutableAcquisitionResolver{
		"file":   filesystemResolver,
		"upload": uploadResolver,
	}
	// Object stores are configuration (OBJECT_STORES_JSON), not code: every
	// S3-compatible store is registered under the scheme it is configured with.
	// CASEBIBLE_R2_CONFIG_PATH is honoured only while OBJECT_STORES_JSON is unset.
	stores, err := objectstores.StoresFromEnv()
	if err != nil {
		return Registrations{}, err
	}
	storeResolvers, err := acquisition.ObjectStoreResolvers(cfg.SourceObjectDir, stores, map[string]string{
		"r2": os.Getenv("CASEBIBLE_R2_CONFIG_PATH"),
		"b2": os.Getenv("B2_CONFIG_PATH"),
	})
	if err != nil {
		return Registrations{}, fmt.Errorf("object store acquisition config: %w", err)
	}
	for scheme, resolver := range storeResolvers {
		resolvers[scheme] = resolver
	}
	acquisitionResolver, err := acquisition.NewSchemeRouter(resolvers)
	if err != nil {
		return Registrations{}, err
	}
	lifecycleRepo, err := platformpostgres.NewSourceLifecycleRepository(pool, acquisitionResolver)
	if err != nil {
		return Registrations{}, err
	}
	manifestFactory, err := runtimeapi.NewFilesystemInventoryManifestFactory(cfg.InventoryManifestDir)
	if err != nil {
		return Registrations{}, err
	}
	observationRepo, err := platformpostgres.NewSourceObservationRepository(pool, manifestFactory)
	if err != nil {
		return Registrations{}, err
	}
	filesystemExtractor, err := runtimeapi.NewFilesystemMetadataExtractor(pool)
	if err != nil {
		return Registrations{}, err
	}
	embeddedExtractor, err := runtimeapi.NewEmbeddedMetadataExtractor(pool)
	if err != nil {
		return Registrations{}, err
	}
	memberEnumerator, err := runtimeapi.NewZIPMemberEnumerator(pool)
	if err != nil {
		return Registrations{}, err
	}
	hashRepo, err := platformpostgres.NewRepository(pool, openObject)
	if err != nil {
		return Registrations{}, err
	}
	structuredELTRepo, err := platformpostgres.NewStructuredELTRepository(pool)
	if err != nil {
		return Registrations{}, err
	}
	decoderAdapters, err := sbvadapter.NewAll(openObject)
	if err != nil {
		return Registrations{}, fmt.Errorf("build handler recommendation decoder capabilities: %w", err)
	}
	decoderRegistry, err := parser.NewRegistry(decoderAdapters...)
	if err != nil {
		return Registrations{}, fmt.Errorf("build handler recommendation decoder registry: %w", err)
	}
	handlerSelectionStore, err := platformpostgres.NewHandlerSelectionStoreWithRegistry(pool, openObject, decoderRegistry)
	if err != nil {
		return Registrations{}, err
	}
	parserBundleFactory, err := runtimeapi.NewFilesystemBundleFactory(pool, cfg.ParserBundleDir)
	if err != nil {
		return Registrations{}, err
	}
	parserStore, err := platformpostgres.NewParserStore(pool, parserBundleFactory)
	if err != nil {
		return Registrations{}, err
	}
	rawRepo, err := platformpostgres.NewRawPipelineRepository(pool, openObject)
	if err != nil {
		return Registrations{}, err
	}
	normalizedWriter, err := runtimeapi.NewFilesystemNormalizedBundleFactory(pool, cfg.NormalizedBundleDir)
	if err != nil {
		return Registrations{}, err
	}
	normalizedReader, err := runtimeapi.NewFilesystemNormalizedBundleReaderFactory(pool, openObject)
	if err != nil {
		return Registrations{}, err
	}
	normalizedRepo, err := platformpostgres.NewNormalizedPipelineRepository(pool, normalizedWriter, normalizedReader)
	if err != nil {
		return Registrations{}, err
	}
	repairStore, err := platformpostgres.NewRepairActivityStore(pool, []string{cfg.SourceObjectDir, cfg.ParserBundleDir, cfg.NormalizedBundleDir})
	if err != nil {
		return Registrations{}, err
	}
	// D-132: Activities reach tools ONLY through the gateway, addressing the
	// source by locator and authenticating with the mounted service token.
	toolsClient, err := runtimeapi.NewToolGatewayClient(cfg.PlatformToolsBaseURL, cfg.ToolGatewayServiceToken)
	if err != nil {
		return Registrations{}, err
	}
	previewStore, err := platformpostgres.NewProfferPreviewStore(pool, nil)
	if err != nil {
		return Registrations{}, err
	}
	n8nClient, err := platformtemporal.NewN8NClient(cfg.temporalConfig())
	if err != nil {
		return Registrations{}, err
	}
	// The batch workflow reads each item's lifecycle through the same
	// Temporal query the HTTP surface uses.
	batchOperations, err := platformtemporal.NewWorkflowStarter(temporalClient, cfg.TemporalTaskQueue)
	if err != nil {
		return Registrations{}, err
	}
	// The derive route streams from, and republishes into, the source's own
	// object store — the same OBJECT_STORES_JSON configuration the
	// acquisition resolvers above use. No provider is named in code.
	// Byline: Claude Code · Opus 5 · 2026-09-20
	deriveStore, err := platformpostgres.NewDeriveStore(pool)
	if err != nil {
		return Registrations{}, err
	}
	// Where derived output lands is configuration (DERIVED_ROOTS_JSON), never
	// code. Unset means every source falls back to beside-the-original; a
	// malformed or unreachable value is a loud boot failure, never a silent
	// fallback (owner, 2026-09-21).
	// Byline: Claude Code · Opus 5 · 2026-09-21
	derivedRoots, err := smsthreads.DerivedRootsFromEnv()
	if err != nil {
		return Registrations{}, err
	}
	if err := derivedRoots.RequireConfiguredSchemes(stores.Schemes()); err != nil {
		return Registrations{}, err
	}
	// One store resolver serves the derive route and the repair tools, so
	// both publish through the same configured clients.
	objectStores := derivationStores(stores)
	repairPlan, err := buildRepairPlanActivities(pool, cfg, stores, derivedRoots, objectStores, catalog)
	if err != nil {
		return Registrations{}, err
	}
	extraction, err := buildExtraction(pool)
	if err != nil {
		return Registrations{}, err
	}
	return Registrations{
		RepairPlan:            repairPlan,
		Extraction:            extraction,
		Lifecycle:             activities.NewSourceLifecycleActivities(lifecycleRepo),
		FilesystemObservation: activities.NewSourceObservationActivities(filesystemExtractor, nil, observationRepo),
		InventoryObservation:  activities.NewSourceObservationActivities(nil, memberEnumerator, observationRepo),
		EmbeddedObservation:   activities.NewSourceObservationActivities(embeddedExtractor, nil, observationRepo),
		N8N:                   platformtemporal.N8NActivities{Client: n8nClient},
		N8NFlows:              platformtemporal.FlowActivities{Client: n8nClient, Registry: flowRegistry},
		Hash:                  activities.NewHashActivities(hashRepo),
		StructuredELT:         activities.NewStructuredELTActivities(structuredELTRepo, parserStore, handlerSelectionStore),
		DeriveSMSThreads: activities.NewDeriveSMSThreadsActivities(
			deriveStore, objectStores, deriveStore,
			derivedRoots, cfg.DeriveScratchDir, cfg.DeriveMaxChunkBytes,
		),
		HandlerSelection: HandlerSelectionActivities{
			Recover: handlerSelectionStore.RecoverHandler,
			Recommend: func(ctx context.Context, req proffer.StageRequest) (proffer.HandlerRecommendationResult, error) {
				attempt := activity.GetInfo(ctx).Attempt
				if attempt < 1 {
					attempt = 1
				}
				return handlerSelectionStore.RecommendHandler(ctx, req, attempt)
			},
			Validate: func(ctx context.Context, req proffer.StageRequest) (proffer.HandlerSelectionValidationResult, error) {
				attempt := activity.GetInfo(ctx).Attempt
				if attempt < 1 {
					attempt = 1
				}
				return handlerSelectionStore.ValidateHandlerSelection(ctx, req, attempt)
			},
		},
		BatchImport: activities.BatchImportActivities{
			Lister: batchFolderLister(stores), Bindings: previewStore, Operations: batchOperations,
		},
		Raw:        activities.NewRawPipelineActivities(rawRepo),
		Normalized: activities.NewNormalizedPipelineActivities(normalizedRepo, normalize.GenericMessageNormalizer{}),
		Repair:     activities.NewRepairActivities(toolsClient, repairStore),
		Preview:    activities.PreviewProjectionActivity{Store: previewStore},
	}, nil
}

// loadConfiguredFlowBindings preserves the legitimate no-extra-flows mode,
// while making an explicitly configured file a startup contract. A typo or a
// missing Coolify mount must not silently become an empty registry.
func loadConfiguredFlowBindings(path string) (*platformtemporal.FlowRegistry, error) {
	if strings.TrimSpace(path) == "" {
		return platformtemporal.LoadFlowBindings("")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("proffer worker: N8N_FLOW_BINDINGS_FILE is configured but unavailable: %w", err)
	}
	registry, err := platformtemporal.LoadFlowBindings(path)
	if err != nil {
		return nil, fmt.Errorf("proffer worker: invalid N8N_FLOW_BINDINGS_FILE: %w", err)
	}
	return registry, nil
}

func prepareSharedPaths(cfg Config) error {
	for name, path := range map[string]string{
		"SOURCE_OBJECT_DIR":      cfg.SourceObjectDir,
		"PARSER_BUNDLE_DIR":      cfg.ParserBundleDir,
		"NORMALIZED_BUNDLE_DIR":  cfg.NormalizedBundleDir,
		"INVENTORY_MANIFEST_DIR": cfg.InventoryManifestDir,
		"DERIVE_SCRATCH_DIR":     cfg.DeriveScratchDir,
	} {
		if err := os.MkdirAll(path, 0o750); err != nil {
			return fmt.Errorf("proffer worker: create %s: %w", name, err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("proffer worker: inspect %s: %w", name, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("proffer worker: %s must be a real directory", name)
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
			return fmt.Errorf("proffer worker: %s must not traverse a symlink or junction", name)
		}
	}
	return nil
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
