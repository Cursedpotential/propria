// Byline: Claude Code · Fable 5.1 · 2026-09-20
// Byline: Claude Code · Opus 5 · 2026-09-21 (reconciled with the duplicate
// derive_structured_text_activity built on feat/derive-sms-activity: ONE
// Activity survives, this one, and it gains the durable seam the Proffer
// workflow needs — locator resolution and one append-only receipt.)
//
// derive_sms_threads_activity is one job: stream an SMS Backup & Restore XML
// once through the SBV decoder and publish media + per-thread NDJSON chunks +
// a manifest into the configured derived vault directory (owner design ruling
// 2026-09-20 18:47; "stream everything" 21:14; derived-root ruling 23:51). It
// parses nothing into PostgreSQL, computes no custody hash, selects no parser
// and starts no successor run: the workflow decides what to do with the
// manifest it names.

package activities

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// DeriveSMSThreadsActivityName is the exact Temporal name, and the stage
// graph's identity for the derive route.
const DeriveSMSThreadsActivityName = string(stagegraph.DeriveSMSThreads)

const (
	// DeriveHandlerID and DeriveHandlerVersion identify this implementation
	// in the durable handler registry exactly as the decoder and DuckDB
	// implementations identify theirs.
	DeriveHandlerID      = "smsthreads_derive"
	DeriveHandlerVersion = "1.0.0"

	// DerivedChunkDeclaredFormat is what a successor Proffer run declares for
	// one derived chunk. smsthreads publishes NDJSON, and the DuckDB
	// `ndjson_v1` template already reads a derived thread line as a
	// record_kind=message row.
	DerivedChunkDeclaredFormat = "ndjson"
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

// DeriveSMSThreadsActivities implements derive_sms_threads_activity.
type DeriveSMSThreadsActivities struct {
	// Locators resolves the retained source's object-store coordinate.
	Locators DeriveSourceLocatorStore
	// Stores resolves a configured object-store scheme ("b2", "r2") to a store.
	Stores func(scheme string) (smsthreads.ObjectStore, error)
	// Receipts records the append-only derivation receipt.
	Receipts DeriveReceiptStore
	// DerivedRoots maps a source locator to its derived vault directory.
	// Empty means every source falls back to beside-the-original placement.
	DerivedRoots smsthreads.DerivedRoots
	// ScratchRoot is an absolute directory on a data volume.
	ScratchRoot string
	MaxChunk    int64
	MaxOpen     int
	// Heartbeat reports liveness during a long stream; nil outside a worker.
	Heartbeat func(context.Context, Progress)
	// HeartbeatEvery defaults to 20 s and bounds the idle heartbeat ticker.
	HeartbeatEvery time.Duration
	Attempt        Attempt
}

func (a DeriveSMSThreadsActivities) validate() error {
	if a.Locators == nil {
		return errors.New("derive sms threads: source locator store is required")
	}
	if a.Stores == nil {
		return errors.New("derive sms threads: object store resolver is required")
	}
	if a.Receipts == nil {
		return errors.New("derive sms threads: receipt store is required")
	}
	root := strings.TrimSpace(a.ScratchRoot)
	if !filepath.IsAbs(root) && !strings.HasPrefix(root, "/") {
		return errors.New("derive sms threads: scratch root must be an absolute path on a data volume")
	}
	return nil
}

func (a DeriveSMSThreadsActivities) attempt(ctx context.Context) int32 {
	if a.Attempt == nil {
		return 1
	}
	if attempt := a.Attempt(ctx); attempt >= 1 {
		return attempt
	}
	return 1
}

// DeriveSMSThreads streams one source and publishes its derived objects.
//
// Retry safety: a finished derivation (its manifest exists at the mapped
// location) is loaded and returned as success, so a second identical attempt
// is a no-op rather than a failure. That decision is an explicit smsthreads
// option, never a parse of an error string.
func (a DeriveSMSThreadsActivities) DeriveSMSThreads(ctx context.Context, req proffer.StageRequest) (proffer.DeriveResult, error) {
	result, err := a.derive(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a DeriveSMSThreadsActivities) derive(ctx context.Context, req proffer.StageRequest) (proffer.DeriveResult, error) {
	if err := a.validate(); err != nil {
		return proffer.DeriveResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return proffer.DeriveResult{}, err
	}
	if strings.TrimSpace(req.RequestID) == "" || req.SourceVersionRef == "" {
		return proffer.DeriveResult{}, permanent(errors.New("derive sms threads requires request and source version references"))
	}
	if strings.TrimSpace(string(req.Refs["original"])) == "" {
		return proffer.DeriveResult{}, permanent(errors.New("derive sms threads requires the retained original reference"))
	}
	if err := requireStructuredELTDecisionRefs(req); err != nil {
		return proffer.DeriveResult{}, permanent(err)
	}

	locator, err := a.Locators.ResolveDeriveSource(ctx, req)
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("resolve derive source locator: %w", err)
	}
	if err := locator.validate(); err != nil {
		return proffer.DeriveResult{}, permanent(err)
	}
	store, err := a.Stores(locator.Scheme)
	if err != nil {
		return proffer.DeriveResult{}, permanent(fmt.Errorf("resolve object store for %q: %w", locator.Scheme, err))
	}

	// Mapped location first, then the legacy beside-the-original location, so
	// switching DERIVED_ROOTS_JSON on neither orphans nor re-derives anything.
	_, reused, err := smsthreads.ResolvePublished(
		ctx, store, a.DerivedRoots, locator.Scheme, locator.Bucket, locator.Key)
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("check existing derivation: %w", err)
	}

	stop := a.idleHeartbeat(ctx)
	_, published, err := smsthreads.Derive(ctx, smsthreads.Options{
		Store: store, Scheme: locator.Scheme, Bucket: locator.Bucket, Key: locator.Key,
		ScratchRoot: a.ScratchRoot, MaxChunk: a.MaxChunk, MaxOpen: a.MaxOpen,
		DerivedRoots:  a.DerivedRoots,
		ReuseExisting: true,
		Progress:      a.progressCallback(ctx),
	})
	stop()
	if err != nil {
		return proffer.DeriveResult{}, fmt.Errorf("derive structured text from %s: %w", locator.URI(), err)
	}
	// Read the published manifest back: it is what a successor run reads, so
	// the digest recorded here must be of the bytes the store actually
	// serves, not of anything held in memory.
	manifest, manifestSHA256, err := smsthreads.LoadManifestWithDigest(ctx, store, published.Bucket, published.ManifestKey())
	if err != nil {
		return proffer.DeriveResult{}, err
	}

	result := deriveResultFromManifest(locator, published, manifest, manifestSHA256, reused)
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
		Stage: stagegraph.DeriveSMSThreads, Status: proffer.StatusSuccess,
		Ref: resultRef, ReceiptRef: receiptRef,
	}
	result.BoundChunks()
	return result, nil
}

// idleHeartbeat keeps the Activity alive across a long stretch with no decoded
// record (opening a multi-gigabyte object, publishing one huge chunk). The
// per-record Progress callback covers the rest.
func (a DeriveSMSThreadsActivities) idleHeartbeat(ctx context.Context) func() {
	if a.Heartbeat == nil {
		return func() {}
	}
	every := a.HeartbeatEvery
	if every <= 0 {
		every = 20 * time.Second
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.Heartbeat(ctx, Progress{Stage: stagegraph.DeriveSMSThreads})
			}
		}
	}()
	return func() { close(done) }
}

func (a DeriveSMSThreadsActivities) progressCallback(ctx context.Context) func(smsthreads.Progress) {
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
			Stage:           stagegraph.DeriveSMSThreads,
			MembersComplete: members,
			BytesComplete:   progress.SourceBytes,
		})
	}
}

// deriveResultFromManifest projects the published manifest into the compact
// wire summary. The manifest object remains the authoritative chunk listing.
func deriveResultFromManifest(
	locator DeriveSourceLocator,
	published smsthreads.DerivedLocation,
	manifest smsthreads.Manifest,
	manifestSHA256 string,
	reused bool,
) proffer.DeriveResult {
	result := proffer.DeriveResult{
		SourceLocator: locator.URI(), SourceSHA256: manifest.SourceSHA256, SourceBytes: manifest.SourceBytes,
		ManifestURI:    published.URI() + smsthreads.ManifestName,
		ManifestSHA256: manifestSHA256,
		DerivedPrefix:  published.URI(),
		ThreadsPrefix:  published.URI() + smsthreads.ThreadsDir,
		Schema:         manifest.Schema,
		Records:        manifest.Records, Rejected: manifest.Rejected,
		MediaObjects: manifest.MediaObjects, MediaBytes: manifest.MediaBytes,
		ChunkCount: len(manifest.Threads), Reused: reused,
		Chunks: make([]proffer.DerivedChunkRef, 0, len(manifest.Threads)),
	}
	threads := make(map[string]struct{}, len(manifest.Threads))
	for _, thread := range manifest.Threads {
		threads[thread.Thread] = struct{}{}
		result.Chunks = append(result.Chunks, proffer.DerivedChunkRef{
			Thread: thread.Thread, Chunk: thread.Chunk,
			URI:    fmt.Sprintf("%s://%s/%s", published.Scheme, published.Bucket, thread.Key),
			SHA256: thread.SHA256, Records: thread.Records, Bytes: thread.Bytes,
			DeclaredFormat: DerivedChunkDeclaredFormat,
		})
	}
	result.ThreadCount = len(threads)
	return result
}
