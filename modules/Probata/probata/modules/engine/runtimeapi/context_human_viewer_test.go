package runtimeapi

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type humanViewFixture struct{ view postgres.HumanSourceView }

func (s humanViewFixture) View(context.Context, string) (postgres.HumanSourceView, error) {
	return s.view, nil
}

// TestHumanViewerPagesNativeBundleAndRejectsCrossSource checks real file pagination and exact native identity binding.
// Inputs: synthetic registered view and derivative files. Outputs: native page/status assertions.
// Effects: disposable fixture files only; no original, database or search writes.
func TestHumanViewerPagesNativeBundleAndRejectsCrossSource(t *testing.T) {
	const id = "018f0000-0000-7000-8000-000000000001"
	root := t.TempDir()
	ref := func(name string) string { return (&url.URL{Scheme: "file", Path: filepath.Join(root, name)}).String() }
	native := humancontext.Record{Ordinal: 1, Status: "complete", Native: json.RawMessage(`{"body":"synthetic native SMS","source_locator":{"start":10,"end":30}}`)}
	raw, _ := json.Marshal(native)
	if err := os.WriteFile(filepath.Join(root, "records.jsonl"), append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	offsets := make([]byte, 8)
	binary.LittleEndian.PutUint64(offsets, 0)
	_ = os.WriteFile(filepath.Join(root, "offsets.bin"), offsets, 0600)
	manifest := humancontext.Manifest{ContractVersion: humancontext.Contract, WorkflowID: "synthetic-workflow", Stage: "extracted", Source: humancontext.Source{SourceRef: "synthetic://sms.xml", SourceVersionRef: id}, Framing: xmlsourceframes.Receipt{Complete: true}, RecordsRef: ref("records.jsonl"), OffsetsRef: ref("offsets.bin"), Records: 1}
	encoded, _ := json.Marshal(manifest)
	_ = os.WriteFile(filepath.Join(root, "manifest.json"), encoded, 0600)
	view := postgres.HumanSourceView{Handle: "human-" + id, SourceVersionRef: id, SourceRef: "synthetic://sms.xml", WorkflowID: "synthetic-workflow", ManifestRef: ref("manifest.json")}
	h := &HumanContextViewerHandler{Metadata: humanViewFixture{view}, Bundles: &humancontext.Bundles{Root: root}}
	request := httptest.NewRequest("GET", "http://synthetic/content?offset=0&limit=1", nil)
	request.SetPathValue("handle", view.Handle)
	response := httptest.NewRecorder()
	h.content(response, request)
	if response.Code != 200 {
		t.Fatalf("native page status %d: %s", response.Code, response.Body.String())
	}
	var page humancontext.Page
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Records) != 1 || page.Records[0].Ordinal != 1 {
		t.Fatalf("native page %+v %v", page, err)
	}
	view.SourceRef = "synthetic://different.xml"
	h.Metadata = humanViewFixture{view}
	response = httptest.NewRecorder()
	h.content(response, request)
	if response.Code != 409 {
		t.Fatalf("cross-source bundle accepted: %d", response.Code)
	}
}
