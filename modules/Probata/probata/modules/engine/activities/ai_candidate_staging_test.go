// Byline: Codex · GPT-6.1-sol · 2026-10-07.
package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
)

type aiStageStore struct {
	verified     int
	resolved     int
	boundPreview string
	staged       []service.AICandidate
	runID        string
	digest       string
}

func (s *aiStageStore) ResolveAIPreview(_ context.Context, _ string, _ service.AISourcePin) (string, error) {
	s.resolved++
	return s.boundPreview, nil
}

func (s *aiStageStore) VerifyAISource(context.Context, string, service.AISourcePin) (string, error) {
	s.verified++
	return "LIVE", nil
}
func (s *aiStageStore) StageAICandidates(_ context.Context, _, runID, digest string, rows []service.AICandidate) ([]string, error) {
	s.runID, s.digest = runID, digest
	s.staged = append([]service.AICandidate(nil), rows...)
	ids := make([]string, len(rows))
	for i := range ids {
		ids[i] = "pending"
	}
	return ids, nil
}
func (*aiStageStore) ListAICandidates(context.Context, string, string, int) ([]service.AIReviewRow, error) {
	return nil, nil
}
func (*aiStageStore) DecideAICandidate(context.Context, service.AISourcePin, string, string, string, entities.Actor, string, time.Time) (string, error) {
	return "", nil
}

func aiStageFixture(t *testing.T) (AICandidateStagingActivities, AICandidateStageInput, map[string]any) {
	t.Helper()
	root := t.TempDir()
	version := "11111111-1111-4111-8111-111111111111"
	object := "22222222-2222-4222-8222-222222222222"
	sourceHash := strings.Repeat("a", 64)
	pin := service.AISourcePin{SourceVersionID: version, SourceObjectID: object, SourceSHA256: sourceHash}
	occurrence := func(start, end int) map[string]any {
		quote := "😀"
		sum := sha256.Sum256([]byte(quote))
		return map[string]any{"source_version_id": version, "source_object_id": object, "version_id": nil,
			"source_sha256": sourceHash, "native_json_pointer": "/conversations/0/text", "span_unit": "unicode_codepoint",
			"source_span": map[string]any{"start": start, "end": end, "sha256": hex.EncodeToString(sum[:]), "unit": "unicode_codepoint"},
			"quote":       quote, "evidence_quote": quote}
	}
	bundle := map[string]any{"version": "ai-content-native-v3", "stage": "candidates",
		"pins":              map[string]any{"source_version_id": version},
		"source":            map[string]any{"source_version_id": version, "original_object_id": object, "original_sha256": sourceHash, "version_id": nil},
		"work_products_ref": "file:///retained/manifest.json",
		"candidates": []any{map[string]any{"kind": "fact", "reported_kind": "fact", "review_domain": "ai_chat_content",
			"bridge_disposition": "pending_go_candidate", "classification_status": "typed", "predicate": "said", "statement": "Example account",
			"confidence": 0.75, "occurred_at": nil, "occurrences": []any{occurrence(0, 1), occurrence(2, 3)}}}}
	path := filepath.Join(root, version, "candidates", "bundle.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	ref := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	input := AICandidateStageInput{PreviewHandle: "preview-1", Source: pin, BundleRef: ref}
	return AICandidateStagingActivities{DerivedRoot: root, Store: &aiStageStore{}}, input, bundle
}

func writeAIStageBundle(t *testing.T, in *AICandidateStageInput, bundle map[string]any) {
	t.Helper()
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(in.BundleRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.FromSlash(u.Path), raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	in.BundleSHA256 = hex.EncodeToString(sum[:])
}

func TestAICandidateStagePreservesUnicodeOccurrencesAndRetryIdentity(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	writeAIStageBundle(t, &input, bundle)
	first, err := activity.StageAICandidateBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	store := activity.Store.(*aiStageStore)
	if first.Candidates != 1 || first.Staged != 2 || first.HeldManifest != 0 || first.WorkProductsRef == "" || len(store.staged) != 2 {
		t.Fatalf("incorrect bounded result: %+v, rows=%d", first, len(store.staged))
	}
	if store.staged[0].SourceSpan.Start != 0 || store.staged[1].SourceSpan.Start != 2 || store.staged[0].EvidenceQuote != "😀" {
		t.Fatalf("lost codepoint occurrences: %+v", store.staged)
	}
	second, err := activity.StageAICandidateBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || store.verified != 2 || store.runID != first.RunID || store.digest != first.RequestDigest {
		t.Fatalf("retry identity drift: first=%+v second=%+v", first, second)
	}
}

func TestAICandidateStageResolvesDurableRequestBeforeStaging(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	store := activity.Store.(*aiStageStore)
	store.boundPreview = input.PreviewHandle
	input.RequestID = "request-1"
	input.PreviewHandle = ""
	writeAIStageBundle(t, &input, bundle)
	got, err := activity.StageAICandidateBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Staged != 2 || store.resolved != 1 || store.verified != 1 {
		t.Fatalf("durable preview was not resolved and verified: %+v", got)
	}
}

func TestAICandidateStageRejectsPreviewThatDiffersFromRequestBinding(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	store := activity.Store.(*aiStageStore)
	store.boundPreview = "other-preview"
	input.RequestID = "request-1"
	writeAIStageBundle(t, &input, bundle)
	if _, err := activity.StageAICandidateBundle(context.Background(), input); err == nil {
		t.Fatal("accepted preview handle inconsistent with durable request binding")
	}
	if store.verified != 0 || len(store.staged) != 0 {
		t.Fatal("used unbound preview")
	}
}

func TestAICandidateStageHoldsNullConfidenceWithoutInventingZero(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	item := bundle["candidates"].([]any)[0].(map[string]any)
	item["confidence"] = nil
	item["classification_status"] = "unclassified"
	item["bridge_disposition"] = "held_unclassified"
	writeAIStageBundle(t, &input, bundle)
	got, err := activity.StageAICandidateBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.HeldUnclassified != 2 || got.Staged != 0 || len(activity.Store.(*aiStageStore).staged) != 0 {
		t.Fatalf("null confidence staged: %+v", got)
	}
}

func TestAICandidateStageReportsDuplicateOccurrenceWithoutDroppingOtherRows(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	candidates := bundle["candidates"].([]any)
	bundle["candidates"] = append(candidates, candidates[0])
	writeAIStageBundle(t, &input, bundle)
	got, err := activity.StageAICandidateBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Staged != 2 || got.HeldDuplicate != 2 || len(activity.Store.(*aiStageStore).staged) != 2 {
		t.Fatalf("duplicate review identity was not accounted for: %+v", got)
	}
}

func TestAICandidateStageRejectsSourceAndPersistedHashMismatch(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	writeAIStageBundle(t, &input, bundle)
	input.BundleSHA256 = strings.Repeat("b", 64)
	if _, err := activity.StageAICandidateBundle(context.Background(), input); err == nil {
		t.Fatal("accepted changed persisted hash")
	}
	writeAIStageBundle(t, &input, bundle)
	bundle["source"].(map[string]any)["original_sha256"] = strings.Repeat("b", 64)
	writeAIStageBundle(t, &input, bundle)
	if _, err := activity.StageAICandidateBundle(context.Background(), input); err == nil {
		t.Fatal("accepted changed original source pin")
	}
}

func TestAICandidateStageRejectsEscapedAndLinkedPaths(t *testing.T) {
	activity, input, bundle := aiStageFixture(t)
	writeAIStageBundle(t, &input, bundle)
	input.BundleRef = "file:///outside/candidates/bundle.json"
	if _, err := activity.StageAICandidateBundle(context.Background(), input); err == nil {
		t.Fatal("accepted outside path")
	}
	// A source-directory symlink cannot make a lexical child of the root safe.
	input.BundleRef = (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(activity.DerivedRoot, input.Source.SourceVersionID, "linked", "bundle.json"))}).String()
	u, _ := url.Parse(input.BundleRef)
	linked := filepath.Dir(filepath.FromSlash(u.Path))
	if err := os.Symlink(filepath.Join(activity.DerivedRoot, input.Source.SourceVersionID, "candidates"), linked); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := activity.StageAICandidateBundle(context.Background(), input); err == nil {
		t.Fatal("accepted linked directory")
	}
}
