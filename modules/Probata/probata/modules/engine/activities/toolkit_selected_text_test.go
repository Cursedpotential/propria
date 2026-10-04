// Byline: Codex, 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// selectedDigest fingerprints fixture bytes for pinned requests.
// Inputs: bytes; outputs: lowercase SHA-256. Effects: none.
// Choose for deterministic fixture expectations without external hash tools.
func selectedDigest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// selectedFixture creates a real inventory and a selected-text request for retained tiny sources.
// Inputs: test handle and exact member names/text; outputs: Activity, request and fixture root.
// Effects: exclusively creates retained sources and inventory receipt. Choose real provenance over mocked inventory.
func selectedFixture(t *testing.T, names, texts []string) (ToolkitPackageInventoryActivities, ToolkitSelectedTextInput, string) {
	t.Helper()
	a, inventory, root := toolkitFixture(t)
	inventory.MaxArchiveBytes = 8 << 20
	inventory.MaxExpandedBytes = 8 << 20
	toolkitWrite(t, filepath.Join(root, "source", "audit.zip"), toolkitZIP(t, names, texts, false))
	if _, err := a.RunToolkitPackageInventory(context.Background(), inventory); err != nil {
		t.Fatal(err)
	}
	receipt := toolkitRead(t, root)
	raw, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	input := ToolkitSelectedTextInput{InventoryReceiptRef: inventory.ReceiptRef, ExpectedInventoryReceiptSHA256: selectedDigest(raw),
		OutputSnapshotRef:  proffer.Ref(strings.Replace(string(inventory.ReceiptRef), "receipt.json", "snapshot.json", 1)),
		MaxSelectedMembers: 64, MaxArchiveBytes: 8 << 20, MaxMemberBytes: 1 << 20, MaxTotalTextBytes: 4 << 20, MaxOutputBytes: 32 << 20}
	for _, member := range receipt.Packages[0].Members {
		input.Selections = append(input.Selections, ToolkitTextSelection{"audit.zip", *receipt.Packages[0].SHA256, member.Path, member.SHA256, member.Bytes})
	}
	return a, input, root
}

// selectedReplace retains the original fixture before exclusively creating a changed version.
// Inputs: path and replacement bytes; outputs: none. Effects: renames original to a preserved sibling.
// Choose to test source changes without deleting or overwriting originals.
func selectedReplace(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.Rename(path, path+".preserved"); err != nil {
		t.Fatal(err)
	}
	toolkitWrite(t, path, raw)
}

// selectedRepinReceipt records a deliberately changed fixture authority for runtime integrity tests.
// Inputs: fixture root, request and edited receipt; outputs: updated receipt fingerprint.
// Effects: preserves original receipt then creates changed fixture. Choose to exercise streaming checks beyond preflight.
func selectedRepinReceipt(t *testing.T, root string, input *ToolkitSelectedTextInput, receipt toolkitReceipt) {
	t.Helper()
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	selectedReplace(t, filepath.Join(root, "receipt.json"), raw)
	input.ExpectedInventoryReceiptSHA256 = selectedDigest(raw)
}

// selectedSnapshot reads an immutable retained output and its exact bytes.
// Inputs: fixture root; outputs: decoded snapshot and bytes. Effects: bounded fixture read.
// Choose for text/provenance assertions rather than output existence.
func selectedSnapshot(t *testing.T, root string) (toolkitTextSnapshot, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var snapshot toolkitTextSnapshot
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot, raw
}

// TestToolkitSelectedTextReplay preserves original text and deduplicates archive hashing across selections.
// Inputs: two selected members with BOM, CRLF, Unicode and JSON-sensitive text; outputs: immutable replay assertions.
// Effects: retained fixtures. Choose to prove exact text, provenance and one ZIP hash per execution.
func TestToolkitSelectedTextReplay(t *testing.T) {
	original := "\ufeffAudit\r\n<source> \"quote\" \\ path é\n"
	a, input, root := selectedFixture(t, []string{"Source Audit.md", "nested/ledger.json"}, []string{original, "{\"valid\":true}\r\n"})
	info, err := os.Stat(filepath.Join(root, "source", "audit.zip"))
	if err != nil {
		t.Fatal(err)
	}
	input.MaxArchiveBytes = info.Size() // Both selections must share exactly one archive hashing budget.
	result, err := a.SnapshotSelectedToolkitText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, before := selectedSnapshot(t, root)
	if snapshot.Members[0].Text != original || snapshot.Members[1].Text != "{\"valid\":true}\r\n" ||
		!snapshot.Complete || snapshot.Request.ExpectedInventoryReceiptSHA256 != input.ExpectedInventoryReceiptSHA256 ||
		result.SnapshotSHA256 != selectedDigest(before) || result.PackageCount != 1 || result.ArchiveBytes != info.Size() || result.MemberCount != 2 {
		t.Fatal("text, provenance or archive deduplication mismatch")
	}
	replay, err := a.SnapshotSelectedToolkitText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	_, after := selectedSnapshot(t, root)
	if !bytes.Equal(before, after) || result != replay {
		t.Fatal("replay changed output or result")
	}
	selectedReplace(t, filepath.Join(root, "source", "audit.zip"), toolkitZIP(t, []string{"Source Audit.md", "nested/ledger.json"}, []string{"changed", "{\"valid\":true}\r\n"}, false))
	if _, err = a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
		t.Fatal("changed ZIP accepted on replay")
	}
	_, after = selectedSnapshot(t, root)
	if !bytes.Equal(before, after) {
		t.Fatal("failed replay changed snapshot")
	}
}

// TestToolkitSelectedTextProvenance refuses unpinned, incomplete, absent or mismatched selections.
// Inputs: modified requests against real receipt; outputs: failures with no output.
// Effects: retained fixtures only. Choose to prove receipt authority before source text is released.
func TestToolkitSelectedTextProvenance(t *testing.T) {
	for _, kind := range []string{"receipt-pin", "member-hash", "package-hash", "missing-member", "wrong-size", "incomplete", "duplicate-selection", "unsafe-path"} {
		t.Run(kind, func(t *testing.T) {
			a, input, root := selectedFixture(t, []string{"audit.md"}, []string{"original"})
			switch kind {
			case "receipt-pin":
				input.ExpectedInventoryReceiptSHA256 = strings.Repeat("0", 64)
			case "member-hash":
				input.Selections[0].ExpectedMemberSHA256 = strings.Repeat("0", 64)
			case "package-hash":
				input.Selections[0].ExpectedPackageSHA256 = strings.Repeat("0", 64)
			case "missing-member":
				input.Selections[0].MemberName = "not-in-receipt.md"
			case "wrong-size":
				input.Selections[0].ExpectedMemberBytes++
			case "incomplete":
				receipt := toolkitRead(t, root)
				receipt.Packages[0].Complete = false
				selectedRepinReceipt(t, root, &input, receipt)
			case "duplicate-selection":
				input.Selections = append(input.Selections, input.Selections[0])
			case "unsafe-path":
				input.Selections[0].MemberName = "../audit.md"
			}
			if _, err := a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
				t.Fatal("invalid provenance accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "snapshot.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("failed provenance created output")
			}
		})
	}
}

// TestToolkitSelectedTextStreamingChecks independently enforces CRC, UTF-8 and actual member SHA-256.
// Inputs: receipts pinned to deliberate test mismatches or malformed text; outputs: visible streaming failures.
// Effects: retains original and changed fixtures. Choose to prove receipt matching cannot bypass source integrity checks.
func TestToolkitSelectedTextStreamingChecks(t *testing.T) {
	for _, kind := range []string{"actual-member-hash", "crc", "utf8", "duplicate-zip-member"} {
		t.Run(kind, func(t *testing.T) {
			text := "unique-source-payload"
			if kind == "utf8" {
				text = string([]byte{0xff, 0xfe})
			}
			a, input, root := selectedFixture(t, []string{"audit.md"}, []string{text})
			receipt := toolkitRead(t, root)
			switch kind {
			case "actual-member-hash":
				receipt.Packages[0].Members[0].SHA256 = strings.Repeat("0", 64)
				input.Selections[0].ExpectedMemberSHA256 = strings.Repeat("0", 64)
				selectedRepinReceipt(t, root, &input, receipt)
			case "crc", "duplicate-zip-member":
				var raw []byte
				if kind == "crc" {
					raw = toolkitZIP(t, []string{"audit.md"}, []string{text}, true)
				} else {
					raw = toolkitZIP(t, []string{"audit.md", "audit.md"}, []string{text, text}, false)
				}
				selectedReplace(t, filepath.Join(root, "source", "audit.zip"), raw)
				digest := selectedDigest(raw)
				receipt.Packages[0].SHA256 = &digest
				input.Selections[0].ExpectedPackageSHA256 = digest
				selectedRepinReceipt(t, root, &input, receipt)
			}
			if _, err := a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
				t.Fatal("streaming integrity failure accepted")
			}
		})
	}
}

// TestToolkitSelectedTextBudgets enforces declared/request/serialized limits without truncating sources.
// Inputs: tiny real sources and lowered or excessive ceilings; outputs: failures.
// Effects: retained fixtures. Choose to prove each independently configured byte/count boundary.
func TestToolkitSelectedTextBudgets(t *testing.T) {
	for _, kind := range []string{"archive", "member", "total", "output", "count", "hard-member", "hard-total", "hard-output", "hard-count"} {
		t.Run(kind, func(t *testing.T) {
			a, input, _ := selectedFixture(t, []string{"a.md", "b.md"}, []string{"123456", "123456"})
			switch kind {
			case "archive":
				input.MaxArchiveBytes = 1
			case "member":
				input.MaxMemberBytes = 5
			case "total":
				input.MaxTotalTextBytes = 11
			case "output":
				input.MaxOutputBytes = 1
			case "count":
				input.MaxSelectedMembers = 1
			case "hard-member":
				input.MaxMemberBytes = (1 << 20) + 1
			case "hard-total":
				input.MaxTotalTextBytes = (4 << 20) + 1
			case "hard-output":
				input.MaxOutputBytes = (32 << 20) + 1
			case "hard-count":
				input.MaxSelectedMembers = 65
			}
			if _, err := a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
				t.Fatal("budget violation accepted")
			}
		})
	}
}

// TestToolkitSelectedTextLedgerSize accepts the actual largest planned ledger without truncation.
// Inputs: 553992-byte ledger matching the parent selection size; outputs: unchanged complete text.
// Effects: retained fixture and snapshot. Choose to guard against the superseded 512 KiB limit.
func TestToolkitSelectedTextLedgerSize(t *testing.T) {
	text := strings.Repeat("x", 553992)
	a, input, root := selectedFixture(t, []string{"ledger.json"}, []string{text})
	result, err := a.SnapshotSelectedToolkitText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := selectedSnapshot(t, root)
	if result.TextBytes != 553992 || snapshot.Members[0].Text != text {
		t.Fatal("ledger truncated or changed")
	}
}

// TestToolkitSelectedTextCancellation checks mid-archive cancellation and Activity heartbeat cancellation.
// Inputs: bounded ledger source and cancel callbacks; outputs: cancellation with source retained and no snapshot.
// Effects: retained fixtures. Choose to prove streaming unwinds without asynchronous subprocesses.
func TestToolkitSelectedTextCancellation(t *testing.T) {
	a, input, root := selectedFixture(t, []string{"ledger.json"}, []string{strings.Repeat("x", 553992)})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, _, err := toolkitReadSelectedPackage(filepath.Join(root, "source", "audit.zip"), input, []int{0}, input.MaxArchiveBytes, func() error {
		calls++
		if calls == 3 {
			cancel()
		}
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("archive read not canceled: %v", err)
	}
	ctx, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	a.Heartbeat = func(context.Context, ToolkitPackageInventoryHeartbeat) { cancel2() }
	if _, err = a.SnapshotSelectedToolkitText(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("Activity heartbeat cancellation failed: %v", err)
	}
	if _, err = os.Stat(filepath.Join(root, "snapshot.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled inspection created snapshot")
	}
}

// TestToolkitSelectedTextPartialReplay retains and rejects an incomplete existing output.
// Inputs: valid request and partial snapshot; outputs: failure without replacement.
// Effects: retains partial fixture. Choose to prove interrupted writes cannot masquerade as completed replay.
func TestToolkitSelectedTextPartialReplay(t *testing.T) {
	a, input, root := selectedFixture(t, []string{"audit.md"}, []string{"original"})
	partial := []byte("{\"schema\":")
	toolkitWrite(t, filepath.Join(root, "snapshot.json"), partial)
	if _, err := a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
		t.Fatal("partial replay accepted")
	}
	after, err := os.ReadFile(filepath.Join(root, "snapshot.json"))
	if err != nil || !bytes.Equal(after, partial) {
		t.Fatal("partial output replaced")
	}
}

// TestToolkitSelectedTextWorkflow runs real selected-text extraction with Temporal heartbeat delivery.
// Inputs: tiny fixture and registered Activity; outputs: immutable reference-only workflow result.
// Effects: retained fixture snapshot, no live server. Choose for existing-worker scheduling integration.
func TestToolkitSelectedTextWorkflow(t *testing.T) {
	_, input, root := selectedFixture(t, []string{"audit.md"}, []string{"original"})
	a := NewToolkitPackageInventoryActivities(root)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var beats atomic.Int32
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, _ converter.EncodedValues) { beats.Add(1) })
	env.RegisterActivityWithOptions(a.SnapshotSelectedToolkitText, activity.RegisterOptions{Name: ToolkitSelectedTextActivityName})
	env.ExecuteWorkflow(ToolkitSelectedTextWorkflow, input)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result ToolkitSelectedTextResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.MemberCount != 1 || result.SnapshotRef != input.OutputSnapshotRef || beats.Load() == 0 {
		t.Fatal("workflow result or heartbeats missing")
	}
}

// TestToolkitSelectedTextMultipleArchives enforces the aggregate hash budget across distinct ZIPs.
// Inputs: two retained archives and a pinned updated inventory; outputs: aggregate rejection then success.
// Effects: retains both inventories and one snapshot. Choose to distinguish archive deduplication from undercounting.
func TestToolkitSelectedTextMultipleArchives(t *testing.T) {
	a, input, root := selectedFixture(t, []string{"audit.md"}, []string{"original"})
	second := toolkitZIP(t, []string{"ledger.json"}, []string{"second"}, false)
	toolkitWrite(t, filepath.Join(root, "source", "second.zip"), second)
	inventory := toolkitRead(t, root).Request
	inventory.ReceiptRef = proffer.Ref(strings.Replace(string(inventory.ReceiptRef), "receipt.json", "receipt-next.json", 1))
	if _, err := a.RunToolkitPackageInventory(context.Background(), inventory); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "receipt-next.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt toolkitReceipt
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	pkg := receipt.Packages[1]
	member := pkg.Members[0]
	input.InventoryReceiptRef = inventory.ReceiptRef
	input.ExpectedInventoryReceiptSHA256 = selectedDigest(raw)
	input.Selections = append(input.Selections, ToolkitTextSelection{pkg.Package, *pkg.SHA256, member.Path, member.SHA256, member.Bytes})
	first, err := os.Stat(filepath.Join(root, "source", "audit.zip"))
	if err != nil {
		t.Fatal(err)
	}
	input.MaxArchiveBytes = first.Size()
	if _, err = a.SnapshotSelectedToolkitText(context.Background(), input); err == nil {
		t.Fatal("second archive exceeded aggregate budget without failing")
	}
	input.MaxArchiveBytes += int64(len(second))
	result, err := a.SnapshotSelectedToolkitText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageCount != 2 || result.ArchiveBytes != input.MaxArchiveBytes {
		t.Fatal("distinct archive hashing undercounted")
	}
}

// TestToolkitSelectedTextWorkflowFailure makes pinned authority mismatch a visible workflow failure.
// Inputs: real fixture with incorrect receipt pin; outputs: failed workflow without a snapshot.
// Effects: retains fixtures only. Choose to prove nonretryable integrity errors propagate through Temporal.
func TestToolkitSelectedTextWorkflowFailure(t *testing.T) {
	_, input, root := selectedFixture(t, []string{"audit.md"}, []string{"original"})
	input.ExpectedInventoryReceiptSHA256 = strings.Repeat("0", 64)
	a := NewToolkitPackageInventoryActivities(root)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(a.SnapshotSelectedToolkitText, activity.RegisterOptions{Name: ToolkitSelectedTextActivityName})
	env.ExecuteWorkflow(ToolkitSelectedTextWorkflow, input)
	if env.GetWorkflowError() == nil {
		t.Fatal("pinned receipt failure reported as successful workflow")
	}
	if _, err := os.Stat(filepath.Join(root, "snapshot.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed workflow created output")
	}
}
