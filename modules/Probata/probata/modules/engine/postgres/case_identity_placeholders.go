// Byline: Claude Code · Sonnet · 2026-10-02
//
// Placeholder people for unidentified numbers, and merging a placeholder into a real person.
//
// Owner 2026-10-02 14:32: no party is ever NULL. Every unidentified number gets a placeholder
// person (registry.entity + registry.person, role_in_case 'unknown', verification_state 'proposed',
// requires_human_review true) carrying the number as a confirmed phone identifier, and every
// imported row for that number is linked to it at once. Naming the placeholder is an ordinary
// person edit (EditPerson); merging it into an existing person moves its identifiers and rows and
// marks it merged, never deleted. Every registry change appends registry.identity_change rows in
// the same transaction. A placeholder is not a confirmed person: the Case page does not list it
// and working.validate_message_projection refuses to approve a third-party projection that
// names one.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// placeholderPredicate is the one definition of "still a placeholder".
const placeholderPredicate = `p.role_in_case = 'unknown' AND p.verification_state = 'proposed'`

// linkRowsSQL fills only NULL entity columns, per number, for the placeholders just created.
// $1 = numbers (10 digits), $2 = entity ids, in the same order.
var linkRowsSQL = []struct{ name, sql string }{
	{"call_from", `UPDATE working.call_log c SET from_entity_id = m.eid
		FROM unnest($1::text[], $2::uuid[]) AS m(num, eid)
		WHERE c.from_entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(c.from_e164, ''), c.from_raw, '')) = m.num`},
	{"call_to", `UPDATE working.call_log c SET to_entity_id = m.eid
		FROM unnest($1::text[], $2::uuid[]) AS m(num, eid)
		WHERE c.to_entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(c.to_e164, ''), c.to_raw, '')) = m.num`},
	{"message_participants", `UPDATE working.message_participant x SET entity_id = m.eid
		FROM unnest($1::text[], $2::uuid[]) AS m(num, eid)
		WHERE x.entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(x.participant_e164, ''), x.participant_raw, '')) = m.num`},
	{"third_party_participants", `UPDATE working.third_party_message_participant x SET entity_id = m.eid
		FROM unnest($1::text[], $2::uuid[]) AS m(num, eid)
		WHERE x.entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(x.participant_e164, ''), x.participant_raw, '')) = m.num`},
}

// AddPlaceholders creates one placeholder person per number that no identifier carries yet, then
// links every imported row for those numbers. Re-running is safe: carried numbers are skipped.
func (s *CaseIdentityStore) AddPlaceholders(ctx context.Context, spec caseidentity.PlaceholderSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidatePlaceholders(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("placeholders")
	tx, rollback, err := s.begin(ctx, "placeholders")
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	var already bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registry.identity_change WHERE idempotency_key LIKE $1)`, key+":%").Scan(&already); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	if already && !spec.DryRun {
		rollback()
		return caseidentity.Receipt{Ref: key, Kind: "registry.placeholders", RecordedAt: s.clock(), Replayed: true}, nil
	}

	seen := map[string]bool{}
	var numbers []string
	invalid := 0
	for _, raw := range spec.Numbers {
		number, ok := caseidentity.NormalizePhone(raw)
		if !ok {
			invalid++
			continue
		}
		if !seen[number] {
			seen[number] = true
			numbers = append(numbers, number)
		}
	}
	// A number is carried when any identifier of anybody, merged or not, has it in a non-retired row.
	carriedRows, err := tx.Query(ctx, `SELECT DISTINCT registry.norm_identifier(a.alias_text::text)
		FROM registry.entity_alias a WHERE a.status <> 'retired' AND registry.norm_identifier(a.alias_text::text) = ANY($1::text[])`, numbers)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	carried := map[string]bool{}
	for carriedRows.Next() {
		var number string
		if err := carriedRows.Scan(&number); err != nil {
			carriedRows.Close()
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		carried[number] = true
	}
	carriedRows.Close()
	if err := carriedRows.Err(); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}

	at := s.clock()
	var madeNumbers, madeIDs []string
	created := map[string]string{}
	for _, number := range numbers {
		if carried[number] {
			continue
		}
		personID, err := uuid.NewV7()
		if err != nil {
			rollback()
			return caseidentity.Receipt{}, err
		}
		name := caseidentity.PlaceholderName(number)
		if _, err := tx.Exec(ctx, `INSERT INTO registry.entity
			    (id, entity_type, display_name, canonical_name, data_tier, provenance, requires_human_review, review_status, safe_for_legal_use)
			VALUES ($1, 'person', $2, $2, 'inferred', ARRAY[ROW('postgres', $3::text, 'workbench.case_identity.placeholder')::ai.source_ref],
			        true, 'unreviewed', false)`, personID, name, personID.String()); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO registry.person (id, role_in_case, connection_to, notes, verification_state)
			VALUES ($1, 'unknown', 'unknown', $2, 'proposed')`, personID,
			"Placeholder: this number appears in imported calls or messages and has not been identified."); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		aliasID, err := uuid.NewV7()
		if err != nil {
			rollback()
			return caseidentity.Receipt{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO registry.entity_alias
			    (id, entity_id, alias_text, alias_kind, status, period, basis, recorded_by, created_at, provenance)
			VALUES ($1, $2::uuid, $3, 'phone', 'confirmed', NULL, $4, $5, $6, ARRAY[ROW('postgres', $7::text, 'workbench.case_identity.placeholder')::ai.source_ref])`,
			aliasID, personID.String(), number, "placeholder: number seen in imported calls or messages, not yet identified", actor.Username, at, aliasID.String()); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		var personAfter, aliasAfter []byte
		var owner string
		if err := tx.QueryRow(ctx, personStateSQL, personID.String()).Scan(&personAfter); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if err := tx.QueryRow(ctx, aliasStateSQL, aliasID.String()).Scan(&aliasAfter, &owner); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if _, err := insertChange(ctx, tx, "registry.person", personID.String(), []byte(`{}`), personAfter, spec.ChangeReason, actor, key+":p:"+number, at); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		if _, err := insertChange(ctx, tx, "registry.entity_alias", aliasID.String(), []byte(`{}`), aliasAfter, spec.ChangeReason, actor, key+":a:"+number, at); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		madeNumbers = append(madeNumbers, number)
		madeIDs = append(madeIDs, personID.String())
		if len(created) < 20 {
			created[number] = personID.String()
		}
	}

	linked := map[string]int64{}
	if len(madeNumbers) > 0 {
		for _, step := range linkRowsSQL {
			tag, err := tx.Exec(ctx, step.sql, madeNumbers, madeIDs)
			if err != nil {
				rollback()
				return caseidentity.Receipt{}, caseIdentityWriteError(err)
			}
			linked[step.name] = tag.RowsAffected()
		}
	}
	detail := map[string]any{
		"created": len(madeNumbers), "already_carried": len(numbers) - len(madeNumbers), "not_a_phone_number": invalid,
		"linked": linked, "dry_run": spec.DryRun, "entity_ids": created,
	}
	if spec.DryRun {
		rollback()
		return caseidentity.Receipt{Ref: key, Kind: "registry.placeholders", RecordedAt: at, Detail: detail}, nil
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: key, Kind: "registry.placeholders", RecordedAt: at, Detail: detail}, nil
}

// MergePerson merges a placeholder into an existing person. Its identifiers move (a spelling the
// target already carries stays on the merged placeholder), every row linked to the placeholder is
// re-pointed to the target, and the placeholder is marked merged. Nothing is deleted.
func (s *CaseIdentityStore) MergePerson(ctx context.Context, spec caseidentity.MergeSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateMerge(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("merge")
	tx, rollback, err := s.begin(ctx, "merge")
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	if receipt, replayed, err := replayChange(ctx, tx, key, "registry.person", spec.FromID, ""); err != nil || replayed {
		rollback()
		return receipt, err
	}
	var isPlaceholder bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registry.person p JOIN registry.entity e ON e.id = p.id
		WHERE p.id = $1::uuid AND e.merged_into_id IS NULL AND `+placeholderPredicate+`)`, spec.FromID).Scan(&isPlaceholder); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	if !isPlaceholder {
		rollback()
		return caseidentity.Receipt{}, fmt.Errorf("%w: only a placeholder can be merged into a person", caseidentity.ErrRejected)
	}
	if err := requirePerson(ctx, tx, spec.IntoID); err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	var before []byte
	if err := tx.QueryRow(ctx, personStateSQL, spec.FromID).Scan(&before); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	at := s.clock()

	aliasRows, err := tx.Query(ctx, `SELECT a.id::text FROM registry.entity_alias a WHERE a.entity_id = $1::uuid
		AND NOT EXISTS (SELECT 1 FROM registry.entity_alias b WHERE b.entity_id = $2::uuid AND lower(b.alias_text::text) = lower(a.alias_text::text))
		ORDER BY a.created_at, a.id`, spec.FromID, spec.IntoID)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	var aliasIDs []string
	for aliasRows.Next() {
		var id string
		if err := aliasRows.Scan(&id); err != nil {
			aliasRows.Close()
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		aliasIDs = append(aliasIDs, id)
	}
	aliasRows.Close()
	if err := aliasRows.Err(); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	for _, aliasID := range aliasIDs {
		var aliasBefore, aliasAfter []byte
		var owner string
		if err := tx.QueryRow(ctx, aliasStateSQL, aliasID).Scan(&aliasBefore, &owner); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE registry.entity_alias SET entity_id = $2::uuid WHERE id = $1::uuid`, aliasID, spec.IntoID); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		if err := tx.QueryRow(ctx, aliasStateSQL, aliasID).Scan(&aliasAfter, &owner); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if _, err := insertChange(ctx, tx, "registry.entity_alias", aliasID, aliasBefore, aliasAfter,
			"merged placeholder into person: "+spec.ChangeReason, actor, key+":a:"+aliasID, at); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
	}

	moved := map[string]int64{}
	for name, statement := range map[string]string{
		"call_from":                "UPDATE working.call_log SET from_entity_id = $2::uuid WHERE from_entity_id = $1::uuid",
		"call_to":                  "UPDATE working.call_log SET to_entity_id = $2::uuid WHERE to_entity_id = $1::uuid",
		"message_participants":     "UPDATE working.message_participant SET entity_id = $2::uuid WHERE entity_id = $1::uuid",
		"third_party_participants": "UPDATE working.third_party_message_participant SET entity_id = $2::uuid WHERE entity_id = $1::uuid",
		"third_party_senders":      "UPDATE working.third_party_message SET sender_entity_id = $2::uuid WHERE sender_entity_id = $1::uuid",
	} {
		tag, err := tx.Exec(ctx, statement, spec.FromID, spec.IntoID)
		if err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		moved[name] = tag.RowsAffected()
	}
	if _, err := tx.Exec(ctx, `UPDATE registry.entity SET merged_into_id = $2::uuid, requires_human_review = false WHERE id = $1::uuid`,
		spec.FromID, spec.IntoID); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	after, err := json.Marshal(map[string]any{"merged_into_id": strings.ToLower(spec.IntoID), "identifiers_moved": len(aliasIDs), "rows_moved": moved})
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, err
	}
	ref, err := insertChange(ctx, tx, "registry.person", spec.FromID, before, after, spec.ChangeReason, actor, key, at)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: ref, Kind: "registry.identity_change", RecordedAt: at,
		Detail: map[string]any{"identifiers_moved": len(aliasIDs), "rows_moved": moved}}, nil
}
