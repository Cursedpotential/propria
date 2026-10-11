// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Package repairplan is the engine side of the repair workflow builder
// (owner ratification 2026-09-25 19:09, option A — a step list built in the
// Workbench, run on Temporal, n8n only for a step that needs it).
//
// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Source of truth: docs/pending-review/2026-09-25-repair-workflow-builder.md,
// "Shared contract". This file holds that contract's JSON shapes exactly; the
// Workbench BFF passes them through unchanged under /api/proffer/repair/.
//
// The package owns four things, each pure and separately testable:
//
//   - the tool registry (registry.go): every repair-capable Activity with its
//     types, params schema, write class and n8n need;
//   - the proposer (proposer.go): a signature table from the source's file
//     name, format and repair report to ordered candidate plans;
//   - the validator (validator.go): fail-closed, one named check per rule;
//   - RepairPlanWorkflow (workflow.go): runs the validated steps in order as
//     Activities, records a receipt per step, and re-enters Proffer on the
//     result.
//
// It holds no Activity bodies and does no I/O of its own; storage, catalog and
// PostgreSQL reach it through narrow interfaces the worker and the starter
// wire (AGENTS.md ATOMICITY rules 3 and 6).
package repairplan

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Cursedpotential/probata/engine/proffer"
)

// Matter modes, exactly as the Workbench spells them.
const (
	ModeDev  = "DEV"
	ModeLive = "LIVE"
	ModeTest = ModeDev
	ModeReal = ModeLive
)

// Plan is the owner's composed repair plan.
type Plan struct {
	PlanID        string  `json:"plan_id"`
	SourceRef     string  `json:"source_ref"`
	PreviewHandle *string `json:"preview_handle"`
	MatterMode    string  `json:"matter_mode"`
	Steps         []Step  `json:"steps"`
	// Reentry carries the run settings the re-entry import needs and no table
	// stores: whose records they are, whose phone or account they come from,
	// and the approval policy. Absent, re-entry runs as before (no perspective,
	// every preview waits in Review). Owner 2026-10-02 19:42 EDT: "Run it
	// through the repair workflow", imported like the device's other files.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	Reentry *ReentryOptions `json:"reentry,omitempty"`
}

// ReentryOptions are passed unchanged to the re-entry run or batch.
type ReentryOptions struct {
	OwnerPersonID       string `json:"owner_person_id,omitempty"`
	PerspectivePersonID string `json:"perspective_person_id,omitempty"`
	AutoApproval        string `json:"auto_approval,omitempty"`
}

// Step is one registered Activity with its parameters.
type Step struct {
	StepID   string          `json:"step_id"`
	Activity string          `json:"activity"`
	Params   json.RawMessage `json:"params"`
}

// Handle is the plan's preview handle, or "" when it names none.
func (p Plan) Handle() string {
	if p.PreviewHandle == nil {
		return ""
	}
	return strings.TrimSpace(*p.PreviewHandle)
}

var (
	planIDPattern        = regexp.MustCompile(`^[A-Za-z0-9_-]{8,96}$`)
	stepIDPattern        = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	previewHandlePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)
	personIDPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// WorkflowIDPattern is what a repair run's workflow id looks like; the
	// runs/{workflow_id} route refuses anything else.
	WorkflowIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,160}$`)
)

// maxParamsBytes bounds one step's params: they are knobs, never content.
// Twelve steps at the bound still fit the 16 KiB request limit.
const maxParamsBytes = 1 << 10

// ShapeError reports a plan that does not match the contract's shape at all
// (a 400, not a failed check). Everything semantic is left to the validator.
func (p Plan) ShapeError() error {
	if !planIDPattern.MatchString(p.PlanID) {
		return errors.New("plan_id must be 8-96 URL-safe characters")
	}
	if strings.TrimSpace(p.SourceRef) == "" || len(p.SourceRef) > 2048 {
		return errors.New("source_ref is required")
	}
	if handle := p.Handle(); p.PreviewHandle != nil && handle != "" && !previewHandlePattern.MatchString(handle) {
		return errors.New("preview_handle must be null or 32-128 URL-safe characters")
	}
	if p.MatterMode != ModeTest && p.MatterMode != ModeReal {
		return errors.New(`matter_mode must be "DEV" or "LIVE"`)
	}
	if p.Steps == nil {
		return errors.New("steps must be a list")
	}
	if r := p.Reentry; r != nil {
		for name, id := range map[string]string{"owner_person_id": r.OwnerPersonID, "perspective_person_id": r.PerspectivePersonID} {
			if id != "" && !personIDPattern.MatchString(id) {
				return fmt.Errorf("reentry.%s must be a person UUID", name)
			}
		}
		if r.AutoApproval != "" && r.AutoApproval != proffer.AutoApprovalCleanChecks {
			return fmt.Errorf("reentry.auto_approval names an unknown policy %q", r.AutoApproval)
		}
	}
	for index, step := range p.Steps {
		if len(step.Params) > maxParamsBytes {
			return fmt.Errorf("step %d params exceed %d bytes", index+1, maxParamsBytes)
		}
	}
	return nil
}

// Tool describes one repair-capable Activity for the Workbench step picker.
type Tool struct {
	ID           string          `json:"id"`
	Description  string          `json:"description"`
	InputTypes   []string        `json:"input_types"`
	OutputTypes  []string        `json:"output_types"`
	ParamsSchema json.RawMessage `json:"params_schema"`
	Writes       string          `json:"writes"`
	NeedsN8N     bool            `json:"needs_n8n"`
}

// ToolsResponse is GET /reference-import/repair/tools.
type ToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// ProposeRequest is POST /reference-import/repair/propose. The Workbench sends
// source_ref and preview_handle; detect_report is optional extra evidence —
// the engine reads the run's own repair assessment when it exists.
type ProposeRequest struct {
	SourceRef     string          `json:"source_ref"`
	PreviewHandle *string         `json:"preview_handle,omitempty"`
	DetectReport  json.RawMessage `json:"detect_report,omitempty"`
}

// Proposal is one ordered candidate plan.
type Proposal struct {
	Signature string `json:"signature"`
	Rationale string `json:"rationale"`
	Steps     []Step `json:"steps"`
	// By is "rule" for the signature table and "agent" for the agent
	// proposer (not wired in this increment).
	By string `json:"by"`
	// AgentAvailable describes what exists now: always false until an agent
	// proposer is wired (owner: never offer what cannot be done).
	AgentAvailable bool `json:"agent_available"`
}

// ProposeResponse lists the table's candidates for the computed signature.
type ProposeResponse struct {
	Signature      string     `json:"signature"`
	Proposals      []Proposal `json:"proposals"`
	AgentAvailable bool       `json:"agent_available"`
}

// Check is one named validator rule with its outcome.
type Check struct {
	Rule   string `json:"rule"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Check statuses.
const (
	CheckPass = "pass"
	CheckFail = "fail"
)

// ValidateResponse is POST /reference-import/repair/validate. OK is true only
// when every check passed.
type ValidateResponse struct {
	OK     bool    `json:"ok"`
	Checks []Check `json:"checks"`
}

// RunResponse is POST /reference-import/repair/run.
type RunResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// Run statuses.
const (
	RunRunning   = "running"
	RunCompleted = "completed"
	RunFailed    = "failed"
)

// Step statuses.
const (
	StepPending   = "pending"
	StepRunning   = "running"
	StepSucceeded = "succeeded"
	StepFailed    = "failed"
	StepSkipped   = "skipped"
)

// RunStatus is GET /reference-import/repair/runs/{workflow_id}. plan_id,
// preview_handle and matter_mode are echoed so the Workbench can prove the
// run's TEST/REAL ownership from the engine instead of process memory.
type RunStatus struct {
	WorkflowID           string       `json:"workflow_id"`
	PlanID               string       `json:"plan_id"`
	PreviewHandle        *string      `json:"preview_handle"`
	MatterMode           string       `json:"matter_mode"`
	Status               string       `json:"status"`
	Reason               string       `json:"reason,omitempty"`
	Steps                []StepStatus `json:"steps"`
	ReentryPreviewHandle string       `json:"reentry_preview_handle,omitempty"`
	// ReentryBatchID is set instead of ReentryPreviewHandle when the result is
	// a folder of derived chunks: re-entry is then one Proffer batch over that
	// folder (GET /reference-import/batches/{id}), not a single run.
	ReentryBatchID string `json:"reentry_batch_id,omitempty"`
	// ReentryReceiptRef is the re-entry's own receipt, which records the
	// supersession link (this run's preview_handle → the re-entry run or batch).
	ReentryReceiptRef string `json:"reentry_receipt_ref,omitempty"`
	// Checks carries the worker's own validation when it refused the plan.
	Checks []Check `json:"checks,omitempty"`
}

// StepStatus is one step's progress. ReceiptRef is always present and empty
// until the step's receipt is recorded.
type StepStatus struct {
	StepID       string          `json:"step_id"`
	Activity     string          `json:"activity"`
	Status       string          `json:"status"`
	ReceiptRef   string          `json:"receipt_ref"`
	OutputRef    string          `json:"output_ref,omitempty"`
	OutputSHA256 string          `json:"output_sha256,omitempty"`
	Summary      json.RawMessage `json:"summary,omitempty"`
	Reason       string          `json:"reason,omitempty"`
}
