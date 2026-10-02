// Package contextsearch owns the pre-approval search object: the shape the
// engine publishes to Weaviate after extraction and BEFORE anything is
// committed to the canonical PostgreSQL tables.
//
// Why this package exists (owner rulings 2026-09-18 20:06-20:07 "IT ALL GOES
// TO WEAVIATE FIRST", restated 2026-09-26): the ruled pipeline order is
//
//	extract -> everything searchable in Weaviate -> owner searches, reads,
//	validates, adds metadata and context, verifies the extraction, repairs
//	the file -> THEN commit to the canonical PostgreSQL tables.
//
// Weaviate is therefore the pre-approval SEARCH surface, not a commitment:
// publishing here asserts nothing about accuracy and needs no approval.
//
// Messages and calls (owner 2026-10-02): a run that carries the conversation-chunk marker no longer publishes them
// here one object each. Postgres holds every message and call; Weaviate holds conversation chunks and one entry per
// call-log file (ProfferChunks20261002, written after the commit by server/context_chunks). The routing below still
// serves histories recorded before that marker, and the objects already published.
//
// Where it publishes (owner answers OD-06, 2026-10-01 07:17, and 2026-10-02):
// messages and calls with people go to the existing MsgEvents20260918, AI chats
// to the existing AiChatEvents20260918, documents to DocEvents20261001. No
// parallel store; Probata's objects are told apart from the Case Bible's by
// origin_system and by run id (ingest_run_id). These collections are searched
// by text and by their text_nim vector, so an object CARRIES the body. The 09-18 ruling superseded D-149 item 8's
// "no text in Weaviate" (spec 2026-09-27 §4).
//
// This package is pure compute: it validates and derives identifiers. It
// performs no I/O. The Weaviate HTTP boundary is engine/weaviate, the embedder
// is engine/embedding, and the Activity that drives them is
// activities/publish_context_search.go.
//
// It deliberately carries the temporal columns and performs NO horizon
// filtering. Carrying occurred_at, knowledge_time and disclosure_tier lets a
// query-time analysis agent apply a horizon; applying one in the write path
// would make this file a hindsight reader and trip engine/contextreview's
// tripwire.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (OD-06: MsgEvents20260918 mapping,
// body carried, uuid5 ids in the collection's own namespace)
// Byline: Claude Code · Opus 5.5 · 2026-10-02 (owner_participant tier from
// engine/disclosure; AI-chat and document record kinds)
package contextsearch

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CollectionNamespace is the uuid5 namespace every writer of
// MsgEvents20260918 already uses (Consignatio comm_timeline_mvp elt_run.py,
// publish_bundle.py, embed_weaviate.py: NS = 6f1d3a52-...). Using the same
// namespace keeps one id construction per collection. Probata dedup keys carry
// their own prefix, so they can never equal a Case Bible dedup key and never
// overwrite a Case Bible object.
var CollectionNamespace = uuid.MustParse("6f1d3a52-1c7e-4b8e-9a51-2d0e3c9b7a18")

// ObjectIDConstruction names the EXACT object-id construction: a name-based
// UUID v5 over the dedup key in CollectionNamespace. A bare version tag once let
// two different-but-valid chains collide under one label, so the namespace is
// part of the name.
const ObjectIDConstruction = "uuid5(ns=6f1d3a52-1c7e-4b8e-9a51-2d0e3c9b7a18, dedup_key)"

// DedupKeyConstruction names the exact dedup-key construction in DedupKey.
const DedupKeyConstruction = "probata-ctxsearch-dedup-v1: 'probata:ctxsearch:nr:v1|<source_version_id>|<srk|sch>|<discriminator>'"

// OriginSystem is the value of the origin_system property on every object
// this engine writes. It is how a reader tells Probata's objects apart from
// the Case Bible's in the shared collection.
const OriginSystem = "probata"

const dedupKeyPrefix = "probata:ctxsearch:nr:v1"

// Disclosure tiers, closed to exactly the values
// working.normalized_record.disclosure_tier's CHECK constraint allows.
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

// The disclosure tier itself is decided by engine/disclosure, the one rule
// for the whole application (owner 2026-10-02: contemporaneous when the owner
// was a participant, discovered when he was not). This package only checks
// that a tier is in the closed set and that it names its basis.

// Record kinds and the collections they route to (owner 2026-10-02): messages
// and calls with people -> MsgEvents20260918 (told apart by record_kind, owner
// 2026-09-24 09:30); conversations with AI -> AiChatEvents20260918; documents
// -> DocEvents20261001. Collection names are configuration (profferworker);
// any other record kind fails closed.
const (
	RecordKindMessage  = "message"
	RecordKindCall     = "call"
	RecordKindAIChat   = "ai_chat"
	RecordKindDocument = "document"
)

// aiChatFormats are the source formats of conversations with AI: the
// engine's own format ids (handler selection, structured ELT templates) and
// the Case Bible writer's (embed_weaviate.py AI_FORMATS), so one source is
// routed the same way whichever lane imported it.
var aiChatFormats = map[string]bool{
	"chatgpt_official_json": true, "chatgpt_json_array": true, "chatgpt_conversations_json": true,
	"claude_conversations_json": true, "gemini_activity_json": true, "ai_markdown_transcript": true,
	"ai_generic_json": true, "ai_conversations_json": true, "ai_chat_file": true,
}

// IsAIChatFormat reports whether a source format is a conversation with AI.
func IsAIChatFormat(format string) bool {
	return aiChatFormats[strings.ToLower(strings.TrimSpace(format))]
}

// ValidRecordKind reports whether a kind has an owner-approved collection.
func ValidRecordKind(kind string) bool {
	switch kind {
	case RecordKindMessage, RecordKindCall, RecordKindAIChat, RecordKindDocument:
		return true
	default:
		return false
	}
}

// Coordinates are the PostgreSQL coordinates needed to get back to the exact
// row this object stands for. All are required.
type Coordinates struct {
	// Schema and Table name the relation holding the row
	// (context / normalized_record_identity).
	Schema string
	Table  string
	// RowID is the row's own primary key.
	RowID uuid.UUID
	// SourceVersionID ties the row to the registered source version, which
	// is the custody anchor (owner option A, 2026-10-01 07:42: records carry
	// source_version_id, artifact_id is nullable). NormalizedGenerationID
	// locates the exact extraction output. OriginalObjectID is the source
	// version's registered original object; optional, and deliberately NOT
	// named artifact_id, which means an evidence artifact in
	// working.normalized_record.
	SourceVersionID        uuid.UUID
	NormalizedGenerationID uuid.UUID
	OriginalObjectID       uuid.UUID
	// MatterID is the matter the source version was registered under.
	MatterID uuid.UUID
}

func (c Coordinates) validate() error {
	if strings.TrimSpace(c.Schema) == "" || strings.TrimSpace(c.Table) == "" {
		return errors.New("context search coordinates require a PostgreSQL schema and table")
	}
	for name, id := range map[string]uuid.UUID{
		"row id": c.RowID, "source version id": c.SourceVersionID,
		"normalized generation id": c.NormalizedGenerationID,
	} {
		if id == uuid.Nil {
			return fmt.Errorf("context search coordinates require a %s", name)
		}
	}
	return nil
}

// Provenance is the provenance every object carries: which retained source
// object the content came from, which parser and normalizer produced it, which
// run and attempt wrote it.
type Provenance struct {
	// SourceObjectSHA256 is the digest of the registered original object.
	// Exactly 32 bytes.
	SourceObjectSHA256 []byte
	// SourceFormat is the declared format of the source version.
	SourceFormat string
	// ParserID/ParserVersion are the handler that produced the raw generation
	// (for example sbv_smsbackuprestore_xml 1.4.0).
	ParserID      string
	ParserVersion string
	// NormalizerID/NormalizerVersion produced the normalized generation.
	NormalizerID      string
	NormalizerVersion string
	// RequestID is the Proffer run id; Attempt is the Temporal attempt.
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
		"parser id": p.ParserID, "parser version": p.ParserVersion,
		"normalizer id": p.NormalizerID, "normalizer version": p.NormalizerVersion,
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

// Temporal carries the temporal values a query-time analysis agent depends on.
// Carried, never filtered on.
type Temporal struct {
	// OccurredAt is when the thing happened as-lived; nil when it could not
	// be recovered (published as absent, never as a zero time).
	OccurredAt *time.Time
	// OccurredAtRaw is the timestamp text exactly as the source wrote it.
	OccurredAtRaw string
	// KnowledgeTime is when the platform could know it
	// (normalized source_available_from).
	KnowledgeTime time.Time
	// DisclosureTier is the parse-time tier, closed to the CHECK set.
	DisclosureTier string
	// DisclosureTierBasis says where the tier came from; required.
	DisclosureTierBasis string
	// TimestampCertainty and TimestampGranularity are carried verbatim from
	// the normalized record.
	TimestampCertainty   string
	TimestampGranularity string
}

func (t Temporal) validate() error {
	if t.KnowledgeTime.IsZero() {
		return errors.New("context search object requires a knowledge time")
	}
	if !validDisclosureTier(t.DisclosureTier) {
		return fmt.Errorf("context search disclosure tier %q is outside the closed set (%s, %s, %s)",
			t.DisclosureTier, TierContemporaneous, TierHindsight, TierDiscovered)
	}
	if strings.TrimSpace(t.DisclosureTierBasis) == "" {
		return errors.New("context search disclosure tier requires its basis (stored or derived)")
	}
	if t.OccurredAt != nil && t.OccurredAt.IsZero() {
		return errors.New("context search occurred-at must be absent rather than a zero time")
	}
	return nil
}

// People are the participants of one record, by role.
type People struct {
	Sender       string
	Recipients   []string
	Participants []string
	// ContactNames are display names the source gave, when it gave any.
	ContactNames []string
}

// Object is one searchable pre-approval object.
type Object struct {
	// DedupKey decides identity; the object id is derived from it.
	DedupKey    string
	Coordinates Coordinates
	Provenance  Provenance
	Temporal    Temporal
	People      People

	// RecordKind is message, call, ai_chat or document (record_kind).
	RecordKind string
	// Direction is incoming/outgoing when the source says.
	Direction string
	// Body is the message text, empty for a call.
	Body string
	// SearchText is the text that was embedded: the body, or for a record
	// without one a short description built from its fields.
	SearchText string
	// ProvenanceClass is the normalized provenance class, verbatim.
	ProvenanceClass string
	// ContentSHA256 is the digest of the row's canonical bytes in PostgreSQL,
	// so a reader can prove the object still matches its row. 32 bytes.
	ContentSHA256 []byte
	// Vector is the text_nim search vector.
	Vector []float32
}

// Validate fails closed on anything that would make an object unusable for
// the owner's validation loop.
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
	if !ValidRecordKind(o.RecordKind) {
		return fmt.Errorf("context search record kind %q has no owner-approved collection (messages, calls, AI chats and documents only)", o.RecordKind)
	}
	if strings.TrimSpace(o.SearchText) == "" {
		return errors.New("context search object requires search text")
	}
	return nil
}

// DedupKey builds the dedup key for one normalized record. Identity is
// anchored to the SOURCE VERSION and a source-stable discriminator, never to
// the row id (minted fresh on every re-extraction), so re-running the same
// source upserts the same objects.
//
// Resolution order: sourceRecordKey when non-empty ("srk"), else the hex of a
// 32-byte source content digest ("sch"), else fail closed.
func DedupKey(sourceVersionID uuid.UUID, sourceRecordKey string, sourceContentSHA256 []byte) (string, error) {
	if sourceVersionID == uuid.Nil {
		return "", errors.New("context search dedup key requires a source version id")
	}
	discriminator := strings.TrimSpace(sourceRecordKey)
	kind := "srk"
	if discriminator == "" {
		if len(sourceContentSHA256) != sha256.Size {
			return "", errors.New("context search dedup key requires either a source record key or a 32-byte source content digest; the row id is not a permitted fallback because it is re-minted on every re-extraction")
		}
		discriminator = fmt.Sprintf("%x", sourceContentSHA256)
		kind = "sch"
	}
	return strings.Join([]string{dedupKeyPrefix, sourceVersionID.String(), kind, discriminator}, "|"), nil
}

// ObjectID derives this object's Weaviate id: uuid5(CollectionNamespace,
// DedupKey). Re-publishing the same record overwrites in place.
func (o Object) ObjectID() (uuid.UUID, error) {
	if err := o.Validate(); err != nil {
		return uuid.Nil, err
	}
	return DeriveObjectID(o.DedupKey)
}

// DeriveObjectID exposes the construction so a verifier can recompute an id
// from a dedup key alone.
func DeriveObjectID(dedupKey string) (uuid.UUID, error) {
	if strings.TrimSpace(dedupKey) == "" {
		return uuid.Nil, errors.New("context search object id requires a dedup key")
	}
	return uuid.NewSHA1(CollectionNamespace, []byte(dedupKey)), nil
}
