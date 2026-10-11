// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Entity and event extraction Activities. Each method is one Activity with a
// bounded input and output: proposals, messages and mentions stay in
// PostgreSQL and move by generation id, never through Temporal history.
// Registration lives in this file so activities/register.go is unchanged.
// Byline: Codex · GPT-5 · 2026-10-05 (durable single-case extraction admission).
package activities

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

// Bounds for the streaming passes.
const (
	extractionPageSize = 400
	maxModelMessages   = 20000
	collectionTitle    = "Case timeline"
)

// EntityExtractionActivities implements the extraction and commit Activities.
type EntityExtractionActivities struct {
	Store service.Store
	// Model is nil when no model is configured; the model Activity then
	// reports itself skipped with ModelUnavailable as the reason.
	Model            model.Completer
	ModelID          string
	ModelUnavailable string
	Heartbeat        func(context.Context, any)
	// Resume loads the last heartbeat details of a retried attempt into
	// the pointer and reports whether there were any.
	Resume func(context.Context, any) bool
}

// NewEntityExtractionActivities binds production heartbeats.
func NewEntityExtractionActivities(store service.Store, completer model.Completer, modelID, unavailable string) EntityExtractionActivities {
	return EntityExtractionActivities{
		Store: store, Model: completer, ModelID: modelID, ModelUnavailable: unavailable,
		Heartbeat: func(ctx context.Context, details any) { activity.RecordHeartbeat(ctx, details) },
		Resume: func(ctx context.Context, into any) bool {
			if !activity.HasHeartbeatDetails(ctx) {
				return false
			}
			return activity.GetHeartbeatDetails(ctx, into) == nil
		},
	}
}

// RegisterEntityExtractionActivities installs every extraction Activity
// under its exact name, plus the two workflows' Activities' shared names.
func RegisterEntityExtractionActivities(registrar ActivityRegistrar, acts EntityExtractionActivities) {
	register := func(fn any, name string) {
		registrar.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
	register(acts.ProposeEntitiesRules, flow.ProposeRulesActivity)
	register(acts.ExtractEntitiesEventsModel, flow.ExtractModelActivity)
	register(acts.ReconcileEntityProposals, flow.ReconcileActivity)
	register(acts.ValidateExtractionCommit, flow.ValidateCommitActivity)
	register(acts.CommitEntities, flow.CommitEntitiesActivity)
	register(acts.CommitEntityAliases, flow.CommitAliasesActivity)
	register(acts.CommitEntityMentions, flow.CommitMentionsActivity)
	register(acts.CommitEventCandidates, flow.CommitEventsActivity)
	register(acts.CommitTimelineMembers, flow.CommitMembersActivity)
	register(acts.FinalizeExtractionCommit, flow.FinalizeCommitActivity)
}

func (a EntityExtractionActivities) heartbeat(ctx context.Context, details any) {
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, details)
	}
}

func (a EntityExtractionActivities) requireStore() error {
	if a.Store == nil {
		return errors.New("entity extraction store is not configured")
	}
	return nil
}

// admitRun checks explicit operation policy before any write, then verifies the
// authoritative receipt and current generation through the store's exact-case resolver.
// Inputs: durable workflow RunRef. Outputs: admission error. Effects: one read only.
// Missing pre-upgrade modes never become LIVE by inference or caller override.
func (a EntityExtractionActivities) admitRun(ctx context.Context, requested flow.RunRef) error {
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(requested.MatterMode)); err != nil {
		return stopRetryingPermanent(permanent(err))
	}
	if err := a.requireStore(); err != nil {
		return err
	}
	durable, err := a.Store.ResolveRun(ctx, requested.PreviewHandle)
	if err != nil {
		return err
	}
	if err := caseidentity.RequireCanonicalWrite(caseidentity.Mode(durable.MatterMode)); err != nil {
		return stopRetryingPermanent(permanent(err))
	}
	if durable.MatterMode != requested.MatterMode || durable.GenerationID != requested.GenerationID || durable.SourceVersionID != requested.SourceVersionID {
		return stopRetryingPermanent(permanent(errors.New("extraction coordinates disagree with the durable operation receipt")))
	}
	return nil
}

// ProposeEntitiesRules proposes people from the run's participant headers
// (DuckDB ELT over the normalized messages) and stages them.
func (a EntityExtractionActivities) ProposeEntitiesRules(ctx context.Context, request flow.ExtractionRequest) (flow.ProposeResult, error) {
	if err := a.admitRun(ctx, request.Run); err != nil {
		return flow.ProposeResult{}, err
	}
	if err := a.requireStore(); err != nil {
		return flow.ProposeResult{}, err
	}
	runID := request.RunID("rules")
	if err := a.Store.BeginRun(ctx, service.RunRow{
		ID: runID, Extractor: entities.RulesExtractor, Version: entities.RulesExtractorVersion,
		Summary: service.RunSummaryText(request.Run), Stats: map[string]any{"extraction_id": request.ExtractionID, "requested_by": request.Actor.Username},
	}); err != nil {
		return flow.ProposeResult{}, err
	}
	rows, err := a.Store.ParticipantAggregates(ctx, request.Run.GenerationID)
	if err != nil {
		_ = a.Store.FinishRun(ctx, runID, "failed", nil, err.Error())
		return flow.ProposeResult{}, err
	}
	proposals := entities.ProposeFromParticipants(request.Run.Scope(), rows)
	if _, err := a.Store.StageEntities(ctx, runID, proposals); err != nil {
		_ = a.Store.FinishRun(ctx, runID, "failed", nil, err.Error())
		return flow.ProposeResult{}, err
	}
	messages := 0
	for _, row := range rows {
		if row.TotalMessages > messages {
			messages = row.TotalMessages
		}
	}
	stats := map[string]any{"extraction_id": request.ExtractionID, "proposals": len(proposals), "messages": messages, "participants": len(rows)}
	if err := a.Store.FinishRun(ctx, runID, "completed", stats, ""); err != nil {
		return flow.ProposeResult{}, err
	}
	return flow.ProposeResult{ExtractionRunID: runID, Messages: messages, Proposals: len(proposals)}, nil
}

// modelProgress is the heartbeat detail that lets a retried attempt resume
// after the last completed batch instead of paying for it again.
type modelProgress struct {
	AfterOrdinal int64               `json:"after_ordinal"`
	NextBatch    int                 `json:"next_batch"`
	Messages     int                 `json:"messages"`
	Proposals    int                 `json:"proposals"`
	Events       int                 `json:"events"`
	Ungrounded   int                 `json:"ungrounded"`
	Retried      int                 `json:"retried"`
	Invalid      []flow.InvalidBatch `json:"invalid"`
	Started      bool                `json:"started"`
}

// heartbeatingCompleter records a heartbeat before each model call.
type heartbeatingCompleter struct {
	inner model.Completer
	beat  func()
}

func (c heartbeatingCompleter) Complete(ctx context.Context, messages []model.Message, options model.CallOptions) (model.Completion, error) {
	c.beat()
	return c.inner.Complete(ctx, messages, options)
}

// ExtractEntitiesEventsModel reads the run's messages in bounded batches and
// asks the configured model for people, places, organizations and events.
// Replies are validated, retried once, and grounded; a batch that fails twice
// is flagged and contributes nothing.
func (a EntityExtractionActivities) ExtractEntitiesEventsModel(ctx context.Context, request flow.ExtractionRequest) (flow.ProposeResult, error) {
	if err := a.admitRun(ctx, request.Run); err != nil {
		return flow.ProposeResult{}, err
	}
	if err := a.requireStore(); err != nil {
		return flow.ProposeResult{}, err
	}
	if a.Model == nil {
		reason := a.ModelUnavailable
		if reason == "" {
			reason = "no extraction model is configured"
		}
		return flow.ProposeResult{Skipped: true, Reason: reason}, nil
	}
	runID := request.RunID("model")
	progress := modelProgress{AfterOrdinal: -1}
	if a.Resume != nil {
		a.Resume(ctx, &progress)
	}
	if !progress.Started {
		if err := a.Store.BeginRun(ctx, service.RunRow{
			ID: runID, Extractor: model.Extractor, Version: model.ExtractorVersion, ModelID: a.ModelID,
			PromptVersion: model.PromptVersion, Summary: service.RunSummaryText(request.Run),
			Stats: map[string]any{"extraction_id": request.ExtractionID, "requested_by": request.Actor.Username},
		}); err != nil {
			return flow.ProposeResult{}, err
		}
		progress.Started = true
	}
	legend, err := a.legend(ctx, request)
	if err != nil {
		return flow.ProposeResult{}, err
	}
	scope := request.Run.Scope()
	// Heartbeat before every model call: one call can take minutes, and a
	// batch may make two.
	completer := heartbeatingCompleter{inner: a.Model, beat: func() { a.heartbeat(ctx, progress) }}
	for progress.Messages < maxModelMessages {
		page, err := a.Store.MessagePage(ctx, request.Run.GenerationID, progress.AfterOrdinal, extractionPageSize)
		if err != nil {
			return flow.ProposeResult{}, err
		}
		if len(page) == 0 {
			break
		}
		for _, batch := range model.Batches(page, legend, progress.NextBatch) {
			outcome, err := model.ExtractBatch(ctx, completer, a.ModelID, batch, scope)
			if err != nil {
				return flow.ProposeResult{}, err
			}
			progress.Retried += len(outcome.RetryReasons)
			if outcome.Invalid {
				progress.Invalid = append(progress.Invalid, flow.InvalidBatch{Index: outcome.Index, FirstOrdinal: outcome.FirstOrd, LastOrdinal: outcome.LastOrd, Reason: outcome.Reason})
			} else {
				timed := outcome.Events[:0]
				for _, event := range outcome.Events {
					if event.OccurredAt == nil {
						// A message without its own clock cannot date an event.
						progress.Ungrounded++
						continue
					}
					timed = append(timed, event)
				}
				if _, err := a.Store.StageEntities(ctx, runID, outcome.Entities); err != nil {
					return flow.ProposeResult{}, err
				}
				if _, err := a.Store.StageEvents(ctx, runID, timed); err != nil {
					return flow.ProposeResult{}, err
				}
				progress.Proposals += len(outcome.Entities)
				progress.Events += len(timed)
				progress.Ungrounded += outcome.Ungrounded
			}
			progress.Messages += len(batch.Messages)
			progress.AfterOrdinal = batch.Messages[len(batch.Messages)-1].Ordinal
			progress.NextBatch = batch.Index + 1
			a.heartbeat(ctx, progress)
		}
	}
	stats := map[string]any{
		"extraction_id": request.ExtractionID, "model": a.ModelID, "messages": progress.Messages,
		"batches": progress.NextBatch, "proposals": progress.Proposals, "events": progress.Events,
		"ungrounded": progress.Ungrounded, "retried_replies": progress.Retried, "invalid_batches": progress.Invalid,
	}
	if progress.Messages >= maxModelMessages {
		stats["truncated_at_messages"] = maxModelMessages
	}
	if err := a.Store.FinishRun(ctx, runID, "completed", stats, ""); err != nil {
		return flow.ProposeResult{}, err
	}
	return flow.ProposeResult{
		ExtractionRunID: runID, Messages: progress.Messages, Proposals: progress.Proposals, Events: progress.Events,
		Batches: progress.NextBatch, Ungrounded: progress.Ungrounded, InvalidBatches: progress.Invalid,
	}, nil
}

// legend labels the run's participants P1..Pn (the device owner first) with
// any name the participant rules found.
func (a EntityExtractionActivities) legend(ctx context.Context, request flow.ExtractionRequest) ([]model.Participant, error) {
	rows, err := a.Store.ParticipantAggregates(ctx, request.Run.GenerationID)
	if err != nil {
		return nil, err
	}
	current, err := a.Store.CurrentEntities(ctx, request.Run.GenerationID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, proposal := range current {
		if !proposal.Included() || entities.LooksLikeAddress(proposal.Name) || strings.HasPrefix(proposal.Name, "Device owner") {
			continue
		}
		for _, alias := range proposal.AddressAliases() {
			names[string(alias.AddressKind)+":"+alias.Normalized] = proposal.Name
		}
	}
	type entry struct {
		address entities.Address
		count   int
	}
	seen := map[string]*entry{}
	for _, row := range rows {
		address := entities.NormalizeAddress(row.Identifier)
		if row.NormalizedAddress != "" {
			address = entities.Address{Kind: entities.AddressKind(row.AddressKind), Normalized: row.NormalizedAddress, Raw: row.Identifier}
		}
		key := string(address.Kind) + ":" + address.Normalized
		if existing, ok := seen[key]; ok {
			existing.count += row.MessageCount
			continue
		}
		seen[key] = &entry{address: address, count: row.MessageCount}
	}
	list := make([]*entry, 0, len(seen))
	for _, value := range seen {
		list = append(list, value)
	}
	sort.Slice(list, func(i, j int) bool {
		if (list[i].address.Kind == entities.AddressSelf) != (list[j].address.Kind == entities.AddressSelf) {
			return list[i].address.Kind == entities.AddressSelf
		}
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return list[i].address.Normalized < list[j].address.Normalized
	})
	legend := make([]model.Participant, 0, len(list))
	for i, value := range list {
		legend = append(legend, model.Participant{
			Label: fmt.Sprintf("P%d", i+1), Address: value.address,
			Name: names[string(value.address.Kind)+":"+value.address.Normalized],
		})
	}
	return legend, nil
}

// ReconcileEntityProposals groups this extraction's partial proposals into
// one proposal per entity, folds them into the owner's current proposals,
// matches committed entities, and counts supporting body mentions.
func (a EntityExtractionActivities) ReconcileEntityProposals(ctx context.Context, request flow.ReconcileRequest) (flow.ReconcileResult, error) {
	if err := a.admitRun(ctx, request.Extraction.Run); err != nil {
		return flow.ReconcileResult{}, err
	}
	if err := a.requireStore(); err != nil {
		return flow.ReconcileResult{}, err
	}
	extraction := request.Extraction
	runID := extraction.RunID("reconcile")
	if err := a.Store.BeginRun(ctx, service.RunRow{
		ID: runID, Extractor: entities.ReconcileExtractor, Version: entities.ReconcileExtractorVersion,
		Summary: service.RunSummaryText(extraction.Run), Stats: map[string]any{"extraction_id": extraction.ExtractionID},
	}); err != nil {
		return flow.ReconcileResult{}, err
	}
	partials, err := a.Store.PendingEntitiesOfRuns(ctx, request.RunIDs)
	if err != nil {
		return flow.ReconcileResult{}, err
	}
	current, err := a.Store.CurrentEntities(ctx, extraction.Run.GenerationID)
	if err != nil {
		return flow.ReconcileResult{}, err
	}
	ours := map[string]bool{}
	for _, id := range request.RunIDs {
		ours[id] = true
	}
	ours[runID] = true
	var existing []entities.Proposal
	for _, proposal := range current {
		if !ours[proposal.ExtractionRunID] {
			existing = append(existing, proposal)
		}
	}
	registry, err := a.Store.Registry(ctx, lookupFor(partials))
	if err != nil {
		return flow.ReconcileResult{}, err
	}
	plan := entities.Reconcile(entities.ReconcileInput{Scope: extraction.Run.Scope(), Partials: partials, Existing: existing, Registry: registry})
	if err := a.supportingMentions(ctx, extraction.Run, plan.Insert); err != nil {
		return flow.ReconcileResult{}, err
	}
	ids := make([]string, len(plan.Insert))
	matched := 0
	for i, proposal := range plan.Insert {
		ids[i] = flow.DeterministicID("reconciled", runID, proposal.ContentHex())
		if proposal.Match != nil {
			matched++
		}
	}
	if err := a.Store.ReplaceEntities(ctx, runID, ids, plan.Insert, plan.Supersede); err != nil {
		return flow.ReconcileResult{}, err
	}
	stats := map[string]any{"extraction_id": extraction.ExtractionID, "proposals": len(plan.Insert), "superseded": len(plan.Supersede), "matched": matched, "dropped": len(plan.Dropped)}
	if err := a.Store.FinishRun(ctx, runID, "completed", stats, ""); err != nil {
		return flow.ReconcileResult{}, err
	}
	return flow.ReconcileResult{ExtractionRunID: runID, Proposals: len(plan.Insert), Superseded: len(plan.Supersede), Matched: matched, Dropped: len(plan.Dropped)}, nil
}

func lookupFor(proposals []entities.Proposal) service.RegistryLookup {
	var lookup service.RegistryLookup
	for _, proposal := range proposals {
		lookup.NormalizedNames = append(lookup.NormalizedNames, entities.RegistryNormalizedName(proposal.Name))
		for _, alias := range proposal.Aliases {
			if alias.IsAddress() {
				lookup.AliasTexts = append(lookup.AliasTexts, strings.ToLower(alias.Normalized))
			} else {
				lookup.AliasTexts = append(lookup.AliasTexts, strings.ToLower(alias.Text))
			}
		}
	}
	return lookup
}

// supportingMentions streams the run once and counts each proposal's body
// mentions, keeping a bounded sample for the reviewer.
func (a EntityExtractionActivities) supportingMentions(ctx context.Context, run flow.RunRef, proposals []entities.Proposal) error {
	if len(proposals) == 0 {
		return nil
	}
	body := make([]int, len(proposals))
	after := int64(-1)
	for {
		page, err := a.Store.MessagePage(ctx, run.GenerationID, after, extractionPageSize)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		for _, message := range page {
			for i := range proposals {
				mentions := entities.BodyMentions(message, proposals[i])
				body[i] += len(mentions)
				proposals[i].AddMentionSample(mentions...)
			}
		}
		after = page[len(page)-1].Ordinal
		a.heartbeat(ctx, map[string]any{"after_ordinal": after})
	}
	for i := range proposals {
		participant := 0
		for _, stat := range proposals[i].Participants {
			participant += stat.MessageCount
		}
		proposals[i].MentionCount = participant + body[i] + len(proposals[i].ModelMentions)
	}
	return nil
}

// ValidateExtractionCommit re-validates at the start of a commit against the
// run's current generation, fail-closed.
func (a EntityExtractionActivities) ValidateExtractionCommit(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	if err := a.requireStore(); err != nil {
		return flow.CommitStepResult{}, err
	}
	run, err := a.currentRun(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	report, err := service.Validate(ctx, a.Store, run)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{Report: &report, Detail: fmt.Sprintf("ok=%v", report.OK)}, nil
}

func (a EntityExtractionActivities) currentRun(ctx context.Context, request flow.CommitRequest) (flow.RunRef, error) {
	if err := a.admitRun(ctx, request.Run); err != nil {
		return flow.RunRef{}, err
	}
	run, err := a.Store.ResolveRun(ctx, request.Run.PreviewHandle)
	if err != nil {
		return flow.RunRef{}, err
	}
	return run, nil
}

// guard reloads the included proposal set and refuses to write when it is
// no longer exactly the set that was validated.
func (a EntityExtractionActivities) guard(ctx context.Context, request flow.CommitRequest) (commitcheck.Snapshot, error) {
	run, err := a.currentRun(ctx, request)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	snapshot, err := service.Snapshot(ctx, a.Store, run)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	if commitcheck.Digest(snapshot) != request.Digest {
		return commitcheck.Snapshot{}, errors.New("the proposals changed during the commit; validate again and re-run")
	}
	return snapshot, nil
}

// CommitEntities writes the new registry.entity rows.
func (a EntityExtractionActivities) CommitEntities(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	snapshot, err := a.guard(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	rows := service.EntityRows(service.IncludedEntities(snapshot.Entities), request.ReceiptID())
	written, err := a.Store.WriteEntities(ctx, rows)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{Written: written, Skipped: len(rows) - written, Detail: fmt.Sprintf("%d new entities", written)}, nil
}

// CommitEntityAliases writes registry.entity_alias rows.
func (a EntityExtractionActivities) CommitEntityAliases(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	snapshot, err := a.guard(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	rows := service.AliasRows(service.IncludedEntities(snapshot.Entities))
	written, err := a.Store.WriteAliases(ctx, rows)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{Written: written, Skipped: len(rows) - written, Detail: fmt.Sprintf("%d aliases", written)}, nil
}

// CommitEntityMentions streams the run and writes every mention of every
// committed entity with its current resolution (append-only mentions).
func (a EntityExtractionActivities) CommitEntityMentions(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	snapshot, err := a.guard(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	included := service.IncludedEntities(snapshot.Entities)
	if len(included) == 0 {
		return flow.CommitStepResult{Detail: "no entities to link"}, nil
	}
	reviewer := service.Reviewer{Username: request.Actor.Username, At: request.RequestedAt}
	run, err := a.currentRun(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	mentions, resolutions, conflicts, planned := 0, 0, 0, 0
	after := int64(-1)
	for {
		page, err := a.Store.MessagePage(ctx, run.GenerationID, after, extractionPageSize)
		if err != nil {
			return flow.CommitStepResult{}, err
		}
		if len(page) == 0 {
			break
		}
		plan := service.NewMentionPlan()
		for _, message := range page {
			plan.MessageMentions(message, included, run.SourceVersionID)
		}
		if len(plan.Mentions) > 0 {
			wroteMentions, wroteResolutions, err := a.Store.WriteMentions(ctx, plan.Mentions, plan.Resolutions, reviewer, request.ReceiptID())
			if err != nil {
				return flow.CommitStepResult{}, err
			}
			mentions += wroteMentions
			resolutions += wroteResolutions
		}
		planned += len(plan.Mentions)
		conflicts += plan.Conflicts
		after = page[len(page)-1].Ordinal
		a.heartbeat(ctx, map[string]any{"after_ordinal": after, "mentions": mentions})
	}
	detail := fmt.Sprintf("%d mentions linked (%d resolutions)", mentions, resolutions)
	if conflicts > 0 {
		detail += fmt.Sprintf("; %d spans claimed by two entities kept their first entity", conflicts)
	}
	return flow.CommitStepResult{Written: mentions, Skipped: planned - mentions, Detail: detail,
		Counts: map[string]int{"mentions": mentions, "resolutions": resolutions, "conflicts": conflicts}}, nil
}

func (a EntityExtractionActivities) eventRows(ctx context.Context, request flow.CommitRequest) ([]service.EventCandidateRow, error) {
	snapshot, err := a.guard(ctx, request)
	if err != nil {
		return nil, err
	}
	return service.EventRows(service.IncludedEvents(snapshot.Events), snapshot.Entities, request.ReceiptID())
}

// CommitEventCandidates writes timeline.event_candidate rows with their
// typed source_available_from anchors.
func (a EntityExtractionActivities) CommitEventCandidates(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	rows, err := a.eventRows(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	if len(rows) == 0 {
		return flow.CommitStepResult{Detail: "no events to commit"}, nil
	}
	written, err := a.Store.WriteEventCandidates(ctx, rows, service.Reviewer{Username: request.Actor.Username, At: request.RequestedAt})
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{Written: written, Skipped: len(rows) - written, Detail: fmt.Sprintf("%d events", written)}, nil
}

// CommitTimelineMembers adds the committed events to the case timeline.
func (a EntityExtractionActivities) CommitTimelineMembers(ctx context.Context, request flow.CommitRequest) (flow.CommitStepResult, error) {
	rows, err := a.eventRows(ctx, request)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	if len(rows) == 0 {
		return flow.CommitStepResult{Detail: "no events to add to the timeline"}, nil
	}
	slug := request.CollectionSlug
	if slug == "" {
		slug = flow.DefaultCollectionSlug
	}
	collectionID, err := a.Store.EnsureCollection(ctx, slug, collectionTitle)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	members := service.MemberRows(rows, collectionID)
	written, err := a.Store.WriteTimelineMembers(ctx, members)
	if err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{Written: written, Skipped: len(members) - written, Detail: fmt.Sprintf("%d timeline members in %q", written, slug)}, nil
}

// FinalizeExtractionCommit records the commit receipt and, on success,
// promotes the staged proposals it committed.
func (a EntityExtractionActivities) FinalizeExtractionCommit(ctx context.Context, request flow.FinalizeRequest) (flow.CommitStepResult, error) {
	if err := a.admitRun(ctx, request.Commit.Run); err != nil {
		return flow.CommitStepResult{}, err
	}
	if err := a.requireStore(); err != nil {
		return flow.CommitStepResult{}, err
	}
	commit := request.Commit
	status := "failed"
	if request.Outcome == flow.OutcomeCommitted {
		status = "completed"
	}
	errText := request.Error
	if status == "failed" && strings.TrimSpace(errText) == "" {
		errText = request.Outcome
	}
	receipt := service.CommitReceipt{
		ID: commit.ReceiptID(), Status: status, Error: errText, Summary: service.RunSummaryText(commit.Run), StartedAt: commit.RequestedAt,
		Stats: map[string]any{
			"outcome": request.Outcome, "counts": request.Counts, "workflow_id": commit.WorkflowID, "commit_id": commit.CommitID,
			"digest": commit.Digest, "actor": commit.Actor, "failed_at": request.FailedAt, "matter_mode": commit.Run.MatterMode,
		},
	}
	var entityPromotions, eventPromotions []service.Promotion
	if request.Promote {
		snapshot, err := a.guard(ctx, commit)
		if err != nil {
			return flow.CommitStepResult{}, err
		}
		entityPromotions, eventPromotions = service.Promotions(service.IncludedEntities(snapshot.Entities), service.IncludedEvents(snapshot.Events))
	}
	if err := a.Store.FinalizeCommit(ctx, receipt, entityPromotions, eventPromotions); err != nil {
		return flow.CommitStepResult{}, err
	}
	return flow.CommitStepResult{
		Written: len(entityPromotions) + len(eventPromotions),
		Detail:  fmt.Sprintf("receipt %s (%s)", receipt.ID, status),
		Counts:  map[string]int{"entities_promoted": len(entityPromotions), "events_promoted": len(eventPromotions)},
	}, nil
}
