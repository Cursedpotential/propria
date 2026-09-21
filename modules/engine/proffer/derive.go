// Byline: Claude Code · Opus 5 · 2026-09-20
//
// Wire types for derive_structured_text_activity. Everything here is a
// reference or a count: the derived chunk bodies stay in object storage and
// the authoritative chunk list stays in the derived manifest object, which
// ManifestURI names.

package proffer

import (
	"errors"
	"fmt"
	"strings"
)

// maxDerivedChunkRefs bounds how many derived chunk references may be echoed
// into Temporal history. Beyond it the manifest object is the only listing —
// history stays compact and the reference stays authoritative either way.
const maxDerivedChunkRefs = 256

// DerivedChunkRef names one published structured-text chunk. It is a locator
// plus its digest and record count, never chunk content.
type DerivedChunkRef struct {
	Thread  string `json:"thread"`
	Chunk   int    `json:"chunk"`
	URI     string `json:"uri"`
	SHA256  string `json:"sha256"`
	Records uint64 `json:"records"`
	Bytes   int64  `json:"bytes"`
	// DeclaredFormat is the format a successor Proffer run declares for this
	// chunk. It is always "ndjson" for the smsthreads schema.
	DeclaredFormat string `json:"declared_format"`
}

// DeriveResult is derive_structured_text_activity's return value: an ordinary
// StageResult plus the compact derivation summary the terminal WorkflowResult
// reports. Chunks is bounded; when the derivation published more chunks than
// history may carry, ChunksTruncated is true and the manifest is the listing.
type DeriveResult struct {
	Result StageResult `json:"result"`

	SourceLocator  string `json:"source_locator"`
	SourceSHA256   string `json:"source_sha256"`
	SourceBytes    int64  `json:"source_bytes"`
	ManifestURI    string `json:"manifest_uri"`
	ManifestSHA256 string `json:"manifest_sha256"`
	DerivedPrefix  string `json:"derived_prefix"`
	// ThreadsPrefix is the folder a batch import can be started on. The
	// derive route deliberately does NOT start it (owner build order step 4):
	// it returns the locator and a human decides.
	ThreadsPrefix string `json:"threads_prefix"`
	Schema        string `json:"schema"`

	Records      uint64 `json:"records"`
	Rejected     uint64 `json:"rejected"`
	MediaObjects uint64 `json:"media_objects"`
	MediaBytes   int64  `json:"media_bytes"`
	ChunkCount   int    `json:"chunk_count"`
	ThreadCount  int    `json:"thread_count"`

	// Reused is true when a finished derivation was found and returned
	// unchanged instead of being recomputed.
	Reused bool `json:"reused"`

	// Chunks is never nil: an empty listing encodes as [], not null.
	Chunks          []DerivedChunkRef `json:"chunks"`
	ChunksTruncated bool              `json:"chunks_truncated"`
}

// BoundChunks trims Chunks to maxDerivedChunkRefs, and guarantees a non-nil
// slice so the JSON payload is [] rather than null. Activities call this
// before returning.
func (d *DeriveResult) BoundChunks() {
	if d.Chunks == nil {
		d.Chunks = []DerivedChunkRef{}
	}
	if len(d.Chunks) > maxDerivedChunkRefs {
		d.Chunks = d.Chunks[:maxDerivedChunkRefs]
		d.ChunksTruncated = true
	}
}

// validateDeriveResult fails closed on a derivation that cannot be handed to
// a successor run: no manifest to read, no chunks to ingest, or a StageResult
// that does not satisfy the ordinary receipt contract.
func validateDeriveResult(result DeriveResult) error {
	if strings.TrimSpace(result.ManifestURI) == "" || strings.TrimSpace(result.ManifestSHA256) == "" {
		return errors.New("derive result lacks a derived manifest reference")
	}
	if strings.TrimSpace(result.DerivedPrefix) == "" {
		return errors.New("derive result lacks a derived prefix")
	}
	if result.ChunkCount <= 0 {
		return errors.New("derive result published no structured-text chunks")
	}
	for index, chunk := range result.Chunks {
		if strings.TrimSpace(chunk.URI) == "" || strings.TrimSpace(chunk.SHA256) == "" {
			return fmt.Errorf("derived chunk %d lacks a locator or digest", index)
		}
		if strings.TrimSpace(chunk.DeclaredFormat) == "" {
			return fmt.Errorf("derived chunk %d lacks the format a successor run must declare", index)
		}
	}
	return nil
}
