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
// The target is the owner-approved MsgEvents20260918 (OD-06, 2026-10-01): one
// place to search, Probata's objects told apart by origin_system and by run id.
//
// It follows the split this package already established (normalized_pipeline.go,
// entity_extraction.go): the Activity validates and converts compact
// Store-provided pages, each Store owns its own I/O, exactly one Activity owns
// each write, and everything fails closed. Records are streamed a page at a
// time; a generation is never materialized in Go or in Temporal history.
//
// Three boundaries, one write:
//   - Source (PostgreSQL) resolves provenance, pages the normalized records and
//     writes the one durable receipt.
//   - Embedder (NVIDIA NIM) turns each page's text into text_nim vectors.
//   - Target (Weaviate) owns the search-object write.
//
// It writes NO canonical PostgreSQL rows; its only PostgreSQL write is its own
// receipt in context.activity_receipt. It carries occurred_at, knowledge_time
// and disclosure_tier and applies NO horizon filter.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (builds; embedder; MsgEvents20260918)
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
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// PublishContextSearchActivityName is the Temporal registration name, taken
// from the stage graph so the registered name and the graph identity cannot
// drift apart.
const PublishContextSearchActivityName = string(stagegraph.PublishContextSearch)

// publishContextSearchPageSize is one embed request and one target write. It
// matches the Case Bible writers' NIM batch (32-64) and stays well inside
// Weaviate's batch ceiling.
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
	Reader      ContextSearchRecordReader
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

// ContextSearchTarget is the Weaviate boundary. EnsureCollection must never
// create, drop or alter an existing property; PublishObjects must be
// idempotent under a repeated object id.
type ContextSearchTarget interface {
	EnsureCollection(context.Context) ([]string, error)
	PublishObjects(context.Context, []contextsearch.Object) (ContextSearchPublishResult, error)
}

// ContextSearchPublishResult is one target write's outcome.
type ContextSearchPublishResult struct {
	Requested int
	Written   int
	ObjectIDs []string
}

// ContextSearchPublicationOutcome is the durable result of one publish pass.
type ContextSearchPublicationOutcome struct {
	Collection           string
	Published            int
	VectorsPublished     int
	ObjectIDConstruction string
	DedupKeyConstruction string
	FirstObjectID        string
	LastObjectID         string
	// PropertiesAdded names collection properties this pass added.
	PropertiesAdded []string
}

// PublishContextSearchActivities implements publish_context_search_activity.
type PublishContextSearchActivities struct {
	Source   ContextSearchSourceStore
	Embedder ContextSearchEmbedder
	Target   ContextSearchTarget
	// Collection is recorded in the receipt; it must equal the Target's.
	Collection string
	Attempt    Attempt
	Heartbeat  Heartbeat
}

func (a PublishContextSearchActivities) validate() error {
	if a.Source == nil {
		return errors.New("publish context search activities: source store is required")
	}
	if a.Embedder == nil {
		return errors.New("publish context search activities: embedder is required; the collection is searched by its text_nim vector")
	}
	if a.Target == nil {
		return errors.New("publish context search activities: target is required")
	}
	if strings.TrimSpace(a.Collection) == "" {
		return errors.New("publish context search activities: collection name is required and has no default; vector-store collection names are owner-approved")
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
// Every reference is required: a missing one means the workflow reached this
// stage without its upstream gate.
func publishContextSearchSpecFrom(req proffer.StageRequest, attempt int32) (PublishContextSearchSpec, error) {
	spec := PublishContextSearchSpec{RequestID: req.RequestID, Attempt: attempt, SourceVersionRef: req.SourceVersionRef}
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
// existing id, so a retry leaves the collection count unchanged.
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

	added, err := a.Target.EnsureCollection(ctx)
	if err != nil {
		return proffer.StageResult{}, fmt.Errorf("ensure context search collection %s: %w", a.Collection, err)
	}

	outcome, err := a.publishStream(ctx, spec, plan)
	if err != nil {
		return proffer.StageResult{}, err
	}
	outcome.PropertiesAdded = added

	resultRef, receiptRef, err := a.Source.PersistContextSearchPublication(ctx, spec, outcome)
	if err != nil {
		return proffer.StageResult{}, fmt.Errorf("persist context search publication: %w", err)
	}
	if resultRef == "" || receiptRef == "" {
		return proffer.StageResult{}, errors.New("persisted context search publication lacks result or activity receipt reference")
	}
	return success(stagegraph.PublishContextSearch, resultRef, receiptRef), nil
}

// publishStream drains the reader a page at a time: build objects, embed the
// page in one request, write the page. Fail-closed: one bad record, a short
// embed response or one rejected object fails the Activity, because a silently
// missing message is one the owner would never find.
func (a PublishContextSearchActivities) publishStream(ctx context.Context, spec PublishContextSearchSpec, plan ContextSearchPlan) (ContextSearchPublicationOutcome, error) {
	outcome := ContextSearchPublicationOutcome{
		Collection:           a.Collection,
		ObjectIDConstruction: contextsearch.ObjectIDConstruction,
		DedupKeyConstruction: contextsearch.DedupKeyConstruction,
	}
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
		for i := range page {
			if len(vectors[i]) == 0 {
				return fmt.Errorf("embedder returned an empty vector for record %s", page[i].Coordinates.RowID)
			}
			page[i].Vector = vectors[i]
		}
		result, err := a.Target.PublishObjects(ctx, page)
		if err != nil {
			return fmt.Errorf("publish context search objects to %s: %w", a.Collection, err)
		}
		if result.Written != len(page) {
			return fmt.Errorf("context search target wrote %d of %d objects", result.Written, len(page))
		}
		if len(result.ObjectIDs) > 0 {
			if outcome.FirstObjectID == "" {
				outcome.FirstObjectID = result.ObjectIDs[0]
			}
			outcome.LastObjectID = result.ObjectIDs[len(result.ObjectIDs)-1]
		}
		outcome.Published += result.Written
		outcome.VectorsPublished += result.Written
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
		object, err := contextSearchObjectFrom(record, plan, spec)
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
	if outcome.Published == 0 {
		return outcome, errors.New("context search publication produced no objects; a verified normalized generation with no publishable records is a defect, not an empty success")
	}
	return outcome, nil
}

// contextSearchObjectFrom converts one record into a validated search object.
//
// The dedup discriminator is the record's ordinal within its source version:
// the normalizer is 1:1 and order-preserving over the raw records
// (normalize.GenericMessageNormalizer), so re-running the same source version
// yields the same ordinals and therefore the same object ids. The row id is
// never used: it is re-minted on every re-extraction.
func contextSearchObjectFrom(record ContextSearchRecord, plan ContextSearchPlan, spec PublishContextSearchSpec) (contextsearch.Object, error) {
	coordinates := plan.Coordinates
	coordinates.RowID = record.RowID
	dedupKey, err := contextsearch.DedupKey(coordinates.SourceVersionID, fmt.Sprintf("record_ordinal:%d", record.Ordinal), nil)
	if err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	provenance := plan.Provenance
	provenance.Attempt = spec.Attempt
	provenance.RequestID = spec.RequestID
	provenance.ExtractionAttemptRef = string(spec.ExtractionAttemptRef)

	people := contextSearchPeople(record.Participants)
	object := contextsearch.Object{
		DedupKey:    dedupKey,
		Coordinates: coordinates,
		Provenance:  provenance,
		Temporal: contextsearch.Temporal{
			OccurredAt:           record.OccurredAt,
			OccurredAtRaw:        record.OccurredAtRaw,
			KnowledgeTime:        record.KnowledgeTime,
			DisclosureTier:       contextsearch.DeriveDisclosureTier(record.OccurredAt, record.KnowledgeTime),
			DisclosureTierBasis:  contextsearch.DisclosureTierBasis,
			TimestampCertainty:   record.TimestampCertainty,
			TimestampGranularity: record.TimestampGranularity,
		},
		People:          people,
		RecordKind:      record.RecordType,
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
