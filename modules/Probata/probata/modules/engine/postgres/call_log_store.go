// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// The PostgreSQL boundary of commit_call_log_activity
// (activities/call_log.go). Owner 2026-10-02: calls follow the same path as
// messages, and no working step requires an evidence hash, so a call's
// working.normalized_record cites its source version (artifact_id NULL) and
// working.call_log.source_artifact_id stays NULL until promotion.
//
//   - gate: the latest decision on this run's own preview must be an approval,
//     and the preview's latest snapshot must be this source version and
//     normalized generation;
//   - writes: one working.normalized_record (record_type 'call') and one
//     working.call_log row per normalized call record, ids copied from
//     context.normalized_record_identity, idempotent ON CONFLICT (id), one
//     transaction, with the derived-write guard armed;
//   - receipt: one context.activity_execution / activity_receipt, through the
//     same retry-recovery helpers the other stores use.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// callLogDeriverVersion is deriver_version on every row this store writes.
const callLogDeriverVersion = "proffer.commit_call_log.v1"

// CallLogStore implements activities.CallLogStore.
type CallLogStore struct {
	db  DB
	now func() time.Time
}

// NewCallLogStore requires a database.
func NewCallLogStore(db DB) (*CallLogStore, error) {
	if db == nil {
		return nil, errors.New("call log store requires a database")
	}
	return &CallLogStore{db: db, now: func() time.Time { return time.Now().UTC() }}, nil
}

// LoadParticipantResolution implements activities.CallLogStore with the shared
// reader the message commit uses.
func (s *CallLogStore) LoadParticipantResolution(ctx context.Context, ref proffer.Ref) (disclosure.Resolution, error) {
	return LoadParticipantResolution(ctx, s.db, string(ref))
}

type callLogReceipt struct {
	RefKind string `json:"ref_kind"`
	RefID   string `json:"ref_id"`
	Preview string `json:"preview_handle"`
	Calls   int    `json:"calls"`
}

type normalizedCallPayload struct {
	Content struct {
		Missed          bool   `json:"missed"`
		Direction       string `json:"direction"`
		Disposition     string `json:"disposition"`
		DurationSeconds *int32 `json:"duration_seconds"`
	} `json:"content"`
	Participants []struct {
		Role       string `json:"role"`
		Identifier string `json:"identifier"`
	} `json:"participants"`
}

// CommitCallLog implements activities.CallLogStore.
func (s *CallLogStore) CommitCallLog(ctx context.Context, spec activities.CallLogCommitSpec) (proffer.Ref, proffer.Ref, int, error) {
	sourceVersionID, err := uuid.Parse(string(spec.SourceVersionRef))
	if err != nil {
		return "", "", 0, fmt.Errorf("source version reference %q: %w", spec.SourceVersionRef, err)
	}
	generationID, err := uuid.Parse(string(spec.NormalizedGenerationRef))
	if err != nil {
		return "", "", 0, fmt.Errorf("normalized generation reference %q: %w", spec.NormalizedGenerationRef, err)
	}
	if err := s.checkGate(ctx, spec, sourceVersionID, generationID); err != nil {
		return "", "", 0, err
	}

	stage := string(stagegraph.CommitCallLog)
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", 0, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	executionID, err := parserEnsureExecution(ctx, tx, sourceVersionID, spec.RequestID, stage, stage+":"+generationID.String())
	if err != nil {
		return "", "", 0, err
	}
	priorID, priorRaw, has, err := normalizeLatestReceipt(ctx, tx, executionID)
	if err != nil {
		return "", "", 0, err
	}
	if has {
		var prior callLogReceipt
		_ = json.Unmarshal(priorRaw, &prior)
		return proffer.Ref(priorID.String()), proffer.Ref(priorID.String()), prior.Calls, nil
	}

	rows, err := s.readCalls(ctx, tx, generationID, spec)
	if err != nil {
		return "", "", 0, err
	}
	if len(rows.ids) > 0 {
		if err := declareDeriver(ctx, tx); err != nil {
			return "", "", 0, err
		}
		if _, err := tx.Exec(ctx, insertCallRecordsSQL,
			rows.ids, sourceVersionID, firstPartyRecordSource, callLogDeriverVersion,
			rows.participants, rows.occurred, rows.tiers, rows.attrs, rows.senders, rows.recipients); err != nil {
			return "", "", 0, fmt.Errorf("write call normalized records: %w", err)
		}
		if _, err := tx.Exec(ctx, insertCallLogSQL,
			rows.ids, rows.fromRaw, rows.fromE164, rows.fromEntity, rows.toRaw, rows.toE164, rows.toEntity,
			rows.callTypes, rows.directions, rows.occurred, rows.durations, rows.blocked, rows.raw); err != nil {
			return "", "", 0, fmt.Errorf("write call_log rows: %w", err)
		}
	}

	receiptID, err := uuid.NewV7()
	if err != nil {
		return "", "", 0, err
	}
	now := s.now()
	if len(rows.ids) == 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, not_applicable_reason)
			VALUES ($1::uuid, $2::uuid, $3, 'not_applicable', $4, $4, $5)`,
			receiptID, executionID, spec.Attempt, now, "the normalized generation holds no call records"); err != nil {
			return "", "", 0, fmt.Errorf("write call log receipt: %w", err)
		}
	} else {
		encoded, err := json.Marshal(callLogReceipt{
			RefKind: "call_log_commit", RefID: receiptID.String(), Preview: string(spec.PreviewHandle), Calls: len(rows.ids),
		})
		if err != nil {
			return "", "", 0, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO context.activity_receipt (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref)
			VALUES ($1::uuid, $2::uuid, $3, 'success', $4, $4, $5::jsonb)`,
			receiptID, executionID, spec.Attempt, now, encoded); err != nil {
			return "", "", 0, fmt.Errorf("write call log receipt: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", 0, err
	}
	ref := proffer.Ref(receiptID.String())
	return ref, ref, len(rows.ids), nil
}

// checkGate proves the run's own preview of this exact generation was
// approved, by the owner or by the clean_checks automatic approval.
func (s *CallLogStore) checkGate(ctx context.Context, spec activities.CallLogCommitSpec, sourceVersionID, generationID uuid.UUID) error {
	var requestID string
	var snapSource, snapGeneration uuid.NullUUID
	var approved pgtype.Bool
	err := s.db.QueryRow(ctx, `
		SELECT binding.request_id, snapshot.source_version_id, snapshot.normalized_generation_id, decision.approved
		FROM context.proffer_preview_binding binding
		LEFT JOIN LATERAL (
		    SELECT source_version_id, normalized_generation_id
		    FROM context.proffer_preview_snapshot
		    WHERE preview_handle = binding.preview_handle
		    ORDER BY snapshot_seq DESC LIMIT 1
		) snapshot ON true
		LEFT JOIN LATERAL (
		    SELECT approved FROM context.proffer_preview_decision
		    WHERE preview_handle = binding.preview_handle
		    ORDER BY recorded_at DESC, id DESC LIMIT 1
		) decision ON true
		WHERE binding.preview_handle = $1`, string(spec.PreviewHandle)).
		Scan(&requestID, &snapSource, &snapGeneration, &approved)
	if err != nil {
		return fmt.Errorf("read the preview gate for %s: %w", spec.PreviewHandle, err)
	}
	if requestID != spec.RequestID || !snapSource.Valid || snapSource.UUID != sourceVersionID ||
		!snapGeneration.Valid || snapGeneration.UUID != generationID {
		return fmt.Errorf("preview %s is not this run's preview of this source version and generation", spec.PreviewHandle)
	}
	if !approved.Valid || !approved.Bool {
		return fmt.Errorf("preview %s has no approval; the call log is not committed", spec.PreviewHandle)
	}
	return nil
}

// callRows is the generation's call records in column-array form.
type callRows struct {
	ids                                                  []uuid.UUID
	participants, tiers, attrs, senders, recipients, raw []string
	occurred                                             []pgtype.Timestamptz
	fromRaw, fromE164, fromEntity, toRaw, toE164         []string
	toEntity, callTypes, directions                      []string
	durations                                            []pgtype.Int4
	blocked                                              []bool
}

func (s *CallLogStore) readCalls(ctx context.Context, tx pgx.Tx, generationID uuid.UUID, spec activities.CallLogCommitSpec) (callRows, error) {
	var out callRows
	rows, err := tx.Query(ctx, `
		SELECT id, occurred_at, normalized_payload
		FROM context.normalized_record_identity
		WHERE normalized_generation_id = $1::uuid AND record_type = 'call'
		ORDER BY record_ordinal`, generationID)
	if err != nil {
		return out, fmt.Errorf("read normalized call records: %w", err)
	}
	defer rows.Close()
	resolution := spec.Resolution
	if resolution.PerspectivePersonID == "" {
		resolution.PerspectivePersonID = spec.PerspectivePersonID
	}
	for rows.Next() {
		var id uuid.UUID
		var occurred pgtype.Timestamptz
		var raw []byte
		if err := rows.Scan(&id, &occurred, &raw); err != nil {
			return out, err
		}
		var payload normalizedCallPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return out, fmt.Errorf("decode normalized call record %s: %w", id, err)
		}
		callType, direction, blocked, err := activities.CallType(payload.Content.Direction, payload.Content.Disposition, payload.Content.Missed)
		if err != nil {
			return out, fmt.Errorf("call record %s: %w", id, err)
		}
		var counterparty string
		for _, party := range payload.Participants {
			if identifier := strings.TrimSpace(party.Identifier); identifier != "" &&
				!strings.EqualFold(identifier, disclosure.SelfIdentifier) {
				counterparty = identifier
				break
			}
		}
		var otherE164, otherEntity string
		if resolved, ok := resolution.Lookup(counterparty); ok {
			otherE164, otherEntity = resolved.Normalized, resolved.EntityID
		}
		tier := disclosure.Contemporaneous
		if strings.TrimSpace(resolution.OwnerPersonID) != "" {
			stated := []string{}
			if counterparty != "" {
				stated = append(stated, counterparty)
			}
			if _, t, _, tierErr := resolution.ForMessage(disclosure.SelfIdentifier, stated); tierErr == nil {
				tier = t
			}
		}
		sender, recipient := disclosure.SelfIdentifier, counterparty
		fromRaw, fromE164, fromEntity := disclosure.SelfIdentifier, "", ""
		toRaw, toE164, toEntity := counterparty, otherE164, otherEntity
		if direction == "inbound" {
			sender, recipient = counterparty, disclosure.SelfIdentifier
			fromRaw, fromE164, fromEntity = counterparty, otherE164, otherEntity
			toRaw, toE164, toEntity = disclosure.SelfIdentifier, "", ""
		}
		participants, _ := json.Marshal(payload.Participants)
		recipients, _ := json.Marshal([]string{recipient})
		attrs, _ := json.Marshal(map[string]any{
			"call_type": callType, "disposition": payload.Content.Disposition,
			"perspective_person_id": resolution.PerspectivePersonID,
		})
		var duration pgtype.Int4
		if payload.Content.DurationSeconds != nil {
			duration = pgtype.Int4{Int32: *payload.Content.DurationSeconds, Valid: true}
		}
		out.ids = append(out.ids, id)
		out.occurred = append(out.occurred, occurred)
		out.participants = append(out.participants, string(participants))
		out.tiers = append(out.tiers, tier)
		out.attrs = append(out.attrs, string(attrs))
		out.senders = append(out.senders, sender)
		out.recipients = append(out.recipients, string(recipients))
		out.raw = append(out.raw, string(raw))
		out.fromRaw, out.fromE164, out.fromEntity = append(out.fromRaw, fromRaw), append(out.fromE164, fromE164), append(out.fromEntity, fromEntity)
		out.toRaw, out.toE164, out.toEntity = append(out.toRaw, toRaw), append(out.toE164, toE164), append(out.toEntity, toEntity)
		out.callTypes, out.directions = append(out.callTypes, callType), append(out.directions, direction)
		out.durations, out.blocked = append(out.durations, duration), append(out.blocked, blocked)
	}
	return out, rows.Err()
}

const insertCallRecordsSQL = `
	INSERT INTO working.normalized_record
	    (id, artifact_id, source_version_id, record_type, source, participants, content, occurred_at,
	     disclosure_tier, attrs, derived_from_raw_table, derived_from_raw_id, deriver_version, derived_at,
	     source_record_key, sender, recipients)
	SELECT u.id, NULL, $2::uuid, 'call', $3, u.participants::jsonb, '', u.occurred_at,
	       u.tier, u.attrs::jsonb, 'context.normalized_record_identity', u.id, $4, now(),
	       u.id::text, NULLIF(u.sender, ''), u.recipients::jsonb
	FROM unnest($1::uuid[], $5::text[], $6::timestamptz[], $7::text[], $8::text[], $9::text[], $10::text[])
	     AS u(id, participants, occurred_at, tier, attrs, sender, recipients)
	ON CONFLICT (id) DO NOTHING`

// source_artifact_id is left NULL: no working step requires an evidence hash
// (owner 2026-10-02).
const insertCallLogSQL = `
	INSERT INTO working.call_log
	    (id, source_artifact_id, from_raw, from_e164, from_entity_id, to_raw, to_e164, to_entity_id,
	     call_type, direction, started_at, duration_s, is_blocked, raw_data)
	SELECT u.id, NULL, NULLIF(u.from_raw, ''), NULLIF(u.from_e164, ''), NULLIF(u.from_entity, '')::uuid,
	       NULLIF(u.to_raw, ''), NULLIF(u.to_e164, ''), NULLIF(u.to_entity, '')::uuid,
	       u.call_type, u.direction, u.started_at, u.duration_s, u.is_blocked, u.raw_data::jsonb
	FROM unnest($1::uuid[], $2::text[], $3::text[], $4::text[], $5::text[], $6::text[], $7::text[],
	            $8::text[], $9::text[], $10::timestamptz[], $11::integer[], $12::boolean[], $13::text[])
	     AS u(id, from_raw, from_e164, from_entity, to_raw, to_e164, to_entity,
	          call_type, direction, started_at, duration_s, is_blocked, raw_data)
	ON CONFLICT (id) DO NOTHING`
