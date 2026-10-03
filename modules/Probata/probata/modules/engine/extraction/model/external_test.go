// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package model

import (
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

func externalMessages() []entities.MessageView { return testBatch().Messages }

// TestGroundExternalTagsEveryProposalWithItsExtractor proves a reply keyed by record id is validated by the shared
// schema, grounded in the window's own messages, and tagged with the extractor that made it.
func TestGroundExternalTagsEveryProposalWithItsExtractor(t *testing.T) {
	page := ExternalPage{
		Extractor: "semantica", ExtractorVersion: "1",
		People: []ExternalEntity{{Name: "Katherine", Aliases: []string{"Kat"}, Mentions: []ExternalMention{
			{RecordID: "r2", Text: "Katherine"}, {RecordID: "r2", Text: "Not in the message"}, {RecordID: "missing", Text: "Katherine"},
		}}},
		Organizations: []ExternalEntity{{Name: "Lincoln Elementary", Mentions: []ExternalMention{{RecordID: "r1", Text: "Lincoln Elementary"}}}},
		Events: []ExternalEvent{
			{Title: "Court date", RecordID: "r2", Date: strPtr("2025-07-02"), Type: "court"},
			{Title: "Odd", RecordID: "r1", Type: "not-a-type", Date: strPtr("tomorrow")},
			{Title: "Orphan", RecordID: "missing"},
		},
	}
	outcome := GroundExternal(page, externalMessages(), entities.RunScope{GenerationID: "gen-1", SourceVersionID: "sv-1", PreviewHandle: "h"})
	if outcome.Invalid {
		t.Fatalf("outcome invalid: %s", outcome.Reason)
	}
	if len(outcome.Entities) != 2 {
		t.Fatalf("entities = %d, want 2 (a person and an organization)", len(outcome.Entities))
	}
	for _, entity := range outcome.Entities {
		if len(entity.Extractors) != 1 || entity.Extractors[0] != "semantica@1" {
			t.Errorf("%s extractors = %v, want semantica@1", entity.Name, entity.Extractors)
		}
	}
	person := outcome.Entities[0]
	if person.Name != "Katherine" || len(person.ModelMentions) != 1 || !strings.HasPrefix(person.ModelMentions[0].Method, "extract:semantica@1") {
		t.Fatalf("person = %+v, want one grounded mention by method extract:semantica@1", person)
	}
	if len(outcome.Events) != 2 {
		t.Fatalf("events = %d, want 2 (the orphan event is dropped)", len(outcome.Events))
	}
	if outcome.Events[0].EventType != "court" || outcome.Events[1].EventType != "other" {
		t.Fatalf("event types = %q, %q, want court then other", outcome.Events[0].EventType, outcome.Events[1].EventType)
	}
	if outcome.Events[1].OccurredAt == nil {
		t.Fatal("an event without a valid stated date must take its message's time")
	}
	if outcome.Ungrounded < 3 {
		t.Fatalf("ungrounded = %d, want the false quote, the missing record and the orphan event counted", outcome.Ungrounded)
	}
}

// TestGroundExternalRefusesAnExtractorThatDoesNotNameItself proves untagged output is never written.
func TestGroundExternalRefusesAnExtractorThatDoesNotNameItself(t *testing.T) {
	outcome := GroundExternal(ExternalPage{}, externalMessages(), entities.RunScope{})
	if !outcome.Invalid {
		t.Fatal("an unnamed extractor's output was accepted")
	}
}

func strPtr(value string) *string { return &value }
