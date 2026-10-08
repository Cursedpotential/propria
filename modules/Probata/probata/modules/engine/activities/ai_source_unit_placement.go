// Byline: Codex · GPT-6.1 · 2026-10-07.
package activities

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AISourceUnitPlacementWorkflowName       = "AISourceUnitPlacementWorkflow"
	AISourceUnitInspectActivityName         = "ai_source_unit_inspect_activity"
	AISourceUnitHashSourceActivityName      = "ai_source_unit_hash_source_activity"
	AISourceUnitCopyActivityName            = "ai_source_unit_copy_activity"
	AISourceUnitHashDestinationActivityName = "ai_source_unit_hash_destination_activity"
	AISourceUnitReadbackActivityName        = "ai_source_unit_readback_activity"
	aiSourceUnitPrefix                      = "consignatio/casevault/KnowledgeBase/ai-chats/_Incoming/"
)

// AISourceUnitPlacementInput pins the complete reviewed export manifest and exclusive receipt namespace.
// Inputs: mounted manifest reference/hash and receipt base
// Outputs: five independent stage receipt pins
// Effects: none as data
// Choose: for versioned B2 export units rather than Markdown-only work products or legal packages.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitPlacementInput struct {
	ManifestRef    proffer.Ref `json:"manifest_ref"`
	ManifestSHA256 string      `json:"manifest_sha256"`
	ReceiptRef     proffer.Ref `json:"receipt_ref"`
}

// AISourceUnitManifest preserves every explicitly reviewed member under one neutral account/export boundary.
// Inputs: unit path, exact source prefix and discovery pin; provider must remain unset
// Outputs: unchanged relative-name mapping
// Effects: none
// Choose: complete membership over extension filters, merges or inferred archive closure.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitManifest struct {
	Provider         string             `json:"provider,omitempty"`
	Unit             string             `json:"unit"`
	Bucket           string             `json:"bucket"`
	SourcePrefix     string             `json:"source_prefix"`
	ProvenanceRef    proffer.Ref        `json:"provenance_ref"`
	ProvenanceSHA256 string             `json:"provenance_sha256"`
	Files            []AISourceUnitFile `json:"files"`
}

// AISourceUnitFile binds one unchanged member to its exact original retained version.
// Inputs: original key/version, SHA1, size and optional independently known SHA256
// Outputs: membership identity
// Effects: none
// Choose: provider-version identity over latest-key reads or filename-based deduplication.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitFile struct {
	Key       string `json:"key"`
	VersionID string `json:"version_id"`
	Bytes     int64  `json:"bytes"`
	SHA1      string `json:"sha1"`
	SHA256    string `json:"sha256,omitempty"`
}

// AISourceUnitStageInput carries only the request and predecessor receipt pin through Temporal.
// Inputs: original request and previous stage reference/hash
// Outputs: bounded stage request
// Effects: none
// Choose: references instead of source bodies or full manifests in workflow history.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitStageInput struct {
	Input          AISourceUnitPlacementInput `json:"input"`
	PreviousRef    proffer.Ref                `json:"previous_ref,omitempty"`
	PreviousSHA256 string                     `json:"previous_sha256,omitempty"`
}

// AISourceUnitSummary distinguishes stage completion from final physical placement verification.
// Inputs: durable stage receipt
// Outputs: pin, count, bytes and final proof state
// Effects: none
// Choose: over process-exit or partial-member completion claims.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitSummary struct {
	ReceiptRef                proffer.Ref `json:"receipt_ref"`
	ReceiptSHA256             string      `json:"receipt_sha256"`
	Objects                   int         `json:"objects"`
	Bytes                     int64       `json:"bytes"`
	Complete                  bool        `json:"complete"`
	PhysicalPlacementVerified bool        `json:"physical_placement_verified"`
}

// AISourceUnitObject retains each source version, independent hashes and actual returned copy version.
// Inputs: reviewed member and stage observations
// Outputs: bounded provenance and conflict evidence
// Effects: none
// Choose: before downstream ingestion/catalog admission without declaring package usability.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitObject struct {
	File               AISourceUnitFile     `json:"file"`
	DestinationKey     string               `json:"destination_key"`
	SourceMetadata     AISourceUnitMetadata `json:"source_metadata"`
	SourceSHA1         string               `json:"source_sha1,omitempty"`
	SourceSHA256       string               `json:"source_sha256,omitempty"`
	DestinationVersion string               `json:"destination_version,omitempty"`
	DestinationSHA1    string               `json:"destination_sha1,omitempty"`
	DestinationSHA256  string               `json:"destination_sha256,omitempty"`
	DestinationBytes   int64                `json:"destination_bytes,omitempty"`
	BeforeVersions     []string             `json:"before_versions"`
	AfterVersions      []string             `json:"after_versions"`
	Status             string               `json:"status"`
	Error              string               `json:"error,omitempty"`
}

// AISourceUnitReceipt records the entire frozen mapping and an explicit all-member stage outcome.
// Inputs: pinned request, manifest and observations
// Outputs: retained physical evidence
// Effects: none as data
// Choose: without adding catalog tables or claiming ingestion or ZIP/attachment closure.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitReceipt struct {
	Stage                     string                 `json:"stage"`
	Request                   AISourceUnitStageInput `json:"request"`
	Manifest                  AISourceUnitManifest   `json:"manifest"`
	Objects                   []AISourceUnitObject   `json:"objects"`
	Complete                  bool                   `json:"complete"`
	PhysicalPlacementVerified bool                   `json:"physical_placement_verified"`
	Error                     string                 `json:"error,omitempty"`
	PackageCompleteness       string                 `json:"package_completeness"`
	IngestionStatus           string                 `json:"ingestion_status"`
	CatalogStatus             string                 `json:"catalog_status"`
}

// AISourceUnitPlacementActivities uses the existing resolver and mounted scratch receipt root.
// Inputs: allowed root, resolver and optional heartbeat
// Outputs: separately callable stage operations
// Effects: method-specific B2 reads/copies and retained receipts
// Choose: without new authentication, workers or stores.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type AISourceUnitPlacementActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewAISourceUnitPlacementActivities constructs the source-export group without opening stores.
// Inputs: existing DeriveScratchDir and objectstores resolver
// Outputs: registration-ready group
// Effects: none until invocation
// Choose: beside the unchanged Markdown/legal placement groups.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func NewAISourceUnitPlacementActivities(root string, stores func(string) (smsthreads.ObjectStore, error)) *AISourceUnitPlacementActivities {
	return &AISourceUnitPlacementActivities{AllowedRoot: root, Stores: stores, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// AISourceUnitPlacementWorkflow sequences metadata admission, source hashing, server copy, destination hashing and final readback.
// Inputs: frozen complete-member manifest pin and receipt base
// Outputs: all-member physical receipt pin
// Effects: five independently tracked Activities, no payloads in history
// Choose: complete AI source units without extending Markdown-only placement or performing ingestion.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func AISourceUnitPlacementWorkflow(ctx workflow.Context, in AISourceUnitPlacementInput) (AISourceUnitSummary, error) {
	if !validSHA256(in.ManifestSHA256) || in.ManifestRef == "" || in.ReceiptRef == "" || in.ManifestRef == in.ReceiptRef {
		return AISourceUnitSummary{}, temporal.NewNonRetryableApplicationError("invalid manifest/receipt request", "AISourceUnitInvalid", nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 20 * time.Minute, ScheduleToCloseTimeout: time.Hour, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2}})
	req := AISourceUnitStageInput{Input: in}
	var out AISourceUnitSummary
	for _, name := range []string{AISourceUnitInspectActivityName, AISourceUnitHashSourceActivityName, AISourceUnitCopyActivityName, AISourceUnitHashDestinationActivityName, AISourceUnitReadbackActivityName} {
		if err := workflow.ExecuteActivity(ctx, name, req).Get(ctx, &out); err != nil {
			return out, err
		}
		if !out.Complete || !validSHA256(out.ReceiptSHA256) || out.Objects < 1 {
			return out, temporal.NewNonRetryableApplicationError("stage lacks complete pinned member receipt", "AISourceUnitIncomplete", nil)
		}
		req.PreviousRef, req.PreviousSHA256 = out.ReceiptRef, out.ReceiptSHA256
	}
	if !out.PhysicalPlacementVerified {
		return out, temporal.NewNonRetryableApplicationError("unit placement unverified", "AISourceUnitIncomplete", nil)
	}
	return out, nil
}

// InspectAISourceUnit admits every exact source-version metadata record and the complete discovery membership.
// Inputs: frozen manifest/provenance pins
// Outputs: exclusive admitted receipt
// Effects: metadata reads only and receipt creation
// Choose: before separately scheduled source hashing, never as byte proof.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) InspectAISourceUnit(ctx context.Context, in AISourceUnitStageInput) (AISourceUnitSummary, error) {
	return a.sourceUnitStage(ctx, "inspect", in)
}

// HashAISourceUnitSource independently streams all original pinned versions through EOF.
// Inputs: admitted metadata receipt
// Outputs: SHA1/SHA256 and exact sizes for all members
// Effects: read-only B2 GETs and retained receipt
// Choose: to establish source bytes before the separate server-copy Activity.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) HashAISourceUnitSource(ctx context.Context, in AISourceUnitStageInput) (AISourceUnitSummary, error) {
	return a.sourceUnitStage(ctx, "hash-source", in)
}

// CopyAISourceUnit preserves every admitted source version by same-bucket server-side copy.
// Inputs: complete source-hash receipt
// Outputs: actual destination versions and per-object checkpoints
// Effects: bounded CopyObject operations only plus metadata/receipt writes, no body hashing or transfer through the worker
// Choose: before independent destination hashing and refuse uncertain replays.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) CopyAISourceUnit(ctx context.Context, in AISourceUnitStageInput) (AISourceUnitSummary, error) {
	return a.sourceUnitStage(ctx, "copy", in)
}

// HashAISourceUnitDestination independently hashes each returned destination version against the complete source hashes.
// Inputs: pinned all-member copy receipt
// Outputs: full-size SHA1/SHA256 comparisons
// Effects: read-only exact-version B2 GETs and retained receipt
// Choose: over HEAD, ETag, API checksum or successful-copy response proof.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) HashAISourceUnitDestination(ctx context.Context, in AISourceUnitStageInput) (AISourceUnitSummary, error) {
	return a.sourceUnitStage(ctx, "hash-destination", in)
}

// ReadbackAISourceUnit verifies complete visible unit membership and unchanged source/destination version metadata.
// Inputs: complete destination-hash receipt
// Outputs: final all-member physical placement receipt
// Effects: metadata reads and retained receipt only
// Choose: before downstream ingestion/catalog, leaving ZIP members and attachment closure unresolved.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) ReadbackAISourceUnit(ctx context.Context, in AISourceUnitStageInput) (AISourceUnitSummary, error) {
	return a.sourceUnitStage(ctx, "readback", in)
}

// aiSourceUnitManifestValid bounds and validates unchanged source-to-unit membership.
// Inputs: manifest
// Outputs: error for unsafe names, unpinned members or excessive scope
// Effects: none
// Choose: before opening any remote object.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceUnitManifestValid(m AISourceUnitManifest) error {
	if m.Bucket != "salem-data" || m.Provider != "" || !toolkitPlacementPath(m.Unit) || len(m.Unit) > 256 || !strings.HasPrefix(m.SourcePrefix, "consignatio/vault/v1/") || !strings.HasSuffix(m.SourcePrefix, "/") || !toolkitPlacementPath(strings.TrimSuffix(m.SourcePrefix, "/")) || !validSHA256(m.ProvenanceSHA256) || m.ProvenanceRef == "" || len(m.Files) < 1 || len(m.Files) > 64 {
		return errors.New("invalid bounded AI source unit")
	}
	seen := map[string]bool{}
	var total int64
	for _, f := range m.Files {
		relative := strings.TrimPrefix(f.Key, m.SourcePrefix)
		if relative == f.Key || !toolkitPlacementPath(relative) || len(f.Key) > 1024 || !validToolkitVersionID(f.VersionID) || !aiSourceHex(f.SHA1, 40) || (f.SHA256 != "" && !validSHA256(f.SHA256)) || f.Bytes < 0 || f.Bytes > 32<<20 || seen[strings.ToLower(relative)] {
			return errors.New("invalid or duplicate pinned source member")
		}
		seen[strings.ToLower(relative)] = true
		total += f.Bytes
	}
	if total > 256<<20 {
		return errors.New("source unit exceeds total byte budget")
	}
	return nil
}

// aiSourceHex validates canonical lower-case digest metadata.
// Inputs: text and required length
// Outputs: boolean
// Effects: none
// Choose: before comparison with streamed hashes.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceHex(s string, n int) bool {
	if len(s) != n || strings.ToLower(s) != s {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}

// aiSourceDestinationPrefix constructs the existing neutral AI intake namespace without renaming members.
// Inputs: validated manifest
// Outputs: fixed AI-home unit prefix
// Effects: none
// Choose: over the workproduct miscellaneous destination.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceDestinationPrefix(m AISourceUnitManifest) string {
	return aiSourceUnitPrefix + m.Unit + "/"
}

// aiSourceProvenance verifies that the placement manifest contains every frozen discovery member exactly once.
// Inputs: provenance bytes and validated mapping
// Outputs: admission error or exact membership proof
// Effects: none
// Choose: without treating a loose listing as ZIP or attachment closure.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceProvenance(raw json.RawMessage, m AISourceUnitManifest) error {
	var p struct {
		Bucket       string `json:"bucket"`
		Provider     string `json:"provider"`
		SourcePrefix string `json:"source_prefix"`
		ObjectCount  int    `json:"object_count"`
		ObjectBytes  int64  `json:"object_bytes"`
		Objects      []struct {
			Bucket  string `json:"bucket"`
			Key     string `json:"key"`
			Version string `json:"object_version_id"`
			Bytes   int64  `json:"size"`
			SHA1    string `json:"sha1"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.Bucket != m.Bucket || p.Provider != "b2" || p.SourcePrefix != m.SourcePrefix || p.ObjectCount != len(m.Files) || len(p.Objects) != len(m.Files) {
		return errors.New("discovery membership differs from reviewed unit")
	}
	var total int64
	for i, f := range m.Files {
		v := p.Objects[i]
		if v.Bucket != m.Bucket || v.Key != f.Key || v.Version != f.VersionID || v.Bytes != f.Bytes || v.SHA1 != f.SHA1 {
			return errors.New("discovery exact-version member mismatch")
		}
		total += f.Bytes
	}
	if total != p.ObjectBytes {
		return errors.New("discovery byte accounting mismatch")
	}
	return nil
}

// aiSourceDigest streams one bounded exact version and computes independent SHA1 and SHA256.
// Inputs: stream, exact byte budget and liveness callback
// Outputs: digests and count
// Effects: stream reads only
// Choose: exclusively inside the separate hash Activities.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceDigest(r io.Reader, size int64, pulse func() error) (string, string, int64, error) {
	a, b := sha1.New(), sha256.New()
	buf := make([]byte, 256<<10)
	var n int64
	for {
		if e := pulse(); e != nil {
			return "", "", n, e
		}
		k, e := r.Read(buf)
		if k > 0 {
			n += int64(k)
			if n > size {
				return "", "", n, errors.New("body exceeds pinned size")
			}
			_, _ = a.Write(buf[:k])
			_, _ = b.Write(buf[:k])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", "", n, e
		}
	}
	if n != size {
		return "", "", n, errors.New("incomplete exact-version body")
	}
	return fmt.Sprintf("%x", a.Sum(nil)), fmt.Sprintf("%x", b.Sum(nil)), n, nil
}

// aiSourceUnitKeys compares the whole bounded visible destination listing with the requested membership.
// Inputs: store, manifest and empty/final mode
// Outputs: error unless membership is exact
// Effects: one metadata listing
// Choose: before copying or final unit admission to prevent merging unreviewed objects.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceUnitKeys(ctx context.Context, s aiSourceUnitStore, m AISourceUnitManifest, empty bool) error {
	got, e := s.UnitKeys(ctx, m.Bucket, aiSourceDestinationPrefix(m))
	if e != nil {
		return e
	}
	want := []string{}
	if !empty {
		for _, f := range m.Files {
			want = append(want, aiSourceDestinationPrefix(m)+strings.TrimPrefix(f.Key, m.SourcePrefix))
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		return errors.New("destination unit contains missing, occupied or unreviewed members")
	}
	return nil
}

// aiSourceOriginalKeys compares current source membership against the frozen reviewed discovery.
// Inputs: store and authenticated manifest
// Outputs: error on missing or newly added source keys
// Effects: one bounded source metadata listing
// Choose: before transport and final readback so complete-unit scope stays visible.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceOriginalKeys(ctx context.Context, s aiSourceUnitStore, m AISourceUnitManifest) error {
	got, e := s.SourceKeys(ctx, m.Bucket, m.SourcePrefix)
	if e != nil {
		return e
	}
	want := make([]string, 0, len(m.Files))
	for _, f := range m.Files {
		want = append(want, f.Key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		return errors.New("source unit current membership differs from frozen discovery")
	}
	return nil
}

// aiSourceObserve rechecks exact source metadata and, when requested, the copied visible version and race observations.
// Inputs: pinned object and verification mode
// Outputs: error for any intervening change
// Effects: metadata reads only
// Choose: independently from body hashing.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func aiSourceObserve(ctx context.Context, s aiSourceUnitStore, bucket string, obj AISourceUnitObject, destination bool) error {
	m, e := s.HeadExact(ctx, bucket, obj.File.Key, obj.File.VersionID)
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(m, obj.SourceMetadata) {
		return errors.New("source version metadata changed")
	}
	if !destination {
		return nil
	}
	if !validToolkitVersionID(obj.DestinationVersion) {
		return errors.New("destination version absent")
	}
	v, e := s.HeadVersion(ctx, bucket, obj.DestinationKey)
	if e != nil {
		return e
	}
	versions, e := s.PlacementVersions(ctx, bucket, obj.DestinationKey)
	if e != nil {
		return e
	}
	if e = toolkitPlacementConflictChecks(obj.BeforeVersions, versions, obj.DestinationVersion, v.VersionID); e != nil {
		return e
	}
	dm, e := s.HeadExact(ctx, bucket, obj.DestinationKey, obj.DestinationVersion)
	if e != nil {
		return e
	}
	if !aiSourceMetadataPreserved(obj.SourceMetadata, dm) {
		return errors.New("copied native metadata or size differs")
	}
	return nil
}

// sourceUnitStage executes one independent operation and retains partial failures and uncertain attempts.
// Inputs: phase and pinned predecessor chain
// Outputs: exclusive stage receipt summary
// Effects: only the selected phase's reads/copies and retained metadata
// Choose: completed pinned replay over repeating writes, and hold incomplete/uncertain attempts for review.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (a *AISourceUnitPlacementActivities) sourceUnitStage(ctx context.Context, stage string, in AISourceUnitStageInput) (out AISourceUnitSummary, retErr error) {
	root, e := canonicalDirectory(a.AllowedRoot)
	if e != nil {
		return out, e
	}
	if !validSHA256(in.Input.ManifestSHA256) || in.Input.ManifestRef == in.Input.ReceiptRef || in.Input.ReceiptRef == "" {
		return out, errors.New("invalid source-unit request")
	}
	var m AISourceUnitManifest
	if _, e = aiWorkproductReadJSON(ctx, root, in.Input.ManifestRef, in.Input.ManifestSHA256, &m); e != nil {
		return out, e
	}
	if e = aiSourceUnitManifestValid(m); e != nil {
		return out, e
	}
	// A pinned raw discovery receipt is checked in every stage, not merely its filename.
	var raw json.RawMessage
	if _, e = aiWorkproductReadJSON(ctx, root, m.ProvenanceRef, m.ProvenanceSHA256, &raw); e != nil {
		return out, e
	}
	if e = aiSourceProvenance(raw, m); e != nil {
		return out, e
	}
	phases := []string{"inspect", "hash-source", "copy", "hash-destination", "readback"}
	index := -1
	for i, p := range phases {
		if p == stage {
			index = i
		}
	}
	if index < 0 {
		return out, errors.New("invalid independent source-unit phase")
	}
	out.ReceiptRef = proffer.Ref(string(in.Input.ReceiptRef) + "." + stage + ".json")
	// The existing receipt guard confines writes to an existing nonlinked mount parent.
	manifestPath, e := resolveFileRef(in.Input.ManifestRef, root, false)
	if e != nil {
		return out, e
	}
	p, e := aiWorkproductReceiptPath(root, manifestPath, out.ReceiptRef)
	if e != nil {
		return out, e
	}
	var prior AISourceUnitReceipt
	if index == 0 {
		if in.PreviousRef != "" || in.PreviousSHA256 != "" {
			return out, errors.New("inspection cannot have predecessor")
		}
	} else {
		if in.PreviousRef != proffer.Ref(string(in.Input.ReceiptRef)+"."+phases[index-1]+".json") || !validSHA256(in.PreviousSHA256) {
			return out, errors.New("incorrect predecessor pin")
		}
		if _, e = aiWorkproductReadJSON(ctx, root, in.PreviousRef, in.PreviousSHA256, &prior); e != nil {
			return out, e
		}
		if !prior.Complete || prior.Stage != phases[index-1] || prior.Request.Input != in.Input || !reflect.DeepEqual(prior.Manifest, m) || len(prior.Objects) != len(m.Files) {
			return out, errors.New("predecessor is incomplete or membership differs")
		}
		for i, o := range prior.Objects {
			if o.File != m.Files[i] || o.DestinationKey != aiSourceDestinationPrefix(m)+strings.TrimPrefix(m.Files[i].Key, m.SourcePrefix) {
				return out, errors.New("predecessor object mapping changed")
			}
		}
	}
	store, e := a.sourceUnitStore()
	if e != nil {
		return out, e
	}
	if e = aiSourceOriginalKeys(ctx, store, m); e != nil {
		return out, e
	}
	if _, e = os.Lstat(p); e == nil {
		var saved AISourceUnitReceipt
		out.ReceiptSHA256, e = aiWorkproductReadJSON(ctx, root, out.ReceiptRef, "", &saved)
		if e != nil {
			return out, e
		}
		if !saved.Complete || saved.Stage != stage || saved.Request != in || !reflect.DeepEqual(saved.Manifest, m) || len(saved.Objects) != len(m.Files) {
			return out, errors.New("retained incomplete or different attempt; review before new receipt")
		}
		for i, o := range saved.Objects {
			if o.File != m.Files[i] || o.DestinationKey != aiSourceDestinationPrefix(m)+strings.TrimPrefix(m.Files[i].Key, m.SourcePrefix) {
				return out, errors.New("saved object mapping differs")
			}
			if e = aiSourceObserve(ctx, store, m.Bucket, o, index >= 2); e != nil {
				return out, e
			}
			out.Bytes += o.File.Bytes
		}
		if index >= 2 {
			if e = aiSourceUnitKeys(ctx, store, m, false); e != nil {
				return out, e
			}
		}
		out.Objects, out.Complete, out.PhysicalPlacementVerified = len(saved.Objects), true, saved.PhysicalPlacementVerified
		return out, nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return out, e
	}
	r := AISourceUnitReceipt{Stage: stage, Request: in, Manifest: m, PackageCompleteness: "Unresolved: all reviewed inventory members preserved; ZIP membership, attachments and referenced sidecars not validated.", IngestionStatus: "pending", CatalogStatus: "pending"}
	if _, e = aiWorkproductWriteExclusive(p+".attempt", r); e != nil {
		return out, fmt.Errorf("uncertain prior attempt retained; review required: %w", e)
	}
	defer func() {
		if retErr != nil {
			r.Error = retErr.Error()
		}
		sha, err := aiWorkproductWriteExclusive(p, r)
		out.ReceiptSHA256 = sha
		out.Objects = len(r.Objects)
		out.Complete = r.Complete && err == nil
		out.PhysicalPlacementVerified = r.PhysicalPlacementVerified && out.Complete
		retErr = errors.Join(retErr, err)
	}()
	pulse := func() error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if a.Heartbeat != nil {
			a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "ai-source-unit-" + stage})
		}
		return nil
	}
	if index == 0 || stage == "copy" {
		if e = aiSourceUnitKeys(ctx, store, m, true); e != nil {
			return out, e
		}
	}
	// Inspect every destination before the first write, so a known later-member conflict cannot cause a partial copy.
	if stage == "copy" {
		for _, o := range prior.Objects {
			vs, err := store.PlacementVersions(ctx, m.Bucket, o.DestinationKey)
			if err != nil {
				return out, err
			}
			if len(vs) != 0 {
				return out, errors.New("destination history occupied before unit copy")
			}
			if err = aiSourceObserve(ctx, store, m.Bucket, o, false); err != nil {
				return out, err
			}
		}
	}
	for i, f := range m.Files {
		if e = pulse(); e != nil {
			return out, e
		}
		obj := AISourceUnitObject{File: f, DestinationKey: aiSourceDestinationPrefix(m) + strings.TrimPrefix(f.Key, m.SourcePrefix), BeforeVersions: []string{}, AfterVersions: []string{}}
		if index > 0 {
			obj = prior.Objects[i]
		}
		r.Objects = append(r.Objects, obj)
		o := &r.Objects[len(r.Objects)-1]
		switch stage {
		case "inspect":
			o.SourceMetadata, e = store.HeadExact(ctx, m.Bucket, f.Key, f.VersionID)
			if e == nil && (o.SourceMetadata.VersionID != f.VersionID || o.SourceMetadata.Bytes != f.Bytes) {
				e = errors.New("source HEAD differs from exact frozen member")
			}
			if e == nil {
				o.BeforeVersions, e = store.PlacementVersions(ctx, m.Bucket, o.DestinationKey)
				if e == nil && len(o.BeforeVersions) != 0 {
					e = errors.New("destination history occupied")
				}
			}
		case "hash-source", "hash-destination":
			e = aiSourceObserve(ctx, store, m.Bucket, *o, stage == "hash-destination")
			if e != nil {
				break
			}
			key, version := f.Key, f.VersionID
			if stage == "hash-destination" {
				key, version = o.DestinationKey, o.DestinationVersion
			}
			var stream io.ReadCloser
			stream, e = store.OpenVersion(ctx, m.Bucket, key, version)
			if e != nil {
				break
			}
			var h1, h256 string
			var n int64
			h1, h256, n, e = aiSourceDigest(stream, f.Bytes, pulse)
			e = errors.Join(e, stream.Close())
			if e != nil {
				break
			}
			if h1 != f.SHA1 || (f.SHA256 != "" && h256 != f.SHA256) {
				e = errors.New("exact source/destination hash conflicts with frozen member")
				break
			}
			if stage == "hash-source" {
				o.SourceSHA1, o.SourceSHA256 = h1, h256
			} else {
				if h1 != o.SourceSHA1 || h256 != o.SourceSHA256 {
					e = errors.New("destination full bytes differ from source hash receipt")
					break
				}
				o.DestinationSHA1, o.DestinationSHA256, o.DestinationBytes = h1, h256, n
			}
			if e == nil {
				e = aiSourceObserve(ctx, store, m.Bucket, *o, stage == "hash-destination")
			}
		case "copy":
			if o.SourceSHA1 != f.SHA1 || !validSHA256(o.SourceSHA256) {
				e = errors.New("source member has no full-byte hash proof")
				break
			}
			e = aiSourceObserve(ctx, store, m.Bucket, *o, false)
			if e != nil {
				break
			}
			o.BeforeVersions, e = store.PlacementVersions(ctx, m.Bucket, o.DestinationKey)
			if e != nil {
				break
			}
			if len(o.BeforeVersions) != 0 {
				e = errors.New("concurrent occupied destination before copy")
				break
			}
			var headErr error
			_, headErr = store.HeadVersion(ctx, m.Bucket, o.DestinationKey)
			if !errors.Is(headErr, smsthreads.ErrRecoveredVersionNotFound) {
				e = errors.New("destination current absence not confirmed")
				break
			}
			o.DestinationVersion, e = store.CopyExact(ctx, m.Bucket, f.Key, f.VersionID, o.DestinationKey, o.SourceMetadata.ETag)
			if o.DestinationVersion != "" {
				_, ce := aiWorkproductWriteExclusive(fmt.Sprintf("%s.object-%02d.json", p, i+1), struct {
					Request AISourceUnitStageInput `json:"request"`
					Object  AISourceUnitObject     `json:"object"`
				}{in, *o})
				e = errors.Join(e, ce)
			}
			if e == nil {
				o.AfterVersions, e = store.PlacementVersions(ctx, m.Bucket, o.DestinationKey)
			}
			if e == nil {
				e = aiSourceObserve(ctx, store, m.Bucket, *o, true)
			}
		case "readback":
			if o.SourceSHA1 != f.SHA1 || !validSHA256(o.SourceSHA256) || o.DestinationSHA1 != o.SourceSHA1 || o.DestinationSHA256 != o.SourceSHA256 || o.DestinationBytes != f.Bytes {
				e = errors.New("member has no complete independent destination hash proof")
				break
			}
			e = aiSourceObserve(ctx, store, m.Bucket, *o, true)
		}
		if e != nil {
			o.Status = "incomplete"
			o.Error = e.Error()
			return out, e
		}
		o.Status = stage + "-complete"
		out.Bytes += f.Bytes
	}
	if index >= 2 {
		if e = aiSourceUnitKeys(ctx, store, m, false); e != nil {
			return out, e
		}
	}
	r.Complete = true
	r.PhysicalPlacementVerified = stage == "readback"
	return out, nil
}
