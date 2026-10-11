// Byline: Codex · GPT-5 · 2026-10-05 (durable single-case write admission)
// Byline: Claude Code · Opus 5.5 · 2026-09-26
//
// Read model behind the Review metadata screen (sourcemeta.Store) and the
// owner's append-only metadata corrections.
//
// Read is read-only over what the Activities already recorded: the source
// registration, the retained original, every context.source_metadata row of
// every class (native JSON verbatim, extractor provenance kept), retained
// members, the attachments the run projected, and custody hash receipts. It
// extracts nothing; extraction belongs to the source-observation Activities.
//
// A correction never touches an observed value. It is a new row in
// context.source_metadata_correction that supersedes exactly the newest
// revision of the same field on the same file (named by its content digest).
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

	"github.com/Cursedpotential/probata/engine/sourcemeta"
)

// SourceMetadataStore reads the metadata screen and persists corrections.
type SourceMetadataStore struct {
	db    DB
	clock func() time.Time
}

// NewSourceMetadataStore requires a database.
func NewSourceMetadataStore(db DB) (*SourceMetadataStore, error) {
	if db == nil {
		return nil, errors.New("source metadata store requires a database")
	}
	return &SourceMetadataStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

// sourceMetadataRunSQL resolves the run's newest source version, its source
// and its retained original. Every join is LEFT: a run that failed before
// register_source still answers with its binding.
const sourceMetadataRunSQL = `
	SELECT binding.request_id, binding.source_ref,
	       version.id::text, source.source_key, source.provenance_class, version.declared_format,
	       version.original_filename, version.acquired_at, version.status,
	       version.matter_id::text, version.court_case_id::text, version.source_context_ref::text,
	       original.id::text, original.storage_class,
	       CASE WHEN original.id IS NULL THEN NULL ELSE encode(original.content_sha256, 'hex') END,
	       original.byte_length, original.immutable_at,
	       snapshot.raw_generation_id::text, snapshot.normalized_generation_id::text, snapshot.snapshot_seq
	FROM context.proffer_preview_binding binding
	LEFT JOIN LATERAL (
	    SELECT candidate.* FROM context.source_version candidate
	    WHERE candidate.workflow_id = binding.workflow_id
	    ORDER BY candidate.version_ordinal DESC LIMIT 1) version ON true
	LEFT JOIN context.source source ON source.id = version.source_id
	LEFT JOIN context.retained_object original ON original.id = version.original_object_id
	LEFT JOIN LATERAL (
	    SELECT latest.raw_generation_id, latest.normalized_generation_id, latest.snapshot_seq
	    FROM context.proffer_preview_snapshot latest
	    WHERE latest.preview_handle = binding.preview_handle
	    ORDER BY latest.snapshot_seq DESC LIMIT 1) snapshot ON true
	WHERE binding.preview_handle = $1`

type sourceMetadataRun struct {
	requestID, sourceRef   string
	source                 *sourcemeta.SourceFacts
	rawGenerationID        *string
	normalizedGenerationID *string
	snapshotSeq            *int64
}

func (s *SourceMetadataStore) resolveRun(ctx context.Context, q rowQuerier, handle string) (sourceMetadataRun, error) {
	var run sourceMetadataRun
	var versionID, sourceKey, provenance, declaredFormat, status *string
	var originalFilename, matterID, courtCaseID, contextRef *string
	var acquiredAt, immutableAt *time.Time
	var objectRef, storageClass, originalSHA *string
	var originalBytes *int64
	err := q.QueryRow(ctx, sourceMetadataRunSQL, handle).Scan(
		&run.requestID, &run.sourceRef,
		&versionID, &sourceKey, &provenance, &declaredFormat,
		&originalFilename, &acquiredAt, &status,
		&matterID, &courtCaseID, &contextRef,
		&objectRef, &storageClass, &originalSHA, &originalBytes, &immutableAt,
		&run.rawGenerationID, &run.normalizedGenerationID, &run.snapshotSeq)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourceMetadataRun{}, sourcemeta.ErrNotFound
	}
	if err != nil {
		return sourceMetadataRun{}, fmt.Errorf("resolve run source: %w", err)
	}
	if versionID == nil {
		return run, nil
	}
	facts := &sourcemeta.SourceFacts{
		SourceVersionRef: *versionID, SourceKey: deref(sourceKey), ProvenanceClass: deref(provenance),
		DeclaredFormat: deref(declaredFormat), OriginalFilename: originalFilename, Status: deref(status),
		MatterID: matterID, CourtCaseID: courtCaseID, SourceContextRef: contextRef,
	}
	if acquiredAt != nil {
		facts.AcquiredAt = *acquiredAt
	}
	if objectRef != nil && originalSHA != nil && originalBytes != nil && immutableAt != nil {
		facts.Original = &sourcemeta.ObjectFacts{
			ObjectRef: *objectRef, StorageClass: deref(storageClass), SHA256: *originalSHA,
			ByteLength: *originalBytes, ImmutableAt: *immutableAt,
		}
	}
	run.source = facts
	return run, nil
}

// Read returns the metadata screen for one file of one run. subjectSHA256 ""
// means the run's source original.
func (s *SourceMetadataStore) Read(ctx context.Context, handle, subjectSHA256 string) (sourcemeta.View, error) {
	if err := sourcemeta.ValidatePreviewHandle(handle); err != nil {
		return sourcemeta.View{}, err
	}
	if err := sourcemeta.ValidateSHA256(subjectSHA256, false); err != nil {
		return sourcemeta.View{}, err
	}
	run, err := s.resolveRun(ctx, s.db, handle)
	if err != nil {
		return sourcemeta.View{}, err
	}
	view := sourcemeta.View{
		PreviewHandle: handle, RequestID: run.requestID, SourceRef: run.sourceRef, Source: run.source,
		Metadata: []sourcemeta.MetadataRow{}, Members: []sourcemeta.Member{}, Hashes: []sourcemeta.HashReceipt{},
		Corrections: []sourcemeta.Correction{},
	}
	if run.source == nil {
		// Nothing was registered: no metadata, members, hashes or corrections exist.
		view.SubjectKind = sourcemeta.SubjectSource
		view.SubjectSHA256 = subjectSHA256
		return view, nil
	}
	versionID := run.source.SourceVersionRef
	if err := s.readMetadata(ctx, versionID, &view); err != nil {
		return sourcemeta.View{}, err
	}
	if err := s.readMembers(ctx, versionID, &view); err != nil {
		return sourcemeta.View{}, err
	}
	if run.snapshotSeq != nil {
		if err := s.db.QueryRow(ctx, `SELECT count(*) FROM context.proffer_preview_attachment
			WHERE preview_handle = $1 AND snapshot_seq = $2`, handle, *run.snapshotSeq).Scan(&view.AttachmentCount); err != nil {
			return sourcemeta.View{}, fmt.Errorf("count projected attachments: %w", err)
		}
	}
	original := ""
	if run.source.Original != nil {
		original = run.source.Original.SHA256
	}
	switch {
	case subjectSHA256 == "" || subjectSHA256 == original:
		view.SubjectKind, view.SubjectSHA256 = sourcemeta.SubjectSource, original
	default:
		kind, attachment, err := s.readSubject(ctx, handle, run, subjectSHA256, view.Members)
		if err != nil {
			return sourcemeta.View{}, err
		}
		view.SubjectKind, view.SubjectSHA256, view.Attachment = kind, subjectSHA256, attachment
	}
	if err := s.readHashes(ctx, versionID, run, &view); err != nil {
		return sourcemeta.View{}, err
	}
	if view.SubjectSHA256 != "" {
		if err := s.readCorrections(ctx, versionID, view.SubjectSHA256, &view); err != nil {
			return sourcemeta.View{}, err
		}
	}
	return view, nil
}

func (s *SourceMetadataStore) readMetadata(ctx context.Context, versionID string, view *sourcemeta.View) error {
	rows, err := s.db.Query(ctx, `
		SELECT metadata.id::text, metadata.metadata_class, metadata.extractor_id, metadata.extractor_version,
		       metadata.generated_at, metadata.extraction_activity_receipt_id::text,
		       CASE WHEN octet_length(metadata.metadata::text) > $2 THEN NULL ELSE metadata.metadata END,
		       octet_length(metadata.metadata::text)
		FROM context.source_metadata metadata
		WHERE metadata.source_version_id = $1::uuid AND metadata.raw_record_id IS NULL
		ORDER BY metadata.metadata_class, metadata.generated_at, metadata.id
		LIMIT $3`, versionID, sourcemeta.MaxRowBytes, sourcemeta.MaxMetadataRows+1)
	if err != nil {
		return fmt.Errorf("read source metadata: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var row sourcemeta.MetadataRow
		var fields []byte
		if err := rows.Scan(&row.MetadataRef, &row.MetadataClass, &row.ExtractorID, &row.ExtractorVersion,
			&row.GeneratedAt, &row.ReceiptRef, &fields, &row.FieldsBytes); err != nil {
			return err
		}
		if fields != nil {
			row.Fields = json.RawMessage(fields)
		} else {
			row.Fields = json.RawMessage("null")
		}
		view.Metadata = append(view.Metadata, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(view.Metadata) > sourcemeta.MaxMetadataRows {
		view.Metadata, view.MetadataTruncated = view.Metadata[:sourcemeta.MaxMetadataRows], true
	}
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM context.source_metadata
		WHERE source_version_id = $1::uuid AND raw_record_id IS NOT NULL`, versionID).Scan(&view.RecordMetadataCount); err != nil {
		return fmt.Errorf("count record metadata: %w", err)
	}
	return nil
}

func (s *SourceMetadataStore) readMembers(ctx context.Context, versionID string, view *sourcemeta.View) error {
	rows, err := s.db.Query(ctx, `
		SELECT member.object_id::text, member.object_role, member.parent_object_id::text,
		       encode(object.content_sha256, 'hex'), object.byte_length, object.storage_class, member.member_locator
		FROM context.source_version_object member
		JOIN context.retained_object object ON object.id = member.object_id
		WHERE member.source_version_id = $1::uuid AND member.object_role <> 'original'
		ORDER BY member.created_at, member.object_id
		LIMIT $2`, versionID, sourcemeta.MaxMembers+1)
	if err != nil {
		return fmt.Errorf("read retained members: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var member sourcemeta.Member
		var locator []byte
		if err := rows.Scan(&member.ObjectRef, &member.Role, &member.ParentObjectRef, &member.SHA256,
			&member.ByteLength, &member.StorageClass, &locator); err != nil {
			return err
		}
		member.MemberLocator = json.RawMessage(locator)
		if len(locator) == 0 {
			member.MemberLocator = json.RawMessage("{}")
		}
		view.Members = append(view.Members, member)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(view.Members) > sourcemeta.MaxMembers {
		view.Members, view.MembersTruncated = view.Members[:sourcemeta.MaxMembers], true
	}
	return nil
}

// readSubject decides whether a digest names a retained member or a projected
// attachment of this run. Anything else is not part of the run.
func (s *SourceMetadataStore) readSubject(ctx context.Context, handle string, run sourceMetadataRun, digest string, members []sourcemeta.Member) (string, *sourcemeta.AttachmentFacts, error) {
	var attachment *sourcemeta.AttachmentFacts
	if run.snapshotSeq != nil {
		rows, err := s.db.Query(ctx, `
			SELECT attachment_id, message_id, filename, media_type, byte_length, source_locator_ref
			FROM context.proffer_preview_attachment
			WHERE preview_handle = $1 AND snapshot_seq = $2 AND sha256 = decode($3, 'hex')
			ORDER BY message_id, attachment_id
			LIMIT 50`, handle, *run.snapshotSeq, digest)
		if err != nil {
			return "", nil, fmt.Errorf("read projected attachment: %w", err)
		}
		for rows.Next() {
			var attachmentID, messageID, locator string
			var filename, mediaType *string
			var byteLength *int64
			if err := rows.Scan(&attachmentID, &messageID, &filename, &mediaType, &byteLength, &locator); err != nil {
				rows.Close()
				return "", nil, err
			}
			if attachment == nil {
				attachment = &sourcemeta.AttachmentFacts{SHA256: digest, SourceLocatorRef: locator, MessageIDs: []string{}}
			}
			if attachment.Filename == nil {
				attachment.Filename = filename
			}
			if attachment.MediaType == nil {
				attachment.MediaType = mediaType
			}
			if attachment.ByteLength == nil {
				attachment.ByteLength = byteLength
			}
			attachment.MessageIDs = append(attachment.MessageIDs, messageID)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return "", nil, err
		}
	}
	if attachment != nil {
		return sourcemeta.SubjectAttachment, attachment, nil
	}
	for _, member := range members {
		if member.SHA256 == digest {
			return sourcemeta.SubjectMember, nil, nil
		}
	}
	var memberExists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM context.source_version_object member
		    JOIN context.retained_object object ON object.id = member.object_id
		    WHERE member.source_version_id = $1::uuid AND object.content_sha256 = decode($2, 'hex'))`,
		run.source.SourceVersionRef, digest).Scan(&memberExists); err != nil {
		return "", nil, fmt.Errorf("resolve retained member: %w", err)
	}
	if memberExists {
		return sourcemeta.SubjectMember, nil, nil
	}
	return "", nil, sourcemeta.ErrSubjectNotInRun
}

func (s *SourceMetadataStore) readHashes(ctx context.Context, versionID string, run sourceMetadataRun, view *sourcemeta.View) error {
	rows, err := s.db.Query(ctx, `
		SELECT receipt.hash_kind, receipt.construction, encode(receipt.digest, 'hex'),
		       receipt.computed_at, receipt.computed_by
		FROM context.hash_receipt receipt
		WHERE (receipt.source_version_id = $1::uuid
		       AND receipt.raw_record_id IS NULL AND receipt.normalized_record_id IS NULL)
		   OR (receipt.hash_kind = 'context_raw_generation_fingerprint' AND receipt.raw_generation_id = $2::uuid)
		   OR (receipt.hash_kind = 'normalized_generation_manifest_digest' AND receipt.normalized_generation_id = $3::uuid)
		ORDER BY receipt.computed_at DESC
		LIMIT 20`, versionID, run.rawGenerationID, run.normalizedGenerationID)
	if err != nil {
		return fmt.Errorf("read hash receipts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var receipt sourcemeta.HashReceipt
		if err := rows.Scan(&receipt.HashKind, &receipt.Construction, &receipt.Digest, &receipt.ComputedAt, &receipt.ComputedBy); err != nil {
			return err
		}
		view.Hashes = append(view.Hashes, receipt)
	}
	return rows.Err()
}

func (s *SourceMetadataStore) readCorrections(ctx context.Context, versionID, digest string, view *sourcemeta.View) error {
	rows, err := s.db.Query(ctx, `
		SELECT correction_ref::text, encode(subject_sha256, 'hex'), field_key, revision, supersedes_ref::text,
		       action, source_value, corrected_value, change_reason, actor_username, receipt_ref, recorded_at
		FROM context.source_metadata_correction
		WHERE source_version_id = $1::uuid AND subject_sha256 = decode($2, 'hex')
		ORDER BY field_key, revision DESC
		LIMIT $3`, versionID, digest, sourcemeta.MaxCorrections)
	if err != nil {
		if overlayError(err, sourcemeta.ErrNotInstalled) == sourcemeta.ErrNotInstalled {
			view.CorrectionsAvailable = false
			return nil
		}
		return fmt.Errorf("read metadata corrections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var correction sourcemeta.Correction
		var sourceValue, correctedValue []byte
		if err := rows.Scan(&correction.CorrectionRef, &correction.SubjectSHA256, &correction.FieldKey,
			&correction.Revision, &correction.SupersedesRef, &correction.Action, &sourceValue, &correctedValue,
			&correction.ChangeReason, &correction.ActorUsername, &correction.ReceiptRef, &correction.RecordedAt); err != nil {
			return err
		}
		correction.SourceValue = rawOrNull(sourceValue)
		correction.CorrectedValue = rawOrNull(correctedValue)
		view.Corrections = append(view.Corrections, correction)
	}
	if err := rows.Err(); err != nil {
		if overlayError(err, sourcemeta.ErrNotInstalled) == sourcemeta.ErrNotInstalled {
			view.Corrections, view.CorrectionsAvailable = []sourcemeta.Correction{}, false
			return nil
		}
		return err
	}
	view.CorrectionsAvailable = true
	return nil
}

// PersistCorrection appends one owner correction revision.
func (s *SourceMetadataStore) PersistCorrection(ctx context.Context, spec sourcemeta.CorrectionSpec) (sourcemeta.Receipt, error) {
	if err := sourcemeta.ValidateCorrection(spec); err != nil {
		return sourcemeta.Receipt{}, err
	}
	subject, err := sourcemeta.DecodeSHA256(spec.SubjectSHA256)
	if err != nil {
		return sourcemeta.Receipt{}, err
	}
	if err := requirePreviewWrite(ctx, s.db, spec.PreviewHandle); err != nil {
		return sourcemeta.Receipt{}, fmt.Errorf("admit source correction: %w: %v", sourcemeta.ErrScopeMissing, err)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sourcemeta.Receipt{}, err
	}
	rollback := func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }
	run, err := s.resolveRun(ctx, tx, spec.PreviewHandle)
	if err != nil {
		rollback()
		return sourcemeta.Receipt{}, err
	}
	if run.source == nil {
		rollback()
		return sourcemeta.Receipt{}, sourcemeta.ErrNotFound
	}
	if run.source.MatterID == nil || run.source.CourtCaseID == nil || !AdmittedCaseIdentity(*run.source.MatterID, *run.source.CourtCaseID) {
		rollback()
		return sourcemeta.Receipt{}, sourcemeta.ErrScopeMissing
	}
	versionID := run.source.SourceVersionRef
	var belongs bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM context.source_version_object member
		    JOIN context.retained_object object ON object.id = member.object_id
		    WHERE member.source_version_id = $1::uuid AND object.content_sha256 = $2)
		OR EXISTS (
		    SELECT 1 FROM context.proffer_preview_attachment attachment
		    WHERE attachment.preview_handle = $3 AND attachment.sha256 = $2)`,
		versionID, subject, spec.PreviewHandle).Scan(&belongs); err != nil {
		rollback()
		return sourcemeta.Receipt{}, fmt.Errorf("resolve corrected file: %w", err)
	}
	if !belongs {
		rollback()
		return sourcemeta.Receipt{}, sourcemeta.ErrSubjectNotInRun
	}
	if err := tx.QueryRow(ctx, `SELECT 1 FROM (SELECT pg_advisory_xact_lock(hashtextextended($1, 0))) AS lock`,
		"metadata-correction:"+versionID+":"+spec.SubjectSHA256+":"+spec.FieldKey).Scan(new(int)); err != nil {
		rollback()
		return sourcemeta.Receipt{}, fmt.Errorf("lock corrected field: %w", err)
	}
	var existing sourcemeta.Receipt
	var existingDigest []byte
	err = tx.QueryRow(ctx, `
		SELECT correction_ref::text, receipt_ref, content_digest, revision, recorded_at
		FROM context.source_metadata_correction WHERE idempotency_key = $1`, spec.IdempotencyKey).Scan(
		&existing.CorrectionRef, &existing.ReceiptRef, &existingDigest, &existing.Revision, &existing.RecordedAt)
	switch {
	case err == nil:
		rollback()
		if hex.EncodeToString(existingDigest) != hex.EncodeToString(spec.ContentDigest[:]) {
			return sourcemeta.Receipt{}, sourcemeta.ErrIdempotencyConflict
		}
		existing.ContentDigest = hex.EncodeToString(existingDigest)
		return existing, nil
	case !errors.Is(err, pgx.ErrNoRows):
		rollback()
		return sourcemeta.Receipt{}, overlayError(err, sourcemeta.ErrNotInstalled)
	}
	var newestRef string
	var newestRevision int
	err = tx.QueryRow(ctx, `
		SELECT correction_ref::text, revision FROM context.source_metadata_correction
		WHERE source_version_id = $1::uuid AND subject_sha256 = $2 AND field_key = $3
		ORDER BY revision DESC LIMIT 1`, versionID, subject, spec.FieldKey).Scan(&newestRef, &newestRevision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		rollback()
		return sourcemeta.Receipt{}, overlayError(err, sourcemeta.ErrNotInstalled)
	}
	if spec.SupersedesRef != newestRef {
		rollback()
		return sourcemeta.Receipt{}, sourcemeta.ErrStaleRevision
	}
	var supersedes any
	if newestRef != "" {
		supersedes = newestRef
	}
	id, err := uuid.NewV7()
	if err != nil {
		rollback()
		return sourcemeta.Receipt{}, err
	}
	at := s.clock()
	receiptRef := "metadata-correction://" + id.String()
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.source_metadata_correction
		    (correction_ref, source_version_id, subject_sha256, field_key, preview_handle, matter_id, court_case_id,
		     revision, supersedes_ref, action, source_value, corrected_value, change_reason,
		     actor_subject_uid, actor_username, idempotency_key, content_digest, receipt_ref, recorded_at)
		VALUES ($1, $2::uuid, $3, $4, $5, $6::uuid, $7::uuid, $8, $9, $10, $11::jsonb, $12::jsonb, $13,
		        $14, $15, $16, $17, $18, $19)`,
		id, versionID, subject, spec.FieldKey, spec.PreviewHandle, *run.source.MatterID, *run.source.CourtCaseID,
		newestRevision+1, supersedes, spec.Action, jsonParam(spec.SourceValue), jsonParam(spec.CorrectedValue),
		spec.ChangeReason, spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey, spec.ContentDigest[:],
		receiptRef, at); err != nil {
		rollback()
		return sourcemeta.Receipt{}, overlayWriteError(err, sourcemeta.ErrNotInstalled, sourcemeta.ErrIdempotencyConflict)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return sourcemeta.Receipt{}, err
	}
	return sourcemeta.Receipt{
		CorrectionRef: id.String(), ReceiptRef: receiptRef, ContentDigest: hex.EncodeToString(spec.ContentDigest[:]),
		Revision: newestRevision + 1, RecordedAt: at,
	}, nil
}

// jsonParam turns an absent or JSON-null value into SQL NULL.
func jsonParam(value json.RawMessage) any {
	trimmed := string(value)
	if len(value) == 0 || trimmed == "null" {
		return nil
	}
	return []byte(value)
}

func rawOrNull(value []byte) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	return json.RawMessage(value)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ sourcemeta.Store = (*SourceMetadataStore)(nil)
