// Byline: Codex · GPT-6 · 2026-10-05. Additive working-file metadata; no source bodies or inventory generations.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitWorkingCatalogWorkflowName         = "ToolkitWorkingCatalogWorkflow"
	ToolkitWorkingCatalogRegisterActivityName = "toolkit_working_catalog_register_activity"
	ToolkitWorkingCatalogReadbackActivityName = "toolkit_working_catalog_readback_activity"
	ToolkitWorkingCatalogSchema               = "toolkit-working-catalog/v1"
	ToolkitWorkingCatalogSource               = "family-court-library"
	ToolkitWorkingCatalogExpectedObjects      = 443
	ToolkitWorkingCatalogReceiptSHA256        = "244061ffa2d62568938de8b8ebd0792fa924a94c7aee3b5c4d868e3c47cf9b84"
	ToolkitWorkingCatalogManifestSHA256       = "3f5a219804149b1e3175ffe048887723c2de233789a50860b7cfbf4d781f1ce4"
	ToolkitWorkingCatalogReferenceMapSHA256   = "6801dcf9fc4d11d812b000180066f941b167471a1dd7e5a2bbdd3010d71232e3"
	toolkitWorkingCatalogLimit                = 2 << 20
)

// ToolkitWorkingCatalogInput pins the owner-approved receipt, complete manifest and private source-unit map.
// Inputs: bounded operation ID, mounted references, exact SHA-256 pins and 443 expected objects.
// Outputs: registration/readback summaries. Effects: none; choose after permanent placement, not recovery registration.
type ToolkitWorkingCatalogInput struct {
	OperationID        string      `json:"operation_id"`
	ReceiptRef         proffer.Ref `json:"receipt_ref"`
	ReceiptSHA256      string      `json:"receipt_sha256"`
	ManifestRef        proffer.Ref `json:"manifest_ref"`
	ManifestSHA256     string      `json:"manifest_sha256"`
	ReferenceMapRef    proffer.Ref `json:"reference_map_ref"`
	ReferenceMapSHA256 string      `json:"reference_map_sha256"`
	ExpectedObjects    int         `json:"expected_objects"`
}

// ToolkitWorkingCatalogLink retains record identities and citation locators without application source bodies.
// Inputs: authenticated source-map record metadata. Outputs: bounded provenance. Effects: none; choose over copying whole records.
type ToolkitWorkingCatalogLink struct {
	ID            string `json:"id"`
	RecordVersion string `json:"record_version,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	ContentHash   string `json:"content_hash,omitempty"`
	Citation      string `json:"citation,omitempty"`
	SourcePath    string `json:"source_path,omitempty"`
	FilePath      string `json:"file_path,omitempty"`
}

// ToolkitWorkingCatalogObject retains a verified B2 version and its complete source-unit provenance.
// Inputs: placement readback and pinned manifest/map. Outputs: immutable ledger row. Effects: none; choose instead of a partial bucket listing.
type ToolkitWorkingCatalogObject struct {
	Placement         ToolkitContentPlacementObject `json:"placement"`
	ArchiveSHA256     string                        `json:"archive_sha256"`
	ArchiveBytes      int64                         `json:"archive_bytes"`
	LogicalSourcePath string                        `json:"logical_source_path"`
	LogicalCategory   string                        `json:"logical_category"`
	ResourceRole      string                        `json:"resource_role"`
	LinkPolicy        string                        `json:"link_policy"`
	ReferenceRecords  []ToolkitWorkingCatalogLink   `json:"reference_records"`
	SourceRecords     []ToolkitWorkingCatalogLink   `json:"source_records"`
}

// ToolkitWorkingCatalogBatch carries authenticated metadata only inside the Activity/repository boundary.
// Inputs: pinned request and verified object rows. Outputs: canonical admission payload. Effects: none; never pass this array through Temporal.
type ToolkitWorkingCatalogBatch struct {
	Schema  string                        `json:"schema"`
	Request ToolkitWorkingCatalogInput    `json:"request"`
	Objects []ToolkitWorkingCatalogObject `json:"objects"`
}

// ToolkitWorkingCatalogResult reports scoped catalog verification independently from all projections.
// Inputs: admission/readback outcome. Outputs: count, bytes, metadata digest, pins and explicit freshness states.
// Effects: none; choose instead of implying whole-bucket, lake or search freshness from registration.
type ToolkitWorkingCatalogResult struct {
	OperationID          string `json:"operation_id"`
	Objects              int    `json:"objects"`
	Bytes                int64  `json:"bytes"`
	MetadataSHA256       string `json:"metadata_sha256"`
	ReceiptSHA256        string `json:"receipt_sha256"`
	ManifestSHA256       string `json:"manifest_sha256"`
	ReferenceMapSHA256   string `json:"reference_map_sha256"`
	CatalogFreshness     string `json:"catalog_freshness"`
	WholeBucketFreshness string `json:"whole_bucket_freshness"`
	ProjectionFreshness  string `json:"projection_freshness"`
}

// ToolkitWorkingCatalogRepository separates working-file writes from the recovery and read-only catalog clients.
// Inputs: metadata batch or operation. Outputs: atomic registration or independently reconstructed rows.
// Effects: implementation-specific scoped SQL only; choose with its own Case Bible login.
type ToolkitWorkingCatalogRepository interface {
	RegisterToolkitWorking(context.Context, ToolkitWorkingCatalogBatch) error
	ReadToolkitWorking(context.Context, string) (ToolkitWorkingCatalogBatch, error)
}

// ToolkitWorkingCatalogActivities binds a mounted metadata root and the dedicated working catalog repository.
// Inputs: parent-owned connections/configuration. Outputs: two separately registered Activities.
// Effects: none at construction; choose alongside placement, never widen its write scope.
type ToolkitWorkingCatalogActivities struct {
	AllowedRoot string
	Catalog     ToolkitWorkingCatalogRepository
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewToolkitWorkingCatalogActivities constructs the registration/readback group without opening databases or applying SQL.
// Inputs: existing mounted root and dedicated repository. Outputs: Activity group. Effects: heartbeat recording on execution.
// Choose for parent worker wiring; production registration belongs to that lane.
func NewToolkitWorkingCatalogActivities(root string, catalog ToolkitWorkingCatalogRepository) ToolkitWorkingCatalogActivities {
	return ToolkitWorkingCatalogActivities{AllowedRoot: root, Catalog: catalog, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// validateToolkitWorkingInput admits only the exact approved 443-object operation evidence.
// Inputs: request. Outputs: nil or rejection before I/O. Effects: none; choose at each Activity boundary.
func validateToolkitWorkingInput(in ToolkitWorkingCatalogInput) error {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`).MatchString(in.OperationID) || in.ExpectedObjects != ToolkitWorkingCatalogExpectedObjects || in.ReceiptSHA256 != ToolkitWorkingCatalogReceiptSHA256 || in.ManifestSHA256 != ToolkitWorkingCatalogManifestSHA256 || in.ReferenceMapSHA256 != ToolkitWorkingCatalogReferenceMapSHA256 {
		return errors.New("working catalog requires exact approved receipt/manifest/map and 443 objects")
	}
	for _, ref := range []proffer.Ref{in.ReceiptRef, in.ManifestRef, in.ReferenceMapRef} {
		if len(ref) > 4096 || !strings.HasPrefix(string(ref), "file://") {
			return errors.New("working catalog requires bounded mounted metadata references")
		}
	}
	if in.ReceiptRef == in.ManifestRef || in.ReceiptRef == in.ReferenceMapRef || in.ManifestRef == in.ReferenceMapRef {
		return errors.New("metadata references must be distinct")
	}
	return nil
}

// ToolkitWorkingCatalogCanonical validates and hashes all row identities without inventing native B2 SHA-1.
// Inputs: exactly 443 verified pinned versions and source-unit/record provenance. Outputs: sorted JSON and SHA-256.
// Effects: none; choose for SQL admission, replay and independent readback with the same identity contract.
func ToolkitWorkingCatalogCanonical(batch ToolkitWorkingCatalogBatch) ([]byte, string, error) {
	if batch.Schema != ToolkitWorkingCatalogSchema || validateToolkitWorkingInput(batch.Request) != nil || len(batch.Objects) != ToolkitWorkingCatalogExpectedObjects {
		return nil, "", errors.New("invalid working catalog batch")
	}
	seen := map[string]bool{}
	batch.Objects = append([]ToolkitWorkingCatalogObject(nil), batch.Objects...)
	for _, row := range batch.Objects {
		p := row.Placement
		if len(p.VersionID) > 1024 || !regexp.MustCompile(`^[A-Za-z0-9_.-]+$`).MatchString(p.VersionID) || len(p.ObjectKey) > 4096 || !toolkitPlacementPath(p.ObjectKey) || len(p.SourceRef) > 4096 || !toolkitPlacementPath(p.UnitID) || len(p.UnitID) > 64 || len(row.LogicalSourcePath) > 4096 || (row.LogicalCategory != "reference-data" && row.LogicalCategory != "benchbooks" && row.LogicalCategory != "case-law") || len(row.ResourceRole) < 1 || len(row.ResourceRole) > 256 || len(row.LinkPolicy) > 8192 {
			return nil, "", errors.New("working identity exceeds SQL admission bounds")
		}
		u, e := url.Parse(string(p.ObjectRef))
		if e != nil || u.Scheme != "b2" || u.Host != "salem-data" || u.Path != "/"+p.ObjectKey || u.Fragment != "" || u.RawQuery != "versionId="+url.QueryEscape(p.VersionID) || !validToolkitVersionID(p.VersionID) || p.LatestVersionID != p.VersionID || p.Status != "verified" || p.Error != "" || !strings.HasPrefix(p.ObjectKey, toolkitLegalPrefix+"reference-data/") || !toolkitPlacementPath(p.Path) || !validSHA256(p.SHA256) || p.Bytes < 0 || p.Bytes > 256<<20 || !validSHA256(row.ArchiveSHA256) || row.ArchiveBytes <= 0 || row.ArchiveBytes > 8<<30 || p.UnitID == "" || p.SourceRef == "" || row.LogicalSourcePath == "" || row.LinkPolicy == "" || seen[p.ObjectKey] {
			return nil, "", errors.New("working catalog object identity/version/provenance invalid")
		}
		found := false
		for _, v := range p.AfterVersions {
			if v == p.VersionID {
				found = true
			}
			if !validToolkitVersionID(v) {
				return nil, "", errors.New("invalid retained version observation")
			}
		}
		for _, v := range p.BeforeVersions {
			if !validToolkitVersionID(v) {
				return nil, "", errors.New("invalid prior version observation")
			}
		}
		if !found {
			return nil, "", errors.New("verified version missing from placement readback")
		}
		for _, links := range [][]ToolkitWorkingCatalogLink{row.ReferenceRecords, row.SourceRecords} {
			for _, link := range links {
				if link.ID == "" || len(link.ID) > 4096 {
					return nil, "", errors.New("invalid record provenance identity")
				}
			}
		}
		seen[p.ObjectKey] = true
		item, e := json.Marshal(row)
		if e != nil || len(item) > 32<<10 {
			return nil, "", errors.New("working row exceeds metadata bound")
		}
	}
	sort.Slice(batch.Objects, func(i, j int) bool {
		return batch.Objects[i].Placement.ObjectKey < batch.Objects[j].Placement.ObjectKey
	})
	raw, e := json.Marshal(batch)
	if e != nil {
		return nil, "", e
	}
	if len(raw) > toolkitWorkingCatalogLimit {
		return nil, "", errors.New("working metadata exceeds 2MiB")
	}
	return raw, digestBytes(raw), nil
}

// toolkitWorkingMapEntry decodes only source-map metadata used by the catalog, excluding source bodies.
// Inputs: pinned private map entry. Outputs: typed linkage/identity. Effects: none; choose instead of persisting arbitrary JSON.
type toolkitWorkingMapEntry struct {
	ToolkitWorkingCatalogObject
	UnitID         string      `json:"unit_id"`
	SourceRef      proffer.Ref `json:"source_ref"`
	ExactZIPMember string      `json:"exact_zip_member"`
	SHA256         string      `json:"sha256"`
	Bytes          int64       `json:"bytes"`
	Bucket         string      `json:"bucket"`
	ObjectKey      string      `json:"object_key"`
}

// workingPulse checks cancellation and emits only bounded phase progress.
// Inputs: Activity context/phase. Outputs: context error or nil. Effects: heartbeat only; choose over private path logging.
func (a ToolkitWorkingCatalogActivities) workingPulse(ctx context.Context, phase string) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: phase})
	}
	return ctx.Err()
}

// loadWorkingMetadata authenticates the exact receipt, manifest and map without reprocessing source bytes.
// Inputs: mounted digest-pinned metadata refs. Outputs: fully matched metadata batch. Effects: bounded reads on the worker only.
// Choose to reuse placement's per-version hash readback; no bucket listing, PUT or archive-body hashing occurs here.
func (a ToolkitWorkingCatalogActivities) loadWorkingMetadata(ctx context.Context, in ToolkitWorkingCatalogInput) (ToolkitWorkingCatalogBatch, error) {
	batch := ToolkitWorkingCatalogBatch{Schema: ToolkitWorkingCatalogSchema, Request: in}
	if e := validateToolkitWorkingInput(in); e != nil {
		return batch, e
	}
	root, e := canonicalDirectory(a.AllowedRoot)
	if e != nil {
		return batch, e
	}
	read := func(ref proffer.Ref, sha string, limit int64) ([]byte, error) {
		path, e := resolveFileRef(ref, root, false)
		if e != nil {
			return nil, e
		}
		raw, digest, e := toolkitReadBoundedJSON(path, limit, func() error { return a.workingPulse(ctx, "authenticate-working-metadata") })
		if e != nil {
			return nil, e
		}
		if digest != sha {
			return nil, errors.New("working metadata SHA-256 mismatch")
		}
		return raw, nil
	}
	raw, e := read(in.ReceiptRef, in.ReceiptSHA256, 1<<20)
	if e != nil {
		return batch, e
	}
	var receipt ToolkitContentPlacementResult
	if decodeToolkitCatalogJSON(raw, &receipt) != nil || len(raw) != 513268 || !receipt.Complete || receipt.Error != "" || len(receipt.Objects) != in.ExpectedObjects || receipt.Input.ReceiptRef != in.ReceiptRef || receipt.Input.ManifestRef != in.ManifestRef || receipt.Input.ManifestSHA256 != in.ManifestSHA256 {
		return batch, errors.New("approved placement receipt identity/completion mismatch")
	}
	raw, e = read(in.ManifestRef, in.ManifestSHA256, 1<<20)
	if e != nil {
		return batch, e
	}
	var manifest ToolkitContentPlacementManifest
	if decodeToolkitCatalogJSON(raw, &manifest) != nil || validateToolkitPlacementManifest(manifest, receipt.Input) != nil {
		return batch, errors.New("invalid approved placement manifest")
	}
	raw, e = read(in.ReferenceMapRef, in.ReferenceMapSHA256, toolkitWorkingCatalogLimit)
	if e != nil {
		return batch, e
	}
	var mapping struct {
		Schema      string                   `json:"schema"`
		ManifestRef proffer.Ref              `json:"manifest_ref"`
		Entries     []toolkitWorkingMapEntry `json:"entries"`
	}
	// The authenticated map includes source-ledger metadata outside this bounded typed projection.
	if json.Unmarshal(raw, &mapping) != nil || mapping.Schema != "toolkit-sourceunit-link-map/v1" || mapping.ManifestRef != in.ManifestRef || len(mapping.Entries) != in.ExpectedObjects {
		return batch, errors.New("invalid approved source-unit map")
	}
	return assembleToolkitWorkingMetadata(in, receipt.Objects, manifest, mapping.Entries)
}

// assembleToolkitWorkingMetadata matches every authenticated receipt object to both source-unit metadata inputs.
// Inputs: pinned request, placement observations, complete manifest and map entries; outputs: canonical metadata batch.
// Effects: none; choose after byte authentication, not as a substitute for receipt SHA verification.
func assembleToolkitWorkingMetadata(in ToolkitWorkingCatalogInput, objects []ToolkitContentPlacementObject, manifest ToolkitContentPlacementManifest, entries []toolkitWorkingMapEntry) (ToolkitWorkingCatalogBatch, error) {
	batch := ToolkitWorkingCatalogBatch{Schema: ToolkitWorkingCatalogSchema, Request: in}
	mapped := map[string]toolkitWorkingMapEntry{}
	if len(entries) != in.ExpectedObjects || len(objects) != in.ExpectedObjects {
		return batch, errors.New("incomplete working source-unit mapping")
	}
	for _, entry := range entries {
		if _, ok := mapped[entry.ObjectKey]; ok {
			return batch, errors.New("duplicate map object")
		}
		mapped[entry.ObjectKey] = entry
	}
	planned := map[string]ToolkitWorkingCatalogObject{}
	for _, unit := range manifest.Units {
		for _, file := range unit.Files {
			key := toolkitLegalPrefix + unit.Destination + "/" + file.Path
			planned[key] = ToolkitWorkingCatalogObject{Placement: ToolkitContentPlacementObject{UnitID: unit.ID, SourceRef: unit.SourceRef, Path: file.Path, SHA256: file.SHA256, Bytes: file.Bytes}, ArchiveSHA256: unit.ArchiveSHA256, ArchiveBytes: unit.ArchiveBytes}
		}
	}
	for _, p := range objects {
		entry, ok := mapped[p.ObjectKey]
		want, plannedOK := planned[p.ObjectKey]
		if !ok || !plannedOK || entry.UnitID != p.UnitID || entry.SourceRef != p.SourceRef || entry.ExactZIPMember != p.Path || entry.SHA256 != p.SHA256 || entry.Bytes != p.Bytes || entry.Bucket != "salem-data" || entry.ArchiveSHA256 != want.ArchiveSHA256 || entry.ArchiveBytes != want.ArchiveBytes || want.Placement.UnitID != p.UnitID || want.Placement.SourceRef != p.SourceRef || want.Placement.Path != p.Path || want.Placement.SHA256 != p.SHA256 || want.Placement.Bytes != p.Bytes {
			return batch, errors.New("placement/manifest/map provenance differs")
		}
		row := entry.ToolkitWorkingCatalogObject
		row.Placement = p
		batch.Objects = append(batch.Objects, row)
	}
	_, _, e := ToolkitWorkingCatalogCanonical(batch)
	return batch, e
}

// workingSummary reduces validated metadata to a bounded result and independent freshness states.
// Inputs: batch and scoped state. Outputs: pins/count/bytes/hash. Effects: none; choose for Temporal history.
func workingSummary(batch ToolkitWorkingCatalogBatch, state string) (ToolkitWorkingCatalogResult, error) {
	_, sha, e := ToolkitWorkingCatalogCanonical(batch)
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	r := ToolkitWorkingCatalogResult{OperationID: batch.Request.OperationID, Objects: len(batch.Objects), MetadataSHA256: sha, ReceiptSHA256: batch.Request.ReceiptSHA256, ManifestSHA256: batch.Request.ManifestSHA256, ReferenceMapSHA256: batch.Request.ReferenceMapSHA256, CatalogFreshness: state, WholeBucketFreshness: "unchanged", ProjectionFreshness: "not_refreshed"}
	for _, o := range batch.Objects {
		r.Bytes += o.Placement.Bytes
	}
	return r, nil
}

// RegisterToolkitWorkingCatalog atomically admits the authenticated permanent working files into the additive ledger.
// Inputs: exact approved receipt/manifest/map pins and operation ID. Outputs: bounded registered summary pending independent readback.
// Effects: dedicated repository metadata writes only; choose after placement, never as inventory-generation publication.
func (a ToolkitWorkingCatalogActivities) RegisterToolkitWorkingCatalog(ctx context.Context, in ToolkitWorkingCatalogInput) (ToolkitWorkingCatalogResult, error) {
	b, e := a.loadWorkingMetadata(ctx, in)
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	if a.Catalog == nil {
		return ToolkitWorkingCatalogResult{}, errors.New("working catalog repository missing")
	}
	if e = a.workingPulse(ctx, "register-working-catalog"); e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	if e = a.Catalog.RegisterToolkitWorking(ctx, b); e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	return workingSummary(b, "registered_pending_readback")
}

// ReadbackToolkitWorkingCatalog independently reconstructs ledger and occurrence rows against authenticated placement metadata.
// Inputs: the same exact pinned request. Outputs: scoped readback-verified summary or explicit mismatch.
// Effects: separate read-only database transaction and bounded metadata reads; choose after registration, never repair rows during verification.
func (a ToolkitWorkingCatalogActivities) ReadbackToolkitWorkingCatalog(ctx context.Context, in ToolkitWorkingCatalogInput) (ToolkitWorkingCatalogResult, error) {
	want, e := a.loadWorkingMetadata(ctx, in)
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	if a.Catalog == nil {
		return ToolkitWorkingCatalogResult{}, errors.New("working catalog repository missing")
	}
	if e = a.workingPulse(ctx, "independent-working-readback"); e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	got, e := a.Catalog.ReadToolkitWorking(ctx, in.OperationID)
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	expected, e := workingSummary(want, "readback_verified")
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	actual, e := workingSummary(got, "readback_verified")
	if e != nil {
		return ToolkitWorkingCatalogResult{}, e
	}
	if actual != expected {
		return ToolkitWorkingCatalogResult{}, errors.New("working catalog independent metadata/count/hash mismatch")
	}
	return actual, nil
}

// ToolkitWorkingCatalogWorkflow schedules registration and independent readback as two tracked Activities.
// Inputs: approved bounded metadata references. Outputs: verified scoped catalog summary. Effects: scheduling on the caller's existing queue only.
// Choose after ToolkitContentPlacementWorkflow; parent owns worker registration, deployment and execution.
func ToolkitWorkingCatalogWorkflow(ctx workflow.Context, in ToolkitWorkingCatalogInput) (ToolkitWorkingCatalogResult, error) {
	if e := validateToolkitWorkingInput(in); e != nil {
		return ToolkitWorkingCatalogResult{}, temporal.NewNonRetryableApplicationError(e.Error(), "ToolkitWorkingCatalogInvalid", e)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}})
	var registered ToolkitWorkingCatalogResult
	if e := workflow.ExecuteActivity(ctx, ToolkitWorkingCatalogRegisterActivityName, in).Get(ctx, &registered); e != nil {
		return registered, e
	}
	var verified ToolkitWorkingCatalogResult
	e := workflow.ExecuteActivity(ctx, ToolkitWorkingCatalogReadbackActivityName, in).Get(ctx, &verified)
	return verified, e
}
