// Byline: Codex · GPT-6.1 · 2026-10-07.
package activities

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// sourceUnitFake models exact-version B2 metadata, retained copies and independent corruption.
// Inputs: synthetic bytes and failure switches
// Outputs: memory-only storage observations
// Effects: fixture maps/counters only
// Choose: for bounded failure tests without live B2 writes.
// Byline: Codex · GPT-6.1 · 2026-10-07.
type sourceUnitFake struct {
	smsthreads.ObjectStore
	toolkitPlacementStore
	bodies                                              map[string][]byte
	heads                                               map[string]AISourceUnitMetadata
	latest                                              map[string]string
	versions                                            map[string][]string
	copies, opens                                       int
	loseResponse, race, corruptDestination, shortSource bool
}

// HeadExact returns the synthetic exact-version metadata snapshot or explicit absence.
// Inputs: pinned coordinates
// Outputs: admitted metadata
// Effects: memory reads only
// Choose: to test changed-version rejection before writes.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) HeadExact(_ context.Context, _, key, version string) (AISourceUnitMetadata, error) {
	m, ok := s.heads[key+"@"+version]
	if !ok {
		return m, smsthreads.ErrRecoveredVersionNotFound
	}
	return m, nil
}

// HeadVersion returns the synthetic visible retained version.
// Inputs: key
// Outputs: version/size or absence
// Effects: memory reads only
// Choose: for conflict and lost-response tests.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) HeadVersion(_ context.Context, _, key string) (smsthreads.ObjectVersion, error) {
	v, ok := s.latest[key]
	if !ok {
		return smsthreads.ObjectVersion{}, smsthreads.ErrRecoveredVersionNotFound
	}
	return smsthreads.ObjectVersion{VersionID: v, Size: s.heads[key+"@"+v].Bytes}, nil
}

// PlacementVersions returns every retained synthetic version for one exact destination.
// Inputs: key
// Outputs: independent version slice
// Effects: memory reads only
// Choose: to test race observations.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) PlacementVersions(_ context.Context, _, key string) ([]string, error) {
	return append([]string{}, s.versions[key]...), nil
}

// UnitKeys returns complete current synthetic destination membership.
// Inputs: prefix
// Outputs: exact current keys
// Effects: memory reads only
// Choose: to test unreviewed extra-member rejection.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) UnitKeys(_ context.Context, _, prefix string) ([]string, error) {
	keys := []string{}
	for key := range s.latest {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// SourceKeys returns the synthetic current source-key membership without including copied destinations.
// Inputs: reviewed source prefix
// Outputs: unique source keys
// Effects: memory reads only
// Choose: to test added or missing source members before any copy.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) SourceKeys(_ context.Context, _, prefix string) ([]string, error) {
	seen := map[string]bool{}
	for coordinate := range s.heads {
		key, _, ok := strings.Cut(coordinate, "@")
		if ok && strings.HasPrefix(key, prefix) {
			seen[key] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	return keys, nil
}

// OpenVersion reads a synthetic exact body with optional source truncation or destination corruption.
// Inputs: pinned coordinates
// Outputs: stream or absence
// Effects: fixture read counter only
// Choose: to demonstrate separate copy and hash responsibilities.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) OpenVersion(_ context.Context, _, key, version string) (io.ReadCloser, error) {
	s.opens++
	b, ok := s.bodies[key+"@"+version]
	if !ok {
		return nil, smsthreads.ErrRecoveredVersionNotFound
	}
	b = append([]byte{}, b...)
	if s.shortSource && strings.HasPrefix(key, "consignatio/vault/") {
		b = b[:len(b)-1]
	}
	if s.corruptDestination && strings.HasPrefix(key, aiSourceUnitPrefix) {
		b[0] ^= 1
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// CopyExact creates a synthetic retained copy without invoking any payload-read or hash method.
// Inputs: pinned source/destination/ETag
// Outputs: returned version or uncertain response
// Effects: memory-only copy and optional competing version
// Choose: to test transport separately from verification.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func (s *sourceUnitFake) CopyExact(_ context.Context, _, source, version, dest, etag string) (string, error) {
	m, ok := s.heads[source+"@"+version]
	if !ok || m.ETag != etag {
		return "", errors.New("source version changed")
	}
	s.copies++
	id := fmt.Sprintf("copied-%d", s.copies)
	m.VersionID = id
	s.heads[dest+"@"+id] = m
	s.bodies[dest+"@"+id] = append([]byte{}, s.bodies[source+"@"+version]...)
	s.latest[dest] = id
	s.versions[dest] = append(s.versions[dest], id)
	if s.race {
		s.versions[dest] = append(s.versions[dest], "competing-version")
		s.latest[dest] = "competing-version"
	}
	if s.loseResponse {
		return "", errors.New("synthetic lost response after successful copy")
	}
	return id, nil
}

// sourceUnitFixture creates retained synthetic manifest/provenance files under the VPS owner-controlled quarantine.
// Inputs: test and member count
// Outputs: configured group, request and fake store
// Effects: exclusive fixture writes retained for owner cleanup
// Choose: over TempDir automatic deletion.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func sourceUnitFixture(t *testing.T, count int) (*AISourceUnitPlacementActivities, AISourceUnitPlacementInput, *sourceUnitFake, AISourceUnitManifest) {
	t.Helper()
	base := os.Getenv("AI_SOURCE_UNIT_TEST_ROOT")
	if filepath.Base(base) != "to_be_deleted" {
		t.Fatal("AI_SOURCE_UNIT_TEST_ROOT must be retained VPS to_be_deleted directory")
	}
	if e := os.MkdirAll(base, 0700); e != nil {
		t.Fatal(e)
	}
	root, e := os.MkdirTemp(base, "ai-source-unit-*")
	if e != nil {
		t.Fatal(e)
	}
	s := &sourceUnitFake{bodies: map[string][]byte{}, heads: map[string]AISourceUnitMetadata{}, latest: map[string]string{}, versions: map[string][]string{}}
	m := AISourceUnitManifest{Unit: "transcript/account-export", Bucket: "salem-data", SourcePrefix: "consignatio/vault/v1/transcript/account-export/"}
	objects := []map[string]any{}
	var total int64
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("member-%02d.md", i)
		b := []byte(fmt.Sprintf("# complete synthetic member %d\n", i))
		if i == 0 {
			name = "conversations.json"
			b = []byte(`{"conversations":[{"text":"complete fixture"}]}`)
		}
		if i == 1 {
			name = "conversations.zip"
			b = []byte{'P', 'K', 3, 4, 0, 255, 0, 1}
		}
		f := AISourceUnitFile{Key: m.SourcePrefix + name, VersionID: fmt.Sprintf("original-%d", i), Bytes: int64(len(b)), SHA1: fmt.Sprintf("%x", sha1.Sum(b))}
		if i == 0 {
			f.SHA256 = digestBytes(b)
		}
		m.Files = append(m.Files, f)
		total += f.Bytes
		s.bodies[f.Key+"@"+f.VersionID] = b
		s.heads[f.Key+"@"+f.VersionID] = AISourceUnitMetadata{VersionID: f.VersionID, Bytes: f.Bytes, ETag: "\"provider-etag\"", ContentType: "application/octet-stream", Metadata: map[string]string{"src_last_modified_millis": "1790000000000"}}
		objects = append(objects, map[string]any{"bucket": m.Bucket, "key": f.Key, "object_version_id": f.VersionID, "size": f.Bytes, "sha1": f.SHA1})
	}
	provenance := map[string]any{"bucket": m.Bucket, "provider": "b2", "source_prefix": m.SourcePrefix, "object_count": count, "object_bytes": total, "objects": objects, "package_completeness": "unresolved"}
	pin, e := aiWorkproductWriteExclusive(filepath.Join(root, "discovery.json"), provenance)
	if e != nil {
		t.Fatal(e)
	}
	m.ProvenanceRef = toolkitFileRef(filepath.Join(root, "discovery.json"))
	m.ProvenanceSHA256 = pin
	pin, e = aiWorkproductWriteExclusive(filepath.Join(root, "manifest.json"), m)
	if e != nil {
		t.Fatal(e)
	}
	in := AISourceUnitPlacementInput{ManifestRef: toolkitFileRef(filepath.Join(root, "manifest.json")), ManifestSHA256: pin, ReceiptRef: toolkitFileRef(filepath.Join(root, "receipt"))}
	a := NewAISourceUnitPlacementActivities(root, func(scheme string) (smsthreads.ObjectStore, error) {
		if scheme != "b2" {
			t.Fatal("wrong existing resolver scheme")
		}
		return s, nil
	})
	a.Heartbeat = nil
	return a, in, s, m
}

// sourceUnitRun executes a bounded synthetic prefix of the independently callable stages.
// Inputs: group, request and stage count
// Outputs: last summary and next pinned request
// Effects: retained fixture receipts and fake-store operations
// Choose: for precise boundary injection tests.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func sourceUnitRun(t *testing.T, a *AISourceUnitPlacementActivities, in AISourceUnitPlacementInput, count int) (AISourceUnitSummary, AISourceUnitStageInput) {
	t.Helper()
	r := AISourceUnitStageInput{Input: in}
	var out AISourceUnitSummary
	for _, stage := range []string{"inspect", "hash-source", "copy", "hash-destination", "readback"}[:count] {
		var e error
		out, e = a.sourceUnitStage(context.Background(), stage, r)
		if e != nil || !out.Complete {
			t.Fatalf("%s failed: %v %+v", stage, e, out)
		}
		r.PreviousRef, r.PreviousSHA256 = out.ReceiptRef, out.ReceiptSHA256
	}
	return out, r
}

// TestAISourceUnitComplete22AndPinnedReplay verifies all formats survive and completed replay performs no new copy.
// Inputs: 22 synthetic members
// Outputs: all-member physical receipt
// Effects: retained fixture only
// Choose: to prove unit completion and idempotence rather than exit-code success.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitComplete22AndPinnedReplay(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 22)
	out, _ := sourceUnitRun(t, a, in, 5)
	if out.Objects != 22 || !out.PhysicalPlacementVerified || s.copies != 22 {
		t.Fatalf("incomplete unit %+v copies=%d", out, s.copies)
	}
	out2, _ := sourceUnitRun(t, a, in, 5)
	if out2.ReceiptSHA256 != out.ReceiptSHA256 || s.copies != 22 {
		t.Fatal("completed pinned replay copied again or changed receipt")
	}
	for _, f := range m.Files {
		if !bytes.Equal(s.bodies[f.Key+"@"+f.VersionID], s.bodies[aiSourceDestinationPrefix(m)+strings.TrimPrefix(f.Key, m.SourcePrefix)+"@"+s.latest[aiSourceDestinationPrefix(m)+strings.TrimPrefix(f.Key, m.SourcePrefix)]]) {
			t.Fatal("member changed")
		}
	}
}

// TestAISourceUnitCopyHasNoPayloadReads proves transport and hashing execute in separate operations.
// Inputs: synthetic inspected/hashed unit
// Outputs: unchanged GET count during copy
// Effects: fixture only
// Choose: to catch hashing or worker uploads hidden inside server copy.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitCopyHasNoPayloadReads(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	before := s.opens
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e != nil || !out.Complete || s.opens != before || s.copies != 3 {
		t.Fatalf("copy conflated payload I/O: %+v %v opens=%d", out, e, s.opens-before)
	}
}

// TestAISourceUnitMissingVersionPreventsCopy verifies metadata admission fails before transport.
// Inputs: synthetic absent source version
// Outputs: retained incomplete receipt and zero copies
// Effects: fixture only
// Choose: for missing-member coverage proof.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitMissingVersionPreventsCopy(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	s.heads[m.Files[1].Key+"@"+m.Files[1].VersionID] = AISourceUnitMetadata{VersionID: "wrong-version"}
	out, e := a.InspectAISourceUnit(context.Background(), AISourceUnitStageInput{Input: in})
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("missing/version-changed source admitted")
	}
}

// TestAISourceUnitSourceHashConflictPreventsCopy rejects full-source bytes inconsistent with the frozen SHA1.
// Inputs: same-size corrupted source fixture
// Outputs: failed source hash and zero copies
// Effects: fixture only
// Choose: over provider checksum metadata as byte proof.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitSourceHashConflictPreventsCopy(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 1)
	b := s.bodies[m.Files[2].Key+"@"+m.Files[2].VersionID]
	b[0] ^= 1
	out, e := a.HashAISourceUnitSource(context.Background(), r)
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("source hash conflict admitted")
	}
}

// TestAISourceUnitTruncatedBodyPreventsCopy rejects EOF before the pinned source size.
// Inputs: truncated source stream
// Outputs: failed complete-byte check
// Effects: fixture only
// Choose: for partial-read coverage.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitTruncatedBodyPreventsCopy(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 1)
	s.shortSource = true
	out, e := a.HashAISourceUnitSource(context.Background(), r)
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("truncated body admitted")
	}
}

// TestAISourceUnitChangedSourceMetadataPreventsCopy rejects intervening exact-version metadata changes.
// Inputs: completed source-hash receipt and changed source HEAD
// Outputs: failure before copying
// Effects: fixture only
// Choose: for version/metadata stability guards.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitChangedSourceMetadataPreventsCopy(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	key := m.Files[2].Key + "@" + m.Files[2].VersionID
	head := s.heads[key]
	head.ETag = "changed"
	s.heads[key] = head
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("changed source copied")
	}
}

// TestAISourceUnitOccupiedLaterMemberPreventsAllCopies rejects a known conflict before the first member write.
// Inputs: completed source proof and occupied final destination
// Outputs: failure and zero copies
// Effects: fixture only
// Choose: for all-member preflight.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitOccupiedLaterMemberPreventsAllCopies(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	key := aiSourceDestinationPrefix(m) + strings.TrimPrefix(m.Files[2].Key, m.SourcePrefix)
	s.latest[key] = "foreign"
	s.versions[key] = []string{"foreign"}
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("occupied unit partially copied")
	}
}

// TestAISourceUnitLostResponseIsHeldWithoutDuplicateCopy rejects an uncertain retained copy attempt on retry.
// Inputs: lost server response after one copy
// Outputs: incomplete retained receipt and no second copy
// Effects: fixture only
// Choose: to prevent matching-latest guesses after transport uncertainty.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitLostResponseIsHeldWithoutDuplicateCopy(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	s.loseResponse = true
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e == nil || out.Complete || s.copies != 1 {
		t.Fatal("lost response accepted")
	}
	_, e = a.CopyAISourceUnit(context.Background(), r)
	if e == nil || s.copies != 1 {
		t.Fatal("uncertain copy repeated")
	}
}

// TestAISourceUnitConcurrentVersionFailsClosed rejects competing destination appearances after CopyObject.
// Inputs: synthetic race after copy
// Outputs: retained incomplete result
// Effects: fixture only
// Choose: to avoid unsupported atomic-create claims.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitConcurrentVersionFailsClosed(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	s.race = true
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e == nil || out.Complete {
		t.Fatal("concurrent version accepted")
	}
}

// TestAISourceUnitDestinationHashConflictFailsPhysicalProof rejects corrupt independent destination reads.
// Inputs: completed copy receipt and corrupted destination stream
// Outputs: failed hash receipt, no physical completion
// Effects: fixture only
// Choose: to prove copy responses are insufficient.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitDestinationHashConflictFailsPhysicalProof(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 3)
	s.corruptDestination = true
	out, e := a.HashAISourceUnitDestination(context.Background(), r)
	if e == nil || out.Complete || out.PhysicalPlacementVerified {
		t.Fatal("destination corruption accepted")
	}
}

// TestAISourceUnitExtraMemberFailsFinalReadback rejects merging an unreviewed object into a completed unit.
// Inputs: independent hash proof and extra visible destination member
// Outputs: failed final membership readback
// Effects: fixture only
// Choose: for account/export boundary preservation.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitExtraMemberFailsFinalReadback(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 4)
	s.latest[aiSourceDestinationPrefix(m)+"extra.json"] = "foreign"
	out, e := a.ReadbackAISourceUnit(context.Background(), r)
	if e == nil || out.Complete || out.PhysicalPlacementVerified {
		t.Fatal("extra member merged")
	}
}

// TestAISourceUnitManifestMembershipAndPathsRejectPartialScope checks omitted, duplicate and unsafe members.
// Inputs: modified synthetic mappings against their original discovery pin
// Outputs: validation failures
// Effects: retained metadata fixtures only
// Choose: to prevent silently dropping ZIPs, JSON or sidecars.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitManifestMembershipAndPathsRejectPartialScope(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	m.Files = m.Files[:2]
	p := filepath.Join(a.AllowedRoot, "partial-manifest.json")
	pin, e := aiWorkproductWriteExclusive(p, m)
	if e != nil {
		t.Fatal(e)
	}
	in.ManifestRef = toolkitFileRef(p)
	in.ManifestSHA256 = pin
	out, e := a.InspectAISourceUnit(context.Background(), AISourceUnitStageInput{Input: in})
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("partial discovery membership admitted")
	}
	_, _, _, m = sourceUnitFixture(t, 3)
	m.Files[1] = m.Files[0]
	if aiSourceUnitManifestValid(m) == nil {
		t.Fatal("duplicate admitted")
	}
	m.Files[0].Key = m.SourcePrefix + "../escape"
	if aiSourceUnitManifestValid(m) == nil {
		t.Fatal("path escape admitted")
	}
}

// TestAISourceUnitNeutralRouteRejectsProviderLabel holds a guessed provider outside the reviewed intake unit.
// Inputs: bounded synthetic unit with an unsupported provider claim
// Outputs: validation rejection before object-store work
// Effects: retained synthetic fixture only
// Choose: to prevent the earlier chatgpt-labelled request from becoming a destination authority.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitNeutralRouteRejectsProviderLabel(t *testing.T) {
	_, _, _, m := sourceUnitFixture(t, 3)
	if got := aiSourceDestinationPrefix(m); got != aiSourceUnitPrefix+"transcript/account-export/" {
		t.Fatalf("neutral destination changed: %s", got)
	}
	m.Provider = "chatgpt"
	if aiSourceUnitManifestValid(m) == nil {
		t.Fatal("unsupported provider claim admitted")
	}
}

// TestAISourceUnitAddedSourceMemberPreventsCopy refuses an unreviewed addition to the source unit.
// Inputs: frozen three-member discovery and one later-added source key
// Outputs: incomplete inspection and zero copies
// Effects: retained synthetic fixture only
// Choose: to prove the current prefix listing participates in unit completeness.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitAddedSourceMemberPreventsCopy(t *testing.T) {
	a, in, s, m := sourceUnitFixture(t, 3)
	s.heads[m.SourcePrefix+"unreviewed.txt@new-version"] = AISourceUnitMetadata{VersionID: "new-version", Bytes: 1}
	out, e := a.InspectAISourceUnit(context.Background(), AISourceUnitStageInput{Input: in})
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatalf("unreviewed source member admitted: %+v err=%v copies=%d", out, e, s.copies)
	}
}

// TestAISourceUnitWrongPredecessorPinPreventsCopy rejects altered receipt references and digests.
// Inputs: inspected/hashed unit with wrong predecessor digest
// Outputs: failure and zero transport
// Effects: fixture only
// Choose: to prove the chain is actually pinned.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitWrongPredecessorPinPreventsCopy(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 3)
	_, r := sourceUnitRun(t, a, in, 2)
	r.PreviousSHA256 = strings.Repeat("0", 64)
	out, e := a.CopyAISourceUnit(context.Background(), r)
	if e == nil || out.Complete || s.copies != 0 {
		t.Fatal("wrong predecessor accepted")
	}
}

// TestAISourceUnitWorkflowFiveRegisteredOperations exercises the real workflow with the five concrete Activity methods.
// Inputs: synthetic B2 fixture and reference-only request
// Outputs: complete 22-member proof
// Effects: retained fixture only
// Choose: for the root registration integration contract.
// Byline: Codex · GPT-6.1 · 2026-10-07.
func TestAISourceUnitWorkflowFiveRegisteredOperations(t *testing.T) {
	a, in, s, _ := sourceUnitFixture(t, 22)
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(AISourceUnitPlacementWorkflow, workflow.RegisterOptions{Name: AISourceUnitPlacementWorkflowName})
	env.RegisterActivityWithOptions(a.InspectAISourceUnit, activity.RegisterOptions{Name: AISourceUnitInspectActivityName})
	env.RegisterActivityWithOptions(a.HashAISourceUnitSource, activity.RegisterOptions{Name: AISourceUnitHashSourceActivityName})
	env.RegisterActivityWithOptions(a.CopyAISourceUnit, activity.RegisterOptions{Name: AISourceUnitCopyActivityName})
	env.RegisterActivityWithOptions(a.HashAISourceUnitDestination, activity.RegisterOptions{Name: AISourceUnitHashDestinationActivityName})
	env.RegisterActivityWithOptions(a.ReadbackAISourceUnit, activity.RegisterOptions{Name: AISourceUnitReadbackActivityName})
	env.ExecuteWorkflow(AISourceUnitPlacementWorkflowName, in)
	if e := env.GetWorkflowError(); e != nil {
		t.Fatal(e)
	}
	var out AISourceUnitSummary
	if e := env.GetWorkflowResult(&out); e != nil {
		t.Fatal(e)
	}
	if !out.PhysicalPlacementVerified || out.Objects != 22 || s.copies != 22 {
		t.Fatalf("invalid result %+v", out)
	}
}
