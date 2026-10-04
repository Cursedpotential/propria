// Byline: Codex, 2026-10-04.
package activities

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitSelectedTextActivityName = "toolkit_selected_text_activity"
	ToolkitSelectedTextWorkflowName = "ToolkitSelectedTextWorkflow"
	toolkitSelectedMaxMembers       = 64
	toolkitSelectedMaxMemberBytes   = 1 << 20
	toolkitSelectedMaxTextBytes     = 4 << 20
)

// ToolkitTextSelection pins one exact ZIP member to its inventoried bytes.
// Inputs: package/member names, SHA-256 values and member byte length.
// Outputs: a verified selection; effects: none.
// Choose explicit identities instead of filename heuristics or archive-wide extraction.
type ToolkitTextSelection struct {
	PackageName           string `json:"package_name"`
	ExpectedPackageSHA256 string `json:"expected_package_sha256"`
	MemberName            string `json:"exact_member_name"`
	ExpectedMemberSHA256  string `json:"expected_member_sha256"`
	ExpectedMemberBytes   int64  `json:"expected_member_bytes"`
}

// ToolkitSelectedTextInput specifies a pinned inventory and bounded ZIP-only text selections.
// Inputs: receipt fingerprint, explicit selections, output reference and positive ceilings.
// Outputs: an immutable snapshot reference; effects: reads sources and exclusively writes JSON.
// Choose after inventory for substantive source review, separately from inventory or dataset import.
type ToolkitSelectedTextInput struct {
	InventoryReceiptRef            proffer.Ref            `json:"inventory_receipt_ref"`
	ExpectedInventoryReceiptSHA256 string                 `json:"expected_inventory_receipt_sha256"`
	Selections                     []ToolkitTextSelection `json:"selections"`
	OutputSnapshotRef              proffer.Ref            `json:"output_snapshot_ref"`
	MaxSelectedMembers             int                    `json:"max_selected_members"`
	MaxArchiveBytes                int64                  `json:"max_archive_bytes"`
	MaxMemberBytes                 int64                  `json:"max_member_bytes"`
	MaxTotalTextBytes              int64                  `json:"max_total_text_bytes"`
	MaxOutputBytes                 int64                  `json:"max_output_bytes"`
}

// ToolkitSelectedTextResult returns only immutable snapshot coordinates through Temporal.
// Inputs: verified snapshot digest and counts; outputs: references, SHA-256 and bounded metrics.
// Effects: none. Choose instead of placing selected text bodies in workflow history.
type ToolkitSelectedTextResult struct {
	SnapshotRef    proffer.Ref `json:"snapshot_ref"`
	SnapshotSHA256 string      `json:"snapshot_sha256"`
	MemberCount    int         `json:"member_count"`
	PackageCount   int         `json:"package_count"`
	TextBytes      int64       `json:"text_bytes"`
	ArchiveBytes   int64       `json:"archive_bytes"`
}
type toolkitSelectedMember struct {
	Selection ToolkitTextSelection `json:"selection"`
	Text      string               `json:"text"`
}
type toolkitTextSnapshot struct {
	Schema   string                   `json:"schema"`
	Byline   string                   `json:"byline"`
	Complete bool                     `json:"complete"`
	Request  ToolkitSelectedTextInput `json:"request"`
	Members  []toolkitSelectedMember  `json:"members"`
}

// ToolkitSelectedTextWorkflow schedules the separate bounded snapshot Activity on the existing worker.
// Inputs: pinned ZIP-only selections; outputs: immutable snapshot coordinates or visible failure.
// Effects: Temporal scheduling only. Choose for source-text snapshots rather than package inventory.
func ToolkitSelectedTextWorkflow(ctx workflow.Context, input ToolkitSelectedTextInput) (ToolkitSelectedTextResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, ScheduleToCloseTimeout: time.Hour, HeartbeatTimeout: time.Minute,
		WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	var result ToolkitSelectedTextResult
	err := workflow.ExecuteActivity(ctx, ToolkitSelectedTextActivityName, input).Get(ctx, &result)
	return result, err
}

// validateSelectedText checks exact identities and aggregate ceilings before any filesystem reads.
// Inputs: proposed snapshot request; outputs: nil or explicit invalid-input error.
// Effects: none. Choose before receipt verification, archive reads or output creation.
func validateSelectedText(input ToolkitSelectedTextInput) error {
	if len(input.InventoryReceiptRef) == 0 || len(input.InventoryReceiptRef) > 4096 || len(input.OutputSnapshotRef) == 0 || len(input.OutputSnapshotRef) > 4096 {
		return errors.New("receipt and output references must be present and at most 4096 bytes")
	}
	if !toolkitSelectedSHA256(input.ExpectedInventoryReceiptSHA256) {
		return errors.New("inventory receipt SHA-256 must be lowercase hexadecimal")
	}
	if input.MaxSelectedMembers < 1 || input.MaxSelectedMembers > toolkitSelectedMaxMembers ||
		len(input.Selections) < 1 || len(input.Selections) > input.MaxSelectedMembers {
		return errors.New("selection count exceeds explicit ceiling or 64-member maximum")
	}
	if input.MaxArchiveBytes < 1 || input.MaxArchiveBytes > toolkitInventoryMaxBytes ||
		input.MaxMemberBytes < 1 || input.MaxMemberBytes > toolkitSelectedMaxMemberBytes ||
		input.MaxTotalTextBytes < 1 || input.MaxTotalTextBytes > toolkitSelectedMaxTextBytes ||
		input.MaxOutputBytes < 1 || input.MaxOutputBytes > toolkitInventoryMaxReceiptBytes {
		return errors.New("byte ceilings must be positive and within archive/member/text/output hard limits")
	}
	seen := map[string]bool{}
	packages := map[string]string{}
	var total int64
	for _, s := range input.Selections {
		if !toolkitSelectedSafeName(s.PackageName) || strings.ContainsAny(s.PackageName, "/\\") ||
			!strings.HasSuffix(s.PackageName, ".zip") || !toolkitSelectedSafeName(s.MemberName) {
			return errors.New("selection requires a safe exact ZIP basename and member path")
		}
		if !toolkitSelectedSHA256(s.ExpectedPackageSHA256) || !toolkitSelectedSHA256(s.ExpectedMemberSHA256) {
			return errors.New("selection SHA-256 values must be lowercase hexadecimal")
		}
		if s.ExpectedMemberBytes < 0 || s.ExpectedMemberBytes > input.MaxMemberBytes || s.ExpectedMemberBytes > input.MaxTotalTextBytes-total {
			return errors.New("selected sizes exceed member or total text budget")
		}
		total += s.ExpectedMemberBytes
		key := s.PackageName + "\x00" + s.MemberName
		if seen[key] {
			return errors.New("duplicate member selection")
		}
		seen[key] = true
		if prior, ok := packages[s.PackageName]; ok && prior != s.ExpectedPackageSHA256 {
			return errors.New("conflicting fingerprints for one package")
		}
		packages[s.PackageName] = s.ExpectedPackageSHA256
	}
	return nil
}

// toolkitSelectedSHA256 validates an unambiguous expected fingerprint.
// Inputs: digest text; outputs: whether it is exactly 64 lowercase hexadecimal characters.
// Effects: none. Choose for receipt and source expectations rather than accepting absent fingerprints.
func toolkitSelectedSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// toolkitSelectedSafeName rejects traversal and ambiguous source coordinates.
// Inputs: exact archive or member name; outputs: safe relative UTF-8 name status.
// Effects: none. Choose for selected source coordinates; output filenames are never derived from members.
func toolkitSelectedSafeName(name string) bool {
	if name == "" || len(name) > 1024 || !utf8.ValidString(name) || strings.ContainsRune(name, '\x00') || strings.Contains(name, ":") {
		return false
	}
	normal := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normal, "/") || strings.HasSuffix(normal, "/") {
		return false
	}
	for _, part := range strings.Split(normal, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// toolkitReadBoundedJSON reads a regular nonlinked file with hashing and cancellation.
// Inputs: canonical path, byte ceiling and progress callback; outputs: bytes and SHA-256.
// Effects: bounded read only. Choose for pinned receipts and replay snapshots before trusting JSON.
func toolkitReadBoundedJSON(path string, limit int64, pulse func() error) ([]byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, "", errors.New("JSON input is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	current, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(info, current) {
		return nil, "", errors.New("JSON input changed while opening")
	}
	var data bytes.Buffer
	digest, _, err := streamToolkitDigest(io.TeeReader(f, &data), limit, pulse)
	if err != nil {
		return nil, "", err
	}
	return data.Bytes(), digest, nil
}

// toolkitValidateSelections requires complete, uniquely inventoried package/member identities.
// Inputs: pinned decoded inventory and explicit selections; outputs: nil or provenance mismatch.
// Effects: none. Choose before reading selected sources so no unlisted member can enter the snapshot.
func toolkitValidateSelections(receipt toolkitReceipt, selections []ToolkitTextSelection) error {
	if receipt.Schema != "toolkit-package-inventory/v1" || len(receipt.Packages) > toolkitInventoryMaxPackages {
		return errors.New("unsupported or oversized inventory receipt")
	}
	for _, s := range selections {
		packageMatches, memberMatches := 0, 0
		for _, pkg := range receipt.Packages {
			if pkg.Package != s.PackageName {
				continue
			}
			packageMatches++
			if !pkg.Complete || len(pkg.Errors) != 0 || pkg.SHA256 == nil || *pkg.SHA256 != s.ExpectedPackageSHA256 {
				return fmt.Errorf("package provenance mismatch or incomplete inventory: %s", s.PackageName)
			}
			for _, member := range pkg.Members {
				if member.Path != s.MemberName {
					continue
				}
				memberMatches++
				if member.SHA256 != s.ExpectedMemberSHA256 || member.Bytes != s.ExpectedMemberBytes {
					return fmt.Errorf("member provenance mismatch: %s/%s", s.PackageName, s.MemberName)
				}
			}
		}
		if packageMatches != 1 || memberMatches != 1 {
			return fmt.Errorf("selection is not uniquely present in inventory: %s/%s", s.PackageName, s.MemberName)
		}
	}
	return nil
}

// toolkitReadSelectedPackage hashes one distinct ZIP and snapshots only its exact selected members.
// Inputs: source path, request, selection indices, remaining archive budget and progress callback.
// Outputs: verified original UTF-8 text per index and archive bytes hashed; effects: source reads only.
// Choose the FetchZipMember streaming pattern with SHA-256 and local-root provenance instead of contacts import.
func toolkitReadSelectedPackage(path string, input ToolkitSelectedTextInput, indices []int, remaining int64, pulse func() error) (map[int]string, int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > remaining {
		return nil, 0, errors.New("ZIP is not a regular file within aggregate archive budget")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !os.SameFile(info, before) {
		return nil, 0, errors.New("ZIP changed while opening")
	}
	digest, count, err := streamToolkitDigest(f, remaining, pulse)
	if err != nil {
		return nil, count, err
	}
	if count != before.Size() || digest != input.Selections[indices[0]].ExpectedPackageSHA256 {
		return nil, count, errors.New("ZIP SHA-256 or size changed since inventory")
	}
	reader := &toolkitReaderAt{reader: f, pulse: pulse, remaining: 16 << 20, metadata: true}
	archive, err := zip.NewReader(reader, before.Size())
	reader.metadata = false
	if err != nil {
		return nil, count, err
	}
	if len(archive.File) > toolkitInventoryMaxMembers {
		return nil, count, errors.New("ZIP exceeds metadata member ceiling")
	}
	wanted := map[string]int{}
	found := map[int]*zip.File{}
	for _, index := range indices {
		wanted[input.Selections[index].MemberName] = index
	}
	for _, member := range archive.File {
		if err := pulse(); err != nil {
			return nil, count, err
		}
		if index, ok := wanted[member.Name]; ok {
			if _, exists := found[index]; exists {
				return nil, count, errors.New("selected ZIP member name is duplicated")
			}
			found[index] = member
		}
	}
	texts := map[int]string{}
	for _, index := range indices {
		selection := input.Selections[index]
		member, ok := found[index]
		if !ok {
			return nil, count, fmt.Errorf("selected member missing: %s", selection.MemberName)
		}
		if member.FileInfo().IsDir() || member.UncompressedSize64 != uint64(selection.ExpectedMemberBytes) {
			return nil, count, errors.New("selected member size or type differs from inventory")
		}
		stream, err := member.Open()
		if err != nil {
			return nil, count, err
		}
		var data bytes.Buffer
		digest, n, readErr := streamToolkitDigest(io.TeeReader(stream, &data), selection.ExpectedMemberBytes, pulse)
		closeErr := stream.Close()
		if readErr != nil {
			return nil, count, readErr
		}
		if closeErr != nil {
			return nil, count, closeErr
		}
		if n != selection.ExpectedMemberBytes || digest != selection.ExpectedMemberSHA256 {
			return nil, count, errors.New("selected member SHA-256 or size mismatch")
		}
		if !utf8.Valid(data.Bytes()) {
			return nil, count, errors.New("selected member is not valid UTF-8")
		}
		texts[index] = data.String()
	}
	after, err := f.Stat()
	if err != nil {
		return nil, count, err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, count, err
	}
	if !current.Mode().IsRegular() || !os.SameFile(before, current) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, count, errors.New("ZIP changed during selected-text inspection")
	}
	return texts, count, nil
}

// SnapshotSelectedToolkitText writes an exclusive provenance snapshot of selected original text.
// Inputs: pinned inventory, exact ZIP/member expectations and explicit budgets.
// Outputs: immutable JSON snapshot reference, SHA-256 and counts, or a visible nonretryable integrity failure.
// Effects: reads staged ZIPs and writes one new snapshot; existing or partial outputs are preserved.
// Choose after inventory for selected audit/changelog/ledger review; no source normalization or dataset writes occur.
func (a ToolkitPackageInventoryActivities) SnapshotSelectedToolkitText(ctx context.Context, input ToolkitSelectedTextInput) (ToolkitSelectedTextResult, error) {
	fail := func(err error) (ToolkitSelectedTextResult, error) {
		if ctx.Err() != nil {
			return ToolkitSelectedTextResult{}, ctx.Err()
		}
		return ToolkitSelectedTextResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("selected text failed; snapshot preserved if created at %s: %v", input.OutputSnapshotRef, err), "ToolkitSelectedTextFailed", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := validateSelectedText(input); err != nil {
		return fail(err)
	}
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return fail(err)
	}
	receiptPath, err := resolveFileRef(input.InventoryReceiptRef, root, false)
	if err != nil {
		return fail(err)
	}
	outputPath, err := resolveFileRef(input.OutputSnapshotRef, root, false)
	if err != nil {
		return fail(err)
	}
	if receiptPath == outputPath {
		return fail(errors.New("output must differ from inventory receipt"))
	}
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
			a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.InventoryReceiptRef, input.OutputSnapshotRef, "selected_text", len(input.Selections)})
			last = time.Now()
		}
		return ctx.Err()
	}
	raw, digest, err := toolkitReadBoundedJSON(receiptPath, toolkitInventoryMaxReceiptBytes, pulse)
	if err != nil {
		return fail(err)
	}
	if digest != input.ExpectedInventoryReceiptSHA256 {
		return fail(errors.New("pinned inventory receipt SHA-256 mismatch"))
	}
	var receipt toolkitReceipt
	if err = json.Unmarshal(raw, &receipt); err != nil {
		return fail(err)
	}
	if err = toolkitValidateSelections(receipt, input.Selections); err != nil {
		return fail(err)
	}
	source, err := resolveFileRef(receipt.Request.SourceRef, root, true)
	if err != nil {
		return fail(err)
	}
	var previous []byte
	if _, err = os.Lstat(outputPath); err == nil {
		previous, _, err = toolkitReadBoundedJSON(outputPath, input.MaxOutputBytes, pulse)
		if err != nil {
			return fail(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fail(err)
	}
	groups := map[string][]int{}
	order := []string{}
	for index, selection := range input.Selections {
		if _, ok := groups[selection.PackageName]; !ok {
			order = append(order, selection.PackageName)
		}
		groups[selection.PackageName] = append(groups[selection.PackageName], index)
	}
	snapshot := toolkitTextSnapshot{Schema: "toolkit-selected-text/v1", Byline: "Codex, 2026-10-04", Complete: true, Request: input, Members: make([]toolkitSelectedMember, len(input.Selections))}
	var archiveBytes, textBytes int64
	for _, name := range order {
		texts, count, err := toolkitReadSelectedPackage(filepath.Join(source, name), input, groups[name], input.MaxArchiveBytes-archiveBytes, pulse)
		if err != nil {
			return fail(err)
		}
		archiveBytes += count
		for index, text := range texts {
			textBytes += int64(len(text))
			if textBytes > input.MaxTotalTextBytes {
				return fail(errors.New("actual text bytes exceed aggregate budget"))
			}
			snapshot.Members[index] = toolkitSelectedMember{input.Selections[index], text}
		}
	}
	raw, err = json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fail(err)
	}
	if int64(len(raw)+1) > input.MaxOutputBytes {
		return fail(errors.New("serialized snapshot exceeds output budget"))
	}
	raw = append(raw, '\n')
	if err = pulse(); err != nil {
		return fail(err)
	}
	if previous != nil {
		if !bytes.Equal(previous, raw) {
			return fail(errors.New("snapshot replay provenance or content mismatch"))
		}
	} else {
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return fail(err)
		}
		n, writeErr := f.Write(raw)
		if writeErr == nil && n != len(raw) {
			writeErr = io.ErrShortWrite
		}
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
	digest, _, err = streamToolkitDigest(bytes.NewReader(raw), input.MaxOutputBytes, pulse)
	if err != nil {
		return fail(err)
	}
	if a.Heartbeat != nil {
		a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.InventoryReceiptRef, input.OutputSnapshotRef, "selected_text_complete", len(input.Selections)})
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return ToolkitSelectedTextResult{input.OutputSnapshotRef, digest, len(input.Selections), len(order), textBytes, archiveBytes}, nil
}
