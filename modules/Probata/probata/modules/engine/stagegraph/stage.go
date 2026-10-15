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
	// RespProposeContext plans the first-party context import and records its
	// digest; it writes no working.* row (the EXTRACT step, D04).
	// RespConfirmContext proves, after the owner's decision, that the plan is
	// unchanged (CONFIRM). RespCommitContext writes working.* first-party
	// context rows (COMMIT); the spine and the thread commits share it, as the
	// persist stages share RespPersist.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	RespProposeContext
	RespConfirmContext
	RespCommitContext
	// RespResolveParticipants resolves every identifier a generation states
	// against the registry once and records the resolution; it writes no row
	// outside its own receipt. Byline: Claude Code · Opus 5.5 · 2026-10-02
	RespResolveParticipants
	// RespRecordDecision records a preview decision in the same durable
	// decision record a human approval writes; it writes nothing else.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	RespRecordDecision
	// RespMatchOccurrences looks up which of a generation's messages another
	// source already committed and records that list; it writes no working row.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	RespMatchOccurrences
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

// AssessSourceIntegrity names the independent whole-stream byte assessment Activity.
// Inputs: retained source/version references. Outputs: assessment and receipt references.
// Effects: retained-byte reads and append-only receipts; choose for zero-content checks,
// never format validation or canonical eligibility. Byline: Codex, 2026-10-06.
const AssessSourceIntegrity StageID = "assess_source_integrity_activity"

// StageAICandidateBundle stages grounded proposals from a retained AI source for owner review.
// Inputs: exact source pin and candidate bundle reference/hash. Outputs: bounded staging receipt.
// Effects: pending working candidates only. Choose on the native source-only AI branch.
const StageAICandidateBundle StageID = "stage_ai_candidate_bundle_activity"

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
// The owner made it mandatory (OD-06, answered 2026-10-01 07:17: target
// MsgEvents20260918, Weaviate first on every run). The workflow schedules it on
// every new history behind the version marker
// proffer-weaviate-first-context-search-v1, so a Weaviate failure stops the run
// before the commit. It stays in OptionalStages, not Stages, only because
// histories recorded before that marker never ran it and must replay.
//
// Byline: Claude Code · Opus 5 · 2026-09-26
// Byline: Claude Code · Opus 5.5 · 2026-10-01 (mandatory, scheduled)
const PublishContextSearch StageID = "publish_context_search_activity"

// The first-party context import (D04, DF-02/DF-03/DF-04): a verified message
// generation becomes working.* spine rows and first-party context threads, on
// the owner's extract -> confirm -> commit order (2026-09-26) and after the
// Weaviate-first search stage. Propose runs before the preview and writes no
// working.* row; confirm and both commits run only after the owner's preview
// decision and before seal_generation, so a refused commit blocks the seal.
// They are optional for the same replay reason as PublishContextSearch, and a
// generation with no message record ends the chain at propose as
// not_applicable. Implementation: engine/activities/first_party_context.go.
//
// Byline: Claude Code · Opus 5.5 · 2026-10-01
//
// ResolveContextParticipants (2026-10-02) runs first: it resolves every
// identifier the generation states against registry.entity_alias_current
// once, and both publish_context_search_activity and the propose / confirm /
// commit stages read that one recorded resolution (ref
// "participant_resolution"), so the disclosure tier in Weaviate and in
// PostgreSQL is the same answer.
const (
	ResolveContextParticipants     StageID = "resolve_context_participants_activity"
	ProposeFirstPartyContext       StageID = "propose_first_party_context_activity"
	ConfirmFirstPartyContext       StageID = "confirm_first_party_context_activity"
	CommitFirstPartyMessages       StageID = "commit_first_party_messages_activity"
	CommitFirstPartyContextThreads StageID = "commit_first_party_context_threads_activity"
)

// RecordAutoApproval (owner 2026-10-02, "auto-approve clean runs") records the
// automatic approval of a run whose every computed check passed, in the same
// context.proffer_preview_decision record a human approval writes, with the
// actor "auto:clean-checks" and the passed checks as its reason. It is
// scheduled only when the run was started with auto-approval switched on (per
// batch, never globally) and only after the preview is published; a run with
// any check that did not pass never reaches it and waits for the owner.
// Byline: Claude Code · Opus 5.5 · 2026-10-02
const RecordAutoApproval StageID = "record_auto_approval_activity"

// CommitCallLog (owner 2026-10-02: "calls follow the same path as
// messages") writes the generation's committed call records into
// working.normalized_record (record_type 'call') and working.call_log, after
// the same preview decision (the owner's or clean_checks) and with the same
// recorded participant resolution and perspective as the message commit. A
// generation with no call record settles not_applicable.
// Byline: Claude Code · Opus 5.5 · 2026-10-02
const CommitCallLog StageID = "commit_call_log_activity"

// MatchMessageOccurrences (owner 2026-10-02: "both Facebook exports, deduped")
// finds the generation's messages another source version already committed,
// before the Weaviate-first stage, so neither search nor commit makes a second
// copy. Byline: Claude Code · Opus 5.5 · 2026-10-02
const MatchMessageOccurrences StageID = "match_message_occurrences_activity"

// AutoApprovalActor is the decided_by value of every automatic approval.
const AutoApprovalActor = "auto:clean-checks"

// OptionalStages describes version-gated stages that are real members of a
// specific route but not universal ancestors of PublishGeneration. Keeping
// these separate preserves the base graph's strong "every listed stage runs"
// invariant while giving conditional Temporal branches a reviewable
// dependency contract.
var OptionalStages = []Descriptor{
	{ID: StageAICandidateBundle, Responsibility: RespPersist, Result: "retained-source AI candidate staging receipt", DependsOn: []StageID{RetainOriginal, CaptureFilesystemMetadata, FingerprintSource, InventoryContainer, ExtractEmbeddedMetadata}},
	{ID: AssessSourceIntegrity, Responsibility: RespVerify, Result: "byte integrity assessment receipt reference", DependsOn: []StageID{RetainOriginal}},
	{
		ID:             PublishContextSearch,
		Responsibility: RespPublishSearch,
		Result:         "context search publication receipt reference",
		// The records it publishes must exist and have passed extraction
		// verification. It deliberately does NOT depend on SealGeneration or
		// PublishGeneration: the whole point is that it runs BEFORE the
		// canonical commit. It applies the run's one recorded participant
		// resolution (2026-10-02), so it follows that stage.
		DependsOn: []StageID{VerifyNormalizedGeneration, ResolveContextParticipants},
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
	{
		ID:             ResolveContextParticipants,
		Responsibility: RespResolveParticipants,
		Result:         "participant resolution receipt reference",
		DependsOn:      []StageID{VerifyNormalizedGeneration},
	},
	{
		ID:             ProposeFirstPartyContext,
		Responsibility: RespProposeContext,
		Result:         "first-party context proposal receipt reference",
		DependsOn:      []StageID{VerifyNormalizedGeneration, ResolveContextParticipants},
	},
	{
		ID:             ConfirmFirstPartyContext,
		Responsibility: RespConfirmContext,
		Result:         "first-party context confirmation receipt reference",
		// The owner's decision on the preview is the confirmation gate.
		DependsOn: []StageID{ProposeFirstPartyContext, PublishPreview},
	},
	{
		ID:             CommitFirstPartyMessages,
		Responsibility: RespCommitContext,
		Result:         "first-party message spine commit receipt reference",
		DependsOn:      []StageID{ConfirmFirstPartyContext},
	},
	{
		ID:             CommitFirstPartyContextThreads,
		Responsibility: RespCommitContext,
		Result:         "first-party context thread commit receipt reference",
		// Thread membership references working.message rows.
		DependsOn: []StageID{CommitFirstPartyMessages},
	},
	{
		ID:             RecordAutoApproval,
		Responsibility: RespRecordDecision,
		Result:         "automatic preview decision receipt reference",
		// It decides on the published preview, using the outcomes of the checks
		// the run already computed. Byline: Claude Code · Opus 5.5 · 2026-10-02
		DependsOn: []StageID{PublishPreview},
	},
	{
		ID:             MatchMessageOccurrences,
		Responsibility: RespMatchOccurrences,
		Result:         "message match-up receipt reference",
		// The plan it matches on is built from the recorded participant
		// resolution. Byline: Claude Code · Opus 5.5 · 2026-10-02
		DependsOn: []StageID{ResolveContextParticipants},
	},
	{
		ID:             CommitCallLog,
		Responsibility: RespCommitContext,
		Result:         "call log commit receipt reference",
		// The preview decision is the gate; the participant resolution names the
		// parties. Byline: Claude Code · Opus 5.5 · 2026-10-02
		DependsOn: []StageID{ResolveContextParticipants, PublishPreview},
	},
}
