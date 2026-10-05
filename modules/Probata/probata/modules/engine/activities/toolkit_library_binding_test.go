// Byline: Codex · GPT-6 · 2026-10-05. Synthetic identity, exact-original, import and Temporal checks.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// bindingFixture mirrors the approved link cardinalities with entirely synthetic records and retained PDF bytes.
// Inputs: none. Outputs: admitted catalog, 193 raw IDs and original-byte map. Effects: memory only.
// Choose to exercise 117 shared aliases plus two source-only originals without using any private corpus text.
func bindingFixture() (ToolkitWorkingCatalogBatch, []string, map[string][]byte) {
	b := workingTestBatch()
	var ids []string
	for i := 0; i < 193; i++ {
		ids = append(ids, fmt.Sprintf("synthetic-%03d", i))
	}
	data := map[string][]byte{}
	for i := range b.Objects {
		row := &b.Objects[i]
		if i < 321 {
			row.ReferenceRecords = []ToolkitWorkingCatalogLink{{ID: fmt.Sprintf("reference:synthetic-%03d", i)}}
		}
		if i < 117 {
			row.SourceRecords = []ToolkitWorkingCatalogLink{{ID: "source:" + ids[i]}}
		}
		if i == 321 || i == 322 {
			row.SourceRecords = []ToolkitWorkingCatalogLink{{ID: "source:" + ids[i-204]}}
		}
		if i < 32 {
			p := &row.Placement
			p.Path = strings.TrimSuffix(p.Path, ".md") + ".pdf"
			p.ObjectKey = strings.TrimSuffix(p.ObjectKey, ".md") + ".pdf"
			p.ObjectRef = proffer.Ref((&url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + p.ObjectKey, RawQuery: "versionId=" + p.VersionID}).String())
			body := []byte(fmt.Sprintf("%%PDF-1.7\nsynthetic original %d\n%%%%EOF", i))
			p.SHA256 = digestBytes(body)
			p.Bytes = int64(len(body))
			data[p.ObjectKey] = body
		}
	}
	return b, ids, data
}

// bindingInput supplies stable, body-free workflow coordinates from a synthetic admitted catalog.
// Inputs: catalog fixture. Outputs: request. Effects: none; choose over sending arrays through workflow history.
func bindingInput(b ToolkitWorkingCatalogBatch) ToolkitLibraryBindingInput {
	return ToolkitLibraryBindingInput{ToolkitWorkingCatalogInput: b.Request, ObservedAt: "2026-10-05T13:30:47Z"}
}

// TestToolkitBindingsMetadata validates complete source qualification, exact cardinalities and order-independent payload identity.
// Inputs: synthetic authenticated catalog. Outputs: deterministic assertions. Effects: memory only.
// Choose to prove citation-only records stay in the full source export set while originals retain occurrence identities.
func TestToolkitBindingsMetadata(t *testing.T) {
	b, ids, _ := bindingFixture()
	in := bindingInput(b)
	p, e := assembleToolkitBindings(b, ids, in.ObservedAt)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Files) != 443 || len(p.SourceIDs) != 193 || p.SourceIDs[0] != "source:synthetic-000" || p.SourceIDs[192] != "source:synthetic-192" {
		t.Fatal("full qualified ledger missing")
	}
	refs, aliases, originals := 0, 0, 0
	for _, f := range p.Files {
		if f.ReferenceID != "" {
			refs++
		}
		if f.SourceID != "" {
			aliases++
		}
		if f.ReferenceID != "" || f.SourceID != "" {
			originals++
		}
		if strings.HasSuffix(f.Key, ".pdf") && f.Pointer.ContentType != "application/pdf" {
			t.Fatal("PDF pointer retains octet-stream")
		}
		if strings.HasSuffix(f.Key, ".md") && f.Pointer.ContentType != "text/markdown" {
			t.Fatal("markdown pointer media mismatch")
		}
	}
	if refs != 321 || aliases != 119 || originals != 323 {
		t.Fatal("incorrect map cardinalities")
	}
	raw, _ := json.Marshal(p)
	b.Objects[0], b.Objects[442] = b.Objects[442], b.Objects[0]
	ids[0], ids[192] = ids[192], ids[0]
	q, e := assembleToolkitBindings(b, ids, in.ObservedAt)
	if e != nil {
		t.Fatal(e)
	}
	again, _ := json.Marshal(q)
	if !bytes.Equal(raw, again) || len(raw) > 2<<20 {
		t.Fatal("retry payload identity/budget differs")
	}
	var keys map[string]json.RawMessage
	if json.Unmarshal(raw, &keys) != nil || len(keys) != 5 {
		t.Fatal("unexpected metadata fields")
	}
	for _, key := range []string{"receipt_sha256", "manifest_sha256", "reference_map_sha256", "files", "source_ids"} {
		if keys[key] == nil {
			t.Fatal("missing contract field", key)
		}
	}
	if bytes.Contains(raw, []byte("%PDF-")) {
		t.Fatal("body entered metadata")
	}
}

// TestToolkitBindingsRejectConflicts fails closed on partial ledgers, version defects and ambiguous record associations.
// Inputs: targeted synthetic mutations. Outputs: rejection assertions. Effects: no I/O.
// Choose before import; unchanged counts alone cannot prove qualified identity agreement.
func TestToolkitBindingsRejectConflicts(t *testing.T) {
	cases := map[string]func(*ToolkitWorkingCatalogBatch, *[]string){
		"wrong receipt":   func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Request.ReceiptSHA256 = strings.Repeat("c", 64) },
		"missing version": func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Objects[0].Placement.VersionID = "" },
		"wrong hash":      func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Objects[0].Placement.SHA256 = "bad" },
		"partial files":   func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Objects = b.Objects[:442] },
		"raw reference ID": func(b *ToolkitWorkingCatalogBatch, _ *[]string) {
			b.Objects[0].ReferenceRecords[0].ID = "synthetic-000"
		},
		"reference conflict": func(b *ToolkitWorkingCatalogBatch, _ *[]string) {
			b.Objects[1].ReferenceRecords = b.Objects[0].ReferenceRecords
		},
		"alias conflict": func(b *ToolkitWorkingCatalogBatch, _ *[]string) {
			b.Objects[1].SourceRecords = b.Objects[0].SourceRecords
		},
		"alias absent": func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Objects[0].SourceRecords[0].ID = "source:absent" },
		"multiple references": func(b *ToolkitWorkingCatalogBatch, _ *[]string) {
			b.Objects[0].ReferenceRecords = append(b.Objects[0].ReferenceRecords, ToolkitWorkingCatalogLink{ID: "reference:extra"})
		},
		"lost source-only original": func(b *ToolkitWorkingCatalogBatch, _ *[]string) { b.Objects[321].SourceRecords = nil },
		"partial ledger":            func(_ *ToolkitWorkingCatalogBatch, ids *[]string) { *ids = (*ids)[:119] },
		"duplicate ledger":          func(_ *ToolkitWorkingCatalogBatch, ids *[]string) { (*ids)[192] = (*ids)[191] },
		"already qualified ledger":  func(_ *ToolkitWorkingCatalogBatch, ids *[]string) { (*ids)[192] = "source:synthetic-192" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b, ids, _ := bindingFixture()
			mutate(&b, &ids)
			if _, e := assembleToolkitBindings(b, ids, bindingInput(b).ObservedAt); e == nil {
				t.Fatal("invalid linkage accepted")
			}
		})
	}
	b, ids, _ := bindingFixture()
	if _, e := assembleToolkitBindings(b, ids, ""); e == nil {
		t.Fatal("unstable observation time accepted")
	}
}

// bindingFakeReader counts exact-version GETs and closure while all ordinary storage operations are tripwires.
// Inputs: synthetic bytes and optional fault. Outputs: read streams. Effects: memory only; choose to prove zero PUT/HEAD/list calls.
type bindingFakeReader struct {
	smsthreads.ObjectStore
	data          map[string][]byte
	fault         string
	opens, closes int
}

// bindingClosingReader counts resource closure without altering retained fixture bytes.
// Inputs: memory reader/close hook. Outputs: ReadCloser. Effects: counter only; choose for early-failure cleanup assertions.
type bindingClosingReader struct {
	io.Reader
	closed func()
}

// Close records stream release in synthetic verification.
// Inputs: none. Outputs: nil. Effects: counter; choose over an unobservable NopCloser in cleanup tests.
func (r bindingClosingReader) Close() error { r.closed(); return nil }

// OpenVersion serves exactly the requested synthetic provider version or an explicit readback fault.
// Inputs: bucket/key/version. Outputs: bounded stream. Effects: counters only; all write operations remain unavailable.
func (s *bindingFakeReader) OpenVersion(_ context.Context, bucket, key, version string) (io.ReadCloser, error) {
	if bucket != "salem-data" || version != "synthetic-v1" {
		return nil, errors.New("not exact approved version")
	}
	s.opens++
	if s.fault == "missing version" {
		return nil, errors.New("missing")
	}
	body, ok := s.data[key]
	if !ok {
		return nil, errors.New("missing key")
	}
	body = append([]byte(nil), body...)
	switch s.fault {
	case "signature":
		body[0] = 'X'
	case "hash":
		body[len(body)-1] = 'X'
	case "short":
		body = body[:len(body)-1]
	case "long":
		body = append(body, 'X')
	}
	return bindingClosingReader{bytes.NewReader(body), func() { s.closes++ }}, nil
}

// bindingFakeBackend captures only import metadata and supplies scoped acknowledgement mutations.
// Inputs: approved pointer JSON. Outputs: bounded response. Effects: counters only; choose to prove failures never make a POST.
type bindingFakeBackend struct {
	calls int
	body  json.RawMessage
	reply json.RawMessage
	err   error
}

// ImportBindings captures the dedicated operation and simulates its sanitized response contract.
// Inputs: context and metadata. Outputs: response or error. Effects: memory only; no arbitrary route is accepted.
func (b *bindingFakeBackend) ImportBindings(_ context.Context, body json.RawMessage) (json.RawMessage, error) {
	b.calls++
	b.body = append(json.RawMessage(nil), body...)
	return b.reply, b.err
}

// bindingAck supplies the current parent route's exact acknowledgement fields.
// Inputs: none. Outputs: linked response bytes. Effects: none; choose for strict decoder/count/pin tests.
func bindingAck() json.RawMessage {
	raw, _ := json.Marshal(toolkitBindingResponse{Status: "linked", OriginalBindings: 323, SourceExportBindings: 193, OriginalAliases: 119, ReceiptSHA256: ToolkitWorkingCatalogReceiptSHA256, ReferenceMapSHA256: ToolkitWorkingCatalogReferenceMapSHA256})
	return raw
}

// TestToolkitBindingsImportVerifiesPDFs checks all 32 retained originals before sending only metadata once.
// Inputs: synthetic exact-version readers/backend. Outputs: count/hash/freshness assertions. Effects: memory only.
// Choose to demonstrate that pointer PDF media is supported by bytes and every opened stream is closed.
func TestToolkitBindingsImportVerifiesPDFs(t *testing.T) {
	b, ids, data := bindingFixture()
	in := bindingInput(b)
	p, e := assembleToolkitBindings(b, ids, in.ObservedAt)
	if e != nil {
		t.Fatal(e)
	}
	store := &bindingFakeReader{data: data}
	backend := &bindingFakeBackend{reply: bindingAck()}
	a := ToolkitLibraryBindingActivities{Backend: backend, Stores: func(scheme string) (smsthreads.ObjectStore, error) {
		if scheme != "b2" {
			t.Fatal("wrong resolver")
		}
		return store, nil
	}}
	r, e := a.importToolkitBindings(context.Background(), in, p)
	if e != nil {
		t.Fatal(e)
	}
	if store.opens != 32 || store.closes != 32 || backend.calls != 1 || r.VerifiedPDFs != 32 || r.Files != 443 || r.MetadataSHA256 != digestBytes(backend.body) || r.OriginalBindings != 323 || r.SourceExportBindings != 193 || r.OriginalAliases != 119 || r.CatalogFreshness != "not_checked" || r.WholeBucketFreshness != "unchanged" || r.ProjectionFreshness != "not_refreshed" {
		t.Fatal("verification or scoped summary mismatch")
	}
	if len(backend.body) > 2<<20 || bytes.Contains(backend.body, []byte("%PDF-")) {
		t.Fatal("original bytes entered HTTP metadata")
	}
	// A retry sends byte-identical pointers and IDs; backend owns conflict-safe additive persistence.
	before := append([]byte(nil), backend.body...)
	if _, e = a.importToolkitBindings(context.Background(), in, p); e != nil || !bytes.Equal(before, backend.body) {
		t.Fatal("retry metadata changed", e)
	}
}

// TestToolkitBindingsPDFReadbackFailures prevents any import after missing/corrupt/truncated/oversized originals.
// Inputs: exact-version fault fixtures. Outputs: rejection/closure assertions. Effects: memory only.
// Choose to cover SHA and size mismatches that extension-based media assignment cannot detect.
func TestToolkitBindingsPDFReadbackFailures(t *testing.T) {
	for _, fault := range []string{"missing version", "signature", "hash", "short", "long"} {
		t.Run(fault, func(t *testing.T) {
			b, ids, data := bindingFixture()
			in := bindingInput(b)
			p, _ := assembleToolkitBindings(b, ids, in.ObservedAt)
			store := &bindingFakeReader{data: data, fault: fault}
			backend := &bindingFakeBackend{reply: bindingAck()}
			a := ToolkitLibraryBindingActivities{Backend: backend, Stores: func(string) (smsthreads.ObjectStore, error) { return store, nil }}
			if _, e := a.importToolkitBindings(context.Background(), in, p); e == nil || backend.calls != 0 {
				t.Fatal("invalid original imported")
			}
			if fault != "missing version" && store.opens != store.closes {
				t.Fatal("failed stream left open")
			}
		})
	}
	b, ids, data := bindingFixture()
	p, _ := assembleToolkitBindings(b, ids, bindingInput(b).ObservedAt)
	store := &bindingFakeReader{data: data}
	backend := &bindingFakeBackend{reply: bindingAck()}
	a := ToolkitLibraryBindingActivities{Backend: backend, Stores: func(string) (smsthreads.ObjectStore, error) { return store, nil }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := a.importToolkitBindings(ctx, bindingInput(b), p); !errors.Is(e, context.Canceled) || store.opens != 0 || backend.calls != 0 {
		t.Fatal("cancelled import caused effects")
	}
}

// TestToolkitBindingsAcknowledgement rejects incorrect counts, evidence pins, excessive JSON and leaked error text.
// Inputs: targeted backend response mutations. Outputs: failure assertions. Effects: synthetic GET/POST only.
// Choose instead of treating HTTP success as linkage proof.
func TestToolkitBindingsAcknowledgement(t *testing.T) {
	for _, fault := range []string{"count", "receipt", "map", "trailing", "unknown", "oversize", "backend"} {
		t.Run(fault, func(t *testing.T) {
			b, ids, data := bindingFixture()
			in := bindingInput(b)
			p, _ := assembleToolkitBindings(b, ids, in.ObservedAt)
			var ack toolkitBindingResponse
			_ = json.Unmarshal(bindingAck(), &ack)
			backend := &bindingFakeBackend{}
			switch fault {
			case "count":
				ack.OriginalBindings = 322
			case "receipt":
				ack.ReceiptSHA256 = strings.Repeat("c", 64)
			case "map":
				ack.ReferenceMapSHA256 = strings.Repeat("c", 64)
			case "backend":
				backend.err = errors.New("sensitive-credential-tripwire")
			}
			backend.reply, _ = json.Marshal(ack)
			switch fault {
			case "trailing":
				backend.reply = append(backend.reply, []byte(`{}`)...)
			case "unknown":
				backend.reply = json.RawMessage(strings.TrimSuffix(string(backend.reply), "}") + `,"body":"forbidden"}`)
			case "oversize":
				backend.reply = append(backend.reply, bytes.Repeat([]byte(" "), 4096)...)
			}
			a := ToolkitLibraryBindingActivities{Backend: backend, Stores: func(string) (smsthreads.ObjectStore, error) { return &bindingFakeReader{data: data}, nil }}
			if _, e := a.importToolkitBindings(context.Background(), in, p); e == nil || strings.Contains(e.Error(), "sensitive-credential") {
				t.Fatal("invalid/unsafe acknowledgement", e)
			}
		})
	}
}

// TestToolkitBindingAuthenticationBeforeEffects rejects changed receipt bytes without resolving storage or calling HTTP.
// Inputs: digest-pinned request with synthetic mismatching mounted bytes. Outputs: fail-closed assertion.
// Effects: retained fixture under to_be_deleted only; choose to verify the real shared authentication path.
func TestToolkitBindingAuthenticationBeforeEffects(t *testing.T) {
	root, e := filepath.Abs("../../../../../../to_be_deleted/toolkit-library-binding-tests")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	dir, e := os.MkdirTemp(root, "wrong-hash-")
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(dir, "receipt.json")
	if e = os.WriteFile(file, []byte(`{"complete":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	b, _, _ := bindingFixture()
	in := bindingInput(b)
	in.ReceiptRef = proffer.Ref((&url.URL{Scheme: "file", Path: filepath.ToSlash(file)}).String())
	if filepath.VolumeName(file) != "" {
		in.ReceiptRef = proffer.Ref("file:///" + filepath.ToSlash(file))
	}
	backend := &bindingFakeBackend{reply: bindingAck()}
	a := ToolkitLibraryBindingActivities{AllowedRoot: dir, Backend: backend, Stores: func(string) (smsthreads.ObjectStore, error) {
		t.Fatal("resolver before authentication")
		return nil, nil
	}}
	if _, e := a.ImportToolkitLibraryBindings(context.Background(), in); e == nil || !strings.Contains(e.Error(), "SHA-256 mismatch") || backend.calls != 0 {
		t.Fatal("receipt did not authenticate before effects", e)
	}
}

// TestToolkitBindingMediaTypes covers deterministic pointer types independently of retained object metadata.
// Inputs: approved extensions and an unsupported extension. Outputs: mapping/rejection assertions. Effects: none.
// Choose to prevent environment-specific MIME defaults from reviving octet-stream PDF pointers.
func TestToolkitBindingMediaTypes(t *testing.T) {
	for ext, want := range map[string]string{".PDF": "application/pdf", ".md": "text/markdown", ".json": "application/json", ".html": "text/html", ".yaml": "application/yaml", ".csv": "text/csv", ".txt": "text/plain", ".log": "text/plain", ".py": "text/x-python", ".sh": "application/x-sh", ".pyc": "application/octet-stream", "": "application/octet-stream"} {
		got, e := toolkitBindingMedia("member" + ext)
		if e != nil || got != want {
			t.Fatal(ext, got, e)
		}
	}
	if _, e := toolkitBindingMedia("unapproved.exe"); e == nil {
		t.Fatal("unsupported type admitted")
	}
}

// TestToolkitBindingWorkflowTracksImport proves body-free coordinates schedule the import as a tracked Activity.
// Inputs: approved request and mocked bounded summary. Outputs: Temporal result assertions. Effects: SDK test environment only.
// Choose instead of claiming worker registration, production POST or deployment proof.
func TestToolkitBindingWorkflowTracksImport(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	a := ToolkitLibraryBindingActivities{}
	env.RegisterActivityWithOptions(a.ImportToolkitLibraryBindings, activity.RegisterOptions{Name: ToolkitLibraryBindingActivityName})
	b, _, _ := bindingFixture()
	in := bindingInput(b)
	want := ToolkitLibraryBindingResult{OperationID: in.OperationID, OriginalBindings: 323, SourceExportBindings: 193, OriginalAliases: 119}
	env.OnActivity(ToolkitLibraryBindingActivityName, mock.Anything, in).Return(want, nil).Once()
	env.ExecuteWorkflow(ToolkitLibraryBindingWorkflow, in)
	if e := env.GetWorkflowError(); e != nil {
		t.Fatal(e)
	}
	var got ToolkitLibraryBindingResult
	if e := env.GetWorkflowResult(&got); e != nil || got != want {
		t.Fatal("wrong tracked workflow result", e)
	}
	env.AssertExpectations(t)
}
