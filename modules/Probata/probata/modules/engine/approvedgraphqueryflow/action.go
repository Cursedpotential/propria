// Package approvedgraphqueryflow runs one revision-scoped approved graph read through Temporal.
// Byline: Codex · GPT-6 · 2026-10-08.
package approvedgraphqueryflow

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

const (
	WorkflowName            = "approved_context_graph_query_workflow"
	ActivityName            = "read_approved_context_graph_activity"
	StatusQuery             = "status"
	WorkflowIDPrefix        = "approved-context-query:"
	ApprovedQueryBucket     = "salem-data"
	ApprovedQueryPrefixRoot = "consignatio/casevault/DerivedKnowledge/analysis/queries/"
	MaxResultBytes          = 8 << 20
	MaxClaims               = 100
)

// Request binds one actor to an exact approved projection and historical perspective.
type Request struct {
	OperatingMode string                         `json:"operating_mode"`
	Actor         entities.Actor                 `json:"actor"`
	RequestID     string                         `json:"request_id"`
	Scope         surrealsink.ApprovedQueryScope `json:"scope"`
}

// Validate rejects unbound or unbounded approved graph reads.
// Inputs: one actor-bound query request. Outputs: validation error. Effects: none.
// Pick at the starter, workflow, and Activity boundaries before any graph or object-store I/O.
func (r Request) Validate() error {
	q := r.Scope
	if r.OperatingMode != string(caseidentity.ModeLive) || !caseidentity.AdmittedIdentity(q.MatterID, q.CourtCaseID) ||
		strings.TrimSpace(r.Actor.SubjectUID) == "" || !safeRequestID(r.RequestID) ||
		q.AccessPolicyID == "" || q.ApprovedRevisionID == "" || q.ProjectionGenerationID == "" ||
		!digest64(q.ApprovalDigest) || !digest64(q.ProjectionHash) || q.Limit < 1 || q.Limit > MaxClaims ||
		(q.Perspective != "as_lived" && q.Perspective != "hindsight") ||
		(q.Perspective == "as_lived" && q.Horizon.IsZero()) || len(q.Cursor) > 2048 {
		return errors.New("approved graph query requires exact actor, revision, projection, perspective and bound")
	}
	for _, value := range []string{q.AccessPolicyID, q.ApprovedRevisionID, q.ProjectionGenerationID} {
		if strings.TrimSpace(value) != value || len(value) > 240 || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("approved graph query has an invalid control identifier")
		}
	}
	return nil
}

func digest64(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}

func safeRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// Reader performs the existing complete-projection, scope, source and provenance checks.
type Reader interface {
	QueryApprovedClaims(context.Context, surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error)
}

// Result contains only an exact-version content reference and approved control pins.
type Result struct {
	Artifact               libraryvalidation.ArtifactRef `json:"artifact"`
	ClaimCount             int                           `json:"claim_count"`
	MatterID               string                        `json:"matter_id"`
	CourtCaseID            string                        `json:"court_case_id"`
	AccessPolicyID         string                        `json:"access_policy_id"`
	ApprovedRevisionID     string                        `json:"approved_revision_id"`
	ApprovalDigest         string                        `json:"approval_digest"`
	ProjectionGenerationID string                        `json:"projection_generation_id"`
	ProjectionHash         string                        `json:"projection_hash"`
	Perspective            string                        `json:"perspective"`
	HasMore                bool                          `json:"has_more"`
	NextCursor             string                        `json:"next_cursor,omitempty"`
}

type resultArtifact struct {
	ActorSubjectUID        string                `json:"actor_subject_uid"`
	MatterID               string                `json:"matter_id"`
	CourtCaseID            string                `json:"court_case_id"`
	AccessPolicyID         string                `json:"access_policy_id"`
	ApprovedRevisionID     string                `json:"approved_revision_id"`
	ApprovalDigest         string                `json:"approval_digest"`
	ProjectionGenerationID string                `json:"projection_generation_id"`
	ProjectionHash         string                `json:"projection_hash"`
	Perspective            string                `json:"perspective"`
	Horizon                *time.Time            `json:"horizon"`
	Limit                  int                   `json:"limit"`
	RequestCursor          string                `json:"request_cursor,omitempty"`
	HasMore                bool                  `json:"has_more"`
	NextCursor             string                `json:"next_cursor,omitempty"`
	Claims                 []approvedgraph.Claim `json:"claims"`
}

// Activities reads a verified projection and writes a cited result to versioned CaseVault storage.
type Activities struct {
	Reader    Reader
	Artifacts libraryvalidation.B2Artifacts
}

// Run stores one bounded cited query result and returns only its verified exact-version reference.
// Inputs: actor-bound revision, generation, perspective and limit. Outputs: ref, pins and count.
// Effects: read-only graph query and versioned derivative write/readback under the approved-query prefix.
// Pick for approved context retrieval, never for raw conversations or an unapproved candidate.
func (a Activities) Run(ctx context.Context, in Request) (Result, error) {
	if err := in.Validate(); err != nil {
		return Result{}, err
	}
	if a.Reader == nil || a.Artifacts.Validate() != nil || a.Artifacts.Bucket != ApprovedQueryBucket || !strings.HasPrefix(a.Artifacts.Prefix, ApprovedQueryPrefixRoot) || len(a.Artifacts.Prefix) <= len(ApprovedQueryPrefixRoot) {
		return Result{}, errors.New("approved graph query requires a separate CaseVault query-derivative prefix and reader")
	}
	got, err := a.Reader.QueryApprovedClaims(ctx, in.Scope)
	if err != nil {
		return Result{}, fmt.Errorf("approved graph query: %w", err)
	}
	if got.Perspective != in.Scope.Perspective || got.ApprovedRevisionID != in.Scope.ApprovedRevisionID || len(got.Claims) > in.Scope.Limit || len(got.Claims) > MaxClaims || got.HasMore != (got.NextCursor != "") || len(got.NextCursor) > 2048 || (got.HasMore && len(got.Claims) == 0) {
		return Result{}, errors.New("approved graph query returned an unbounded or different projection")
	}
	for _, claim := range got.Claims {
		if claim.MatterID != in.Scope.MatterID || claim.CourtCaseID != in.Scope.CourtCaseID || claim.ReceiptID != in.Scope.ApprovedRevisionID || claim.ApprovalDigest != in.Scope.ApprovalDigest || claim.ControlGenerationID != in.Scope.ProjectionGenerationID ||
			claim.ID == "" || claim.RecordID == "" || !digest64(claim.RecordSHA256) || !digest64(claim.CandidateSHA256) || !digest64(claim.SourceSHA256) || claim.SourceVersionID == "" || claim.SourceObjectID == "" || claim.SourceObjectURI == "" ||
			(in.Scope.Perspective == "as_lived" && (claim.SourceAvailableFrom == nil || claim.SourceAvailableFrom.After(in.Scope.Horizon))) {
			return Result{}, errors.New("approved graph query returned an uncited or out-of-scope claim")
		}
	}
	var horizon *time.Time
	if in.Scope.Perspective == "as_lived" {
		horizon = &in.Scope.Horizon
	}
	body, err := json.Marshal(resultArtifact{ActorSubjectUID: in.Actor.SubjectUID, MatterID: in.Scope.MatterID, CourtCaseID: in.Scope.CourtCaseID,
		AccessPolicyID:     in.Scope.AccessPolicyID,
		ApprovedRevisionID: in.Scope.ApprovedRevisionID, ApprovalDigest: in.Scope.ApprovalDigest,
		ProjectionGenerationID: in.Scope.ProjectionGenerationID, ProjectionHash: in.Scope.ProjectionHash,
		Perspective: in.Scope.Perspective, Horizon: horizon, Limit: in.Scope.Limit, RequestCursor: in.Scope.Cursor,
		HasMore: got.HasMore, NextCursor: got.NextCursor, Claims: got.Claims})
	if err != nil || len(body) == 0 || len(body) > MaxResultBytes {
		return Result{}, errors.New("approved graph query JSON exceeds the eight-megabyte result bound")
	}
	ref, err := a.Artifacts.Put(ctx, "approved-context/"+in.RequestID+".json", body, "application/json")
	if err != nil {
		return Result{}, fmt.Errorf("approved graph query content write/readback: %w", err)
	}
	if ref.SHA256 != libraryvalidation.Hash(body) || ref.Bytes != int64(len(body)) || ref.URI == "" || ref.VersionID == "" {
		return Result{}, errors.New("approved graph query result reference differs from exact bytes")
	}
	return Result{Artifact: ref, ClaimCount: len(got.Claims), MatterID: in.Scope.MatterID,
		CourtCaseID: in.Scope.CourtCaseID, AccessPolicyID: in.Scope.AccessPolicyID, ApprovedRevisionID: in.Scope.ApprovedRevisionID,
		ApprovalDigest: in.Scope.ApprovalDigest, ProjectionGenerationID: in.Scope.ProjectionGenerationID,
		ProjectionHash: in.Scope.ProjectionHash, Perspective: in.Scope.Perspective,
		HasMore: got.HasMore, NextCursor: got.NextCursor}, nil
}
