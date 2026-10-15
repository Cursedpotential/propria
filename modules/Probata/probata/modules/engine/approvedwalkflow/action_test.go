package approvedwalkflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/approvedgraph"
	"github.com/Cursedpotential/probata/engine/approvedgraphqueryflow"
	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"github.com/Cursedpotential/probata/engine/surrealsink"
)

type memoryVersions struct {
	bodies map[string][]byte
	next   int
}

func (s *memoryVersions) PutRecoveredVersion(_ context.Context, bucket, key string, reader io.ReadSeeker, _ int64, _, _ string, _ func(int64)) (string, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	s.next++
	version := fmt.Sprint(s.next)
	s.bodies[bucket+"/"+key+"/"+version] = raw
	return version, nil
}
func (s *memoryVersions) OpenVersion(_ context.Context, bucket, key, version string) (io.ReadCloser, error) {
	raw, ok := s.bodies[bucket+"/"+key+"/"+version]
	if !ok {
		return nil, fmt.Errorf("unknown exact version")
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

type horizonReader struct {
	claims  []approvedgraph.Claim
	calls   []surrealsink.ApprovedQueryScope
	hasMore bool
}

func (r *horizonReader) QueryApprovedClaims(_ context.Context, q surrealsink.ApprovedQueryScope) (surrealsink.ApprovedQueryResult, error) {
	r.calls = append(r.calls, q)
	var claims []approvedgraph.Claim
	for _, c := range r.claims {
		if q.Perspective == "hindsight" || (c.SourceAvailableFrom != nil && !c.SourceAvailableFrom.After(q.Horizon)) {
			claims = append(claims, c)
		}
	}
	cursor := ""
	if r.hasMore {
		cursor = "bounded-next"
	}
	return surrealsink.ApprovedQueryResult{Claims: claims, Perspective: q.Perspective, ApprovedRevisionID: q.ApprovedRevisionID, HasMore: r.hasMore, NextCursor: cursor}, nil
}
func fixture() (Request, Activities, *horizonReader) {
	first := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	last := first.Add(24 * time.Hour)
	q := approvedgraphqueryflow.Request{OperatingMode: string(caseidentity.ModeLive), Actor: entities.Actor{SubjectUID: "test-actor"}, RequestID: "walk-test", Scope: surrealsink.ApprovedQueryScope{
		MatterID: caseidentity.AuthoritativeMatterID, CourtCaseID: caseidentity.AuthoritativeCourtCaseID, AccessPolicyID: "policy", ApprovedRevisionID: "revision", ApprovalDigest: strings.Repeat("a", 64), ProjectionGenerationID: "generation", ProjectionHash: strings.Repeat("b", 64), Perspective: "as_lived", Horizon: last, Limit: 100}}
	base := approvedgraph.Claim{ID: "early", Text: "An attributed test account", Kind: "event_account", MatterID: q.Scope.MatterID, CourtCaseID: q.Scope.CourtCaseID, ReceiptID: "revision", ApprovalDigest: q.Scope.ApprovalDigest, ControlGenerationID: "generation", CandidateID: "candidate", CandidateSHA256: strings.Repeat("c", 64), SourceID: "11111111-1111-4111-8111-111111111111", SourceVersionID: "22222222-2222-4222-8222-222222222222", SourceObjectID: "33333333-3333-4333-8333-333333333333", SourceObjectURI: "b2://fixture/source", SourceSHA256: strings.Repeat("d", 64), RecordID: "record", RecordSHA256: strings.Repeat("e", 64), SourceAvailableFrom: &first, ApprovedAt: last, ApprovedBy: "test-actor"}
	later := base
	later.ID = "later"
	later.RecordID = "record-later"
	later.SourceAvailableFrom = &last
	unknown := base
	unknown.ID = "unknown"
	unknown.RecordID = "record-unknown"
	unknown.SourceAvailableFrom = nil
	reader := &horizonReader{claims: []approvedgraph.Claim{later, unknown, base}}
	store := &memoryVersions{bodies: map[string][]byte{}}
	a := Activities{Query: approvedgraphqueryflow.Activities{Reader: reader, Artifacts: libraryvalidation.B2Artifacts{Store: store, Bucket: "salem-data", Prefix: approvedgraphqueryflow.ApprovedQueryPrefixRoot + "test"}}, Artifacts: libraryvalidation.B2Artifacts{Store: store, Bucket: "salem-data", Prefix: PrefixRoot + "test"}}
	return Request{Query: q, Cutoffs: []time.Time{first, last}}, a, reader
}
func checkpoint(t *testing.T, a Activities, in Request, prepared Prepared, index int) Reference {
	t.Helper()
	var input stepBody
	if err := a.read(context.Background(), prepared.Artifact, &input); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, c := range input.Claims {
		ids = append(ids, c.ID)
	}
	found := []Finding{}
	if len(input.Claims) > 0 {
		c := input.Claims[0]
		found = append(found, Finding{Kind: "uncertainty", Text: "Account truth remains unresolved", Citations: []Citation{{ClaimID: c.ID, Quote: "attributed", SourcePin: c}}})
	}
	check := Checkpoint{InputArtifact: prepared.Artifact, Schema: "approved-walk-checkpoint-v1", Perspective: input.Perspective, Horizon: input.Horizon, ScopeDigest: scopeDigest(in), StepIndex: index, ClaimIDs: ids, Findings: found, Model: json.RawMessage(`{"provider":"fixture","model_id":"inert"}`), PromptVersion: "approved-walk-balanced-v1", PromptSHA256: strings.Repeat("f", 64), ReviewStatus: "pending"}
	raw, _ := json.Marshal(check)
	ref, err := a.Artifacts.Put(context.Background(), fmt.Sprintf("test/check-%d.json", index), raw, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	return Reference{Artifact: ref}
}

func TestPreparePrefiltersEveryHorizonAndKeepsHindsightIndependent(t *testing.T) {
	in, a, reader := fixture()
	ctx := context.Background()
	first, err := a.Prepare(ctx, StepRequest{Request: in, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	if reader.calls[0].Perspective != "as_lived" || !reader.calls[0].Horizon.Equal(in.Cutoffs[0]) || first.ClaimCount != 1 {
		t.Fatal("as-lived retrieval was not prefiltered")
	}
	prior := checkpoint(t, a, in, first, 0)
	second, err := a.Prepare(ctx, StepRequest{Request: in, Index: 1, Previous: &prior.Artifact})
	if err != nil {
		t.Fatal(err)
	}
	var body stepBody
	if err := a.read(ctx, second.Artifact, &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Claims) != 2 || body.Claims[0].ID != "early" || body.Claims[1].ID != "later" || body.Previous == nil {
		t.Fatal("eligible material was not chronological with prior state")
	}
	hindsight, err := a.Prepare(ctx, StepRequest{Request: in, Index: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.read(ctx, hindsight.Artifact, &body); err != nil {
		t.Fatal(err)
	}
	// Decode into a fresh value; omitted previous must not inherit a reused decoder target.
	var separate stepBody
	if err := a.read(ctx, hindsight.Artifact, &separate); err != nil {
		t.Fatal(err)
	}
	if separate.Perspective != "hindsight" || separate.Previous != nil || hindsight.ClaimCount != 3 || separate.Claims[2].ID != "unknown" {
		t.Fatal("hindsight leaked state or unknown time was invented")
	}
	if _, err := a.Prepare(ctx, StepRequest{Request: in, Index: 2, Previous: &prior.Artifact}); err == nil {
		t.Fatal("as-lived state was passed into hindsight")
	}
}

func TestFinalCitationRevalidationAndRetryUseExactVersions(t *testing.T) {
	in, a, reader := fixture()
	ctx := context.Background()
	final := FinalizeRequest{Request: in}
	var previous *libraryvalidation.ArtifactRef
	for i := 0; i <= len(in.Cutoffs); i++ {
		if i == len(in.Cutoffs) {
			previous = nil
		}
		input, err := a.Prepare(ctx, StepRequest{Request: in, Index: i, Previous: previous})
		if err != nil {
			t.Fatal(err)
		}
		ref := checkpoint(t, a, in, input, i)
		final.Inputs = append(final.Inputs, input)
		final.Steps = append(final.Steps, ref)
		copy := ref.Artifact
		previous = &copy
	}
	got, err := a.Finalize(ctx, final)
	if err != nil {
		t.Fatal(err)
	}
	again, err := a.Finalize(ctx, final)
	if err != nil || got.Artifact.SHA256 != again.Artifact.SHA256 {
		t.Fatal("identical retry changed derived content", err)
	}
	if got.ReviewStatus != "pending" || got.Partial {
		t.Fatal("unexpected review/coverage status")
	}
	reader.claims[2].SourceSHA256 = strings.Repeat("9", 64)
	if _, err := a.Finalize(ctx, final); err == nil {
		t.Fatal("stale source pin survived final re-resolution")
	}
}

func TestBoundsPartialCoverageAndUnresolvablePriorState(t *testing.T) {
	in, a, reader := fixture()
	in.Cutoffs = []time.Time{in.Query.Scope.Horizon, in.Query.Scope.Horizon}
	if err := in.Validate(); err == nil {
		t.Fatal("duplicate cutoffs accepted")
	}
	in, a, reader = fixture()
	reader.hasMore = true
	first, err := a.Prepare(context.Background(), StepRequest{Request: in, Index: 0})
	if err != nil || !first.HasMore {
		t.Fatal("partial coverage hidden", err)
	}
	ref := checkpoint(t, a, in, first, 0)
	reader.claims = reader.claims[:1]
	if _, err := a.Prepare(context.Background(), StepRequest{Request: in, Index: 1, Previous: &ref.Artifact}); err == nil {
		t.Fatal("prior findings used without eligible cited claim")
	}
	in, a, reader = fixture()
	reader.claims[2].Text = strings.Repeat("x", MaxPromptBytes)
	if _, err := a.Prepare(context.Background(), StepRequest{Request: in, Index: 0}); err == nil {
		t.Fatal("unbounded prompt prepared")
	}
}

func TestEmptyHistoricalPageUsesArrayWithoutInventingAvailability(t *testing.T) {
	in, a, _ := fixture()
	in.Cutoffs[0] = in.Cutoffs[0].Add(-24 * time.Hour)
	input, err := a.Prepare(context.Background(), StepRequest{Request: in, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := a.Artifacts.Read(context.Background(), input.Artifact, approvedgraphqueryflow.MaxResultBytes)
	if err != nil {
		t.Fatal(err)
	}
	if input.ClaimCount != 0 || !bytes.Contains(raw, []byte(`"claims":[]`)) {
		t.Fatal("empty eligible page was not a valid bounded JSON array")
	}
}
