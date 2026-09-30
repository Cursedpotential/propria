// Package contextsearch owns the pre-approval search object: the shape the
// engine publishes to the Weaviate context collection after extraction and
// BEFORE anything is committed to the canonical PostgreSQL tables.
//
// Why this package exists (owner rulings 2026-09-18 20:06-20:07, restated
// 2026-09-26): the ruled pipeline order is
//
//	extract -> everything searchable in Weaviate -> owner searches, reads,
//	validates, adds metadata and context, verifies the extraction, repairs
//	the file -> THEN commit to the canonical PostgreSQL tables.
//
// Weaviate is therefore the pre-approval SEARCH surface, not a commitment:
// publishing here asserts nothing about accuracy and needs no approval.
// Canonical PostgreSQL remains the post-approval commit.
//
// D-149 item 8 fixes the payload: an object carries the vector, the
// PostgreSQL coordinates and the member ids -- NOT the text. Messaging keeps
// its canonical text in PostgreSQL (D-158, storage splits by source type), so
// a message body is never duplicated here. content_sha256 travels instead, so
// a reader can prove an object still matches its PostgreSQL row without the
// body ever leaving PostgreSQL.
//
// This package is pure compute: it validates and it derives identifiers. It
// performs no I/O. The Weaviate HTTP boundary is engine/weaviate; the
// Activity that drives both is activities/publish_context_search.go.
//
// It deliberately carries the temporal columns and deliberately performs NO
// horizon filtering. Carrying occurred_at, knowledge_time and
// disclosure_tier lets a query-time analysis agent apply a horizon; applying
// one in the write path would make this file a hindsight reader and trip
// engine/contextreview's tripwire, which is exactly the leak that test
// exists to catch.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
package contextsearch

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ObjectIDConstruction names the EXACT deterministic object-id construction
// below -- genesis label and fold formula, not just a version number. A bare
// version tag once let two different-but-valid chains collide under one
// label; every object records this string so a future reader can tell which
// construction produced its id.
const ObjectIDConstruction = "probata-ctxsearch-oid-v1"

// DedupKeyConstruction names the exact dedup-key construction in DedupKey.
const DedupKeyConstruction = "probata-ctxsearch-dedup-v1"

// dedupKeyPrefix and the 0x1F unit separator domain-separate the digest input
// so a dedup key can never be confused with any other hashed string in the
// platform.
const (
	dedupKeyPrefix = "probata:ctxsearch:nr:v1"
	unitSeparator  = "\x1f"
)

// Disclosure tiers, closed to exactly the values
// working.normalized_record.disclosure_tier's CHECK constraint allows. A value
// outside this set fails closed rather than reaching Weaviate, because a
// downstream horizon analysis keys off it.
const (
	TierContemporaneous = "contemporaneous"
	TierHindsight       = "hindsight"
	TierDiscovered      = "discovered"
)

func validDisclosureTier(tier string) bool {
	switch tier {
	case TierContemporaneous, TierHindsight, TierDiscovered:
		return true
	default:
		return false
	}
}

// Coordinates are the PostgreSQL coordinates needed to get back to the exact
// row this object stands for. Without every one of these an object is a
// dead end -- the owner could find it in search and not be able to open it --
// so all of them are required.
type Coordinates struct {
	// Schema and Table name the relation holding the row (for the messaging
	// route: working / normalized_record).
	Schema string
	Table  string
	// RowID is the row's own primary key.
	RowID uuid.UUID
	// SourceVersionID ties the row back to the registered source version that
	// produced it; ArtifactID and NormalizedGenerationID locate the exact
	// extraction output.
	SourceVersionID        uuid.UUID
	ArtifactID             uuid.UUID
	NormalizedGenerationID uuid.UUID
}

func (c Coordinates) validate() error {
	if strings.TrimSpace(c.Schema) == "" || strings.TrimSpace(c.Table) == "" {
		return errors.New("context search coordinates require a PostgreSQL schema and table")
	}
	for name, id := range map[string]uuid.UUID{
		"row id": c.RowID, "source version id": c.SourceVersionID,
		"artifact id": c.ArtifactID, "normalized generation id": c.NormalizedGenerationID,
	} {
		if id == uuid.Nil {
			return fmt.Errorf("context search coordinates require a %s", name)
		}
	}
	return nil
}

// Provenance is the full provenance every object carries: which retained
// source object the content came from, which ELT template and version read
// it, which extraction attempt produced it, and the dedup key that decides
// the object's identity.
type Provenance struct {
	// SourceObjectSHA256 is the digest of the retained source object -- the
	// bytes extraction actually read. Exactly 32 bytes.
	SourceObjectSHA256 []byte
	// TemplateID and TemplateVersion are the pinned DuckDB ELT template
	// identifiers (for example ndjson_v1) and the implementation version that
	// owns them.
	TemplateID      string
	TemplateVersion string
	// ParserID and ParserVersion are the handler that ran (for example
	// duckdb_structured_elt 1.0.0).
	ParserID      string
	ParserVersion string
	// RequestID and Attempt are the Temporal idempotency coordinate.
	RequestID string
	Attempt   int32
	// ExtractionAttemptRef is the durable extraction-attempt reference.
	ExtractionAttemptRef string
}

func (p Provenance) validate() error {
	if len(p.SourceObjectSHA256) != sha256.Size {
		return fmt.Errorf("context search provenance requires a %d-byte source object digest, got %d", sha256.Size, len(p.SourceObjectSHA256))
	}
	for name, value := range map[string]string{
		"template id": p.TemplateID, "template version": p.TemplateVersion,
		"parser id": p.ParserID, "parser version": p.ParserVersion,
		"request id": p.RequestID, "extraction attempt ref": p.ExtractionAttemptRef,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("context search provenance requires a %s", name)
		}
	}
	if p.Attempt < 1 {
		return errors.New("context search provenance attempt must be positive")
	}
	return nil
}

// Temporal carries the three temporal columns verbatim from
// working.normalized_record. A query-time analysis agent depends on them.
// This struct is carried, never filtered on.
type Temporal struct {
	// OccurredAt is when the thing happened as-lived. It is genuinely
	// nullable in PostgreSQL (a record whose timestamp could not be
	// recovered), so it is a pointer here and is published as null rather
	// than as a zero time, which would read as year 1 and silently corrupt
	// any ordering built on it.
	OccurredAt *time.Time
	// KnowledgeTime is when the platform learned it. NOT NULL in PostgreSQL.
	KnowledgeTime time.Time
	// DisclosureTier is the parse-time tier, closed to the CHECK set.
	DisclosureTier string
}

func (t Temporal) validate() error {
	if t.KnowledgeTime.IsZero() {
		return errors.New("context search object requires a knowledge time")
	}
	if !validDisclosureTier(t.DisclosureTier) {
		return fmt.Errorf("context search disclosure tier %q is outside the closed set (%s, %s, %s)",
			t.DisclosureTier, TierContemporaneous, TierHindsight, TierDiscovered)
	}
	if t.OccurredAt != nil && t.OccurredAt.IsZero() {
		return errors.New("context search occurred-at must be absent rather than a zero time")
	}
	return nil
}

// Object is one searchable pre-approval object. It carries no text body: see
// the package comment and D-149 item 8.
type Object struct {
	// DedupKey decides identity. ObjectID is derived from it, so two
	// publishes of the same logical record produce the same Weaviate object
	// rather than a duplicate.
	DedupKey    string
	Coordinates Coordinates
	Provenance  Provenance
	Temporal    Temporal
	// MemberIDs are the constituent row ids when this object stands for a
	// composed unit; empty for a single record.
	MemberIDs []uuid.UUID
	// RecordType, Source and ConversationID are search facets, not content.
	RecordType     string
	Source         string
	ConversationID string
	// ContentSHA256 lets a reader prove the object still matches its
	// PostgreSQL row without the body ever leaving PostgreSQL. Exactly 32
	// bytes.
	ContentSHA256 []byte
	// Vector is the search vector. It is optional at this layer: the
	// embedding provider is injected by the caller, and a collection created
	// with vectorizer "none" accepts an object without one. An object
	// published without a vector is still structurally searchable and can be
	// given a vector later without changing its id.
	Vector []float32
}

// Validate fails closed on anything that would make an object unusable for
// the owner's validation loop: a missing coordinate he could not open, a
// missing provenance field he could not audit, or a temporal value a
// downstream horizon could not read.
func (o Object) Validate() error {
	if strings.TrimSpace(o.DedupKey) == "" {
		return errors.New("context search object requires a dedup key")
	}
	if err := o.Coordinates.validate(); err != nil {
		return err
	}
	if err := o.Provenance.validate(); err != nil {
		return err
	}
	if err := o.Temporal.validate(); err != nil {
		return err
	}
	if len(o.ContentSHA256) != sha256.Size {
		return fmt.Errorf("context search object requires a %d-byte content digest, got %d", sha256.Size, len(o.ContentSHA256))
	}
	if strings.TrimSpace(o.RecordType) == "" || strings.TrimSpace(o.Source) == "" {
		return errors.New("context search object requires a record type and source")
	}
	for _, member := range o.MemberIDs {
		if member == uuid.Nil {
			return errors.New("context search object member ids must all be set")
		}
	}
	return nil
}

// DedupKey builds the dedup key for one normalized record. Identity is
// deliberately anchored to the SOURCE, not to the extraction attempt: the
// owner validates a message, not a run. Re-extracting the same source under a
// new generation must therefore land on the SAME object rather than a second
// copy of the same message.
//
// Resolution order, most stable first:
//
//  1. sourceRecordKey -- the source-native key
//     (working.normalized_record.source_record_key) when the source provides
//     one. Survives re-extraction and re-normalization.
//  2. hex(sourceContentSHA256) -- the digest of the source-native content
//     when there is no native key. Survives re-extraction of identical bytes.
//  3. fail closed. The row id is deliberately NOT a fallback: it is minted
//     fresh by uuidv7() on every re-extraction, so using it would silently
//     duplicate every message on the owner's second pass -- the exact failure
//     this key exists to prevent.
func DedupKey(sourceVersionID uuid.UUID, sourceRecordKey string, sourceContentSHA256 []byte) (string, error) {
	if sourceVersionID == uuid.Nil {
		return "", errors.New("context search dedup key requires a source version id")
	}
	discriminator := strings.TrimSpace(sourceRecordKey)
	kind := "srk"
	if discriminator == "" {
		if len(sourceContentSHA256) != sha256.Size {
			return "", errors.New("context search dedup key requires either a source record key or a 32-byte source content digest; the row id is not a permitted fallback because uuidv7() re-mints it on every re-extraction")
		}
		discriminator = hex.EncodeToString(sourceContentSHA256)
		kind = "sch"
	}
	return strings.Join([]string{dedupKeyPrefix, sourceVersionID.String(), kind, discriminator}, "|"), nil
}

// ObjectID derives this object's Weaviate id deterministically from its dedup
// key, so re-publishing an attempt overwrites in place instead of growing the
// collection. Nothing here is random.
//
// The construction, named by ObjectIDConstruction, is:
//
//	digest = SHA256( ObjectIDConstruction || 0x1F || DedupKey )
//	id[0:6]  = 48-bit big-endian unix milliseconds of the anchor time
//	id[6:16] = digest[0:10]
//	id[6]    = (id[6] & 0x0F) | 0x70   // version 7
//	id[8]    = (id[8] & 0x3F) | 0x80   // RFC 9562 variant
//
// The anchor time is the object's KnowledgeTime, which PostgreSQL sets once
// and never rewrites, so the timestamp field is durable rather than
// "now"-dependent.
//
// Why this shape rather than a name-based UUID v5: persisted identifiers in
// this platform are UUID v7 (uuidv7() is the schema default in 167 places)
// and are relied upon to sort by time. A v5 id would be deterministic but
// would sort randomly and would announce the wrong version. This construction
// is both fully deterministic AND correctly v7-shaped. Masking the version
// and variant nibbles spends 6 of the digest's bits, leaving 74 bits of
// digest entropy inside one source version's keyspace.
func (o Object) ObjectID() (uuid.UUID, error) {
	if err := o.Validate(); err != nil {
		return uuid.Nil, err
	}
	return DeriveObjectID(o.DedupKey, o.Temporal.KnowledgeTime)
}

// DeriveObjectID exposes the construction on its own so a verifier can
// recompute an id from a dedup key and anchor time without rebuilding a whole
// Object.
func DeriveObjectID(dedupKey string, anchor time.Time) (uuid.UUID, error) {
	if strings.TrimSpace(dedupKey) == "" {
		return uuid.Nil, errors.New("context search object id requires a dedup key")
	}
	if anchor.IsZero() {
		return uuid.Nil, errors.New("context search object id requires a non-zero anchor time")
	}
	milli := anchor.UTC().UnixMilli()
	if milli < 0 {
		return uuid.Nil, fmt.Errorf("context search object id anchor time %s precedes the unix epoch", anchor.UTC().Format(time.RFC3339))
	}
	digest := sha256.Sum256([]byte(ObjectIDConstruction + unitSeparator + dedupKey))

	var id uuid.UUID
	var stamp [8]byte
	binary.BigEndian.PutUint64(stamp[:], uint64(milli))
	copy(id[0:6], stamp[2:8])
	copy(id[6:16], digest[0:10])
	id[6] = (id[6] & 0x0f) | 0x70
	id[8] = (id[8] & 0x3f) | 0x80
	return id, nil
}
