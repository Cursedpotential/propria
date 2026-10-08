// Byline: Codex · GPT-6 · 2026-10-07
package surrealsink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// ApprovedClaimsSink projects exact cited assertions through the existing fct/analysis client.
type ApprovedClaimsSink struct {
	Client           *Client
	AccessPolicyID   string
	CreatedByService string
}

var _ approvedgraph.Sink = (*ApprovedClaimsSink)(nil)

type approvedNodeKey struct {
	Table string `json:"table"`
	ID    string `json:"id"`
	Hash  string `json:"hash"`
}

func approvedNodeSetHash(claims []approvedgraph.Claim) string {
	keys := make([]approvedNodeKey, 0, len(claims))
	for _, claim := range claims {
		table, _ := approvedClaimTable(claim.Kind)
		keys = append(keys, approvedNodeKey{Table: table, ID: claim.ID, Hash: graphHash(claim)})
	}
	return hashApprovedNodeKeys(keys)
}

func hashApprovedNodeKeys(keys []approvedNodeKey) string {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Table == keys[j].Table {
			return keys[i].ID < keys[j].ID
		}
		return keys[i].Table < keys[j].Table
	})
	return graphHash(keys)
}

// ApplyApprovedClaims writes bounded, immutable typed nodes and source-version relations.
// Inputs: claims from approvedgraph.BuildClaims. Outputs: error after readback.
// Effects: idempotent fct/analysis writes; pick only after exact owner receipt resolution.
func (s *ApprovedClaimsSink) ApplyApprovedClaims(ctx context.Context, claims []approvedgraph.Claim) error {
	if s == nil || s.Client == nil || !graphIdentifier(s.AccessPolicyID) || !graphIdentifier(s.CreatedByService) || len(claims) == 0 || len(claims) > 7000 {
		return errors.New("approved graph: explicit sink and bounded nonempty claims required")
	}
	if err := validateAnalysisConfig(s.Client.cfg); err != nil {
		return err
	}
	if err := s.approvedSchemaReady(ctx); err != nil {
		return err
	}
	ordered := append([]approvedgraph.Claim(nil), claims...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	first := ordered[0]
	for i, claim := range ordered {
		if (i > 0 && claim.ID == ordered[i-1].ID) || claim.MatterID != first.MatterID || claim.CourtCaseID != first.CourtCaseID || claim.ReceiptID != first.ReceiptID || claim.ApprovalDigest != first.ApprovalDigest || claim.ControlGenerationID != first.ControlGenerationID {
			return errors.New("approved graph: projection must contain one complete unique revision")
		}
	}
	for _, claim := range ordered {
		if err := s.applyClaim(ctx, claim); err != nil {
			return err
		}
	}
	return s.completeProjection(ctx, first, len(ordered), approvedgraph.ClaimSetDigest(ordered), approvedNodeSetHash(ordered))
}

func approvedClaimTable(kind string) (string, error) {
	switch kind {
	case "entity_mention":
		return "ctx_entity_mention", nil
	case "statement":
		return "ctx_statement", nil
	case "event_account":
		return "ctx_event_account", nil
	default:
		return "", errors.New("approved graph: unsupported assertion kind")
	}
}

func (s *ApprovedClaimsSink) approvedSchemaReady(ctx context.Context) error {
	results, err := s.Client.graphQuery(ctx, "INFO FOR DB;", nil)
	if err != nil || len(results) != 1 {
		return errors.New("approved graph: analysis schema unavailable")
	}
	var db struct {
		Tables map[string]string `json:"tables"`
	}
	if json.Unmarshal(results[0], &db) != nil {
		return errors.New("approved graph: analysis schema metadata invalid")
	}
	for _, table := range []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account", "ctx_approved_source_version", "ctx_approved_record", "ctx_approved_provenance", "ctx_approved_record_source", "ana_approved_projection"} {
		definition := db.Tables[table]
		if definition == "" || !strings.Contains(definition, "SCHEMAFULL") {
			return errors.New("approved graph: additive typed schema is not admitted")
		}
		if (table == "ctx_approved_provenance" || table == "ctx_approved_record_source") && !strings.Contains(definition, "TYPE RELATION") {
			return errors.New("approved graph: provenance relation schema is not admitted")
		}
	}
	tables := []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account", "ctx_approved_source_version", "ctx_approved_record", "ctx_approved_provenance", "ctx_approved_record_source", "ana_approved_projection"}
	var sql strings.Builder
	for _, table := range tables {
		fmt.Fprintf(&sql, "INFO FOR TABLE %s;\n", table)
	}
	results, err = s.Client.graphQuery(ctx, sql.String(), nil)
	if err != nil || len(results) != len(tables) {
		return errors.New("approved graph: typed field metadata unavailable")
	}
	for i, raw := range results {
		var info struct {
			Fields map[string]string `json:"fields"`
		}
		if json.Unmarshal(raw, &info) != nil {
			return errors.New("approved graph: typed field metadata invalid")
		}
		fields := map[string]string{}
		switch tables[i] {
		case "ctx_entity_mention", "ctx_statement", "ctx_event_account":
			for name, typ := range graphFields("ctx_statement") {
				fields[name] = typ
			}
			for name, typ := range approvedClaimFields {
				fields[name] = typ
			}
		case "ctx_approved_source_version":
			for _, name := range []string{"matter_id", "case_id", "source_id", "source_version_id", "source_object_id", "source_object_sha256", "source_object_uri"} {
				fields[name] = "string"
			}
		case "ctx_approved_record":
			for _, name := range []string{"matter_id", "case_id", "source_version_id", "record_id", "record_sha256"} {
				fields[name] = "string"
			}
			fields["source_available_from"], fields["occurred_at"] = "option<datetime>", "option<datetime>"
		case "ctx_approved_provenance":
			fields["in"], fields["out"] = "record", "record"
			for _, name := range []string{"matter_id", "case_id", "approved_revision_id", "approval_digest", "candidate_id", "record_id", "record_sha256", "source_version_id"} {
				fields[name] = "string"
			}
		case "ctx_approved_record_source":
			fields["in"], fields["out"] = "record", "record"
			for _, name := range []string{"matter_id", "case_id", "source_version_id", "record_id", "record_sha256"} {
				fields[name] = "string"
			}
		case "ana_approved_projection":
			for _, name := range []string{"matter_id", "case_id", "access_policy_id", "approved_revision_id", "approval_digest", "control_generation_id", "claims_hash", "node_set_hash"} {
				fields[name] = "string"
			}
			fields["claim_count"], fields["completed_at"] = "int", "datetime"
		}
		for name, typ := range fields {
			if !strings.Contains(info.Fields[name], "TYPE "+typ) {
				return errors.New("approved graph: typed field absent or incompatible")
			}
		}
	}
	return nil
}

func (s *ApprovedClaimsSink) applyClaim(ctx context.Context, claim approvedgraph.Claim) error {
	table, err := approvedClaimTable(claim.Kind)
	if err != nil {
		return err
	}
	if !caseidentity.AdmittedIdentity(claim.MatterID, claim.CourtCaseID) || !graphIdentifier(claim.ID) || !graphIdentifier(claim.MatterID) || !graphIdentifier(claim.CourtCaseID) || !graphIdentifier(claim.ReceiptID) || !graphIdentifier(claim.ControlGenerationID) || !graphIdentifier(claim.CandidateID) || !graphIdentifier(claim.RecordID) || !graphIdentifier(claim.SourceID) || !graphIdentifier(claim.SourceVersionID) || !graphIdentifier(claim.SourceObjectID) || !graphDigest(claim.ApprovalDigest) || !graphDigest(claim.CandidateSHA256) || !graphDigest(claim.RecordSHA256) || !graphDigest(claim.SourceSHA256) || !graphExternalRef(claim.SourceObjectURI) || claim.Text == "" || (claim.SourceAvailableFrom != nil && claim.SourceAvailableFrom.IsZero()) || claim.ApprovedAt.IsZero() || claim.ApprovedBy == "" {
		return errors.New("approved graph: incomplete approved assertion")
	}
	pin := ContextSourcePin{SourceID: claim.SourceID, SourceVersionID: claim.SourceVersionID, SourceHash: claim.SourceSHA256,
		Locator: claim.SourceObjectURI + "#record=" + claim.RecordID, ValidationRef: claim.ReceiptID}
	if err := validateGraphPins([]ContextSourcePin{pin}); err != nil {
		return err
	}
	raw, err := json.Marshal(claim)
	if err != nil {
		return err
	}
	if len(raw) > 65536 {
		return errors.New("approved graph: assertion exceeds payload bound")
	}
	row := graphRow{MatterID: claim.MatterID, CaseID: claim.CourtCaseID, AccessPolicyID: s.AccessPolicyID,
		CreatedByService: s.CreatedByService, GenerationID: claim.ControlGenerationID, GenerationKey: "ag_" + claim.ControlGenerationID,
		BundleHash: graphHash(claim), ExtractionRunRef: claim.ReceiptID, Active: true,
		NodeID: claim.ID, Kind: table, DerivativeKind: "approved_extraction", SourceOrigin: "owner_commit",
		SourceAvailableFrom: claim.SourceAvailableFrom, OccurredAt: claim.OccurredAt,
		TimePrecision: "unknown", RelativeAnchorRef: "", SourcePins: []ContextSourcePin{pin}, PayloadJSON: string(raw)}
	vars := graphRowMap(row, false, false)
	for key, value := range map[string]any{
		"approved_revision_id": claim.ReceiptID, "approval_digest": claim.ApprovalDigest, "control_generation_id": claim.ControlGenerationID,
		"approved_at": claim.ApprovedAt, "approved_by": claim.ApprovedBy,
		"candidate_id": claim.CandidateID, "candidate_sha256": claim.CandidateSHA256,
		"record_id": claim.RecordID, "record_sha256": claim.RecordSHA256,
		"source_id": claim.SourceID, "source_version_id": claim.SourceVersionID,
		"source_object_id": claim.SourceObjectID, "source_object_sha256": claim.SourceSHA256,
		"source_object_uri": claim.SourceObjectURI, "claim_text": claim.Text,
	} {
		vars[key] = value
	}
	sourceKey := flow.DeterministicID("approved_graph_source", claim.MatterID, claim.CourtCaseID, claim.SourceVersionID, claim.SourceSHA256)
	recordKey := flow.DeterministicID("approved_graph_record", claim.SourceVersionID, claim.RecordID, claim.RecordSHA256)
	edgeKey := flow.DeterministicID("approved_graph_provenance", claim.ID)
	recordEdgeKey := flow.DeterministicID("approved_graph_record_source", recordKey, sourceKey)
	bound := map[string]any{
		"claim": vars, "claim_key": claim.ID, "source_key": sourceKey, "record_key": recordKey,
		"edge_key": edgeKey, "record_edge_key": recordEdgeKey,
		"source": map[string]any{"matter_id": claim.MatterID, "case_id": claim.CourtCaseID, "source_id": claim.SourceID,
			"source_version_id": claim.SourceVersionID, "source_object_id": claim.SourceObjectID,
			"source_object_sha256": claim.SourceSHA256, "source_object_uri": claim.SourceObjectURI},
		"record": map[string]any{"matter_id": claim.MatterID, "case_id": claim.CourtCaseID,
			"source_version_id": claim.SourceVersionID, "record_id": claim.RecordID,
			"record_sha256": claim.RecordSHA256, "source_available_from": claim.SourceAvailableFrom,
			"occurred_at": claim.OccurredAt},
		"record_edge": map[string]any{"matter_id": claim.MatterID, "case_id": claim.CourtCaseID,
			"source_version_id": claim.SourceVersionID, "record_id": claim.RecordID,
			"record_sha256": claim.RecordSHA256},
		"edge": map[string]any{"matter_id": claim.MatterID, "case_id": claim.CourtCaseID,
			"approved_revision_id": claim.ReceiptID, "approval_digest": claim.ApprovalDigest, "candidate_id": claim.CandidateID,
			"record_id": claim.RecordID, "record_sha256": claim.RecordSHA256, "source_version_id": claim.SourceVersionID},
	}
	sql := fmt.Sprintf(`BEGIN TRANSACTION;
LET $source_old = (SELECT * FROM type::record('ctx_approved_source_version', $source_key))[0];
IF $source_old != NONE AND ($source_old.source_object_sha256 != $source.source_object_sha256 OR $source_old.source_object_uri != $source.source_object_uri) { THROW 'source pin changed'; };
IF $source_old = NONE { CREATE type::record('ctx_approved_source_version', $source_key) CONTENT $source; };
LET $record_old = (SELECT * FROM type::record('ctx_approved_record', $record_key))[0];
IF $record_old != NONE AND ($record_old.record_sha256 != $record.record_sha256 OR $record_old.source_version_id != $record.source_version_id) { THROW 'record pin changed'; };
IF $record_old = NONE { CREATE type::record('ctx_approved_record', $record_key) CONTENT object::extend($record, {source_available_from: <option<datetime>> $record.source_available_from, occurred_at: <option<datetime>> $record.occurred_at}); };
LET $old_record_edge = (SELECT * FROM type::record('ctx_approved_record_source', $record_edge_key))[0];
LET $record_ref = type::record('ctx_approved_record', $record_key);
LET $source_ref = type::record('ctx_approved_source_version', $source_key);
LET $record_edge_ref = type::record('ctx_approved_record_source', $record_edge_key);
IF $old_record_edge = NONE { RELATE $record_ref->$record_edge_ref->$source_ref CONTENT $record_edge; };
LET $old = (SELECT * FROM type::record('%s', $claim_key))[0];
IF $old != NONE AND ($old.bundle_hash != $claim.bundle_hash OR $old.approved_revision_id != $claim.approved_revision_id) { THROW 'approved assertion changed'; };
IF $old = NONE {
 CREATE type::record('%s', $claim_key) CONTENT object::extend($claim, {source_available_from: <option<datetime>> $claim.source_available_from, occurred_at: <option<datetime>> $claim.occurred_at, approved_at: <datetime> $claim.approved_at});
};
LET $old_edge = (SELECT * FROM type::record('ctx_approved_provenance', $edge_key))[0];
LET $claim_ref = type::record('%s', $claim_key);
LET $claim_edge_ref = type::record('ctx_approved_provenance', $edge_key);
IF $old_edge = NONE {
 RELATE $claim_ref->$claim_edge_ref->$record_ref CONTENT $edge;
};
COMMIT TRANSACTION;`, table, table, table)
	if n, err := contextGraphRPCBytes(sql, bound); err != nil || n > maxBody {
		return errors.New("approved graph: assertion RPC exceeds bound")
	}
	if _, err := s.Client.graphQuery(ctx, sql, bound); err != nil {
		return err
	}
	results, err := s.Client.graphQuery(ctx, fmt.Sprintf("SELECT bundle_hash, approved_revision_id, record_sha256 FROM type::record('%s', $claim_key) LIMIT 1; SELECT in = type::record('%s', $claim_key) AND out = type::record('ctx_approved_record', $record_key) AS linked FROM type::record('ctx_approved_provenance', $edge_key) LIMIT 1; SELECT in = type::record('ctx_approved_record', $record_key) AND out = type::record('ctx_approved_source_version', $source_key) AS linked FROM type::record('ctx_approved_record_source', $record_edge_key) LIMIT 1;", table, table), map[string]any{"claim_key": claim.ID, "edge_key": edgeKey, "source_key": sourceKey, "record_key": recordKey, "record_edge_key": recordEdgeKey})
	if err != nil || len(results) != 3 {
		return errors.New("approved graph: assertion readback failed")
	}
	var nodes []struct {
		Hash       string `json:"bundle_hash"`
		Revision   string `json:"approved_revision_id"`
		RecordHash string `json:"record_sha256"`
	}
	var edges []struct {
		Linked bool `json:"linked"`
	}
	var recordEdges []struct {
		Linked bool `json:"linked"`
	}
	if json.Unmarshal(results[0], &nodes) != nil || json.Unmarshal(results[1], &edges) != nil || json.Unmarshal(results[2], &recordEdges) != nil || len(nodes) != 1 || len(edges) != 1 || len(recordEdges) != 1 || nodes[0].Hash != row.BundleHash || nodes[0].Revision != claim.ReceiptID || nodes[0].RecordHash != claim.RecordSHA256 || !edges[0].Linked || !recordEdges[0].Linked {
		return errors.New("approved graph: assertion or provenance readback differs")
	}
	return nil
}

// completeProjection publishes a checkpoint only after every typed assertion and relation read back.
func (s *ApprovedClaimsSink) completeProjection(ctx context.Context, first approvedgraph.Claim, count int, hash, nodeSetHash string) error {
	key := first.ControlGenerationID
	vars := map[string]any{"key": key, "row": map[string]any{
		"matter_id": first.MatterID, "case_id": first.CourtCaseID,
		"access_policy_id":     s.AccessPolicyID,
		"approved_revision_id": first.ReceiptID, "approval_digest": first.ApprovalDigest,
		"control_generation_id": key, "claims_hash": hash, "node_set_hash": nodeSetHash, "claim_count": count,
		"completed_at": time.Now().UTC(),
	}}
	sql := `BEGIN TRANSACTION;
LET $old = (SELECT * FROM type::record('ana_approved_projection', $key))[0];
IF $old != NONE AND ($old.claims_hash != $row.claims_hash OR $old.node_set_hash != $row.node_set_hash OR $old.claim_count != $row.claim_count OR $old.approval_digest != $row.approval_digest OR $old.matter_id != $row.matter_id OR $old.case_id != $row.case_id OR $old.access_policy_id != $row.access_policy_id OR $old.approved_revision_id != $row.approved_revision_id OR $old.control_generation_id != $row.control_generation_id) { THROW 'approved projection checkpoint changed'; };
IF $old = NONE { CREATE type::record('ana_approved_projection', $key) CONTENT object::extend($row, {completed_at: <datetime> $row.completed_at}); };
COMMIT TRANSACTION;`
	if _, err := s.Client.graphQuery(ctx, sql, vars); err != nil {
		return err
	}
	results, err := s.Client.graphQuery(ctx, "SELECT claims_hash, node_set_hash, claim_count, approval_digest, matter_id, case_id, access_policy_id, approved_revision_id, control_generation_id FROM type::record('ana_approved_projection', $key) LIMIT 1;", map[string]any{"key": key})
	if err != nil || len(results) != 1 {
		return errors.New("approved graph: projection checkpoint readback failed")
	}
	var rows []struct {
		Hash        string `json:"claims_hash"`
		NodeSetHash string `json:"node_set_hash"`
		Count       int    `json:"claim_count"`
		Digest      string `json:"approval_digest"`
		MatterID    string `json:"matter_id"`
		CourtCaseID string `json:"case_id"`
		Policy      string `json:"access_policy_id"`
		Revision    string `json:"approved_revision_id"`
		Generation  string `json:"control_generation_id"`
	}
	if json.Unmarshal(results[0], &rows) != nil || len(rows) != 1 || rows[0].Hash != hash || rows[0].NodeSetHash != nodeSetHash || rows[0].Count != count || rows[0].Digest != first.ApprovalDigest || rows[0].MatterID != first.MatterID || rows[0].CourtCaseID != first.CourtCaseID || rows[0].Policy != s.AccessPolicyID || rows[0].Revision != first.ReceiptID || rows[0].Generation != key {
		return errors.New("approved graph: projection checkpoint differs")
	}
	return nil
}
