// Byline: Codex · GPT-6-Sol · 2026-10-05.
package activities

import (
	"context"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// aiSearchSource provides a provenance-resolved plan and captures its publication receipt.
// Inputs: synthetic verified plan. Outputs: plan and stable receipt refs. Effects: test memory only.
// Choose for the full Activity boundary; SQL tests independently exercise persisted source verification.
type aiSearchSource struct {
	plan    ContextSearchPlan
	outcome ContextSearchPublicationOutcome
}

func (s *aiSearchSource) OpenContextSearchRecords(context.Context, PublishContextSearchSpec) (ContextSearchPlan, error) {
	return s.plan, nil
}
func (s *aiSearchSource) PersistContextSearchPublication(_ context.Context, _ PublishContextSearchSpec, o ContextSearchPublicationOutcome) (proffer.Ref, proffer.Ref, error) {
	s.outcome = o
	return "publication", "receipt", nil
}

// TestAIRecordPublicationRejectsVerifiedAIFormats rejects the retired AI per-record publisher.
// Inputs: verified AI declared/raw formats, source times/role labels and a human message skip list. Outputs: assertions.
// Effects: no target or receipt writes. Choose to prove both persisted format signals close the old path.
func TestAIRecordPublicationRejectsVerifiedAIFormats(t *testing.T) {
	for _, rawOnly := range []bool{false, true} {
		plan := skipTestPlan(t, "message")
		plan.Provenance.SourceFormat = "chatgpt_official_json"
		if rawOnly {
			plan.Provenance.SourceFormat = "json"
			plan.FormatID = "claude_conversations_json"
		}
		plan.Resolution = disclosure.Resolution{}
		reader := plan.Reader.(*skipSliceReader)
		occurred := time.Date(2021, 7, 3, 14, 32, 0, 0, time.UTC)
		reader.records[0].OccurredAt = &occurred
		reader.records[0].OccurredAtRaw = "2021-07-03T14:32:00Z"
		reader.records[0].KnowledgeTime = occurred.Add(time.Hour)
		reader.records[0].TimestampCertainty = "exact"
		reader.records[0].TimestampGranularity = "second"
		reader.records[0].Participants = []ContextSearchParticipant{{Role: "assistant", Identifier: "assistant"}, {Role: "user", Identifier: "user"}}
		source := &aiSearchSource{plan: plan}
		embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
		a := skipTestActivities(embedder, target)
		a.Source = source
		req := proffer.StageRequest{RequestID: "r", SourceVersionRef: "s", DeclaredFormat: "smsbackuprestore_xml", Refs: map[string]proffer.Ref{"normalized_generation": "g", "normalized_verification": "v", "extraction_attempt": "e", "message_matches": "stale-human-match", SkipRecordKindsRefKey: "message,call"}}
		result, err := a.PublishContextSearch(context.Background(), req)
		if err == nil || result.Status == proffer.StatusSuccess || len(target.published) != 0 || len(embedder.texts) != 0 || source.outcome.Published != 0 {
			t.Fatalf("AI per-record path was not closed: result=%+v err=%v target=%v", result, err, target.published)
		}
	}
}

// TestAINeutralSearchRequestLabelCannotWaiveHumanResolution defends the trust boundary from a spoofed incoming label.
// Inputs: verified human plan and request claiming AI. Outputs: assertions. Effects: no target writes.
// Choose as the strict human regression counterpart of resolution-free AI publication.
func TestAINeutralSearchRequestLabelCannotWaiveHumanResolution(t *testing.T) {
	source := &aiSearchSource{plan: skipTestPlan(t, "message")}
	embedder, target := &skipRecordingEmbedder{}, &skipRecordingTarget{}
	a := skipTestActivities(embedder, target)
	a.Source = source
	req := proffer.StageRequest{RequestID: "r", SourceVersionRef: "s", DeclaredFormat: "chatgpt_official_json", Refs: map[string]proffer.Ref{"normalized_generation": "g", "normalized_verification": "v", "extraction_attempt": "e"}}
	if _, err := a.PublishContextSearch(context.Background(), req); err == nil || len(target.published) > 0 || len(embedder.texts) > 0 {
		t.Fatalf("human validation bypass: err=%v objects=%v", err, target.published)
	}
}
