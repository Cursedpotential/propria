// Byline: Codex · GPT-6 · 2026-10-08.
package profferworker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphprojectionflow"
	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

const approvedGraphEnabledEnv = "APPROVED_CONTEXT_GRAPH_ENABLED"

// ApprovedGraphGroup binds the verified PostgreSQL revision to analytical projection and paged reads.
// Inputs: existing platform DB, analytical client and versioned B2 store. Outputs: two Activity groups.
// Effects: none until registered Activities run. Pick for the optional approved graph worker lane.
type ApprovedGraphGroup struct {
	Projection activities.ApprovedContextGraphActivities
	Query      approvedgraphqueryflow.Activities
}

type approvedGraphResolver struct{ reader approvedgraph.Reader }

// ResolveApprovedGraph resolves the exact persisted receipt, candidates and source pins.
// Inputs: one canonical scope. Outputs: verified revision. Effects: read-only PostgreSQL I/O.
// Pick as the concrete adapter for ProjectApprovedContextGraph, never for candidate creation.
func (r approvedGraphResolver) ResolveApprovedGraph(ctx context.Context, scope approvedgraph.Scope) (approvedgraph.Revision, error) {
	if err := approvedgraphprojectionflow.ValidateScope(scope); err != nil {
		return approvedgraph.Revision{}, err
	}
	return approvedgraph.Resolve(ctx, r.reader, scope)
}

var _ activities.ApprovedGraphResolver = approvedGraphResolver{}

// BuildApprovedGraph builds both optional Activity groups from the existing service identities.
// Inputs: platform DB and the worker's existing versioned B2 store. Outputs: nil when disabled, or a configured group.
// Effects: mounted credential reads and client construction only; no graph, source or B2 I/O.
// Pick at worker boot; malformed opt-in or mounted configuration fails visibly without gating ordinary intake.
func BuildApprovedGraph(db platformpostgres.DB, versionStore libraryvalidation.VersionStore) (*ApprovedGraphGroup, error) {
	switch strings.TrimSpace(os.Getenv(approvedGraphEnabledEnv)) {
	case "":
		return nil, nil
	case "true":
	default:
		return nil, fmt.Errorf("proffer worker: %s must be true or unset", approvedGraphEnabledEnv)
	}
	if db == nil || versionStore == nil {
		return nil, errors.New("proffer worker: approved graph requires existing platform DB and versioned B2 store")
	}
	reader, err := platformpostgres.NewApprovedGraphReader(db)
	if err != nil {
		return nil, fmt.Errorf("proffer worker: approved graph reader: %w", err)
	}
	cfg, err := surrealsink.AnalysisConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("proffer worker: approved graph analysis configuration: %w", err)
	}
	client, err := surrealsink.NewAnalysis(cfg)
	if err != nil {
		return nil, fmt.Errorf("proffer worker: approved graph analysis client: %w", err)
	}
	policy := strings.TrimSpace(os.Getenv("ANALYSIS_GRAPH_ACCESS_POLICY_ID"))
	service := strings.TrimSpace(os.Getenv("ANALYSIS_GRAPH_CREATED_BY_SERVICE"))
	if policy == "" || service == "" {
		return nil, errors.New("proffer worker: approved graph requires existing analytical access policy and service IDs")
	}
	sink := &surrealsink.ApprovedClaimsSink{Client: client, AccessPolicyID: policy, CreatedByService: service}
	artifacts := libraryvalidation.B2Artifacts{Store: versionStore, Bucket: approvedgraphqueryflow.ApprovedQueryBucket,
		Prefix: approvedgraphqueryflow.ApprovedQueryPrefixRoot + "approved-context"}
	if err := artifacts.Validate(); err != nil {
		return nil, fmt.Errorf("proffer worker: approved graph query artifact store: %w", err)
	}
	return &ApprovedGraphGroup{Projection: activities.ApprovedContextGraphActivities{Resolver: approvedGraphResolver{reader}, Sink: sink},
		Query: approvedgraphqueryflow.Activities{Reader: sink, Artifacts: artifacts}}, nil
}

// RegisterApprovedGraph installs the projection and query workflows and atomic Activities on one queue.
// Inputs: existing worker registrar and configured group. Outputs: none. Effects: Temporal registration only.
// Pick in the existing proffer worker registration path when BuildApprovedGraph returns a group.
func RegisterApprovedGraph(registrar ExtractionRegistrar, group *ApprovedGraphGroup) {
	if group == nil {
		return
	}
	registrar.RegisterWorkflowWithOptions(approvedgraphprojectionflow.Workflow, workflow.RegisterOptions{Name: approvedgraphprojectionflow.WorkflowName})
	registrar.RegisterActivityWithOptions(group.Projection.ProjectApprovedContextGraph, activity.RegisterOptions{Name: approvedgraphprojectionflow.ActivityName})
	registrar.RegisterWorkflowWithOptions(approvedgraphqueryflow.Workflow, workflow.RegisterOptions{Name: approvedgraphqueryflow.WorkflowName})
	registrar.RegisterActivityWithOptions(group.Query.Run, activity.RegisterOptions{Name: approvedgraphqueryflow.ActivityName})
}
