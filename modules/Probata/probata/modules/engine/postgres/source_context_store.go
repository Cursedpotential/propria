// Byline: Codex · GPT-5 · 2026-10-05 (pre-import operating policy)
// Byline: Codex · GPT-5.6-Sol · 2026-08-30 (append-only Proffer source context)
// Byline: Claude Code · Opus 5.5 · 2026-09-25 (read-back of current context and registration)
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

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/sourcecontext"
)

type SourceContextStore struct {
	db    DB
	clock func() time.Time
}

func NewSourceContextStore(db DB) (*SourceContextStore, error) {
	if db == nil {
		return nil, errors.New("source context store requires a database")
	}
	return &SourceContextStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *SourceContextStore) PersistSourceContext(ctx context.Context, spec sourcecontext.Spec) (sourcecontext.Receipt, error) {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(spec.OperatingMode)); err != nil {
		return sourcecontext.Receipt{}, err
	}
	if !caseidentity.AdmittedIdentity(spec.MatterID, spec.CourtCaseID) {
		return sourcecontext.Receipt{}, errors.New("source context requires the exact approved matter and court case")
	}
	observed, err := json.Marshal(spec.ObservedSource)
	if err != nil {
		return sourcecontext.Receipt{}, err
	}
	assertions, err := json.Marshal(spec.Assertions)
	if err != nil {
		return sourcecontext.Receipt{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return sourcecontext.Receipt{}, err
	}
	recordedAt := s.clock()
	receiptRef := "proffer-source-context://" + id.String()
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sourcecontext.Receipt{}, err
	}
	rollback := func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }
	revision := 1
	var supersedes any
	var previousAssertions any
	if spec.SupersedesRef != "" {
		var priorAssertions []byte
		err = tx.QueryRow(ctx, `
			SELECT revision, assertions
			FROM context.proffer_source_context_revision
			WHERE source_context_ref=$1::uuid AND request_id=$2
			  AND matter_id=$3::uuid AND court_case_id=$4::uuid
			  AND source_ref=$5 AND observed_source=$6::jsonb
			FOR UPDATE`, spec.SupersedesRef, spec.RequestID, spec.MatterID,
			spec.CourtCaseID, spec.SourceRef, observed).Scan(&revision, &priorAssertions)
		if errors.Is(err, pgx.ErrNoRows) {
			rollback()
			return sourcecontext.Receipt{}, errors.New("superseded source context does not own the same request and immutable source")
		}
		if err != nil {
			rollback()
			return sourcecontext.Receipt{}, fmt.Errorf("resolve superseded source context: %w", err)
		}
		revision++
		supersedes = spec.SupersedesRef
		previousAssertions = priorAssertions
	}
	result, err := tx.Exec(ctx, `
		INSERT INTO context.proffer_source_context_revision
		    (source_context_ref, request_id, revision, supersedes_ref, matter_id, court_case_id,
		     source_ref, observed_source, previous_assertions, assertions, change_reason,
		     actor_subject_uid, actor_username, idempotency_key, content_digest,
		     receipt_ref, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (idempotency_key) DO NOTHING`, id, spec.RequestID, revision, supersedes,
		spec.MatterID, spec.CourtCaseID, spec.SourceRef, observed, previousAssertions, assertions, spec.ChangeReason,
		spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey,
		spec.ContentDigest[:], receiptRef, recordedAt)
	if err != nil {
		rollback()
		return sourcecontext.Receipt{}, fmt.Errorf("persist source context: %w", err)
	}
	if result.RowsAffected() == 1 {
		if err := tx.Commit(ctx); err != nil {
			rollback()
			return sourcecontext.Receipt{}, err
		}
		return sourcecontext.Receipt{SourceContextRef: id.String(), ReceiptRef: receiptRef,
			ContentDigest: hex.EncodeToString(spec.ContentDigest[:]), Revision: revision, RecordedAt: recordedAt}, nil
	}
	var existing sourcecontext.Receipt
	var digest []byte
	err = tx.QueryRow(ctx, `
		SELECT source_context_ref::text, receipt_ref, content_digest, revision, recorded_at
		FROM context.proffer_source_context_revision WHERE idempotency_key=$1`, spec.IdempotencyKey).Scan(
		&existing.SourceContextRef, &existing.ReceiptRef, &digest, &existing.Revision, &existing.RecordedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		rollback()
		return sourcecontext.Receipt{}, errors.New("source context idempotency conflict could not be resolved")
	}
	if err != nil {
		rollback()
		return sourcecontext.Receipt{}, err
	}
	rollback()
	existing.ContentDigest = hex.EncodeToString(digest)
	if existing.ContentDigest != hex.EncodeToString(spec.ContentDigest[:]) {
		return sourcecontext.Receipt{}, errors.New("idempotency key is already bound to different source context")
	}
	return existing, nil
}

func (s *SourceContextStore) ValidateSourceContext(
	ctx context.Context, ref, requestID, matterID, courtCaseID, sourceRef string,
) error {
	var valid bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM context.proffer_source_context_revision
			WHERE source_context_ref=$1::uuid AND request_id=$2
			  AND matter_id=$3::uuid AND court_case_id=$4::uuid AND source_ref=$5
		)`, ref, requestID, matterID, courtCaseID, sourceRef).Scan(&valid)
	if err != nil {
		return fmt.Errorf("validate source context: %w", err)
	}
	if !valid {
		return errors.New("source context does not own the requested intake scope")
	}
	return nil
}

// CurrentSourceContext returns the newest revision of the operator context
// recorded for requestID against sourceRef, or found=false when none exists.
// Read-only. Byline: Claude Code · Opus 5.5 · 2026-09-25
func (s *SourceContextStore) CurrentSourceContext(
	ctx context.Context, requestID, sourceRef string,
) (sourcecontext.Revision, bool, error) {
	var current sourcecontext.Revision
	var observed, assertions []byte
	err := s.db.QueryRow(ctx, `
		SELECT source_context_ref::text, revision, observed_source, assertions,
		       change_reason, actor_username, receipt_ref, recorded_at
		FROM context.proffer_source_context_revision
		WHERE request_id=$1 AND source_ref=$2
		ORDER BY revision DESC
		LIMIT 1`, requestID, sourceRef).Scan(&current.SourceContextRef, &current.Revision, &observed, &assertions,
		&current.ChangeReason, &current.ActorUsername, &current.ReceiptRef, &current.RecordedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourcecontext.Revision{}, false, nil
	}
	if err != nil {
		return sourcecontext.Revision{}, false, fmt.Errorf("read current source context: %w", err)
	}
	if err := json.Unmarshal(observed, &current.ObservedSource); err != nil {
		return sourcecontext.Revision{}, false, fmt.Errorf("decode source context observation: %w", err)
	}
	if err := json.Unmarshal(assertions, &current.Assertions); err != nil {
		return sourcecontext.Revision{}, false, fmt.Errorf("decode source context assertions: %w", err)
	}
	return current, true, nil
}

// SourceRegistration returns what register_source recorded for requestID
// (context.source_version.workflow_id is the request id), including the
// retained original's digest and length once retain_original has run.
// Read-only. Byline: Claude Code · Opus 5.5 · 2026-09-25
func (s *SourceContextStore) SourceRegistration(
	ctx context.Context, requestID string,
) (sourcecontext.Registration, bool, error) {
	var registration sourcecontext.Registration
	err := s.db.QueryRow(ctx, `
		SELECT version.id::text, version.declared_format, version.source_context_ref::text,
		       version.original_filename,
		       CASE WHEN retained.id IS NULL THEN NULL ELSE encode(retained.content_sha256, 'hex') END,
		       retained.byte_length
		FROM context.source_version version
		LEFT JOIN context.retained_object retained ON retained.id = version.original_object_id
		WHERE version.workflow_id = $1
		ORDER BY version.version_ordinal DESC
		LIMIT 1`, requestID).Scan(&registration.SourceVersionRef, &registration.DeclaredFormat,
		&registration.SourceContextRef, &registration.OriginalFilename,
		&registration.OriginalSHA256, &registration.OriginalBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourcecontext.Registration{}, false, nil
	}
	if err != nil {
		return sourcecontext.Registration{}, false, fmt.Errorf("read source registration: %w", err)
	}
	return registration, true, nil
}

var _ sourcecontext.Writer = (*SourceContextStore)(nil)
var _ sourcecontext.Validator = (*SourceContextStore)(nil)
var _ sourcecontext.Reader = (*SourceContextStore)(nil)
