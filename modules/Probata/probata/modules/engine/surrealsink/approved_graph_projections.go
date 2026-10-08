// Byline: Codex · GPT-6 · 2026-10-08.
package surrealsink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

const (
	defaultApprovedProjectionLimit = 10
	maxApprovedProjectionLimit     = 50
	maxApprovedProjectionPins      = 100
)

// ErrApprovedProjectionInput marks a rejected case, limit or cursor without hiding backend failures.
var ErrApprovedProjectionInput = errors.New("approved graph: invalid projection list input")

// ApprovedProjectionScope selects completed checkpoints in one canonical case and fixed service policy.
type ApprovedProjectionScope struct {
	MatterID    string
	CourtCaseID string
	Limit       int
	Cursor      string
}

// ApprovedSourcePin is one retained original reference without claim or source bodies.
type ApprovedSourcePin struct {
	SourceID           string `json:"source_id"`
	SourceVersionID    string `json:"source_version_id"`
	SourceObjectID     string `json:"source_object_id"`
	SourceObjectSHA256 string `json:"source_object_sha256"`
	SourceObjectURI    string `json:"source_object_uri"`
}

// ApprovedProjectionDescriptor identifies one completed analytical checkpoint and its source roots.
type ApprovedProjectionDescriptor struct {
	MatterID               string              `json:"matter_id"`
	CourtCaseID            string              `json:"court_case_id"`
	AccessPolicyID         string              `json:"access_policy_id"`
	ApprovedRevisionID     string              `json:"approved_revision_id"`
	ApprovalDigest         string              `json:"approval_digest"`
	ProjectionGenerationID string              `json:"projection_generation_id"`
	ProjectionHash         string              `json:"projection_hash"`
	ClaimCount             int                 `json:"claim_count"`
	CompletedAt            time.Time           `json:"completed_at"`
	SourcePins             []ApprovedSourcePin `json:"source_pins"`
}

// ApprovedProjectionPage contains a bounded descriptor page and continuation key.
type ApprovedProjectionPage struct {
	Items      []ApprovedProjectionDescriptor `json:"items"`
	HasMore    bool                           `json:"has_more"`
	NextCursor string                         `json:"next_cursor,omitempty"`
}

type approvedProjectionCursor struct {
	MatterID    string `json:"matter_id"`
	CourtCaseID string `json:"court_case_id"`
	PolicyID    string `json:"policy_id"`
	Generation  string `json:"generation"`
}

// encodeProjectionCursor binds a keyset position to the server selected case and policy.
// Inputs: case, policy and generation. Outputs: bounded opaque cursor. Effects: none.
// Pick after a complete page; the cursor never grants access to a projection.
func encodeProjectionCursor(scope ApprovedProjectionScope, policy, generation string) string {
	raw, _ := json.Marshal(approvedProjectionCursor{MatterID: scope.MatterID, CourtCaseID: scope.CourtCaseID, PolicyID: policy, Generation: generation})
	return base64.RawURLEncoding.EncodeToString(raw)
}

// decodeProjectionCursor rejects malformed or cross-scope continuation keys.
// Inputs: opaque cursor and server case/policy. Outputs: generation or error. Effects: none.
// Pick before ledger I/O; authentication remains at the HTTP service boundary.
func decodeProjectionCursor(scope ApprovedProjectionScope, policy string) (string, error) {
	if scope.Cursor == "" {
		return "", nil
	}
	if len(scope.Cursor) > 1024 {
		return "", fmt.Errorf("%w: cursor exceeds bound", ErrApprovedProjectionInput)
	}
	raw, err := base64.RawURLEncoding.DecodeString(scope.Cursor)
	if err != nil {
		return "", fmt.Errorf("%w: cursor is malformed", ErrApprovedProjectionInput)
	}
	var cursor approvedProjectionCursor
	if json.Unmarshal(raw, &cursor) != nil || cursor.MatterID != scope.MatterID || cursor.CourtCaseID != scope.CourtCaseID || cursor.PolicyID != policy || !graphIdentifier(cursor.Generation) || encodeProjectionCursor(scope, policy, cursor.Generation) != scope.Cursor {
		return "", fmt.Errorf("%w: cursor scope differs", ErrApprovedProjectionInput)
	}
	return cursor.Generation, nil
}

// ListApprovedProjections lists completed checkpoints and bounded original-source pins.
// Inputs: canonical case, limit and cursor; sink policy is server configured. Outputs: descriptor page.
// Effects: read-only fct/analysis metadata queries; choose before a user selects an exact query scope.
func (s *ApprovedClaimsSink) ListApprovedProjections(ctx context.Context, scope ApprovedProjectionScope) (ApprovedProjectionPage, error) {
	if !caseidentity.AdmittedIdentity(scope.MatterID, scope.CourtCaseID) {
		return ApprovedProjectionPage{}, fmt.Errorf("%w: canonical case required", ErrApprovedProjectionInput)
	}
	if s == nil || s.Client == nil || !graphIdentifier(s.AccessPolicyID) {
		return ApprovedProjectionPage{}, errors.New("approved graph: configured analytical client and access policy required")
	}
	if scope.Limit == 0 {
		scope.Limit = defaultApprovedProjectionLimit
	}
	if scope.Limit < 1 || scope.Limit > maxApprovedProjectionLimit {
		return ApprovedProjectionPage{}, fmt.Errorf("%w: page limit must be 1 through 50", ErrApprovedProjectionInput)
	}
	after, err := decodeProjectionCursor(scope, s.AccessPolicyID)
	if err != nil {
		return ApprovedProjectionPage{}, err
	}
	if err := validateAnalysisConfig(s.Client.cfg); err != nil {
		return ApprovedProjectionPage{}, err
	}
	if err := s.approvedSchemaReady(ctx); err != nil {
		return ApprovedProjectionPage{}, err
	}
	const sql = `SELECT matter_id, case_id, access_policy_id, approved_revision_id, approval_digest,
 control_generation_id, claims_hash, node_set_hash, claim_count, completed_at
 FROM ana_approved_projection
 WHERE matter_id=$matter AND case_id=$case AND access_policy_id=$policy
 AND completed_at != NONE AND control_generation_id > $after
 ORDER BY control_generation_id ASC LIMIT $scan_limit;`
	results, err := s.Client.graphQuery(ctx, sql, map[string]any{"matter": scope.MatterID, "case": scope.CourtCaseID,
		"policy": s.AccessPolicyID, "after": after, "scan_limit": scope.Limit + 1})
	if err != nil || len(results) != 1 {
		return ApprovedProjectionPage{}, errors.New("approved graph: completed projection ledger unavailable")
	}
	var rows []struct {
		MatterID    string    `json:"matter_id"`
		CaseID      string    `json:"case_id"`
		Policy      string    `json:"access_policy_id"`
		Revision    string    `json:"approved_revision_id"`
		Digest      string    `json:"approval_digest"`
		Generation  string    `json:"control_generation_id"`
		Hash        string    `json:"claims_hash"`
		NodeSetHash string    `json:"node_set_hash"`
		Count       int       `json:"claim_count"`
		CompletedAt time.Time `json:"completed_at"`
	}
	if json.Unmarshal(results[0], &rows) != nil || len(rows) > scope.Limit+1 {
		return ApprovedProjectionPage{}, errors.New("approved graph: malformed or unbounded projection ledger")
	}
	page := ApprovedProjectionPage{Items: make([]ApprovedProjectionDescriptor, 0, min(len(rows), scope.Limit))}
	page.HasMore = len(rows) > scope.Limit
	if page.HasMore {
		rows = rows[:scope.Limit]
	}
	previous := after
	for _, row := range rows {
		if row.MatterID != scope.MatterID || row.CaseID != scope.CourtCaseID || row.Policy != s.AccessPolicyID || !graphIdentifier(row.Revision) || !graphDigest(row.Digest) || !graphIdentifier(row.Generation) || !graphDigest(row.Hash) || !graphDigest(row.NodeSetHash) || row.Count < 1 || row.Count > 7000 || row.CompletedAt.IsZero() || row.Generation <= previous {
			return ApprovedProjectionPage{}, errors.New("approved graph: projection checkpoint is incomplete or escaped scope")
		}
		pins, err := s.approvedProjectionPins(ctx, row.MatterID, row.CaseID, row.Policy, row.Revision, row.Digest, row.Generation)
		if err != nil {
			return ApprovedProjectionPage{}, err
		}
		page.Items = append(page.Items, ApprovedProjectionDescriptor{MatterID: row.MatterID, CourtCaseID: row.CaseID,
			AccessPolicyID: row.Policy, ApprovedRevisionID: row.Revision, ApprovalDigest: row.Digest,
			ProjectionGenerationID: row.Generation, ProjectionHash: row.Hash, ClaimCount: row.Count,
			CompletedAt: row.CompletedAt, SourcePins: pins})
		previous = row.Generation
	}
	if page.HasMore {
		page.NextCursor = encodeProjectionCursor(scope, s.AccessPolicyID, previous)
	}
	return page, nil
}

// approvedProjectionPins reads distinct retained-source roots from the three scoped assertion tables.
// Inputs: completed checkpoint pins. Outputs: at most 100 sorted source references. Effects: read-only graph I/O.
// Pick for descriptor display; it never selects claim text, records or source bodies.
func (s *ApprovedClaimsSink) approvedProjectionPins(ctx context.Context, matter, courtCase, policy, revision, digest, generation string) ([]ApprovedSourcePin, error) {
	const where = `matter_id=$matter AND case_id=$case AND access_policy_id=$policy AND approved_revision_id=$revision AND approval_digest=$digest AND generation_id=$generation AND control_generation_id=$generation`
	var sql strings.Builder
	for _, table := range []string{"ctx_entity_mention", "ctx_statement", "ctx_event_account"} {
		fmt.Fprintf(&sql, `SELECT source_id, source_version_id, source_object_id, source_object_sha256, source_object_uri FROM %s WHERE %s GROUP BY source_id, source_version_id, source_object_id, source_object_sha256, source_object_uri LIMIT 101;`+"\n", table, where)
	}
	results, err := s.Client.graphQuery(ctx, sql.String(), map[string]any{"matter": matter, "case": courtCase,
		"policy": policy, "revision": revision, "digest": digest, "generation": generation})
	if err != nil || len(results) != 3 {
		return nil, errors.New("approved graph: scoped projection source pins unavailable")
	}
	unique := make(map[string]ApprovedSourcePin)
	for _, raw := range results {
		var pins []ApprovedSourcePin
		if json.Unmarshal(raw, &pins) != nil || len(pins) > maxApprovedProjectionPins {
			return nil, errors.New("approved graph: projection source pin count exceeds bound")
		}
		for _, pin := range pins {
			if !graphIdentifier(pin.SourceID) || !graphIdentifier(pin.SourceVersionID) || !graphIdentifier(pin.SourceObjectID) || !graphDigest(pin.SourceObjectSHA256) || !graphExternalRef(pin.SourceObjectURI) {
				return nil, errors.New("approved graph: projection source pin is incomplete")
			}
			if prior, ok := unique[pin.SourceVersionID]; ok && prior != pin {
				return nil, errors.New("approved graph: conflicting original source pin for one version")
			}
			unique[pin.SourceVersionID] = pin
			if len(unique) > maxApprovedProjectionPins {
				return nil, errors.New("approved graph: projection source pin count exceeds bound")
			}
		}
	}
	if len(unique) == 0 {
		return nil, errors.New("approved graph: completed projection has no source pins")
	}
	pins := make([]ApprovedSourcePin, 0, len(unique))
	for _, pin := range unique {
		pins = append(pins, pin)
	}
	sort.Slice(pins, func(i, j int) bool { return pins[i].SourceVersionID < pins[j].SourceVersionID })
	return pins, nil
}
