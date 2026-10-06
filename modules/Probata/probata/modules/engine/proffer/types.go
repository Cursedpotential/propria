// Byline: Codex · GPT-5 · 2026-10-05 (single-case operating contract)
// Package proffer implements ProfferWorkflow, the single Temporal
// workflow every source — every format, client, and entrypoint — runs
// through, per
// docs/reviews/2026-08-25-schema-audit/SBV-GO-TEMPORAL-RUNTIME-BOUNDARY.html
// (Lane C). It orchestrates the exact stage graph locked in
// engine/stagegraph and does not implement any Activity body: Activities are
// invoked by their canon name and are registered by whichever worker lands
// in a later lane, per the boundary document's lane table (Lane C depends on
// Lane A contracts and Lane B PostgreSQL interfaces; it does not provide
// them).
//
// Package proffer (formerly uiw / Universal Import Workflow; renamed D-140, 2026-09-05).
package proffer

import (
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// ActivityName is the Temporal-registered name for one Activity in
// ProfferWorkflow. It is always identical to the canon StageID from
// engine/stagegraph, so the orchestration graph in workflow.go and the
// Temporal task-dispatch table can never drift apart.
type ActivityName = stagegraph.StageID

// Ref is a compact opaque pointer into external storage — a PostgreSQL row
// id, an immutable-object-storage key, or a receipt id. A Ref is never a
// payload. Per the boundary document's acceptance gate 6 ("Temporal history
// contains compact references only"), nothing in this package may carry a
// file, a raw record, a normalized record, or a metadata payload — those
// stay in PostgreSQL/immutable storage and Activities resolve them by
// following the Refs handed across the workflow boundary.
type Ref string

// PreviewPublicationRequest is the compact reference-only payload used by
// publish_preview_activity. Workflow and run identifiers stay in the durable
// preview binding created by the starter and are never exposed as the browser
// handle.
type PreviewPublicationRequest struct {
	OperatingMode           string         `json:"operating_mode,omitempty"`
	MatterID                string         `json:"matter_id,omitempty"`
	CourtCaseID             string         `json:"court_case_id,omitempty"`
	RequestID               string         `json:"request_id"`
	PackageRef              Ref            `json:"package_ref,omitempty"`
	AttemptRef              Ref            `json:"attempt_ref,omitempty"`
	SourceVersionRef        Ref            `json:"source_version_ref"`
	SourceRepresentationRef Ref            `json:"source_representation_ref,omitempty"`
	RawGenerationRef        Ref            `json:"raw_generation_ref"`
	NormalizedGenerationRef Ref            `json:"normalized_generation_ref"`
	ChunkGenerationRef      Ref            `json:"chunk_generation_ref,omitempty"`
	ChunkReceiptRef         Ref            `json:"chunk_receipt_ref,omitempty"`
	ParserSelectionRef      Ref            `json:"parser_selection_ref"`
	ParserOptionsRef        Ref            `json:"parser_options_ref"`
	ReceiptRefs             map[string]Ref `json:"receipt_refs"`
}

// ContextChunkingInput opts a non-messaging package into the D-158 context
// chunk path. It contains only opaque references and short, version-pinned
// controlled-vocabulary identifiers. Source bytes, extracted records, chunk
// text, and parser/template bodies never enter Temporal history.
//
// Messaging imports leave the corresponding WorkflowInput fields empty and
// preserve the existing normalized-message preview path. A caller must not
// use this as a generic "chunk every import" flag: message chunking has its
// own ordered-record contract.
type ContextChunkingInput struct {
	PackageRef    Ref
	AttemptRef    Ref
	ContextKind   string
	Signature     string
	PolicyID      string
	PolicyVersion string
}

func (in ContextChunkingInput) validate() error {
	if in.PackageRef == "" || in.AttemptRef == "" {
		return fmt.Errorf("non-messaging context chunking requires package and extraction-attempt references")
	}
	if in.ContextKind != "non_messaging" {
		return fmt.Errorf("context chunking requires context kind %q", "non_messaging")
	}
	for name, value := range map[string]string{
		"signature":      in.Signature,
		"policy id":      in.PolicyID,
		"policy version": in.PolicyVersion,
	} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("non-messaging context chunking requires %s", name)
		}
		if len(trimmed) > 256 {
			return fmt.Errorf("non-messaging context chunking %s exceeds 256 bytes", name)
		}
	}
	return nil
}

// Status is the recorded business outcome of one stage. success and
// not_applicable are both valid terminal states that let the workflow
// proceed to the stage's dependents; failed halts every descendant and the
// seal/publish stages, per the boundary document's per-Activity contract.
type Status string

const (
	StatusSuccess       Status = "success"
	StatusNotApplicable Status = "not_applicable"
	StatusFailed        Status = "failed"
)

// WorkflowInput starts ProfferWorkflow. It names the not-yet-
// retained acquisition object and the client idempotency coordinate;
// register_source_activity (stage 1) turns SourceRef into the durable
// source/version reference every later stage keys off.
type WorkflowInput struct {
	// OperatingMode records explicit operating context in durable workflow history.
	OperatingMode string
	// RequestID is the client-supplied idempotency key. Callers are expected
	// to use it as the Temporal workflow ID so a duplicate submission joins
	// the existing run rather than starting a second one. It is also carried
	// on every StageRequest (see StageRequest.RequestID) so an Activity can
	// key its own idempotency/dedup checks off the same coordinate the
	// workflow itself uses.
	RequestID   string
	MatterID    string
	CourtCaseID string
	// SourceRef points at the not-yet-retained acquisition object (upload,
	// watcher-discovered file, or other external pointer). It is never the
	// bytes themselves.
	SourceRef Ref
	// DeclaredFormat is the short format tag from the boundary document's
	// ParserInput contract (section 3), e.g. "whatsapp_export_json". It is
	// an identifier, never file content.
	DeclaredFormat string
	// ParserOptionsRef is the ParserInput.parser_options_ref: a reference to
	// parser configuration, not the configuration payload itself.
	ParserOptionsRef Ref
	// SourceContextRef points to the append-only, actor-bound operator
	// assertion receipt. Metadata values never enter Temporal history.
	SourceContextRef Ref
	// The context-chunk fields enable the D-158 non-messaging context path.
	// Empty fields preserve the established messaging path. The flat wire
	// shape deliberately uses only Refs and bounded strings.
	PackageRef                Ref
	AttemptRef                Ref
	ContextKind               string
	ContextChunkSignature     string
	ContextChunkPolicyID      string
	ContextChunkPolicyVersion string
	// OwnerPersonID and PerspectivePersonID are the registry.person ids the
	// first-party context import (D04) writes threads for: the case owner, and
	// whose device or export the source is. They are explicit run inputs,
	// never derived. A run whose generation holds messages fails loudly at
	// propose_first_party_context_activity when either is absent.
	// Byline: Claude Code · Opus 5.5 · 2026-10-01
	OwnerPersonID       string `json:",omitempty"`
	PerspectivePersonID string `json:",omitempty"`
	// AutoApproval names the preview-approval policy for this run only; a
	// batch sets it on each item it starts. Empty (the default) means the owner
	// decides every preview. AutoApprovalCleanChecks is the owner's
	// "auto-approve clean runs" policy: when every check in AutoApprovalChecks
	// settled success, the run records an automatic approval
	// (record_auto_approval_activity) instead of waiting; otherwise it waits for
	// the owner exactly as it does when the policy is off.
	// Byline: Claude Code · Opus 5.5 · 2026-10-02
	AutoApproval string `json:",omitempty"`
}

// AutoApprovalCleanChecks is the only automatic approval policy (owner
// 2026-10-02). Byline: Claude Code · Opus 5.5 · 2026-10-02
const AutoApprovalCleanChecks = "clean_checks"

// ValidAutoApproval reports whether policy is empty or a known policy name.
func ValidAutoApproval(policy string) bool {
	return policy == "" || policy == AutoApprovalCleanChecks
}

// AutoApprovalChecks are the checks the pipeline already computes that must
// all settle success before a run may approve itself (owner 2026-10-02):
// record accounting, byte coverage, raw coverage against the source,
// normalized verification, and participant resolution. A not_applicable
// outcome counts as not passed, with one exception: LocatorlessFormats.
var AutoApprovalChecks = []stagegraph.StageID{
	stagegraph.ReconcileRecordAccounting,
	stagegraph.ReconcileByteCoverage,
	stagegraph.VerifyRawCoverageAgainstSource,
	stagegraph.VerifyNormalizedGeneration,
	stagegraph.ResolveContextParticipants,
}

// LocatorlessFormats are the detected formats whose raw records carry no byte
// locators into the retained original, so reconcile_byte_coverage always
// settles not_applicable for them: the derived SMS thread chunks (ndjson) and
// Facebook Messenger thread JSON and HTML, and generic HTML documents. For these formats alone a receipted
// not_applicable byte-coverage check counts as passed (owner 2026-10-02,
// option A: "This is all supposed to be programmatic"); every other check must
// still be a receipted success. Byline: Claude Code · Opus 5.5 · 2026-10-02
var LocatorlessFormats = map[string]bool{
	"ndjson":                  true,
	"facebook_messenger_json": true,
	// The HTML flavors are re-serialized blocks, not byte ranges of the original.
	"facebook_messenger_html": true,
	"generic_html_document":   true,
}

// AutoApprovalCheckPasses reports whether one check satisfies the clean_checks
// policy for a source of the given detected format.
func AutoApprovalCheckPasses(stage stagegraph.StageID, status Status, receipt Ref, detectedFormat string) bool {
	if strings.TrimSpace(string(receipt)) == "" {
		return false
	}
	if status == StatusSuccess {
		return true
	}
	return stage == stagegraph.ReconcileByteCoverage && status == StatusNotApplicable && LocatorlessFormats[detectedFormat]
}

// AutoApprovalCheck is one passed check, by reference.
type AutoApprovalCheck struct {
	Stage      ActivityName `json:"stage"`
	Status     Status       `json:"status"`
	ReceiptRef Ref          `json:"receipt_ref"`
}

// AutoApprovalRequest is record_auto_approval_activity's compact input.
// Byline: Claude Code · Opus 5.5 · 2026-10-02
type AutoApprovalRequest struct {
	OperatingMode    string              `json:"operating_mode,omitempty"`
	MatterID         string              `json:"matter_id,omitempty"`
	CourtCaseID      string              `json:"court_case_id,omitempty"`
	RequestID        string              `json:"request_id"`
	PreviewHandle    Ref                 `json:"preview_handle"`
	SelectionRef     Ref                 `json:"selection_ref"`
	ParserOptionsRef Ref                 `json:"parser_options_ref"`
	Checks           []AutoApprovalCheck `json:"checks"`
	// DetectedFormat lets the Activity re-check the LocatorlessFormats rule
	// itself. Byline: Claude Code · Opus 5.5 · 2026-10-02
	DetectedFormat string `json:"detected_format,omitempty"`
}

// personRefs carries the explicit person ids to the first-party context
// stages as references. Absent ids are left out, so the Activity, not the
// workflow, reports the missing identity.
func (in WorkflowInput) personRefs(refs map[string]Ref) map[string]Ref {
	if id := strings.TrimSpace(in.OwnerPersonID); id != "" {
		refs["owner_person"] = Ref(id)
	}
	if id := strings.TrimSpace(in.PerspectivePersonID); id != "" {
		refs["perspective_person"] = Ref(id)
	}
	return refs
}

func (in WorkflowInput) contextChunkingInput() *ContextChunkingInput {
	if in.PackageRef == "" && in.AttemptRef == "" && strings.TrimSpace(in.ContextKind) == "" &&
		strings.TrimSpace(in.ContextChunkSignature) == "" && strings.TrimSpace(in.ContextChunkPolicyID) == "" &&
		strings.TrimSpace(in.ContextChunkPolicyVersion) == "" {
		return nil
	}
	return &ContextChunkingInput{
		PackageRef: in.PackageRef, AttemptRef: in.AttemptRef, ContextKind: strings.TrimSpace(in.ContextKind),
		Signature: strings.TrimSpace(in.ContextChunkSignature), PolicyID: strings.TrimSpace(in.ContextChunkPolicyID),
		PolicyVersion: strings.TrimSpace(in.ContextChunkPolicyVersion),
	}
}

// StageRequest is the single compact wire type sent to every Activity: the
// running source/version reference, the declared-format tag, and whichever
// named upstream references that stage's contract declares as inputs.
// Activities resolve anything else they need (file bytes, records,
// metadata) from PostgreSQL/immutable storage by following these
// references — never from the workflow payload.
type StageRequest struct {
	// OperatingMode is explicit durable policy; absent legacy payloads are unknown.
	OperatingMode string `json:"OperatingMode,omitempty"`
	// RequestID is WorkflowInput.RequestID, propagated to every Activity
	// invocation (not just register_source_activity) so any Activity can
	// key its own idempotency/dedup checks off the same client-supplied
	// coordinate the workflow uses as its Temporal workflow ID.
	RequestID        string
	MatterID         string
	CourtCaseID      string
	SourceVersionRef Ref
	DeclaredFormat   string
	// Refs is a small, named set of upstream references this stage
	// consumes, e.g. {"original": ..., "raw_generation": ...}. Keys are
	// stable, documented names, not row content.
	Refs map[string]Ref
}

// StageResult is the single compact wire type every Activity returns. Every
// terminal Status (success, not_applicable, or a business-reported failed)
// must carry a durable ReceiptRef; see validateStageResult in workflow.go for
// the exact per-Status contract this type's fields are held to.
type StageResult struct {
	Stage ActivityName
	// AIChatSource records the participant-resolution Activity's verified persisted AI classification.
	// Input: store-resolved source/raw formats, never the request label. Output: a workflow routing fact.
	// Effects: none beyond the Activity result history. Use only with its not-applicable receipt; search rechecks provenance.
	AIChatSource bool `json:"ai_chat_source,omitempty"`
	// Status is the business outcome; see the Status constants.
	Status Status
	// Ref is this stage's compact usable result registry. Required (must be
	// non-empty) when Status is StatusSuccess. Always empty for
	// StatusFailed — a business failure has nothing usable to hand
	// downstream, only a receipt of the outcome. For StatusNotApplicable,
	// Ref is usually empty but MAY carry a durable reference when one
	// genuinely exists (e.g. reconcile_byte_coverage_activity recording a
	// "no byte coverage to reconcile" marker). When present it propagates to
	// dependent stages exactly like a success Ref. When absent, settle uses
	// the required ReceiptRef as the compact downstream dependency Ref while
	// preserving this StageResult exactly as returned. Thus a later stage can
	// cite the durable N/A determination instead of receiving an empty string
	// indistinguishable from "nothing recorded at all."
	Ref Ref
	// ReceiptRef is this stage's durable receipt reference — proof the
	// stage's outcome was recorded, independent of Ref. Required (must be
	// non-empty) for every terminal Status: StatusSuccess,
	// StatusNotApplicable, and a business-reported StatusFailed all
	// certify something happened and must be provable after the fact. It is
	// never populated when Get returns a Temporal execution error, because
	// the Activity may have crashed before producing any receipt at all.
	ReceiptRef Ref
	// Reason is required (must be non-empty) when Status is not
	// StatusSuccess: a short, operator-facing explanation. It is never a
	// payload.
	Reason string
}

// WorkflowResult is the terminal, compact summary of one workflow
// execution.
type WorkflowResult struct {
	SourceVersionRef Ref
	// PublicationRef is publish_generation_activity's receipt. Empty unless
	// Status is StatusSuccess.
	PublicationRef Ref
	Status         Status
	// Stages is the ordered receipt of every stage that actually ran, in
	// the order its result was recorded. A failed run's Stages ends at the
	// first failure; no descendant or seal/publish receipt follows it.
	Stages []StageResult
	// Derived is set, and PublicationRef empty, only on the derive route: a
	// source no in-place extractor can read is republished as structured
	// text beside the original and this run ends there. Each derived chunk
	// is ingested by its own successor Proffer run, so this run has no raw
	// generation to seal or publish.
	//
	// Byline: Claude Code · Opus 5 · 2026-09-20
	Derived *DeriveResult `json:"derived,omitempty"`
	// AutoExtraction says whether entity and event extraction was started as a child after the
	// messages were committed: "started: <workflow id>" or "not started: <reason>". A failure to
	// start never fails the import.
	AutoExtraction string `json:"auto_extraction,omitempty"`
}
