// Byline: Codex, 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

// ledgerFixture produces a real pinned selected-text snapshot for synthetic ledger versions.
// Inputs: ledger JSON texts; outputs: configured group, comparison request and retained fixture root.
// Effects: creates retained ZIP, inventory and snapshot fixtures. Choose actual sibling outputs over fabricated schemas.
func ledgerFixture(t *testing.T, texts []string) (ToolkitPackageInventoryActivities, ToolkitLedgerComparisonInput, string) {
	t.Helper()
	names := make([]string, len(texts))
	indices := make([]int, len(texts))
	for i := range texts {
		names[i] = fmt.Sprintf("ledger-%d.json", i)
		indices[i] = i
	}
	a, selected, root := selectedFixture(t, names, texts)
	result, err := a.SnapshotSelectedToolkitText(context.Background(), selected)
	if err != nil {
		t.Fatal(err)
	}
	input := ToolkitLedgerComparisonInput{SnapshotRef: result.SnapshotRef, ExpectedSnapshotSHA256: result.SnapshotSHA256,
		LedgerIndices: indices, MaxRows: 1000, MaxOutputBytes: 4 << 20,
		OutputRef: proffer.Ref(strings.Replace(string(result.SnapshotRef), "snapshot.json", "comparison.json", 1))}
	return a, input, root
}

// ledgerRead decodes a retained comparison without changing it.
// Inputs: fixture root; outputs: comparison and original bytes. Effects: reads fixture only.
// Choose for deterministic output/provenance assertions.
func ledgerRead(t *testing.T, root string) (toolkitLedgerComparison, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "comparison.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result toolkitLedgerComparison
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result, raw
}

// TestToolkitLedgerComparisonDifferences emits sorted added/removed IDs and differing field names only.
// Inputs: three synthetic versions including nested objects and missing/null fields; outputs: pairwise differences.
// Effects: retained fixtures. Choose to prove mechanical differences without copying source values or choosing a version.
func TestToolkitLedgerComparisonDifferences(t *testing.T) {
	left := `[{"id":"same","nested":{"x":1,"y":2}},{"id":"changed","value":"left-secret","nullable":null},{"id":"removed"}]`
	right := `[{"nested":{"y":2,"x":1},"id":"same"},{"id":"changed","value":"right-secret","extra":false},{"id":"added"}]`
	a, input, root := ledgerFixture(t, []string{left, right, left})
	result, err := a.CompareToolkitLedgers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	comparison, raw := ledgerRead(t, root)
	if result.CandidateCount != 3 || result.PairCount != 3 || result.ComparisonSHA256 != selectedDigest(raw) {
		t.Fatal("result counts/digest mismatch")
	}
	pair := comparison.Pairs[0]
	if pair.LeftMemberIndex != 0 || pair.RightMemberIndex != 1 ||
		!reflect.DeepEqual(pair.AddedIDs, []string{"added"}) || !reflect.DeepEqual(pair.RemovedIDs, []string{"removed"}) ||
		!reflect.DeepEqual(pair.DifferingFields, []toolkitLedgerFieldDifference{{"changed", []string{"extra", "nullable", "value"}}}) {
		t.Fatalf("unexpected mechanical differences: %#v", pair)
	}
	equal := comparison.Pairs[1]
	if len(equal.AddedIDs)+len(equal.RemovedIDs)+len(equal.DifferingFields) != 0 {
		t.Fatal("equal versions differed")
	}
	for _, candidate := range comparison.Candidates {
		if candidate.RowCount != 3 || candidate.Selection.ExpectedMemberSHA256 != comparison.Candidates[0].Selection.ExpectedMemberSHA256 && candidate.MemberIndex == 2 {
			t.Fatal("candidate count/root identity changed")
		}
	}
	if bytes.Contains(raw, []byte("left-secret")) || bytes.Contains(raw, []byte("right-secret")) {
		t.Fatal("source field values leaked into comparison")
	}
	before := append([]byte(nil), raw...)
	input.LedgerIndices = []int{2, 0, 1}
	replay, err := a.CompareToolkitLedgers(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	_, after := ledgerRead(t, root)
	if replay != result || !bytes.Equal(before, after) {
		t.Fatal("reordered indices changed deterministic replay")
	}
}

// TestToolkitLedgerComparisonActualIndices supports the parent's explicit members 14, 15 and 16.
// Inputs: fourteen unselected text members and three ledger members; outputs: correct evidence pointers.
// Effects: retained small fixtures. Choose to prove indexing uses the selected snapshot rather than a separate ledger list.
func TestToolkitLedgerComparisonActualIndices(t *testing.T) {
	texts := make([]string, 17)
	for i := 0; i < 14; i++ {
		texts[i] = "unselected audit text"
	}
	for i := 14; i < 17; i++ {
		texts[i] = `[{"id":"entry"}]`
	}
	a, input, root := ledgerFixture(t, texts)
	input.LedgerIndices = []int{14, 15, 16}
	if _, err := a.CompareToolkitLedgers(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	result, _ := ledgerRead(t, root)
	for i, candidate := range result.Candidates {
		if candidate.MemberIndex != 14+i || candidate.RowCount != 1 {
			t.Fatal("evidence indices changed")
		}
	}
}

// TestToolkitLedgerComparisonInvalidRows rejects ambiguous IDs and non-array ledger structures.
// Inputs: duplicate/empty/missing IDs and invalid array rows; outputs: visible failures.
// Effects: retained fixtures only. Choose strict source interpretation instead of repairing ledger JSON.
func TestToolkitLedgerComparisonInvalidRows(t *testing.T) {
	cases := map[string]string{
		"duplicate": `[{"id":"x"},{"id":"x"}]`, "empty": `[{"id":" "}]`, "missing": `[{"name":"x"}]`,
		"numeric": `[{"id":1}]`, "object": `{"id":"x"}`, "null-row": `[null]`, "trailing": `[{"id":"x"}] []`,
	}
	for kind, text := range cases {
		t.Run(kind, func(t *testing.T) {
			a, input, root := ledgerFixture(t, []string{text, `[]`})
			_, err := a.CompareToolkitLedgers(context.Background(), input)
			if err == nil {
				t.Fatal("invalid ledger accepted")
			}
			if kind == "duplicate" && !strings.Contains(err.Error(), "duplicate ledger id") {
				t.Fatalf("duplicate identity failure hidden: %v", err)
			}
			if _, err = os.Stat(filepath.Join(root, "comparison.json")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid rows created output")
			}
		})
	}
}

// TestToolkitLedgerComparisonBounds validates source pins, member indices and row/output ceilings.
// Inputs: small source arrays and altered request limits; outputs: explicit failures.
// Effects: retained fixtures only. Choose to prove independent request and result budgets.
func TestToolkitLedgerComparisonBounds(t *testing.T) {
	for _, kind := range []string{"pin", "duplicate-index", "negative-index", "outside-index", "rows", "output", "hard-rows", "hard-output"} {
		t.Run(kind, func(t *testing.T) {
			a, input, _ := ledgerFixture(t, []string{`[{"id":"a"},{"id":"b"}]`, `[]`})
			switch kind {
			case "pin":
				input.ExpectedSnapshotSHA256 = strings.Repeat("0", 64)
			case "duplicate-index":
				input.LedgerIndices = []int{0, 0}
			case "negative-index":
				input.LedgerIndices = []int{-1, 1}
			case "outside-index":
				input.LedgerIndices = []int{0, 2}
			case "rows":
				input.MaxRows = 1
			case "output":
				input.MaxOutputBytes = 1
			case "hard-rows":
				input.MaxRows = 1001
			case "hard-output":
				input.MaxOutputBytes = (4 << 20) + 1
			}
			if _, err := a.CompareToolkitLedgers(context.Background(), input); err == nil {
				t.Fatal("invalid request or budget accepted")
			}
		})
	}
}

// TestToolkitLedgerComparisonReplayMismatch retains old output and rejects changed output/source provenance.
// Inputs: completed comparison followed by changed request, partial output or changed pinned source.
// Outputs: failures without replacement. Effects: preserves every original fixture.
// Choose to prove retries never silently bless changed comparison inputs.
func TestToolkitLedgerComparisonReplayMismatch(t *testing.T) {
	for _, kind := range []string{"request", "partial", "source"} {
		t.Run(kind, func(t *testing.T) {
			a, input, root := ledgerFixture(t, []string{`[{"id":"a"}]`, `[{"id":"a"}]`})
			if _, err := a.CompareToolkitLedgers(context.Background(), input); err != nil {
				t.Fatal(err)
			}
			_, before := ledgerRead(t, root)
			switch kind {
			case "request":
				input.MaxRows = 999
			case "partial":
				before = []byte("{")
				selectedReplace(t, filepath.Join(root, "comparison.json"), before)
			case "source":
				raw, err := os.ReadFile(filepath.Join(root, "snapshot.json"))
				if err != nil {
					t.Fatal(err)
				}
				selectedReplace(t, filepath.Join(root, "snapshot.json"), append(raw, ' '))
			}
			if _, err := a.CompareToolkitLedgers(context.Background(), input); err == nil {
				t.Fatal("changed replay accepted")
			}
			after, err := os.ReadFile(filepath.Join(root, "comparison.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed replay changed output")
			}
		})
	}
}

// TestToolkitLedgerComparisonNumericPrecision preserves distinct JSON numeric lexemes without float rounding.
// Inputs: adjacent integers above float64 exactness; outputs: differing numeric field name.
// Effects: reads memory only. Choose to prove mechanical differences do not silently collapse large values.
func TestToolkitLedgerComparisonNumericPrecision(t *testing.T) {
	pulse := func() error { return nil }
	left, err := parseToolkitLedger(`[{"id":"x","number":9007199254740992}]`, 1000, pulse)
	if err != nil {
		t.Fatal(err)
	}
	right, err := parseToolkitLedger(`[{"id":"x","number":9007199254740993}]`, 1000, pulse)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := compareToolkitLedgerRows(left, right, 14, 15, pulse)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pair.DifferingFields, []toolkitLedgerFieldDifference{{"x", []string{"number"}}}) {
		t.Fatal("large numbers rounded into equality")
	}
}

// TestToolkitLedgerComparisonWorkflow verifies scheduling, heartbeat delivery and source-preserving failure.
// Inputs: tiny pinned snapshot and real registered Activity; outputs: reference-only result and heartbeat.
// Effects: retained comparison fixture, no live service. Choose to prove the existing worker execution seam.
func TestToolkitLedgerComparisonWorkflow(t *testing.T) {
	_, input, root := ledgerFixture(t, []string{`[]`, `[]`})
	a := NewToolkitPackageInventoryActivities(root)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	var beats atomic.Int32
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, _ converter.EncodedValues) { beats.Add(1) })
	env.RegisterActivityWithOptions(a.CompareToolkitLedgers, activity.RegisterOptions{Name: ToolkitLedgerComparisonActivityName})
	env.ExecuteWorkflow(ToolkitLedgerComparisonWorkflow, input)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var result ToolkitLedgerComparisonResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.PairCount != 1 || result.ComparisonRef != input.OutputRef || beats.Load() == 0 {
		t.Fatal("workflow result or heartbeat missing")
	}
}

// TestToolkitLedgerComparisonCancellation checks cancellation inside row parsing and at Activity heartbeat.
// Inputs: memory ledger and retained snapshot with cancel callbacks; outputs: context cancellation.
// Effects: retains fixtures without output. Choose to prove parsing and Activity execution unwind synchronously.
func TestToolkitLedgerComparisonCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := parseToolkitLedger(`[{"id":"a"}]`, 1000, func() error { cancel(); return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatal("ledger parser ignored cancellation")
	}
	a, input, root := ledgerFixture(t, []string{`[]`, `[]`})
	ctx, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	a.Heartbeat = func(context.Context, ToolkitPackageInventoryHeartbeat) { cancel2() }
	if _, err = a.CompareToolkitLedgers(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatal("Activity ignored heartbeat cancellation")
	}
	if _, err = os.Stat(filepath.Join(root, "comparison.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled Activity wrote output")
	}
}
