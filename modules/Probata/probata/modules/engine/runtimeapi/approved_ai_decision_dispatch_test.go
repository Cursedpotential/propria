package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraphai"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/stretchr/testify/require"
)

type aiProjectionDispatchRecorder struct {
	ExtractionWorkflows
	scopes []approvedgraphai.Scope
	err    error
}

// StartApprovedAIProjection records the exact scope without a Temporal or database connection.
// Inputs: approved test scope. Outputs: deterministic fixture execution or injected error.
// Effects: test recorder only. Choose for post-commit dispatch contract tests.
func (r *aiProjectionDispatchRecorder) StartApprovedAIProjection(_ context.Context, scope approvedgraphai.Scope) (ApprovedAIProjectionStarted, error) {
	r.scopes = append(r.scopes, scope)
	if r.err != nil {
		return ApprovedAIProjectionStarted{}, r.err
	}
	id, err := approvedgraphai.WorkflowID(scope)
	return ApprovedAIProjectionStarted{WorkflowID: id, RunID: "actual-test-run"}, err
}

// TestCommittedAIDecisionDispatchPreservesScopeAndRetries checks approval-only, retry-stable enqueue metadata.
// Inputs: synthetic unit identifiers only. Outputs: assertions. Effects: no I/O.
// Choose to verify response composition independently of graph-store changes.
func TestCommittedAIDecisionDispatchPreservesScopeAndRetries(t *testing.T) {
	r := &aiProjectionDispatchRecorder{}
	h := &EntityExtractionHTTPHandler{workflows: r}
	source, candidate, decision := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"
	digest := strings.Repeat("a", 64)
	for _, rejected := range []string{"rejected", "needs_more_info"} {
		got := h.dispatchApprovedAIDecisionProjection(context.Background(), rejected, source, candidate, decision, digest)
		require.Equal(t, "not_requested", got.Status)
		require.False(t, got.Retryable)
	}
	require.Empty(t, r.scopes)
	r.err = errors.New("private backend detail must not enter response")
	failed := h.dispatchApprovedAIDecisionProjection(context.Background(), "approved", source, candidate, decision, digest)
	require.Equal(t, "pending", failed.Status)
	require.True(t, failed.Retryable)
	require.Equal(t, "projection_enqueue_failed", failed.Error)
	require.Empty(t, failed.RunID)
	r.err = nil
	again := h.dispatchApprovedAIDecisionProjection(context.Background(), "approved", source, candidate, decision, digest)
	require.Equal(t, "enqueued", again.Status)
	require.False(t, again.Retryable)
	require.Equal(t, failed.WorkflowID, again.WorkflowID)
	require.Equal(t, "actual-test-run", again.RunID)
	require.Len(t, r.scopes, 2)
	require.Equal(t, r.scopes[0], r.scopes[1])
	require.Equal(t, approvedgraphai.Scope{MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, SourceVersionID: source, CandidateID: candidate, DecisionID: decision, RequestDigest: digest}, r.scopes[0])
}

// TestCommittedAIDecisionDispatchUnavailableKeepsPending checks optional composition without losing the decision receipt.
// Inputs: a handler without the optional dispatcher. Outputs: pending metadata. Effects: no I/O.
// Choose for deployments that have not yet enabled the existing approved graph worker.
func TestCommittedAIDecisionDispatchUnavailableKeepsPending(t *testing.T) {
	h := &EntityExtractionHTTPHandler{}
	got := h.dispatchApprovedAIDecisionProjection(context.Background(), "approved", "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333", strings.Repeat("b", 64))
	require.Equal(t, "pending", got.Status)
	require.Equal(t, "projection_dispatch_unavailable", got.Error)
	require.True(t, got.Retryable)
	require.NotEmpty(t, got.WorkflowID)
}

type aiDispatchDecisionStore struct {
	service.Store
	service.AIReviewStore
	decisionErr error
	digest      string
	decisions   int
}

// VerifyAISource supplies a verified live source for the isolated handler contract test.
// Inputs: fixture handle and pin. Outputs: LIVE. Effects: none; no retained original is accessed.
// Choose only to isolate post-decision response and dispatch behavior.
func (s *aiDispatchDecisionStore) VerifyAISource(context.Context, string, service.AISourcePin) (string, error) {
	return "LIVE", nil
}

// DecideAICandidate records the receipt inputs or returns an injected store failure.
// Inputs: handler-verified decision request. Outputs: fixture decision ID. Effects: recorder only.
// Choose to prove failed decisions never dispatch and committed receipts survive enqueue failure.
func (s *aiDispatchDecisionStore) DecideAICandidate(_ context.Context, _ service.AISourcePin, _, _, _ string, _ entities.Actor, digest string, _ time.Time) (string, error) {
	s.decisions++
	s.digest = digest
	return "33333333-3333-4333-8333-333333333333", s.decisionErr
}

// TestAIDecisionHTTPReceiptSurvivesDispatchFailure checks the handler's post-commit hook end to end.
// Inputs: synthetic unit requests and stub store/dispatcher. Outputs: HTTP receipt assertions.
// Effects: no network or database I/O. Choose with the parent-applied tiny decision handler hook.
func TestAIDecisionHTTPReceiptSurvivesDispatchFailure(t *testing.T) {
	s := &aiDispatchDecisionStore{decisionErr: errors.New("store write failed")}
	d := &aiProjectionDispatchRecorder{err: errors.New("Temporal unavailable")}
	h, err := NewEntityExtractionHTTPHandler(s, d, serviceTokenPath(t))
	require.NoError(t, err)
	body := aiDecisionRequest{
		runRequest:  runRequest{PreviewHandle: extractionHandle, MatterMode: "LIVE"},
		AISourcePin: service.AISourcePin{SourceVersionID: "11111111-1111-4111-8111-111111111111", SourceObjectID: "44444444-4444-4444-8444-444444444444", SourceSHA256: strings.Repeat("c", 64)},
		CandidateID: "22222222-2222-4222-8222-222222222222", ExpectedContentSHA256: strings.Repeat("d", 64), Decision: "approved",
	}
	request := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.decideAICandidate(w, extractionRequest(http.MethodPost, "/reference-import/ai/candidates/decide", body, "same-key"))
		return w
	}
	w := request()
	require.NotEqual(t, http.StatusOK, w.Code)
	require.Empty(t, d.scopes)
	s.decisionErr = nil
	w = request()
	require.Equal(t, http.StatusOK, w.Code)
	var receipt struct {
		DecisionCommitted bool                 `json:"decision_committed"`
		DecisionID        string               `json:"decision_id"`
		RequestDigest     string               `json:"request_digest"`
		Projection        AIDecisionProjection `json:"projection"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &receipt))
	require.True(t, receipt.DecisionCommitted)
	require.Equal(t, "33333333-3333-4333-8333-333333333333", receipt.DecisionID)
	require.Equal(t, s.digest, receipt.RequestDigest)
	require.Equal(t, "pending", receipt.Projection.Status)
	require.True(t, receipt.Projection.Retryable)
	stableID := receipt.Projection.WorkflowID
	d.err = nil
	w = request()
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &receipt))
	require.True(t, receipt.DecisionCommitted)
	require.Equal(t, "enqueued", receipt.Projection.Status)
	require.Equal(t, stableID, receipt.Projection.WorkflowID)
	require.Equal(t, receipt.RequestDigest, d.scopes[1].RequestDigest)
}
