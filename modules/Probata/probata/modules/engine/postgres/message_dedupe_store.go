// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// PostgreSQL side of dedupe.MessageDedupeWorkflow (see that package for the
// rule and the end state of a copy). Every mutating statement runs under the
// NOLOGIN role message_dedupe_writer, which platform_runtime may SET but does
// not inherit (scripts/2026-10-02-message-dedupe.sql): the always-on worker
// keeps no DELETE privilege outside these transactions.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/dedupe"
	"github.com/Cursedpotential/probata/engine/firstparty"
)

const dedupeWriterRole = "message_dedupe_writer"

type dedupeQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// MessageDedupeStore implements activities.MessageDedupeStore.
type MessageDedupeStore struct {
	db  DB
	now func() time.Time
}

// NewMessageDedupeStore wires the store to the platform pool.
func NewMessageDedupeStore(db DB) (*MessageDedupeStore, error) {
	if db == nil {
		return nil, errors.New("message dedupe store: database is required")
	}
	return &MessageDedupeStore{db: db, now: func() time.Time { return time.Now().UTC() }}, nil
}

var _ activities.MessageDedupeStore = (*MessageDedupeStore)(nil)

// planCopiesSQL freezes the copy set: per match key the earliest primary
// occurrence is kept; every other primary from a different source version is a
// copy when it is the same device (never cross-device), the same perspective
// and the same projection kind, and the kept row still has its working row.
const planCopiesSQL = `
	INSERT INTO working.message_dedupe_copy
	    (dedupe_id, record_id, keeper_record_id, projection_kind, match_key, source_version_id, keeper_source_version_id)
	WITH primaries AS (
	    SELECT o.normalized_record_id AS id, o.match_key, o.source_version_id, o.projection_kind,
	           o.perspective_person_id, o.cross_device,
	           first_value(o.normalized_record_id) OVER w AS keeper,
	           first_value(o.source_version_id) OVER w AS keeper_source,
	           first_value(o.perspective_person_id) OVER w AS keeper_perspective,
	           first_value(o.projection_kind) OVER w AS keeper_kind
	    FROM working.message_occurrence o
	    WHERE o.normalized_record_id = o.primary_record_id
	    WINDOW w AS (PARTITION BY o.match_key ORDER BY o.recorded_at, o.normalized_record_id))
	SELECT $1, p.id, p.keeper, p.projection_kind, p.match_key, p.source_version_id, p.keeper_source
	FROM primaries p
	WHERE p.id <> p.keeper AND p.source_version_id <> p.keeper_source
	  AND NOT p.cross_device
	  AND p.perspective_person_id IS NOT DISTINCT FROM p.keeper_perspective
	  AND p.projection_kind = p.keeper_kind
	  AND (EXISTS (SELECT 1 FROM working.message k WHERE k.id = p.keeper)
	       OR EXISTS (SELECT 1 FROM working.third_party_message k WHERE k.normalized_record_id = p.keeper))`

// heldBackSQL counts what the copy rule saw but did not plan, by reason.
const heldBackSQL = `
	WITH primaries AS (
	    SELECT o.normalized_record_id AS id, o.source_version_id, o.projection_kind, o.perspective_person_id, o.cross_device,
	           first_value(o.normalized_record_id) OVER w AS keeper,
	           first_value(o.source_version_id) OVER w AS keeper_source,
	           first_value(o.perspective_person_id) OVER w AS keeper_perspective,
	           first_value(o.projection_kind) OVER w AS keeper_kind
	    FROM working.message_occurrence o
	    WHERE o.normalized_record_id = o.primary_record_id
	    WINDOW w AS (PARTITION BY o.match_key ORDER BY o.recorded_at, o.normalized_record_id)),
	candidates AS (SELECT * FROM primaries WHERE id <> keeper)
	SELECT count(*) FILTER (WHERE source_version_id = keeper_source),
	       count(*) FILTER (WHERE source_version_id <> keeper_source AND cross_device),
	       count(*) FILTER (WHERE source_version_id <> keeper_source AND NOT cross_device
	                          AND (perspective_person_id IS DISTINCT FROM keeper_perspective OR projection_kind <> keeper_kind)),
	       count(*) FILTER (WHERE source_version_id <> keeper_source AND NOT cross_device
	                          AND perspective_person_id IS NOT DISTINCT FROM keeper_perspective AND projection_kind = keeper_kind
	                          AND NOT EXISTS (SELECT 1 FROM working.message k WHERE k.id = keeper)
	                          AND NOT EXISTS (SELECT 1 FROM working.third_party_message k WHERE k.normalized_record_id = keeper))
	FROM candidates`

// dependents are the rows that would hold a copy which the removal does not
// know how to move; any of them refuses the plan instead of being lost.
var dependents = []struct{ name, sql string }{
	{"third_party_thread_membership", `SELECT count(*) FROM working.third_party_context_thread_message m JOIN working.third_party_message t ON t.id = m.message_id JOIN working.message_dedupe_copy c ON c.record_id = t.normalized_record_id WHERE c.dedupe_id = $1`},
	{"first_party_relative_time_anchor", `SELECT count(*) FROM context.first_party_thread_message_relative_time_anchor l JOIN working.message_dedupe_copy c ON c.record_id = l.message_id WHERE c.dedupe_id = $1`},
	{"third_party_relative_time_anchor", `SELECT count(*) FROM context.third_party_thread_message_relative_time_anchor l JOIN working.message_dedupe_copy c ON c.record_id = l.message_id WHERE c.dedupe_id = $1`},
	{"attachment", `SELECT count(*) FROM working.attachment a JOIN working.message_dedupe_copy c ON c.record_id = a.message_id WHERE c.dedupe_id = $1`},
	{"content_chunk_message", `SELECT count(*) FROM working.content_chunk_message x JOIN working.message_dedupe_copy c ON c.record_id = x.message_id WHERE c.dedupe_id = $1`},
	{"record_visible_from", `SELECT count(*) FROM working.record_visible_from x JOIN working.message_dedupe_copy c ON c.record_id = x.record_id WHERE c.dedupe_id = $1`},
	{"event_source_record", `SELECT count(*) FROM working.event_source_record x JOIN working.message_dedupe_copy c ON c.record_id = x.record_id WHERE c.dedupe_id = $1`},
	{"realization_event", `SELECT count(*) FROM working.realization_event x JOIN working.message_dedupe_copy c ON c.record_id = x.trigger_record_id WHERE c.dedupe_id = $1`},
	{"realization_event_record", `SELECT count(*) FROM working.realization_event_record x JOIN working.message_dedupe_copy c ON c.record_id = x.normalized_record_id WHERE c.dedupe_id = $1`},
	{"walk_step", `SELECT count(*) FROM working.walk_step x JOIN working.message_dedupe_copy c ON c.record_id = x.record_id WHERE c.dedupe_id = $1`},
	{"walk_step_retrieval", `SELECT count(*) FROM working.walk_step_retrieval x JOIN working.message_dedupe_copy c ON c.record_id = x.record_id WHERE c.dedupe_id = $1`},
	{"timeline_event", `SELECT count(*) FROM analysis.timeline_event x JOIN working.message_dedupe_copy c ON c.record_id = x.primary_record_id WHERE c.dedupe_id = $1`},
	{"knowledge_evidence_promotion", `SELECT count(*) FROM analysis.knowledge_evidence_promotion x JOIN working.message_dedupe_copy c ON c.record_id = x.normalized_record_id WHERE c.dedupe_id = $1`},
	{"evidence_item", `SELECT count(*) FROM evidence.evidence_item x JOIN working.message_dedupe_copy c ON c.record_id = x.normalized_record_id WHERE c.dedupe_id = $1`},
	{"message_neighbour_link", `SELECT count(*) FROM working.message m JOIN working.message_dedupe_copy c ON c.record_id IN (m.prev_message_id, m.next_message_id) WHERE c.dedupe_id = $1`},
}

// PlanDedupe freezes (or re-reads) the plan and records its receipt.
func (s *MessageDedupeStore) PlanDedupe(ctx context.Context, request dedupe.StepRequest, attempt int32) (dedupe.Receipt, error) {
	started := s.now()
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dedupe.Receipt{}, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '10s'`); err != nil {
		return dedupe.Receipt{}, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO working.message_dedupe_run (dedupe_id, rule, workflow_id, run_id, planned_at)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (dedupe_id) DO NOTHING`,
		request.DedupeID, dedupe.Rule, request.WorkflowID, request.RunID, started)
	if err != nil {
		return dedupe.Receipt{}, fmt.Errorf("record the dedupe run: %w", err)
	}
	reused := tag.RowsAffected() == 0
	counts := map[string]int64{}
	if !reused {
		if _, err := tx.Exec(ctx, planCopiesSQL, request.DedupeID); err != nil {
			return dedupe.Receipt{}, fmt.Errorf("plan the copies: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO working.message_dedupe_thread_version (dedupe_id, family, thread_version_id)
			SELECT DISTINCT $1, 'first_party', m.thread_version_id
			FROM working.first_party_context_thread_message m
			JOIN working.message_dedupe_copy c ON c.record_id = m.message_id
			WHERE c.dedupe_id = $1`, request.DedupeID); err != nil {
			return dedupe.Receipt{}, fmt.Errorf("plan the thread versions: %w", err)
		}
		var sameSource, crossDevice, otherPerspective, keeperGone int64
		if err := tx.QueryRow(ctx, heldBackSQL).Scan(&sameSource, &crossDevice, &otherPerspective, &keeperGone); err != nil {
			return dedupe.Receipt{}, fmt.Errorf("count what the rule held back: %w", err)
		}
		counts["held_back_same_source_version"] = sameSource
		counts["held_back_cross_device"] = crossDevice
		counts["held_back_other_perspective_or_kind"] = otherPerspective
		counts["held_back_keeper_without_working_row"] = keeperGone
		// The dependent tables are readable by the writer role only (read grants,
		// scripts/2026-10-02-message-dedupe.sql); platform_runtime does not see them.
		blocked := []string{}
		if err := asWriter(ctx, tx, func() error {
			for _, dependent := range dependents {
				var n int64
				if err := tx.QueryRow(ctx, dependent.sql, request.DedupeID).Scan(&n); err != nil {
					return fmt.Errorf("count %s rows on the copies: %w", dependent.name, err)
				}
				counts["dependent_"+dependent.name] = n
				if n > 0 {
					blocked = append(blocked, fmt.Sprintf("%s %d", dependent.name, n))
				}
			}
			return nil
		}); err != nil {
			return dedupe.Receipt{}, err
		}
		if len(blocked) > 0 {
			return dedupe.Receipt{}, dedupe.Refusal{Reason: "copies have rows the removal does not move (" + strings.Join(blocked, ", ") + "); nothing was planned"}
		}
	}
	if err := planCounts(ctx, tx, request.DedupeID, counts); err != nil {
		return dedupe.Receipt{}, err
	}
	if request.ExpectedCopies > 0 && counts["copies"] != request.ExpectedCopies {
		return dedupe.Receipt{}, dedupe.Refusal{Reason: fmt.Sprintf("the plan holds %d copies, expected %d; nothing was planned", counts["copies"], request.ExpectedCopies)}
	}
	if !reused {
		if _, err := tx.Exec(ctx, `UPDATE working.message_dedupe_run SET copies = $2 WHERE dedupe_id = $1`, request.DedupeID, counts["copies"]); err != nil {
			return dedupe.Receipt{}, err
		}
	}
	receipt, err := s.writeReceipt(ctx, tx, request, attempt, counts, started)
	if err != nil {
		return dedupe.Receipt{}, err
	}
	receipt.Reused = reused
	return receipt, tx.Commit(ctx)
}

// planCounts adds the frozen plan's totals to counts.
func planCounts(ctx context.Context, q dedupeQueryer, dedupeID string, counts map[string]int64) error {
	var copies, firstParty, thirdParty, versions int64
	if err := q.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE projection_kind = 'first_party'),
		       count(*) FILTER (WHERE projection_kind = 'acquired_third_party'),
		       (SELECT count(*) FROM working.message_dedupe_thread_version WHERE dedupe_id = $1)
		FROM working.message_dedupe_copy WHERE dedupe_id = $1`, dedupeID).Scan(&copies, &firstParty, &thirdParty, &versions); err != nil {
		return fmt.Errorf("read the plan: %w", err)
	}
	counts["copies"], counts["first_party"], counts["third_party"], counts["thread_versions"] = copies, firstParty, thirdParty, versions
	rows, err := q.Query(ctx, `
		SELECT coalesce(substring(src.source_key FROM '/sms-backup-restore/([0-9]+)/'), 'other:' || c.projection_kind), count(*)
		FROM working.message_dedupe_copy c
		JOIN context.source_version sv ON sv.id = c.source_version_id
		JOIN context.source src ON src.id = sv.source_id
		WHERE c.dedupe_id = $1 GROUP BY 1`, dedupeID)
	if err != nil {
		return fmt.Errorf("count the plan by device: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var device string
		var n int64
		if err := rows.Scan(&device, &n); err != nil {
			return err
		}
		counts["device:"+device] = n
	}
	return rows.Err()
}

// RunDedupeStep runs one step. A dry run replays every step up to and
// including this one in one transaction and rolls it back; a live run checks
// the earlier steps' live receipts, then works through the plan in batches.
func (s *MessageDedupeStore) RunDedupeStep(ctx context.Context, request dedupe.StepRequest, attempt int32, progress func(done int64)) (dedupe.Receipt, error) {
	index := stepIndex(request.Step)
	if index < 0 {
		return dedupe.Receipt{}, dedupe.Refusal{Reason: fmt.Sprintf("unknown step %q", request.Step)}
	}
	if progress == nil {
		progress = func(int64) {}
	}
	started := s.now()
	ids, versions, err := s.loadPlan(ctx, request.DedupeID)
	if err != nil {
		return dedupe.Receipt{}, err
	}
	var counts map[string]int64
	if request.DryRun {
		counts, err = s.dryRun(ctx, request.DedupeID, index, ids, versions)
	} else {
		if receipt, done, err := s.liveReceipt(ctx, request.DedupeID, request.Step); err != nil {
			return dedupe.Receipt{}, err
		} else if done {
			receipt.Reused = true
			return receipt, nil
		}
		for _, earlier := range dedupe.Steps[:index] {
			if _, done, err := s.liveReceipt(ctx, request.DedupeID, earlier); err != nil {
				return dedupe.Receipt{}, err
			} else if !done {
				return dedupe.Receipt{}, dedupe.Refusal{Reason: fmt.Sprintf("step %s has no live receipt; %s runs after it", earlier, request.Step)}
			}
		}
		counts, err = s.live(ctx, request, ids, versions, progress)
	}
	if err != nil {
		return dedupe.Receipt{}, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dedupe.Receipt{}, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	receipt, err := s.writeReceipt(ctx, tx, request, attempt, counts, started)
	if err != nil {
		return dedupe.Receipt{}, err
	}
	return receipt, tx.Commit(ctx)
}

func stepIndex(step dedupe.Step) int {
	for index, candidate := range dedupe.Steps {
		if candidate == step {
			return index
		}
	}
	return -1
}

func (s *MessageDedupeStore) loadPlan(ctx context.Context, dedupeID string) ([]uuid.UUID, []uuid.UUID, error) {
	var planned bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM working.message_dedupe_run WHERE dedupe_id = $1)`, dedupeID).Scan(&planned); err != nil {
		return nil, nil, err
	}
	if !planned {
		return nil, nil, dedupe.Refusal{Reason: "no plan is frozen under " + dedupeID}
	}
	ids, err := collectUUIDs(ctx, s.db, `SELECT record_id FROM working.message_dedupe_copy WHERE dedupe_id = $1 ORDER BY record_id`, dedupeID)
	if err != nil {
		return nil, nil, fmt.Errorf("read the planned copies: %w", err)
	}
	versions, err := collectUUIDs(ctx, s.db, `
		SELECT thread_version_id FROM working.message_dedupe_thread_version
		WHERE dedupe_id = $1 AND family = 'first_party' ORDER BY thread_version_id`, dedupeID)
	if err != nil {
		return nil, nil, fmt.Errorf("read the planned thread versions: %w", err)
	}
	return ids, versions, nil
}

func collectUUIDs(ctx context.Context, q dedupeQueryer, sql string, args ...any) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *MessageDedupeStore) liveReceipt(ctx context.Context, dedupeID string, step dedupe.Step) (dedupe.Receipt, bool, error) {
	var id uuid.UUID
	var encoded []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, counts FROM working.message_dedupe_receipt
		WHERE dedupe_id = $1 AND step = $2 AND NOT dry_run AND status = 'success'`, dedupeID, string(step)).Scan(&id, &encoded)
	if errors.Is(err, pgx.ErrNoRows) {
		return dedupe.Receipt{}, false, nil
	}
	if err != nil {
		return dedupe.Receipt{}, false, err
	}
	receipt := dedupe.Receipt{ReceiptID: id.String(), Step: step, Counts: map[string]int64{}}
	if err := json.Unmarshal(encoded, &receipt.Counts); err != nil {
		return dedupe.Receipt{}, false, err
	}
	return receipt, true, nil
}

func (s *MessageDedupeStore) dryRun(ctx context.Context, dedupeID string, index int, ids, versions []uuid.UUID) (map[string]int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '10s'`); err != nil {
		return nil, err
	}
	var counts map[string]int64
	for _, step := range dedupe.Steps[:index+1] {
		counts = map[string]int64{}
		if err := applyStep(ctx, tx, dedupeID, step, ids, versions, counts); err != nil {
			return nil, err
		}
	}
	// The deferred projection and thread checks run now, as a commit would run them.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
		return nil, dedupe.Refusal{Reason: "the dry run fails the deferred checks: " + err.Error()}
	}
	return counts, nil // the deferred Rollback discards every change
}

func (s *MessageDedupeStore) live(ctx context.Context, request dedupe.StepRequest, ids, versions []uuid.UUID, progress func(int64)) (map[string]int64, error) {
	total := map[string]int64{}
	var units []uuid.UUID
	switch request.Step {
	case dedupe.StepRecomputeThreadVersions:
		units = versions
	case dedupe.StepVerify:
		units = nil
	default:
		units = ids
	}
	if request.Step == dedupe.StepVerify {
		return total, s.inTx(ctx, func(tx pgx.Tx) error {
			return applyStep(ctx, tx, request.DedupeID, request.Step, ids, versions, total)
		})
	}
	batch := request.BatchSize
	if request.Step == dedupe.StepRecomputeThreadVersions {
		batch = 1 // one thread version per transaction
	}
	if batch <= 0 {
		batch = dedupe.DefaultBatchSize
	}
	var done int64
	for start := 0; start < len(units); start += batch {
		end := start + batch
		if end > len(units) {
			end = len(units)
		}
		part := units[start:end]
		counts := map[string]int64{}
		err := s.inTx(ctx, func(tx pgx.Tx) error {
			if request.Step == dedupe.StepRecomputeThreadVersions {
				return applyStep(ctx, tx, request.DedupeID, request.Step, ids, part, counts)
			}
			return applyStep(ctx, tx, request.DedupeID, request.Step, part, versions, counts)
		})
		if err != nil {
			return nil, fmt.Errorf("%s, batch at %d of %d: %w", request.Step, start, len(units), err)
		}
		for name, n := range counts {
			total[name] += n
		}
		done += int64(len(part))
		progress(done)
	}
	return total, nil
}

func (s *MessageDedupeStore) inTx(ctx context.Context, body func(pgx.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '10s'`); err != nil {
		return err
	}
	if err := body(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); err != nil {
		return fmt.Errorf("deferred checks: %w", err)
	}
	return tx.Commit(ctx)
}

// asWriter runs statements under message_dedupe_writer and returns to the
// session role, so the deferred checks at commit run as platform_runtime.
func asWriter(ctx context.Context, tx pgx.Tx, body func() error) error {
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+dedupeWriterRole); err != nil {
		return fmt.Errorf("assume %s: %w", dedupeWriterRole, err)
	}
	if err := body(); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SET LOCAL ROLE NONE`)
	return err
}

func execCount(ctx context.Context, tx pgx.Tx, counts map[string]int64, name, sql string, args ...any) error {
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	counts[name] += tag.RowsAffected()
	return nil
}

// applyStep is one step over the given copies (or, for the recompute, the
// given thread versions). Every statement is idempotent: a retried batch
// finds nothing left to change.
func applyStep(ctx context.Context, tx pgx.Tx, dedupeID string, step dedupe.Step, ids, versions []uuid.UUID, counts map[string]int64) error {
	switch step {
	case dedupe.StepRepointOccurrences:
		return asWriter(ctx, tx, func() error {
			if err := execCount(ctx, tx, counts, "copy_occurrences_repointed", `
				UPDATE working.message_occurrence o SET primary_record_id = c.keeper_record_id
				FROM working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[])
				  AND o.normalized_record_id = c.record_id AND o.primary_record_id <> c.keeper_record_id`, dedupeID, ids); err != nil {
				return err
			}
			return execCount(ctx, tx, counts, "further_occurrences_repointed", `
				UPDATE working.message_occurrence o SET primary_record_id = c.keeper_record_id
				FROM working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[])
				  AND o.primary_record_id = c.record_id AND o.normalized_record_id <> c.record_id`, dedupeID, ids)
		})
	case dedupe.StepRemoveThreadMemberships:
		return asWriter(ctx, tx, func() error {
			return execCount(ctx, tx, counts, "first_party_thread_memberships_removed", `
				DELETE FROM working.first_party_context_thread_message m
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND m.message_id = c.record_id`, dedupeID, ids)
		})
	case dedupe.StepRecomputeThreadVersions:
		for _, version := range versions {
			changed, err := recomputeThreadVersion(ctx, tx, version)
			if err != nil {
				return err
			}
			counts["thread_versions_recomputed"]++
			if changed {
				counts["thread_versions_changed"]++
			}
		}
		return nil
	case dedupe.StepRemoveFirstPartyMessages:
		return asWriter(ctx, tx, func() error {
			if err := execCount(ctx, tx, counts, "first_party_participants_removed", `
				DELETE FROM working.message_participant p
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'first_party'
				  AND p.message_id = c.record_id
				  AND EXISTS (SELECT 1 FROM working.message k WHERE k.id = c.keeper_record_id)`, dedupeID, ids); err != nil {
				return err
			}
			if err := execCount(ctx, tx, counts, "first_party_messages_removed", `
				DELETE FROM working.message m
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'first_party'
				  AND m.id = c.record_id
				  AND EXISTS (SELECT 1 FROM working.message k WHERE k.id = c.keeper_record_id)`, dedupeID, ids); err != nil {
				return err
			}
			return execCount(ctx, tx, counts, "first_party_routes_removed", `
				DELETE FROM working.message_projection_route r
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'first_party'
				  AND r.normalized_record_id = c.record_id AND r.projection_kind = 'first_party'
				  AND NOT EXISTS (SELECT 1 FROM working.message m WHERE m.derived_from_record_id = c.record_id)`, dedupeID, ids)
		})
	case dedupe.StepRemoveThirdPartyMessages:
		return asWriter(ctx, tx, func() error {
			if err := execCount(ctx, tx, counts, "third_party_participants_removed", `
				DELETE FROM working.third_party_message_participant p
				USING working.third_party_message t, working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'acquired_third_party'
				  AND t.normalized_record_id = c.record_id AND p.message_id = t.id
				  AND EXISTS (SELECT 1 FROM working.third_party_message k WHERE k.normalized_record_id = c.keeper_record_id)`, dedupeID, ids); err != nil {
				return err
			}
			if err := execCount(ctx, tx, counts, "third_party_messages_removed", `
				DELETE FROM working.third_party_message t
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'acquired_third_party'
				  AND t.normalized_record_id = c.record_id
				  AND EXISTS (SELECT 1 FROM working.third_party_message k WHERE k.normalized_record_id = c.keeper_record_id)`, dedupeID, ids); err != nil {
				return err
			}
			return execCount(ctx, tx, counts, "third_party_routes_removed", `
				DELETE FROM working.message_projection_route r
				USING working.message_dedupe_copy c
				WHERE c.dedupe_id = $1 AND c.record_id = ANY($2::uuid[]) AND c.projection_kind = 'acquired_third_party'
				  AND r.normalized_record_id = c.record_id AND r.projection_kind = 'acquired_third_party'
				  AND NOT EXISTS (SELECT 1 FROM working.third_party_message t WHERE t.normalized_record_id = c.record_id)`, dedupeID, ids)
		})
	case dedupe.StepVerify:
		return verifyPlan(ctx, tx, dedupeID, versions, counts)
	}
	return dedupe.Refusal{Reason: fmt.Sprintf("unknown step %q", step)}
}

// recomputeThreadVersion recomputes one first-party thread version from the
// rows it still holds, exactly as extendThread does when it adds members
// (bounds, horizon, membership digest over the ordered member ids), and proves
// it with the deferred validator's own function. Ordinals keep their values:
// a removed copy leaves a gap, as extendThread's next-ordinal rule allows.
func recomputeThreadVersion(ctx context.Context, tx pgx.Tx, version uuid.UUID) (bool, error) {
	var before []byte
	if err := tx.QueryRow(ctx, `
		SELECT assertion_digest FROM working.first_party_context_thread_version WHERE id = $1 FOR UPDATE`, version).Scan(&before); err != nil {
		return false, fmt.Errorf("lock thread version %s: %w", version, err)
	}
	rows, err := tx.Query(ctx, `
		SELECT message_id::text FROM working.first_party_context_thread_message
		WHERE thread_version_id = $1 ORDER BY thread_ordinal`, version)
	if err != nil {
		return false, err
	}
	ordered := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return false, err
		}
		ordered = append(ordered, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(ordered) == 0 {
		return false, dedupe.Refusal{Reason: fmt.Sprintf("thread version %s would hold no message", version)}
	}
	digest := firstparty.MembershipDigest(ordered)
	if _, err := tx.Exec(ctx, `
		UPDATE working.first_party_context_thread_version version
		SET first_occurred_at = membership.first_at,
		    last_occurred_at = membership.last_at,
		    knowledge_available_from = GREATEST(membership.required_at, sources.required_at),
		    assertion_digest = $2::bytea
		FROM (SELECT min(occurred_at) AS first_at, max(occurred_at) AS last_at,
		             max(source_available_from) FILTER (WHERE required_for_horizon) AS required_at
		      FROM working.first_party_context_thread_message WHERE thread_version_id = $1::uuid) membership,
		     (SELECT max(source_available_from) FILTER (WHERE required_for_horizon) AS required_at
		      FROM working.first_party_context_thread_source WHERE thread_version_id = $1::uuid) sources
		WHERE version.id = $1::uuid`, version, digest); err != nil {
		return false, fmt.Errorf("recompute thread version %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, `SELECT working.validate_first_party_context_thread_version($1::uuid)`, version); err != nil {
		return false, dedupe.Refusal{Reason: fmt.Sprintf("thread version %s fails its validation after the recompute: %v", version, err)}
	}
	return string(before) != string(digest), nil
}

// verifyPlan asserts the end state over the whole plan; any failure refuses.
func verifyPlan(ctx context.Context, tx pgx.Tx, dedupeID string, versions []uuid.UUID, counts map[string]int64) error {
	checks := []struct{ name, sql string }{
		{"copies_with_first_party_message", `SELECT count(*) FROM working.message m JOIN working.message_dedupe_copy c ON c.record_id = m.id WHERE c.dedupe_id = $1`},
		{"copies_with_third_party_message", `SELECT count(*) FROM working.third_party_message t JOIN working.message_dedupe_copy c ON c.record_id = t.normalized_record_id WHERE c.dedupe_id = $1`},
		{"copies_with_route", `SELECT count(*) FROM working.message_projection_route r JOIN working.message_dedupe_copy c ON c.record_id = r.normalized_record_id WHERE c.dedupe_id = $1`},
		{"copies_with_thread_membership", `SELECT count(*) FROM working.first_party_context_thread_message m JOIN working.message_dedupe_copy c ON c.record_id = m.message_id WHERE c.dedupe_id = $1`},
		{"copies_not_pointing_at_keeper", `SELECT count(*) FROM working.message_occurrence o JOIN working.message_dedupe_copy c ON c.record_id = o.normalized_record_id WHERE c.dedupe_id = $1 AND o.primary_record_id <> c.keeper_record_id`},
		{"occurrences_naming_a_copy_as_primary", `SELECT count(*) FROM working.message_occurrence o JOIN working.message_dedupe_copy c ON c.record_id = o.primary_record_id WHERE c.dedupe_id = $1`},
		{"keepers_without_working_row", `SELECT count(DISTINCT c.keeper_record_id) FROM working.message_dedupe_copy c WHERE c.dedupe_id = $1 AND NOT EXISTS (SELECT 1 FROM working.message k WHERE k.id = c.keeper_record_id) AND NOT EXISTS (SELECT 1 FROM working.third_party_message k WHERE k.normalized_record_id = c.keeper_record_id)`},
		{"copies_cross_device", `SELECT count(*) FROM working.message_occurrence o JOIN working.message_dedupe_copy c ON c.record_id = o.normalized_record_id WHERE c.dedupe_id = $1 AND o.cross_device`},
		{"copies_without_normalized_record", `SELECT count(*) FROM working.message_dedupe_copy c WHERE c.dedupe_id = $1 AND NOT EXISTS (SELECT 1 FROM working.normalized_record r WHERE r.id = c.record_id)`},
	}
	failed := []string{}
	for _, check := range checks {
		var n int64
		if err := tx.QueryRow(ctx, check.sql, dedupeID).Scan(&n); err != nil {
			return fmt.Errorf("verify %s: %w", check.name, err)
		}
		counts[check.name] = n
		if n != 0 {
			failed = append(failed, fmt.Sprintf("%s %d", check.name, n))
		}
	}
	for _, version := range versions {
		if _, err := tx.Exec(ctx, `SELECT working.validate_first_party_context_thread_version($1::uuid)`, version); err != nil {
			failed = append(failed, fmt.Sprintf("thread version %s: %v", version, err))
			break
		}
		counts["thread_versions_valid"]++
	}
	for name, sql := range map[string]string{
		"total_working_message":     `SELECT count(*) FROM working.message`,
		"total_third_party_message": `SELECT count(*) FROM working.third_party_message`,
		"total_occurrences":         `SELECT count(*) FROM working.message_occurrence`,
	} {
		var n int64
		if err := tx.QueryRow(ctx, sql).Scan(&n); err != nil {
			return err
		}
		counts[name] = n
	}
	if len(failed) > 0 {
		sort.Strings(failed)
		return dedupe.Refusal{Reason: "verification failed: " + strings.Join(failed, "; ")}
	}
	return nil
}

func (s *MessageDedupeStore) writeReceipt(ctx context.Context, tx pgx.Tx, request dedupe.StepRequest, attempt int32,
	counts map[string]int64, started time.Time) (dedupe.Receipt, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return dedupe.Receipt{}, err
	}
	encoded, err := json.Marshal(counts)
	if err != nil {
		return dedupe.Receipt{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO working.message_dedupe_receipt
		    (id, dedupe_id, step, dry_run, status, counts, workflow_id, run_id, attempt, started_at, completed_at)
		VALUES ($1, $2, $3, $4, 'success', $5::jsonb, $6, $7, $8, $9, $10)`,
		id, request.DedupeID, string(request.Step), request.DryRun, encoded, request.WorkflowID, request.RunID,
		attempt, started, s.now()); err != nil {
		return dedupe.Receipt{}, fmt.Errorf("write the %s receipt: %w", request.Step, err)
	}
	return dedupe.Receipt{ReceiptID: id.String(), Step: request.Step, DryRun: request.DryRun, Counts: counts}, nil
}
