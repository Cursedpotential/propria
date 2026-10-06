// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Opus 5.5 · 2026-09-25

package runtimeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

const extractionHandle = "handle_abcdefghijklmnopqrstuvwxyz0123"

type apiStore struct {
	entities []entities.Proposal
	events   []events.Proposal
	message  entities.MessageView
}

func (s *apiStore) ResolveRun(_ context.Context, handle string) (flow.RunRef, error) {
	if handle != extractionHandle {
		return flow.RunRef{}, service.ErrNotFound
	}
	return flow.RunRef{MatterMode: "LIVE", PreviewHandle: handle, GenerationID: "gen-1", SourceVersionID: "src-1"}, nil
}
func (s *apiStore) ParticipantAggregates(context.Context, string) ([]entities.ParticipantAggregate, error) {
	return nil, nil
}
func (s *apiStore) MessagePage(context.Context, string, int64, int) ([]entities.MessageView, error) {
	return nil, nil
}
func (s *apiStore) MessageByID(_ context.Context, _, id string) (entities.MessageView, error) {
	if id != s.message.RecordID {
		return entities.MessageView{}, service.ErrNotFound
	}
	return s.message, nil
}
func (s *apiStore) RecordsMissing(context.Context, string, []string) ([]string, error) {
	return nil, nil
}
func (s *apiStore) BeginRun(context.Context, service.RunRow) error { return nil }
func (s *apiStore) FinishRun(context.Context, string, string, map[string]any, string) error {
	return nil
}
func (s *apiStore) ExtractionRuns(context.Context, string) ([]service.RunSummary, error) {
	return nil, nil // as the Postgres store answers a run nobody has extracted yet
}
func (s *apiStore) StageEntities(context.Context, string, []entities.Proposal) ([]string, error) {
	return nil, nil
}
func (s *apiStore) StageEvents(context.Context, string, []events.Proposal) ([]string, error) {
	return nil, nil
}
func (s *apiStore) CurrentEntities(context.Context, string) ([]entities.Proposal, error) {
	return s.entities, nil
}
func (s *apiStore) CurrentEvents(context.Context, string) ([]events.Proposal, error) {
	return s.events, nil
}
func (s *apiStore) PendingEntitiesOfRuns(context.Context, []string) ([]entities.Proposal, error) {
	return nil, nil
}
func (s *apiStore) ReplaceEntities(_ context.Context, _ string, ids []string, insert []entities.Proposal, supersede []string) error {
	for i, proposal := range insert {
		proposal.CandidateID, proposal.ReviewState = ids[i], entities.StatePending
		s.entities = append(s.entities, proposal)
	}
	kept := s.entities[:0]
	for _, proposal := range s.entities {
		superseded := false
		for _, id := range supersede {
			superseded = superseded || proposal.CandidateID == id
		}
		if !superseded {
			kept = append(kept, proposal)
		}
	}
	s.entities = kept
	return nil
}
func (s *apiStore) ReplaceEvents(_ context.Context, _ string, ids []string, insert []events.Proposal, _ []string) error {
	for i, event := range insert {
		event.CandidateID, event.ReviewState = ids[i], entities.StatePending
		s.events = append(s.events, event)
	}
	return nil
}
func (s *apiStore) Registry(context.Context, service.RegistryLookup) ([]entities.RegistryEntity, error) {
	return nil, nil
}
func (s *apiStore) SearchRegistry(context.Context, string, int) ([]entities.RegistryEntity, error) {
	return []entities.RegistryEntity{{ID: "e-1", DisplayName: "Katherine Doe", RegistryType: "person"}}, nil
}
func (s *apiStore) WriteEntities(context.Context, []service.EntityRow) (int, error) { return 0, nil }
func (s *apiStore) WriteAliases(context.Context, []service.AliasRow) (int, error)   { return 0, nil }
func (s *apiStore) WriteMentions(context.Context, []service.MentionRow, []service.ResolutionRow, service.Reviewer, string) (int, int, error) {
	return 0, 0, nil
}
func (s *apiStore) WriteEventCandidates(context.Context, []service.EventCandidateRow, service.Reviewer) (int, error) {
	return 0, nil
}
func (s *apiStore) EnsureCollection(context.Context, string, string) (string, error) { return "c", nil }
func (s *apiStore) WriteTimelineMembers(context.Context, []service.TimelineMemberRow) (int, error) {
	return 0, nil
}
func (s *apiStore) FinalizeCommit(context.Context, service.CommitReceipt, []service.Promotion, []service.Promotion) error {
	return nil
}

type workflowRecorder struct {
	extractions []flow.ExtractionRequest
	commits     []flow.CommitRequest
	progress    *flow.Progress
}

func (w *workflowRecorder) StartExtraction(_ context.Context, request flow.ExtractionRequest) (flow.Started, error) {
	w.extractions = append(w.extractions, request)
	return flow.Started{WorkflowID: flow.ExtractionWorkflowID(request), RunID: "run"}, nil
}
func (w *workflowRecorder) StartCommit(_ context.Context, request flow.CommitRequest) (flow.Started, error) {
	w.commits = append(w.commits, request)
	return flow.Started{WorkflowID: flow.CommitWorkflowID(request.Run.PreviewHandle, request.Digest), RunID: "run"}, nil
}
func (w *workflowRecorder) Status(_ context.Context, id string) (flow.Progress, error) {
	if w.progress != nil {
		return *w.progress, nil
	}
	return flow.Progress{Outcome: flow.OutcomeRunning, Steps: []flow.StepResult{{Step: "rules", Status: flow.StepRunning}}}, nil
}

func extractionTestHandler(t *testing.T) (*EntityExtractionHTTPHandler, *apiStore, *workflowRecorder) {
	t.Helper()
	at := time.Date(2025, 6, 1, 14, 46, 0, 0, time.UTC)
	available := time.Date(2026, 9, 21, 1, 14, 39, 0, time.UTC)
	person := entities.Proposal{CandidateID: "p-1", Name: "Katherine", RegistryType: "person", ReviewState: entities.StatePending, GenerationID: "gen-1"}
	person.AddAlias(entities.NewAddressAlias(entities.NormalizeAddress("+18105550101"), entities.SourceParticipant, ""))
	event := events.Proposal{CandidateID: "v-1", Title: "Court", EventType: "court", OccurredAt: &at, TemporalPrecision: "point",
		ReviewState: entities.StatePending, GenerationID: "gen-1", SourceRecords: []events.SourceRecord{{RecordID: "r-1", SourceAvailableFrom: &available}},
		EntityKeys: []string{entities.AnyNameKey("Katherine")}}
	store := &apiStore{entities: []entities.Proposal{person}, events: []events.Proposal{event},
		message: entities.MessageView{RecordID: "r-1", OccurredAt: &at, SourceAvailableFrom: &available, Body: "Court on Thursday"}}
	recorder := &workflowRecorder{}
	handler, err := NewEntityExtractionHTTPHandler(store, recorder, serviceTokenPath(t))
	require.NoError(t, err)
	return handler, store, recorder
}

func extractionRequest(method, target string, body any, key string) *http.Request {
	raw, _ := json.Marshal(body)
	req := newPreviewRequest(method, target, raw)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	return req
}

func TestExtractionRoutesRequireTheServiceToken(t *testing.T) {
	handler, _, _ := extractionTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/reference-import/entities/proposals?preview_handle="+extractionHandle, nil)
	req.RemoteAddr = "100.64.1.9:3456"
	recorder := httptest.NewRecorder()
	handler.Routes().ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestExtractStartsOneWorkflowPerClick(t *testing.T) {
	handler, _, workflows := extractionTestHandler(t)
	body := map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL"}
	first := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/extract", body, "click-1"))
	again := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/extract", body, "click-1"))
	require.Equal(t, http.StatusAccepted, first.Code, first.Body.String())
	require.Equal(t, http.StatusAccepted, again.Code)
	require.Len(t, workflows.extractions, 2)
	require.Equal(t, workflows.extractions[0].ExtractionID, workflows.extractions[1].ExtractionID, "a retried click joins the same extraction")
	require.True(t, workflows.extractions[0].UseModel)
	require.Equal(t, "operator", workflows.extractions[0].Actor.Username)
	missingKey := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/extract", body, ""))
	require.Equal(t, http.StatusUnauthorized, missingKey.Code)
}

func TestValidateAndCommitAreFailClosed(t *testing.T) {
	handler, _, workflows := extractionTestHandler(t)
	testMode := map[string]any{"preview_handle": extractionHandle, "matter_mode": "TEST"}
	validation := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/validate", testMode, ""))
	require.Equal(t, http.StatusConflict, validation.Code)
	var report struct {
		OK     bool   `json:"ok"`
		Digest string `json:"digest"`
	}
	require.Empty(t, workflows.commits)
	blocked := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/commit",
		map[string]any{"preview_handle": extractionHandle, "matter_mode": "TEST", "digest": report.Digest}, "commit-1"))
	require.Equal(t, http.StatusConflict, blocked.Code)
	require.Contains(t, blocked.Body.String(), "isolated")
	require.Empty(t, workflows.commits)

	live := map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL"}
	validation = servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/validate", live, ""))
	require.NoError(t, json.Unmarshal(validation.Body.Bytes(), &report))
	require.True(t, report.OK, validation.Body.String())
	stale := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/commit",
		map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL", "digest": "not-the-digest"}, "commit-2"))
	require.Equal(t, http.StatusUnprocessableEntity, stale.Code)
	require.Contains(t, stale.Body.String(), "changed since they were validated")
	started := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/commit",
		map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL", "digest": report.Digest}, "commit-3"))
	require.Equal(t, http.StatusAccepted, started.Code, started.Body.String())
	require.Len(t, workflows.commits, 1)
	require.Equal(t, flow.DefaultProjectionTaskQueue, workflows.commits[0].ProjectionTaskQueue)
}

func TestStatusIsScopedToTheRunsOwnWorkflows(t *testing.T) {
	handler, _, _ := extractionTestHandler(t)
	own := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/extractions/entity-extraction:"+extractionHandle+":x?preview_handle="+extractionHandle, nil)
	require.Equal(t, http.StatusOK, own.Code)
	foreign := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/extractions/proffer-workflow-123?preview_handle="+extractionHandle, nil)
	require.Equal(t, http.StatusNotFound, foreign.Code)
}

// The BFF validates lists as lists: a fresh run and a workflow that is not
// queryable yet must answer [] where the panel expects a list, never null.
func TestEmptyListsAreListsNotNull(t *testing.T) {
	handler, store, workflows := extractionTestHandler(t)
	workflows.progress = &flow.Progress{Outcome: flow.OutcomeRunning}
	status := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/extractions/entity-extraction:"+extractionHandle+":x?preview_handle="+extractionHandle, nil)
	require.Equal(t, http.StatusOK, status.Code)
	require.Contains(t, status.Body.String(), `"steps":[]`)

	store.entities = append(store.entities, entities.Proposal{CandidateID: "p-2", Name: "Nobody", RegistryType: "person", ReviewState: entities.StatePending})
	store.events = append(store.events, events.Proposal{CandidateID: "v-2", Title: "Bare", EventType: "other", ReviewState: entities.StatePending})
	proposals := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/proposals?preview_handle="+extractionHandle+"&matter_mode=REAL", nil)
	require.Equal(t, http.StatusOK, proposals.Code)
	body := proposals.Body.String()
	require.Contains(t, body, `"extractions":[]`)
	require.NotContains(t, body, `null`, "no list field may be null: %s", body)
}

func TestCorrectionsMarksRecordsAndRegistry(t *testing.T) {
	handler, store, _ := extractionTestHandler(t)
	stale := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/corrections", map[string]any{
		"preview_handle": extractionHandle, "matter_mode": "REAL", "target": "entity",
		"entity": map[string]any{"op": "rename", "candidate_ids": []string{"gone"}, "name": "X"},
	}, "fix-1"))
	require.Equal(t, http.StatusConflict, stale.Code, stale.Body.String())
	renamed := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/corrections", map[string]any{
		"preview_handle": extractionHandle, "matter_mode": "REAL", "target": "entity",
		"entity": map[string]any{"op": "rename", "candidate_ids": []string{"p-1"}, "name": "Katherine Doe"},
	}, "fix-2"))
	require.Equal(t, http.StatusOK, renamed.Code, renamed.Body.String())
	require.Equal(t, "Katherine Doe", store.entities[len(store.entities)-1].Name)
	invalid := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/entities/corrections", map[string]any{
		"preview_handle": extractionHandle, "matter_mode": "REAL", "target": "entity",
		"entity": map[string]any{"op": "retype", "candidate_ids": []string{store.entities[len(store.entities)-1].CandidateID}, "registry_type": "dragon"},
	}, "fix-3"))
	require.Equal(t, http.StatusUnprocessableEntity, invalid.Code)

	marked := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/events/from-record", map[string]any{
		"preview_handle": extractionHandle, "matter_mode": "REAL", "record_id": "r-1", "title": "Court on Thursday",
	}, "mark-1"))
	require.Equal(t, http.StatusCreated, marked.Code, marked.Body.String())
	require.Contains(t, marked.Body.String(), `"detected_by":"owner"`)
	require.Contains(t, marked.Body.String(), `"occurred_at":"2025-06-01T14:46:00Z"`)

	record := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/records/r-1?preview_handle="+extractionHandle, nil)
	require.Equal(t, http.StatusOK, record.Code)
	require.True(t, bytes.Contains(record.Body.Bytes(), []byte("Court on Thursday")))
	missing := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/records/r-9?preview_handle="+extractionHandle, nil)
	require.Equal(t, http.StatusNotFound, missing.Code)
	found := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/registry?q=kath", nil)
	require.Equal(t, http.StatusOK, found.Code)
	require.True(t, strings.Contains(found.Body.String(), "Katherine Doe"))

	proposals := servePreview(handler.Routes(), http.MethodGet, "/reference-import/entities/proposals?preview_handle="+extractionHandle+"&matter_mode=REAL", nil)
	require.Equal(t, http.StatusOK, proposals.Code)
	require.Contains(t, proposals.Body.String(), `"resolved_entity_candidate_ids"`)
	require.Contains(t, proposals.Body.String(), `"event_types"`)
}
