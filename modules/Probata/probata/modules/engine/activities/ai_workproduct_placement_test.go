// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"bytes"
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

// aiWorkproductFake models returned versions, uncertain responses and independent GET corruption.
// Inputs: synthetic switches; outputs: memory-only B2 capabilities. Effects: synthetic retained maps only; choose for no-network custody failure testing.
type aiWorkproductFake struct {
	*placementFake
	lostResponse bool
	corruptRead  bool
}

// PutRecoveredVersion retains a synthetic object and can lose its response afterwards.
// Inputs: one authenticated fixture body; outputs: synthetic version or explicit lost response. Effects: memory-only version insertion; choose to prove uncertain writes never silently recover from matching latest bytes.
func (s *aiWorkproductFake) PutRecoveredVersion(ctx context.Context, bucket, key string, r io.ReadSeeker, n int64, kind, sha string, progress func(int64)) (string, error) {
	id, err := s.toolkitPreservationMemoryStore.PutRecoveredVersion(ctx, bucket, key, r, n, kind, sha, progress)
	if err == nil && s.lostResponse {
		return "", errors.New("synthetic lost PUT response")
	}
	return id, err
}

// OpenVersion can corrupt only the independently read retained bytes.
// Inputs: exact version coordinates; outputs: exact or intentionally corrupt memory stream. Effects: fixture read counter only; choose to distinguish copy success from readback success.
func (s *aiWorkproductFake) OpenVersion(ctx context.Context, bucket, key, id string) (io.ReadCloser, error) {
	if s.corruptRead {
		return io.NopCloser(strings.NewReader("corrupt")), nil
	}
	return s.toolkitPreservationMemoryStore.OpenVersion(ctx, bucket, key, id)
}

// aiWorkproductFixture retains a bounded synthetic source set and manifest without automatic deletion.
// Inputs: test and count; outputs: Activity group/request/store/root. Effects: creates new fixtures under to_be_deleted on the VPS; choose over TempDir because only the owner removes retained files.
func aiWorkproductFixture(t *testing.T, count int) (*AIWorkproductPlacementActivities, AIWorkproductPlacementInput, *aiWorkproductFake, string) {
	t.Helper()
	base := os.Getenv("AI_WORKPRODUCT_TEST_ROOT")
	if base == "" {
		t.Fatal("AI_WORKPRODUCT_TEST_ROOT must identify the retained VPS fixture directory")
	}
	if filepath.Base(base) != "to_be_deleted" {
		t.Fatal("fixture base must be the owner-controlled to_be_deleted directory")
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, "ai-workproduct-*")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(root, "incoming"), 0700); err != nil {
		t.Fatal(err)
	}
	provenance := []byte(`{"transport":"synthetic immutable original-path manifest"}`)
	if err = os.WriteFile(filepath.Join(root, "incoming", "transport-manifest.json"), provenance, 0600); err != nil {
		t.Fatal(err)
	}
	m := AIWorkproductManifest{SourceUnit: "six-case-notes-20261005", SourceRef: toolkitFileRef(filepath.Join(root, "incoming")), Provider: "unknown", ProvenanceRef: toolkitFileRef(filepath.Join(root, "incoming", "transport-manifest.json")), ProvenanceSHA256: digestBytes(provenance)}
	for i := 1; i <= count; i++ {
		b := []byte(fmt.Sprintf("# Supplied note %d\n\nUnicode résumé and complete source text.\n", i))
		f := AIWorkproductFile{SourcePath: fmt.Sprintf("source-%02d", i), Name: fmt.Sprintf("Original supplied note %d.md", i), SHA256: digestBytes(b), Bytes: int64(len(b))}
		if err = os.WriteFile(filepath.Join(root, "incoming", f.SourcePath), b, 0600); err != nil {
			t.Fatal(err)
		}
		m.Files = append(m.Files, f)
	}
	data, _ := json.Marshal(m)
	if err = os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	in := AIWorkproductPlacementInput{ManifestRef: toolkitFileRef(filepath.Join(root, "manifest.json")), ManifestSHA256: digestBytes(data), ReceiptRef: toolkitFileRef(filepath.Join(root, "receipt.json"))}
	s := &aiWorkproductFake{placementFake: &placementFake{toolkitPreservationMemoryStore: &toolkitPreservationMemoryStore{objects: map[string][]byte{}, versions: map[string]map[string][]byte{}}}}
	a := NewAIWorkproductPlacementActivities(root, func(scheme string) (smsthreads.ObjectStore, error) {
		if scheme != "b2" {
			t.Fatalf("wrong scheme %s", scheme)
		}
		return s, nil
	})
	a.Heartbeat = nil
	return a, in, s, root
}

// aiWorkproductRepin changes only a retained synthetic manifest for a validation case.
// Inputs: fixture/request and transformation; outputs: changed fixture pin. Effects: synthetic fixture metadata only; choose for invalid-path/provider/limit testing.
func aiWorkproductRepin(t *testing.T, root string, in AIWorkproductPlacementInput, change func(*AIWorkproductManifest)) AIWorkproductPlacementInput {
	t.Helper()
	p := filepath.Join(root, "manifest.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m AIWorkproductManifest
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	change(&m)
	b, _ = json.Marshal(m)
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	in.ManifestSHA256 = digestBytes(b)
	return in
}

// aiWorkproductInspectCopy executes the first two independent fixture stages.
// Inputs: synthetic group and request; outputs: pinned copy stage predecessor. Effects: fixture-only source read/copy; choose for independent readback failure tests.
func aiWorkproductInspectCopy(t *testing.T, a *AIWorkproductPlacementActivities, in AIWorkproductPlacementInput) (AIWorkproductSummary, AIWorkproductStageInput) {
	t.Helper()
	first, err := a.InspectAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in})
	if err != nil {
		t.Fatal(err)
	}
	req := AIWorkproductStageInput{Input: in, PreviousRef: first.ReceiptRef, PreviousSHA256: first.ReceiptSHA256}
	second, err := a.CopyAIWorkproduct(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	return second, AIWorkproductStageInput{Input: in, PreviousRef: second.ReceiptRef, PreviousSHA256: second.ReceiptSHA256}
}

// TestAIWorkproductPlacementSixNotes proves complete names, source immutability, independent version reads and idempotent receipts.
// Inputs: six synthetic notes; outputs: exact source/target/hash/receipt assertions. Effects: retained fixtures and memory-only versions; choose as the primary placement proof.
func TestAIWorkproductPlacementSixNotes(t *testing.T) {
	a, in, s, root := aiWorkproductFixture(t, 6)
	copyResult, readRequest := aiWorkproductInspectCopy(t, a, in)
	if len(s.versionedReads) != 0 || !copyResult.Complete {
		t.Fatal("copy conflated independent readback")
	}
	final, err := a.ReadbackAIWorkproduct(context.Background(), readRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !final.Complete || final.Objects != 6 || len(s.versionedReads) != 6 || s.versionedWrites != 6 {
		t.Fatalf("incorrect complete proof %+v reads=%d writes=%d", final, len(s.versionedReads), s.versionedWrites)
	}
	var receipt AIWorkproductReceipt
	if _, err = aiWorkproductReadJSON(context.Background(), root, final.ReceiptRef, final.ReceiptSHA256, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Manifest.Provider != "unknown" || receipt.IngestionStatus != "pending" || receipt.CatalogStatus != "pending" {
		t.Fatal("inferred provider or false ingestion/catalog claim")
	}
	for i, f := range receipt.Manifest.Files {
		b, e := os.ReadFile(filepath.Join(root, "incoming", f.SourcePath))
		if e != nil || digestBytes(b) != f.SHA256 {
			t.Fatal("original changed")
		}
		o := receipt.Objects[i]
		want := aiWorkproductPrefix + receipt.Manifest.SourceUnit + "/" + f.Name
		if o.ObjectKey != want || o.VersionID != "version-1" || o.Status != "verified" || !bytes.Equal(b, s.versions["salem-data/"+want][o.VersionID]) {
			t.Fatalf("mapping not preserved %+v", o)
		}
		checkpoint, e := os.ReadFile(filepath.Join(root, fmt.Sprintf("receipt.json.copy.json.object-%02d.json", i+1)))
		if e != nil {
			t.Fatal(e)
		}
		var retained AIWorkproductReceipt
		if e = json.Unmarshal(checkpoint, &retained); e != nil || len(retained.Objects) != 1 || retained.Objects[0].VersionID != o.VersionID || retained.Complete {
			t.Fatal("actual PUT response not retained independently before verification")
		}
	}
	first, e := a.InspectAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in})
	if e != nil {
		t.Fatal(e)
	}
	copyReplay, e := a.CopyAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in, PreviousRef: first.ReceiptRef, PreviousSHA256: first.ReceiptSHA256})
	if e != nil || copyReplay.ReceiptSHA256 != copyResult.ReceiptSHA256 {
		t.Fatalf("copy replay failed %v", e)
	}
	replayed, e := a.ReadbackAIWorkproduct(context.Background(), readRequest)
	if e != nil || replayed.ReceiptSHA256 != final.ReceiptSHA256 || s.versionedWrites != 6 || len(s.versionedReads) != 12 {
		t.Fatalf("replay did not independently recheck pinned objects: %v", e)
	}
	if s.plainWrites != 0 || s.conditionalWrites != 0 {
		t.Fatal("unsafe sibling writer used")
	}
}

// TestAIWorkproductPlacementValidation rejects unsafe names, pins, links, binary bytes and oversized batches before copy.
// Inputs: independently retained invalid fixtures; outputs: visible errors and zero writes. Effects: synthetic fixture edits only; choose as the source-boundary proof.
func TestAIWorkproductPlacementValidation(t *testing.T) {
	for _, name := range []string{"manifest-pin", "provenance-pin", "provenance-escape", "provenance-utf8", "source-pin", "escape", "unit-escape", "name-extension", "name-truncation", "zero", "oversize", "count", "duplicate", "unknown-provider-empty", "invalid-utf8", "nul", "symlink", "source-symlink", "receipt-source", "receipt-escape"} {
		t.Run(name, func(t *testing.T) {
			a, in, s, root := aiWorkproductFixture(t, 1)
			switch name {
			case "manifest-pin":
				in.ManifestSHA256 = digestBytes([]byte("wrong"))
			case "source-pin":
				if err := os.WriteFile(filepath.Join(root, "incoming", "source-01"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "receipt-source":
				in.ReceiptRef = toolkitFileRef(filepath.Join(root, "incoming", "receipt.json"))
			case "receipt-escape":
				in.ReceiptRef = toolkitFileRef(filepath.Join(filepath.Dir(root), "escaped.json"))
			default:
				in = aiWorkproductRepin(t, root, in, func(m *AIWorkproductManifest) {
					f := &m.Files[0]
					switch name {
					case "provenance-pin":
						m.ProvenanceSHA256 = digestBytes([]byte("wrong"))
					case "provenance-escape":
						m.ProvenanceRef = toolkitFileRef(filepath.Join(filepath.Dir(root), "transport-manifest.json"))
					case "provenance-utf8":
						b := []byte{'"', 0xff, '"'}
						if err := os.WriteFile(filepath.Join(root, "incoming", "transport-manifest.json"), b, 0600); err != nil {
							t.Fatal(err)
						}
						m.ProvenanceSHA256 = digestBytes(b)
					case "escape":
						f.SourcePath = "../manifest.json"
					case "unit-escape":
						m.SourceUnit = "../legal"
					case "name-extension":
						f.Name = "legal.pdf"
					case "name-truncation":
						f.Name = strings.Repeat("a", 253) + ".md"
					case "zero":
						f.Bytes = 0
					case "oversize":
						f.Bytes = (1 << 20) + 1
					case "count":
						for len(m.Files) < 17 {
							m.Files = append(m.Files, *f)
						}
					case "duplicate":
						m.Files = append(m.Files, *f)
					case "unknown-provider-empty":
						m.Provider = ""
					case "invalid-utf8", "nul":
						b := []byte{0xff}
						if name == "nul" {
							b = []byte{'a', 0, 'b'}
						}
						if err := os.WriteFile(filepath.Join(root, "incoming", f.SourcePath), b, 0600); err != nil {
							t.Fatal(err)
						}
						f.Bytes = int64(len(b))
						f.SHA256 = digestBytes(b)
					case "symlink":
						if err := os.Symlink(filepath.Join(root, "incoming", f.SourcePath), filepath.Join(root, "incoming", "linked")); err != nil {
							t.Fatal(err)
						}
						f.SourcePath = "linked"
					case "source-symlink":
						if err := os.Symlink(filepath.Join(root, "incoming"), filepath.Join(root, "linked-root")); err != nil {
							t.Fatal(err)
						}
						m.SourceRef = toolkitFileRef(filepath.Join(root, "linked-root"))
					}
				})
			}
			if _, err := a.InspectAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in}); err == nil {
				t.Fatal("invalid inspection accepted")
			}
			if s.versionedWrites != 0 {
				t.Fatal("invalid source wrote B2")
			}
		})
	}
}

// TestAIWorkproductPlacementUncertainAndRace preserves conflicting writes and blocks retries without guessing versions.
// Inputs: synthetic response loss, version race or crash marker; outputs: incomplete pinned receipt and no retry PUT. Effects: retained fixtures and synthetic versions only; choose for idempotency under uncertain outcomes.
func TestAIWorkproductPlacementUncertainAndRace(t *testing.T) {
	for _, name := range []string{"lost-response", "race", "pre-existing", "crash-marker", "cancel"} {
		t.Run(name, func(t *testing.T) {
			a, in, s, root := aiWorkproductFixture(t, 1)
			first, err := a.InspectAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in})
			if err != nil {
				t.Fatal(err)
			}
			req := AIWorkproductStageInput{Input: in, PreviousRef: first.ReceiptRef, PreviousSHA256: first.ReceiptSHA256}
			ctx := context.Background()
			switch name {
			case "lost-response":
				s.lostResponse = true
			case "race":
				s.race = true
			case "pre-existing":
				s.versions["salem-data/"+aiWorkproductPrefix+"six-case-notes-20261005/Original supplied note 1.md"] = map[string][]byte{"version-1": []byte("prior")}
			case "crash-marker":
				if _, err = aiWorkproductWriteExclusive(filepath.Join(root, "receipt.json.copy.json.attempt"), req); err != nil {
					t.Fatal(err)
				}
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				a.Heartbeat = func(context.Context, ToolkitPackagePreservationHeartbeat) { cancel() }
			}
			out, err := a.CopyAIWorkproduct(ctx, req)
			if err == nil || out.Complete {
				t.Fatal("uncertain attempt reported complete")
			}
			writes := s.versionedWrites
			if _, err = a.CopyAIWorkproduct(context.Background(), req); err == nil || s.versionedWrites != writes {
				t.Fatal("retry guessed success or wrote again")
			}
			if name == "lost-response" {
				var r AIWorkproductReceipt
				if _, err = aiWorkproductReadJSON(context.Background(), root, out.ReceiptRef, out.ReceiptSHA256, &r); err != nil {
					t.Fatal(err)
				}
				if len(r.Objects) != 1 || r.Objects[0].VersionID != "" || len(r.Objects[0].AfterVersions) != 1 || r.Error == "" {
					t.Fatal("lost response provenance missing")
				}
			}
		})
	}
}

// TestAIWorkproductPlacementReadbackFailures rejects incorrect predecessor pins, changed sources and corrupt or racing remote versions.
// Inputs: synthetic copy/readback faults; outputs: visible failure without further PUTs. Effects: synthetic fixture/memory edits; choose for independent readback and mutable-source separation proof.
func TestAIWorkproductPlacementReadbackFailures(t *testing.T) {
	for _, name := range []string{"predecessor-pin", "corrupt-version", "readback-race", "copy-source-changed", "readback-mapping"} {
		t.Run(name, func(t *testing.T) {
			a, in, s, root := aiWorkproductFixture(t, 1)
			if name == "copy-source-changed" {
				first, err := a.InspectAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in})
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, "incoming", "source-01"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err = a.CopyAIWorkproduct(context.Background(), AIWorkproductStageInput{Input: in, PreviousRef: first.ReceiptRef, PreviousSHA256: first.ReceiptSHA256}); err == nil || s.versionedWrites != 0 {
					t.Fatal("changed source was copied")
				}
				return
			}
			_, req := aiWorkproductInspectCopy(t, a, in)
			switch name {
			case "predecessor-pin":
				req.PreviousSHA256 = digestBytes([]byte("wrong"))
			case "corrupt-version":
				s.corruptRead = true
			case "readback-race":
				key := "salem-data/" + aiWorkproductPrefix + "six-case-notes-20261005/Original supplied note 1.md"
				s.versions[key]["external-version"] = []byte("external")
			case "readback-mapping":
				var r AIWorkproductReceipt
				if _, err := aiWorkproductReadJSON(context.Background(), root, req.PreviousRef, req.PreviousSHA256, &r); err != nil {
					t.Fatal(err)
				}
				r.Objects[0].ObjectKey = "consignatio/casevault/KnowledgeBase/legal/false.md"
				b, _ := json.Marshal(r)
				if err := os.WriteFile(filepath.Join(root, "receipt.json.copy.json"), b, 0600); err != nil {
					t.Fatal(err)
				}
				req.PreviousSHA256 = digestBytes(b)
			}
			if out, err := a.ReadbackAIWorkproduct(context.Background(), req); err == nil || out.Complete {
				t.Fatal("failed independent proof accepted")
			}
			if s.versionedWrites != 1 {
				t.Fatal("readback performed a PUT")
			}
		})
	}
}

// TestAIWorkproductPlacementWorkflow schedules the three distinct Activities with receipt-only predecessors.
// Inputs: native Temporal test environment and synthetic fixtures; outputs: completed workflow with exact Activity names. Effects: fixture/memory operations only; choose to prove orchestration remains outside independent units.
func TestAIWorkproductPlacementWorkflow(t *testing.T) {
	a, in, s, _ := aiWorkproductFixture(t, 6)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(a.InspectAIWorkproduct, activity.RegisterOptions{Name: AIWorkproductInspectActivityName})
	env.RegisterActivityWithOptions(a.CopyAIWorkproduct, activity.RegisterOptions{Name: AIWorkproductCopyActivityName})
	env.RegisterActivityWithOptions(a.ReadbackAIWorkproduct, activity.RegisterOptions{Name: AIWorkproductReadbackActivityName})
	env.ExecuteWorkflow(AIWorkproductPlacementWorkflow, in)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var out AIWorkproductSummary
	if err := env.GetWorkflowResult(&out); err != nil {
		t.Fatal(err)
	}
	if !out.Complete || out.Objects != 6 || s.versionedWrites != 6 || len(s.versionedReads) != 6 {
		t.Fatalf("workflow incomplete %+v", out)
	}
	if !strings.HasSuffix(string(out.ReceiptRef), ".readback.json") {
		t.Fatal("workflow returned wrong stage")
	}
}
