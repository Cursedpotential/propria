// Byline: Codex · GPT-6 · 2026-10-04. Separate Case Bible metadata registration; originals never change.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitCatalogRegistrationWorkflowName = "ToolkitCatalogRegistrationWorkflow"
	ToolkitCatalogMetadataActivityName     = "toolkit_catalog_verify_metadata_activity"
	ToolkitCatalogRegisterActivityName     = "toolkit_catalog_register_activity"
	ToolkitCatalogReadbackActivityName     = "toolkit_catalog_readback_activity"
	toolkitCatalogMetadataLimit            = 256 << 10
	ToolkitRecoveryCatalogSchema           = "toolkit-recovery-catalog/v1"
)

// ToolkitCatalogRegistrationInput pins one complete fifteen-package preservation result.
// Inputs: operation ID, preservation namespace, result ref/SHA/exact remote version and mounted metadata output ref.
// Outputs: independent metadata verification, registration and catalog readback. Effects: workflow scheduling only.
// Choose after versioned preservation, never as source ingestion or a global catalog-generation replacement.
type ToolkitCatalogRegistrationInput struct {
	OperationID           string      `json:"operation_id"`
	PreservationNamespace string      `json:"preservation_namespace"`
	ResultRef             proffer.Ref `json:"result_ref"`
	ResultSHA256          string      `json:"result_sha256"`
	ResultVersionID       string      `json:"result_version_id,omitempty"`
	MetadataRef           proffer.Ref `json:"metadata_ref"`
}

// ToolkitRecoveryCatalogBatch preserves every selected original occurrence and all exact receipt versions.
// Inputs: authenticated result coordinates and independently read-back receipt metadata. Outputs: deterministic bounded ledger payload.
// Effects: none. Choose over content-hash deduplication: identical archive bytes do not collapse source occurrences.
type ToolkitRecoveryCatalogBatch struct {
	Schema           string                              `json:"schema"`
	Request          ToolkitCatalogRegistrationInput     `json:"request"`
	InventoryRef     proffer.Ref                         `json:"inventory_ref"`
	InventorySHA256  string                              `json:"inventory_sha256"`
	Packages         []ToolkitPackagePreservationReceipt `json:"packages"`
	ResultProvenance map[string]any                      `json:"result_provenance,omitempty"`
}

// toolkitCatalogPreservationEnvelope admits the existing exported preservation result with its parent-added provenance.
// Inputs: digest-pinned bounded result JSON. Outputs: existing receipt fields and retained provenance metadata.
// Effects: none. Choose over changing the preservation Activity schema; provenance claims do not authorize catalog/projection success.
type toolkitCatalogPreservationEnvelope struct {
	ToolkitPackagePreservationResult
	Provenance map[string]any `json:"provenance,omitempty"`
}

// ToolkitCatalogMetadataResult carries only immutable metadata coordinates through workflow history.
// Inputs: completed metadata verification. Outputs: operation, pinned metadata reference/hash and fifteen-row count.
// Effects: none. Choose instead of passing receipt arrays or archive bodies to the next Activity.
type ToolkitCatalogMetadataResult struct {
	OperationID    string      `json:"operation_id"`
	MetadataRef    proffer.Ref `json:"metadata_ref"`
	MetadataSHA256 string      `json:"metadata_sha256"`
	Count          int         `json:"count"`
}

// ToolkitRecoveryCatalogRepository separates the new writer from the existing read-only catalog client.
// Inputs: one validated metadata batch or operation identity. Outputs: atomic registration or bounded independent readback.
// Effects: implementation-specific metadata writes/reads only. Choose for recovery receipts, never corpus/working-record writes.
type ToolkitRecoveryCatalogRepository interface {
	RegisterToolkitRecovery(context.Context, ToolkitRecoveryCatalogBatch) error
	ReadToolkitRecovery(context.Context, string) (ToolkitRecoveryCatalogBatch, error)
}

// ToolkitCatalogRegistrationActivities binds the existing object store, mounted metadata root and separate repository.
// Inputs: injected seams. Outputs: separately callable verification, registration and readback Activities.
// Effects: none until called. Choose alongside preservation; do not wire the read-only CatalogVersionStore as a writer.
type ToolkitCatalogRegistrationActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Catalog     ToolkitRecoveryCatalogRepository
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewToolkitCatalogRegistrationActivities constructs metadata operations without opening clients or applying schema.
// Inputs: existing mounted root/store resolver and separately configured Case Bible writer. Outputs: Activity group.
// Effects: heartbeat recording when executed. Choose for parent-owned worker registration/configuration.
func NewToolkitCatalogRegistrationActivities(root string, stores func(string) (smsthreads.ObjectStore, error), catalog ToolkitRecoveryCatalogRepository) ToolkitCatalogRegistrationActivities {
	return ToolkitCatalogRegistrationActivities{AllowedRoot: root, Stores: stores, Catalog: catalog, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// ToolkitCatalogRegistrationWorkflow verifies metadata, atomically registers it, then independently reads it back.
// Inputs: a digest-pinned complete preservation result. Outputs: readback-verified metadata receipt or visible failure.
// Effects: schedules three independent Activities on the existing worker; no deployment/schema/source mutation.
// Choose after preservation completion; registration success alone is never readback or projection success.
func ToolkitCatalogRegistrationWorkflow(ctx workflow.Context, input ToolkitCatalogRegistrationInput) (ToolkitCatalogMetadataResult, error) {
	if err := validateToolkitCatalogInput(input); err != nil {
		return ToolkitCatalogMetadataResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitCatalogInvalid", err)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 5 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}})
	var metadata ToolkitCatalogMetadataResult
	if err := workflow.ExecuteActivity(ctx, ToolkitCatalogMetadataActivityName, input).Get(ctx, &metadata); err != nil {
		return metadata, err
	}
	if err := workflow.ExecuteActivity(ctx, ToolkitCatalogRegisterActivityName, metadata).Get(ctx, nil); err != nil {
		return metadata, err
	}
	var verified ToolkitCatalogMetadataResult
	err := workflow.ExecuteActivity(ctx, ToolkitCatalogReadbackActivityName, metadata).Get(ctx, &verified)
	return verified, err
}

// validateToolkitCatalogInput checks hard reference/identity bounds before any I/O.
// Inputs: registration request. Outputs: nil or explicit rejection. Effects: none.
// Choose at every independently callable Activity boundary, not only workflow entry.
func validateToolkitCatalogInput(input ToolkitCatalogRegistrationInput) error {
	if input.OperationID == "" || input.PreservationNamespace == "" || !safePreservationNamespace(input.OperationID) || !safePreservationNamespace(input.PreservationNamespace) || !validSHA256(input.ResultSHA256) || len(input.ResultRef) == 0 || len(input.ResultRef) > 4096 || len(input.MetadataRef) == 0 || len(input.MetadataRef) > 4096 {
		return errors.New("bounded operation, namespace, result digest and references required")
	}
	first := input.OperationID[0]
	if !(first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z' || first >= '0' && first <= '9') {
		return errors.New("operation ID must begin with an alphanumeric character")
	}
	if strings.HasPrefix(string(input.ResultRef), "b2://") {
		if _, _, err := toolkitObjectCoordinates(input.ResultRef); err != nil {
			return err
		}
		if !validToolkitVersionID(input.ResultVersionID) {
			return errors.New("B2 result requires exact VersionId")
		}
	} else if !strings.HasPrefix(string(input.ResultRef), "file://") || input.ResultVersionID != "" {
		return errors.New("result must be mounted file or pinned B2 version")
	}
	return nil
}

// decodeToolkitCatalogJSON strictly decodes bounded metadata without accepting unknown/trailing payloads.
// Inputs: JSON bytes and destination. Outputs: decoded value or error. Effects: none.
// Choose for catalog admission rather than permissive corpus parsers.
func decodeToolkitCatalogJSON(raw []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return errors.New("invalid catalog metadata JSON")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("trailing catalog metadata JSON")
	}
	return nil
}

// ToolkitRecoveryCatalogCanonical returns the deterministic admission payload and its complete metadata hash.
// Inputs: fifteen unique original occurrences and pinned refs/versions. Outputs: sorted canonical JSON plus SHA-256.
// Effects: none; rejects malformed identities. Choose in repositories and readback to share the same collision contract.
func ToolkitRecoveryCatalogCanonical(batch ToolkitRecoveryCatalogBatch) ([]byte, string, error) {
	if batch.Schema != ToolkitRecoveryCatalogSchema || validateToolkitCatalogInput(batch.Request) != nil || batch.InventoryRef == "" || len(batch.InventoryRef) > 4096 || !validSHA256(batch.InventorySHA256) || len(batch.Packages) != 15 {
		return nil, "", errors.New("catalog batch must contain exactly fifteen pinned occurrences")
	}
	seenNames, seenRefs := map[string]bool{}, map[proffer.Ref]bool{}
	for _, p := range batch.Packages {
		if !safeToolkitPackageName(p.PackageName) || seenNames[p.PackageName] || p.OriginalRef == "" || len(p.OriginalRef) > 4096 || seenRefs[p.OriginalRef] || p.StorageMode != ToolkitPackagePreservationModeVersioned || !validSHA256(p.SHA256) || p.Bytes <= 0 || p.Bytes > toolkitPreservationMaxSinglePut || !validToolkitVersionID(p.ArchiveVersionID) || !validToolkitVersionID(p.ReceiptVersionID) || !validToolkitVersionID(p.VerificationVersionID) || !validSHA256(p.ReceiptSHA256) || !validSHA256(p.VerificationReceiptSHA256) || p.ReceiptBytes <= 0 || p.ReceiptBytes > 16<<10 || p.VerificationReceiptBytes <= 0 || p.VerificationReceiptBytes > 16<<10 {
			return nil, "", errors.New("invalid or duplicate catalog occurrence")
		}
		archive, receipt := toolkitPreservationRefs(batch.Request.PreservationNamespace, batch.InventorySHA256, p.PackageName)
		if p.PreservedRef != archive || p.ReceiptRef != receipt || p.VerificationRef != proffer.Ref(string(receipt)+".verified.json") {
			return nil, "", errors.New("catalog references differ from pinned preservation namespace")
		}
		seenNames[p.PackageName] = true
		seenRefs[p.OriginalRef] = true
	}
	batch.Packages = append([]ToolkitPackagePreservationReceipt(nil), batch.Packages...)
	sort.Slice(batch.Packages, func(i, j int) bool { return batch.Packages[i].OriginalRef < batch.Packages[j].OriginalRef })
	raw, err := json.Marshal(batch)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > toolkitCatalogMetadataLimit {
		return nil, "", errors.New("catalog metadata exceeds 256KiB")
	}
	return raw, digestBytes(raw), nil
}

// catalogPulse checks cancellation and emits bounded progress for one independent operation.
// Inputs: Activity context and phase. Outputs: cancellation error or nil. Effects: optional heartbeat only.
// Choose instead of logging source contents or credentials.
func (a ToolkitCatalogRegistrationActivities) catalogPulse(ctx context.Context, phase string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: phase})
	}
	return ctx.Err()
}

// readCatalogRemote pins one bounded metadata object using the existing exact-version store seam.
// Inputs: fixed recovery ref, VersionId and positive byte limit. Outputs: at most the limit of metadata bytes.
// Effects: exact-version GET only; never PUT or archive-body reads. Choose for result/receipt verification.
func (a ToolkitCatalogRegistrationActivities) readCatalogRemote(ctx context.Context, ref proffer.Ref, version string, limit int64) ([]byte, error) {
	if a.Stores == nil {
		return nil, errors.New("catalog metadata object-store resolver missing")
	}
	bucket, key, err := toolkitObjectCoordinates(ref)
	if err != nil {
		return nil, err
	}
	store, err := a.Stores("b2")
	if err != nil {
		return nil, errors.New("B2 metadata store unavailable")
	}
	return readToolkitRemoteBoundedVersion(ctx, store, bucket, key, version, limit, a.Heartbeat, "")
}

// checkCatalogArchiveVersion confirms that an exact retained archive version still opens without reading its body.
// Inputs: context and receipt-pinned archive coordinates. Outputs: visible missing/version/cancellation error or nil.
// Effects: exact-version GET headers and immediate stream close only. Choose over rehashing originals already verified by preservation.
func (a ToolkitCatalogRegistrationActivities) checkCatalogArchiveVersion(ctx context.Context, ref proffer.Ref, version string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Stores == nil {
		return errors.New("catalog object-store resolver missing")
	}
	bucket, key, err := toolkitObjectCoordinates(ref)
	if err != nil {
		return err
	}
	store, err := a.Stores("b2")
	if err != nil {
		return errors.New("B2 archive version store unavailable")
	}
	pinned, ok := store.(toolkitVersionedObjectStore)
	if !ok {
		return errors.New("exact archive version capability required")
	}
	stream, err := pinned.OpenVersion(ctx, bucket, key, version)
	if err != nil {
		return errors.New("retained archive version unavailable")
	}
	if err = stream.Close(); err != nil {
		return errors.New("retained archive version stream close failed")
	}
	return ctx.Err()
}

// VerifyToolkitCatalogMetadata independently verifies pinned preservation/verification receipt versions.
// Inputs: mounted or exact-version B2 result and exclusive mounted metadata output. Outputs: pinned canonical metadata coordinates.
// Effects: reads bounded metadata, opens/closes exact archive versions without body reads, and exclusively retains metadata; partial outputs remain.
// Choose before catalog writes; this verifies preservation evidence, not archive parsing or final legal material.
func (a ToolkitCatalogRegistrationActivities) VerifyToolkitCatalogMetadata(ctx context.Context, input ToolkitCatalogRegistrationInput) (ToolkitCatalogMetadataResult, error) {
	if err := validateToolkitCatalogInput(input); err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	output, err := resolveFileRef(input.MetadataRef, root, false)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	pulse := func() error { return a.catalogPulse(ctx, "verify-catalog-metadata") }
	if err = pulse(); err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	var raw []byte
	if strings.HasPrefix(string(input.ResultRef), "file://") {
		path, e := resolveFileRef(input.ResultRef, root, false)
		if e != nil {
			return ToolkitCatalogMetadataResult{}, e
		}
		raw, _, err = toolkitReadBoundedJSON(path, toolkitCatalogMetadataLimit, pulse)
	} else {
		raw, err = a.readCatalogRemote(ctx, input.ResultRef, input.ResultVersionID, toolkitCatalogMetadataLimit)
	}
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	if digestBytes(raw) != input.ResultSHA256 {
		return ToolkitCatalogMetadataResult{}, errors.New("preservation result digest mismatch")
	}
	var result toolkitCatalogPreservationEnvelope
	if err = decodeToolkitCatalogJSON(raw, &result); err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	batch := ToolkitRecoveryCatalogBatch{Schema: ToolkitRecoveryCatalogSchema, Request: input, InventoryRef: result.InventoryRef, InventorySHA256: result.InventorySHA256, Packages: result.Packages, ResultProvenance: result.Provenance}
	if result.StorageMode != ToolkitPackagePreservationModeVersioned {
		return ToolkitCatalogMetadataResult{}, errors.New("catalog requires versioned recovery result")
	}
	encoded, digest, err := ToolkitRecoveryCatalogCanonical(batch)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	for _, p := range batch.Packages {
		if err = pulse(); err != nil {
			return ToolkitCatalogMetadataResult{}, err
		}
		receiptRaw, e := a.readCatalogRemote(ctx, p.ReceiptRef, p.ReceiptVersionID, 16<<10)
		if e != nil {
			return ToolkitCatalogMetadataResult{}, e
		}
		verificationRaw, e := a.readCatalogRemote(ctx, p.VerificationRef, p.VerificationVersionID, 16<<10)
		if e != nil {
			return ToolkitCatalogMetadataResult{}, e
		}
		if int64(len(receiptRaw)) != p.ReceiptBytes || digestBytes(receiptRaw) != p.ReceiptSHA256 || int64(len(verificationRaw)) != p.VerificationReceiptBytes || digestBytes(verificationRaw) != p.VerificationReceiptSHA256 {
			return ToolkitCatalogMetadataResult{}, errors.New("exact-version preservation evidence hash/size mismatch")
		}
		var receipt toolkitPreservationObjectReceipt
		var verified toolkitPreservationVerificationReceipt
		if decodeToolkitCatalogJSON(receiptRaw, &receipt) != nil || decodeToolkitCatalogJSON(verificationRaw, &verified) != nil {
			return ToolkitCatalogMetadataResult{}, errors.New("malformed preservation evidence")
		}
		wantReceipt := toolkitPreservationObjectReceipt{Schema: "toolkit-package-preservation/v1", StorageMode: p.StorageMode, InventoryRef: batch.InventoryRef, InventorySHA256: batch.InventorySHA256, PackageName: p.PackageName, OriginalRef: p.OriginalRef, PreservedRef: p.PreservedRef, SHA256: p.SHA256, Bytes: p.Bytes, ArchiveVersionID: p.ArchiveVersionID}
		wantVerified := toolkitPreservationVerificationReceipt{Schema: "toolkit-package-verification/v1", StorageMode: p.StorageMode, InventoryRef: batch.InventoryRef, InventorySHA256: batch.InventorySHA256, PackageName: p.PackageName, OriginalRef: p.OriginalRef, PreservedRef: p.PreservedRef, ReceiptRef: p.ReceiptRef, SHA256: p.SHA256, Bytes: p.Bytes, ArchiveVersionID: p.ArchiveVersionID, PreservationReceiptVersionID: p.ReceiptVersionID, PreservationReceiptSHA256: p.ReceiptSHA256, PreservationReceiptBytes: p.ReceiptBytes}
		if receipt != wantReceipt || verified != wantVerified {
			return ToolkitCatalogMetadataResult{}, errors.New("preservation evidence identities disagree")
		}
		if err = a.checkCatalogArchiveVersion(ctx, p.PreservedRef, p.ArchiveVersionID); err != nil {
			return ToolkitCatalogMetadataResult{}, err
		}
	}
	if err = pulse(); err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	f, e := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(e) {
		old, _, readErr := toolkitReadBoundedJSON(output, toolkitCatalogMetadataLimit, pulse)
		if readErr != nil || !bytes.Equal(old, encoded) {
			return ToolkitCatalogMetadataResult{}, errors.New("metadata replay collision or partial output")
		}
	} else if e != nil {
		return ToolkitCatalogMetadataResult{}, e
	} else {
		n, writeErr := f.Write(encoded)
		if writeErr == nil && n != len(encoded) {
			writeErr = io.ErrShortWrite
		}
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil {
			return ToolkitCatalogMetadataResult{}, writeErr
		}
		if closeErr != nil {
			return ToolkitCatalogMetadataResult{}, closeErr
		}
	}
	return ToolkitCatalogMetadataResult{OperationID: input.OperationID, MetadataRef: input.MetadataRef, MetadataSHA256: digest, Count: 15}, nil
}

// loadToolkitCatalogBatch admits a mounted verified metadata snapshot for one independent downstream Activity.
// Inputs: digest-pinned metadata coordinates. Outputs: validated canonical batch. Effects: bounded local metadata read only.
// Choose instead of rereading archive bodies or accepting workflow-supplied row arrays.
func (a ToolkitCatalogRegistrationActivities) loadToolkitCatalogBatch(ctx context.Context, input ToolkitCatalogMetadataResult) (ToolkitRecoveryCatalogBatch, error) {
	var batch ToolkitRecoveryCatalogBatch
	if input.Count != 15 || !safePreservationNamespace(input.OperationID) || !validSHA256(input.MetadataSHA256) || len(input.MetadataRef) > 4096 {
		return batch, errors.New("invalid verified metadata coordinates")
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return batch, err
	}
	path, err := resolveFileRef(input.MetadataRef, root, false)
	if err != nil {
		return batch, err
	}
	raw, digest, err := toolkitReadBoundedJSON(path, toolkitCatalogMetadataLimit, func() error { return a.catalogPulse(ctx, "read-verified-metadata") })
	if err != nil {
		return batch, err
	}
	if digest != input.MetadataSHA256 {
		return batch, errors.New("verified metadata digest changed")
	}
	if err = decodeToolkitCatalogJSON(raw, &batch); err != nil {
		return batch, err
	}
	_, canonical, err := ToolkitRecoveryCatalogCanonical(batch)
	if err != nil {
		return batch, err
	}
	if canonical != input.MetadataSHA256 || batch.Request.OperationID != input.OperationID || batch.Request.MetadataRef != input.MetadataRef {
		return batch, errors.New("verified metadata identity mismatch")
	}
	return batch, nil
}

// RegisterToolkitCatalog writes only the already verified bounded metadata into the separate recovery ledger.
// Inputs: metadata ref/hash/count. Outputs: error or successful atomic registration. Effects: repository metadata writes only.
// Choose after verification; retries compare complete operation/occurrence identities and never overwrite collisions.
func (a ToolkitCatalogRegistrationActivities) RegisterToolkitCatalog(ctx context.Context, input ToolkitCatalogMetadataResult) error {
	batch, err := a.loadToolkitCatalogBatch(ctx, input)
	if err != nil {
		return err
	}
	if a.Catalog == nil {
		return errors.New("Case Bible recovery writer missing")
	}
	if err = a.catalogPulse(ctx, "register-recovery-catalog"); err != nil {
		return err
	}
	return a.Catalog.RegisterToolkitRecovery(ctx, batch)
}

// ReadbackToolkitCatalog independently compares all fifteen catalog occurrences against pinned metadata.
// Inputs: verified metadata coordinates. Outputs: the same coordinates only after count/full-metadata hash equality.
// Effects: separate database read, no writes. Choose after registration; does not claim projection refresh or final-material approval.
func (a ToolkitCatalogRegistrationActivities) ReadbackToolkitCatalog(ctx context.Context, input ToolkitCatalogMetadataResult) (ToolkitCatalogMetadataResult, error) {
	batch, err := a.loadToolkitCatalogBatch(ctx, input)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	if a.Catalog == nil {
		return ToolkitCatalogMetadataResult{}, errors.New("Case Bible recovery reader missing")
	}
	if err = a.catalogPulse(ctx, "independent-catalog-readback"); err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	actual, err := a.Catalog.ReadToolkitRecovery(ctx, batch.Request.OperationID)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	_, digest, err := ToolkitRecoveryCatalogCanonical(actual)
	if err != nil {
		return ToolkitCatalogMetadataResult{}, err
	}
	if digest != input.MetadataSHA256 {
		return ToolkitCatalogMetadataResult{}, fmt.Errorf("independent catalog metadata hash/count mismatch")
	}
	return input, nil
}
