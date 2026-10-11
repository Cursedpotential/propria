// Byline: Codex · GPT-6 · 2026-10-11
package runtimeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/Cursedpotential/probata/engine/discovery"
	"github.com/stretchr/testify/require"
)

const discoveryTestRun = "11111111-1111-4111-8111-111111111111"
const discoveryTestFeedback = "22222222-2222-4222-8222-222222222222"
const discoveryTestCandidate = "33333333-3333-4333-8333-333333333333"

type discoveryHTTPStore struct {
	discovery.Store
	creates, corrections, dispatches int
	actor                            discovery.Actor
	run                              discovery.Run
}

func (s *discoveryHTTPStore) CreateRun(_ context.Context, r discovery.RunRequest) (discovery.Run, error) {
	s.creates++
	s.actor = r.Actor
	s.run = discovery.Run{Request: r, Revision: 1, DispatchPending: true}
	return s.run, nil
}
func (s *discoveryHTTPStore) ReadRun(_ context.Context, _ string, _ discovery.Scope) (discovery.Run, error) {
	return s.run, nil
}
func (s *discoveryHTTPStore) SubmitFeedback(_ context.Context, _ string, _ discovery.Scope, r discovery.FeedbackRequest, a discovery.Actor) (discovery.FeedbackReceipt, error) {
	s.corrections++
	s.actor = a
	s.run.Revision++
	s.run.FeedbackRevision++
	s.run.DispatchPending = true
	return discovery.FeedbackReceipt{Feedback: discovery.Feedback{FeedbackRequest: r, Actor: a, Revision: s.run.FeedbackRevision}, Run: s.run}, nil
}
func (s *discoveryHTTPStore) RecordDispatch(_ context.Context, _ string, _ discovery.Scope, _ int64, workflowID, runID string) error {
	s.dispatches++
	s.run.DispatchPending = false
	s.run.WorkflowID = workflowID
	s.run.TemporalRunID = runID
	return nil
}

type discoveryHTTPDispatcher struct {
	calls int
	err   error
}

func (d *discoveryHTTPDispatcher) Start(_ context.Context, _ discovery.Run) (string, string, error) {
	d.calls++
	return "discovery-test", "temporal-test", d.err
}

func discoveryHTTPFixture(t *testing.T) (*discoveryHTTPStore, *discoveryHTTPDispatcher, http.Handler) {
	t.Helper()
	store := &discoveryHTTPStore{}
	dispatcher := &discoveryHTTPDispatcher{}
	h, err := NewDiscoveryHTTPHandler(store, dispatcher, serviceTokenPath(t))
	require.NoError(t, err)
	return store, dispatcher, h.Routes()
}
func discoveryCreateBody(actor *discovery.Actor, mode string) []byte {
	body := map[string]any{"run_id": discoveryTestRun, "scope": map[string]string{"mode": mode, "matter_id": "44444444-4444-4444-8444-444444444444", "court_case_id": "55555555-5555-4555-8555-555555555555"}, "input_backend": "postgresql", "dataset_id": "selected", "selection_revision": "1", "generation_ids": []string{"66666666-6666-4666-8666-666666666666"}, "hypotheses": []map[string]string{{"hypothesis_id": "request", "text": "a request about an arrangement"}}, "filters": map[string]string{"perspective": "hindsight"}, "max_records": 20, "max_candidates": 10}
	if actor != nil {
		body["actor"] = actor
	}
	raw, _ := json.Marshal(body)
	return raw
}

// TestDiscoveryActorAndOperatingBoundaries prevents spoofed writes and premature dispatch.
// Inputs: authenticated requests with mismatched identity, mode or idempotency.
// Outputs: rejection and zero canonical effects. Effects: synthetic counters only.
func TestDiscoveryActorAndOperatingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, query, bodyMode, key string
		actor                      *discovery.Actor
		status                     int
	}{
		{"body-actor", "LIVE", "LIVE", discoveryTestRun, &discovery.Actor{SubjectUID: "spoof", Username: "spoof"}, 422},
		{"dev-body", "LIVE", "DEV", discoveryTestRun, nil, 409},
		{"dev-query", "DEV", "LIVE", discoveryTestRun, nil, 409},
		{"unknown-query", "unknown", "LIVE", discoveryTestRun, nil, 422},
		{"wrong-key", "LIVE", "LIVE", discoveryTestFeedback, nil, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, dispatcher, routes := discoveryHTTPFixture(t)
			r := newPreviewRequest("POST", "/discovery/runs?mode="+tc.query, discoveryCreateBody(tc.actor, tc.bodyMode))
			r.Header.Set("Idempotency-Key", tc.key)
			w := servePreviewRequest(routes, r)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			require.Zero(t, store.creates)
			require.Zero(t, dispatcher.calls)
		})
	}
}

// TestDiscoverySavedRunSurvivesDispatchFailure proves PG acknowledgement precedes scheduling.
// Inputs: valid request and failed Temporal dispatcher. Outputs: persisted pending run.
// Effects: synthetic store only. Choose to prevent false loss after an accepted PG save.
func TestDiscoverySavedRunSurvivesDispatchFailure(t *testing.T) {
	store, dispatcher, routes := discoveryHTTPFixture(t)
	dispatcher.err = errors.New("Temporal offline")
	r := newPreviewRequest("POST", "/discovery/runs?mode=LIVE", discoveryCreateBody(nil, "LIVE"))
	r.Header.Set("Idempotency-Key", discoveryTestRun)
	w := servePreviewRequest(routes, r)
	require.Equal(t, 202, w.Code, w.Body.String())
	require.Equal(t, 1, store.creates)
	require.Equal(t, 1, dispatcher.calls)
	require.Zero(t, store.dispatches)
	var run discovery.Run
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &run))
	require.True(t, run.DispatchPending)
	require.Equal(t, "authentik-user-1", store.actor.SubjectUID)
}

// TestDiscoverySavedCorrectionSurvivesDispatchFailure keeps durable feedback after outages.
// Inputs: authenticated correction with its revision and UUID key. Outputs: saved pending feedback.
// Effects: synthetic store only. Choose to prevent retraining on an unsaved correction.
func TestDiscoverySavedCorrectionSurvivesDispatchFailure(t *testing.T) {
	store, dispatcher, routes := discoveryHTTPFixture(t)
	dispatcher.err = errors.New("Temporal offline")
	store.run = discovery.Run{Request: discovery.RunRequest{RunID: discoveryTestRun}, Revision: 3}
	raw, _ := json.Marshal(discovery.FeedbackRequest{FeedbackID: discoveryTestFeedback, CandidateID: discoveryTestCandidate, ExpectedRevision: 3, Kind: "wrong_person", AttributedActorIDs: []string{"source-person"}})
	r := newPreviewRequest("POST", "/discovery/runs/"+discoveryTestRun+"/feedback?mode=LIVE&matter_id=44444444-4444-4444-8444-444444444444&court_case_id=55555555-5555-4555-8555-555555555555", raw)
	r.Header.Set("Idempotency-Key", discoveryTestFeedback)
	w := servePreviewRequest(routes, r)
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, 1, store.corrections)
	require.Equal(t, 1, dispatcher.calls)
	require.Zero(t, store.dispatches)
	var receipt discovery.FeedbackReceipt
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &receipt))
	require.True(t, receipt.Run.DispatchPending)
	require.Equal(t, "wrong_person", receipt.Feedback.Kind)
}
