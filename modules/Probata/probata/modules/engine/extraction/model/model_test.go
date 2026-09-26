// Byline: Claude Code · Opus 5.5 · 2026-09-25

package model

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

func ts(value string) *time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return &parsed
}

func testBatch() Batch {
	return Batch{Index: 3,
		Participants: []Participant{
			{Label: "P1", Address: entities.NormalizeAddress("self")},
			{Label: "P2", Address: entities.NormalizeAddress("+18105550101")},
		},
		Messages: []entities.MessageView{
			{RecordID: "r1", Ordinal: 10, OccurredAt: ts("2025-06-01T14:46:00Z"), SourceAvailableFrom: ts("2026-09-21T01:14:39Z"),
				Body:         "Morning Kat, can you get Emma from Lincoln Elementary at 3?",
				Participants: []entities.MessageAddressee{{Role: "sender", Identifier: "self"}, {Role: "recipient", Identifier: "+18105550101"}}},
			{RecordID: "r2", Ordinal: 11, OccurredAt: ts("2025-06-01T15:02:00Z"), SourceAvailableFrom: ts("2026-09-21T01:14:39Z"),
				Body:         "Yes. Court is on 2025-07-02, Katherine",
				Participants: []entities.MessageAddressee{{Role: "sender", Identifier: "+18105550101"}, {Role: "recipient", Identifier: "self"}}},
		}}
}

const validReply = `{
 "people": [
  {"name": "Katherine", "aliases": ["Kat", "Kathy"], "participant": "P2", "mentions": [{"message": "m1", "text": "Kat"}, {"message": "m2", "text": "Katherine"}, {"message": "m2", "text": "Karina"}]},
  {"name": "Emma", "aliases": [], "participant": null, "mentions": [{"message": "m1", "text": "Emma"}]},
  {"name": "Zed", "aliases": [], "participant": null, "mentions": []}
 ],
 "places": [],
 "organizations": [{"name": "Lincoln Elementary", "aliases": [], "participant": null, "mentions": [{"message": "m1", "text": "Lincoln Elementary"}]}],
 "events": [
  {"title": "School pickup", "description": "Asked to get Emma at 3", "message": "m1", "date": null, "when": "at 3", "type": "custody_exchange", "people": ["Kat", "Emma"], "places": [], "organizations": ["Lincoln Elementary"]},
  {"title": "Court date", "description": "", "message": "m2", "date": "2025-07-02", "when": null, "type": "court", "people": ["Katherine"], "places": [], "organizations": []}
 ]
}`

type scripted struct {
	replies []Completion
	calls   int
	modes   []bool
}

func (s *scripted) Complete(_ context.Context, _ []Message, options CallOptions) (Completion, error) {
	if options.Thinking != nil {
		s.modes = append(s.modes, *options.Thinking)
	}
	reply := s.replies[s.calls]
	s.calls++
	return reply, nil
}

func TestExtractBatchGroundsEverything(t *testing.T) {
	completer := &scripted{replies: []Completion{{Content: validReply, FinishReason: "stop"}}}
	outcome, err := ExtractBatch(context.Background(), completer, "moonshotai/kimi-k3", testBatch(), entities.RunScope{GenerationID: "gen-1", SourceVersionID: "src-1"})
	if err != nil || outcome.Invalid {
		t.Fatalf("err=%v outcome=%+v", err, outcome)
	}
	var katherine *entities.Proposal
	for i := range outcome.Entities {
		if outcome.Entities[i].Name == "Katherine" {
			katherine = &outcome.Entities[i]
		}
		if outcome.Entities[i].Name == "Zed" {
			t.Fatal("an entity with no grounded mention that never appears must be dropped")
		}
	}
	if katherine == nil {
		t.Fatalf("no Katherine: %+v", outcome.Entities)
	}
	var linked, kat, kathy bool
	for _, alias := range katherine.Aliases {
		linked = linked || alias.Normalized == "+18105550101"
		kat = kat || alias.Text == "Kat"
		kathy = kathy || alias.Text == "Kathy"
	}
	if !linked || !kat || kathy {
		t.Fatalf("participant link and grounded aliases only (Kathy never appears): %+v", katherine.Aliases)
	}
	if len(katherine.ModelMentions) != 2 || outcome.Ungrounded < 3 {
		t.Fatalf("the invented 'Karina' mention must be dropped: mentions=%d ungrounded=%d", len(katherine.ModelMentions), outcome.Ungrounded)
	}
	mention := katherine.ModelMentions[0]
	if mention.Start == nil || string([]rune(testBatch().Messages[0].Body)[*mention.Start:*mention.End]) != "Kat" {
		t.Fatalf("mention span is not the verbatim text: %+v", mention)
	}
	if len(outcome.Events) != 2 {
		t.Fatalf("events = %+v", outcome.Events)
	}
	for _, event := range outcome.Events {
		switch event.Title {
		case "School pickup":
			if event.TemporalPrecision != "uncertain" || !event.OccurredAt.Equal(*testBatch().Messages[0].OccurredAt) || event.WhenStated != "at 3" {
				t.Fatalf("an event without a stated date is dated by its message and marked uncertain: %+v", event)
			}
		case "Court date":
			if event.OccurredAt.Format("2006-01-02") != "2025-07-02" {
				t.Fatalf("stated date not used: %+v", event)
			}
			if got := event.SourceAvailableFrom(); got == nil || !got.Equal(*ts("2026-09-21T01:14:39Z")) {
				t.Fatalf("event must inherit the source's availability: %v", got)
			}
		}
	}
}

func TestExtractBatchRetriesOnceThenFlags(t *testing.T) {
	bad := Completion{Content: `{"people": [], "places": [], "organizations": [], "events": [], "extra": 1}`, FinishReason: "stop"}
	completer := &scripted{replies: []Completion{bad, bad}}
	outcome, err := ExtractBatch(context.Background(), completer, "m", testBatch(), entities.RunScope{})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Invalid || completer.calls != 2 || len(outcome.Entities) != 0 {
		t.Fatalf("two invalid replies must flag the batch and write nothing: calls=%d %+v", completer.calls, outcome)
	}
	recovering := &scripted{replies: []Completion{{Content: "", FinishReason: "length"}, {Content: validReply, FinishReason: "stop"}}}
	outcome, err = ExtractBatch(context.Background(), recovering, "m", testBatch(), entities.RunScope{})
	if err != nil || outcome.Invalid || outcome.Attempts != 2 {
		t.Fatalf("a truncated reply is retried once: err=%v %+v", err, outcome)
	}
}

func TestShortPromptsStartUnthinkingAndRetryThinkingOnJunk(t *testing.T) {
	junk := &scripted{replies: []Completion{{Content: "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", FinishReason: "stop"}, {Content: validReply, FinishReason: "stop"}}}
	outcome, err := ExtractBatch(context.Background(), junk, "m", testBatch(), entities.RunScope{})
	if err != nil || outcome.Invalid || len(junk.modes) != 2 || junk.modes[0] || !junk.modes[1] {
		t.Fatalf("a short prompt runs thinking off, then retries thinking on: err=%v modes=%v outcome=%+v", err, junk.modes, outcome)
	}
	if !strings.Contains(outcome.RetryReasons[0], "junk") {
		t.Fatalf("junk must be named: %v", outcome.RetryReasons)
	}
	empty := &scripted{replies: []Completion{{Content: "", FinishReason: "stop"}, {Content: "   ", FinishReason: "stop"}}}
	outcome, _ = ExtractBatch(context.Background(), empty, "m", testBatch(), entities.RunScope{})
	if !outcome.Invalid || len(outcome.Entities) != 0 || strings.Join(outcome.Modes, ",") != "off,on" {
		t.Fatalf("two empty replies flag the batch and write nothing: %+v", outcome)
	}
}

func TestLongPromptsStartThinking(t *testing.T) {
	batch := testBatch()
	long := strings.Repeat("Katherine said the pickup moved again. ", 900)
	batch.Messages[0].Body = long
	if EstimateTokens(Prompt(batch)) <= LongPromptTokens {
		// The shown body is capped per message, so build a batch of many.
		for i := 0; i < 40; i++ {
			message := batch.Messages[0]
			message.RecordID, message.Ordinal = message.RecordID+string(rune('a'+i%26)), int64(100+i)
			batch.Messages = append(batch.Messages, message)
		}
	}
	if EstimateTokens(Prompt(batch)) <= LongPromptTokens {
		t.Fatalf("fixture is not long: %d tokens", EstimateTokens(Prompt(batch)))
	}
	thinking := &scripted{replies: []Completion{{Content: validReply, FinishReason: "stop"}}}
	if _, err := ExtractBatch(context.Background(), thinking, "m", batch, entities.RunScope{}); err != nil {
		t.Fatal(err)
	}
	if len(thinking.modes) != 1 || !thinking.modes[0] {
		t.Fatalf("a long prompt must start with thinking on: %v", thinking.modes)
	}
	// Owner 2026-09-26 (option B): a long prompt retries with thinking on again.
	retry := &scripted{replies: []Completion{{Content: "", FinishReason: "stop"}, {Content: validReply, FinishReason: "stop"}}}
	if _, err := ExtractBatch(context.Background(), retry, "m", batch, entities.RunScope{}); err != nil {
		t.Fatal(err)
	}
	if len(retry.modes) != 2 || !retry.modes[0] || !retry.modes[1] {
		t.Fatalf("a long prompt must retry with thinking on again: %v", retry.modes)
	}
}

func TestDecodeRejectsSchemaViolations(t *testing.T) {
	batch := testBatch()
	for name, reply := range map[string]string{
		"missing key":       `{"people": [], "places": [], "organizations": []}`,
		"unknown label":     `{"people": [{"name": "A", "aliases": [], "participant": null, "mentions": [{"message": "m9", "text": "A"}]}], "places": [], "organizations": [], "events": []}`,
		"bad participant":   `{"people": [{"name": "A", "aliases": [], "participant": "P7", "mentions": []}], "places": [], "organizations": [], "events": []}`,
		"place participant": `{"people": [], "places": [{"name": "A", "aliases": [], "participant": "P1", "mentions": []}], "organizations": [], "events": []}`,
		"bad type":          `{"people": [], "places": [], "organizations": [], "events": [{"title": "t", "description": "", "message": "m1", "date": null, "when": null, "type": "party", "people": [], "places": [], "organizations": []}]}`,
		"bad date":          `{"people": [], "places": [], "organizations": [], "events": [{"title": "t", "description": "", "message": "m1", "date": "July 2", "when": null, "type": "court", "people": [], "places": [], "organizations": []}]}`,
		"not json":          "Here you go: {}",
	} {
		if _, err := Decode(reply, batch); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
	if _, err := Decode(validReply, batch); err != nil {
		t.Fatalf("valid reply rejected: %v", err)
	}
}

func TestClientBacksOffOnRateLimitAndSendsJSONObjectFormat(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		kwargs, _ := body["chat_template_kwargs"].(map[string]any)
		if body["response_format"].(map[string]any)["type"] != "json_object" || r.Header.Get("Authorization") != "Bearer test-key-123" || kwargs["thinking"] != false {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"model":"moonshotai/kimi-k3","choices":[{"message":{"content":"{}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, ModelID: "moonshotai/kimi-k3", APIKey: "test-key-123", MaxTokens: 2000})
	if err != nil {
		t.Fatal(err)
	}
	var slept time.Duration
	client.InitialBackoff = 500 * time.Millisecond // Retry-After (1s) is longer and wins
	client.Sleep = func(_ context.Context, d time.Duration) error { slept += d; return nil }
	off := false
	completion, err := client.Complete(context.Background(), []Message{{Role: "user", Content: "x"}}, CallOptions{Thinking: &off})
	if err != nil || completion.Content != "{}" || calls != 2 || slept != time.Second {
		t.Fatalf("err=%v completion=%+v calls=%d slept=%v", err, completion, calls, slept)
	}
}

func TestClientDoesNotRetryClientErrors(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"guided_json not supported"}`))
	}))
	defer server.Close()
	client, _ := NewClient(Config{BaseURL: server.URL, ModelID: "moonshotai/kimi-k3", APIKey: "test-key-123", MaxTokens: 2000})
	_, err := client.Complete(context.Background(), nil, CallOptions{})
	var transport *TransportError
	if !errors.As(err, &transport) || transport.Status != 400 || calls != 1 {
		t.Fatalf("a 400 is final: err=%v calls=%d", err, calls)
	}
}

func TestGLMAndTinyTokenBudgetsAreRefused(t *testing.T) {
	if err := (Config{BaseURL: "https://x", ModelID: "z-ai/glm-5.1", APIKey: "k", MaxTokens: 4000}).Validate(); err == nil || !strings.Contains(err.Error(), "GLM") {
		t.Fatalf("GLM must be refused: %v", err)
	}
	if err := (Config{BaseURL: "https://x", ModelID: "moonshotai/kimi-k3", APIKey: "k", MaxTokens: 800}).Validate(); err == nil {
		t.Fatal("a reasoning model needs at least 1500 tokens")
	}
}

func TestGroundingCleansPossessivesCasingAndLabels(t *testing.T) {
	reply := `{
 "people": [{"name": "katherine", "aliases": ["Katherine's", "Kat"], "participant": null, "mentions": [{"message": "m2", "text": "Katherine"}]}],
 "places": [], "organizations": [],
 "events": [{"title": "P2 confirms court with P1", "description": "P2 replied", "message": "m2", "date": null, "when": null, "type": "court", "people": [], "places": [], "organizations": []}]
}`
	completer := &scripted{replies: []Completion{{Content: reply, FinishReason: "stop"}}}
	outcome, err := ExtractBatch(context.Background(), completer, "m", testBatch(), entities.RunScope{})
	if err != nil || len(outcome.Entities) != 1 || len(outcome.Events) != 1 {
		t.Fatalf("err=%v outcome=%+v", err, outcome)
	}
	person := outcome.Entities[0]
	if person.Name != "Katherine" {
		t.Fatalf("the body's capitalized spelling should win, got %q", person.Name)
	}
	for _, alias := range person.Aliases {
		if strings.Contains(alias.Text, "'s") {
			t.Fatalf("a possessive is not an alias: %+v", person.Aliases)
		}
	}
	if got := outcome.Events[0].Title; got != "+1 (810) 555-0101 confirms court with the device owner" {
		t.Fatalf("participant labels must not reach the owner: %q", got)
	}
}

func TestBatchesAreBounded(t *testing.T) {
	var messages []entities.MessageView
	for i := 0; i < 95; i++ {
		messages = append(messages, entities.MessageView{RecordID: "r", Ordinal: int64(i), Body: "hello"})
	}
	batches := Batches(messages, nil, 0)
	if len(batches) != 3 || len(batches[0].Messages) != MaxBatchMessages || batches[2].Index != 2 {
		t.Fatalf("batches: %d first=%d", len(batches), len(batches[0].Messages))
	}
}
