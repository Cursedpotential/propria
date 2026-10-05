// Package activities: this file owns publish_context_search_activity only.
//
// It is the Weaviate-first step of the owner's ruled pipeline order
// (2026-09-18 20:06-20:07 "IT ALL GOES TO WEAVIATE FIRST", restated
// 2026-09-26):
//
//	extract -> everything searchable in Weaviate -> owner searches, reads,
//	validates, adds metadata and context, verifies the extraction, repairs
//	the file -> THEN commit to the canonical PostgreSQL tables.
//
// It runs on every Proffer run after verify_normalized_generation and before
// the preview, the owner's approval, seal_generation and publish_generation.
// A failure here stops the run before the commit: that is what "first" means.
//
// Targets are owner-approved and routed by record kind (OD-06, 2026-10-01;
// owner 2026-10-02): messages and calls -> MsgEvents20260918, AI chats ->
// AiChatEvents20260918, documents -> DocEvents20261001; any other kind fails
// closed. Probata's objects are told apart by origin_system and by run id.
//
// For human messages and calls, the disclosure tier is the application's one rule (engine/disclosure, owner
// 2026-10-02): contemporaneous when the owner took part in the record,
// discovered when he did not. It reads the run's participant resolution
// (resolve_context_participants_activity) by reference and never re-resolves,
// so this stage and the first-party commit cannot disagree.
// Verified AI sources preserve their normalized source dates and role labels in the AI collection,
// without a human participant resolution, disclosure tier or disclosure basis (ADR-0053 neutral AI landing).
//
// It follows the split this package already established (normalized_pipeline.go,
// entity_extraction.go): the Activity validates and converts compact
// Store-provided pages, each Store owns its own I/O, exactly one Activity owns
// each write, and everything fails closed. Records are streamed a page at a
// time; a generation is never materialized in Go or in Temporal history.
//
// Three boundaries, one write:
//   - Source (PostgreSQL) resolves provenance, loads the participant
//     resolution, pages the normalized records and writes the one receipt.
//   - Embedder (NVIDIA NIM) turns each page's text into text_nim vectors.
//   - Target (Weaviate) owns the search-object writes.
//
// It writes NO canonical PostgreSQL rows; its only PostgreSQL write is its own
// receipt in context.activity_receipt. It carries occurred_at, knowledge_time
// and human disclosure_tier and applies NO horizon filter.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (builds; embedder; MsgEvents20260918)
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (owner_participant tier; routing by kind)
// Byline: Claude Code · Sonnet 5.5 · 2026-10-02 (skip_record_kinds: messages and calls publish as chunks after the commit)
package activities

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/contextsearch"
	"github.com/Cursedpotential/probata/engine/disclosure"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// PublishContextSearchActivityName is the Temporal registration name, taken
// from the stage graph so the registered name and the graph identity cannot
// drift apart.
const PublishContextSearchActivityName = string(stagegraph.PublishContextSearch)

// publishContextSearchPageSize is one embed request. It matches the Case Bible
// writers' NIM batch (32-64) and stays well inside Weaviate's batch ceiling.
const publishContextSearchPageSize = 64

// PublishContextSearchSpec is the compact, already-resolved input.
type PublishContextSearchSpec struct {
	RequestID string
	Attempt   int32
	// SourceVersionRef and NormalizedGenerationRef select exactly which
	// extraction output becomes searchable.
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	// NormalizedVerificationRef proves extraction was verified before anything
	// is published for the owner to read.
	NormalizedVerificationRef proffer.Ref
	// ExtractionAttemptRef is carried into every object's provenance.
	ExtractionAttemptRef proffer.Ref
	// ParticipantResolutionRef is the run's one participant resolution
	// (resolve_context_participants_activity). Required for human sources; verified AI sources omit it.
	ParticipantResolutionRef proffer.Ref
	// OwnerPersonRef and PerspectivePersonRef are the run's explicit person
	// ids; when given they must match the resolution's.
	OwnerPersonRef       proffer.Ref
	PerspectivePersonRef proffer.Ref
	// MessageMatchesRef is the match-up receipt (match_message_occurrences_activity);
	// the messages it names already have a search object from an earlier source
	// and are not published again. Optional. Byline: Claude Code · Opus 5.5 · 2026-10-02
	MessageMatchesRef proffer.Ref
	// SkipRecordKinds names record kinds this pass does not publish per record because the run publishes them as
	// conversation chunks after the commit (owner 2026-10-02: Postgres holds every message, Weaviate only chunks;
	// calls are one entry per call-log file). Set from the "skip_record_kinds" reference, a comma-separated list of
	// message and call. AI chats and documents are never skipped. Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	SkipRecordKinds map[string]bool
}

// SkipRecordKindsRefKey is the StageRequest reference that carries SkipRecordKinds.
const SkipRecordKindsRefKey = proffer.SkipRecordKindsRefKey

// parseSkipRecordKinds reads the comma-separated list; only message and call may be skipped.
func parseSkipRecordKinds(ref proffer.Ref) (map[string]bool, error) {
	skip := map[string]bool{}
	for _, kind := range strings.Split(string(ref), ",") {
		kind = strings.TrimSpace(kind)
		switch kind {
		case "":
		case contextsearch.RecordKindMessage, contextsearch.RecordKindCall:
			skip[kind] = true
		default:
			return nil, fmt.Errorf("%s: record kind %q cannot be skipped (message and call only)", SkipRecordKindsRefKey, kind)
		}
	}
	return skip, nil
}

func (s PublishContextSearchSpec) validate() error {
	if strings.TrimSpace(s.RequestID) == "" {
		return fmt.Errorf("%s requires a request id", stagegraph.PublishContextSearch)
	}
	for _, field := range []struct {
		name string
		ref  proffer.Ref
	}{
		{"source version", s.SourceVersionRef},
		{"normalized generation", s.NormalizedGenerationRef},
		{"normalized verification", s.NormalizedVerificationRef},
		{"extraction attempt", s.ExtractionAttemptRef},
	} {
		if strings.TrimSpace(string(field.ref)) == "" {
			return fmt.Errorf("%s requires a %s reference", stagegraph.PublishContextSearch, field.name)
		}
	}
	if s.Attempt < 1 {
		return fmt.Errorf("%s attempt must be positive", stagegraph.PublishContextSearch)
	}
	return nil
}

// ContextSearchParticipant is one participant of a normalized record.
type ContextSearchParticipant struct {
	Role        string
	Identifier  string
	DisplayName string
}

// ContextSearchRecord is one compact normalized record, as the Source store
// pages it out of context.normalized_record_identity.
type ContextSearchRecord struct {
	RowID      uuid.UUID
	Ordinal    int64
	RecordType string

	OccurredAt           *time.Time
	OccurredAtRaw        string
	KnowledgeTime        time.Time
	ProvenanceClass      string
	TimestampCertainty   string
	TimestampGranularity string

	Body            string
	Direction       string
	Disposition     string
	DurationSeconds string
	Missed          string
	Participants    []ContextSearchParticipant

	// ContentSHA256 is sha256 of the row's canonical bytes, computed in SQL.
	ContentSHA256 []byte
}

// ContextSearchRecordReader streams records a page at a time. Next returns
// io.EOF when the stream is exhausted.
type ContextSearchRecordReader interface {
	Next(context.Context) (ContextSearchRecord, error)
	Close() error
}

// ContextSearchPlan is what the Source store resolves once per generation.
type ContextSearchPlan struct {
	// Provenance is shared by every object; RequestID, Attempt and
	// ExtractionAttemptRef are overwritten from the spec.
	Provenance contextsearch.Provenance
	// Coordinates carries the generation-wide coordinates; RowID is per record.
	Coordinates contextsearch.Coordinates
	// FormatID is the raw generation's format; with Provenance.SourceFormat it
	// decides whether the source is a conversation with AI.
	FormatID string
	// Resolution is the run's participant resolution.
	Resolution disclosure.Resolution
	Reader     ContextSearchRecordReader
}

// ContextSearchSourceStore is the PostgreSQL boundary.
type ContextSearchSourceStore interface {
	OpenContextSearchRecords(context.Context, PublishContextSearchSpec) (ContextSearchPlan, error)
	// PersistContextSearchPublication writes the one receipt. It is
	// retry-safe through context.activity_execution: a repeated coordinate
	// returns the existing durable outcome.
	PersistContextSearchPublication(context.Context, PublishContextSearchSpec, ContextSearchPublicationOutcome) (proffer.Ref, proffer.Ref, error)
}

// ContextSearchEmbedder turns texts into vectors, one per text, in order.
type ContextSearchEmbedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

// ContextSearchTarget is the Weaviate boundary. EnsureCollection never drops
// or alters an existing property and creates only a collection the owner
// approved for creation; PublishObjects is idempotent under a repeated id.
type ContextSearchTarget interface {
	EnsureCollection(ctx context.Context, collection string) ([]string, error)
	PublishObjects(ctx context.Context, collection string, objects []contextsearch.Object) (ContextSearchPublishResult, error)
}

// ContextSearchPublishResult is one target write's outcome.
type ContextSearchPublishResult struct {
	Requested int
	Written   int
	ObjectIDs []string
}

// ContextSearchPublicationOutcome is the durable result of one publish pass.
type ContextSearchPublicationOutcome struct {
	// Collections is objects written per collection.
	Collections          map[string]int
	Published            int
	VectorsPublished     int
	ObjectIDConstruction string
	DedupKeyConstruction string
	DisclosureBasis      string
	// Tiers is objects per disclosure tier.
	Tiers         map[string]int
	FirstObjectID string
	LastObjectID  string
	// PropertiesAdded names, per collection, properties this pass added (every
	// property, for a collection this pass created).
	PropertiesAdded map[string][]string
	// SkippedMatched counts messages not published because an earlier source
	// already holds them (the match-up). Byline: Claude Code · Opus 5.5 · 2026-10-02
	SkippedMatched int `json:"skipped_matched,omitempty"`
	// SkippedToChunks counts message and call records not published one by one because the run publishes them as
	// conversation chunks and call-log files after the commit. Byline: Claude Code · Sonnet 5.5 · 2026-10-02
	SkippedToChunks int `json:"skipped_to_chunks,omitempty"`
}

// PublishContextSearchActivities implements publish_context_search_activity.
type PublishContextSearchActivities struct {
	Source   ContextSearchSourceStore
	Embedder ContextSearchEmbedder
	Target   ContextSearchTarget
	// Collections routes each record kind to its owner-approved collection.
	// Every kind must be routed; there are no defaults.
	Collections map[string]string
	Attempt     Attempt
	Heartbeat   Heartbeat
	// Matches loads a match-up receipt's record ids; required only when a run
	// passes one. Byline: Claude Code · Opus 5.5 · 2026-10-02
	Matches interface {
		LoadMatchedRecords(ctx context.Context, ref proffer.Ref) (map[string]bool, error)
	}
}

// ContextSearchRecordKinds are the kinds that must each have a collection.
var ContextSearchRecordKinds = []string{
	contextsearch.RecordKindMessage, contextsearch.RecordKindCall,
	contextsearch.RecordKindAIChat, contextsearch.RecordKindDocument,
}

func (a PublishContextSearchActivities) validate() error {
	if a.Source == nil {
		return errors.New("publish context search activities: source store is required")
	}
	if a.Embedder == nil {
		return errors.New("publish context search activities: embedder is required; the collections are searched by their text_nim vector")
	}
	if a.Target == nil {
		return errors.New("publish context search activities: target is required")
	}
	for _, kind := range ContextSearchRecordKinds {
		if strings.TrimSpace(a.Collections[kind]) == "" {
			return fmt.Errorf("publish context search activities: no collection is configured for record kind %q; vector-store collection names are owner-approved and have no default", kind)
		}
	}
	return nil
}

func (a PublishContextSearchActivities) attempt(ctx context.Context) int32 {
	if a.Attempt == nil {
		return 1
	}
	if attempt := a.Attempt(ctx); attempt >= 1 {
		return attempt
	}
	return 1
}

func (a PublishContextSearchActivities) heartbeat(ctx context.Context, done int64) {
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, Progress{Stage: stagegraph.PublishContextSearch, MembersComplete: done})
	}
}

// publishContextSearchSpecFrom resolves the compact spec from the StageRequest.
// A missing required reference means the workflow reached this stage without
// its upstream gate.
func publishContextSearchSpecFrom(req proffer.StageRequest, attempt int32) (PublishContextSearchSpec, error) {
	skip, err := parseSkipRecordKinds(req.Refs[SkipRecordKindsRefKey])
	if err != nil {
		return PublishContextSearchSpec{}, err
	}
	spec := PublishContextSearchSpec{
		RequestID: req.RequestID, Attempt: attempt, SourceVersionRef: req.SourceVersionRef,
		OwnerPersonRef: req.Refs["owner_person"], PerspectivePersonRef: req.Refs["perspective_person"],
		MessageMatchesRef: req.Refs["message_matches"], SkipRecordKinds: skip,
		ParticipantResolutionRef: req.Refs["participant_resolution"],
	}
	for _, field := range []struct {
		name   string
		target *proffer.Ref
	}{
		{"normalized_generation", &spec.NormalizedGenerationRef},
		{"normalized_verification", &spec.NormalizedVerificationRef},
		{"extraction_attempt", &spec.ExtractionAttemptRef},
	} {
		ref, ok := req.Refs[field.name]
		if !ok || strings.TrimSpace(string(ref)) == "" {
			return PublishContextSearchSpec{}, fmt.Errorf("%s requires the %q reference", stagegraph.PublishContextSearch, field.name)
		}
		*field.target = ref
	}
	if err := spec.validate(); err != nil {
		return PublishContextSearchSpec{}, err
	}
	return spec, nil
}

// PublishContextSearch makes one verified extraction output searchable in
// Weaviate, before any canonical PostgreSQL commit. It is idempotent: object
// ids derive from source-anchored dedup keys and the target write replaces an
// existing id, so a retry leaves every collection's count unchanged.
func (a PublishContextSearchActivities) PublishContextSearch(ctx context.Context, req proffer.StageRequest) (proffer.StageResult, error) {
	if err := a.validate(); err != nil {
		return proffer.StageResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return proffer.StageResult{}, err
	}
	spec, err := publishContextSearchSpecFrom(req, a.attempt(ctx))
	if err != nil {
		return proffer.StageResult{}, err
	}

	plan, err := a.Source.OpenContextSearchRecords(ctx, spec)
	if err != nil {
		return proffer.StageResult{}, fmt.Errorf("open context search records: %w", err)
	}
	if plan.Reader == nil {
		return proffer.StageResult{}, errors.New("context search plan carries no record reader")
	}
	defer plan.Reader.Close()
	aiChat := contextsearch.IsAIChatFormat(plan.Provenance.SourceFormat) || contextsearch.IsAIChatFormat(plan.FormatID)
	if !aiChat {
		if spec.ParticipantResolutionRef == "" {
			return proffer.StageResult{}, errors.New("human context search requires a participant resolution reference")
		}
		if err := checkResolutionMatchesRun(plan.Resolution, spec); err != nil {
			return proffer.StageResult{}, err
		}
	}

	matched := map[string]bool{}
	if !aiChat && spec.MessageMatchesRef != "" {
		if a.Matches == nil {
			return proffer.StageResult{}, errors.New("the run passes a message match-up receipt but this worker has no reader for it")
		}
		if matched, err = a.Matches.LoadMatchedRecords(ctx, spec.MessageMatchesRef); err != nil {
			return proffer.StageResult{}, err
		}
	}
	outcome, err := a.publishStream(ctx, spec, plan, matched)
	if err != nil {
		return proffer.StageResult{}, err
	}

	resultRef, receiptRef, err := a.Source.PersistContextSearchPublication(ctx, spec, outcome)
	if err != nil {
		return proffer.StageResult{}, fmt.Errorf("persist context search publication: %w", err)
	}
	if resultRef == "" || receiptRef == "" {
		return proffer.StageResult{}, errors.New("persisted context search publication lacks result or activity receipt reference")
	}
	return success(stagegraph.PublishContextSearch, resultRef, receiptRef), nil
}

// checkResolutionMatchesRun fails closed when the resolution was made for a
// different owner or perspective than this run names.
func checkResolutionMatchesRun(resolution disclosure.Resolution, spec PublishContextSearchSpec) error {
	if err := resolution.Validate(); err != nil {
		return err
	}
	if owner := strings.TrimSpace(string(spec.OwnerPersonRef)); owner != "" && !strings.EqualFold(owner, resolution.OwnerPersonID) {
		return fmt.Errorf("participant resolution was made for owner %s, but the run names %s", resolution.OwnerPersonID, owner)
	}
	if perspective := strings.TrimSpace(string(spec.PerspectivePersonRef)); perspective != "" && !strings.EqualFold(perspective, resolution.PerspectivePersonID) {
		return fmt.Errorf("participant resolution was made for perspective %q, but the run names %s", resolution.PerspectivePersonID, perspective)
	}
	return nil
}

// publishStream drains the reader a page at a time: build objects, embed the
// page in one request, write each collection's share of the page. Fail-closed:
// one bad record, a short embed response or one rejected object fails the
// Activity, because a silently missing record is one the owner would never find.
func (a PublishContextSearchActivities) publishStream(ctx context.Context, spec PublishContextSearchSpec, plan ContextSearchPlan, matched map[string]bool) (ContextSearchPublicationOutcome, error) {
	outcome := ContextSearchPublicationOutcome{
		Collections:          map[string]int{},
		ObjectIDConstruction: contextsearch.ObjectIDConstruction,
		DedupKeyConstruction: contextsearch.DedupKeyConstruction,
		DisclosureBasis:      disclosure.Basis,
		Tiers:                map[string]int{},
		PropertiesAdded:      map[string][]string{},
	}
	ensured := map[string]bool{}
	aiChat := contextsearch.IsAIChatFormat(plan.Provenance.SourceFormat) || contextsearch.IsAIChatFormat(plan.FormatID)
	if aiChat {
		outcome.DisclosureBasis = ""
	}
	resolution := plan.Resolution
	page := make([]contextsearch.Object, 0, publishContextSearchPageSize)

	flush := func() error {
		if len(page) == 0 {
			return nil
		}
		texts := make([]string, len(page))
		for i := range page {
			texts[i] = page[i].SearchText
		}
		vectors, err := a.Embedder.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed context search page: %w", err)
		}
		if len(vectors) != len(page) {
			return fmt.Errorf("embedder returned %d vectors for %d records", len(vectors), len(page))
		}
		byCollection := map[string][]contextsearch.Object{}
		var order []string
		for i := range page {
			if len(vectors[i]) == 0 {
				return fmt.Errorf("embedder returned an empty vector for record %s", page[i].Coordinates.RowID)
			}
			page[i].Vector = vectors[i]
			collection := a.Collections[page[i].RecordKind]
			if _, seen := byCollection[collection]; !seen {
				order = append(order, collection)
			}
			byCollection[collection] = append(byCollection[collection], page[i])
		}
		for _, collection := range order {
			objects := byCollection[collection]
			if !ensured[collection] {
				added, err := a.Target.EnsureCollection(ctx, collection)
				if err != nil {
					return fmt.Errorf("ensure context search collection %s: %w", collection, err)
				}
				ensured[collection] = true
				if len(added) > 0 {
					outcome.PropertiesAdded[collection] = added
				}
			}
			result, err := a.Target.PublishObjects(ctx, collection, objects)
			if err != nil {
				return fmt.Errorf("publish context search objects to %s: %w", collection, err)
			}
			if result.Written != len(objects) {
				return fmt.Errorf("context search target wrote %d of %d objects to %s", result.Written, len(objects), collection)
			}
			if len(result.ObjectIDs) > 0 {
				if outcome.FirstObjectID == "" {
					outcome.FirstObjectID = result.ObjectIDs[0]
				}
				outcome.LastObjectID = result.ObjectIDs[len(result.ObjectIDs)-1]
			}
			outcome.Collections[collection] += result.Written
			outcome.Published += result.Written
			outcome.VectorsPublished += result.Written
			for _, object := range objects {
				if object.Temporal.DisclosureTier != "" {
					outcome.Tiers[object.Temporal.DisclosureTier]++
				}
			}
		}
		page = page[:0]
		a.heartbeat(ctx, int64(outcome.Published))
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return outcome, err
		}
		record, err := plan.Reader.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return outcome, fmt.Errorf("read context search record: %w", err)
		}
		if !aiChat && matched[strings.ToLower(record.RowID.String())] {
			outcome.SkippedMatched++
			continue
		}
		// A kind the run publishes as chunks after the commit: counted, not published, not embedded. A record whose
		// kind cannot be routed falls through so contextSearchObjectFrom reports it, exactly as before.
		if kind, kindErr := contextSearchRecordKind(record.RecordType, aiChat); kindErr == nil && spec.SkipRecordKinds[kind] {
			outcome.SkippedToChunks++
			continue
		}
		object, err := contextSearchObjectFrom(record, plan, &resolution, spec, aiChat)
		if err != nil {
			return outcome, err
		}
		page = append(page, object)
		if len(page) >= publishContextSearchPageSize {
			if err := flush(); err != nil {
				return outcome, err
			}
		}
	}
	if err := flush(); err != nil {
		return outcome, err
	}
	if outcome.Published == 0 && outcome.SkippedMatched == 0 && outcome.SkippedToChunks == 0 {
		return outcome, errors.New("context search publication produced no objects; a verified normalized generation with no publishable records is a defect, not an empty success")
	}
	return outcome, nil
}

// contextSearchRecordKind routes a normalized record type to a record kind.
// Anything without an owner-approved collection fails closed.
func contextSearchRecordKind(recordType string, aiChat bool) (string, error) {
	switch recordType {
	case contextsearch.RecordKindMessage:
		if aiChat {
			return contextsearch.RecordKindAIChat, nil
		}
		return contextsearch.RecordKindMessage, nil
	case contextsearch.RecordKindCall:
		if aiChat {
			return "", errors.New("a call record in a conversation-with-AI source has no owner-approved collection")
		}
		return contextsearch.RecordKindCall, nil
	case contextsearch.RecordKindDocument:
		return contextsearch.RecordKindDocument, nil
	default:
		return "", fmt.Errorf("record type %q has no owner-approved collection (messages, calls, AI chats and documents only)", recordType)
	}
}

// statedParticipants splits a record's participants into the sender and
// recipients the disclosure rule takes, exactly as the source states them.
// A participant of unknown role is a party all the same, so it counts with the
// recipients. A call log names only the other party: the device's own owner is
// a party to every call on it, so a call carries the device marker "self",
// which the rule resolves through the run's perspective person.
func statedParticipants(record ContextSearchRecord) (string, []string) {
	var sender string
	var recipients []string
	for _, participant := range record.Participants {
		identifier := strings.TrimSpace(participant.Identifier)
		if identifier == "" {
			continue
		}
		if participant.Role == "sender" && sender == "" {
			sender = participant.Identifier
			continue
		}
		recipients = append(recipients, participant.Identifier)
	}
	if record.RecordType == contextsearch.RecordKindCall {
		recipients = append(recipients, disclosure.SelfIdentifier)
	}
	return sender, recipients
}

// contextSearchObjectFrom converts one verified record into a source-linked search object.
// Inputs: persisted plan, normalized record, attempt refs and source classification. Output: validated human or neutral AI object.
// Effects: none. Choose for per-record publication; human resolution remains required and AI roles are copied verbatim.
//
// The dedup discriminator is the record's ordinal within its source version:
// the normalizer is 1:1 and order-preserving over the raw records
// (normalize.GenericMessageNormalizer), so re-running the same source version
// yields the same ordinals and therefore the same object ids. The row id is
// never used: it is re-minted on every re-extraction.
func contextSearchObjectFrom(record ContextSearchRecord, plan ContextSearchPlan, resolution *disclosure.Resolution, spec PublishContextSearchSpec, aiChat bool) (contextsearch.Object, error) {
	kind, err := contextSearchRecordKind(record.RecordType, aiChat)
	if err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	coordinates := plan.Coordinates
	coordinates.RowID = record.RowID
	dedupKey, err := contextsearch.DedupKey(coordinates.SourceVersionID, fmt.Sprintf("record_ordinal:%d", record.Ordinal), nil)
	if err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	provenance := plan.Provenance
	provenance.RawFormatID = plan.FormatID
	provenance.Attempt = spec.Attempt
	provenance.RequestID = spec.RequestID
	provenance.ExtractionAttemptRef = string(spec.ExtractionAttemptRef)

	var tier, basis string
	if !aiChat {
		if resolution == nil {
			return contextsearch.Object{}, errors.New("human context search requires participant resolution")
		}
		sender, recipients := statedParticipants(record)
		_, tier, basis, err = resolution.ForMessage(sender, recipients)
		if err != nil {
			return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
		}
	}

	people := contextSearchPeople(record.Participants)
	if aiChat {
		for _, participant := range record.Participants {
			if participant.Role != "" {
				people.RoleLabels = append(people.RoleLabels, participant.Role)
			}
		}
	}
	object := contextsearch.Object{
		DedupKey:    dedupKey,
		Coordinates: coordinates,
		Provenance:  provenance,
		Temporal: contextsearch.Temporal{
			OccurredAt:           record.OccurredAt,
			OccurredAtRaw:        record.OccurredAtRaw,
			KnowledgeTime:        record.KnowledgeTime,
			DisclosureTier:       tier,
			DisclosureTierBasis:  basis,
			TimestampCertainty:   record.TimestampCertainty,
			TimestampGranularity: record.TimestampGranularity,
		},
		People:          people,
		RecordKind:      kind,
		Direction:       record.Direction,
		Body:            record.Body,
		SearchText:      contextSearchText(record, people),
		ProvenanceClass: record.ProvenanceClass,
		ContentSHA256:   record.ContentSHA256,
	}
	if err := object.Validate(); err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	return object, nil
}

func contextSearchPeople(participants []ContextSearchParticipant) contextsearch.People {
	var people contextsearch.People
	seen := map[string]bool{}
	names := map[string]bool{}
	for _, participant := range participants {
		identifier := strings.TrimSpace(participant.Identifier)
		if identifier == "" {
			continue
		}
		switch participant.Role {
		case "sender":
			if people.Sender == "" {
				people.Sender = identifier
			}
		case "recipient":
			people.Recipients = append(people.Recipients, identifier)
		}
		if !seen[identifier] {
			seen[identifier] = true
			people.Participants = append(people.Participants, identifier)
		}
		if name := strings.TrimSpace(participant.DisplayName); name != "" && !names[name] {
			names[name] = true
			people.ContactNames = append(people.ContactNames, name)
		}
	}
	return people
}

// contextSearchText is the text embedded and BM25-searched: the body, or for
// a record without one (a call), a short description from its own fields --
// the same shape the Case Bible writer builds (elt_run.py publish).
func contextSearchText(record ContextSearchRecord, people contextsearch.People) string {
	if body := strings.TrimSpace(record.Body); body != "" {
		return body
	}
	parts := []string{record.RecordType}
	if record.Missed == "true" {
		parts = append(parts, "missed")
	}
	for _, value := range []string{record.Direction, record.Disposition} {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, value)
		}
	}
	if strings.TrimSpace(record.DurationSeconds) != "" {
		parts = append(parts, record.DurationSeconds+"s")
	}
	if len(people.Participants) > 0 {
		parts = append(parts, "with "+strings.Join(people.Participants, ", "))
	}
	return strings.Join(parts, " ")
}
