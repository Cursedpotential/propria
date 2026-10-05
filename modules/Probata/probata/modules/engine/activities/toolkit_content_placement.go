// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitContentPlacementWorkflowName = "ToolkitContentPlacementWorkflow"
	ToolkitContentPlacementActivityName = "toolkit_content_placement_activity"
	toolkitLegalPrefix                  = "consignatio/casevault/KnowledgeBase/legal/"
)

// ToolkitContentPlacementInput pins a reviewed complete-unit manifest and explicit execution ceilings.
// Inputs: mounted JSON identity, exclusive receipt ref, file/total/archive budgets; outputs: placement receipt. Effects: B2 writes only through the Activity; choose for actual permanent content, never ZIP recovery or catalog refresh.
type ToolkitContentPlacementInput struct {
	ManifestRef     proffer.Ref `json:"manifest_ref"`
	ManifestSHA256  string      `json:"manifest_sha256"`
	ReceiptRef      proffer.Ref `json:"receipt_ref"`
	MaxFiles        int         `json:"max_files"`
	MaxFileBytes    int64       `json:"max_file_bytes"`
	MaxTotalBytes   int64       `json:"max_total_bytes"`
	MaxArchiveBytes int64       `json:"max_archive_bytes"`
}

// ToolkitContentPlacementManifest preserves reviewed unit boundaries and relative paths without selecting winners.
// Inputs: explicit units; outputs: bounded execution plan. Effects: none; choose instead of implicit recursive discovery.
type ToolkitContentPlacementManifest struct {
	Units []ToolkitContentPlacementUnit `json:"units"`
}

// ToolkitContentPlacementUnit names a mounted local directory, pinned local ZIP, or exact preserved B2 ZIP version and its permanent destination.
// Inputs: ID, source root, archive hash/size for either ZIP mode, destination relative to legal/, complete reviewed files; outputs: source mapping. Effects: none; choose mounted ZIP mode for server canonical packages, B2 ZIP mode only with an exact provider version.
type ToolkitContentPlacementUnit struct {
	ID            string                        `json:"id"`
	SourceRef     proffer.Ref                   `json:"source_ref"`
	ArchiveSHA256 string                        `json:"archive_sha256,omitempty"`
	ArchiveBytes  int64                         `json:"archive_bytes,omitempty"`
	Destination   string                        `json:"destination"`
	Files         []ToolkitContentPlacementFile `json:"files"`
}

// ToolkitContentPlacementFile authenticates unchanged bytes at one unit-relative source/destination path.
// Inputs: exact relative path, lowercase SHA-256 and size; outputs: file identity. Effects: none; choose over inferred archive extraction paths.
type ToolkitContentPlacementFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// ToolkitContentPlacementObject records exact retained source-to-B2 identity and conflicting version observations.
// Inputs: processed manifest file; outputs: provenance and status without source bodies. Effects: none; choose as evidence for later catalog/CAS publication, never blind latest reads.
type ToolkitContentPlacementObject struct {
	UnitID          string      `json:"unit_id"`
	SourceRef       proffer.Ref `json:"source_ref"`
	Path            string      `json:"path"`
	ObjectKey       string      `json:"object_key"`
	ObjectRef       proffer.Ref `json:"object_ref"`
	VersionID       string      `json:"version_id,omitempty"`
	SHA256          string      `json:"sha256"`
	Bytes           int64       `json:"bytes"`
	BeforeVersions  []string    `json:"before_versions"`
	AfterVersions   []string    `json:"after_versions"`
	LatestVersionID string      `json:"latest_version_id,omitempty"`
	Status          string      `json:"status"`
	Error           string      `json:"error,omitempty"`
}

// ToolkitContentPlacementResult is an exclusive replay receipt including partial/conflict evidence.
// Inputs: pinned request and object observations; outputs: bounded metadata and receipt SHA. Effects: receipt persisted locally; choose instead of passing bodies through Temporal history.
type ToolkitContentPlacementResult struct {
	Input         ToolkitContentPlacementInput    `json:"input"`
	Objects       []ToolkitContentPlacementObject `json:"objects"`
	Complete      bool                            `json:"complete"`
	Error         string                          `json:"error,omitempty"`
	ReceiptSHA256 string                          `json:"receipt_sha256,omitempty"`
}

// ToolkitContentPlacementSummary carries only receipt identity through Temporal history.
// Inputs: completed or failed placement; outputs: count, status and mounted receipt pin. Effects: none; choose over returning the full path/version manifest to the workflow.
type ToolkitContentPlacementSummary struct {
	ReceiptRef    proffer.Ref `json:"receipt_ref"`
	ReceiptSHA256 string      `json:"receipt_sha256"`
	Objects       int         `json:"objects"`
	Complete      bool        `json:"complete"`
}

// toolkitPlacementStore extends existing retained-version storage with bounded version observations.
// Inputs: existing store; outputs: exact version lists including delete markers. Effects: remote metadata reads; choose to detect intervening writes without claiming atomic create.
type toolkitPlacementStore interface {
	toolkitVersionedObjectStore
	PlacementVersions(context.Context, string, string) ([]string, error)
}

// toolkitPlacementS3 reuses the worker's S3 client with no credentials or new runtime.
// Inputs: configured S3Store; outputs: placement capabilities. Effects: none at construction; choose when resolving the concrete existing B2 adapter.
type toolkitPlacementS3 struct{ smsthreads.S3Store }

// PutRecoveredVersion retains one fresh placement payload and returns only its actual PUT response version.
// Inputs: absence-prechecked coordinates, seekable bytes, SHA/size and progress; outputs: returned provider VersionId. Effects: one retained PUT, no delete or lost-response guess; choose over the archive sibling's matching-latest recovery so concurrent appearances remain distinguishable by before/after observations.
func (s toolkitPlacementS3) PutRecoveredVersion(ctx context.Context, bucket, key string, body io.ReadSeeker, size int64, contentType, expectedSHA string, progress func(int64)) (string, error) {
	if s.Client == nil || size < 0 || size > 256<<20 || !validSHA256(expectedSHA) {
		return "", errors.New("invalid bounded placement write")
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	sha, n, err := streamToolkitDigest(body, size, func() error { return ctx.Err() })
	if err != nil {
		return "", err
	}
	if sha != expectedSHA || n != size {
		return "", errors.New("placement spool changed before upload")
	}
	if _, err = body.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	out, err := s.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(key), Body: body, ContentLength: aws.Int64(size), ContentType: aws.String(contentType)})
	if err != nil {
		return "", err
	}
	id := aws.ToString(out.VersionId)
	if !validToolkitVersionID(id) {
		return "", errors.New("PUT response lacks retained VersionId; inspect conflict receipt before retry")
	}
	if progress != nil {
		progress(size)
	}
	return id, nil
}

// PlacementVersions observes at most 64 versions/markers under the exact key prefix and rejects truncated observations.
// Inputs: B2 coordinates; outputs: sorted exact-key identities. Effects: one bounded SDK request; choose before/after retained writes, never as whole-bucket inventory.
func (s toolkitPlacementS3) PlacementVersions(ctx context.Context, bucket, key string) ([]string, error) {
	if s.Client == nil {
		return nil, errors.New("placement requires configured B2 client")
	}
	out, err := s.Client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{Bucket: aws.String(bucket), Prefix: aws.String(key), MaxKeys: aws.Int32(64)})
	if err != nil {
		return nil, err
	}
	if aws.ToBool(out.IsTruncated) {
		return nil, errors.New("version observation exceeds 64 entries; refusing incomplete race check")
	}
	versions := []string{}
	for _, v := range out.Versions {
		if aws.ToString(v.Key) == key {
			if !validToolkitVersionID(aws.ToString(v.VersionId)) {
				return nil, errors.New("invalid observed VersionId")
			}
			versions = append(versions, aws.ToString(v.VersionId))
		}
	}
	for _, v := range out.DeleteMarkers {
		if aws.ToString(v.Key) == key {
			if !validToolkitVersionID(aws.ToString(v.VersionId)) {
				return nil, errors.New("invalid observed delete marker VersionId")
			}
			versions = append(versions, "delete:"+aws.ToString(v.VersionId))
		}
	}
	sort.Strings(versions)
	if err := toolkitPlacementObservations(versions); err != nil {
		return nil, err
	}
	return versions, nil
}

// ToolkitContentPlacementActivities executes reviewed placement using existing storage and mounted scratch roots.
// Inputs: allowed root, store resolver, optional heartbeat; outputs: exclusive receipts. Effects: retained B2 writes and scratch files; choose beside preservation, not as an automatic sync watcher.
type ToolkitContentPlacementActivities struct {
	AllowedRoot string
	Stores      func(string) (smsthreads.ObjectStore, error)
	Heartbeat   func(context.Context, ToolkitPackagePreservationHeartbeat)
}

// NewToolkitContentPlacementActivities creates the separately registered placement group.
// Inputs: existing mounted root/resolver; outputs: pointer for worker registration. Effects: none until invoked; choose to reuse the preservation worker seam.
func NewToolkitContentPlacementActivities(root string, stores func(string) (smsthreads.ObjectStore, error)) *ToolkitContentPlacementActivities {
	return &ToolkitContentPlacementActivities{AllowedRoot: root, Stores: stores, Heartbeat: func(ctx context.Context, p ToolkitPackagePreservationHeartbeat) { activity.RecordHeartbeat(ctx, p) }}
}

// ToolkitContentPlacementWorkflow schedules one bounded, cancellation-aware manifest placement.
// Inputs: pinned manifest and budgets; outputs: receipt or visible failure. Effects: one Activity, no DB/source writes; choose for parent-reviewed placement, not winner selection or directory synchronization.
func ToolkitContentPlacementWorkflow(ctx workflow.Context, in ToolkitContentPlacementInput) (ToolkitContentPlacementSummary, error) {
	if err := validateToolkitPlacementInput(in); err != nil {
		return ToolkitContentPlacementSummary{}, temporal.NewNonRetryableApplicationError(err.Error(), "ToolkitPlacementInvalid", err)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 24 * time.Hour, ScheduleToCloseTimeout: 72 * time.Hour, HeartbeatTimeout: time.Minute, WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2}})
	var out ToolkitContentPlacementSummary
	err := workflow.ExecuteActivity(ctx, ToolkitContentPlacementActivityName, in).Get(ctx, &out)
	return out, err
}

// PlaceToolkitContent returns compact metadata after retaining the complete placement/conflict receipt.
// Inputs: pinned manifest/request; outputs: receipt summary or visible error. Effects: delegates bounded storage work; choose as the registered Activity instead of returning full version observations in history.
func (a *ToolkitContentPlacementActivities) PlaceToolkitContent(ctx context.Context, in ToolkitContentPlacementInput) (ToolkitContentPlacementSummary, error) {
	r, err := a.placeToolkitContent(ctx, in)
	return ToolkitContentPlacementSummary{ReceiptRef: in.ReceiptRef, ReceiptSHA256: r.ReceiptSHA256, Objects: len(r.Objects), Complete: r.Complete}, err
}

// validateToolkitPlacementInput enforces fixed ceilings before local or remote reads.
// Inputs: request; outputs: validation error. Effects: none; choose before loading any manifest.
func validateToolkitPlacementInput(in ToolkitContentPlacementInput) error {
	if !validSHA256(in.ManifestSHA256) || in.ManifestRef == in.ReceiptRef || in.MaxFiles < 1 || in.MaxFiles > 512 || in.MaxFileBytes < 1 || in.MaxFileBytes > 256<<20 || in.MaxTotalBytes < 1 || in.MaxTotalBytes > 2<<30 || in.MaxArchiveBytes < 1 || in.MaxArchiveBytes > 8<<30 {
		return errors.New("invalid placement pin or explicit budgets (512 files, 256 MiB/file, 2 GiB total, 8 GiB archives maximum)")
	}
	return nil
}

// toolkitPlacementPath accepts only exact portable relative paths without aliases.
// Inputs: manifest path; outputs: safe status. Effects: none; choose over normalizing ambiguous ZIP/destination names.
func toolkitPlacementPath(p string) bool {
	return toolkitSelectedSafeName(p) && !strings.Contains(p, "\\") && !strings.ContainsAny(p, "?#")
}

// validateToolkitPlacementManifest validates complete-unit assertions, unique targets and aggregate source budgets.
// Inputs: manifest/request; outputs: error before writes. Effects: none; choose before reading source bodies; completeness is supplied by the reviewed manifest, not inferred here.
func validateToolkitPlacementManifest(m ToolkitContentPlacementManifest, in ToolkitContentPlacementInput) error {
	if len(m.Units) < 1 || len(m.Units) > 32 {
		return errors.New("unit budget exceeded")
	}
	ids, destinations, archives := map[string]bool{}, map[string]bool{}, map[string]bool{}
	archivePins := map[proffer.Ref]ToolkitContentPlacementUnit{}
	count, total, archiveTotal := 0, int64(0), int64(0)
	for _, u := range m.Units {
		if !toolkitPlacementPath(u.ID) || ids[u.ID] || !toolkitPlacementPath(u.Destination) || len(u.Files) == 0 {
			return errors.New("invalid or duplicate unit identity/destination")
		}
		ids[u.ID] = true
		category := strings.Split(u.Destination, "/")[0]
		if category != "case-law" && category != "benchbooks" && category != "reference-data" {
			return errors.New("destination must be an existing legal category")
		}
		parsed, err := url.Parse(string(u.SourceRef))
		if err != nil {
			return err
		}
		if parsed.Scheme == "b2" {
			query, queryErr := url.ParseQuery(parsed.RawQuery)
			if queryErr != nil || len(query) != 1 {
				return errors.New("invalid B2 version query")
			}
			if parsed.Host != "salem-data" || parsed.Fragment != "" || len(parsed.Query()) != 1 || len(parsed.Query()["versionId"]) != 1 || !validToolkitVersionID(parsed.Query().Get("versionId")) || !toolkitPlacementPath(strings.TrimPrefix(parsed.Path, "/")) || !validSHA256(u.ArchiveSHA256) || u.ArchiveBytes < 1 || u.ArchiveBytes > in.MaxArchiveBytes {
				return errors.New("archive requires exact B2 version, SHA and byte budget")
			}
		} else if parsed.Scheme != "file" {
			return errors.New("source must be mounted local directory, pinned local ZIP or pinned B2 ZIP; no provider fallback")
		} else if u.ArchiveBytes != 0 || u.ArchiveSHA256 != "" {
			if !validSHA256(u.ArchiveSHA256) || u.ArchiveBytes < 1 || u.ArchiveBytes > in.MaxArchiveBytes {
				return errors.New("local ZIP requires both exact archive SHA and bounded positive bytes")
			}
		}
		if u.ArchiveBytes > 0 {
			if !archives[string(u.SourceRef)] {
				archiveTotal += u.ArchiveBytes
				archives[string(u.SourceRef)] = true
				archivePins[u.SourceRef] = u
			} else if previous := archivePins[u.SourceRef]; previous.ArchiveSHA256 != u.ArchiveSHA256 || previous.ArchiveBytes != u.ArchiveBytes {
				return errors.New("same archive source has contradictory manifest pins")
			}
		}
		for _, f := range u.Files {
			key := u.Destination + "/" + f.Path
			if !toolkitPlacementPath(f.Path) || !validSHA256(f.SHA256) || f.Bytes < 0 || f.Bytes > in.MaxFileBytes || destinations[key] {
				return errors.New("invalid file identity, size or duplicate target")
			}
			destinations[key] = true
			count++
			total += f.Bytes
		}
	}
	if count > in.MaxFiles || total > in.MaxTotalBytes || archiveTotal > in.MaxArchiveBytes {
		return errors.New("aggregate placement budget exceeded")
	}
	return nil
}

// toolkitPlacementLocalFile resolves a nonlinked regular file strictly within a canonical source unit.
// Inputs: canonical unit root and exact relative path; outputs: source path. Effects: metadata reads only; choose before opening original bytes.
func toolkitPlacementLocalFile(root, relative string) (string, error) {
	p := root
	for _, part := range strings.Split(relative, "/") {
		p = filepath.Join(p, part)
		info, err := os.Lstat(p)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("linked source member refused")
		}
	}
	info, err := os.Stat(p)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("source member is not regular")
	}
	return p, nil
}

// toolkitPlacementOpenLocal checks opened-file identity and canonical containment against concurrent path replacement.
// Inputs: canonical unit root and exact relative path; outputs: read-only regular source handle. Effects: source read handle only; choose over an unchecked os.Open after path validation.
func toolkitPlacementOpenLocal(root, relative string) (*os.File, error) {
	p, err := toolkitPlacementLocalFile(root, relative)
	if err != nil {
		return nil, err
	}
	before, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	current, err := f.Stat()
	if err != nil || !os.SameFile(before, current) {
		f.Close()
		return nil, errors.New("local source identity changed while opening")
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		f.Close()
		return nil, err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		f.Close()
		return nil, errors.New("local source escaped its unit root")
	}
	return f, nil
}

// toolkitPlacementOpenLocalZIP admits an absolute mounted file reference and opens a nonlinked regular ZIP under AllowedRoot.
// Inputs: canonical allowed root and exact file:// ZIP locator; outputs: read-only source handle. Effects: metadata/read handle only; choose instead of the directory/JSON reference helpers when archive SHA and bytes are explicitly pinned.
// Byline: Codex · GPT-6 · 2026-10-04.
func toolkitPlacementOpenLocalZIP(root string, ref proffer.Ref) (*os.File, error) {
	parsed, err := url.Parse(string(ref))
	if err != nil {
		return nil, err
	}
	p := filepath.FromSlash(parsed.Path)
	if len(p) > 1 && p[0] == os.PathSeparator && filepath.VolumeName(p[1:]) != "" {
		p = p[1:]
	}
	if parsed.Scheme != "file" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !filepath.IsAbs(p) || !strings.EqualFold(filepath.Ext(p), ".zip") {
		return nil, errors.New("local archive must be an absolute file:// .zip without host, query or fragment")
	}
	relative, err := filepath.Rel(root, p)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return nil, errors.New("local ZIP must lie strictly beneath AllowedRoot")
	}
	relative = filepath.ToSlash(relative)
	if !toolkitPlacementPath(relative) {
		return nil, errors.New("unsafe local ZIP path")
	}
	return toolkitPlacementOpenLocal(root, relative)
}

// toolkitPlacementDecode rejects unknown JSON fields and trailing data in bounded metadata.
// Inputs: JSON bytes and target; outputs: decode error. Effects: none; choose for pinned manifests and receipts.
func toolkitPlacementDecode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}

// toolkitPlacementZIPBounds checks the classic ZIP central-directory ceiling before archive/zip allocations.
// Inputs: retained seekable ZIP and exact size; outputs: bounded format acceptance. Effects: at most 65557 bytes read; choose before NewReader; ZIP64 and over 8192 entries/4 MiB central directories fail visibly in this first placement pass.
func toolkitPlacementZIPBounds(f *os.File, size int64) error {
	n := size
	if n > 65557 {
		n = 65557
	}
	if n < 22 {
		return errors.New("ZIP is too short")
	}
	tail := make([]byte, n)
	if _, err := f.ReadAt(tail, size-n); err != nil {
		return err
	}
	for i := len(tail) - 22; i >= 0; i-- {
		if !bytes.Equal(tail[i:i+4], []byte{'P', 'K', 5, 6}) {
			continue
		}
		if i+22+int(binary.LittleEndian.Uint16(tail[i+20:])) != len(tail) {
			continue
		}
		entries := binary.LittleEndian.Uint16(tail[i+10:])
		central := binary.LittleEndian.Uint32(tail[i+12:])
		if binary.LittleEndian.Uint16(tail[i+4:]) != 0 || binary.LittleEndian.Uint16(tail[i+6:]) != 0 || entries > 8192 || central > 4<<20 || binary.LittleEndian.Uint32(tail[i+16:]) == 0xffffffff {
			return errors.New("ZIP central directory exceeds supported bounded single-disk format")
		}
		return nil
	}
	return errors.New("ZIP end directory missing")
}

// toolkitPlacementObservations bounds retained-version metadata independently of the storage adapter.
// Inputs: one version list; outputs: validity error. Effects: none; choose before serializing observations so receipts remain within 32 MiB.
func toolkitPlacementObservations(versions []string) error {
	if len(versions) > 64 {
		return errors.New("version observation budget exceeded")
	}
	seen := map[string]bool{}
	for _, v := range versions {
		if v == "" || len(v) > 256 || seen[v] {
			return errors.New("invalid or oversized version observation")
		}
		seen[v] = true
	}
	return nil
}

// toolkitPlacementBody spools authenticated source bytes to retained scratch for seekable SDK upload.
// Inputs: source reader, pinned identity, scratch and pulse; outputs: seekable file. Effects: scratch creation only; choose instead of body buffering or shell extraction.
func toolkitPlacementBody(r io.Reader, f ToolkitContentPlacementFile, scratch string, pulse func() error) (*os.File, error) {
	out, err := os.CreateTemp(scratch, "placement-member-*")
	if err != nil {
		return nil, err
	}
	sha, n, err := streamToolkitDigest(io.TeeReader(r, out), f.Bytes, pulse)
	if err != nil || sha != f.SHA256 || n != f.Bytes {
		out.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("source SHA/size mismatch")
	}
	if _, err = out.Seek(0, io.SeekStart); err != nil {
		out.Close()
		return nil, err
	}
	return out, nil
}

// toolkitPlacementConflictChecks checks retained observations and latest identity after a write/reuse.
// Inputs: prior/after lists, chosen/latest IDs; outputs: explicit conflict. Effects: none; choose to detect races without promising an atomic lock against external writers.
func toolkitPlacementConflictChecks(before, after []string, chosen, latest string) error {
	observed := map[string]bool{}
	for _, v := range after {
		observed[v] = true
	}
	for _, v := range before {
		if !observed[v] {
			return errors.New("prior retained version observation disappeared")
		}
	}
	prior := map[string]bool{}
	for _, v := range before {
		prior[v] = true
	}
	for _, v := range after {
		if !prior[v] && v != chosen {
			return errors.New("concurrent version appeared")
		}
	}
	if !observed[chosen] || latest != chosen {
		return errors.New("latest version differs from verified placement version")
	}
	return nil
}

// placeToolkitContent authenticates sources and writes/reuses only verified retained B2 versions at permanent legal paths.
// Inputs: pinned complete-unit manifest and bounds; outputs: exclusive metadata receipt including partial/conflict IDs. Effects: B2 retained writes and preserved scratch/receipt files only; choose for actual placement, not automatic two-way synchronization or Surreal publication.
func (a *ToolkitContentPlacementActivities) placeToolkitContent(ctx context.Context, in ToolkitContentPlacementInput) (result ToolkitContentPlacementResult, retErr error) {
	result.Input = in
	if err := validateToolkitPlacementInput(in); err != nil {
		return result, err
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return result, err
	}
	pulse := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.Heartbeat != nil {
			a.Heartbeat(ctx, ToolkitPackagePreservationHeartbeat{Phase: "content-placement"})
		}
		return nil
	}
	manifestPath, err := resolveFileRef(in.ManifestRef, root, false)
	if err != nil {
		return result, err
	}
	data, sha, err := toolkitReadBoundedJSON(manifestPath, 1<<20, pulse)
	if err != nil {
		return result, err
	}
	if sha != in.ManifestSHA256 {
		return result, errors.New("manifest SHA mismatch")
	}
	var manifest ToolkitContentPlacementManifest
	if err = toolkitPlacementDecode(data, &manifest); err != nil {
		return result, err
	}
	if err = validateToolkitPlacementManifest(manifest, in); err != nil {
		return result, err
	}
	receiptPath, err := resolveFileRef(in.ReceiptRef, root, false)
	if err != nil {
		return result, err
	}
	if receiptPath == manifestPath {
		return result, errors.New("receipt must not alias the input manifest")
	}
	for _, u := range manifest.Units {
		p, _ := url.Parse(string(u.SourceRef))
		if p.Scheme == "file" {
			if u.ArchiveBytes > 0 {
				f, e := toolkitPlacementOpenLocalZIP(root, u.SourceRef)
				if e != nil {
					return result, e
				}
				info, e := f.Stat()
				f.Close()
				if e != nil {
					return result, e
				}
				if info.Size() != u.ArchiveBytes {
					return result, errors.New("local ZIP size differs from manifest pin")
				}
				continue
			}
			unitRoot, e := resolveFileRef(u.SourceRef, root, true)
			if e != nil {
				return result, e
			}
			rel, e := filepath.Rel(unitRoot, receiptPath)
			if e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return result, errors.New("receipt must lie outside source units")
			}
		}
	}
	if a.Stores == nil {
		return result, errors.New("B2 store resolver required")
	}
	base, err := a.Stores("b2")
	if err != nil {
		return result, err
	}
	if s, ok := base.(smsthreads.S3Store); ok {
		base = toolkitPlacementS3{s}
	}
	if s, ok := base.(*smsthreads.S3Store); ok && s != nil {
		base = toolkitPlacementS3{*s}
	}
	store, ok := base.(toolkitPlacementStore)
	if !ok {
		return result, errors.New("B2 store requires retained versions and bounded version observations")
	}
	var replay *ToolkitContentPlacementResult
	if old, err := os.Lstat(receiptPath); err == nil {
		if !old.Mode().IsRegular() {
			return result, errors.New("receipt is not regular")
		}
		encoded, _, err := toolkitReadBoundedJSON(receiptPath, 32<<20, pulse)
		if err != nil {
			return result, err
		}
		var r ToolkitContentPlacementResult
		if err = toolkitPlacementDecode(encoded, &r); err != nil {
			return result, err
		}
		if r.Input != in || !r.Complete {
			return result, errors.New("receipt replay input mismatch or retained incomplete attempt; use a new reviewed receipt ref")
		}
		replay = &r
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	var receipt *os.File
	if replay == nil {
		receipt, err = os.OpenFile(receiptPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return result, err
		}
		defer func() {
			if retErr != nil {
				result.Error = retErr.Error()
			}
			encoded, e := json.Marshal(result)
			if e == nil {
				_, e = receipt.Write(encoded)
			}
			if e == nil {
				e = receipt.Sync()
			}
			ce := receipt.Close()
			if e == nil {
				e = ce
			}
			if e != nil {
				retErr = errors.Join(retErr, e)
			} else {
				result.ReceiptSHA256 = digestBytes(encoded)
			}
		}()
	}
	scratch, err := os.MkdirTemp(root, "placement-scratch-*")
	if err != nil {
		return result, err
	}
	archives := map[proffer.Ref]*os.File{}
	archiveHashes := map[proffer.Ref]string{}
	defer func() {
		for _, f := range archives {
			f.Close()
		}
	}()
	replayIndex := 0
	for _, u := range manifest.Units {
		parsed, _ := url.Parse(string(u.SourceRef))
		var localRoot string
		var archive *zip.Reader
		if parsed.Scheme == "file" && u.ArchiveBytes == 0 {
			localRoot, err = resolveFileRef(u.SourceRef, root, true)
			if err != nil {
				return result, err
			}
		} else {
			af := archives[u.SourceRef]
			if af == nil {
				var stream io.ReadCloser
				var e error
				if parsed.Scheme == "file" {
					stream, e = toolkitPlacementOpenLocalZIP(root, u.SourceRef)
				} else {
					stream, e = store.OpenVersion(ctx, parsed.Host, strings.TrimPrefix(parsed.Path, "/"), parsed.Query().Get("versionId"))
				}
				if e != nil {
					return result, e
				}
				af, e = toolkitPlacementBody(stream, ToolkitContentPlacementFile{SHA256: u.ArchiveSHA256, Bytes: u.ArchiveBytes}, scratch, pulse)
				stream.Close()
				if e != nil {
					return result, e
				}
				archives[u.SourceRef] = af
				archiveHashes[u.SourceRef] = u.ArchiveSHA256
			}
			info, e := af.Stat()
			if e != nil {
				return result, e
			}
			if info.Size() != u.ArchiveBytes || archiveHashes[u.SourceRef] != u.ArchiveSHA256 {
				return result, errors.New("deduplicated archive identity differs")
			}
			if err = toolkitPlacementZIPBounds(af, u.ArchiveBytes); err != nil {
				return result, err
			}
			archive, err = zip.NewReader(af, u.ArchiveBytes)
			if err != nil {
				return result, err
			}
		}
		for _, f := range u.Files {
			if err = pulse(); err != nil {
				return result, err
			}
			var reader io.ReadCloser
			if archive == nil {
				reader, err = toolkitPlacementOpenLocal(localRoot, f.Path)
				if err != nil {
					return result, err
				}
			} else {
				var member *zip.File
				for _, z := range archive.File {
					if z.Name == f.Path {
						if member != nil {
							return result, errors.New("duplicate ZIP member")
						}
						member = z
					}
				}
				if member == nil || !member.Mode().IsRegular() || member.UncompressedSize64 != uint64(f.Bytes) {
					return result, errors.New("missing/nonregular ZIP member or size mismatch")
				}
				reader, err = member.Open()
				if err != nil {
					return result, err
				}
			}
			body, e := toolkitPlacementBody(reader, f, scratch, pulse)
			reader.Close()
			if e != nil {
				return result, e
			}
			obj := ToolkitContentPlacementObject{UnitID: u.ID, SourceRef: u.SourceRef, Path: f.Path, ObjectKey: toolkitLegalPrefix + u.Destination + "/" + f.Path, SHA256: f.SHA256, Bytes: f.Bytes, Status: "pending"}
			if replay != nil {
				if replayIndex >= len(replay.Objects) {
					body.Close()
					return result, errors.New("receipt replay mapping missing")
				}
				prior := replay.Objects[replayIndex]
				replayIndex++
				if prior.UnitID != obj.UnitID || prior.SourceRef != obj.SourceRef || prior.Path != obj.Path || prior.ObjectKey != obj.ObjectKey || prior.SHA256 != obj.SHA256 || prior.Bytes != obj.Bytes || prior.Status != "verified" {
					body.Close()
					return result, errors.New("receipt replay mapping mismatch")
				}
				obj.VersionID = prior.VersionID
			}
			index := len(result.Objects)
			result.Objects = append(result.Objects, obj)
			err = toolkitPlaceOne(ctx, store, body, &result.Objects[index], replay != nil, a.Heartbeat, pulse)
			body.Close()
			if err != nil {
				result.Objects[index].Status = "conflict-or-failure"
				result.Objects[index].Error = err.Error()
				return result, err
			}
		}
	}
	if replay != nil && replayIndex != len(replay.Objects) {
		return result, errors.New("receipt has extra mapping entries")
	}
	result.Complete = true
	if replay != nil {
		encoded, _, e := toolkitReadBoundedJSON(receiptPath, 32<<20, pulse)
		if e != nil {
			return result, e
		}
		result.ReceiptSHA256 = digestBytes(encoded)
	}
	return result, nil
}

// toolkitPlaceOne prechecks, retains, independently reads back, and reports observed concurrent versions.
// Inputs: authenticated seekable body, mapping, replay mode and liveness; outputs: pinned verified mapping or conflict. Effects: retained B2 write only when absent, no deletes; choose instead of atomic-create claims.
func toolkitPlaceOne(ctx context.Context, store toolkitPlacementStore, body io.ReadSeeker, obj *ToolkitContentPlacementObject, replay bool, hb func(context.Context, ToolkitPackagePreservationHeartbeat), pulse func() error) error {
	var err error
	body = &toolkitHeartbeatReadSeeker{Reader: body, ctx: ctx, heartbeat: hb, packageName: obj.UnitID}
	obj.BeforeVersions, err = store.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	if err = toolkitPlacementObservations(obj.BeforeVersions); err != nil {
		return err
	}
	head, err := store.HeadVersion(ctx, "salem-data", obj.ObjectKey)
	if err == nil {
		obj.LatestVersionID = head.VersionID
		if replay && obj.VersionID != head.VersionID {
			return errors.New("replay destination changed")
		}
		obj.VersionID = head.VersionID
		if head.Size != obj.Bytes {
			return errors.New("existing destination size conflict")
		}
		if !containsToolkitPackage(obj.BeforeVersions, head.VersionID) {
			return errors.New("concurrent destination appeared during precheck")
		}
	} else if !errors.Is(err, smsthreads.ErrRecoveredVersionNotFound) {
		return err
	} else {
		if replay || len(obj.BeforeVersions) > 0 {
			return errors.New("destination disappeared or prior versions exist; refusing fresh write")
		}
		obj.VersionID, err = store.PutRecoveredVersion(ctx, "salem-data", obj.ObjectKey, body, obj.Bytes, "application/octet-stream", obj.SHA256, func(n int64) {
			if hb != nil {
				hb(ctx, ToolkitPackagePreservationHeartbeat{Phase: "content-upload", Bytes: n})
			}
		})
		if err != nil {
			obj.AfterVersions, _ = store.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
			if latest, e := store.HeadVersion(ctx, "salem-data", obj.ObjectKey); e == nil {
				obj.LatestVersionID = latest.VersionID
			}
			return err
		}
	}
	if !validToolkitVersionID(obj.VersionID) {
		return errors.New("placement returned invalid VersionId")
	}
	stream, err := store.OpenVersion(ctx, "salem-data", obj.ObjectKey, obj.VersionID)
	if err != nil {
		return err
	}
	sha, n, err := streamToolkitDigest(stream, obj.Bytes, pulse)
	stream.Close()
	if err != nil {
		return err
	}
	if sha != obj.SHA256 || n != obj.Bytes {
		return errors.New("pinned destination SHA/size conflict")
	}
	obj.AfterVersions, err = store.PlacementVersions(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	if err = toolkitPlacementObservations(obj.AfterVersions); err != nil {
		return err
	}
	head, err = store.HeadVersion(ctx, "salem-data", obj.ObjectKey)
	if err != nil {
		return err
	}
	obj.LatestVersionID = head.VersionID
	if err = toolkitPlacementConflictChecks(obj.BeforeVersions, obj.AfterVersions, obj.VersionID, head.VersionID); err != nil {
		return err
	}
	pinned := url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + obj.ObjectKey, RawQuery: url.Values{"versionId": []string{obj.VersionID}}.Encode()}
	obj.ObjectRef = proffer.Ref(pinned.String())
	obj.Status = "verified"
	return nil
}
