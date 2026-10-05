// Byline: Codex · GPT-6 · 2026-10-05. Tracked, bounded linkage of approved working files; metadata only.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitLibraryBindingWorkflowName = "ToolkitLibraryBindingWorkflow"
	ToolkitLibraryBindingActivityName = "toolkit_library_binding_import_activity"
)

// ToolkitLibraryBindingInput pins approved placement evidence and a stable observation time for replay.
// Inputs: the catalog's exact receipt/manifest/map coordinates plus RFC3339 observed_at.
// Outputs: bounded binding counts and evidence hashes. Effects: none; choose after placement/catalog readback.
type ToolkitLibraryBindingInput struct {
	ToolkitWorkingCatalogInput
	ObservedAt string `json:"observed_at"`
}

// ToolkitLibraryBindingBackend exposes only the parent's dedicated bindings import route.
// Inputs: at most 2MiB of typed metadata JSON. Outputs: at most 4KiB of response JSON.
// Effects: POST /api/internal/library-sync/bindings/import via librarysync.NewHTTPBackend's fixed origin/token.
// Choose for parent-owned HTTPBackend.ImportBindings wiring; never accept a URL or token from workflow inputs.
type ToolkitLibraryBindingBackend interface {
	ImportBindings(context.Context, json.RawMessage) (json.RawMessage, error)
}

// toolkitBindingVersionReader exposes exact-version reads without any storage mutation capability.
// Inputs: fixed bucket/key/version. Outputs: original byte stream. Effects: GET only; choose for PDF verification.
type toolkitBindingVersionReader interface {
	OpenVersion(context.Context, string, string, string) (io.ReadCloser, error)
}

// ToolkitLibraryBindingActivities connects mounted metadata, existing storage and the dedicated backend.
// Inputs: parent-owned seams. Outputs: one separately tracked import Activity.
// Effects: none at construction; choose beside catalog registration, never substitute for its SQL readback.
type ToolkitLibraryBindingActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Backend     ToolkitLibraryBindingBackend
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewToolkitLibraryBindingActivities constructs binding import without creating clients or applying schema.
// Inputs: mounted metadata root, worker store resolver and fixed-route backend. Outputs: Activity group.
// Effects: heartbeats during execution; choose for parent worker registration with existing library-sync configuration.
func NewToolkitLibraryBindingActivities(root string, stores func(string) (smsthreads.ObjectStore, error), backend ToolkitLibraryBindingBackend) ToolkitLibraryBindingActivities {
	return ToolkitLibraryBindingActivities{AllowedRoot: root, Stores: stores, Backend: backend, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// toolkitBindingPointer describes an immutable retained version with verified reader media type.
// Inputs: approved placement identity and observation time. Outputs: metadata pointer. Effects: none.
// Choose instead of changing original B2 Content-Type, including retained application/octet-stream PDFs.
type toolkitBindingPointer struct {
	VersionID   string `json:"version_id"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ObservedAt  string `json:"observed_at"`
}

// toolkitBindingFile retains one physical occurrence and optional qualified record identities.
// Inputs: verified catalog row. Outputs: import metadata. Effects: none; choose over copying application record bodies.
type toolkitBindingFile struct {
	Key         string                `json:"key"`
	Pointer     toolkitBindingPointer `json:"pointer"`
	ReferenceID string                `json:"reference_id,omitempty"`
	SourceID    string                `json:"source_id,omitempty"`
}

// toolkitBindingPayload restricts HTTP import to the approved three evidence pins and complete identity sets.
// Inputs: authenticated metadata. Outputs: <=2MiB JSON. Effects: none; never include this array in Temporal history.
type toolkitBindingPayload struct {
	ReceiptSHA256      string               `json:"receipt_sha256"`
	ManifestSHA256     string               `json:"manifest_sha256"`
	ReferenceMapSHA256 string               `json:"reference_map_sha256"`
	Files              []toolkitBindingFile `json:"files"`
	SourceIDs          []string             `json:"source_ids"`
}

// toolkitBindingResponse decodes only the dedicated seed route's bounded acknowledgement.
// Inputs: backend JSON. Outputs: counts and returned evidence pins. Effects: none; choose before accepting import success.
type toolkitBindingResponse struct {
	Status               string `json:"status"`
	OriginalBindings     int    `json:"original_bindings"`
	SourceExportBindings int    `json:"source_export_bindings"`
	OriginalAliases      int    `json:"original_aliases"`
	ReceiptSHA256        string `json:"receipt_sha256"`
	ReferenceMapSHA256   string `json:"reference_map_sha256"`
}

// ToolkitLibraryBindingResult reports imported linkage separately from catalog and projection freshness.
// Inputs: verified backend acknowledgement and sent metadata digest. Outputs: bounded counts/hashes only.
// Effects: none; choose for workflow history, not as proof of SQL catalog or whole-bucket refresh.
type ToolkitLibraryBindingResult struct {
	OperationID          string `json:"operation_id"`
	OriginalBindings     int    `json:"original_bindings"`
	SourceExportBindings int    `json:"source_export_bindings"`
	OriginalAliases      int    `json:"original_aliases"`
	Files                int    `json:"files"`
	VerifiedPDFs         int    `json:"verified_pdfs"`
	MetadataSHA256       string `json:"metadata_sha256"`
	ReceiptSHA256        string `json:"receipt_sha256"`
	ManifestSHA256       string `json:"manifest_sha256"`
	ReferenceMapSHA256   string `json:"reference_map_sha256"`
	BindingFreshness     string `json:"binding_freshness"`
	CatalogFreshness     string `json:"catalog_freshness"`
	WholeBucketFreshness string `json:"whole_bucket_freshness"`
	ProjectionFreshness  string `json:"projection_freshness"`
}

// validateToolkitBindingInput rejects changed evidence pins and unstable observation timestamps before I/O.
// Inputs: workflow/Activity request. Outputs: admission error or nil. Effects: none; choose at both boundaries.
func validateToolkitBindingInput(in ToolkitLibraryBindingInput) error {
	if err := validateToolkitWorkingInput(in.ToolkitWorkingCatalogInput); err != nil {
		return err
	}
	if len(in.ObservedAt) > 40 {
		return errors.New("binding observed_at exceeds bound")
	}
	if _, err := time.Parse(time.RFC3339Nano, in.ObservedAt); err != nil {
		return errors.New("binding requires stable RFC3339 observed_at")
	}
	return nil
}

// toolkitBindingQualified admits the same bounded qualified record identity syntax as the seed route.
// Inputs: record ID and fixed family. Outputs: validity. Effects: none; choose before assigning any aliases.
func toolkitBindingQualified(id, family string) bool {
	return regexp.MustCompile(`^` + family + `:[^\s:]{1,200}$`).MatchString(id)
}

// toolkitBindingMedia maps approved original extensions to explicit reader media types.
// Inputs: physical object key. Outputs: deterministic type or rejection. Effects: none; PDF bytes require a separate verification.
// Choose over OS MIME registries and B2 octet-stream metadata, which cannot establish an original's format.
func toolkitBindingMedia(key string) (string, error) {
	switch strings.ToLower(path.Ext(key)) {
	case ".pdf":
		return "application/pdf", nil
	case ".md":
		return "text/markdown", nil
	case ".json":
		return "application/json", nil
	case ".html":
		return "text/html", nil
	case ".yaml", ".yml":
		return "application/yaml", nil
	case ".csv":
		return "text/csv", nil
	case ".txt", ".log":
		return "text/plain", nil
	case ".py":
		return "text/x-python", nil
	case ".sh":
		return "application/x-sh", nil
	case ".pyc", "":
		return "application/octet-stream", nil
	default:
		return "", errors.New("unsupported approved binding extension")
	}
}

// assembleToolkitBindings joins qualified records and the complete source ledger without retaining source bodies.
// Inputs: authenticated 443-row catalog, raw ledger IDs and stable observation time. Outputs: sorted bounded HTTP metadata.
// Effects: none; choose only after the shared catalog loader authenticates every placement/manifest/map join.
func assembleToolkitBindings(batch ToolkitWorkingCatalogBatch, rawSourceIDs []string, observedAt string) (toolkitBindingPayload, error) {
	p := toolkitBindingPayload{ReceiptSHA256: batch.Request.ReceiptSHA256, ManifestSHA256: batch.Request.ManifestSHA256, ReferenceMapSHA256: batch.Request.ReferenceMapSHA256}
	if err := validateToolkitBindingInput(ToolkitLibraryBindingInput{batch.Request, observedAt}); err != nil {
		return p, err
	}
	if _, _, err := ToolkitWorkingCatalogCanonical(batch); err != nil {
		return p, err
	}
	sources, refs, aliases := map[string]bool{}, map[string]bool{}, map[string]bool{}
	if len(rawSourceIDs) != 193 {
		return p, errors.New("binding requires complete 193-source ledger")
	}
	for _, raw := range rawSourceIDs {
		id := "source:" + raw
		if !toolkitBindingQualified(id, "source") || sources[id] {
			return p, errors.New("source ledger identity invalid or duplicate")
		}
		sources[id] = true
		p.SourceIDs = append(p.SourceIDs, id)
	}
	originals, pdfs := 0, 0
	for _, row := range batch.Objects {
		obj := row.Placement
		media, err := toolkitBindingMedia(obj.ObjectKey)
		if err != nil || obj.Bytes > 20<<20 {
			return p, errors.New("binding pointer media/size invalid")
		}
		f := toolkitBindingFile{Key: obj.ObjectKey, Pointer: toolkitBindingPointer{obj.VersionID, obj.SHA256, obj.Bytes, media, observedAt}}
		if len(row.ReferenceRecords) > 1 || len(row.SourceRecords) > 1 {
			return p, errors.New("ambiguous binding record identity")
		}
		if len(row.ReferenceRecords) == 1 {
			f.ReferenceID = row.ReferenceRecords[0].ID
			if !toolkitBindingQualified(f.ReferenceID, "reference") || refs[f.ReferenceID] {
				return p, errors.New("reference identity invalid or duplicate")
			}
			refs[f.ReferenceID] = true
		}
		if len(row.SourceRecords) == 1 {
			f.SourceID = row.SourceRecords[0].ID
			if !toolkitBindingQualified(f.SourceID, "source") || !sources[f.SourceID] || aliases[f.SourceID] {
				return p, errors.New("source alias invalid, duplicate or absent from ledger")
			}
			aliases[f.SourceID] = true
		}
		if f.ReferenceID != "" || f.SourceID != "" {
			originals++
		}
		if media == "application/pdf" {
			pdfs++
		}
		p.Files = append(p.Files, f)
	}
	if len(refs) != 321 || len(aliases) != 119 || originals != 323 || pdfs != 32 {
		return p, errors.New("binding identity/media cardinalities differ from approved map")
	}
	sort.Strings(p.SourceIDs)
	sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Key < p.Files[j].Key })
	raw, err := json.Marshal(p)
	if err != nil || len(raw) > toolkitWorkingCatalogLimit {
		return p, errors.New("binding metadata exceeds 2MiB")
	}
	return p, nil
}

// loadToolkitBindings reuses exact catalog authentication and separately reads the pinned full source ledger.
// Inputs: approved mounted evidence. Outputs: metadata payload. Effects: bounded private metadata reads only.
// Choose over selecting the 119 file-linked sources, which would omit 74 current citation-only source records.
func (a ToolkitLibraryBindingActivities) loadToolkitBindings(ctx context.Context, in ToolkitLibraryBindingInput) (toolkitBindingPayload, error) {
	if err := validateToolkitBindingInput(in); err != nil {
		return toolkitBindingPayload{}, err
	}
	loader := ToolkitWorkingCatalogActivities{AllowedRoot: a.AllowedRoot, Heartbeat: a.Heartbeat}
	batch, err := loader.loadWorkingMetadata(ctx, in.ToolkitWorkingCatalogInput)
	if err != nil {
		return toolkitBindingPayload{}, err
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return toolkitBindingPayload{}, err
	}
	file, err := resolveFileRef(in.ReferenceMapRef, root, false)
	if err != nil {
		return toolkitBindingPayload{}, err
	}
	raw, sha, err := toolkitReadBoundedJSON(file, toolkitWorkingCatalogLimit, func() error { return loader.workingPulse(ctx, "authenticate-source-ledger") })
	if err != nil {
		return toolkitBindingPayload{}, err
	}
	if sha != ToolkitWorkingCatalogReferenceMapSHA256 {
		return toolkitBindingPayload{}, errors.New("binding source ledger SHA-256 mismatch")
	}
	var mapping struct {
		Ledger []struct {
			ID string `json:"id"`
		} `json:"complete_current_source_ledger"`
	}
	if json.Unmarshal(raw, &mapping) != nil {
		return toolkitBindingPayload{}, errors.New("binding source ledger malformed")
	}
	ids := make([]string, 0, len(mapping.Ledger))
	for _, row := range mapping.Ledger {
		ids = append(ids, row.ID)
	}
	return assembleToolkitBindings(batch, ids, in.ObservedAt)
}

// verifyToolkitBindingPDF checks exact retained bytes, SHA-256, size and PDF signature without changing originals.
// Inputs: pinned PDF pointer and version reader. Outputs: explicit verification error or nil.
// Effects: bounded GET/stream hash and progress only; choose before asserting application/pdf in a reader pointer.
func verifyToolkitBindingPDF(ctx context.Context, store toolkitBindingVersionReader, file toolkitBindingFile, pulse func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r, err := store.OpenVersion(ctx, "salem-data", file.Key, file.Pointer.VersionID)
	if err != nil {
		return errors.New("binding PDF retained version unavailable")
	}
	defer r.Close()
	header := make([]byte, 5)
	if _, err = io.ReadFull(r, header); err != nil || !bytes.Equal(header, []byte("%PDF-")) {
		return errors.New("binding PDF signature mismatch")
	}
	sha, n, err := streamToolkitDigest(io.MultiReader(bytes.NewReader(header), r), file.Pointer.Size, pulse)
	if err != nil {
		return err
	}
	if sha != file.Pointer.SHA256 || n != file.Pointer.Size {
		return errors.New("binding PDF exact-version SHA/size mismatch")
	}
	return ctx.Err()
}

// importToolkitBindings verifies all PDF versions before making one bounded dedicated import request.
// Inputs: authenticated typed payload. Outputs: checked bounded acknowledgement. Effects: 32 exact-version GETs and one metadata POST.
// Choose inside the tracked Activity; never run a desktop seed or broaden the backend's origin/credential scope.
func (a ToolkitLibraryBindingActivities) importToolkitBindings(ctx context.Context, in ToolkitLibraryBindingInput, payload toolkitBindingPayload) (ToolkitLibraryBindingResult, error) {
	result := ToolkitLibraryBindingResult{}
	if a.Backend == nil || a.Stores == nil {
		return result, errors.New("binding backend and existing B2 resolver required")
	}
	pulse := func() error {
		return (ToolkitWorkingCatalogActivities{Heartbeat: a.Heartbeat}).workingPulse(ctx, "verify-binding-pdf-versions")
	}
	if err := pulse(); err != nil {
		return result, err
	}
	base, err := a.Stores("b2")
	if err != nil {
		return result, errors.New("binding B2 resolver failed")
	}
	if s, ok := base.(smsthreads.S3Store); ok {
		base = toolkitPlacementS3{s}
	}
	if s, ok := base.(*smsthreads.S3Store); ok && s != nil {
		base = toolkitPlacementS3{*s}
	}
	store, ok := base.(toolkitBindingVersionReader)
	if !ok {
		return result, errors.New("binding B2 adapter lacks exact-version reads")
	}
	pdfs := 0
	for _, file := range payload.Files {
		if file.Pointer.ContentType == "application/pdf" {
			if err := verifyToolkitBindingPDF(ctx, store, file, pulse); err != nil {
				return result, err
			}
			pdfs++
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > toolkitWorkingCatalogLimit || pdfs != 32 {
		return result, errors.New("binding payload/media budget mismatch")
	}
	if err := (ToolkitWorkingCatalogActivities{Heartbeat: a.Heartbeat}).workingPulse(ctx, "import-library-bindings"); err != nil {
		return result, err
	}
	reply, err := a.Backend.ImportBindings(ctx, json.RawMessage(raw))
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		return result, errors.New("dedicated binding backend import failed")
	}
	var ack toolkitBindingResponse
	if len(reply) > 4096 || decodeToolkitCatalogJSON(reply, &ack) != nil || ack.Status != "linked" || ack.OriginalBindings != 323 || ack.SourceExportBindings != 193 || ack.OriginalAliases != 119 || ack.ReceiptSHA256 != in.ReceiptSHA256 || ack.ReferenceMapSHA256 != in.ReferenceMapSHA256 {
		return result, errors.New("binding import acknowledgement count/pin mismatch")
	}
	return ToolkitLibraryBindingResult{OperationID: in.OperationID, OriginalBindings: ack.OriginalBindings, SourceExportBindings: ack.SourceExportBindings, OriginalAliases: ack.OriginalAliases, Files: len(payload.Files), VerifiedPDFs: pdfs, MetadataSHA256: digestBytes(raw), ReceiptSHA256: in.ReceiptSHA256, ManifestSHA256: in.ManifestSHA256, ReferenceMapSHA256: in.ReferenceMapSHA256, BindingFreshness: "linked", CatalogFreshness: "not_checked", WholeBucketFreshness: "unchanged", ProjectionFreshness: "not_refreshed"}, nil
}

// ImportToolkitLibraryBindings authenticates all placement evidence and imports only working-file linkage metadata.
// Inputs: pinned mounted receipt/manifest/map and stable observation time. Outputs: bounded counts/hashes/freshness.
// Effects: exact PDF version verification plus one dedicated backend metadata POST; no DB, B2 PUT, body export or generation writes.
// Choose after placement/catalog readback; parent owns backend migration, configuration, registration and deployment.
func (a ToolkitLibraryBindingActivities) ImportToolkitLibraryBindings(ctx context.Context, in ToolkitLibraryBindingInput) (ToolkitLibraryBindingResult, error) {
	payload, err := a.loadToolkitBindings(ctx, in)
	if err != nil {
		return ToolkitLibraryBindingResult{}, err
	}
	return a.importToolkitBindings(ctx, in, payload)
}

// ToolkitLibraryBindingWorkflow schedules approved linkage import as one tracked Temporal Activity.
// Inputs: bounded evidence coordinates and stable observation time. Outputs: checked import summary only.
// Effects: Activity scheduling on the parent's queue; choose instead of ad-hoc seeding or passing metadata arrays through history.
func ToolkitLibraryBindingWorkflow(ctx workflow.Context, in ToolkitLibraryBindingInput) (ToolkitLibraryBindingResult, error) {
	if err := validateToolkitBindingInput(in); err != nil {
		return ToolkitLibraryBindingResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitLibraryBindingInvalid", err)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Minute, ScheduleToCloseTimeout: 30 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}})
	var result ToolkitLibraryBindingResult
	err := workflow.ExecuteActivity(ctx, ToolkitLibraryBindingActivityName, in).Get(ctx, &result)
	return result, err
}
