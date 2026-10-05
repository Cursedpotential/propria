// Byline: Codex, GPT-6, 2026-10-04. Shared library validation uses the existing tracked worker queue.
package main

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"net/http"
	"os"
	"strings"
)

type toolkitValidationStarter struct {
	temporal  client.Client
	taskQueue string
}

// StartLibrarySync starts or joins a sealed outbox export using its stable operation identity.
// Inputs: operation UUID. Outputs: workflow/run IDs. Effects: Temporal dispatch only; database/B2 checks occur in tracked Activities.
// Choose for durable console outbox retries; running operations are joined and closed retries reconcile their saved intent.
// Byline: Codex · GPT-6 · 2026-10-05.
func (s toolkitValidationStarter) StartLibrarySync(ctx context.Context, operationID string) (string, string, error) {
	workflowID := "library-sync-write-" + operationID
	run, err := s.temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: workflowID, TaskQueue: s.taskQueue, WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE}, librarysync.WriteWorkflowName, librarysync.WriteInput{OperationID: operationID})
	if err != nil {
		var already *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &already) {
			return workflowID, already.RunId, nil
		}
		return "", "", err
	}
	return workflowID, run.GetRunID(), nil
}

// StartLibraryValidation starts or joins the exact proposal's validation workflow.
// Inputs: retained proposal identity. Outputs: workflow and run ids. Effects: idempotent Temporal start.
// Choose for shared toolkit service dispatch; retries rerun failed workflows only and never duplicate a successful run.
func (s toolkitValidationStarter) StartLibraryValidation(ctx context.Context, proposalID string) (string, string, error) {
	workflowID := "library-validation-" + strings.TrimPrefix(proposalID, "library_proposal:")
	input := struct {
		ProposalID string `json:"proposal_id"`
	}{ProposalID: proposalID}
	run, err := s.temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: workflowID, TaskQueue: s.taskQueue, WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY}, "ToolkitLibraryValidationWorkflow", input)
	if err != nil {
		var already *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &already) {
			return workflowID, already.RunId, nil
		}
		return "", "", err
	}
	return workflowID, run.GetRunID(), nil
}

// mountToolkitLibraryValidationRoutes adds retained-proposal dispatch without changing existing routes.
// Inputs: existing mux, Temporal client, queue and mounted token. Outputs: composed handler.
// Effects: none until a request arrives. Choose during starter composition; case contents remain in the shared database.
func mountToolkitLibraryValidationRoutes(existing http.Handler, c client.Client, queue, tokenFile string) (http.Handler, error) {
	if existing == nil || c == nil || strings.TrimSpace(queue) == "" {
		return nil, errors.New("toolkit validation routes require existing handler, client and queue")
	}
	if configured := strings.TrimSpace(os.Getenv("TOOLKIT_VALIDATION_TOKEN_FILE")); configured != "" {
		tokenFile = configured
	}
	handler, err := runtimeapi.NewToolkitLibraryValidationHandler(toolkitValidationStarter{temporal: c, taskQueue: queue}, tokenFile)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	if strings.TrimSpace(os.Getenv(librarysync.EnvBackendURL)) != "" {
		reader, err := librarysync.NewOriginalReaderFromEnv()
		if err != nil {
			return nil, err
		}
		syncHandler, err := runtimeapi.NewToolkitLibrarySyncHandler(toolkitValidationStarter{temporal: c, taskQueue: queue}, reader, os.Getenv(librarysync.EnvTokenFile))
		if err != nil {
			return nil, err
		}
		mux.Handle("/toolkit/library/files/", syncHandler)
		mux.Handle("/toolkit/library/sync", syncHandler)
	}
	mux.Handle("/toolkit/library/", handler)
	mux.Handle("/", existing)
	return mux, nil
}
