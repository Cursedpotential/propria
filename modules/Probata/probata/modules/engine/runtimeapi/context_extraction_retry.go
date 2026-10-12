package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// contextExtractionRetries extends the existing starter with additive source-pinned retry.
// Inputs: exact parent coordinates and an idempotent ID. Output: actual execution IDs.
// Effects: delegated Temporal start; pick only when the configured starter supports this workflow.
type contextExtractionRetries interface {
	StartExtractionRetry(context.Context, string, proffer.ContextExtractionRetryInput) (proffer.ContextStarted, error)
}

// retryExtraction verifies the original actor and source before scheduling the existing candidate activities again.
// Inputs: parent workflow path, registered version, configured model selector, actor and click key.
// Output: actual child attempt IDs and parent/source refs. Effects: additive Temporal workflow only.
// Pick after prepared/work-product bundles exist; original attempts, corrections and source content remain immutable.
func (h *ContextSourceHTTPHandler) retryExtraction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SourceVersionRef string `json:"source_version_ref"`
		ModelID          string `json:"model_id"`
	}
	if err := decodePreviewJSON(w, r, &body); err != nil {
		previewError(w, 400, err)
		return
	}
	actor, key, err := actorAndKey(r)
	if err != nil {
		previewError(w, 401, err)
		return
	}
	if body.ModelID != "configured" {
		previewError(w, 422, errors.New("native worker currently supports its configured extraction model only"))
		return
	}
	parent := r.PathValue("workflow_id")
	progress, err := h.workflows.Status(r.Context(), parent)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	if progress.ActorSubjectUID != actor.SubjectUID {
		previewError(w, 404, errors.New("unknown context workflow"))
		return
	}
	if progress.SourceVersionRef != body.SourceVersionRef || progress.PreparedRef == "" || progress.WorkProductsRef == "" {
		previewError(w, 409, errors.New("retry requires the original registered version and settled predecessor bundles"))
		return
	}
	prepared, err := readContextReviewBundle(progress.PreparedRef)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	products, err := readContextReviewBundle(progress.WorkProductsRef)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	source, ok := prepared["source"].(map[string]any)
	if !ok || prepared["stage"] != "prepared" || products["stage"] != "work_products" || products["prepared_ref"] != progress.PreparedRef || !reflect.DeepEqual(products["source"], source) {
		previewError(w, 409, errors.New("retry predecessor source/version coordinates disagree"))
		return
	}
	ref, _ := source["source_ref"].(string)
	if progress.ReviewSource == nil || ref == "" || progress.ReviewSource.SourceRef != ref || progress.ReviewSource.SourceVersionID != body.SourceVersionRef {
		previewError(w, 409, errors.New("retry requires the independently verified registered native source"))
		return
	}
	starter, ok := h.workflows.(contextExtractionRetries)
	if !ok {
		previewError(w, 503, errors.New("configured starter lacks native extraction retry"))
		return
	}
	format, _ := source["source_format"].(string)
	packageRef, _ := source["package_ref"].(string)
	in := proffer.ContextExtractionRetryInput{ParentWorkflowID: parent, ActorSubjectUID: actor.SubjectUID,
		Source: progress.ReviewSource.ContextReviewSource, PreparedRef: progress.PreparedRef, WorkProductsRef: progress.WorkProductsRef,
		SourceFormat: format, PackageRef: packageRef}
	// Existing single-case identity constants are independently rechecked against the source by staging.
	in.MatterID, in.CourtCaseID = caseidentity.AuthoritativeMatterID, caseidentity.AuthoritativeCourtCaseID
	id := proffer.ContextWorkflowIDPrefix + "retry:" + flow.DeterministicID("context_extraction_retry", parent, body.SourceVersionRef, actor.SubjectUID, key)
	started, err := starter.StartExtractionRetry(r.Context(), id, in)
	if err != nil {
		previewError(w, 503, err)
		return
	}
	previewJSON(w, http.StatusAccepted, map[string]any{"workflow_id": started.WorkflowID, "run_id": started.RunID,
		"parent_workflow_id": parent, "source_version_ref": body.SourceVersionRef, "model_id": "configured"})
}
