// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// toolkitPreservationMemoryStore is an in-memory object-store fixture with atomic create-only semantics.
// Inputs: none; outputs: empty store and zeroed write counters.
// Side effects: allocates only process memory. Choose for focused storage contract tests without network or fixture cleanup.
// Byline: Codex · GPT-6 · 2026-10-04.
type toolkitPreservationMemoryStore struct {
	objects           map[string][]byte
	versions          map[string]map[string][]byte
	versionedReads    []string
	versionedWrites   int
	conditionalWrites int
	plainWrites       int
}

// Open returns an independent in-memory reader for one remote object.
// Inputs: context, bucket and key; outputs: a read stream or missing-object error.
// Side effects: none. Choose to exercise the same stream-based readback path as S3Store.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	data, ok := s.objects[bucket+"/"+key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Put records an unsafe plain-write attempt so tests can prove production logic never calls it.
// Inputs: ordinary ObjectStore put arguments; outputs: explicit failure.
// Side effects: increments a memory counter only. Choose as a tripwire against overwrite-capable fallback paths.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) Put(_ context.Context, _, _ string, _ io.ReadSeeker, _ int64, _ string) error {
	s.plainWrites++
	return errors.New("plain Put is forbidden in preservation tests")
}

// Exists checks the fixture map without changing it.
// Inputs: context, bucket and key; outputs: whether the exact object key is present.
// Side effects: none. Choose to drive idempotency checks in the activity helper.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) Exists(_ context.Context, bucket, key string) (bool, error) {
	_, ok := s.objects[bucket+"/"+key]
	return ok, nil
}

// PutIfAbsent emulates a provider-enforced conditional write for the injected seam.
// Inputs: object coordinates, seekable body, exact byte count and content type; outputs: create or already-exists error.
// Side effects: inserts a copied byte slice only when the key is absent.
// Choose to test atomic create behavior independently from AWS SDK and B2 network semantics.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) PutIfAbsent(_ context.Context, bucket, key string, body io.ReadSeeker, size int64, _ string) error {
	s.conditionalWrites++
	objectKey := bucket + "/" + key
	if _, ok := s.objects[objectKey]; ok {
		return errors.New("conditional create refused existing object")
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if int64(len(data)) != size {
		return errors.New("fixture write size mismatch")
	}
	s.objects[objectKey] = data
	return nil
}

// PutRecoveredVersion emulates retained-version writes without touching conditional or plain Put paths.
// Inputs: coordinates, body, byte count, type and expected digest; outputs: a stable synthetic VersionId or mismatch.
// Side effects: adds only an in-memory test version. Choose to assert explicit storage-mode routing in Activities.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) PutRecoveredVersion(_ context.Context, bucket, key string, body io.ReadSeeker, size int64, _ string, expectedSHA string, progress func(int64)) (string, error) {
	s.versionedWrites++
	objectKey := bucket + "/" + key
	if _, ok := s.versions[objectKey]; ok {
		version, _ := s.HeadVersion(context.Background(), bucket, key)
		if err := s.verifyVersion(objectKey, version.VersionID, expectedSHA, size); err != nil {
			return "", err
		}
		return version.VersionID, nil
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	data, err := io.ReadAll(body)
	if err != nil || int64(len(data)) != size || digestBytes(data) != expectedSHA {
		return "", errors.New("versioned fixture source identity mismatch")
	}
	if s.versions == nil {
		s.versions = make(map[string]map[string][]byte)
	}
	s.versions[objectKey] = map[string][]byte{"version-1": append([]byte(nil), data...)}
	s.objects[objectKey] = append([]byte(nil), data...)
	if progress != nil {
		progress(int64(len(data)))
	}
	return "version-1", nil
}

// OpenVersion returns one exact in-memory fixture version.
// Inputs: context, coordinates and VersionId; outputs: an independent reader or not-found error.
// Side effects: none. Choose to make version-mode tests detect accidental reads of the mutable current key.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) OpenVersion(_ context.Context, bucket, key, versionID string) (io.ReadCloser, error) {
	s.versionedReads = append(s.versionedReads, bucket+"/"+key+"#"+versionID)
	data, ok := s.versions[bucket+"/"+key][versionID]
	if !ok {
		return nil, errors.New("fixture version missing")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// HeadVersion returns the current synthetic fixture version and its size.
// Inputs: context and object coordinates; outputs: exact latest test VersionId and size.
// Side effects: none. Choose for retry preflight assertions without provider calls.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) HeadVersion(_ context.Context, bucket, key string) (smsthreads.ObjectVersion, error) {
	objectKey := bucket + "/" + key
	versions := s.versions[objectKey]
	if len(versions) == 0 {
		return smsthreads.ObjectVersion{}, smsthreads.ErrRecoveredVersionNotFound
	}
	versionID := "version-1"
	return smsthreads.ObjectVersion{VersionID: versionID, Size: int64(len(versions[versionID]))}, nil
}

// verifyVersion checks fixture bytes by exact size and digest.
// Inputs: key, VersionId, expected SHA and size; outputs: nil only for an exact match.
// Side effects: none. Choose for fake lost-response replay without weakening assertions to current-key equality.
// Byline: Codex · GPT-6 · 2026-10-04.
func (s *toolkitPreservationMemoryStore) verifyVersion(key, versionID, expectedSHA string, size int64) error {
	data, ok := s.versions[key][versionID]
	if !ok || int64(len(data)) != size || digestBytes(data) != expectedSHA {
		return smsthreads.ErrRecoveredVersionMismatch
	}
	return nil
}

// TestToolkitPackagePreservationCreateOnlyRetry proves matching retries reuse bytes and mismatches refuse overwrite.
// Inputs: one synthetic archive body and a fake conditional object store; outputs: exact assertions on identity and writes.
// Side effects: mutates only the in-memory fixture. Choose to verify bounded create-only and idempotent readback behavior.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationCreateOnlyRetry(t *testing.T) {
	ctx := context.Background()
	store := &toolkitPreservationMemoryStore{objects: map[string][]byte{}}
	body := []byte("PK\x03\x04synthetic-whole-zip-unit")
	want := digestBytes(body)
	if err := createToolkitObjectIfAbsent(ctx, store, "salem-data", "consignatio/casevault/recovery/library-sources/run-a/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/sample.zip", bytes.NewReader(body), int64(len(body)), want, "application/zip", nil, "sample.zip"); err != nil {
		t.Fatal(err)
	}
	if err := createToolkitObjectIfAbsent(ctx, store, "salem-data", "consignatio/casevault/recovery/library-sources/run-a/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/sample.zip", bytes.NewReader(body), int64(len(body)), want, "application/zip", nil, "sample.zip"); err != nil {
		t.Fatalf("matching retry should verify and reuse existing bytes: %v", err)
	}
	changed := []byte("different body")
	err := createToolkitObjectIfAbsent(ctx, store, "salem-data", "consignatio/casevault/recovery/library-sources/run-a/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/sample.zip", bytes.NewReader(changed), int64(len(changed)), digestBytes(changed), "application/zip", nil, "sample.zip")
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("mismatched existing object should be refused, got %v", err)
	}
	if store.conditionalWrites != 1 || store.plainWrites != 0 {
		t.Fatalf("expected one conditional create and no plain writes, got conditional=%d plain=%d", store.conditionalWrites, store.plainWrites)
	}
	if !bytes.Equal(store.objects["salem-data/consignatio/casevault/recovery/library-sources/run-a/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/sample.zip"], body) {
		t.Fatal("existing archive bytes changed after retry or mismatch")
	}
}

// TestToolkitPackagePreservationSelectionIsExplicit checks runtime inventory identities and the 1-to-15 selection bound.
// Inputs: synthetic lowercase inventory digest and package-name lists; outputs: acceptance or rejection assertions.
// Side effects: none. Choose to prove subsets are valid while malformed or oversized selections fail closed.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationSelectionIsExplicit(t *testing.T) {
	input := ToolkitPackagePreservationInput{
		InventoryRef: "file:///worker/inventory.json", InventorySHA256: strings.Repeat("a", 64),
		PackageNames:    make([]string, toolkitPreservationMaxPackages),
		MaxArchiveBytes: 1 << 30, MaxSinglePutBytes: 1 << 30,
	}
	for i := range input.PackageNames {
		input.PackageNames[i] = "package-" + strconv.Itoa(i) + ".zip"
	}
	if err := validateToolkitPackageSelection(input); err != nil {
		t.Fatalf("complete explicit selection rejected: %v", err)
	}
	input.PackageNames[14] = input.PackageNames[13]
	if err := validateToolkitPackageSelection(input); err == nil {
		t.Fatal("duplicate package selection accepted")
	}
	input.PackageNames = input.PackageNames[:1]
	if err := validateToolkitPackageSelection(input); err != nil {
		t.Fatalf("bounded explicit subset rejected: %v", err)
	}
	input.PackageNames = make([]string, toolkitPreservationMaxPackages+1)
	if err := validateToolkitPackageSelection(input); err == nil {
		t.Fatal("selection beyond fifteen accepted")
	}
	input.PackageNames = []string{"package-1.zip"}
	input.InventorySHA256 = strings.ToUpper(strings.Repeat("a", 64))
	if err := validateToolkitPackageSelection(input); err == nil {
		t.Fatal("uppercase digest accepted")
	}
}

// TestToolkitPackagePreservationVersionedModeIsExplicit enforces default compatibility and safe namespace bounds.
// Inputs: preservation requests across empty, conditional, and versioned modes; outputs: validation outcomes.
// Side effects: none. Choose to catch accidental mode fallback before any Activity touches an archive or B2.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationVersionedModeIsExplicit(t *testing.T) {
	base := ToolkitPackagePreservationInput{
		InventoryRef: "file:///worker/inventory.json", InventorySHA256: strings.Repeat("a", 64),
		PackageNames: []string{"unit.zip"}, MaxArchiveBytes: 1 << 30, MaxSinglePutBytes: 1 << 30,
	}
	if err := validateToolkitPackageSelection(base); err != nil || effectiveToolkitStorageMode(base.StorageMode) != ToolkitPackagePreservationModeConditional {
		t.Fatalf("legacy empty mode should remain conditional, got mode=%q err=%v", effectiveToolkitStorageMode(base.StorageMode), err)
	}
	base.StorageMode = ToolkitPackagePreservationModeConditional
	if err := validateToolkitPackageSelection(base); err != nil {
		t.Fatalf("explicit conditional mode rejected: %v", err)
	}
	base.StorageMode = ToolkitPackagePreservationModeVersioned
	if err := validateToolkitPackageSelection(base); err == nil {
		t.Fatal("versioned-recovery without an operation namespace was accepted")
	}
	base.OperationNamespace = "fresh-run-20261004"
	if err := validateToolkitPackageSelection(base); err != nil {
		t.Fatalf("safe versioned-recovery namespace rejected: %v", err)
	}
	base.StorageMode = "anything-else"
	if err := validateToolkitPackageSelection(base); err == nil {
		t.Fatal("unknown storage mode silently fell back")
	}
}

// TestToolkitPackageVersionedWriteUsesExactVersionSeam proves version mode avoids both ordinary write methods.
// Inputs: synthetic store, archive bytes and a unique namespace-compatible key; outputs: exact VersionId readback.
// Side effects: mutates only process-memory fixture state. Choose to test Activity version routing separately from the AWS adapter.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackageVersionedWriteUsesExactVersionSeam(t *testing.T) {
	store := &toolkitPreservationMemoryStore{objects: map[string][]byte{}, versions: map[string]map[string][]byte{}}
	body := []byte("synthetic recovered unit")
	key := "consignatio/casevault/recovery/library-sources/fresh-run-20261004/" + strings.Repeat("a", 64) + "/unit.zip"
	versionID, err := putToolkitRecoveredVersion(context.Background(), store, "salem-data", key, bytes.NewReader(body), int64(len(body)), digestBytes(body), "application/zip", nil, "unit.zip")
	if err != nil || versionID != "version-1" {
		t.Fatalf("versioned write got VersionId=%q err=%v", versionID, err)
	}
	if err = verifyToolkitRemoteObjectVersion(context.Background(), store, "salem-data", key, versionID, digestBytes(body), int64(len(body)), nil, "unit.zip"); err != nil {
		t.Fatalf("exact-version stream verification failed: %v", err)
	}
	if store.versionedWrites != 1 || store.conditionalWrites != 0 || store.plainWrites != 0 {
		t.Fatalf("expected one versioned write and no other writer, got versioned=%d conditional=%d plain=%d", store.versionedWrites, store.conditionalWrites, store.plainWrites)
	}
}

// TestToolkitPackagePreservationVerifyPinsBothInputsAndWritesVersionedReceipt proves archive/receipt IDs are read exactly and verification gets its own ID.
// Inputs: temporary pinned inventory, synthetic remote versions and the versioned verify Activity; outputs: read identities and receipt references.
// Side effects: writes temporary synthetic files and in-memory objects only. Choose to verify the full versioned Activity handoff without B2.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationVerifyPinsBothInputsAndWritesVersionedReceipt(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "archives")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	archive := []byte("synthetic whole archive unit")
	archivePath := filepath.Join(sourceDir, "unit.zip")
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	archiveSHA := digestBytes(archive)
	archiveSHAField := archiveSHA
	inventory := toolkitReceipt{
		Schema:   "toolkit-package-inventory/v1",
		Request:  ToolkitPackageInventoryInput{SourceRef: toolkitFileRef(sourceDir)},
		Packages: []toolkitPackage{{Package: "unit.zip", SHA256: &archiveSHAField, Complete: true}},
	}
	inventoryBytes, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	inventoryPath := filepath.Join(root, "inventory.json")
	if err = os.WriteFile(inventoryPath, inventoryBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	inventoryDigest := digestBytes(inventoryBytes)
	operation := "fresh-verification-run-20261004"
	archiveRef, receiptRef := toolkitPreservationRefs(operation, inventoryDigest, "unit.zip")
	archiveBucket, archiveKey, err := toolkitObjectCoordinates(archiveRef)
	if err != nil {
		t.Fatal(err)
	}
	receiptBucket, receiptKey, err := toolkitObjectCoordinates(receiptRef)
	if err != nil {
		t.Fatal(err)
	}
	originalRef := toolkitFileRef(archivePath)
	archiveVersionID, receiptVersionID := "archive-v1", "receipt-v1"
	archiveReceipt := toolkitPreservationObjectReceipt{
		Schema: "toolkit-package-preservation/v1", StorageMode: ToolkitPackagePreservationModeVersioned,
		InventoryRef: toolkitFileRef(inventoryPath), InventorySHA256: inventoryDigest, PackageName: "unit.zip",
		OriginalRef: originalRef, PreservedRef: archiveRef, SHA256: archiveSHA, Bytes: int64(len(archive)), ArchiveVersionID: archiveVersionID,
	}
	receiptBytes, err := json.Marshal(archiveReceipt)
	if err != nil {
		t.Fatal(err)
	}
	store := &toolkitPreservationMemoryStore{
		objects: map[string][]byte{},
		versions: map[string]map[string][]byte{
			archiveBucket + "/" + archiveKey: {archiveVersionID: archive},
			receiptBucket + "/" + receiptKey: {receiptVersionID: receiptBytes},
		},
	}
	activities := NewToolkitPackagePreservationActivities(root, func(string) (smsthreads.ObjectStore, error) { return store, nil })
	activities.Heartbeat = nil
	result, err := activities.VerifyToolkitPackagePreservation(context.Background(), ToolkitPackagePreservationVerifyInput{
		InventoryRef: toolkitFileRef(inventoryPath), InventorySHA256: inventoryDigest,
		StorageMode: ToolkitPackagePreservationModeVersioned, OperationNamespace: operation,
		PackageName: "unit.zip", PackageNames: []string{"unit.zip"}, SHA256: archiveSHA, Bytes: int64(len(archive)),
		MaxArchiveBytes: 1 << 30, MaxSinglePutBytes: 256 << 20,
		OriginalRef: originalRef, PreservedRef: archiveRef, ReceiptRef: receiptRef,
		ArchiveVersionID: archiveVersionID, ReceiptVersionID: receiptVersionID,
		ReceiptSHA256: digestBytes(receiptBytes), ReceiptBytes: int64(len(receiptBytes)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.VersionID != "version-1" || result.VerificationRef != toolkitVerificationRef(ToolkitPackagePreservationVerifyInput{ReceiptRef: receiptRef}) || result.SHA256 == "" || result.Bytes <= 0 {
		t.Fatalf("verification result did not identify its new receipt version and bytes: %+v", result)
	}
	wantArchiveRead := archiveBucket + "/" + archiveKey + "#" + archiveVersionID
	wantReceiptRead := receiptBucket + "/" + receiptKey + "#" + receiptVersionID
	verifyBucket, verifyKey, err := toolkitObjectCoordinates(result.VerificationRef)
	if err != nil {
		t.Fatal(err)
	}
	wantVerificationRead := verifyBucket + "/" + verifyKey + "#" + result.VersionID
	if len(store.versionedReads) != 3 || store.versionedReads[0] != wantArchiveRead || store.versionedReads[1] != wantReceiptRead || store.versionedReads[2] != wantVerificationRead {
		t.Fatalf("verification did not read the exact input versions: %v", store.versionedReads)
	}
	if err = store.verifyVersion(verifyBucket+"/"+verifyKey, result.VersionID, result.SHA256, result.Bytes); err != nil {
		t.Fatalf("verification receipt version is missing or failed its returned SHA/size: %v", err)
	}
	if store.conditionalWrites != 0 || store.plainWrites != 0 {
		t.Fatalf("versioned verification used a non-versioned writer: conditional=%d plain=%d", store.conditionalWrites, store.plainWrites)
	}
}

// TestToolkitPackagePreservationReceiptIsRootBound authenticates a runtime receipt inside a temporary configured root.
// Inputs: a synthetic versioned receipt, runtime digest and two file URIs; outputs: authenticated acceptance or visible scope/digest failures.
// Side effects: writes only temporary test files, which the test framework removes after completion.
// Choose to prove receipt location and pinning are request inputs rather than hardcoded deployment constants.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationReceiptIsRootBound(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "archives")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	receipt := toolkitReceipt{Schema: "toolkit-package-inventory/v1", Request: ToolkitPackageInventoryInput{SourceRef: toolkitFileRef(source)}}
	for i := 0; i < 16; i++ {
		receipt.Packages = append(receipt.Packages, toolkitPackage{Package: "unit-" + strconv.Itoa(i) + ".zip", Complete: true})
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "inventory-v1.json")
	if err = os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	sha := hex.EncodeToString(digest[:])
	got, err := readPinnedToolkitInventory(context.Background(), toolkitFileRef(path), sha, root, nil)
	if err != nil || len(got.Packages) != 16 {
		t.Fatalf("runtime receipt rejected: packages=%d err=%v", len(got.Packages), err)
	}
	if _, err = readPinnedToolkitInventory(context.Background(), toolkitFileRef(path), strings.Repeat("0", 64), root, nil); err == nil {
		t.Fatal("wrong runtime digest accepted")
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err = os.WriteFile(outside, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = readPinnedToolkitInventory(context.Background(), toolkitFileRef(outside), sha, root, nil); err == nil {
		t.Fatal("receipt outside configured root accepted")
	}
}

// TestToolkitPackagePreservationDestinationIsFixed proves arbitrary bucket and namespace references are rejected.
// Inputs: one allowed and two out-of-scope object references; outputs: coordinate assertions.
// Side effects: none. Choose to protect the fixed b2:salem-data/consignatio/casevault recovery boundary.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitPackagePreservationDestinationIsFixed(t *testing.T) {
	allowed, _ := toolkitPreservationRefs("run-20261004", strings.Repeat("a", 64), "sample.zip")
	if bucket, key, err := toolkitObjectCoordinates(allowed); err != nil || bucket != "salem-data" || !strings.HasPrefix(key, "consignatio/casevault/") {
		t.Fatalf("fixed destination was rejected: bucket=%q key=%q err=%v", bucket, key, err)
	}
	for _, ref := range []proffer.Ref{
		"b2://other-bucket/consignatio/casevault/file.zip",
		"b2://salem-data/other-prefix/file.zip",
		"b2://salem-data/consignatio/casevault/../outside.zip",
	} {
		if _, _, err := toolkitObjectCoordinates(ref); err == nil {
			t.Errorf("out-of-scope destination accepted: %s", ref)
		}
	}
	archiveA, _ := toolkitPreservationRefs("operation-a", strings.Repeat("a", 64), "sample.zip")
	archiveB, _ := toolkitPreservationRefs("operation-b", strings.Repeat("a", 64), "sample.zip")
	if archiveA == archiveB {
		t.Fatal("distinct operation namespaces collided")
	}
}
