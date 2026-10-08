// Command proffer-starter exposes the small authenticated HTTP
// surface n8n's start/decision/preview workflows call, since n8n has no
// native Temporal client: start a ProfferWorkflow run, signal a
// human preview decision into a held run, and read back its preview state.
//
// It holds no parsing, persistence, or Activity logic of its own, and no
// in-process state shared with cmd/worker — Decide/Preview go through the
// Temporal server as a real Signal/Query against the workflow's own durable
// history, so this binary can run as a separate process (or many replicas)
// from the worker. See the engine/temporal package doc comment.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"

	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	platformtemporal "github.com/Cursedpotential/probata/engine/temporal"
	"github.com/Cursedpotential/probata/engine/tsnetlisten"
)

const (
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
	gracefulStopTimeout = 20 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("universal import starter stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := platformtemporal.LoadConfig()
	if err != nil {
		return err
	}

	// The RAW value, not LoadConfig's default: with the tsnet listener on, a
	// configured host bind means this service is not tailnet-only and must not
	// start (tsnetlisten.GuardExclusiveBind).
	tsnetEnabled := tsnetlisten.Enabled()
	if err := tsnetlisten.GuardExclusiveBind(tsnetEnabled, os.Getenv("REFERENCE_STARTER_ADDR"), false); err != nil {
		return err
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		return fmt.Errorf("dial temporal client: %w", err)
	}
	defer c.Close()
	databaseURL, err := platformDatabaseURL()
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return errors.New("configure platform preview database: invalid configuration")
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		return errors.New("connect platform preview database: unavailable")
	}
	if err := platformpostgres.ProbeProfferSchema(context.Background(), pool); err != nil {
		return err
	}

	starter, err := platformtemporal.NewWorkflowStarter(c, cfg.TemporalTaskQueue)
	if err != nil {
		return err
	}
	handler, err := platformtemporal.NewStarterHTTPHandler(starter)
	if err != nil {
		return err
	}
	uploadIngress, err := newUploadIngress()
	if err != nil {
		return err
	}
	routes, err := starterRoutes(handler.Routes(), uploadIngress)
	if err != nil {
		return err
	}
	previewStore, err := platformpostgres.NewProfferPreviewStore(pool, nil)
	if err != nil {
		return err
	}
	repairStore, err := platformpostgres.NewRepairActivityStore(pool, []string{os.Getenv("SOURCE_OBJECT_DIR")})
	if err != nil {
		return err
	}
	retainedOpener, err := runtimeapi.NewRetainedObjectOpener(pool)
	if err != nil {
		return err
	}
	handlerSelectionStore, err := platformpostgres.NewHandlerSelectionStore(pool, retainedOpener)
	if err != nil {
		return err
	}
	cursorKey, err := previewCursorKey()
	if err != nil {
		return err
	}
	serviceTokenFile, err := previewServiceTokenFile()
	if err != nil {
		return err
	}
	sourceContextStore, err := platformpostgres.NewSourceContextStore(pool)
	if err != nil {
		return err
	}
	previewHandler, err := runtimeapi.NewPreviewHTTPHandler(starter, previewStore, repairStore, handlerSelectionStore, cursorKey, serviceTokenFile, sourceContextStore)
	if err != nil {
		return err
	}
	// Batch-by-folder intake shares this service's Temporal client and task
	// queue. Byline: Claude Code · Opus 5 · 2026-09-21
	batchStarter, err := platformtemporal.NewBatchStarter(c, cfg.TemporalTaskQueue)
	if err != nil {
		return err
	}
	if err := previewHandler.UseBatchWorkflow(batchStarter); err != nil {
		return err
	}
	// Repair workflow builder: tools, propose, validate, run, runs/{id}.
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	repairService, err := newRepairPlanService(pool, c, cfg.TemporalTaskQueue)
	if err != nil {
		return err
	}
	if err := previewHandler.UseRepairPlans(repairService); err != nil {
		return err
	}
	routes, err = mountPreviewRoutes(routes, previewHandler.Routes())
	if err != nil {
		return err
	}
	// Review overlays: the metadata screen with owner corrections, and context
	// review with the hindsight-only foreshadowing flag.
	// Byline: Claude Code · Opus 5.5 · 2026-09-26
	metadataRoutes, contextReviewRoutes, err := reviewOverlayHandlers(pool, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountReviewOverlayRoutes(routes, metadataRoutes, contextReviewRoutes); err != nil {
		return err
	}
	sourceContextHandler, err := runtimeapi.NewSourceContextHTTPHandler(sourceContextStore, serviceTokenFile)
	if err != nil {
		return err
	}
	routes, err = mountSourceContextRoutes(routes, sourceContextHandler.Routes())
	if err != nil {
		return err
	}
	// Entity/event extraction: propose -> correct -> commit.
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	extractionRoutes, err := entityExtractionHandler(pool, c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountEntityExtractionRoutes(routes, extractionRoutes); err != nil {
		return err
	}
	// Conversation extraction and the Surreal send (Workbench: Extract and Send to Surreal).
	// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	conversationRoutes, err := conversationExtractionHandler(c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountConversationExtractionRoutes(routes, conversationRoutes); err != nil {
		return err
	}
	// Source-pinned atomic tool actions share this Temporal client and queue.
	// Inputs: existing service token and actor-bound action routes. Output: mounted API.
	// Effects: route registration only; choose for operator-started tool actions.
	atomicToolRoutes, err := atomicToolHandler(c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountAtomicToolRoutes(routes, atomicToolRoutes); err != nil {
		return err
	}
	// Approved context reads share the starter client, queue and service token.
	// Inputs: actor-bound query and status routes. Output: mounted API.
	// Effects: route registration only; choose for revision-pinned graph queries.
	approvedQueryRoutes, err := approvedGraphQueryHandler(c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountApprovedGraphQueryRoutes(routes, approvedQueryRoutes); err != nil {
		return err
	}
	// Projection choices share the existing analytical identity and service token.
	// Inputs: guarded listing handler; output: mounted exact GET route.
	// Effects: configuration reads/route registration; choose before query selection.
	projectionRoutes, err := approvedGraphProjectionsHandler(serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountApprovedGraphProjectionsRoutes(routes, projectionRoutes); err != nil {
		return err
	}
	// Context-first intake reuses this Go starter's Temporal client, queue and service token.
	// Inputs: native source pointer and actor-bound polling. Output: mounted API.
	// Effects: route registration only; choose for context-v1 before legacy preview gates.
	contextRoutes, err := contextSourceHandler(c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountContextSourceRoutes(routes, contextRoutes); err != nil {
		return err
	}
	// Case identity: the Workbench Case page reads and edits registry, the one
	// identity store. Byline: Claude Code · Opus 5.5 · 2026-10-01
	caseIdentityRoutes, err := caseIdentityHandler(pool, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountCaseIdentityRoutes(routes, caseIdentityRoutes); err != nil {
		return err
	}
	legalContextRoutes, err := legalContextHandler(pool, serviceTokenFile)
	if err != nil {
		return err
	}
	if routes, err = mountLegalContextRoutes(routes, legalContextRoutes); err != nil {
		return err
	}

	routes, err = mountToolkitLibraryValidationRoutes(routes, c, cfg.TemporalTaskQueue, serviceTokenFile)
	if err != nil {
		return err
	}
	// Tailnet-only listener (owner directive 2026-09-07; D-134). With the
	// rollout flag on, the starter's only socket belongs to its own Tailscale
	// identity: no host bind, no published docker port, no Traefik router. The
	// guard above already refused to start if a host bind was also configured.
	listener, closeTsnet, address, err := starterListener(tsnetEnabled, cfg.StarterAddr)
	if err != nil {
		return err
	}
	defer closeTsnet()

	server := &http.Server{
		Handler:           routes,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErrors := make(chan error, 1)
	go func() {
		slog.Info("proffer starter listening",
			"address", address, "tsnet", tsnetEnabled, "task_queue", cfg.TemporalTaskQueue)
		serveErrors <- server.Serve(listener)
	}()
	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownContext.Done():
		stopContext, cancel := context.WithTimeout(context.Background(), gracefulStopTimeout)
		defer cancel()
		return server.Shutdown(stopContext)
	}
}

// starterListener returns the one socket this process serves on.
//
// tsnet mode: the service's own Tailscale identity (svc:proffer-starter by
// default), so which host the starter lands on stops mattering — the same
// decoupling D-132/D-134 built for the tool gateway. Legacy mode: the
// REFERENCE_STARTER_ADDR bind the deployed app uses today.
//
// Byline: Claude Code subagent · Opus 5 · 2026-09-07.
func starterListener(tsnetEnabled bool, hostAddr string) (net.Listener, func(), string, error) {
	if !tsnetEnabled {
		listener, err := net.Listen("tcp", hostAddr)
		if err != nil {
			return nil, nil, "", err
		}
		return listener, func() {}, hostAddr, nil
	}
	listener, server, err := tsnetlisten.ListenWith(
		context.Background(), tsnetlisten.FromEnv(tsnetlisten.ProfferStarter))
	if err != nil {
		return nil, nil, "", err
	}
	return listener, func() { _ = server.Close() }, tsnetlisten.Describe(listener), nil
}
