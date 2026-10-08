// Byline: Codex · GPT-6 · 2026-10-07
package surrealsink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// ApprovedQueryScope selects an approval revision independently of a historical perspective.
type ApprovedQueryScope struct {
	MatterID               string    `json:"matter_id"`
	CourtCaseID            string    `json:"court_case_id"`
	AccessPolicyID         string    `json:"access_policy_id"`
	ApprovedRevisionID     string    `json:"approved_revision_id"`
	ApprovalDigest         string    `json:"approval_digest"`
	ProjectionGenerationID string    `json:"projection_generation_id"`
	ProjectionHash         string    `json:"projection_hash"`
	Perspective            string    `json:"perspective"`
	Horizon                time.Time `json:"horizon"`
	Limit                  int       `json:"limit"`
	Cursor                 string    `json:"cursor,omitempty"`
}

// ApprovedQueryResult contains only bounded source-cited claims from the selected revision.
type ApprovedQueryResult struct {
	Perspective        string                `json:"perspective"`
	ApprovedRevisionID string                `json:"approved_revision_id"`
	Claims             []approvedgraph.Claim `json:"claims"`
	HasMore            bool                  `json:"has_more"`
	NextCursor         string                `json:"next_cursor,omitempty"`
}

const maxApprovedQueryLimit = 100

// QueryApprovedClaims prefilters case, approval revision, generation and source knowledge before relation traversal.
// Inputs: explicit scope and either an as-lived historical horizon or hindsight perspective.
// Outputs: bounded cited claims. Effects: read-only; pick for approved context retrieval, never raw-turn search.
func (s *ApprovedClaimsSink) QueryApprovedClaims(ctx context.Context, q ApprovedQueryScope) (ApprovedQueryResult, error) {
	if s == nil || s.Client == nil || !caseidentity.AdmittedIdentity(q.MatterID, q.CourtCaseID) || !graphIdentifier(q.AccessPolicyID) || !graphIdentifier(q.ApprovedRevisionID) || !graphDigest(q.ApprovalDigest) || !graphIdentifier(q.ProjectionGenerationID) || !graphDigest(q.ProjectionHash) || q.AccessPolicyID != s.AccessPolicyID || !validApprovedLimit(q.Limit) {
		return ApprovedQueryResult{}, errors.New("approved graph: complete bounded query scope required")
	}
	if q.Perspective != "as_lived" && q.Perspective != "hindsight" {
		return ApprovedQueryResult{}, errors.New("approved graph: explicit perspective required")
	}
	if q.Perspective == "as_lived" && q.Horizon.IsZero() {
		return ApprovedQueryResult{}, errors.New("approved graph: historical horizon required")
	}
	if err := validateAnalysisConfig(s.Client.cfg); err != nil {
		return ApprovedQueryResult{}, err
	}
	if err := s.approvedSchemaReady(ctx); err != nil {
		return ApprovedQueryResult{}, err
	}
	if err := s.projectionReady(ctx, q); err != nil {
		return ApprovedQueryResult{}, err
	}
	var cursor *approvedCursor
	if q.Cursor != "" {
		parsed, err := decodeApprovedCursor(q, q.Cursor)
		if err != nil {
			return ApprovedQueryResult{}, err
		}
		if err := s.verifyApprovedCursor(ctx, q, parsed); err != nil {
			return ApprovedQueryResult{}, err
		}
		cursor = &parsed
	}
	vars := map[string]any{"matter": q.MatterID, "case": q.CourtCaseID, "policy": q.AccessPolicyID,
		"revision": q.ApprovedRevisionID, "digest": q.ApprovalDigest,
		"generation": q.ProjectionGenerationID, "horizon": q.Horizon, "scan_limit": q.Limit + 1}
	var sql string
	for _, table := range []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"} {
		where := approvedWhere(q.Perspective)
		if cursor != nil {
			where += approvedAfterCursor(*cursor, table)
			vars["cursor_id"], vars["cursor_table"], vars["cursor_available"] = cursor.ID, cursor.Table, cursor.Available
		}
		sql += fmt.Sprintf("SELECT node_id, kind, bundle_hash, payload_json, claim_text, matter_id, case_id, approved_revision_id, approval_digest, control_generation_id, candidate_id, candidate_sha256, source_id, source_version_id, source_object_id, source_object_sha256, source_object_uri, record_id, record_sha256, occurred_at, source_available_from, approved_at, approved_by FROM %s WHERE %s ORDER BY source_available_from DESC, node_id ASC LIMIT $scan_limit;\n", table, where)
	}
	results, err := s.Client.graphQuery(ctx, sql, vars)
	if err != nil || len(results) != 3 {
		return ApprovedQueryResult{}, errors.New("approved graph: scoped assertion read failed")
	}
	var claims []approvedgraph.Claim
	for i, raw := range results {
		var rows []struct {
			ID                  string     `json:"node_id"`
			Kind                string     `json:"kind"`
			BundleHash          string     `json:"bundle_hash"`
			PayloadJSON         string     `json:"payload_json"`
			Text                string     `json:"claim_text"`
			MatterID            string     `json:"matter_id"`
			CourtCaseID         string     `json:"case_id"`
			ReceiptID           string     `json:"approved_revision_id"`
			ApprovalDigest      string     `json:"approval_digest"`
			ControlGenerationID string     `json:"control_generation_id"`
			CandidateID         string     `json:"candidate_id"`
			CandidateSHA256     string     `json:"candidate_sha256"`
			SourceID            string     `json:"source_id"`
			SourceVersionID     string     `json:"source_version_id"`
			SourceObjectID      string     `json:"source_object_id"`
			SourceSHA256        string     `json:"source_object_sha256"`
			SourceObjectURI     string     `json:"source_object_uri"`
			RecordID            string     `json:"record_id"`
			RecordSHA256        string     `json:"record_sha256"`
			OccurredAt          *time.Time `json:"occurred_at"`
			SourceAvailableFrom *time.Time `json:"source_available_from"`
			ApprovedAt          time.Time  `json:"approved_at"`
			ApprovedBy          string     `json:"approved_by"`
		}
		if json.Unmarshal(raw, &rows) != nil || len(rows) > q.Limit+1 {
			return ApprovedQueryResult{}, errors.New("approved graph: malformed or unbounded assertion result")
		}
		for _, row := range rows {
			kind := []string{"entity_mention", "statement", "event_account"}[i]
			if row.Kind != []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"}[i] || row.MatterID != q.MatterID || row.CourtCaseID != q.CourtCaseID || row.ReceiptID != q.ApprovedRevisionID || row.ApprovalDigest != q.ApprovalDigest || row.ControlGenerationID != q.ProjectionGenerationID || !eligibleAt(q.Perspective, row.SourceAvailableFrom, q.Horizon) {
				return ApprovedQueryResult{}, errors.New("approved graph: assertion escaped prefilter")
			}
			claim := approvedgraph.Claim{ID: row.ID, Kind: kind, Text: row.Text, MatterID: row.MatterID, CourtCaseID: row.CourtCaseID,
				ReceiptID: row.ReceiptID, ApprovalDigest: row.ApprovalDigest, ControlGenerationID: row.ControlGenerationID, CandidateID: row.CandidateID,
				CandidateSHA256: row.CandidateSHA256, SourceID: row.SourceID, SourceVersionID: row.SourceVersionID,
				SourceObjectID: row.SourceObjectID, SourceSHA256: row.SourceSHA256, SourceObjectURI: row.SourceObjectURI,
				RecordID: row.RecordID, RecordSHA256: row.RecordSHA256, OccurredAt: row.OccurredAt,
				SourceAvailableFrom: row.SourceAvailableFrom, ApprovedAt: row.ApprovedAt, ApprovedBy: row.ApprovedBy}
			if err := restoreApprovedNativeFields(row.PayloadJSON, &claim); err != nil {
				return ApprovedQueryResult{}, err
			}
			if graphHash(claim) != row.BundleHash {
				return ApprovedQueryResult{}, errors.New("approved graph: assertion content differs from immutable hash")
			}
			if err := s.verifyProvenance(ctx, []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"}[i], claim); err != nil {
				return ApprovedQueryResult{}, err
			}
			claims = append(claims, claim)
		}
	}
	sort.Slice(claims, func(i, j int) bool { return approvedClaimLess(claims[i], claims[j]) })
	result := ApprovedQueryResult{Perspective: q.Perspective, ApprovedRevisionID: q.ApprovedRevisionID}
	if len(claims) > q.Limit {
		result.HasMore = true
		result.NextCursor = encodeApprovedCursor(q, claims[q.Limit-1])
		claims = claims[:q.Limit]
	}
	result.Claims = claims
	return result, nil
}

// restoreApprovedNativeFields restores only optional typed AI citation fields from the immutable payload.
// Inputs: stored full claim JSON and column-reconstructed claim. Outputs: error or completed claim.
// Effects: mutates the supplied claim. Pick before bundle-hash verification, including legacy SMS rows.
func restoreApprovedNativeFields(payload string, claim *approvedgraph.Claim) error {
	var native struct {
		Predicate         string `json:"predicate"`
		NativeJSONPointer string `json:"native_json_pointer"`
		NativeSpanStart   *int   `json:"native_span_start"`
		NativeSpanEnd     *int   `json:"native_span_end"`
		NativeSpanUnit    string `json:"native_span_unit"`
		NativeSpanSHA256  string `json:"native_span_sha256"`
	}
	if claim == nil || payload == "" || json.Unmarshal([]byte(payload), &native) != nil {
		return errors.New("approved graph: assertion payload unavailable")
	}
	claim.Predicate, claim.NativeJSONPointer = native.Predicate, native.NativeJSONPointer
	claim.NativeSpanStart, claim.NativeSpanEnd, claim.NativeSpanUnit, claim.NativeSpanSHA256 = native.NativeSpanStart, native.NativeSpanEnd, native.NativeSpanUnit, native.NativeSpanSHA256
	return nil
}

// approvedWhere selects indexed scope and applies source availability only to as-lived reads.
func approvedWhere(perspective string) string {
	where := `matter_id=$matter AND case_id=$case AND access_policy_id=$policy AND approved_revision_id=$revision AND approval_digest=$digest AND generation_id=$generation AND control_generation_id=$generation`
	if perspective == "as_lived" {
		where += ` AND source_available_from != NONE AND source_available_from <= <datetime> $horizon`
	}
	return where
}

// eligibleAt enforces the historical source clock independently of approval time.
func eligibleAt(perspective string, available *time.Time, horizon time.Time) bool {
	if available != nil && available.IsZero() {
		return false
	}
	return perspective == "hindsight" || (perspective == "as_lived" && available != nil && !available.After(horizon))
}

func validApprovedLimit(limit int) bool { return limit >= 1 && limit <= maxApprovedQueryLimit }

// projectionReady rejects partial projections before any assertion traversal or ranking.
func (s *ApprovedClaimsSink) projectionReady(ctx context.Context, q ApprovedQueryScope) error {
	results, err := s.Client.graphQuery(ctx, "SELECT matter_id, case_id, access_policy_id, approved_revision_id, approval_digest, control_generation_id, claims_hash, node_set_hash, claim_count FROM type::record('ana_approved_projection', $key) LIMIT 1;", map[string]any{"key": q.ProjectionGenerationID})
	if err != nil || len(results) != 1 {
		return errors.New("approved graph: selected projection checkpoint unavailable")
	}
	var rows []struct {
		MatterID    string `json:"matter_id"`
		CourtCaseID string `json:"case_id"`
		Policy      string `json:"access_policy_id"`
		Revision    string `json:"approved_revision_id"`
		Digest      string `json:"approval_digest"`
		Generation  string `json:"control_generation_id"`
		Hash        string `json:"claims_hash"`
		NodeSetHash string `json:"node_set_hash"`
		Count       int    `json:"claim_count"`
	}
	if json.Unmarshal(results[0], &rows) != nil || len(rows) != 1 || rows[0].MatterID != q.MatterID || rows[0].CourtCaseID != q.CourtCaseID || rows[0].Policy != q.AccessPolicyID || rows[0].Revision != q.ApprovedRevisionID || rows[0].Digest != q.ApprovalDigest || rows[0].Generation != q.ProjectionGenerationID || rows[0].Hash != q.ProjectionHash || !graphDigest(rows[0].NodeSetHash) || rows[0].Count < 1 || rows[0].Count > 7000 {
		return errors.New("approved graph: projection incomplete or scope differs")
	}
	where := `matter_id=$matter AND case_id=$case AND access_policy_id=$policy AND approved_revision_id=$revision AND approval_digest=$digest AND generation_id=$generation AND control_generation_id=$generation`
	vars := map[string]any{"matter": q.MatterID, "case": q.CourtCaseID, "policy": q.AccessPolicyID, "revision": q.ApprovedRevisionID, "digest": q.ApprovalDigest, "generation": q.ProjectionGenerationID}
	var countSQL, keysSQL string
	tables := []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"}
	for _, table := range tables {
		countSQL += fmt.Sprintf("SELECT count() AS n FROM %s WHERE %s GROUP ALL;\n", table, where)
		keysSQL += fmt.Sprintf("SELECT node_id, bundle_hash FROM %s WHERE %s LIMIT 7001;\n", table, where)
	}
	counts, err := s.Client.graphQuery(ctx, countSQL, vars)
	if err != nil || len(counts) != len(tables) {
		return errors.New("approved graph: scoped node count unavailable")
	}
	total := 0
	for _, raw := range counts {
		var entries []struct {
			N int `json:"n"`
		}
		if json.Unmarshal(raw, &entries) != nil || len(entries) > 1 {
			return errors.New("approved graph: scoped node count malformed")
		}
		if len(entries) == 1 {
			total += entries[0].N
		}
	}
	if total != rows[0].Count {
		return errors.New("approved graph: scoped node count differs from checkpoint")
	}
	sets, err := s.Client.graphQuery(ctx, keysSQL, vars)
	if err != nil || len(sets) != len(tables) {
		return errors.New("approved graph: scoped node digest unavailable")
	}
	keys := make([]approvedNodeKey, 0, total)
	for i, raw := range sets {
		var entries []struct {
			ID   string `json:"node_id"`
			Hash string `json:"bundle_hash"`
		}
		if json.Unmarshal(raw, &entries) != nil || len(entries) > total {
			return errors.New("approved graph: scoped node digest malformed")
		}
		for _, entry := range entries {
			keys = append(keys, approvedNodeKey{Table: tables[i], ID: entry.ID, Hash: entry.Hash})
		}
	}
	if len(keys) != total || hashApprovedNodeKeys(keys) != rows[0].NodeSetHash {
		return errors.New("approved graph: scoped node set differs from checkpoint")
	}
	return nil
}

func (s *ApprovedClaimsSink) verifyProvenance(ctx context.Context, table string, claim approvedgraph.Claim) error {
	edgeKey := flow.DeterministicID("approved_graph_provenance", claim.ID)
	sourceKey := flow.DeterministicID("approved_graph_source", claim.MatterID, claim.CourtCaseID, claim.SourceVersionID, claim.SourceSHA256)
	recordKey := flow.DeterministicID("approved_graph_record", claim.SourceVersionID, claim.RecordID, claim.RecordSHA256)
	recordEdgeKey := flow.DeterministicID("approved_graph_record_source", recordKey, sourceKey)
	sql := fmt.Sprintf("SELECT in = type::record('%s', $claim_key) AND out = type::record('ctx_approved_record', $record_key) AND matter_id=$matter AND case_id=$case AND approved_revision_id=$revision AND approval_digest=$digest AND record_sha256=$record_hash AS linked FROM type::record('ctx_approved_provenance', $edge_key) LIMIT 1; SELECT in = type::record('ctx_approved_record', $record_key) AND out = type::record('ctx_approved_source_version', $source_key) AND record_sha256=$record_hash AS linked FROM type::record('ctx_approved_record_source', $record_edge_key) LIMIT 1;", table)
	results, err := s.Client.graphQuery(ctx, sql, map[string]any{"claim_key": claim.ID, "source_key": sourceKey, "edge_key": edgeKey,
		"record_key": recordKey, "record_edge_key": recordEdgeKey,
		"matter": claim.MatterID, "case": claim.CourtCaseID, "revision": claim.ReceiptID,
		"digest": claim.ApprovalDigest, "record_hash": claim.RecordSHA256})
	if err != nil || len(results) != 2 {
		return errors.New("approved graph: source relation unavailable")
	}
	var rows []struct {
		Linked bool `json:"linked"`
	}
	var sourceRows []struct {
		Linked bool `json:"linked"`
	}
	if json.Unmarshal(results[0], &rows) != nil || json.Unmarshal(results[1], &sourceRows) != nil || len(rows) != 1 || len(sourceRows) != 1 || !rows[0].Linked || !sourceRows[0].Linked {
		return errors.New("approved graph: source relation mismatch")
	}
	return nil
}
