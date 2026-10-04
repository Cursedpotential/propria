// Byline: Codex, 2026-10-04.
package activities

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// toolkitFixture preserves tiny test sources and receipts in the worktree quarantine.
// Inputs: test handle; outputs: configured activity, bounded request and fixture root.
// Effects: creates fixtures without automatic deletion. Choose over TempDir under the owner retention rule.
func toolkitFixture(t *testing.T) (ToolkitPackageInventoryActivities, ToolkitPackageInventoryInput, string) {
	t.Helper()
	base, err := filepath.Abs("../../../../../../to_be_deleted/toolkit-inventory-tests")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(base, "run-")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("preserved fixture:", root)
	source := filepath.Join(root, "source")
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	ref := func(path string) proffer.Ref {
		path = filepath.ToSlash(path)
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return proffer.Ref((&url.URL{Scheme: "file", Path: path}).String())
	}
	input := ToolkitPackageInventoryInput{ref(source), ref(filepath.Join(root, "receipt.json")), 4, 1 << 20, 1 << 20, 20, 30}
	return ToolkitPackageInventoryActivities{AllowedRoot: root}, input, root
}

// toolkitZIP builds tiny stored-member ZIP fixtures, including malformed member names.
// Inputs: names, payloads and optional CRC corruption; outputs: archive bytes.
// Effects: none. Choose for deterministic integrity tests without large corpora.
func toolkitZIP(t *testing.T, names, contents []string, corrupt bool) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for i, name := range names {
		w, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(contents[i])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	raw := data.Bytes()
	if corrupt {
		index := bytes.Index(raw, []byte(contents[0]))
		if index < 0 {
			t.Fatal("payload missing")
		}
		raw[index] ^= 1
	}
	return raw
}

// toolkitWrite preserves an exclusively created source fixture.
// Inputs: path and bytes; outputs: none. Effects: creates one file.
// Choose to avoid overwriting another fixture.
func toolkitWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

// toolkitRead decodes a preserved receipt for behavioral assertions.
// Inputs: fixture root; outputs: receipt. Effects: reads JSON only.
// Choose for receipt-content validation rather than checking file existence.
func toolkitRead(t *testing.T, root string) toolkitReceipt {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt toolkitReceipt
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	return receipt
}

// TestToolkitInventoryReplay verifies hashes, duplicate candidates and unchanged replay bytes.
// Inputs: tiny ZIPs; outputs: assertions. Effects: retained fixtures.
// Choose to prove retry behavior and Python-compatible core receipt fields.
func TestToolkitInventoryReplay(t *testing.T) {
	a, input, root := toolkitFixture(t)
	for _, name := range []string{"a.zip", "b.zip"} {
		toolkitWrite(t, filepath.Join(root, "source", name), toolkitZIP(t, []string{"audit.txt", "empty.txt"}, []string{"hello", ""}, false))
	}
	result, err := a.RunToolkitPackageInventory(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageCount != 2 || result.ReceiptRef != input.ReceiptRef {
		t.Fatalf("unexpected result: %#v", result)
	}
	receipt := toolkitRead(t, root)
	digest := sha256.Sum256([]byte("hello"))
	if receipt.Schema != "toolkit-package-inventory/v1" || len(receipt.Identical) != 2 || !receipt.Packages[0].Complete ||
		receipt.Packages[0].Members[0].SHA256 != hex.EncodeToString(digest[:]) || !receipt.Packages[0].Members[0].AuditCandidate ||
		!receipt.Packages[0].Members[1].Empty {
		t.Fatalf("invalid receipt: %#v", receipt)
	}
	before, _ := os.ReadFile(filepath.Join(root, "receipt.json"))
	if _, err = a.RunToolkitPackageInventory(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "receipt.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("replay rewrote receipt")
	}
	original := filepath.Join(root, "source", "a.zip")
	if err = os.Rename(original, original+".preserved"); err != nil {
		t.Fatal(err)
	}
	toolkitWrite(t, original, toolkitZIP(t, []string{"audit.txt"}, []string{"changed"}, false))
	if _, err = a.RunToolkitPackageInventory(context.Background(), input); err == nil || !strings.Contains(err.Error(), "replay verification failed") {
		t.Fatalf("source mutation accepted: %v", err)
	}
	after, _ = os.ReadFile(filepath.Join(root, "receipt.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed replay overwrote receipt")
	}
}

// TestToolkitInventoryFailureReceipts verifies corruption and unsafe/duplicate paths stay visible.
// Inputs: corrupt or unsafe small ZIP; outputs: incomplete receipt assertions.
// Effects: retained JSON and source fixtures. Choose to distinguish integrity failures from successful inventories.
func TestToolkitInventoryFailureReceipts(t *testing.T) {
	for _, kind := range []string{"crc", "unsafe", "duplicate", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			a, input, root := toolkitFixture(t)
			var raw []byte
			switch kind {
			case "crc":
				raw = toolkitZIP(t, []string{"data.txt"}, []string{"unique-crc-payload"}, true)
			case "unsafe":
				raw = toolkitZIP(t, []string{"../escape", "C:drive", "/absolute"}, []string{"x", "y", "z"}, false)
			case "duplicate":
				raw = toolkitZIP(t, []string{"same", "same"}, []string{"x", "x"}, false)
			case "invalid":
				raw = []byte("not a ZIP")
			}
			toolkitWrite(t, filepath.Join(root, "source", "a.zip"), raw)
			if _, err := a.RunToolkitPackageInventory(context.Background(), input); err == nil {
				t.Fatal("failure accepted")
			}
			receipt := toolkitRead(t, root)
			if receipt.Packages[0].Complete || len(receipt.Packages[0].Errors) == 0 {
				t.Fatal("failure hidden")
			}
			before, _ := os.ReadFile(filepath.Join(root, "receipt.json"))
			if _, err := a.RunToolkitPackageInventory(context.Background(), input); err == nil {
				t.Fatal("incomplete replay accepted")
			}
			after, _ := os.ReadFile(filepath.Join(root, "receipt.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("incomplete replay changed receipt")
			}
		})
	}
}

// TestToolkitInventoryBounds verifies enumeration, compression, expansion and aggregate member ceilings.
// Inputs: tiny sources above selected budgets; outputs: explicit failure assertions.
// Effects: retained fixtures. Choose to prove bounded behavior across multiple packages.
func TestToolkitInventoryBounds(t *testing.T) {
	for _, kind := range []string{"packages", "directory", "compressed", "expanded", "members", "root", "receipt", "unconfigured"} {
		t.Run(kind, func(t *testing.T) {
			a, input, root := toolkitFixture(t)
			for _, name := range []string{"a.zip", "b.zip"} {
				toolkitWrite(t, filepath.Join(root, "source", name), toolkitZIP(t, []string{"data"}, []string{"123456"}, false))
			}
			switch kind {
			case "packages":
				input.MaxPackages = 1
			case "directory":
				input.MaxPackages = 1
				input.MaxDirectoryEntries = 1
			case "compressed":
				input.MaxArchiveBytes = 1
			case "expanded":
				input.MaxExpandedBytes = 8
			case "members":
				input.MaxMembers = 1
			case "root":
				input.SourceRef = proffer.Ref("file:///outside")
			case "receipt":
				toolkitWrite(t, filepath.Join(root, "receipt.json"), []byte("{"))
			case "unconfigured":
				a.AllowedRoot = ""
			}
			if _, err := a.RunToolkitPackageInventory(context.Background(), input); err == nil {
				t.Fatal("bounded failure accepted")
			}
		})
	}
}

// TestToolkitInventoryCancellation verifies interruption during streaming without a child process.
// Inputs: stream and callback that cancels after two chunks; outputs: context cancellation.
// Effects: reads memory only. Choose to prove mid-stream cancellation beyond pre-canceled requests.
func TestToolkitInventoryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, _, err := streamToolkitDigest(bytes.NewReader(bytes.Repeat([]byte("x"), 256<<10)), 1<<20, func() error {
		calls++
		if calls == 3 {
			cancel()
		}
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stream not canceled: %v", err)
	}
}

// TestToolkitInventoryContainment rejects existing directories outside the worker root and linked sources.
// Inputs: tiny local fixtures and symlinks when supported; outputs: boundary failures.
// Effects: retains all fixtures and links. Choose to prove canonical root enforcement beyond missing-path errors.
func TestToolkitInventoryContainment(t *testing.T) {
	a, input, root := toolkitFixture(t)
	input.SourceRef = proffer.Ref(string(input.SourceRef) + "/../..")
	if _, err := a.RunToolkitPackageInventory(context.Background(), input); err == nil || !strings.Contains(err.Error(), "beneath") {
		t.Fatalf("outside directory accepted: %v", err)
	}
	_, input, _ = toolkitFixture(t)
	input.ReceiptRef = proffer.Ref(strings.Replace(string(input.ReceiptRef), "receipt.json", "../outside.json", 1))
	if _, err := a.RunToolkitPackageInventory(context.Background(), input); err == nil {
		t.Fatal("outside receipt accepted")
	}
	source := filepath.Join(root, "source")
	archive := filepath.Join(root, "retained-source.zip")
	toolkitWrite(t, archive, toolkitZIP(t, []string{"data"}, []string{"hello"}, false))
	if err := os.Symlink(archive, filepath.Join(source, "linked.zip")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := boundedPackages(source, ToolkitPackageInventoryInput{MaxPackages: 4, MaxDirectoryEntries: 30, MaxArchiveBytes: 1 << 20}); err == nil {
		t.Fatal("linked archive accepted")
	}
}

// TestToolkitInventoryExhaustedBudgets rejects an already exhausted stream budget without a slice panic.
// Inputs: retained tiny ZIP and exhausted aggregate counters; outputs: incomplete package.
// Effects: retained source fixture. Choose to verify recovery after a previous package exceeded actual bytes.
func TestToolkitInventoryExhaustedBudgets(t *testing.T) {
	_, input, root := toolkitFixture(t)
	path := filepath.Join(root, "source", "a.zip")
	toolkitWrite(t, path, toolkitZIP(t, []string{"data"}, []string{"hello"}, false))
	compressed, expanded, members := input.MaxArchiveBytes+1, int64(0), 0
	pkg, err := inspectToolkitPackage(path, input, &compressed, &expanded, &members, func() error { return nil })
	if err != nil || pkg.Complete || len(pkg.Errors) == 0 {
		t.Fatalf("exhausted budget accepted: %#v, %v", pkg, err)
	}
}

// TestToolkitInventoryActivityCancellation interrupts the real Activity through its heartbeat callback.
// Inputs: tiny ZIP and cancellation on its first heartbeat; outputs: canceled result without a receipt.
// Effects: retains source fixture only. Choose to verify heartbeat cancellation unwinds the Activity synchronously.
func TestToolkitInventoryActivityCancellation(t *testing.T) {
	a, input, root := toolkitFixture(t)
	raw := toolkitZIP(t, []string{"data"}, []string{"hello"}, false)
	path := filepath.Join(root, "source", "a.zip")
	toolkitWrite(t, path, raw)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.HeartbeatInterval = time.Nanosecond
	a.Heartbeat = func(context.Context, ToolkitPackageInventoryHeartbeat) { cancel() }
	if _, err := a.RunToolkitPackageInventory(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("Activity not canceled: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "receipt.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled inspection created a receipt")
	}
	retained, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(raw, retained) {
		t.Fatal("cancellation changed source")
	}
}

// TestToolkitInventoryWorkflowFailure exposes incomplete ZIP integrity as a workflow failure.
// Inputs: unsafe ZIP and real Activity in the test worker; outputs: failed workflow and preserved receipt.
// Effects: retained fixtures, no live server. Choose to prove errors cannot appear as successful execution.
func TestToolkitInventoryWorkflowFailure(t *testing.T) {
	_, input, root := toolkitFixture(t)
	toolkitWrite(t, filepath.Join(root, "source", "a.zip"), toolkitZIP(t, []string{"../unsafe"}, []string{"hello"}, false))
	a := NewToolkitPackageInventoryActivities(root)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(a.RunToolkitPackageInventory, activity.RegisterOptions{Name: ToolkitPackageInventoryActivityName})
	env.ExecuteWorkflow(ToolkitPackageInventoryWorkflow, input)
	if env.GetWorkflowError() == nil {
		t.Fatal("incomplete inventory workflow succeeded")
	}
	if toolkitRead(t, root).Packages[0].Complete {
		t.Fatal("failure receipt reports complete")
	}
}

// TestToolkitInventoryWorkflow executes the registered single Activity with real Temporal heartbeat context.
// Inputs: tiny ZIP and test worker environment; outputs: bounded result and recorded heartbeats.
// Effects: retained fixture receipt, no live service. Choose for scheduling and callback integration.
func TestToolkitInventoryWorkflow(t *testing.T) {
	a, input, root := toolkitFixture(t)
	toolkitWrite(t, filepath.Join(root, "source", "a.zip"), toolkitZIP(t, []string{"hello"}, []string{"world"}, false))
	a = NewToolkitPackageInventoryActivities(root)
	a.HeartbeatInterval = time.Nanosecond
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var beats atomic.Int32
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, _ converter.EncodedValues) { beats.Add(1) })
	env.RegisterActivityWithOptions(a.RunToolkitPackageInventory, activity.RegisterOptions{Name: ToolkitPackageInventoryActivityName})
	env.ExecuteWorkflow(ToolkitPackageInventoryWorkflow, input)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result ToolkitPackageInventoryResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, ToolkitPackageInventoryResult{input.ReceiptRef, 1}) {
		t.Fatalf("unexpected result: %#v", result)
	}
	if beats.Load() == 0 {
		t.Fatal("Temporal received no heartbeat")
	}
}
