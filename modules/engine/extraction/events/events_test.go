// Byline: Claude Code · Opus 5.5 · 2026-09-25

package events

import (
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

func ts(value string) *time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return &parsed
}

var owner = entities.Actor{SubjectUID: "uid-1", Username: "owner"}

func TestOwnerMarkIsDatedByTheSourceRecordNotByNow(t *testing.T) {
	record := entities.MessageView{RecordID: "r-7", Ordinal: 7, OccurredAt: ts("2025-06-01T14:46:00Z"),
		SourceAvailableFrom: ts("2026-09-21T01:14:39Z"), Body: "Pickup moved to the school at 3"}
	now := time.Date(2026, 9, 25, 23, 0, 0, 0, time.UTC)
	event, err := FromRecord(record, MarkRequest{RecordID: "r-7"}, entities.RunScope{GenerationID: "gen-1"}, owner, now)
	if err != nil {
		t.Fatal(err)
	}
	if event.OccurredAt == nil || !event.OccurredAt.Equal(*record.OccurredAt) {
		t.Fatalf("event must carry the message's own time, got %v", event.OccurredAt)
	}
	if event.DetectedBy != entities.DetectedOwner || event.TemporalPrecision != PrecisionPoint {
		t.Fatalf("owner mark provenance lost: %+v", event)
	}
	if got := event.SourceAvailableFrom(); got == nil || !got.Equal(*record.SourceAvailableFrom) {
		t.Fatalf("availability must come from the source record, got %v", got)
	}
	if event.Title != "Pickup moved to the school at 3" {
		t.Fatalf("default title = %q", event.Title)
	}
	if _, err := FromRecord(entities.MessageView{RecordID: "r-8"}, MarkRequest{RecordID: "r-8"}, entities.RunScope{}, owner, now); err == nil {
		t.Fatal("a record without its own time cannot date an event")
	}
}

func TestSourceAvailabilityIsTheLatestAndFailsClosed(t *testing.T) {
	event := Proposal{SourceRecords: []SourceRecord{
		{RecordID: "a", SourceAvailableFrom: ts("2026-01-01T00:00:00Z")},
		{RecordID: "b", SourceAvailableFrom: ts("2026-03-01T00:00:00Z")},
	}}
	if got := event.SourceAvailableFrom(); !got.Equal(*ts("2026-03-01T00:00:00Z")) {
		t.Fatalf("an event is visible only once every source is: got %v", got)
	}
	event.SourceRecords = append(event.SourceRecords, SourceRecord{RecordID: "c"})
	if event.SourceAvailableFrom() != nil {
		t.Fatal("a source without a clock must fail closed")
	}
}

func TestEventCorrections(t *testing.T) {
	now := time.Date(2026, 9, 25, 23, 0, 0, 0, time.UTC)
	a := Proposal{CandidateID: "a", Title: "Court date", EventType: "court", OccurredAt: ts("2025-07-01T09:00:00Z"),
		SourceRecords: []SourceRecord{{RecordID: "r1", Ordinal: 1}}, EntityKeys: []string{"name:person:katherine"}, ReviewState: entities.StatePending}
	b := Proposal{CandidateID: "b", Title: "Hearing", EventType: "court", OccurredAt: ts("2025-06-30T09:00:00Z"),
		SourceRecords: []SourceRecord{{RecordID: "r2", Ordinal: 2}}, ReviewState: entities.StatePending, DetectedBy: entities.DetectedOwner}
	current := map[string]Proposal{"a": a, "b": b}
	title := "Custody hearing"
	when := time.Date(2025, 7, 2, 13, 30, 0, 0, time.UTC)
	edited, err := ApplyCorrection(current, CorrectionRequest{Op: OpEdit, CandidateIDs: []string{"a"}, Title: &title, OccurredAt: &when, SetEntities: true, EntityKeys: []string{}}, owner, now, "d")
	if err != nil {
		t.Fatal(err)
	}
	got := edited.Insert[0]
	if got.Title != title || !got.OccurredAt.Equal(when) || len(got.EntityKeys) != 0 || got.Correction.Actor != owner {
		t.Fatalf("edit not applied/attributed: %+v", got)
	}
	merged, err := ApplyCorrection(current, CorrectionRequest{Op: OpMerge, CandidateIDs: []string{"a", "b"}}, owner, now, "d2")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Insert[0].SourceRecords) != 2 || !merged.Insert[0].OccurredAt.Equal(*b.OccurredAt) || merged.Insert[0].DetectedBy != entities.DetectedOwner {
		t.Fatalf("merge must keep both sources, the earliest time and the owner marker: %+v", merged.Insert[0])
	}
	bad := "not-a-type"
	if _, err := ApplyCorrection(current, CorrectionRequest{Op: OpEdit, CandidateIDs: []string{"a"}, EventType: &bad}, owner, now, "d3"); err == nil {
		t.Fatal("unknown event types are refused")
	}
}

func TestResolveEntityKeysThroughAliases(t *testing.T) {
	person := entities.Proposal{CandidateID: "p", Name: "Katherine", RegistryType: "person", ReviewState: entities.StatePending}
	person.AddAlias(entities.NewNameAlias("Kat", entities.AliasNickname, entities.SourceModel, 0.7))
	event := Proposal{EntityKeys: []string{"name:person:kat", "name:person:nobody"}}
	resolved, unresolved, ambiguous := ResolveEntityKeys(event, []entities.Proposal{person})
	if resolved["name:person:kat"] != "p" || len(unresolved) != 1 || len(ambiguous) != 0 {
		t.Fatalf("resolved=%v unresolved=%v ambiguous=%v", resolved, unresolved, ambiguous)
	}
}
