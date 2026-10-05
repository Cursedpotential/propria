// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/stretchr/testify/require"
)

func currencyReviewFixture(t *testing.T) (*CurrencyReviewService, CurrencyReviewApproval, *memoryRepository, *int) {
	t.Helper()
	s, repo, store := serviceFixture(t)
	plan, err := s.Prepare(t.Context(), repo.proposal.ID)
	require.NoError(t, err)
	primary, err := s.SourceSnapshot(t.Context(), ClaimInput{Plan: plan, Index: 0})
	require.NoError(t, err)
	calls := new(int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		user, _, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "currency-writer-only", user)
		var payload struct {
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		var sql string
		require.NoError(t, json.Unmarshal(payload.Params[0], &sql))
		require.Equal(t, currencyCommitSQL, sql)
		var vars struct {
			Evidence CurrencyEvidence `json:"evidence"`
		}
		require.NoError(t, json.Unmarshal(payload.Params[1], &vars))
		unsigned := vars.Evidence
		unsigned.Signature = ""
		require.True(t, matchesSignature(unsigned, vars.Evidence.Signature, fixtureCurrencyKey))
		raw, _ := json.Marshal(vars.Evidence)
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &fields))
		repo.records["library_currency:mcr3215"] = Record{ID: "library_currency:mcr3215", Fields: fields}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": []any{map[string]any{"status": "OK", "result": map[string]any{"signature": vars.Evidence.Signature}}}})
	}))
	t.Cleanup(server.Close)
	reviewer := &CurrencyReviewService{Repository: repo, Artifacts: store, Extractor: s.Extractor, Fetcher: s.Fetcher, SigningKey: fixtureKey, CurrencyKey: fixtureCurrencyKey, ApprovalKey: []byte(strings.Repeat("protected-approval-authority-", 2)), Now: s.Now, DB: SQLClient{Config: surrealsink.Config{URL: server.URL, Namespace: "fct", Database: "fixture", AuthLevel: "database", User: "currency-writer-only", Password: "fixture"}, HTTP: server.Client()}}
	release, err := reviewer.CaptureRelease(t.Context(), fixtureURL)
	require.NoError(t, err)
	approval := CurrencyReviewApproval{ReviewID: "synthetic-reviewed-release", ReviewerID: "authenticated-reviewer", DecisionID: "authorized-decision", ReviewScope: CurrencyReviewScope, SourceSnapshotRef: primary.Ref, ReleaseSnapshotRef: release, ReleaseVersion: "fixture-current-release", ReleasePinpoint: "MCR 3.215(E)(4)", ReleaseQuote: fixtureQuote, EffectiveAt: fixtureNow.Add(-time.Hour), ReviewedThrough: fixtureNow, ApprovedAt: fixtureNow, ExpiresAt: fixtureNow.Add(time.Hour)}
	approval, err = SignCurrencyReviewApproval(approval, reviewer.ApprovalKey)
	require.NoError(t, err)
	return reviewer, approval, repo, calls
}

func TestCurrencyCreatorVerifiesRetainedEvidenceAndIndependentAuthority(t *testing.T) {
	s, approval, repo, calls := currencyReviewFixture(t)
	evidence, err := s.CommitApprovedReview(t.Context(), approval)
	require.NoError(t, err)
	require.Equal(t, 1, *calls)
	require.Equal(t, Cleared, evidence.Status)
	require.Equal(t, approval.ReviewerID, evidence.ReviewerID)
	require.Equal(t, approval.SourceSnapshotRef, evidence.SourceSnapshotRef)
	require.True(t, validPinnedRef(evidence.ApprovalRef))
	require.Equal(t, strings.TrimPrefix(Hash([]byte(fixtureQuote)), "sha256:"), evidence.ReleaseQuoteSHA256)
	var snapshot Snapshot
	require.NoError(t, readSigned(t.Context(), s.Artifacts, approval.SourceSnapshotRef, &snapshot, s.SigningKey))
	snapshot.Currency = &evidence
	require.True(t, currencyCleared(snapshot, fixtureNow, s.CurrencyKey))
	require.Zero(t, repo.commits, "currency creator must not commit validations or publish")
	require.Equal(t, "pending_validation", repo.proposal.Status)
	again, err := s.CommitApprovedReview(t.Context(), approval)
	require.NoError(t, err)
	require.Equal(t, evidence, again)
	require.Equal(t, 1, *calls, "lost-reply retry must reuse the original signed provider-pinned audit")
}

func TestCurrencyCreatorRejectsFlagsOldHorizonFalseQuoteAndStaleSource(t *testing.T) {
	for _, mode := range []string{"unsigned flag", "wrong authority", "old review horizon", "scope incomplete", "quote fabricated", "wrong pinpoint", "source changed", "snapshot tampered", "shared keys"} {
		t.Run(mode, func(t *testing.T) {
			s, a, repo, calls := currencyReviewFixture(t)
			switch mode {
			case "unsigned flag":
				a.Signature = "cleared"
			case "wrong authority":
				a.Signature, _ = signJSON(CurrencyReviewApproval{}, fixtureKey)
			case "old review horizon":
				a.ReviewedThrough = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
			case "scope incomplete":
				a.ReviewScope = "current_html_text_only"
			case "quote fabricated":
				a.ReleaseQuote = "An automatic signature waiver exists."
			case "wrong pinpoint":
				a.ReleasePinpoint = "MCR 3.215(E)(5)"
			case "source changed":
				r := repo.records["source:mcr3215"]
				r.Version = "sha256:" + strings.Repeat("9", 64)
				repo.records[r.ID] = r
			case "snapshot tampered":
				s.Artifacts.(*memoryArtifacts).bodies[a.ReleaseSnapshotRef.URI][0] ^= 1
			case "shared keys":
				s.ApprovalKey = s.SigningKey
			}
			if mode != "unsigned flag" && mode != "wrong authority" {
				var err error
				a, err = SignCurrencyReviewApproval(a, s.ApprovalKey)
				require.NoError(t, err)
			}
			_, err := s.CommitApprovedReview(t.Context(), a)
			require.Error(t, err)
			require.Zero(t, *calls, "invalid review must never reach privileged writer")
		})
	}
}
