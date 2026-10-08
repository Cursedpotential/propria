// Byline: Codex · GPT-6 · 2026-10-07
package activities

import (
	"context"
	"errors"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
)

// ApprovedGraphResolver resolves one committed candidate revision from durable stores.
// Inputs: an exact scope. Outputs: verified revision. Effects: read-only.
// Pick it for graph projection after the owner commit receipt has completed.
type ApprovedGraphResolver interface {
	ResolveApprovedGraph(context.Context, approvedgraph.Scope) (approvedgraph.Revision, error)
}

// ApprovedContextGraphActivities hosts the separately schedulable approved projection.
type ApprovedContextGraphActivities struct {
	Resolver ApprovedGraphResolver
	Sink     approvedgraph.Sink
}

// ApprovedProjectionResult is the completed control snapshot; it contains no source body.
type ApprovedProjectionResult struct {
	Count               int    `json:"count"`
	ApprovedRevisionID  string `json:"approved_revision_id"`
	ApprovalDigest      string `json:"approval_digest"`
	ControlGenerationID string `json:"control_generation_id"`
	ClaimsHash          string `json:"claims_hash"`
}

// ProjectApprovedContextGraph sends only cited assertions from one exact owner commit.
// Inputs: receipt, canonical case, and source pins. Outputs: projected claim count.
// Effects: idempotent graph writes through Sink. Pick after a completed extraction commit.
func (a ApprovedContextGraphActivities) ProjectApprovedContextGraph(ctx context.Context, scope approvedgraph.Scope) (ApprovedProjectionResult, error) {
	if a.Resolver == nil || a.Sink == nil {
		return ApprovedProjectionResult{}, errors.New("approved graph: resolver and sink required")
	}
	revision, err := a.Resolver.ResolveApprovedGraph(ctx, scope)
	if err != nil {
		return ApprovedProjectionResult{}, err
	}
	claims, err := approvedgraph.BuildClaims(revision)
	if err != nil {
		return ApprovedProjectionResult{}, err
	}
	if len(claims) == 0 {
		return ApprovedProjectionResult{}, errors.New("approved graph: no cited claims")
	}
	if err := a.Sink.ApplyApprovedClaims(ctx, claims); err != nil {
		return ApprovedProjectionResult{}, err
	}
	return ApprovedProjectionResult{Count: len(claims), ApprovedRevisionID: revision.Receipt.ID,
		ApprovalDigest: revision.Receipt.Digest, ControlGenerationID: claims[0].ControlGenerationID,
		ClaimsHash: approvedgraph.ClaimSetDigest(claims)}, nil
}
