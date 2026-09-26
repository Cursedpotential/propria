// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// repair.salvage_truncated_xml — one job: stream a cut-off XML backup once,
// keep every record up to the last complete one, close the document, and
// publish that as a NEW derived object with its sha256 (derive/xmlsalvage
// does the measuring). The original is never written: the output lives under
// the source's configured derived location, in its own salvaged/ folder.
//
// The local scratch copy exists only while the salvage needs it and is
// removed on every path (owner 2026-09-22: a local copy only while a repair
// needs it). A finished salvage is recognised by its manifest, written last,
// so a retried Activity returns the published result instead of re-streaming
// a multi-gigabyte source — unless the source changed, which is refused
// rather than silently overwriting the published copy.

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/derive/xmlsalvage"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/repairplan"
	"github.com/Cursedpotential/probata/engine/stagegraph"
)

// SalvageManifestSchema versions the salvage manifest.
const SalvageManifestSchema = "xmlsalvage/v1"

const maxSalvageManifestBytes = 1 << 20

// SalvageManifest is written beside the salvaged object, last.
type SalvageManifest struct {
	Schema        string `json:"schema"`
	Source        string `json:"source"`
	SourceSHA256  string `json:"source_sha256"`
	SourceBytes   int64  `json:"source_bytes"`
	SourceETag    string `json:"source_etag,omitempty"`
	Derived       string `json:"derived"`
	DerivedSHA256 string `json:"derived_sha256"`
	DerivedBytes  int64  `json:"derived_bytes"`
	Root          string `json:"root"`
	RecordsKept   uint64 `json:"records_kept"`
	ClaimedCount  int64  `json:"claimed_count"`
	BytesDropped  int64  `json:"bytes_dropped"`
	Truncated     bool   `json:"truncated"`
	Stopped       string `json:"stopped,omitempty"`
	ClosingTag    string `json:"closing_tag"`
	SalvagedAt    string `json:"salvaged_at"`
}

// RepairSalvageTruncatedXMLActivity implements repair.salvage_truncated_xml.
type RepairSalvageTruncatedXMLActivity struct {
	// Stores resolves a configured scheme; the store must also HEAD and
	// upload large objects (smsthreads.S3Store does both).
	Stores       func(scheme string) (smsthreads.ObjectStore, error)
	DerivedRoots smsthreads.DerivedRoots
	// Roots are the admissible source roots: the salvaged copy must lie
	// inside one or it could never re-enter Proffer.
	Roots       objectstores.Roots
	ScratchRoot string
	Heartbeat   Heartbeat
	Now         func() time.Time
}

type salvageStore interface {
	smsthreads.ObjectStore
	smsthreads.ObjectStatter
	smsthreads.LargeObjectPutter
}

type salvageSummary struct {
	RecordsKept  uint64 `json:"records_kept"`
	ClaimedCount int64  `json:"claimed_count"`
	BytesDropped int64  `json:"bytes_dropped"`
	SourceBytes  int64  `json:"source_bytes"`
	DerivedBytes int64  `json:"derived_bytes"`
	Truncated    bool   `json:"truncated"`
	Root         string `json:"root"`
	Stopped      string `json:"stopped,omitempty"`
	SourceSHA256 string `json:"source_sha256"`
	ManifestRef  string `json:"manifest_ref"`
	Reused       bool   `json:"reused"`
}

// SalvageTruncatedXML publishes a salvaged derived copy of the source.
func (a RepairSalvageTruncatedXMLActivity) SalvageTruncatedXML(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	result, err := a.salvage(ctx, request)
	return result, stopRetryingPermanent(err)
}

// SalvageTarget is where a salvage of source publishes: the salvaged object
// and its manifest, under the source's derived location.
func SalvageTarget(roots smsthreads.DerivedRoots, scheme, bucket, key string) (smsthreads.DerivedLocation, string, string, error) {
	location, err := roots.Locate(scheme, bucket, key)
	if err != nil {
		return smsthreads.DerivedLocation{}, "", "", err
	}
	derivedKey := location.Prefix + repairplan.SalvagedSubfolder + path.Base(key)
	return location, derivedKey, derivedKey + ".salvage.json", nil
}

func (a RepairSalvageTruncatedXMLActivity) salvage(ctx context.Context, request repairplan.StepRequest) (repairplan.StepResult, error) {
	var params struct{}
	if err := decodeStepParams(request.Params, &params); err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	if request.SourceType != repairplan.TypeSMSBackupXML && request.SourceType != repairplan.TypeXML {
		return repairplan.StepResult{}, permanent(fmt.Errorf("salvage reads XML; the source is %q", request.SourceType))
	}
	scratchRoot := strings.TrimSpace(a.ScratchRoot)
	if a.Stores == nil || (!filepath.IsAbs(scratchRoot) && !strings.HasPrefix(scratchRoot, "/")) {
		return repairplan.StepResult{}, errors.New("salvage: object stores and an absolute scratch root on a data volume are required")
	}
	source, err := parseRepairLocator(request.SourceRef)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	store, err := a.store(source.Scheme)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	location, derivedKey, manifestKey, err := SalvageTarget(a.DerivedRoots, source.Scheme, source.Bucket, source.Key)
	if err != nil {
		return repairplan.StepResult{}, permanent(err)
	}
	if location.Bucket == source.Bucket && (derivedKey == source.Key || manifestKey == source.Key) {
		return repairplan.StepResult{}, permanent(errors.New("salvage refused: the derived location would overwrite the original"))
	}
	if _, admitted := a.Roots.Match(location.Scheme, location.Bucket, derivedKey); !admitted {
		return repairplan.StepResult{}, permanent(fmt.Errorf("salvage refused: %s://%s/%s is outside every configured source root, so it could not re-enter Proffer",
			location.Scheme, location.Bucket, derivedKey))
	}
	derivedRef := repairLocator{Scheme: location.Scheme, Bucket: location.Bucket, Key: derivedKey}
	manifestRef := repairLocator{Scheme: location.Scheme, Bucket: location.Bucket, Key: manifestKey}

	sourceInfo, exists, err := store.Stat(ctx, source.Bucket, source.Key)
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("stat the source %s: %w", source.URI(), err)
	}
	if !exists {
		return repairplan.StepResult{}, permanent(fmt.Errorf("the source %s does not exist", source.URI()))
	}

	// A finished salvage is reused, never re-streamed or overwritten.
	if published, err := store.Exists(ctx, location.Bucket, manifestKey); err != nil {
		return repairplan.StepResult{}, fmt.Errorf("check for a published salvage: %w", err)
	} else if published {
		manifest, err := loadSalvageManifest(ctx, store, location.Bucket, manifestKey)
		if err != nil {
			return repairplan.StepResult{}, err
		}
		if manifest.Source != source.URI() || manifest.SourceBytes != sourceInfo.Size ||
			(manifest.SourceETag != "" && sourceInfo.ETag != "" && manifest.SourceETag != sourceInfo.ETag) {
			return repairplan.StepResult{}, permanent(fmt.Errorf(
				"salvage refused: %s changed since its salvage was published at %s; a new salvage would overwrite it",
				source.URI(), derivedRef.URI()))
		}
		return salvageResult(request, derivedRef, manifestRef, manifest, true), nil
	}

	scratch, err := os.MkdirTemp(scratchRoot, "salvage-")
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("salvage: create scratch: %w", err)
	}
	defer os.RemoveAll(scratch)
	file, err := os.Create(filepath.Join(scratch, "salvaged.xml"))
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("salvage: create scratch file: %w", err)
	}
	defer file.Close()

	body, err := store.Open(ctx, source.Bucket, source.Key)
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("open the source %s: %w", source.URI(), err)
	}
	stop := startIdleHeartbeat(ctx, a.Heartbeat, stagegraph.RepairSalvageTruncatedXML, 0)
	outcome, err := xmlsalvage.Salvage(ctx, body, file, func(read int64) {
		if a.Heartbeat != nil {
			a.Heartbeat(ctx, Progress{Stage: stagegraph.RepairSalvageTruncatedXML, BytesComplete: read})
		}
	})
	body.Close()
	if err != nil {
		stop()
		if errors.Is(err, xmlsalvage.ErrNothingToSalvage) {
			return repairplan.StepResult{}, permanent(fmt.Errorf("salvage of %s: %w", source.URI(), err))
		}
		return repairplan.StepResult{}, err
	}
	if outcome.SourceBytes != sourceInfo.Size {
		stop()
		return repairplan.StepResult{}, fmt.Errorf("salvage: read %d bytes of %s but the store reports %d; the source changed while it was read",
			outcome.SourceBytes, source.URI(), sourceInfo.Size)
	}
	err = store.PutLarge(ctx, location.Bucket, derivedKey, file, outcome.DerivedBytes, "application/xml")
	stop()
	if err != nil {
		return repairplan.StepResult{}, fmt.Errorf("publish the salvaged copy %s: %w", derivedRef.URI(), err)
	}
	// Read back what the store now holds before recording it as done.
	if info, exists, err := store.Stat(ctx, location.Bucket, derivedKey); err != nil {
		return repairplan.StepResult{}, fmt.Errorf("confirm the salvaged copy: %w", err)
	} else if !exists || info.Size != outcome.DerivedBytes {
		return repairplan.StepResult{}, fmt.Errorf("the salvaged copy %s did not land intact (%d of %d bytes)", derivedRef.URI(), info.Size, outcome.DerivedBytes)
	}
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}
	manifest := SalvageManifest{
		Schema: SalvageManifestSchema, Source: source.URI(), SourceSHA256: outcome.SourceSHA256,
		SourceBytes: outcome.SourceBytes, SourceETag: sourceInfo.ETag,
		Derived: derivedRef.URI(), DerivedSHA256: outcome.DerivedSHA256, DerivedBytes: outcome.DerivedBytes,
		Root: outcome.Root, RecordsKept: outcome.Records, ClaimedCount: outcome.ClaimedCount,
		BytesDropped: outcome.BytesDropped, Truncated: outcome.Truncated(), Stopped: outcome.Stopped,
		ClosingTag: outcome.ClosingTag, SalvagedAt: now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return repairplan.StepResult{}, err
	}
	if err := store.Put(ctx, location.Bucket, manifestKey, bytes.NewReader(encoded), int64(len(encoded)), "application/json"); err != nil {
		return repairplan.StepResult{}, fmt.Errorf("publish the salvage manifest: %w", err)
	}
	return salvageResult(request, derivedRef, manifestRef, manifest, false), nil
}

func salvageResult(request repairplan.StepRequest, derived, manifestRef repairLocator, manifest SalvageManifest, reused bool) repairplan.StepResult {
	return repairplan.StepResult{
		OutputRef: derived.URI(), OutputType: request.SourceType, OutputKind: repairplan.OutputDerivedObject,
		OutputSHA256: manifest.DerivedSHA256, Reused: reused,
		Summary: repairSummary(salvageSummary{
			RecordsKept: manifest.RecordsKept, ClaimedCount: manifest.ClaimedCount, BytesDropped: manifest.BytesDropped,
			SourceBytes: manifest.SourceBytes, DerivedBytes: manifest.DerivedBytes, Truncated: manifest.Truncated,
			Root: manifest.Root, Stopped: manifest.Stopped, SourceSHA256: manifest.SourceSHA256,
			ManifestRef: manifestRef.URI(), Reused: reused,
		}),
	}
}

func loadSalvageManifest(ctx context.Context, store smsthreads.ObjectStore, bucket, key string) (SalvageManifest, error) {
	body, err := store.Open(ctx, bucket, key)
	if err != nil {
		return SalvageManifest{}, fmt.Errorf("open the salvage manifest: %w", err)
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, maxSalvageManifestBytes+1))
	if err != nil {
		return SalvageManifest{}, fmt.Errorf("read the salvage manifest: %w", err)
	}
	if len(raw) > maxSalvageManifestBytes {
		return SalvageManifest{}, permanent(errors.New("the salvage manifest is implausibly large"))
	}
	var manifest SalvageManifest
	if err := json.Unmarshal(raw, &manifest); err != nil || manifest.Schema != SalvageManifestSchema || manifest.DerivedSHA256 == "" {
		return SalvageManifest{}, permanent(fmt.Errorf("the published salvage manifest %s is not a %s manifest", key, SalvageManifestSchema))
	}
	return manifest, nil
}

func (a RepairSalvageTruncatedXMLActivity) store(scheme string) (salvageStore, error) {
	store, err := a.Stores(scheme)
	if err != nil {
		return nil, fmt.Errorf("object store %q: %w", scheme, err)
	}
	capable, ok := store.(salvageStore)
	if !ok {
		return nil, fmt.Errorf("object store %q cannot stat and upload large objects", scheme)
	}
	return capable, nil
}
