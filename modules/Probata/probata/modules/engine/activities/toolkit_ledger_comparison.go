// Byline: Codex, 2026-10-04.
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/proffer"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ToolkitLedgerComparisonActivityName = "toolkit_ledger_comparison_activity"
	ToolkitLedgerComparisonWorkflowName = "ToolkitLedgerComparisonWorkflow"
	toolkitLedgerMaxRows                = 1000
	toolkitLedgerMaxOutputBytes         = 4 << 20
)

// ToolkitLedgerComparisonInput pins selected-text ledger members for mechanical pairwise comparison.
// Inputs: snapshot reference/SHA-256, unique member indices, per-ledger row limit and output limit/reference.
// Outputs: immutable differences reference; effects: reads snapshot and exclusively writes JSON.
// Choose for version differences without interpreting legal status or selecting a surviving version.
type ToolkitLedgerComparisonInput struct {
	SnapshotRef            proffer.Ref `json:"snapshot_ref"`
	ExpectedSnapshotSHA256 string      `json:"expected_snapshot_sha256"`
	LedgerIndices          []int       `json:"ledger_indices"`
	MaxRows                int         `json:"max_rows"`
	MaxOutputBytes         int64       `json:"max_output_bytes"`
	OutputRef              proffer.Ref `json:"output_ref"`
}

// ToolkitLedgerComparisonResult carries bounded output coordinates through Temporal.
// Inputs: completed comparison digest/counts; outputs: reference, SHA-256 and candidate/pair counts.
// Effects: none. Choose instead of copying ledger values or comparison bodies into workflow history.
type ToolkitLedgerComparisonResult struct {
	ComparisonRef    proffer.Ref `json:"comparison_ref"`
	ComparisonSHA256 string      `json:"comparison_sha256"`
	CandidateCount   int         `json:"candidate_count"`
	PairCount        int         `json:"pair_count"`
}
type toolkitLedgerCandidate struct {
	MemberIndex int                  `json:"snapshot_member_index"`
	Selection   ToolkitTextSelection `json:"selection"`
	RowCount    int                  `json:"row_count"`
}
type toolkitLedgerFieldDifference struct {
	ID         string   `json:"id"`
	FieldNames []string `json:"field_names"`
}
type toolkitLedgerPair struct {
	LeftMemberIndex  int                            `json:"left_member_index"`
	RightMemberIndex int                            `json:"right_member_index"`
	AddedIDs         []string                       `json:"added_ids"`
	RemovedIDs       []string                       `json:"removed_ids"`
	DifferingFields  []toolkitLedgerFieldDifference `json:"differing_fields"`
}
type toolkitLedgerComparison struct {
	Schema     string                       `json:"schema"`
	Byline     string                       `json:"byline"`
	Request    ToolkitLedgerComparisonInput `json:"request"`
	Candidates []toolkitLedgerCandidate     `json:"candidates"`
	Pairs      []toolkitLedgerPair          `json:"pairs"`
}

// ToolkitLedgerComparisonWorkflow schedules the separate ledger comparison Activity on the existing worker.
// Inputs: pinned selected-text snapshot and bounded indices; outputs: comparison coordinates or visible failure.
// Effects: Temporal scheduling only. Choose after text snapshotting, without ZIP reads or dataset imports.
func ToolkitLedgerComparisonWorkflow(ctx workflow.Context, input ToolkitLedgerComparisonInput) (ToolkitLedgerComparisonResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute, ScheduleToCloseTimeout: 5 * time.Minute, HeartbeatTimeout: time.Minute,
		WaitForCancellation: true, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 2},
	})
	var result ToolkitLedgerComparisonResult
	err := workflow.ExecuteActivity(ctx, ToolkitLedgerComparisonActivityName, input).Get(ctx, &result)
	return result, err
}

// validateLedgerComparison checks pinned references, unique indices and hard ceilings.
// Inputs: comparison request; outputs: nil or explicit invalid-input error.
// Effects: none. Choose before reading snapshot or output files.
func validateLedgerComparison(input ToolkitLedgerComparisonInput) error {
	if len(input.SnapshotRef) == 0 || len(input.SnapshotRef) > 4096 || len(input.OutputRef) == 0 || len(input.OutputRef) > 4096 ||
		!toolkitSelectedSHA256(input.ExpectedSnapshotSHA256) {
		return errors.New("bounded snapshot/output references and lowercase SHA-256 are required")
	}
	if input.MaxRows < 1 || input.MaxRows > toolkitLedgerMaxRows || input.MaxOutputBytes < 1 || input.MaxOutputBytes > toolkitLedgerMaxOutputBytes {
		return errors.New("row/output ceilings must be positive and at most 1000 rows and 4 MiB")
	}
	if len(input.LedgerIndices) < 2 || len(input.LedgerIndices) > toolkitSelectedMaxMembers {
		return errors.New("comparison requires 2 through 64 explicit ledger indices")
	}
	seen := map[int]bool{}
	for _, index := range input.LedgerIndices {
		if index < 0 || index >= toolkitSelectedMaxMembers || seen[index] {
			return errors.New("ledger indices must be unique and within selected-member bounds")
		}
		seen[index] = true
	}
	return nil
}

// parseToolkitLedger streams a JSON array into unique nonempty string-ID rows with a strict count limit.
// Inputs: one selected ledger's text, row ceiling and progress callback; outputs: rows indexed by exact ID.
// Effects: parses memory only. Choose strict JSON, preserving numeric lexemes, rather than repairing or inferring missing identities.
func parseToolkitLedger(text string, maxRows int, pulse func() error) (map[string]map[string]interface{}, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('[') {
		return nil, errors.New("ledger must be a JSON array")
	}
	rows := map[string]map[string]interface{}{}
	for decoder.More() {
		if err := pulse(); err != nil {
			return nil, err
		}
		if len(rows) >= maxRows {
			return nil, errors.New("ledger exceeds row budget")
		}
		var row map[string]interface{}
		if err := decoder.Decode(&row); err != nil {
			return nil, err
		}
		id, ok := row["id"].(string)
		if !ok || strings.TrimSpace(id) == "" {
			return nil, errors.New("ledger row requires a nonempty string id")
		}
		if _, exists := rows[id]; exists {
			return nil, fmt.Errorf("duplicate ledger id: %s", id)
		}
		rows[id] = row
	}
	if _, err = decoder.Token(); err != nil {
		return nil, err
	}
	var extra interface{}
	if err = decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("ledger contains trailing JSON or invalid data")
	}
	return rows, nil
}

// compareToolkitLedgerRows computes directional added/removed IDs and top-level differing field names.
// Inputs: two ID-indexed ledgers and member indices; outputs: sorted mechanical differences.
// Effects: none beyond progress reporting. Choose structural JSON equality with numeric lexemes preserved; no field values are copied.
func compareToolkitLedgerRows(left, right map[string]map[string]interface{}, leftIndex, rightIndex int, pulse func() error) (toolkitLedgerPair, error) {
	pair := toolkitLedgerPair{leftIndex, rightIndex, []string{}, []string{}, []toolkitLedgerFieldDifference{}}
	for id, leftRow := range left {
		if err := pulse(); err != nil {
			return pair, err
		}
		rightRow, exists := right[id]
		if !exists {
			pair.RemovedIDs = append(pair.RemovedIDs, id)
			continue
		}
		fields := map[string]bool{}
		for name := range leftRow {
			fields[name] = true
		}
		for name := range rightRow {
			fields[name] = true
		}
		names := []string{}
		for name := range fields {
			leftValue, leftExists := leftRow[name]
			rightValue, rightExists := rightRow[name]
			if leftExists != rightExists || !reflect.DeepEqual(leftValue, rightValue) {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			sort.Strings(names)
			pair.DifferingFields = append(pair.DifferingFields, toolkitLedgerFieldDifference{id, names})
		}
	}
	for id := range right {
		if err := pulse(); err != nil {
			return pair, err
		}
		if _, exists := left[id]; !exists {
			pair.AddedIDs = append(pair.AddedIDs, id)
		}
	}
	sort.Strings(pair.AddedIDs)
	sort.Strings(pair.RemovedIDs)
	sort.Slice(pair.DifferingFields, func(i, j int) bool { return pair.DifferingFields[i].ID < pair.DifferingFields[j].ID })
	return pair, nil
}

// CompareToolkitLedgers compares pinned ledger arrays while retaining source-selection identities.
// Inputs: pinned complete snapshot, explicit member indices, row/output ceilings and new JSON destination.
// Outputs: deterministic candidate counts and pairwise IDs/field names with snapshot-index evidence pointers.
// Effects: reads snapshot only and exclusively creates output; verifies existing bytes on replay, retaining partials.
// Choose for mechanistic version differences; source selection, legal review and dataset writes remain separate operations.
func (a ToolkitPackageInventoryActivities) CompareToolkitLedgers(ctx context.Context, input ToolkitLedgerComparisonInput) (ToolkitLedgerComparisonResult, error) {
	fail := func(err error) (ToolkitLedgerComparisonResult, error) {
		if ctx.Err() != nil {
			return ToolkitLedgerComparisonResult{}, ctx.Err()
		}
		return ToolkitLedgerComparisonResult{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("ledger comparison failed; output preserved if created at %s: %v", input.OutputRef, err), "ToolkitLedgerComparisonFailed", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := validateLedgerComparison(input); err != nil {
		return fail(err)
	}
	input.LedgerIndices = append([]int(nil), input.LedgerIndices...)
	sort.Ints(input.LedgerIndices)
	root, err := canonicalDirectory(a.AllowedRoot)
	if err != nil {
		return fail(err)
	}
	snapshotPath, err := resolveFileRef(input.SnapshotRef, root, false)
	if err != nil {
		return fail(err)
	}
	outputPath, err := resolveFileRef(input.OutputRef, root, false)
	if err != nil {
		return fail(err)
	}
	if snapshotPath == outputPath {
		return fail(errors.New("output must differ from source snapshot"))
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
			a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.SnapshotRef, input.OutputRef, "ledger_comparison", len(input.LedgerIndices)})
			last = time.Now()
		}
		return ctx.Err()
	}
	raw, digest, err := toolkitReadBoundedJSON(snapshotPath, toolkitInventoryMaxReceiptBytes, pulse)
	if err != nil {
		return fail(err)
	}
	if digest != input.ExpectedSnapshotSHA256 {
		return fail(errors.New("pinned snapshot SHA-256 mismatch"))
	}
	var snapshot toolkitTextSnapshot
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		return fail(err)
	}
	if snapshot.Schema != "toolkit-selected-text/v1" || !snapshot.Complete || len(snapshot.Members) != len(snapshot.Request.Selections) {
		return fail(errors.New("snapshot schema/completeness/member provenance is invalid"))
	}
	if err = validateSelectedText(snapshot.Request); err != nil {
		return fail(err)
	}
	output := toolkitLedgerComparison{Schema: "toolkit-ledger-comparison/v1", Byline: "Codex, 2026-10-04", Request: input, Candidates: []toolkitLedgerCandidate{}, Pairs: []toolkitLedgerPair{}}
	rows := map[int]map[string]map[string]interface{}{}
	for _, index := range input.LedgerIndices {
		if index >= len(snapshot.Members) {
			return fail(errors.New("ledger index exceeds snapshot member count"))
		}
		member := snapshot.Members[index]
		if member.Selection != snapshot.Request.Selections[index] || int64(len(member.Text)) != member.Selection.ExpectedMemberBytes {
			return fail(errors.New("snapshot member provenance or text size mismatch"))
		}
		memberDigest, _, err := streamToolkitDigest(strings.NewReader(member.Text), toolkitSelectedMaxMemberBytes, pulse)
		if err != nil {
			return fail(err)
		}
		if memberDigest != member.Selection.ExpectedMemberSHA256 {
			return fail(errors.New("snapshot member SHA-256 mismatch"))
		}
		rows[index], err = parseToolkitLedger(member.Text, input.MaxRows, pulse)
		if err != nil {
			return fail(fmt.Errorf("snapshot member %d: %w", index, err))
		}
		output.Candidates = append(output.Candidates, toolkitLedgerCandidate{index, member.Selection, len(rows[index])})
	}
	// Bound accumulated pairs before retaining them; final serialization also accounts for metadata.
	var pairBytes int64
	for i, left := range input.LedgerIndices {
		for _, right := range input.LedgerIndices[i+1:] {
			pair, err := compareToolkitLedgerRows(rows[left], rows[right], left, right, pulse)
			if err != nil {
				return fail(err)
			}
			encoded, err := json.Marshal(pair)
			if err != nil {
				return fail(err)
			}
			pairBytes += int64(len(encoded))
			if pairBytes > input.MaxOutputBytes {
				return fail(errors.New("comparison pairs exceed output budget"))
			}
			output.Pairs = append(output.Pairs, pair)
		}
	}
	raw, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fail(err)
	}
	raw = append(raw, '\n')
	if int64(len(raw)) > input.MaxOutputBytes {
		return fail(errors.New("serialized comparison exceeds output budget"))
	}
	if err = pulse(); err != nil {
		return fail(err)
	}
	if _, err = os.Lstat(outputPath); err == nil {
		previous, _, err := toolkitReadBoundedJSON(outputPath, input.MaxOutputBytes, pulse)
		if err != nil {
			return fail(err)
		}
		if !bytes.Equal(previous, raw) {
			return fail(errors.New("comparison replay provenance or content mismatch"))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fail(err)
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
		a.Heartbeat(ctx, ToolkitPackageInventoryHeartbeat{input.SnapshotRef, input.OutputRef, "ledger_comparison_complete", len(input.LedgerIndices)})
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return ToolkitLedgerComparisonResult{input.OutputRef, digest, len(output.Candidates), len(output.Pairs)}, nil
}
