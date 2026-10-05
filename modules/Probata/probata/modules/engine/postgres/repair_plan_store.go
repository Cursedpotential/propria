// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Durable side of the repair workflow builder: resolve a plan's Review run
// (its anchor) and record one append-only receipt per plan step.
//
// No schema change: the anchor is read from context.proffer_preview_binding,
// context.source_version, context.handler_detected_format and
// context.repair_assessment, and receipts land in context.activity_execution /
// context.activity_receipt exactly like every other Activity's — both accept a
// new activity name, have no triggers, and the engine role platform_runtime
// holds SELECT+INSERT on all six (verified read-only against the live
// platform database on 2026-09-25). Because receipts are keyed to the anchor
// run's source version, repair steps also appear in that run's operation
// history in Review.

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

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/repairplan"
)

// maxAnchorEvidenceBytes bounds the persisted repair assessment copied into
// an anchor; it informs the proposer and never enters workflow history.
const maxAnchorEvidenceBytes = 64 << 10

// RepairPlanStore implements repairplan.AnchorResolver and
// activities.RepairStepReceiptStore.
type RepairPlanStore struct {
	db    DB
	clock func() time.Time
}

// NewRepairPlanStore wraps the platform database.
func NewRepairPlanStore(db DB) (*RepairPlanStore, error) {
	if db == nil {
		return nil, errors.New("postgres repair plan store: database is required")
	}
	return &RepairPlanStore{db: db, clock: func() time.Time { return time.Now().UTC() }}, nil
}

const anchorColumns = `
	SELECT binding.preview_handle, binding.request_id, binding.source_ref, binding.workflow_id,
	       binding.parser_options_ref, version.id, COALESCE(version.declared_format, ''),
	       version.matter_id, version.court_case_id,
 COALESCE((SELECT detail FROM context.proffer_preview_event WHERE preview_handle=binding.preview_handle AND event_id=0), '')
	FROM context.proffer_preview_binding binding
	LEFT JOIN LATERAL (
	    SELECT id, declared_format, matter_id, court_case_id
	    FROM context.source_version
	    WHERE workflow_id = binding.workflow_id
	    ORDER BY version_ordinal DESC, created_at DESC LIMIT 1
	) version ON true`

// ResolveAnchor finds the plan's Review run: by preview handle when given,
// else the newest run bound to the source reference.
func (s *RepairPlanStore) ResolveAnchor(ctx context.Context, sourceRef, previewHandle string) (repairplan.Anchor, error) {
	var row pgx.Row
	if handle := strings.TrimSpace(previewHandle); handle != "" {
		row = s.db.QueryRow(ctx, anchorColumns+`
	WHERE binding.preview_handle = $1`, handle)
	} else {
		if strings.TrimSpace(sourceRef) == "" {
			return repairplan.Anchor{}, repairplan.ErrAnchorNotFound
		}
		row = s.db.QueryRow(ctx, anchorColumns+`
	WHERE binding.source_ref = $1
	ORDER BY binding.created_at DESC, binding.preview_handle DESC LIMIT 1`, strings.TrimSpace(sourceRef))
	}
	var anchor repairplan.Anchor
	var versionID, matterID, courtCaseID *uuid.UUID
	var modeDetail string
	if err := row.Scan(&anchor.PreviewHandle, &anchor.RequestID, &anchor.SourceRef, &anchor.WorkflowID,
		&anchor.ParserOptionsRef, &versionID, &anchor.DeclaredFormat, &matterID, &courtCaseID, &modeDetail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repairplan.Anchor{}, repairplan.ErrAnchorNotFound
		}
		return repairplan.Anchor{}, fmt.Errorf("read the repair plan's Review run: %w", err)
	}
	anchor.OperatingMode = recordedOperatingMode(modeDetail)
	if matterID != nil {
		anchor.MatterID = matterID.String()
	}
	if courtCaseID != nil {
		anchor.CourtCaseID = courtCaseID.String()
	}
	if versionID == nil {
		return anchor, nil
	}
	anchor.SourceVersionID = versionID.String()
	if err := s.db.QueryRow(ctx, `
		SELECT format_id FROM context.handler_detected_format
		WHERE source_version_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, *versionID).Scan(&anchor.DetectedFormat); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return repairplan.Anchor{}, fmt.Errorf("read the run's detected format: %w", err)
	}
	var detection, report []byte
	if err := s.db.QueryRow(ctx, `
		SELECT detection, preview->'report' FROM context.repair_assessment
		WHERE source_version_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, *versionID).Scan(&detection, &report); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return repairplan.Anchor{}, fmt.Errorf("read the run's repair assessment: %w", err)
	}
	anchor.RepairDetection = boundedEvidence(detection)
	anchor.RepairReport = boundedEvidence(report)
	return anchor, nil
}

func boundedEvidence(raw []byte) json.RawMessage {
	if len(raw) == 0 || len(raw) > maxAnchorEvidenceBytes || string(raw) == "null" || !json.Valid(raw) {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

// repairStepIdempotencyKey names one step of one run of one plan. A retried
// receipt Activity lands on the same execution and returns its first receipt;
// a new run of the same plan records its own receipts.
func repairStepIdempotencyKey(request repairplan.ReceiptRequest) string {
	return fmt.Sprintf("repair-plan:%s:%s:%02d:%s", request.WorkflowID, request.RunID, request.StepIndex, request.StepID)
}

// RecordRepairStepReceipt writes one append-only receipt for one plan step
// and returns its id. It is idempotent on (source version, activity, step).
func (s *RepairPlanStore) RecordRepairStepReceipt(ctx context.Context, request repairplan.ReceiptRequest, attempt int32) (string, error) {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(request.OperatingMode)); err != nil {
		return "", err
	}
	if !caseidentity.AdmittedIdentity(request.MatterID, request.CourtCaseID) {
		return "", errors.New("repair receipt requires the approved case identity")
	}
	sourceID, err := uuid.Parse(strings.TrimSpace(request.SourceVersionID))
	if err != nil {
		return "", fmt.Errorf("repair step receipt source version: %w", err)
	}
	if attempt < 1 {
		attempt = 1
	}
	resultRef, errorDetail, err := repairReceiptBodies(request)
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	rollback := true
	defer func() {
		if rollback {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	executionID, err := parserEnsureExecution(ctx, tx, sourceID, request.WorkflowID, request.Activity, repairStepIdempotencyKey(request))
	if err != nil {
		return "", err
	}
	var prior uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM context.activity_receipt
		WHERE activity_execution_id = $1 ORDER BY attempt LIMIT 1`, executionID).Scan(&prior)
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			return "", err
		}
		rollback = false
		return prior.String(), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	receiptID, now := uuid.New(), s.clock()
	if _, err = tx.Exec(ctx, `
		INSERT INTO context.activity_receipt
		    (id, activity_execution_id, attempt, status, started_at, completed_at, result_ref, error_detail)
		VALUES ($1, $2, $3::integer, $4, $5, $5, $6, $7)`,
		receiptID, executionID, attempt, request.Status, now, resultRef, errorDetail); err != nil {
		return "", fmt.Errorf("persist repair step receipt: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	rollback = false
	return receiptID.String(), nil
}

// repairReceiptBodies renders the receipt's result_ref (success) or
// error_detail (failure) — exactly one, as context.activity_receipt_check
// requires. Both carry references and counts only.
func repairReceiptBodies(request repairplan.ReceiptRequest) (resultRef, errorDetail []byte, err error) {
	common := map[string]any{
		"ref_kind": "repair_plan_step", "plan_id": request.PlanID, "workflow_id": request.WorkflowID,
		"run_id": request.RunID, "step_id": request.StepID, "step_index": request.StepIndex,
		"activity": request.Activity, "input_ref": request.InputRef,
	}
	switch request.Status {
	case repairplan.ReceiptSuccess:
		if request.Result == nil || strings.TrimSpace(request.Result.OutputRef) == "" {
			return nil, nil, errors.New("a success receipt requires the step's output reference")
		}
		common["ref"] = request.Result.OutputRef
		common["output_ref"] = request.Result.OutputRef
		common["output_type"] = request.Result.OutputType
		common["output_kind"] = request.Result.OutputKind
		common["output_sha256"] = request.Result.OutputSHA256
		common["reentry_ref"] = request.Result.ReentryRef
		common["reused"] = request.Result.Reused
		if len(request.Result.Summary) > 0 && len(request.Result.Summary) <= maxAnchorEvidenceBytes && json.Valid(request.Result.Summary) {
			common["summary"] = request.Result.Summary
		}
		resultRef, err = json.Marshal(common)
		return resultRef, nil, err
	case repairplan.ReceiptFailed:
		if strings.TrimSpace(request.Error) == "" {
			return nil, nil, errors.New("a failure receipt requires the failure reason")
		}
		common["message"] = request.Error
		errorDetail, err = json.Marshal(common)
		return nil, errorDetail, err
	}
	return nil, nil, fmt.Errorf("unknown repair receipt status %q", request.Status)
}
