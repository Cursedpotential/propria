package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/jackc/pgx/v5"
	"go.temporal.io/sdk/temporal"
)

// RepairBindingAdmission appends an attributed explicit readmission receipt to the existing preview event ledger.
// Inputs: owner LIVE request, actual existing source version and approved case. Output: immutable receipt coordinates.
// Effects: one append under the existing preview advisory lock; event zero, source and successful derivatives are unchanged.
// Pick only to repair unknown historical authority, never to replace a valid conflicting receipt or re-register a source.
func (s *ProfferPreviewStore) RepairBindingAdmission(ctx context.Context, in proffer.PreviewAdmissionRepairInput) (proffer.PreviewAdmissionRepairResult, error) {
	refuse := func(err error) (proffer.PreviewAdmissionRepairResult, error) {
		return proffer.PreviewAdmissionRepairResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "InvalidBindingRepair", err)
	}
	if err := proffer.ValidatePreviewAdmissionRepair(in); err != nil {
		return refuse(err)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	defer func() { cleanup, cancel := boundedCleanup(ctx); defer cancel(); _ = tx.Rollback(cleanup) }()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, in.PreviewHandle); err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	var version, source, matter, court, workflowID, runID, requestID string
	err = tx.QueryRow(ctx, `SELECT sn.source_version_id::text,b.source_ref,v.matter_id::text,v.court_case_id::text,b.workflow_id,b.run_id,b.request_id
 FROM context.proffer_preview_binding b JOIN LATERAL (SELECT source_version_id FROM context.proffer_preview_snapshot WHERE preview_handle=b.preview_handle ORDER BY snapshot_seq DESC LIMIT 1) sn ON true
 JOIN context.source_version v ON v.id=sn.source_version_id JOIN context.source src ON src.id=v.source_id
 WHERE b.preview_handle=$1 AND v.workflow_id=b.workflow_id AND src.source_key=b.source_ref`, in.PreviewHandle).Scan(&version, &source, &matter, &court, &workflowID, &runID, &requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return refuse(errors.New("repair binding has no verified registered source lineage"))
	}
	if err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	if version != in.SourceVersionRef || matter != in.MatterID || court != in.CourtCaseID {
		return refuse(errors.New("repair source version or case differs from the existing binding"))
	}
	raw, _ := json.Marshal(in)
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	var eventID int64
	var previous, initial string
	err = tx.QueryRow(ctx, `SELECT event_id,detail FROM context.proffer_preview_event WHERE preview_handle=$1 AND event_type='receipt_recorded' AND phase='operating_mode_repair' ORDER BY event_id DESC LIMIT 1`, in.PreviewHandle).Scan(&eventID, &previous)
	if err == nil {
		var recorded map[string]any
		if json.Unmarshal([]byte(previous), &recorded) != nil || recorded["content_digest"] != digest {
			return refuse(errors.New("binding already has a different explicit repair receipt"))
		}
		return proffer.PreviewAdmissionRepairResult{Status: "completed", PreviewHandle: in.PreviewHandle, SourceVersionRef: version, EventID: eventID, ReceiptRef: fmt.Sprintf("preview-event://%s/%d", in.PreviewHandle, eventID)}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT detail FROM context.proffer_preview_event WHERE preview_handle=$1 AND event_id=0`, in.PreviewHandle).Scan(&initial); err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	binding := previewmodel.Binding{}
	applyBindingAdmission(&binding, initial)
	if binding.OperatingMode != "" {
		return refuse(errors.New("binding already has verified authority; repair refuses replacement"))
	}
	// A malformed or explicit conflicting original mode is not an unknown legacy omission.
	var original map[string]any
	if initial != "" && json.Unmarshal([]byte(initial), &original) != nil {
		return refuse(errors.New("malformed original admission requires separate review"))
	}
	if mode, ok := original["operating_mode"]; ok && mode != "" && mode != in.OperatingMode {
		return refuse(errors.New("explicit original operating mode conflicts with requested repair"))
	}
	for key, want := range map[string]string{"matter_id": matter, "court_case_id": court} {
		if value, ok := original[key]; ok && value != "" && value != want {
			return refuse(errors.New("original admission case conflicts with repair"))
		}
	}
	initialHash := sha256.Sum256([]byte(initial))
	detail, _ := json.Marshal(map[string]any{"receipt_type": "operating_mode_repair", "operating_mode": in.OperatingMode, "matter_id": matter, "court_case_id": court, "source_version_ref": version, "source_ref": source, "original_workflow_id": workflowID, "original_run_id": runID, "original_request_id": requestID, "original_event_id": 0, "original_detail_sha256": hex.EncodeToString(initialHash[:]), "actor_subject_uid": in.ActorSubjectUID, "actor_username": in.ActorUsername, "idempotency_key": in.IdempotencyKey, "reason": in.Reason, "content_digest": digest})
	if len(detail) > 4000 {
		return refuse(errors.New("repair receipt exceeds existing event detail bound"))
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(event_id)+1,0) FROM context.proffer_preview_event WHERE preview_handle=$1`, in.PreviewHandle).Scan(&eventID); err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO context.proffer_preview_event(preview_handle,event_id,event_type,occurred_at,phase,detail) VALUES($1,$2,'receipt_recorded',$3,'operating_mode_repair',$4)`, in.PreviewHandle, eventID, s.clock(), string(detail)); err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return proffer.PreviewAdmissionRepairResult{}, err
	}
	return proffer.PreviewAdmissionRepairResult{Status: "completed", PreviewHandle: in.PreviewHandle, SourceVersionRef: version, EventID: eventID, ReceiptRef: fmt.Sprintf("preview-event://%s/%d", in.PreviewHandle, eventID)}, nil
}
