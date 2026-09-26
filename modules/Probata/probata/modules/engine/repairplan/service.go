// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrPlanInvalid reports a plan the validator refused; the HTTP surface
// answers 422 with the checks.
var ErrPlanInvalid = errors.New("repairplan: the plan failed validation")

// ErrRunNotFound reports a workflow id with no repair run.
var ErrRunNotFound = errors.New("repairplan: no repair run with that workflow id")

// ErrAnchorMismatch reports a preview handle that belongs to another source.
var ErrAnchorMismatch = errors.New("repairplan: the preview handle belongs to a different source")

// RunClient is the Temporal boundary for repair runs.
type RunClient interface {
	// StartPlan starts RepairPlanWorkflow under workflowID, or joins the run
	// already using it, and returns that run's id.
	StartPlan(ctx context.Context, workflowID string, input RunInput) (runID string, err error)
	// PlanStatus queries a run's RunStatus.
	PlanStatus(ctx context.Context, workflowID string) (RunStatus, error)
}

// Service composes the registry, proposer, validator and runner behind the
// five contract routes.
type Service struct {
	Env  Environment
	Runs RunClient
}

func (s Service) registry() *Registry {
	if s.Env.Registry == nil {
		return DefaultRegistry()
	}
	return s.Env.Registry
}

// Tools lists every repair-capable Activity.
func (s Service) Tools() ToolsResponse {
	return ToolsResponse{Tools: s.registry().Tools()}
}

// Propose returns the rule proposals for the source's signature, keeping
// only those that would pass validation here and now: the Workbench never
// offers a plan that cannot run (owner, 2026-09-25). The wait option has no
// steps and is always offered when the table lists it.
func (s Service) Propose(ctx context.Context, request ProposeRequest) (ProposeResponse, error) {
	if strings.TrimSpace(request.SourceRef) == "" {
		return ProposeResponse{}, errors.New("source_ref is required")
	}
	if s.Env.Anchors == nil {
		return ProposeResponse{}, errors.New("repairplan: no anchor resolver is configured")
	}
	handle := ""
	if request.PreviewHandle != nil {
		handle = strings.TrimSpace(*request.PreviewHandle)
	}
	anchor, err := s.Env.Anchors.ResolveAnchor(ctx, request.SourceRef, handle)
	if err != nil {
		return ProposeResponse{}, err
	}
	if anchor.SourceRef != strings.TrimSpace(request.SourceRef) {
		return ProposeResponse{}, fmt.Errorf("%w: Review run %s imported %s, not %s", ErrAnchorMismatch, anchor.PreviewHandle, anchor.SourceRef, request.SourceRef)
	}
	signature, _, _ := Signature(ProposalEvidence{SourceRef: request.SourceRef, Anchor: &anchor, DetectReport: request.DetectReport})
	candidates, covered := TableProposals(signature)
	response := ProposeResponse{Signature: signature, Proposals: []Proposal{}}
	if !covered {
		return response, nil
	}
	mode := ""
	if s.Env.MatterMode != nil {
		mode, _ = s.Env.MatterMode(anchor.MatterID, anchor.CourtCaseID)
	}
	for _, candidate := range candidates {
		if len(candidate.Steps) == 0 {
			response.Proposals = append(response.Proposals, candidate)
			continue
		}
		plan := Plan{
			PlanID: "proposal-feasibility", SourceRef: request.SourceRef, PreviewHandle: &anchor.PreviewHandle,
			MatterMode: mode, Steps: candidate.Steps,
		}
		if validated := s.Env.ValidateAgainst(plan, &anchor); validated.OK {
			response.Proposals = append(response.Proposals, candidate)
		}
	}
	return response, nil
}

// Validate runs every rule against the plan.
func (s Service) Validate(ctx context.Context, plan Plan) (ValidatedPlan, error) {
	if err := plan.ShapeError(); err != nil {
		return ValidatedPlan{}, err
	}
	return s.Env.Validate(ctx, plan)
}

// Run validates and, only when every check passes, starts the plan's run.
// A refused plan returns ErrPlanInvalid with the validation attached.
func (s Service) Run(ctx context.Context, plan Plan) (RunResponse, ValidatedPlan, error) {
	if s.Runs == nil {
		return RunResponse{}, ValidatedPlan{}, errors.New("repairplan: repair runs are not configured on this service")
	}
	validated, err := s.Validate(ctx, plan)
	if err != nil {
		return RunResponse{}, ValidatedPlan{}, err
	}
	if !validated.OK {
		return RunResponse{}, validated, ErrPlanInvalid
	}
	workflowID := WorkflowIDFor(plan)
	runID, err := s.Runs.StartPlan(ctx, workflowID, RunInput{Plan: plan})
	if err != nil {
		return RunResponse{}, validated, fmt.Errorf("start repair run: %w", err)
	}
	return RunResponse{WorkflowID: workflowID, RunID: runID}, validated, nil
}

// Status reads one run.
func (s Service) Status(ctx context.Context, workflowID string) (RunStatus, error) {
	if s.Runs == nil {
		return RunStatus{}, errors.New("repairplan: repair runs are not configured on this service")
	}
	if !WorkflowIDPattern.MatchString(workflowID) {
		return RunStatus{}, ErrRunNotFound
	}
	return s.Runs.PlanStatus(ctx, workflowID)
}
