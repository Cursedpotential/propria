package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testID = "11111111-1111-4111-8111-111111111111"
const testOther = "22222222-2222-4222-8222-222222222222"

func testScope() Scope { return Scope{Mode: "LIVE", MatterID: testID, CourtCaseID: testOther} }
func TestEpisodesHonorNativeConversationAndExplicitCrossPlatformOnly(t *testing.T) {
	rs := []SourceRecord{{RecordID: "a", SourceVersionID: "sms", ConversationID: "native"}, {RecordID: "b", SourceVersionID: "sms", ConversationID: "native"}, {RecordID: "c", SourceVersionID: "mail", ConversationID: "native"}, {RecordID: "d", SourceVersionID: "sms", ExplicitEpisodeID: "owner-link"}, {RecordID: "e", SourceVersionID: "mail", ExplicitEpisodeID: "owner-link"}, {RecordID: "f"}, {RecordID: "g"}}
	e := Episodes(rs)
	if len(e) != 5 || len(e[0].RecordIDs) != 2 || len(e[2].RecordIDs) != 2 {
		t.Fatalf("unjustified grouping: %+v", e)
	}
}
func TestHistoricalFiltersAndServerSelection(t *testing.T) {
	f := Filters{Perspective: "as_lived"}
	if f.Validate() == nil {
		t.Fatal("missing cutoff admitted")
	}
	now := time.Now()
	f.Cutoff = &now
	if e := f.Validate(); e != nil {
		t.Fatal(e)
	}
	d := Dataset{DatasetID: "bound", InputBackend: "postgresql", SelectionRevision: "v1", Scope: testScope(), GenerationIDs: []string{testID}}
	r := RunRequest{DatasetID: d.DatasetID, InputBackend: d.InputBackend, SelectionRevision: d.SelectionRevision, Scope: d.Scope, GenerationIDs: []string{testID}}
	if _, e := Selection(r, []Dataset{d}); e != nil {
		t.Fatal(e)
	}
	r.GenerationIDs = []string{testOther}
	if _, e := Selection(r, []Dataset{d}); !errors.Is(e, ErrScope) {
		t.Fatal("foreign generation admitted", e)
	}
	r.GenerationIDs = d.GenerationIDs
	r.Scope.MatterID = testOther
	if _, e := Selection(r, []Dataset{d}); !errors.Is(e, ErrScope) {
		t.Fatal("foreign case admitted", e)
	}
}
func TestFilesCrossLanguageExactBytesAndRootEscape(t *testing.T) {
	root := t.TempDir()
	f, e := NewFiles(root)
	if e != nil {
		t.Fatal(e)
	}
	ref, e := f.Put(testID, "batch-f0", map[string]string{"text": "quoted <text> ☃"})
	if e != nil {
		t.Fatal(e)
	}
	var decoded map[string]string
	if e = f.Read(ref, &decoded); e != nil || decoded["text"] != "quoted <text> ☃" {
		t.Fatal(e, decoded)
	}
	ref2, e := f.Put(testID, "batch-f0", map[string]string{"text": "quoted <text> ☃"})
	if e != nil || ref2 != ref {
		t.Fatal("retry changed identity", e)
	}
	// Python canonical JSON is allowed to use a different key order and literal UTF-8.
	raw := []byte(`{"text":"<☃>","a":1}`)
	path := filepath.Join(root, testID, "python.json")
	if e = os.WriteFile(path, raw, 0400); e != nil {
		t.Fatal(e)
	}
	pyRef := Ref{URI: "file://" + path, SHA256: DigestBytes(raw), Bytes: int64(len(raw))}
	var value any
	if e = f.Read(pyRef, &value); e != nil {
		t.Fatal("cross-language raw checksum rejected", e)
	}
	pyRef.URI = "file:///etc/passwd"
	if !errors.Is(f.Read(pyRef, &value), ErrScope) {
		t.Fatal("escaped root read admitted")
	}
	pyRef.URI = "file://" + path
	pyRef.SHA256 = DigestBytes([]byte("different"))
	if f.Read(pyRef, &value) == nil {
		t.Fatal("changed derivative accepted")
	}
}
func TestCitationCannotEscapeEpisodeOrSourceLocator(t *testing.T) {
	b := Batch{Schema: BatchSchema, RunID: testID, Scope: testScope(), FeedbackRevision: 0, MaxCandidates: 10, Records: []SourceRecord{{RecordID: testID, SourceVersionID: testOther, Text: "a☃b", Locators: []Locator{{LocatorID: testID, SourceVersionID: testOther}}}}, Episodes: []Episode{{EpisodeID: "episode", RecordIDs: []string{testID}}}}
	c := Candidate{CandidateID: "candidate", EpisodeID: "episode", Title: "candidate", Score: .7, Citations: []Citation{{RecordID: testID, LocatorID: testID, StartChar: 1, EndChar: 2, Quote: "☃"}}}
	r := Result{Schema: ResultSchema, RunID: testID, ModelVersion: "local-v1", Candidates: []Candidate{c}}
	if e := ValidateResult(r, b); e != nil {
		t.Fatal(e)
	}
	r.Candidates[0].Citations[0].Quote = "different"
	if ValidateResult(r, b) == nil {
		t.Fatal("ungrounded quotation accepted")
	}
	r.Candidates[0] = c
	r.Candidates[0].Citations = []Citation{{RecordID: testOther, LocatorID: testID, StartChar: 1, EndChar: 2, Quote: "☃"}}
	if ValidateResult(r, b) == nil {
		t.Fatal("foreign record accepted")
	}
}

type savedStore struct {
	Store
	run       Run
	manifests map[int64]Batch
	refs      map[int64]Ref
	feedback  []Feedback
	readCount int
}

func (s *savedStore) ReadRun(context.Context, string, Scope) (Run, error) { return s.run, nil }
func (s *savedStore) LoadManifest(_ context.Context, _ string, _ Scope, rev int64) (Ref, Batch, error) {
	b, ok := s.manifests[rev]
	if !ok {
		return Ref{}, Batch{}, ErrNotFound
	}
	return s.refs[rev], b, nil
}
func (s *savedStore) FeedbackForRun(context.Context, string, Scope) ([]Feedback, error) {
	return s.feedback, nil
}
func (s *savedStore) SaveManifest(_ context.Context, _ string, _ Scope, rev int64, ref Ref, b Batch) error {
	s.manifests[rev] = b
	s.refs[rev] = ref
	return nil
}

type countedSources struct{ reads *int }

func (s countedSources) ReadSources(context.Context, RunRequest) ([]SourceRecord, bool, error) {
	*s.reads++
	return []SourceRecord{{RecordID: testID, Text: "immutable source", InputRevision: "v1"}}, false, nil
}
func TestAssemblyReplaysImmutableManifestAndRetainsCompatibleLearning(t *testing.T) {
	files, _ := NewFiles(t.TempDir())
	reads := 0
	run := Run{Request: RunRequest{RunID: testID, Scope: testScope(), MaxCandidates: 10, Filters: Filters{Perspective: "hindsight"}}}
	store := &savedStore{run: run, manifests: map[int64]Batch{}, refs: map[int64]Ref{}, feedback: []Feedback{{RunID: testOther, Revision: 12, InputPins: map[string]string{testID: "v1"}, FeedbackRequest: FeedbackRequest{Kind: "skip"}}}}
	a := Activities{Store: store, Reader: countedSources{&reads}, Files: files, ModelConfig: json.RawMessage(`{"model_revision":"pinned-v1"}`)}
	req := WorkRequest{RunID: testID, Scope: testScope()}
	first, e := a.Assemble(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	second, e := a.Assemble(context.Background(), req)
	if e != nil || first != second || reads != 1 {
		t.Fatal("replay reread mutable sources", e, reads)
	}
	if len(store.manifests[0].Feedback) != 1 {
		t.Fatal("new run lost compatible durable training")
	}
	store.run.FeedbackRevision = 1
	req.FeedbackRevision = 1
	if _, e = a.Assemble(context.Background(), req); e != nil || reads != 1 {
		t.Fatal("rescore changed input membership", e, reads)
	}
}
