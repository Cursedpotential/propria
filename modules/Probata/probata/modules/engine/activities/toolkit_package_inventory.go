// Byline: Codex, 2026-10-04.
package activities

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	toolkitInventoryHeartbeatInterval   = 15 * time.Second
	toolkitInventoryMaxPackages         = 100
	toolkitInventoryMaxDirEntries       = 1000
	toolkitInventoryMaxBytes            = 8 << 30
	toolkitInventoryMaxMembers          = 100000
	toolkitInventoryMaxReceiptBytes     = 32 << 20
	ToolkitPackageInventoryActivityName = "toolkit_package_inventory_activity"
	ToolkitPackageInventoryWorkflowName = "ToolkitPackageInventoryWorkflow"
)

// ToolkitPackageInventoryInput bounds server-local ZIP inspection.
// Inputs: file references and positive aggregate byte, member, package and listing ceilings.
// Outputs: a receipt reference; effects: reads sources and exclusively creates a receipt.
// Choose for toolkit comparison preparation rather than database-backed InventoryContainer.
type ToolkitPackageInventoryInput struct {
	SourceRef           proffer.Ref `json:"source_ref"`
	ReceiptRef          proffer.Ref `json:"receipt_ref"`
	MaxPackages         int         `json:"max_packages"`
	MaxArchiveBytes     int64       `json:"max_archive_bytes"`
	MaxExpandedBytes    int64       `json:"max_expanded_bytes"`
	MaxMembers          int         `json:"max_members"`
	MaxDirectoryEntries int         `json:"max_directory_entries"`
}

// ToolkitPackageInventoryResult returns bounded workflow history data.
// Inputs: verified receipt location and count; outputs: references only.
// Effects: none. Choose instead of returning archive contents through Temporal.
type ToolkitPackageInventoryResult struct {
	ReceiptRef   proffer.Ref `json:"receipt_ref"`
	PackageCount int         `json:"package_count"`
}

// ToolkitPackageInventoryHeartbeat describes liveness without member payloads.
// Inputs: source, receipt, phase and count; outputs: compact Temporal progress.
// Effects: none. Choose for tracked streaming and cancellation.
type ToolkitPackageInventoryHeartbeat struct {
	SourceRef    proffer.Ref `json:"source_ref"`
	ReceiptRef   proffer.Ref `json:"receipt_ref"`
	Phase        string      `json:"phase"`
	PackageCount int         `json:"package_count"`
}

// ToolkitPackageInventoryActivities inspects ZIPs natively under one configured root.
// Inputs: allowed root and heartbeat callback; outputs: verified inventory receipt.
// Effects: read-only sources, exclusive receipt creation. Choose over a subprocess on the Go worker.
type ToolkitPackageInventoryActivities struct {
	AllowedRoot       string
	Heartbeat         func(context.Context, ToolkitPackageInventoryHeartbeat)
	HeartbeatInterval time.Duration
}

// NewToolkitPackageInventoryActivities configures native inspection.
// Inputs: absolute server-owned root; outputs: fail-closed activity group.
// Effects: none. Choose for the existing Proffer worker with no Python dependency.
func NewToolkitPackageInventoryActivities(root string) ToolkitPackageInventoryActivities {
	return ToolkitPackageInventoryActivities{AllowedRoot: root, HeartbeatInterval: toolkitInventoryHeartbeatInterval,
		Heartbeat: func(ctx context.Context, detail ToolkitPackageInventoryHeartbeat) {
			activity.RecordHeartbeat(ctx, detail)
		}}
}

// ToolkitPackageInventoryWorkflow schedules one isolated inventory Activity.
// Inputs: bounded inspection request; outputs: receipt reference or visible failure.
// Effects: Temporal scheduling only, no import or database writes.
// Choose over ProfferWorkflow when only package integrity inventory is required.
func ToolkitPackageInventoryWorkflow(ctx workflow.Context, input ToolkitPackageInventoryInput) (ToolkitPackageInventoryResult, error) {
	var result ToolkitPackageInventoryResult
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, ScheduleToCloseTimeout: time.Hour, HeartbeatTimeout: time.Minute,
		WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	err := workflow.ExecuteActivity(ctx, ToolkitPackageInventoryActivityName, input).Get(ctx, &result)
	return result, err
}

type toolkitMember struct {
	Path           string `json:"path"`
	Bytes          int64  `json:"bytes"`
	SHA256         string `json:"sha256"`
	Empty          bool   `json:"empty"`
	AuditCandidate bool   `json:"audit_candidate"`
}
type toolkitIssue struct {
	Member string `json:"member,omitempty"`
	Error  string `json:"error"`
}
type toolkitPackage struct {
	Package  string          `json:"package"`
	SHA256   *string         `json:"package_sha256"`
	Members  []toolkitMember `json:"members"`
	Errors   []toolkitIssue  `json:"errors"`
	Complete bool            `json:"complete"`
}
type toolkitOccurrence struct {
	Package string `json:"package"`
	Member  string `json:"member"`
}
type toolkitReceipt struct {
	Schema          string                       `json:"schema"`
	Byline          string                       `json:"byline"`
	CreatedAt       string                       `json:"created_at"`
	Packages        []toolkitPackage             `json:"packages"`
	Identical       [][]toolkitOccurrence        `json:"identical_member_candidates"`
	LegalValidation string                       `json:"legal_validation"`
	Request         ToolkitPackageInventoryInput `json:"request"`
}

// RunToolkitPackageInventory streams bounded ZIPs and verifies existing receipts on retries.
// Inputs: server-local references and aggregate budgets; outputs: receipt reference or explicit failure.
// Effects: reads source bytes, checks ZIP CRCs, exclusively writes JSON; preserves incomplete receipts.
// Choose for integrity comparison preparation; legal validation and import remain separate.
func (a ToolkitPackageInventoryActivities) RunToolkitPackageInventory(ctx context.Context, input ToolkitPackageInventoryInput) (ToolkitPackageInventoryResult, error) {
	fail := func(err error) (ToolkitPackageInventoryResult, error) {
		if ctx.Err() != nil {
			return ToolkitPackageInventoryResult{}, ctx.Err()
		}
		return ToolkitPackageInventoryResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("toolkit inventory failed; preserved receipt (if created): %s: %v", input.ReceiptRef, err),
			"ToolkitPackageInventoryFailed", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := input.validate(); err != nil {
		return fail(err)
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return fail(err)
	}
	source, err := resolveFileRef(input.SourceRef, root, true)
	if err != nil {
		return fail(err)
	}
	receiptPath, err := resolveFileRef(input.ReceiptRef, root, false)
	if err != nil {
		return fail(err)
	}
	var previous *toolkitReceipt
	if info, e := os.Lstat(receiptPath); e == nil {
		if !info.Mode().IsRegular() || info.Size() > toolkitInventoryMaxReceiptBytes {
			return fail(errors.New("existing receipt is not a bounded regular file"))
		}
		raw, e := os.ReadFile(receiptPath)
		if e != nil {
			return fail(e)
		}
		var receipt toolkitReceipt
		if e = json.Unmarshal(raw, &receipt); e != nil {
			return fail(fmt.Errorf("existing receipt is invalid JSON: %w", e))
		}
		previous = &receipt
	} else if !errors.Is(e, os.ErrNotExist) {
		return fail(e)
	}
	paths, err := boundedPackages(source, input)
	if err != nil {
		return fail(err)
	}
	sort.Strings(paths)
	interval := a.HeartbeatInterval
	if interval <= 0 {
		interval = toolkitInventoryHeartbeatInterval
	}
	last := time.Time{}
	pulse := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.Heartbeat != nil && time.Since(last) >= interval {
			a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.SourceRef, input.ReceiptRef, "inspecting", len(paths)})
			last = time.Now()
		}
		return ctx.Err()
	}
	receipt := toolkitReceipt{Schema: "toolkit-package-inventory/v1", Byline: "Codex, 2026-10-04",
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Packages: []toolkitPackage{},
		Identical: [][]toolkitOccurrence{}, LegalValidation: "not performed; inspect substantive audit records separately", Request: input}
	var expanded int64
	var compressed int64
	var members int
	occurrences := map[string][]toolkitOccurrence{}
	order := []string{}
	complete := true
	for _, path := range paths {
		if err := pulse(); err != nil {
			return fail(err)
		}
		pkg, err := inspectToolkitPackage(path, input, &compressed, &expanded, &members, pulse)
		if err != nil {
			return fail(err)
		}
		receipt.Packages = append(receipt.Packages, pkg)
		complete = complete && pkg.Complete
		for _, member := range pkg.Members {
			if _, ok := occurrences[member.SHA256]; !ok {
				order = append(order, member.SHA256)
			}
			occurrences[member.SHA256] = append(occurrences[member.SHA256], toolkitOccurrence{pkg.Package, member.Path})
		}
	}
	for _, digest := range order {
		if len(occurrences[digest]) > 1 {
			receipt.Identical = append(receipt.Identical, occurrences[digest])
		}
	}
	if previous != nil {
		receipt.CreatedAt = previous.CreatedAt
		if previous.CreatedAt == "" || !reflect.DeepEqual(*previous, receipt) {
			return fail(errors.New("receipt replay verification failed: source inventory or request differs"))
		}
	} else {
		raw, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			return fail(err)
		}
		if len(raw)+1 > toolkitInventoryMaxReceiptBytes {
			return fail(errors.New("receipt exceeds 32 MiB output limit"))
		}
		if err := pulse(); err != nil {
			return fail(err)
		}
		f, err := os.OpenFile(receiptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fail(err)
		}
		_, writeErr := f.Write(append(raw, '\n'))
		if writeErr == nil {
			writeErr = f.Sync()
		}
		closeErr := f.Close()
		if writeErr != nil {
			return fail(writeErr)
		}
		if closeErr != nil {
			return fail(closeErr)
		}
	}
	if !complete {
		return fail(errors.New("one or more packages are incomplete; inspect receipt errors"))
	}
	if err := pulse(); err != nil {
		return fail(err)
	}
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.SourceRef, input.ReceiptRef, "complete", len(paths)})
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	return ToolkitPackageInventoryResult{input.ReceiptRef, len(paths)}, nil
}

// streamToolkitDigest hashes a stream with cancellation and a strict byte ceiling.
// Inputs: reader, maximum bytes and liveness callback; outputs: digest and bytes read.
// Effects: reads bounded chunks only. Choose for archive and member integrity, never custody registration.
func streamToolkitDigest(r io.Reader, limit int64, pulse func() error) (string, int64, error) {
	hash := sha256.New()
	buf := make([]byte, 64*1024)
	var count int64
	for {
		if err := pulse(); err != nil {
			return "", count, err
		}
		want := int64(len(buf))
		if limit-count+1 < want {
			want = limit - count + 1
		}
		n, err := r.Read(buf[:int(want)])
		count += int64(n)
		if count > limit {
			return "", count, errors.New("actual bytes exceed inspection budget")
		}
		if n > 0 {
			_, _ = hash.Write(buf[:n])
		}
		if err == io.EOF {
			return hex.EncodeToString(hash.Sum(nil)), count, nil
		}
		if err != nil {
			return "", count, err
		}
		if n == 0 {
			return "", count, io.ErrNoProgress
		}
	}
}

// inspectToolkitPackage checks one ZIP using shared aggregate expansion and member budgets.
// Inputs: path, request budgets, counters and cancellation callback; outputs: compatible package receipt.
// Effects: reads source only and records unsafe paths, CRC errors and duplicates.
// Choose for comparison fingerprints rather than extraction or importing archive members.
func inspectToolkitPackage(path string, input ToolkitPackageInventoryInput, compressed, expanded *int64, members *int, pulse func() error) (toolkitPackage, error) {
	pkg := toolkitPackage{Package: filepath.Base(path), Members: []toolkitMember{}, Errors: []toolkitIssue{}}
	issue := func(name string, err error) {
		if len(name) > 1024 {
			name = name[:1024]
		}
		pkg.Errors = append(pkg.Errors, toolkitIssue{name, err.Error()})
	}
	if *expanded > input.MaxExpandedBytes {
		issue("", errors.New("aggregate expansion budget exhausted"))
		return pkg, nil
	}
	if *compressed > input.MaxArchiveBytes {
		issue("", errors.New("aggregate compressed budget exhausted"))
		return pkg, nil
	}
	f, err := os.Open(path)
	if err != nil {
		issue("", err)
		return pkg, nil
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return pkg, err
	}
	digest, compressedCount, err := streamToolkitDigest(f, input.MaxArchiveBytes-*compressed, pulse)
	*compressed += compressedCount
	if err != nil {
		if pulseErr := pulse(); pulseErr != nil {
			return pkg, pulseErr
		}
		issue("", err)
		return pkg, nil
	}
	if compressedCount != before.Size() {
		issue("", errors.New("source size changed during hashing"))
		return pkg, nil
	}
	pkg.SHA256 = &digest
	reader := &toolkitReaderAt{reader: f, pulse: pulse, remaining: 16 << 20, metadata: true}
	archive, err := zip.NewReader(reader, before.Size())
	reader.metadata = false
	if err != nil {
		issue("", err)
		return pkg, nil
	}
	if len(archive.File) > input.MaxMembers-*members {
		issue("", errors.New("archive exceeds aggregate member budget"))
		return pkg, nil
	}
	*members += len(archive.File)
	var declared uint64
	for _, entry := range archive.File {
		if entry.UncompressedSize64 > uint64(input.MaxExpandedBytes-*expanded)-declared {
			issue("", errors.New("archive exceeds aggregate expansion budget"))
			return pkg, nil
		}
		declared += entry.UncompressedSize64
	}
	seen := map[string]bool{}
	for _, entry := range archive.File {
		if err := pulse(); err != nil {
			return pkg, err
		}
		name := entry.Name
		normal := strings.ReplaceAll(name, "\\", "/")
		unsafe := len(name) > 1024 || !utf8.ValidString(name) || strings.ContainsRune(name, '\x00') || strings.HasPrefix(normal, "/") || strings.Contains(normal, ":")
		for _, part := range strings.Split(normal, "/") {
			unsafe = unsafe || part == ".."
		}
		if unsafe {
			issue(name, errors.New("unsafe member path"))
			continue
		}
		if seen[name] {
			issue(name, errors.New("duplicate member path"))
		}
		seen[name] = true
		if entry.FileInfo().IsDir() {
			continue
		}
		stream, err := entry.Open()
		if err != nil {
			issue(name, err)
			continue
		}
		digest, count, readErr := streamToolkitDigest(stream, input.MaxExpandedBytes-*expanded, pulse)
		closeErr := stream.Close()
		*expanded += count
		if err := pulse(); err != nil {
			return pkg, err
		}
		if readErr != nil {
			issue(name, readErr)
			if *expanded > input.MaxExpandedBytes {
				break
			}
			continue
		}
		if closeErr != nil {
			issue(name, closeErr)
			continue
		}
		if uint64(count) != entry.UncompressedSize64 {
			issue(name, errors.New("member size differs from declared size"))
			continue
		}
		candidate := false
		for _, word := range []string{"audit", "ledger", "validation", "verification", "changelog"} {
			candidate = candidate || strings.Contains(strings.ToLower(name), word)
		}
		pkg.Members = append(pkg.Members, toolkitMember{name, count, digest, count == 0, candidate})
	}
	after, err := f.Stat()
	if err != nil {
		return pkg, err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return pkg, err
	}
	if !os.SameFile(before, current) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		issue("", errors.New("source changed during inspection"))
	}
	pkg.Complete = len(pkg.Errors) == 0
	return pkg, nil
}

type toolkitReaderAt struct {
	reader    io.ReaderAt
	pulse     func() error
	remaining int
	metadata  bool
}

// ReadAt checks cancellation and bounds ZIP metadata reads before allocating member inventories.
// Inputs: requested slice and offset; outputs: source bytes or a limit/cancellation error.
// Effects: reads source and reports heartbeat. Choose over a raw file for ZIP central-directory reads.
func (r *toolkitReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.pulse(); err != nil {
		return 0, err
	}
	if r.metadata {
		if len(p) > r.remaining {
			return 0, errors.New("ZIP metadata exceeds 16 MiB read limit")
		}
		r.remaining -= len(p)
	}
	return r.reader.ReadAt(p, off)
}

// validate rejects missing references, non-positive budgets, and values above the server-side hard ceilings.
// Inputs: the caller-supplied toolkit inventory request.
// Outputs: nil only when every explicit package, byte, member, and directory-entry budget is bounded.
// Side effects: none.
// Choose this check before resolving paths or streaming ZIP inspection.
func (input ToolkitPackageInventoryInput) validate() error {
	if strings.TrimSpace(string(input.SourceRef)) == "" || strings.TrimSpace(string(input.ReceiptRef)) == "" {
		return errors.New("toolkit inventory requires source and receipt references")
	}
	if len(input.SourceRef) > 4096 || len(input.ReceiptRef) > 4096 {
		return errors.New("references exceed 4096 bytes")
	}
	if input.MaxPackages < 1 || input.MaxPackages > toolkitInventoryMaxPackages {
		return fmt.Errorf("max_packages must be between 1 and %d", toolkitInventoryMaxPackages)
	}
	if input.MaxArchiveBytes < 1 || input.MaxArchiveBytes > toolkitInventoryMaxBytes {
		return fmt.Errorf("max_archive_bytes must be between 1 and %d", toolkitInventoryMaxBytes)
	}
	if input.MaxExpandedBytes < 1 || input.MaxExpandedBytes > toolkitInventoryMaxBytes {
		return fmt.Errorf("max_expanded_bytes must be between 1 and %d", toolkitInventoryMaxBytes)
	}
	if input.MaxMembers < 1 || input.MaxMembers > toolkitInventoryMaxMembers {
		return fmt.Errorf("max_members must be between 1 and %d", toolkitInventoryMaxMembers)
	}
	if input.MaxDirectoryEntries < 1 || input.MaxDirectoryEntries > toolkitInventoryMaxDirEntries || input.MaxPackages > input.MaxDirectoryEntries {
		return fmt.Errorf("max_directory_entries must be between max_packages and %d", toolkitInventoryMaxDirEntries)
	}
	return nil
}

// canonicalDirectory resolves the configured root to an existing absolute directory.
// Inputs: an absolute host path supplied by worker configuration.
// Outputs: its cleaned canonical path or an error when it is missing, relative, or not a directory.
// Side effects: reads filesystem metadata only.
// Choose this helper to anchor both source and receipt references to one server-owned root.
func canonicalDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("allowed root must be an absolute path")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("allowed root must be an existing directory")
	}
	return filepath.Clean(resolved), nil
}

// resolveFileRef accepts one absolute file:// reference contained by the configured root.
// Inputs: a file locator, canonical allowed root, and whether the target must already exist as a directory.
// Outputs: a canonical filesystem path contained beneath the root.
// Side effects: reads path metadata; it never creates or changes a file.
// Choose this helper for server-local inputs and exclusive receipt destinations, not network or object-store URIs.
func resolveFileRef(ref proffer.Ref, root string, sourceDirectory bool) (string, error) {
	parsed, err := url.Parse(string(ref))
	if err != nil {
		return "", err
	}
	path := filepath.FromSlash(parsed.Path)
	if len(path) > 1 && path[0] == os.PathSeparator && filepath.VolumeName(path[1:]) != "" {
		path = path[1:]
	}
	if parsed.Scheme != "file" || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !filepath.IsAbs(path) {
		return "", errors.New("reference must be an absolute file:// path without host, query, or fragment")
	}
	clean := filepath.Clean(path)
	if sourceDirectory {
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return "", err
		}
		clean = filepath.Clean(resolved)
		info, err := os.Stat(clean)
		if err != nil || !info.IsDir() {
			return "", errors.New("source reference must name an existing directory")
		}
	} else {
		parent, err := filepath.EvalSymlinks(filepath.Dir(clean))
		if err != nil {
			return "", errors.New("receipt parent directory must already exist")
		}
		clean = filepath.Join(parent, filepath.Base(clean))
		if strings.ToLower(filepath.Ext(clean)) != ".json" {
			return "", errors.New("receipt reference must use a .json filename")
		}
	}
	relative, err := filepath.Rel(root, clean)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("reference must be strictly beneath the configured allowed root")
	}
	return clean, nil
}

// boundedPackages checks a capped directory listing and total compressed ZIP bytes before native inspection.
// Inputs: an existing source directory and the explicit package, archive-byte, and directory-entry budgets.
// Outputs: the lowercase .zip paths and an error for excess entries, unsafe package files, or exhausted budgets.
// Side effects: reads directory entries and file metadata only; archive contents are not opened.
// Choose this preflight to bound source enumeration before reading archive contents.
func boundedPackages(source string, input ToolkitPackageInventoryInput) ([]string, error) {
	directory, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("open toolkit package directory: %w", err)
	}
	defer directory.Close()
	names, err := directory.Readdirnames(input.MaxDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read toolkit package directory: %w", err)
	}
	if len(names) > input.MaxDirectoryEntries {
		return nil, fmt.Errorf("toolkit package directory exceeds max_directory_entries=%d", input.MaxDirectoryEntries)
	}
	packages := make([]string, 0, input.MaxPackages)
	var totalArchiveBytes int64
	for _, name := range names {
		if !strings.HasSuffix(name, ".zip") {
			continue
		}
		if len(packages) == input.MaxPackages {
			return nil, fmt.Errorf("toolkit package directory exceeds max_packages=%d", input.MaxPackages)
		}
		path := filepath.Join(source, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("toolkit archive %q must be a regular file", name)
		}
		if info.Size() < 0 || info.Size() > input.MaxArchiveBytes-totalArchiveBytes {
			return nil, fmt.Errorf("toolkit archives exceed max_archive_bytes=%d", input.MaxArchiveBytes)
		}
		totalArchiveBytes += info.Size()
		packages = append(packages, path)
	}
	if len(packages) == 0 {
		return nil, errors.New("toolkit package directory contains no lowercase .zip archives")
	}
	return packages, nil
}
