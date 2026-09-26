// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// The repair routes through PreviewHTTPHandler.Routes(): same tailnet +
// service-token authorization as every preview route, the shared contract's
// JSON shapes, and a `detail` string on every error. The real repairplan
// Service runs behind them with in-memory anchors and runs.

package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/repairplan"
)

const (
	repairTestHandle = "HandleHandleHandleHandleHandle0123456789_-"
	repairTestSource = "b2://salem-data/consignatio/vault/v1/sms-20250617122400.xml"
)

type repairAnchors struct{ err error }

func (a repairAnchors) ResolveAnchor(_ context.Context, sourceRef, handle string) (repairplan.Anchor, error) {
	if a.err != nil {
		return repairplan.Anchor{}, a.err
	}
	if handle != "" && handle != repairTestHandle || sourceRef == "" {
		return repairplan.Anchor{}, repairplan.ErrAnchorNotFound
	}
	return repairplan.Anchor{
		PreviewHandle: repairTestHandle, RequestID: "r", WorkflowID: "r", SourceRef: repairTestSource,
		SourceVersionID: "0199aaaa-0000-7000-8000-000000000001", DeclaredFormat: "smsbackuprestore_xml",
		ParserOptionsRef: "pending-handler-selection/v1",
		MatterID:         "deadbeef-dead-beef-dead-beefdeadbeef", CourtCaseID: "cafebabe-cafe-babe-cafe-babecafebabe",
		RepairReport: json.RawMessage(`{"clean":false,"truncated":true}`),
	}, nil
}

type repairRuns struct {
	startErr error
	started  []string
}

func (r *repairRuns) StartPlan(_ context.Context, workflowID string, _ repairplan.RunInput) (string, error) {
	if r.startErr != nil {
		return "", r.startErr
	}
	r.started = append(r.started, workflowID)
	return "run-1", nil
}

func (r *repairRuns) PlanStatus(_ context.Context, workflowID string) (repairplan.RunStatus, error) {
	if len(r.started) == 0 || workflowID != r.started[0] {
		return repairplan.RunStatus{}, repairplan.ErrRunNotFound
	}
	handle := repairTestHandle
	return repairplan.RunStatus{WorkflowID: workflowID, PlanID: "plan-0001-abcd", PreviewHandle: &handle,
		MatterMode: repairplan.ModeTest, Status: repairplan.RunRunning}, nil
}

func repairHandler(t *testing.T, anchors repairplan.AnchorResolver, runs *repairRuns) http.Handler {
	t.Helper()
	handler, _, _ := previewTestHandler(t)
	roots, err := objectstores.ParseRoots(`[{"id":"b2-bucket","label":"B2","url":"b2://salem-data/"}]`)
	require.NoError(t, err)
	require.NoError(t, handler.UseRepairPlans(repairplan.Service{
		Env: repairplan.Environment{
			Registry: repairplan.DefaultRegistry(), Anchors: anchors,
			Stores: objectstores.Stores{"b2": "/run/secrets/b2.json"}, SourceRoots: roots,
			MatterMode: func(matter, court string) (string, bool) {
				return repairplan.ModeTest, matter == "deadbeef-dead-beef-dead-beefdeadbeef"
			},
		},
		Runs: runs,
	}))
	return handler.Routes()
}

func planBody(activities ...string) []byte {
	steps := make([]map[string]any, 0, len(activities))
	for index, activity := range activities {
		steps = append(steps, map[string]any{"step_id": "s" + string(rune('1'+index)), "activity": activity, "params": map[string]any{}})
	}
	body, _ := json.Marshal(map[string]any{
		"plan_id": "plan-0001-abcd", "source_ref": repairTestSource, "preview_handle": repairTestHandle,
		"matter_mode": "TEST", "steps": steps,
	})
	return body
}

func detailOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Detail string `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body), recorder.Body.String())
	require.NotEmpty(t, body.Detail, recorder.Body.String())
	return body.Detail
}

func TestRepairRoutesAnswer503UntilConfiguredAndRequireAuthorization(t *testing.T) {
	handler, _, _ := previewTestHandler(t)
	unconfigured := servePreview(handler.Routes(), http.MethodGet, "/reference-import/repair/tools", nil)
	require.Equal(t, http.StatusServiceUnavailable, unconfigured.Code)
	detailOf(t, unconfigured)

	routes := repairHandler(t, repairAnchors{}, &repairRuns{})
	req := httptest.NewRequest(http.MethodGet, "/reference-import/repair/tools", nil)
	req.RemoteAddr = "100.64.1.9:3456" // tailnet peer, but no service token
	recorder := httptest.NewRecorder()
	routes.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestRepairToolsRoute(t *testing.T) {
	recorder := servePreview(repairHandler(t, repairAnchors{}, &repairRuns{}), http.MethodGet, "/reference-import/repair/tools", nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Tools []map[string]json.RawMessage `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Tools, 3)
	for _, tool := range body.Tools {
		for _, key := range []string{"id", "description", "input_types", "output_types", "params_schema", "writes", "needs_n8n"} {
			require.Contains(t, tool, key)
		}
	}
}

func TestRepairProposeRoute(t *testing.T) {
	routes := repairHandler(t, repairAnchors{}, &repairRuns{})
	ok := servePreview(routes, http.MethodPost, "/reference-import/repair/propose",
		[]byte(`{"source_ref":"`+repairTestSource+`","preview_handle":"`+repairTestHandle+`"}`))
	require.Equal(t, http.StatusOK, ok.Code, ok.Body.String())
	var body repairplan.ProposeResponse
	require.NoError(t, json.Unmarshal(ok.Body.Bytes(), &body))
	require.Equal(t, "sms_backup_xml:truncated", body.Signature)
	require.Len(t, body.Proposals, 4)
	require.False(t, body.AgentAvailable)
	require.Contains(t, ok.Body.String(), `"steps":[]`, "the wait option must serialise steps as [], never null")

	missing := servePreview(routes, http.MethodPost, "/reference-import/repair/propose",
		[]byte(`{"source_ref":"`+repairTestSource+`","preview_handle":"MissingMissingMissingMissing0123456789"}`))
	require.Equal(t, http.StatusNotFound, missing.Code)
	detailOf(t, missing)

	noRun := servePreview(repairHandler(t, repairAnchors{err: repairplan.ErrAnchorNotFound}, &repairRuns{}), http.MethodPost,
		"/reference-import/repair/propose", []byte(`{"source_ref":"`+repairTestSource+`"}`))
	require.Equal(t, http.StatusUnprocessableEntity, noRun.Code)
	require.Contains(t, detailOf(t, noRun), "start it in Review first")

	mismatch := servePreview(routes, http.MethodPost, "/reference-import/repair/propose",
		[]byte(`{"source_ref":"b2://salem-data/other.xml","preview_handle":"`+repairTestHandle+`"}`))
	require.Equal(t, http.StatusUnprocessableEntity, mismatch.Code)
	detailOf(t, mismatch)

	unknownField := servePreview(routes, http.MethodPost, "/reference-import/repair/propose", []byte(`{"source_ref":"x","extra":1}`))
	require.Equal(t, http.StatusBadRequest, unknownField.Code)
}

func TestRepairValidateRoute(t *testing.T) {
	routes := repairHandler(t, repairAnchors{}, &repairRuns{})
	ok := servePreview(routes, http.MethodPost, "/reference-import/repair/validate", planBody("repair.salvage_truncated_xml"))
	require.Equal(t, http.StatusOK, ok.Code, ok.Body.String())
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(ok.Body.Bytes(), &body))
	require.Equal(t, "true", string(body["ok"]))
	var checks []repairplan.Check
	require.NoError(t, json.Unmarshal(body["checks"], &checks))
	require.Len(t, checks, 9)
	for _, check := range checks {
		require.Equal(t, repairplan.CheckPass, check.Status, check.Rule+": "+check.Reason)
	}

	refused := servePreview(routes, http.MethodPost, "/reference-import/repair/validate", planBody("repair.lenient_decode", "repair.salvage_truncated_xml"))
	require.Equal(t, http.StatusOK, refused.Code)
	require.Contains(t, refused.Body.String(), `"ok":false`)

	for name, body := range map[string][]byte{
		"malformed json":   []byte(`{"plan_id":`),
		"bad matter mode":  []byte(strings.Replace(string(planBody("repair.salvage_truncated_xml")), `"TEST"`, `"LIVE"`, 1)),
		"an unknown field": []byte(strings.Replace(string(planBody("repair.salvage_truncated_xml")), `"matter_mode"`, `"extra":1,"matter_mode"`, 1)),
	} {
		bad := servePreview(routes, http.MethodPost, "/reference-import/repair/validate", body)
		require.Equal(t, http.StatusBadRequest, bad.Code, name)
		detailOf(t, bad)
	}

	outage := servePreview(repairHandler(t, repairAnchors{err: errors.New("database unavailable")}, &repairRuns{}), http.MethodPost,
		"/reference-import/repair/validate", planBody("repair.salvage_truncated_xml"))
	require.Equal(t, http.StatusServiceUnavailable, outage.Code)
	detailOf(t, outage)
}

func TestRepairRunAndStatusRoutes(t *testing.T) {
	runs := &repairRuns{}
	routes := repairHandler(t, repairAnchors{}, runs)

	refused := servePreview(routes, http.MethodPost, "/reference-import/repair/run", planBody("repair.lenient_decode", "repair.salvage_truncated_xml"))
	require.Equal(t, http.StatusUnprocessableEntity, refused.Code)
	require.Contains(t, detailOf(t, refused), "type_chain:")
	var refusedBody struct {
		Checks []repairplan.Check `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(refused.Body.Bytes(), &refusedBody))
	require.Len(t, refusedBody.Checks, 9)
	require.Empty(t, runs.started, "a refused plan must never start")

	started := servePreview(routes, http.MethodPost, "/reference-import/repair/run", planBody("repair.salvage_truncated_xml"))
	require.Equal(t, http.StatusCreated, started.Code, started.Body.String())
	var response repairplan.RunResponse
	require.NoError(t, json.Unmarshal(started.Body.Bytes(), &response))
	require.Equal(t, "run-1", response.RunID)
	require.True(t, strings.HasPrefix(response.WorkflowID, "repair-plan-plan-0001-abcd-"))

	status := servePreview(routes, http.MethodGet, "/reference-import/repair/runs/"+response.WorkflowID, nil)
	require.Equal(t, http.StatusOK, status.Code, status.Body.String())
	var statusBody map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(status.Body.Bytes(), &statusBody))
	for _, key := range []string{"workflow_id", "plan_id", "preview_handle", "matter_mode", "status", "steps"} {
		require.Contains(t, statusBody, key)
	}
	require.Equal(t, "[]", string(statusBody["steps"]))

	missing := servePreview(routes, http.MethodGet, "/reference-import/repair/runs/repair-plan-unknown-000000000000", nil)
	require.Equal(t, http.StatusNotFound, missing.Code)
	detailOf(t, missing)
	badID := servePreview(routes, http.MethodGet, "/reference-import/repair/runs/bad%20id", nil)
	require.Equal(t, http.StatusBadRequest, badID.Code)

	runs.startErr = errors.New("temporal unavailable")
	failed := servePreview(routes, http.MethodPost, "/reference-import/repair/run", planBody("repair.find_other_version"))
	require.Equal(t, http.StatusServiceUnavailable, failed.Code)
	detailOf(t, failed)
}
