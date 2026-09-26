// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// The repair workflow builder's five routes (shared contract in
// docs/pending-review/2026-09-25-repair-workflow-builder.md), under
// /reference-import/repair/ with the same tailnet + service-token
// authorization as every other preview route. The Workbench BFF passes them
// through as /api/proffer/repair/{tools,propose,validate,run,runs/{id}}.
//
// Every error keeps its status and a `detail` string; a refused run is 422
// with the failed rules in `detail` and the full checklist in `checks`.

package runtimeapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Cursedpotential/probata/engine/repairplan"
)

// RepairPlanService is what the repair routes need; repairplan.Service
// implements it.
type RepairPlanService interface {
	Tools() repairplan.ToolsResponse
	Propose(context.Context, repairplan.ProposeRequest) (repairplan.ProposeResponse, error)
	Validate(context.Context, repairplan.Plan) (repairplan.ValidatedPlan, error)
	Run(context.Context, repairplan.Plan) (repairplan.RunResponse, repairplan.ValidatedPlan, error)
	Status(context.Context, string) (repairplan.RunStatus, error)
}

// UseRepairPlans enables the repair routes. Without it they answer 503, so a
// misconfigured deployment is visible instead of looking unbuilt.
func (h *PreviewHTTPHandler) UseRepairPlans(service RepairPlanService) error {
	if service == nil {
		return errors.New("repair plans: service is required")
	}
	h.repair = service
	return nil
}

func (h *PreviewHTTPHandler) repairService(w http.ResponseWriter) (RepairPlanService, bool) {
	if h.repair == nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("repair plans are not configured on this service"))
		return nil, false
	}
	return h.repair, true
}

func (h *PreviewHTTPHandler) repairTools(w http.ResponseWriter, _ *http.Request) {
	service, ok := h.repairService(w)
	if !ok {
		return
	}
	response := service.Tools()
	if response.Tools == nil {
		response.Tools = []repairplan.Tool{}
	}
	previewJSON(w, http.StatusOK, response)
}

func (h *PreviewHTTPHandler) repairPropose(w http.ResponseWriter, r *http.Request) {
	service, ok := h.repairService(w)
	if !ok {
		return
	}
	var request repairplan.ProposeRequest
	if err := decodePreviewJSON(w, r, &request); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return
	}
	response, err := service.Propose(r.Context(), request)
	switch {
	case err == nil:
	case errors.Is(err, repairplan.ErrAnchorNotFound):
		if request.PreviewHandle != nil && *request.PreviewHandle != "" {
			previewError(w, http.StatusNotFound, errors.New("preview handle not found"))
			return
		}
		previewError(w, http.StatusUnprocessableEntity, errors.New("no Review run exists for this source; start it in Review first"))
		return
	case errors.Is(err, repairplan.ErrAnchorMismatch):
		previewError(w, http.StatusUnprocessableEntity, err)
		return
	case request.SourceRef == "":
		previewError(w, http.StatusBadRequest, err)
		return
	default:
		previewError(w, http.StatusServiceUnavailable, errors.New("repair proposals are unavailable: "+err.Error()))
		return
	}
	if response.Proposals == nil {
		response.Proposals = []repairplan.Proposal{}
	}
	for index := range response.Proposals {
		if response.Proposals[index].Steps == nil {
			response.Proposals[index].Steps = []repairplan.Step{}
		}
	}
	previewJSON(w, http.StatusOK, response)
}

func decodeRepairPlan(w http.ResponseWriter, r *http.Request) (repairplan.Plan, bool) {
	var plan repairplan.Plan
	if err := decodePreviewJSON(w, r, &plan); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return repairplan.Plan{}, false
	}
	if err := plan.ShapeError(); err != nil {
		previewError(w, http.StatusBadRequest, err)
		return repairplan.Plan{}, false
	}
	return plan, true
}

func (h *PreviewHTTPHandler) repairValidate(w http.ResponseWriter, r *http.Request) {
	service, ok := h.repairService(w)
	if !ok {
		return
	}
	plan, ok := decodeRepairPlan(w, r)
	if !ok {
		return
	}
	validated, err := service.Validate(r.Context(), plan)
	if err != nil {
		previewError(w, http.StatusServiceUnavailable, errors.New("the plan could not be validated: "+err.Error()))
		return
	}
	previewJSON(w, http.StatusOK, validated.Response())
}

// repairRunRefused is the 422 body: the BFF reads `detail`; the checklist
// rides along for a caller that shows it.
type repairRunRefused struct {
	Detail string             `json:"detail"`
	OK     bool               `json:"ok"`
	Checks []repairplan.Check `json:"checks"`
}

func (h *PreviewHTTPHandler) repairRun(w http.ResponseWriter, r *http.Request) {
	service, ok := h.repairService(w)
	if !ok {
		return
	}
	plan, ok := decodeRepairPlan(w, r)
	if !ok {
		return
	}
	response, validated, err := service.Run(r.Context(), plan)
	switch {
	case err == nil:
		previewJSON(w, http.StatusCreated, response)
	case errors.Is(err, repairplan.ErrPlanInvalid):
		checks := validated.Response().Checks
		previewJSON(w, http.StatusUnprocessableEntity, repairRunRefused{Detail: validated.FailedSummary(), Checks: checks})
	default:
		previewError(w, http.StatusServiceUnavailable, errors.New("the repair run could not start: "+err.Error()))
	}
}

func (h *PreviewHTTPHandler) repairRunStatus(w http.ResponseWriter, r *http.Request) {
	service, ok := h.repairService(w)
	if !ok {
		return
	}
	workflowID := r.PathValue("workflow_id")
	if !repairplan.WorkflowIDPattern.MatchString(workflowID) {
		previewError(w, http.StatusBadRequest, errors.New("workflow_id must be 8-160 URL-safe characters"))
		return
	}
	status, err := service.Status(r.Context(), workflowID)
	switch {
	case errors.Is(err, repairplan.ErrRunNotFound):
		previewError(w, http.StatusNotFound, errors.New("repair run not found"))
		return
	case err != nil:
		previewError(w, http.StatusServiceUnavailable, errors.New("the repair run's status is unavailable: "+err.Error()))
		return
	}
	if status.Steps == nil {
		status.Steps = []repairplan.StepStatus{}
	}
	previewJSON(w, http.StatusOK, status)
}
