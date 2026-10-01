// Byline: Claude Code · Opus 5.5 · 2026-10-01

package firstparty

import (
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/contextthread"
)

func testIdentity() contextthread.Identity {
	return contextthread.Identity{
		OwnerPersonID:       "01a0f751-e07b-76b6-afcb-63acfbba373e",
		MatterID:            "01a0f751-e07b-75cc-9ad5-63ad9449a8ba",
		CourtCaseID:         "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		PerspectivePersonID: "01a0f751-e07b-76c7-8c0f-65692ad656b8",
	}
}

func testSource() Source {
	return Source{
		SourceVersionID: "01a0c181-02a6-7217-93ee-9512956dfc56", NormalizedGenerationID: "01a0c181-02a6-7217-93ee-9512956dfc57",
		DeclaredFormat: "ndjson", SourceKey: "b2://vault/sms.xml.derived/threads/8105550100.ndjson",
		Platform: "sms", CaptureKind: "sms_backup_restore", RepresentationKind: "json",
	}
}

func at(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &parsed
}

func testRecords() []SourceMessage {
	return []SourceMessage{
		{RecordID: "01a0c181-0000-7000-8000-000000000003", Ordinal: 2, OccurredAt: at("2024-11-02T10:00:00Z"),
			Sender: "+18105550100", Recipients: []string{"self"}, Parties: []string{"+18105550100", "self"}, Body: "third"},
		{RecordID: "01a0c181-0000-7000-8000-000000000001", Ordinal: 0, OccurredAt: at("2024-11-01T10:00:00Z"),
			Sender: "self", Recipients: []string{"+18105550100"}, Parties: []string{"self", "+18105550100"}, Body: "first"},
		{RecordID: "01a0c181-0000-7000-8000-000000000002", Ordinal: 1, OccurredAt: at("2024-11-01T11:00:00Z"),
			Sender: "self", Recipients: []string{"+18105550199"}, Parties: []string{"self", "+1 (810) 555-0199"}, Body: "other thread"},
	}
}

func TestBuildGroupsByConversationInOrdinalOrder(t *testing.T) {
	plan, err := Build(testIdentity(), testSource(), testRecords())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if plan.MessageCount != 3 || len(plan.Conversations) != 2 {
		t.Fatalf("plan has %d messages in %d conversations, want 3 in 2", plan.MessageCount, len(plan.Conversations))
	}
	first := plan.Conversations[0]
	if first.Key != "8105550100" || len(first.Messages) != 2 {
		t.Fatalf("first conversation = %+v, want key 8105550100 with two messages", first)
	}
	if first.Messages[0].Ordinal != 0 || first.Messages[1].Ordinal != 2 {
		t.Fatalf("messages out of ordinal order: %+v", first.Messages)
	}
	if first.Messages[0].Direction != "outbound" || first.Messages[1].Direction != "inbound" {
		t.Fatalf("directions = %s, %s; want outbound then inbound from the device owner's side", first.Messages[0].Direction, first.Messages[1].Direction)
	}
	if !first.Messages[0].RouteApproved {
		t.Fatal("a message naming its sender and a recipient must have an approved first-party route")
	}
	if plan.Conversations[1].Key != "8105550199" {
		t.Fatalf("second conversation key = %q, want the normalized NANP number", plan.Conversations[1].Key)
	}
	if !strings.HasPrefix(first.ScopedKey, "first_party/"+testIdentity().MatterID+"/") || !strings.HasSuffix(first.ScopedKey, "/sms/8105550100") {
		t.Fatalf("scoped key = %q", first.ScopedKey)
	}
}

func TestDigestIsStableAndBindsContent(t *testing.T) {
	records := testRecords()
	plan, err := Build(testIdentity(), testSource(), records)
	if err != nil {
		t.Fatal(err)
	}
	reversed := []SourceMessage{records[2], records[1], records[0]}
	again, err := Build(testIdentity(), testSource(), reversed)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Digest != again.Digest || len(plan.Digest) != 64 {
		t.Fatalf("digest depends on input order: %s vs %s", plan.Digest, again.Digest)
	}
	if err := again.Rebuilt(plan.Digest); err != nil {
		t.Fatalf("Rebuilt refused an identical plan: %v", err)
	}
	edited := testRecords()
	edited[0].Body = "edited after the owner saw it"
	changed, err := Build(testIdentity(), testSource(), edited)
	if err != nil {
		t.Fatal(err)
	}
	if err := changed.Rebuilt(plan.Digest); err == nil {
		t.Fatal("a changed message body kept the recorded digest")
	}
	other := testIdentity()
	other.PerspectivePersonID = other.OwnerPersonID
	moved, err := Build(other, testSource(), testRecords())
	if err != nil {
		t.Fatal(err)
	}
	if moved.Digest == plan.Digest {
		t.Fatal("a different perspective person kept the same digest")
	}
}

func TestDisclosureTierFollowsThePerspective(t *testing.T) {
	identity := testIdentity()
	if DisclosureTierFor(identity) != DisclosureDiscovered {
		t.Fatal("another person's device must be discovered, not contemporaneous")
	}
	identity.PerspectivePersonID = identity.OwnerPersonID
	if DisclosureTierFor(identity) != DisclosureContemporaneous {
		t.Fatal("the owner's own device must be contemporaneous")
	}
}

func TestBuildRefusesMissingIdentityAndUnidentifiableConversations(t *testing.T) {
	identity := testIdentity()
	identity.PerspectivePersonID = ""
	if _, err := Build(identity, testSource(), testRecords()); err == nil {
		t.Fatal("a plan without a perspective person was accepted")
	}
	source := testSource()
	source.Platform = ""
	if _, err := Build(testIdentity(), source, testRecords()); err == nil {
		t.Fatal("a plan without a platform was accepted")
	}
	selfOnly := []SourceMessage{{RecordID: "01a0c181-0000-7000-8000-000000000009", Sender: "self", Parties: []string{"self"}, Body: "note"}}
	if _, err := Build(testIdentity(), testSource(), selfOnly); err == nil {
		t.Fatal("a message naming only the device owner was given a conversation")
	}
	duplicate := append(testRecords(), testRecords()[0])
	if _, err := Build(testIdentity(), testSource(), duplicate); err == nil {
		t.Fatal("a record appearing twice was accepted")
	}
}

func TestNewThreadVersionSatisfiesTheThreadContract(t *testing.T) {
	plan, err := Build(testIdentity(), testSource(), testRecords())
	if err != nil {
		t.Fatal(err)
	}
	commit := plan.NewThreadVersion("01a0f751-0000-7000-8000-00000000aaaa", plan.Conversations[0])
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
	plan, err := Build(testIdentity(), testSource(), records)
	if err != nil {
		t.Fatal(err)
	}
	commit := plan.NewThreadVersion("", plan.Conversations[0])
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
