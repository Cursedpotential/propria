// Package postgres: this file is the PostgreSQL boundary of
// publish_context_search_activity (activities/publish_context_search.go).
//
// It reads; its only write is the Activity's own receipt in
// context.activity_execution / context.activity_receipt, through the same
// retry-recovery path every normalized-pipeline repository method uses. It
// never writes a canonical row: the Weaviate-first stage runs BEFORE the
// owner's approval and the canonical commit.
//
// Records are paged by record_ordinal (keyset), so no connection is held open
// while a page is embedded and written to Weaviate.
//
// Byline: Claude Code · Opus 5.5 · 2026-10-01
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (participant resolution by ref; routing fields)
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

const contextSearchPageRows = 500

// ContextSearchStore implements activities.ContextSearchSourceStore.
type ContextSearchStore struct {
	db  DB
	now func() time.Time
}

// NewContextSearchStore requires a database handle.
func NewContextSearchStore(db DB) (*ContextSearchStore, error) {
	if db == nil {
		return nil, errors.New("context search store requires a database")
	}
	return &ContextSearchStore{db: db, now: time.Now}, nil
}

// OpenContextSearchRecords resolves the generation's provenance after proving
// the verification receipt is a successful verification of THIS generation of
// THIS source version, then opens the record stream.
func (s *ContextSearchStore) OpenContextSearchRecords(ctx context.Context, spec activities.PublishContextSearchSpec) (activities.ContextSearchPlan, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return activities.ContextSearchPlan{}, fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	generationID, err := uuid.Parse(string(spec.NormalizedGenerationRef))
	if err != nil {
		return activities.ContextSearchPlan{}, fmt.Errorf("normalized generation reference %q: %w", spec.NormalizedGenerationRef, err)
	}
	verificationID, err := uuid.Parse(string(spec.NormalizedVerificationRef))
	if err != nil {
		return activities.ContextSearchPlan{}, fmt.Errorf("normalized verification reference %q: %w", spec.NormalizedVerificationRef, err)
	}

	var verifiedGenerationID uuid.UUID
	var verifyStatus string
	if err := s.db.QueryRow(ctx, `
		SELECT normalized_generation_id, status FROM context.reconciliation_receipt
		WHERE id = $1::uuid AND reconciliation_kind = 'normalized_generation_verification'`, verificationID).Scan(&verifiedGenerationID, &verifyStatus); err != nil {
		return activities.ContextSearchPlan{}, fmt.Errorf("read normalized generation verification receipt: %w", err)
	}
	if verifyStatus != "success" {
		return activities.ContextSearchPlan{}, errors.New("context search requires a successful normalized generation verification")
	}
	if verifiedGenerationID != generationID {
		return activities.ContextSearchPlan{}, errors.New("verification receipt certifies a different normalized generation")
	}

	var (
		generationSourceID                uuid.UUID
		normalizerID, normalizerVersion   string
		parserID, parserVersion, declared string
		formatID                          string
		originalObjectID, matterID        uuid.NullUUID
		originalDigest                    []byte
	)
	if err := s.db.QueryRow(ctx, `
		SELECT generation.source_version_id, generation.normalizer_id, generation.normalizer_version,
		       raw.parser_id, raw.parser_version, version.declared_format, coalesce(raw.format_id, ''),
		       version.original_object_id, version.matter_id, original.content_sha256
		FROM context.normalized_generation generation
		JOIN context.raw_generation raw ON raw.id = generation.raw_generation_id
		JOIN context.source_version version ON version.id = generation.source_version_id
		LEFT JOIN context.retained_object original ON original.id = version.original_object_id
		WHERE generation.id = $1::uuid`, generationID).Scan(
		&generationSourceID, &normalizerID, &normalizerVersion, &parserID, &parserVersion, &declared, &formatID,
		&originalObjectID, &matterID, &originalDigest); err != nil {
		return activities.ContextSearchPlan{}, fmt.Errorf("resolve context search provenance: %w", err)
	}
	if generationSourceID != sourceVersionID {
		return activities.ContextSearchPlan{}, errors.New("normalized generation does not belong to this source version")
	}
	if !originalObjectID.Valid {
		return activities.ContextSearchPlan{}, errors.New("source version has no registered original object")
	}
	if len(originalDigest) != sha256.Size {
		return activities.ContextSearchPlan{}, fmt.Errorf("registered original object has a %d-byte digest, want %d", len(originalDigest), sha256.Size)
	}
	coordinates := contextsearch.Coordinates{
		Schema: "context", Table: "normalized_record_identity",
		SourceVersionID: sourceVersionID, OriginalObjectID: originalObjectID.UUID, NormalizedGenerationID: generationID,
	}
	if matterID.Valid {
		coordinates.MatterID = matterID.UUID
	}
	resolution, err := LoadParticipantResolution(ctx, s.db, string(spec.ParticipantResolutionRef))
	if err != nil {
		return activities.ContextSearchPlan{}, err
	}
	return activities.ContextSearchPlan{
		FormatID:   formatID,
		Resolution: resolution,
		Provenance: contextsearch.Provenance{
			SourceObjectSHA256: originalDigest, SourceFormat: declared,
			ParserID: parserID, ParserVersion: parserVersion,
			NormalizerID: normalizerID, NormalizerVersion: normalizerVersion,
		},
		Coordinates: coordinates,
		Reader:      &contextSearchRows{db: s.db, generationID: generationID, after: -1},
	}, nil
}

type contextSearchRows struct {
	db           DB
	generationID uuid.UUID
	after        int64
	buffer       []activities.ContextSearchRecord
	done         bool
}

const contextSearchPageSQL = `
	SELECT id, record_ordinal, record_type, occurred_at,
	       coalesce(normalized_payload->>'occurred_at_raw', ''),
	       coalesce(normalized_payload->>'source_available_from', ''),
	       coalesce(normalized_payload->>'provenance_class', ''),
	       coalesce(normalized_payload->>'timestamp_certainty', ''),
	       coalesce(normalized_payload->>'timestamp_granularity', ''),
	       coalesce(normalized_payload->'content'->>'body', normalized_payload->'content'->>'text', ''),
	       coalesce(normalized_payload->'content'->>'direction', ''),
	       coalesce(normalized_payload->'content'->>'disposition', ''),
	       coalesce(normalized_payload->'content'->>'duration_seconds', ''),
	       coalesce(normalized_payload->'content'->>'missed', ''),
	       coalesce(normalized_payload->'participants', '[]'::jsonb)::text,
	       sha256(canonical_bytes)
	FROM context.normalized_record_identity
	WHERE normalized_generation_id = $1::uuid AND record_ordinal > $2
	ORDER BY record_ordinal
	LIMIT $3`

func (r *contextSearchRows) Next(ctx context.Context) (activities.ContextSearchRecord, error) {
	if len(r.buffer) == 0 {
		if r.done {
			return activities.ContextSearchRecord{}, io.EOF
		}
		if err := r.fill(ctx); err != nil {
			return activities.ContextSearchRecord{}, err
		}
		if len(r.buffer) == 0 {
			r.done = true
			return activities.ContextSearchRecord{}, io.EOF
		}
	}
	record := r.buffer[0]
	r.buffer = r.buffer[1:]
	return record, nil
}

func (r *contextSearchRows) fill(ctx context.Context) error {
	rows, err := r.db.Query(ctx, contextSearchPageSQL, r.generationID, r.after, contextSearchPageRows)
	if err != nil {
		return fmt.Errorf("page normalized records: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			record                  activities.ContextSearchRecord
			occurredAt              *time.Time
			available, participants string
		)
		if err := rows.Scan(&record.RowID, &record.Ordinal, &record.RecordType, &occurredAt,
			&record.OccurredAtRaw, &available, &record.ProvenanceClass, &record.TimestampCertainty,
			&record.TimestampGranularity, &record.Body, &record.Direction, &record.Disposition,
			&record.DurationSeconds, &record.Missed, &participants, &record.ContentSHA256); err != nil {
			return fmt.Errorf("scan normalized record: %w", err)
		}
		if occurredAt != nil {
			utc := occurredAt.UTC()
			record.OccurredAt = &utc
		}
		knowledge, err := time.Parse(time.RFC3339Nano, available)
		if err != nil {
			return fmt.Errorf("normalized record %s source_available_from %q: %w", record.RowID, available, err)
		}
		record.KnowledgeTime = knowledge.UTC()
		var decoded []struct {
			Role        string `json:"role"`
			Identifier  string `json:"identifier"`
			DisplayName string `json:"display_name"`
		}
		if err := json.Unmarshal([]byte(participants), &decoded); err != nil {
			return fmt.Errorf("normalized record %s participants: %w", record.RowID, err)
		}
		for _, participant := range decoded {
			record.Participants = append(record.Participants, activities.ContextSearchParticipant{
				Role: participant.Role, Identifier: participant.Identifier, DisplayName: participant.DisplayName,
			})
		}
		r.buffer = append(r.buffer, record)
		r.after = record.Ordinal
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("page normalized records: %w", err)
	}
	// fill only runs on an empty buffer, so a short page is the last page.
	if len(r.buffer) < contextSearchPageRows {
		r.done = true
	}
	return nil
}

func (r *contextSearchRows) Close() error { return nil }

// PersistContextSearchPublication writes the one receipt. A repeated
// idempotency coordinate returns the existing durable outcome.
func (s *ContextSearchStore) PersistContextSearchPublication(ctx context.Context, spec activities.PublishContextSearchSpec, outcome activities.ContextSearchPublicationOutcome) (proffer.Ref, proffer.Ref, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	if outcome.Published < 1 || len(outcome.Collections) == 0 {
		return "", "", errors.New("context search receipt requires at least one published object and its collection")
	}
	key := fmt.Sprintf("publish-context-search:%s:%s:%s", spec.RequestID, spec.NormalizedGenerationRef, spec.NormalizedVerificationRef)
	activityName := string(stagegraph.PublishContextSearch)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", fmt.Errorf("begin %s receipt transaction: %w", activityName, err)
	}
	committed := false
	defer func() {
		if !committed {
			cleanupCtx, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanupCtx)
		}
	}()
	executionID, err := parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, activityName, key)
	if err != nil {
		return "", "", err
	}
	priorReceipt, priorResult, found, err := normalizeLatestReceipt(ctx, tx, executionID)
	if err != nil {
		return "", "", err
	}
	if found {
		kind, id, err := decodeNormalizedRef(priorResult)
		if err != nil {
			return "", "", err
		}
		if kind != "context_search_publication" {
			return "", "", fmt.Errorf("existing %s receipt has unexpected ref kind %q", activityName, kind)
		}
		if err := tx.Commit(ctx); err != nil {
			return "", "", err
		}
		committed = true
		return proffer.Ref(id), proffer.Ref(priorReceipt.String()), nil
	}

	publicationID := uuid.New()
	receiptID := uuid.New()
	now := s.now()
	result, err := json.Marshal(map[string]any{
		"ref_kind":               "context_search_publication",
		"ref_id":                 publicationID.String(),
		"collections":            outcome.Collections,
		"published":              outcome.Published,
		"vectors_published":      outcome.VectorsPublished,
		"object_id_construction": outcome.ObjectIDConstruction,
		"dedup_key_construction": outcome.DedupKeyConstruction,
		"first_object_id":        outcome.FirstObjectID,
		"last_object_id":         outcome.LastObjectID,
		"properties_added":       outcome.PropertiesAdded,
		"disclosure_basis":       outcome.DisclosureBasis,
		"disclosure_tiers":       outcome.Tiers,
		"participant_resolution": string(spec.ParticipantResolutionRef),
		"normalized_generation":  string(spec.NormalizedGenerationRef),
		"origin_system":          contextsearch.OriginSystem,
		"ingest_run_id":          spec.RequestID,
	})
	if err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
		VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $5, $6::jsonb)`, receiptID, executionID, spec.Attempt, now, now, result); err != nil {
		return "", "", fmt.Errorf("write %s receipt: %w", activityName, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", fmt.Errorf("commit %s receipt: %w", activityName, err)
	}
	committed = true
	return proffer.Ref(publicationID.String()), proffer.Ref(receiptID.String()), nil
}

var _ activities.ContextSearchSourceStore = (*ContextSearchStore)(nil)
