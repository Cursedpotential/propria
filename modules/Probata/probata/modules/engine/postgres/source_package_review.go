// Package review uses SourceMetadataStore's append-only corrections for native source versions.
// Inputs are reviewed metadata/derived assertions and exact source locators; source bodies are excluded.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Cursedpotential/probata/engine/sourcemeta"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PackageReviewSpec names one native metadata or derived-assertion revision.
// Inputs: registered version, displayed revision, attribution and bounded fields/locators.
// Outputs: correction receipt; effects occur only in PersistPackageReview, never original mutation.
type PackageReviewSpec struct {
	SourceVersionRef       string           `json:"source_version_ref"`
	Operation              string           `json:"operation"`
	ItemID                 string           `json:"item_id"`
	Revision               string           `json:"revision"`
	Fields                 map[string]any   `json:"fields"`
	SourceRefs             []map[string]any `json:"source_refs"`
	Reason                 string           `json:"reason"`
	ActorSubjectUID        string           `json:"-"`
	ActorUsername          string           `json:"-"`
	IdempotencyKey         string           `json:"-"`
	CandidateContentSHA256 string           `json:"candidate_content_sha256,omitempty"`
}

// packagePayloadAllowed excludes native conversation/message bodies from durable review metadata.
// Inputs: decoded metadata or derived assertion. Output: admissibility; effects: none.
// Choose before serialization, while full source/chunk/document content remains in native bundles.
func packagePayloadAllowed(v any) bool {
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			switch strings.ToLower(key) {
			case "body", "messages", "turns", "raw_body", "full_text", "message_body", "content", "text", "payload", "raw_reply", "reply", "replies":
				return false
			}
			if !packagePayloadAllowed(child) {
				return false
			}
		}
	case []any:
		for _, child := range value {
			if !packagePayloadAllowed(child) {
				return false
			}
		}
	case []map[string]any:
		for _, child := range value {
			if !packagePayloadAllowed(child) {
				return false
			}
		}
	}
	return true
}

// ValidatePackageReview checks native references and review transitions without custody prerequisites.
// Inputs: one attributed request. Output: validation error or nil; effects: none.
// Choose for registered source versions that have no legacy preview or retained-object digest.
func ValidatePackageReview(spec PackageReviewSpec) error {
	if _, err := uuid.Parse(spec.SourceVersionRef); err != nil {
		return errors.New("source_version_ref must be a UUID")
	}
	if spec.ItemID == "" || len(spec.ItemID) > 480 || strings.ContainsAny(spec.ItemID, "\x00\r\n") {
		return errors.New("bounded item_id required")
	}
	if spec.Revision != "" {
		if _, err := uuid.Parse(spec.Revision); err != nil {
			return errors.New("revision must name the current correction")
		}
	}
	if strings.HasPrefix(spec.ItemID, "candidate_") {
		parts := strings.SplitN(spec.ItemID, ":", 2)
		if len(parts) != 2 {
			return errors.New("canonical review requires an actual working candidate coordinate")
		}
		switch parts[0] {
		case "candidate_entity", "candidate_event", "candidate_fact":
		default:
			return errors.New("unsupported working candidate table")
		}
		if _, err := uuid.Parse(parts[1]); err != nil {
			return errors.New("canonical review requires the actual working candidate UUID")
		}
		if spec.Operation == "metadata" {
			return errors.New("canonical candidate review requires edit, accept, reject or commit")
		}
		hash, err := hex.DecodeString(spec.CandidateContentSHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(spec.CandidateContentSHA256) != spec.CandidateContentSHA256 {
			return errors.New("canonical candidate review requires the displayed content SHA256")
		}
		for key := range spec.Fields {
			switch key {
			case "name", "entity_type", "event_type", "predicate", "statement", "confidence", "occurred_at":
			default:
				return fmt.Errorf("unsupported canonical candidate field %q", key)
			}
		}
	}
	switch spec.Operation {
	case "metadata", "edit", "accept", "reject", "commit":
	default:
		return errors.New("unsupported package review operation")
	}
	if strings.TrimSpace(spec.Reason) == "" || len(spec.Reason) > 4000 {
		return errors.New("bounded review reason required")
	}
	for _, v := range []string{spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey} {
		if strings.TrimSpace(v) == "" || len(v) > 512 || strings.ContainsAny(v, "\x00\r\n") {
			return errors.New("existing actor and idempotency key required")
		}
	}
	if !packagePayloadAllowed(spec.Fields) || !packagePayloadAllowed(spec.SourceRefs) {
		return errors.New("native source/message bodies belong in external bundles, not review metadata")
	}
	if len(spec.SourceRefs) == 0 || len(spec.SourceRefs) > 64 {
		return errors.New("exact source/version/locator references required")
	}
	matched := false
	for _, ref := range spec.SourceRefs {
		if ref["source_version_ref"] == spec.SourceVersionRef && ref["source_ref"] != "" && ref["locator"] != nil {
			matched = true
		}
	}
	if !matched {
		return errors.New("review must retain the registered root source/version/locator")
	}
	raw, err := json.Marshal(spec)
	if err != nil || len(raw) > sourcemeta.MaxValueBytes {
		return errors.New("review exceeds existing metadata correction byte budget")
	}
	return nil
}

// PersistPackageReview appends an attributed native review revision using the existing correction ledger.
// Inputs: real source version, expected correction reference and one review action. Output: durable receipt.
// Effects: correction insert with replay/CAS and atomic native working-candidate edits or existing canonical decisions.
// Choose for metadata/derived assertions; canonical commit requires the immediately accepted revision and exact candidate hash.
func (s *SourceMetadataStore) PersistPackageReview(ctx context.Context, spec PackageReviewSpec) (sourcemeta.Receipt, error) {
	if err := ValidatePackageReview(spec); err != nil {
		return sourcemeta.Receipt{}, err
	}
	raw, _ := json.Marshal(spec)
	sum := sha256.Sum256(append(raw, []byte("\x00"+spec.ActorSubjectUID+"\x00"+spec.ActorUsername+"\x00"+spec.IdempotencyKey)...))
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sourcemeta.Receipt{}, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	var source string
	var matter, court *string
	err = tx.QueryRow(ctx, `SELECT s.source_key,v.matter_id::text,v.court_case_id::text FROM context.source_version v JOIN context.source s ON s.id=v.source_id WHERE v.id=$1::uuid`, spec.SourceVersionRef).Scan(&source, &matter, &court)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourcemeta.Receipt{}, sourcemeta.ErrNotFound
	}
	if err != nil {
		return sourcemeta.Receipt{}, err
	}
	rootMatched := false
	for _, ref := range spec.SourceRefs {
		if ref["source_version_ref"] == spec.SourceVersionRef && ref["source_ref"] == source {
			rootMatched = true
		}
	}
	if !rootMatched {
		return sourcemeta.Receipt{}, fmt.Errorf("%w: review source differs from registered native source", sourcemeta.ErrStaleRevision)
	}
	field := "package:" + spec.ItemID
	if err = tx.QueryRow(ctx, `SELECT 1 FROM (SELECT pg_advisory_xact_lock(hashtextextended($1,0))) lock`, "package-review:"+spec.SourceVersionRef+":"+field).Scan(new(int)); err != nil {
		return sourcemeta.Receipt{}, err
	}
	var existing sourcemeta.Receipt
	var digest []byte
	err = tx.QueryRow(ctx, `SELECT correction_ref::text,receipt_ref,content_digest,revision,recorded_at FROM context.source_metadata_correction WHERE idempotency_key=$1`, spec.IdempotencyKey).Scan(&existing.CorrectionRef, &existing.ReceiptRef, &digest, &existing.Revision, &existing.RecordedAt)
	if err == nil {
		if hex.EncodeToString(digest) != hex.EncodeToString(sum[:]) {
			return existing, sourcemeta.ErrIdempotencyConflict
		}
		existing.ContentDigest = hex.EncodeToString(digest)
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return existing, err
	}
	var priorRef string
	var priorRevision int
	var priorRaw []byte
	err = tx.QueryRow(ctx, `SELECT correction_ref::text,revision,corrected_value FROM context.source_metadata_correction WHERE source_version_id=$1::uuid AND subject_sha256 IS NULL AND field_key=$2 ORDER BY revision DESC LIMIT 1`, spec.SourceVersionRef, field).Scan(&priorRef, &priorRevision, &priorRaw)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return existing, err
	}
	if priorRef != spec.Revision {
		return existing, sourcemeta.ErrStaleRevision
	}
	prior := map[string]any{}
	if len(priorRaw) > 0 {
		if err = json.Unmarshal(priorRaw, &prior); err != nil {
			return existing, err
		}
	}
	state := "unreviewed"
	switch spec.Operation {
	case "accept":
		state = "accepted"
	case "reject":
		state = "rejected"
	case "commit":
		if prior["review_state"] != "accepted" {
			return existing, fmt.Errorf("%w: interpretation commit requires the inspected accepted revision", sourcemeta.ErrStaleRevision)
		}
		if len(spec.Fields) != 0 {
			return existing, fmt.Errorf("%w: commit cannot edit accepted fields; edit and accept a new revision first", sourcemeta.ErrStaleRevision)
		}
		previousRefs, _ := json.Marshal(prior["source_refs"])
		requestRefs, _ := json.Marshal(spec.SourceRefs)
		if string(previousRefs) != string(requestRefs) {
			return existing, fmt.Errorf("%w: commit must retain the accepted source references", sourcemeta.ErrStaleRevision)
		}
		state = "committed"
	case "metadata":
		state = "reviewed"
	}
	previousFields, _ := prior["fields"].(map[string]any)
	if (spec.Operation == "accept" || spec.Operation == "reject") && len(spec.Fields) > 0 && !reflect.DeepEqual(previousFields, spec.Fields) {
		return existing, fmt.Errorf("%w: edit fields before deciding their revision", sourcemeta.ErrStaleRevision)
	}
	spec.Fields = mergePackageFields(previousFields, spec.Fields)
	valueFields := map[string]any{"item_id": spec.ItemID, "operation": spec.Operation, "fields": spec.Fields, "source_refs": spec.SourceRefs, "review_state": state}
	if strings.HasPrefix(spec.ItemID, "candidate_") {
		canonical, canonicalErr := s.applyPackageCandidateTx(ctx, tx, spec, prior, hex.EncodeToString(sum[:]))
		if canonicalErr != nil {
			return existing, canonicalErr
		}
		valueFields["canonical"] = canonical
	}
	value, _ := json.Marshal(valueFields)
	id, err := uuid.NewV7()
	if err != nil {
		return existing, err
	}
	at := s.clock()
	receipt := "metadata-correction://" + id.String()
	var supersedes any
	if priorRef != "" {
		supersedes = priorRef
	}
	_, err = tx.Exec(ctx, `INSERT INTO context.source_metadata_correction
 (correction_ref,source_version_id,subject_sha256,field_key,preview_handle,matter_id,court_case_id,revision,supersedes_ref,action,source_value,corrected_value,change_reason,actor_subject_uid,actor_username,idempotency_key,content_digest,receipt_ref,recorded_at)
 VALUES($1,$2::uuid,NULL,$3,NULL,$4::uuid,$5::uuid,$6,$7,'correct',$8::jsonb,$9::jsonb,$10,$11,$12,$13,$14,$15,$16)`, id, spec.SourceVersionRef, field, matter, court, priorRevision+1, supersedes, jsonParam(priorRaw), value, spec.Reason, spec.ActorSubjectUID, spec.ActorUsername, spec.IdempotencyKey, sum[:], receipt, at)
	if err != nil {
		return existing, err
	}
	if err = tx.Commit(ctx); err != nil {
		return existing, err
	}
	return sourcemeta.Receipt{CorrectionRef: id.String(), ReceiptRef: receipt, ContentDigest: hex.EncodeToString(sum[:]), Revision: priorRevision + 1, RecordedAt: at}, nil
}

// mergePackageFields accumulates partial owner corrections without replacing earlier edits.
// Inputs: prior field overrides and explicit changes (including null). Output: fresh merged map.
// Effects: none; choose for successive editor actions so re-opening preserves every correction.
func mergePackageFields(previous, changes map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range previous {
		result[key] = value
	}
	for key, value := range changes {
		child, ok := value.(map[string]any)
		old, oldOK := result[key].(map[string]any)
		if ok && oldOK {
			result[key] = mergePackageFields(old, child)
		} else {
			result[key] = value
		}
	}
	return result
}
