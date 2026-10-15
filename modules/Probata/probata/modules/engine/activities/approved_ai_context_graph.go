package activities

import (
	"context"
	"errors"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
)

// ApprovedAIContextGraphActivities projects one already reviewed native AI candidate.
// Inputs: read-only owner ledger reader and existing graph sink. Outputs: Activity host.
// Effects: none until ProjectApprovedAIContextGraph runs. Pick for native AI review decisions.
type ApprovedAIContextGraphActivities struct {
	Reader approvedgraphai.Reader
	Sink   approvedgraph.Sink
}

// ProjectApprovedAIContextGraph verifies the committed decision and writes its cited claim.
// Inputs: exact decision scope. Outputs: existing projection control receipt.
// Effects: idempotent analysis graph write only. Pick after owner approval, never as an intake gate.
func (a ApprovedAIContextGraphActivities) ProjectApprovedAIContextGraph(ctx context.Context, scope approvedgraphai.Scope) (ApprovedProjectionResult, error) {
	if a.Reader == nil || a.Sink == nil {
		return ApprovedProjectionResult{}, errors.New("approved AI graph: reader and sink required")
	}
	claim, err := approvedgraphai.Resolve(ctx, a.Reader, scope)
	if err != nil {
		return ApprovedProjectionResult{}, err
	}
	if err := a.Sink.ApplyApprovedClaims(ctx, []approvedgraph.Claim{claim}); err != nil {
		return ApprovedProjectionResult{}, err
	}
	return ApprovedProjectionResult{Count: 1, ApprovedRevisionID: claim.ReceiptID, ApprovalDigest: claim.ApprovalDigest, ControlGenerationID: claim.ControlGenerationID, ClaimsHash: approvedgraph.ClaimSetDigest([]approvedgraph.Claim{claim})}, nil
}
