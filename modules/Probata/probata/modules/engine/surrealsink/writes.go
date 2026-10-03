// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package surrealsink

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// The conv_ tables. They are separate from every table the Family Court Toolkit
// owns in the same database (person, message, event, source, note, ...).
const (
	TableThread  = "conv_thread"
	TableMessage = "conv_message"
	TableRun     = "conv_extraction_run"
	TableEntity  = "conv_entity"
	TableEvent   = "conv_event"
)

// Thread is the conversation record.
type Thread struct {
	ID            string              `json:"id"`
	MatterID      string              `json:"matter_id"`
	ExportKey     string              `json:"export_key"`
	Conv          string              `json:"conv"`
	SourceFiles   []SourceFile        `json:"source_files"`
	Participants  []ThreadParticipant `json:"participants"`
	MessageCount  int                 `json:"message_count"`
	FirstAt       *time.Time          `json:"first_at,omitempty"`
	LastAt        *time.Time          `json:"last_at,omitempty"`
	SentAt        time.Time           `json:"sent_at"`
	SentBy        string              `json:"sent_by"`
	SendRequestID string              `json:"send_request_id"`
}

// SourceFile is one source version (derived thread file) the conversation was read from.
type SourceFile struct {
	SourceVersionID string `json:"source_version_id"`
	SourceKey       string `json:"source_key"`
	GenerationID    string `json:"generation_id"`
}

// ThreadParticipant is one identifier seen in the conversation.
type ThreadParticipant struct {
	Identifier string `json:"identifier"`
	Messages   int    `json:"messages"`
}

// Message is one message record.
type Message struct {
	ID              string          `json:"id"`
	ThreadID        string          `json:"thread_id"`
	MatterID        string          `json:"matter_id"`
	At              *time.Time      `json:"at,omitempty"`
	Body            string          `json:"body"`
	Sender          string          `json:"sender,omitempty"`
	Recipients      []string        `json:"recipients"`
	Participants    json.RawMessage `json:"participants"`
	Party           string          `json:"party,omitempty"`
	Attachments     int             `json:"attachments"`
	Certainty       string          `json:"certainty,omitempty"`
	SourceVersionID string          `json:"source_version_id"`
	Ordinal         int64           `json:"ordinal"`
}

// Run is one extraction run, tagged with its extractor.
type Run struct {
	ID          string          `json:"id"`
	ThreadID    string          `json:"thread_id"`
	Extractor   string          `json:"extractor"`
	Version     string          `json:"extractor_version"`
	ModelID     string          `json:"model_id,omitempty"`
	Status      string          `json:"status"`
	CompareOnly bool            `json:"compare_only"`
	Stats       json.RawMessage `json:"stats"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
}

// Entity is one extracted entity.
type Entity struct {
	ID           string          `json:"id"`
	ThreadID     string          `json:"thread_id"`
	RunID        string          `json:"run_id"`
	Extractor    string          `json:"extractor"`
	Name         string          `json:"name"`
	EntityType   string          `json:"entity_type"`
	Aliases      []string        `json:"aliases"`
	MentionCount int             `json:"mention_count"`
	Mentions     json.RawMessage `json:"mentions"`
	Confidence   float64         `json:"confidence"`
	ReviewState  string          `json:"review_state"`
}

// Event is one extracted event.
type Event struct {
	ID          string     `json:"id"`
	ThreadID    string     `json:"thread_id"`
	RunID       string     `json:"run_id"`
	Extractor   string     `json:"extractor"`
	Title       string     `json:"title"`
	EventType   string     `json:"event_type"`
	OccurredAt  *time.Time `json:"occurred_at,omitempty"`
	Precision   string     `json:"precision,omitempty"`
	Confidence  float64    `json:"confidence"`
	RecordIDs   []string   `json:"record_ids"`
	ReviewState string     `json:"review_state"`
}

func (c *Client) run(ctx context.Context, sql string, vars map[string]any) error {
	_, err := c.Query(ctx, sql, vars)
	return err
}

// UpsertThread writes the conversation record.
func (c *Client) UpsertThread(ctx context.Context, t Thread) error {
	return c.run(ctx, `UPSERT type::record('`+TableThread+`', $t.id) SET
  matter_id = $t.matter_id, export_key = $t.export_key, conv = $t.conv,
  case_ref = type::record('case_status', $case),
  source_files = $t.source_files, participants = $t.participants, message_count = $t.message_count,
  first_at = <option<datetime>> $t.first_at, last_at = <option<datetime>> $t.last_at,
  sent_at = <datetime> $t.sent_at, sent_by = $t.sent_by, send_request_id = $t.send_request_id;`,
		map[string]any{"t": t, "case": c.cfg.CaseRef})
}

// UpsertMessages writes one batch of messages (keep batches at or below 100).
func (c *Client) UpsertMessages(ctx context.Context, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}
	return c.run(ctx, `FOR $m IN $rows {
  UPSERT type::record('`+TableMessage+`', $m.id) SET
    thread = type::record('`+TableThread+`', $m.thread_id), matter_id = $m.matter_id,
    at = <option<datetime>> $m.at, body = $m.body, sender = $m.sender, recipients = $m.recipients,
    participants = $m.participants, party = $m.party, attachments = $m.attachments, certainty = $m.certainty,
    source_version_id = $m.source_version_id, ordinal = $m.ordinal;
};`, map[string]any{"rows": messages})
}

// UpsertRuns writes extraction run records.
func (c *Client) UpsertRuns(ctx context.Context, runs []Run) error {
	if len(runs) == 0 {
		return nil
	}
	return c.run(ctx, `FOR $r IN $rows {
  UPSERT type::record('`+TableRun+`', $r.id) SET
    thread = type::record('`+TableThread+`', $r.thread_id), extractor = $r.extractor,
    extractor_version = $r.extractor_version, model_id = $r.model_id, status = $r.status,
    compare_only = $r.compare_only, stats = $r.stats,
    started_at = <option<datetime>> $r.started_at, finished_at = <option<datetime>> $r.finished_at;
};`, map[string]any{"rows": runs})
}

// UpsertEntities writes extracted entities (batches of at most 100).
func (c *Client) UpsertEntities(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}
	return c.run(ctx, `FOR $e IN $rows {
  UPSERT type::record('`+TableEntity+`', $e.id) SET
    thread = type::record('`+TableThread+`', $e.thread_id), run = type::record('`+TableRun+`', $e.run_id),
    extractor = $e.extractor, name = $e.name, entity_type = $e.entity_type, aliases = $e.aliases,
    mention_count = $e.mention_count, mentions = $e.mentions, confidence = $e.confidence, review_state = $e.review_state;
};`, map[string]any{"rows": entities})
}

// UpsertEvents writes extracted events (batches of at most 100).
func (c *Client) UpsertEvents(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}
	return c.run(ctx, `FOR $e IN $rows {
  UPSERT type::record('`+TableEvent+`', $e.id) SET
    thread = type::record('`+TableThread+`', $e.thread_id), run = type::record('`+TableRun+`', $e.run_id),
    extractor = $e.extractor, title = $e.title, event_type = $e.event_type,
    occurred_at = <option<datetime>> $e.occurred_at, precision = $e.precision, confidence = $e.confidence,
    record_ids = $e.record_ids, review_state = $e.review_state;
};`, map[string]any{"rows": events})
}

// Counts is what the Surreal side holds for one thread.
type Counts struct {
	Messages int `json:"messages"`
	Entities int `json:"entities"`
	Events   int `json:"events"`
}

// CountThread reads the thread's record counts back (read-only).
func (c *Client) CountThread(ctx context.Context, threadID string) (Counts, error) {
	results, err := c.Query(ctx, `
SELECT count() AS n FROM `+TableMessage+` WHERE thread = type::record('`+TableThread+`', $id) GROUP ALL;
SELECT count() AS n FROM `+TableEntity+` WHERE thread = type::record('`+TableThread+`', $id) GROUP ALL;
SELECT count() AS n FROM `+TableEvent+` WHERE thread = type::record('`+TableThread+`', $id) GROUP ALL;`,
		map[string]any{"id": threadID})
	if err != nil {
		return Counts{}, err
	}
	if len(results) != 3 {
		return Counts{}, fmt.Errorf("surreal returned %d result sets, expected 3", len(results))
	}
	read := func(raw json.RawMessage) int {
		var rows []struct {
			N int `json:"n"`
		}
		if json.Unmarshal(raw, &rows) != nil || len(rows) == 0 {
			return 0
		}
		return rows[0].N
	}
	return Counts{Messages: read(results[0]), Entities: read(results[1]), Events: read(results[2])}, nil
}
