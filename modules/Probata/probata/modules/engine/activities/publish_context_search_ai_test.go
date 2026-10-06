// Byline: Codex · GPT-6-Sol · 2026-10-05.
package activities

import (
	"context"
	"reflect"
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

// TestAINeutralSearchNeedsNoHumanResolutionAndKeepsDatesRoles verifies the complete AI Activity payload.
// Inputs: verified AI declared/raw formats, source times/role labels and a human message skip list. Outputs: assertions.
// Effects: in-memory target/receipt writes only. Choose to prove no owner alias, match suppression or invented tier is needed.
func TestAINeutralSearchNeedsNoHumanResolutionAndKeepsDatesRoles(t *testing.T) {
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
		if err != nil || result.Status != proffer.StatusSuccess {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		objects := target.published["Chats"]
		if len(objects) != 1 || source.outcome.DisclosureBasis != "" || len(source.outcome.Tiers) != 0 || source.outcome.SkippedMatched != 0 {
			t.Fatalf("objects=%+v outcome=%+v", objects, source.outcome)
		}
		o := objects[0]
		if !reflect.DeepEqual(o.People.RoleLabels, []string{"assistant", "user"}) || !reflect.DeepEqual(o.People.Participants, []string{"assistant", "user"}) || o.Temporal.DisclosureTier != "" || o.Temporal.DisclosureTierBasis != "" || !o.Temporal.OccurredAt.Equal(occurred) || !o.Temporal.KnowledgeTime.Equal(occurred.Add(time.Hour)) || o.Temporal.TimestampCertainty != "exact" || o.Temporal.OccurredAtRaw != reader.records[0].OccurredAtRaw || o.Provenance.RawFormatID != plan.FormatID {
			t.Fatalf("AI source fields changed: %+v", o)
		}
		if err := o.Validate(); err != nil {
			t.Fatal(err)
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
