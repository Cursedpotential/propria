// Byline: Claude Code · Opus 5 · 2026-09-20
//
// derive_structured_text_activity is the Temporal boundary around
// engine/derive/smsthreads. One unit, one job (AGENTS.md ATOMICITY): it
// streams one retained source from its own object store and publishes
// memory-safe structured text beside it. It writes no raw records, computes
// no custody hash, selects no parser, and starts no successor run — the
// workflow owns all of that.
//
// It is a NEW Activity rather than a widening of execute_parser_activity
// because its output is not a parser bundle: nothing downstream of it is a
// raw generation. The derived NDJSON chunks re-enter the platform as ordinary
// `ndjson` sources, each through its own Proffer run.

package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// DeriveSourceLocator is one object-store coordinate: the scheme names a
// configured store, never a provider in code.
type DeriveSourceLocator struct {
	Scheme string
	Bucket string
	Key    string
}

// URI spells the locator back exactly as the source declared it.
func (l DeriveSourceLocator) URI() string {
	return fmt.Sprintf("%s://%s/%s", l.Scheme, l.Bucket, l.Key)
}

func (l DeriveSourceLocator) validate() error {
	if strings.TrimSpace(l.Scheme) == "" || strings.TrimSpace(l.Bucket) == "" || strings.TrimSpace(l.Key) == "" {
		return errors.New("derive source locator requires a scheme, bucket and key")
	}
	return nil
}

// DeriveSourceLocatorStore resolves the retained source's own acquisition
// locator. A retained file:// copy belongs to one worker host and is not a
// place new objects may be published, so a source without an object-store
// locator is refused rather than silently derived somewhere else.
type DeriveSourceLocatorStore interface {
	ResolveDeriveSource(context.Context, proffer.StageRequest) (DeriveSourceLocator, error)
}

// DeriveObjectStores hands back a ready store for one configured scheme.
type DeriveObjectStores interface {
	StoreForScheme(scheme string) (smsthreads.ObjectStore, error)
}

// DeriveReceiptSpec is what the durable receipt records: references, digests
// and counts. The derived bodies stay in object storage.
type DeriveReceiptSpec struct {
	RequestID        string
	SourceVersionRef proffer.Ref
	SourceLocator    string
	ManifestURI      string
	ManifestSHA256   string
	DerivedPrefix    string
	Schema           string
	Records          uint64
	Rejected         uint64
	MediaObjects     uint64
	ChunkCount       int
	ThreadCount      int
	Reused           bool
	Attempt          int32
}

// DeriveReceiptStore persists the derivation outcome and returns the compact
// result reference and the durable activity receipt reference.
type DeriveReceiptStore interface {
	PersistDerivedGeneration(context.Context, DeriveReceiptSpec) (result proffer.Ref, receipt proffer.Ref, err error)
}

// DeriveActivities implements derive_structured_text_activity.
type DeriveActivities struct {
	Locators DeriveSourceLocatorStore
	Stores   DeriveObjectStores
	Receipts DeriveReceiptStore
	// ScratchRoot must be an absolute path on a data volume, never a system
	// temp dir: a multi-gigabyte derivation stages its chunks there.
	ScratchRoot string
	MaxChunk    int64
	MaxOpen     int
	Heartbeat   func(context.Context, Progress)
	Attempt     Attempt
}

func (a DeriveActivities) validate() error {
	if a.Locators == nil {
		return errors.New("derive activities: source locator store is required")
	}
	if a.Stores == nil {
		return errors.New("derive activities: object store resolver is required")
	}
	if a.Receipts == nil {
		return errors.New("derive activities: receipt store is required")
	}
	if strings.TrimSpace(a.ScratchRoot) == "" {
		return errors.New("derive activities: scratch root is required")
	}
	return nil
}

func (a DeriveActivities) attempt(ctx context.Context) int32 {
	if a.Attempt == nil {
		return 1
	}
	attempt := a.Attempt(ctx)
	if attempt < 1 {
		return 1
	}
	return attempt
}

// DeriveStructuredText streams one source and publishes its derived objects.
//
// Retry safety: a finished derivation (its manifest exists) is loaded and
// returned as success, so a second identical attempt is a no-op rather than a
// failure. That decision is an explicit smsthreads option, never a parse of
// an error string.
func (a DeriveActivities) DeriveStructuredText(ctx context.Context, req proffer.StageRequest) (proffer.DeriveResult, error) {
	if err := a.validate(); err != nil {
		return proffer.DeriveResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return proffer.DeriveResult{}, err
	}
	if strings.TrimSpace(req.RequestID) == "" || req.SourceVersionRef == "" {
		return proffer.DeriveResult{}, errors.New("derive structured text requires request and source version references")
	}
	if strings.TrimSpace(string(req.Refs["original"])) == "" {
		return proffer.DeriveResult{}, errors.New("derive structured text requires the retained original reference")
	}
	if err := requireStructuredELTDecisionRefs(req); err != nil {
		return proffer.DeriveResult{}, err
	}

	locator, err := a.Locators.ResolveDeriveSource(ctx, req)
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("resolve derive source locator: %w", err)
	}
	if err := locator.validate(); err != nil {
		return proffer.DeriveResult{}, err
	}
	store, err := a.Stores.StoreForScheme(locator.Scheme)
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("resolve object store for %q: %w", locator.Scheme, err)
	}

	manifestKey := smsthreads.ManifestKey(locator.Key)
	reused, err := store.Exists(ctx, locator.Bucket, manifestKey)
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("check existing derivation: %w", err)
	}

	if _, err := smsthreads.Derive(ctx, smsthreads.Options{
		Store: store, Scheme: locator.Scheme, Bucket: locator.Bucket, Key: locator.Key,
		ScratchRoot: a.ScratchRoot, MaxChunk: a.MaxChunk, MaxOpen: a.MaxOpen,
		ReuseExisting: true,
		Progress:      a.progressCallback(ctx),
	}); err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("derive structured text from %s: %w", locator.URI(), err)
	}
	// Read the published manifest back: it is what a successor run reads, so
	// the digest recorded here must be of the bytes the store actually
	// serves, not of anything held in memory.
	manifest, manifestSHA256, err := smsthreads.LoadManifestWithDigest(ctx, store, locator.Bucket, manifestKey)
	if err != nil {
		return proffer.DeriveResult{}, err
	}

	result := deriveResultFromManifest(locator, manifest, manifestSHA256, reused)
	resultRef, receiptRef, err := a.Receipts.PersistDerivedGeneration(ctx, DeriveReceiptSpec{
		RequestID: req.RequestID, SourceVersionRef: req.SourceVersionRef,
		SourceLocator: locator.URI(), ManifestURI: result.ManifestURI, ManifestSHA256: result.ManifestSHA256,
		DerivedPrefix: result.DerivedPrefix, Schema: result.Schema,
		Records: result.Records, Rejected: result.Rejected, MediaObjects: result.MediaObjects,
		ChunkCount: result.ChunkCount, ThreadCount: result.ThreadCount, Reused: reused,
		Attempt: a.attempt(ctx),
	})
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("persist derived generation: %w", err)
	}
	if resultRef == "" || receiptRef == "" {
		return proffer.DeriveResult{}, errors.New("persisted derived generation lacks result or activity receipt reference")
	}
	result.Result = proffer.StageResult{
		Stage: stagegraph.DeriveStructuredText, Status: proffer.StatusSuccess,
		Ref: resultRef, ReceiptRef: receiptRef,
	}
	result.BoundChunks()
	return result, nil
}

func (a DeriveActivities) progressCallback(ctx context.Context) func(smsthreads.Progress) {
	if a.Heartbeat == nil {
		return nil
	}
	return func(progress smsthreads.Progress) {
		// The shared Progress shape is reused rather than widened: members
		// are decoded records (plus rejects), bytes are source bytes read.
		members := int64(progress.Records + progress.Rejected)
		if progress.Phase == "publishing" {
			members = int64(progress.PublishedObjects)
		}
		a.Heartbeat(ctx, Progress{
			Stage:           stagegraph.DeriveStructuredText,
			MembersComplete: members,
			BytesComplete:   progress.SourceBytes,
		})
	}
}

// deriveResultFromManifest projects the published manifest into the compact
// wire summary. The manifest object remains the authoritative chunk listing.
func deriveResultFromManifest(locator DeriveSourceLocator, manifest smsthreads.Manifest, manifestSHA256 string, reused bool) proffer.DeriveResult {
	result := proffer.DeriveResult{
		SourceLocator: locator.URI(), SourceSHA256: manifest.SourceSHA256, SourceBytes: manifest.SourceBytes,
		ManifestURI:   manifest.DerivedPrefix + smsthreads.ManifestName, ManifestSHA256: manifestSHA256,
		DerivedPrefix: manifest.DerivedPrefix, Schema: manifest.Schema,
		Records: manifest.Records, Rejected: manifest.Rejected,
		MediaObjects: manifest.MediaObjects, MediaBytes: manifest.MediaBytes,
		ChunkCount: len(manifest.Threads), Reused: reused,
		Chunks: make([]proffer.DerivedChunkRef, 0, len(manifest.Threads)),
	}
	threads := make(map[string]struct{}, len(manifest.Threads))
	for _, thread := range manifest.Threads {
		threads[thread.Thread] = struct{}{}
		result.Chunks = append(result.Chunks, proffer.DerivedChunkRef{
			Thread: thread.Thread, Chunk: thread.Chunk,
			URI:    fmt.Sprintf("%s://%s/%s", locator.Scheme, locator.Bucket, thread.Key),
			SHA256: thread.SHA256, Records: thread.Records, Bytes: thread.Bytes,
			DeclaredFormat: DerivedChunkDeclaredFormat,
		})
	}
	result.ThreadCount = len(threads)
	return result
}

// DerivedChunkDeclaredFormat is what a successor Proffer run declares for one
// derived chunk. smsthreads publishes NDJSON, and the DuckDB `ndjson_v1`
// template already reads a derived thread line as a record_kind=message row.
const DerivedChunkDeclaredFormat = "ndjson"

const (
	// DeriveHandlerID and DeriveHandlerVersion identify this implementation
	// in the durable handler registry exactly as the decoder and DuckDB
	// implementations identify theirs.
	DeriveHandlerID      = "smsthreads_derive"
	DeriveHandlerVersion = "1.0.0"
)

// DeriveEligibleFormat is the signature registry for the derive route: one
// entry per content signature, no primary/fallback ladder (owner standing
// rule — the router is signature-based).
//
// smsbackuprestore_xml is here because the DuckDB `sms_xml_v1` template
// cannot read a real backup: pg_duckdb 1.1.1 (DuckDB 1.4.3) has no streaming
// XML reader and a 16 MB default cap, and valid backups run 500 MB to several
// gigabytes. The derived NDJSON chunks are then read by `ndjson_v1`.
func DeriveEligibleFormat(detected string) bool {
	return strings.TrimSpace(detected) == "smsbackuprestore_xml"
}
