// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5 · 2026-09-21
//
// The four Activities a batch-by-folder import needs (owner 2026-09-20 23:52:
// "it gets batched by folder. Being careful not to overload any systems and
// process them one at a time ... batching is absolutely part of this"; build
// order step 4 in docs/decisions/2026-09-20-bulk-intake-owner-requirements.md).
//
// Each is one job, and none of them orchestrates: sequencing, the in-flight
// bound, skipping and failure accounting all belong to the batch workflow
// (AGENTS.md ATOMICITY rules 1 and 6).
//
//   - list_batch_folder_activity     one page of object keys under a prefix
//   - bind_import_operation_activity one preview binding for a started run
//   - read_import_operation_activity one run's lifecycle, by reference
//   - find_import_bindings_activity  existing bindings for one source_ref
//
// Preview bindings are what make a batch item indistinguishable from a
// hand-started one in Review: the HTTP start handler creates exactly this
// binding, with exactly these fields.

package activities

import (
	"context"
	"errors"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
)

const (
	ListBatchFolderActivityName     = "list_batch_folder_activity"
	BindImportOperationActivityName = "bind_import_operation_activity"
	ReadImportOperationActivityName = "read_import_operation_activity"
	FindImportBindingsActivityName  = "find_import_bindings_activity"
)

// maxBatchListingPage bounds one listing page so a batch never pulls a whole
// folder listing into memory or into Temporal history.
const maxBatchListingPage = 200

// ListBatchFolderRequest names one page of one folder.
type ListBatchFolderRequest struct {
	OperatingMode string `json:"operating_mode"`
	MatterID      string `json:"matter_id"`
	CourtCaseID   string `json:"court_case_id"`
	Scheme        string `json:"scheme"`
	Bucket        string `json:"bucket"`
	Prefix        string `json:"prefix"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int32  `json:"limit,omitempty"`
	// KeySuffix keeps only keys ending in it; empty keeps every key.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	KeySuffix string `json:"key_suffix,omitempty"`
}

// ListBatchFolderResult is one page: keys plus the cursor for the next call.
type ListBatchFolderResult struct {
	Keys       []string `json:"keys"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

// BindImportOperationRequest is the durable identity of one started run.
type BindImportOperationRequest struct {
	MatterID         string      `json:"matter_id"`
	CourtCaseID      string      `json:"court_case_id"`
	OperatingMode    string      `json:"operating_mode"`
	RequestID        string      `json:"request_id"`
	SourceRef        proffer.Ref `json:"source_ref"`
	WorkflowID       string      `json:"workflow_id"`
	RunID            string      `json:"run_id"`
	ParserOptionsRef proffer.Ref `json:"parser_options_ref"`
}

// BindImportOperationResult carries the opaque handle Review addresses.
type BindImportOperationResult struct {
	PreviewHandle string `json:"preview_handle"`
}

// ReadImportOperationRequest addresses one run by its durable workflow id.
type ReadImportOperationRequest struct {
	WorkflowID string `json:"workflow_id"`
}

// ReadImportOperationResult is the compact lifecycle a batch needs to decide
// whether an item still holds its in-flight slot.
type ReadImportOperationResult struct {
	Lifecycle string `json:"lifecycle"`
	Wait      string `json:"wait,omitempty"`
	Terminal  bool   `json:"terminal"`
	Reason    string `json:"reason,omitempty"`
	// Available is false when durable workflow state could not be read; the
	// caller must treat that as "unknown", never as "finished".
	Available bool `json:"available"`
	// SourceVersionRef is set once the run has registered its source, which
	// is when its matter becomes provable from durable state. Repair re-entry
	// binds a run to Review only after this. Byline: Claude Code · Opus 5.5 · 2026-09-25
	SourceVersionRef string `json:"source_version_ref,omitempty"`
}

// FindImportBindingsRequest asks whether this exact source was imported before.
type FindImportBindingsRequest struct {
	SourceRef proffer.Ref `json:"source_ref"`
	Limit     int         `json:"limit,omitempty"`
}

// ImportBindingRef is one prior run of the same source, by reference only.
type ImportBindingRef struct {
	PreviewHandle string `json:"preview_handle"`
	RequestID     string `json:"request_id"`
	WorkflowID    string `json:"workflow_id"`
}

// FindImportBindingsResult is never nil: an empty listing encodes as [].
type FindImportBindingsResult struct {
	Bindings []ImportBindingRef `json:"bindings"`
}

// BatchFolderLister returns one page of object keys under a prefix. It is a
// plain function rather than the acquisition package's interface because
// acquisition imports postgres, which imports this package: the seam keeps
// the dependency pointing one way. The worker adapts acquisition.S3Lister.
type BatchFolderLister func(ctx context.Context, scheme, bucket, prefix, cursor string, limit int32) (keys []string, nextCursor string, err error)

// ImportBindingStore is the durable preview-binding seam. The HTTP start
// handler writes through the same store, so a batch item and a hand-started
// one are the same kind of row.
type ImportBindingStore interface {
	Create(context.Context, previewmodel.Binding) (previewmodel.Binding, error)
	BindingsBySourceRef(ctx context.Context, sourceRef proffer.Ref, limit int) ([]previewmodel.Binding, error)
}

// ImportOperationReader reads one run's durable lifecycle.
type ImportOperationReader interface {
	Operation(ctx context.Context, workflowID string) (proffer.OperationState, error)
}

// BatchImportActivities implements all four. Every field is required.
type BatchImportActivities struct {
	Lister     BatchFolderLister
	Bindings   ImportBindingStore
	Operations ImportOperationReader
}

// ListBatchFolder returns one page of object keys under a prefix.
func (a BatchImportActivities) ListBatchFolder(ctx context.Context, req ListBatchFolderRequest) (ListBatchFolderResult, error) {
	if a.Lister == nil {
		return ListBatchFolderResult{}, errors.New("batch import: object lister is required")
	}
	if strings.TrimSpace(req.Scheme) == "" || strings.TrimSpace(req.Bucket) == "" || strings.TrimSpace(req.Prefix) == "" {
		return ListBatchFolderResult{}, stopRetryingPermanent(permanent(
			errors.New("batch import: a folder locator requires a scheme, bucket and prefix")))
	}
	limit := req.Limit
	if limit <= 0 || limit > maxBatchListingPage {
		limit = maxBatchListingPage
	}
	keys, nextCursor, err := a.Lister(ctx, req.Scheme, req.Bucket, req.Prefix, req.Cursor, limit)
	if err != nil {
		return ListBatchFolderResult{}, err
	}
	// A derivation publishes beside its source (<source>.derived/...), so a
	// folder that already holds one derived source lists its manifest, thread
	// chunks and attachment files too. Those are outputs of an earlier run,
	// never sources of this folder's batch: importing them made a run per
	// attachment (live 2026-10-02). Keys under a ".derived/" segment below the
	// batch prefix are dropped; a batch whose prefix IS a derived folder (the
	// threads batch) still lists its own files, because the segment is above
	// the prefix there. Byline: Claude Code · Opus 5.5 · 2026-10-02
	sources := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.Contains(strings.TrimPrefix(key, req.Prefix), ".derived/") {
			continue
		}
		if req.KeySuffix != "" && !strings.HasSuffix(key, req.KeySuffix) {
			continue
		}
		sources = append(sources, key)
	}
	return ListBatchFolderResult{Keys: sources, NextCursor: nextCursor}, nil
}

// BindImportOperation records the preview binding for one started run.
func (a BatchImportActivities) BindImportOperation(ctx context.Context, req BindImportOperationRequest) (BindImportOperationResult, error) {
	if a.Bindings == nil {
		return BindImportOperationResult{}, errors.New("batch import: preview binding store is required")
	}
	if strings.TrimSpace(req.RequestID) == "" || strings.TrimSpace(string(req.SourceRef)) == "" ||
		strings.TrimSpace(req.WorkflowID) == "" || strings.TrimSpace(req.RunID) == "" ||
		strings.TrimSpace(string(req.ParserOptionsRef)) == "" {
		return BindImportOperationResult{}, stopRetryingPermanent(permanent(
			errors.New("batch import: a preview binding requires request, source, workflow, run and parser option references")))
	}
	// Create is idempotent on request_id, so a retried Activity returns the
	// first binding instead of minting a second handle.
	binding, err := a.Bindings.Create(ctx, previewmodel.Binding{
		OperatingMode: req.OperatingMode, RequestID: req.RequestID, SourceRef: req.SourceRef,
		WorkflowID: req.WorkflowID, RunID: req.RunID, ParserOptionsRef: req.ParserOptionsRef,
	})
	if err != nil {
		return BindImportOperationResult{}, err
	}
	return BindImportOperationResult{PreviewHandle: binding.Handle}, nil
}

// ReadImportOperation reports one run's lifecycle. An unreadable run is
// reported as unavailable rather than as an error: a batch must keep moving,
// and treating "unknown" as "finished" would release the slot too early.
func (a BatchImportActivities) ReadImportOperation(ctx context.Context, req ReadImportOperationRequest) (ReadImportOperationResult, error) {
	if a.Operations == nil {
		return ReadImportOperationResult{}, errors.New("batch import: operation reader is required")
	}
	if strings.TrimSpace(req.WorkflowID) == "" {
		return ReadImportOperationResult{}, stopRetryingPermanent(permanent(
			errors.New("batch import: reading an operation requires its workflow reference")))
	}
	state, err := a.Operations.Operation(ctx, req.WorkflowID)
	if err != nil {
		return ReadImportOperationResult{Lifecycle: string(proffer.OperationUnavailable)}, nil
	}
	return ReadImportOperationResult{
		Lifecycle: string(state.Lifecycle), Wait: string(state.Wait),
		Terminal: state.Terminal, Reason: state.Reason, Available: true,
		SourceVersionRef: string(state.SourceVersionRef),
	}, nil
}

// FindImportBindings lists prior runs of the same source so a re-run of the
// same batch can skip what already finished.
func (a BatchImportActivities) FindImportBindings(ctx context.Context, req FindImportBindingsRequest) (FindImportBindingsResult, error) {
	if a.Bindings == nil {
		return FindImportBindingsResult{}, errors.New("batch import: preview binding store is required")
	}
	if strings.TrimSpace(string(req.SourceRef)) == "" {
		return FindImportBindingsResult{}, stopRetryingPermanent(permanent(
			errors.New("batch import: finding prior imports requires a source reference")))
	}
	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	bindings, err := a.Bindings.BindingsBySourceRef(ctx, req.SourceRef, limit)
	if err != nil {
		return FindImportBindingsResult{}, err
	}
	// A nil slice marshals to null and the BFF rejects it.
	result := FindImportBindingsResult{Bindings: make([]ImportBindingRef, 0, len(bindings))}
	for _, binding := range bindings {
		result.Bindings = append(result.Bindings, ImportBindingRef{
			PreviewHandle: binding.Handle, RequestID: binding.RequestID, WorkflowID: binding.WorkflowID,
		})
	}
	return result, nil
}
