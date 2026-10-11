// Byline: Codex · GPT-6.1-Sol · 2026-10-06.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/normalize"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"io"
	"strings"
)

// LoadPersistedNormalizeExecution verifies historical success before recomputing a bundle.
// Inputs are the actual source/raw/request refs. Outputs are the original bundle
// and receipt or absence. Effects are read-only SQL and streaming integrity
// verification via the existing reader factory. Pick before normalization,
// mirroring repair's prior-result read while preserving the historical header.
func (r *NormalizedPipelineRepository) LoadPersistedNormalizeExecution(ctx context.Context, req proffer.StageRequest) (proffer.Ref, proffer.Ref, bool, error) {
	sourceID, err := uuid.Parse(string(req.SourceVersionRef))
	if err != nil {
		return "", "", false, err
	}
	rawID, err := uuid.Parse(string(req.Refs["raw_generation"]))
	if err != nil {
		return "", "", false, err
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return "", "", false, errors.New("normalize replay requires request ownership")
	}
	key := normalizeExecutionKey(activities.NormalizeExecutionSpec{RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef, RawGenerationRef: req.Refs["raw_generation"]})
	var receiptID uuid.UUID
	var resultJSON []byte
	var bound bool
	err = r.db.QueryRow(ctx, `SELECT receipt.id,receipt.result_ref,
        EXISTS (SELECT 1 FROM context.source_version_object link
            JOIN context.retained_object object ON object.id=link.object_id
            WHERE link.source_version_id=source.id AND object.id::text=receipt.result_ref->>'ref_id'
            AND link.object_role='derived_reference' AND link.parent_object_id=raw.extraction_bundle_object_id
            AND link.member_locator->>'kind'='normalized_bundle')
        FROM context.activity_execution execution
        JOIN context.activity_receipt receipt ON receipt.activity_execution_id=execution.id
        JOIN context.source_version source ON source.id=execution.source_version_id
        JOIN context.raw_generation raw ON raw.id=$2 AND raw.source_version_id=source.id
        WHERE source.id=$1 AND source.workflow_id=$3 AND source.status='retained' AND raw.status='sealed'
        AND execution.workflow_id=$3 AND execution.activity_name=$4 AND execution.idempotency_key=$5
        AND receipt.status='success' ORDER BY receipt.attempt DESC LIMIT 1`, sourceID, rawID, req.RequestID, string(stagegraph.NormalizeGeneration), key).Scan(&receiptID, &resultJSON, &bound)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("read historical normalize receipt: %w", err)
	}
	kind, id, err := decodeNormalizedRef(resultJSON)
	if err != nil {
		return "", "", false, err
	}
	if kind != "normalized_bundle" || !bound {
		return "", "", false, errors.New("historical normalize receipt is not bound to the retained source/raw bundle")
	}
	reader, err := r.readerFactory(ctx, proffer.Ref(id))
	if err != nil {
		return "", "", false, fmt.Errorf("verify historical normalized bundle: %w", err)
	}
	if reader == nil {
		return "", "", false, errors.New("historical normalized bundle reader is nil")
	}
	defer reader.Close()
	header := reader.Header()
	if header.ContractVersion != normalize.ContractVersion || header.SourceVersionRef != string(req.SourceVersionRef) || header.RawGenerationRef != string(req.Refs["raw_generation"]) || strings.TrimSpace(header.NormalizerID) == "" || strings.TrimSpace(header.NormalizerVersion) == "" {
		return "", "", false, errors.New("historical normalized bundle header conflicts with source/raw/version identity")
	}
	for {
		record, err := reader.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", "", false, fmt.Errorf("verify historical normalized bundle stream: %w", err)
		}
		if err := record.Validate(); err != nil {
			return "", "", false, fmt.Errorf("verify historical normalized record: %w", err)
		}
	}
	return proffer.Ref(id), proffer.Ref(receiptID.String()), true, nil
}
