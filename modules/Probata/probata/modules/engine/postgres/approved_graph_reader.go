// Byline: Codex · GPT-6 · 2026-10-07
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// ApprovedGraphReader reads one completed commit and its exact promoted candidates.
// Inputs: the existing engine database. Outputs: persisted receipt and proposals.
// Effects: read-only. Pick it for approvedgraph.Resolve, never for unreviewed previews.
type ApprovedGraphReader struct{ db DB }

// NewApprovedGraphReader binds the read-only resolver to the platform database.
// Inputs: existing DB. Outputs: a reader. Effects: none. Pick it for graph activities.
func NewApprovedGraphReader(db DB) (*ApprovedGraphReader, error) {
	if db == nil {
		return nil, errors.New("approved graph: database required")
	}
	return &ApprovedGraphReader{db: db}, nil
}

var _ approvedgraph.Reader = (*ApprovedGraphReader)(nil)

// ReadReceipt reads the completed extraction receipt without accepting caller flags.
// Inputs: receipt UUID. Outputs: persisted receipt. Effects: read-only. Pick before candidate reads.
func (r *ApprovedGraphReader) ReadReceipt(ctx context.Context, id string) (approvedgraph.Receipt, error) {
	var out approvedgraph.Receipt
	var actor []byte
	err := r.db.QueryRow(ctx, `SELECT id::text, status, coalesce(stats->>'outcome', ''), source_summary,
 coalesce(stats->>'digest', ''), coalesce(stats->>'matter_mode', ''), coalesce(stats->'actor', '{}'::jsonb), finished_at
FROM working.extraction_run WHERE id=$1::uuid AND extractor=$2`, id, service.CommitExtractor).
		Scan(&out.ID, &out.Status, &out.Outcome, &out.Summary, &out.Digest, &out.MatterMode, &actor, &out.FinishedAt)
	if err != nil {
		return approvedgraph.Receipt{}, fmt.Errorf("read approved graph receipt: %w", err)
	}
	if err := json.Unmarshal(actor, &out.Actor); err != nil {
		return approvedgraph.Receipt{}, fmt.Errorf("decode approved graph actor: %w", err)
	}
	return out, nil
}

// ReadSourcePin reads the retained original mapping and its recorded root hash.
// Inputs: exact case and source revision. Outputs: source URI and SHA-256.
// Effects: read-only. Pick before projection so every assertion carries a root citation.
func (r *ApprovedGraphReader) ReadSourcePin(ctx context.Context, scope approvedgraph.Scope) (approvedgraph.SourcePin, error) {
	var pin approvedgraph.SourcePin
	err := r.db.QueryRow(ctx, `SELECT v.source_id::text, o.id::text, o.object_uri, encode(o.content_sha256, 'hex')
FROM context.proffer_preview_binding b
JOIN context.proffer_preview_snapshot sn ON sn.preview_handle=b.preview_handle
JOIN context.source_version v ON v.id=sn.source_version_id
JOIN context.source_version_object svo ON svo.source_version_id=v.id AND svo.object_role='original' AND svo.object_id=v.original_object_id
JOIN context.retained_object o ON o.id=svo.object_id
WHERE b.preview_handle=$1 AND sn.normalized_generation_id=$2::uuid AND sn.source_version_id=$3::uuid
 AND v.matter_id=$4::uuid AND v.court_case_id=$5::uuid AND o.storage_class='immutable_object_store'
LIMIT 1`, scope.PreviewHandle, scope.GenerationID, scope.SourceVersionID, scope.MatterID, scope.CourtCaseID).
		Scan(&pin.SourceID, &pin.ObjectID, &pin.ObjectURI, &pin.SHA256)
	if err != nil {
		return approvedgraph.SourcePin{}, fmt.Errorf("read approved graph source pin: %w", err)
	}
	return pin, nil
}

// ReadRecordPin reads only canonical metadata for one normalized record of the source revision.
// Inputs: exact generation, source version, and record ID. Outputs: hash and source clocks.
// Effects: read-only; it never selects normalized message bodies. Pick for SMS citations only.
func (r *ApprovedGraphReader) ReadRecordPin(ctx context.Context, scope approvedgraph.Scope, recordID string) (approvedgraph.RecordPin, error) {
	var pin approvedgraph.RecordPin
	err := r.db.QueryRow(ctx, approvedRecordPinSQL,
		recordID, scope.GenerationID, scope.SourceVersionID).
		Scan(&pin.ID, &pin.SourceVersionID, &pin.SHA256, &pin.OccurredAt, &pin.SourceAvailableFrom)
	if err != nil {
		return approvedgraph.RecordPin{}, fmt.Errorf("read approved graph canonical record %s: %w", recordID, err)
	}
	return pin, nil
}

// A context record receives a source clock only through its exact approved
// working spine route. The canonical function applies first-party event time
// or approved third-party custody acquisition, and returns NULL when unknown.
const approvedRecordPinSQL = `SELECT c.id::text, c.source_version_id::text,
 encode(sha256(c.canonical_bytes), 'hex'), c.occurred_at,
 CASE WHEN route.normalized_record_id IS NOT NULL THEN working.source_available_from(w.id) END
FROM context.normalized_record_identity c
LEFT JOIN working.normalized_record w ON w.id=c.id
 AND w.source_version_id=c.source_version_id
 AND w.derived_from_raw_table='context.normalized_record_identity'
 AND w.derived_from_raw_id=c.id AND w.record_type='message'
LEFT JOIN working.message_projection_route route ON route.normalized_record_id=w.id
 AND route.decision_state='approved'
 AND route.projection_kind IN ('first_party', 'acquired_third_party')
WHERE c.id=$1::uuid AND c.normalized_generation_id=$2::uuid AND c.source_version_id=$3::uuid`

// ReadPromotedEntities reads all entity rows promoted at the receipt transaction timestamp.
// Inputs: exact timestamp. Outputs: bounded entity rows. Effects: read-only. Pick after receipt read.
func (r *ApprovedGraphReader) ReadPromotedEntities(ctx context.Context, at time.Time) ([]approvedgraph.Entity, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, extraction_run_id::text, name, attrs, review_state,
 coalesce(promoted_to_id, ''), encode(content_sha256, 'hex'), promoted_at
FROM working.candidate_entity WHERE promoted_at=$1 AND review_state='approved'
ORDER BY id LIMIT 2001`, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []approvedgraph.Entity
	for rows.Next() {
		var row approvedgraph.Entity
		var raw []byte
		var id, runID, name, state, promoted string
		if err := rows.Scan(&id, &runID, &name, &raw, &state, &promoted, &row.ContentHex, &row.PromotedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &row.Proposal); err != nil {
			return nil, fmt.Errorf("candidate entity %s attrs: %w", id, err)
		}
		row.Proposal.CandidateID, row.Proposal.ExtractionRunID, row.Proposal.Name = id, runID, name
		row.Proposal.ReviewState, row.Proposal.PromotedToID = state, promoted
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 2000 {
		return nil, errors.New("approved graph: entity commit exceeds bound")
	}
	return out, nil
}

// ReadPromotedEvents reads all event rows promoted at the receipt transaction timestamp.
// Inputs: exact timestamp. Outputs: bounded event rows. Effects: read-only. Pick after receipt read.
func (r *ApprovedGraphReader) ReadPromotedEvents(ctx context.Context, at time.Time) ([]approvedgraph.Event, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, extraction_run_id::text, summary, attrs, review_state,
 coalesce(promoted_to_id, ''), encode(content_sha256, 'hex'), promoted_at
FROM working.candidate_event WHERE promoted_at=$1 AND review_state='approved'
ORDER BY id LIMIT 5001`, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []approvedgraph.Event
	for rows.Next() {
		var row approvedgraph.Event
		var raw []byte
		var id, runID, title, state, promoted string
		if err := rows.Scan(&id, &runID, &title, &raw, &state, &promoted, &row.ContentHex, &row.PromotedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &row.Proposal); err != nil {
			return nil, fmt.Errorf("candidate event %s attrs: %w", id, err)
		}
		row.Proposal.CandidateID, row.Proposal.ExtractionRunID, row.Proposal.Title = id, runID, title
		row.Proposal.ReviewState, row.Proposal.PromotedToID = state, promoted
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) > 5000 {
		return nil, errors.New("approved graph: event commit exceeds bound")
	}
	return out, nil
}

// ResolveApprovedGraph verifies one commit against its current canonical source pin.
// Inputs: exact receipt and preview coordinates. Outputs: verified revision.
// Effects: read-only. Pick it before any approved assertion materialization.
func (s *EntityExtractionStore) ResolveApprovedGraph(ctx context.Context, scope approvedgraph.Scope) (approvedgraph.Revision, error) {
	if scope.MatterID != authoritativeMatterID || scope.CourtCaseID != authoritativeCourtCaseID || scope.MatterMode != "LIVE" {
		return approvedgraph.Revision{}, errors.New("approved graph: canonical LIVE case required")
	}
	var pinned bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM context.proffer_preview_binding b
 JOIN context.proffer_preview_snapshot sn ON sn.preview_handle=b.preview_handle
 JOIN context.source_version v ON v.id=sn.source_version_id
 WHERE b.preview_handle=$1 AND sn.normalized_generation_id=$2::uuid AND sn.source_version_id=$3::uuid
   AND v.matter_id=$4::uuid AND v.court_case_id=$5::uuid
)`, scope.PreviewHandle, scope.GenerationID, scope.SourceVersionID, scope.MatterID, scope.CourtCaseID).Scan(&pinned)
	if err != nil {
		return approvedgraph.Revision{}, err
	}
	if !pinned {
		return approvedgraph.Revision{}, errors.New("approved graph: exact source binding absent from canonical case")
	}
	reader, err := NewApprovedGraphReader(s.db)
	if err != nil {
		return approvedgraph.Revision{}, err
	}
	return approvedgraph.Resolve(ctx, reader, scope)
}
