// Byline: Claude Code · Sonnet 5.5 · 2026-10-02

package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// ConversationStore resolves the Workbench's conversations (an export file plus a
// conversation key) to Proffer runs and reads them for the Surreal send. It is
// separate from Store so the extraction Activities keep their narrow seam.
type ConversationStore interface {
	// ResolveConversation returns one RunRef per source version of the conversation
	// (its current normalized generation and the preview that generation was shown in).
	ResolveConversation(ctx context.Context, matterID string, ref flow.ConversationRef) ([]flow.RunRef, error)
	// ConversationInfo counts what PostgreSQL holds for the conversation.
	ConversationInfo(ctx context.Context, matterID string, ref flow.ConversationRef) (ConversationInfo, error)
	// ConversationMessages is one keyset page of the messages of some generations, oldest first.
	ConversationMessages(ctx context.Context, generationIDs []string, after MessageCursor, limit int) ([]ConversationMessage, error)
	// ConversationExtractions reads every run, entity and event the generations hold, every extractor.
	ConversationExtractions(ctx context.Context, generationIDs []string) (ExtractionExport, error)
}

// SourceFile is one source version of a conversation.
type SourceFile struct {
	SourceVersionID string `json:"source_version_id"`
	SourceKey       string `json:"source_key"`
	GenerationID    string `json:"generation_id"`
}

// ParticipantCount is an identifier and how many messages it appears in.
type ParticipantCount struct {
	Identifier string `json:"identifier"`
	Messages   int    `json:"messages"`
}

// ConversationInfo is the count and shape of one conversation.
type ConversationInfo struct {
	Messages     int                `json:"messages"`
	SourceFiles  []SourceFile       `json:"source_files"`
	Participants []ParticipantCount `json:"participants"`
	FirstAt      *time.Time         `json:"first_at,omitempty"`
	LastAt       *time.Time         `json:"last_at,omitempty"`
}

// MessageCursor is the keyset position after the last message read.
type MessageCursor struct {
	At *time.Time `json:"at,omitempty"`
	ID string     `json:"id,omitempty"`
}

// ConversationMessage is one message with what the Surreal record needs.
type ConversationMessage struct {
	ID              string
	OccurredAt      *time.Time
	Body            string
	Participants    json.RawMessage
	Certainty       string
	ProjectionKind  string
	AttachmentCount int
	SourceVersionID string
	Ordinal         int64
}

// ExportedRun is one extraction run with its generation.
type ExportedRun struct {
	ID           string          `json:"id"`
	GenerationID string          `json:"generation_id"`
	Extractor    string          `json:"extractor"`
	Version      string          `json:"extractor_version"`
	ModelID      string          `json:"model_id,omitempty"`
	Status       string          `json:"status"`
	CompareOnly  bool            `json:"compare_only"`
	Stats        json.RawMessage `json:"stats"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	FinishedAt   *time.Time      `json:"finished_at,omitempty"`
}

// ExportedEntity is one non-superseded entity candidate.
type ExportedEntity struct {
	ID           string          `json:"id"`
	RunID        string          `json:"run_id"`
	GenerationID string          `json:"generation_id"`
	Name         string          `json:"name"`
	EntityType   string          `json:"entity_type"`
	Confidence   float64         `json:"confidence"`
	ReviewState  string          `json:"review_state"`
	Attrs        json.RawMessage `json:"attrs"`
}

// ExportedEvent is one non-superseded event candidate.
type ExportedEvent struct {
	ID           string          `json:"id"`
	RunID        string          `json:"run_id"`
	GenerationID string          `json:"generation_id"`
	Title        string          `json:"title"`
	EventType    string          `json:"event_type"`
	OccurredAt   *time.Time      `json:"occurred_at,omitempty"`
	Confidence   float64         `json:"confidence"`
	ReviewState  string          `json:"review_state"`
	Attrs        json.RawMessage `json:"attrs"`
}

// ExtractionExport is everything extracted from some generations.
type ExtractionExport struct {
	Runs     []ExportedRun    `json:"runs"`
	Entities []ExportedEntity `json:"entities"`
	Events   []ExportedEvent  `json:"events"`
}
