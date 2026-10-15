package runtimeapi

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/surrealsink"
	"testing"
)

type freshnessFixture struct{ err error }

// CheckCurrentApprovedProjection supplies one live-check outcome without reading any artifact.
// Inputs: workflow-selected scope. Outputs: fixture error or current. Effects: none; choose for display freshness tests.
func (f freshnessFixture) CheckCurrentApprovedProjection(context.Context, surrealsink.ApprovedQueryScope) error {
	return f.err
}

// TestApprovedResultFreshnessNeverInfersCurrent checks explicit current, stale, unavailable and legacy unknown states.
// Inputs: configured/current or missing checker/scope. Outputs: exact display metadata. Effects: none; choose for result suppression.
func TestApprovedResultFreshnessNeverInfersCurrent(t *testing.T) {
	q := &surrealsink.ApprovedQueryScope{}
	for _, tc := range []struct {
		checker ApprovedProjectionFreshness
		scope   *surrealsink.ApprovedQueryScope
		want    string
	}{{freshnessFixture{}, q, "current"}, {freshnessFixture{approvedgraphai.ErrSupersededDecision}, q, "stale"}, {freshnessFixture{errors.New("DB offline")}, q, "unavailable"}, {nil, q, "unknown"}, {freshnessFixture{}, nil, "unknown"}} {
		if got := approvedResultFreshness(context.Background(), tc.checker, tc.scope); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
}
