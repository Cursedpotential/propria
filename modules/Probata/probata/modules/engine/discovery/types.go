// Byline: Codex · GPT-6.1-sol · 2026-10-11
// Package discovery defines optional source-linked analysis without changing normalized inputs.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	MaxRecords    = 400
	MaxFeedback   = 256
	MaxCandidates = 100
	MaxBytes      = 8 << 20
	BatchSchema   = "adaptive-discovery.batch.v1"
	ResultSchema  = "adaptive-discovery.result.v1"
	ModelVersion  = "discovery-cpu-v1"
)

var (
	ErrInvalid     = errors.New("discovery contract invalid")
	ErrNotFound    = errors.New("discovery record not found")
	ErrConflict    = errors.New("discovery revision or idempotency conflict")
	ErrScope       = errors.New("discovery selection or case scope differs")
	ErrUnavailable = errors.New("discovery input unavailable")
)

// Scope selects one operating policy and case; input IDs, output value, no effects.
type Scope struct {
	Mode        string `json:"mode"`
	MatterID    string `json:"matter_id"`
	CourtCaseID string `json:"court_case_id"`
}

// Actor is the authenticated operator, separate from a speaker or attributed conduct actor.
type Actor struct {
	SubjectUID string `json:"subject_uid"`
	Username   string `json:"username"`
}

// Filters selects source time/participants/platforms before scoring; unknown availability is excluded as-lived.
type Filters struct {
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	Participants []string   `json:"participants,omitempty"`
	Platforms    []string   `json:"platforms,omitempty"`
	Perspective  string     `json:"perspective"`
	Cutoff       *time.Time `json:"cutoff,omitempty"`
}

// Hypothesis configures one independent detector; input text/prototypes, output value, no effects.
type Hypothesis struct {
	HypothesisID string   `json:"hypothesis_id"`
	Text         string   `json:"text"`
	Category     string   `json:"category,omitempty"`
	Prototypes   []string `json:"prototypes,omitempty"`
}

// Dataset is an owner-selected server definition; callers cannot supply URLs, SQL or table names.
type Dataset struct {
	DatasetID         string   `json:"dataset_id"`
	Label             string   `json:"label"`
	InputBackend      string   `json:"input_backend"`
	SelectionRevision string   `json:"selection_revision"`
	GenerationIDs     []string `json:"generation_ids"`
	Scope             Scope    `json:"scope"`
	SurrealTable      string   `json:"-"`
}

// RunRequest starts one finite analysis; actor is supplied by the authenticated server boundary.
type RunRequest struct {
	RunID             string       `json:"run_id"`
	Scope             Scope        `json:"scope"`
	Actor             Actor        `json:"actor"`
	InputBackend      string       `json:"input_backend"`
	DatasetID         string       `json:"dataset_id"`
	SelectionRevision string       `json:"selection_revision"`
	GenerationIDs     []string     `json:"generation_ids"`
	Hypotheses        []Hypothesis `json:"hypotheses"`
	Filters           Filters      `json:"filters"`
	MaxRecords        int          `json:"max_records"`
	MaxCandidates     int          `json:"max_candidates"`
}

// Ref identifies one immutable external derivative; its checksum is unrelated to source custody.
type Ref struct {
	URI    string `json:"uri"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// Run is canonical PG control state; durable dispatch survives failed Temporal starts and process restarts.
type Run struct {
	Request          RunRequest `json:"request"`
	Revision         int64      `json:"revision"`
	Status           string     `json:"status"`
	InputRecordCount int        `json:"input_record_count"`
	InputTruncated   bool       `json:"input_truncated"`
	ManifestRef      *Ref       `json:"manifest_ref,omitempty"`
	ScoreRevision    int64      `json:"score_revision"`
	FeedbackRevision int64      `json:"feedback_revision"`
	WorkflowID       string     `json:"workflow_id"`
	TemporalRunID    string     `json:"temporal_run_id"`
	DispatchPending  bool       `json:"dispatch_pending"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	FailureStage     string     `json:"failure_stage,omitempty"`
	Error            string     `json:"error,omitempty"`
}

// Participant preserves the source's recorded sender/recipient roles without inferring conduct actors.
type Participant struct {
	Role        string `json:"role"`
	Identifier  string `json:"identifier"`
	DisplayName string `json:"display_name,omitempty"`
}

// Locator is a genuine typed half-open source range, never an invented navigation coordinate.
type Locator struct {
	LocatorID                     string          `json:"locator_id"`
	SourceVersionID               string          `json:"source_version_id"`
	CoordinateSystem              string          `json:"coordinate_system"`
	RangeStart                    int64           `json:"range_start"`
	RangeEnd                      int64           `json:"range_end"`
	Projection                    json.RawMessage `json:"projection"`
	ExactSliceSHA256              string          `json:"exact_slice_sha256,omitempty"`
	VerificationActivityReceiptID string          `json:"verification_activity_receipt_id,omitempty"`
}

// SourceRecord is the shared bounded PG/Surreal input; input_revision checks projection freshness, never custody.
type SourceRecord struct {
	RecordID                    string        `json:"record_id"`
	SourceVersionID             string        `json:"source_version_id"`
	GenerationID                string        `json:"generation_id"`
	Ordinal                     int64         `json:"ordinal"`
	RecordType                  string        `json:"record_type"`
	RecordRevision              string        `json:"record_revision"`
	InputRevision               string        `json:"input_revision"`
	Platform                    string        `json:"platform"`
	SourceKind                  string        `json:"source_kind"`
	ConversationID              string        `json:"conversation_id"`
	ContextUncertainty          []string      `json:"context_uncertainty,omitempty"`
	GroupRevision               string        `json:"group_revision,omitempty"`
	WorkingMessageID            string        `json:"working_message_id,omitempty"`
	ImportedSourceID            string        `json:"imported_source_id,omitempty"`
	ImportedThreadID            string        `json:"imported_thread_id,omitempty"`
	ImportedMessageID           string        `json:"imported_message_id,omitempty"`
	NativeSourceAvailableFrom   *time.Time    `json:"native_source_available_from,omitempty"`
	GroupKnowledgeAvailableFrom *time.Time    `json:"group_knowledge_available_from,omitempty"`
	ExplicitEpisodeID           string        `json:"explicit_episode_id"`
	OccurredAt                  *time.Time    `json:"occurred_at,omitempty"`
	SourceAvailableFrom         *time.Time    `json:"source_available_from,omitempty"`
	Text                        string        `json:"text"`
	Participants                []Participant `json:"participants"`
	Locators                    []Locator     `json:"locators"`
	SourceRef                   string        `json:"source_ref,omitempty"`
	SourceSHA256                string        `json:"source_sha256,omitempty"`
	SourceObjectID              string        `json:"source_object_id,omitempty"`
	SourceProviderVersion       string        `json:"source_provider_version,omitempty"`
	SpeechForm                  string        `json:"speech_form,omitempty"`
	AttributedActorIDs          []string      `json:"attributed_actor_ids,omitempty"`
	AttributionBasis            string        `json:"attribution_basis,omitempty"`
}

// Episode groups only explicit links or native conversations scoped to a source version.
type Episode struct {
	EpisodeID string   `json:"episode_id"`
	RecordIDs []string `json:"record_ids"`
}

// Citation grounds one analytical span in an input record and its recorded locator.
type Citation struct {
	RecordID  string `json:"record_id"`
	LocatorID string `json:"locator_id"`
	Quote     string `json:"quote"`
	StartChar int    `json:"start_char"`
	EndChar   int    `json:"end_char"`
}

// Candidate is a ranked hypothesis with independent labels and separate speaker/conduct attribution.
type Candidate struct {
	CandidateID        string             `json:"candidate_id"`
	EpisodeID          string             `json:"episode_id"`
	Title              string             `json:"title"`
	Summary            string             `json:"summary"`
	Labels             []string           `json:"labels"`
	Score              float64            `json:"score"`
	LabelScores        map[string]float64 `json:"label_scores,omitempty"`
	DetectorScores     map[string]float64 `json:"detector_scores,omitempty"`
	SpeakerIDs         []string           `json:"speaker_ids"`
	AttributedActorIDs []string           `json:"attributed_actor_ids"`
	AttributionBasis   string             `json:"attribution_basis"`
	Citations          []Citation         `json:"citations"`
	Rationale          string             `json:"rationale"`
	Uncertainty        []string           `json:"uncertainty"`
	ReviewState        string             `json:"review_state,omitempty"`
}

// FeedbackRequest is a compare-and-swap owner correction, not a promotion or intake gate.
type FeedbackRequest struct {
	FeedbackID         string     `json:"feedback_id"`
	CandidateID        string     `json:"candidate_id"`
	ExpectedRevision   int64      `json:"expected_revision"`
	Kind               string     `json:"kind"`
	Labels             []string   `json:"labels,omitempty"`
	AttributedActorIDs []string   `json:"attributed_actor_ids,omitempty"`
	Note               string     `json:"note"`
	HypothesisID       string     `json:"hypothesis_id,omitempty"`
	RecordIDs          []string   `json:"record_ids,omitempty"`
	EpisodeGroups      [][]string `json:"episode_groups,omitempty"`
}

// Feedback retains the original act and example context so future runs can learn from a durable decision.
type Feedback struct {
	RunID string `json:"run_id"`
	FeedbackRequest
	Revision       int64             `json:"revision"`
	Actor          Actor             `json:"actor"`
	CreatedAt      time.Time         `json:"created_at"`
	ExampleText    string            `json:"example_text,omitempty"`
	PreviousLabels []string          `json:"previous_labels"`
	InputPins      map[string]string `json:"input_pins,omitempty"`
}

// Batch holds scoring bodies outside Temporal history with the exact feedback and input selection revision.
type Batch struct {
	Schema            string          `json:"schema"`
	RunID             string          `json:"run_id"`
	Scope             Scope           `json:"scope"`
	DatasetID         string          `json:"dataset_id"`
	SelectionRevision string          `json:"selection_revision"`
	FeedbackRevision  int64           `json:"feedback_revision"`
	Records           []SourceRecord  `json:"records"`
	Episodes          []Episode       `json:"episodes"`
	Hypotheses        []Hypothesis    `json:"hypotheses"`
	Feedback          []Feedback      `json:"feedback"`
	Filters           Filters         `json:"filters"`
	ModelVersion      string          `json:"model_version"`
	MaxCandidates     int             `json:"max_candidates"`
	Truncated         bool            `json:"truncated"`
	Config            json.RawMessage `json:"config,omitempty"`
	Scoring           json.RawMessage `json:"scoring,omitempty"`
}

// ScoreRequest is the complete reference-only Python Activity payload.
type ScoreRequest struct {
	RunID            string `json:"run_id"`
	BatchRef         Ref    `json:"batch_ref"`
	FeedbackRevision int64  `json:"feedback_revision"`
}

// ScoreReceipt is the bounded Python result reference and finite accounting.
type ScoreReceipt struct {
	RunID            string `json:"run_id"`
	ResultRef        Ref    `json:"result_ref"`
	CandidateCount   int    `json:"candidate_count"`
	FeedbackRevision int64  `json:"feedback_revision"`
	ModelVersion     string `json:"model_version"`
}

// Result is an immutable external score generation; no bodies cross Temporal history.
type Result struct {
	Schema                string          `json:"schema"`
	RunID                 string          `json:"run_id"`
	FeedbackRevision      int64           `json:"feedback_revision"`
	ModelVersion          string          `json:"model_version"`
	ScoringPolicyRevision string          `json:"score_revision,omitempty"`
	Pins                  json.RawMessage `json:"pins,omitempty"`
	Candidates            []Candidate     `json:"candidates"`
	Abstained             int             `json:"abstained,omitempty"`
	ModelMetadata         json.RawMessage `json:"model_metadata,omitempty"`
	Counts                json.RawMessage `json:"counts,omitempty"`
}

// CandidatePage is a bounded HTTP queue; current_revision is the feedback compare-and-swap token.
type CandidatePage struct {
	Items           []Candidate `json:"items"`
	CurrentRevision int64       `json:"current_revision"`
	Revision        int64       `json:"revision"`
	NextCursor      string      `json:"next_cursor,omitempty"`
}

// Context includes full bounded source records at the current candidate revision for HTTP only.
type Context struct {
	Candidate       Candidate      `json:"candidate"`
	Records         []SourceRecord `json:"records"`
	CurrentRevision int64          `json:"current_revision"`
	Revision        int64          `json:"revision"`
	ReviewState     string         `json:"review_state"`
}

// FeedbackReceipt reports the persisted decision and pending durable rescore dispatch.
type FeedbackReceipt struct {
	Feedback Feedback `json:"feedback"`
	Run      Run      `json:"run"`
}

// Store owns analysis state in PG; inputs are scoped values, outputs bounded state, effects analysis writes only.
type Store interface {
	Datasets(context.Context, Scope) ([]Dataset, error)
	CreateRun(context.Context, RunRequest) (Run, error)
	ReadRun(context.Context, string, Scope) (Run, error)
	ListCandidates(context.Context, string, Scope, string, int) (CandidatePage, error)
	ReadContext(context.Context, string, Scope, string) (Context, error)
	SubmitFeedback(context.Context, string, Scope, FeedbackRequest, Actor) (FeedbackReceipt, error)
	FeedbackForRun(context.Context, string, Scope) ([]Feedback, error)
	SaveManifest(context.Context, string, Scope, int64, Ref, Batch) error
	LoadManifest(context.Context, string, Scope, int64) (Ref, Batch, error)
	PersistResult(context.Context, string, Scope, ScoreReceipt, Result, Batch) (Run, error)
	RecordDispatch(context.Context, string, Scope, int64, string, string) error
	PendingDispatch(context.Context, int) ([]Run, error)
	MarkFailed(context.Context, string, Scope, int64, string) error
}

// SourceReader resolves server-bound selections read-only before an external immutable batch is retained.
type SourceReader interface {
	ReadSources(context.Context, RunRequest) ([]SourceRecord, bool, error)
}

// SourcePin binds feedback examples to normalized bytes and explicit merged membership revision.
func SourcePin(r SourceRecord) string {
	if r.GroupRevision == "" {
		return r.InputRevision
	}
	return r.InputRevision + "/" + r.GroupRevision
}
