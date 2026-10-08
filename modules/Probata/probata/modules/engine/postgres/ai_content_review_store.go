package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ service.AIReviewStore = (*EntityExtractionStore)(nil)

// ResolveAIPreview finds the sole existing preview binding for a native AI request and source pin.
// Inputs: admission request ID and retained source version/object/hash. Output: opaque preview handle.
// Effects: read-only. Choose in the Activity when workflow history has no browser preview handle.
func (s *EntityExtractionStore) ResolveAIPreview(ctx context.Context, requestID string, pin service.AISourcePin) (string, error) {
	if strings.TrimSpace(requestID) == "" || len(requestID) > 512 {
		return "", service.ErrInvalid{Err: errors.New("bounded request_id is required")}
	}
	rows, err := s.db.Query(ctx, `SELECT binding.preview_handle
FROM context.proffer_preview_binding binding
JOIN context.source_version version ON version.workflow_id=binding.workflow_id
WHERE binding.request_id=$1 AND version.id=$2::uuid
  AND version.original_object_id=$3::uuid
  AND version.matter_id=$4::uuid AND version.court_case_id=$5::uuid
ORDER BY binding.preview_handle LIMIT 2`, requestID, pin.SourceVersionID, pin.SourceObjectID,
		authoritativeMatterID, authoritativeCourtCaseID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var handle string
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", service.ErrNotFound
	}
	if err := rows.Scan(&handle); err != nil {
		return "", err
	}
	if rows.Next() {
		return "", service.ErrInvalid{Err: errors.New("AI source request has multiple preview bindings")}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if _, err := s.VerifyAISource(ctx, handle, pin); err != nil {
		return "", err
	}
	return handle, nil
}

// VerifyAISource admits a retained original only when its custody pin and initial operating receipt agree.
// Inputs: preview handle and exact source version/object/SHA pin. Outputs: durable matter mode.
// Effects: read-only. Choose for AI review before any candidate staging or owner decision.
func (s *EntityExtractionStore) VerifyAISource(ctx context.Context, previewHandle string, pin service.AISourcePin) (string, error) {
	// The sealed local opener has no independent provider-version readback. Reject
	// caller-supplied versions until an opener can verify them against the original.
	if pin.VersionID != nil {
		return "", service.ErrInvalid{Err: errors.New("version_id cannot be verified for the retained local original")}
	}
	var detail, objectID, sha string
	err := s.db.QueryRow(ctx, `SELECT event.detail, version.original_object_id::text, encode(original.content_sha256, 'hex')
FROM context.proffer_preview_binding binding
JOIN context.proffer_preview_event event ON event.preview_handle=binding.preview_handle AND event.event_id=0
JOIN context.source_version version ON version.workflow_id=binding.workflow_id
JOIN context.retained_object original ON original.id=version.original_object_id
WHERE binding.preview_handle=$1 AND version.id=$2::uuid AND version.status='retained'
  AND version.matter_id=$3::uuid AND version.court_case_id=$4::uuid`,
		previewHandle, pin.SourceVersionID, authoritativeMatterID, authoritativeCourtCaseID).Scan(&detail, &objectID, &sha)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", service.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if objectID != pin.SourceObjectID || sha != pin.SourceSHA256 {
		return "", service.ErrInvalid{Err: errors.New("retained original custody pin disagrees with source version")}
	}
	admission := previewmodel.Binding{OperatingMode: recordedOperatingMode(detail)}
	applyBindingAdmission(&admission, detail)
	return admission.OperatingMode, nil
}

// StageAICandidates records bounded proposals against context.source_version without normalizing conversation text.
// Inputs: preview, deterministic run ID and candidates with a common retained source pin.
// Outputs: stable candidate UUIDs. Effects: one extraction receipt and pending working rows.
// Choose for AI source bundles; no registry or timeline promotion occurs here.
func (s *EntityExtractionStore) StageAICandidates(ctx context.Context, previewHandle, runID, requestDigest string, candidates []service.AICandidate) ([]string, error) {
	if len(candidates) == 0 || len(candidates) > 500 {
		return nil, service.ErrInvalid{Err: errors.New("AI candidate batch must contain 1-500 rows")}
	}
	pin := candidates[0].AISourcePin
	for _, candidate := range candidates {
		if err := service.ValidateAICandidate(candidate); err != nil {
			return nil, service.ErrInvalid{Err: err}
		}
		if candidate.AISourcePin.SourceVersionID != pin.SourceVersionID || candidate.SourceObjectID != pin.SourceObjectID || candidate.SourceSHA256 != pin.SourceSHA256 || !sameVersionID(candidate.VersionID, pin.VersionID) {
			return nil, service.ErrInvalid{Err: errors.New("AI batch changes its retained source pin")}
		}
	}
	mode, err := s.VerifyAISource(ctx, previewHandle, pin)
	if err != nil {
		return nil, err
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(mode)); err != nil {
		return nil, err
	}
	if err := aiOpenerAvailable(s.aiOriginalOpener); err != nil {
		return nil, err
	}
	if len(requestDigest) != 64 {
		return nil, service.ErrInvalid{Err: errors.New("request digest is required")}
	}
	if err := s.verifyAIEvidence(ctx, pin, candidates); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previousDigest string
	err = tx.QueryRow(ctx, `SELECT coalesce(stats->>'request_digest','') FROM working.extraction_run WHERE id=$1::uuid FOR UPDATE`, runID).Scan(&previousDigest)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil && !sameAIRunDigest(previousDigest, requestDigest) {
		return nil, entities.ErrConflict
	}
	stats, _ := json.Marshal(map[string]any{"source_version_id": pin.SourceVersionID, "preview_handle": previewHandle, "source_raw_table": "context.source_version", "request_digest": requestDigest})
	_, err = tx.Exec(ctx, `INSERT INTO working.extraction_run
 (id, extractor, extractor_version, source_summary, status, finished_at, stats)
 VALUES ($1::uuid, 'probata.ai_content.candidates', '1', $2, 'completed', now(), $3::jsonb)
 ON CONFLICT (id) DO NOTHING`, runID, "preview:"+previewHandle+" source:"+pin.SourceVersionID, string(stats))
	if err != nil {
		return nil, err
	}
	var persistedDigest, persistedSource string
	if err := tx.QueryRow(ctx, `SELECT coalesce(stats->>'request_digest',''), coalesce(stats->>'source_version_id','')
FROM working.extraction_run WHERE id=$1::uuid FOR UPDATE`, runID).Scan(&persistedDigest, &persistedSource); err != nil {
		return nil, err
	}
	if !sameAIRunDigest(persistedDigest, requestDigest) || persistedSource != pin.SourceVersionID {
		return nil, entities.ErrConflict
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		originalKind := candidate.Kind
		candidate = aiCandidateForStaging(candidate)
		attrs := map[string]any{"candidate": candidate, "reported_kind": originalKind,
			"source_raw_table": "context.source_version", "source_raw_id": pin.SourceVersionID,
			"review_domain": candidate.ReviewDomain, "graph_promotion": "held", "projection_scope": "none"}
		raw, err := json.Marshal(attrs)
		if err != nil {
			return nil, err
		}
		digest := service.AICandidateDigest(candidate)
		id := flow.DeterministicID("ai_candidate", pin.SourceVersionID, hex.EncodeToString(digest[:]))
		ids = append(ids, id)
		switch candidate.Kind {
		case "entity":
			name := strings.TrimSpace(candidate.Name)
			normalized := entities.RegistryNormalizedName(name)
			if normalized == "" {
				normalized = "unnamed"
			}
			var tag pgconn.CommandTag
			tag, err = tx.Exec(ctx, `INSERT INTO working.candidate_entity
 (id, extraction_run_id, source_raw_table, source_raw_id, entity_type, name, normalized_name, confidence, attrs, content_sha256, review_state, domain, ontology_version)
 VALUES ($1::uuid,$2::uuid,'context.source_version',$3,$4,$5,$6,$7,$8::jsonb,$9,'pending','context','probata.ai_content/v1')
 ON CONFLICT DO NOTHING`, id, runID, pin.SourceVersionID, entities.CandidateTypeFor(entities.RegistryType(candidate.EntityType)), name, normalized, candidate.Confidence, string(raw), digest[:])
			if err == nil && tag.RowsAffected() == 0 {
				err = verifyExistingAICandidate(ctx, tx, "working.candidate_entity", id, runID, pin.SourceVersionID, digest[:])
			}
		case "event":
			var tag pgconn.CommandTag
			tag, err = tx.Exec(ctx, `INSERT INTO working.candidate_event
 (id, extraction_run_id, source_raw_table, source_raw_id, event_type, summary, occurred_at, temporal_confidence, confidence, attrs, content_sha256, review_state, domain, ontology_version)
 VALUES ($1::uuid,$2::uuid,'context.source_version',$3,$4,$5,$6,0,$7,$8::jsonb,$9,'pending','context','probata.ai_content/v1')
 ON CONFLICT DO NOTHING`, id, runID, pin.SourceVersionID, candidate.EventType, candidate.Statement, candidate.OccurredAt, candidate.Confidence, string(raw), digest[:])
			if err == nil && tag.RowsAffected() == 0 {
				err = verifyExistingAICandidate(ctx, tx, "working.candidate_event", id, runID, pin.SourceVersionID, digest[:])
			}
		case "fact":
			var tag pgconn.CommandTag
			tag, err = tx.Exec(ctx, `INSERT INTO working.candidate_fact
 (id, extraction_run_id, source_raw_table, source_raw_id, predicate, statement, evidence_quote, confidence, attrs, content_sha256, review_state, domain, ontology_version)
 VALUES ($1::uuid,$2::uuid,'context.source_version',$3,$4,$5,$6,$7,$8::jsonb,$9,'pending','context','probata.ai_content/v1')
 ON CONFLICT DO NOTHING`, id, runID, pin.SourceVersionID, candidate.Predicate, candidate.Statement, candidate.EvidenceQuote, candidate.Confidence, string(raw), digest[:])
			if err == nil && tag.RowsAffected() == 0 {
				err = verifyExistingAICandidate(ctx, tx, "working.candidate_fact", id, runID, pin.SourceVersionID, digest[:])
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return ids, tx.Commit(ctx)
}

func sameVersionID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// sameAIRunDigest permits exact retry and rejects a reused run ID with a changed request.
// Inputs: stored and requested digest. Outputs: replay match. Effects: none.
// Choose at the extraction_run idempotency boundary.
func sameAIRunDigest(stored, requested string) bool {
	return len(requested) == 64 && stored == requested
}

// aiCandidateForStaging preserves an undated event account as a fact without assigning an event time.
// Inputs: validated candidate. Outputs: table-ready candidate. Effects: none.
// Choose before hashing and inserting a retained-source proposal.
func aiCandidateForStaging(candidate service.AICandidate) service.AICandidate {
	if candidate.Kind == "event" && candidate.OccurredAt == nil {
		candidate.Kind, candidate.Predicate = "fact", "undated_event_account"
	}
	if candidate.Kind == "strategy" || candidate.Kind == "history" {
		candidate.Predicate = candidate.Kind + "_account"
		candidate.Kind = "fact"
	}
	return candidate
}

// aiOpenerAvailable prevents stage writes until the approved sealed-object reader is injected.
// Inputs: opener. Outputs: availability error. Effects: none.
// Choose at the retained-source staging boundary.
func aiOpenerAvailable(opener func(context.Context, string) (io.ReadCloser, error)) error {
	if opener == nil {
		return errors.New("retained-source AI staging needs an original-object opener")
	}
	return nil
}

// verifyExistingAICandidate makes ON CONFLICT a checked replay, never silent loss of another proposal.
// Inputs: fixed table name, candidate ID, source ID and content digest. Outputs: conflict or replay success.
// Effects: read-only inside caller transaction. Choose after an insert reports zero rows.
func verifyExistingAICandidate(ctx context.Context, tx pgx.Tx, table, id, runID, sourceVersionID string, digest []byte) error {
	var sourceTable, sourceID, storedRunID string
	var existing []byte
	err := tx.QueryRow(ctx, `SELECT source_raw_table, source_raw_id, content_sha256, extraction_run_id::text FROM `+table+` WHERE id=$1::uuid`, id).Scan(&sourceTable, &sourceID, &existing, &storedRunID)
	if err != nil {
		return entities.ErrConflict
	}
	if !sameAIExistingCandidate(sourceTable, sourceID, existing, sourceVersionID, digest) || storedRunID != runID {
		return entities.ErrConflict
	}
	return nil
}

// sameAIExistingCandidate verifies a conflicting insert is the exact same candidate and source.
// Inputs: stored source/digest and expected source/digest. Outputs: replay match.
// Effects: none. Choose only after ON CONFLICT reports no inserted row.
func sameAIExistingCandidate(table, source string, digest []byte, expectedSource string, expectedDigest []byte) bool {
	return table == "context.source_version" && source == expectedSource && hex.EncodeToString(digest) == hex.EncodeToString(expectedDigest)
}

// verifyAIEvidence reopens the sealed original and checks each native-pointer codepoint slice against its quote and hash.
// Inputs: custody pin and bounded candidates. Outputs: validation error or exact source proof.
// Effects: reads the retained original; no writes. Choose immediately before staging, even on retries.
func (s *EntityExtractionStore) verifyAIEvidence(ctx context.Context, pin service.AISourcePin, candidates []service.AICandidate) error {
	var storageClass, uri string
	var length int64
	err := s.db.QueryRow(ctx, `SELECT original.storage_class, original.object_uri, original.byte_length
FROM context.source_version version JOIN context.retained_object original ON original.id=version.original_object_id
WHERE version.id=$1::uuid AND version.original_object_id=$2::uuid`, pin.SourceVersionID, pin.SourceObjectID).Scan(&storageClass, &uri, &length)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := aiStorageAvailable(storageClass); err != nil {
		return err
	}
	const maxOriginalBytes = 32 << 20
	if length < 0 || length > maxOriginalBytes {
		return service.ErrInvalid{Err: errors.New("retained original exceeds bounded AI review verifier")}
	}
	reader, err := s.aiOriginalOpener(ctx, uri)
	if err != nil {
		return err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(reader, maxOriginalBytes+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) != length || len(raw) > maxOriginalBytes {
		return service.ErrInvalid{Err: errors.New("retained original length changed")}
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != pin.SourceSHA256 {
		return service.ErrInvalid{Err: errors.New("retained original SHA256 changed")}
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return service.ErrInvalid{Err: fmt.Errorf("retained original is not JSON: %w", err)}
	}
	return verifyAISpans(document, candidates)
}

// aiStorageAvailable permits only originals that the sealed local opener can read by immutable URI.
// Inputs: retained object storage class. Outputs: availability error. Effects: none.
// Choose before opening; remote objects require a separately verified version-aware adapter.
func aiStorageAvailable(storageClass string) error {
	if storageClass != "inline" && storageClass != "filesystem" {
		return service.ErrInvalid{Err: errors.New("remote retained original needs a version-aware opener before AI staging")}
	}
	return nil
}

// verifyAISpans compares every candidate quote with its exact codepoint slice from the native JSON field.
// Inputs: parsed original and candidates. Outputs: validation error or success.
// Effects: none. Choose after whole-object length and SHA validation.
func verifyAISpans(document any, candidates []service.AICandidate) error {
	for _, candidate := range candidates {
		value, err := aiStringAtPointer(document, candidate.NativeJSONPointer)
		if err != nil {
			return service.ErrInvalid{Err: err}
		}
		points := []rune(value)
		if candidate.SourceSpan.End > len(points) {
			return service.ErrInvalid{Err: errors.New("AI source span exceeds native string")}
		}
		slice := string(points[candidate.SourceSpan.Start:candidate.SourceSpan.End])
		sum := sha256.Sum256([]byte(slice))
		if hex.EncodeToString(sum[:]) != candidate.SourceSpan.SHA256 || slice != candidate.EvidenceQuote {
			return service.ErrInvalid{Err: errors.New("AI evidence quote or span hash disagrees with retained original")}
		}
	}
	return nil
}

// aiStringAtPointer resolves an RFC6901 pointer to a native JSON string for grounded excerpt checks.
// Inputs: parsed original JSON and pointer. Outputs: string or validation error.
// Effects: none. Choose for source spans; arrays use exact zero-based indexes.
func aiStringAtPointer(document any, pointer string) (string, error) {
	if !strings.HasPrefix(pointer, "/") {
		return "", errors.New("native JSON pointer must identify a string field")
	}
	var value any = document
	for _, encoded := range strings.Split(pointer[1:], "/") {
		for i := 0; i < len(encoded); i++ {
			if encoded[i] == '~' && (i+1 == len(encoded) || encoded[i+1] != '0' && encoded[i+1] != '1') {
				return "", errors.New("native JSON pointer has an invalid escape")
			}
			if encoded[i] == '~' {
				i++
			}
		}
		segment := strings.ReplaceAll(strings.ReplaceAll(encoded, "~1", "/"), "~0", "~")
		switch node := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = node[segment]
			if !ok {
				return "", errors.New("native JSON pointer is absent")
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) || strconv.Itoa(index) != segment {
				return "", errors.New("native JSON pointer array index is invalid")
			}
			value = node[index]
		default:
			return "", errors.New("native JSON pointer traverses a scalar")
		}
	}
	result, ok := value.(string)
	if !ok {
		return "", errors.New("native JSON pointer does not resolve to a string")
	}
	return result, nil
}

// ListAICandidates reads retained-source proposal rows across the existing entity/event/fact review tables.
// Inputs: source version UUID and result limit. Outputs: bounded pending and decided proposal views.
// Effects: read-only. Choose for AI review, never normalized-message review.
func (s *EntityExtractionStore) ListAICandidates(ctx context.Context, sourceVersionID, afterID string, limit int) ([]service.AIReviewRow, error) {
	if limit < 1 || limit > 500 {
		limit = 500
	}
	rows, err := s.db.Query(ctx, `SELECT id::text, kind, coalesce(attrs->>'reported_kind',kind), coalesce(attrs->>'review_domain',''), review_state, encode(content_sha256,'hex'), coalesce(attrs->'review_decision'->>'decision_id',''), attrs->'candidate' FROM (
 SELECT id, 'entity' AS kind, review_state, content_sha256, attrs FROM working.candidate_entity WHERE source_raw_table='context.source_version' AND source_raw_id=$1
 UNION ALL SELECT id, 'event', review_state, content_sha256, attrs FROM working.candidate_event WHERE source_raw_table='context.source_version' AND source_raw_id=$1
 UNION ALL SELECT id, 'fact', review_state, content_sha256, attrs FROM working.candidate_fact WHERE source_raw_table='context.source_version' AND source_raw_id=$1
) candidates WHERE id::text > $2 ORDER BY id LIMIT $3`, sourceVersionID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]service.AIReviewRow, 0)
	for rows.Next() {
		var row service.AIReviewRow
		if err := rows.Scan(&row.ID, &row.Kind, &row.ReportedKind, &row.ReviewDomain, &row.ReviewState, &row.ContentSHA256, &row.DecisionID, &row.Candidate); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// DecideAICandidate applies an attributed, digest-bound owner decision to one retained-source candidate.
// Inputs: source version, candidate UUID, decision, actor, request digest and decision time.
// Outputs: conflict or success. Effects: review_state and bounded decision receipt in attrs only.
// Choose after owner inspection; this method never promotes a candidate.
func (s *EntityExtractionStore) DecideAICandidate(ctx context.Context, pin service.AISourcePin, candidateID, expectedContentSHA256, decision string, actor entities.Actor, digest string, at time.Time) (string, error) {
	sourceVersionID := pin.SourceVersionID
	if decision != "approved" && decision != "rejected" && decision != "needs_info" {
		return "", service.ErrInvalid{Err: errors.New("decision must be approved, rejected or needs_info")}
	}
	if actor.SubjectUID == "" || actor.Username == "" || len(digest) != 64 || len(expectedContentSHA256) != 64 {
		return "", service.ErrInvalid{Err: errors.New("actor, request digest and expected content digest are required")}
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	decisionID := flow.DeterministicID("ai_review_decision", candidateID, digest)
	receipt, _ := json.Marshal(map[string]any{"decision_id": decisionID, "actor": actor, "request_digest": digest, "approved_content_sha256": expectedContentSHA256, "at": at.UTC(), "decision": decision})
	for _, table := range []string{"working.candidate_entity", "working.candidate_event", "working.candidate_fact"} {
		var current, priorDigest, currentPromotion, currentScope string
		var storedDigest, candidateJSON []byte
		var reportedKind, reviewDomain string
		err := tx.QueryRow(ctx, `SELECT review_state, coalesce(attrs->'review_decision'->>'request_digest',''), content_sha256, attrs->'candidate', coalesce(attrs->>'reported_kind',''), coalesce(attrs->>'review_domain',''), coalesce(attrs->>'graph_promotion',''), coalesce(attrs->>'projection_scope','') FROM `+table+`
WHERE id=$1::uuid AND source_raw_table='context.source_version' AND source_raw_id=$2 FOR UPDATE`, candidateID, sourceVersionID).Scan(&current, &priorDigest, &storedDigest, &candidateJSON, &reportedKind, &reviewDomain, &currentPromotion, &currentScope)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return "", err
		}
		var candidate service.AICandidate
		if err := json.Unmarshal(candidateJSON, &candidate); err != nil {
			return "", entities.ErrConflict
		}
		if candidate.SourceVersionID != pin.SourceVersionID || candidate.SourceObjectID != pin.SourceObjectID || candidate.SourceSHA256 != pin.SourceSHA256 || !sameVersionID(candidate.VersionID, pin.VersionID) {
			return "", entities.ErrConflict
		}
		if candidate.ReportedKind != reportedKind || candidate.ReviewDomain != reviewDomain {
			return "", entities.ErrConflict
		}
		recomputed := service.AICandidateDigest(candidate)
		if hex.EncodeToString(storedDigest) != expectedContentSHA256 || hex.EncodeToString(recomputed[:]) != expectedContentSHA256 {
			return "", entities.ErrConflict
		}
		promotion := aiGraphPromotion(decision, candidate)
		scope := aiProjectionScope(promotion)
		rationale, _ := json.Marshal(aiReviewRationale(sourceVersionID, expectedContentSHA256, actor.SubjectUID, digest, table, reportedKind, reviewDomain, promotion))
		if priorDigest == digest && current == decision {
			if currentPromotion != promotion || currentScope != scope {
				return "", entities.ErrConflict
			}
			var storedRationale, storedDecision, storedTarget, storedReviewer string
			err := tx.QueryRow(ctx, `SELECT rationale, decision, target_kind, reviewer FROM analysis.review_decision WHERE decision_id=$1::uuid AND target_id=$2::uuid`, decisionID, candidateID).Scan(&storedRationale, &storedDecision, &storedTarget, &storedReviewer)
			if err != nil || storedRationale != string(rationale) || storedTarget != table || storedDecision != aiLedgerDecision(decision) || storedReviewer != actor.Username {
				return "", entities.ErrConflict
			}
			return decisionID, tx.Commit(ctx)
		}
		if current != "pending" && current != "needs_info" {
			return "", entities.ErrConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO analysis.review_decision
 (decision_id,target_kind,target_id,reviewer,decision,court_readiness,rationale,decided_at)
 VALUES ($1::uuid,$2,$3::uuid,$4,$5,'not_reviewed',$6,$7)
 ON CONFLICT (decision_id) DO NOTHING`, decisionID, table, candidateID, actor.Username, aiLedgerDecision(decision), string(rationale), at.UTC())
		if err != nil {
			return "", err
		}
		var existingRationale, existingDecision, existingTarget, existingReviewer string
		if err := tx.QueryRow(ctx, `SELECT rationale, decision, target_kind, reviewer FROM analysis.review_decision WHERE decision_id=$1::uuid AND target_id=$2::uuid`, decisionID, candidateID).Scan(&existingRationale, &existingDecision, &existingTarget, &existingReviewer); err != nil {
			return "", err
		}
		if existingRationale != string(rationale) || existingDecision != aiLedgerDecision(decision) || existingTarget != table || existingReviewer != actor.Username {
			return "", entities.ErrConflict
		}
		_, err = tx.Exec(ctx, `UPDATE `+table+` SET review_state=$3,
 attrs=jsonb_set(jsonb_set(jsonb_set(attrs,'{review_decision}',$4::jsonb,true),'{graph_promotion}',to_jsonb($5::text),true),'{projection_scope}',to_jsonb($6::text),true)
WHERE id=$1::uuid AND source_raw_id=$2`, candidateID, sourceVersionID, decision, string(receipt), promotion, scope)
		if err != nil {
			return "", err
		}
		return decisionID, tx.Commit(ctx)
	}
	return "", service.ErrNotFound
}

// aiGraphPromotion marks only owner-approved grounded AI proposals as analysis context.
// Inputs: attributed decision and source-pinned typed candidate. Outputs: projection eligibility.
// Effects: none. Choose at the existing owner-review gate; historical availability is checked by readers.
func aiGraphPromotion(decision string, candidate service.AICandidate) string {
	if decision == "approved" && (candidate.ReviewDomain == "ai_chat_content" || candidate.ReviewDomain == "ai_chat_account") {
		switch candidate.Kind {
		case "entity", "event", "fact":
			return "approved_context"
		}
	}
	return "held"
}

// aiProjectionScope records that eligible AI claims enter attributed analysis context only.
// Inputs: approval promotion label. Outputs: context or none. Effects: none.
// Choose with aiGraphPromotion so replay checks bind both candidate attrs and the owner ledger.
func aiProjectionScope(promotion string) string {
	if promotion == "approved_context" {
		return "context"
	}
	return "none"
}

// aiReviewRationale pins the existing owner decision to an attributed AI context projection.
// Inputs: candidate identity, owner request and eligibility. Outputs: bounded ledger metadata.
// Effects: none. Choose for analysis.review_decision rather than a second approval record.
func aiReviewRationale(sourceVersionID, candidateSHA256, actorUID, requestDigest, table, reportedKind, reviewDomain, promotion string) map[string]any {
	return map[string]any{
		"source_version_id":        sourceVersionID,
		"candidate_content_sha256": candidateSHA256,
		"actor_subject_uid":        actorUID,
		"request_digest":           requestDigest,
		"candidate_kind":           strings.TrimPrefix(table, "working.candidate_"),
		"reported_kind":            reportedKind,
		"review_domain":            reviewDomain,
		"graph_promotion":          promotion,
		"projection_scope":         aiProjectionScope(promotion),
	}
}

// aiLedgerDecision maps the working review state to analysis.review_decision's allowed vocabulary.
// Inputs: approved, rejected or needs_info. Outputs: approved, rejected or needs_context.
// Effects: none. Choose only for the append-only owner decision receipt.
func aiLedgerDecision(decision string) string {
	if decision == "needs_info" {
		return "needs_context"
	}
	return decision
}
