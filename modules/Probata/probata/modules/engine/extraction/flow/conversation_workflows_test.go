// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package flow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type activityLog struct {
	mu     sync.Mutex
	starts []string
	queues map[string]string
}

func (l *activityLog) record(info *activity.Info) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.starts = append(l.starts, info.ActivityType.Name)
	if l.queues == nil {
		l.queues = map[string]string{}
	}
	l.queues[info.ActivityType.Name] = info.TaskQueue
}

func (l *activityLog) count(name string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, started := range l.starts {
		if started == name {
			n++
		}
	}
	return n
}

func unmocked[In any, Out any](context.Context, In) (Out, error) {
	var zero Out
	return zero, errors.New("placeholder ran unmocked")
}

func requestEnv() (*testsuite.TestWorkflowEnvironment, *activityLog) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	RegisterWorkflows(env)
	RegisterConversationWorkflows(env)
	reg := func(fn any, name string) { env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name}) }
	reg(unmocked[ResolveRequest, ResolveResult], ResolveConversationsActivity)
	reg(unmocked[ExternalRunStart, ExternalRunStart], BeginExternalRunActivity)
	reg(unmocked[StageExternalPage, StagePageResult], StageExternalPageActivity)
	reg(func(context.Context, FinishExternalRun) error { return errors.New("placeholder ran unmocked") }, FinishExternalRunActivity)
	reg(unmocked[ExtractionRequest, ProposeResult], ProposeRulesActivity)
	reg(unmocked[ExtractionRequest, ProposeResult], ExtractModelActivity)
	reg(unmocked[ReconcileRequest, ReconcileResult], ReconcileActivity)
	reg(unmocked[ExternalPageRequest, json.RawMessage], SemanticaExtractActivity)
	reg(unmocked[ExternalPageRequest, json.RawMessage], LangExtractExtractActivity)
	reg(unmocked[SendTarget, SendPlan], PlanSurrealSendActivity)
	reg(unmocked[SendTarget, SendWritten], UpsertConversationActivity)
	reg(unmocked[SendTarget, SendWritten], UpsertExtractionsActivity)
	reg(unmocked[VerifyRequest, SendVerified], VerifySurrealSendActivity)
	log := &activityLog{}
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) { log.record(info) })
	return env, log
}

func twoRuns() ResolveResult {
	return ResolveResult{Targets: []ConversationTarget{{
		Ref: ConversationRef{ExportKey: "exports/sms-001.xml", Conv: "8105550101"},
		Runs: []RunRef{
			{MatterMode: "LIVE", PreviewHandle: "h1", GenerationID: "g1", SourceVersionID: "s1"},
			{MatterMode: "LIVE", PreviewHandle: "h2", GenerationID: "g2", SourceVersionID: "s2"},
		},
	}}}
}

func mockDefaultExtractorSucceeds(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(ProposeRulesActivity, mock.Anything, mock.Anything).Return(ProposeResult{ExtractionRunID: "r1", Proposals: 2, Messages: 10}, nil)
	env.OnActivity(ExtractModelActivity, mock.Anything, mock.Anything).Return(ProposeResult{ExtractionRunID: "r2", Proposals: 3, Events: 1, Batches: 1}, nil)
	env.OnActivity(ReconcileActivity, mock.Anything, mock.Anything).Return(ReconcileResult{ExtractionRunID: "r3", Proposals: 4}, nil)
}

func mockExternalRunsOnePage(env *testsuite.TestWorkflowEnvironment, name string) {
	env.OnActivity(BeginExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, start ExternalRunStart) (ExternalRunStart, error) { return start, nil })
	env.OnActivity(name, mock.Anything, mock.Anything).Return(json.RawMessage(`{"extractor":"x","extractor_version":"1","messages":5,"last_ordinal":9,"done":true}`), nil)
	env.OnActivity(StageExternalPageActivity, mock.Anything, mock.Anything).Return(StagePageResult{Entities: 3, Events: 1, Messages: 5, Last: 9, Done: true}, nil)
	env.OnActivity(FinishExternalRunActivity, mock.Anything, mock.Anything).Return(nil)
}

func request(extractors ...string) RequestInput {
	return RequestInput{OperatingMode: "LIVE", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		RequestID: "req-1", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba",
		Conversations: []ConversationRef{{ExportKey: "exports/sms-001.xml", Conv: "8105550101"}},
		Extractors:    extractors,
	}
}

// TestRequestWorkflowRunsEveryChosenExtractorOnEveryRun proves one click fans out over the conversation's runs,
// the default extractor reuses the existing three-step workflow, and an external extractor's Activity is scheduled
// on the Python queue and staged by the Go Activity.
func TestRequestWorkflowRunsEveryChosenExtractorOnEveryRun(t *testing.T) {
	env, log := requestEnv()
	env.OnActivity(ResolveConversationsActivity, mock.Anything, mock.Anything).Return(twoRuns(), nil).Once()
	mockDefaultExtractorSucceeds(env)
	mockExternalRunsOnePage(env, SemanticaExtractActivity)

	env.ExecuteWorkflow(RequestWorkflowName, request(DefaultExtractorID, "semantica"))
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("workflow error: %v", env.GetWorkflowError())
	}
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCompleted {
		t.Fatalf("outcome = %s, steps = %+v", progress.Outcome, progress.Steps)
	}
	lines := 0
	for _, step := range progress.Steps {
		if step.Conversation == "" {
			continue
		}
		lines++
		if step.Status != StepCompleted {
			t.Errorf("%s = %s (%s)", step.Step, step.Status, step.Detail)
		}
	}
	if lines != 4 {
		t.Fatalf("progress lines = %d, want 2 runs x 2 extractors", lines)
	}
	if log.count(ProposeRulesActivity) != 2 || log.count(SemanticaExtractActivity) != 2 || log.count(StageExternalPageActivity) != 2 {
		t.Fatalf("activity starts = %v", log.starts)
	}
	if log.queues[SemanticaExtractActivity] != PythonTaskQueue {
		t.Fatalf("semantica ran on %q, want %q", log.queues[SemanticaExtractActivity], PythonTaskQueue)
	}
}

// TestOneFailingExtractorDoesNotStopTheOthers proves a failed external extractor is reported on its own line
// while the default extractor still completes.
func TestOneFailingExtractorDoesNotStopTheOthers(t *testing.T) {
	env, _ := requestEnv()
	env.OnActivity(ResolveConversationsActivity, mock.Anything, mock.Anything).Return(twoRuns(), nil).Once()
	mockDefaultExtractorSucceeds(env)
	env.OnActivity(BeginExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, start ExternalRunStart) (ExternalRunStart, error) { return start, nil })
	env.OnActivity(LangExtractExtractActivity, mock.Anything, mock.Anything).Return(nil, temporal.NewNonRetryableApplicationError("NIM refused", "Upstream", nil))
	env.OnActivity(FinishExternalRunActivity, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(RequestWorkflowName, request(DefaultExtractorID, "langextract"))
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	failed, done := 0, 0
	for _, step := range progress.Steps {
		switch {
		case step.Extractor == "langextract" && step.Status == StepFailed:
			failed++
		case step.Extractor == DefaultExtractorID && step.Status == StepCompleted:
			done++
		}
	}
	if failed != 2 || done != 2 {
		t.Fatalf("failed=%d done=%d steps=%+v", failed, done, progress.Steps)
	}
	if progress.Outcome != OutcomeCompletedFlagged {
		t.Fatalf("outcome = %s, want completed_with_flags", progress.Outcome)
	}
}

// TestRequestWorkflowReportsAConversationWithoutRuns proves a conversation that resolves to nothing is skipped with its reason.
func TestRequestWorkflowReportsAConversationWithoutRuns(t *testing.T) {
	env, _ := requestEnv()
	env.OnActivity(ResolveConversationsActivity, mock.Anything, mock.Anything).Return(ResolveResult{Targets: []ConversationTarget{{
		Ref: ConversationRef{ExportKey: "exports/a", Conv: "c"}, Skipped: "the conversation was not found in this matter",
	}}}, nil)
	env.ExecuteWorkflow(RequestWorkflowName, request(DefaultExtractorID))
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, step := range progress.Steps {
		if step.Extractor == DefaultExtractorID && step.Status == StepSkipped && strings.Contains(step.Detail, "not found") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no skipped line with the reason: %+v", progress.Steps)
	}
}

// TestRequestWorkflowRejectsAnUnknownExtractor proves a request naming an extractor that is not registered runs nothing.
func TestRequestWorkflowRejectsAnUnknownExtractor(t *testing.T) {
	env, log := requestEnv()
	env.ExecuteWorkflow(RequestWorkflowName, request("not-an-extractor"))
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeFailed || len(log.starts) != 0 {
		t.Fatalf("outcome = %s, started = %v", progress.Outcome, log.starts)
	}
}

// TestExternalWorkflowReadsWindowsUntilTheRunIsDone proves the window cursor moves to the last staged ordinal.
func TestExternalWorkflowReadsWindowsUntilTheRunIsDone(t *testing.T) {
	env, log := requestEnv()
	var mu sync.Mutex
	var afters []int64
	env.OnActivity(BeginExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, start ExternalRunStart) (ExternalRunStart, error) { return start, nil })
	env.OnActivity(SemanticaExtractActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, request ExternalPageRequest) (json.RawMessage, error) {
			mu.Lock()
			afters = append(afters, request.AfterOrdinal)
			mu.Unlock()
			return json.RawMessage(`{}`), nil
		})
	pages := []StagePageResult{{Messages: 200, Last: 199, Entities: 1}, {Messages: 200, Last: 399, Entities: 1}, {Messages: 7, Last: 406, Done: true}}
	next := 0
	env.OnActivity(StageExternalPageActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, _ StageExternalPage) (StagePageResult, error) {
			page := pages[next]
			next++
			return page, nil
		})
	var finish FinishExternalRun
	env.OnActivity(FinishExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, request FinishExternalRun) error { finish = request; return nil })

	env.ExecuteWorkflow(ExternalWorkflowName, ExternalRunInput{ExtractionID: "e1", Run: RunRef{MatterMode: "LIVE", PreviewHandle: "h", GenerationID: "g", SourceVersionID: "s"}, Extractor: "semantica"})
	if env.GetWorkflowError() != nil {
		t.Fatalf("workflow error: %v", env.GetWorkflowError())
	}
	if len(afters) != 3 || afters[0] != -1 || afters[1] != 199 || afters[2] != 399 {
		t.Fatalf("window cursors = %v, want -1, 199, 399", afters)
	}
	if finish.Status != "completed" || finish.Stats["messages"] != float64(407) || finish.Stats["entities"] != float64(2) {
		t.Fatalf("finish = %+v", finish)
	}
	if log.count(StageExternalPageActivity) != 3 {
		t.Fatalf("staged %d windows, want 3", log.count(StageExternalPageActivity))
	}
}

// TestExternalWorkflowReportsAnExtractorThatCannotRunAsSkipped proves a not-configured extractor leaves a failed run row and a skipped line, not a hang.
func TestExternalWorkflowReportsAnExtractorThatCannotRunAsSkipped(t *testing.T) {
	env, _ := requestEnv()
	env.OnActivity(BeginExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, start ExternalRunStart) (ExternalRunStart, error) { return start, nil })
	env.OnActivity(LangExtractExtractActivity, mock.Anything, mock.Anything).Return(json.RawMessage(`{"skipped":true,"reason":"no key"}`), nil)
	env.OnActivity(StageExternalPageActivity, mock.Anything, mock.Anything).Return(StagePageResult{Skipped: true, Reason: "no key"}, nil)
	var finish FinishExternalRun
	env.OnActivity(FinishExternalRunActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, request FinishExternalRun) error { finish = request; return nil })
	env.ExecuteWorkflow(ExternalWorkflowName, ExternalRunInput{ExtractionID: "e1", Run: RunRef{MatterMode: "LIVE", GenerationID: "g"}, Extractor: "langextract"})
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Steps[0].Status != StepSkipped || finish.Status != "failed" || finish.Error != "no key" {
		t.Fatalf("progress = %+v finish = %+v", progress.Steps, finish)
	}
}

func sendInput() SendInput {
	return SendInput{OperatingMode: "LIVE", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
		RequestID: "send-1", MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba", IncludeExtractions: true,
		Conversations: []ConversationRef{{ExportKey: "exports/a", Conv: "c1"}, {ExportKey: "exports/a", Conv: "c2"}},
	}
}

// TestSendWorkflowReturnsOneReceiptPerConversation proves each conversation is planned, upserted, extraction-upserted and read back.
func TestSendWorkflowReturnsOneReceiptPerConversation(t *testing.T) {
	env, _ := requestEnv()
	env.OnActivity(PlanSurrealSendActivity, mock.Anything, mock.Anything).Return(SendPlan{Messages: 12, ThreadID: "t"}, nil)
	env.OnActivity(UpsertConversationActivity, mock.Anything, mock.Anything).Return(SendWritten{Threads: 1, Messages: 12}, nil)
	env.OnActivity(UpsertExtractionsActivity, mock.Anything, mock.Anything).Return(SendWritten{Runs: 2, Entities: 3, Events: 1}, nil)
	env.OnActivity(VerifySurrealSendActivity, mock.Anything, mock.Anything).Return(SendVerified{Messages: 12, Entities: 3, Events: 1, Match: true}, nil)
	env.ExecuteWorkflow(SendWorkflowName, sendInput())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCompleted || len(progress.Receipts) != 2 {
		t.Fatalf("outcome = %s, receipts = %d", progress.Outcome, len(progress.Receipts))
	}
	receipt := progress.Receipts[0]
	if receipt.Written.Messages != 12 || receipt.Written.Entities != 3 || !receipt.Verified.Match || receipt.SentAt.IsZero() {
		t.Fatalf("receipt = %+v", receipt)
	}
}

// TestSendWorkflowFlagsAReadBackMismatchAndKeepsGoing proves a conversation whose Surreal count differs is reported, and the next one still goes.
func TestSendWorkflowFlagsAReadBackMismatchAndKeepsGoing(t *testing.T) {
	env, _ := requestEnv()
	env.OnActivity(PlanSurrealSendActivity, mock.Anything, mock.Anything).Return(SendPlan{Messages: 12, ThreadID: "t"}, nil)
	env.OnActivity(UpsertConversationActivity, mock.Anything, mock.Anything).Return(SendWritten{Threads: 1, Messages: 12}, nil)
	env.OnActivity(UpsertExtractionsActivity, mock.Anything, mock.Anything).Return(SendWritten{}, nil)
	calls := 0
	env.OnActivity(VerifySurrealSendActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, _ VerifyRequest) (SendVerified, error) {
			calls++
			if calls == 1 {
				return SendVerified{Messages: 10, Match: false}, nil
			}
			return SendVerified{Messages: 12, Match: true}, nil
		})
	env.ExecuteWorkflow(SendWorkflowName, sendInput())
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeCompletedFlagged || calls != 2 {
		t.Fatalf("outcome = %s, verify calls = %d", progress.Outcome, calls)
	}
}

// TestSendWorkflowDoesNotRetryAMissingSurrealConnection proves a not-configured sink fails once, with its reason.
func TestSendWorkflowDoesNotRetryAMissingSurrealConnection(t *testing.T) {
	env, log := requestEnv()
	env.OnActivity(PlanSurrealSendActivity, mock.Anything, mock.Anything).Return(SendPlan{},
		temporal.NewNonRetryableApplicationError("surreal-case is not configured", SurrealNotConfiguredErrorType, nil))
	in := sendInput()
	in.Conversations = in.Conversations[:1]
	env.ExecuteWorkflow(SendWorkflowName, in)
	var progress Progress
	if err := env.GetWorkflowResult(&progress); err != nil {
		t.Fatal(err)
	}
	if progress.Outcome != OutcomeFailed || log.count(PlanSurrealSendActivity) != 1 {
		t.Fatalf("outcome = %s, plan attempts = %d", progress.Outcome, log.count(PlanSurrealSendActivity))
	}
}

// TestExtractorRegistryNormalizesASelection proves the picker's contract: empty means the default, unknown ids fail, order is the registry's.
func TestExtractorRegistryNormalizesASelection(t *testing.T) {
	specs, err := NormalizeExtractors(nil)
	if err != nil || len(specs) != 1 || specs[0].ID != DefaultExtractorID {
		t.Fatalf("empty selection = %v, %v", specs, err)
	}
	specs, err = NormalizeExtractors([]string{"langextract", "semantica", "semantica"})
	if err != nil || len(specs) != 2 || specs[0].ID != "semantica" || specs[1].ID != "langextract" {
		t.Fatalf("selection = %v, %v", specs, err)
	}
	if _, err := NormalizeExtractors([]string{"nope"}); err == nil {
		t.Fatal("an unknown extractor was accepted")
	}
	defaults := 0
	for _, spec := range Registry {
		if spec.Default {
			defaults++
		}
		found := false
		for _, run := range spec.RunExtractors {
			found = found || run == spec.RunExtractor
		}
		if !found {
			t.Errorf("%s: RunExtractors %v must include the extractor new runs are written with (%s)", spec.ID, spec.RunExtractors, spec.RunExtractor)
		}
		if spec.Kind == KindExternal && (spec.Activity == "" || spec.TaskQueue == "" || !spec.CompareOnly) {
			t.Errorf("%s is external without an Activity, a queue or compare-only", spec.ID)
		}
	}
	if defaults != 1 {
		t.Fatalf("%d default extractors, want 1", defaults)
	}
}
