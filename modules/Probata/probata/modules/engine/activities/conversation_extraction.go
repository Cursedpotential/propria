// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
//
// Conversation-level extraction and the Surreal send, as Activities. Each one has a
// bounded input and output: messages and proposals stay in PostgreSQL (and, for the
// send, flow straight from PostgreSQL to surreal-case inside one Activity), and only
// references and counts move through Temporal history.
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

// surrealBatch is how many records go to Surreal in one request.
const surrealBatch = 100

// ConversationActivities implements the conversation extraction and Surreal send Activities.
type ConversationActivities struct {
	Store         service.Store
	Conversations service.ConversationStore
	// Surreal is nil when no connection is configured; the send Activities then fail with
	// a non-retryable error that says so.
	Surreal       *surrealsink.Client
	SurrealReason string
	Heartbeat     func(context.Context, any)
	Resume        func(context.Context, any) bool
	Now           func() time.Time
}

// NewConversationActivities binds production heartbeats and the clock.
func NewConversationActivities(store service.Store, conversations service.ConversationStore, sink *surrealsink.Client, sinkReason string) ConversationActivities {
	return ConversationActivities{
		Store: store, Conversations: conversations, Surreal: sink, SurrealReason: sinkReason,
		Heartbeat: func(ctx context.Context, details any) { activity.RecordHeartbeat(ctx, details) },
		Resume: func(ctx context.Context, into any) bool {
			if !activity.HasHeartbeatDetails(ctx) {
				return false
			}
			return activity.GetHeartbeatDetails(ctx, into) == nil
		},
		Now: time.Now,
	}
}

// RegisterConversationActivities installs every conversation Activity under its exact name.
func RegisterConversationActivities(registrar ActivityRegistrar, acts ConversationActivities) {
	register := func(fn any, name string) {
		registrar.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
	register(acts.ResolveConversations, flow.ResolveConversationsActivity)
	register(acts.BeginExternalExtractionRun, flow.BeginExternalRunActivity)
	register(acts.StageExternalExtractionPage, flow.StageExternalPageActivity)
	register(acts.FinishExternalExtractionRun, flow.FinishExternalRunActivity)
	register(acts.PlanSurrealSend, flow.PlanSurrealSendActivity)
	register(acts.UpsertConversationToSurreal, flow.UpsertConversationActivity)
	register(acts.UpsertExtractionsToSurreal, flow.UpsertExtractionsActivity)
	register(acts.VerifySurrealSend, flow.VerifySurrealSendActivity)
}

func (a ConversationActivities) beat(ctx context.Context, details any) {
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, details)
	}
}

func (a ConversationActivities) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}

// ResolveConversations resolves each requested conversation to the runs (source versions with a current generation and a preview) it consists of.
func (a ConversationActivities) ResolveConversations(ctx context.Context, request flow.ResolveRequest) (flow.ResolveResult, error) {
	if a.Conversations == nil {
		return flow.ResolveResult{}, errors.New("conversation store is not configured")
	}
	var result flow.ResolveResult
	for _, ref := range request.Conversations {
		runs, err := a.Conversations.ResolveConversation(ctx, request.MatterID, ref)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				result.Targets = append(result.Targets, flow.ConversationTarget{Ref: ref, Skipped: "the conversation was not found in this matter"})
				continue
			}
			return flow.ResolveResult{}, err
		}
		target := flow.ConversationTarget{Ref: ref, Runs: runs}
		if len(runs) == 0 {
			target.Skipped = "the conversation has no imported generation with a preview yet"
		}
		result.Targets = append(result.Targets, target)
	}
	return result, nil
}

// BeginExternalExtractionRun opens the compare-only working.extraction_run row of one external extractor over one run.
func (a ConversationActivities) BeginExternalExtractionRun(ctx context.Context, start flow.ExternalRunStart) (flow.ExternalRunStart, error) {
	if a.Store == nil {
		return start, errors.New("entity extraction store is not configured")
	}
	spec := start.Spec
	if spec.Kind != flow.KindExternal {
		return start, temporal.NewNonRetryableApplicationError("not an external extractor: "+spec.ID, "BadExtractor", nil)
	}
	err := a.Store.BeginRun(ctx, service.RunRow{
		ID: start.Input.ExternalRunID(), Extractor: spec.RunExtractor, Version: spec.Version,
		Summary: service.RunSummaryText(start.Input.Run),
		Stats: map[string]any{
			"extraction_id": start.Input.ExtractionID, "extractor_id": spec.ID, "compare_only": spec.CompareOnly,
			"requested_by": start.Input.Actor.Username,
		},
	})
	return start, err
}

// StageExternalExtractionPage validates an external extractor's reply for one window of messages, grounds it in those messages and stages it under the run.
func (a ConversationActivities) StageExternalExtractionPage(ctx context.Context, stage flow.StageExternalPage) (flow.StagePageResult, error) {
	if a.Store == nil {
		return flow.StagePageResult{}, errors.New("entity extraction store is not configured")
	}
	var page model.ExternalPage
	if err := json.Unmarshal(stage.Page, &page); err != nil {
		return flow.StagePageResult{Invalid: true, Reason: "the extractor's reply is not JSON: " + err.Error()}, nil
	}
	if page.Skipped {
		return flow.StagePageResult{Skipped: true, Reason: firstText(page.Reason, "the extractor reported itself skipped")}, nil
	}
	request := stage.Request
	limit := request.Limit
	if limit <= 0 || limit > flow.ExternalPageMessages {
		limit = flow.ExternalPageMessages
	}
	window, err := a.Store.MessagePage(ctx, request.GenerationID, request.AfterOrdinal, limit)
	if err != nil {
		return flow.StagePageResult{}, err
	}
	if page.LastOrdinal > 0 || page.Messages > 0 {
		kept := window[:0]
		for _, message := range window {
			if message.Ordinal <= page.LastOrdinal {
				kept = append(kept, message)
			}
		}
		window = kept
	}
	result := flow.StagePageResult{Messages: len(window), Done: page.Done || len(window) < limit}
	if len(window) > 0 {
		result.Last = window[len(window)-1].Ordinal
	} else {
		result.Done = true
		return result, nil
	}
	scope := stage.Input.Run.Scope()
	outcome := model.GroundExternal(page, window, scope)
	if outcome.Invalid {
		result.Invalid, result.Reason = true, outcome.Reason
		return result, nil
	}
	timed := make([]events.Proposal, 0, len(outcome.Events))
	for _, event := range outcome.Events {
		if event.OccurredAt == nil {
			outcome.Ungrounded++
			continue
		}
		timed = append(timed, event)
	}
	if _, err := a.Store.StageEntities(ctx, stage.RunID, outcome.Entities); err != nil {
		return result, err
	}
	if _, err := a.Store.StageEvents(ctx, stage.RunID, timed); err != nil {
		return result, err
	}
	result.Entities, result.Events, result.Ungrounded = len(outcome.Entities), len(timed), outcome.Ungrounded
	return result, nil
}

// FinishExternalExtractionRun closes an external extractor's run row with its counts.
func (a ConversationActivities) FinishExternalExtractionRun(ctx context.Context, finish flow.FinishExternalRun) error {
	if a.Store == nil {
		return errors.New("entity extraction store is not configured")
	}
	return a.Store.FinishRun(ctx, finish.RunID, finish.Status, finish.Stats, finish.Error)
}

func (a ConversationActivities) sink() (*surrealsink.Client, error) {
	if a.Surreal == nil {
		reason := a.SurrealReason
		if reason == "" {
			reason = surrealsink.ErrNotConfigured.Error()
		}
		return nil, temporal.NewNonRetryableApplicationError(reason, flow.SurrealNotConfiguredErrorType, nil)
	}
	return a.Surreal, nil
}

func (a ConversationActivities) requireConversations() error {
	if a.Conversations == nil {
		return errors.New("conversation store is not configured")
	}
	return nil
}

// PlanSurrealSend counts what PostgreSQL holds for a conversation and derives its deterministic Surreal thread id.
func (a ConversationActivities) PlanSurrealSend(ctx context.Context, target flow.SendTarget) (flow.SendPlan, error) {
	if _, err := a.sink(); err != nil {
		return flow.SendPlan{}, err
	}
	if err := a.requireConversations(); err != nil {
		return flow.SendPlan{}, err
	}
	info, err := a.Conversations.ConversationInfo(ctx, target.MatterID, target.Ref)
	if errors.Is(err, service.ErrNotFound) {
		return flow.SendPlan{}, temporal.NewNonRetryableApplicationError("the conversation was not found in this matter", "ConversationNotFound", nil)
	}
	if err != nil {
		return flow.SendPlan{}, err
	}
	plan := flow.SendPlan{
		Messages: info.Messages, SourceFiles: len(info.SourceFiles), Participants: len(info.Participants),
		ThreadID: flow.SurrealThreadID(target.MatterID, target.Ref),
	}
	if info.FirstAt != nil {
		plan.FirstAt = *info.FirstAt
	}
	if info.LastAt != nil {
		plan.LastAt = *info.LastAt
	}
	for _, file := range info.SourceFiles {
		plan.Generations = append(plan.Generations, file.GenerationID)
	}
	if info.Messages == 0 {
		return plan, temporal.NewNonRetryableApplicationError("the conversation has no messages to send", "ConversationEmpty", nil)
	}
	return plan, nil
}

// sendProgress is the heartbeat detail that lets a retried upsert resume after the last page.
type sendProgress struct {
	Cursor   service.MessageCursor `json:"cursor"`
	Messages int                   `json:"messages"`
	Thread   bool                  `json:"thread"`
}

// UpsertConversationToSurreal reads the conversation's messages from PostgreSQL a page at a time and upserts the thread and each message into surreal-case under deterministic ids.
func (a ConversationActivities) UpsertConversationToSurreal(ctx context.Context, target flow.SendTarget) (flow.SendWritten, error) {
	client, err := a.sink()
	if err != nil {
		return flow.SendWritten{}, err
	}
	if err := a.requireConversations(); err != nil {
		return flow.SendWritten{}, err
	}
	info, err := a.Conversations.ConversationInfo(ctx, target.MatterID, target.Ref)
	if err != nil {
		return flow.SendWritten{}, err
	}
	threadID := flow.SurrealThreadID(target.MatterID, target.Ref)
	generations := make([]string, 0, len(info.SourceFiles))
	files := make([]surrealsink.SourceFile, 0, len(info.SourceFiles))
	for _, file := range info.SourceFiles {
		generations = append(generations, file.GenerationID)
		files = append(files, surrealsink.SourceFile{SourceVersionID: file.SourceVersionID, SourceKey: file.SourceKey, GenerationID: file.GenerationID})
	}
	participants := make([]surrealsink.ThreadParticipant, 0, len(info.Participants))
	for _, participant := range info.Participants {
		participants = append(participants, surrealsink.ThreadParticipant{Identifier: participant.Identifier, Messages: participant.Messages})
	}
	progress := sendProgress{}
	if a.Resume != nil {
		a.Resume(ctx, &progress)
	}
	if !progress.Thread {
		if err := client.UpsertThread(ctx, surrealsink.Thread{
			ID: threadID, MatterID: target.MatterID, ExportKey: target.Ref.ExportKey, Conv: target.Ref.Conv,
			SourceFiles: files, Participants: participants, MessageCount: info.Messages, FirstAt: info.FirstAt, LastAt: info.LastAt,
			SentAt: a.now(), SentBy: target.Actor.Username, SendRequestID: target.Request,
		}); err != nil {
			return flow.SendWritten{}, err
		}
		progress.Thread = true
	}
	for {
		page, err := a.Conversations.ConversationMessages(ctx, generations, progress.Cursor, surrealBatch)
		if err != nil {
			return flow.SendWritten{}, err
		}
		if len(page) == 0 {
			break
		}
		batch := make([]surrealsink.Message, 0, len(page))
		for _, message := range page {
			batch = append(batch, surrealMessage(threadID, target.MatterID, message))
		}
		if err := client.UpsertMessages(ctx, batch); err != nil {
			return flow.SendWritten{}, err
		}
		last := page[len(page)-1]
		progress.Cursor = service.MessageCursor{At: last.OccurredAt, ID: last.ID}
		progress.Messages += len(page)
		a.beat(ctx, progress)
		if len(page) < surrealBatch {
			break
		}
	}
	return flow.SendWritten{Threads: 1, Messages: progress.Messages}, nil
}

// surrealMessage maps one PostgreSQL message to its Surreal record.
func surrealMessage(threadID, matterID string, message service.ConversationMessage) surrealsink.Message {
	var participants []struct {
		Identifier string `json:"identifier"`
		Role       string `json:"role"`
	}
	_ = json.Unmarshal(message.Participants, &participants)
	out := surrealsink.Message{
		ID: message.ID, ThreadID: threadID, MatterID: matterID, At: message.OccurredAt, Body: message.Body,
		Recipients: []string{}, Participants: message.Participants, Party: message.ProjectionKind,
		Attachments: message.AttachmentCount, Certainty: message.Certainty, SourceVersionID: message.SourceVersionID, Ordinal: message.Ordinal,
	}
	for _, participant := range participants {
		switch strings.ToLower(participant.Role) {
		case "sender", "from":
			if out.Sender == "" {
				out.Sender = participant.Identifier
			}
		case "recipient", "to", "cc", "bcc":
			out.Recipients = append(out.Recipients, participant.Identifier)
		}
	}
	if len(out.Participants) == 0 {
		out.Participants = json.RawMessage("[]")
	}
	return out
}

// UpsertExtractionsToSurreal upserts every extraction run, entity and event the conversation's generations hold, each tagged with its extractor.
func (a ConversationActivities) UpsertExtractionsToSurreal(ctx context.Context, target flow.SendTarget) (flow.SendWritten, error) {
	client, err := a.sink()
	if err != nil {
		return flow.SendWritten{}, err
	}
	if err := a.requireConversations(); err != nil {
		return flow.SendWritten{}, err
	}
	info, err := a.Conversations.ConversationInfo(ctx, target.MatterID, target.Ref)
	if err != nil {
		return flow.SendWritten{}, err
	}
	generations := make([]string, 0, len(info.SourceFiles))
	for _, file := range info.SourceFiles {
		generations = append(generations, file.GenerationID)
	}
	export, err := a.Conversations.ConversationExtractions(ctx, generations)
	if err != nil {
		return flow.SendWritten{}, err
	}
	threadID := flow.SurrealThreadID(target.MatterID, target.Ref)
	extractorOf := map[string]string{}
	runs := make([]surrealsink.Run, 0, len(export.Runs))
	for _, run := range export.Runs {
		extractorOf[run.ID] = run.Extractor
		runs = append(runs, surrealsink.Run{
			ID: run.ID, ThreadID: threadID, Extractor: run.Extractor, Version: run.Version, ModelID: run.ModelID, Status: run.Status,
			CompareOnly: run.CompareOnly, Stats: nonNilJSON(run.Stats), StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
		})
	}
	for start := 0; start < len(runs); start += surrealBatch {
		if err := client.UpsertRuns(ctx, runs[start:min(start+surrealBatch, len(runs))]); err != nil {
			return flow.SendWritten{}, err
		}
	}
	entityRows := make([]surrealsink.Entity, 0, len(export.Entities))
	for _, entity := range export.Entities {
		var proposal entities.Proposal
		_ = json.Unmarshal(entity.Attrs, &proposal)
		aliases := make([]string, 0, len(proposal.Aliases))
		for _, alias := range proposal.Aliases {
			aliases = append(aliases, alias.Text)
		}
		sample := proposal.ModelMentions
		if len(sample) > 10 {
			sample = sample[:10]
		}
		type mentionOut struct {
			RecordID string `json:"record_id"`
			Surface  string `json:"surface"`
			Snippet  string `json:"snippet,omitempty"`
		}
		mentions := make([]mentionOut, 0, len(sample))
		for _, mention := range sample {
			mentions = append(mentions, mentionOut{RecordID: mention.RecordID, Surface: mention.Surface, Snippet: mention.Snippet})
		}
		encoded, _ := json.Marshal(mentions)
		count := proposal.MentionCount
		if count == 0 {
			count = len(proposal.ModelMentions)
		}
		entityRows = append(entityRows, surrealsink.Entity{
			ID: entity.ID, ThreadID: threadID, RunID: entity.RunID, Extractor: extractorOf[entity.RunID], Name: entity.Name,
			EntityType: entity.EntityType, Aliases: aliases, MentionCount: count, Mentions: encoded,
			Confidence: entity.Confidence, ReviewState: entity.ReviewState,
		})
	}
	for start := 0; start < len(entityRows); start += surrealBatch {
		if err := client.UpsertEntities(ctx, entityRows[start:min(start+surrealBatch, len(entityRows))]); err != nil {
			return flow.SendWritten{}, err
		}
		a.beat(ctx, start)
	}
	eventRows := make([]surrealsink.Event, 0, len(export.Events))
	for _, event := range export.Events {
		var proposal events.Proposal
		_ = json.Unmarshal(event.Attrs, &proposal)
		recordIDs := make([]string, 0, len(proposal.SourceRecords))
		for _, record := range proposal.SourceRecords {
			recordIDs = append(recordIDs, record.RecordID)
		}
		eventRows = append(eventRows, surrealsink.Event{
			ID: event.ID, ThreadID: threadID, RunID: event.RunID, Extractor: extractorOf[event.RunID], Title: event.Title,
			EventType: event.EventType, OccurredAt: event.OccurredAt, Precision: proposal.TemporalPrecision,
			Confidence: event.Confidence, RecordIDs: recordIDs, ReviewState: event.ReviewState,
		})
	}
	for start := 0; start < len(eventRows); start += surrealBatch {
		if err := client.UpsertEvents(ctx, eventRows[start:min(start+surrealBatch, len(eventRows))]); err != nil {
			return flow.SendWritten{}, err
		}
		a.beat(ctx, start)
	}
	return flow.SendWritten{Runs: len(runs), Entities: len(entityRows), Events: len(eventRows)}, nil
}

func nonNilJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("{}")
	}
	return value
}

// VerifySurrealSend reads the thread's record counts back from surreal-case and compares them with what PostgreSQL held and what was written.
func (a ConversationActivities) VerifySurrealSend(ctx context.Context, request flow.VerifyRequest) (flow.SendVerified, error) {
	client, err := a.sink()
	if err != nil {
		return flow.SendVerified{}, err
	}
	counts, err := client.CountThread(ctx, request.Plan.ThreadID)
	if err != nil {
		return flow.SendVerified{}, err
	}
	// Messages must match exactly. Extractions only ever grow on the Surreal side (a send upserts,
	// it never deletes), so an earlier send's rows may still be there.
	match := counts.Messages == request.Plan.Messages &&
		counts.Entities >= request.Written.Entities && counts.Events >= request.Written.Events
	return flow.SendVerified{Messages: counts.Messages, Entities: counts.Entities, Events: counts.Events, Match: match}, nil
}
