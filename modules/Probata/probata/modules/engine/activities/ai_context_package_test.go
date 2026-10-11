package activities

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
)

type contextPackageMetadataFake struct {
	calls  int
	result proffer.ContextPackageResult
}

// RecordContextPackage observes compact package metadata without a database.
// Inputs: package coordinates and summary. Outputs: nil. Effects: memory-only test observation.
// Choose to distinguish verified physical placement from catalog success.
func (s *contextPackageMetadataFake) RecordContextPackage(_ context.Context, _ proffer.ContextPackageRequest, result proffer.ContextPackageResult) error {
	s.calls++
	s.result = result
	return nil
}

// RecordContextPackageCatalog observes the separate catalog completion receipt in memory.
// Inputs: verified package summary. Outputs: nil. Effects: test observation only.
// Choose to ensure physical placement and catalog readback are recorded separately.
func (s *contextPackageMetadataFake) RecordContextPackageCatalog(_ context.Context, _ proffer.ContextPackageRequest, result proffer.ContextPackageResult) error {
	s.calls++
	s.result = result
	return nil
}

// contextPackageFixture retains a tiny complete native unit under the owner-controlled fixture root.
// Inputs: test. Outputs: existing placement fake, package adapter and request. Effects: retained fixture files only.
// Choose for source integrity and uncertain-write tests without a database or provider.
func contextPackageFixture(t *testing.T) (AIContextPackageActivities, proffer.ContextPackageRequest, *aiWorkproductFake, string) {
	t.Helper()
	if os.Getenv("AI_WORKPRODUCT_TEST_ROOT") == "" {
		t.Skip("requires retained VPS AI_WORKPRODUCT_TEST_ROOT fixture directory")
	}
	placement, _, store, root := aiWorkproductFixture(t, 1)
	unit := filepath.Join(root, "incoming", "native")
	if err := os.MkdirAll(filepath.Join(unit, "created_works"), 0700); err != nil {
		t.Fatal(err)
	}
	ref := func(name string) string { return string(toolkitFileRef(filepath.Join(unit, filepath.FromSlash(name)))) }
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(unit, filepath.FromSlash(name)), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	in := proffer.ContextPackageRequest{RequestID: "context-source:tiny", WorkflowID: "context-source:tiny", RunID: "run", SourceRef: "file:///private/unchanged.md", SourceFormat: "gemini_markdown", SourceVersionRef: "11111111-1111-4111-8111-111111111111", RegistrationReceiptRef: "22222222-2222-4222-8222-222222222222", PreparedRef: ref("prepared.json"), WorkProductsRef: ref("work_products.json")}
	original := []byte("User: exact original\nAssistant: complete created work\n")
	write("original.md", original)
	write("created_works/0-1-0.txt", []byte("complete created work"))
	source := aiContextPackageSource{SourceRef: in.SourceRef, SourceFormat: in.SourceFormat}
	prepared, _ := json.Marshal(map[string]any{"contract_version": "ai-context-v1", "stage": "prepared", "source": source, "original_ref": ref("original.md"), "original_sha256": digestBytes(original), "records": []map[string]any{{"body": "all source records retained"}}, "chunks": []any{}, "counts": map[string]int{"records": 2, "chunks": 1}})
	works, _ := json.Marshal(map[string]any{"contract_version": "ai-context-v1", "stage": "work_products", "source": source, "prepared_ref": in.PreparedRef, "work_products": []map[string]string{{"content": "complete created work", "file_ref": ref("created_works/0-1-0.txt")}}, "counts": map[string]int{"work_products": 1}})
	write("prepared.json", prepared)
	write("work_products.json", works)
	return AIContextPackageActivities{Placement: placement, Metadata: &contextPackageMetadataFake{}}, in, store, unit
}

type contextPackageCatalogFake struct {
	corrupt bool
	calls   int
}

// RegisterAIContextPackage models canonical metadata registration and independently returned rows.
// Inputs: complete metadata and exact digest. Outputs: equivalent or intentionally damaged payload.
// Effects: memory-only observations; choose to reject catalog claims unsupported by row readback.
func (s *contextPackageCatalogFake) RegisterAIContextPackage(_ context.Context, raw []byte, sha string) ([]byte, error) {
	s.calls++
	if digestBytes(raw) != sha {
		return nil, errors.New("metadata digest differs")
	}
	var payload aiContextCatalogPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if s.corrupt {
		payload.Members[0].SHA256 = strings.Repeat("0", 64)
	}
	return json.Marshal(payload)
}

// TestAIContextPackageCatalogIndependentReadback verifies actual completion separately from pending physical placement.
// Inputs: complete memory-backed package and narrow catalog fake. Outputs: exact immutable receipt and tamper assertions.
// Effects: retained fixture receipts and memory-only metadata; choose to prevent false catalog success.
func TestAIContextPackageCatalogIndependentReadback(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact", true: "stored-row-tamper"}[corrupt], func(t *testing.T) {
			a, in, _, _ := contextPackageFixture(t)
			physical, err := a.RetainAIContextPackage(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			catalog := &contextPackageCatalogFake{corrupt: corrupt}
			a.Catalog = func(context.Context) (ContextPackageCatalog, func(), error) { return catalog, func() {}, nil }
			result, err := a.CatalogAIContextPackage(context.Background(), proffer.ContextCatalogRequest{Request: in, Package: physical})
			if corrupt {
				if err == nil || result.CatalogStatus == "complete" {
					t.Fatal("damaged catalog rows claimed complete")
				}
				return
			}
			if err != nil || result.CatalogStatus != "complete" || result.CatalogReceiptRef == "" || result.CatalogReceiptSHA256 == "" {
				t.Fatalf("catalog not verified %+v %v", result, err)
			}
			var original aiContextPackageReceipt
			if _, err = aiWorkproductReadJSON(context.Background(), a.Placement.AllowedRoot, proffer.Ref(physical.ReceiptRef), physical.ReceiptSHA256, &original); err != nil || original.Result.CatalogStatus != "pending" {
				t.Fatal("physical receipt replaced by catalog outcome")
			}
			again, err := a.CatalogAIContextPackage(context.Background(), proffer.ContextCatalogRequest{Request: in, Package: physical})
			if err != nil || again != result {
				t.Fatalf("catalog retry changed pins %+v %v", again, err)
			}
		})
	}
}

// TestAIContextPackageCompleteAndRetry verifies one colocated complete unit and exact pinned replay.
// Inputs: unchanged tiny native export and fake retained versions. Outputs: body/hash/membership assertions.
// Effects: retained fixture files and memory-only object versions; choose to prove no raw source is rewritten.
func TestAIContextPackageCompleteAndRetry(t *testing.T) {
	a, in, store, unit := contextPackageFixture(t)
	before, err := os.ReadFile(filepath.Join(unit, "original.md"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.RetainAIContextPackage(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.Files != 5 || result.CatalogStatus != "pending" || result.ManifestSHA256 == "" || result.ReceiptSHA256 == "" {
		t.Fatalf("bad package: %+v", result)
	}
	var receipt aiContextPackageReceipt
	if _, err = aiWorkproductReadJSON(context.Background(), a.Placement.AllowedRoot, proffer.Ref(result.ReceiptRef), result.ReceiptSHA256, &receipt); err != nil {
		t.Fatal(err)
	}
	for _, obj := range receipt.Objects {
		if !strings.Contains(obj.ObjectKey, "/ai-chats/_Incoming/context-") || obj.Status != "verified" || obj.VersionID == "" {
			t.Fatalf("bad colocated pin %+v", obj)
		}
	}
	again, err := a.RetainAIContextPackage(context.Background(), in)
	if err != nil || again != result {
		t.Fatalf("replay changed pins %+v %v", again, err)
	}
	if len(store.versions) != 5 {
		t.Fatal("retry wrote additional package objects")
	}
	after, err := os.ReadFile(filepath.Join(unit, "original.md"))
	if err != nil || string(before) != string(after) {
		t.Fatal("original changed")
	}
}

// TestAIContextPackageRejectsTamperAndUncertainWrites checks integrity and collision failure paths.
// Inputs: corrupted native bytes, remote readback, or a lost write response. Outputs: explicit failures.
// Effects: retained fixtures and memory-only versions; choose to prevent false package/catalog completion.
func TestAIContextPackageRejectsTamperAndUncertainWrites(t *testing.T) {
	for _, mode := range []string{"original-hash", "created-work-hash", "corrupt-readback", "lost-put"} {
		t.Run(mode, func(t *testing.T) {
			a, in, store, unit := contextPackageFixture(t)
			switch mode {
			case "original-hash":
				if err := os.WriteFile(filepath.Join(unit, "original.md"), []byte("tampered"), 0600); err != nil {
					t.Fatal(err)
				}
			case "created-work-hash":
				if err := os.WriteFile(filepath.Join(unit, "created_works", "0-1-0.txt"), []byte("tampered"), 0600); err != nil {
					t.Fatal(err)
				}
			case "corrupt-readback":
				store.corruptRead = true
			case "lost-put":
				store.lostResponse = true
			}
			result, err := a.RetainAIContextPackage(context.Background(), in)
			if err == nil || result.Complete {
				t.Fatal("invalid package claimed complete")
			}
			if a.Metadata.(*contextPackageMetadataFake).calls != 0 {
				t.Fatal("incomplete package entered metadata")
			}
			if (mode == "original-hash" || mode == "created-work-hash") && len(store.versions) != 0 {
				t.Fatal("tampered source member uploaded")
			}
			if mode == "lost-put" {
				store.lostResponse = false
				if _, err = a.RetainAIContextPackage(context.Background(), in); err == nil {
					t.Fatal("uncertain write silently replayed")
				}
			}
		})
	}
}
