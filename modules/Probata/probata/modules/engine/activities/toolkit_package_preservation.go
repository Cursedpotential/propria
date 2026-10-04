// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
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
	toolkitPreservationDestination               = "b2://salem-data/consignatio/casevault/recovery/library-sources/"
	toolkitPreservationMaxPackages               = 15
	toolkitPreservationMaxArchiveBytes           = int64(8 << 30)
	toolkitPreservationMaxSinglePut              = int64(5_000_000_000)
	toolkitPreservationMaxNamespaceBytes         = 64
	ToolkitPackagePreservationCopyActivityName   = "toolkit_package_preservation_copy_activity"
	ToolkitPackagePreservationVerifyActivityName = "toolkit_package_preservation_verify_activity"
	ToolkitPackagePreservationWorkflowName       = "ToolkitPackagePreservationWorkflow"
	toolkitPreservationHeartbeatEvery            = 15 * time.Second
)

// ToolkitPackagePreservationInput names a bounded explicit package selection from a digest-pinned inventory.
// Inputs: an inventory reference under the configured root, lowercase receipt digest, bounded names, operation namespace and byte budgets.
// Outputs: durable package receipt references and byte identities only.
// Side effects: schedules bounded copy and independent readback Activities; never changes local sources.
// Choose beside ToolkitPackageInventoryWorkflow when intact whole ZIP archives must be retained remotely.
type ToolkitPackagePreservationInput struct {
	InventoryRef       proffer.Ref `json:"inventory_ref"`
	InventorySHA256    string      `json:"inventory_sha256"`
	OperationNamespace string      `json:"operation_namespace,omitempty"`
	PackageNames       []string    `json:"package_names"`
	MaxArchiveBytes    int64       `json:"max_archive_bytes"`
	MaxSinglePutBytes  int64       `json:"max_single_put_bytes"`
}

// ToolkitPackagePreservationResult contains references and byte identities for the complete bounded set.
// Inputs: a validated preservation request; outputs: at most fifteen compact per-package receipts.
// Side effects: none in the Workflow itself; child Activities perform storage operations.
// Choose instead of passing archive bodies or package member inventories through Temporal history.
type ToolkitPackagePreservationResult struct {
	InventoryRef    proffer.Ref                         `json:"inventory_ref"`
	InventorySHA256 string                              `json:"inventory_sha256"`
	Packages        []ToolkitPackagePreservationReceipt `json:"packages"`
}

// ToolkitPackagePreservationReceipt is a compact durable reference to one preserved whole archive.
// Inputs: package identity and source/remote byte checks; outputs: archive and receipt object references.
// Side effects: none; this record contains no archive or member content.
// Choose as a bounded workflow result instead of returning corpus data.
type ToolkitPackagePreservationReceipt struct {
	PackageName     string      `json:"package_name"`
	SHA256          string      `json:"sha256"`
	Bytes           int64       `json:"bytes"`
	OriginalRef     proffer.Ref `json:"original_ref"`
	PreservedRef    proffer.Ref `json:"preserved_ref"`
	ReceiptRef      proffer.Ref `json:"receipt_ref"`
	VerificationRef proffer.Ref `json:"verification_ref"`
}

// ToolkitPackagePreservationHeartbeat reports bounded liveness without archive data.
// Inputs: package name and current phase; outputs: compact Temporal progress.
// Side effects: records a heartbeat when called by a worker.
// Choose over carrying byte chunks or member-level details in progress state.
type ToolkitPackagePreservationHeartbeat struct {
	PackageName string `json:"package_name"`
	Phase       string `json:"phase"`
	Bytes       int64  `json:"bytes,omitempty"`
}

// ToolkitPackagePreservationCopyInput identifies one archive from the pinned complete inventory.
// Inputs: exact inventory reference/digest and one selected package name.
// Outputs: one durable bounded preservation receipt.
// Side effects: reads a local archive and create-only writes its complete bytes and compact receipt remotely.
// Choose one package per Activity so retries and cancellation remain independently bounded.
type ToolkitPackagePreservationCopyInput struct {
	InventoryRef       proffer.Ref `json:"inventory_ref"`
	InventorySHA256    string      `json:"inventory_sha256"`
	OperationNamespace string      `json:"operation_namespace,omitempty"`
	PackageName        string      `json:"package_name"`
	PackageNames       []string    `json:"package_names"`
	MaxArchiveBytes    int64       `json:"max_archive_bytes"`
	MaxSinglePutBytes  int64       `json:"max_single_put_bytes"`
}

// ToolkitPackagePreservationVerifyInput independently names the archive and its expected identity.
// Inputs: inventory identity, package name, expected SHA-256/size, and destination references.
// Outputs: a compact verification reference after a fresh remote stream matches both byte measures.
// Side effects: reads the remote archive and preservation receipt only.
// Choose after copy rather than treating a successful PUT response as proof of retention.
type ToolkitPackagePreservationVerifyInput struct {
	InventoryRef       proffer.Ref `json:"inventory_ref"`
	InventorySHA256    string      `json:"inventory_sha256"`
	OperationNamespace string      `json:"operation_namespace,omitempty"`
	PackageName        string      `json:"package_name"`
	PackageNames       []string    `json:"package_names"`
	SHA256             string      `json:"sha256"`
	Bytes              int64       `json:"bytes"`
	MaxArchiveBytes    int64       `json:"max_archive_bytes"`
	MaxSinglePutBytes  int64       `json:"max_single_put_bytes"`
	OriginalRef        proffer.Ref `json:"original_ref"`
	PreservedRef       proffer.Ref `json:"preserved_ref"`
	ReceiptRef         proffer.Ref `json:"receipt_ref"`
}

// ToolkitPackagePreservationActivities preserves complete ZIP archives through the configured object-store resolver.
// Inputs: the existing object-store resolver and optional liveness callback; outputs: bounded Activity results.
// Side effects: remote create-only object writes and readback; no database or catalog writes.
// Choose this Activity group for source-archive retention, not derivation or legal validation.
type ToolkitPackagePreservationActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewToolkitPackagePreservationActivities uses a configured inventory root and the worker's existing store resolver.
// Inputs: absolute allowed inventory root and the same scheme-to-store resolver used by derivation; outputs: preservation Activities.
// Side effects: none until an Activity runs; no clients, credentials, or configuration are created here.
// Choose this constructor to reuse OBJECT_STORES_JSON resolution without changing worker setup.
// Byline: Codex · GPT-6 · 2026-10-04.
func NewToolkitPackagePreservationActivities(root string, stores func(string) (smsthreads.ObjectStore, error)) ToolkitPackagePreservationActivities {
	return ToolkitPackagePreservationActivities{
		AllowedRoot: root, Stores: stores,
		Heartbeat: func(ctx context.Context, progress ToolkitPackagePreservationHeartbeat) {
			activity.RecordHeartbeat(ctx, progress)
		},
	}
}

// ToolkitPackagePreservationWorkflow retains and independently verifies each explicitly selected intact archive.
// Inputs: an authenticated inventory identity, one to fifteen package names, operation namespace and positive byte budgets.
// Outputs: bounded archive and verification receipt references, or a visible failure.
// Side effects: schedules sequential copy/readback Activities; it does not remove or modify source files.
// Choose beside the inventory workflow when byte-preserving whole-package retention is required.
// Byline: Codex · GPT-6 · 2026-10-04.
func ToolkitPackagePreservationWorkflow(ctx workflow.Context, input ToolkitPackagePreservationInput) (ToolkitPackagePreservationResult, error) {
	if err := validateToolkitPackageSelection(input); err != nil {
		return ToolkitPackagePreservationResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitPackagePreservationInvalid", err)
	}
	input.PackageNames = sortToolkitPackageNames(input.PackageNames)
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour, ScheduleToCloseTimeout: 72 * time.Hour,
		HeartbeatTimeout: time.Minute, WaitForCancellation: true,
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	result := ToolkitPackagePreservationResult{InventoryRef: input.InventoryRef, InventorySHA256: input.InventorySHA256,
		Packages: make([]ToolkitPackagePreservationReceipt, 0, len(input.PackageNames))}
	for _, name := range input.PackageNames {
		var copied ToolkitPackagePreservationReceipt
		copyInput := ToolkitPackagePreservationCopyInput{
			InventoryRef: input.InventoryRef, InventorySHA256: input.InventorySHA256,
			OperationNamespace: input.OperationNamespace, PackageName: name,
			PackageNames:    input.PackageNames,
			MaxArchiveBytes: input.MaxArchiveBytes, MaxSinglePutBytes: input.MaxSinglePutBytes,
		}
		if err := workflow.ExecuteActivity(ctx, ToolkitPackagePreservationCopyActivityName, copyInput).Get(ctx, &copied); err != nil {
			return result, err
		}
		verifyInput := ToolkitPackagePreservationVerifyInput{
			InventoryRef: input.InventoryRef, InventorySHA256: input.InventorySHA256,
			OperationNamespace: input.OperationNamespace,
			PackageName:        copied.PackageName, PackageNames: input.PackageNames, SHA256: copied.SHA256, Bytes: copied.Bytes,
			MaxArchiveBytes: input.MaxArchiveBytes, MaxSinglePutBytes: input.MaxSinglePutBytes,
			OriginalRef: copied.OriginalRef, PreservedRef: copied.PreservedRef, ReceiptRef: copied.ReceiptRef,
		}
		var verified proffer.Ref
		if err := workflow.ExecuteActivity(ctx, ToolkitPackagePreservationVerifyActivityName, verifyInput).Get(ctx, &verified); err != nil {
			return result, err
		}
		copied.VerificationRef = verified
		result.Packages = append(result.Packages, copied)
	}
	return result, nil
}

// CopyToolkitPackagePreservation streams one complete source ZIP into its digest-scoped recovery namespace.
// Inputs: one package selection from the pinned inventory; outputs: destination and durable receipt references.
// Side effects: reads the local ZIP and uses conditional create-only writes for the archive and receipt.
// Choose before VerifyToolkitPackagePreservation; a plain Put fallback is deliberately unavailable.
// Byline: Codex · GPT-6 · 2026-10-04.
func (a ToolkitPackagePreservationActivities) CopyToolkitPackagePreservation(ctx context.Context, input ToolkitPackagePreservationCopyInput) (ToolkitPackagePreservationReceipt, error) {
	fail := func(err error) (ToolkitPackagePreservationReceipt, error) {
		if ctx.Err() != nil {
			return ToolkitPackagePreservationReceipt{}, ctx.Err()
		}
		return ToolkitPackagePreservationReceipt{}, temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitPackagePreservationCopyFailed", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := validateToolkitCopyInput(input); err != nil {
		return fail(err)
	}
	if a.Stores == nil {
		return fail(errors.New("existing object-store resolver is required"))
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return fail(fmt.Errorf("resolve configured inventory root: %w", err))
	}
	receipt, err := readPinnedToolkitInventory(ctx, input.InventoryRef, input.InventorySHA256, root, a.Heartbeat)
	if err != nil {
		return fail(err)
	}
	entry, ok := toolkitPackageByName(receipt, input.PackageName)
	if err = validateSelectedInventoryBudget(receipt, root, input.PackageNames, input.MaxArchiveBytes, input.MaxSinglePutBytes); err != nil {
		return fail(err)
	}
	if !ok || !entry.Complete || entry.SHA256 == nil || !validSHA256(*entry.SHA256) {
		return fail(fmt.Errorf("package %q is missing, incomplete, or unhashed in the pinned inventory", input.PackageName))
	}
	source, err := toolkitPackageSource(receipt, root, input.PackageName)
	if err != nil {
		return fail(err)
	}
	file, err := os.Open(source)
	if err != nil {
		return fail(fmt.Errorf("open original archive: %w", err))
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > input.MaxArchiveBytes || before.Size() > input.MaxSinglePutBytes {
		return fail(errors.New("source archive is not regular or exceeds the configured archive/single-PUT byte budget"))
	}
	sha, count, err := hashPreservationSource(ctx, file, a.Heartbeat, input.PackageName)
	if err != nil {
		return fail(err)
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || count != before.Size() {
		return fail(errors.New("source archive changed or size differed while hashing"))
	}
	if sha != *entry.SHA256 {
		return fail(fmt.Errorf("source archive SHA-256 differs from pinned inventory for %q", input.PackageName))
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return fail(fmt.Errorf("rewind source archive: %w", err))
	}
	destination, receiptRef := toolkitPreservationRefs(input.OperationNamespace, input.InventorySHA256, input.PackageName)
	store, err := a.Stores("b2")
	if err != nil {
		return fail(fmt.Errorf("resolve configured b2 store: %w", err))
	}
	bucket, key, err := toolkitObjectCoordinates(destination)
	if err != nil {
		return fail(err)
	}
	if err = createToolkitObjectIfAbsent(ctx, store, bucket, key, file, before.Size(), sha, "application/zip", a.Heartbeat, input.PackageName); err != nil {
		return fail(fmt.Errorf("create-only archive write: %w", err))
	}
	if err = verifyToolkitRemoteObject(ctx, store, bucket, key, sha, before.Size(), a.Heartbeat, input.PackageName); err != nil {
		return fail(fmt.Errorf("archive existence readback: %w", err))
	}
	archiveReceipt := toolkitPreservationObjectReceipt{
		Schema: "toolkit-package-preservation/v1", InventoryRef: input.InventoryRef,
		InventorySHA256: input.InventorySHA256, PackageName: input.PackageName,
		OriginalRef: toolkitFileRef(source), PreservedRef: destination,
		SHA256: sha, Bytes: before.Size(),
	}
	receiptBytes, err := json.Marshal(archiveReceipt)
	if err != nil {
		return fail(err)
	}
	receiptBucket, receiptKey, err := toolkitObjectCoordinates(receiptRef)
	if err != nil {
		return fail(err)
	}
	if err = createToolkitObjectIfAbsent(ctx, store, receiptBucket, receiptKey, strings.NewReader(string(receiptBytes)), int64(len(receiptBytes)), digestBytes(receiptBytes), "application/json", a.Heartbeat, input.PackageName); err != nil {
		return fail(fmt.Errorf("create-only preservation receipt write: %w", err))
	}
	if err = verifyToolkitRemoteObject(ctx, store, receiptBucket, receiptKey, digestBytes(receiptBytes), int64(len(receiptBytes)), a.Heartbeat, input.PackageName); err != nil {
		return fail(fmt.Errorf("preservation receipt readback: %w", err))
	}
	return ToolkitPackagePreservationReceipt{PackageName: input.PackageName, SHA256: sha, Bytes: before.Size(),
		OriginalRef: archiveReceipt.OriginalRef, PreservedRef: destination, ReceiptRef: receiptRef}, nil
}

// VerifyToolkitPackagePreservation independently streams the remote archive and validates its durable receipt.
// Inputs: expected package identity and object references returned by the copy Activity.
// Outputs: the compact durable verification-receipt reference after SHA-256 and size match.
// Side effects: remote reads and one create-only verification receipt write; it never repairs archive bytes.
// Choose after copy so provider acknowledgement is not treated as independent byte verification.
// Byline: Codex · GPT-6 · 2026-10-04.
func (a ToolkitPackagePreservationActivities) VerifyToolkitPackagePreservation(ctx context.Context, input ToolkitPackagePreservationVerifyInput) (proffer.Ref, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if a.Stores == nil {
		return "", errors.New("existing object-store resolver is required")
	}
	if err := validateToolkitVerifyInput(input); err != nil {
		return "", temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitPackagePreservationVerifyInvalid", err)
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return "", fmt.Errorf("resolve configured inventory root: %w", err)
	}
	inventory, err := readPinnedToolkitInventory(ctx, input.InventoryRef, input.InventorySHA256, root, a.Heartbeat)
	if err != nil {
		return "", err
	}
	entry, ok := toolkitPackageByName(inventory, input.PackageName)
	if !ok || !entry.Complete || entry.SHA256 == nil || *entry.SHA256 != input.SHA256 {
		return "", errors.New("verification digest does not match the complete pinned inventory entry")
	}
	if err = validateSelectedInventoryBudget(inventory, root, input.PackageNames, input.MaxArchiveBytes, input.MaxSinglePutBytes); err != nil {
		return "", err
	}
	sourcePath, err := toolkitPackageSource(inventory, root, input.PackageName)
	if err != nil {
		return "", err
	}
	if input.OriginalRef != toolkitFileRef(sourcePath) {
		return "", errors.New("verification original reference does not match the pinned inventory source")
	}
	store, err := a.Stores("b2")
	if err != nil {
		return "", fmt.Errorf("resolve configured b2 store: %w", err)
	}
	bucket, key, err := toolkitObjectCoordinates(input.PreservedRef)
	if err != nil {
		return "", err
	}
	if err = verifyToolkitRemoteObject(ctx, store, bucket, key, input.SHA256, input.Bytes, a.Heartbeat, input.PackageName); err != nil {
		return "", fmt.Errorf("independent remote archive verification: %w", err)
	}
	receiptBucket, receiptKey, err := toolkitObjectCoordinates(input.ReceiptRef)
	if err != nil {
		return "", err
	}
	raw, err := readToolkitRemoteBounded(ctx, store, receiptBucket, receiptKey, 16<<10)
	if err != nil {
		return "", fmt.Errorf("read preservation receipt: %w", err)
	}
	var receipt toolkitPreservationObjectReceipt
	if err = json.Unmarshal(raw, &receipt); err != nil {
		return "", err
	}
	if receipt.Schema != "toolkit-package-preservation/v1" || receipt.InventoryRef != input.InventoryRef ||
		receipt.InventorySHA256 != input.InventorySHA256 || receipt.PackageName != input.PackageName ||
		receipt.SHA256 != input.SHA256 || receipt.Bytes != input.Bytes || receipt.OriginalRef != input.OriginalRef ||
		receipt.PreservedRef != input.PreservedRef {
		return "", errors.New("remote preservation receipt does not match the requested archive identity")
	}
	verification := toolkitPreservationVerificationReceipt{
		Schema: "toolkit-package-verification/v1", InventoryRef: input.InventoryRef,
		InventorySHA256: input.InventorySHA256, PackageName: input.PackageName,
		OriginalRef: input.OriginalRef, PreservedRef: input.PreservedRef,
		ReceiptRef: input.ReceiptRef, SHA256: input.SHA256, Bytes: input.Bytes,
	}
	verificationBytes, err := json.Marshal(verification)
	if err != nil {
		return "", err
	}
	verificationRef := toolkitVerificationRef(input)
	verifyBucket, verifyKey, err := toolkitObjectCoordinates(verificationRef)
	if err != nil {
		return "", err
	}
	if err = createToolkitObjectIfAbsent(ctx, store, verifyBucket, verifyKey,
		strings.NewReader(string(verificationBytes)), int64(len(verificationBytes)), digestBytes(verificationBytes), "application/json", a.Heartbeat, input.PackageName); err != nil {
		return "", fmt.Errorf("create-only verification receipt: %w", err)
	}
	if err = verifyToolkitRemoteObject(ctx, store, verifyBucket, verifyKey,
		digestBytes(verificationBytes), int64(len(verificationBytes)), nil, ""); err != nil {
		return "", fmt.Errorf("verification receipt readback: %w", err)
	}
	return verificationRef, nil
}

type toolkitPreservationObjectReceipt struct {
	Schema          string      `json:"schema"`
	InventoryRef    proffer.Ref `json:"inventory_ref"`
	InventorySHA256 string      `json:"inventory_sha256"`
	PackageName     string      `json:"package_name"`
	OriginalRef     proffer.Ref `json:"original_ref"`
	PreservedRef    proffer.Ref `json:"preserved_ref"`
	SHA256          string      `json:"sha256"`
	Bytes           int64       `json:"bytes"`
}

// toolkitPreservationVerificationReceipt durably records only the verified archive identity and its references.
// Inputs: a completed remote readback and matching preservation receipt; outputs: bounded verification metadata.
// Side effects: create-only written after independent archive SHA-256 and size verification.
// Choose instead of claiming legal accuracy, catalog registration, or database state.
type toolkitPreservationVerificationReceipt struct {
	Schema          string      `json:"schema"`
	InventoryRef    proffer.Ref `json:"inventory_ref"`
	InventorySHA256 string      `json:"inventory_sha256"`
	PackageName     string      `json:"package_name"`
	OriginalRef     proffer.Ref `json:"original_ref"`
	PreservedRef    proffer.Ref `json:"preserved_ref"`
	ReceiptRef      proffer.Ref `json:"receipt_ref"`
	SHA256          string      `json:"sha256"`
	Bytes           int64       `json:"bytes"`
}

// validateToolkitPackageSelection accepts a bounded explicit subset of an authenticated inventory.
// Inputs: inventory file reference, lowercase digest, one to fifteen safe ZIP basenames, namespace and byte ceilings; outputs: nil only for valid unique selections.
// Side effects: none. Choose before scheduling any copy so omissions cannot become silent successes.
// Byline: Codex · GPT-6 · 2026-10-04.
func validateToolkitPackageSelection(input ToolkitPackagePreservationInput) error {
	if input.InventoryRef == "" || !validSHA256(input.InventorySHA256) {
		return errors.New("preservation requires a file inventory reference and lowercase SHA-256")
	}
	if len(input.PackageNames) == 0 || len(input.PackageNames) > toolkitPreservationMaxPackages {
		return fmt.Errorf("preservation requires an explicit selection of 1 to %d package names", toolkitPreservationMaxPackages)
	}
	if err := validatePreservationBudgets(input.MaxArchiveBytes, input.MaxSinglePutBytes); err != nil {
		return err
	}
	if !safePreservationNamespace(input.OperationNamespace) {
		return errors.New("operation namespace must be empty or a safe bounded path segment")
	}
	seen := make(map[string]bool, len(input.PackageNames))
	for _, name := range input.PackageNames {
		if !safeToolkitPackageName(name) || seen[name] {
			return fmt.Errorf("package selection contains an unsafe or duplicate name %q", name)
		}
		seen[name] = true
	}
	return nil
}

// validateToolkitCopyInput rejects malformed inventory identity, namespace, budgets and selections before source access.
// Inputs: one package copy request; outputs: nil only for valid bounded identities.
// Side effects: none. Choose at each Activity boundary because Temporal can invoke Activities directly.
// Byline: Codex · GPT-6 · 2026-10-04.
func validateToolkitCopyInput(input ToolkitPackagePreservationCopyInput) error {
	if input.InventoryRef == "" || !validSHA256(input.InventorySHA256) || !safeToolkitPackageName(input.PackageName) || !safePreservationNamespace(input.OperationNamespace) {
		return errors.New("copy requires a valid inventory digest, package name, and operation namespace")
	}
	if len(input.PackageNames) == 0 || len(input.PackageNames) > toolkitPreservationMaxPackages || !containsToolkitPackage(input.PackageNames, input.PackageName) {
		return errors.New("copy requires its package in an explicit selection of at most fifteen names")
	}
	return validatePreservationBudgets(input.MaxArchiveBytes, input.MaxSinglePutBytes)
}

// validateToolkitVerifyInput restricts verification to the fixed recovery/source namespace.
// Inputs: the expected archive identity and references; outputs: nil only when every identity field is bounded and pinned.
// Side effects: none. Choose before any remote read to avoid turning this verifier into a general object reader.
// Byline: Codex · GPT-6 · 2026-10-04.
func validateToolkitVerifyInput(input ToolkitPackagePreservationVerifyInput) error {
	if input.InventoryRef == "" || !validSHA256(input.InventorySHA256) || !safeToolkitPackageName(input.PackageName) || !validSHA256(input.SHA256) || input.Bytes < 0 ||
		len(input.PackageNames) == 0 || len(input.PackageNames) > toolkitPreservationMaxPackages || !containsToolkitPackage(input.PackageNames, input.PackageName) ||
		!safePreservationNamespace(input.OperationNamespace) || validatePreservationBudgets(input.MaxArchiveBytes, input.MaxSinglePutBytes) != nil ||
		input.Bytes > input.MaxArchiveBytes || input.Bytes > input.MaxSinglePutBytes {
		return errors.New("verification identity is invalid or outside configured preservation bounds")
	}
	wantArchive, wantReceipt := toolkitPreservationRefs(input.OperationNamespace, input.InventorySHA256, input.PackageName)
	if input.PreservedRef != wantArchive || input.ReceiptRef != wantReceipt || input.OriginalRef == "" {
		return errors.New("verification references do not match the fixed recovery/source namespace")
	}
	return nil
}

// readPinnedToolkitInventory reads and authenticates a bounded receipt beneath the configured root.
// Inputs: file reference, caller-pinned lowercase SHA-256, canonical allowed root and optional heartbeat; outputs: a validated inventory receipt.
// Side effects: reads at most 32 MiB; no source bytes are changed.
// Choose instead of trusting workflow-supplied package digests or names.
// Byline: Codex · GPT-6 · 2026-10-04.
func readPinnedToolkitInventory(ctx context.Context, ref proffer.Ref, expectedSHA, allowedRoot string, heartbeat func(context.Context, ToolkitPackagePreservationHeartbeat)) (toolkitReceipt, error) {
	var receipt toolkitReceipt
	if !validSHA256(expectedSHA) {
		return receipt, errors.New("inventory expected SHA-256 must be lowercase hexadecimal")
	}
	path, err := resolveFileRef(ref, allowedRoot, false)
	if err != nil {
		return receipt, fmt.Errorf("resolve inventory receipt under TOOLKIT_INVENTORY_ROOT: %w", err)
	}
	lastPulse := time.Now()
	pulse := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if heartbeat != nil && time.Since(lastPulse) >= toolkitPreservationHeartbeatEvery {
			heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "reading-inventory"})
			lastPulse = time.Now()
		}
		return nil
	}
	raw, digest, err := readPreservationBoundedJSON(path, toolkitInventoryMaxReceiptBytes, pulse)
	if err != nil {
		return receipt, err
	}
	if digest != expectedSHA {
		return receipt, errors.New("pinned inventory receipt SHA-256 mismatch")
	}
	if err = json.Unmarshal(raw, &receipt); err != nil {
		return receipt, fmt.Errorf("decode pinned inventory: %w", err)
	}
	if receipt.Schema != "toolkit-package-inventory/v1" || len(receipt.Packages) == 0 || len(receipt.Packages) > toolkitInventoryMaxPackages {
		return receipt, errors.New("inventory schema or package count exceeds the bounded inventory contract")
	}
	return receipt, nil
}

// toolkitPackageSource resolves one inventory package basename under its recorded source directory.
// Inputs: authenticated receipt and package name; outputs: a canonical regular-file path beneath the recorded source directory.
// Side effects: reads filesystem metadata only. Choose rather than accepting source paths from the workflow request.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitPackageSource(receipt toolkitReceipt, allowedRoot, name string) (string, error) {
	root, err := resolveFileRef(receipt.Request.SourceRef, allowedRoot, true)
	if err != nil {
		return "", fmt.Errorf("resolve pinned inventory source directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("pinned inventory source is not an existing directory")
	}
	path := filepath.Join(root, name)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve source archive: %w", err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative != name {
		return "", errors.New("archive path escaped the pinned inventory source directory")
	}
	return resolved, nil
}

// toolkitPackageByName finds exactly one entry and refuses duplicate names in a pinned receipt.
// Inputs: authenticated inventory and safe package basename; outputs: entry/presence pair.
// Side effects: none. Choose over map-based lookup so duplicate receipt entries remain visible.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitPackageByName(receipt toolkitReceipt, name string) (toolkitPackage, bool) {
	var found toolkitPackage
	count := 0
	for _, item := range receipt.Packages {
		if item.Package == name {
			found = item
			count++
		}
	}
	return found, count == 1
}

// validateSelectedInventoryBudget preflights every selected source size before any archive upload.
// Inputs: authenticated receipt, unique bounded package names, and aggregate byte ceiling; outputs: nil only when each source is present and aggregate bytes fit.
// Side effects: filesystem metadata reads only; no source contents are opened or changed.
// Choose before the first PUT so the workflow cannot partially exceed its configured total archive budget.
// Byline: Codex · GPT-6 · 2026-10-04.
func validateSelectedInventoryBudget(receipt toolkitReceipt, allowedRoot string, names []string, maxArchiveBytes, maxSinglePutBytes int64) error {
	_, err := resolveFileRef(receipt.Request.SourceRef, allowedRoot, true)
	if err != nil {
		return fmt.Errorf("resolve inventory source for budget preflight: %w", err)
	}
	var total int64
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if !safeToolkitPackageName(name) || seen[name] {
			return fmt.Errorf("invalid or duplicate selected package %q", name)
		}
		seen[name] = true
		entry, ok := toolkitPackageByName(receipt, name)
		if !ok || !entry.Complete || entry.SHA256 == nil || !validSHA256(*entry.SHA256) {
			return fmt.Errorf("selected package %q is missing, incomplete, or unhashed in the authenticated inventory", name)
		}
		sourcePath, sourceErr := toolkitPackageSource(receipt, allowedRoot, name)
		if sourceErr != nil {
			return fmt.Errorf("resolve selected package %q: %w", name, sourceErr)
		}
		info, statErr := os.Stat(sourcePath)
		if statErr != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxSinglePutBytes {
			return fmt.Errorf("selected package %q is unavailable or exceeds the single-PUT maximum", name)
		}
		if total > maxArchiveBytes-info.Size() {
			return errors.New("selected archive bytes exceed configured max_archive_bytes")
		}
		total += info.Size()
	}
	return nil
}

// containsToolkitPackage checks membership in one small explicit selection.
// Inputs: selected package names and one candidate; outputs: true only for an exact name match.
// Side effects: none. Choose at Activity boundaries to prevent an unselected archive from being processed.
// Byline: Codex · GPT-6 · 2026-10-04.
func containsToolkitPackage(names []string, candidate string) bool {
	for _, name := range names {
		if name == candidate {
			return true
		}
	}
	return false
}

// validatePreservationBudgets enforces positive caller budgets within implementation and B2 single-PUT ceilings.
// Inputs: aggregate archive ceiling and per-object single-PUT ceiling; outputs: nil only for positive bounded values.
// Side effects: none. Choose before inventory reads or remote store access.
// Byline: Codex · GPT-6 · 2026-10-04.
func validatePreservationBudgets(maxArchiveBytes, maxSinglePutBytes int64) error {
	if maxArchiveBytes <= 0 || maxArchiveBytes > toolkitPreservationMaxArchiveBytes || maxSinglePutBytes <= 0 || maxSinglePutBytes > toolkitPreservationMaxSinglePut {
		return errors.New("max_archive_bytes and max_single_put_bytes must be positive and within preservation limits")
	}
	return nil
}

// safePreservationNamespace validates an optional single path segment used to isolate recovery operations.
// Inputs: caller-selected operation namespace; outputs: true for empty default or one safe bounded segment.
// Side effects: none. Choose before deriving destination keys so request text cannot escape the fixed Casevault root.
// Byline: Codex · GPT-6 · 2026-10-04.
func safePreservationNamespace(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > toolkitPreservationMaxNamespaceBytes || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// readPreservationBoundedJSON streams one regular JSON receipt under a byte ceiling while hashing it.
// Inputs: validated local file path, positive maximum, and cancellation/liveness pulse; outputs: bounded bytes and lowercase digest.
// Side effects: reads the receipt only and rechecks file identity after opening.
// Choose this branch-local equivalent when toolkitReadBoundedJSON is absent from the checked-out integration base.
// Byline: Codex · GPT-6 · 2026-10-04.
func readPreservationBoundedJSON(path string, max int64, pulse func() error) ([]byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > max {
		return nil, "", errors.New("inventory JSON is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(info, opened) {
		return nil, "", errors.New("inventory receipt changed while opening")
	}
	var data bytes.Buffer
	digest, _, err := streamToolkitDigest(io.TeeReader(f, &data), max, pulse)
	if err != nil {
		return nil, "", err
	}
	return data.Bytes(), digest, nil
}

// toolkitPreservationRefs derives immutable archive/receipt keys from the pinned inventory digest and basename.
// Inputs: a validated inventory SHA-256 and package basename; outputs: fixed b2://salem-data references beneath casevault.
// Side effects: none. Choose deterministic references so retries converge on the same create-only objects.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitPreservationRefs(namespace, inventorySHA, name string) (proffer.Ref, proffer.Ref) {
	if namespace == "" {
		namespace = "inventory-" + inventorySHA[:16]
	}
	prefix := strings.TrimSuffix(toolkitPreservationDestination, "/") + "/" + namespace + "/" + inventorySHA + "/" + name
	archive := proffer.Ref(prefix)
	return archive, proffer.Ref(prefix + ".preservation.json")
}

// toolkitFileRef encodes a canonical local path as a file URI without losing spaces or Unicode.
// Inputs: canonical host path; outputs: opaque file reference used only in compact receipts.
// Side effects: none. Choose instead of joining raw paths onto a URI prefix.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitFileRef(path string) proffer.Ref {
	path = filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" {
		path = "/" + path
	}
	return proffer.Ref((&url.URL{Scheme: "file", Path: path}).String())
}

// toolkitObjectCoordinates parses only references inside the fixed B2 Casevault namespace.
// Inputs: an opaque object reference; outputs: bucket/key after strict scheme, bucket, and prefix checks.
// Side effects: none. Choose instead of letting request data select a destination bucket or namespace.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitObjectCoordinates(ref proffer.Ref) (string, string, error) {
	parsed, err := url.Parse(string(ref))
	if err != nil || parsed.Scheme != "b2" || parsed.Host != "salem-data" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.New("object reference must use the fixed salem-data B2 bucket")
	}
	key := strings.TrimPrefix(parsed.Path, "/")
	prefix := strings.TrimPrefix(strings.TrimSuffix(toolkitPreservationDestination, "/"), "b2://salem-data/") + "/"
	if !strings.HasPrefix(key, prefix) || strings.Contains(key, "..") || strings.Contains(key, "\\") {
		return "", "", errors.New("object reference escaped the fixed Casevault recovery/source namespace")
	}
	return parsed.Host, key, nil
}

// createToolkitObjectIfAbsent performs a conditional S3 create and never falls back to ObjectStore.Put.
// Inputs: resolved store, fixed bucket/key, seekable bytes, size, and content type; outputs: nil only after create or verified pre-existing content.
// Side effects: at most one conditional remote create attempt; existence races are resolved by readback verification.
// Choose over S3Store.Put because that sibling has overwrite semantics.
// Byline: Codex · GPT-6 · 2026-10-04.
func createToolkitObjectIfAbsent(ctx context.Context, store smsthreads.ObjectStore, bucket, key string, body io.ReadSeeker, size int64, expectedSHA, contentType string, heartbeat func(context.Context, ToolkitPackagePreservationHeartbeat), packageName string) error {
	if size < 0 || size > toolkitPreservationMaxSinglePut+16<<10 {
		return errors.New("conditional object size is outside the bounded preservation limit")
	}
	exists, err := store.Exists(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("check existing object: %w", err)
	}
	if exists {
		return verifyToolkitRemoteObject(ctx, store, bucket, key, expectedSHA, size, nil, "")
	}
	if _, err = body.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind conditional object body: %w", err)
	}
	trackedBody := &toolkitHeartbeatReadSeeker{Reader: body, ctx: ctx, heartbeat: heartbeat, packageName: packageName}
	if writer, supported := store.(toolkitConditionalObjectWriter); supported {
		err = writer.PutIfAbsent(ctx, bucket, key, trackedBody, size, contentType)
	} else {
		return errors.New("resolved store lacks the conditional S3 create seam; refusing unsafe Put")
	}
	if err == nil {
		return nil
	}
	// A lost response or a concurrent create is accepted only after a fresh byte-for-byte identity check.
	if verifyErr := verifyToolkitRemoteObject(ctx, store, bucket, key, expectedSHA, size, nil, ""); verifyErr == nil {
		return nil
	}
	return fmt.Errorf("conditional S3 create failed (no overwrite fallback): %w", err)
}

// toolkitConditionalObjectWriter is the narrow injectable seam for atomic create-only object writes.
// Inputs: object coordinates, seekable body, exact size and content type; outputs: error if create is refused.
// Side effects: one conditional object creation attempt and never an overwrite.
// Choose for deterministic fake stores and adapters that expose provider-enforced If-None-Match semantics.
type toolkitConditionalObjectWriter interface {
	PutIfAbsent(context.Context, string, string, io.ReadSeeker, int64, string) error
}

// toolkitHeartbeatReadSeeker wraps upload bytes so long streamed writes keep their Temporal Activity alive.
// Inputs: a seekable body, Activity context, and optional heartbeat callback; outputs: the same bytes with liveness updates.
// Side effects: reads the source and records bounded progress; it does not buffer the object.
// Choose around the conditional writer because the S3 SDK may stream for longer than the Activity heartbeat timeout.
// Byline: Codex · GPT-6 · 2026-10-04.
type toolkitHeartbeatReadSeeker struct {
	Reader      io.ReadSeeker
	ctx         context.Context
	heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
	packageName string
	bytes       int64
	nextBeat    time.Time
}

// Read forwards upload bytes and reports periodic streamed progress.
// Inputs: requested buffer; outputs: bytes from the wrapped source or its error.
// Side effects: advances the source and may record one compact Temporal heartbeat.
// Choose through the conditional writer instead of a plain file so slow uploads remain observable.
// Byline: Codex · GPT-6 · 2026-10-04.
func (r *toolkitHeartbeatReadSeeker) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytes += int64(n)
	if n > 0 && r.heartbeat != nil && (r.nextBeat.IsZero() || time.Now().After(r.nextBeat)) {
		r.heartbeat(r.ctx, ToolkitPackagePreservationHeartbeat{PackageName: r.packageName, Phase: "uploading", Bytes: r.bytes})
		r.nextBeat = time.Now().Add(toolkitPreservationHeartbeatEvery)
	}
	return n, err
}

// Seek forwards S3 SDK rewind and checksum positioning to the underlying file or small receipt reader.
// Inputs: offset and whence; outputs: resulting source position or seek error.
// Side effects: changes only the read cursor. Choose to preserve io.ReadSeeker compatibility with PutObject.
// Byline: Codex · GPT-6 · 2026-10-04.
func (r *toolkitHeartbeatReadSeeker) Seek(offset int64, whence int) (int64, error) {
	return r.Reader.Seek(offset, whence)
}

// verifyToolkitRemoteObject hashes a complete remote stream and compares exact size plus any supplied digest.
// Inputs: resolved store, fixed coordinates, expected digest/size, and optional heartbeat; outputs: nil only for a full exact readback.
// Side effects: remote reads only. Choose for idempotent reuse and post-write readback, never HEAD/ETag-only checks.
// Byline: Codex · GPT-6 · 2026-10-04.
func verifyToolkitRemoteObject(ctx context.Context, store smsthreads.ObjectStore, bucket, key, expectedSHA string, expectedBytes int64, heartbeat func(context.Context, ToolkitPackagePreservationHeartbeat), packageName string) error {
	stream, err := store.Open(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("open remote object: %w", err)
	}
	defer stream.Close()
	sha, count, err := hashPreservationSource(ctx, stream, heartbeat, packageName)
	if err != nil {
		return err
	}
	if count != expectedBytes || (expectedSHA != "" && sha != expectedSHA) {
		return fmt.Errorf("remote readback mismatch: got %d bytes SHA-256 %s; want %d bytes SHA-256 %s", count, sha, expectedBytes, expectedSHA)
	}
	return nil
}

// readToolkitRemoteBounded reads a small remote receipt with a hard byte ceiling.
// Inputs: resolved store, object coordinates, and positive maximum; outputs: complete receipt bytes or a limit/error.
// Side effects: reads one remote object only. Choose instead of loading an unbounded object into memory.
// Byline: Codex · GPT-6 · 2026-10-04.
func readToolkitRemoteBounded(ctx context.Context, store smsthreads.ObjectStore, bucket, key string, max int64) ([]byte, error) {
	if max < 1 {
		return nil, errors.New("remote receipt read limit must be positive")
	}
	stream, err := store.Open(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	data, err := io.ReadAll(io.LimitReader(stream, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("remote receipt exceeds %d-byte limit", max)
	}
	return data, nil
}

// hashPreservationSource streams bytes through SHA-256 with cancellation and periodic heartbeats.
// Inputs: reader, context and optional heartbeat callback; outputs: lowercase SHA-256 and exact byte count.
// Side effects: reads bounded 1 MiB chunks; it stores no source content.
// Choose for source prechecks and remote readback instead of buffering archives in memory.
// Byline: Codex · GPT-6 · 2026-10-04.
func hashPreservationSource(ctx context.Context, reader io.Reader, heartbeat func(context.Context, ToolkitPackagePreservationHeartbeat), packageName string) (string, int64, error) {
	hash := sha256.New()
	buffer := make([]byte, 1<<20)
	var count int64
	nextBeat := time.Now().Add(toolkitPreservationHeartbeatEvery)
	for {
		if err := ctx.Err(); err != nil {
			return "", count, err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			count += int64(n)
			if count > toolkitPreservationMaxSinglePut {
				return "", count, errors.New("stream exceeds bounded preservation size")
			}
			_, _ = hash.Write(buffer[:n])
		}
		if heartbeat != nil && time.Now().After(nextBeat) {
			heartbeat(ctx, ToolkitPackagePreservationHeartbeat{PackageName: packageName, Phase: "streaming", Bytes: count})
			nextBeat = time.Now().Add(toolkitPreservationHeartbeatEvery)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", count, readErr
		}
		if n == 0 {
			return "", count, io.ErrNoProgress
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), count, nil
}

// safeToolkitPackageName accepts one bounded lowercase ZIP basename with no path syntax.
// Inputs: a candidate inventory package name; outputs: true only for a safe flat ZIP filename.
// Side effects: none. Choose before joining any package name to a local or remote root.
// Byline: Codex · GPT-6 · 2026-10-04.
func safeToolkitPackageName(name string) bool {
	return len(name) > 4 && len(name) <= 255 && strings.HasSuffix(name, ".zip") &&
		filepath.Base(name) == name && !strings.ContainsAny(name, "/\\:") && name != ".zip" && name != "..zip"
}

// validSHA256 recognizes one lowercase 64-character SHA-256 hex digest.
// Inputs: digest text; outputs: true only for a canonical SHA-256 reference.
// Side effects: none. Choose before comparing caller-supplied digests.
// Byline: Codex · GPT-6 · 2026-10-04.
func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// digestBytes computes the SHA-256 identity of one bounded in-memory receipt.
// Inputs: receipt bytes; outputs: lowercase digest text. Side effects: none.
// Choose only for small metadata receipts, never archive bodies.
// Byline: Codex · GPT-6 · 2026-10-04.
func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// toolkitVerificationRef names the durable receipt as the compact verification result.
// Inputs: a validated verification request; outputs: the receipt reference already written by the copy Activity.
// Side effects: none. Choose instead of manufacturing a database/catalog claim.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitVerificationRef(input ToolkitPackagePreservationVerifyInput) proffer.Ref {
	return proffer.Ref(string(input.ReceiptRef) + ".verified.json")
}

// sortToolkitPackageNames returns a stable copy for explicit selection comparisons.
// Inputs: bounded package names; outputs: a sorted independent slice. Side effects: none.
// Choose to make selection validation deterministic without changing caller-owned input.
// Byline: Codex · GPT-6 · 2026-10-04.
func sortToolkitPackageNames(names []string) []string {
	copyNames := append([]string(nil), names...)
	sort.Strings(copyNames)
	return copyNames
}
