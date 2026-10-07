// Byline: Codex · GPT-5 · 2026-10-05 (versioned import admission).
// Byline: Codex · GPT-6 · 2026-10-07 (optional real analytical graph registration).
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
	"github.com/Cursedpotential/probata/engine/contacts"
	"github.com/Cursedpotential/probata/engine/contextgraphflow"
	"github.com/Cursedpotential/probata/engine/dedupe"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/Cursedpotential/probata/engine/normalize"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/parser"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/Cursedpotential/probata/engine/superindex"
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
	// Independent byte-only check; never inserted into the base Proffer sequence.
	// Byline: Codex, 2026-10-06.
	Integrity        activities.SourceIntegrityActivities
	StructuredELT    activities.StructuredELTActivities
	DeriveSMSThreads activities.DeriveSMSThreadsActivities
	HandlerSelection HandlerSelectionActivities
	Raw              activities.RawPipelineActivities
	Normalized       activities.NormalizedPipelineActivities
	Repair           activities.RepairActivities
	Preview          activities.PreviewProjectionActivity
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
	// Conversation serves the conversation-level extraction request and the
	// Surreal send (conversation_extraction.go).
	// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	Conversation activities.ConversationActivities
	// ContextGraph serves independent downstream projection when explicitly configured.
	// Inputs are sealed source refs; outputs are verified checkpoints. Registration
	// alone has no data effects and does not add an intake requirement.
	ContextGraph *activities.ContextGraphActivities
	// ContextSearch is publish_context_search_activity, the Weaviate-first
	// stage every new Proffer run schedules before the owner's approval.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	ContextSearch activities.PublishContextSearchActivities
	// FirstPartyContext is the first-party context import (D04): propose,
	// confirm, and the spine and thread commits.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	FirstPartyContext activities.FirstPartyContextActivities
	// CallLog commits a generation's call records to working.call_log (owner
	// 2026-10-02). Byline: Claude Code · Opus 5.5 · 2026-10-02
	CallLog activities.CallLogActivities
	// MessageMatch is match_message_occurrences_activity (owner 2026-10-02,
	// message match-up). Byline: Claude Code · Opus 5.5 · 2026-10-02
	MessageMatch activities.MessageMatchActivities
	// MessageDedupe is the step Activities of message_dedupe_workflow (owner
	// 2026-10-02 20:02: Temporal, traceable). Byline: Claude Code · Opus 5.5 · 2026-10-02
	MessageDedupe activities.MessageDedupeActivities
	// AutoApproval is record_auto_approval_activity (owner 2026-10-02,
	// "auto-approve clean runs"). Byline: Claude Code · Opus 5.5 · 2026-10-02
	AutoApproval activities.AutoApprovalActivity
	// Contacts are the six Activities of ContactsImportWorkflow (owner 2026-10-02,
	// "traceable Temporal activities"). Byline: Claude Code · Sonnet · 2026-10-02
	Contacts activities.ContactsActivities
	// ToolkitInventory reads local ZIPs without importing them (Codex, 2026-10-04).
	ToolkitInventory activities.ToolkitPackageInventoryActivities
	// ToolkitPreservation retains whole recovered archives and independently verifies remote bytes.
	// Inputs: configured inventory root and object-store resolver; outputs: bounded copy/readback Activities.
	// Effects: no transfers until invoked. Choose separately from inventory and catalog projection.
	// Byline: Codex, 2026-10-04.
	ToolkitPreservation activities.ToolkitPackagePreservationActivities
	// ToolkitContentPlacement writes reviewed complete units to permanent B2 legal homes with pinned readback.
	// Inputs: mounted inventory root and B2 resolver. Outputs: tracked placement Activity group.
	// Effects: none until invoked; use after inventory and reviewed final-unit selection, before shared publication.
	// Byline: Codex · GPT-6 · 2026-10-04.
	ToolkitContentPlacement *activities.ToolkitContentPlacementActivities
	// AIWorkproductPlacement preserves reviewed Markdown units under the B2 AI-chat knowledge home.
	// Inputs: existing derive-scratch root and object-store resolver. Outputs: three independent Activities.
	// Effects: none until invoked; choose for source inspection, retained copy and pinned readback before ingestion.
	// Byline: Codex · GPT-6 · 2026-10-05.
	AIWorkproductPlacement *activities.AIWorkproductPlacementActivities
	// AIWorkproductCatalog authenticates retained notes and registers their original source occurrences.
	// Inputs: existing placement adapters and separately admitted catalog writer. Outputs: three optional Activities.
	// Effects: none until invoked; choose after placement, independently of content indexing and bucket listing refresh.
	// Byline: Codex · GPT-6 · 2026-10-05.
	AIWorkproductCatalog *activities.AIWorkproductCatalogActivities
	// ToolkitCatalog registers verified recovery metadata only when the separate writer is explicitly configured.
	// Inputs: existing preservation root/store resolver and admitted Case Bible writer. Outputs: optional Activity group.
	// Effects: none until invoked. Choose alongside preservation; the dated catalog client remains read-only.
	// Byline: Codex · GPT-6 · 2026-10-04.
	ToolkitCatalog *activities.ToolkitCatalogRegistrationActivities
	// ToolkitWorkingCatalog registers verified permanent files using its separately admitted writer.
	// Inputs: approved metadata root and scoped catalog connection; outputs: registration/readback Activities.
	// Effects: none until invoked; choose after permanent placement, independently of archive recovery.
	// Byline: Codex · GPT-6 · 2026-10-05.
	ToolkitWorkingCatalog *activities.ToolkitWorkingCatalogActivities
	// ToolkitValidation validates saved library proposals through the existing source/parser/NIM contracts when explicitly enabled.
	// Inputs: admitted validation service. Outputs: optional four-Activity group. Effects: none until invoked.
	// Choose separately from preservation/catalog registration; trusted validation does not publish automatically.
	// Byline: Codex · GPT-6.1 · 2026-10-04.
	ToolkitValidation *activities.ToolkitLibraryValidationActivities
	// ToolkitSync optionally observes B2 legal versions and exports guarded saved outbox revisions.
	// Inputs: the existing validator dependencies and explicit sync service configuration; outputs: two workflows/sixteen Activities.
	// Effects: registration only until invoked. Choose alongside validation; parent owns schedule and durable outbox dispatch.
	// Byline: Codex · GPT-6.1 · 2026-10-05.
	ToolkitSync *activities.ToolkitLibrarySyncActivities
	// ToolkitBindings links approved permanent versions after independent placement/catalog verification.
	// Inputs: existing metadata root, storage and private sync backend; outputs: tracked metadata-only import.
	// Effects: none until invoked; choose once before recurring directory observation.
	// Byline: Codex · GPT-6 · 2026-10-05.
	ToolkitBindings *activities.ToolkitLibraryBindingActivities
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

// RegisterAll installs the Proffer workflows and their complete Activity registry on one worker.
// Inputs: registrar and the configured Activity groups. Outputs: registered workflow and Activity names.
// Side effects: mutates the worker registry, including the Super Index workflow whose Activities use Python.
// Pick this complete registry when polling the Proffer queue; partial registries cannot serve that queue.
func RegisterAll(registrar interface {
	activities.ActivityRegistrar
	RegisterWorkflow(interface{})
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
}, registrations Registrations) {
	registrar = operatingRegistrar{registrar}
	registrar.RegisterWorkflow(proffer.ProfferWorkflow)
	// Reuse exact verified AI generations after content-stage failures; no new-source admission.
	// Byline: Codex / 2026-10-06.
	registrar.RegisterWorkflowWithOptions(proffer.AIContentResumeWorkflow, workflow.RegisterOptions{Name: proffer.AIContentResumeWorkflowName})
	if registrations.ContextGraph != nil {
		registrar.RegisterWorkflowWithOptions(contextgraphflow.ContextGraphWorkflow, workflow.RegisterOptions{Name: contextgraphflow.WorkflowName})
		activities.RegisterContextGraphActivities(registrar, *registrations.ContextGraph)
	}
	registrar.RegisterWorkflowWithOptions(proffer.BatchWorkflow, workflow.RegisterOptions{Name: proffer.BatchWorkflowName})
	// Back-fill of call logs imported before commit_call_log existed.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	registrar.RegisterWorkflowWithOptions(proffer.CallLogBackfillWorkflow, workflow.RegisterOptions{Name: proffer.CallLogBackfillWorkflowName})
	// Explicit wrapper and Activity reuse retained originals and existing receipt tables.
	// Byline: Codex, 2026-10-06.
	registrar.RegisterWorkflowWithOptions(proffer.SourceIntegrityWorkflow, workflow.RegisterOptions{Name: proffer.SourceIntegrityWorkflowName})
	activities.RegisterSourceIntegrityActivity(registrar, registrations.Integrity)
	// The re-chunk of committed data and the removal of the per-message objects (owner 2026-10-02: Temporal, traceable).
	registrar.RegisterWorkflowWithOptions(proffer.ConversationChunksBackfillWorkflow, workflow.RegisterOptions{Name: proffer.ConversationChunksBackfillWorkflowName})
	registrar.RegisterWorkflowWithOptions(proffer.ConversationChunksRemovalWorkflow, workflow.RegisterOptions{Name: proffer.ConversationChunksRemovalWorkflowName})
	// The removal of same-device duplicates committed before the match-up rule.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	registrar.RegisterWorkflowWithOptions(dedupe.MessageDedupeWorkflow, workflow.RegisterOptions{Name: dedupe.WorkflowName})
	registrar.RegisterWorkflowWithOptions(superindex.CycleWorkflow, workflow.RegisterOptions{Name: superindex.WorkflowName})
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
	activities.RegisterPublishContextSearchActivity(registrar, registrations.ContextSearch)
	activities.RegisterFirstPartyContextActivities(registrar, registrations.FirstPartyContext)
	activities.RegisterAutoApprovalActivity(registrar, registrations.AutoApproval)
	activities.RegisterCallLogActivities(registrar, registrations.CallLog)
	activities.RegisterMessageMatchActivities(registrar, registrations.MessageMatch)
	// Contacts import: manifest, fetch, parse, people, placeholders, re-link (Claude Code · Sonnet · 2026-10-02).
	registrar.RegisterWorkflowWithOptions(contacts.ContactsImportWorkflow, workflow.RegisterOptions{Name: contacts.WorkflowName})
	activities.RegisterContactsActivities(registrar, registrations.Contacts)
	// Register isolated source inspection on this same worker, not an import flow.
	// Inputs: configured native inventory group; outputs: named workflow and Activity.
	// Effects: worker registration only. Choose when ZIP integrity alone is requested.
	// Byline: Codex, 2026-10-04.
	registrar.RegisterWorkflowWithOptions(activities.ToolkitPackageInventoryWorkflow, workflow.RegisterOptions{Name: activities.ToolkitPackageInventoryWorkflowName})
	registrar.RegisterActivityWithOptions(registrations.ToolkitInventory.RunToolkitPackageInventory, activity.RegisterOptions{Name: activities.ToolkitPackageInventoryActivityName})
	// Register ZIP-only selected-text snapshots as a separate operation on the same worker.
	// Inputs: the configured toolkit group; outputs: named workflow and Activity.
	// Effects: registration only. Choose for pinned audit text rather than whole-package inventory.
	// Byline: Codex, 2026-10-04.
	registrar.RegisterWorkflowWithOptions(activities.ToolkitSelectedTextWorkflow, workflow.RegisterOptions{Name: activities.ToolkitSelectedTextWorkflowName})
	registrar.RegisterActivityWithOptions(registrations.ToolkitInventory.SnapshotSelectedToolkitText, activity.RegisterOptions{Name: activities.ToolkitSelectedTextActivityName})
	// Register mechanistic ledger differences as a separate operation (Codex, 2026-10-04).
	// Inputs: toolkit group; outputs: named workflow and Activity. Effects: registration only.
	// Choose after a pinned text snapshot; this operation never chooses a surviving version.
	registrar.RegisterWorkflowWithOptions(activities.ToolkitLedgerComparisonWorkflow, workflow.RegisterOptions{Name: activities.ToolkitLedgerComparisonWorkflowName})
	registrar.RegisterActivityWithOptions(registrations.ToolkitInventory.CompareToolkitLedgers, activity.RegisterOptions{Name: activities.ToolkitLedgerComparisonActivityName})
	// Register recovered archive copy and independent verification on the existing queue.
	// Inputs: configured preservation group; outputs: named workflow and two Activities.
	// Effects: registration only. Choose for durable original retention without catalog claims.
	// Byline: Codex, 2026-10-04.
	if registrations.ToolkitContentPlacement != nil {
		registrar.RegisterWorkflowWithOptions(activities.ToolkitContentPlacementWorkflow, workflow.RegisterOptions{Name: activities.ToolkitContentPlacementWorkflowName})
		registrar.RegisterActivityWithOptions(registrations.ToolkitContentPlacement.PlaceToolkitContent, activity.RegisterOptions{Name: activities.ToolkitContentPlacementActivityName})
	}
	if registrations.AIWorkproductPlacement != nil {
		registrar.RegisterWorkflowWithOptions(activities.AIWorkproductPlacementWorkflow, workflow.RegisterOptions{Name: activities.AIWorkproductPlacementWorkflowName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductPlacement.InspectAIWorkproduct, activity.RegisterOptions{Name: activities.AIWorkproductInspectActivityName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductPlacement.CopyAIWorkproduct, activity.RegisterOptions{Name: activities.AIWorkproductCopyActivityName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductPlacement.ReadbackAIWorkproduct, activity.RegisterOptions{Name: activities.AIWorkproductReadbackActivityName})
	}
	if registrations.AIWorkproductCatalog != nil {
		registrar.RegisterWorkflowWithOptions(activities.AIWorkproductCatalogWorkflow, workflow.RegisterOptions{Name: activities.AIWorkproductCatalogWorkflowName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductCatalog.AuthenticateAIWorkproductCatalog, activity.RegisterOptions{Name: activities.AIWorkproductCatalogAuthenticateActivityName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductCatalog.RegisterAIWorkproductCatalog, activity.RegisterOptions{Name: activities.AIWorkproductCatalogRegisterActivityName})
		registrar.RegisterActivityWithOptions(registrations.AIWorkproductCatalog.ReadbackAIWorkproductCatalog, activity.RegisterOptions{Name: activities.AIWorkproductCatalogReadbackActivityName})
	}
	registrar.RegisterWorkflowWithOptions(activities.ToolkitPackagePreservationWorkflow, workflow.RegisterOptions{Name: activities.ToolkitPackagePreservationWorkflowName})
	registrar.RegisterActivityWithOptions(registrations.ToolkitPreservation.CopyToolkitPackagePreservation, activity.RegisterOptions{Name: activities.ToolkitPackagePreservationCopyActivityName})
	registrar.RegisterActivityWithOptions(registrations.ToolkitPreservation.VerifyToolkitPackagePreservation, activity.RegisterOptions{Name: activities.ToolkitPackagePreservationVerifyActivityName})
	// Register metadata verification, separate catalog registration and independent readback only after explicit writer admission.
	// Inputs: optional configured group. Outputs: one workflow and three named Activities. Effects: registry additions only.
	// Choose after preservation; disabled configuration does not add a writable catalog seam.
	// Byline: Codex · GPT-6 · 2026-10-04.
	if registrations.ToolkitCatalog != nil {
		registrar.RegisterWorkflowWithOptions(activities.ToolkitCatalogRegistrationWorkflow, workflow.RegisterOptions{Name: activities.ToolkitCatalogRegistrationWorkflowName})
		registrar.RegisterActivityWithOptions(registrations.ToolkitCatalog.VerifyToolkitCatalogMetadata, activity.RegisterOptions{Name: activities.ToolkitCatalogMetadataActivityName})
		registrar.RegisterActivityWithOptions(registrations.ToolkitCatalog.RegisterToolkitCatalog, activity.RegisterOptions{Name: activities.ToolkitCatalogRegisterActivityName})
		registrar.RegisterActivityWithOptions(registrations.ToolkitCatalog.ReadbackToolkitCatalog, activity.RegisterOptions{Name: activities.ToolkitCatalogReadbackActivityName})
	}
	if err := RegisterToolkitWorkingCatalog(registrar, registrations.ToolkitWorkingCatalog); err != nil {
		panic("proffer worker: working catalog registry admission failed")
	}
	// Register substantive proposal validation only after configured service/runtime preflight succeeds.
	// Inputs: optional validation group. Outputs: one workflow and four Activities. Effects: registry additions only.
	// Choose after a saved proposal exists; inventory, preservation and catalog workflows remain independent.
	// Byline: Codex · GPT-6.1 · 2026-10-04.
	if registrations.ToolkitValidation != nil {
		libraryvalidation.RegisterWorkflow(registrar)
		activities.RegisterToolkitLibraryValidationActivities(registrar, *registrations.ToolkitValidation)
	}
	if registrations.ToolkitSync != nil {
		if registrations.ToolkitValidation == nil || registrations.ToolkitValidation.Service == nil || registrations.ToolkitSync.Service == nil {
			panic("proffer worker: toolkit sync requires configured validator and sync service")
		}
		librarysync.RegisterWorkflows(registrar)
		activities.RegisterToolkitLibrarySyncActivities(registrar, *registrations.ToolkitSync)
	}
	if registrations.ToolkitBindings != nil {
		registrar.RegisterWorkflowWithOptions(activities.ToolkitLibraryBindingWorkflow, workflow.RegisterOptions{Name: activities.ToolkitLibraryBindingWorkflowName})
		registrar.RegisterActivityWithOptions(registrations.ToolkitBindings.ImportToolkitLibraryBindings, activity.RegisterOptions{Name: activities.ToolkitLibraryBindingActivityName})
	}
	// Probe the exact preservation adapter with synthetic bytes before any original transfer.
	// Inputs: the existing preservation group; outputs: separately tracked probe workflow and Activity.
	// Effects: registration only. Choose for provider semantics, never source inspection.
	// Byline: Codex, 2026-10-04.
	registrar.RegisterWorkflowWithOptions(activities.ToolkitPackageConditionalWriteProbeWorkflow, workflow.RegisterOptions{Name: activities.ToolkitPackageConditionalWriteProbeWorkflowName})
	probe := activities.NewToolkitPackageConditionalWriteProbeActivities(registrations.ToolkitPreservation)
	registrar.RegisterActivityWithOptions(probe.RunToolkitPackageConditionalWriteProbe, activity.RegisterOptions{Name: activities.ToolkitPackageConditionalWriteProbeActivityName})
	activities.RegisterMessageDedupeActivities(registrar, registrations.MessageDedupe)
}

// Run constructs concrete production adapters, verifies PostgreSQL and shared
// storage before polling, and serves the dedicated Proffer queue until shutdown.
// Inputs: cancellation context and existing worker configuration; optional CASEBIBLE_RECOVERY_DATABASE_URL_FILE admits a separate writer.
// TOOLKIT_VALIDATION_CASE_MCP_URL additionally opts into bounded parser/service admission before validation registration.
// TOOLKIT_LIBRARY_SYNC_BACKEND_URL opts into sync only after validator admission; its workflows use proffer-v1.
// Outputs: startup/shutdown error or nil. Effects: opens/closes clients, registers and polls existing workflows; no automatic recovery writes or DDL.
// Choose for the existing Proffer worker; recovery registration requires its own explicit workflow invocation.
// Byline: Codex · GPT-6 · 2026-10-04 (optional recovery catalog wiring).
// Validator admission integration: Codex · GPT-6.1 · 2026-10-04.
// Optional sync integration: Codex · GPT-6.1 · 2026-10-05.
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
	toolkitCatalog, closeToolkitCatalog, err := configureToolkitCatalog(ctx, os.Getenv("CASEBIBLE_RECOVERY_DATABASE_URL_FILE"), registrations.ToolkitPreservation)
	if err != nil {
		return err
	}
	if closeToolkitCatalog != nil {
		defer closeToolkitCatalog()
	}
	registrations.ToolkitCatalog = toolkitCatalog
	registrations.AIWorkproductCatalog, err = configureAIWorkproductCatalog(ctx, toolkitCatalog, registrations.AIWorkproductPlacement)
	if err != nil {
		return err
	}
	toolkitWorking, closeToolkitWorking, err := ConfigureToolkitWorkingCatalog(ctx, os.Getenv("CASEBIBLE_WORKING_DATABASE_URL_FILE"), registrations.ToolkitPreservation.AllowedRoot)
	if err != nil {
		return err
	}
	if closeToolkitWorking != nil {
		defer closeToolkitWorking()
	}
	registrations.ToolkitWorkingCatalog = toolkitWorking
	toolkitValidation, err := configureToolkitValidation(ctx)
	if err != nil {
		return err
	}
	registrations.ToolkitValidation = toolkitValidation
	// Admit sync only after validator construction so pinned extraction/artifacts/signing are reused.
	// Inputs: explicit sync environment and configured validator; outputs: optional Activity group.
	// Effects: bounded configuration reads; no scheduling, source writes or live worker changes.
	// Byline: Codex · GPT-6.1 · 2026-10-05.
	toolkitSync, err := configureToolkitSync(ctx, toolkitValidation)
	if err != nil {
		return err
	}
	if toolkitSync != nil && cfg.TemporalTaskQueue != librarysync.TaskQueue {
		return errors.New("proffer worker: toolkit sync requires proffer-v1 task queue")
	}
	registrations.ToolkitSync = toolkitSync
	if toolkitSync != nil {
		backend, ok := toolkitSync.Service.Backend.(activities.ToolkitLibraryBindingBackend)
		if !ok {
			return errors.New("proffer worker: library binding backend unavailable")
		}
		group := activities.NewToolkitLibraryBindingActivities(registrations.ToolkitPreservation.AllowedRoot, registrations.ToolkitPreservation.Stores, backend)
		registrations.ToolkitBindings = &group
	}

	temporalWorker := worker.New(temporalClient, cfg.TemporalTaskQueue, workerOptions(cfg))
	RegisterAll(temporalWorker, registrations)
	RegisterExtraction(temporalWorker, registrations.Extraction)
	RegisterConversationExtraction(temporalWorker, registrations.Conversation)
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
	// Bind only the already-retained source version through the existing original opener.
	// Byline: Codex, 2026-10-06.
	integrityStore, err := platformpostgres.NewSourceIntegrityStore(pool, hashRepo)
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
	if err := flowRegistry.AdmitCaseConnected(repairPlan.Validate.Environment.Registry.CaseFlowNames()); err != nil {
		return Registrations{}, fmt.Errorf("repair flow scope classification: %w", err)
	}
	extraction, err := buildExtraction(pool)
	if err != nil {
		return Registrations{}, err
	}
	conversation, err := buildConversationActivities(pool, nil)
	if err != nil {
		return Registrations{}, err
	}
	var contextGraph *activities.ContextGraphActivities
	if strings.TrimSpace(os.Getenv("ANALYSIS_GRAPH_ROOT")) != "" {
		group, graphErr := activities.ContextGraphActivitiesFromEnv()
		if graphErr != nil {
			return Registrations{}, fmt.Errorf("proffer worker: configured analytical graph: %w", graphErr)
		}
		contextGraph = &group
	}
	contextSearch, err := buildContextSearch(pool, cfg.ContextSearch)
	if err != nil {
		return Registrations{}, err
	}
	firstPartyStore, err := platformpostgres.NewFirstPartyContextStore(pool)
	if err != nil {
		return Registrations{}, err
	}
	messageMatchStore, err := platformpostgres.NewMessageMatchStore(pool)
	if err != nil {
		return Registrations{}, err
	}
	contextSearch.Matches = messageMatchStore
	messageDedupeStore, err := platformpostgres.NewMessageDedupeStore(pool)
	if err != nil {
		return Registrations{}, err
	}
	callLogStore, err := platformpostgres.NewCallLogStore(pool)
	if err != nil {
		return Registrations{}, err
	}
	contactsActivities, err := buildContacts(context.Background(), pool, stores, cfg)
	if err != nil {
		return Registrations{}, err
	}
	return Registrations{
		// Optional worker-owned root: unset fails visibly when invoked; no source or DB writes.
		// Byline: Codex, 2026-10-04.
		ToolkitInventory:        activities.NewToolkitPackageInventoryActivities(strings.TrimSpace(os.Getenv("TOOLKIT_INVENTORY_ROOT"))),
		ToolkitPreservation:     activities.NewToolkitPackagePreservationActivities(strings.TrimSpace(os.Getenv("TOOLKIT_INVENTORY_ROOT")), objectStores),
		ToolkitContentPlacement: activities.NewToolkitContentPlacementActivities(strings.TrimSpace(os.Getenv("TOOLKIT_INVENTORY_ROOT")), objectStores),
		AIWorkproductPlacement:  activities.NewAIWorkproductPlacementActivities(cfg.DeriveScratchDir, objectStores),
		Contacts:                contactsActivities,
		ContextSearch:           contextSearch,
		FirstPartyContext:       activities.NewFirstPartyContextActivities(firstPartyStore),
		CallLog:                 activities.NewCallLogActivities(callLogStore),
		MessageMatch:            activities.NewMessageMatchActivities(firstPartyStore, messageMatchStore),
		MessageDedupe:           activities.NewMessageDedupeActivities(messageDedupeStore),
		RepairPlan:              repairPlan,
		Extraction:              extraction,
		Conversation:            conversation,
		ContextGraph:            contextGraph,
		Lifecycle:               activities.NewSourceLifecycleActivities(lifecycleRepo),
		FilesystemObservation:   activities.NewSourceObservationActivities(filesystemExtractor, nil, observationRepo),
		InventoryObservation:    activities.NewSourceObservationActivities(nil, memberEnumerator, observationRepo),
		EmbeddedObservation:     activities.NewSourceObservationActivities(embeddedExtractor, nil, observationRepo),
		N8N:                     platformtemporal.N8NActivities{Client: n8nClient},
		N8NFlows:                platformtemporal.FlowActivities{Client: n8nClient, Registry: flowRegistry},
		Hash:                    activities.NewHashActivities(hashRepo),
		Integrity:               activities.NewSourceIntegrityActivities(integrityStore),
		StructuredELT:           activities.NewStructuredELTActivities(structuredELTRepo, parserStore, handlerSelectionStore),
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
		// The automatic approval writes the same decision record Review does.
		// Byline: Claude Code · Opus 5.5 · 2026-10-02
		AutoApproval: activities.AutoApprovalActivity{Store: previewStore},
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
