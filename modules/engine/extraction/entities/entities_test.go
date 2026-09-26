// Byline: Claude Code · Opus 5.5 · 2026-09-25

package entities

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeAddress(t *testing.T) {
	cases := []struct {
		raw        string
		kind       AddressKind
		normalized string
	}{
		{"+1 (810) 555-0101", AddressPhone, "+18105550101"},
		{"8105550101", AddressPhone, "+18105550101"},
		{"18105550101", AddressPhone, "+18105550101"},
		{"555-0101", AddressPhone, "5550101"},
		{"self", AddressSelf, "self"},
		{" Someone@Example.COM ", AddressEmail, "someone@example.com"},
		{"@Handle_1", AddressHandle, "@handle_1"},
		{"87654", AddressOther, "87654"},
		{"Katherine", AddressOther, "katherine"},
	}
	for _, test := range cases {
		got := NormalizeAddress(test.raw)
		if got.Kind != test.kind || got.Normalized != test.normalized {
			t.Errorf("NormalizeAddress(%q) = %s/%q, want %s/%q", test.raw, got.Kind, got.Normalized, test.kind, test.normalized)
		}
	}
	if DisplayAddress(NormalizeAddress("8105550101")) != "+1 (810) 555-0101" {
		t.Fatalf("display = %q", DisplayAddress(NormalizeAddress("8105550101")))
	}
}

func TestRegistryNormalizedNameMatchesGovernedCreatePath(t *testing.T) {
	// server/api/entity_routes.py: " ".join(display_name.casefold().split())
	if got := RegistryNormalizedName("  Katherine   Doe "); got != "katherine doe" {
		t.Fatalf("got %q", got)
	}
}

func TestRelateNamesSpellingButNotDifferentNames(t *testing.T) {
	cases := []struct {
		a, b string
		want NameRelation
	}{
		{"Katherine", "katherine", NameSame},
		{"Katherine", "Catherine", NameSpelling}, // K/C spelling variant: one entity
		{"Karina", "Carina", NameSpelling},
		{"Philip", "Filip", NameSpelling},
		{"John", "Jon", NameSpelling},
		{"Sarah", "Sara", NameSpelling},
		{"Katie", "Katy", NameSpelling},
		{"Marc", "Mark", NameSpelling},
		{"Christina", "Kristina", NameSpelling},
		{"Katherine", "Kathryn", NameUnrelated}, // sounds alike, different name: owner decides
		{"Maria", "Mario", NameUnrelated},
		{"Jon", "Jan", NameUnrelated},
		{"Katherine", "Katherine Doe", NameContained},
		{"Mark", "Maria", NameUnrelated},
	}
	for _, test := range cases {
		if got := RelateNames(test.a, test.b); got != test.want {
			t.Errorf("RelateNames(%q,%q) = %q, want %q (keys %q %q)", test.a, test.b, got, test.want, SpellingKey(test.a), SpellingKey(test.b))
		}
	}
}

func scope() RunScope {
	return RunScope{PreviewHandle: "handle_abcdefghijklmnopqrstuvwxyz0123", GenerationID: "gen-1", SourceVersionID: "src-1"}
}

func at(value string) *time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return &parsed
}

func TestProposeFromParticipantsGroupsAddressesAndScopesSelf(t *testing.T) {
	rows := []ParticipantAggregate{
		{Identifier: "self", SenderCount: 3, RecipientCount: 2, MessageCount: 5, FirstAt: at("2025-06-01T10:00:00Z"), LastAt: at("2025-06-03T10:00:00Z")},
		{Identifier: "+18105550101", DisplayNames: []string{"Katherine"}, SenderCount: 2, RecipientCount: 3, MessageCount: 5,
			Samples: []ParticipantSeen{{RecordID: "r1", Ordinal: 1, Role: "sender"}}},
		{Identifier: "8105550101", MessageCount: 1},                                        // same number, other format
		{Identifier: "+18105550199", DisplayNames: []string{"katherine"}, MessageCount: 2}, // same contact card name
		{Identifier: "+18105550123", MessageCount: 4},
	}
	proposals := ProposeFromParticipants(scope(), rows)
	if len(proposals) != 3 {
		t.Fatalf("want 3 people (self, Katherine with two numbers, unknown number), got %d: %+v", len(proposals), proposals)
	}
	var katherine, self, unknown *Proposal
	for i := range proposals {
		switch {
		case proposals[i].SourceOwner:
			self = &proposals[i]
		case proposals[i].Name == "Katherine":
			katherine = &proposals[i]
		default:
			unknown = &proposals[i]
		}
	}
	if katherine == nil || self == nil || unknown == nil {
		t.Fatalf("missing groups: %+v", proposals)
	}
	if got := len(katherine.AddressAliases()); got != 2 {
		t.Fatalf("Katherine should carry both numbers, got %d: %+v", got, katherine.Aliases)
	}
	for _, alias := range self.AddressAliases() {
		if alias.Scope != "source:src-1" {
			t.Fatalf("self alias must be scoped to its source: %+v", alias)
		}
	}
	if keys := strings.Join(self.Keys(), ","); !strings.Contains(keys, "participant:self@source:src-1") {
		t.Fatalf("self key is not source scoped: %s", keys)
	}
	if unknown.Name != "+1 (810) 555-0123" {
		t.Fatalf("unnamed number should be named by its display form, got %q", unknown.Name)
	}
	if len(katherine.MentionSample) != 1 || katherine.MentionSample[0].Kind != "phone" {
		t.Fatalf("participant sample not kept: %+v", katherine.MentionSample)
	}
}

func TestReconcileMergesModelPartialsIntoParticipantAndSpellingVariants(t *testing.T) {
	rules := ProposeFromParticipants(scope(), []ParticipantAggregate{{Identifier: "+18105550101", MessageCount: 9}})
	rules[0].CandidateID = "c-rules"
	modelPerson := Proposal{Name: "Katherine", RegistryType: "person", Extractors: []string{"model:x@1"}, CandidateID: "c-model-1", GenerationID: "gen-1"}
	modelPerson.AddAlias(NewAddressAlias(NormalizeAddress("+18105550101"), SourceModel, "source:src-1"))
	modelPerson.AddAlias(NewNameAlias("Kat", AliasNickname, SourceModel, 0.7))
	spelling := Proposal{Name: "Catherine", RegistryType: "person", Extractors: []string{"model:x@1"}, CandidateID: "c-model-2", GenerationID: "gen-1"}
	other := Proposal{Name: "Kathryn", RegistryType: "person", Extractors: []string{"model:x@1"}, CandidateID: "c-model-3", GenerationID: "gen-1"}
	place := Proposal{Name: "Lincoln Elementary", RegistryType: "org", Extractors: []string{"model:x@1"}, CandidateID: "c-model-4", GenerationID: "gen-1"}
	plan := Reconcile(ReconcileInput{Scope: scope(), Partials: []Proposal{rules[0], modelPerson, spelling, other, place}})
	if len(plan.Insert) != 3 {
		t.Fatalf("want Katherine(+Catherine,+number,+Kat), Kathryn, school; got %d: %+v", len(plan.Insert), names(plan.Insert))
	}
	var merged *Proposal
	for i := range plan.Insert {
		if plan.Insert[i].Name == "Katherine" {
			merged = &plan.Insert[i]
		}
	}
	if merged == nil {
		t.Fatalf("no merged Katherine: %v", names(plan.Insert))
	}
	var sawNumber, sawNick, sawSpelling bool
	for _, alias := range merged.Aliases {
		switch {
		case alias.Normalized == "+18105550101":
			sawNumber = true
		case alias.Text == "Kat" && alias.Kind == AliasNickname:
			sawNick = true
		case alias.Text == "Catherine" && alias.Kind == AliasMisspelling:
			sawSpelling = true
		}
	}
	if !sawNumber || !sawNick || !sawSpelling {
		t.Fatalf("aliases not grouped: %+v", merged.Aliases)
	}
	if strings.Join(merged.Supersedes, ",") != "c-model-1,c-model-2,c-rules" {
		t.Fatalf("merged proposal must supersede its partials, got %v", merged.Supersedes)
	}
	if len(plan.Supersede) != 5 {
		t.Fatalf("every partial is superseded by its group, got %v", plan.Supersede)
	}
}

func names(proposals []Proposal) []string {
	var out []string
	for _, proposal := range proposals {
		out = append(out, proposal.Name)
	}
	return out
}

func TestReconcileKeepsOwnerEditsAndRespectsRejections(t *testing.T) {
	existing := Proposal{CandidateID: "owner-edited", Name: "Kate D.", RegistryType: "person", ReviewState: StatePending,
		Correction: &Correction{Op: OpRename, Actor: Actor{SubjectUID: "u", Username: "owner"}}, GenerationID: "gen-1"}
	existing.AddAlias(NewAddressAlias(NormalizeAddress("+18105550101"), SourceParticipant, ""))
	rejected := Proposal{CandidateID: "spam", Name: "+1 (810) 555-0199", RegistryType: "person", ReviewState: StateRejected, GenerationID: "gen-1"}
	rejected.AddAlias(NewAddressAlias(NormalizeAddress("+18105550199"), SourceParticipant, ""))

	fresh := Proposal{CandidateID: "p1", Name: "Katherine", RegistryType: "person", Extractors: []string{"model:x@1"}, GenerationID: "gen-1"}
	fresh.AddAlias(NewAddressAlias(NormalizeAddress("+18105550101"), SourceModel, ""))
	spam := Proposal{CandidateID: "p2", Name: "+1 (810) 555-0199", RegistryType: "person", Extractors: []string{RulesExtractor + "@1"}, GenerationID: "gen-1"}
	spam.AddAlias(NewAddressAlias(NormalizeAddress("+18105550199"), SourceParticipant, ""))

	plan := Reconcile(ReconcileInput{Scope: scope(), Partials: []Proposal{fresh, spam}, Existing: []Proposal{existing, rejected}})
	if len(plan.Insert) != 1 {
		t.Fatalf("want only the folded owner proposal, got %v", names(plan.Insert))
	}
	folded := plan.Insert[0]
	if folded.Name != "Kate D." {
		t.Fatalf("owner's name was overwritten: %q", folded.Name)
	}
	if !strings.Contains(strings.Join(folded.Supersedes, ","), "owner-edited") {
		t.Fatalf("folded proposal must supersede the owner's current row: %v", folded.Supersedes)
	}
	if len(plan.Dropped) != 1 || plan.Dropped[0].RejectedBy != "spam" {
		t.Fatalf("rejected entity must not be re-proposed: %+v", plan.Dropped)
	}
}

func TestReconcileMatchesCommittedEntitiesByAliasAndName(t *testing.T) {
	registry := []RegistryEntity{
		{ID: "e-1", DisplayName: "Katherine Doe", RegistryType: "person", NormalizedName: "katherine doe", Aliases: []RegistryAlias{{Text: "+18105550101", Kind: AliasHandle}}},
		{ID: "e-2", DisplayName: "Lincoln Elementary", RegistryType: "school", NormalizedName: "lincoln elementary"},
	}
	byNumber := Proposal{CandidateID: "a", Name: "+1 (810) 555-0101", RegistryType: "person", GenerationID: "gen-1"}
	byNumber.AddAlias(NewAddressAlias(NormalizeAddress("8105550101"), SourceParticipant, ""))
	byName := Proposal{CandidateID: "b", Name: "Lincoln Elementary", RegistryType: "org", GenerationID: "gen-1"}
	plan := Reconcile(ReconcileInput{Scope: scope(), Partials: []Proposal{byNumber, byName}, Registry: registry})
	for _, proposal := range plan.Insert {
		if proposal.Match == nil {
			t.Fatalf("%q was not matched to its committed entity", proposal.Name)
		}
	}
}

func TestCorrectionsRenameMergeSplitAliasRejectRestore(t *testing.T) {
	actor := Actor{SubjectUID: "uid-1", Username: "owner"}
	now := time.Date(2026, 9, 25, 20, 0, 0, 0, time.UTC)
	a := Proposal{CandidateID: "a", Name: "+1 (810) 555-0101", RegistryType: "person", ReviewState: StatePending, GenerationID: "gen-1"}
	a.AddAlias(NewAddressAlias(NormalizeAddress("+18105550101"), SourceParticipant, ""))
	a.MentionSample = []Mention{{RecordID: "r1", Surface: "+18105550101", Kind: "phone", Role: RoleSender}}
	b := Proposal{CandidateID: "b", Name: "Katherine", RegistryType: "person", ReviewState: StatePending, GenerationID: "gen-1"}
	b.AddAlias(NewNameAlias("Kat", AliasNickname, SourceModel, 0.7))
	current := map[string]Proposal{"a": a, "b": b}

	merged, err := ApplyCorrection(current, CorrectionRequest{Op: OpMerge, CandidateIDs: []string{"a", "b"}}, actor, now, "d1")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Insert) != 1 || merged.Insert[0].Name != "Katherine" || len(merged.Insert[0].Aliases) != 2 {
		t.Fatalf("merge did not combine aliases: %+v", merged.Insert)
	}
	if merged.Insert[0].Correction == nil || merged.Insert[0].Correction.Actor != actor || !merged.Insert[0].Correction.At.Equal(now) {
		t.Fatalf("merge is not attributed: %+v", merged.Insert[0].Correction)
	}
	if strings.Join(merged.Supersede, ",") != "a,b" {
		t.Fatalf("merge must supersede both: %v", merged.Supersede)
	}

	combined := merged.Insert[0]
	combined.CandidateID = "c"
	current = map[string]Proposal{"c": combined}
	renamed, err := ApplyCorrection(current, CorrectionRequest{Op: OpRename, CandidateIDs: []string{"c"}, Name: "Katherine Doe"}, actor, now, "d2")
	if err != nil {
		t.Fatal(err)
	}
	var keptOldName bool
	for _, alias := range renamed.Insert[0].Aliases {
		if alias.Text == "Katherine" {
			keptOldName = true
		}
	}
	if renamed.Insert[0].Name != "Katherine Doe" || !keptOldName {
		t.Fatalf("rename must keep the old name as an alias: %+v", renamed.Insert[0])
	}

	split, err := ApplyCorrection(current, CorrectionRequest{Op: OpSplit, CandidateIDs: []string{"c"}, Name: "Unknown caller", SplitAliases: []string{"+18105550101"}}, actor, now, "d3")
	if err != nil {
		t.Fatal(err)
	}
	if len(split.Insert) != 2 || len(split.Insert[1].AddressAliases()) != 1 || len(split.Insert[0].AddressAliases()) != 0 {
		t.Fatalf("split must move the number and its mentions: %+v", split.Insert)
	}
	if len(split.Insert[1].MentionSample) != 1 {
		t.Fatalf("mentions must follow the moved alias: %+v", split.Insert[1].MentionSample)
	}

	added, err := ApplyCorrection(current, CorrectionRequest{Op: OpAddAlias, CandidateIDs: []string{"c"}, AliasText: "Katie"}, actor, now, "d4")
	if err != nil || len(added.Insert[0].Aliases) != 3 {
		t.Fatalf("add alias: %v %+v", err, added.Insert)
	}
	if _, err := ApplyCorrection(current, CorrectionRequest{Op: OpAddAlias, CandidateIDs: []string{"c"}, AliasText: "self"}, actor, now, "d5"); err == nil {
		t.Fatal(`"self" must never become an alias`)
	}
	removed, err := ApplyCorrection(current, CorrectionRequest{Op: OpRemoveAlias, CandidateIDs: []string{"c"}, AliasText: "Kat"}, actor, now, "d6")
	if err != nil || len(removed.Insert[0].Aliases) != 1 {
		t.Fatalf("remove alias: %v %+v", err, removed.Insert)
	}

	rejected, err := ApplyCorrection(current, CorrectionRequest{Op: OpReject, CandidateIDs: []string{"c"}}, actor, now, "d7")
	if err != nil || rejected.Insert[0].ReviewState != StateRejected {
		t.Fatalf("reject: %v %+v", err, rejected.Insert)
	}
	rejectedRow := rejected.Insert[0]
	rejectedRow.CandidateID = "r"
	if _, err := ApplyCorrection(map[string]Proposal{"r": rejectedRow}, CorrectionRequest{Op: OpRename, CandidateIDs: []string{"r"}, Name: "x"}, actor, now, "d8"); err == nil {
		t.Fatal("a rejected proposal must be restored before it is edited")
	}
	restored, err := ApplyCorrection(map[string]Proposal{"r": rejectedRow}, CorrectionRequest{Op: OpRestore, CandidateIDs: []string{"r"}}, actor, now, "d9")
	if err != nil || restored.Insert[0].ReviewState != StatePending {
		t.Fatalf("restore: %v %+v", err, restored.Insert)
	}
	committed := combined
	committed.ReviewState = StateApproved
	if _, err := ApplyCorrection(map[string]Proposal{"c": committed}, CorrectionRequest{Op: OpRename, CandidateIDs: []string{"c"}, Name: "x"}, actor, now, "d10"); err == nil {
		t.Fatal("a committed proposal must not be edited here")
	}
	if _, err := ApplyCorrection(current, CorrectionRequest{Op: OpRename, CandidateIDs: []string{"missing"}, Name: "x"}, actor, now, "d11"); err == nil {
		t.Fatal("a stale id must conflict")
	}
}

func TestFindOccurrencesAreWordBoundedAndCapitalized(t *testing.T) {
	body := "will Katherine come? I told katherine. Kat said Katherine's car. Call 810-555-0101 now"
	spans := FindNameOccurrences(body, "Katherine")
	if len(spans) != 2 {
		t.Fatalf("want 2 capitalized occurrences, got %v", spans)
	}
	if len(FindNameOccurrences(body, "Will")) != 0 {
		t.Fatal(`lower-case "will" must not match the name Will`)
	}
	if len(FindNameOccurrences("Katherines are", "Katherine")) != 0 {
		t.Fatal("a name inside a longer word must not match")
	}
	digits := FindDigitsOccurrences(body, "8105550101")
	if len(digits) != 1 || string([]rune(body)[digits[0].Start:digits[0].End]) != "810-555-0101" {
		t.Fatalf("typed number not found: %v", digits)
	}
	if _, ok := FindTextOccurrence(body, "kat said"); !ok {
		t.Fatal("case-insensitive grounding failed")
	}
	if _, ok := FindTextOccurrence(body, "Kathryn"); ok {
		t.Fatal("text not in the body must not ground")
	}
}

func TestParticipantMentionsHonourSelfScope(t *testing.T) {
	proposal := Proposal{Name: "Device owner (this source)", RegistryType: "person"}
	proposal.AddAlias(NewAddressAlias(NormalizeAddress("self"), SourceParticipant, "source:src-1"))
	message := MessageView{RecordID: "r1", Participants: []MessageAddressee{{Role: "sender", Identifier: "self"}, {Role: "unknown", Identifier: "self"}}}
	if got := ParticipantMentions(message, proposal, "src-1"); len(got) != 1 || got[0].Role != RoleSender {
		t.Fatalf("one sender mention expected in its own source, got %+v", got)
	}
	if got := ParticipantMentions(message, proposal, "src-2"); len(got) != 0 {
		t.Fatalf(`"self" of one source must not resolve in another: %+v`, got)
	}
}

func TestContentHashIsStableAcrossAliasOrder(t *testing.T) {
	a := Proposal{Name: "Katherine", RegistryType: "person", GenerationID: "g"}
	a.AddAlias(NewNameAlias("Kat", AliasNickname, SourceModel, 0.7))
	a.AddAlias(NewNameAlias("Katie", AliasNickname, SourceModel, 0.7))
	b := Proposal{Name: "Katherine", RegistryType: "person", GenerationID: "g"}
	b.AddAlias(NewNameAlias("Katie", AliasNickname, SourceModel, 0.7))
	b.AddAlias(NewNameAlias("Kat", AliasNickname, SourceModel, 0.7))
	if a.ContentHex() != b.ContentHex() {
		t.Fatal("content hash must not depend on alias order (retry-safe staging)")
	}
}
