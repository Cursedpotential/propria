// Package stagegraph locks the atomic stage graph of the future Temporal
// ProfferWorkflow described in
// docs/reviews/2026-08-25-schema-audit/SBV-GO-TEMPORAL-RUNTIME-BOUNDARY.html.
//
// It has no Temporal dependency. It exists so the exact stage set, their
// single-responsibility boundaries, and their dependency edges can be
// reviewed and tested before any Temporal SDK code is written.
package stagegraph

// StageID names one atomic Activity in the ProfferWorkflow, matching
// the canon activity names in the boundary document section 2.
type StageID string

const (
	RegisterSource                 StageID = "register_source_activity"
	RetainOriginal                 StageID = "retain_original_activity"
	AssessSourceRepair             StageID = "assess_source_repair_activity"
	ResolveSourceRepair            StageID = "resolve_source_repair_activity"
	CaptureFilesystemMetadata      StageID = "capture_filesystem_metadata_activity"
	FingerprintSource              StageID = "fingerprint_source_activity"
	InventoryContainer             StageID = "inventory_container_activity"
	ExtractEmbeddedMetadata        StageID = "extract_embedded_metadata_activity"
	SelectParser                   StageID = "select_parser_activity"
	ExecuteParser                  StageID = "execute_parser_activity"
	PersistRawGeneration           StageID = "persist_raw_generation_activity"
	FingerprintRawRecords          StageID = "fingerprint_raw_records_activity"
	FingerprintRawGeneration       StageID = "fingerprint_raw_generation_activity"
	ReconcileRecordAccounting      StageID = "reconcile_record_accounting_activity"
	ReconcileByteCoverage          StageID = "reconcile_byte_coverage_activity"
	VerifyRawCoverageAgainstSource StageID = "verify_raw_coverage_against_source_activity"
	NormalizeGeneration            StageID = "normalize_generation_activity"
	PersistNormalizedGeneration    StageID = "persist_normalized_generation_activity"
	PersistLineage                 StageID = "persist_lineage_activity"
	ValidateRawLineage             StageID = "validate_raw_lineage_activity"
	HashNormalizedRecords          StageID = "hash_normalized_records_activity"
	HashNormalizedGeneration       StageID = "hash_normalized_generation_activity"
	VerifyNormalizedGeneration     StageID = "verify_normalized_generation_activity"
	PublishPreview                 StageID = "publish_preview_activity"
	SealGeneration                 StageID = "seal_generation_activity"
	PublishGeneration              StageID = "publish_generation_activity"
)

// Responsibility is a single-bit tag naming the one atomic side-effect a
// stage owns. A Descriptor must carry exactly one bit: the boundary document
// requires "one atomic responsibility, one side-effect boundary" per Activity.
type Responsibility uint32

const (
	RespRegisterIdentity Responsibility = 1 << iota
	RespRetain
	RespCaptureMetadata
	RespComputeHash
	RespInventory
	RespExtractMetadata
	RespSelect
	RespParse
	RespPersist
	RespReconcile
	RespVerify
	RespNormalize
	RespValidate
	RespSeal
	RespPublish
	RespAssessRepair
	RespResolveRepair
	RespProjectPreview
	RespChunk
	RespDerive
	// RespLocate is a read-only lookup that names another existing object; it
	// writes nothing (repair.find_other_version).
	// Byline: Claude Code · Opus 5.5 · 2026-09-25
	RespLocate
	// RespPublishSearch writes to the pre-approval search surface only. It is
	// deliberately a distinct bit from RespPublish: RespPublish is the
	// post-approval canonical publication, while this is the searchable
	// projection the owner validates against beforehand. Collapsing them would
	// let a search write masquerade as a canonical commit.
	// Byline: Claude Code · Opus 5 · 2026-09-26
	RespPublishSearch
)

// Descriptor is the static, dependency-free description of one stage: its
// single responsibility, the compact result it hands downstream, and which
// other stages must complete before it may start.
type Descriptor struct {
	ID             StageID
	Responsibility Responsibility
	Result         string
	DependsOn      []StageID
}

// ChunkDocument is the canon Activity name for a versioned non-messaging
// context chunk generation. It is optional because messaging imports retain
// their ordered normalized-message path. On the D-158 route the Temporal
// workflow schedules it after VerifyNormalizedGeneration and before
// PublishPreview, and its output is a sealed immutable chunk-generation Ref.
const ChunkDocument StageID = "chunk_document_activity"

// DeriveSMSThreads is the canon Activity name for streaming one oversized
// source that no in-place extractor can read and publishing memory-safe
// structured text into the configured derived vault directory (owner rulings
// 2026-09-20: "have sbv extract it and split out media and create structured
// text ... then use duckdb to extract the text" 18:47, and at 23:51 the
// derived-root ruling that superseded "next to original").
//
// It is its own Activity, never a widening of execute_parser: it produces no
// parser bundle, no raw generation, and no custody hash. Its single
// side-effect is publishing derived objects to the source's own object store,
// and its result is a reference to the derived manifest. The derived chunks
// then re-enter the platform as ordinary `ndjson` sources, each through its
// own Proffer run.
//
// Byline: Claude Code · Opus 5 · 2026-09-20
//
// The name is derive_sms_threads_activity: origin/main shipped that Activity
// first (commit 2817d82) and a second, identically-scoped
// derive_structured_text_activity was built on this branch before the
// collision was found. One Activity survives; this is its name.
const DeriveSMSThreads StageID = "derive_sms_threads_activity"

// PublishContextSearch is the canon Activity name for the Weaviate-first step
// of the owner's ruled pipeline order (2026-09-18 20:06-20:07, restated
// 2026-09-26): extraction output is made SEARCHABLE before anything is
// committed to the canonical PostgreSQL tables, so the owner can search, read,
// validate, annotate, verify the extraction and repair the file first.
//
// Weaviate is the pre-approval search surface, so this stage asserts nothing
// about accuracy and requires no approval to run. It is scheduled after
// VerifyNormalizedGeneration -- the records it makes searchable must exist and
// have passed extraction verification -- and before SealGeneration and
// PublishGeneration, which are the post-approval canonical commit.
//
// Its single side-effect is writing search objects to the Weaviate context
// collection. It never writes canonical PostgreSQL rows, never creates
// evidence or custody state, and deliberately applies NO horizon filter: it
// carries occurred_at, knowledge_time and disclosure_tier so a query-time
// analysis agent can apply one. Filtering here would make it a hindsight
// reader and trip engine/contextreview's tripwire.
//
// It is an OptionalStage rather than a member of Stages because promoting it
// to a universal ancestor of PublishGeneration would make a Weaviate outage
// block every run from completing. That coupling is an owner decision, not a
// default.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
const PublishContextSearch StageID = "publish_context_search_activity"

// OptionalStages describes version-gated stages that are real members of a
// specific route but not universal ancestors of PublishGeneration. Keeping
// these separate preserves the base graph's strong "every listed stage runs"
// invariant while giving conditional Temporal branches a reviewable
// dependency contract.
var OptionalStages = []Descriptor{
	{
		ID:             PublishContextSearch,
		Responsibility: RespPublishSearch,
		Result:         "context search publication receipt reference",
		// The records it publishes must exist and have passed extraction
		// verification. It deliberately does NOT depend on SealGeneration or
		// PublishGeneration: the whole point is that it runs BEFORE the
		// canonical commit.
		DependsOn: []StageID{VerifyNormalizedGeneration},
	},
	{
		ID:             ChunkDocument,
		Responsibility: RespChunk,
		Result:         "sealed context chunk generation reference",
		DependsOn:      []StageID{VerifyNormalizedGeneration},
	},
	{
		ID:             DeriveSMSThreads,
		Responsibility: RespDerive,
		Result:         "derived structured-text manifest reference",
		// It reads the retained original's own acquisition locator, so the
		// retained object must exist. It deliberately does NOT depend on
		// parser selection: it replaces extraction for this route rather
		// than following it.
		DependsOn: []StageID{RetainOriginal},
	},
}
