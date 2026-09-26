// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// RepairPlanWorkflow in the Temporal test environment, with the real
// validator standing in for the worker's validate Activity and mocked step
// Activities and child workflows. A unit test — not evidence that a deployed
// worker repaired a real object in B2.

package repairplan

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/Cursedpotential/probata/engine/proffer"
)

const testWorkflowID = "repair-plan-plan-0001-abcd-0123456789ab"

const (
	salvagedRef  = "b2://salem-data/consignatio/vault/v1/sms-20250617122400.xml.derived/salvaged/sms-20250617122400.xml"
	otherCopyRef = "b2://salem-data/consignatio/intake/raw-dedupe/v1/sms-20250617122400.xml"
	lenientRef   = "b2://salem-data/consignatio/vault/v1/sms-20250617122400.xml.derived/lenient/manifest.json"
	threadsRef   = "b2://salem-data/consignatio/vault/v1/sms-20250617122400.xml.derived/lenient/threads/"
	digestA      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type receiptLog struct {
	mu       sync.Mutex
	requests []ReceiptRequest
}

func (l *receiptLog) record(_ context.Context, request ReceiptRequest) (ReceiptResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = append(l.requests, request)
	return ReceiptResult{ReceiptRef: "receipt-" + request.StepID}, nil
}

func newRepairEnv(t *testing.T, plan Plan) (*testsuite.TestWorkflowEnvironment, *receiptLog) {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: testWorkflowID})
	env.RegisterWorkflow(proffer.ProfferWorkflow)
	env.RegisterWorkflowWithOptions(proffer.BatchWorkflow, workflow.RegisterOptions{Name: proffer.BatchWorkflowName})
	validation := testEnv(t)
	anchor := testAnchor()
	env.RegisterActivityWithOptions(func(_ context.Context, request ValidatePlanRequest) (ValidatedPlan, error) {
		return validation.ValidateAgainst(request.Plan, &anchor), nil
	}, activity.RegisterOptions{Name: ValidatePlanActivityName})
	stub := func(context.Context, StepRequest) (StepResult, error) {
		return StepResult{}, errors.New("unmocked step")
	}
	for _, name := range []string{findID, salvageID, lenientID} {
		env.RegisterActivityWithOptions(stub, activity.RegisterOptions{Name: name})
	}
	receipts := &receiptLog{}
	env.RegisterActivityWithOptions(receipts.record, activity.RegisterOptions{Name: RecordStepReceiptActivityName})
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{"preview_handle": "ReentryHandleReentryHandleReentryHandle01"}, nil
	}, activity.RegisterOptions{Name: bindImportOperationActivityName})
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		return nil, errors.New("unmocked flow")
	}, activity.RegisterOptions{Name: runFlowActivityName})
	registerOperationRead(env)
	return env, receipts
}

// registerOperationRead reports the re-entry run as having registered its
// source, which is what releases the Review binding.
func registerOperationRead(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{"lifecycle": "running", "available": true, "source_version_ref": "0199cccc-0000-7000-8000-000000000001"}, nil
	}, activity.RegisterOptions{Name: readImportOperationActivityName})
}

func salvageResult(req StepRequest) StepResult {
	return StepResult{OutputRef: salvagedRef, OutputType: req.SourceType, OutputKind: OutputDerivedObject,
		OutputSHA256: digestA, Summary: json.RawMessage(`{"records_kept":2,"bytes_dropped":1234}`)}
}

func runRepair(t *testing.T, env *testsuite.TestWorkflowEnvironment, plan Plan) (RunStatus, error) {
	t.Helper()
	env.ExecuteWorkflow(RepairPlanWorkflow, RunInput{Plan: plan})
	require.True(t, env.IsWorkflowCompleted())
	var status RunStatus
	encoded, err := env.QueryWorkflow(StatusQueryName)
	require.NoError(t, err)
	require.NoError(t, encoded.Get(&status))
	return status, env.GetWorkflowError()
}

func TestRepairPlanRunsStepsRecordsReceiptsAndReentersProffer(t *testing.T) {
	plan := testPlan(salvageID)
	env, receipts := newRepairEnv(t, plan)
	env.OnActivity(salvageID, mock.Anything, mock.Anything).Return(func(_ context.Context, req StepRequest) (StepResult, error) {
		require.Equal(t, testSource, req.SourceRef)
		require.Equal(t, TypeSMSBackupXML, req.SourceType)
		return salvageResult(req), nil
	}).Once()
	var child proffer.WorkflowInput
	env.OnWorkflow(proffer.ProfferWorkflow, mock.Anything, mock.Anything).
		Return(func(_ workflow.Context, in proffer.WorkflowInput) (proffer.WorkflowResult, error) {
			child = in
			return proffer.WorkflowResult{Status: proffer.StatusSuccess}, nil
		})

	status, err := runRepair(t, env, plan)
	require.NoError(t, err)
	require.Equal(t, RunCompleted, status.Status)
	require.Equal(t, "ReentryHandleReentryHandleReentryHandle01", status.ReentryPreviewHandle)
	require.Equal(t, plan.PlanID, status.PlanID)
	require.Equal(t, ModeTest, status.MatterMode)
	require.NotNil(t, status.PreviewHandle)
	require.Equal(t, testHandle, *status.PreviewHandle)
	require.Len(t, status.Steps, 1)
	require.Equal(t, StepStatus{StepID: "s1", Activity: salvageID, Status: StepSucceeded, ReceiptRef: "receipt-s1",
		OutputRef: salvagedRef, OutputSHA256: digestA, Summary: json.RawMessage(`{"records_kept":2,"bytes_dropped":1234}`)}, status.Steps[0])

	// The re-entry is an ordinary Proffer run on the derived copy, under the
	// Review run's matter, court case, declared format and parser options.
	require.Equal(t, proffer.Ref(salvagedRef), child.SourceRef)
	require.Equal(t, testMatter, child.MatterID)
	require.Equal(t, testCourt, child.CourtCaseID)
	require.Equal(t, "smsbackuprestore_xml", child.DeclaredFormat)
	require.Equal(t, proffer.Ref("pending-handler-selection/v1"), child.ParserOptionsRef)
	require.True(t, strings.HasPrefix(child.RequestID, testWorkflowID+"-reentry-"))

	// One receipt per step, then the re-entry's own receipt with the link.
	require.Len(t, receipts.requests, 2)
	receipt := receipts.requests[0]
	require.Equal(t, ReceiptSuccess, receipt.Status)
	require.Equal(t, testAnchor().SourceVersionID, receipt.SourceVersionID)
	require.Equal(t, testSource, receipt.InputRef)
	require.NotNil(t, receipt.Result)
	require.Equal(t, salvagedRef, receipt.Result.OutputRef)

	reentryReceipt := receipts.requests[1]
	require.Equal(t, ReentryReceiptActivity, reentryReceipt.Activity)
	require.Equal(t, reentryStepID, reentryReceipt.StepID)
	require.Equal(t, 1, reentryReceipt.StepIndex)
	require.Equal(t, ReceiptSuccess, reentryReceipt.Status)
	require.Equal(t, salvagedRef, reentryReceipt.InputRef)
	require.Equal(t, child.RequestID, reentryReceipt.Result.OutputRef)
	var link ReentryLink
	require.NoError(t, json.Unmarshal(reentryReceipt.Result.Summary, &link))
	require.Equal(t, ReentryLink{
		Link: supersededByLink, FromPreviewHandle: testHandle, FromWorkflowID: "req-1",
		ToPreviewHandle: "ReentryHandleReentryHandleReentryHandle01", ToWorkflowID: child.RequestID,
		SourceRef: salvagedRef, GateClosed: false, GateNote: gateLeftOpenNote,
	}, link)
	require.Equal(t, "receipt-"+reentryStepID, status.ReentryReceiptRef)
}

// A re-entry run that ends before registering its source is never bound to
// Review: an unprovable binding would make the Workbench refuse the whole
// Review list (owner report 2026-09-25).
func TestAReentryRunThatNeverRegistersIsNotBoundToReview(t *testing.T) {
	plan := testPlan(salvageID)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: testWorkflowID})
	validation, anchor := testEnv(t), testAnchor()
	env.RegisterActivityWithOptions(func(_ context.Context, request ValidatePlanRequest) (ValidatedPlan, error) {
		return validation.ValidateAgainst(request.Plan, &anchor), nil
	}, activity.RegisterOptions{Name: ValidatePlanActivityName})
	env.RegisterActivityWithOptions(func(_ context.Context, req StepRequest) (StepResult, error) { return salvageResult(req), nil },
		activity.RegisterOptions{Name: salvageID})
	receipts := &receiptLog{}
	env.RegisterActivityWithOptions(receipts.record, activity.RegisterOptions{Name: RecordStepReceiptActivityName})
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{"lifecycle": "failed", "terminal": true, "available": true, "reason": "acquisition refused the source"}, nil
	}, activity.RegisterOptions{Name: readImportOperationActivityName})
	bound := false
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		bound = true
		return map[string]any{"preview_handle": "x"}, nil
	}, activity.RegisterOptions{Name: bindImportOperationActivityName})
	env.RegisterWorkflow(proffer.ProfferWorkflow)
	env.OnWorkflow(proffer.ProfferWorkflow, mock.Anything, mock.Anything).
		Return(proffer.WorkflowResult{}, errors.New("acquisition refused the source"))

	status, err := runRepair(t, env, plan)
	require.Error(t, err)
	require.False(t, bound, "an unregistered re-entry run must never be bound to Review")
	require.Equal(t, RunFailed, status.Status)
	require.Contains(t, status.Reason, "before registering its source")
	require.Equal(t, StepSucceeded, status.Steps[0].Status, "the salvage itself succeeded and keeps its receipt")
	require.Empty(t, status.ReentryPreviewHandle)
	// The failed re-entry is receipted too, with the link as far as it got.
	require.Len(t, receipts.requests, 2)
	require.Equal(t, ReceiptFailed, receipts.requests[1].Status)
	require.Equal(t, ReentryReceiptActivity, receipts.requests[1].Activity)
	require.Contains(t, receipts.requests[1].Error, "before registering its source")
	require.Contains(t, receipts.requests[1].Error, `"to_workflow_id":"`+testWorkflowID+`-reentry-`)
	require.Equal(t, "receipt-"+reentryStepID, status.ReentryReceiptRef)
}

func TestFindRepointsTheSourceForTheFollowingSteps(t *testing.T) {
	plan := testPlan(findID, salvageID)
	env, receipts := newRepairEnv(t, plan)
	env.OnActivity(findID, mock.Anything, mock.Anything).Return(func(_ context.Context, req StepRequest) (StepResult, error) {
		return StepResult{OutputRef: otherCopyRef, OutputType: req.SourceType, OutputKind: OutputExistingObject,
			Summary: json.RawMessage(`{"candidates":1}`)}, nil
	}).Once()
	env.OnActivity(salvageID, mock.Anything, mock.Anything).Return(func(_ context.Context, req StepRequest) (StepResult, error) {
		require.Equal(t, otherCopyRef, req.SourceRef, "salvage must read the re-pointed copy")
		return salvageResult(req), nil
	}).Once()
	env.OnWorkflow(proffer.ProfferWorkflow, mock.Anything, mock.Anything).Return(proffer.WorkflowResult{Status: proffer.StatusSuccess}, nil)

	status, err := runRepair(t, env, plan)
	require.NoError(t, err)
	require.Equal(t, RunCompleted, status.Status)
	require.Len(t, receipts.requests, 3, "two step receipts and the re-entry receipt")
	require.Equal(t, otherCopyRef, receipts.requests[1].InputRef)
	require.Equal(t, ReentryReceiptActivity, receipts.requests[2].Activity)
}

func TestLenientDecodeReentersAsOneBoundedBatch(t *testing.T) {
	plan := testPlan(lenientID)
	env, receipts := newRepairEnv(t, plan)
	env.OnActivity(lenientID, mock.Anything, mock.Anything).Return(StepResult{
		OutputRef: lenientRef, OutputType: TypeDerivedThreads, OutputKind: OutputDerivedChunkFolder,
		OutputSHA256: digestA, ReentryRef: threadsRef,
	}, nil).Once()
	var batch proffer.BatchInput
	env.OnWorkflow(proffer.BatchWorkflowName, mock.Anything, mock.Anything).
		Return(func(_ workflow.Context, in proffer.BatchInput) (proffer.BatchStatus, error) {
			batch = in
			return proffer.BatchStatus{BatchID: in.BatchID, Terminal: true, Items: []proffer.BatchItem{}}, nil
		})

	status, err := runRepair(t, env, plan)
	require.NoError(t, err)
	require.Equal(t, RunCompleted, status.Status)
	require.Empty(t, status.ReentryPreviewHandle)
	require.Regexp(t, `^[A-Za-z0-9_-]{32,128}$`, status.ReentryBatchID, "the Workbench batch route must accept the id")
	require.Equal(t, status.ReentryBatchID, batch.BatchID)
	require.Equal(t, "b2", batch.Scheme)
	require.Equal(t, "salem-data", batch.Bucket)
	require.Equal(t, "consignatio/vault/v1/sms-20250617122400.xml.derived/lenient/threads/", batch.Prefix)
	require.Equal(t, DerivedThreadsDeclaredFormat, batch.DeclaredFormat)
	require.Equal(t, 1, batch.MaxInFlight)
	require.Equal(t, testMatter, batch.MatterID)

	// The link records the batch that superseded the Review run.
	require.Len(t, receipts.requests, 2)
	var link ReentryLink
	require.NoError(t, json.Unmarshal(receipts.requests[1].Result.Summary, &link))
	require.Equal(t, status.ReentryBatchID, link.ToBatchID)
	require.Equal(t, testHandle, link.FromPreviewHandle)
	require.Empty(t, link.ToPreviewHandle)
	require.Equal(t, "reentry_batch", receipts.requests[1].Result.OutputKind)
}

func TestAFailedStepIsReceiptedAndStopsThePlan(t *testing.T) {
	plan := testPlan(salvageID, lenientID)
	env, receipts := newRepairEnv(t, plan)
	env.OnActivity(salvageID, mock.Anything, mock.Anything).
		Return(StepResult{}, temporal.NewNonRetryableApplicationError("no complete record precedes the cut-off", "permanent_input_failure", nil)).Once()
	childStarted := false
	env.OnWorkflow(proffer.ProfferWorkflow, mock.Anything, mock.Anything).
		Return(func(workflow.Context, proffer.WorkflowInput) (proffer.WorkflowResult, error) {
			childStarted = true
			return proffer.WorkflowResult{}, nil
		})

	status, err := runRepair(t, env, plan)
	require.Error(t, err)
	require.Equal(t, RunFailed, status.Status)
	require.Equal(t, StepFailed, status.Steps[0].Status)
	require.Equal(t, "receipt-s1", status.Steps[0].ReceiptRef)
	require.Contains(t, status.Steps[0].Reason, "no complete record")
	require.Equal(t, StepSkipped, status.Steps[1].Status)
	require.Empty(t, status.Steps[1].ReceiptRef)
	require.Len(t, receipts.requests, 1)
	require.Equal(t, ReceiptFailed, receipts.requests[0].Status)
	require.Contains(t, receipts.requests[0].Error, "no complete record")
	require.False(t, childStarted, "a failed plan must never re-enter Proffer")
	require.Empty(t, status.ReentryPreviewHandle)
}

func TestTheWorkerRefusesAPlanItsOwnConfigurationRejects(t *testing.T) {
	plan := testPlan(salvageID)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: testWorkflowID})
	refused := ValidatedPlan{OK: false, Checks: []Check{{Rule: RuleDestination, Status: CheckFail, Reason: "outside every configured source root"}}}
	env.RegisterActivityWithOptions(func(context.Context, ValidatePlanRequest) (ValidatedPlan, error) { return refused, nil },
		activity.RegisterOptions{Name: ValidatePlanActivityName})
	stepRan := false
	env.RegisterActivityWithOptions(func(context.Context, StepRequest) (StepResult, error) { stepRan = true; return StepResult{}, nil },
		activity.RegisterOptions{Name: salvageID})

	status, err := runRepair(t, env, plan)
	require.Error(t, err)
	require.False(t, stepRan)
	require.Equal(t, RunFailed, status.Status)
	require.Len(t, status.Checks, 1)
	require.Contains(t, status.Reason, "outside every configured source root")
	require.Equal(t, StepSkipped, status.Steps[0].Status)
}

func TestAStepResultThatBreaksTheValidatedContractFailsThePlan(t *testing.T) {
	plan := testPlan(salvageID)
	env, receipts := newRepairEnv(t, plan)
	env.OnActivity(salvageID, mock.Anything, mock.Anything).Return(StepResult{
		OutputRef: salvagedRef, OutputType: TypeSMSBackupXML, OutputKind: OutputDerivedObject, // no sha256
	}, nil).Once()
	status, err := runRepair(t, env, plan)
	require.Error(t, err)
	require.Equal(t, StepFailed, status.Steps[0].Status)
	require.Contains(t, status.Steps[0].Reason, "without its sha256")
	require.Equal(t, ReceiptFailed, receipts.requests[0].Status)
}

func TestAStepWithoutAReceiptNeverCountsAsDone(t *testing.T) {
	plan := testPlan(salvageID)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	validation, anchor := testEnv(t), testAnchor()
	env.RegisterActivityWithOptions(func(_ context.Context, request ValidatePlanRequest) (ValidatedPlan, error) {
		return validation.ValidateAgainst(request.Plan, &anchor), nil
	}, activity.RegisterOptions{Name: ValidatePlanActivityName})
	env.RegisterActivityWithOptions(func(_ context.Context, req StepRequest) (StepResult, error) { return salvageResult(req), nil },
		activity.RegisterOptions{Name: salvageID})
	env.RegisterActivityWithOptions(func(context.Context, ReceiptRequest) (ReceiptResult, error) {
		return ReceiptResult{}, temporal.NewNonRetryableApplicationError("receipt store unavailable", "x", nil)
	}, activity.RegisterOptions{Name: RecordStepReceiptActivityName})
	env.RegisterWorkflow(proffer.ProfferWorkflow)

	status, err := runRepair(t, env, plan)
	require.Error(t, err)
	require.Equal(t, StepFailed, status.Steps[0].Status)
	require.Contains(t, status.Steps[0].Reason, "receipt could not be recorded")
	require.Empty(t, status.ReentryPreviewHandle)
}

// A step whose tool is an n8n flow runs through run_n8n_flow_activity with the
// source as a named reference and the params as inputs. No first-increment
// tool needs n8n; this proves the branch is wired.
func TestAnN8NStepRunsThroughTheFlowActivity(t *testing.T) {
	plan := testPlan(salvageID)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: testWorkflowID})
	validation, anchor := testEnv(t), testAnchor()
	env.RegisterActivityWithOptions(func(_ context.Context, request ValidatePlanRequest) (ValidatedPlan, error) {
		validated := validation.ValidateAgainst(request.Plan, &anchor)
		validated.Steps[0].NeedsN8N, validated.Steps[0].FlowName = true, "xml_salvage_flow"
		return validated, nil
	}, activity.RegisterOptions{Name: ValidatePlanActivityName})
	var flow map[string]any
	env.RegisterActivityWithOptions(func(_ context.Context, request map[string]any) (map[string]any, error) {
		flow = request
		return map[string]any{"flow": "xml_salvage_flow", "status": "success", "ref": salvagedRef,
			"outputs": map[string]any{"output_sha256": digestA, "records_kept": 7}}, nil
	}, activity.RegisterOptions{Name: runFlowActivityName})
	receipts := &receiptLog{}
	env.RegisterActivityWithOptions(receipts.record, activity.RegisterOptions{Name: RecordStepReceiptActivityName})
	env.RegisterActivityWithOptions(func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{"preview_handle": "ReentryHandleReentryHandleReentryHandle01"}, nil
	}, activity.RegisterOptions{Name: bindImportOperationActivityName})
	registerOperationRead(env)
	env.RegisterWorkflow(proffer.ProfferWorkflow)
	env.OnWorkflow(proffer.ProfferWorkflow, mock.Anything, mock.Anything).Return(proffer.WorkflowResult{}, nil)

	status, err := runRepair(t, env, plan)
	require.NoError(t, err)
	require.Equal(t, RunCompleted, status.Status)
	require.Equal(t, "xml_salvage_flow", flow["flow"])
	require.Equal(t, map[string]any{"source": testSource}, flow["refs"])
	require.Equal(t, testMatter, flow["matter_id"])
	require.Equal(t, salvagedRef, status.Steps[0].OutputRef)
	require.Equal(t, digestA, status.Steps[0].OutputSHA256)
}
