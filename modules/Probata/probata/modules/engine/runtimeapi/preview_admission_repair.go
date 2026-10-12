package runtimeapi

import (
	"context"
	"errors"
	"github.com/Cursedpotential/probata/engine/proffer"
	"net/http"
)

// previewAdmissionRepairWorkflows uses the existing starter client for independent source-bound repairs.
// Inputs: validated owner action or returned attempt. Output: actual Temporal identities/status. Effects: delegated only.
type previewAdmissionRepairWorkflows interface {
	StartPreviewAdmissionRepair(context.Context, proffer.PreviewAdmissionRepairInput) (proffer.ContextStarted, error)
	PreviewAdmissionRepairStatus(context.Context, string, string) (proffer.PreviewAdmissionRepairResult, error)
}

// repairAdmission starts an explicit readmission workflow while preserving the invalid historical initial event.
// Inputs: exact preview path, actual version/case, explicit mode, reason, existing actor/key headers.
// Output: accepted workflow/run/status URL. Effects: Temporal scheduling only, no HTTP database repair or extraction.
// Pick only for missing historical mode; existing service authentication remains required.
func (h *PreviewHTTPHandler) repairAdmission(w http.ResponseWriter, r *http.Request) {
	var in proffer.PreviewAdmissionRepairInput
	if err := decodePreviewJSON(w, r, &in); err != nil {
		previewError(w, 400, err)
		return
	}
	handle := r.PathValue("preview_handle")
	if in.PreviewHandle != "" && in.PreviewHandle != handle {
		previewError(w, 409, errors.New("repair handle differs from path"))
		return
	}
	in.PreviewHandle = handle
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, 401, err)
		return
	}
	in.ActorSubjectUID, in.ActorUsername, in.IdempotencyKey = actor.SubjectUID, actor.Username, key
	if err := proffer.ValidatePreviewAdmissionRepair(in); err != nil {
		previewError(w, 422, err)
		return
	}
	starter, ok := h.workflow.(previewAdmissionRepairWorkflows)
	if !ok {
		previewError(w, 503, errors.New("starter has no registered admission repair workflow"))
		return
	}
	started, err := starter.StartPreviewAdmissionRepair(r.Context(), in)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	previewJSON(w, 202, map[string]any{"workflow_id": started.WorkflowID, "run_id": started.RunID, "preview_handle": handle, "source_version_ref": in.SourceVersionRef, "status_path": "/reference-import/admission-repairs/" + started.WorkflowID})
}

// admissionRepairStatus exposes actor-bound real repair business status through the existing service boundary.
// Inputs: workflow ID and owner headers. Output: durable result/failure. Effects: Temporal reads only.
func (h *PreviewHTTPHandler) admissionRepairStatus(w http.ResponseWriter, r *http.Request) {
	actor, _, err := authenticatedActor(r)
	if err != nil {
		previewError(w, 401, err)
		return
	}
	starter, ok := h.workflow.(previewAdmissionRepairWorkflows)
	if !ok {
		previewError(w, 503, errors.New("starter lacks repair status"))
		return
	}
	result, err := starter.PreviewAdmissionRepairStatus(r.Context(), r.PathValue("workflow_id"), actor)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	previewJSON(w, 200, result)
}
