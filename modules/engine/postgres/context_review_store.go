// Byline: Claude Code · Opus 5.5 · 2026-09-26
//
// Durable context review overlays (contextreview.Store): to / about / about the
// child / relevant as append-only revisions, and the hindsight-only
// foreshadowing flag in its own rows.
//
// KNOWLEDGE HORIZON (AGENTS.md "WHY THIS EXISTS"): the foreshadowing flag is a
// hindsight fact. The as-lived read path is a PRE-filter by construction — its
// read plan holds no statement that touches context.record_foreshadowing_flag,
// so no row of it can ever be fetched, cached or leaked on that path. Only an
// explicit HorizonHindsight read adds the flag statement.
//
// Writers serialize on a transaction-scoped advisory lock keyed by the record
// and matter. SELECT ... FOR UPDATE would need UPDATE privilege, which the
// engine role deliberately lacks on append-only tables (see
// proffer_preview_store.go, previewHandleLockNote).
package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cursedpotential/probata/engine/contextreview"
)

// ContextReviewStore persists and reads context review overlays.
type ContextReviewStore struct {
	db    DB
	clock func() time.Time
}

// NewContextReviewStore requires a database.
func NewContextReviewStore(db DB) (*ContextReviewStore, error) {
	if db == nil {
		return nil, errors.New("context review store requires a database")
	}
	return &ContextReviewStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// contextReviewSubjectSQL resolves a Review message to its normalized record
// and the run's matter scope. The record must belong to a normalized
// generation this preview handle actually published.
const contextReviewSubjectSQL = `
	SELECT record.id::text, version.matter_id::text, version.court_case_id::text
	FROM context.proffer_preview_binding binding
	JOIN context.normalized_record_identity record ON record.id = $2::uuid
	JOIN context.source_version version ON version.id = record.source_version_id
	WHERE binding.preview_handle = $1
	  AND EXISTS (
	      SELECT 1 FROM context.proffer_preview_snapshot snapshot
	      WHERE snapshot.preview_handle = binding.preview_handle
	        AND snapshot.normalized_generation_id = record.normalized_generation_id)`

// contextReviewHistorySQL never names the foreshadowing table.
const contextReviewHistorySQL = `
	SELECT review_ref::text, revision, addressed_to, about, about_child, relevant,
	       change_reason, actor_username, receipt_ref, recorded_at
	FROM context.record_context_review_revision
	WHERE normalized_record_id = $1::uuid AND matter_id = $2::uuid
	ORDER BY revision DESC
	LIMIT $3`

const foreshadowingHistorySQL = `
	SELECT flag_ref::text, revision, foreshadowing, note, horizon, knowledge_time,
	       change_reason, actor_username, receipt_ref
	FROM context.record_foreshadowing_flag
	WHERE normalized_record_id = $1::uuid AND matter_id = $2::uuid AND horizon = 'hindsight'
	ORDER BY revision DESC
	LIMIT $3`

// contextReviewReadPlan is every statement one Read executes after the subject
// is resolved. The as-lived plan has no foreshadowing statement at all.
type contextReviewReadPlan struct {
	reviews       string
	foreshadowing string
}

func planContextReviewRead(horizon contextreview.Horizon) contextReviewReadPlan {
	plan := contextReviewReadPlan{reviews: contextReviewHistorySQL}
	if horizon == contextreview.HorizonHindsight {
		plan.foreshadowing = foreshadowingHistorySQL
	}
	return plan
}

type reviewScope struct {
	recordID    string
	matterID    string
	courtCaseID string
}

func resolveReviewScope(ctx context.Context, q rowQuerier, subject contextreview.Subject) (reviewScope, error) {
	var scope reviewScope
	var matterID, courtCaseID *string
	err := q.QueryRow(ctx, contextReviewSubjectSQL, subject.PreviewHandle, subject.MessageID).Scan(&scope.recordID, &matterID, &courtCaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return reviewScope{}, contextreview.ErrSubjectNotFound
	}
	if err != nil {
		return reviewScope{}, fmt.Errorf("resolve reviewed message: %w", err)
	}
	if matterID == nil || courtCaseID == nil {
		return reviewScope{}, contextreview.ErrScopeMissing
	}
	scope.matterID, scope.courtCaseID = *matterID, *courtCaseID
	return scope, nil
}

// Read returns one record's review history (and, on the hindsight path only,
// its foreshadowing history), newest first.
func (s *ContextReviewStore) Read(ctx context.Context, subject contextreview.Subject, horizon contextreview.Horizon) (contextreview.View, error) {
	if err := contextreview.ValidateSubject(subject); err != nil {
		return contextreview.View{}, err
	}
	scope, err := resolveReviewScope(ctx, s.db, subject)
	if err != nil {
		return contextreview.View{}, err
	}
	plan := planContextReviewRead(horizon)
	view := contextreview.View{
		PreviewHandle: subject.PreviewHandle, MessageID: subject.MessageID,
		Horizon: horizon, Reviews: []contextreview.ReviewRevision{},
	}
	if view.Horizon != contextreview.HorizonHindsight {
		view.Horizon = contextreview.HorizonAsLived
	}
	rows, err := s.db.Query(ctx, plan.reviews, scope.recordID, scope.matterID, contextreview.MaxHistoryItems)
	if err != nil {
		return contextreview.View{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	for rows.Next() {
		var revision contextreview.ReviewRevision
		var addressedTo, about []byte
		if err := rows.Scan(&revision.ReviewRef, &revision.Revision, &addressedTo, &about,
			&revision.Assertions.AboutChild, &revision.Assertions.Relevant, &revision.ChangeReason,
			&revision.ActorUsername, &revision.ReceiptRef, &revision.RecordedAt); err != nil {
			rows.Close()
			return contextreview.View{}, err
		}
		if err := decodeParties(addressedTo, &revision.Assertions.AddressedTo); err != nil {
			rows.Close()
			return contextreview.View{}, err
		}
		if err := decodeParties(about, &revision.Assertions.About); err != nil {
			rows.Close()
			return contextreview.View{}, err
		}
		view.Reviews = append(view.Reviews, revision)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return contextreview.View{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	if plan.foreshadowing == "" {
		return view, nil
	}
	flags := []contextreview.ForeshadowingRevision{}
	flagRows, err := s.db.Query(ctx, plan.foreshadowing, scope.recordID, scope.matterID, contextreview.MaxHistoryItems)
	if err != nil {
		return contextreview.View{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	for flagRows.Next() {
		var flag contextreview.ForeshadowingRevision
		var horizon string
		if err := flagRows.Scan(&flag.FlagRef, &flag.Revision, &flag.Foreshadowing, &flag.Note, &horizon,
			&flag.KnowledgeTime, &flag.ChangeReason, &flag.ActorUsername, &flag.ReceiptRef); err != nil {
			flagRows.Close()
			return contextreview.View{}, err
		}
		flag.Horizon = contextreview.Horizon(horizon)
		flags = append(flags, flag)
	}
	flagRows.Close()
	if err := flagRows.Err(); err != nil {
		return contextreview.View{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	view.Foreshadowing = &flags
	return view, nil
}

// PersistReview appends one as-lived-safe review revision.
func (s *ContextReviewStore) PersistReview(ctx context.Context, spec contextreview.ReviewSpec) (contextreview.Receipt, error) {
	assertions := spec.Assertions.Normalized()
	addressedTo, err := json.Marshal(assertions.AddressedTo)
	if err != nil {
		return contextreview.Receipt{}, err
	}
	about, err := json.Marshal(assertions.About)
	if err != nil {
		return contextreview.Receipt{}, err
	}
	return s.persist(ctx, overlayWrite{
		subject: spec.Subject, supersedes: spec.SupersedesRef, key: spec.IdempotencyKey, digest: spec.ContentDigest[:],
		lockPrefix: "context-review:", horizon: contextreview.HorizonAsLived, receiptScheme: "context-review://",
		existingSQL: `SELECT review_ref::text, receipt_ref, content_digest, revision, recorded_at
			FROM context.record_context_review_revision WHERE idempotency_key = $1`,
		newestSQL: `SELECT review_ref::text, revision FROM context.record_context_review_revision
			WHERE normalized_record_id = $1::uuid AND matter_id = $2::uuid ORDER BY revision DESC LIMIT 1`,
		insert: func(tx pgx.Tx, id uuid.UUID, scope reviewScope, revision int, supersedes any, receiptRef string, at time.Time) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO context.record_context_review_revision
				    (review_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision,
				     supersedes_ref, addressed_to, about, about_child, relevant, change_reason,
				     actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref, recorded_at)
				VALUES ($1, $2::uuid, $3, $4::uuid, $5::uuid, $6, $7, $8::jsonb, $9::jsonb, $10, $11, $12,
				        $13, $14, $15, $16, $17, $18)`,
				id, scope.recordID, spec.PreviewHandle, scope.matterID, scope.courtCaseID, revision,
				supersedes, addressedTo, about, assertions.AboutChild, assertions.Relevant, spec.ChangeReason,
				spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey, spec.ContentDigest[:], receiptRef, at)
			return err
		},
	})
}

// PersistForeshadowing appends one hindsight-only foreshadowing revision.
func (s *ContextReviewStore) PersistForeshadowing(ctx context.Context, spec contextreview.ForeshadowingSpec) (contextreview.Receipt, error) {
	return s.persist(ctx, overlayWrite{
		subject: spec.Subject, supersedes: spec.SupersedesRef, key: spec.IdempotencyKey, digest: spec.ContentDigest[:],
		lockPrefix: "foreshadowing:", horizon: contextreview.HorizonHindsight, receiptScheme: "foreshadowing://",
		existingSQL: `SELECT flag_ref::text, receipt_ref, content_digest, revision, knowledge_time
			FROM context.record_foreshadowing_flag WHERE idempotency_key = $1`,
		newestSQL: `SELECT flag_ref::text, revision FROM context.record_foreshadowing_flag
			WHERE normalized_record_id = $1::uuid AND matter_id = $2::uuid ORDER BY revision DESC LIMIT 1`,
		insert: func(tx pgx.Tx, id uuid.UUID, scope reviewScope, revision int, supersedes any, receiptRef string, at time.Time) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO context.record_foreshadowing_flag
				    (flag_ref, normalized_record_id, preview_handle, matter_id, court_case_id, revision,
				     supersedes_ref, foreshadowing, horizon, knowledge_time, note, change_reason,
				     actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref)
				VALUES ($1, $2::uuid, $3, $4::uuid, $5::uuid, $6, $7, $8, 'hindsight', $9, $10, $11,
				        $12, $13, $14, $15, $16)`,
				id, scope.recordID, spec.PreviewHandle, scope.matterID, scope.courtCaseID, revision,
				supersedes, spec.Foreshadowing, at, spec.Note, spec.ChangeReason,
				spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey, spec.ContentDigest[:], receiptRef)
			return err
		},
	})
}

type overlayWrite struct {
	subject       contextreview.Subject
	supersedes    string
	key           string
	digest        []byte
	lockPrefix    string
	horizon       contextreview.Horizon
	receiptScheme string
	existingSQL   string
	newestSQL     string
	insert        func(pgx.Tx, uuid.UUID, reviewScope, int, any, string, time.Time) error
}

func (s *ContextReviewStore) persist(ctx context.Context, write overlayWrite) (contextreview.Receipt, error) {
	if err := contextreview.ValidateSubject(write.subject); err != nil {
		return contextreview.Receipt{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return contextreview.Receipt{}, err
	}
	rollback := func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }
	scope, err := resolveReviewScope(ctx, tx, write.subject)
	if err != nil {
		rollback()
		return contextreview.Receipt{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT 1 FROM (SELECT pg_advisory_xact_lock(hashtextextended($1, 0))) AS lock`,
		write.lockPrefix+scope.recordID+":"+scope.matterID).Scan(new(int)); err != nil {
		rollback()
		return contextreview.Receipt{}, fmt.Errorf("lock reviewed record: %w", err)
	}
	var existing contextreview.Receipt
	var existingDigest []byte
	err = tx.QueryRow(ctx, write.existingSQL, write.key).Scan(&existing.Ref, &existing.ReceiptRef, &existingDigest, &existing.Revision, &existing.RecordedAt)
	switch {
	case err == nil:
		rollback()
		if hex.EncodeToString(existingDigest) != hex.EncodeToString(write.digest) {
			return contextreview.Receipt{}, contextreview.ErrIdempotencyConflict
		}
		existing.ContentDigest, existing.Horizon = hex.EncodeToString(existingDigest), write.horizon
		return existing, nil
	case !errors.Is(err, pgx.ErrNoRows):
		rollback()
		return contextreview.Receipt{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	var newestRef string
	var newestRevision int
	err = tx.QueryRow(ctx, write.newestSQL, scope.recordID, scope.matterID).Scan(&newestRef, &newestRevision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		rollback()
		return contextreview.Receipt{}, overlayError(err, contextreview.ErrNotInstalled)
	}
	if write.supersedes != newestRef {
		rollback()
		return contextreview.Receipt{}, contextreview.ErrStaleRevision
	}
	var supersedes any
	if newestRef != "" {
		supersedes = newestRef
	}
	id, err := uuid.NewV7()
	if err != nil {
		rollback()
		return contextreview.Receipt{}, err
	}
	at := s.clock()
	receiptRef := write.receiptScheme + id.String()
	if err := write.insert(tx, id, scope, newestRevision+1, supersedes, receiptRef, at); err != nil {
		rollback()
		return contextreview.Receipt{}, overlayWriteError(err, contextreview.ErrNotInstalled, contextreview.ErrIdempotencyConflict)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return contextreview.Receipt{}, err
	}
	return contextreview.Receipt{
		Ref: id.String(), ReceiptRef: receiptRef, ContentDigest: hex.EncodeToString(write.digest),
		Revision: newestRevision + 1, RecordedAt: at, Horizon: write.horizon,
	}, nil
}

func decodeParties(raw []byte, into *[]contextreview.Party) error {
	parties := []contextreview.Party{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parties); err != nil {
			return fmt.Errorf("decode review parties: %w", err)
		}
	}
	*into = parties
	return nil
}

// overlayError maps "relation does not exist" (the overlay SQL is not applied
// yet) to the package's not-installed error and leaves everything else alone.
func overlayError(err, notInstalled error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return notInstalled
	}
	return err
}

// overlayWriteError additionally maps a unique violation (the same
// idempotency key already used for another subject) to the conflict error.
func overlayWriteError(err, notInstalled, conflict error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return conflict
	}
	return overlayError(err, notInstalled)
}

var _ contextreview.Store = (*ContextReviewStore)(nil)
