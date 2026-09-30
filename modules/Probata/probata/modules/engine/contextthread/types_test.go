// Byline: Claude Code · Opus 5 · 2026-09-26
package contextthread

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func ptr[T any](value T) *T { return &value }

func at(spec string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, spec)
	if err != nil {
		panic(err)
	}
	return &parsed
}

// The ids below are UUID v7 shaped (version nibble 7) and obviously synthetic.
// The matter and court case are the D-126 pre-launch DEV sentinels; whether they
// are ADMITTED is postgres.MatterModeForIdentity's decision, not this package's.
const (
	testOwner       = "0d040000-0000-7000-8000-00000000e1e1"
	testMatter      = "deadbeef-dead-beef-dead-beefdeadbeef"
	testCourtCase   = "cafebabe-cafe-babe-cafe-babecafebabe"
	testMessageOne  = "0d040000-0000-7000-8000-000000000dd1"
	testMessageTwo  = "0d040000-0000-7000-8000-000000000dd2"
	testSourceVer   = "0d040000-0000-7000-8000-000000000cd2"
	testThreadVer   = "0d040000-0000-7000-8000-000000000fe1"
	testSourceAssrt = "0d040000-0000-7000-8000-000000000fd1"
)

func digest(fill byte) []byte { return bytes.Repeat([]byte{fill}, DigestBytes) }

func validCommit() VersionCommit {
	return VersionCommit{
		Identity: Identity{
			OwnerPersonID:       testOwner,
			MatterID:            testMatter,
			CourtCaseID:         testCourtCase,
			PerspectivePersonID: testOwner,
		},
		VersionOrdinal:    1,
		ClassifierID:      "first-party-thread-classifier",
		ClassifierVersion: "1.0.0",
		AssertionDigest:   digest(0xbb),
		Confidence:        0.9,
		ReviewState:       ReviewProposed,
		Members: []Member{
			{MessageID: testMessageOne, Ordinal: 0, OccurredAt: at("2026-03-01T10:00:00Z"),
				MembershipConfidence: 1, RequiredForHorizon: true},
			{MessageID: testMessageTwo, Ordinal: 1, OccurredAt: at("2026-03-01T11:30:00Z"),
				MembershipConfidence: 1, RequiredForHorizon: true},
		},
		Sources: []SourceAssertion{{
			SourceVersionID:          testSourceVer,
			AnchorOrdinal:            0,
			Platform:                 "sms",
			PlatformConversationKey:  "thread-key-1",
			RepresentationKind:       "native_export",
			CaptureKind:              "device_export",
			DeclaredFormat:           "smsbackuprestore_xml",
			CoverageFirstOccurredAt:  at("2026-03-01T10:00:00Z"),
			CoverageLastOccurredAt:   at("2026-03-01T11:30:00Z"),
			CoverageMessageCount:     ptr(int64(2)),
			RequiredForHorizon:       true,
			MetadataClockKind:        "export_created",
			MetadataClockBasis:       "export header declared creation time",
			MetadataReviewState:      "unreviewed",
			MetadataExtractorID:      "sbv-export-header",
			MetadataExtractorVersion: "1.0.0",
			AssertionVersion:         1,
			Confidence:               0.9,
			ReviewState:              ReviewProposed,
			ProvenanceDigest:         digest(0xcc),
			AssertedBy:               "first-party-thread-activity",
		}},
	}
}

func TestValidCommitPasses(t *testing.T) {
	if err := validCommit().Validate(); err != nil {
		t.Fatalf("a valid commit was rejected: %v", err)
	}
}

// Identity is never defaulted or derived. Each missing piece must be named.
func TestIdentityIsMandatoryAndUUIDShaped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mutit func(*Identity)
		want  string
	}{
		{"no owner", func(i *Identity) { i.OwnerPersonID = "" }, "owner person"},
		{"no matter", func(i *Identity) { i.MatterID = "" }, "matter"},
		{"no court case", func(i *Identity) { i.CourtCaseID = "" }, "court case"},
		{"no perspective", func(i *Identity) { i.PerspectivePersonID = "" }, "perspective person"},
		{"matter is the primary literal", func(i *Identity) { i.MatterID = "primary" }, "not a uuid"},
		{"owner is a name", func(i *Identity) { i.OwnerPersonID = "owner" }, "not a uuid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commit := validCommit()
			tc.mutit(&commit.Identity)
			err := commit.Validate()
			if err == nil {
				t.Fatal("an incomplete identity was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

// The deferred validator rejects a version with no membership, and one with no
// required source. Neither can be deferred to the database as a maybe.
func TestVersionRequiresMembershipAndARequiredSource(t *testing.T) {
	commit := validCommit()
	commit.Members = nil
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "at least one message") {
		t.Fatalf("a membership-free version was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources = nil
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "at least one source assertion") {
		t.Fatalf("a source-free version was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources[0].RequiredForHorizon = false
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "required for the horizon") {
		t.Fatalf("a version with no required source was accepted or misreported: %v", err)
	}
}

func TestOrdinalsAndMessagesAreUniquePerVersion(t *testing.T) {
	commit := validCommit()
	commit.Members[1].Ordinal = 0
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "ordinal 0 is used twice") {
		t.Fatalf("a duplicate thread ordinal was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Members[1].MessageID = testMessageOne
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "appears twice") {
		t.Fatalf("a duplicated message was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources = append(commit.Sources, commit.Sources[0])
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "anchor ordinal 0 is used twice") {
		t.Fatalf("a duplicate source anchor ordinal was accepted or misreported: %v", err)
	}
}

// An unknown clock on anything required needs a relative-time anchor first. The
// refusal has to name that, because the database's own message does not.
func TestUnknownClockOnARequiredPartIsRefusedWithTheAnchorNamed(t *testing.T) {
	commit := validCommit()
	commit.Members[0].OccurredAt = nil
	err := commit.Validate()
	if err == nil || !strings.Contains(err.Error(), "primary_fallback") {
		t.Fatalf("a clockless required member was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources[0].CoverageLastOccurredAt = nil
	err = commit.Validate()
	if err == nil || !strings.Contains(err.Error(), "primary_fallback") {
		t.Fatalf("a required source with no coverage end was accepted or misreported: %v", err)
	}
}

// A member that is NOT required may have an unknown clock: it simply does not
// constrain the horizon.
func TestUnknownClockOnANonRequiredMemberIsAllowed(t *testing.T) {
	commit := validCommit()
	commit.Members = append(commit.Members, Member{
		MessageID: "0d040000-0000-7000-8000-000000000dd3", Ordinal: 2,
		MembershipConfidence: 0.5, RequiredForHorizon: false,
	})
	if err := commit.Validate(); err != nil {
		t.Fatalf("a non-required member with an unknown clock was rejected: %v", err)
	}
}

func TestApprovalRequiresAttributionAndNothingElseCarriesIt(t *testing.T) {
	commit := validCommit()
	commit.ReviewState = ReviewApproved
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "reviewer's identity") {
		t.Fatalf("an unattributed approval was accepted or misreported: %v", err)
	}

	commit.ReviewedBy = "owner"
	commit.ReviewedAt = at("2026-03-04T12:00:00Z")
	if err := commit.Validate(); err != nil {
		t.Fatalf("an attributed approval was rejected: %v", err)
	}

	// Attribution on a proposal would claim a review that did not happen.
	commit = validCommit()
	commit.ReviewedBy = "owner"
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "must not carry reviewer attribution") {
		t.Fatalf("a proposal carrying attribution was accepted or misreported: %v", err)
	}
}

func TestDigestsAreExactlyThirtyTwoBytes(t *testing.T) {
	commit := validCommit()
	commit.AssertionDigest = digest(0xbb)[:16]
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "assertion digest must be 32 bytes") {
		t.Fatalf("a short assertion digest was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources[0].ProvenanceDigest = nil
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "provenance digest must be 32 bytes") {
		t.Fatalf("a missing provenance digest was accepted or misreported: %v", err)
	}
}

func TestClosedVocabulariesAreEnforced(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mutit func(*VersionCommit)
		want  string
	}{
		{"review state", func(c *VersionCommit) { c.ReviewState = "pending" }, "review state"},
		{"representation kind", func(c *VersionCommit) { c.Sources[0].RepresentationKind = "zip" }, "representation kind"},
		{"metadata clock kind", func(c *VersionCommit) { c.Sources[0].MetadataClockKind = "guessed" }, "metadata clock kind"},
		{"metadata review state", func(c *VersionCommit) { c.Sources[0].MetadataReviewState = "ok" }, "metadata review state"},
		{"source review state", func(c *VersionCommit) { c.Sources[0].ReviewState = "done" }, "source review state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commit := validCommit()
			tc.mutit(&commit)
			err := commit.Validate()
			if err == nil {
				t.Fatalf("an out-of-vocabulary %s was accepted", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestASourceAssertionRequiresItsSelectedSourceVersion(t *testing.T) {
	commit := validCommit()
	commit.Sources[0].SourceVersionID = ""
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "context.source_version") {
		t.Fatalf("a source assertion with no selected source version was accepted or misreported: %v", err)
	}
}

// DeriveBounds must reproduce what the deferred validator recomputes: bounds are
// min/max over ALL membership ignoring unknown clocks, and the horizon is the
// greatest availability across required members and required sources.
func TestDeriveBoundsMatchesTheValidatorsArithmetic(t *testing.T) {
	commit := validCommit()
	bounds := commit.DeriveBounds()
	if bounds.FirstOccurredAt == nil || !bounds.FirstOccurredAt.Equal(*at("2026-03-01T10:00:00Z")) {
		t.Fatalf("first occurred at = %v", bounds.FirstOccurredAt)
	}
	if bounds.LastOccurredAt == nil || !bounds.LastOccurredAt.Equal(*at("2026-03-01T11:30:00Z")) {
		t.Fatalf("last occurred at = %v", bounds.LastOccurredAt)
	}
	if bounds.KnowledgeAvailableFrom == nil || !bounds.KnowledgeAvailableFrom.Equal(*at("2026-03-01T11:30:00Z")) {
		t.Fatalf("knowledge available from = %v", bounds.KnowledgeAvailableFrom)
	}
}

// A source whose coverage ends after the last message pushes the horizon out:
// the thread was not fully knowable until that export existed.
func TestASourceCanPushTheHorizonPastTheLastMessage(t *testing.T) {
	commit := validCommit()
	commit.Sources[0].CoverageLastOccurredAt = at("2026-03-05T00:00:00Z")
	bounds := commit.DeriveBounds()
	if !bounds.LastOccurredAt.Equal(*at("2026-03-01T11:30:00Z")) {
		t.Fatalf("a source must not move the message bounds: %v", bounds.LastOccurredAt)
	}
	if !bounds.KnowledgeAvailableFrom.Equal(*at("2026-03-05T00:00:00Z")) {
		t.Fatalf("knowledge available from = %v; want the source coverage end", bounds.KnowledgeAvailableFrom)
	}
}

// An unknown clock on a non-required member must not drag the bounds to nil.
func TestUnknownClocksAreIgnoredByTheBoundsExactlyAsSQLIgnoresNull(t *testing.T) {
	commit := validCommit()
	commit.Members = append(commit.Members, Member{
		MessageID: "0d040000-0000-7000-8000-000000000dd3", Ordinal: 2,
		MembershipConfidence: 0.5, RequiredForHorizon: false,
	})
	bounds := commit.DeriveBounds()
	if bounds.FirstOccurredAt == nil || bounds.LastOccurredAt == nil {
		t.Fatal("an unknown clock nulled the bounds")
	}
	if !bounds.LastOccurredAt.Equal(*at("2026-03-01T11:30:00Z")) {
		t.Fatalf("last occurred at = %v", bounds.LastOccurredAt)
	}
}

func TestOrderedMembersAndSourcesSortByOrdinal(t *testing.T) {
	commit := validCommit()
	commit.Members[0], commit.Members[1] = commit.Members[1], commit.Members[0]
	ordered := commit.OrderedMembers()
	if ordered[0].Ordinal != 0 || ordered[1].Ordinal != 1 {
		t.Fatalf("members not ordered: %d, %d", ordered[0].Ordinal, ordered[1].Ordinal)
	}
	// The caller's slice must not be reordered underneath it.
	if commit.Members[0].Ordinal != 1 {
		t.Fatal("OrderedMembers mutated the caller's slice")
	}

	commit.Sources = append(commit.Sources, commit.Sources[0])
	commit.Sources[1].AnchorOrdinal = 1
	commit.Sources[0], commit.Sources[1] = commit.Sources[1], commit.Sources[0]
	orderedSources := commit.OrderedSources()
	if orderedSources[0].AnchorOrdinal != 0 || orderedSources[1].AnchorOrdinal != 1 {
		t.Fatalf("sources not ordered: %d, %d", orderedSources[0].AnchorOrdinal, orderedSources[1].AnchorOrdinal)
	}
}

func TestConfidenceAndOrdinalRanges(t *testing.T) {
	commit := validCommit()
	commit.Confidence = 1.5
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "confidence must be within") {
		t.Fatalf("an out-of-range confidence was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.VersionOrdinal = 0
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "ordinal must be positive") {
		t.Fatalf("a zero version ordinal was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Sources[0].AssertionVersion = 0
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "assertion version must be positive") {
		t.Fatalf("a zero assertion version was accepted or misreported: %v", err)
	}

	commit = validCommit()
	commit.Members[0].MembershipConfidence = -0.1
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "membership confidence") {
		t.Fatalf("a negative membership confidence was accepted or misreported: %v", err)
	}
}

func TestCoverageMustNotEndBeforeItStarts(t *testing.T) {
	commit := validCommit()
	commit.Sources[0].CoverageFirstOccurredAt = at("2026-03-02T00:00:00Z")
	commit.Sources[0].CoverageLastOccurredAt = at("2026-03-01T00:00:00Z")
	if err := commit.Validate(); err == nil || !strings.Contains(err.Error(), "end before it starts") {
		t.Fatalf("inverted coverage was accepted or misreported: %v", err)
	}
}
