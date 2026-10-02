// Package postgres: the registry side of the one disclosure rule
// (engine/disclosure). It reads the case owner and resolves stated identifiers
// live, normalizing with registry.norm_identifier, the same function the
// registry's aliases are matched with, so Go never carries a second copy of
// that normalization.
//
// Who calls what (parent ruling 2026-10-02): resolve_context_participants_activity
// (the first-party context import lane) calls LoadOwner and ResolveParticipants
// once per run and records the Resolution in its receipt; the Weaviate-first
// publish and the first-party commit both read it back with
// LoadParticipantResolution and never re-resolve.
//
// Byline: Claude Code · Opus 5.5 · 2026-10-02
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/disclosure"
)

// ResolveParticipantsActivityName is the stage whose receipt holds a run's
// participant resolution.
const ResolveParticipantsActivityName = "resolve_context_participants_activity"

// LoadOwner reads the explicit owner person and his confirmed identifiers
// (registry.entity_alias_current, status 'confirmed'). It fails closed when
// the person is missing, merged away, not role_in_case 'user', or has no
// confirmed alias.
func LoadOwner(ctx context.Context, db DB, ownerPersonID, perspectivePersonID string) (disclosure.Owner, error) {
	ownerID, err := uuid.Parse(strings.TrimSpace(ownerPersonID))
	if err != nil {
		return disclosure.Owner{}, fmt.Errorf("owner person id %q is not a uuid; the run must name the case owner explicitly", ownerPersonID)
	}
	perspective := strings.TrimSpace(perspectivePersonID)
	if perspective != "" {
		if _, err := uuid.Parse(perspective); err != nil {
			return disclosure.Owner{}, fmt.Errorf("perspective person id %q is not a uuid", perspectivePersonID)
		}
	}
	var role string
	var merged bool
	err = db.QueryRow(ctx, `
		SELECT person.role_in_case, entity.merged_into_id IS NOT NULL
		FROM registry.person person
		JOIN registry.entity entity ON entity.id = person.id
		WHERE person.id = $1::uuid`, ownerID).Scan(&role, &merged)
	if errors.Is(err, pgx.ErrNoRows) {
		return disclosure.Owner{}, fmt.Errorf("owner person %s is not in registry.person", ownerID)
	}
	if err != nil {
		return disclosure.Owner{}, fmt.Errorf("read owner person %s: %w", ownerID, err)
	}
	if merged {
		return disclosure.Owner{}, fmt.Errorf("owner person %s was merged into another entity", ownerID)
	}
	if role != "user" {
		return disclosure.Owner{}, fmt.Errorf("person %s has role_in_case %q, not 'user'; it is not the case owner", ownerID, role)
	}
	rows, err := db.Query(ctx, `
		SELECT DISTINCT value FROM (
			SELECT nullif(btrim(alias.normalized), '') AS value
			FROM registry.entity_alias_current alias
			WHERE alias.entity_id = $1::uuid AND alias.status = 'confirmed'
			UNION
			SELECT registry.norm_identifier(alias.alias_text::text)
			FROM registry.entity_alias_current alias
			WHERE alias.entity_id = $1::uuid AND alias.status = 'confirmed'
		) identifiers WHERE value IS NOT NULL`, ownerID)
	if err != nil {
		return disclosure.Owner{}, fmt.Errorf("read the case owner's identifiers: %w", err)
	}
	defer rows.Close()
	owner := disclosure.Owner{PersonID: ownerID.String(), PerspectivePersonID: perspective, Identifiers: map[string]bool{}}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return disclosure.Owner{}, fmt.Errorf("read the case owner's identifiers: %w", err)
		}
		owner.Identifiers[value] = true
	}
	if err := rows.Err(); err != nil {
		return disclosure.Owner{}, fmt.Errorf("read the case owner's identifiers: %w", err)
	}
	if err := owner.Validate(); err != nil {
		return disclosure.Owner{}, err
	}
	return owner, nil
}

// NormalizeIdentifiers applies registry.norm_identifier to each identifier, in
// order, in one round trip.
func NormalizeIdentifiers(ctx context.Context, db DB, raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	rows, err := db.Query(ctx, `
		SELECT coalesce(registry.norm_identifier(value), '')
		FROM unnest($1::text[]) WITH ORDINALITY AS input(value, position)
		ORDER BY position`, raw)
	if err != nil {
		return nil, fmt.Errorf("normalize identifiers: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, len(raw))
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("normalize identifiers: %w", err)
		}
		out = append(out, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("normalize identifiers: %w", err)
	}
	if len(out) != len(raw) {
		return nil, fmt.Errorf("normalize identifiers returned %d of %d values", len(out), len(raw))
	}
	return out, nil
}

// ResolveParticipants resolves every distinct stated identifier once against
// registry.entity_alias_current (confirmed aliases only) and returns the run's
// Resolution. An identifier matching aliases of more than one entity fails
// closed rather than picking one.
func ResolveParticipants(ctx context.Context, db DB, owner disclosure.Owner, raws []string) (disclosure.Resolution, error) {
	if err := owner.Validate(); err != nil {
		return disclosure.Resolution{}, err
	}
	distinct := make([]string, 0, len(raws))
	seen := map[string]bool{}
	for _, raw := range raws {
		if strings.TrimSpace(raw) == "" || strings.EqualFold(strings.TrimSpace(raw), disclosure.SelfIdentifier) || seen[raw] {
			continue
		}
		seen[raw] = true
		distinct = append(distinct, raw)
	}
	resolved := make([]disclosure.ResolvedIdentifier, 0, len(distinct))
	if len(distinct) > 0 {
		rows, err := db.Query(ctx, `
			WITH input AS (
				SELECT value AS raw, coalesce(registry.norm_identifier(value), '') AS normalized
				FROM unnest($1::text[]) AS value
			)
			SELECT input.raw, input.normalized,
			       coalesce(array_agg(DISTINCT alias.entity_id::text) FILTER (WHERE alias.entity_id IS NOT NULL), '{}')
			FROM input
			LEFT JOIN registry.entity_alias_current alias
			  ON alias.status = 'confirmed' AND input.normalized <> ''
			 AND (alias.normalized = input.normalized OR registry.norm_identifier(alias.alias_text::text) = input.normalized)
			GROUP BY input.raw, input.normalized`, distinct)
		if err != nil {
			return disclosure.Resolution{}, fmt.Errorf("resolve participants: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var identifier disclosure.ResolvedIdentifier
			var entities []string
			if err := rows.Scan(&identifier.Raw, &identifier.Normalized, &entities); err != nil {
				return disclosure.Resolution{}, fmt.Errorf("resolve participants: %w", err)
			}
			if len(entities) > 1 {
				return disclosure.Resolution{}, fmt.Errorf("identifier %q is a confirmed alias of %d registry entities; resolve the registry before importing", identifier.Raw, len(entities))
			}
			if len(entities) == 1 {
				identifier.EntityID = entities[0]
			}
			identifier.IsOwner = identifier.EntityID != "" && strings.EqualFold(identifier.EntityID, owner.PersonID) ||
				(identifier.Normalized != "" && owner.Identifiers[identifier.Normalized])
			resolved = append(resolved, identifier)
		}
		if err := rows.Err(); err != nil {
			return disclosure.Resolution{}, fmt.Errorf("resolve participants: %w", err)
		}
		if len(resolved) != len(distinct) {
			return disclosure.Resolution{}, fmt.Errorf("resolve participants returned %d of %d identifiers", len(resolved), len(distinct))
		}
	}
	return disclosure.NewResolution(owner, resolved)
}

// LoadParticipantResolution reads the Resolution a successful
// resolve_context_participants_activity recorded. The receipt's result_ref is
// {"ref_kind":"participant_resolution","ref_id":<ref>,"resolution":<Resolution>}.
func LoadParticipantResolution(ctx context.Context, db DB, ref string) (disclosure.Resolution, error) {
	if _, err := uuid.Parse(strings.TrimSpace(ref)); err != nil {
		return disclosure.Resolution{}, fmt.Errorf("participant resolution reference %q is not a uuid", ref)
	}
	var encoded []byte
	err := db.QueryRow(ctx, `
		SELECT receipt.result_ref->'resolution'
		FROM context.activity_receipt receipt
		JOIN context.activity_execution execution ON execution.id = receipt.activity_execution_id
		WHERE execution.activity_name = $1 AND receipt.status = 'success'
		  AND receipt.result_ref->>'ref_kind' = 'participant_resolution'
		  AND receipt.result_ref->>'ref_id' = $2
		ORDER BY receipt.attempt DESC LIMIT 1`, ResolveParticipantsActivityName, strings.TrimSpace(ref)).Scan(&encoded)
	if errors.Is(err, pgx.ErrNoRows) {
		return disclosure.Resolution{}, fmt.Errorf("no successful %s receipt holds participant resolution %s", ResolveParticipantsActivityName, ref)
	}
	if err != nil {
		return disclosure.Resolution{}, fmt.Errorf("read participant resolution %s: %w", ref, err)
	}
	var resolution disclosure.Resolution
	if err := json.Unmarshal(encoded, &resolution); err != nil {
		return disclosure.Resolution{}, fmt.Errorf("decode participant resolution %s: %w", ref, err)
	}
	if err := resolution.Validate(); err != nil {
		return disclosure.Resolution{}, err
	}
	return resolution, nil
}
