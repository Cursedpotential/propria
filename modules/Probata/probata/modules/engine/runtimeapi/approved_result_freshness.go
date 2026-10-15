package runtimeapi

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"

	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// ApprovedProjectionFreshness checks live owner-decision metadata before exposing a stored result reference.
// Inputs: workflow-selected projection scope. Outputs: current approval or error. Effects: read-only.
// Choose for existing actor-bound query/walk status; historical artifacts remain preserved.
type ApprovedProjectionFreshness interface {
	CheckCurrentApprovedProjection(context.Context, surrealsink.ApprovedQueryScope) error
}

// approvedResultFreshness withholds references when the selected approval cannot be checked as current.
// Inputs: configured PG checker and original workflow scope. Outputs: current, stale, unavailable or unknown.
// Effects: existing metadata reads; choose before status JSON exposes protected result/checkpoint references.
func approvedResultFreshness(ctx context.Context, checker ApprovedProjectionFreshness, scope *surrealsink.ApprovedQueryScope) string {
	if checker == nil || scope == nil {
		return "unknown"
	}
	if err := checker.CheckCurrentApprovedProjection(ctx, *scope); err != nil {
		if errors.Is(err, approvedgraphai.ErrSupersededDecision) {
			return "stale"
		}
		return "unavailable"
	}
	return "current"
}
