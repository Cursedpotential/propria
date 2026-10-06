// Byline: Codex · GPT-6.1 · 2026-10-05.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// aiCatalogFake retains exact admitted metadata without implementing another catalog or touching source bytes.
// Inputs: bounded batch; outputs: immutable synthetic rows. Effects: memory only; choose for stage separation/readback failures.
type aiCatalogFake struct {
	rows   []AIWorkproductOccurrence
	pin    string
	writes int
	reads  int
}

// RegisterAIWorkproductOccurrences admits an identical replay or rejects changed synthetic provenance.
// Inputs: canonical batch. Outputs: collision/error. Effects: memory only; choose to exercise Activity handoff semantics.
func (f *aiCatalogFake) RegisterAIWorkproductOccurrences(ctx context.Context, b AIWorkproductCatalogBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, pin, err := AIWorkproductCatalogCanonical(b)
	if err != nil {
		return err
	}
	if f.pin != "" && f.pin != pin {
		return errors.New("synthetic collision")
	}
	f.pin = pin
	f.rows = append([]AIWorkproductOccurrence(nil), b.Rows...)
	for i := range f.rows {
		f.rows[i].Metadata.CatalogBatchSHA256 = pin
	}
	f.writes++
	return nil
}

// ReadAIWorkproductOccurrences returns independent copies with their stored batch pins.
// Inputs: operation/pin. Outputs: fixture rows. Effects: read counter only; choose for independent full metadata comparison.
func (f *aiCatalogFake) ReadAIWorkproductOccurrences(ctx context.Context, op, pin string) ([]AIWorkproductOccurrence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.reads++
	return append([]AIWorkproductOccurrence(nil), f.rows...), nil
}

// aiCatalogFixture executes synthetic physical placement and pins a complete observed transport document.
// Inputs: test and source count. Outputs: catalog group/request/store/root. Effects: retained VPS fixtures/memory versions only; choose over owner source files.
func aiCatalogFixture(t *testing.T, count int) (*AIWorkproductCatalogActivities, AIWorkproductCatalogInput, *aiWorkproductFake, *aiCatalogFake, string) {
	t.Helper()
	p, in, store, root := aiWorkproductFixture(t, count)
	in = aiWorkproductRepin(t, root, in, func(m *AIWorkproductManifest) {
		tr := aiWorkproductTransport{ObservedAt: "2026-10-05T12:00:00Z"}
		for _, f := range m.Files {
			tr.Sources = append(tr.Sources, struct {
				OriginalPath string `json:"original_path"`
				OriginalName string `json:"original_name"`
				ServerPath   string `json:"server_path"`
				Bytes        int64  `json:"bytes"`
				LocalMtimeNS int64  `json:"local_mtime_ns"`
				Unchanged    bool   `json:"local_unchanged_size_mtime"`
			}{AIWorkproductOccurrenceScope + "/" + f.Name, f.Name, "/synthetic/incoming/" + f.SourcePath, f.Bytes, 1759681000123456789, true})
		}
		raw, e := json.Marshal(tr)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "incoming", "transport-manifest.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		m.ProvenanceSHA256 = digestBytes(raw)
	})
	_, readreq := aiWorkproductInspectCopy(t, p, in)
	readback, err := p.ReadbackAIWorkproduct(context.Background(), readreq)
	if err != nil {
		t.Fatal(err)
	}
	repo := &aiCatalogFake{}
	a := NewAIWorkproductCatalogActivities(root, p.Stores, repo)
	a.Heartbeat = nil
	return a, AIWorkproductCatalogInput{Operation: "synthetic-catalog", ReadbackRef: readback.ReceiptRef, ReadbackSHA256: readback.ReceiptSHA256, MetadataRef: toolkitFileRef(filepath.Join(root, "catalog.json"))}, store, repo, root
}

// TestAIWorkproductCatalogStages proves six original occurrences, exact nanoseconds, empty local IDs and ref-only replay.
// Inputs: six synthetic notes. Outputs: provenance, independent readback and immutability assertions. Effects: retained fixture metadata only.
func TestAIWorkproductCatalogStages(t *testing.T) {
	a, in, store, repo, _ := aiCatalogFixture(t, 6)
	writes := store.versionedWrites
	reads := len(store.versionedReads)
	out, err := a.AuthenticateAIWorkproductCatalog(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Count != 6 || len(store.versionedReads) != reads+6 || repo.writes != 0 || store.versionedWrites != writes {
		t.Fatal("authentication crossed independent mutation boundaries")
	}
	b, err := a.loadAIWorkproductCatalog(context.Background(), out)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Rows {
		if r.SourceID != "" || r.Metadata.ObservedMtimeNS != 1759681000123456789 || r.Metadata.EventDateStatus != "unknown" || r.Metadata.Provider != "unknown" {
			t.Fatal("invented identity, chronology or lossy nanoseconds")
		}
	}
	if err = a.RegisterAIWorkproductCatalog(context.Background(), out); err != nil {
		t.Fatal(err)
	}
	if repo.reads != 0 {
		t.Fatal("registration claimed independent readback")
	}
	verified, err := a.ReadbackAIWorkproductCatalog(context.Background(), out)
	if err != nil || !reflect.DeepEqual(verified, out) {
		t.Fatal(err)
	}
	again, err := a.AuthenticateAIWorkproductCatalog(context.Background(), in)
	if err != nil || again != out {
		t.Fatal("exact replay differs", err)
	}
	if store.versionedWrites != writes {
		t.Fatal("catalog wrote object bytes")
	}
	repo.rows[0].Metadata.ProvenanceSHA256 = digestBytes([]byte("different"))
	if _, err = a.ReadbackAIWorkproductCatalog(context.Background(), out); err == nil {
		t.Fatal("readback accepted changed root provenance")
	}
}

// TestAIWorkproductCatalogRejectsTampering proves pins, path escapes, cancellation, incomplete readback and exclusive outputs fail visibly.
// Inputs: retained synthetic metadata changes. Outputs: no database writes. Effects: synthetic fixture changes only.
func TestAIWorkproductCatalogRejectsTampering(t *testing.T) {
	for _, which := range []string{"wrong-pin", "output-in-source", "cancel", "incomplete", "manifest-pin", "output-collision", "symlink"} {
		t.Run(which, func(t *testing.T) {
			a, in, _, repo, root := aiCatalogFixture(t, 1)
			ctx := context.Background()
			switch which {
			case "wrong-pin":
				in.ReadbackSHA256 = digestBytes([]byte("wrong"))
			case "output-in-source":
				in.MetadataRef = toolkitFileRef(filepath.Join(root, "incoming", "catalog.json"))
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "incomplete", "manifest-pin":
				var r AIWorkproductReceipt
				path, _ := resolveFileRef(in.ReadbackRef, root, false)
				raw, _ := os.ReadFile(path)
				json.Unmarshal(raw, &r)
				if which == "incomplete" {
					r.Complete = false
				} else {
					r.Request.Input.ManifestSHA256 = digestBytes([]byte("wrong"))
				}
				raw, _ = json.Marshal(r)
				os.WriteFile(path, raw, 0600)
				in.ReadbackSHA256 = digestBytes(raw)
			case "output-collision":
				os.WriteFile(filepath.Join(root, "catalog.json"), []byte(`{"different":"retained"}`), 0600)
			case "symlink":
				if err := os.Symlink(filepath.Join(root, "manifest.json"), filepath.Join(root, "catalog.json")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := a.AuthenticateAIWorkproductCatalog(ctx, in); err == nil {
				t.Fatal("accepted unsafe metadata", which)
			}
			if repo.writes != 0 {
				t.Fatal("invalid authentication wrote catalog")
			}
		})
	}
}

// TestAIWorkproductCatalogWorkflow proves separate named Activities exchange only bounded reference summaries.
// Inputs: synthetic group/request. Outputs: workflow completion and independent repository read. Effects: fixture-only metadata.
func TestAIWorkproductCatalogWorkflow(t *testing.T) {
	a, in, _, repo, _ := aiCatalogFixture(t, 2)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(a.AuthenticateAIWorkproductCatalog, activity.RegisterOptions{Name: AIWorkproductCatalogAuthenticateActivityName})
	env.RegisterActivityWithOptions(a.RegisterAIWorkproductCatalog, activity.RegisterOptions{Name: AIWorkproductCatalogRegisterActivityName})
	env.RegisterActivityWithOptions(a.ReadbackAIWorkproductCatalog, activity.RegisterOptions{Name: AIWorkproductCatalogReadbackActivityName})
	env.ExecuteWorkflow(AIWorkproductCatalogWorkflow, in)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if repo.writes != 1 || repo.reads != 1 {
		t.Fatal("three stages did not independently execute")
	}
}
