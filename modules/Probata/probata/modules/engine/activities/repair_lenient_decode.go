// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// repair.lenient_decode — one job: run the SBV SMS Backup & Restore decoder in
// lenient mode and publish derived per-thread NDJSON plus a manifest. It is
// derive/smsthreads with Lenient set, not a second decoder: records the
// decoder refuses are set aside as rejects exactly as in a strict run, and a
// decode that stops on damaged bytes keeps everything before the stop. The
// output lives under <derived location>/lenient/, so it can never be mistaken
// for, or reused as, a strict derivation. The original is never written.

package activities

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// RepairLenientDecodeActivity implements repair.lenient_decode.
type RepairLenientDecodeActivity struct {
	Stores       func(scheme string) (smsthreads.ObjectStore, error)
	DerivedRoots smsthreads.DerivedRoots
	// Roots are the admissible source roots: the derived threads must lie
	// inside one or they could never re-enter Proffer.
	Roots       objectstores.Roots
	ScratchRoot string
	MaxChunk    int64
	Heartbeat   Heartbeat
}

type lenientSummary struct {
	Records      uint64 `json:"records"`
	Rejected     uint64 `json:"rejected"`
	MediaObjects uint64 `json:"media_objects"`
	ChunkCount   int    `json:"chunk_count"`
	ThreadCount  int    `json:"thread_count"`
	StreamError  string `json:"stream_error,omitempty"`
	DecodedBytes int64  `json:"decoded_bytes,omitempty"`
	SourceBytes  int64  `json:"source_bytes"`
	SourceSHA256 string `json:"source_sha256"`
	ThreadsRef   string `json:"threads_ref"`
	Reused       bool   `json:"reused"`
}

// LenientDecode publishes a lenient derivation of the source.
func (a RepairLenientDecodeActivity) LenientDecode(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	result, err := a.decode(ctx, request)
	return result, stopRetryingPermanent(err)
}

func (a RepairLenientDecodeActivity) decode(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	var params struct{}
	if err := decodeStepParams(request.Params, &params); err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	if request.SourceType != repairplan.TypeSMSBackupXML {
		return repairplan.StepResult{}, permanent(fmt.Errorf("lenient decode reads an SMS backup; the source is %q", request.SourceType))
	}
	scratchRoot := strings.TrimSpace(a.ScratchRoot)
	if a.Stores == nil || (!filepath.IsAbs(scratchRoot) && !strings.HasPrefix(scratchRoot, "/")) {
		return repairplan.StepResult{}, errors.New("lenient decode: object stores and an absolute scratch root on a data volume are required")
	}
	source, err := parseRepairLocator(request.SourceRef)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	store, err := a.Stores(source.Scheme)
	if err != nil {
		return repairplan.StepResult{}, permanent(fmt.Errorf("object store %q: %w", source.Scheme, err))
	}
	target, err := a.DerivedRoots.Locate(source.Scheme, source.Bucket, source.Key)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	target = target.WithVariant(repairplan.LenientVariant)
	if _, admitted := a.Roots.Match(target.Scheme, target.Bucket, target.Prefix+smsthreads.ThreadsDir+"x"); !admitted {
		return repairplan.StepResult{}, permanent(fmt.Errorf("lenient decode refused: %s is outside every configured source root, so its threads could not re-enter Proffer", target.URI()))
	}
	_, reused, err := smsthreads.ResolvePublishedVariant(ctx, store, a.DerivedRoots, repairplan.LenientVariant, source.Scheme, source.Bucket, source.Key)
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("check for a published lenient derivation: %w", err)
	}
	stop := startIdleHeartbeat(ctx, a.Heartbeat, stagegraph.RepairLenientDecode, 0)
	_, published, err := smsthreads.Derive(ctx, smsthreads.Options{
		Store: store, Scheme: source.Scheme, Bucket: source.Bucket, Key: source.Key,
		ScratchRoot: scratchRoot, MaxChunk: a.MaxChunk, DerivedRoots: a.DerivedRoots,
		ReuseExisting: true, Lenient: true, Variant: repairplan.LenientVariant,
		Progress: a.progress(ctx),
	})
	stop()
	if err != nil {
		if errors.Is(err, smsthreads.ErrNoRecords) {
			return repairplan.StepResult{}, permanent(fmt.Errorf("lenient decode of %s: %w", source.URI(), err))
		}
		return repairplan.StepResult{}, fmt.Errorf("lenient decode of %s: %w", source.URI(), err)
	}
	// The digest recorded is of the manifest bytes the store actually serves.
	manifest, digest, err := smsthreads.LoadManifestWithDigest(ctx, store, published.Bucket, published.ManifestKey())
	if err != nil {
		return repairplan.StepResult{}, err
	}
	if manifest.Records == 0 {
		return repairplan.StepResult{}, permanent(fmt.Errorf("lenient decode of %s kept no readable record (%d rejected)", source.URI(), manifest.Rejected))
	}
	if !manifest.Lenient || manifest.Variant != repairplan.LenientVariant {
		return repairplan.StepResult{}, permanent(fmt.Errorf("the derivation at %s is not a lenient one", published.URI()))
	}
	threads := map[string]bool{}
	for _, file := range manifest.Threads {
		threads[file.Thread] = true
	}
	threadsRef := published.URI() + smsthreads.ThreadsDir
	return repairplan.StepResult{
		OutputRef: published.URI() + smsthreads.ManifestName, OutputType: repairplan.TypeDerivedThreads,
		OutputKind: repairplan.OutputDerivedChunkFolder, OutputSHA256: digest, ReentryRef: threadsRef, Reused: reused,
		Summary: repairSummary(lenientSummary{
			Records: manifest.Records, Rejected: manifest.Rejected, MediaObjects: manifest.MediaObjects,
			ChunkCount: len(manifest.Threads), ThreadCount: len(threads), StreamError: manifest.StreamError,
			DecodedBytes: manifest.DecodedBytes, SourceBytes: manifest.SourceBytes, SourceSHA256: manifest.SourceSHA256,
			ThreadsRef: threadsRef, Reused: reused,
		}),
	}, nil
}

func (a RepairLenientDecodeActivity) progress(ctx context.Context) func(smsthreads.Progress) {
	if a.Heartbeat == nil {
		return nil
	}
	return func(progress smsthreads.Progress) {
		members := int64(progress.Records + progress.Rejected)
		if progress.Phase == "publishing" {
			members = int64(progress.PublishedObjects)
		}
		a.Heartbeat(ctx, Progress{Stage: stagegraph.RepairLenientDecode, MembersComplete: members, BytesComplete: progress.SourceBytes})
	}
}
