// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Package flow holds the two Temporal workflows of entity/event extraction
// and their wire types. Everything that crosses Temporal history is a
// reference or a count; proposals themselves stay in PostgreSQL staging.
//
//	EntityExtractionWorkflow  propose (rules) -> extract (model) -> reconcile
//	ExtractionCommitWorkflow  validate -> entities -> aliases -> mentions ->
//	                          events -> timeline members -> finalize -> projection
package flow

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
)

// Workflow and Activity names. These are standalone Activities (like the
// batch-import ones): the extraction workflows are their only callers and
// none of them is one of the 26 Proffer stages.
const (
	ExtractionWorkflowName = "entity_extraction_workflow"
	CommitWorkflowName     = "extraction_commit_workflow"

	ProposeRulesActivity    = "propose_entities_rules_activity"
	ExtractModelActivity    = "extract_entities_events_model_activity"
	ReconcileActivity       = "reconcile_entity_proposals_activity"
	ValidateCommitActivity  = "validate_extraction_commit_activity"
	CommitEntitiesActivity  = "commit_entities_activity"
	CommitAliasesActivity   = "commit_entity_aliases_activity"
	CommitMentionsActivity  = "commit_entity_mentions_activity"
	CommitEventsActivity    = "commit_event_candidates_activity"
	CommitMembersActivity   = "commit_timeline_members_activity"
	FinalizeCommitActivity  = "finalize_extraction_commit_activity"
	BuildProjectionActivity = "build_timeline_generation_activity"

	StatusQuery = "status"

	// DefaultProjectionTaskQueue is where the Python worker registers
	// build_timeline_generation_activity (server/temporal/worker.py).
	DefaultProjectionTaskQueue = "evidence-pipeline"
	// DefaultCollectionSlug is the single-case timeline collection
	// (D-072) that server/timeline/generation.py builds by default.
	DefaultCollectionSlug = "primary"
)

// Step statuses shown in Review.
const (
	StepPending   = "pending"
	StepRunning   = "running"
	StepCompleted = "completed"
	StepSkipped   = "skipped"
	StepFailed    = "failed"
)

// Workflow outcomes.
const (
	OutcomeRunning          = "running"
	OutcomeCompleted        = "completed"
	OutcomeCompletedFlagged = "completed_with_flags"
	OutcomeValidationFailed = "validation_failed"
	OutcomeFailed           = "failed"
	OutcomeCommitted        = "committed"
)

// Namespace for deterministic identities (UUIDv5).
var Namespace = uuid.MustParse("7b0c9a52-4c1e-5f58-9a33-0d6f3c2e8b17")

// DeterministicID derives a stable UUID (v5) so every retried write targets
// the same row.
func DeterministicID(parts ...string) string {
	return uuid.NewSHA1(Namespace, []byte(strings.Join(parts, "|"))).String()
}

// RunRef identifies the Review run a workflow works on.
type RunRef struct {
	PreviewHandle   string `json:"preview_handle"`
	GenerationID    string `json:"normalized_generation_id"`
	SourceVersionID string `json:"source_version_id"`
	MatterMode      string `json:"matter_mode"`
}

// Scope converts a run to the extraction scope.
func (r RunRef) Scope() entities.RunScope {
	return entities.RunScope{PreviewHandle: r.PreviewHandle, GenerationID: r.GenerationID, SourceVersionID: r.SourceVersionID}
}

// ExtractionRequest starts EntityExtractionWorkflow.
type ExtractionRequest struct {
	ExtractionID string         `json:"extraction_id"`
	Run          RunRef         `json:"run"`
	Actor        entities.Actor `json:"actor"`
	UseModel     bool           `json:"use_model"`
	RequestedAt  time.Time      `json:"requested_at"`
}

// RunID is the deterministic working.extraction_run id of one step.
func (r ExtractionRequest) RunID(step string) string {
	return DeterministicID("extraction_run", r.ExtractionID, step)
}

// InvalidBatch is one model batch whose reply failed validation twice.
type InvalidBatch struct {
	Index        int    `json:"index"`
	FirstOrdinal int64  `json:"first_ordinal"`
	LastOrdinal  int64  `json:"last_ordinal"`
	Reason       string `json:"reason"`
}

// ProposeResult is the bounded summary of a proposing Activity.
type ProposeResult struct {
	ExtractionRunID string         `json:"extraction_run_id"`
	Messages        int            `json:"messages"`
	Proposals       int            `json:"proposals"`
	Events          int            `json:"events"`
	Batches         int            `json:"batches"`
	Ungrounded      int            `json:"ungrounded"`
	InvalidBatches  []InvalidBatch `json:"invalid_batches,omitempty"`
	Skipped         bool           `json:"skipped,omitempty"`
	Reason          string         `json:"reason,omitempty"`
}

// ReconcileRequest names the runs whose partials are grouped.
type ReconcileRequest struct {
	Extraction ExtractionRequest `json:"extraction"`
	RunIDs     []string          `json:"run_ids"`
}

// ReconcileResult summarizes the grouping step.
type ReconcileResult struct {
	ExtractionRunID string `json:"extraction_run_id"`
	Proposals       int    `json:"proposals"`
	Superseded      int    `json:"superseded"`
	Matched         int    `json:"matched"`
	Dropped         int    `json:"dropped"`
}

// StepResult is one line of the status list.
type StepResult struct {
	Step   string          `json:"step"`
	Status string          `json:"status"`
	Detail string          `json:"detail,omitempty"`
	Counts map[string]int  `json:"counts,omitempty"`
	Ref    string          `json:"ref,omitempty"`
	Flags  []entities.Flag `json:"flags,omitempty"`
}

// Progress is the workflow's queryable status.
type Progress struct {
	Outcome string       `json:"outcome"`
	Steps   []StepResult `json:"steps"`
}

func (p *Progress) set(step string, status, detail string, counts map[string]int) {
	for i := range p.Steps {
		if p.Steps[i].Step == step {
			p.Steps[i].Status, p.Steps[i].Detail = status, detail
			if counts != nil {
				p.Steps[i].Counts = counts
			}
			return
		}
	}
	p.Steps = append(p.Steps, StepResult{Step: step, Status: status, Detail: detail, Counts: counts})
}

// CommitRequest starts ExtractionCommitWorkflow and is every commit step's
// input. Digest binds the commit to the validated proposal set.
type CommitRequest struct {
	CommitID            string         `json:"commit_id"`
	WorkflowID          string         `json:"workflow_id"`
	Run                 RunRef         `json:"run"`
	Actor               entities.Actor `json:"actor"`
	Digest              string         `json:"digest"`
	RequestedAt         time.Time      `json:"requested_at"`
	CollectionSlug      string         `json:"collection_slug"`
	ProjectionTaskQueue string         `json:"projection_task_queue"`
}

// ReceiptID is the working.extraction_run row that records the commit.
func (r CommitRequest) ReceiptID() string { return DeterministicID("extraction_commit", r.CommitID) }

// CommitStepResult is one commit Activity's bounded summary.
type CommitStepResult struct {
	Written int                 `json:"written"`
	Skipped int                 `json:"skipped"`
	Detail  string              `json:"detail,omitempty"`
	Counts  map[string]int      `json:"counts,omitempty"`
	Report  *commitcheck.Report `json:"report,omitempty"`
}

// FinalizeRequest closes a commit, successful or not.
type FinalizeRequest struct {
	Commit   CommitRequest  `json:"commit"`
	Outcome  string         `json:"outcome"`
	Error    string         `json:"error,omitempty"`
	Counts   map[string]int `json:"counts"`
	Promote  bool           `json:"promote"`
	FailedAt string         `json:"failed_at,omitempty"`
}

// ProjectionRequest is the Python build_timeline_generation_activity input.
type ProjectionRequest struct {
	CollectionSlug string `json:"collection_slug"`
	CreatedBy      string `json:"created_by"`
	CommitID       string `json:"commit_id"`
}

// ProjectionResult mirrors server.timeline.models.GenerationResult.
type ProjectionResult struct {
	GenerationID                     string   `json:"generation_id"`
	Sequence                         int64    `json:"sequence"`
	Created                          bool     `json:"created"`
	MemberCount                      int      `json:"member_count"`
	SkippedUnresolvedGovernedMembers []string `json:"skipped_unresolved_governed_members"`
}
