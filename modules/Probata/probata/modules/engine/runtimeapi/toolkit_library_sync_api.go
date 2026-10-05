// Byline: Codex · GPT-6 · 2026-10-05. Existing starter transport for durable sync and pinned originals.
package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"regexp"

	"github.com/Cursedpotential/probata/engine/extraction/librarysync"
)

// ToolkitLibrarySyncStarter starts or joins an immutable durable-outbox operation.
// Inputs: exact operation UUID. Outputs: Temporal workflow/run identities. Effects: idempotent workflow dispatch.
// Choose after the database snapshot is sealed; no editable body or provider coordinates are accepted.
type ToolkitLibrarySyncStarter interface {
	StartLibrarySync(context.Context, string) (string, string, error)
}

var toolkitSyncOperationPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// NewToolkitLibrarySyncHandler exposes private dispatch and original reads on the existing starter.
// Inputs: workflow starter, read-only original reader, dedicated literal mounted credential and private service CIDRs.
// Outputs: authenticated HTTP handler or visible startup error. Effects: bounded file read at construction; requests dispatch/read only.
// Choose beside validation routes; caller proxies must authorize the human before supplying the private service credential.
func NewToolkitLibrarySyncHandler(starter ToolkitLibrarySyncStarter, reader librarysync.OriginalReader, tokenFile string) (http.Handler, error) {
	if starter == nil || reader == nil {
		return nil, errors.New("toolkit sync requires workflow starter and original reader")
	}
	if _, err := loadServiceToken(tokenFile); err != nil {
		return nil, err
	}
	networks, err := toolkitValidationServiceNetworks(os.Getenv("TOOLKIT_LIBRARY_SYNC_SERVICE_CIDRS"))
	if err != nil {
		return nil, err
	}
	original, err := librarysync.NewOriginalHandler(reader, func(*http.Request) bool { return true })
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/toolkit/library/files/", toolkitValidationServiceAuth(tokenFile, networks, original.ServeHTTP))
	mux.HandleFunc("POST /toolkit/library/sync", toolkitValidationServiceAuth(tokenFile, networks, func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OperationID string `json:"operation_id"`
		}
		if err := decodePreviewJSON(w, r, &input); err != nil {
			previewError(w, http.StatusBadRequest, errors.New("invalid sync operation request"))
			return
		}
		if !toolkitSyncOperationPattern.MatchString(input.OperationID) {
			previewError(w, http.StatusUnprocessableEntity, errors.New("exact sync operation UUID required"))
			return
		}
		workflowID, runID, err := starter.StartLibrarySync(r.Context(), input.OperationID)
		if err != nil {
			previewError(w, http.StatusServiceUnavailable, errors.New("sync workflow could not start; durable operation retained"))
			return
		}
		previewJSON(w, http.StatusAccepted, map[string]string{"workflow_id": workflowID, "run_id": runID})
	}))
	return mux, nil
}
