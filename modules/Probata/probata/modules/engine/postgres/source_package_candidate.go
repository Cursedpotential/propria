package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/sourcemeta"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// packageCandidateEdit applies only supported derived assertion changes while preserving the entire original source pin.
// Inputs: bounded typed candidate and accumulated owner overrides. Output: validated replacement candidate.
// Effects: none. Pick for native candidates; normalized-message corrections retain service.Correct's distinct contract.
func packageCandidateEdit(candidate service.AICandidate, fields map[string]any) (service.AICandidate, error) {
	raw, _ := json.Marshal(candidate)
	value := map[string]any{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return candidate, err
	}
	for key, v := range fields {
		switch key {
		case "name", "entity_type", "event_type", "predicate", "statement", "confidence", "occurred_at":
			value[key] = v
		default:
			return candidate, fmt.Errorf("unsupported canonical candidate field %q; source pins, locators and quotes are immutable", key)
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return candidate, err
	}
	if err := json.Unmarshal(raw, &candidate); err != nil {
		return candidate, err
	}
	validation := candidate
	// Existing staging stores strategy/history accounts in candidate_fact while preserving reported_kind.
	if validation.Kind == "fact" && (validation.ReportedKind == "strategy" || validation.ReportedKind == "history") {
		validation.Kind = validation.ReportedKind
	}
	return candidate, service.ValidateAICandidate(validation)
}

// applyPackageCandidateTx binds package edits and canonical decisions to the same append-only correction transaction.
// Inputs: locked package revision, existing working candidate ID/hash and owner action. Output: genuine canonical IDs/hash.
// Effects: edits create source-preserving replacement working rows; commit/reject reuse the existing analysis decision writer.
// Pick for native source-version candidates only; Accept records inspection without projection eligibility.
func (s *SourceMetadataStore) applyPackageCandidateTx(ctx context.Context, tx pgx.Tx, spec PackageReviewSpec, prior map[string]any, digest string) (map[string]any, error) {
	parts := strings.SplitN(spec.ItemID, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("canonical item requires its actual table and candidate UUID")
	}
	table := "working." + parts[0]
	switch table {
	case "working.candidate_entity", "working.candidate_event", "working.candidate_fact":
	default:
		return nil, errors.New("unsupported working candidate table")
	}
	id := parts[1]
	if canonical, ok := prior["canonical"].(map[string]any); ok {
		if current, ok := canonical["candidate_id"].(string); ok {
			id = current
		}
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.New("actual canonical candidate UUID required")
	}
	var raw, stored []byte
	var state, reviewKey string
	err := tx.QueryRow(ctx, `SELECT attrs->'candidate',content_sha256,review_state,coalesce(attrs->>'package_review_item_id','') FROM `+table+`
 WHERE id=$1::uuid AND source_raw_table='context.source_version' AND source_raw_id=$2 FOR UPDATE`, id, spec.SourceVersionRef).Scan(&raw, &stored, &state, &reviewKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, sourcemeta.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(spec.CandidateContentSHA256) != 64 || hex.EncodeToString(stored) != spec.CandidateContentSHA256 || (reviewKey != "" && reviewKey != spec.ItemID) {
		return nil, sourcemeta.ErrStaleRevision
	}
	var candidate service.AICandidate
	if err := json.Unmarshal(raw, &candidate); err != nil {
		return nil, err
	}
	recomputed := service.AICandidateDigest(candidate)
	if candidate.SourceVersionID != spec.SourceVersionRef || hex.EncodeToString(recomputed[:]) != spec.CandidateContentSHA256 {
		return nil, sourcemeta.ErrStaleRevision
	}
	store := &EntityExtractionStore{db: s.db}
	if candidate.SourceRef == "" {
		return nil, errors.New("retained AI candidate requires its existing preview review route")
	}
	if err := store.verifyAIContextEvidence(ctx, candidate.AISourcePin, []service.AICandidate{candidate}); err != nil {
		return nil, err
	}
	result := map[string]any{"candidate_id": id, "candidate_table": table, "content_sha256": spec.CandidateContentSHA256}
	switch spec.Operation {
	case "edit":
		if state != "pending" && state != "needs_info" {
			return nil, sourcemeta.ErrStaleRevision
		}
		edited, err := packageCandidateEdit(candidate, spec.Fields)
		if err != nil {
			return nil, service.ErrInvalid{Err: err}
		}
		if err := store.verifyAIContextEvidence(ctx, edited.AISourcePin, []service.AICandidate{edited}); err != nil {
			return nil, err
		}
		hash := service.AICandidateDigest(edited)
		newHash := hex.EncodeToString(hash[:])
		if newHash == spec.CandidateContentSHA256 {
			return result, nil
		}
		newID := flow.DeterministicID("ai_candidate", spec.SourceVersionRef, newHash)
		candidateRaw, _ := json.Marshal(edited)
		patch := map[string]any{"id": newID, "review_state": "pending", "confidence": edited.Confidence,
			"promoted_to_table": nil, "promoted_to_id": nil, "promoted_at": nil}
		switch table {
		case "working.candidate_entity":
			patch["name"], patch["normalized_name"], patch["entity_type"] = edited.Name, entities.RegistryNormalizedName(edited.Name), entities.CandidateTypeFor(entities.RegistryType(edited.EntityType))
		case "working.candidate_event":
			patch["summary"], patch["event_type"], patch["occurred_at"] = edited.Statement, edited.EventType, edited.OccurredAt
		case "working.candidate_fact":
			patch["statement"], patch["predicate"] = edited.Statement, edited.Predicate
		}
		patchRaw, _ := json.Marshal(patch)
		// Clone every actual taxonomy/provenance column, then replace only typed assertion fields.
		tag, err := tx.Exec(ctx, `INSERT INTO `+table+` SELECT (jsonb_populate_record(NULL::`+table+`,to_jsonb(c)||$3::jsonb||
 jsonb_build_object('content_sha256',to_jsonb($4::bytea),'created_at',now(),'attrs',(c.attrs-'review_decision')||
 jsonb_build_object('candidate',$5::jsonb,'package_review_item_id',$6::text,'supersedes_candidate_id',$1::text,'graph_promotion','held','projection_scope','none')))).*
 FROM `+table+` c WHERE c.id=$1::uuid AND c.source_raw_id=$2 ON CONFLICT DO NOTHING`, id, spec.SourceVersionRef, patchRaw, hash[:], candidateRaw, spec.ItemID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, sourcemeta.ErrStaleRevision
		}
		if _, err := tx.Exec(ctx, `UPDATE `+table+` SET review_state='superseded' WHERE id=$1::uuid`, id); err != nil {
			return nil, err
		}
		result["candidate_id"], result["content_sha256"], result["supersedes_candidate_id"] = newID, newHash, id
	case "accept":
		if state != "pending" && state != "needs_info" {
			return nil, sourcemeta.ErrStaleRevision
		}
	case "commit", "reject":
		if spec.Operation == "commit" {
			canonical, _ := prior["canonical"].(map[string]any)
			if canonical["content_sha256"] != spec.CandidateContentSHA256 || canonical["candidate_id"] != id {
				return nil, sourcemeta.ErrStaleRevision
			}
		}
		decision := "rejected"
		if spec.Operation == "commit" {
			decision = "approved"
		}
		decisionID, err := store.decideAICandidateTx(ctx, tx, candidate.AISourcePin, id, spec.CandidateContentSHA256, decision,
			entities.Actor{SubjectUID: spec.ActorSubjectUID, Username: spec.ActorUsername}, digest, s.clock(), nil)
		if err != nil {
			return nil, err
		}
		result["decision_id"] = decisionID
	default:
		return nil, errors.New("canonical candidates require edit, accept, reject or commit")
	}
	return result, nil
}
