package proffer

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const ContextExtractionRetryWorkflowName = "ContextExtractionRetryWorkflow"

// ContextExtractionRetryInput pins a new configured-model attempt to existing native source bundles.
// Inputs: actor, parent workflow, registered source version and immutable predecessor references.
// Output: reference-only retry workflow input. Effects: none; choose over reingesting or changing source IDs.
type ContextExtractionRetryInput struct {
	ParentWorkflowID string              `json:"parent_workflow_id"`
	ActorSubjectUID  string              `json:"actor_subject_uid"`
	Source           ContextReviewSource `json:"source"`
	PreparedRef      string              `json:"prepared_ref"`
	WorkProductsRef  string              `json:"work_products_ref"`
	SourceFormat     string              `json:"source_format"`
	PackageRef       string              `json:"package_ref"`
	MatterID         string              `json:"matter_id"`
	CourtCaseID      string              `json:"court_case_id"`
}

// ContextExtractionRetryWorkflow reruns candidate extraction and staging without replacing previous attempts or reviews.
// Inputs: server-verified parent/source/bundle coordinates. Output: standard context result with new candidate refs.
// Effects: existing Python extraction and Go candidate staging activities; bundles use the actual new Temporal run ID.
// Pick for configured-model retry after preparation, independently of source/chunk/search availability.
func ContextExtractionRetryWorkflow(ctx workflow.Context, in ContextExtractionRetryInput) (WorkflowResult, error) {
	summary := ContextSummary{ActorSubjectUID: in.ActorSubjectUID, SourceVersionRef: in.Source.SourceVersionID,
		PreparedRef: in.PreparedRef, WorkProductsRef: in.WorkProductsRef, Status: "enriching"}
	if in.ActorSubjectUID == "" || in.ParentWorkflowID == "" || in.Source.SourceVersionID == "" || in.Source.SourceRef == "" || in.PreparedRef == "" || in.WorkProductsRef == "" {
		return WorkflowResult{}, errors.New("extraction retry requires the verified parent, actor, source and predecessor bundles")
	}
	if err := workflow.SetQueryHandler(ctx, ContextStatusQueryName, func() (ContextProgress, error) { return ContextProgress{summary}, nil }); err != nil {
		return WorkflowResult{}, err
	}
	finish := func(reason string) (WorkflowResult, error) {
		summary.Status, summary.Reason = "failed", reason
		return WorkflowResult{Context: &summary}, nil
	}
	info := workflow.GetInfo(ctx)
	providerVersion := ""
	if in.Source.VersionID != nil {
		providerVersion = *in.Source.VersionID
	}
	request := aiContextRequest{ContractVersion: "ai-context-v1", SourceRef: in.Source.SourceRef,
		ProviderVersionID: providerVersion, PackageRef: in.PackageRef, SourceFormat: in.SourceFormat,
		RequestID: in.ParentWorkflowID, TemporalRunID: info.WorkflowExecution.RunID,
		PreparedRef: in.PreparedRef, WorkProductsRef: in.WorkProductsRef,
		ContextResourceBounds: (ContextResourceBounds{}).defaults()}
	var result aiContextReceipt
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, chunkActivityOptions(2*time.Hour)), contextCandidatesActivityName, request).Get(ctx, &result)
	if err != nil {
		return finish(err.Error())
	}
	if err := result.validate("candidates", request.ContextResourceBounds); err != nil {
		return finish(err.Error())
	}
	summary.CandidatesRef, summary.Candidates = result.BundleRef, result.Candidates
	review := ContextReviewRequest{ContractVersion: "ai-context-v1", OperatingMode: "LIVE", MatterID: in.MatterID,
		CourtCaseID: in.CourtCaseID, RequestID: in.ParentWorkflowID, Source: in.Source,
		PreparedRef: in.PreparedRef, BundleRef: result.BundleRef}
	var staged ContextReviewResult
	options := workflow.ActivityOptions{StartToCloseTimeout: 2 * time.Minute, ScheduleToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 3}}
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, options), "stage_ai_candidate_bundle_activity", review).Get(ctx, &staged); err != nil {
		return finish(err.Error())
	}
	if staged.Source == nil || staged.Source.SourceVersionID != in.Source.SourceVersionID || staged.Source.SourceRef != in.Source.SourceRef || staged.Source.PreparedRef != in.PreparedRef {
		return finish("retry staging changed the original source/version/prepared pins")
	}
	summary.Review, summary.ReviewSource = &staged, staged.Source
	summary.CandidateStage = &ContextCandidateStage{Status: "staged", Candidates: staged.Candidates, Staged: staged.Staged}
	summary.Status = result.Status
	if summary.Status == "" {
		summary.Status = "complete"
	}
	return WorkflowResult{Context: &summary}, nil
}
