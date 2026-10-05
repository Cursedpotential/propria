// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package proffer

import (
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

const (
	// autoExtractionChangeID versions the automatic start of entity and event extraction once a
	// run's messages are committed (owner 2026-10-02: "once it gets accepted we should be
	// extracting entities ... extracting events. That's part of the workflow.").
	autoExtractionChangeID = "proffer-auto-extraction-after-commit-v1"
	autoExtractionVersion  = workflow.Version(1)
)

// autoExtractionActor attributes the automatic extraction to the workflow, not to a person.
var autoExtractionActor = entities.Actor{SubjectUID: "system:proffer-auto-extraction", Username: "proffer-auto-extraction"}

// startAutoExtraction starts extraction_request_workflow as an abandoned child with the default extractor once the run's messages are committed.
// A failure to start is recorded on the run's result and logged; it never fails the import.
// The child outlives this run (parent-close policy abandon), so a slow model call cannot hold the import open.
func (r *run) startAutoExtraction(ctx workflow.Context, previewHandle string, sourceVersionRef, generationRef Ref) {
	if workflow.GetVersion(ctx, autoExtractionChangeID, workflow.DefaultVersion, autoExtractionVersion) == workflow.DefaultVersion {
		return
	}
	if previewHandle == "" || generationRef == "" {
		r.autoExtraction = "not started: the run has no preview handle or normalized generation"
		return
	}
	input := flow.RequestInput{
		RequestID: "auto:" + previewHandle,
		MatterID:  r.matterID,
		Runs: []flow.RunRef{{
			MatterMode: r.operatingMode, PreviewHandle: previewHandle, GenerationID: string(generationRef), SourceVersionID: string(sourceVersionRef),
		}},
		Extractors:  []string{flow.DefaultExtractorID},
		Actor:       autoExtractionActor,
		Auto:        true,
		RequestedAt: workflow.Now(ctx).UTC(),
	}
	child := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               flow.AutoExtractionWorkflowIDPrefix + previewHandle,
		WorkflowExecutionTimeout: 12 * time.Hour,
		ParentClosePolicy:        enumspb.PARENT_CLOSE_POLICY_ABANDON,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	})
	var execution workflow.Execution
	if err := workflow.ExecuteChildWorkflow(child, flow.RequestWorkflowName, input).GetChildWorkflowExecution().Get(ctx, &execution); err != nil {
		r.autoExtraction = "not started: " + err.Error()
		workflow.GetLogger(ctx).Warn("automatic extraction did not start; the import is unaffected", "error", err.Error())
		return
	}
	r.autoExtraction = "started: " + execution.ID
}
