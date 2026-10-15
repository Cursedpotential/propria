package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

type projectionListerRecorder struct {
	scope surrealsink.ApprovedProjectionScope
	page  surrealsink.ApprovedProjectionPage
	err   error
	calls int
}

// ListApprovedProjections records the HTTP handoff without inventing a graph result.
// Inputs: validated scope. Outputs: injected page or error. Effects: records one test call.
// Pick for request validation and status mapping tests.
func (f *projectionListerRecorder) ListApprovedProjections(_ context.Context, scope surrealsink.ApprovedProjectionScope) (surrealsink.ApprovedProjectionPage, error) {
	f.scope, f.calls = scope, f.calls+1
	return f.page, f.err
}

// TestApprovedProjectionGETBindsCaseAndReturnsEmptyItems verifies the exact authenticated contract.
// Inputs: canonical case and actor/service headers. Outputs: 200 with nonnil empty items.
// Effects: recorder call only; choose when discovery endpoint shape changes.
func TestApprovedProjectionGETBindsCaseAndReturnsEmptyItems(t *testing.T) {
	recorder := &projectionListerRecorder{}
	handler, err := NewApprovedGraphProjectionsHTTPHandler(recorder, serviceTokenPath(t))
	if err != nil {
		t.Fatal(err)
	}
	target := "/reference-import/analysis/projections?matter_id=" + caseidentity.AuthoritativeMatterID + "&court_case_id=" + caseidentity.AuthoritativeCourtCaseID + "&limit=50"
	response := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, target, nil))
	if response.Code != http.StatusOK || recorder.calls != 1 || recorder.scope.Limit != 50 || recorder.scope.MatterID != caseidentity.AuthoritativeMatterID {
		t.Fatalf("projection listing handoff failed: code=%d scope=%+v", response.Code, recorder.scope)
	}
	var body struct {
		Items   []surrealsink.ApprovedProjectionDescriptor `json:"items"`
		HasMore bool                                       `json:"has_more"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Items == nil || len(body.Items) != 0 || body.HasMore {
		t.Fatalf("empty projection listing shape differs: body=%s err=%v", response.Body.String(), err)
	}
}

// TestApprovedProjectionGETRejectsBadParamsActorAndBackendFailure preserves distinct HTTP errors.
// Inputs: unscoped, over-limit, cross-policy and unauthenticated requests plus backend failure.
// Outputs: 400/422/401/503. Effects: recorder calls only; choose when admission changes.
func TestApprovedProjectionGETRejectsBadParamsActorAndBackendFailure(t *testing.T) {
	recorder := &projectionListerRecorder{}
	handler, err := NewApprovedGraphProjectionsHTTPHandler(recorder, serviceTokenPath(t))
	if err != nil {
		t.Fatal(err)
	}
	base := "/reference-import/analysis/projections?matter_id=" + caseidentity.AuthoritativeMatterID + "&court_case_id=" + caseidentity.AuthoritativeCourtCaseID
	for target, want := range map[string]int{
		"/reference-import/analysis/projections":                  http.StatusUnprocessableEntity,
		base + "&limit=51":                                        http.StatusUnprocessableEntity,
		base + "&access_policy_id=client-policy":                  http.StatusBadRequest,
		base + "&matter_id=" + caseidentity.AuthoritativeMatterID: http.StatusBadRequest,
	} {
		if got := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, target, nil)).Code; got != want {
			t.Errorf("%s returned %d, want %d", target, got, want)
		}
	}
	request := newPreviewRequest(http.MethodGet, base, nil)
	request.Header.Del("X-authentik-uid")
	if got := servePreviewRequest(handler.Routes(), request).Code; got != http.StatusUnauthorized {
		t.Fatalf("missing actor returned %d", got)
	}
	if recorder.calls != 0 {
		t.Fatal("invalid request reached analytical lister")
	}
	recorder.err = surrealsink.ErrApprovedProjectionInput
	if got := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, base+"&cursor=bad", nil)).Code; got != http.StatusUnprocessableEntity {
		t.Fatalf("bad cursor returned %d", got)
	}
	recorder.err = errors.New("analysis schema unavailable")
	if got := servePreviewRequest(handler.Routes(), newPreviewRequest(http.MethodGet, base, nil)).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("missing graph returned %d", got)
	}
}
