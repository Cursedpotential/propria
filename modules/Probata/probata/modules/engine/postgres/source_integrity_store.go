// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package postgres

// Byline: Codex · 2026-10-04. Receipt/reference sibling: DeriveStore; no schema migration or source_metadata writes.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourceintegrity"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// IntegrityOriginalOpener is the existing retained-original read seam, without any hash operations.
// Input: retained-object reference. Output: read-only stream. Side effects: source reads only.
type IntegrityOriginalOpener interface {
	OpenOriginal(context.Context, proffer.Ref) (io.ReadCloser, error)
}

// SourceIntegrityStore binds one byte assessment to the existing context Activity execution and receipt tables.
// Inputs: database and retained-original opener. Outputs: streams and immutable operation refs.
// Side effects: source resolution and integrity receipts only; pick instead of metadata or hash repositories for zero-content checks.
type SourceIntegrityStore struct {
	db        DB
	originals IntegrityOriginalOpener
	clock     func() time.Time
}

// NewSourceIntegrityStore constructs a fail-closed receipt adapter using the existing retained source resolver.
// Inputs: database and opener. Output: adapter/error. Side effects: none; connections are used only by operation methods.
// Pick alongside DeriveStore for independent integrity jobs; it creates no schema or processing pipeline.
func NewSourceIntegrityStore(db DB, originals IntegrityOriginalOpener) (*SourceIntegrityStore, error) {
	if db == nil || originals == nil {
		return nil, errors.New("source integrity requires database and original opener")
	}
	return &SourceIntegrityStore{db: db, originals: originals, clock: func() time.Time { return time.Now().UTC() }}, nil
}

// integrityEvidence is bounded persisted coverage with exact source/check/operation identity.
// Inputs: references, scanner evidence, run ID. Output: receipt JSON; no body, format interpretation, hashes, or source metadata.
type integrityEvidence struct {
	RefKind          string                     `json:"ref_kind,omitempty"`
	RefID            string                     `json:"ref_id,omitempty"`
	SourceVersionRef proffer.Ref                `json:"source_version_ref"`
	OriginalRef      proffer.Ref                `json:"original_ref"`
	OperationID      string                     `json:"operation_workflow_id"`
	OperationRunID   string                     `json:"operation_run_id,omitempty"`
	ActivityID       string                     `json:"activity_id,omitempty"`
	ExecutionHost    string                     `json:"execution_host,omitempty"`
	Assessment       sourceintegrity.Assessment `json:"assessment"`
}

// integrityCoordinate validates compact references and pins one operation to its immutable source and check version.
// Input: StageRequest. Output: source/original UUIDs and idempotency key. Side effects: none.
// Pick before receipt reads/writes; repeating the same operation recovers its terminal receipt rather than retrying processing.
func integrityCoordinate(req proffer.StageRequest) (uuid.UUID, uuid.UUID, string, error) {
	source, err := uuid.Parse(string(req.SourceVersionRef))
	if err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("source integrity version reference must be a UUID")
	}
	original, err := uuid.Parse(string(req.Refs["original"]))
	if err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("source integrity original reference must be a UUID")
	}
	op := string(req.Refs["integrity_operation"])
	if strings.TrimSpace(req.RequestID) == "" || len(req.RequestID) > 256 || strings.TrimSpace(op) == "" || len(op) > 256 || len(req.DeclaredFormat) > 128 || len(req.Refs) != 2 || req.MatterID != "" || req.CourtCaseID != "" {
		return uuid.Nil, uuid.Nil, "", errors.New("source integrity requires request and operation IDs")
	}
	return source, original, fmt.Sprintf("source-integrity:%s:%s:%s", op, original, sourceintegrity.CheckVersion), nil
}

// integritySource verifies original/source/workflow binding without opening bytes.
// Inputs: query boundary, request, optional transactional lock. Output: authoritative retained byte length.
// Side effects: bounded SQL read; pick before both replay and a fresh scan so wrong-source receipts cannot be reused.
func integritySource(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, req proffer.StageRequest, lock bool) (int64, error) {
	source, original, _, err := integrityCoordinate(req)
	if err != nil {
		return -1, err
	}
	sql := `SELECT version.workflow_id, version.status, version.declared_format, object.byte_length
		FROM context.source_version version JOIN context.retained_object object ON object.id=version.original_object_id
		WHERE version.id=$1::uuid AND version.original_object_id=$2::uuid`
	if lock {
		sql += " FOR UPDATE OF version"
	}
	var workflowID, status, declared string
	var size int64
	if err = query.QueryRow(ctx, sql, source, original).Scan(&workflowID, &status, &declared, &size); err != nil {
		return -1, fmt.Errorf("resolve integrity source binding: %w", err)
	}
	if workflowID != req.RequestID || status != "retained" || size < 0 || (req.DeclaredFormat != "" && declared != req.DeclaredFormat) {
		return -1, errors.New("integrity source is not the retained original of this request")
	}
	return size, nil
}

// OpenSourceIntegrity opens only the original already bound to this retained source request.
// Input: reference-only request. Output: read-only stream and authoritative expected size/error.
// Side effects: SQL resolution and source opening; pick after LoadSourceIntegrity found no prior outcome.
func (s *SourceIntegrityStore) OpenSourceIntegrity(ctx context.Context, req proffer.StageRequest) (io.ReadCloser, int64, error) {
	size, err := integritySource(ctx, s.db, req, false)
	if err != nil {
		return nil, -1, err
	}
	reader, err := s.originals.OpenOriginal(ctx, req.Refs["original"])
	return reader, size, err
}

// LoadSourceIntegrity recovers the exact prior terminal outcome before any source stream is reopened.
// Input: source/check/operation references. Output: immutable refs, found flag, or binding/storage error.
// Side effects: bounded SQL reads only; failures are terminal for this operation ID, and an explicit new job may assess again.
func (s *SourceIntegrityStore) LoadSourceIntegrity(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, bool, error) {
	size, err := integritySource(ctx, s.db, req, false)
	if err != nil {
		return proffer.StageResult{}, false, err
	}
	source, _, key, _ := integrityCoordinate(req)
	return loadIntegrityReceipt(ctx, s.db, req, size, source, key)
}

// loadIntegrityReceipt reads and validates persisted evidence instead of inferring an assessment from receipt existence.
// Inputs: query boundary and pinned coordinates. Outputs: exact terminal Activity result/found/error.
// Side effects: one bounded receipt query; pick for both normal recovery and concurrent completion serialization.
func loadIntegrityReceipt(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, req proffer.StageRequest, size int64, source uuid.UUID, key string) (proffer.StageResult, bool, error) {
	var id uuid.UUID
	var status string
	var payload []byte
	err := query.QueryRow(ctx, `SELECT receipt.id, receipt.status, COALESCE(receipt.result_ref, receipt.error_detail)
		FROM context.activity_receipt receipt JOIN context.activity_execution execution ON execution.id=receipt.activity_execution_id
		WHERE execution.source_version_id=$1::uuid AND execution.workflow_id=$2 AND execution.activity_name=$3
		AND execution.idempotency_key=$4 AND receipt.status IN ('success','failed') ORDER BY receipt.attempt LIMIT 1`,
		source, req.RequestID, string(stagegraph.AssessSourceIntegrity), key).Scan(&id, &status, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return proffer.StageResult{}, false, nil
	}
	if err != nil {
		return proffer.StageResult{}, false, fmt.Errorf("load integrity receipt: %w", err)
	}
	var evidence integrityEvidence
	if json.Unmarshal(payload, &evidence) != nil || evidence.SourceVersionRef != req.SourceVersionRef || evidence.OriginalRef != req.Refs["original"] || evidence.OperationID != string(req.Refs["integrity_operation"]) {
		return proffer.StageResult{}, false, errors.New("stored integrity receipt identity is inconsistent")
	}
	if err = validateIntegrityAssessment(evidence.Assessment, size); err != nil {
		return proffer.StageResult{}, false, err
	}
	result := proffer.StageResult{Stage: stagegraph.AssessSourceIntegrity, ReceiptRef: proffer.Ref(id.String())}
	if status == "success" && evidence.Assessment.Complete && evidence.RefKind == "source_integrity_assessment" && evidence.RefID == id.String() {
		result.Status, result.Ref = proffer.StatusSuccess, proffer.Ref(evidence.RefID)
	} else if status == "failed" && !evidence.Assessment.Complete && evidence.RefID == "" && evidence.RefKind == "" {
		result.Status, result.Reason = proffer.StatusFailed, evidence.Assessment.Reason
	} else {
		return proffer.StageResult{}, false, errors.New("stored integrity receipt outcome is inconsistent")
	}
	return result, true, nil
}

// validateIntegrityAssessment rejects evidence that exceeds this unit's byte-only contract.
// Inputs: assessment and retained byte length. Output: validation error. Side effects: none.
// Pick before persistence/replay; complete nonzero checks stay unknown and cannot become verified_good.
func validateIntegrityAssessment(a sourceintegrity.Assessment, size int64) error {
	if a.CheckVersion != sourceintegrity.CheckVersion || a.FormatValidation != "unassessed" || a.CheckedBytes < 0 || a.ExpectedBytes != size {
		return errors.New("integrity coverage identity or expected length is inconsistent")
	}
	if a.Complete && (!a.EOFReached || a.CheckedBytes != size) {
		return errors.New("complete integrity coverage lacks EOF or exact size")
	}
	switch a.Reason {
	case "stream_not_checked", "invalid_stream_input", "context_canceled", "source_size_mismatch", "format_validation_unassessed", "source_zero_bytes", "every_source_byte_zero", "stream_read_failed", "stream_no_progress", "source_access_failed", "source_reader_unavailable", "integrity_stream_close_failed":
	default:
		return errors.New("integrity reason is not a fixed evidence code")
	}
	switch a.Status {
	case "zero_byte":
		if !a.Complete || size != 0 || a.NonzeroObserved || a.Reason != "source_zero_bytes" {
			return errors.New("invalid zero-byte coverage")
		}
	case "all_zero":
		if !a.Complete || size == 0 || a.NonzeroObserved || a.Reason != "every_source_byte_zero" {
			return errors.New("invalid all-zero coverage")
		}
	case "unknown":
		if a.Complete && (!a.NonzeroObserved || a.Reason != "format_validation_unassessed") {
			return errors.New("nonzero coverage must leave usability unassessed")
		}
	default:
		return errors.New("unsupported integrity status")
	}
	return nil
}

// PersistSourceIntegrity appends one exact integrity receipt with a short transaction after streaming ends.
// Input: bound request, attempt, run identity and bounded assessment. Output: exact result/receipt refs.
// Side effects: only context.activity_execution/activity_receipt inserts; no source, metadata, hash or canonical writes.
// Pick independently of DeriveStore; terminal recovery serializes concurrent completions without overwriting prior evidence.
func (s *SourceIntegrityStore) PersistSourceIntegrity(ctx context.Context, spec activities.SourceIntegrityReceiptSpec) (proffer.StageResult, error) {
	source, _, key, err := integrityCoordinate(spec.Request)
	if err != nil || spec.Attempt < 1 {
		return proffer.StageResult{}, errors.New("integrity receipt requires valid coordinates and positive attempt")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return proffer.StageResult{}, err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	size, err := integritySource(ctx, tx, spec.Request, true)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if spec.Assessment.ExpectedBytes == -1 && !spec.Assessment.Complete {
		spec.Assessment.ExpectedBytes = size
	}
	if err = validateIntegrityAssessment(spec.Assessment, size); err != nil {
		return proffer.StageResult{}, err
	}
	execution, err := parserEnsureExecution(ctx, tx, source, spec.Request.RequestID, string(stagegraph.AssessSourceIntegrity), key)
	if err != nil {
		return proffer.StageResult{}, err
	}
	prior, found, err := loadIntegrityReceipt(ctx, tx, spec.Request, size, source, key)
	if err != nil {
		return proffer.StageResult{}, err
	}
	if found {
		if err = tx.Commit(ctx); err != nil {
			return proffer.StageResult{}, err
		}
		rollback = false
		return prior, nil
	}
	id, now := uuid.New(), s.clock().UTC()
	if spec.StartedAt.IsZero() || spec.StartedAt.After(now) {
		return proffer.StageResult{}, errors.New("integrity receipt requires observed start time before completion")
	}
	evidence := integrityEvidence{SourceVersionRef: spec.Request.SourceVersionRef, OriginalRef: spec.Request.Refs["original"], OperationID: string(spec.Request.Refs["integrity_operation"]), OperationRunID: spec.OperationRunID, Assessment: spec.Assessment, ActivityID: spec.ActivityID, ExecutionHost: spec.ExecutionHost}
	result := proffer.StageResult{Stage: stagegraph.AssessSourceIntegrity, ReceiptRef: proffer.Ref(id.String())}
	status := "failed"
	if spec.Assessment.Complete {
		status = "success"
		evidence.RefKind, evidence.RefID = "source_integrity_assessment", id.String()
		result.Status, result.Ref = proffer.StatusSuccess, proffer.Ref(id.String())
	} else {
		result.Status, result.Reason = proffer.StatusFailed, spec.Assessment.Reason
	}
	payload, err := json.Marshal(evidence)
	if err != nil {
		return proffer.StageResult{}, err
	}
	var resultJSON, errorJSON any
	if status == "success" {
		resultJSON = payload
	} else {
		errorJSON = payload
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.activity_receipt(id,activity_execution_id,attempt,status,started_at,completed_at,result_ref,error_detail)
		VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb)`, id, execution, spec.Attempt, status, spec.StartedAt.UTC(), now, resultJSON, errorJSON); err != nil {
		return proffer.StageResult{}, fmt.Errorf("persist integrity receipt: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return proffer.StageResult{}, err
	}
	rollback = false
	return result, nil
}

var _ activities.SourceIntegrityStore = (*SourceIntegrityStore)(nil)
