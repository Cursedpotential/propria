package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/runtimeapi/previewmodel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResolveOperationResults routes exact source identity to its existing content store before resolving artifacts.
// Inputs: opaque operation handle. Outputs: source-kind/store discriminator and truthful availability.
// Effects: SELECT only. Human/other legacy generations use receipt pins; AI chats use native Weaviate references, never PG chat rows.
func (s *ProfferPreviewStore) ResolveOperationResults(ctx context.Context, handle string) (previewmodel.OperationResults, error) {
	binding, err := s.Binding(ctx, handle)
	if err != nil {
		return previewmodel.OperationResults{}, err
	}
	var sourceCount int
	var sourceRef *string
	var sourceMatches bool
	var declaredFormat *string
	err = s.db.QueryRow(ctx, `SELECT count(*), min(version.id::text), COALESCE(bool_and(source.source_key=$3),true), min(version.declared_format)
 FROM context.source_version version JOIN context.source source ON source.id=version.source_id
 WHERE version.workflow_id IN ($1,$2)`, binding.RequestID, binding.WorkflowID, binding.SourceRef).Scan(&sourceCount, &sourceRef, &sourceMatches, &declaredFormat)
	if err != nil {
		return previewmodel.OperationResults{}, fmt.Errorf("resolve exact operation source: %w", err)
	}
	if sourceCount > 1 {
		return previewmodel.OperationResults{Status: "inconsistent", Reason: "Operation request and workflow identify multiple source versions; no generation was selected."}, nil
	}
	if !sourceMatches {
		return previewmodel.OperationResults{Status: "inconsistent", Reason: "Registered source locator differs from the operation's exact source reference."}, nil
	}
	if sourceRef != nil {
		sourceID, parseErr := uuid.Parse(*sourceRef)
		if parseErr != nil {
			return previewmodel.OperationResults{}, parseErr
		}
		if binding.SourceVersionID != uuid.Nil && binding.SourceVersionID != sourceID {
			return previewmodel.OperationResults{Status: "inconsistent", Reason: "Preview source does not match the exact operation source."}, nil
		}
		binding.SourceVersionID = sourceID
	}
	if declaredFormat != nil {
		if result, ai := aiOperationResults(binding, *declaredFormat); ai {
			return result, nil
		}
	}
	stages, err := s.OperationStages(ctx, handle)
	if err != nil {
		return previewmodel.OperationResults{}, err
	}
	result := operationResultPins(binding, stages)
	result.ContentStore, result.SourceKind, result.SourceRef = "postgres", "unknown", string(binding.SourceRef)
	return s.validateOperationResultLineage(ctx, result, binding.RequestID, binding.WorkflowID)
}

// aiOperationResults identifies only registered AI formats and refuses a guessed historical-to-native mapping.
// Inputs: exact source binding and stored declared format. Outputs: Weaviate route with unqueried availability.
// Effects: none. Choose before any generation lookup; JSON filenames and generic message formats are not AI proof.
func aiOperationResults(binding previewmodel.Binding, declaredFormat string) (previewmodel.OperationResults, bool) {
	format := strings.ToLower(strings.TrimSpace(declaredFormat))
	nativeAI := format == "chatgpt" || format == "claude" || format == "gemini_markdown"
	if !nativeAI && !contextsearch.IsAIChatFormat(format) {
		return previewmodel.OperationResults{}, false
	}
	return previewmodel.OperationResults{Status: "not_queried", ContentStore: "weaviate", SourceKind: "ai_chat", SourceRef: string(binding.SourceRef), SourceVersionID: binding.SourceVersionID, Reason: "AI chat content uses the native Weaviate context reader. This historical operation has no verified current index reference; inspect its original source or use context search. PostgreSQL chat generations are not applicable. No Weaviate availability check was performed."}, true
}

// validateOperationResultLineage verifies source and generation membership for exact receipt pins.
// Inputs: selected pins and request ID. Outputs: validated availability or a precise inconsistency.
// Effects: SELECT only; use after resolving references, never as a generation-selection query.
func (s *ProfferPreviewStore) validateOperationResultLineage(ctx context.Context, result previewmodel.OperationResults, requestID, workflowID string) (previewmodel.OperationResults, error) {
	if result.Status == "inconsistent" || result.SourceVersionID == uuid.Nil {
		return result, nil
	}
	// Registration records the request ID as workflow_id; do not join by a browser flag or latest source.
	var exactSource bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (
   SELECT 1 FROM context.source_version WHERE id=$1::uuid AND workflow_id IN ($2,$3)
 )`, result.SourceVersionID, requestID, workflowID).Scan(&exactSource)
	if err != nil {
		return result, fmt.Errorf("validate operation source: %w", err)
	}
	if !exactSource {
		result.Status, result.Reason = "inconsistent", "Source version does not belong to this operation request."
		return result, nil
	}
	if result.RawGenerationID == uuid.Nil {
		return result, nil
	}
	var rawSource uuid.UUID
	err = s.db.QueryRow(ctx, `SELECT source_version_id FROM context.raw_generation WHERE id=$1::uuid`, result.RawGenerationID).Scan(&rawSource)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status, result.Reason = "inconsistent", "Persisted raw-generation reference has no matching generation."
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("validate raw generation: %w", err)
	}
	if rawSource != result.SourceVersionID {
		result.Status, result.Reason = "inconsistent", "Raw generation belongs to a different source version."
		return result, nil
	}
	if result.NormalizedGenerationID == uuid.Nil {
		return result, nil
	}
	var normalizedSource, normalizedRaw uuid.UUID
	err = s.db.QueryRow(ctx, `SELECT source_version_id, raw_generation_id FROM context.normalized_generation WHERE id=$1::uuid`, result.NormalizedGenerationID).Scan(&normalizedSource, &normalizedRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		result.Status, result.Reason = "inconsistent", "Persisted normalized-generation reference has no matching generation."
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("validate normalized generation: %w", err)
	}
	if normalizedSource != result.SourceVersionID || normalizedRaw != result.RawGenerationID {
		result.Status, result.Reason = "inconsistent", "Normalized and raw generation references do not share this operation's source lineage."
		return result, nil
	}
	result.Status, result.Reason = "available", ""
	return result, nil
}

// operationResultPins reconciles snapshot pins with successful persistence receipts.
// Inputs: one binding and its exact operation stages. Outputs: unique pins or missing/inconsistent diagnostics.
// Effects: none. Choose instead of selecting the newest generation for a source.
func operationResultPins(binding previewmodel.Binding, stages []previewmodel.OperationStage) previewmodel.OperationResults {
	result := previewmodel.OperationResults{Status: "missing", Reason: "No successful normalized-generation persistence receipt is available for this operation.", SourceVersionID: binding.SourceVersionID, RawGenerationID: binding.RawGenerationID, NormalizedGenerationID: binding.NormalizedGenerationID, PreviewAvailable: binding.NormalizedGenerationID != uuid.Nil}
	for _, stage := range stages {
		var pin *uuid.UUID
		var receipt *string
		switch stage.Stage {
		case "persist_raw_generation_activity":
			pin, receipt = &result.RawGenerationID, &result.RawReceiptRef
		case "persist_normalized_generation_activity":
			pin, receipt = &result.NormalizedGenerationID, &result.NormalizedReceiptRef
		default:
			continue
		}
		if stage.Status != "success" && stage.Status != "completed" {
			continue
		}
		id, err := uuid.Parse(stage.Ref)
		if err != nil || id == uuid.Nil || stage.ReceiptRef == "" {
			result.Status, result.Reason = "inconsistent", "Successful persistence receipt has an invalid or missing generation reference: "+stage.ReceiptRef
			return result
		}
		if *pin != uuid.Nil && *pin != id {
			result.Status, result.Reason = "inconsistent", "Persistence receipts and preview disagree about the generation: "+stage.ReceiptRef
			return result
		}
		*pin, *receipt = id, stage.ReceiptRef
	}
	if result.SourceVersionID == uuid.Nil {
		result.Reason = "No exact source-version registration is available for this operation."
	} else if result.RawGenerationID == uuid.Nil {
		result.Reason = "No successful raw-generation persistence receipt is available for this operation."
	}
	return result
}

// SourceFingerprint reads only source-byte fingerprints belonging to the selected immutable version.
// Inputs: operation binding. Outputs: receipt, algorithm, digest, construction, time and actor or an explicit gap.
// Effects: SELECT only. Choose for hash inspection, never ETags or generation/package checksums.
func (s *ProfferPreviewStore) SourceFingerprint(ctx context.Context, binding previewmodel.Binding) (previewmodel.SourceFingerprint, error) {
	result := previewmodel.SourceFingerprint{Status: "unavailable", SourceVersionID: binding.SourceVersionID, Reason: "No successful source-byte fingerprint receipt is available for this exact source version."}
	if binding.SourceVersionID == uuid.Nil {
		return result, nil
	}
	var retainedDigest *string
	err := s.db.QueryRow(ctx, `
 SELECT hash.hash_kind, hash.algorithm, encode(hash.digest,'hex'), hash.construction,
        hash.computed_at, hash.computed_by, hash.id::text, hash.activity_receipt_id::text,
        CASE WHEN retained.id IS NULL THEN NULL ELSE encode(retained.content_sha256,'hex') END
 FROM context.hash_receipt hash
 JOIN context.activity_receipt receipt ON receipt.id=hash.activity_receipt_id AND receipt.status='success'
 JOIN context.activity_execution execution ON execution.id=receipt.activity_execution_id
 JOIN context.source_version source ON source.id=hash.source_version_id
 LEFT JOIN context.retained_object retained ON retained.id=source.original_object_id
 WHERE hash.source_version_id=$1::uuid AND hash.hash_kind='context_source_fingerprint'
   AND execution.source_version_id=source.id AND execution.workflow_id IN ($2,$3) AND source.workflow_id IN ($2,$3)
 ORDER BY hash.computed_at DESC, hash.id DESC LIMIT 1`, binding.SourceVersionID, binding.RequestID, binding.WorkflowID).Scan(
		&result.HashKind, &result.Algorithm, &result.Digest, &result.Construction, &result.ComputedAt, &result.ComputedBy, &result.ReceiptRef, &result.ActivityReceiptRef, &retainedDigest)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("read exact source fingerprint: %w", err)
	}
	if result.Algorithm != "sha256" || !previewmodel.ValidDigest(result.Digest) || result.Construction != "context-source-fingerprint-v1" || (retainedDigest != nil && *retainedDigest != result.Digest) {
		result.Status, result.Reason = "inconsistent", "Source fingerprint receipt does not validate against the retained original identity."
		return result, nil
	}
	result.Status, result.Reason = "available", ""
	return result, nil
}
