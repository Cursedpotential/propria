// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (participant split, one disclosure rule)

package firstparty

import (
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/contextthread"
	"github.com/Cursedpotential/probata/engine/disclosure"
)

const (
	ownerPhone = "+18105550100"
	otherPhone = "+18105550199"
)

func testIdentity() contextthread.Identity {
	return contextthread.Identity{
		OwnerPersonID:       "01a0f751-e07b-76b6-afcb-63acfbba373e",
		MatterID:            "01a0f751-e07b-75cc-9ad5-63ad9449a8ba",
		CourtCaseID:         "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		PerspectivePersonID: "01a0f751-e07b-76c7-8c0f-65692ad656b8",
	}
}

// testResolution: the owner's phone is a confirmed identifier of the owner;
// the other number is unknown to the registry.
func testResolution(identity contextthread.Identity) disclosure.Resolution {
	return disclosure.Resolution{
		Basis: disclosure.Basis, OwnerPersonID: identity.OwnerPersonID, PerspectivePersonID: identity.PerspectivePersonID,
		Identifiers: []disclosure.ResolvedIdentifier{
			{Raw: ownerPhone, Normalized: "8105550100", EntityID: identity.OwnerPersonID, IsOwner: true},
			{Raw: otherPhone, Normalized: "8105550199"},
			{Raw: "+1 (810) 555-0199", Normalized: "8105550199"},
		},
	}
}

func testSource() Source {
	return Source{
		SourceVersionID: "01a0c181-02a6-7217-93ee-9512956dfc56", NormalizedGenerationID: "01a0c181-02a6-7217-93ee-9512956dfc57",
		DeclaredFormat: "ndjson", SourceKey: "b2://vault/sms.xml.derived/threads/8105550100.ndjson",
		Platform: "sms", CaptureKind: "sms_backup_restore", RepresentationKind: "json",
		DerivedFromSourceVersionID: "01a0c181-02a6-7217-93ee-9512956dfc55",
	}
}

func at(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &parsed
}

// testRecords: two messages between the device owner (Katrina, "self") and
// the case owner's phone, one between her and somebody else.
func testRecords() []SourceMessage {
	return []SourceMessage{
		{RecordID: "01a0c181-0000-7000-8000-000000000003", Ordinal: 2, OccurredAt: at("2024-11-02T10:00:00Z"),
			Sender: ownerPhone, Recipients: []string{"self"}, Parties: []string{ownerPhone, "self"}, Body: "third"},
		{RecordID: "01a0c181-0000-7000-8000-000000000001", Ordinal: 0, OccurredAt: at("2024-11-01T10:00:00Z"),
			Sender: "self", Recipients: []string{ownerPhone}, Parties: []string{"self", ownerPhone}, Body: "first"},
		{RecordID: "01a0c181-0000-7000-8000-000000000002", Ordinal: 1, OccurredAt: at("2024-11-01T11:00:00Z"),
			Sender: "self", Recipients: []string{otherPhone}, Parties: []string{"self", "+1 (810) 555-0199"}, Body: "other thread"},
	}
}

func build(t *testing.T, identity contextthread.Identity, records []SourceMessage) Plan {
	t.Helper()
	plan, err := Build(identity, testSource(), records, testResolution(identity))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return plan
}

// TestAIFormatsCannotBuildMessagingPlan rejects AI provenance even when callers supply a plausible SMS platform.
// Inputs: synthetic baseline with persisted AI formats. Outputs: assertions. Effects: none.
// Use to protect direct planning callers as well as Activity rebuilds.
func TestAIFormatsCannotBuildMessagingPlan(t *testing.T) {
	for _, format := range []string{"chatgpt_official_json", "chatgpt_json_array", "claude_conversations_json", "claude_ai_export_json", "gemini_activity_json", "ai_markdown_transcript", "ai_generic_json", "ai_conversations_json", "ai_chat_file"} {
		t.Run(format, func(t *testing.T) {
			source := testSource()
			source.DeclaredFormat = format
			if _, err := Build(testIdentity(), source, testRecords(), testResolution(testIdentity())); err == nil || !strings.Contains(err.Error(), "AI chat") {
				t.Fatalf("AI source built a messaging plan: %v", err)
			}
		})
	}
}

func TestOwnerParticipationSplitsTheGeneration(t *testing.T) {
	plan := build(t, testIdentity(), testRecords())
	first, third := plan.FirstParty(), plan.ThirdParty()
	if plan.MessageCount != 3 || len(first) != 1 || len(third) != 1 {
		t.Fatalf("plan has %d messages, %d first-party and %d third-party conversations; want 3, 1, 1", plan.MessageCount, len(first), len(third))
	}
	if first[0].Key != "8105550100" || len(first[0].Messages) != 2 {
		t.Fatalf("first-party conversation = %+v, want the owner's number with two messages", first[0])
	}
	for _, message := range first[0].Messages {
		if !message.OwnerTookPart || message.DisclosureTier != "contemporaneous" || !message.RouteApproved {
			t.Fatalf("owner-participant message = %+v, want contemporaneous, approved route", message)
		}
	}
	if first[0].Messages[0].Ordinal != 0 || first[0].Messages[1].Ordinal != 2 {
		t.Fatalf("messages out of ordinal order: %+v", first[0].Messages)
	}
	other := third[0].Messages[0]
	if third[0].Key != "8105550199" || other.OwnerTookPart || other.DisclosureTier != "discovered" || other.RouteApproved {
		t.Fatalf("third-party message = %+v in %q, want discovered with no approved route", other, third[0].Key)
	}
	if !strings.HasPrefix(first[0].ScopedKey, CorpusFirstParty+"/") || !strings.HasPrefix(third[0].ScopedKey, CorpusThirdParty+"/") {
		t.Fatalf("scoped keys %q / %q do not carry their corpus", first[0].ScopedKey, third[0].ScopedKey)
	}
	if plan.Source.AcquiredSourceVersionID() != testSource().DerivedFromSourceVersionID {
		t.Fatal("a derived chunk must file third-party conversations under the acquired backup")
	}
}

func TestParticipantsCarryTheirResolvedEntity(t *testing.T) {
	plan := build(t, testIdentity(), testRecords())
	message := plan.FirstParty()[0].Messages[0]
	if len(message.Participants) != 2 {
		t.Fatalf("participants = %+v", message.Participants)
	}
	for _, participant := range message.Participants {
		switch participant.Raw {
		case "self":
			if participant.EntityID != testIdentity().PerspectivePersonID || participant.Role != "from" {
				t.Fatalf("self = %+v, want the perspective person as sender", participant)
			}
		case ownerPhone:
			if participant.EntityID != testIdentity().OwnerPersonID || participant.Role != "to" {
				t.Fatalf("owner phone = %+v, want the owner as recipient", participant)
			}
		}
	}
	unresolved := plan.ThirdParty()[0].Messages[0].Participants[1]
	if unresolved.EntityID != "" {
		t.Fatalf("an identifier the registry does not know was given entity %q", unresolved.EntityID)
	}
}

func TestOwnDeviceIsAllFirstParty(t *testing.T) {
	identity := testIdentity()
	identity.PerspectivePersonID = identity.OwnerPersonID
	plan := build(t, identity, testRecords())
	if len(plan.ThirdParty()) != 0 || len(plan.FirstParty()) != 2 {
		t.Fatalf("own device split into %d first-party and %d third-party conversations; want everything first-party", len(plan.FirstParty()), len(plan.ThirdParty()))
	}
	for _, conversation := range plan.Conversations {
		for _, message := range conversation.Messages {
			if message.DisclosureTier != "contemporaneous" {
				t.Fatalf("message on the owner's own device = %s, want contemporaneous", message.DisclosureTier)
			}
		}
	}
}

func TestDigestIsStableAndBindsContentAndResolution(t *testing.T) {
	records := testRecords()
	plan := build(t, testIdentity(), records)
	again := build(t, testIdentity(), []SourceMessage{records[2], records[1], records[0]})
	if plan.Digest != again.Digest || len(plan.Digest) != 64 {
		t.Fatalf("digest depends on input order: %s vs %s", plan.Digest, again.Digest)
	}
	if err := again.Rebuilt(plan.Digest); err != nil {
		t.Fatalf("Rebuilt refused an identical plan: %v", err)
	}
	edited := testRecords()
	edited[0].Body = "edited after the owner saw it"
	if err := build(t, testIdentity(), edited).Rebuilt(plan.Digest); err == nil {
		t.Fatal("a changed message body kept the recorded digest")
	}
	resolution := testResolution(testIdentity())
	resolution.Identifiers[1].IsOwner = true
	moved, err := Build(testIdentity(), testSource(), testRecords(), resolution)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Digest == plan.Digest {
		t.Fatal("a different participant resolution kept the same digest")
	}
}

func TestBuildRefusesWhatItCannotDecide(t *testing.T) {
	identity := testIdentity()
	if _, err := Build(identity, testSource(), testRecords(), disclosure.Resolution{Basis: disclosure.Basis, OwnerPersonID: identity.OwnerPersonID}); err == nil {
		t.Fatal("a resolution without a perspective person was accepted")
	}
	other := testResolution(identity)
	other.OwnerPersonID = identity.PerspectivePersonID
	if _, err := Build(identity, testSource(), testRecords(), other); err == nil {
		t.Fatal("a resolution made for a different owner was accepted")
	}
	unknown := append(testRecords(), SourceMessage{RecordID: "01a0c181-0000-7000-8000-000000000004", Ordinal: 3,
		Sender: "+18105550111", Recipients: []string{"self"}, Parties: []string{"+18105550111", "self"}, Body: "?"})
	if _, err := Build(identity, testSource(), unknown, testResolution(identity)); err == nil {
		t.Fatal("an identifier missing from the resolution was guessed instead of refused")
	}
	source := testSource()
	source.Platform = ""
	if _, err := Build(identity, source, testRecords(), testResolution(identity)); err == nil {
		t.Fatal("a plan without a platform was accepted")
	}
	selfOnly := []SourceMessage{{RecordID: "01a0c181-0000-7000-8000-000000000009", Sender: "self", Parties: []string{"self"}, Body: "note"}}
	if _, err := Build(identity, testSource(), selfOnly, testResolution(identity)); err == nil {
		t.Fatal("a message naming only the device owner was given a conversation")
	}
	if _, err := Build(identity, testSource(), append(testRecords(), testRecords()[0]), testResolution(identity)); err == nil {
		t.Fatal("a record appearing twice was accepted")
	}
}

func TestNewThreadVersionSatisfiesTheThreadContract(t *testing.T) {
	plan := build(t, testIdentity(), testRecords())
	conversation := plan.FirstParty()[0]
	commit := plan.NewThreadVersion("01a0f751-0000-7000-8000-00000000aaaa", conversation)
	if err := commit.Validate(); err != nil {
		t.Fatalf("version 1 fails the thread contract: %v", err)
	}
	bounds := commit.DeriveBounds()
	if !bounds.FirstOccurredAt.Equal(*at("2024-11-01T10:00:00Z")) || !bounds.LastOccurredAt.Equal(*at("2024-11-02T10:00:00Z")) ||
		!bounds.KnowledgeAvailableFrom.Equal(*at("2024-11-02T10:00:00Z")) {
		t.Fatalf("bounds = %+v", bounds)
	}
	source := commit.Sources[0]
	if source.PlatformConversationKey != "8105550100" || *source.CoverageMessageCount != 2 || !source.RequiredForHorizon {
		t.Fatalf("source assertion = %+v", source)
	}
	if string(commit.AssertionDigest) != string(MembershipDigest([]string{
		"01a0c181-0000-7000-8000-000000000001", "01a0c181-0000-7000-8000-000000000003",
	})) {
		t.Fatal("version digest is not the membership digest of its ordered members")
	}
}

func TestClocklessMessageIsAMemberButNotRequired(t *testing.T) {
	records := testRecords()
	records[0].OccurredAt = nil
	plan := build(t, testIdentity(), records)
	commit := plan.NewThreadVersion("", plan.FirstParty()[0])
	if err := commit.Validate(); err != nil {
		t.Fatalf("a thread with one clockless member fails the contract: %v", err)
	}
	for _, member := range commit.Members {
		if member.OccurredAt == nil && member.RequiredForHorizon {
			t.Fatal("a clockless member was made required for the horizon")
		}
	}
}

func TestPlatformComesOnlyFromARegisteredDerivation(t *testing.T) {
	if platform, _, _, ok := PlatformForDerivation("smsthreads_derive", "ndjson"); !ok || platform != "sms" {
		t.Fatalf("smsthreads ndjson = %q, %v; want sms", platform, ok)
	}
	if _, _, _, ok := PlatformForDerivation("smsthreads_derive", "csv"); ok {
		t.Fatal("an unregistered format was given a platform")
	}
	if _, _, _, ok := PlatformForDerivation("", "ndjson"); ok {
		t.Fatal("ndjson with no derivation was given a platform")
	}
}
