// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Conversation-level extraction and the Surreal send. A conversation is what the
// Workbench lists: one export file plus one conversation key, which the import
// splits into one or more source versions (derived thread files), each with a
// current normalized generation. Everything that crosses Temporal history is a
// reference or a count; messages stay in PostgreSQL.
//
//	extraction_request_workflow   resolve -> per run and per chosen extractor -> report
//	external_extraction_workflow  one external extractor over one run, a window at a time
//	send_to_surreal_workflow      plan -> upsert -> upsert extractions -> verify, per conversation

package flow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// Workflow and Activity names for the conversation-level surface.
const (
	RequestWorkflowName  = "extraction_request_workflow"
	ExternalWorkflowName = "external_extraction_workflow"
	SendWorkflowName     = "send_to_surreal_workflow"

	ResolveConversationsActivity = "resolve_conversations_activity"
	BeginExternalRunActivity     = "begin_external_extraction_run_activity"
	StageExternalPageActivity    = "stage_external_extraction_page_activity"
	FinishExternalRunActivity    = "finish_external_extraction_run_activity"

	PlanSurrealSendActivity        = "plan_surreal_send_activity"
	UpsertConversationActivity     = "upsert_conversation_to_surreal_activity"
	UpsertExtractionsActivity      = "upsert_extractions_to_surreal_activity"
	VerifySurrealSendActivity      = "verify_surreal_send_activity"
	SemanticaExtractActivity       = "extract_entities_events_semantica_activity"
	LangExtractExtractActivity     = "extract_entities_events_langextract_activity"
	AutoExtractionWorkflowIDPrefix = "auto-extraction:"

	// PythonTaskQueue is where the external extractor Activities are registered
	// (server/temporal/worker.py).
	PythonTaskQueue = DefaultProjectionTaskQueue

	// ExternalPageMessages is the window an external extractor reads per Activity call.
	ExternalPageMessages = 200
	// MaxExternalMessages bounds one external extractor run over one generation.
	MaxExternalMessages = 20000
	// MaxConversationsPerRequest bounds one click.
	MaxConversationsPerRequest = 200
)

// Extractor kinds.
const (
	KindInternal = "internal"
	KindExternal = "external"
)

// ExtractorSpec describes one selectable extractor. The registry is what the
// picker lists; every extractor stays selectable and every result is tagged with
// the id of the extractor that made it, so outputs can be compared side by side.
type ExtractorSpec struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	// Default marks the extractor that runs automatically after a Proffer commit
	// and that is pre-checked in the picker.
	Default bool `json:"default"`
	// RunExtractor is the working.extraction_run.extractor value the run carries
	// (the tag results are grouped by).
	RunExtractor string `json:"run_extractor"`
	// RunExtractors lists every working.extraction_run.extractor value that belongs to this extractor, so a viewer
	// can group runs by extractor: the default's three steps (rules, model, reconcile) are one extractor.
	RunExtractors []string `json:"run_extractors"`
	// Version is the extractor version recorded on the run row.
	Version string `json:"version"`
	// Activity and TaskQueue are set for external extractors.
	Activity  string `json:"-"`
	TaskQueue string `json:"-"`
	// CompareOnly extractors stage candidates that stay out of the Review
	// proposals (and so out of any registry commit) until the owner picks them.
	CompareOnly bool `json:"compare_only"`
}

// DefaultExtractorID is Go plus kimi-k3 on NVIDIA NIM (rules, model, reconcile).
const DefaultExtractorID = "go-kimi-k3"

// Registry lists every extractor that can be run. An extractor that is not here
// is not registerable; the reasons are in docs/LOG.md and the task report.
var Registry = []ExtractorSpec{
	{
		ID: DefaultExtractorID, Label: "Go + kimi-k3 (default)",
		Description:  "Participant rules, then kimi-k3 on NVIDIA NIM over the messages, then reconcile into one proposal per entity. Feeds the Review proposals.",
		Kind:         KindInternal,
		Default:      true,
		RunExtractor: "probata.extract.model", Version: "1",
		RunExtractors: []string{entities.RulesExtractor, "probata.extract.model", entities.ReconcileExtractor},
	},
	{
		ID: "semantica", Label: "Semantica (patterns)",
		Description:  "The vendored Semantica pattern extractors (names, organizations, places, dated events). No model call; fast and literal.",
		Kind:         KindExternal,
		RunExtractor: "semantica", Version: "1", RunExtractors: []string{"semantica"},
		Activity: SemanticaExtractActivity, TaskQueue: PythonTaskQueue, CompareOnly: true,
	},
	{
		ID: "langextract", Label: "LangExtract (kimi-k3)",
		Description:  "Google LangExtract over kimi-k3 on NVIDIA NIM: few-shot entity and event extraction with source grounding.",
		Kind:         KindExternal,
		RunExtractor: "langextract", Version: "1", RunExtractors: []string{"langextract"},
		Activity: LangExtractExtractActivity, TaskQueue: PythonTaskQueue, CompareOnly: true,
	},
}

// ExtractorByID returns a registered extractor.
func ExtractorByID(id string) (ExtractorSpec, bool) {
	for _, spec := range Registry {
		if spec.ID == id {
			return spec, true
		}
	}
	return ExtractorSpec{}, false
}

// DefaultExtractor returns the default extractor.
func DefaultExtractor() ExtractorSpec {
	for _, spec := range Registry {
		if spec.Default {
			return spec
		}
	}
	return Registry[0]
}

// NormalizeExtractors validates a selection: unknown ids are an error, duplicates
// collapse, an empty selection means the default, and the order is the registry's.
func NormalizeExtractors(ids []string) ([]ExtractorSpec, error) {
	if len(ids) == 0 {
		return []ExtractorSpec{DefaultExtractor()}, nil
	}
	chosen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if _, ok := ExtractorByID(id); !ok {
			return nil, fmt.Errorf("unknown extractor %q", id)
		}
		chosen[id] = true
	}
	var out []ExtractorSpec
	for _, spec := range Registry {
		if chosen[spec.ID] {
			out = append(out, spec)
		}
	}
	return out, nil
}

// ConversationRef names one conversation the way the Workbench lists it.
type ConversationRef struct {
	ExportKey string `json:"export_key"`
	Conv      string `json:"conv"`
}

// Key is a stable text key for the conversation.
func (c ConversationRef) Key() string { return c.ExportKey + "\n" + c.Conv }

// Label is the short name used in progress lines.
func (c ConversationRef) Label() string {
	name := c.ExportKey
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name + " / " + c.Conv
}

// ConversationTarget is a conversation resolved to the runs it consists of.
type ConversationTarget struct {
	Ref     ConversationRef `json:"ref"`
	Runs    []RunRef        `json:"runs"`
	Skipped string          `json:"skipped,omitempty"`
}

// ResolveRequest asks the resolver for the runs of some conversations.
type ResolveRequest struct {
	MatterID      string            `json:"matter_id"`
	Conversations []ConversationRef `json:"conversations"`
}

// ResolveResult is the resolver's answer, one target per requested conversation.
type ResolveResult struct {
	Targets []ConversationTarget `json:"targets"`
}

// RequestInput starts extraction_request_workflow. Conversations are resolved to
// runs; Runs adds runs that are already known (the automatic start after a
// Proffer commit knows its own run).
type RequestInput struct {
	OperatingMode string            `json:"operating_mode,omitempty"`
	CourtCaseID   string            `json:"court_case_id,omitempty"`
	RequestID     string            `json:"request_id"`
	MatterID      string            `json:"matter_id"`
	Conversations []ConversationRef `json:"conversations,omitempty"`
	Runs          []RunRef          `json:"runs,omitempty"`
	Extractors    []string          `json:"extractors"`
	Actor         entities.Actor    `json:"actor"`
	Auto          bool              `json:"auto,omitempty"`
	RequestedAt   time.Time         `json:"requested_at"`
}

// Validate bounds a request.
func (r RequestInput) Validate() error {
	if strings.TrimSpace(r.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if len(r.Conversations)+len(r.Runs) == 0 {
		return fmt.Errorf("name at least one conversation")
	}
	if len(r.Conversations) > MaxConversationsPerRequest {
		return fmt.Errorf("at most %d conversations per request", MaxConversationsPerRequest)
	}
	for _, c := range r.Conversations {
		if strings.TrimSpace(c.ExportKey) == "" || strings.TrimSpace(c.Conv) == "" {
			return fmt.Errorf("every conversation needs an export_key and a conv")
		}
	}
	if len(r.Conversations) > 0 && strings.TrimSpace(r.MatterID) == "" {
		return fmt.Errorf("matter_id is required to resolve conversations")
	}
	_, err := NormalizeExtractors(r.Extractors)
	return err
}

// ExtractorSelectionID is a stable id for the sorted selection (workflow ids).
func ExtractorSelectionID(specs []ExtractorSpec) string {
	ids := make([]string, len(specs))
	for i, spec := range specs {
		ids[i] = spec.ID
	}
	sort.Strings(ids)
	return strings.Join(ids, "+")
}

// ExternalRunInput starts external_extraction_workflow for one run.
type ExternalRunInput struct {
	ExtractionID string         `json:"extraction_id"`
	Run          RunRef         `json:"run"`
	Extractor    string         `json:"extractor"`
	Actor        entities.Actor `json:"actor"`
}

// ExternalRunID is the deterministic working.extraction_run id of one external run.
func (i ExternalRunInput) ExternalRunID() string {
	return DeterministicID("extraction_run", i.ExtractionID, "external", i.Extractor, i.Run.GenerationID)
}

// ExternalRunStart opens the run row.
type ExternalRunStart struct {
	Input   ExternalRunInput `json:"input"`
	Spec    ExtractorSpec    `json:"spec"`
	Version string           `json:"version,omitempty"`
	ModelID string           `json:"model_id,omitempty"`
}

// ExternalPageRequest is the input of an external extractor Activity (the Python
// side reads the same JSON field names).
type ExternalPageRequest struct {
	GenerationID    string `json:"generation_id"`
	SourceVersionID string `json:"source_version_id"`
	PreviewHandle   string `json:"preview_handle"`
	Extractor       string `json:"extractor"`
	AfterOrdinal    int64  `json:"after_ordinal"`
	Limit           int    `json:"limit"`
}

// StageExternalPage stages what an external extractor returned for one window.
type StageExternalPage struct {
	Input   ExternalRunInput    `json:"input"`
	RunID   string              `json:"run_id"`
	Request ExternalPageRequest `json:"request"`
	// Page is the extractor's reply, JSON as it came back (kept opaque here).
	Page json.RawMessage `json:"page"`
}

// StagePageResult is the bounded summary of one staged window.
type StagePageResult struct {
	Entities   int    `json:"entities"`
	Events     int    `json:"events"`
	Messages   int    `json:"messages"`
	Ungrounded int    `json:"ungrounded"`
	Invalid    bool   `json:"invalid,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Last       int64  `json:"last_ordinal"`
	// Done is true when the extractor reached the end of the run's messages.
	Done bool `json:"done"`
	// Skipped is true when the extractor could not run (not configured); Reason says why.
	Skipped bool `json:"skipped,omitempty"`
}

// FinishExternalRun closes the run row.
type FinishExternalRun struct {
	Run    RunRef         `json:"run"`
	RunID  string         `json:"run_id"`
	Status string         `json:"status"`
	Error  string         `json:"error,omitempty"`
	Stats  map[string]any `json:"stats,omitempty"`
}

// SendInput starts send_to_surreal_workflow.
type SendInput struct {
	OperatingMode      string            `json:"operating_mode,omitempty"`
	CourtCaseID        string            `json:"court_case_id,omitempty"`
	RequestID          string            `json:"request_id"`
	MatterID           string            `json:"matter_id"`
	Conversations      []ConversationRef `json:"conversations"`
	IncludeExtractions bool              `json:"include_extractions"`
	Actor              entities.Actor    `json:"actor"`
	RequestedAt        time.Time         `json:"requested_at"`
}

// Validate bounds a send.
func (s SendInput) Validate() error {
	if strings.TrimSpace(s.RequestID) == "" || strings.TrimSpace(s.MatterID) == "" {
		return fmt.Errorf("request_id and matter_id are required")
	}
	if len(s.Conversations) == 0 || len(s.Conversations) > MaxConversationsPerRequest {
		return fmt.Errorf("send between 1 and %d conversations", MaxConversationsPerRequest)
	}
	for _, c := range s.Conversations {
		if strings.TrimSpace(c.ExportKey) == "" || strings.TrimSpace(c.Conv) == "" {
			return fmt.Errorf("every conversation needs an export_key and a conv")
		}
	}
	return nil
}

// SendTarget is one conversation in a send, with its runs.
type SendTarget struct {
	OperatingMode string          `json:"operating_mode,omitempty"`
	CourtCaseID   string          `json:"court_case_id,omitempty"`
	MatterID      string          `json:"matter_id"`
	Ref           ConversationRef `json:"ref"`
	Request       string          `json:"request_id"`
	Actor         entities.Actor  `json:"actor"`
}

// SendPlan is what a conversation holds before it is sent.
type SendPlan struct {
	Messages     int       `json:"messages"`
	SourceFiles  int       `json:"source_files"`
	Participants int       `json:"participants"`
	FirstAt      time.Time `json:"first_at,omitempty"`
	LastAt       time.Time `json:"last_at,omitempty"`
	ThreadID     string    `json:"thread_id"`
	Generations  []string  `json:"generations"`
}

// SendWritten is the count one upsert step wrote.
type SendWritten struct {
	Threads  int `json:"threads"`
	Messages int `json:"messages"`
	Entities int `json:"entities"`
	Events   int `json:"events"`
	Runs     int `json:"runs"`
}

// SendVerified is what the Surreal side holds after the send, read back.
type SendVerified struct {
	Messages int  `json:"messages"`
	Entities int  `json:"entities"`
	Events   int  `json:"events"`
	Match    bool `json:"match"`
}

// SendReceipt is the receipt of one conversation.
type SendReceipt struct {
	Conversation ConversationRef `json:"conversation"`
	ThreadID     string          `json:"thread_id"`
	Plan         SendPlan        `json:"plan"`
	Written      SendWritten     `json:"written"`
	Verified     SendVerified    `json:"verified"`
	SentAt       time.Time       `json:"sent_at"`
}

// SurrealThreadID is the deterministic id of a conversation's Surreal thread
// record, so a resend lands on the same record.
func SurrealThreadID(matterID string, ref ConversationRef) string {
	return DeterministicID("surreal_thread", matterID, ref.ExportKey, ref.Conv)
}
