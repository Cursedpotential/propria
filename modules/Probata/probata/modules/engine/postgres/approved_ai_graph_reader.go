package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// ApprovedAIGraphReader fetches one native AI decision and retained source without normalized turns.
// Inputs: existing platform database. Outputs: an approvedgraphai.Reader.
// Effects: read-only when called. Pick for native AI review decisions, not SMS commit receipts.
type ApprovedAIGraphReader struct{ db DB }

// NewApprovedAIGraphReader binds the native AI decision reader to the existing database.
// Inputs: platform DB. Outputs: reader or configuration error. Effects: none.
// Pick when registering optional native AI graph projection.
func NewApprovedAIGraphReader(db DB) (*ApprovedAIGraphReader, error) {
	if db == nil {
		return nil, errors.New("approved AI graph: database required")
	}
	return &ApprovedAIGraphReader{db: db}, nil
}

var _ approvedgraphai.Reader = (*ApprovedAIGraphReader)(nil)

// ReadApprovedAI reads the exact candidate, owner ledger and registered original metadata.
// Inputs: canonical case and actual review IDs. Outputs: at most one snapshot.
// Effects: read-only SQL. Pick after approval; the resolver checks every returned field.
func (r *ApprovedAIGraphReader) ReadApprovedAI(ctx context.Context, s approvedgraphai.Scope) (approvedgraphai.Snapshot, error) {
	if err := approvedgraphai.ValidateScope(s); err != nil {
		return approvedgraphai.Snapshot{}, err
	}
	rows, err := r.db.Query(ctx, `WITH candidates AS (
 SELECT 'working.candidate_entity' AS table_name,id,source_raw_table,source_raw_id,attrs,content_sha256,review_state FROM working.candidate_entity WHERE id=$1::uuid AND source_raw_id=$2
 UNION ALL SELECT 'working.candidate_event',id,source_raw_table,source_raw_id,attrs,content_sha256,review_state FROM working.candidate_event WHERE id=$1::uuid AND source_raw_id=$2
 UNION ALL SELECT 'working.candidate_fact',id,source_raw_table,source_raw_id,attrs,content_sha256,review_state FROM working.candidate_fact WHERE id=$1::uuid AND source_raw_id=$2
)
SELECT c.table_name,c.review_state,c.source_raw_table,c.source_raw_id,c.attrs,encode(c.content_sha256,'hex'),
 d.decision,d.target_kind,d.target_id::text,d.reviewer,d.rationale,d.decided_at,
 v.source_id::text,coalesce(o.id::text,''),coalesce(o.object_uri,source.source_key),coalesce(encode(o.content_sha256,'hex'),'')
FROM candidates c
JOIN analysis.review_decision d ON d.decision_id=$3::uuid AND d.target_id=c.id
JOIN context.source_version v ON v.id=$2::uuid AND v.id::text=c.source_raw_id
JOIN context.source source ON source.id=v.source_id
LEFT JOIN context.retained_object o ON o.id=v.original_object_id
WHERE v.matter_id=$4::uuid AND v.court_case_id=$5::uuid AND (v.original_object_id IS NULL OR v.status='retained')
ORDER BY c.table_name LIMIT 2`, s.CandidateID, s.SourceVersionID, s.DecisionID, s.MatterID, s.CourtCaseID)
	if err != nil {
		return approvedgraphai.Snapshot{}, fmt.Errorf("approved AI graph: read review: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return approvedgraphai.Snapshot{}, err
		}
		return approvedgraphai.Snapshot{}, errors.New("approved AI graph: exact decision not found")
	}
	var out approvedgraphai.Snapshot
	if err := rows.Scan(&out.Table, &out.ReviewState, &out.SourceRawTable, &out.SourceRawID, &out.Attrs, &out.CandidateSHA256, &out.LedgerDecision, &out.LedgerTargetKind, &out.LedgerTargetID, &out.LedgerReviewer, &out.LedgerRationale, &out.DecidedAt, &out.SourceID, &out.SourceObjectID, &out.SourceObjectURI, &out.SourceSHA256); err != nil {
		return approvedgraphai.Snapshot{}, err
	}
	if rows.Next() {
		return approvedgraphai.Snapshot{}, errors.New("approved AI graph: candidate ID resolves to multiple review tables")
	}
	if err := rows.Err(); err != nil {
		return approvedgraphai.Snapshot{}, err
	}
	if out.SourceObjectID == "" {
		var attrs struct {
			Candidate service.AICandidate `json:"candidate"`
		}
		if err := json.Unmarshal(out.Attrs, &attrs); err != nil {
			return approvedgraphai.Snapshot{}, fmt.Errorf("approved AI graph: native candidate attrs: %w", err)
		}
		pin := attrs.Candidate.AISourcePin
		if pin.SourceVersionID != s.SourceVersionID || pin.SourceRef != out.SourceObjectURI || pin.SourceObjectID != "" {
			return approvedgraphai.Snapshot{}, errors.New("approved AI graph: native source pin differs")
		}
		if _, err := verifyAIContextSource(ctx, r.db, pin); err != nil {
			return approvedgraphai.Snapshot{}, err
		}
		out.SourceSHA256 = pin.SourceSHA256
	}
	return out, nil
}
