// Byline: Claude Code · Opus 5.5 · 2026-09-25

package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/commitcheck"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/flow"
)

// Owner review run: every owner correction and "worth recalling" mark of a
// generation is staged under one working.extraction_run row.
const (
	OwnerReviewExtractor = "owner.review"
	OwnerReviewVersion   = "1"
	CommitExtractor      = "probata.extraction.commit"
	CommitVersion        = "1"
	EntityRefPrefix      = "registry.entity:"
	SourceSystem         = "probata.context"
)

// OwnerRunID is the generation's owner-review run.
func OwnerRunID(generationID string) string {
	return flow.DeterministicID("owner_review", generationID)
}

// RunSummaryText is the extraction_run.source_summary for a run.
func RunSummaryText(run flow.RunRef) string {
	return "preview:" + run.PreviewHandle + " generation:" + run.GenerationID + " source:" + run.SourceVersionID
}

func ensureOwnerRun(ctx context.Context, store Store, run flow.RunRef) error {
	id := OwnerRunID(run.GenerationID)
	if err := store.BeginRun(ctx, RunRow{ID: id, Extractor: OwnerReviewExtractor, Version: OwnerReviewVersion, Summary: RunSummaryText(run), Stats: map[string]any{"preview_handle": run.PreviewHandle}}); err != nil {
		return err
	}
	return store.FinishRun(ctx, id, "completed", map[string]any{"preview_handle": run.PreviewHandle}, "")
}

// Snapshot loads everything commit validation reads.
func Snapshot(ctx context.Context, store Store, run flow.RunRef) (commitcheck.Snapshot, error) {
	current, err := store.CurrentEntities(ctx, run.GenerationID)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	currentEvents, err := store.CurrentEvents(ctx, run.GenerationID)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	var lookup RegistryLookup
	records := map[string]bool{}
	for _, proposal := range current {
		if !proposal.Included() {
			continue
		}
		if proposal.Match != nil {
			lookup.IDs = append(lookup.IDs, proposal.Match.EntityID)
		}
		lookup.NormalizedNames = append(lookup.NormalizedNames, entities.RegistryNormalizedName(proposal.Name))
		for _, alias := range proposal.Aliases {
			if alias.IsAddress() {
				lookup.AliasTexts = append(lookup.AliasTexts, strings.ToLower(alias.Normalized))
			} else {
				lookup.AliasTexts = append(lookup.AliasTexts, strings.ToLower(alias.Text))
			}
		}
		for _, mention := range proposal.MentionSample {
			records[mention.RecordID] = true
		}
		for _, mention := range proposal.ModelMentions {
			records[mention.RecordID] = true
		}
	}
	for _, event := range currentEvents {
		if !event.Included() {
			continue
		}
		for _, record := range event.SourceRecords {
			records[record.RecordID] = true
		}
	}
	registry, err := store.Registry(ctx, lookup)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	missing, err := store.RecordsMissing(ctx, run.GenerationID, ids)
	if err != nil {
		return commitcheck.Snapshot{}, err
	}
	return commitcheck.Snapshot{
		MatterMode: run.MatterMode, PreviewHandle: run.PreviewHandle, CurrentGenerationID: run.GenerationID,
		Entities: current, Events: currentEvents, Registry: registry, MissingRecords: missing,
	}, nil
}

// Validate runs commit validation for a run.
func Validate(ctx context.Context, store Store, run flow.RunRef) (commitcheck.Report, error) {
	snapshot, err := Snapshot(ctx, store, run)
	if err != nil {
		return commitcheck.Report{}, err
	}
	return commitcheck.Validate(snapshot), nil
}

// CorrectionEnvelope carries one owner edit to entity or event proposals.
type CorrectionEnvelope struct {
	Target string                      `json:"target"`
	Entity *entities.CorrectionRequest `json:"entity,omitempty"`
	Event  *events.CorrectionRequest   `json:"event,omitempty"`
}

// ErrInvalid wraps a request the owner can fix.
type ErrInvalid struct{ Err error }

func (e ErrInvalid) Error() string { return e.Err.Error() }
func (e ErrInvalid) Unwrap() error { return e.Err }

// CorrectionIDs are the deterministic ids of a correction's new rows: a
// retried request writes the same rows, never a second copy.
func CorrectionIDs(digest string, count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = flow.DeterministicID("correction", digest, fmt.Sprint(i))
	}
	return ids
}

// Correct applies one owner edit and stages its result.
func Correct(ctx context.Context, store Store, run flow.RunRef, envelope CorrectionEnvelope, actor entities.Actor, at time.Time, digest string) error {
	if err := ensureOwnerRun(ctx, store, run); err != nil {
		return err
	}
	switch envelope.Target {
	case "entity":
		if envelope.Entity == nil {
			return ErrInvalid{errors.New("entity correction body is required")}
		}
		current, err := store.CurrentEntities(ctx, run.GenerationID)
		if err != nil {
			return err
		}
		byID := map[string]entities.Proposal{}
		for _, proposal := range current {
			byID[proposal.CandidateID] = proposal
		}
		request := *envelope.Entity
		if request.Op == entities.OpSetMatch && request.MatchEntity != nil {
			found, err := store.Registry(ctx, RegistryLookup{IDs: []string{request.MatchEntity.EntityID}})
			if err != nil {
				return err
			}
			if len(found) != 1 {
				return ErrInvalid{fmt.Errorf("committed entity %s does not exist or was merged", request.MatchEntity.EntityID)}
			}
			request.MatchEntity.DisplayName = found[0].DisplayName
		}
		result, err := entities.ApplyCorrection(byID, request, actor, at, digest)
		if err != nil {
			if errors.Is(err, entities.ErrConflict) && replayed(ctx, store, run, CorrectionIDs(digest, 1)[0]) {
				return nil
			}
			if errors.Is(err, entities.ErrConflict) {
				return err
			}
			return ErrInvalid{err}
		}
		for i := range result.Insert {
			result.Insert[i].GenerationID = run.GenerationID
		}
		return store.ReplaceEntities(ctx, OwnerRunID(run.GenerationID), CorrectionIDs(digest, len(result.Insert)), result.Insert, result.Supersede)
	case "event":
		if envelope.Event == nil {
			return ErrInvalid{errors.New("event correction body is required")}
		}
		current, err := store.CurrentEvents(ctx, run.GenerationID)
		if err != nil {
			return err
		}
		byID := map[string]events.Proposal{}
		for _, event := range current {
			byID[event.CandidateID] = event
		}
		result, err := events.ApplyCorrection(byID, *envelope.Event, actor, at, digest)
		if err != nil {
			if errors.Is(err, entities.ErrConflict) && replayedEvent(ctx, store, run, CorrectionIDs(digest, 1)[0]) {
				return nil
			}
			if errors.Is(err, entities.ErrConflict) {
				return err
			}
			return ErrInvalid{err}
		}
		for i := range result.Insert {
			result.Insert[i].GenerationID = run.GenerationID
		}
		return store.ReplaceEvents(ctx, OwnerRunID(run.GenerationID), CorrectionIDs(digest, len(result.Insert)), result.Insert, result.Supersede)
	default:
		return ErrInvalid{fmt.Errorf("target must be entity or event, not %q", envelope.Target)}
	}
}

// replayed reports whether a correction's first row already exists among
// the generation's current or superseded proposals (a retried request).
func replayed(ctx context.Context, store Store, run flow.RunRef, id string) bool {
	current, err := store.CurrentEntities(ctx, run.GenerationID)
	if err != nil {
		return false
	}
	for _, proposal := range current {
		if proposal.CandidateID == id {
			return true
		}
	}
	return false
}

func replayedEvent(ctx context.Context, store Store, run flow.RunRef, id string) bool {
	current, err := store.CurrentEvents(ctx, run.GenerationID)
	if err != nil {
		return false
	}
	for _, event := range current {
		if event.CandidateID == id {
			return true
		}
	}
	return false
}

// MarkEvent stages the owner's "event worth recalling" on one record.
func MarkEvent(ctx context.Context, store Store, run flow.RunRef, request events.MarkRequest, actor entities.Actor, at time.Time, digest string) (events.Proposal, error) {
	id := flow.DeterministicID("owner_mark", digest)
	if replayedEvent(ctx, store, run, id) {
		current, _ := store.CurrentEvents(ctx, run.GenerationID)
		for _, event := range current {
			if event.CandidateID == id {
				return event, nil
			}
		}
	}
	message, err := store.MessageByID(ctx, run.GenerationID, request.RecordID)
	if err != nil {
		return events.Proposal{}, err
	}
	proposal, err := events.FromRecord(message, request, run.Scope(), actor, at)
	if err != nil {
		return events.Proposal{}, ErrInvalid{err}
	}
	if err := ensureOwnerRun(ctx, store, run); err != nil {
		return events.Proposal{}, err
	}
	if err := store.ReplaceEvents(ctx, OwnerRunID(run.GenerationID), []string{id}, []events.Proposal{proposal}, nil); err != nil {
		return events.Proposal{}, err
	}
	proposal.CandidateID = id
	proposal.ReviewState = entities.StatePending
	return proposal, nil
}

// ---- commit row builders (pure) -------------------------------------------

// IncludedEntities filters proposals that take part in a commit.
func IncludedEntities(list []entities.Proposal) []entities.Proposal {
	var out []entities.Proposal
	for _, proposal := range list {
		if proposal.Included() {
			out = append(out, proposal)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CandidateID < out[j].CandidateID })
	return out
}

// IncludedEvents filters events that take part in a commit.
func IncludedEvents(list []events.Proposal) []events.Proposal {
	var out []events.Proposal
	for _, event := range list {
		if event.Included() {
			out = append(out, event)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CandidateID < out[j].CandidateID })
	return out
}

// TargetEntityID is the registry entity a proposal commits into: its match,
// the entity it was already promoted to, or a new id derived from it.
func TargetEntityID(proposal entities.Proposal) string {
	switch {
	case proposal.ReviewState == entities.StateApproved && proposal.PromotedToID != "":
		return proposal.PromotedToID
	case proposal.Match != nil && proposal.Match.EntityID != "":
		return proposal.Match.EntityID
	default:
		return flow.DeterministicID("registry.entity", proposal.CandidateID)
	}
}

// EntityRows are the new registry.entity rows (proposals without a match).
func EntityRows(included []entities.Proposal, receiptID string) []EntityRow {
	var rows []EntityRow
	for _, proposal := range included {
		if proposal.Match != nil {
			continue
		}
		tier := "inferred"
		for _, extractor := range proposal.Extractors {
			if strings.HasPrefix(extractor, entities.RulesExtractor) {
				tier = "extracted"
			}
		}
		rows = append(rows, EntityRow{
			ID: TargetEntityID(proposal), RegistryType: string(proposal.RegistryType), Name: proposal.Name,
			NormalizedName: entities.RegistryNormalizedName(proposal.Name), DataTier: tier,
			Confidence: roundConfidence(proposal.Confidence), CandidateID: proposal.CandidateID, ReceiptID: receiptID,
		})
	}
	return rows
}

// AliasRows are the registry.entity_alias rows of the included proposals.
func AliasRows(included []entities.Proposal) []AliasRow {
	var rows []AliasRow
	for _, proposal := range included {
		entityID := TargetEntityID(proposal)
		if proposal.Match != nil && proposal.Match.DisplayName != "" && entities.NameKey(proposal.Name) != entities.NameKey(proposal.Match.DisplayName) && !entities.LooksLikeAddress(proposal.Name) && !proposal.SourceOwner {
			// Merging into a committed entity: the proposal's own name is an
			// alias of that entity.
			rows = append(rows, AliasRow{
				ID:       flow.DeterministicID("registry.entity_alias", entityID, string(entities.AliasOther), entities.NameKey(proposal.Name)),
				EntityID: entityID, Text: proposal.Name, Kind: string(entities.AliasOther), Confidence: roundConfidence(proposal.Confidence), CandidateID: proposal.CandidateID,
			})
		}
		for _, alias := range commitcheck.WritableAliases(proposal) {
			text := alias.Text
			normalized := entities.NameKey(alias.Text)
			if alias.IsAddress() {
				text, normalized = alias.Normalized, alias.Normalized
			}
			rows = append(rows, AliasRow{
				ID:       flow.DeterministicID("registry.entity_alias", entityID, string(alias.Kind), normalized),
				EntityID: entityID, Text: text, Kind: string(alias.Kind), Confidence: roundConfidence(alias.Confidence), CandidateID: proposal.CandidateID,
			})
		}
	}
	return rows
}

func roundConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return float64(int(value*1000+0.5)) / 1000
}

// MentionID is entity-independent: re-resolving a span never duplicates it.
func MentionID(mention entities.Mention) string {
	return flow.DeterministicID("working.entity_mention", mention.SpanKey())
}

// MentionPlan collects one page's mentions and resolutions, first claim wins
// per span so two entities can never resolve the same text.
type MentionPlan struct {
	Mentions    []MentionRow
	Resolutions []ResolutionRow
	Conflicts   int
	claimed     map[string]string
}

// NewMentionPlan starts an empty plan.
func NewMentionPlan() *MentionPlan { return &MentionPlan{claimed: map[string]string{}} }

// Add records one mention of one proposal.
func (p *MentionPlan) Add(mention entities.Mention, proposal entities.Proposal) {
	id := MentionID(mention)
	entityID := TargetEntityID(proposal)
	if owner, ok := p.claimed[id]; ok {
		if owner != entityID {
			p.Conflicts++
		}
		return
	}
	p.claimed[id] = entityID
	p.Mentions = append(p.Mentions, MentionRow{
		ID: id, Surface: mention.Surface, Kind: mention.Kind, RecordID: mention.RecordID,
		Start: mention.Start, End: mention.End, Snippet: mention.Snippet, Method: mention.Method,
		Confidence: roundConfidence(mention.Confidence),
	})
	method, resolvedBy := "resolved", "rule"
	switch {
	case mention.Method == entities.MethodParticipant:
		method = "exact"
	case strings.HasPrefix(mention.Method, "model:"):
		resolvedBy = "model"
	}
	p.Resolutions = append(p.Resolutions, ResolutionRow{
		ID: flow.DeterministicID("working.entity_resolution", id, entityID), MentionID: id, EntityID: entityID,
		MatchMethod: method, ResolvedBy: resolvedBy, Score: roundConfidence(mention.Confidence),
		Metrics:     map[string]any{"surface": mention.Surface, "method": mention.Method, "role": mention.Role, "candidate_id": proposal.CandidateID},
		CandidateID: proposal.CandidateID,
	})
}

// MessageMentions adds every mention one message makes of the included
// proposals: model mentions first (most specific), then participant
// headers, then body occurrences of each proposal's name aliases.
func (p *MentionPlan) MessageMentions(message entities.MessageView, included []entities.Proposal, sourceVersionID string) {
	for _, proposal := range included {
		for _, mention := range proposal.ModelMentions {
			if mention.RecordID == message.RecordID {
				p.Add(mention, proposal)
			}
		}
	}
	for _, proposal := range included {
		for _, mention := range entities.ParticipantMentions(message, proposal, sourceVersionID) {
			p.Add(mention, proposal)
		}
	}
	for _, proposal := range included {
		for _, mention := range entities.BodyMentions(message, proposal) {
			p.Add(mention, proposal)
		}
	}
}

// EventRows builds the timeline.event_candidate rows of the included events.
func EventRows(included []events.Proposal, entityProposals []entities.Proposal, receiptID string) ([]EventCandidateRow, error) {
	byCandidate := map[string]entities.Proposal{}
	for _, proposal := range entityProposals {
		byCandidate[proposal.CandidateID] = proposal
	}
	var rows []EventCandidateRow
	for _, event := range included {
		available := event.SourceAvailableFrom()
		if available == nil || event.OccurredAt == nil || len(event.SourceRecords) == 0 {
			return nil, fmt.Errorf("event %q is missing its time, sources or source clock", event.Title)
		}
		resolved, unresolved, ambiguous := events.ResolveEntityKeys(event, entityProposals)
		if len(unresolved)+len(ambiguous) > 0 {
			return nil, fmt.Errorf("event %q names entities that do not resolve", event.Title)
		}
		refSet := map[string]bool{}
		for _, candidateID := range resolved {
			refSet[EntityRefPrefix+TargetEntityID(byCandidate[candidateID])] = true
		}
		refs := make([]string, 0, len(refSet))
		for ref := range refSet {
			refs = append(refs, ref)
		}
		sort.Strings(refs)
		eventID := flow.DeterministicID("timeline.event_candidate", event.CandidateID)
		var recordIDs, values []string
		var locatorRecords []map[string]any
		for _, record := range event.SourceRecords {
			recordIDs = append(recordIDs, record.RecordID)
			entry := map[string]any{"record_id": record.RecordID, "ordinal": record.Ordinal}
			if record.OccurredAt != nil {
				entry["occurred_at"] = record.OccurredAt.UTC().Format(time.RFC3339Nano)
			}
			if record.SourceAvailableFrom != nil {
				entry["source_available_from"] = record.SourceAvailableFrom.UTC().Format(time.RFC3339Nano)
				values = append(values, record.SourceAvailableFrom.UTC().Format(time.RFC3339Nano))
			}
			if record.Start != nil && record.End != nil {
				entry["start_char"], entry["end_char"] = *record.Start, *record.End
			}
			locatorRecords = append(locatorRecords, entry)
		}
		anchorMetadata := map[string]any{
			"rule":    "latest source_available_from among the event's source records",
			"records": recordIDs, "values": values,
		}
		digestInput, _ := json.Marshal(map[string]any{"event_candidate_id": eventID, "metadata": anchorMetadata})
		digest := sha256.Sum256(digestInput)
		rows = append(rows, EventCandidateRow{
			ID: eventID, SourceRecordID: event.PrimaryRecordID(),
			SourceRecordVersion: event.GenerationID + "#" + event.ContentHex()[:16],
			SourceLocator: map[string]any{
				"schema": "probata.event-source/v1", "preview_handle": event.PreviewHandle,
				"normalized_generation_id": event.GenerationID, "records": locatorRecords,
				"candidate_event_id": event.CandidateID, "detected_by": event.DetectedBy,
				"title": event.Title, "description": event.Description, "when_stated": event.WhenStated,
				"source_available_from": available.UTC().Format(time.RFC3339Nano),
			},
			ExtractionRunID: receiptID, Precision: event.TemporalPrecision, OccurredAt: event.OccurredAt,
			TemporalConfidence: event.TemporalConfidence, Summary: event.Title, EventType: event.EventType,
			EntityRefs: refs, AvailableFrom: *available,
			AnchorID:       flow.DeterministicID("relative_time_anchor", eventID, "source_available_from"),
			AnchorKey:      flow.DeterministicID("relative_time_anchor_key", eventID, "source_available_from"),
			AnchorMetadata: anchorMetadata, ProvenanceDigest: digest[:], CandidateID: event.CandidateID,
		})
	}
	return rows, nil
}

// MemberRows puts committed event candidates into a timeline collection.
func MemberRows(rows []EventCandidateRow, collectionID string) []TimelineMemberRow {
	out := make([]TimelineMemberRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, TimelineMemberRow{
			ID: flow.DeterministicID("timeline.timeline_member", collectionID, row.ID), CollectionID: collectionID, CandidateID: row.ID,
		})
	}
	return out
}

// Promotions lists the staging rows a successful commit promotes.
func Promotions(included []entities.Proposal, includedEvents []events.Proposal) ([]Promotion, []Promotion) {
	var entityPromotions, eventPromotions []Promotion
	for _, proposal := range included {
		entityPromotions = append(entityPromotions, Promotion{CandidateID: proposal.CandidateID, Table: "registry.entity", TargetID: TargetEntityID(proposal)})
	}
	for _, event := range includedEvents {
		eventPromotions = append(eventPromotions, Promotion{CandidateID: event.CandidateID, Table: "timeline.event_candidate", TargetID: flow.DeterministicID("timeline.event_candidate", event.CandidateID)})
	}
	return entityPromotions, eventPromotions
}
