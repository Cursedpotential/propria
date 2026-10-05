// Byline: Codex · GPT-6 · 2026-10-04. Synthetic metadata only; test artifacts are retained.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// catalogArchiveTripwire fails any archive body read while permitting exact-version open/close checks.
// Inputs: none. Outputs: errors on Read, nil on Close. Effects: none; choose over archive fixture bodies.
type catalogArchiveTripwire struct{}

// Read rejects archive hashing or transfer by this metadata operation.
// Inputs: buffer. Outputs: explicit error. Effects: none; choose to detect corpus reads.
func (catalogArchiveTripwire) Read([]byte) (int, error) {
	return 0, errors.New("archive body read forbidden")
}

// Close releases the synthetic exact-version handle.
// Inputs/outputs: none/nil error. Effects: none; choose for metadata-only existence checks.
func (catalogArchiveTripwire) Close() error { return nil }

// catalogFixtureStore reuses the preservation mock with archive body tripwires.
// Inputs: exact metadata versions. Outputs: pinned streams. Effects: counters only; choose without a network.
type catalogFixtureStore struct {
	*toolkitPreservationMemoryStore
	archives map[string]bool
	opens    int
}

// OpenVersion serves metadata or a no-body exact archive handle.
// Inputs: pinned coordinates. Outputs: stream or missing-version error. Effects: count; choose to prove exact archive existence checks.
func (s *catalogFixtureStore) OpenVersion(ctx context.Context, bucket, key, version string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.HasSuffix(key, ".zip") {
		s.opens++
		if !s.archives[bucket+"/"+key+"#"+version] {
			return nil, errors.New("missing archive version")
		}
		return catalogArchiveTripwire{}, nil
	}
	return s.toolkitPreservationMemoryStore.OpenVersion(ctx, bucket, key, version)
}

// catalogFixture builds fifteen synthetic occurrences and preserved, bounded test files.
// Inputs: test handle. Outputs: Activity group, request and result. Effects: retained files and memory only; choose instead of original sources.
func catalogFixture(t *testing.T) (ToolkitCatalogRegistrationActivities, ToolkitCatalogRegistrationInput, ToolkitPackagePreservationResult, *catalogFixtureStore) {
	t.Helper()
	base := filepath.Join("E:/AI_Workspace/Projects/Propria/_worktrees/toolkit-catalog-registration-20261004/to_be_deleted", "catalog-tests")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, "metadata-")
	if err != nil {
		t.Fatal(err)
	}
	store := &catalogFixtureStore{toolkitPreservationMemoryStore: &toolkitPreservationMemoryStore{objects: map[string][]byte{}, versions: map[string]map[string][]byte{}}, archives: map[string]bool{}}
	input := ToolkitCatalogRegistrationInput{OperationID: "synthetic-operation", PreservationNamespace: "synthetic-recovery", ResultRef: toolkitFileRef(filepath.Join(root, "result.json")), MetadataRef: toolkitFileRef(filepath.Join(root, "metadata.json"))}
	result := ToolkitPackagePreservationResult{InventoryRef: toolkitFileRef(filepath.Join(root, "inventory.json")), InventorySHA256: strings.Repeat("a", 64), StorageMode: ToolkitPackagePreservationModeVersioned}
	for i := 0; i < 15; i++ {
		name := fmt.Sprintf("synthetic-%02d.zip", i)
		archive, receiptRef := toolkitPreservationRefs(input.PreservationNamespace, result.InventorySHA256, name)
		p := ToolkitPackagePreservationReceipt{StorageMode: result.StorageMode, PackageName: name, OriginalRef: toolkitFileRef(filepath.Join(root, name)), PreservedRef: archive, ReceiptRef: receiptRef, VerificationRef: receiptRef + ".verified.json", SHA256: strings.Repeat("b", 64), Bytes: 100, ArchiveVersionID: "archive-v1", ReceiptVersionID: "receipt-v1", VerificationVersionID: "verification-v1"}
		r := toolkitPreservationObjectReceipt{Schema: "toolkit-package-preservation/v1", StorageMode: p.StorageMode, InventoryRef: result.InventoryRef, InventorySHA256: result.InventorySHA256, PackageName: name, OriginalRef: p.OriginalRef, PreservedRef: archive, SHA256: p.SHA256, Bytes: p.Bytes, ArchiveVersionID: p.ArchiveVersionID}
		raw, _ := json.Marshal(r)
		p.ReceiptSHA256 = digestBytes(raw)
		p.ReceiptBytes = int64(len(raw))
		bucket, key, _ := toolkitObjectCoordinates(receiptRef)
		store.versions[bucket+"/"+key] = map[string][]byte{p.ReceiptVersionID: raw}
		v := toolkitPreservationVerificationReceipt{Schema: "toolkit-package-verification/v1", StorageMode: p.StorageMode, InventoryRef: result.InventoryRef, InventorySHA256: result.InventorySHA256, PackageName: name, OriginalRef: p.OriginalRef, PreservedRef: archive, ReceiptRef: receiptRef, SHA256: p.SHA256, Bytes: p.Bytes, ArchiveVersionID: p.ArchiveVersionID, PreservationReceiptVersionID: p.ReceiptVersionID, PreservationReceiptSHA256: p.ReceiptSHA256, PreservationReceiptBytes: p.ReceiptBytes}
		raw, _ = json.Marshal(v)
		p.VerificationReceiptSHA256 = digestBytes(raw)
		p.VerificationReceiptBytes = int64(len(raw))
		bucket, key, _ = toolkitObjectCoordinates(p.VerificationRef)
		store.versions[bucket+"/"+key] = map[string][]byte{p.VerificationVersionID: raw}
		bucket, key, _ = toolkitObjectCoordinates(archive)
		store.archives[bucket+"/"+key+"#"+p.ArchiveVersionID] = true
		result.Packages = append(result.Packages, p)
	}
	a := ToolkitCatalogRegistrationActivities{AllowedRoot: root, Stores: func(string) (smsthreads.ObjectStore, error) { return store, nil }, Heartbeat: func(context.Context, ToolkitPackagePreservationHeartbeat) {}}
	catalogWriteResult(t, a, &input, result)
	return a, input, result, store
}

// catalogWriteResult replaces only a retained synthetic input and refreshes its test digest.
// Inputs: fixture/result. Outputs: updated request digest. Effects: synthetic metadata write; choose for admission corruption tests.
func catalogWriteResult(t *testing.T, a ToolkitCatalogRegistrationActivities, input *ToolkitCatalogRegistrationInput, result ToolkitPackagePreservationResult) {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	path, err := resolveFileRef(input.ResultRef, a.AllowedRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	input.ResultSHA256 = digestBytes(raw)
}

// TestToolkitCatalogMetadataReplay verifies exact receipts, all occurrences and changed-proof replay rejection.
// Inputs: synthetic receipts. Outputs: assertions. Effects: retained metadata only; choose to cover preservation/catalog separation.
func TestToolkitCatalogMetadataReplay(t *testing.T) {
	a, input, _, store := catalogFixture(t)
	ctx := context.Background()
	got, err := a.VerifyToolkitCatalogMetadata(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := a.loadToolkitCatalogBatch(ctx, got)
	if err != nil || len(batch.Packages) != 15 {
		t.Fatalf("readback: %v", err)
	}
	if store.opens != 15 || store.plainWrites+store.versionedWrites+store.conditionalWrites != 0 {
		t.Fatal("unexpected source reads/writes")
	}
	again, err := a.VerifyToolkitCatalogMetadata(ctx, input)
	if err != nil || again != got {
		t.Fatalf("identical replay: %v", err)
	}
	p := batch.Packages[0]
	bucket, key, _ := toolkitObjectCoordinates(p.ReceiptRef)
	store.versions[bucket+"/"+key][p.ReceiptVersionID] = []byte("{}")
	if _, err = a.VerifyToolkitCatalogMetadata(ctx, input); err == nil {
		t.Fatal("changed remote evidence accepted on replay")
	}
	if _, err = a.loadToolkitCatalogBatch(ctx, got); err != nil {
		t.Fatal("retained output changed on rejection", err)
	}
}

// TestToolkitCatalogMetadataRejects exercises pins, occurrence counts, receipt identities, bounds and exact version absence.
// Inputs: independent synthetic fixtures. Outputs: fail-visible assertions. Effects: retained test metadata only; choose for admission guard coverage.
func TestToolkitCatalogMetadataRejects(t *testing.T) {
	for _, kind := range []string{"pin", "count", "duplicate", "identity", "missing-version", "receipt-budget", "partial-output", "outside-root"} {
		t.Run(kind, func(t *testing.T) {
			a, input, result, store := catalogFixture(t)
			switch kind {
			case "pin":
				input.ResultSHA256 = strings.Repeat("c", 64)
			case "count":
				result.Packages = result.Packages[:14]
				catalogWriteResult(t, a, &input, result)
			case "duplicate":
				result.Packages[1].OriginalRef = result.Packages[0].OriginalRef
				catalogWriteResult(t, a, &input, result)
			case "identity":
				result.Packages[0].OriginalRef = toolkitFileRef(filepath.Join(a.AllowedRoot, "different.zip"))
				catalogWriteResult(t, a, &input, result)
			case "missing-version":
				store.archives = map[string]bool{}
			case "receipt-budget":
				p := result.Packages[0]
				bucket, key, _ := toolkitObjectCoordinates(p.ReceiptRef)
				store.versions[bucket+"/"+key][p.ReceiptVersionID] = []byte(strings.Repeat("x", (16<<10)+1))
			case "partial-output":
				path, _ := resolveFileRef(input.MetadataRef, a.AllowedRoot, false)
				if err := os.WriteFile(path, []byte("partial"), 0600); err != nil {
					t.Fatal(err)
				}
			case "outside-root":
				input.MetadataRef = toolkitFileRef(filepath.Join(filepath.Dir(a.AllowedRoot), "outside.json"))
			}
			if _, err := a.VerifyToolkitCatalogMetadata(context.Background(), input); err == nil {
				t.Fatal("unsafe admission accepted")
			}
			if kind == "partial-output" {
				path, _ := resolveFileRef(input.MetadataRef, a.AllowedRoot, false)
				raw, _ := os.ReadFile(path)
				if string(raw) != "partial" {
					t.Fatal("partial output overwritten")
				}
			}
		})
	}
}

// TestToolkitCatalogCancellation checks cancellation during metadata verification before output publication.
// Inputs: heartbeat-canceled fixture. Outputs: context cancellation assertion. Effects: retained fixture only; choose for tracked cancellation.
func TestToolkitCatalogCancellation(t *testing.T) {
	a, input, _, store := catalogFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	beats := 0
	a.Heartbeat = func(context.Context, ToolkitPackagePreservationHeartbeat) {
		beats++
		if beats == 4 {
			cancel()
		}
	}
	_, err := a.VerifyToolkitCatalogMetadata(ctx, input)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if store.opens >= 15 {
		t.Fatal("continued after cancellation")
	}
	path, _ := resolveFileRef(input.MetadataRef, a.AllowedRoot, false)
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("published canceled metadata")
	}
}

// TestToolkitCatalogCancellationAtPublication rejects cancellation delivered by the final heartbeat.
// Inputs: synthetic fully checked metadata. Outputs: no-output/cancellation assertions. Effects: retained fixture only.
// Choose to cover the cancellation boundary immediately before exclusive receipt creation.
func TestToolkitCatalogCancellationAtPublication(t *testing.T) {
	a, input, _, store := catalogFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.Heartbeat = func(_ context.Context, h ToolkitPackagePreservationHeartbeat) {
		if store.opens == 15 && h.Phase == "verify-catalog-metadata" {
			cancel()
		}
	}
	if _, err := a.VerifyToolkitCatalogMetadata(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("publication cancellation: %v", err)
	}
	path, _ := resolveFileRef(input.MetadataRef, a.AllowedRoot, false)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("published canceled metadata")
	}
}

// TestToolkitCatalogRemoteResultAndProvenance admits the exported result envelope at an exact B2 version.
// Inputs: synthetic provenance and exact metadata versions. Outputs: retained provenance assertions. Effects: retained metadata only.
// Choose to cover the supplied generation-six result shape without reading original archives or committing its metadata.
func TestToolkitCatalogRemoteResultAndProvenance(t *testing.T) {
	a, input, result, store := catalogFixture(t)
	input.ResultRef = "b2://salem-data/consignatio/casevault/recovery/library-sources/synthetic-result.json"
	input.ResultVersionID = "result-v6"
	envelope := toolkitCatalogPreservationEnvelope{ToolkitPackagePreservationResult: result, Provenance: map[string]any{"byline": "synthetic", "catalog_registered": false, "projection_updated": false}}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	input.ResultSHA256 = digestBytes(raw)
	bucket, key, _ := toolkitObjectCoordinates(input.ResultRef)
	store.versions[bucket+"/"+key] = map[string][]byte{input.ResultVersionID: raw}
	got, err := a.VerifyToolkitCatalogMetadata(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := a.loadToolkitCatalogBatch(context.Background(), got)
	if err != nil || batch.ResultProvenance["byline"] != "synthetic" {
		t.Fatalf("lost provenance: %v", err)
	}
	input.ResultVersionID = "not-the-pinned-version"
	if _, err = a.VerifyToolkitCatalogMetadata(context.Background(), input); err == nil {
		t.Fatal("wrong result version accepted")
	}
}

// TestToolkitCatalogSuppliedResultShape optionally validates the parent's exact local metadata receipt without network reads.
// Inputs: explicit TOOLKIT_CATALOG_TEST_RESULT_FILE and expected SHA environment values. Outputs: shape/canonical admission assertions.
// Effects: bounded metadata file read only; choose to prove live receipt compatibility without copying source metadata into Git.
func TestToolkitCatalogSuppliedResultShape(t *testing.T) {
	path := os.Getenv("TOOLKIT_CATALOG_TEST_RESULT_FILE")
	if path == "" {
		t.Skip("explicit metadata receipt not supplied")
	}
	expected := os.Getenv("TOOLKIT_CATALOG_TEST_RESULT_SHA256")
	if !validSHA256(expected) {
		t.Fatal("explicit receipt pin required")
	}
	raw, sha, err := toolkitReadBoundedJSON(path, toolkitCatalogMetadataLimit, func() error { return nil })
	if err != nil || sha != expected {
		t.Fatal("supplied metadata pin mismatch")
	}
	var r toolkitCatalogPreservationEnvelope
	if err = decodeToolkitCatalogJSON(raw, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Packages) != 15 || r.StorageMode != ToolkitPackagePreservationModeVersioned {
		t.Fatal("supplied receipt count/mode mismatch")
	}
	remainder := strings.TrimPrefix(string(r.Packages[0].PreservedRef), toolkitPreservationDestination)
	parts := strings.SplitN(remainder, "/", 3)
	if len(parts) != 3 {
		t.Fatal("supplied receipt namespace missing")
	}
	batch := ToolkitRecoveryCatalogBatch{Schema: ToolkitRecoveryCatalogSchema, Request: ToolkitCatalogRegistrationInput{OperationID: "supplied-shape-check", PreservationNamespace: parts[0], ResultRef: toolkitFileRef(path), ResultSHA256: expected, MetadataRef: "file:///synthetic/metadata.json"}, InventoryRef: r.InventoryRef, InventorySHA256: r.InventorySHA256, Packages: r.Packages, ResultProvenance: r.Provenance}
	if _, _, err = ToolkitRecoveryCatalogCanonical(batch); err != nil {
		t.Fatal("supplied result not admissible", err)
	}
}

// catalogMemoryRepository retains one batch for independent Activity/workflow tests.
// Inputs: canonical batch. Outputs: same batch or injected row loss. Effects: process memory only; choose without a database.
type catalogMemoryRepository struct {
	batch       ToolkitRecoveryCatalogBatch
	registerErr error
	reads       int
}

// RegisterToolkitRecovery stores metadata or returns a synthetic failure.
// Inputs: batch. Outputs: error. Effects: memory; choose to verify workflow gating.
func (r *catalogMemoryRepository) RegisterToolkitRecovery(_ context.Context, b ToolkitRecoveryCatalogBatch) error {
	if r.registerErr != nil {
		return r.registerErr
	}
	r.batch = b
	return nil
}

// ReadToolkitRecovery returns an independent test batch.
// Inputs: operation ID. Outputs: stored batch. Effects: read counter; choose for workflow/readback checks.
func (r *catalogMemoryRepository) ReadToolkitRecovery(context.Context, string) (ToolkitRecoveryCatalogBatch, error) {
	r.reads++
	return r.batch, nil
}

// TestToolkitCatalogWorkflow proves three separately registered Activities and readback damage detection.
// Inputs: synthetic mounted metadata. Outputs: workflow/error assertions. Effects: retained fixture and Temporal test environment only.
// Choose over direct method tests to validate the existing worker registration seam.
func TestToolkitCatalogWorkflow(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			a, input, _, _ := catalogFixture(t)
			repo := &catalogMemoryRepository{}
			a.Catalog = repo
			if failure {
				repo.registerErr = errors.New("synthetic registration failure")
			}
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterActivityWithOptions(a.VerifyToolkitCatalogMetadata, activity.RegisterOptions{Name: ToolkitCatalogMetadataActivityName})
			env.RegisterActivityWithOptions(a.RegisterToolkitCatalog, activity.RegisterOptions{Name: ToolkitCatalogRegisterActivityName})
			env.RegisterActivityWithOptions(a.ReadbackToolkitCatalog, activity.RegisterOptions{Name: ToolkitCatalogReadbackActivityName})
			env.ExecuteWorkflow(ToolkitCatalogRegistrationWorkflow, input)
			if (env.GetWorkflowError() != nil) != failure {
				t.Fatalf("workflow result: %v", env.GetWorkflowError())
			}
			if failure {
				if repo.reads != 0 {
					t.Fatal("readback followed failed registration")
				}
				return
			}
			var got ToolkitCatalogMetadataResult
			if err := env.GetWorkflowResult(&got); err != nil {
				t.Fatal(err)
			}
			if repo.reads != 1 {
				t.Fatal("missing independent readback")
			}
			repo.batch.Packages = repo.batch.Packages[:14]
			if _, err := a.ReadbackToolkitCatalog(context.Background(), got); err == nil {
				t.Fatal("missing occurrence accepted")
			}
		})
	}
}
