// Byline: Codex · GPT-6 · 2026-10-08.
package surrealsink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
)

type approvedCursor struct {
	Version   int        `json:"version"`
	ScopeHash string     `json:"scope_hash"`
	Table     string     `json:"table"`
	ID        string     `json:"id"`
	Hash      string     `json:"hash"`
	Available *time.Time `json:"available"`
	Checksum  string     `json:"checksum"`
}

func approvedScopeHash(q ApprovedQueryScope) string {
	return graphHash(struct {
		Matter, Case, Policy, Revision, Digest, Generation, ProjectionHash, Perspective string
		Horizon                                                                         time.Time
		Limit                                                                           int
	}{q.MatterID, q.CourtCaseID, q.AccessPolicyID, q.ApprovedRevisionID, q.ApprovalDigest,
		q.ProjectionGenerationID, q.ProjectionHash, q.Perspective, q.Horizon.UTC(), q.Limit})
}

func approvedClaimLess(a, b approvedgraph.Claim) bool {
	availableA, availableB := a.SourceAvailableFrom, b.SourceAvailableFrom
	if availableA == nil || availableB == nil {
		if availableA != nil || availableB != nil {
			return availableA != nil
		}
	} else if !availableA.Equal(*availableB) {
		return availableA.After(*availableB)
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	tableA, _ := approvedClaimTable(a.Kind)
	tableB, _ := approvedClaimTable(b.Kind)
	return tableA < tableB
}

func encodeApprovedCursor(q ApprovedQueryScope, last approvedgraph.Claim) string {
	table, _ := approvedClaimTable(last.Kind)
	cursor := approvedCursor{Version: 1, ScopeHash: approvedScopeHash(q), Table: table,
		ID: last.ID, Hash: graphHash(last), Available: last.SourceAvailableFrom}
	cursor.Checksum = graphHash(cursor)
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeApprovedCursor(q ApprovedQueryScope, encoded string) (approvedCursor, error) {
	if len(encoded) > 2048 {
		return approvedCursor{}, errors.New("approved graph: continuation cursor exceeds bound")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return approvedCursor{}, errors.New("approved graph: malformed continuation cursor")
	}
	var cursor approvedCursor
	if json.Unmarshal(raw, &cursor) != nil {
		return approvedCursor{}, errors.New("approved graph: malformed continuation cursor")
	}
	checksum := cursor.Checksum
	cursor.Checksum = ""
	if cursor.Version != 1 || cursor.ScopeHash != approvedScopeHash(q) || !approvedCursorTable(cursor.Table) || !graphIdentifier(cursor.ID) || !graphDigest(cursor.Hash) ||
		(cursor.Available != nil && cursor.Available.IsZero()) || (q.Perspective == "as_lived" && cursor.Available == nil) || !graphDigest(checksum) || graphHash(cursor) != checksum {
		return approvedCursor{}, errors.New("approved graph: continuation cursor or scope changed")
	}
	cursor.Checksum = checksum
	return cursor, nil
}

func approvedCursorTable(table string) bool {
	return table == "ctx_entity_mention" || table == "ctx_statement" || table == "ctx_event_account"
}

func approvedAfterCursor(cursor approvedCursor, table string) string {
	if cursor.Available == nil {
		return fmt.Sprintf(" AND source_available_from = NONE AND (node_id > $cursor_id OR (node_id = $cursor_id AND '%s' > $cursor_table))", table)
	}
	return fmt.Sprintf(" AND (source_available_from = NONE OR source_available_from < <datetime> $cursor_available OR (source_available_from = <datetime> $cursor_available AND (node_id > $cursor_id OR (node_id = $cursor_id AND '%s' > $cursor_table))))", table)
}

// verifyApprovedCursor binds a continuation to one persisted scoped node before paging.
// Inputs: a decoded cursor and selected approval scope. Outputs: error on any missing or changed node.
// Effects: read-only graph lookup; pick before keyset traversal, never as an authorization check.
func (s *ApprovedClaimsSink) verifyApprovedCursor(ctx context.Context, q ApprovedQueryScope, cursor approvedCursor) error {
	where := approvedWhere(q.Perspective)
	sql := fmt.Sprintf("SELECT node_id, bundle_hash, source_available_from FROM %s WHERE %s AND node_id=$cursor_id LIMIT 1;", cursor.Table, where)
	results, err := s.Client.graphQuery(ctx, sql, map[string]any{"matter": q.MatterID, "case": q.CourtCaseID,
		"policy": q.AccessPolicyID, "revision": q.ApprovedRevisionID, "digest": q.ApprovalDigest,
		"generation": q.ProjectionGenerationID, "horizon": q.Horizon, "cursor_id": cursor.ID})
	if err != nil || len(results) != 1 {
		return errors.New("approved graph: continuation node unavailable")
	}
	var rows []struct {
		ID        string     `json:"node_id"`
		Hash      string     `json:"bundle_hash"`
		Available *time.Time `json:"source_available_from"`
	}
	if json.Unmarshal(results[0], &rows) != nil || len(rows) != 1 || rows[0].ID != cursor.ID || rows[0].Hash != cursor.Hash || !sameApprovedAvailability(rows[0].Available, cursor.Available) {
		return errors.New("approved graph: continuation node changed")
	}
	return nil
}

func sameApprovedAvailability(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
