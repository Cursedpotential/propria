// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// skip_record_kinds (owner 2026-10-02): messages and calls are published as chunks after the commit, so the
// Weaviate-first stage leaves them out; AI exports use their own conversation-content Activities.
package activities

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
)

type skipSliceReader struct {
	records []ContextSearchRecord
	next    int
}

func (r *skipSliceReader) Next(context.Context) (ContextSearchRecord, error) {
	if r.next >= len(r.records) {
		return ContextSearchRecord{}, io.EOF
	}
	r.next++
	return r.records[r.next-1], nil
}

func (r *skipSliceReader) Close() error { return nil }

type skipRecordingEmbedder struct{ texts []string }

func (e *skipRecordingEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.texts = append(e.texts, texts...)
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.1, 0.2}
	}
	return out, nil
}

type skipRecordingTarget struct {
	published map[string][]contextsearch.Object
}

func (t *skipRecordingTarget) EnsureCollection(context.Context, string) ([]string, error) {
	return nil, nil
}

func (t *skipRecordingTarget) PublishObjects(_ context.Context, collection string, objects []contextsearch.Object) (ContextSearchPublishResult, error) {
	if t.published == nil {
		t.published = map[string][]contextsearch.Object{}
	}
	t.published[collection] = append(t.published[collection], objects...)
	return ContextSearchPublishResult{Requested: len(objects), Written: len(objects), ObjectIDs: []string{"id"}}, nil
}

func skipSpec(skip map[string]bool) PublishContextSearchSpec {
	return PublishContextSearchSpec{RequestID: "r", Attempt: 1, ExtractionAttemptRef: "e", SkipRecordKinds: skip}
}

func skipTestPlan(t *testing.T, types ...string) ContextSearchPlan {
	t.Helper()
	sha := bytes.Repeat([]byte{7}, 32)
	resolution, err := disclosure.NewResolution(
		disclosure.Owner{PersonID: "owner", PerspectivePersonID: "owner", Identifiers: map[string]bool{"8105550100": true}},
		[]disclosure.ResolvedIdentifier{
			{Raw: "+1 810 555 0100", Normalized: "8105550100", EntityID: "owner", IsOwner: true},
			{Raw: "+18105550199", Normalized: "8105550199", EntityID: "partner"},
		})
	if err != nil {
		t.Fatal(err)
	}
	var records []ContextSearchRecord
	for i, recordType := range types {
		records = append(records, ContextSearchRecord{
			RowID: uuid.New(), Ordinal: int64(i), RecordType: recordType, KnowledgeTime: time.Now(),
			Body: recordType + " body", ContentSHA256: sha,
			Participants: []ContextSearchParticipant{{Role: "sender", Identifier: "+18105550199"}, {Role: "recipient", Identifier: "+1 810 555 0100"}},
		})
	}
	return ContextSearchPlan{
		Provenance: contextsearch.Provenance{
			SourceObjectSHA256: sha, SourceFormat: "smsbackuprestore_xml", ParserID: "p", ParserVersion: "1",
			NormalizerID: "n", NormalizerVersion: "1", RequestID: "r", Attempt: 1, ExtractionAttemptRef: "e",
		},
		Coordinates: contextsearch.Coordinates{
			Schema: "context", Table: "normalized_record_identity",
			SourceVersionID: uuid.New(), NormalizedGenerationID: uuid.New(),
		},
		Resolution: resolution, Reader: &skipSliceReader{records: records},
	}
}

func skipTestActivities(embedder *skipRecordingEmbedder, target *skipRecordingTarget) PublishContextSearchActivities {
	return PublishContextSearchActivities{
		Embedder: embedder, Target: target,
		Collections: map[string]string{
			contextsearch.RecordKindMessage: "Msgs", contextsearch.RecordKindCall: "Msgs",
			contextsearch.RecordKindAIChat: "Chats", contextsearch.RecordKindDocument: "Docs",
		},
	}
}

func TestParseSkipRecordKindsAllowsOnlyMessageAndCall(t *testing.T) {
	got, err := parseSkipRecordKinds("message, call,")
	if err != nil || !got["message"] || !got["call"] || len(got) != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got, err := parseSkipRecordKinds(""); err != nil || len(got) != 0 {
		t.Fatalf("empty list: %v, %v", got, err)
	}
	for _, bad := range []string{"ai_chat", "document", "message,document"} {
		if _, err := parseSkipRecordKinds(proffer.Ref(bad)); err == nil {
			t.Errorf("%q was accepted; only message and call may be skipped", bad)
		}
	}
	spec, err := publishContextSearchSpecFrom(proffer.StageRequest{
		RequestID: "r", SourceVersionRef: "sv",
		Refs: map[string]proffer.Ref{
			"normalized_generation": "g", "normalized_verification": "v", "extraction_attempt": "a",
			"participant_resolution": "p", SkipRecordKindsRefKey: "message,call",
		},
	}, 1)
	if err != nil || !spec.SkipRecordKinds["message"] || !spec.SkipRecordKinds["call"] {
		t.Fatalf("spec = %+v, %v; want the skip list carried through", spec, err)
	}
}

func TestSkippedKindsAreCountedNotPublishedNorEmbedded(t *testing.T) {
	embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
	a := skipTestActivities(embedder, target)
	spec := skipSpec(map[string]bool{"message": true, "call": true})
	outcome, err := a.publishStream(context.Background(), spec, skipTestPlan(t, "message", "call", "message", "document"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Published != 1 || outcome.SkippedToChunks != 3 || len(embedder.texts) != 1 || len(target.published["Docs"]) != 1 || len(target.published["Msgs"]) != 0 {
		t.Fatalf("outcome %+v, embedded %v, published %v; want only the document published", outcome, embedder.texts, target.published)
	}
}

func TestAMessagesOnlyGenerationPublishesNothingPerMessageAndIsNotAnError(t *testing.T) {
	embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
	spec := skipSpec(map[string]bool{"message": true, "call": true})
	outcome, err := skipTestActivities(embedder, target).publishStream(context.Background(), spec, skipTestPlan(t, "message", "message", "call"), nil)
	if err != nil || outcome.Published != 0 || outcome.SkippedToChunks != 3 || len(embedder.texts) != 0 || len(target.published) != 0 {
		t.Fatalf("outcome %+v, err %v; want 3 skipped, nothing embedded or written", outcome, err)
	}
}

func TestWithoutTheSkipListEveryKindIsStillPublishedPerRecord(t *testing.T) {
	embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
	outcome, err := skipTestActivities(embedder, target).publishStream(context.Background(), skipSpec(nil), skipTestPlan(t, "message", "call", "document"), nil)
	if err != nil || outcome.Published != 3 || outcome.SkippedToChunks != 0 || len(target.published["Msgs"]) != 2 {
		t.Fatalf("outcome %+v, err %v; want the old per-record behaviour", outcome, err)
	}
}

func TestAIChatsCannotPublishIndividualRecords(t *testing.T) {
	embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
	plan := skipTestPlan(t, "message")
	plan.Provenance.SourceFormat = "chatgpt_official_json"
	spec := skipSpec(map[string]bool{"message": true, "call": true})
	outcome, err := skipTestActivities(embedder, target).publishStream(context.Background(), spec, plan, nil)
	if err == nil || outcome.Published != 0 || len(target.published) != 0 || len(embedder.texts) != 0 {
		t.Fatalf("outcome %+v, err %v; AI per-record publication must fail before embedding or writing", outcome, err)
	}
}
