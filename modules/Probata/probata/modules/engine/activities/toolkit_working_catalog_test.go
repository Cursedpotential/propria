// Byline: Codex · GPT-6 · 2026-10-05. Synthetic bounded admission/Temporal tests; retained fixtures only.
package activities

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// workingTestBatch supplies 443 distinct occurrence/version identities with deliberately identical content hashes.
// Inputs: none. Outputs: synthetic admitted metadata. Effects: none; choose to prove hash equality cannot collapse source units.
func workingTestBatch() ToolkitWorkingCatalogBatch {
	b := ToolkitWorkingCatalogBatch{Schema: ToolkitWorkingCatalogSchema, Request: ToolkitWorkingCatalogInput{OperationID: "synthetic-working", ReceiptRef: "file:///synthetic/receipt.json", ReceiptSHA256: ToolkitWorkingCatalogReceiptSHA256, ManifestRef: "file:///synthetic/manifest.json", ManifestSHA256: ToolkitWorkingCatalogManifestSHA256, ReferenceMapRef: "file:///synthetic/map.json", ReferenceMapSHA256: ToolkitWorkingCatalogReferenceMapSHA256, ExpectedObjects: 443}}
	for i := 0; i < 443; i++ {
		key := fmt.Sprintf("%sreference-data/synthetic/member-%03d.md", toolkitLegalPrefix, i)
		ref := url.URL{Scheme: "b2", Host: "salem-data", Path: "/" + key, RawQuery: "versionId=synthetic-v1"}
		b.Objects = append(b.Objects, ToolkitWorkingCatalogObject{Placement: ToolkitContentPlacementObject{UnitID: "synthetic-unit", SourceRef: "file:///synthetic/unit.zip", Path: fmt.Sprintf("member-%03d.md", i), ObjectKey: key, ObjectRef: proffer.Ref(ref.String()), VersionID: "synthetic-v1", LatestVersionID: "synthetic-v1", SHA256: strings.Repeat("a", 64), Bytes: 10, BeforeVersions: []string{}, AfterVersions: []string{"synthetic-v1"}, Status: "verified"}, ArchiveSHA256: strings.Repeat("b", 64), ArchiveBytes: 4430, LogicalSourcePath: fmt.Sprintf("member-%03d.md", i), LogicalCategory: "reference-data", ResourceRole: "synthetic-reference", LinkPolicy: "Resolve through pinned private map", ReferenceRecords: []ToolkitWorkingCatalogLink{}, SourceRecords: []ToolkitWorkingCatalogLink{}})
	}
	return b
}

// TestToolkitWorkingCanonical checks complete identity retention, order independence, and fail-closed admission.
// Inputs: synthetic batches with targeted defects. Outputs: assertions. Effects: none; choose before SQL admission proof.
func TestToolkitWorkingCanonical(t *testing.T) {
	b := workingTestBatch()
	_, sha, e := ToolkitWorkingCatalogCanonical(b)
	if e != nil {
		t.Fatal(e)
	}
	b.Objects[0], b.Objects[442] = b.Objects[442], b.Objects[0]
	_, reordered, e := ToolkitWorkingCatalogCanonical(b)
	if e != nil || reordered != sha {
		t.Fatal("row order changed metadata identity")
	}
	cases := map[string]func(*ToolkitWorkingCatalogBatch){
		"wrong receipt":            func(b *ToolkitWorkingCatalogBatch) { b.Request.ReceiptSHA256 = strings.Repeat("c", 64) },
		"wrong map":                func(b *ToolkitWorkingCatalogBatch) { b.Request.ReferenceMapSHA256 = strings.Repeat("c", 64) },
		"partial":                  func(b *ToolkitWorkingCatalogBatch) { b.Objects = b.Objects[:442] },
		"missing version":          func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.VersionID = "" },
		"wrong hash":               func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.SHA256 = "sha1-is-not-sha256" },
		"missing version readback": func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.AfterVersions = nil },
		"conflicting latest":       func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.LatestVersionID = "other-v" },
		"duplicate occurrence":     func(b *ToolkitWorkingCatalogBatch) { b.Objects[1] = b.Objects[0] },
		"recovery home": func(b *ToolkitWorkingCatalogBatch) {
			b.Objects[0].Placement.ObjectKey = "consignatio/casevault/recovery/source.md"
		},
		"unverified":  func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.Status = "pending" },
		"path escape": func(b *ToolkitWorkingCatalogBatch) { b.Objects[0].Placement.Path = "../member.md" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b := workingTestBatch()
			mutate(&b)
			if _, _, e := ToolkitWorkingCatalogCanonical(b); e == nil {
				t.Fatal("invalid admission accepted")
			}
		})
	}
	for _, category := range []string{"benchbooks", "case-law"} {
		b := workingTestBatch()
		b.Objects[0].LogicalCategory = category
		if _, _, e := ToolkitWorkingCatalogCanonical(b); e != nil {
			t.Fatal("logical category changed physical placement", e)
		}
	}
	r, e := workingSummary(workingTestBatch(), "readback_verified")
	if e != nil || r.Objects != 443 || r.Bytes != 4430 || r.WholeBucketFreshness != "unchanged" || r.ProjectionFreshness != "not_refreshed" {
		t.Fatal("scoped summary/freshness invalid")
	}
}

// TestToolkitWorkingWrongReceiptHash rejects changed mounted bytes before any repository call and retains the fixture.
// Inputs: approved pins with a synthetic mismatching file. Outputs: rejection assertion. Effects: fixture under to_be_deleted, no deletion.
// Choose to exercise byte authentication rather than only shape validation.
func TestToolkitWorkingWrongReceiptHash(t *testing.T) {
	root, e := filepath.Abs("../../../../../../to_be_deleted/toolkit-working-catalog-tests")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	dir, e := os.MkdirTemp(root, "wrong-hash-")
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "receipt.json")
	if e = os.WriteFile(path, []byte(`{"complete":true}`), 0600); e != nil {
		t.Fatal(e)
	}
	in := workingTestBatch().Request
	in.ReceiptRef = proffer.Ref((&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String())
	// Windows file refs use file:///E:/... rather than file://E:/...
	if filepath.VolumeName(path) != "" {
		in.ReceiptRef = proffer.Ref("file:///" + filepath.ToSlash(path))
	}
	a := ToolkitWorkingCatalogActivities{AllowedRoot: dir}
	if _, e = a.RegisterToolkitWorkingCatalog(context.Background(), in); e == nil || !strings.Contains(e.Error(), "SHA-256 mismatch") {
		t.Fatalf("receipt authentication failed: %v", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = a.ReadbackToolkitWorkingCatalog(ctx, in); e == nil {
		t.Fatal("cancellation ignored")
	}
}

// TestToolkitWorkingSourceUnitMapping verifies manifest/map/receipt agreement and rejects altered provenance without I/O.
// Inputs: synthetic complete source unit and targeted mismatches. Outputs: admission assertions. Effects: memory only.
// Choose to cover provenance joins that metadata digest tests alone cannot exercise.
func TestToolkitWorkingSourceUnitMapping(t *testing.T) {
	fixture := workingTestBatch()
	unit := ToolkitContentPlacementUnit{ID: "synthetic-unit", SourceRef: "file:///synthetic/unit.zip", ArchiveSHA256: strings.Repeat("b", 64), ArchiveBytes: 4430, Destination: "reference-data/synthetic"}
	var entries []toolkitWorkingMapEntry
	var placements []ToolkitContentPlacementObject
	for _, row := range fixture.Objects {
		p := row.Placement
		placements = append(placements, p)
		unit.Files = append(unit.Files, ToolkitContentPlacementFile{Path: p.Path, SHA256: p.SHA256, Bytes: p.Bytes})
		entries = append(entries, toolkitWorkingMapEntry{ToolkitWorkingCatalogObject: row, UnitID: p.UnitID, SourceRef: p.SourceRef, ExactZIPMember: p.Path, SHA256: p.SHA256, Bytes: p.Bytes, Bucket: "salem-data", ObjectKey: p.ObjectKey})
	}
	manifest := ToolkitContentPlacementManifest{Units: []ToolkitContentPlacementUnit{unit}}
	b, e := assembleToolkitWorkingMetadata(fixture.Request, placements, manifest, entries)
	if e != nil || len(b.Objects) != 443 {
		t.Fatal("valid mapping", e)
	}
	for _, name := range []string{"archive-hash", "source-unit", "member-hash", "record-link"} {
		t.Run(name, func(t *testing.T) {
			changed := append([]toolkitWorkingMapEntry(nil), entries...)
			switch name {
			case "archive-hash":
				changed[0].ArchiveSHA256 = strings.Repeat("c", 64)
			case "source-unit":
				changed[0].UnitID = "unrelated-unit"
			case "member-hash":
				changed[0].SHA256 = strings.Repeat("c", 64)
			case "record-link":
				changed[0].SourceRecords = []ToolkitWorkingCatalogLink{{ID: ""}}
			}
			if _, e := assembleToolkitWorkingMetadata(fixture.Request, placements, manifest, changed); e == nil {
				t.Fatal("provenance mismatch accepted")
			}
		})
	}
	if _, e := assembleToolkitWorkingMetadata(fixture.Request, placements, manifest, entries[:442]); e == nil {
		t.Fatal("partial map accepted")
	}
}

// TestToolkitWorkingWorkflowTracksTwoActivities proves registration and readback are separate tracked calls with bounded results.
// Inputs: approved coordinates and mocked Activity outcomes. Outputs: verified summary. Effects: Temporal test environment only.
// Choose instead of claiming worker registration or a live queue execution.
func TestToolkitWorkingWorkflowTracksTwoActivities(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	a := ToolkitWorkingCatalogActivities{}
	env.RegisterActivityWithOptions(a.RegisterToolkitWorkingCatalog, activity.RegisterOptions{Name: ToolkitWorkingCatalogRegisterActivityName})
	env.RegisterActivityWithOptions(a.ReadbackToolkitWorkingCatalog, activity.RegisterOptions{Name: ToolkitWorkingCatalogReadbackActivityName})
	in := workingTestBatch().Request
	registered, _ := workingSummary(workingTestBatch(), "registered_pending_readback")
	verified, _ := workingSummary(workingTestBatch(), "readback_verified")
	env.OnActivity(ToolkitWorkingCatalogRegisterActivityName, mock.Anything, in).Return(registered, nil).Once()
	env.OnActivity(ToolkitWorkingCatalogReadbackActivityName, mock.Anything, in).Return(verified, nil).Once()
	env.ExecuteWorkflow(ToolkitWorkingCatalogWorkflow, in)
	if e := env.GetWorkflowError(); e != nil {
		t.Fatal(e)
	}
	var got ToolkitWorkingCatalogResult
	if e := env.GetWorkflowResult(&got); e != nil || got != verified {
		t.Fatal("wrong workflow result", e)
	}
	env.AssertExpectations(t)
}
