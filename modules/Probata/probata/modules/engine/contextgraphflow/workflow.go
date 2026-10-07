// Byline: Codex · GPT-6.1-sol · 2026-10-07
package contextgraphflow

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func boundedRef(s string) bool {
	return len(s) > 0 && len(s) <= 2000 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func boundedID(s string) bool { return boundedRef(s) && len(s) <= 240 }
func digest(s string) bool {
	raw, e := hex.DecodeString(s)
	return e == nil && len(raw) == 32 && strings.ToLower(s) == s
}
func validateRequest(r Request) error {
	for _, s := range []string{r.RequestID, r.SourceVersionID, r.NormalizedGenerationID, r.VerificationID, r.OperatingMode, r.MatterID, r.CourtCaseID, r.AccessPolicyID, r.CreatedByService} {
		if !boundedID(s) {
			return errors.New("verified AI binding and managed graph scope required")
		}
	}
	if !boundedRef(r.PreparedRef) || !boundedRef(r.WorkProductsRef) || r.ExpectedSourceTurns < 1 || r.ExpectedSourceTurns > 65536 || r.ExpectedCreatedWorks < 0 || r.ExpectedCreatedWorks > 65536 || r.ExpectedConversations < 1 || r.ExpectedConversations > 65536 || len(r.SourcePins) > 64 {
		return errors.New("bounded predecessor refs and explicit completeness counts required")
	}
	return nil
}

func validatePreparation(r Request, p PreparationResult, workflowID, runID string) error {
	if p.RequestID != r.RequestID || p.SourceVersionID != r.SourceVersionID || p.NormalizedGenerationID != r.NormalizedGenerationID || p.VerificationID != r.VerificationID || p.OperatingMode != r.OperatingMode || p.MatterID != r.MatterID || p.CourtCaseID != r.CourtCaseID || p.TemporalWorkflowID != workflowID || p.TemporalRunID != runID {
		return errors.New("preparation changed verified AI bindings or Temporal execution")
	}
	raw, e := json.Marshal(p)
	if e != nil || len(raw) > 256<<10 || !boundedRef(p.ManifestRef) || !digest(p.ManifestHash) || p.Batches < 1 || p.Batches > 512 || len(p.BatchRefs) != p.Batches || p.SourceTurns != r.ExpectedSourceTurns || p.CreatedWorks != r.ExpectedCreatedWorks || p.Conversations != r.ExpectedConversations {
		return errors.New("preparation manifest or completeness counts differ")
	}
	turns, works := 0, 0
	seen := map[string]bool{}
	refs := map[string]bool{}
	for _, b := range p.BatchRefs {
		if !boundedRef(b.BundleRef) || !digest(b.BundleSHA256) || !boundedID(b.GenerationID) || b.ExtractionRunRef != r.PreparedRef || seen[b.GenerationID] || refs[b.BundleRef] || b.SourceTurns < 0 || b.CreatedWorks < 0 || b.SourceTurns > b.Nodes || b.CreatedWorks > b.Nodes || b.Nodes < 1 || b.Nodes > 128 || b.Edges < 1 || b.Edges > 256 {
			return errors.New("invalid, duplicated or incomplete projection batch")
		}
		seen[b.GenerationID] = true
		refs[b.BundleRef] = true
		turns += b.SourceTurns
		works += b.CreatedWorks
	}
	if turns != p.SourceTurns || works != p.CreatedWorks {
		return errors.New("batch manifest omitted or duplicated source/work counts")
	}
	return nil
}

func validateBatchResult(b BatchRef, r BatchResult, workflowID, runID string) error {
	if r.BundleSHA256 != b.BundleSHA256 || r.Receipt.GenerationID != b.GenerationID || !r.Receipt.Active || r.Receipt.NodeCount != b.Nodes || r.Receipt.EdgeCount != b.Edges || r.SourceTurns != b.SourceTurns || r.CreatedWorks != b.CreatedWorks || !boundedRef(r.Receipt.GenerationRef) || !boundedRef(r.Receipt.CheckpointRef) || !digest(r.Receipt.BundleHash) || r.TemporalWorkflowID != workflowID || r.TemporalRunID != runID || !boundedID(r.TemporalActivityID) || r.TemporalAttempt < 1 {
		return errors.New("projection readback changed batch pins, counts or actual execution")
	}
	return nil
}

// ContextGraphWorkflow prepares, projects and independently verifies complete retained AI graphs.
// Input is the exact existing AI binding/predecessor refs, managed scope and explicit
// expected counts; output is reference-only manifest/checkpoints/counts. Effects
// schedule Python preparation on evidence-pipeline and atomic Go project/readback
// Activities on this workflow's existing task queue. Pick downstream of existing
// extraction; this standalone orchestrator never alters Proffer ingestion or runs a model.
func ContextGraphWorkflow(ctx workflow.Context, request Request) (Result, error) {
	if e := validateRequest(request); e != nil {
		return Result{}, e
	}
	info := workflow.GetInfo(ctx)
	if info.TaskQueueName == "" {
		return Result{}, errors.New("existing Go worker task queue required")
	}
	retries := &temporal.RetryPolicy{InitialInterval: time.Second, BackoffCoefficient: 2, MaximumInterval: 30 * time.Second, MaximumAttempts: 4}
	prepareCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{TaskQueue: PythonTaskQueue, StartToCloseTimeout: 20 * time.Minute, ScheduleToCloseTimeout: 90 * time.Minute, HeartbeatTimeout: time.Minute, RetryPolicy: retries})
	var prepared PreparationResult
	if e := workflow.ExecuteActivity(prepareCtx, PrepareActivityName, request).Get(ctx, &prepared); e != nil {
		return Result{}, e
	}
	if e := validatePreparation(request, prepared, info.WorkflowExecution.ID, info.WorkflowExecution.RunID); e != nil {
		return Result{}, e
	}
	goCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{TaskQueue: info.TaskQueueName, StartToCloseTimeout: 3 * time.Minute, ScheduleToCloseTimeout: 20 * time.Minute, HeartbeatTimeout: 30 * time.Second, RetryPolicy: retries})
	result := Result{ManifestRef: prepared.ManifestRef, ManifestHash: prepared.ManifestHash, Conversations: prepared.Conversations, Batches: prepared.Batches, TemporalWorkflowID: info.WorkflowExecution.ID, TemporalRunID: info.WorkflowExecution.RunID}
	for _, batch := range prepared.BatchRefs {
		input := BatchRequest{Request: request, BundleRef: batch.BundleRef, BundleSHA256: batch.BundleSHA256, CaseID: request.CourtCaseID, GenerationID: batch.GenerationID, ExtractionRunRef: batch.ExtractionRunRef}
		var projected, verified BatchResult
		if e := workflow.ExecuteActivity(goCtx, ProjectActivityName, input).Get(ctx, &projected); e != nil {
			return Result{}, e
		}
		if e := validateBatchResult(batch, projected, info.WorkflowExecution.ID, info.WorkflowExecution.RunID); e != nil {
			return Result{}, e
		}
		if e := workflow.ExecuteActivity(goCtx, VerifyActivityName, input).Get(ctx, &verified); e != nil {
			return Result{}, e
		}
		if e := validateBatchResult(batch, verified, info.WorkflowExecution.ID, info.WorkflowExecution.RunID); e != nil {
			return Result{}, e
		}
		if projected.Receipt != verified.Receipt {
			return Result{}, errors.New("independent readback checkpoint differs from projection")
		}
		result.SourceTurns += verified.SourceTurns
		result.CreatedWorks += verified.CreatedWorks
		result.Verified = append(result.Verified, verified.Receipt)
	}
	if result.SourceTurns != request.ExpectedSourceTurns || result.CreatedWorks != request.ExpectedCreatedWorks || len(result.Verified) != prepared.Batches {
		return Result{}, errors.New("independently verified graph completeness differs")
	}
	raw, e := json.Marshal(result)
	if e != nil || len(raw) > 256<<10 {
		return Result{}, errors.New("workflow reference result exceeds bound")
	}
	return result, nil
}
