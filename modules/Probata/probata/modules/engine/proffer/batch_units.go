package proffer

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"go.temporal.io/sdk/workflow"
)

const FreezeBatchUnitActivityName = "freeze_batch_unit_activity"
const BatchUnitContract = "proffer-batch-unit-v1"

// BatchUnitRequest identifies one explicit package inventory within its batch and case scope.
// Inputs: authorized root and small scope identifiers. Outputs: frozen inventory via the Activity.
// Effects: none here; choose this contract instead of recursively importing attachment objects.
type BatchUnitRequest struct {
	OperatingMode  string `json:"operating_mode"`
	MatterID       string `json:"matter_id"`
	CourtCaseID    string `json:"court_case_id"`
	BatchID        string `json:"batch_id"`
	UnitRoot       string `json:"unit_root"`
	DeclaredFormat string `json:"declared_format"`
}

// BatchUnitMember preserves one original object's exact path and provider identity outside workflow history.
type BatchUnitMember struct {
	RelativePath      string `json:"relative_path"`
	SourceRef         string `json:"source_ref"`
	ProviderVersionID string `json:"provider_version_id"`
	ETag              string `json:"etag"`
	ByteLength        int64  `json:"byte_length"`
}

// BatchUnitManifest freezes original membership and the byte-verified primary entrypoint externally.
type BatchUnitManifest struct {
	Contract       string            `json:"contract"`
	Request        BatchUnitRequest  `json:"request"`
	PrimaryRef     Ref               `json:"primary_ref"`
	DetectedFormat string            `json:"detected_format"`
	SelectedFormat string            `json:"selected_format"`
	Signature      string            `json:"signature"`
	Members        []BatchUnitMember `json:"members"`
}

// BatchUnitResult carries the compact manifest identity and selected primary into workflow history.
type BatchUnitResult struct {
	UnitRoot        string `json:"unit_root"`
	UnitManifestRef Ref    `json:"unit_manifest_ref"`
	ManifestSHA256  string `json:"manifest_sha256"`
	MemberCount     int    `json:"member_count"`
	PrimaryRef      Ref    `json:"primary_ref"`
	DetectedFormat  string `json:"detected_format"`
	SelectedFormat  string `json:"selected_format"`
	Signature       string `json:"signature"`
}

// CanonicalBatchUnitRoots validates and sorts exact nonoverlapping roots under the already-authorized folder.
// Inputs: folder coordinates and at most 32 roots. Outputs: canonical encoded URIs or a refusal.
// Effects: none; callers must separately apply configured-root authority before dispatch.
func CanonicalBatchUnitRoots(scheme, bucket, prefix string, roots []string) ([]string, error) {
	if len(roots) > 32 {
		return nil, errors.New("unit_roots may contain at most 32 package roots")
	}
	out := make([]string, 0, len(roots))
	keys := make([]string, 0, len(roots))
	for _, value := range roots {
		u, err := url.Parse(value)
		if err != nil || value != strings.TrimSpace(value) || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Scheme != scheme || u.Host != bucket {
			return nil, errors.New("unit root must be an exact folder URI in the batch source store")
		}
		key := strings.TrimPrefix(u.Path, "/")
		if !strings.HasSuffix(key, "/") || !strings.HasPrefix(key, prefix) || strings.ContainsAny(key, "\\\x00") {
			return nil, errors.New("unit root must lie under the batch folder and end in /")
		}
		for _, part := range strings.Split(strings.TrimSuffix(key, "/"), "/") {
			if part == "" || part == "." || part == ".." {
				return nil, errors.New("unit root contains an unsafe path segment")
			}
		}
		for _, prior := range keys {
			if strings.HasPrefix(key, prior) || strings.HasPrefix(prior, key) {
				return nil, errors.New("unit roots must be distinct and nonoverlapping")
			}
		}
		keys = append(keys, key)
		out = append(out, u.String())
	}
	sort.Strings(out)
	return out, nil
}

// runBatchUnits freezes each selected package and starts exactly one primary child per successful unit.
// Inputs: validated roots and existing batch state. Outputs: complete per-unit status and counts.
// Effects: invokes atomic inventory and ordinary child imports; attachments never become child jobs.
func runBatchUnits(ctx workflow.Context, in BatchInput, state *batchState, maxInFlight int) (BatchStatus, error) {
	for i, root := range in.UnitRoots {
		state.byIndex[i] = &BatchItem{Key: root, UnitRoot: root, RequestID: fmt.Sprintf("%s-%05d", in.BatchID, i), Status: BatchItemQueued}
		state.order = append(state.order, i)
	}
	freezeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Minute, RetryPolicy: retryPolicy(2*time.Second, 3)})
	for _, i := range state.order {
		if err := workflow.Await(ctx, func() bool { return state.running < maxInFlight }); err != nil {
			return state.snapshot(), err
		}
		item := state.byIndex[i]
		state.running++
		workflow.Go(ctx, func(itemCtx workflow.Context) {
			defer func() { state.running-- }()
			var result BatchUnitResult
			err := workflow.ExecuteActivity(freezeCtx, FreezeBatchUnitActivityName, BatchUnitRequest{OperatingMode: in.OperatingMode, MatterID: in.MatterID, CourtCaseID: in.CourtCaseID, BatchID: in.BatchID, UnitRoot: item.UnitRoot, DeclaredFormat: in.DeclaredFormat}).Get(itemCtx, &result)
			if err != nil {
				item.Status, item.Reason = BatchItemFailed, "package inventory: "+err.Error()
				return
			}
			if result.UnitRoot != item.UnitRoot || result.PrimaryRef == "" || result.UnitManifestRef == "" || result.MemberCount < 1 || result.ManifestSHA256 == "" || result.SelectedFormat == "" {
				item.Status, item.Reason = BatchItemFailed, "package inventory returned incomplete or mismatched identity"
				return
			}
			item.SourceRef = result.PrimaryRef
			item.UnitManifestRef = result.UnitManifestRef
			item.UnitManifestSHA256 = result.ManifestSHA256
			item.MemberCount = &result.MemberCount
			item.SelectedFormat = result.SelectedFormat
			runBatchItem(itemCtx, in, item)
		})
	}
	if err := workflow.Await(ctx, func() bool { return state.running == 0 }); err != nil {
		return state.snapshot(), err
	}
	state.status.Terminal = true
	return state.snapshot(), nil
}
