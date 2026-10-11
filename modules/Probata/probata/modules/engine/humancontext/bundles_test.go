package humancontext

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type testSource struct {
	content     string
	opens       int
	lengthDelta int64
}

func (s *testSource) Open(_ context.Context, pin Source) (io.ReadCloser, Source, error) {
	s.opens++
	pin.ByteLength = int64(len(s.content)) + s.lengthDelta
	return io.NopCloser(strings.NewReader(s.content)), pin, nil
}

type testExtractor struct{}

func (testExtractor) Extract(_ context.Context, source Source, rows []xmlsourceframes.Lexeme) (Record, error) {
	var ordinal int64
	var body string
	for _, row := range rows {
		if row.RecordOrdinal > 0 {
			ordinal = row.RecordOrdinal
		}
		if row.Name == "body" {
			value, _ := base64.StdEncoding.DecodeString(row.RawValueBase64)
			body = string(value)
		}
	}
	native, _ := json.Marshal(map[string]any{"record_kind": "message", "body": body, "direction": "received", "participants": []string{"synthetic"}, "source_locator": map[string]any{"source_ref": source.SourceRef, "source_version_ref": source.SourceVersionRef, "record_ordinal": ordinal}})
	return Record{Ordinal: ordinal, Status: "complete", Native: native}, nil
}

type testMetadata map[string]Result

func (m testMetadata) LookupStage(_ context.Context, _ Request, stage string) (Result, bool, error) {
	r, ok := m[stage]
	return r, ok, nil
}
func (m testMetadata) RecordStage(_ context.Context, _ Request, r Result) (Result, error) {
	r.ReceiptRef = "synthetic-receipt-" + r.Stage
	m[r.Stage] = r
	return r, nil
}

type testEmbedder struct{}

func (testEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for j := range vectors {
		vectors[j] = []float32{0.25, 0.5}
	}
	return vectors, nil
}

type testTarget struct {
	objects    map[string]SearchObject
	puts, gets int
	corrupt    bool
}

func (t *testTarget) Put(_ context.Context, o SearchObject) error {
	t.puts++
	t.objects[o.ID] = o
	return nil
}
func (t *testTarget) Get(_ context.Context, id string) (SearchObject, error) {
	t.gets++
	o, ok := t.objects[id]
	if !ok {
		return o, errors.New("missing")
	}
	if t.corrupt {
		o.Vector = []float32{9, 9}
	}
	return o, nil
}

// TestNativePathAndIndependentReadback covers stage retry, source order, pagination, independent indexing and readback failure.
// Fixture: synthetic native XML only. Effects: disposable derivative files under the test root, no live stores.
func TestNativePathAndIndependentReadback(t *testing.T) {
	ctx := context.Background()
	source := &testSource{content: `<smses><sms body="first"/><sms body="second"/><sms body="third"/></smses>`}
	meta := testMetadata{}
	b := &Bundles{Root: t.TempDir(), Opener: source, Extractor: testExtractor{}, Metadata: meta}
	req := Request{ContractVersion: Contract, WorkflowID: "synthetic-workflow", Source: Source{SourceRef: "synthetic://sms.xml", SourceVersionRef: "synthetic-version", RegistrationReceiptRef: "synthetic-registration", Format: "smsbackuprestore_xml"}}
	framed, err := b.Frame(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := b.Frame(ctx, req)
	if err != nil || again != framed || source.opens != 1 {
		t.Fatalf("framing retry: %+v %v opens %d", again, err, source.opens)
	}
	req.ManifestRef = framed.ManifestRef
	extracted, err := b.Extract(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	page, err := b.ReadPage(ctx, extracted.ManifestRef, 0, 2)
	if err != nil || len(page.Records) != 2 || page.NextOffset == nil || *page.NextOffset != 2 || page.Total != 3 {
		t.Fatalf("page %+v %v", page, err)
	}
	next, err := b.ReadPage(ctx, extracted.ManifestRef, *page.NextOffset, 2)
	if err != nil || len(next.Records) != 1 || next.Records[0].Ordinal != 3 || next.NextOffset != nil {
		t.Fatalf("next %+v %v", next, err)
	}
	// Synthetic externally planned conversation chunk: three native records become one search object.
	m, err := b.ReadManifest(extracted.ManifestRef)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := b.begin(req, "chunked")
	if err != nil {
		t.Fatal(err)
	}
	chunk := NativeChunk{ThreadID: "native-conversation:synthetic", Text: "first\nsecond\nthird", Chunker: "neural_distilbert", ChunkerVersion: "synthetic-neural", BoundaryVersion: "synthetic-boundary", Overlap: 0}
	for ordinal := int64(1); ordinal <= 3; ordinal++ {
		member := NativeChunkMember{Ordinal: ordinal, BodyEnd: 5}
		member.Locator.SourceRef = m.Source.SourceRef
		member.Locator.SourceVersionRef = m.Source.SourceVersionRef
		member.Locator.Ordinal = ordinal
		chunk.Members = append(chunk.Members, member)
		chunk.MessageRefs = append(chunk.MessageRefs, "native-record:"+m.Source.SourceVersionRef+":"+strconv.FormatInt(ordinal, 10)+":0:5")
	}
	raw, err := EncodeNativeChunk(chunk)
	if err != nil {
		t.Fatal(err)
	}
	chunkPath := filepath.Join(dir, "chunks.jsonl")
	if err := os.WriteFile(chunkPath, append(raw, '\n'), 0640); err != nil {
		t.Fatal(err)
	}
	m.Stage = "chunked"
	m.ChunksRef = fileRef(chunkPath)
	m.Chunks = 1
	m.IndexKind = IndexKind
	chunked, err := b.save(ctx, req, dir, m)
	if err != nil {
		t.Fatal(err)
	}
	target := &testTarget{objects: map[string]SearchObject{}}
	indexer := &Indexer{Bundles: b, Embedder: testEmbedder{}, Target: target}
	req.ManifestRef = chunked.ManifestRef
	published, err := indexer.Publish(ctx, req)
	if err != nil || published.Published != 1 || target.gets != 0 {
		t.Fatalf("publication %+v %v", published, err)
	}
	req.ManifestRef = published.ManifestRef
	target.corrupt = true
	if _, err := indexer.Verify(ctx, req); err == nil {
		t.Fatal("acknowledged writes incorrectly verified corrupted vector")
	}
	target.corrupt = false
	verified, err := indexer.Verify(ctx, req)
	if err != nil || verified.Verified != 1 || target.gets != 2 {
		t.Fatalf("readback %+v %v gets %d", verified, err, target.gets)
	}
	req.Source.SourceVersionRef = "wrong-source"
	delete(meta, "verified")
	if _, err := indexer.Verify(ctx, req); err == nil {
		t.Fatal("cross-source manifest accepted")
	}
	if _, err := b.ReadPage(ctx, "file:///etc/passwd", 0, 1); err == nil {
		t.Fatal("escaped root accepted")
	}
}

// TestPinMismatchAndMalformedRemainder preserves honest partial status and rejects changed original lengths.
// Inputs: synthetic mismatched metadata/truncated XML. Outputs: errors/partial receipts; no original writes.
func TestPinMismatchAndMalformedRemainder(t *testing.T) {
	req := Request{ContractVersion: Contract, WorkflowID: "synthetic", Source: Source{SourceRef: "synthetic://sms.xml", SourceVersionRef: "version", RegistrationReceiptRef: "registration", ByteLength: 100}}
	b := &Bundles{Root: t.TempDir(), Opener: &testSource{content: `<smses/>`}, Extractor: testExtractor{}}
	if _, err := b.Frame(context.Background(), req); err == nil {
		t.Fatal("changed length accepted")
	}
	req.Source.ByteLength = 0
	b.Opener = &testSource{content: `<smses><sms body="first"/><sms body="cut`}
	result, err := b.Frame(context.Background(), req)
	if err != nil || result.Status != "partial" {
		t.Fatalf("malformed framing %+v %v", result, err)
	}
	manifest, err := b.ReadManifest(result.ManifestRef)
	if err != nil || manifest.Framing.Complete || manifest.Framing.Bytes != int64(len(b.Opener.(*testSource).content)) {
		t.Fatalf("accounting %+v %v", manifest, err)
	}
	p, err := b.path(manifest.SpansRef)
	if err != nil {
		t.Fatal(err)
	}
	spans, err := os.ReadFile(p)
	if err != nil || !strings.Contains(string(spans), `"kind":"unparsed"`) {
		t.Fatalf("unparsed remainder missing %s %v", spans, err)
	}
}
