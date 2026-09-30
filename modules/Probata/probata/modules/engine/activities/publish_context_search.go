// Package activities: this file owns publish_context_search_activity only.
//
// It is the Weaviate-first step of the owner's ruled pipeline order
// (2026-09-18 20:06-20:07, restated 2026-09-26):
//
//	extract -> everything searchable in Weaviate -> owner searches, reads,
//	validates, adds metadata and context, verifies the extraction, repairs
//	the file -> THEN commit to the canonical PostgreSQL tables.
//
// Before this Activity existed the engine went extract -> PostgreSQL directly,
// which inverted that order and left the owner with nothing to search, so the
// validation step in the middle was impossible to perform at all.
//
// It follows the split this package already established (see
// normalized_pipeline.go's header, and entity_extraction.go for the
// extract/confirm/commit shape): the Activity does compute and validation over
// compact Store-provided streams, the Store owns its I/O, exactly one Activity
// owns each write, and everything fails closed. Records move by generation
// reference and are streamed -- a whole generation is never materialized in Go
// or in Temporal history.
//
// Two stores, because there are two side-effect boundaries:
//   - Source (PostgreSQL) resolves provenance, streams the publishable records
//     and writes the one durable receipt.
//   - Target (Weaviate) owns the search-object write.
//
// This Activity writes NO canonical PostgreSQL rows. Its only PostgreSQL write
// is its own append-only receipt in context.activity_receipt, which is what
// every Activity in this package owes the workflow.
//
// It carries occurred_at, knowledge_time and disclosure_tier and applies NO
// horizon filter. Carrying the dates lets a query-time analysis agent apply a
// horizon; applying one here would make this file a hindsight reader and trip
// engine/contextreview's tripwire.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
package activities

import (
	"context"
	"errors"
	"fmt"
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

// publishContextSearchPageSize bounds one publish batch handed to the target.
// It matches the streaming discipline the rest of this package uses: the
// Activity holds at most one page of compact records at a time, never a whole
// generation.
const publishContextSearchPageSize = 500

// PublishContextSearchSpec is the compact, already-resolved input to
// publish_context_search_activity, built from proffer.StageRequest by
// publishContextSearchSpecFrom.
type PublishContextSearchSpec struct {
	RequestID string
	Attempt   int32
	// SourceVersionRef and NormalizedGenerationRef select exactly which
	// extraction output becomes searchable.
	SourceVersionRef        proffer.Ref
	NormalizedGenerationRef proffer.Ref
	// NormalizedVerificationRef is the verification receipt that proves
	// extraction was verified before anything was published for the owner to
	// read. Required: publishing unverified extraction into the surface he
	// validates against would defeat the point of the ordering.
	NormalizedVerificationRef proffer.Ref
	// ExtractionAttemptRef is the durable extraction-attempt reference carried
	// into every object's provenance.
	ExtractionAttemptRef proffer.Ref
}

func (s PublishContextSearchSpec) validate() error {
	if strings.TrimSpace(s.RequestID) == "" {
		return fmt.Errorf("%s requires a request id", stagegraph.PublishContextSearch)
	}
	for name, ref := range map[string]proffer.Ref{
		"source version":         s.SourceVersionRef,
		"normalized generation":  s.NormalizedGenerationRef,
		"normalized verification": s.NormalizedVerificationRef,
		"extraction attempt":     s.ExtractionAttemptRef,
	} {
		if strings.TrimSpace(string(ref)) == "" {
			return fmt.Errorf("%s requires a %s reference", stagegraph.PublishContextSearch, name)
		}
	}
	if s.Attempt < 1 {
		return fmt.Errorf("%s attempt must be positive", stagegraph.PublishContextSearch)
	}
	return nil
}

// ContextSearchRecord is one compact publishable row. It deliberately carries
// NO message body: the Source store computes ContentSHA256 in SQL so the text
// never leaves PostgreSQL at all (D-149 item 8, D-158).
type ContextSearchRecord struct {
	RowID                  uuid.UUID
	ArtifactID             uuid.UUID
	SourceVersionID        uuid.UUID
	NormalizedGenerationID uuid.UUID

	RecordType     string
	Source         string
	ConversationID string

	// The three temporal columns, verbatim. OccurredAt is genuinely nullable.
	OccurredAt     *time.Time
	KnowledgeTime  time.Time
	DisclosureTier string

	// SourceRecordKey and SourceContentSHA256 are the dedup-key inputs, in
	// that precedence. See contextsearch.DedupKey.
	SourceRecordKey     string
	SourceContentSHA256 []byte
	// ContentSHA256 is the digest of the normalized content, computed in SQL.
	ContentSHA256 []byte
	// MemberIDs are constituent row ids when the row stands for a composed
	// unit; empty for a single record.
	MemberIDs []uuid.UUID
	// Vector is the search vector when the Source store has an embedding
	// provider configured, and empty otherwise. This is the Phase 2 seam: an
	// embedder is I/O and therefore belongs in the store, so wiring one in
	// later requires no change to this Activity. A collection created with
	// vectorizer "none" accepts an object with or without a vector, and an
	// object's id does not depend on its vector, so a vector can be added
	// later without duplicating anything.
	Vector []float32
}

// ContextSearchRecordReader streams publishable rows so a generation is never
// materialized whole. Next returns io.EOF when the stream is exhausted, like
// every other reader in this package.
type ContextSearchRecordReader interface {
	Next(context.Context) (ContextSearchRecord, error)
	Close() error
}

// ContextSearchPlan is what the Source store resolves in one round trip:
// the provenance shared by every object in this generation, the relation the
// rows live in, and the stream itself.
type ContextSearchPlan struct {
	Provenance contextsearch.Provenance
	// Schema and Table name the relation the rows came from, carried into every
	// object's coordinates so a reader can get back to the row.
	Schema string
	Table  string
	Reader ContextSearchRecordReader
}

// ContextSearchSourceStore is the PostgreSQL boundary. OpenContextSearchRecords
// resolves provenance and opens the stream; PersistContextSearchPublication is
// the only write, and must be retry-safe through context.activity_execution and
// context.activity_receipt exactly like every other repository in this package,
// so a repeated idempotency coordinate returns the existing durable outcome
// rather than writing a second receipt.
type ContextSearchSourceStore interface {
	OpenContextSearchRecords(context.Context, PublishContextSearchSpec) (ContextSearchPlan, error)
	PersistContextSearchPublication(context.Context, PublishContextSearchSpec, ContextSearchPublicationOutcome) (proffer.Ref, proffer.Ref, error)
}

// ContextSearchTarget is the Weaviate boundary. engine/weaviate.Store satisfies
// it. EnsureCollection must never mutate an existing collection, and
// PublishObjects must be idempotent under a repeated object id.
type ContextSearchTarget interface {
	EnsureCollection(context.Context) (bool, error)
	PublishObjects(context.Context, []contextsearch.Object) (contextSearchPublishResult, error)
}

// contextSearchPublishResult is the minimal result this Activity needs from a
// target write. It is declared here, in the consumer, so the Activity does not
// depend on the Weaviate package's concrete types; engine/weaviate's
// PublishOutcome satisfies it structurally through the adapter in
// register.go.
type contextSearchPublishResult = ContextSearchPublishResult

// ContextSearchPublishResult is one target write's outcome.
type ContextSearchPublishResult struct {
	Requested int
	Written   int
	ObjectIDs []string
}

// ContextSearchPublicationOutcome is the durable result of one publish pass.
type ContextSearchPublicationOutcome struct {
	// Collection is the Weaviate collection written to.
	Collection string
	// Published is how many objects were written; ObjectIDConstruction and
	// DedupKeyConstruction name the exact id constructions used, so a future
	// reader can tell which construction produced these ids rather than
	// guessing from a bare version tag.
	Published              int
	ObjectIDConstruction   string
	DedupKeyConstruction   string
	// FirstObjectID and LastObjectID bound the written set without carrying
	// every id into the receipt.
	FirstObjectID string
	LastObjectID  string
	// VectorsPublished counts objects that carried a search vector. Zero means
	// the objects are structurally searchable but not yet semantically
	// searchable -- the Phase 2 embedder seam.
	VectorsPublished int
}

// PublishContextSearchActivities implements publish_context_search_activity.
// Source and Target are the two side-effect boundaries; Attempt is injectable
// for tests and bound to the Temporal attempt by the constructor in
// register.go, exactly like every other Activities struct in this package.
type PublishContextSearchActivities struct {
	Source ContextSearchSourceStore
	Target ContextSearchTarget
	// Collection is recorded in the receipt so a reader knows where the
	// objects went. The Target owns the actual collection name; this must
	// match it and is validated as non-empty.
	Collection string
	Attempt    Attempt
}

func (a PublishContextSearchActivities) validate() error {
	if a.Source == nil {
		return errors.New("publish context search activities: source store is required")
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
	attempt := a.Attempt(ctx)
	if attempt < 1 {
		return 1
	}
	return attempt
}

// publishContextSearchSpecFrom resolves the compact spec from the canonical
// StageRequest reference set. Every reference is required: a missing one means
// the workflow reached this stage without the upstream gate, and inventing a
// default would publish something the owner could not trace.
func publishContextSearchSpecFrom(req proffer.StageRequest, attempt int32) (PublishContextSearchSpec, error) {
	spec := PublishContextSearchSpec{
		RequestID:        req.RequestID,
		Attempt:          attempt,
		SourceVersionRef: req.SourceVersionRef,
	}
	for name, target := range map[string]*proffer.Ref{
		"normalized_generation":   &spec.NormalizedGenerationRef,
		"normalized_verification": &spec.NormalizedVerificationRef,
		"extraction_attempt":      &spec.ExtractionAttemptRef,
	} {
		ref, ok := req.Refs[name]
		if !ok || strings.TrimSpace(string(ref)) == "" {
			return PublishContextSearchSpec{}, fmt.Errorf("%s requires the %q reference", stagegraph.PublishContextSearch, name)
		}
		*target = ref
	}
	if err := spec.validate(); err != nil {
		return PublishContextSearchSpec{}, err
	}
	return spec, nil
}

// PublishContextSearch makes one verified extraction output searchable in
// Weaviate, before any canonical PostgreSQL commit.
//
// It is idempotent without needing a PostgreSQL idempotency coordinate to be
// safe: every object's id is derived from its dedup key, and the target write
// replaces an existing id rather than adding a row, so a Temporal retry
// re-publishes the same ids and leaves the collection count unchanged. The
// durable receipt is still written, because the workflow requires every
// terminal status to carry one.
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
	if strings.TrimSpace(plan.Schema) == "" || strings.TrimSpace(plan.Table) == "" {
		return proffer.StageResult{}, errors.New("context search plan requires the PostgreSQL schema and table the rows came from")
	}
	if err := plan.Provenance.Validate(); err != nil {
		return proffer.StageResult{}, fmt.Errorf("context search provenance: %w", err)
	}

	if _, err := a.Target.EnsureCollection(ctx); err != nil {
		return proffer.StageResult{}, fmt.Errorf("ensure context search collection: %w", err)
	}

	outcome, err := a.publishStream(ctx, spec, plan)
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

// publishStream drains the reader a page at a time, converting each compact
// record into a validated search object and handing whole pages to the target.
// Fail-closed: one bad record or one rejected page fails the Activity rather
// than leaving a partially searchable generation, because a silently missing
// message is one the owner would never find.
func (a PublishContextSearchActivities) publishStream(
	ctx context.Context,
	spec PublishContextSearchSpec,
	plan ContextSearchPlan,
) (ContextSearchPublicationOutcome, error) {
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
		result, err := a.Target.PublishObjects(ctx, page)
		if err != nil {
			return fmt.Errorf("publish context search objects: %w", err)
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
		page = page[:0]
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return outcome, err
		}
		record, err := plan.Reader.Next(ctx)
		if err != nil {
			if isStreamEnd(err) {
				break
			}
			return outcome, fmt.Errorf("read context search record: %w", err)
		}
		object, err := contextSearchObjectFrom(record, plan, spec)
		if err != nil {
			return outcome, err
		}
		if len(object.Vector) > 0 {
			outcome.VectorsPublished++
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

// contextSearchObjectFrom converts one compact record into a validated search
// object, deriving the dedup key from the record's source-anchored identity so
// a re-extraction of the same source lands on the same object.
func contextSearchObjectFrom(
	record ContextSearchRecord,
	plan ContextSearchPlan,
	spec PublishContextSearchSpec,
) (contextsearch.Object, error) {
	dedupKey, err := contextsearch.DedupKey(record.SourceVersionID, record.SourceRecordKey, record.SourceContentSHA256)
	if err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	object := contextsearch.Object{
		DedupKey: dedupKey,
		Coordinates: contextsearch.Coordinates{
			Schema:                 plan.Schema,
			Table:                  plan.Table,
			RowID:                  record.RowID,
			SourceVersionID:        record.SourceVersionID,
			ArtifactID:             record.ArtifactID,
			NormalizedGenerationID: record.NormalizedGenerationID,
		},
		Provenance: plan.Provenance,
		Temporal: contextsearch.Temporal{
			OccurredAt:     record.OccurredAt,
			KnowledgeTime:  record.KnowledgeTime,
			DisclosureTier: record.DisclosureTier,
		},
		MemberIDs:      record.MemberIDs,
		RecordType:     record.RecordType,
		Source:         record.Source,
		ConversationID: record.ConversationID,
		ContentSHA256:  record.ContentSHA256,
		Vector:         record.Vector,
	}
	// The spec's attempt is the authority for the attempt recorded on every
	// object in this pass, so a retry cannot leave objects claiming an older
	// attempt than the receipt that certifies them.
	object.Provenance.Attempt = spec.Attempt
	object.Provenance.RequestID = spec.RequestID
	object.Provenance.ExtractionAttemptRef = string(spec.ExtractionAttemptRef)
	if err := object.Validate(); err != nil {
		return contextsearch.Object{}, fmt.Errorf("context search record %s: %w", record.RowID, err)
	}
	return object, nil
}
