package runtimeapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/stretchr/testify/require"
)

type aiAPIStore struct {
	*apiStore
	stageDigest  string
	decideDigest string
	failStage    bool
}

func (s *aiAPIStore) VerifyAISource(context.Context, string, service.AISourcePin) (string, error) {
	return "REAL", nil
}
func (s *aiAPIStore) ResolveAIPreview(context.Context, string, service.AISourcePin) (string, error) {
	return extractionHandle, nil
}
func (s *aiAPIStore) StageAICandidates(_ context.Context, _, _, digest string, _ []service.AICandidate) ([]string, error) {
	s.stageDigest = digest
	if s.failStage {
		return nil, errors.New("retained-source AI staging needs an original-object opener")
	}
	return []string{"33333333-3333-4333-8333-333333333333"}, nil
}
func (s *aiAPIStore) ListAICandidates(context.Context, string, string, int) ([]service.AIReviewRow, error) {
	return []service.AIReviewRow{{ID: "33333333-3333-4333-8333-333333333333", Kind: "fact", ReportedKind: "history", ReviewDomain: "ai_chat_account", ReviewState: "pending", ContentSHA256: strings.Repeat("a", 64), Candidate: []byte(`{"kind":"fact"}`)}}, nil
}
func (s *aiAPIStore) DecideAICandidate(_ context.Context, _ service.AISourcePin, _, expected, _ string, _ entities.Actor, digest string, _ time.Time) (string, error) {
	s.decideDigest = digest
	if expected != strings.Repeat("a", 64) {
		return "", entities.ErrConflict
	}
	return "44444444-4444-4444-8444-444444444444", nil
}

// TestAIReviewRoutesRequireActorKeyAndExpectedDigest checks retained-source HTTP gates without a database.
// Inputs: synthetic source pin, candidate and owner requests. Outputs: stage/list/decision and fail-closed statuses.
// Effects: mock state only. Choose to guard the new route contract while SMS routes retain their own tests.
func TestAIReviewRoutesRequireActorKeyAndExpectedDigest(t *testing.T) {
	store := &aiAPIStore{apiStore: &apiStore{}}
	handler, err := NewEntityExtractionHTTPHandler(store, &workflowRecorder{}, serviceTokenPath(t))
	require.NoError(t, err)
	prefix := map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL", "source_version_id": "11111111-1111-4111-8111-111111111111", "source_object_id": "22222222-2222-4222-8222-222222222222", "version_id": nil, "source_sha256": strings.Repeat("a", 64)}
	stage := map[string]any{"preview_handle": extractionHandle, "matter_mode": "REAL", "candidates": []map[string]any{{"source_version_id": prefix["source_version_id"], "source_object_id": prefix["source_object_id"], "source_sha256": prefix["source_sha256"], "version_id": nil, "kind": "history", "reported_kind": "history", "review_domain": "ai_chat_account", "native_json_pointer": "/turns/0/text", "span_unit": "unicode_codepoint", "source_span": map[string]any{"start": 0, "end": 4, "sha256": strings.Repeat("b", 64)}, "evidence_quote": "text", "statement": "A bounded account", "confidence": 0.8}}}
	missing := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/ai-candidates/stage", stage, ""))
	require.Equal(t, http.StatusUnauthorized, missing.Code)
	started := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/ai-candidates/stage", stage, "stage-1"))
	require.Equal(t, http.StatusAccepted, started.Code, started.Body.String())
	require.Len(t, store.stageDigest, 64)
	read := servePreview(handler.Routes(), http.MethodGet, "/reference-import/ai-candidates?preview_handle="+extractionHandle+"&matter_mode=REAL&source_version_id=11111111-1111-4111-8111-111111111111&source_object_id=22222222-2222-4222-8222-222222222222&source_sha256="+strings.Repeat("a", 64), nil)
	require.Equal(t, http.StatusOK, read.Code, read.Body.String())
	require.Contains(t, read.Body.String(), `"reported_kind":"history"`)
	decision := map[string]any{}
	for key, value := range prefix {
		decision[key] = value
	}
	decision["candidate_id"] = "33333333-3333-4333-8333-333333333333"
	decision["decision"] = "approved"
	decision["expected_content_sha256"] = strings.Repeat("b", 64)
	stale := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/ai-candidates/decision", decision, "decision-1"))
	require.Equal(t, http.StatusConflict, stale.Code, stale.Body.String())
	decision["expected_content_sha256"] = strings.Repeat("a", 64)
	approved := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/ai-candidates/decision", decision, "decision-2"))
	require.Equal(t, http.StatusOK, approved.Code, approved.Body.String())
	require.Contains(t, approved.Body.String(), `"decision_id":"44444444-4444-4444-8444-444444444444"`)
	require.Len(t, store.decideDigest, 64)
	store.failStage = true
	unavailable := servePreviewRequest(handler.Routes(), extractionRequest(http.MethodPost, "/reference-import/ai-candidates/stage", stage, "stage-2"))
	require.Equal(t, http.StatusServiceUnavailable, unavailable.Code)
}
