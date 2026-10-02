// Byline: Claude Code · Opus 5 · 2026-09-26
//
// The first-party context thread store: the only writer of
// working.first_party_context_thread and its version / membership / source
// assertion rows.
//
// ONE TRANSACTION, AND WHY IT CANNOT BE SPLIT. working.first_party_context_thread
// _version, _message and _source each carry a CONSTRAINT TRIGGER that is
// DEFERRABLE INITIALLY DEFERRED and calls
// working.validate_first_party_context_thread_version. Deferred means it runs at
// COMMIT, and that validator requires the version's membership to already exist:
// it recomputes count(*), min(occurred_at) and max(occurred_at) over
// _message and raises "first-party thread bounds must equal its message
// occurred_at bounds" when a version stands alone. So a version, its membership
// and its source assertions MUST be inserted in a single transaction. Splitting
// them into one activity per table — which is this package's usual convention,
// see normalized_pipeline.go — would make each fragment fail on its own commit.
// Do not "simplify" CommitVersion into fanned-out writes; it is one write.
//
// NO UPDATE IN THIS FILE: it writes a whole new version. platform_runtime holds
// UPDATE on these four tables (owner, OD-07, 2026-10-01 07:17: "so a thread can
// be extended in place"); its one user is first_party_context_store.go, which
// extends a thread's current version with a later chunk's messages in any
// review state (owner 2026-10-02: nothing is immutable until it is promoted to
// evidence; no append-only triggers outside evidence).
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (re-conformed to the 10-02 rule)
//
// The whole boundary is proven live under platform_runtime in
// sql/validation/2026-09-26-d04-first-party-thread-role-privileges-test.sql, and
// the write contract in
// sql/validation/2026-09-26-d04-first-party-thread-projection-test.sql.
//
// IDENTITY IS NEVER INVENTED. MatterModeForIdentity decides whether the supplied
// matter/court-case pair is an admitted identity (D-125/D-126: the pre-launch DEV
// sentinel is TEST, the go-live identity is REAL, anything else is neither). An
// unadmitted identity is refused here rather than written, and appending to an
// existing thread verifies that thread's own owner, matter and court case match
// what the caller claims. Fabricated provenance on evidence is worse than a
// refused import.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/contextthread"
)

// FirstPartyThreadStore writes first-party context thread versions.
type FirstPartyThreadStore struct {
	db DB
}

// NewFirstPartyThreadStore requires a database.
func NewFirstPartyThreadStore(db DB) (*FirstPartyThreadStore, error) {
	if db == nil {
		return nil, errors.New("first-party thread store requires a database")
	}
	return &FirstPartyThreadStore{db: db}, nil
}

// insertThreadSQL creates the thread and takes the id back from the uuidv7()
// column default. case_key is left to its default: its CHECK pins it to
// 'primary', so passing anything would only be a way to get it wrong.
const insertThreadSQL = `
	INSERT INTO working.first_party_context_thread (owner_person_id, matter_id, court_case_id)
	VALUES ($1::uuid, $2::uuid, $3::uuid)
	RETURNING context_thread_id::text`

// insertThreadWithIDSQL is the retry-safe form: the caller names the thread, so a
// second attempt after a lost result does not create a second thread.
const insertThreadWithIDSQL = `
	INSERT INTO working.first_party_context_thread
	    (context_thread_id, owner_person_id, matter_id, court_case_id)
	VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
	ON CONFLICT (context_thread_id) DO NOTHING
	RETURNING context_thread_id::text`

// selectThreadIdentitySQL reads back the identity a thread was created with, so
// a version can never be appended to a thread belonging to someone else.
const selectThreadIdentitySQL = `
	SELECT owner_person_id::text, matter_id::text, court_case_id::text
	FROM working.first_party_context_thread
	WHERE context_thread_id = $1::uuid`

// selectExistingVersionSQL is the idempotency probe: the natural key of a
// version is (context_thread_id, version_ordinal), which the schema enforces
// with a unique constraint.
const selectExistingVersionSQL = `
	SELECT id::text, assertion_digest
	FROM working.first_party_context_thread_version
	WHERE context_thread_id = $1::uuid AND version_ordinal = $2`

const insertVersionSQL = `
	INSERT INTO working.first_party_context_thread_version
	    (context_thread_id, version_ordinal, classifier_id, classifier_version, assertion_digest,
	     confidence, review_state, first_occurred_at, last_occurred_at, knowledge_available_from,
	     supersedes_id, reviewed_by, reviewed_at, rationale)
	VALUES ($1::uuid, $2, $3, $4, $5::bytea,
	        $6, $7, $8, $9, $10,
	        $11::uuid, $12, $13, $14)
	RETURNING id::text`

// insertMembershipSQL writes occurred_at into source_available_from as well:
// first_party_context_thread_message_check forces them equal, so there is only
// one value and no opportunity for them to drift.
const insertMembershipSQL = `
	INSERT INTO working.first_party_context_thread_message
	    (thread_version_id, context_thread_id, message_id, thread_ordinal, occurred_at,
	     source_available_from, required_for_horizon, membership_confidence)
	VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $5, $6, $7)`

// insertSourceSQL writes coverage_last_occurred_at into source_available_from as
// well, for the same reason: first_party_context_thread_source_check1 forces them
// equal.
const insertSourceSQL = `
	INSERT INTO working.first_party_context_thread_source
	    (thread_version_id, context_thread_id, source_version_id, source_anchor_ordinal, platform,
	     platform_conversation_key, representation_kind, capture_kind, declared_format,
	     originating_device_id, perspective_person_id, coverage_first_occurred_at,
	     coverage_last_occurred_at, coverage_message_count, source_available_from,
	     required_for_horizon, metadata_clock_kind, metadata_timestamp, metadata_timezone,
	     metadata_clock_basis, metadata_confidence, metadata_review_state, metadata_ambiguity,
	     raw_metadata, metadata_extractor_id, metadata_extractor_version, assertion_version,
	     confidence, review_state, supersedes_id, provenance_digest, asserted_by)
	VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5,
	        $6, $7, $8, $9,
	        $10::uuid, $11::uuid, $12,
	        $13, $14, $13,
	        $15, $16, $17, $18,
	        $19, $20, $21, $22,
	        $23::jsonb, $24, $25, $26,
	        $27, $28, $29::uuid, $30::bytea, $31)`

// CommitVersion writes one whole thread-version snapshot in a single transaction.
//
// A retried attempt that finds this exact version already written returns it with
// AlreadyPresent set and writes nothing. A retry whose assertion digest differs
// at the same ordinal is a changed proposal, not a retry, and is refused.
func (s *FirstPartyThreadStore) CommitVersion(
	ctx context.Context,
	commit contextthread.VersionCommit,
) (contextthread.CommitResult, error) {
	if err := commit.Validate(); err != nil {
		return contextthread.CommitResult{}, err
	}
	mode, admitted := MatterModeForIdentity(commit.Identity.MatterID, commit.Identity.CourtCaseID)
	if !admitted {
		// Neither the DEV sentinel nor the go-live identity. Refuse rather than
		// write a thread scoped to a case the platform does not admit.
		return contextthread.CommitResult{}, fmt.Errorf(
			"matter %s with court case %s is not an admitted platform identity; first-party thread writes fail closed rather than guess",
			commit.Identity.MatterID, commit.Identity.CourtCaseID)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return contextthread.CommitResult{}, fmt.Errorf("begin first-party thread transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := s.commitVersionTx(ctx, tx, commit, mode)
	if err != nil {
		return contextthread.CommitResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return contextthread.CommitResult{}, fmt.Errorf("commit first-party thread version %s: %w", result.ThreadVersionID, err)
	}
	return result, nil
}

// commitVersionTx is CommitVersion inside a caller's transaction, which the
// caller commits. The first-party context import uses it so the thread is
// created under the same conversation lock as its lookup.
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (split out of CommitVersion, behavior unchanged)
func (s *FirstPartyThreadStore) commitVersionTx(
	ctx context.Context,
	tx pgx.Tx,
	commit contextthread.VersionCommit,
	mode string,
) (contextthread.CommitResult, error) {
	threadID, created, err := s.resolveThread(ctx, tx, commit)
	if err != nil {
		return contextthread.CommitResult{}, err
	}

	if !created {
		existing, found, err := s.existingVersion(ctx, tx, threadID, commit)
		if err != nil {
			return contextthread.CommitResult{}, err
		}
		if found {
			// Already written by an earlier attempt: report it as a no-op.
			return contextthread.CommitResult{
				ContextThreadID: threadID,
				ThreadVersionID: existing,
				AlreadyPresent:  true,
				MembersWritten:  0,
				SourcesWritten:  0,
				Bounds:          commit.DeriveBounds(),
				MatterMode:      mode,
			}, nil
		}
	}

	// The bounds and horizon are derived from the rows written in this same
	// transaction, never taken from the caller, so the version row cannot
	// disagree with its own membership. The deferred validator recomputes both.
	bounds := commit.DeriveBounds()

	var versionID string
	if err := tx.QueryRow(ctx, insertVersionSQL,
		threadID, commit.VersionOrdinal, commit.ClassifierID, commit.ClassifierVersion, commit.AssertionDigest,
		commit.Confidence, commit.ReviewState, bounds.FirstOccurredAt, bounds.LastOccurredAt, bounds.KnowledgeAvailableFrom,
		nullableUUID(commit.SupersedesID), nullableText(commit.ReviewedBy), commit.ReviewedAt, nullableText(commit.Rationale),
	).Scan(&versionID); err != nil {
		return contextthread.CommitResult{}, fmt.Errorf("insert first-party thread version: %w", err)
	}

	members := commit.OrderedMembers()
	for _, member := range members {
		if _, err := tx.Exec(ctx, insertMembershipSQL,
			versionID, threadID, member.MessageID, member.Ordinal, member.OccurredAt,
			member.RequiredForHorizon, member.MembershipConfidence,
		); err != nil {
			return contextthread.CommitResult{}, fmt.Errorf(
				"insert first-party thread membership for message %s at ordinal %d: %w",
				member.MessageID, member.Ordinal, err)
		}
	}

	sources := commit.OrderedSources()
	for _, source := range sources {
		if err := insertThreadSource(ctx, tx, versionID, threadID, commit.Identity.PerspectivePersonID, source); err != nil {
			return contextthread.CommitResult{}, err
		}
	}

	// Run the deferred completeness validator NOW rather than at commit, so a
	// contract failure surfaces here with its own error instead of arriving as an
	// opaque commit failure.
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		return contextthread.CommitResult{}, fmt.Errorf(
			"first-party thread version %s failed its completeness validation: %w", versionID, err)
	}

	return contextthread.CommitResult{
		ContextThreadID: threadID,
		ThreadVersionID: versionID,
		ThreadCreated:   created,
		MembersWritten:  len(members),
		SourcesWritten:  len(sources),
		Bounds:          bounds,
		MatterMode:      mode,
	}, nil
}

// resolveThread returns the thread this version belongs to, creating it when the
// caller named none. When the caller DID name one, its stored identity must match
// what the caller claims: a version is never appended to another person's thread.
func (s *FirstPartyThreadStore) resolveThread(
	ctx context.Context,
	tx pgx.Tx,
	commit contextthread.VersionCommit,
) (string, bool, error) {
	identity := commit.Identity
	if commit.ContextThreadID == "" {
		var threadID string
		if err := tx.QueryRow(ctx, insertThreadSQL,
			identity.OwnerPersonID, identity.MatterID, identity.CourtCaseID,
		).Scan(&threadID); err != nil {
			return "", false, fmt.Errorf("create first-party context thread: %w", err)
		}
		return threadID, true, nil
	}

	var threadID string
	err := tx.QueryRow(ctx, insertThreadWithIDSQL,
		commit.ContextThreadID, identity.OwnerPersonID, identity.MatterID, identity.CourtCaseID,
	).Scan(&threadID)
	switch {
	case err == nil:
		return threadID, true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return "", false, fmt.Errorf("create first-party context thread %s: %w", commit.ContextThreadID, err)
	}

	// ON CONFLICT DO NOTHING returned no row, so the thread already exists.
	// Verify it is the same thread the caller thinks it is.
	var owner, matter, courtCase string
	if err := tx.QueryRow(ctx, selectThreadIdentitySQL, commit.ContextThreadID).Scan(&owner, &matter, &courtCase); err != nil {
		return "", false, fmt.Errorf("read first-party context thread %s: %w", commit.ContextThreadID, err)
	}
	if owner != identity.OwnerPersonID || matter != identity.MatterID || courtCase != identity.CourtCaseID {
		return "", false, fmt.Errorf(
			"context thread %s belongs to owner %s in matter %s case %s, not to the identity this commit claims",
			commit.ContextThreadID, owner, matter, courtCase)
	}
	return commit.ContextThreadID, false, nil
}

// existingVersion reports whether this exact version was already written. A row
// at the same ordinal carrying a different assertion digest is a changed
// proposal, not a retry, and is refused rather than silently duplicated at the
// next free ordinal.
func (s *FirstPartyThreadStore) existingVersion(
	ctx context.Context,
	tx pgx.Tx,
	threadID string,
	commit contextthread.VersionCommit,
) (string, bool, error) {
	var versionID string
	var digest []byte
	err := tx.QueryRow(ctx, selectExistingVersionSQL, threadID, commit.VersionOrdinal).Scan(&versionID, &digest)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("probe first-party thread version %d of %s: %w", commit.VersionOrdinal, threadID, err)
	}
	if !bytesEqual(digest, commit.AssertionDigest) {
		return "", false, fmt.Errorf(
			"context thread %s already has version %d with a different assertion digest; this is a changed proposal, not a retry",
			threadID, commit.VersionOrdinal)
	}
	return versionID, true, nil
}

// insertThreadSource writes one source assertion row of a thread version.
func insertThreadSource(ctx context.Context, tx pgx.Tx, versionID, threadID, perspectivePersonID string, source contextthread.SourceAssertion) error {
	if _, err := tx.Exec(ctx, insertSourceSQL,
		versionID, threadID, source.SourceVersionID, source.AnchorOrdinal, source.Platform,
		source.PlatformConversationKey, source.RepresentationKind, source.CaptureKind, source.DeclaredFormat,
		nullableUUID(source.OriginatingDeviceID), perspectivePersonID, source.CoverageFirstOccurredAt,
		source.CoverageLastOccurredAt, source.CoverageMessageCount,
		source.RequiredForHorizon, source.MetadataClockKind, source.MetadataTimestamp, nullableText(source.MetadataTimezone),
		source.MetadataClockBasis, source.MetadataConfidence, source.MetadataReviewState, nullableText(source.MetadataAmbiguity),
		rawMetadataOrEmpty(source.RawMetadata), source.MetadataExtractorID, source.MetadataExtractorVersion, source.AssertionVersion,
		source.Confidence, source.ReviewState, nullableUUID(source.SupersedesID), source.ProvenanceDigest, source.AssertedBy,
	); err != nil {
		return fmt.Errorf(
			"insert first-party thread source assertion for source version %s at anchor %d: %w",
			source.SourceVersionID, source.AnchorOrdinal, err)
	}
	return nil
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// nullableText sends NULL for an absent optional text value rather than an empty
// string, which several CHECK constraints would reject as blank.
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// nullableUUID sends NULL for an absent optional reference.
func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// rawMetadataOrEmpty keeps raw_metadata a JSON object: the column is NOT NULL
// with a jsonb_typeof = 'object' CHECK.
func rawMetadataOrEmpty(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}
