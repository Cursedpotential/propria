// Byline: Claude Code · Opus 5 · 2026-09-21
//
// Batch-by-folder workflow tests. The Temporal test environment with mocked
// Activities and a mocked child workflow — a unit test, not evidence that a
// deployed worker imports a real B2 folder.

package proffer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func batchInput() BatchInput {
	return BatchInput{
		BatchID:     "batch-0000000000000000000000000000000001",
		MatterID:    "11111111-1111-1111-1111-111111111111",
		CourtCaseID: "22222222-2222-2222-2222-222222222222",
		Scheme:      "b2", Bucket: "bucket", Prefix: "vault/v1/sms /",
		DeclaredFormat: "smsbackuprestore_xml", ParserOptionsRef: Ref("options-1"),
	}
}

// registerBatchActivityStubs registers the four batch Activities under their
// exact names with signatures the workflow's map payloads decode into.
func registerBatchActivityStubs(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(
		func(context.Context, map[string]any) (map[string]any, error) { return nil, nil },
		activityOptions(listBatchFolderActivityName))
	env.RegisterActivityWithOptions(
		func(context.Context, map[string]any) (map[string]any, error) { return nil, nil },
		activityOptions(bindImportOperationActivityName))
	env.RegisterActivityWithOptions(
		func(context.Context, map[string]any) (map[string]any, error) { return nil, nil },
		activityOptions(readImportOperationActivityName))
	env.RegisterActivityWithOptions(
		func(context.Context, map[string]any) (map[string]any, error) { return nil, nil },
		activityOptions(findImportBindingsActivityName))
}

func mockEmptyPriorImports(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(findImportBindingsActivityName, mock.Anything, mock.Anything).
		Return(map[string]any{"bindings": []any{}}, nil)
}

func mockBindings(env *testsuite.TestWorkflowEnvironment) {
	env.OnActivity(bindImportOperationActivityName, mock.Anything, mock.Anything).
		Return(map[string]any{"preview_handle": "handle"}, nil)
}

func mockOneListingPage(env *testsuite.TestWorkflowEnvironment, keys []any) {
	env.OnActivity(listBatchFolderActivityName, mock.Anything, mock.Anything).
		Return(map[string]any{"keys": keys, "next_cursor": ""}, nil).Once()
}

func runBatch(t *testing.T, env *testsuite.TestWorkflowEnvironment, in BatchInput) BatchStatus {
	t.Helper()
	env.ExecuteWorkflow(BatchWorkflow, in)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var status BatchStatus
	require.NoError(t, env.GetWorkflowResult(&status))
	return status
}

// Every object under the folder becomes its own ordinary run, bound to a
// preview handle so it shows up in Review like a hand-started one.
func TestBatchWorkflowImportsEveryObjectInTheFolder(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	registerBatchActivityStubs(env)
	mockOneListingPage(env, []any{"vault/v1/sms /a.xml", "vault/v1/sms /b.xml", "vault/v1/sms /c.xml"})
	mockEmptyPriorImports(env)
	mockBindings(env)

	var started []string
	env.OnWorkflow(ProfferWorkflow, mock.Anything, mock.Anything).
		Return(func(_ workflow.Context, in WorkflowInput) (WorkflowResult, error) {
			started = append(started, string(in.SourceRef))
			return WorkflowResult{SourceVersionRef: "v", Status: StatusSuccess}, nil
		})

	status := runBatch(t, env, batchInput())

	require.Equal(t, 3, status.Counts.Total)
	require.Equal(t, 3, status.Counts.Done)
	require.Equal(t, 0, status.Counts.Failed)
	require.True(t, status.Terminal)
	require.Len(t, status.Items, 3)
	// The bounded in-flight count means items start in listing order.
	require.Equal(t, []string{
		"b2://bucket/vault/v1/sms /a.xml",
		"b2://bucket/vault/v1/sms /b.xml",
		"b2://bucket/vault/v1/sms /c.xml",
	}, started)
	for _, item := range status.Items {
		require.Equal(t, BatchItemDone, item.Status)
		require.Equal(t, "handle", item.PreviewHandle)
		require.NotEmpty(t, item.RequestID)
	}
}

// A failed item is recorded and the batch keeps going — the owner's 2 TB does
// not stop on one bad file.
func TestBatchWorkflowRecordsFailuresWithoutStopping(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	registerBatchActivityStubs(env)
	mockOneListingPage(env, []any{"vault/v1/sms /a.xml", "vault/v1/sms /bad.xml", "vault/v1/sms /c.xml"})
	mockEmptyPriorImports(env)
	mockBindings(env)

	env.OnWorkflow(ProfferWorkflow, mock.Anything, mock.Anything).
		Return(func(_ workflow.Context, in WorkflowInput) (WorkflowResult, error) {
			if in.SourceRef == "b2://bucket/vault/v1/sms /bad.xml" {
				return WorkflowResult{}, errors.New("decoder refused this source")
			}
			return WorkflowResult{SourceVersionRef: "v", Status: StatusSuccess}, nil
		})

	status := runBatch(t, env, batchInput())

	require.Equal(t, 3, status.Counts.Total)
	require.Equal(t, 2, status.Counts.Done)
	require.Equal(t, 1, status.Counts.Failed)
	require.True(t, status.Terminal)
	require.Equal(t, BatchItemFailed, status.Items[1].Status)
	require.Contains(t, status.Items[1].Reason, "decoder refused")
}

// Re-running the same batch must not import anything twice: an object whose
// earlier run completed is skipped.
func TestBatchWorkflowSkipsObjectsAlreadyCompleted(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(ProfferWorkflow)
	registerBatchActivityStubs(env)
	mockOneListingPage(env, []any{"vault/v1/sms /a.xml", "vault/v1/sms /b.xml"})
	mockBindings(env)

	env.OnActivity(findImportBindingsActivityName, mock.Anything, mock.Anything).
		Return(func(_ context.Context, req map[string]any) (map[string]any, error) {
			if req["source_ref"] == "b2://bucket/vault/v1/sms /a.xml" {
				return map[string]any{"bindings": []any{map[string]any{
					"preview_handle": "prior-handle", "request_id": "prior", "workflow_id": "prior-workflow",
				}}}, nil
			}
			return map[string]any{"bindings": []any{}}, nil
		})
	env.OnActivity(readImportOperationActivityName, mock.Anything, mock.Anything).
		Return(map[string]any{
			"lifecycle": string(OperationCompleted), "terminal": true, "available": true,
		}, nil)

	var started int
	env.OnWorkflow(ProfferWorkflow, mock.Anything, mock.Anything).
		Return(func(workflow.Context, WorkflowInput) (WorkflowResult, error) {
			started++
			return WorkflowResult{SourceVersionRef: "v", Status: StatusSuccess}, nil
		})

	status := runBatch(t, env, batchInput())

	require.Equal(t, 1, status.Counts.Skipped)
	require.Equal(t, 1, status.Counts.Done)
	require.Equal(t, 1, started, "an already-completed object must not be re-imported")
	require.Equal(t, BatchItemSkipped, status.Items[0].Status)
	require.Equal(t, "prior-handle", status.Items[0].PreviewHandle)
}

func TestBatchWorkflowRejectsAnIncompleteInput(t *testing.T) {
	for name, mutate := range map[string]func(*BatchInput){
		"no batch id":     func(in *BatchInput) { in.BatchID = "" },
		"no matter":       func(in *BatchInput) { in.MatterID = "" },
		"no format":       func(in *BatchInput) { in.DeclaredFormat = "" },
		"not a folder":    func(in *BatchInput) { in.Prefix = "vault/v1/a.xml" },
		"negative bound":  func(in *BatchInput) { in.MaxInFlight = -1 },
		"no parser refs":  func(in *BatchInput) { in.ParserOptionsRef = "" },
		"no court case":   func(in *BatchInput) { in.CourtCaseID = "" },
		"no bucket given": func(in *BatchInput) { in.Bucket = "" },
	} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		registerBatchActivityStubs(env)
		in := batchInput()
		mutate(&in)
		env.ExecuteWorkflow(BatchWorkflow, in)
		require.True(t, env.IsWorkflowCompleted(), name)
		require.Error(t, env.GetWorkflowError(), name)
	}
}

// activityOptions is a tiny helper so the stubs above read as one line each.
func activityOptions(name string) activity.RegisterOptions {
	return activity.RegisterOptions{Name: name}
}
