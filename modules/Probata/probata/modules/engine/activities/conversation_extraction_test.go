// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// fakeConversations is an in-memory service.ConversationStore: one conversation of n messages.
type fakeConversations struct {
	messages []service.ConversationMessage
	export   service.ExtractionExport
}

func newFakeConversations(n int) *fakeConversations {
	f := &fakeConversations{}
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		at := base.Add(time.Duration(i) * time.Minute)
		f.messages = append(f.messages, service.ConversationMessage{
			ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", i), OccurredAt: &at, Body: fmt.Sprintf("message %d", i),
			Participants:   json.RawMessage(`[{"role":"sender","identifier":"self"},{"role":"recipient","identifier":"+18105550101"}]`),
			ProjectionKind: "first_party", SourceVersionID: "sv-1", Ordinal: int64(i),
		})
	}
	return f
}

func (f *fakeConversations) ResolveConversation(context.Context, string, flow.ConversationRef) ([]flow.RunRef, error) {
	return []flow.RunRef{{MatterMode: "LIVE", PreviewHandle: "h", GenerationID: "g1", SourceVersionID: "sv-1"}}, nil
}

func (f *fakeConversations) ConversationInfo(context.Context, string, flow.ConversationRef) (service.ConversationInfo, error) {
	first, last := f.messages[0].OccurredAt, f.messages[len(f.messages)-1].OccurredAt
	return service.ConversationInfo{
		Messages: len(f.messages), FirstAt: first, LastAt: last,
		SourceFiles:  []service.SourceFile{{SourceVersionID: "sv-1", SourceKey: "exports/a.derived/threads/c.ndjson", GenerationID: "g1"}},
		Participants: []service.ParticipantCount{{Identifier: "self", Messages: len(f.messages)}},
	}, nil
}

func (f *fakeConversations) ConversationMessages(_ context.Context, _ []string, after service.MessageCursor, limit int) ([]service.ConversationMessage, error) {
	var out []service.ConversationMessage
	for _, message := range f.messages {
		if after.ID != "" && message.ID <= after.ID {
			continue
		}
		if len(out) < limit {
			out = append(out, message)
		}
	}
	return out, nil
}

func (f *fakeConversations) ConversationExtractions(context.Context, []string) (service.ExtractionExport, error) {
	return f.export, nil
}

// surrealRecorder is an httptest stand-in for surreal-case that records every upsert batch.
type surrealRecorder struct {
	mu      sync.Mutex
	batches []int
	threads int
	count   int
}

func (s *surrealRecorder) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var request struct{ Params []json.RawMessage }
	_ = json.NewDecoder(r.Body).Decode(&request)
	var sql string
	_ = json.Unmarshal(request.Params[0], &sql)
	var vars map[string]json.RawMessage
	_ = json.Unmarshal(request.Params[1], &vars)
	switch {
	case strings.Contains(sql, "count()"):
		row := func(n int) map[string]any {
			return map[string]any{"status": "OK", "result": []map[string]int{{"n": n}}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{row(s.count), row(0), row(0)}})
		return
	case strings.Contains(sql, "UPSERT type::record('conv_message'"):
		var rows []json.RawMessage
		_ = json.Unmarshal(vars["rows"], &rows)
		s.batches = append(s.batches, len(rows))
		s.count += len(rows)
	case strings.Contains(sql, "UPSERT type::record('conv_thread'"):
		s.threads++
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{map[string]any{"status": "OK", "result": nil}}})
}

func newSender(t *testing.T, conversations service.ConversationStore) (ConversationActivities, *surrealRecorder) {
	t.Helper()
	recorder := &surrealRecorder{}
	server := httptest.NewServer(http.HandlerFunc(recorder.serve))
	t.Cleanup(server.Close)
	client, err := surrealsink.New(surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "case", User: "u", Password: "p", AuthLevel: "database", CaseRef: "current"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	acts := NewConversationActivities(nil, conversations, client, "")
	acts.Heartbeat = func(context.Context, any) {}
	acts.Resume = func(context.Context, any) bool { return false }
	return acts, recorder
}

var sendTarget = flow.SendTarget{OperatingMode: "LIVE", CourtCaseID: "01a0f751-e07b-76a1-a738-eb3e3aa3e68c",
	MatterID: "01a0f751-e07b-75cc-9ad5-63ad9449a8ba",
	Ref:      flow.ConversationRef{ExportKey: "exports/a", Conv: "c"},
	Request:  "send-1",
}

// TestSendReadsPagesAndUpsertsEveryMessageOnce proves 250 messages go in batches of 100 under one deterministic thread,
// that the plan and the read-back agree, and that the thread id is the same on every send.
func TestSendReadsPagesAndUpsertsEveryMessageOnce(t *testing.T) {
	store := newFakeConversations(250)
	acts, recorder := newSender(t, store)
	ctx := context.Background()
	plan, err := acts.PlanSurrealSend(ctx, sendTarget)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Messages != 250 || plan.ThreadID != flow.SurrealThreadID(sendTarget.MatterID, sendTarget.Ref) || len(plan.Generations) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	written, err := acts.UpsertConversationToSurreal(ctx, sendTarget)
	if err != nil {
		t.Fatal(err)
	}
	if written.Messages != 250 || written.Threads != 1 {
		t.Fatalf("written = %+v", written)
	}
	if fmt.Sprint(recorder.batches) != "[100 100 50]" || recorder.threads != 1 {
		t.Fatalf("batches = %v, threads = %d", recorder.batches, recorder.threads)
	}
	verified, err := acts.VerifySurrealSend(ctx, flow.VerifyRequest{Target: sendTarget, Plan: plan, Written: written})
	if err != nil || !verified.Match || verified.Messages != 250 {
		t.Fatalf("verified = %+v, %v", verified, err)
	}
	// A short read-back is a mismatch, not a success.
	recorder.count = 249
	verified, _ = acts.VerifySurrealSend(ctx, flow.VerifyRequest{Target: sendTarget, Plan: plan, Written: written})
	if verified.Match {
		t.Fatal("a read-back one message short was reported as a match")
	}
}

// TestSendResumesAfterTheLastHeartbeatedPage proves a retried upsert does not start over.
func TestSendResumesAfterTheLastHeartbeatedPage(t *testing.T) {
	store := newFakeConversations(250)
	acts, recorder := newSender(t, store)
	acts.Resume = func(_ context.Context, into any) bool {
		progress := into.(*sendProgress)
		progress.Thread, progress.Messages = true, 100
		progress.Cursor = service.MessageCursor{At: store.messages[99].OccurredAt, ID: store.messages[99].ID}
		return true
	}
	written, err := acts.UpsertConversationToSurreal(context.Background(), sendTarget)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(recorder.batches) != "[100 50]" || recorder.threads != 0 || written.Messages != 250 {
		t.Fatalf("batches = %v, threads = %d, written = %+v", recorder.batches, recorder.threads, written)
	}
}

// TestSendWithoutASurrealConnectionFailsOnceWithItsReason proves the missing sink is a non-retryable, explained error.
func TestSendWithoutASurrealConnectionFailsOnceWithItsReason(t *testing.T) {
	acts := NewConversationActivities(nil, newFakeConversations(1), nil, "SURREAL_CASE_URL is not set")
	_, err := acts.PlanSurrealSend(context.Background(), sendTarget)
	var app *temporal.ApplicationError
	if err == nil || !errors.As(err, &app) || !app.NonRetryable() || app.Type() != flow.SurrealNotConfiguredErrorType || !strings.Contains(app.Error(), "SURREAL_CASE_URL") {
		t.Fatalf("error = %v", err)
	}
}

// TestSurrealMessageSplitsSenderFromRecipients proves the participant roles map onto the Surreal record.
func TestSurrealMessageSplitsSenderFromRecipients(t *testing.T) {
	message := newFakeConversations(1).messages[0]
	record := surrealMessage("t1", "m1", message)
	if record.Sender != "self" || len(record.Recipients) != 1 || record.Recipients[0] != "+18105550101" || record.Party != "first_party" {
		t.Fatalf("record = %+v", record)
	}
}

// TestExternalPageIsGroundedTaggedAndStagedOnce proves an external extractor's reply is staged under the run, tagged with the
// extractor, and that replaying the same window writes nothing twice.
func TestExternalPageIsGroundedTaggedAndStagedOnce(t *testing.T) {
	store := newMemStore()
	acts := NewConversationActivities(store, nil, nil, "")
	runID := "44444444-4444-4444-8444-444444444444"
	record := store.messages[0].RecordID
	page := model.ExternalPage{
		Extractor: "semantica", ExtractorVersion: "1", LastOrdinal: 2, Messages: 3, Done: true,
		People: []model.ExternalEntity{{Name: "Katherine", Mentions: []model.ExternalMention{{RecordID: record, Text: "Katherine"}}}},
		Events: []model.ExternalEvent{{Title: "Court date", RecordID: store.messages[1].RecordID, Date: strPtr("2025-07-02"), Type: "court"}},
	}
	raw, _ := json.Marshal(page)
	stage := flow.StageExternalPage{
		Input: flow.ExternalRunInput{ExtractionID: "e1", Run: store.run, Extractor: "semantica"}, RunID: runID,
		Request: flow.ExternalPageRequest{GenerationID: store.run.GenerationID, AfterOrdinal: -1, Limit: 200}, Page: raw,
	}
	for i := 0; i < 2; i++ {
		result, err := acts.StageExternalExtractionPage(context.Background(), stage)
		if err != nil {
			t.Fatal(err)
		}
		if result.Entities != 1 || result.Events != 1 || result.Messages != 3 || !result.Done || result.Last != 2 {
			t.Fatalf("attempt %d result = %+v", i, result)
		}
	}
	if len(store.entities) != 1 || len(store.events) != 1 {
		t.Fatalf("staged %d entities and %d events after a replay, want 1 and 1", len(store.entities), len(store.events))
	}
	for _, proposal := range store.entities {
		if len(proposal.Extractors) != 1 || proposal.Extractors[0] != "semantica@1" {
			t.Fatalf("extractors = %v", proposal.Extractors)
		}
	}
}

// TestExternalPageThatIsSkippedOrMalformedStagesNothing proves a skipped extractor and a non-JSON reply write no proposals.
func TestExternalPageThatIsSkippedOrMalformedStagesNothing(t *testing.T) {
	store := newMemStore()
	acts := NewConversationActivities(store, nil, nil, "")
	skipped, _ := json.Marshal(model.ExternalPage{Skipped: true, Reason: "no key"})
	result, err := acts.StageExternalExtractionPage(context.Background(), flow.StageExternalPage{Page: skipped})
	if err != nil || !result.Skipped || result.Reason != "no key" {
		t.Fatalf("skipped result = %+v, %v", result, err)
	}
	result, err = acts.StageExternalExtractionPage(context.Background(), flow.StageExternalPage{Page: json.RawMessage(`not json`)})
	if err != nil || !result.Invalid {
		t.Fatalf("malformed result = %+v, %v", result, err)
	}
	if len(store.entities) != 0 || len(store.events) != 0 {
		t.Fatal("something was staged")
	}
}

func strPtr(value string) *string { return &value }
