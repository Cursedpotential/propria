// Byline: Claude Code · Opus 5 · 2026-09-26
//
// Package contextthread defines the first-party context thread: THE
// cross-platform model of one human conversation carried across SMS, iMessage,
// Facebook and anything else, as working.first_party_context_thread and its
// version / membership / source assertion rows.
//
// APPEND-ONLY BY PRIVILEGE. The engine connects as platform_runtime, which holds
// SELECT and INSERT on the four tables and no UPDATE at all. That is intent, not
// an oversight: the same role carries explicit column-scoped UPDATE grants on
// working.extraction_run and working.content_chunk_generation, so the schema
// author grants UPDATE narrowly where they mean it. Nothing in this package or
// its store ever updates a thread row. An approval, a reclassification and a
// corrected source assertion are all APPENDS carrying SupersedesID, and currency
// is derived — the row nothing supersedes — never stamped onto an older row.
// Proven live under platform_runtime in
// sql/validation/2026-09-26-d04-first-party-thread-role-privileges-test.sql.
//
// The consequence is worth stating: the exact version the owner was shown when
// he approved stays permanently readable. An in-place update would have
// destroyed the record of what he consented to, which on custody-case evidence
// is the difference between an auditable approval and an assertion that one
// happened.
//
// This package is pure: it validates a proposed commit and derives the bounds
// and knowledge horizon the deferred database validator will recompute. It holds
// no SQL and no identity constants — postgres.MatterModeForIdentity owns the
// fail-closed check of WHICH matter/court-case identity is admitted.
package contextthread

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Review states shared by a thread version and a source assertion.
const (
	ReviewProposed   = "proposed"
	ReviewApproved   = "approved"
	ReviewRejected   = "rejected"
	ReviewSuperseded = "superseded"
)

var reviewStates = map[string]bool{
	ReviewProposed: true, ReviewApproved: true, ReviewRejected: true, ReviewSuperseded: true,
}

var representationKinds = map[string]bool{
	"native_export": true, "screenshot": true, "ocr_derived": true, "pdf": true, "html": true,
	"json": true, "xml": true, "csv": true, "other": true,
}

var metadataClockKinds = map[string]bool{
	"screenshot_capture": true, "export_created": true, "filesystem_observed": true, "other": true,
}

var metadataReviewStates = map[string]bool{
	"unreviewed": true, "approved": true, "rejected": true, "ambiguous": true,
}

// DigestBytes is the exact width every assertion and provenance digest must be.
const DigestBytes = 32

// Identity is the fail-closed identity a first-party thread write requires. Every
// field is mandatory and nothing here is ever defaulted, derived from a path, or
// inferred from a platform key: fabricated provenance on evidence is worse than a
// refused import. The matter and court case must additionally be an identity the
// platform admits, which postgres.MatterModeForIdentity decides.
type Identity struct {
	OwnerPersonID       string
	MatterID            string
	CourtCaseID         string
	PerspectivePersonID string
}

// Validate checks that all four identities are present and UUID-shaped.
func (i Identity) Validate() error {
	for _, field := range []struct{ name, value string }{
		{"owner person", i.OwnerPersonID},
		{"matter", i.MatterID},
		{"court case", i.CourtCaseID},
		{"perspective person", i.PerspectivePersonID},
	} {
		trimmed := strings.TrimSpace(field.value)
		if trimmed == "" {
			return fmt.Errorf("first-party thread identity requires an explicit %s id", field.name)
		}
		if _, err := uuid.Parse(trimmed); err != nil {
			return fmt.Errorf("first-party thread %s id %q is not a uuid", field.name, field.value)
		}
	}
	return nil
}

// Member is one message's membership in one thread version. MessageID is the
// working.message id, which by message_id_fkey IS the working.normalized_record
// id — it is copied, never minted, so this carries whatever the spine writer
// produced without caring how it was generated.
//
// OccurredAt nil means the message's clock is unknown. The database forces
// source_available_from to equal occurred_at, so a required member with an
// unknown clock needs a primary_fallback relative-time anchor before the
// deferred validator will accept its version. This package refuses that case
// loudly rather than attempting a write that fails with a generic message;
// anchors are a separate concern with their own review path.
type Member struct {
	MessageID            string
	Ordinal              int64
	OccurredAt           *time.Time
	MembershipConfidence float64
	RequiredForHorizon   bool
}

// SourceAssertion is one source's claim about this thread version: which
// context.source_version it came from, whose perspective it is, what it covers,
// and the metadata provenance behind that claim.
//
// CoverageLastOccurredAt doubles as the source's availability: the database
// forces source_available_from to equal it. A required source with no coverage
// end is refused here for the same reason a clockless required member is.
type SourceAssertion struct {
	SourceVersionID          string
	AnchorOrdinal            int64
	Platform                 string
	PlatformConversationKey  string
	RepresentationKind       string
	CaptureKind              string
	DeclaredFormat           string
	OriginatingDeviceID      string
	CoverageFirstOccurredAt  *time.Time
	CoverageLastOccurredAt   *time.Time
	CoverageMessageCount     *int64
	RequiredForHorizon       bool
	MetadataClockKind        string
	MetadataTimestamp        *time.Time
	MetadataTimezone         string
	MetadataClockBasis       string
	MetadataConfidence       *float64
	MetadataReviewState      string
	MetadataAmbiguity        string
	RawMetadata              []byte
	MetadataExtractorID      string
	MetadataExtractorVersion string
	AssertionVersion         int
	Confidence               float64
	ReviewState              string
	SupersedesID             string
	ProvenanceDigest         []byte
	AssertedBy               string
}

// VersionCommit is one whole thread-version snapshot: the version row, its
// ordered membership and its source assertions, all of which land in a single
// transaction because the completeness triggers are DEFERRABLE INITIALLY
// DEFERRED and fire at COMMIT. A version written without its membership is
// rejected, so these cannot be split across activities.
//
// ContextThreadID empty means the thread row does not exist yet and the store
// creates it, taking the id back from the uuidv7() column default. A non-empty
// value appends a version to an existing thread.
type VersionCommit struct {
	ContextThreadID   string
	Identity          Identity
	VersionOrdinal    int
	ClassifierID      string
	ClassifierVersion string
	AssertionDigest   []byte
	Confidence        float64
	ReviewState       string
	// SupersedesID is the version this one replaces. An approval, a
	// reclassification and a repair are all appends that set this.
	SupersedesID string
	ReviewedBy   string
	ReviewedAt   *time.Time
	Rationale    string
	Members      []Member
	Sources      []SourceAssertion
}

// Bounds are the three timestamps the database recomputes and compares against
// the version row. The store derives them from the members and sources in the
// same commit rather than trusting a caller-supplied copy, so a version row can
// never disagree with the rows written beside it.
type Bounds struct {
	FirstOccurredAt        *time.Time
	LastOccurredAt         *time.Time
	KnowledgeAvailableFrom *time.Time
}

// DeriveBounds computes the version's occurrence bounds and knowledge horizon.
//
// FirstOccurredAt and LastOccurredAt are min and max of every member's
// occurred_at, ignoring unknown clocks exactly as SQL min/max ignore NULL.
//
// KnowledgeAvailableFrom is the GREATEST availability across required members
// and required sources — the moment every required part of this version was
// available. Validate rejects an unknown clock on anything required, so by the
// time this runs the horizon is always exact.
func (c VersionCommit) DeriveBounds() Bounds {
	var bounds Bounds
	for _, member := range c.Members {
		if member.OccurredAt == nil {
			continue
		}
		at := *member.OccurredAt
		if bounds.FirstOccurredAt == nil || at.Before(*bounds.FirstOccurredAt) {
			bounds.FirstOccurredAt = &at
		}
		if bounds.LastOccurredAt == nil || at.After(*bounds.LastOccurredAt) {
			bounds.LastOccurredAt = &at
		}
		if member.RequiredForHorizon {
			if bounds.KnowledgeAvailableFrom == nil || at.After(*bounds.KnowledgeAvailableFrom) {
				bounds.KnowledgeAvailableFrom = &at
			}
		}
	}
	for _, source := range c.Sources {
		if !source.RequiredForHorizon || source.CoverageLastOccurredAt == nil {
			continue
		}
		at := *source.CoverageLastOccurredAt
		if bounds.KnowledgeAvailableFrom == nil || at.After(*bounds.KnowledgeAvailableFrom) {
			bounds.KnowledgeAvailableFrom = &at
		}
	}
	return bounds
}

// Validate refuses a commit the database would reject, and refuses one the
// database would ACCEPT but that carries invented provenance. Every rule here
// corresponds to a constraint proven live in
// sql/validation/2026-09-26-d04-first-party-thread-projection-test.sql.
func (c VersionCommit) Validate() error {
	if err := c.Identity.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.ContextThreadID) != "" {
		if _, err := uuid.Parse(strings.TrimSpace(c.ContextThreadID)); err != nil {
			return fmt.Errorf("context thread id %q is not a uuid", c.ContextThreadID)
		}
	}
	if c.VersionOrdinal <= 0 {
		return fmt.Errorf("thread version ordinal must be positive, got %d", c.VersionOrdinal)
	}
	if strings.TrimSpace(c.ClassifierID) == "" || strings.TrimSpace(c.ClassifierVersion) == "" {
		return errors.New("thread version requires a classifier id and version")
	}
	if len(c.AssertionDigest) != DigestBytes {
		return fmt.Errorf("thread version assertion digest must be %d bytes, got %d", DigestBytes, len(c.AssertionDigest))
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("thread version confidence must be within [0,1], got %v", c.Confidence)
	}
	if !reviewStates[c.ReviewState] {
		return fmt.Errorf("thread version review state %q is not one of proposed/approved/rejected/superseded", c.ReviewState)
	}
	// The database CHECK refuses an approval that names nobody. Refuse it here
	// too, with a message that says what is missing.
	if c.ReviewState == ReviewApproved && (strings.TrimSpace(c.ReviewedBy) == "" || c.ReviewedAt == nil) {
		return errors.New("an approved thread version requires both the reviewer's identity and the time they approved")
	}
	if c.ReviewState != ReviewApproved && (strings.TrimSpace(c.ReviewedBy) != "" || c.ReviewedAt != nil) {
		return fmt.Errorf("a %s thread version must not carry reviewer attribution", c.ReviewState)
	}
	if strings.TrimSpace(c.SupersedesID) != "" {
		if _, err := uuid.Parse(strings.TrimSpace(c.SupersedesID)); err != nil {
			return fmt.Errorf("superseded thread version id %q is not a uuid", c.SupersedesID)
		}
	}
	if err := c.validateMembers(); err != nil {
		return err
	}
	return c.validateSources()
}

func (c VersionCommit) validateMembers() error {
	if len(c.Members) == 0 {
		// The deferred validator rejects a membership-free version, so this
		// cannot be deferred to the database as a "maybe".
		return errors.New("a thread version requires at least one message; the database rejects a version with no membership")
	}
	ordinals := make(map[int64]bool, len(c.Members))
	messages := make(map[string]bool, len(c.Members))
	for _, member := range c.Members {
		id := strings.TrimSpace(member.MessageID)
		if id == "" {
			return errors.New("thread membership requires a message id")
		}
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("thread membership message id %q is not a uuid", member.MessageID)
		}
		if messages[id] {
			return fmt.Errorf("message %s appears twice in one thread version", id)
		}
		messages[id] = true
		if member.Ordinal < 0 {
			return fmt.Errorf("thread ordinal must not be negative, got %d", member.Ordinal)
		}
		if ordinals[member.Ordinal] {
			return fmt.Errorf("thread ordinal %d is used twice in one version", member.Ordinal)
		}
		ordinals[member.Ordinal] = true
		if member.MembershipConfidence < 0 || member.MembershipConfidence > 1 {
			return fmt.Errorf("membership confidence must be within [0,1], got %v", member.MembershipConfidence)
		}
		if member.RequiredForHorizon && member.OccurredAt == nil {
			return fmt.Errorf("message %s is required for the horizon but has no known clock; it needs a primary_fallback relative-time anchor before this version can be committed", id)
		}
	}
	return nil
}

func (c VersionCommit) validateSources() error {
	if len(c.Sources) == 0 {
		return errors.New("a thread version requires at least one source assertion; the database rejects a version with no required source")
	}
	anchors := make(map[int64]bool, len(c.Sources))
	assertions := make(map[string]bool, len(c.Sources))
	required := 0
	for _, source := range c.Sources {
		sourceVersionID := strings.TrimSpace(source.SourceVersionID)
		if sourceVersionID == "" {
			return errors.New("a source assertion requires the selected context.source_version id")
		}
		if _, err := uuid.Parse(sourceVersionID); err != nil {
			return fmt.Errorf("source version id %q is not a uuid", source.SourceVersionID)
		}
		if source.AnchorOrdinal < 0 {
			return fmt.Errorf("source anchor ordinal must not be negative, got %d", source.AnchorOrdinal)
		}
		if anchors[source.AnchorOrdinal] {
			return fmt.Errorf("source anchor ordinal %d is used twice in one version", source.AnchorOrdinal)
		}
		anchors[source.AnchorOrdinal] = true
		if source.AssertionVersion <= 0 {
			return fmt.Errorf("source assertion version must be positive, got %d", source.AssertionVersion)
		}
		key := fmt.Sprintf("%s|%d", sourceVersionID, source.AssertionVersion)
		if assertions[key] {
			return fmt.Errorf("source %s already has assertion version %d in this thread version", sourceVersionID, source.AssertionVersion)
		}
		assertions[key] = true
		for _, field := range []struct{ name, value string }{
			{"platform", source.Platform},
			{"platform conversation key", source.PlatformConversationKey},
			{"capture kind", source.CaptureKind},
			{"declared format", source.DeclaredFormat},
			{"metadata clock basis", source.MetadataClockBasis},
			{"metadata extractor id", source.MetadataExtractorID},
			{"metadata extractor version", source.MetadataExtractorVersion},
			{"asserter", source.AssertedBy},
		} {
			if strings.TrimSpace(field.value) == "" {
				return fmt.Errorf("a source assertion requires a %s", field.name)
			}
		}
		if !representationKinds[source.RepresentationKind] {
			return fmt.Errorf("source representation kind %q is not one of the accepted kinds", source.RepresentationKind)
		}
		if !metadataClockKinds[source.MetadataClockKind] {
			return fmt.Errorf("source metadata clock kind %q is not one of screenshot_capture/export_created/filesystem_observed/other", source.MetadataClockKind)
		}
		if !metadataReviewStates[source.MetadataReviewState] {
			return fmt.Errorf("source metadata review state %q is not one of unreviewed/approved/rejected/ambiguous", source.MetadataReviewState)
		}
		if !reviewStates[source.ReviewState] {
			return fmt.Errorf("source review state %q is not one of proposed/approved/rejected/superseded", source.ReviewState)
		}
		if len(source.ProvenanceDigest) != DigestBytes {
			return fmt.Errorf("source provenance digest must be %d bytes, got %d", DigestBytes, len(source.ProvenanceDigest))
		}
		if source.Confidence < 0 || source.Confidence > 1 {
			return fmt.Errorf("source confidence must be within [0,1], got %v", source.Confidence)
		}
		if source.MetadataConfidence != nil && (*source.MetadataConfidence < 0 || *source.MetadataConfidence > 1) {
			return fmt.Errorf("source metadata confidence must be within [0,1], got %v", *source.MetadataConfidence)
		}
		if source.CoverageMessageCount != nil && *source.CoverageMessageCount < 0 {
			return fmt.Errorf("source coverage message count must not be negative, got %d", *source.CoverageMessageCount)
		}
		if source.CoverageFirstOccurredAt != nil && source.CoverageLastOccurredAt != nil &&
			source.CoverageLastOccurredAt.Before(*source.CoverageFirstOccurredAt) {
			return errors.New("source coverage must not end before it starts")
		}
		if strings.TrimSpace(source.OriginatingDeviceID) != "" {
			if _, err := uuid.Parse(strings.TrimSpace(source.OriginatingDeviceID)); err != nil {
				return fmt.Errorf("originating device id %q is not a uuid", source.OriginatingDeviceID)
			}
		}
		if strings.TrimSpace(source.SupersedesID) != "" {
			if _, err := uuid.Parse(strings.TrimSpace(source.SupersedesID)); err != nil {
				return fmt.Errorf("superseded source assertion id %q is not a uuid", source.SupersedesID)
			}
		}
		if source.RequiredForHorizon {
			required++
			if source.CoverageLastOccurredAt == nil {
				return fmt.Errorf("source %s is required for the horizon but declares no coverage end; it needs a primary_fallback relative-time anchor before this version can be committed", sourceVersionID)
			}
		}
	}
	if required == 0 {
		return errors.New("a thread version requires at least one source that is required for the horizon")
	}
	return nil
}

// OrderedMembers returns the membership sorted by thread ordinal, so the store
// writes a thread in its own order rather than the caller's map iteration order.
func (c VersionCommit) OrderedMembers() []Member {
	ordered := make([]Member, len(c.Members))
	copy(ordered, c.Members)
	sort.Slice(ordered, func(a, b int) bool { return ordered[a].Ordinal < ordered[b].Ordinal })
	return ordered
}

// OrderedSources returns the source assertions sorted by anchor ordinal.
func (c VersionCommit) OrderedSources() []SourceAssertion {
	ordered := make([]SourceAssertion, len(c.Sources))
	copy(ordered, c.Sources)
	sort.Slice(ordered, func(a, b int) bool { return ordered[a].AnchorOrdinal < ordered[b].AnchorOrdinal })
	return ordered
}

// CommitResult names the rows one commit wrote.
type CommitResult struct {
	ContextThreadID string
	ThreadVersionID string
	ThreadCreated   bool
	MembersWritten  int
	SourcesWritten  int
	Bounds          Bounds
	MatterMode      string
}
