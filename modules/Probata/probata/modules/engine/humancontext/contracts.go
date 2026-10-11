// Package humancontext defines native human-message bundles without AI or evidence-admission shapes.
// Inputs are registered source coordinates; outputs are external bundle references and compact receipts.
package humancontext

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/Cursedpotential/probata/engine/xmlsourceframes"
)

const Contract = "human-source-v1"
const IndexKind = "native-human-conversation-chunks-v1"
const FrameActivity = "frame_context_human_source_activity"
const ExtractActivity = "extract_context_human_source_activity"
const ChunkActivity = "chunk_context_human_source_activity"
const IndexActivity = "index_context_human_source_activity"
const VerifyActivity = "verify_context_human_source_index_activity"

// SameETag compares opaque provider tags with the optional HTTP quoting removed.
// Inputs: catalog/provider representations. Outputs: equality only. Effects: none; never decodes a digest.
func SameETag(left,right string)bool{return strings.Trim(strings.TrimSpace(left),"\"")==strings.Trim(strings.TrimSpace(right),"\"")}

// Source identifies actual registered native bytes and explicit source attribution.
// Digests observed after registration are provenance, never a prerequisite for registration.
type Source struct {
	SourceRef string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id,omitempty"`
	SourceVersionRef string `json:"source_version_ref"`
	RegistrationReceiptRef string `json:"registration_receipt_ref"`
	OwnerPersonRef string `json:"owner_person_ref,omitempty"`
	PerspectivePersonRef string `json:"perspective_person_ref,omitempty"`
	Perspective string `json:"perspective,omitempty"`
	Format string `json:"format"`
	ByteLength int64 `json:"byte_length"`
	ETag string `json:"etag,omitempty"`
}

// Request carries only source/manifest references across an atomic Temporal boundary.
// Inputs never include messages, vectors, attachment payloads or a fabricated retained-object ID.
type Request struct {
	ContractVersion string `json:"contract_version"`
	WorkflowID string `json:"workflow_id"`
	Source Source `json:"source"`
	ManifestRef string `json:"manifest_ref,omitempty"`
 ChunkManifestRef string `json:"chunk_manifest_ref,omitempty"`
}

// Result is a compact durable stage receipt, including separate publication and readback counts.
// Complete extraction does not imply indexed, verified, evidence-promoted or attachment-materialized.
type Result struct {
	ContractVersion string `json:"contract_version"`
	Stage string `json:"stage"`
	Status string `json:"status"`
	ManifestRef string `json:"manifest_ref"`
	ReceiptRef string `json:"receipt_ref,omitempty"`
	IndexKind string `json:"index_kind,omitempty"`
	Records int64 `json:"records"`
	Pending int64 `json:"pending"`
	Chunks int64 `json:"chunks"`
 Published int64 `json:"published"`
	Verified int64 `json:"verified"`
}

// Manifest binds stage files and source pins without putting native bodies into PostgreSQL or history.
// Files are immutable derivatives below the configured human bundle root; original bytes stay in place.
type Manifest struct {
	ContractVersion string `json:"contract_version"`
	WorkflowID string `json:"workflow_id"`
	Stage string `json:"stage"`
	Source Source `json:"source"`
	Framing xmlsourceframes.Receipt `json:"framing"`
	LexemesRef string `json:"lexemes_ref,omitempty"`
	SpansRef string `json:"spans_ref,omitempty"`
	RecordsRef string `json:"records_ref,omitempty"`
	OffsetsRef string `json:"offsets_ref,omitempty"`
	ChunksRef string `json:"chunks_ref,omitempty"`
 PublicationRef string `json:"publication_ref,omitempty"`
	IndexKind string `json:"index_kind,omitempty"`
	ObservedSHA256 string `json:"observed_sha256,omitempty"`
	Records int64 `json:"records"`
	Pending int64 `json:"pending"`
	Chunks int64 `json:"chunks"`
 Published int64 `json:"published"`
	Verified int64 `json:"verified"`
}

// Record contains DuckDB's native semantic output; Go only transports and projects these fields.
// Ordinal and locator always refer to the original source order, not a normalized-generation identity.
type Record struct {
	Ordinal int64 `json:"ordinal"`
	Status string `json:"status"`
	Native json.RawMessage `json:"native"`
}

// SourceOpener opens the registered source with actual version and ETag pin checks.
// Effects are read-only; a changed source must fail rather than silently following its latest version.
type SourceOpener interface { Open(context.Context,Source)(io.ReadCloser,Source,error) }

// Extractor performs native field mapping in DuckDB from one bounded generic lexical unit.
// No Go SMS mapping, legacy handler receipt or retained original is accepted as an alternative.
type Extractor interface { Extract(context.Context,Source,[]xmlsourceframes.Lexeme)(Record,error) }

// MetadataStore records bundle/operation references only after each completed atomic stage.
// Choose for source and viewer discoverability; do not persist native bodies in this interface.
type MetadataStore interface {
	RecordStage(context.Context,Request,Result)(Result,error)
	LookupStage(context.Context,Request,string)(Result,bool,error)
}
