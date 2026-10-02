// Byline: Claude Code · Opus 5.5 · 2026-10-02
package activities

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
)

type sliceReader struct{ records []ContextSearchRecord }

func (r *sliceReader) Next(context.Context) (ContextSearchRecord, error) {
	if len(r.records) == 0 {
		return ContextSearchRecord{}, io.EOF
	}
	next := r.records[0]
	r.records = r.records[1:]
	return next, nil
}
func (r *sliceReader) Close() error { return nil }

type matchSource struct {
	records []ContextSearchRecord
	outcome ContextSearchPublicationOutcome
}

func (s *matchSource) OpenContextSearchRecords(context.Context, PublishContextSearchSpec) (ContextSearchPlan, error) {
	return ContextSearchPlan{
		Resolution: disclosure.Resolution{Basis: disclosure.Basis, OwnerPersonID: "owner"},
		Reader:     &sliceReader{records: s.records},
	}, nil
}
func (s *matchSource) PersistContextSearchPublication(_ context.Context, _ PublishContextSearchSpec, outcome ContextSearchPublicationOutcome) (proffer.Ref, proffer.Ref, error) {
	s.outcome = outcome
	return "publication", "publication-receipt", nil
}

type countingTarget struct{ published int }

func (t *countingTarget) EnsureCollection(context.Context, string) ([]string, error) { return nil, nil }
func (t *countingTarget) PublishObjects(_ context.Context, _ string, objects []contextsearch.Object) (ContextSearchPublishResult, error) {
	t.published += len(objects)
	return ContextSearchPublishResult{Requested: len(objects), Written: len(objects)}, nil
}

type noEmbedder struct{}

func (noEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1}
	}
	return out, nil
}

type fixedMatches map[string]bool

func (m fixedMatches) LoadMatchedRecords(context.Context, proffer.Ref) (map[string]bool, error) {
	return m, nil
}

// A message an earlier source already holds is not published to Weaviate a
// second time, and a generation whose every message is matched is a recorded
// outcome, not the "no publishable records" defect.
func TestPublishContextSearchSkipsMatchedMessages(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	source := &matchSource{records: []ContextSearchRecord{
		{RowID: first, RecordType: "message"}, {RowID: second, RecordType: "message"},
	}}
	target := &countingTarget{}
	collections := map[string]string{}
	for _, kind := range ContextSearchRecordKinds {
		collections[kind] = "Collection"
	}
	acts := PublishContextSearchActivities{
		Source: source, Embedder: noEmbedder{}, Target: target, Collections: collections,
		Matches: fixedMatches{first.String(): true, second.String(): true},
	}
	result, err := acts.PublishContextSearch(context.Background(), proffer.StageRequest{
		RequestID: "req", SourceVersionRef: "sv", Refs: map[string]proffer.Ref{
			"normalized_generation": "ng", "normalized_verification": "nv", "extraction_attempt": "ea",
			"participant_resolution": "pr", "message_matches": "mm",
		},
	})
	if err != nil {
		t.Fatalf("PublishContextSearch error = %v", err)
	}
	if result.Status != proffer.StatusSuccess || target.published != 0 || source.outcome.SkippedMatched != 2 || source.outcome.Published != 0 {
		t.Fatalf("result = %+v, published = %d, outcome = %+v", result, target.published, source.outcome)
	}

	// A run that passes a match-up receipt to a worker that cannot read it fails closed.
	acts.Matches = nil
	source.records = []ContextSearchRecord{{RowID: first, RecordType: "message"}}
	if _, err := acts.PublishContextSearch(context.Background(), proffer.StageRequest{
		RequestID: "req", SourceVersionRef: "sv", Refs: map[string]proffer.Ref{
			"normalized_generation": "ng", "normalized_verification": "nv", "extraction_attempt": "ea",
			"participant_resolution": "pr", "message_matches": "mm",
		},
	}); err == nil {
		t.Fatal("a match-up receipt without a reader was accepted")
	}
}
