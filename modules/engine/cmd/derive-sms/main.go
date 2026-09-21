// Command derive-sms runs derive/smsthreads on one object: it streams an SMS
// Backup & Restore XML from a configured object store and publishes per-thread
// NDJSON plus decoded media beside it. The same unit is what a Temporal
// Activity calls; this is the direct call shape.
//
// Byline: Claude Code · Fable 5.1 · 2026-09-20
//
//	derive-sms -source b2://salem-data/path/sms-2026.xml -scratch /data/scratch
//
// Credentials come from OBJECT_STORES_JSON (scheme -> credential file), the
// same configuration the worker and gateway read. Nothing secret is printed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
)

func main() {
	source := flag.String("source", "", "object URI, e.g. b2://bucket/key.xml")
	scratch := flag.String("scratch", "", "absolute scratch directory on a data volume")
	validate := flag.Bool("validate", false, "check a published derivation against its manifest instead of deriving")
	maxChunk := flag.Int64("max-chunk", 0, "bytes per thread chunk (0 = default 64 MiB)")
	flag.Parse()
	if err := run(*source, *scratch, *validate, *maxChunk); err != nil {
		fmt.Fprintln(os.Stderr, "derive-sms:", err)
		os.Exit(1)
	}
}

func run(source, scratch string, validate bool, maxChunk int64) error {
	scheme, rest, found := strings.Cut(source, "://")
	bucket, key, hasKey := strings.Cut(rest, "/")
	if !found || !hasKey || bucket == "" || key == "" {
		return fmt.Errorf("source must be <scheme>://<bucket>/<key>")
	}
	stores, err := objectstores.StoresFromEnv()
	if err != nil {
		return err
	}
	credentialFile, ok := stores[scheme]
	if !ok {
		return fmt.Errorf("scheme %q is not in OBJECT_STORES_JSON (configured: %v)", scheme, stores.Schemes())
	}
	cfg, err := acquisition.LoadObjectStorageConfigFile(credentialFile)
	if err != nil {
		return err
	}
	client, err := acquisition.NewS3Client(cfg)
	if err != nil {
		return err
	}
	// Placement is configuration (DERIVED_ROOTS_JSON), so the CLI resolves it
	// exactly as the Activity does and the two can never disagree.
	// Byline: Claude Code · Opus 5 · 2026-09-21
	roots, err := smsthreads.DerivedRootsFromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if validate {
		report, err := smsthreads.Validate(ctx, smsthreads.S3Store{Client: client}, roots, scheme, bucket, key, maxChunk)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			return err
		}
		if !report.OK {
			return fmt.Errorf("derivation failed validation: %d problems, %d oversize chunks", len(report.Problems), len(report.OversizeChunks))
		}
		return nil
	}
	manifest, location, err := smsthreads.Derive(ctx, smsthreads.Options{
		Store: smsthreads.S3Store{Client: client}, Scheme: scheme, Bucket: bucket, Key: key,
		ScratchRoot: scratch, MaxChunk: maxChunk, DerivedRoots: roots,
	})
	if err != nil {
		return err
	}
	summary := map[string]any{
		"source": manifest.Source, "source_sha256": manifest.SourceSHA256, "source_bytes": manifest.SourceBytes,
		"derived_prefix": manifest.DerivedPrefix, "derived_root_mapped": location.Mapped,
		"threads_prefix": location.URI() + smsthreads.ThreadsDir,
		"records":        manifest.Records, "rejected": manifest.Rejected,
		"threads": len(manifest.Threads), "media_objects": manifest.MediaObjects, "media_bytes": manifest.MediaBytes,
	}
	if !location.Mapped {
		fmt.Fprintf(os.Stderr,
			"derive-sms: no %s pair matched %s; published beside the original at %s\n",
			smsthreads.DerivedRootsEnv, manifest.Source, location.URI())
	}
	return json.NewEncoder(os.Stdout).Encode(summary)
}
