// Byline: Claude Code · Opus 5.5 · 2026-09-25

package repairplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

const (
	// WorkflowName is the exact Temporal name the proffer worker registers.
	// It is a separate workflow type: the 26-stage ProfferWorkflow is
	// untouched, and a plan re-enters it only as an ordinary new run.
	WorkflowName = "RepairPlanWorkflow"
	// StatusQueryName returns the run's RunStatus.
	StatusQueryName = "repair_plan_status"
	// FailureType marks a repair plan that ended failed.
	FailureType = "repair_plan_failed"

	// Activity names spelled here rather than imported: this package must not
	// depend on the activities or temporal packages (they depend on it).
	ValidatePlanActivityName        = string(stagegraph.RepairValidatePlan)
	RecordStepReceiptActivityName   = string(stagegraph.RepairRecordStepReceipt)
	bindImportOperationActivityName = "bind_import_operation_activity"
	readImportOperationActivityName = "read_import_operation_activity"
	runFlowActivityName             = "run_n8n_flow_activity"

	// reentryRegisterPoll and reentryRegisterLimit bound the wait for the
	// re-entry run to register its source before it is bound to Review.
	reentryRegisterPoll  = 5 * time.Second
	reentryRegisterLimit = 30 * time.Minute
)

// RunInput starts RepairPlanWorkflow. The plan is small and bounded (at most
// 12 steps of at most 1 KiB params each); no source bytes travel.
type RunInput struct {
	Plan Plan `json:"plan"`
}

// ValidatePlanRequest asks the worker to validate with its own configuration.
type ValidatePlanRequest struct {
	WorkflowID string `json:"workflow_id"`
	Plan       Plan   `json:"plan"`
}

// StepRequest is what one repair Activity receives: a locator and a type,
// never bytes.
type StepRequest struct {
	WorkflowID string          `json:"workflow_id"`
	RunID      string          `json:"run_id"`
	PlanID     string          `json:"plan_id"`
	StepID     string          `json:"step_id"`
	StepIndex  int             `json:"step_index"`
	Activity   string          `json:"activity"`
	SourceRef  string          `json:"source_ref"`
	SourceType string          `json:"source_type"`
	Params     json.RawMessage `json:"params,omitempty"`
}

// StepResult is what one repair Activity returns, by reference.
type StepResult struct {
	// OutputRef locates the step's output: a derived object, a derived
	// manifest, or the other existing copy a re-pointing step chose.
	OutputRef  string `json:"output_ref"`
	OutputType string `json:"output_type"`
	OutputKind string `json:"output_kind"`
	// OutputSHA256 is the sha256 of the derived object (or manifest) exactly
	// as the store serves it. Empty for a re-pointing step.
	OutputSHA256 string `json:"output_sha256,omitempty"`
	// ReentryRef is the folder a batch re-entry imports, for chunk output.
	ReentryRef string `json:"reentry_ref,omitempty"`
	// Reused is true when a finished earlier result was returned unchanged.
	Reused bool `json:"reused,omitempty"`
	// Summary is the step's bounded counts, rendered for the Workbench.
	Summary json.RawMessage `json:"summary,omitempty"`
}

// ReceiptRequest records one step's outcome, success or failure.
type ReceiptRequest struct {
	WorkflowID      string      `json:"workflow_id"`
	RunID           string      `json:"run_id"`
	PlanID          string      `json:"plan_id"`
	StepID          string      `json:"step_id"`
	StepIndex       int         `json:"step_index"`
	Activity        string      `json:"activity"`
	SourceVersionID string      `json:"source_version_id"`
	InputRef        string      `json:"input_ref"`
	Status          string      `json:"status"`
	Result          *StepResult `json:"result,omitempty"`
	Error           string      `json:"error,omitempty"`
}

// Receipt statuses.
const (
	ReceiptSuccess = "success"
	ReceiptFailed  = "failed"
)

// ReceiptResult is the durable receipt reference.
type ReceiptResult struct {
	ReceiptRef string `json:"receipt_ref"`
}

// flowRequest / flowResult mirror run_n8n_flow_activity's JSON contract
// (engine/temporal/flowactivity.go) without importing it.
type flowRequest struct {
	Flow             string            `json:"flow"`
	RequestID        string            `json:"request_id"`
	MatterID         string            `json:"matter_id,omitempty"`
	CourtCaseID      string            `json:"court_case_id,omitempty"`
	SourceVersionRef string            `json:"source_version_ref,omitempty"`
	DeclaredFormat   string            `json:"declared_format,omitempty"`
	Refs             map[string]string `json:"refs,omitempty"`
	Inputs           map[string]any    `json:"inputs,omitempty"`
}

type flowResult struct {
	Flow       string         `json:"flow"`
	Status     string         `json:"status"`
	Ref        string         `json:"ref,omitempty"`
	ReceiptRef string         `json:"receipt_ref,omitempty"`
	Outputs    map[string]any `json:"outputs,omitempty"`
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// WorkflowIDFor names a plan's run: the plan id plus a digest of exactly what
// the plan asks for. Submitting the same plan again joins its run; an edited
// plan is a different run.
func WorkflowIDFor(plan Plan) string {
	canonical := struct {
		PlanID, SourceRef, PreviewHandle, MatterMode string
		Steps                                        [][3]string
	}{PlanID: plan.PlanID, SourceRef: plan.SourceRef, PreviewHandle: plan.Handle(), MatterMode: plan.MatterMode}
	for _, step := range plan.Steps {
		canonical.Steps = append(canonical.Steps, [3]string{step.StepID, step.Activity, canonicalParams(step.Params)})
	}
	encoded, _ := json.Marshal(canonical)
	digest := sha256.Sum256(encoded)
	return "repair-plan-" + plan.PlanID + "-" + hex.EncodeToString(digest[:])[:12]
}

// runState is the query-visible progress. Only the workflow goroutine
// touches it.
type runState struct {
	status RunStatus
}

func newRunState(workflowID string, plan Plan) *runState {
	steps := make([]StepStatus, len(plan.Steps))
	for index, step := range plan.Steps {
		steps[index] = StepStatus{StepID: step.StepID, Activity: step.Activity, Status: StepPending}
	}
	var handle *string
	if value := plan.Handle(); value != "" {
		handle = &value
	}
	return &runState{status: RunStatus{
		PlanID: plan.PlanID, WorkflowID: workflowID, PreviewHandle: handle, MatterMode: plan.MatterMode,
		Status: RunRunning, Steps: steps,
	}}
}

func (s *runState) snapshot() RunStatus {
	out := s.status
	out.Steps = append([]StepStatus(nil), s.status.Steps...)
	out.Checks = append([]Check(nil), s.status.Checks...)
	if out.Steps == nil {
		out.Steps = []StepStatus{}
	}
	if s.status.PreviewHandle != nil {
		handle := *s.status.PreviewHandle
		out.PreviewHandle = &handle
	}
	return out
}

// fail ends the run failed; every step that never ran is marked skipped.
func (s *runState) fail(reason string) (RunStatus, error) {
	s.status.Status, s.status.Reason = RunFailed, reason
	for index := range s.status.Steps {
		if s.status.Steps[index].Status == StepPending || s.status.Steps[index].Status == StepRunning {
			s.status.Steps[index].Status = StepSkipped
		}
	}
	return s.snapshot(), temporal.NewNonRetryableApplicationError(reason, FailureType, nil)
}

func stepOptions(step ResolvedStep) workflow.ActivityOptions {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(step.StartToCloseSeconds) * time.Second,
		RetryPolicy:         boundedRetry(5*time.Second, step.MaxAttempts),
	}
	if step.HeartbeatSeconds > 0 {
		options.HeartbeatTimeout = time.Duration(step.HeartbeatSeconds) * time.Second
	}
	return options
}

func boundedRetry(initial time.Duration, attempts int32) *temporal.RetryPolicy {
	if attempts < 1 {
		attempts = 1
	}
	return &temporal.RetryPolicy{
		InitialInterval: initial, BackoffCoefficient: 2, MaximumInterval: initial * 20, MaximumAttempts: attempts,
	}
}

// RepairPlanWorkflow runs a validated plan's steps in order, one Activity per
// step, records a receipt per step, and re-enters Proffer on the result.
//
// It fails closed: the worker re-validates the plan with its own storage
// configuration before any step; a step failure stops the plan (later steps
// are skipped, never guessed around); and a step without a recorded receipt
// never counts as done. Nothing here reads or writes source bytes — each step
// Activity streams its own input by locator.
func RepairPlanWorkflow(ctx workflow.Context, in RunInput) (RunStatus, error) {
	info := workflow.GetInfo(ctx)
	workflowID, runID := info.WorkflowExecution.ID, info.WorkflowExecution.RunID
	state := newRunState(workflowID, in.Plan)
	if err := workflow.SetQueryHandler(ctx, StatusQueryName, func() (RunStatus, error) {
		return state.snapshot(), nil
	}); err != nil {
		return state.snapshot(), err
	}

	var validated ValidatedPlan
	validateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, RetryPolicy: boundedRetry(2*time.Second, 3),
	})
	if err := workflow.ExecuteActivity(validateCtx, ValidatePlanActivityName,
		ValidatePlanRequest{WorkflowID: workflowID, Plan: in.Plan}).Get(ctx, &validated); err != nil {
		return state.fail("the worker could not validate the plan: " + err.Error())
	}
	if handle := validated.Anchor.PreviewHandle; handle != "" {
		state.status.PreviewHandle = &handle
	}
	if !validated.OK {
		state.status.Checks = validated.Checks
		return state.fail("the worker refused the plan with its own configuration — " + validated.FailedSummary())
	}
	if len(validated.Steps) != len(in.Plan.Steps) {
		return state.fail("the validated plan does not match the submitted steps")
	}

	currentRef, currentType := validated.SourceRef, validated.SourceType
	var terminal StepResult
	for index, step := range validated.Steps {
		state.status.Steps[index].Status = StepRunning
		result, stepErr := executeStep(ctx, workflowID, runID, validated, index, step, currentRef, currentType)
		if stepErr == nil {
			stepErr = checkStepResult(step, result)
		}
		receipt := ReceiptRequest{
			WorkflowID: workflowID, RunID: runID, PlanID: validated.PlanID, StepID: step.StepID, StepIndex: index,
			Activity: step.Activity, SourceVersionID: validated.Anchor.SourceVersionID, InputRef: currentRef,
			Status: ReceiptSuccess,
		}
		if stepErr != nil {
			receipt.Status, receipt.Error = ReceiptFailed, boundedReason(stepErr.Error())
		} else {
			resultCopy := result
			receipt.Result = &resultCopy
		}
		var recorded ReceiptResult
		receiptCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute, RetryPolicy: boundedRetry(2*time.Second, 5),
		})
		receiptErr := workflow.ExecuteActivity(receiptCtx, RecordStepReceiptActivityName, receipt).Get(ctx, &recorded)
		state.status.Steps[index].ReceiptRef = recorded.ReceiptRef
		if stepErr != nil {
			reason := boundedReason(stepErr.Error())
			if receiptErr != nil {
				reason += "; its failure receipt could not be recorded: " + boundedReason(receiptErr.Error())
			}
			state.status.Steps[index].Status, state.status.Steps[index].Reason = StepFailed, reason
			return state.fail(fmt.Sprintf("step %s (%s) failed: %s", step.StepID, step.Activity, reason))
		}
		if receiptErr != nil || recorded.ReceiptRef == "" {
			reason := "the step finished but its receipt could not be recorded"
			if receiptErr != nil {
				reason += ": " + boundedReason(receiptErr.Error())
			}
			state.status.Steps[index].Status, state.status.Steps[index].Reason = StepFailed, reason
			return state.fail(fmt.Sprintf("step %s (%s): %s", step.StepID, step.Activity, reason))
		}
		state.status.Steps[index].Status = StepSucceeded
		state.status.Steps[index].OutputRef = result.OutputRef
		state.status.Steps[index].OutputSHA256 = result.OutputSHA256
		state.status.Steps[index].Summary = result.Summary
		currentRef, currentType, terminal = result.OutputRef, result.OutputType, result
	}

	if err := reenter(ctx, workflowID, runID, validated, terminal, state); err != nil {
		return state.fail("every step succeeded but re-entry into Proffer failed: " + boundedReason(err.Error()))
	}
	state.status.Status = RunCompleted
	return state.snapshot(), nil
}

func executeStep(ctx workflow.Context, workflowID, runID string, validated ValidatedPlan, index int, step ResolvedStep, sourceRef, sourceType string) (StepResult, error) {
	stepCtx := workflow.WithActivityOptions(ctx, stepOptions(step))
	if step.NeedsN8N {
		return executeFlowStep(ctx, stepCtx, workflowID, runID, validated, index, step, sourceRef)
	}
	var result StepResult
	err := workflow.ExecuteActivity(stepCtx, step.Activity, StepRequest{
		WorkflowID: workflowID, RunID: runID, PlanID: validated.PlanID, StepID: step.StepID, StepIndex: index,
		Activity: step.Activity, SourceRef: sourceRef, SourceType: sourceType, Params: step.Params,
	}).Get(ctx, &result)
	return result, err
}

// executeFlowStep runs a step whose tool is an n8n flow through the generic
// run_n8n_flow_activity (n8n owns the node, Temporal owns durability).
func executeFlowStep(ctx, stepCtx workflow.Context, workflowID, runID string, validated ValidatedPlan, index int, step ResolvedStep, sourceRef string) (StepResult, error) {
	inputs := map[string]any{}
	if len(step.Params) > 0 {
		if err := json.Unmarshal(step.Params, &inputs); err != nil {
			return StepResult{}, fmt.Errorf("step params are not an object: %w", err)
		}
	}
	var flow flowResult
	err := workflow.ExecuteActivity(stepCtx, runFlowActivityName, flowRequest{
		Flow: step.FlowName, RequestID: fmt.Sprintf("%s-%s-%02d-%s", workflowID, shortRunID(runID), index, step.StepID),
		MatterID: validated.Anchor.MatterID, CourtCaseID: validated.Anchor.CourtCaseID,
		SourceVersionRef: validated.Anchor.SourceVersionID, DeclaredFormat: validated.Anchor.DeclaredFormat,
		Refs: map[string]string{"source": sourceRef}, Inputs: inputs,
	}).Get(ctx, &flow)
	if err != nil {
		return StepResult{}, err
	}
	if flow.Status != string(proffer.StatusSuccess) {
		return StepResult{}, fmt.Errorf("n8n flow %q reported %q", step.FlowName, flow.Status)
	}
	summary, _ := json.Marshal(flow.Outputs)
	result := StepResult{OutputRef: flow.Ref, OutputType: step.OutputType, OutputKind: step.OutputKind, Summary: summary}
	if value, ok := flow.Outputs["output_sha256"].(string); ok {
		result.OutputSHA256 = value
	}
	if value, ok := flow.Outputs["reentry_ref"].(string); ok {
		result.ReentryRef = value
	}
	return result, nil
}

// checkStepResult holds an Activity to the contract the validator promised.
func checkStepResult(step ResolvedStep, result StepResult) error {
	switch {
	case strings.TrimSpace(result.OutputRef) == "":
		return errors.New("the step reported no output reference")
	case result.OutputType != step.OutputType:
		return fmt.Errorf("the step produced type %q where the plan was validated for %q", result.OutputType, step.OutputType)
	case result.OutputKind != step.OutputKind:
		return fmt.Errorf("the step produced output kind %q where the plan was validated for %q", result.OutputKind, step.OutputKind)
	}
	switch step.OutputKind {
	case OutputDerivedObject, OutputDerivedChunkFolder:
		if !sha256Pattern.MatchString(result.OutputSHA256) {
			return errors.New("the step published derived output without its sha256")
		}
	}
	if step.OutputKind == OutputDerivedChunkFolder && !strings.HasSuffix(result.ReentryRef, "/") {
		return errors.New("the step published chunks without the folder to re-enter")
	}
	return nil
}

// reenter starts the ordinary Proffer route on the plan's result. The child
// is abandoned on purpose: a re-entry run parked at a human gate must outlive
// the repair run that started it (the owner answers it later, in Review).
func reenter(ctx workflow.Context, workflowID, runID string, validated ValidatedPlan, terminal StepResult, state *runState) error {
	reentry := validated.Reentry
	short := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, RetryPolicy: boundedRetry(2*time.Second, 4),
	})
	switch reentry.Kind {
	case ReentrySingleRun:
		requestID := workflowID + "-reentry-" + shortRunID(runID)
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: requestID, ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_ABANDON,
		})
		child := workflow.ExecuteChildWorkflow(childCtx, proffer.ProfferWorkflow, proffer.WorkflowInput{
			RequestID: requestID, MatterID: reentry.MatterID, CourtCaseID: reentry.CourtCaseID,
			SourceRef: proffer.Ref(terminal.OutputRef), DeclaredFormat: reentry.DeclaredFormat,
			ParserOptionsRef: proffer.Ref(reentry.ParserOptionsRef),
		})
		var execution workflow.Execution
		if err := child.GetChildWorkflowExecution().Get(ctx, &execution); err != nil {
			return fmt.Errorf("start the re-entry run: %w", err)
		}
		if err := awaitReentryRegistration(ctx, short, child, execution.ID); err != nil {
			return err
		}
		var bound struct {
			PreviewHandle string `json:"preview_handle"`
		}
		if err := workflow.ExecuteActivity(short, bindImportOperationActivityName, map[string]any{
			"request_id": requestID, "source_ref": terminal.OutputRef,
			"workflow_id": execution.ID, "run_id": execution.RunID,
			"parser_options_ref": reentry.ParserOptionsRef,
		}).Get(ctx, &bound); err != nil || bound.PreviewHandle == "" {
			if err == nil {
				err = errors.New("no preview handle was returned")
			}
			return fmt.Errorf("the re-entry run %s started but its Review binding failed: %w", execution.ID, err)
		}
		state.status.ReentryPreviewHandle = bound.PreviewHandle
		return nil
	case ReentryBatch:
		scheme, bucket, prefix, err := splitFolderRef(terminal.ReentryRef)
		if err != nil {
			return err
		}
		digest := sha256.Sum256([]byte(workflowID + "/" + runID))
		batchID := "repair-" + hex.EncodeToString(digest[:])[:40]
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: batchID, ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_ABANDON,
		})
		child := workflow.ExecuteChildWorkflow(childCtx, proffer.BatchWorkflowName, proffer.BatchInput{
			BatchID: batchID, MatterID: reentry.MatterID, CourtCaseID: reentry.CourtCaseID,
			Scheme: scheme, Bucket: bucket, Prefix: prefix,
			DeclaredFormat: reentry.DeclaredFormat, ParserOptionsRef: proffer.Ref(reentry.ParserOptionsRef),
			MaxInFlight: 1,
		})
		var execution workflow.Execution
		if err := child.GetChildWorkflowExecution().Get(ctx, &execution); err != nil {
			return fmt.Errorf("start the re-entry batch: %w", err)
		}
		state.status.ReentryBatchID = batchID
		return nil
	}
	return fmt.Errorf("unknown re-entry kind %q", reentry.Kind)
}

// awaitReentryRegistration holds the Review binding until the re-entry run
// has registered its source. Only then is the run's matter provable from
// durable state, and the Workbench refuses its whole Review list over one
// binding whose TEST/REAL ownership it cannot prove. A run that ends before
// registering is reported and never bound.
func awaitReentryRegistration(ctx, short workflow.Context, child workflow.ChildWorkflowFuture, runWorkflowID string) error {
	deadline := workflow.Now(ctx).Add(reentryRegisterLimit)
	for {
		var state struct {
			Lifecycle        string `json:"lifecycle"`
			Terminal         bool   `json:"terminal"`
			Reason           string `json:"reason,omitempty"`
			Available        bool   `json:"available"`
			SourceVersionRef string `json:"source_version_ref,omitempty"`
		}
		err := workflow.ExecuteActivity(short, readImportOperationActivityName,
			map[string]any{"workflow_id": runWorkflowID}).Get(ctx, &state)
		if err == nil && state.Available {
			if state.SourceVersionRef != "" {
				return nil
			}
			if state.Terminal {
				return fmt.Errorf("the re-entry run %s ended (%s) before registering its source: %s", runWorkflowID, state.Lifecycle, state.Reason)
			}
		}
		if workflow.Now(ctx).After(deadline) {
			return fmt.Errorf("the re-entry run %s did not register its source within %s, so it was not bound to Review", runWorkflowID, reentryRegisterLimit)
		}
		finished := false
		var childErr error
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(child, func(future workflow.Future) {
			finished = true
			childErr = future.Get(ctx, nil)
		})
		selector.AddFuture(workflow.NewTimer(ctx, reentryRegisterPoll), func(workflow.Future) {})
		selector.Select(ctx)
		if finished {
			if childErr != nil {
				return fmt.Errorf("the re-entry run %s failed before it was bound to Review: %w", runWorkflowID, childErr)
			}
			// A run that completed has registered its source.
			return nil
		}
	}
}

func splitFolderRef(ref string) (scheme, bucket, prefix string, err error) {
	scheme, rest, found := strings.Cut(ref, "://")
	bucket, prefix, hasPrefix := strings.Cut(rest, "/")
	if !found || scheme == "" || bucket == "" || !hasPrefix || prefix == "" || !strings.HasSuffix(prefix, "/") {
		return "", "", "", fmt.Errorf("re-entry folder %q is not <scheme>://<bucket>/<prefix>/", ref)
	}
	return scheme, bucket, prefix, nil
}

func shortRunID(runID string) string {
	if len(runID) > 8 {
		return runID[:8]
	}
	return runID
}

// boundedReason keeps operator-facing reasons short enough for history and
// the receipt ledger.
func boundedReason(reason string) string {
	const limit = 1500
	if len(reason) > limit {
		return reason[:limit] + "…"
	}
	return reason
}
