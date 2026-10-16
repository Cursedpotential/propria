package humancontext

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
)

// pinnedTestSource returns fully known synthetic source pins without private source content.
// Inputs: none. Outputs: metadata fixture. Effects: none. Choose for exact source-binding tests.
func pinnedTestSource() Source {
	return Source{SourceRef: "synthetic://source.xml?version=one", ProviderVersionID: "provider-one", SourceVersionRef: "registered-one",
		RegistrationReceiptRef: "registration-one", Format: "smsbackuprestore_xml", OwnerPersonRef: "owner-one",
		PerspectivePersonRef: "person-one", Perspective: "device-one", ByteLength: 123, ETag: "opaque-one"}
}

// sourcePinMutations lists independent source substitutions and known-pin removals.
// Inputs: none. Outputs: metadata-only mutation table. Effects: none. Choose across all downstream boundaries.
func sourcePinMutations() map[string]func(*Source) {
	return map[string]func(*Source){
		"uri":                      func(s *Source) { s.SourceRef = "synthetic://other.xml" },
		"registered_version":       func(s *Source) { s.SourceVersionRef = "registered-other" },
		"registration_receipt":     func(s *Source) { s.RegistrationReceiptRef = "registration-other" },
		"format":                   func(s *Source) { s.Format = "other" },
		"owner":                    func(s *Source) { s.OwnerPersonRef = "owner-other" },
		"perspective_person":       func(s *Source) { s.PerspectivePersonRef = "person-other" },
		"perspective":              func(s *Source) { s.Perspective = "device-other" },
		"provider_version":         func(s *Source) { s.ProviderVersionID = "provider-other" },
		"removed_provider_version": func(s *Source) { s.ProviderVersionID = "" },
		"length":                   func(s *Source) { s.ByteLength++ },
		"removed_length":           func(s *Source) { s.ByteLength = 0 },
		"negative_length":          func(s *Source) { s.ByteLength = -1 },
		"etag":                     func(s *Source) { s.ETag = "opaque-other" },
		"removed_etag":             func(s *Source) { s.ETag = "" },
	}
}

// TestValidateSourcePinsAllowsOnlyUnknownObservations checks identity and directional pin rules.
// Inputs: synthetic exact/mutated pins. Outputs: acceptance/rejection assertions. Effects: none.
// Choose to preserve opaque ETag comparison while prohibiting invented identity or attribution.
func TestValidateSourcePinsAllowsOnlyUnknownObservations(t *testing.T) {
	expected := pinnedTestSource()
	if err := ValidateSourcePins(expected, expected); err != nil {
		t.Fatal(err)
	}
	quoted := expected
	quoted.ETag = `"opaque-one"`
	if err := ValidateSourcePins(expected, quoted); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range sourcePinMutations() {
		t.Run(name, func(t *testing.T) {
			observed := expected
			mutate(&observed)
			if ValidateSourcePins(expected, observed) == nil {
				t.Fatal("changed known pin accepted")
			}
		})
	}
	unknown := expected
	unknown.ProviderVersionID = ""
	unknown.ByteLength = 0
	unknown.ETag = ""
	if err := ValidateSourcePins(unknown, expected); err != nil {
		t.Fatalf("unknown native metadata could not become observed: %v", err)
	}
	if err := ValidateSourcePins(unknown, unknown); err != nil {
		t.Fatal(err)
	}
	unknown.OwnerPersonRef = ""
	if ValidateSourcePins(unknown, expected) == nil {
		t.Fatal("unknown attribution invented from derivative")
	}
	for _, field := range []string{"uri", "version", "receipt"} {
		broken := expected
		switch field {
		case "uri":
			broken.SourceRef = ""
		case "version":
			broken.SourceVersionRef = ""
		case "receipt":
			broken.RegistrationReceiptRef = ""
		}
		if ValidateSourcePins(broken, broken) == nil {
			t.Fatalf("missing registered %s accepted", field)
		}
	}
}

type forbiddenExtractor struct{ calls int }

// Extract records a forbidden downstream call for the source-binding regression tests.
// Inputs: unused synthetic source/lexemes. Outputs: error. Effects: increments probe only.
// Choose as a dependency that must never run for a mismatched manifest.
func (e *forbiddenExtractor) Extract(context.Context, Source, []xmlsourceframes.Lexeme) (Record, error) {
	e.calls++
	return Record{}, errors.New("extractor must not run")
}

type forbiddenEmbedder struct{ calls int }

// Embed records a forbidden downstream model call without contacting any provider.
// Inputs: unused texts. Outputs: error. Effects: increments probe only.
// Choose to prove manifest validation precedes embedding.
func (e *forbiddenEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	e.calls++
	return nil, errors.New("embedder must not run")
}

// writePinTestManifest saves a synthetic bounded manifest under the test bundle root.
// Inputs: test context, bundles and fixture. Output: file reference. Effects: fixture write only.
// Choose for boundary tests without opening original sources or live stores.
func writePinTestManifest(t *testing.T, b *Bundles, m Manifest, name string) string {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(b.Root, name+".json")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return fileRef(p)
}

// TestDownstreamSourceMismatchFailsBeforeWork checks every known pin at all Go consumer boundaries.
// Inputs: substituted framed/chunked/published manifests, with valid registered request metadata.
// Outputs: rejection before extraction, embeddings, search reads/writes or derivative creation.
// Effects: synthetic fixture files only; choose to prevent same-version manifest substitution.
func TestDownstreamSourceMismatchFailsBeforeWork(t *testing.T) {
	for _, stage := range []string{"framed", "chunked", "published"} {
		for name, mutate := range sourcePinMutations() {
			t.Run(stage+"/"+name, func(t *testing.T) {
				extractor, embedder := &forbiddenExtractor{}, &forbiddenEmbedder{}
				target := &testTarget{objects: map[string]SearchObject{"synthetic-object": {ID: "synthetic-object"}}}
				b := &Bundles{Root: t.TempDir(), Extractor: extractor}
				req := Request{ContractVersion: Contract, WorkflowID: "workflow-one", Source: pinnedTestSource()}
				observed := req.Source
				mutate(&observed)
				// A readable publication spool ensures Verify cannot fail merely because the fixture is absent.
				spool := filepath.Join(b.Root, "objects.jsonl")
				raw, _ := json.Marshal(SearchObject{ID: "synthetic-object"})
				if err := os.WriteFile(spool, append(raw, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
				m := Manifest{ContractVersion: Contract, WorkflowID: req.WorkflowID, Stage: stage, Source: observed, LexemesRef: fileRef(spool), ChunksRef: fileRef(spool), PublicationRef: fileRef(spool), Published: 1}
				req.ManifestRef = writePinTestManifest(t, b, m, "input")
				indexer := &Indexer{Bundles: b, Embedder: embedder, Target: target}
				before, err := os.ReadDir(b.Root)
				if err != nil {
					t.Fatal(err)
				}
				switch stage {
				case "framed":
					_, err = b.Extract(context.Background(), req)
				case "chunked":
					_, err = indexer.Publish(context.Background(), req)
				case "published":
					_, err = indexer.Verify(context.Background(), req)
				}
				if err == nil {
					t.Fatal("substituted manifest accepted")
				}
				after, e := os.ReadDir(b.Root)
				if e != nil {
					t.Fatal(e)
				}
				if extractor.calls != 0 || embedder.calls != 0 || target.puts != 0 || target.gets != 0 || len(after) != len(before) {
					t.Fatalf("work occurred before source validation: extract=%d embed=%d put=%d get=%d entries=%d/%d", extractor.calls, embedder.calls, target.puts, target.gets, len(before), len(after))
				}
			})
		}
	}
}

// TestCachedStageCannotBypassManifestPins checks cached outputs and predecessor observations.
// Inputs: metadata cache pointing to a substituted source manifest. Outputs: rejected cache reuse.
// Effects: fixture files and metadata reads only. Choose to protect idempotent retry boundaries.
func TestCachedStageCannotBypassManifestPins(t *testing.T) {
	for _, stage := range []string{"framed", "extracted", "published", "verified"} {
		t.Run(stage, func(t *testing.T) {
			req := Request{ContractVersion: Contract, WorkflowID: "workflow-one", Source: pinnedTestSource()}
			b := &Bundles{Root: t.TempDir(), Metadata: testMetadata{}}
			m := Manifest{ContractVersion: Contract, WorkflowID: req.WorkflowID, Stage: stage, Source: req.Source}
			m.Source.OwnerPersonRef = "substituted-owner"
			ref := writePinTestManifest(t, b, m, "cached")
			b.Metadata.(testMetadata)[stage] = Result{ContractVersion: Contract, Stage: stage, ManifestRef: ref, ReceiptRef: "synthetic-stage-receipt"}
			if _, _, err := b.cachedStage(context.Background(), req, stage, req.Source); err == nil {
				t.Fatal("cache returned different source attribution")
			}
			// An unknown registration pin may become observed, but may not replace an already observed predecessor pin.
			req.Source.ProviderVersionID = ""
			m.Source = req.Source
			m.Source.ProviderVersionID = "provider-two"
			ref = writePinTestManifest(t, b, m, "changed-observation")
			b.Metadata.(testMetadata)[stage] = Result{ManifestRef: ref}
			predecessor := req.Source
			predecessor.ProviderVersionID = "provider-one"
			if _, _, err := b.cachedStage(context.Background(), req, stage, predecessor); err == nil {
				t.Fatal("cache replaced an observed predecessor pin")
			}
		})
	}
}

type pinProbeReader struct{ reads, closes int }

// Read records whether changed-source bytes were consumed during framing validation.
// Inputs: unused buffer. Outputs: EOF. Effects: increments probe only; choose before-stream tests.
func (r *pinProbeReader) Read([]byte) (int, error) { r.reads++; return 0, io.EOF }

// Close records resource release after a source pin mismatch.
// Inputs: none. Outputs: nil error. Effects: increments probe only; choose opener cleanup tests.
func (r *pinProbeReader) Close() error { r.closes++; return nil }

type changedPinOpener struct {
	reader *pinProbeReader
	mutate func(*Source)
}

// Open returns substituted metadata without reading any original bytes.
// Inputs: expected synthetic source. Outputs: probe reader and mutated pins. Effects: metadata mutation only.
// Choose to verify opener attribution and registration pins before framing creates derivatives.
func (o changedPinOpener) Open(_ context.Context, source Source) (io.ReadCloser, Source, error) {
	o.mutate(&source)
	return o.reader, source, nil
}

// TestFrameRejectsChangedPinsBeforeStreaming checks the opener boundary uses the shared source validator.
// Inputs: each known pin substitution. Outputs: rejection, closed reader and no byte reads or derivative writes.
// Effects: disposable empty test root only; choose to protect attribution before lexical framing begins.
func TestFrameRejectsChangedPinsBeforeStreaming(t *testing.T) {
	for name, mutate := range sourcePinMutations() {
		t.Run(name, func(t *testing.T) {
			reader := &pinProbeReader{}
			b := &Bundles{Root: t.TempDir(), Opener: changedPinOpener{reader: reader, mutate: mutate}}
			req := Request{ContractVersion: Contract, WorkflowID: "workflow-one", Source: pinnedTestSource()}
			if _, err := b.Frame(context.Background(), req); err == nil {
				t.Fatal("changed opener pins accepted")
			}
			entries, err := os.ReadDir(b.Root)
			if err != nil {
				t.Fatal(err)
			}
			if reader.reads != 0 || reader.closes != 1 || len(entries) != 0 {
				t.Fatalf("framing consumed or wrote before validation: reads=%d closes=%d files=%d", reader.reads, reader.closes, len(entries))
			}
		})
	}
}
