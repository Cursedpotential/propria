// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package service holds the extraction operations shared by the Temporal
// Activities (modules/engine/activities/entity_extraction.go) and the
// starter's HTTP API (modules/engine/runtimeapi/entity_extraction_api.go),
// over one durable Store seam implemented in modules/engine/postgres.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// ErrNotFound means the run, record or entity does not exist.
var ErrNotFound = errors.New("not found")

// RunRow is one working.extraction_run row.
type RunRow struct {
	ID            string         `json:"id"`
	Extractor     string         `json:"extractor"`
	Version       string         `json:"extractor_version"`
	ModelID       string         `json:"model_id,omitempty"`
	PromptVersion string         `json:"prompt_version,omitempty"`
	Summary       string         `json:"source_summary"`
	Stats         map[string]any `json:"stats"`
}

// RunSummary is a run as the Review panel shows it.
type RunSummary struct {
	ID         string         `json:"id"`
	Extractor  string         `json:"extractor"`
	Version    string         `json:"extractor_version"`
	ModelID    string         `json:"model_id,omitempty"`
	Status     string         `json:"status"`
	Error      string         `json:"error,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Stats      map[string]any `json:"stats"`
}

// RegistryLookup narrows the committed entities a decision needs.
type RegistryLookup struct {
	IDs             []string
	NormalizedNames []string
	AliasTexts      []string
}

// EntityRow is one registry.entity insert.
type EntityRow struct {
	ID             string
	RegistryType   string
	Name           string
	NormalizedName string
	DataTier       string
	Confidence     float64
	CandidateID    string
	ReceiptID      string
}

// AliasRow is one registry.entity_alias insert.
type AliasRow struct {
	ID          string
	EntityID    string
	Text        string
	Kind        string
	Confidence  float64
	CandidateID string
}

// MentionRow is one working.entity_mention insert.
type MentionRow struct {
	ID         string
	Surface    string
	Kind       string
	RecordID   string
	Start      *int
	End        *int
	Snippet    string
	Method     string
	Confidence float64
}

// ResolutionRow is one working.entity_resolution insert.
type ResolutionRow struct {
	ID          string
	MentionID   string
	EntityID    string
	MatchMethod string
	ResolvedBy  string
	Score       float64
	Metrics     map[string]any
	CandidateID string
}

// EventCandidateRow is one timeline.event_candidate insert plus its typed
// source_available_from anchor.
type EventCandidateRow struct {
	ID                  string
	SourceRecordID      string
	SourceRecordVersion string
	SourceLocator       map[string]any
	ExtractionRunID     string
	Precision           string
	OccurredAt          *time.Time
	TemporalConfidence  float64
	Summary             string
	EventType           string
	EntityRefs          []string
	AvailableFrom       time.Time
	AnchorID            string
	AnchorKey           string
	AnchorMetadata      map[string]any
	ProvenanceDigest    []byte
	CandidateID         string
}

// TimelineMemberRow is one timeline.timeline_member insert.
type TimelineMemberRow struct {
	ID           string
	CollectionID string
	CandidateID  string
}

// Promotion marks one staged proposal as committed.
type Promotion struct {
	CandidateID string
	Table       string
	TargetID    string
}

// CommitReceipt is the working.extraction_run row recording a commit.
type CommitReceipt struct {
	ID        string
	Status    string
	Error     string
	Summary   string
	StartedAt time.Time
	Stats     map[string]any
}

// Reviewer attributes commit writes.
type Reviewer struct {
	Username string
	At       time.Time
}

// Store is the durable seam. Every write is idempotent on a deterministic
// id; every read is bounded.
type Store interface {
	ResolveRun(ctx context.Context, previewHandle string) (flow.RunRef, error)
	ParticipantAggregates(ctx context.Context, generationID string) ([]entities.ParticipantAggregate, error)
	MessagePage(ctx context.Context, generationID string, afterOrdinal int64, limit int) ([]entities.MessageView, error)
	MessageByID(ctx context.Context, generationID, recordID string) (entities.MessageView, error)
	RecordsMissing(ctx context.Context, generationID string, recordIDs []string) ([]string, error)

	BeginRun(ctx context.Context, run RunRow) error
	FinishRun(ctx context.Context, runID, status string, stats map[string]any, errText string) error
	ExtractionRuns(ctx context.Context, generationID string) ([]RunSummary, error)
	StageEntities(ctx context.Context, runID string, proposals []entities.Proposal) ([]string, error)
	StageEvents(ctx context.Context, runID string, proposals []events.Proposal) ([]string, error)
	CurrentEntities(ctx context.Context, generationID string) ([]entities.Proposal, error)
	CurrentEvents(ctx context.Context, generationID string) ([]events.Proposal, error)
	PendingEntitiesOfRuns(ctx context.Context, runIDs []string) ([]entities.Proposal, error)
	ReplaceEntities(ctx context.Context, runID string, ids []string, insert []entities.Proposal, supersede []string) error
	ReplaceEvents(ctx context.Context, runID string, ids []string, insert []events.Proposal, supersede []string) error
	Registry(ctx context.Context, lookup RegistryLookup) ([]entities.RegistryEntity, error)
	SearchRegistry(ctx context.Context, query string, limit int) ([]entities.RegistryEntity, error)

	WriteEntities(ctx context.Context, rows []EntityRow) (int, error)
	WriteAliases(ctx context.Context, rows []AliasRow) (int, error)
	WriteMentions(ctx context.Context, mentions []MentionRow, resolutions []ResolutionRow, reviewer Reviewer, receiptID string) (int, int, error)
	WriteEventCandidates(ctx context.Context, rows []EventCandidateRow, reviewer Reviewer) (int, error)
	EnsureCollection(ctx context.Context, slug, title string) (string, error)
	WriteTimelineMembers(ctx context.Context, rows []TimelineMemberRow) (int, error)
	FinalizeCommit(ctx context.Context, receipt CommitReceipt, entityPromotions, eventPromotions []Promotion) error
}
