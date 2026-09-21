// Byline: Claude Code · Fable 5.1 · 2026-09-20
package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
)

// DeriveSMSThreadsActivityName is one job: stream an SMS Backup & Restore XML
// once through the SBV decoder and publish media + per-thread NDJSON chunks +
// a manifest beside the original (owner design ruling 2026-09-20 18:47; "stream
// everything" 21:14). It parses nothing into PostgreSQL and routes nothing: the
// workflow decides what to do with the manifest it names.
const DeriveSMSThreadsActivityName = "derive_sms_threads_activity"

// DeriveSMSThreadsRequest passes a locator, never bytes.
type DeriveSMSThreadsRequest struct {
	SourceURI     string `json:"source_uri"` // <scheme>://<bucket>/<key>
	MaxChunkBytes int64  `json:"max_chunk_bytes,omitempty"`
}

// DeriveSMSThreadsResult names the manifest; the thread and media lists stay in
// the object store, not in Temporal history.
type DeriveSMSThreadsResult struct {
	ManifestURI    string `json:"manifest_uri"`
	SourceSHA256   string `json:"source_sha256"`
	SourceBytes    int64  `json:"source_bytes"`
	Records        uint64 `json:"records"`
	Rejected       uint64 `json:"rejected"`
	ThreadChunks   int    `json:"thread_chunks"`
	MediaObjects   uint64 `json:"media_objects"`
	AlreadyDerived bool   `json:"already_derived"`
}

type DeriveSMSThreadsActivities struct {
	// Stores resolves a configured object-store scheme ("b2", "r2") to a store.
	Stores func(scheme string) (smsthreads.ObjectStore, error)
	// ScratchRoot is an absolute directory on a data volume.
	ScratchRoot string
	// Heartbeat reports liveness during a long stream; nil outside a worker.
	Heartbeat func(ctx context.Context, details ...interface{})
	// HeartbeatEvery defaults to 20 s.
	HeartbeatEvery time.Duration
}

func (a DeriveSMSThreadsActivities) DeriveSMSThreads(ctx context.Context, req DeriveSMSThreadsRequest) (DeriveSMSThreadsResult, error) {
	result, err := a.deriveSMSThreads(ctx, req)
	return result, stopRetryingPermanent(err)
}

func (a DeriveSMSThreadsActivities) deriveSMSThreads(ctx context.Context, req DeriveSMSThreadsRequest) (DeriveSMSThreadsResult, error) {
	if a.Stores == nil {
		return DeriveSMSThreadsResult{}, errors.New("derive sms threads: object store resolver is required")
	}
	if !filepath.IsAbs(a.ScratchRoot) && !strings.HasPrefix(a.ScratchRoot, "/") {
		return DeriveSMSThreadsResult{}, errors.New("derive sms threads: scratch root must be an absolute path on a data volume")
	}
	scheme, rest, found := strings.Cut(strings.TrimSpace(req.SourceURI), "://")
	bucket, key, hasKey := strings.Cut(rest, "/")
	if !found || !hasKey || scheme == "" || bucket == "" || key == "" {
		return DeriveSMSThreadsResult{}, permanent(fmt.Errorf("derive sms threads: source_uri %q is not <scheme>://<bucket>/<key>", req.SourceURI))
	}
	if req.MaxChunkBytes < 0 {
		return DeriveSMSThreadsResult{}, permanent(errors.New("derive sms threads: max_chunk_bytes cannot be negative"))
	}
	store, err := a.Stores(scheme)
	if err != nil {
		return DeriveSMSThreadsResult{}, permanent(fmt.Errorf("derive sms threads: object store scheme %q: %w", scheme, err))
	}

	// A retried Activity must not fail because its first attempt finished:
	// Derive refuses to overwrite a published manifest, so return that one.
	manifestKey := key + smsthreads.DerivedSuffix + "/" + smsthreads.ManifestName
	if done, existsErr := store.Exists(ctx, bucket, manifestKey); existsErr != nil {
		return DeriveSMSThreadsResult{}, fmt.Errorf("derive sms threads: check existing manifest: %w", existsErr)
	} else if done {
		manifest, readErr := readSMSThreadsManifest(ctx, store, bucket, manifestKey)
		if readErr != nil {
			return DeriveSMSThreadsResult{}, readErr
		}
		return smsThreadsResult(scheme, bucket, manifestKey, manifest, true), nil
	}

	stop := a.heartbeatWhile(ctx, req.SourceURI)
	manifest, err := smsthreads.Derive(ctx, smsthreads.Options{
		Store: store, Scheme: scheme, Bucket: bucket, Key: key,
		ScratchRoot: a.ScratchRoot, MaxChunk: req.MaxChunkBytes,
	})
	stop()
	if err != nil {
		return DeriveSMSThreadsResult{}, fmt.Errorf("derive sms threads: %w", err)
	}
	return smsThreadsResult(scheme, bucket, manifestKey, manifest, false), nil
}

func (a DeriveSMSThreadsActivities) heartbeatWhile(ctx context.Context, detail string) func() {
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
				a.Heartbeat(ctx, detail)
			}
		}
	}()
	return func() { close(done) }
}

func readSMSThreadsManifest(ctx context.Context, store smsthreads.ObjectStore, bucket, key string) (smsthreads.Manifest, error) {
	body, err := store.Open(ctx, bucket, key)
	if err != nil {
		return smsthreads.Manifest{}, fmt.Errorf("derive sms threads: open existing manifest: %w", err)
	}
	defer body.Close()
	var manifest smsthreads.Manifest
	if err := json.NewDecoder(io.LimitReader(body, 64<<20)).Decode(&manifest); err != nil {
		return smsthreads.Manifest{}, permanent(fmt.Errorf("derive sms threads: existing manifest is not valid JSON: %w", err))
	}
	return manifest, nil
}

func smsThreadsResult(scheme, bucket, manifestKey string, manifest smsthreads.Manifest, already bool) DeriveSMSThreadsResult {
	return DeriveSMSThreadsResult{
		ManifestURI:  scheme + "://" + bucket + "/" + manifestKey,
		SourceSHA256: manifest.SourceSHA256, SourceBytes: manifest.SourceBytes,
		Records: manifest.Records, Rejected: manifest.Rejected,
		ThreadChunks: len(manifest.Threads), MediaObjects: manifest.MediaObjects,
		AlreadyDerived: already,
	}
}
