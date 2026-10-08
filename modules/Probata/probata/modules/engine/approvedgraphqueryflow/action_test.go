// Byline: Codex · GPT-6 · 2026-10-08.
package approvedgraphqueryflow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

type queryReaderFixture struct {
	result surrealsink.ApprovedQueryResult
}

func (f queryReaderFixture) QueryApprovedClaims(context.Context, surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error) {
	return f.result, nil
}

type versionStoreFixture struct {
	body            []byte
	corruptReadback bool
}

func (f *versionStoreFixture) PutRecoveredVersion(_ context.Context, _, _ string, source io.ReadSeeker, _ int64, _, _ string, _ func(int64)) (string, error) {
	var err error
	f.body, err = io.ReadAll(source)
	return "fixture-version", err
}

func (f *versionStoreFixture) OpenVersion(context.Context, string, string, string) (io.ReadCloser, error) {
	if f.corruptReadback {
		return io.NopCloser(strings.NewReader("different bytes")), nil
	}
	return io.NopCloser(bytes.NewReader(f.body)), nil
}

func queryFixture() (Request, approvedgraph.Claim) {
	approved := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	scope := surrealsink.ApprovedQueryScope{MatterID: caseidentity.AuthoritativeMatterID,
		CourtCaseID: caseidentity.AuthoritativeCourtCaseID, AccessPolicyID: "policy", ApprovedRevisionID: "revision",
		ApprovalDigest: strings.Repeat("a", 64), ProjectionGenerationID: "generation", ProjectionHash: strings.Repeat("b", 64),
		Perspective: "hindsight", Limit: 1}
	claim := approvedgraph.Claim{ID: "claim", Kind: "event_account", Text: "Synthetic cited account",
		MatterID: scope.MatterID, CourtCaseID: scope.CourtCaseID, ReceiptID: scope.ApprovedRevisionID,
		ApprovalDigest: scope.ApprovalDigest, ControlGenerationID: scope.ProjectionGenerationID,
		CandidateID: "candidate", CandidateSHA256: strings.Repeat("c", 64),
		SourceID: "source", SourceVersionID: "version", SourceObjectID: "object", SourceObjectURI: "b2://synthetic/version",
		SourceSHA256: strings.Repeat("d", 64), RecordID: "record", RecordSHA256: strings.Repeat("e", 64),
		ApprovedAt: approved, ApprovedBy: "owner", SourceAvailableFrom: nil}
	return Request{OperatingMode: string(caseidentity.ModeLive), Actor: entities.Actor{SubjectUID: "actor"},
		RequestID: "query-1", Scope: scope}, claim
}

func TestActivityStoresExactHindsightPageWithoutInventingAvailability(t *testing.T) {
	in, claim := queryFixture()
	store := &versionStoreFixture{}
	activity := Activities{Reader: queryReaderFixture{surrealsink.ApprovedQueryResult{
		Perspective: "hindsight", ApprovedRevisionID: "revision", Claims: []approvedgraph.Claim{claim}}},
		Artifacts: libraryvalidation.B2Artifacts{Store: store, Bucket: ApprovedQueryBucket, Prefix: ApprovedQueryPrefixRoot + "approved_context"}}
	got, err := activity.Run(context.Background(), in)
	if err != nil || got.ClaimCount != 1 || got.Artifact.VersionID != "fixture-version" || got.Artifact.SHA256 != libraryvalidation.Hash(store.body) ||
		got.AccessPolicyID != in.Scope.AccessPolicyID || got.MatterID != in.Scope.MatterID ||
		!strings.Contains(string(store.body), `"source_available_from":null`) ||
		!strings.Contains(string(store.body), `"actor_subject_uid":"actor"`) ||
		!strings.Contains(got.Artifact.URI, "versionId=fixture-version") {
		t.Fatalf("exact cited result was not retained: result=%+v err=%v", got, err)
	}
	activity.Artifacts.Prefix = "DerivedKnowledge/analysis/other"
	if _, err := activity.Run(context.Background(), in); err == nil {
		t.Fatal("query result escaped the dedicated CaseVault prefix")
	}
	activity.Artifacts.Prefix = ApprovedQueryPrefixRoot + "approved_context"
	activity.Artifacts.Bucket = "other-bucket"
	if _, err := activity.Run(context.Background(), in); err == nil {
		t.Fatal("query result escaped the CaseVault bucket")
	}
}

func TestActivityRejectsFailedExactVersionReadback(t *testing.T) {
	in, claim := queryFixture()
	store := &versionStoreFixture{corruptReadback: true}
	activity := Activities{Reader: queryReaderFixture{surrealsink.ApprovedQueryResult{
		Perspective: "hindsight", ApprovedRevisionID: "revision", Claims: []approvedgraph.Claim{claim}}},
		Artifacts: libraryvalidation.B2Artifacts{Store: store, Bucket: ApprovedQueryBucket, Prefix: ApprovedQueryPrefixRoot + "approved_context"}}
	if _, err := activity.Run(context.Background(), in); err == nil || len(store.body) == 0 {
		t.Fatal("unverified exact-version result was returned")
	}
}

func TestActivityRejectsOversizePageBeforeVersionedWrite(t *testing.T) {
	in, claim := queryFixture()
	claim.Text = strings.Repeat("x", MaxResultBytes)
	store := &versionStoreFixture{}
	activity := Activities{Reader: queryReaderFixture{surrealsink.ApprovedQueryResult{
		Perspective: "hindsight", ApprovedRevisionID: "revision", Claims: []approvedgraph.Claim{claim}}},
		Artifacts: libraryvalidation.B2Artifacts{Store: store, Bucket: ApprovedQueryBucket, Prefix: ApprovedQueryPrefixRoot + "approved_context"}}
	if _, err := activity.Run(context.Background(), in); err == nil || len(store.body) != 0 {
		t.Fatal("oversize page was written or silently clipped")
	}
}
