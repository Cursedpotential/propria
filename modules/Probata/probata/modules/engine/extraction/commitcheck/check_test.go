// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package commitcheck

import (
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
)

func when(value string) *time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return &parsed
}

func goodSnapshot() Snapshot {
	person := entities.Proposal{CandidateID: "p1", Name: "Katherine", RegistryType: "person", ReviewState: entities.StatePending, GenerationID: "gen-1"}
	person.AddAlias(entities.NewAddressAlias(entities.NormalizeAddress("+18105550101"), entities.SourceParticipant, ""))
	person.AddAlias(entities.NewNameAlias("Kat", entities.AliasNickname, entities.SourceModel, 0.7))
	self := entities.Proposal{CandidateID: "p2", Name: "Device owner (this source)", RegistryType: "person", ReviewState: entities.StatePending, GenerationID: "gen-1", SourceOwner: true}
	self.AddAlias(entities.NewAddressAlias(entities.NormalizeAddress("self"), entities.SourceParticipant, "source:src-1"))
	event := events.Proposal{CandidateID: "v1", Title: "School pickup", EventType: "custody_exchange", OccurredAt: when("2025-06-01T15:00:00Z"),
		TemporalPrecision: events.PrecisionPoint, GenerationID: "gen-1", ReviewState: entities.StatePending,
		SourceRecords: []events.SourceRecord{{RecordID: "r1", SourceAvailableFrom: when("2026-09-21T01:14:39Z")}},
		EntityKeys:    []string{"name:person:kat"}}
	return Snapshot{MatterMode: "LIVE", PreviewHandle: "h", CurrentGenerationID: "gen-1",
		Entities: []entities.Proposal{person, self}, Events: []events.Proposal{event}}
}

func failing(report Report) map[string]string {
	out := map[string]string{}
	for _, check := range report.Checks {
		if check.Status == Fail {
			out[check.Rule] = check.Reason
		}
	}
	return out
}

func TestValidSnapshotPassesEveryRule(t *testing.T) {
	report := Validate(goodSnapshot())
	if !report.OK {
		t.Fatalf("expected ok, failing: %v", failing(report))
	}
	if report.Counts.NewEntities != 2 || report.Counts.Events != 1 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	// The scoped "self" alias and the entity's own name are never written.
	if report.Counts.Aliases != 2 {
		t.Fatalf("writable aliases = %d, want 2 (number + Kat)", report.Counts.Aliases)
	}
	if report.Digest == "" || report.Digest != Validate(goodSnapshot()).Digest {
		t.Fatal("digest must be stable for an unchanged proposal set")
	}
}

func TestTestModeNeverCommits(t *testing.T) {
	snapshot := goodSnapshot()
	snapshot.MatterMode = "TEST"
	report := Validate(snapshot)
	if report.OK || failing(report)["live_mode"] == "" {
		t.Fatalf("TEST-mode proposals must not reach the canonical registry: %v", failing(report))
	}
}

func TestDuplicateOfCommittedEntityFails(t *testing.T) {
	snapshot := goodSnapshot()
	snapshot.Registry = []entities.RegistryEntity{{ID: "e-9", DisplayName: "Katherine D.", RegistryType: "person", NormalizedName: "katherine d.",
		Aliases: []entities.RegistryAlias{{Text: "+18105550101", Kind: entities.AliasHandle}}}}
	report := Validate(snapshot)
	if reason := failing(report)["no_duplicate_committed"]; !strings.Contains(reason, "+18105550101") {
		t.Fatalf("an alias already committed to another entity must fail: %v", failing(report))
	}
	// Merging into that entity instead passes.
	snapshot.Entities[0].Match = &entities.Match{EntityID: "e-9", DisplayName: "Katherine D."}
	if reason := failing(Validate(snapshot))["no_duplicate_committed"]; reason != "" {
		t.Fatalf("merging into the owner of the alias must pass, got %q", reason)
	}
}

func TestAliasClaimedTwiceAndDeadMatchFail(t *testing.T) {
	snapshot := goodSnapshot()
	twin := entities.Proposal{CandidateID: "p3", Name: "Kathy", RegistryType: "person", ReviewState: entities.StatePending, GenerationID: "gen-1",
		Match: &entities.Match{EntityID: "gone"}}
	twin.AddAlias(entities.NewNameAlias("Kat", entities.AliasNickname, entities.SourceOwner, 1))
	snapshot.Entities = append(snapshot.Entities, twin)
	fails := failing(Validate(snapshot))
	if fails["aliases_unique"] == "" || fails["match_targets_live"] == "" || fails["event_entities_resolve"] == "" {
		t.Fatalf("expected alias clash, dead match and ambiguous event entity: %v", fails)
	}
}

func TestEventRulesAndStaleness(t *testing.T) {
	snapshot := goodSnapshot()
	snapshot.Events[0].OccurredAt = nil
	snapshot.Events[0].SourceRecords[0].SourceAvailableFrom = nil
	snapshot.Entities[0].GenerationID = "gen-0"
	snapshot.MissingRecords = []string{"r-x"}
	fails := failing(Validate(snapshot))
	for _, rule := range []string{"events_have_time", "events_source_clock", "current_generation", "mentions_resolve"} {
		if fails[rule] == "" {
			t.Fatalf("rule %s should fail: %v", rule, fails)
		}
	}
}

func TestRejectedProposalsAreExcludedNotBlocking(t *testing.T) {
	snapshot := goodSnapshot()
	snapshot.Entities[1].ReviewState = entities.StateRejected
	report := Validate(snapshot)
	if !report.OK || report.Counts.Rejected != 1 || report.Counts.Entities != 1 {
		t.Fatalf("rejected rows are simply left out: ok=%v counts=%+v fails=%v", report.OK, report.Counts, failing(report))
	}
}
