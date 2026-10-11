// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-10-02
//
// CallLogBackfillWorkflow runs commit_call_log_activity once for a run that
// completed before the call-log stage existed (owner 2026-10-02: back-fill the
// call logs already imported). It re-uses that run's own references (source
// version, normalized generation, participant resolution and approved preview),
// so nothing is parsed, normalized or published to Weaviate a second time; the
// Activity's own gate still requires the approved decision on that run's
// preview of that generation.
package proffer

import (
	"errors"
	"strings"

	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// CallLogBackfillWorkflowName is the registered workflow type.
const CallLogBackfillWorkflowName = "proffer_call_log_backfill_workflow"

// CallLogBackfillInput names one completed run by its own references.
type CallLogBackfillInput struct {
	OperatingMode            string `json:"operating_mode,omitempty"`
	RequestID                string `json:"request_id"`
	MatterID                 string `json:"matter_id"`
	CourtCaseID              string `json:"court_case_id"`
	SourceVersionRef         Ref    `json:"source_version_ref"`
	NormalizedGenerationRef  Ref    `json:"normalized_generation_ref"`
	PreviewHandle            Ref    `json:"preview_handle"`
	ParticipantResolutionRef Ref    `json:"participant_resolution_ref,omitempty"`
	OwnerPersonID            string `json:"owner_person_id,omitempty"`
	PerspectivePersonID      string `json:"perspective_person_id,omitempty"`
	DeclaredFormat           string `json:"declared_format"`
}

// CallLogBackfillWorkflow is CallLogBackfillWorkflowName.
func CallLogBackfillWorkflow(ctx workflow.Context, in CallLogBackfillInput) (StageResult, error) {
	for _, required := range []struct{ name, value string }{
		{"request_id", in.RequestID}, {"source_version_ref", string(in.SourceVersionRef)},
		{"normalized_generation_ref", string(in.NormalizedGenerationRef)}, {"preview_handle", string(in.PreviewHandle)},
	} {
		if strings.TrimSpace(required.value) == "" {
			return StageResult{}, errors.New("call log back-fill requires " + required.name)
		}
	}
	refs := WorkflowInput{OwnerPersonID: in.OwnerPersonID, PerspectivePersonID: in.PerspectivePersonID}.personRefs(map[string]Ref{
		"normalized_generation": in.NormalizedGenerationRef,
		"preview_handle":        in.PreviewHandle,
	})
	if in.ParticipantResolutionRef != "" {
		refs["participant_resolution"] = in.ParticipantResolutionRef
	}
	req := StageRequest{
		OperatingMode: in.OperatingMode, RequestID: in.RequestID, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID,
		SourceVersionRef: in.SourceVersionRef, DeclaredFormat: in.DeclaredFormat, Refs: refs,
	}
	var result StageResult
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, optionsFor(stagegraph.CommitCallLog)),
		string(stagegraph.CommitCallLog), req).Get(ctx, &result)
	return result, err
}
