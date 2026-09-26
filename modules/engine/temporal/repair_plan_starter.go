// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// The Temporal client side of the repair workflow builder: start one
// RepairPlanWorkflow and read its status. A separate seam from
// WorkflowStarter and BatchStarter so neither surface nor its test doubles
// change.

package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"

	"github.com/Cursedpotential/probata/engine/repairplan"
)

// repairRunPrefix is how every repair run's workflow id begins
// (repairplan.WorkflowIDFor); a status read refuses any other workflow.
const repairRunPrefix = "repair-plan-"

// RepairPlanStarter implements repairplan.RunClient over a Temporal client.
type RepairPlanStarter struct {
	client    client.Client
	taskQueue string
}

// NewRepairPlanStarter wraps an already-dialed client; it does not own it.
func NewRepairPlanStarter(c client.Client, taskQueue string) (*RepairPlanStarter, error) {
	if c == nil {
		return nil, errors.New("temporal: repair plan starter requires a Temporal client")
	}
	if strings.TrimSpace(taskQueue) == "" {
		return nil, errors.New("temporal: repair plan starter requires a task queue")
	}
	return &RepairPlanStarter{client: c, taskQueue: taskQueue}, nil
}

// StartPlan starts RepairPlanWorkflow under workflowID. The id carries the
// plan's digest, so submitting the same plan again joins the run already
// using it, and a finished plan is never re-run under its old id; only a
// failed one may run again (after the owner fixes what failed).
func (s *RepairPlanStarter) StartPlan(ctx context.Context, workflowID string, input repairplan.RunInput) (string, error) {
	if !strings.HasPrefix(workflowID, repairRunPrefix) || !repairplan.WorkflowIDPattern.MatchString(workflowID) {
		return "", fmt.Errorf("temporal: %q is not a repair run id", workflowID)
	}
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: workflowID, TaskQueue: s.taskQueue,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}, repairplan.WorkflowName, input)
	if err != nil {
		if runID, ok := alreadyStartedRunID(err); ok {
			return runID, nil
		}
		return "", fmt.Errorf("temporal: start repair plan workflow: %w", err)
	}
	return run.GetRunID(), nil
}

// PlanStatus queries the run's status. Only repair runs are read: any other
// workflow id is reported as not found rather than queried.
func (s *RepairPlanStarter) PlanStatus(ctx context.Context, workflowID string) (repairplan.RunStatus, error) {
	if !strings.HasPrefix(workflowID, repairRunPrefix) {
		return repairplan.RunStatus{}, repairplan.ErrRunNotFound
	}
	value, err := s.client.QueryWorkflow(ctx, workflowID, "", repairplan.StatusQueryName)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return repairplan.RunStatus{}, repairplan.ErrRunNotFound
		}
		return repairplan.RunStatus{}, fmt.Errorf("temporal: query repair plan status: %w", err)
	}
	var status repairplan.RunStatus
	if err := value.Get(&status); err != nil {
		return repairplan.RunStatus{}, fmt.Errorf("temporal: decode repair plan status: %w", err)
	}
	return status, nil
}
