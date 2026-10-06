// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AIWorkproductCatalogWorkflowName             = "AIWorkproductCatalogWorkflow"
	AIWorkproductCatalogAuthenticateActivityName = "ai_workproduct_catalog_authenticate_activity"
	AIWorkproductCatalogRegisterActivityName     = "ai_workproduct_catalog_register_activity"
	AIWorkproductCatalogReadbackActivityName     = "ai_workproduct_catalog_readback_activity"
	AIWorkproductCatalogSchema                   = "ai-workproduct-source-occurrences/v1"
	AIWorkproductOccurrenceSource                = "local/F-Downloads"
	AIWorkproductOccurrenceScope                 = "F:/Users/matts/Downloads"
)

// AIWorkproductCatalogInput authenticates one bounded physical-placement result without manufacturing source identities.
// Inputs: actual operation label, complete readback reference/SHA and exclusive metadata output. Outputs: three independently scheduled catalog stages. Effects: none as data; choose after placement rather than an inventory-generation or recovery-package workflow.
type AIWorkproductCatalogInput struct {
	Operation      string      `json:"operation"`
	ReadbackRef    proffer.Ref `json:"readback_ref"`
	ReadbackSHA256 string      `json:"readback_sha256"`
	MetadataRef    proffer.Ref `json:"metadata_ref"`
}

// AIWorkproductOccurrenceMetadata carries source observations and exact retained-version validation evidence.
// Inputs: pinned receipts and transport metadata. Outputs: immutable catalog provenance. Effects: none; choose observed filesystem mtime and unknown event date instead of inferred document chronology or provider-native hashes.
type AIWorkproductOccurrenceMetadata struct {
	Schema                     string      `json:"schema"`
	Operation                  string      `json:"operation"`
	OriginalPath               string      `json:"original_path"`
	ObservedMtimeNS            int64       `json:"observed_mtime_ns"`
	TransportObservedAt        string      `json:"transport_observed_at"`
	OriginalUnchangedSizeMtime bool        `json:"original_unchanged_size_mtime"`
	EventDateStatus            string      `json:"event_date_status"`
	Provider                   string      `json:"provider"`
	SourceUnit                 string      `json:"source_unit"`
	SourceRef                  proffer.Ref `json:"source_ref"`
	TransportServerPath        string      `json:"transport_server_path"`
	ProvenanceRef              proffer.Ref `json:"provenance_ref"`
	ProvenanceSHA256           string      `json:"provenance_sha256"`
	PlacementManifestRef       proffer.Ref `json:"placement_manifest_ref"`
	PlacementManifestSHA256    string      `json:"placement_manifest_sha256"`
	ReadbackRef                proffer.Ref `json:"readback_ref"`
	ReadbackSHA256             string      `json:"readback_sha256"`
	ObjectRef                  proffer.Ref `json:"object_ref"`
	VersionID                  string      `json:"version_id"`
	SHA256                     string      `json:"sha256"`
	CatalogBatchSHA256         string      `json:"catalog_batch_sha256,omitempty"`
}

// AIWorkproductOccurrence models the existing source-occurrence identity and only the values this slice admits.
// Inputs: source observation and placement evidence. Outputs: one existing-table row; source_id stays empty. Effects: none; choose six original occurrences instead of a new SQL identity or second catalog.
type AIWorkproductOccurrence struct {
	Source      string                          `json:"source"`
	Scope       string                          `json:"scope"`
	Path        string                          `json:"path"`
	SourceID    string                          `json:"source_id"`
	Size        int64                           `json:"size"`
	Disposition string                          `json:"disposition"`
	B2Key       string                          `json:"b2_key"`
	Metadata    AIWorkproductOccurrenceMetadata `json:"metadata"`
}

// AIWorkproductCatalogBatch seals all original occurrences into one bounded immutable metadata payload.
// Inputs: reviewed request and authenticated rows. Outputs: deterministic metadata for atomic admission. Effects: none; choose one complete batch over a partial bucket generation or package ledger.
type AIWorkproductCatalogBatch struct {
	Schema  string                    `json:"schema"`
	Request AIWorkproductCatalogInput `json:"request"`
	Rows    []AIWorkproductOccurrence `json:"rows"`
}

// AIWorkproductCatalogSummary passes only pinned metadata coordinates between independent Activities.
// Inputs: verified metadata file. Outputs: operation, reference, SHA and bounded row count. Effects: none; choose references instead of source bodies or full row arrays in Temporal history.
type AIWorkproductCatalogSummary struct {
	Operation      string      `json:"operation"`
	MetadataRef    proffer.Ref `json:"metadata_ref"`
	MetadataSHA256 string      `json:"metadata_sha256"`
	Count          int         `json:"count"`
}

// AIWorkproductOccurrenceCatalog is the narrow extension of the existing separately configured writer lifecycle.
// Inputs: authenticated batch or operation/batch pin. Outputs: immutable registration or bounded independent readback. Effects: metadata operations through restricted database functions only; choose instead of widening the read-only catalog pool.
type AIWorkproductOccurrenceCatalog interface {
	RegisterAIWorkproductOccurrences(context.Context, AIWorkproductCatalogBatch) error
	ReadAIWorkproductOccurrences(context.Context, string, string) ([]AIWorkproductOccurrence, error)
}

// AIWorkproductCatalogActivities binds the existing scratch mount/store resolver to the narrow admitted repository.
// Inputs: existing seams and optional heartbeat. Outputs: independent metadata authentication, registration and readback. Effects: none until called; choose alongside placement, leaving content indexing and lake refresh separate.
type AIWorkproductCatalogActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Catalog     AIWorkproductOccurrenceCatalog
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewAIWorkproductCatalogActivities constructs the group without reading sources or applying database schema.
// Inputs: existing DeriveScratchDir, store resolver and separately admitted writer. Outputs: Activity group. Effects: none until execution; choose the existing worker lifecycle rather than another catalog runtime.
func NewAIWorkproductCatalogActivities(root string, stores func(string) (smsthreads.ObjectStore, error), catalog AIWorkproductOccurrenceCatalog) *AIWorkproductCatalogActivities {
	return &AIWorkproductCatalogActivities{AllowedRoot: root, Stores: stores, Catalog: catalog, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// AIWorkproductCatalogWorkflow authenticates metadata, registers immutable existing-table rows and independently reads them back.
// Inputs: complete placement receipt pin and metadata destination. Outputs: readback-verified metadata pin. Effects: three named Activities; choose after physical placement without asserting ingestion, projection refresh or event dates.
func AIWorkproductCatalogWorkflow(ctx workflow.Context, in AIWorkproductCatalogInput) (AIWorkproductCatalogSummary, error) {
	if err := validateAIWorkproductCatalogInput(in); err != nil {
		return AIWorkproductCatalogSummary{}, temporal.NewNonRetryableApplicationError(err.Error(), "AIWorkproductCatalogInvalid", err)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}})
	var out AIWorkproductCatalogSummary
	if err := workflow.ExecuteActivity(ctx, AIWorkproductCatalogAuthenticateActivityName, in).Get(ctx, &out); err != nil {
		return out, err
	}
	if err := workflow.ExecuteActivity(ctx, AIWorkproductCatalogRegisterActivityName, out).Get(ctx, nil); err != nil {
		return out, err
	}
	err := workflow.ExecuteActivity(ctx, AIWorkproductCatalogReadbackActivityName, out).Get(ctx, &out)
	return out, err
}

// validateAIWorkproductCatalogInput checks request bounds before reading any file or calling a store.
// Inputs: independently callable request. Outputs: explicit invalid-input error. Effects: none; choose before Activity I/O rather than relying only on workflow admission.
func validateAIWorkproductCatalogInput(in AIWorkproductCatalogInput) error {
	if !aiCatalogNamespace(in.Operation, 64) || !validSHA256(in.ReadbackSHA256) || len(in.ReadbackRef) < 1 || len(in.ReadbackRef) > 4096 || len(in.MetadataRef) < 1 || len(in.MetadataRef) > 4096 || in.MetadataRef == in.ReadbackRef {
		return errors.New("bounded operation and distinct pinned metadata references required")
	}
	return nil
}

// aiCatalogNamespace admits bounded ASCII operation/source-unit labels shared with the restricted SQL contract.
// Inputs: label and ceiling. Outputs: validity. Effects: none; choose over optional preservation namespaces for required identities.
func aiCatalogNamespace(value string, limit int) bool {
	return len(value) > 0 && len(value) <= limit && safePreservationNamespace(value) && value[0] != '-' && value[0] != '_'
}

// AIWorkproductCatalogCanonical validates and sorts the complete immutable admission payload.
// Inputs: authenticated existing-table rows. Outputs: canonical bounded JSON and SHA-256. Effects: none; choose in registration and readback to share one collision contract without inventing source IDs.
func AIWorkproductCatalogCanonical(batch AIWorkproductCatalogBatch) ([]byte, string, error) {
	if batch.Schema != AIWorkproductCatalogSchema || validateAIWorkproductCatalogInput(batch.Request) != nil || len(batch.Rows) < 1 || len(batch.Rows) > 16 {
		return nil, "", errors.New("invalid bounded catalog batch")
	}
	seen := map[string]bool{}
	var total int64
	for _, r := range batch.Rows {
		m := r.Metadata
		if r.Source != AIWorkproductOccurrenceSource || r.Scope != AIWorkproductOccurrenceScope || r.SourceID != "" || !toolkitPlacementPath(r.Path) || strings.Contains(r.Path, "/") || len(r.Path) > 255 || !strings.EqualFold(filepath.Ext(r.Path), ".md") || seen[r.Path] || r.Size < 1 || r.Size > 1<<20 || r.Disposition != "copied" || m.Schema != AIWorkproductCatalogSchema || m.Operation != batch.Request.Operation || !toolkitPlacementPath(m.SourceUnit) || strings.Contains(m.SourceUnit, "/") || len(m.SourceUnit) > 128 || r.B2Key != aiWorkproductPrefix+m.SourceUnit+"/"+r.Path || strings.ReplaceAll(m.OriginalPath, "\\", "/") != r.Scope+"/"+r.Path || m.ObservedMtimeNS < 0 || !m.OriginalUnchangedSizeMtime || m.EventDateStatus != "unknown" || len(m.Provider) < 1 || len(m.Provider) > 64 || m.CatalogBatchSHA256 != "" || m.ReadbackRef != batch.Request.ReadbackRef || m.ReadbackSHA256 != batch.Request.ReadbackSHA256 || !validSHA256(m.SHA256) || !validSHA256(m.ProvenanceSHA256) || !validSHA256(m.PlacementManifestSHA256) || !validToolkitVersionID(m.VersionID) {
			return nil, "", errors.New("occurrence identity, chronology or provenance collision")
		}
		if !aiCatalogNamespace(m.SourceUnit, 128) || strings.IndexFunc(r.Path, unicode.IsControl) >= 0 || strings.IndexFunc(m.VersionID, unicode.IsControl) >= 0 {
			return nil, "", errors.New("invalid required namespace or controlled filename/version")
		}
		if _, err := time.Parse(time.RFC3339Nano, m.TransportObservedAt); err != nil {
			return nil, "", errors.New("transport observation timestamp is invalid")
		}
		for _, ref := range []proffer.Ref{m.SourceRef, m.ProvenanceRef, m.PlacementManifestRef, m.ReadbackRef} {
			u, err := url.Parse(string(ref))
			if err != nil || u.Scheme != "file" || u.Host != "" || !strings.HasPrefix(u.Path, "/") || u.RawQuery != "" || u.Fragment != "" || len(ref) > 4096 {
				return nil, "", errors.New("catalog requires bounded mounted provenance references")
			}
		}
		pinned := url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + r.B2Key, RawQuery: url.Values{"versionId": []string{m.VersionID}}.Encode()}
		if m.ObjectRef != proffer.Ref(pinned.String()) || len(m.TransportServerPath) < 1 || len(m.TransportServerPath) > 4096 {
			return nil, "", errors.New("exact retained object or transport locator differs")
		}
		seen[r.Path] = true
		total += r.Size
	}
	if total > 16<<20 {
		return nil, "", errors.New("16 MiB source batch ceiling exceeded")
	}
	batch.Rows = append([]AIWorkproductOccurrence(nil), batch.Rows...)
	sort.Slice(batch.Rows, func(i, j int) bool { return batch.Rows[i].Path < batch.Rows[j].Path })
	raw, err := json.Marshal(batch)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > 256<<10 {
		return nil, "", errors.New("256 KiB metadata ceiling exceeded")
	}
	return raw, digestBytes(raw), nil
}

// aiWorkproductTransport records only observed transfer metadata from the pinned root provenance document.
// Inputs: full digest-authenticated transport JSON. Outputs: exact source mappings and observation timestamp. Effects: none; choose typed int64 mtime fields to avoid floating-point timestamp loss while the full document stays pinned by hash.
type aiWorkproductTransport struct {
	ObservedAt string `json:"observed_at"`
	Sources    []struct {
		OriginalPath string `json:"original_path"`
		OriginalName string `json:"original_name"`
		ServerPath   string `json:"server_path"`
		Bytes        int64  `json:"bytes"`
		LocalMtimeNS int64  `json:"local_mtime_ns"`
		Unchanged    bool   `json:"local_unchanged_size_mtime"`
	} `json:"sources"`
}

// aiCatalogPulse checks cancellation and records bounded progress without source contents.
// Inputs: Activity context and phase. Outputs: cancellation state. Effects: heartbeat only; choose for all independent metadata operations.
func (a *AIWorkproductCatalogActivities) aiCatalogPulse(ctx context.Context, phase string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: phase})
	}
	return ctx.Err()
}

// AuthenticateAIWorkproductCatalog verifies complete placement and transport metadata before sealing occurrence rows.
// Inputs: exact readback pin and exclusive metadata output. Outputs: canonical metadata ref/hash/count. Effects: bounded metadata reads, exact-version GET open/close without source-body reads, and exclusive metadata write; choose before database registration without rehashing, parsing or indexing notes.
func (a *AIWorkproductCatalogActivities) AuthenticateAIWorkproductCatalog(ctx context.Context, in AIWorkproductCatalogInput) (AIWorkproductCatalogSummary, error) {
	var out AIWorkproductCatalogSummary
	if err := validateAIWorkproductCatalogInput(in); err != nil {
		return out, err
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return out, err
	}
	if err = a.aiCatalogPulse(ctx, "authenticate-ai-workproduct-catalog"); err != nil {
		return out, err
	}
	var receipt AIWorkproductReceipt
	if _, err = aiWorkproductReadJSON(ctx, root, in.ReadbackRef, in.ReadbackSHA256, &receipt); err != nil {
		return out, err
	}
	if receipt.Stage != "readback" || !receipt.Complete || receipt.Error != "" || receipt.IngestionStatus != "pending" || receipt.CatalogStatus != "pending" || len(receipt.Objects) != len(receipt.Manifest.Files) || aiWorkproductManifestValid(receipt.Manifest) != nil {
		return out, errors.New("complete independently verified placement receipt required")
	}
	var manifest AIWorkproductManifest
	if _, err = aiWorkproductReadJSON(ctx, root, receipt.Request.Input.ManifestRef, receipt.Request.Input.ManifestSHA256, &manifest); err != nil || !reflect.DeepEqual(manifest, receipt.Manifest) {
		return out, errors.New("placement manifest pin/mapping mismatch")
	}
	if in.ReadbackRef != proffer.Ref(string(receipt.Request.Input.ReceiptRef)+".readback.json") || receipt.Request.PreviousRef != proffer.Ref(string(receipt.Request.Input.ReceiptRef)+".copy.json") || !validSHA256(receipt.Request.PreviousSHA256) {
		return out, errors.New("readback/copy receipt reference chain differs")
	}
	var copied AIWorkproductReceipt
	if _, err = aiWorkproductReadJSON(ctx, root, receipt.Request.PreviousRef, receipt.Request.PreviousSHA256, &copied); err != nil || copied.Stage != "copy" || !copied.Complete || copied.Error != "" || !reflect.DeepEqual(copied.Manifest, manifest) || copied.Request.Input != receipt.Request.Input || len(copied.Objects) != len(receipt.Objects) {
		return out, errors.New("copy predecessor pin/mapping mismatch")
	}
	var rawTransport json.RawMessage
	if _, err = aiWorkproductReadJSON(ctx, root, manifest.ProvenanceRef, manifest.ProvenanceSHA256, &rawTransport); err != nil {
		return out, err
	}
	var transport aiWorkproductTransport
	if err = json.Unmarshal(rawTransport, &transport); err != nil || len(transport.Sources) != len(manifest.Files) {
		return out, errors.New("complete transport source mappings required")
	}
	sourceRoot, err := resolveFileRef(manifest.SourceRef, root, true)
	if err != nil {
		return out, err
	}
	output, err := aiWorkproductReceiptPath(root, sourceRoot, in.MetadataRef)
	if err != nil {
		return out, err
	}
	batch := AIWorkproductCatalogBatch{Schema: AIWorkproductCatalogSchema, Request: in}
	byName := map[string]int{}
	for i, s := range transport.Sources {
		if _, exists := byName[s.OriginalName]; exists {
			return out, errors.New("duplicate original transport occurrence")
		}
		byName[s.OriginalName] = i
	}
	if a.Stores == nil {
		return out, errors.New("existing B2 resolver required")
	}
	base, err := a.Stores("b2")
	if err != nil {
		return out, err
	}
	store, ok := base.(toolkitVersionedObjectStore)
	if !ok {
		return out, errors.New("exact-version metadata observation capability required")
	}
	for i, f := range manifest.Files {
		if err = a.aiCatalogPulse(ctx, "authenticate-ai-workproduct-version"); err != nil {
			return out, err
		}
		obj := receipt.Objects[i]
		copyObj := copied.Objects[i]
		idx, exists := byName[f.Name]
		if !exists {
			return out, errors.New("supplied name missing from root transport provenance")
		}
		s := transport.Sources[idx]
		serverBase := strings.Split(strings.ReplaceAll(s.ServerPath, "\\", "/"), "/")
		sourceBase := strings.Split(f.SourcePath, "/")
		if s.Bytes != f.Bytes || s.OriginalName != f.Name || !s.Unchanged || len(serverBase) == 0 || serverBase[len(serverBase)-1] != sourceBase[len(sourceBase)-1] || obj.Status != "verified" || obj.Error != "" || obj.UnitID != manifest.SourceUnit || obj.Path != f.Name || obj.Bytes != f.Bytes || obj.SHA256 != f.SHA256 || obj.SourceRef != toolkitFileRef(filepath.Join(sourceRoot, filepath.FromSlash(f.SourcePath))) || obj.ObjectKey != aiWorkproductPrefix+manifest.SourceUnit+"/"+f.Name || len(obj.BeforeVersions) != 0 || toolkitPlacementConflictChecks(obj.BeforeVersions, obj.AfterVersions, obj.VersionID, obj.LatestVersionID) != nil || copyObj.VersionID != obj.VersionID || copyObj.ObjectRef != obj.ObjectRef || copyObj.SHA256 != obj.SHA256 || copyObj.Bytes != obj.Bytes || copyObj.Status != "copied-awaiting-independent-readback" || copyObj.Error != "" {
			return out, errors.New("transport/copy/readback identity or version evidence mismatch")
		}
		stream, e := store.OpenVersion(ctx, "salem-data", obj.ObjectKey, obj.VersionID)
		if e != nil {
			return out, errors.New("retained placed version unavailable")
		}
		if e = stream.Close(); e != nil {
			return out, e
		}
		batch.Rows = append(batch.Rows, AIWorkproductOccurrence{Source: AIWorkproductOccurrenceSource, Scope: AIWorkproductOccurrenceScope, Path: f.Name, SourceID: "", Size: f.Bytes, Disposition: "copied", B2Key: obj.ObjectKey, Metadata: AIWorkproductOccurrenceMetadata{Schema: AIWorkproductCatalogSchema, Operation: in.Operation, OriginalPath: s.OriginalPath, ObservedMtimeNS: s.LocalMtimeNS, TransportObservedAt: transport.ObservedAt, OriginalUnchangedSizeMtime: s.Unchanged, EventDateStatus: "unknown", Provider: manifest.Provider, SourceUnit: manifest.SourceUnit, SourceRef: obj.SourceRef, TransportServerPath: s.ServerPath, ProvenanceRef: manifest.ProvenanceRef, ProvenanceSHA256: manifest.ProvenanceSHA256, PlacementManifestRef: receipt.Request.Input.ManifestRef, PlacementManifestSHA256: receipt.Request.Input.ManifestSHA256, ReadbackRef: in.ReadbackRef, ReadbackSHA256: in.ReadbackSHA256, ObjectRef: obj.ObjectRef, VersionID: obj.VersionID, SHA256: obj.SHA256}})
	}
	raw, digest, err := AIWorkproductCatalogCanonical(batch)
	if err != nil {
		return out, err
	}
	out = AIWorkproductCatalogSummary{Operation: in.Operation, MetadataRef: in.MetadataRef, MetadataSHA256: digest, Count: len(batch.Rows)}
	var prior AIWorkproductCatalogBatch
	if actual, e := aiWorkproductReadJSON(ctx, root, in.MetadataRef, "", &prior); e == nil {
		if actual != digest {
			return out, errors.New("exclusive metadata output retains a different operation")
		}
		return out, nil
	}
	// An existing malformed or linked output cannot be replaced by the exclusive writer.
	if _, err = aiWorkproductWriteExclusive(output, json.RawMessage(raw)); err != nil {
		return out, err
	}
	return out, nil
}

// loadAIWorkproductCatalog reloads and validates the exact sealed metadata at each independent boundary.
// Inputs: compact predecessor pin. Outputs: complete canonical batch or visible mismatch. Effects: bounded mounted metadata reads only; choose instead of trusting mutable in-memory predecessor rows.
func (a *AIWorkproductCatalogActivities) loadAIWorkproductCatalog(ctx context.Context, in AIWorkproductCatalogSummary) (AIWorkproductCatalogBatch, error) {
	var batch AIWorkproductCatalogBatch
	if !aiCatalogNamespace(in.Operation, 64) || !validSHA256(in.MetadataSHA256) || in.Count < 1 || in.Count > 16 {
		return batch, errors.New("invalid bounded catalog predecessor")
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return batch, err
	}
	if _, err = aiWorkproductReadJSON(ctx, root, in.MetadataRef, in.MetadataSHA256, &batch); err != nil {
		return batch, err
	}
	_, digest, err := AIWorkproductCatalogCanonical(batch)
	if err != nil || digest != in.MetadataSHA256 || batch.Request.Operation != in.Operation || batch.Request.MetadataRef != in.MetadataRef || len(batch.Rows) != in.Count {
		return batch, errors.New("sealed catalog metadata identity/count/hash mismatch")
	}
	return batch, nil
}

// RegisterAIWorkproductCatalog atomically admits only the already authenticated existing source occurrences.
// Inputs: sealed metadata pin. Outputs: collision or successful immutable registration. Effects: restricted repository function call only; choose after authentication without hashing, source reads, recovery rows or inventory generation.
func (a *AIWorkproductCatalogActivities) RegisterAIWorkproductCatalog(ctx context.Context, in AIWorkproductCatalogSummary) error {
	batch, err := a.loadAIWorkproductCatalog(ctx, in)
	if err != nil {
		return err
	}
	if a.Catalog == nil {
		return errors.New("narrow Case Bible source-occurrence writer required")
	}
	if err = a.aiCatalogPulse(ctx, "register-ai-workproduct-occurrences"); err != nil {
		return err
	}
	return a.Catalog.RegisterAIWorkproductOccurrences(ctx, batch)
}

// ReadbackAIWorkproductCatalog independently compares exact stored occurrences and full immutable metadata.
// Inputs: sealed metadata pin. Outputs: same pin after complete readback equality. Effects: separate restricted database read only; choose instead of interpreting registration commit as proof or refreshing content indexes.
func (a *AIWorkproductCatalogActivities) ReadbackAIWorkproductCatalog(ctx context.Context, in AIWorkproductCatalogSummary) (AIWorkproductCatalogSummary, error) {
	batch, err := a.loadAIWorkproductCatalog(ctx, in)
	if err != nil {
		return AIWorkproductCatalogSummary{}, err
	}
	if a.Catalog == nil {
		return AIWorkproductCatalogSummary{}, errors.New("narrow Case Bible source-occurrence reader required")
	}
	if err = a.aiCatalogPulse(ctx, "readback-ai-workproduct-occurrences"); err != nil {
		return AIWorkproductCatalogSummary{}, err
	}
	rows, err := a.Catalog.ReadAIWorkproductOccurrences(ctx, in.Operation, in.MetadataSHA256)
	if err != nil {
		return AIWorkproductCatalogSummary{}, err
	}
	if len(rows) != in.Count {
		return AIWorkproductCatalogSummary{}, errors.New("independent catalog occurrence count mismatch")
	}
	for i := range rows {
		if rows[i].Metadata.CatalogBatchSHA256 != in.MetadataSHA256 {
			return AIWorkproductCatalogSummary{}, errors.New("stored occurrence batch pin mismatch")
		}
		rows[i].Metadata.CatalogBatchSHA256 = ""
	}
	batch.Rows = rows
	_, digest, err := AIWorkproductCatalogCanonical(batch)
	if err != nil || digest != in.MetadataSHA256 {
		return AIWorkproductCatalogSummary{}, fmt.Errorf("independent catalog full metadata mismatch")
	}
	return in, nil
}
