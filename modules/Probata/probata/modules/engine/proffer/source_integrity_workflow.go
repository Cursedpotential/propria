// Byline: Codex · GPT-6.1 · 2026-10-06. Selective port from held 7a1db624; independent byte-only operation.
package proffer

// Byline: Codex · 2026-10-04. Standalone sibling: CallLogBackfillWorkflow.

import (
	"errors"
	"strings"

	"github.com/Cursedpotential/probata/engine/stagegraph"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// SourceIntegrityWorkflowName is the separately callable integrity workflow registration.
const SourceIntegrityWorkflowName = "proffer_source_integrity_workflow"

// SourceIntegrityInput identifies one retained original without carrying source bytes in history.
// Inputs: original source workflow ID, version/object refs and optional format binding.
// Output: one Activity request; side effects: none. Pick for an explicit independent check.
type SourceIntegrityInput struct {
	RequestID        string `json:"request_id"`
	SourceVersionRef Ref    `json:"source_version_ref"`
	OriginalRef      Ref    `json:"original_ref"`
	DeclaredFormat   string `json:"declared_format,omitempty"`
}

// SourceIntegrityWorkflow schedules exactly one independent integrity Activity for a retained source.
// Input: source references; the actual workflow ID supplies the operation idempotency coordinate.
// Output: exact result/receipt refs or a failed/canceled history retaining those bounded details.
// Side effects: one Activity only; pick instead of a full Proffer run when only byte assessment is wanted.
func SourceIntegrityWorkflow(ctx workflow.Context, in SourceIntegrityInput) (StageResult, error) {
	if strings.TrimSpace(in.RequestID) == "" || len(in.RequestID) > 256 || strings.TrimSpace(string(in.SourceVersionRef)) == "" || len(in.SourceVersionRef) > 64 || strings.TrimSpace(string(in.OriginalRef)) == "" || len(in.OriginalRef) > 64 || len(in.DeclaredFormat) > 128 {
		return StageResult{}, errors.New("source integrity requires request, version and original references")
	}
	req := StageRequest{RequestID: in.RequestID, SourceVersionRef: in.SourceVersionRef, DeclaredFormat: in.DeclaredFormat,
		Refs: map[string]Ref{"original": in.OriginalRef, "integrity_operation": Ref(workflow.GetInfo(ctx).WorkflowExecution.ID)}}
	var result StageResult
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, optionsFor(stagegraph.AssessSourceIntegrity)), string(stagegraph.AssessSourceIntegrity), req).Get(ctx, &result)
	if err != nil {
		return result, err
	}
	if result.Stage != stagegraph.AssessSourceIntegrity {
		return StageResult{}, errors.New("source integrity returned a different operation")
	}
	if err = validateStageResult(result); err != nil {
		return StageResult{}, err
	}
	if result.Status != StatusSuccess {
		return result, temporal.NewNonRetryableApplicationError("source integrity incomplete", "SourceIntegrityIncomplete", nil, result)
	}
	return result, nil
}
