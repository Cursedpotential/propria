// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package flow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// RegisterConversationWorkflows installs the conversation-level workflows under their exact names.
func RegisterConversationWorkflows(registrar WorkflowRegistrar) {
	registrar.RegisterWorkflowWithOptions(ExtractionRequestWorkflow, workflowOptions(RequestWorkflowName))
	registrar.RegisterWorkflowWithOptions(ExternalExtractionWorkflow, workflowOptions(ExternalWorkflowName))
	registrar.RegisterWorkflowWithOptions(SendToSurrealWorkflow, workflowOptions(SendWorkflowName))
}

// stepKey is the unique Step id of one (conversation, extractor) line.
func stepKey(label, extractor string) string { return label + "|" + extractor }

func (p *Progress) line(label, extractor, status, detail string, counts map[string]int) {
	key := stepKey(label, extractor)
	for i := range p.Steps {
		if p.Steps[i].Step == key {
			p.Steps[i].Status, p.Steps[i].Detail = status, detail
			if counts != nil {
				p.Steps[i].Counts = counts
			}
			return
		}
	}
	p.Steps = append(p.Steps, StepResult{Step: key, Status: status, Detail: detail, Counts: counts, Conversation: label, Extractor: extractor})
}

type requestItem struct {
	label string
	run   RunRef
}

// ExtractionRequestWorkflow runs the chosen extractors over the chosen conversations.
// It resolves each conversation to its runs, then for every run starts one child per
// extractor (the default extractor reuses EntityExtractionWorkflow unchanged, every
// other extractor is an ExternalExtractionWorkflow). A failing extractor is reported
// on its own line and never stops the others. A query ("status") reports progress.
func ExtractionRequestWorkflow(ctx workflow.Context, in RequestInput) (Progress, error) {
	progress := Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "resolve", Status: StepPending}}}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	if err := in.Validate(); err != nil {
		progress.set("resolve", StepFailed, err.Error(), nil)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	specs, _ := NormalizeExtractors(in.Extractors)

	var items []requestItem
	if len(in.Conversations) > 0 {
		progress.set("resolve", StepRunning, "", nil)
		var resolved ResolveResult
		if err := workflow.ExecuteActivity(shortOptions(ctx), ResolveConversationsActivity,
			ResolveRequest{MatterID: in.MatterID, Conversations: in.Conversations}).Get(ctx, &resolved); err != nil {
			progress.set("resolve", StepFailed, err.Error(), nil)
			progress.Outcome = OutcomeFailed
			return progress, nil
		}
		skipped := 0
		for _, target := range resolved.Targets {
			if len(target.Runs) == 0 {
				skipped++
				for _, spec := range specs {
					progress.line(target.Ref.Label(), spec.ID, StepSkipped, firstNonEmpty(target.Skipped, "the conversation has no imported messages yet"), nil)
				}
				continue
			}
			for _, run := range target.Runs {
				items = append(items, requestItem{label: target.Ref.Label(), run: run})
			}
		}
		progress.set("resolve", StepCompleted, fmt.Sprintf("%d conversation(s), %d run(s), %d without runs", len(resolved.Targets), len(items), skipped),
			map[string]int{"conversations": len(resolved.Targets), "runs": len(items), "skipped": skipped})
	} else {
		progress.set("resolve", StepSkipped, "runs were given directly", nil)
	}
	for _, run := range in.Runs {
		items = append(items, requestItem{label: "run " + shortHandle(run.PreviewHandle), run: run})
	}
	// Several runs of one conversation share a label: number them so each line is unique.
	seen := map[string]int{}
	for i := range items {
		seen[items[i].label]++
		if seen[items[i].label] > 1 {
			items[i].label = fmt.Sprintf("%s #%d", items[i].label, seen[items[i].label])
		}
		for _, spec := range specs {
			progress.line(items[i].label, spec.ID, StepPending, "", nil)
		}
	}

	failed, flagged, completed := 0, false, 0
	for _, item := range items {
		futures := make([]workflow.Future, len(specs))
		for i, spec := range specs {
			progress.line(item.label, spec.ID, StepRunning, "", nil)
			futures[i] = startExtractorChild(ctx, in, item.run, spec)
		}
		for i, spec := range specs {
			var child Progress
			if err := futures[i].Get(ctx, &child); err != nil {
				failed++
				progress.line(item.label, spec.ID, StepFailed, err.Error(), nil)
				continue
			}
			status, detail := childStatus(child)
			switch status {
			case StepFailed:
				failed++
			case StepSkipped:
				flagged = true
			default:
				completed++
			}
			if child.Outcome == OutcomeCompletedFlagged {
				flagged = true
			}
			progress.line(item.label, spec.ID, status, detail, sumCounts(child))
		}
	}
	switch {
	case failed > 0 && completed == 0:
		progress.Outcome = OutcomeFailed
	case failed > 0 || flagged:
		progress.Outcome = OutcomeCompletedFlagged
	default:
		progress.Outcome = OutcomeCompleted
	}
	return progress, nil
}

func startExtractorChild(ctx workflow.Context, in RequestInput, run RunRef, spec ExtractorSpec) workflow.Future {
	if spec.Kind == KindInternal {
		request := ExtractionRequest{
			ExtractionID: DeterministicID("conversation_extraction", in.RequestID, run.GenerationID),
			Run:          run, Actor: in.Actor, UseModel: true, RequestedAt: in.RequestedAt,
		}
		child := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:               ExtractionWorkflowID(request),
			WorkflowExecutionTimeout: 6 * time.Hour,
			WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		})
		return workflow.ExecuteChildWorkflow(child, EntityExtractionWorkflow, request)
	}
	input := ExternalRunInput{
		ExtractionID: DeterministicID("conversation_extraction", in.RequestID, run.GenerationID), Run: run, Extractor: spec.ID, Actor: in.Actor,
	}
	child := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               "external-extraction:" + input.ExtractionID + ":" + spec.ID,
		WorkflowExecutionTimeout: 6 * time.Hour,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	})
	return workflow.ExecuteChildWorkflow(child, ExternalWorkflowName, input)
}

func childStatus(child Progress) (string, string) {
	var details []string
	status := StepCompleted
	for _, step := range child.Steps {
		if step.Status == StepFailed {
			status = StepFailed
		}
		if step.Detail != "" && (step.Status == StepFailed || step.Status == StepSkipped || step.Status == StepCompleted) {
			details = append(details, step.Step+": "+step.Detail)
		}
	}
	if child.Outcome == OutcomeFailed {
		status = StepFailed
	}
	allSkipped := len(child.Steps) > 0
	for _, step := range child.Steps {
		if step.Status != StepSkipped {
			allSkipped = false
		}
	}
	if allSkipped {
		status = StepSkipped
	}
	return status, strings.Join(details, "; ")
}

func sumCounts(child Progress) map[string]int {
	total := map[string]int{}
	for _, step := range child.Steps {
		for key, value := range step.Counts {
			total[key] += value
		}
	}
	if len(total) == 0 {
		return nil
	}
	return total
}

func shortHandle(handle string) string {
	if len(handle) > 8 {
		return handle[:8]
	}
	return handle
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func externalActivityOptions(ctx workflow.Context, spec ExtractorSpec) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:              spec.TaskQueue,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    45 * time.Minute,
		HeartbeatTimeout:       10 * time.Minute,
		RetryPolicy:            &temporal.RetryPolicy{InitialInterval: 30 * time.Second, BackoffCoefficient: 2, MaximumAttempts: 2},
	})
}

// ExternalExtractionWorkflow runs one external extractor over one run, a window of messages at a time.
// Each window is one external Activity call (Python, evidence-pipeline queue) followed by one Go Activity
// that validates the reply against the default extractor's schema, grounds it in the window's own
// messages and stages it under a compare-only working.extraction_run tagged with the extractor.
func ExternalExtractionWorkflow(ctx workflow.Context, in ExternalRunInput) (Progress, error) {
	progress := Progress{Outcome: OutcomeRunning, Steps: []StepResult{{Step: "extract", Status: StepPending}}}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	spec, ok := ExtractorByID(in.Extractor)
	if !ok || spec.Kind != KindExternal {
		progress.set("extract", StepFailed, fmt.Sprintf("%q is not an external extractor", in.Extractor), nil)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	runID := in.ExternalRunID()
	var started ExternalRunStart
	started = ExternalRunStart{Input: in, Spec: spec}
	if err := workflow.ExecuteActivity(shortOptions(ctx), BeginExternalRunActivity, started).Get(ctx, &started); err != nil {
		progress.set("extract", StepFailed, err.Error(), nil)
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	progress.set("extract", StepRunning, "", nil)
	finish := func(status, errText string, stats map[string]any) {
		_ = workflow.ExecuteActivity(shortOptions(ctx), FinishExternalRunActivity,
			FinishExternalRun{RunID: runID, Status: status, Error: errText, Stats: stats}).Get(ctx, nil)
	}

	totals := map[string]int{"messages": 0, "entities": 0, "events": 0, "ungrounded": 0, "invalid_windows": 0, "windows": 0}
	after := int64(-1)
	for totals["messages"] < MaxExternalMessages {
		request := ExternalPageRequest{
			GenerationID: in.Run.GenerationID, SourceVersionID: in.Run.SourceVersionID, PreviewHandle: in.Run.PreviewHandle,
			Extractor: spec.ID, AfterOrdinal: after, Limit: ExternalPageMessages,
		}
		var raw json.RawMessage
		if err := workflow.ExecuteActivity(externalActivityOptions(ctx, spec), spec.Activity, request).Get(ctx, &raw); err != nil {
			finish("failed", err.Error(), statsOf(totals))
			progress.set("extract", StepFailed, err.Error(), totals)
			progress.Outcome = OutcomeFailed
			return progress, nil
		}
		var staged StagePageResult
		if err := workflow.ExecuteActivity(shortOptions(ctx), StageExternalPageActivity,
			StageExternalPage{Input: in, RunID: runID, Request: request, Page: raw}).Get(ctx, &staged); err != nil {
			finish("failed", err.Error(), statsOf(totals))
			progress.set("extract", StepFailed, err.Error(), totals)
			progress.Outcome = OutcomeFailed
			return progress, nil
		}
		if staged.Skipped {
			finish("failed", staged.Reason, statsOf(totals))
			progress.set("extract", StepSkipped, staged.Reason, nil)
			progress.Outcome = OutcomeCompletedFlagged
			return progress, nil
		}
		totals["windows"]++
		totals["messages"] += staged.Messages
		totals["entities"] += staged.Entities
		totals["events"] += staged.Events
		totals["ungrounded"] += staged.Ungrounded
		if staged.Invalid {
			totals["invalid_windows"]++
		}
		progress.set("extract", StepRunning, fmt.Sprintf("%d messages read", totals["messages"]), totals)
		if staged.Done || staged.Messages == 0 {
			break
		}
		after = staged.Last
	}
	finish("completed", "", statsOf(totals))
	detail := fmt.Sprintf("%d entities and %d events from %d messages", totals["entities"], totals["events"], totals["messages"])
	progress.set("extract", StepCompleted, detail, totals)
	progress.Outcome = OutcomeCompleted
	if totals["invalid_windows"] > 0 {
		progress.Outcome = OutcomeCompletedFlagged
		progress.Steps[0].Flags = append(progress.Steps[0].Flags, flagInvalidWindows(totals["invalid_windows"]))
	}
	return progress, nil
}

func flagInvalidWindows(count int) entities.Flag {
	return entities.Flag{Code: "external_window_invalid", Detail: fmt.Sprintf("%d window(s) of messages came back malformed and contributed nothing", count)}
}

func statsOf(totals map[string]int) map[string]any {
	out := make(map[string]any, len(totals))
	for key, value := range totals {
		out[key] = value
	}
	return out
}

// SendToSurrealWorkflow sends whole conversations to surreal-case, one conversation at a time.
// Per conversation it plans (counts what PostgreSQL holds), upserts the thread and its
// messages under deterministic ids (a resend never duplicates), upserts the extractions that
// exist, and reads the Surreal side back to compare counts. The result carries one receipt
// per conversation; a conversation that fails is reported and the others still go.
func SendToSurrealWorkflow(ctx workflow.Context, in SendInput) (Progress, error) {
	progress := Progress{Outcome: OutcomeRunning}
	if err := workflow.SetQueryHandler(ctx, StatusQuery, func() (Progress, error) { return progress, nil }); err != nil {
		return progress, err
	}
	if err := in.Validate(); err != nil {
		progress.Steps = append(progress.Steps, StepResult{Step: "validate", Status: StepFailed, Detail: err.Error()})
		progress.Outcome = OutcomeFailed
		return progress, nil
	}
	for _, ref := range in.Conversations {
		progress.line(ref.Label(), "surreal", StepPending, "", nil)
	}
	options := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Hour,
		HeartbeatTimeout:    5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 10 * time.Second, BackoffCoefficient: 2, MaximumAttempts: 3,
			NonRetryableErrorTypes: []string{SurrealNotConfiguredErrorType},
		},
	})
	failed := 0
	for _, ref := range in.Conversations {
		label := ref.Label()
		target := SendTarget{MatterID: in.MatterID, Ref: ref, Request: in.RequestID, Actor: in.Actor}
		fail := func(step string, err error) {
			failed++
			progress.line(label, "surreal", StepFailed, step+": "+err.Error(), nil)
		}
		progress.line(label, "surreal", StepRunning, "planning", nil)
		var plan SendPlan
		if err := workflow.ExecuteActivity(options, PlanSurrealSendActivity, target).Get(ctx, &plan); err != nil {
			fail("plan", err)
			continue
		}
		receipt := SendReceipt{Conversation: ref, ThreadID: plan.ThreadID, Plan: plan}
		progress.line(label, "surreal", StepRunning, fmt.Sprintf("sending %d messages", plan.Messages), nil)
		var written SendWritten
		if err := workflow.ExecuteActivity(options, UpsertConversationActivity, target).Get(ctx, &written); err != nil {
			fail("upsert", err)
			continue
		}
		receipt.Written = written
		if in.IncludeExtractions {
			var extracted SendWritten
			if err := workflow.ExecuteActivity(options, UpsertExtractionsActivity, target).Get(ctx, &extracted); err != nil {
				fail("extractions", err)
				continue
			}
			receipt.Written.Entities, receipt.Written.Events, receipt.Written.Runs = extracted.Entities, extracted.Events, extracted.Runs
		}
		var verified SendVerified
		if err := workflow.ExecuteActivity(options, VerifySurrealSendActivity, VerifyRequest{Target: target, Plan: plan, Written: receipt.Written}).Get(ctx, &verified); err != nil {
			fail("verify", err)
			continue
		}
		receipt.Verified = verified
		receipt.SentAt = workflow.Now(ctx).UTC()
		progress.Receipts = append(progress.Receipts, receipt)
		status := StepCompleted
		detail := fmt.Sprintf("%d messages, %d entities, %d events sent and read back", verified.Messages, verified.Entities, verified.Events)
		if !verified.Match {
			status = StepFailed
			failed++
			detail = fmt.Sprintf("read-back mismatch: planned %d messages, found %d", plan.Messages, verified.Messages)
		}
		progress.line(label, "surreal", status, detail, map[string]int{
			"messages": verified.Messages, "entities": verified.Entities, "events": verified.Events,
		})
	}
	switch {
	case failed == len(in.Conversations):
		progress.Outcome = OutcomeFailed
	case failed > 0:
		progress.Outcome = OutcomeCompletedFlagged
	default:
		progress.Outcome = OutcomeCompleted
	}
	return progress, nil
}

// VerifyRequest is the input of verify_surreal_send_activity.
type VerifyRequest struct {
	Target  SendTarget  `json:"target"`
	Plan    SendPlan    `json:"plan"`
	Written SendWritten `json:"written"`
}

// SurrealNotConfiguredErrorType is the application error type for a missing Surreal connection (never retried).
const SurrealNotConfiguredErrorType = "SurrealNotConfigured"
