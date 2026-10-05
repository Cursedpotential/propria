// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/stretchr/testify/require"
)

func TestStatutePinpointCannotBorrowQuoteFromFollowingBareStatuteHeading(t *testing.T) {
	passage, err := LocatePassage([]string{"552.507 First statute.\n(1) Exact first text.\n552.508 Next statute.\n(1) Different unsupported claim."}, "MCL 552.507(1)")
	require.NoError(t, err)
	require.Contains(t, passage, "Exact first text.")
	require.NotContains(t, passage, "Different unsupported claim")
}

var fixtureNow = time.Date(2026, 10, 4, 23, 0, 0, 0, time.UTC)
var fixtureKey = []byte(strings.Repeat("validation-signing-", 3))
var fixtureCurrencyKey = []byte(strings.Repeat("independent-release-review-", 2))

const fixtureURL = "https://www.courts.michigan.gov/rules"
const fixtureQuote = "The parties may consent in writing to waive a judicial hearing."
const fixtureText = "Rule 3.215 Domestic Relations Referees\n(E) Judicial Hearings\n(3) Other text.\n(4) " + fixtureQuote + "\n(5) A different deadline.\n(F) Other rules.\nRule 3.216 Mediation\nDifferent rules."

func signedCurrency(t *testing.T, c Citation, sha string) *CurrencyEvidence {
	t.Helper()
	e := CurrencyEvidence{SourceID: c.SourceID, SourceVersion: c.SourceVersion, SnapshotSHA256: sha, PrimaryURL: fixtureURL, ReleaseURL: "https://www.courts.michigan.gov/current-releases", ReleaseVersion: "release-20261004-reviewed", ReleaseSHA256: strings.Repeat("c", 64), ReviewID: "trusted-review-20261004", EffectiveAt: fixtureNow.Add(-time.Hour), CheckedAt: fixtureNow.Add(-time.Minute), ExpiresAt: fixtureNow.Add(time.Hour), Status: Cleared}
	e.ReviewerID, e.DecisionID, e.ReviewedThrough = "fixture-authorized-reviewer", "fixture-decision", e.CheckedAt
	e.ReleasePinpoint, e.ReleaseQuoteSHA256 = "page:1", strings.Repeat("b", 64)
	e.ApprovalRef = ArtifactRef{URI: "b2://fixture/approval?versionId=v1", VersionID: "v1", SHA256: "sha256:" + strings.Repeat("c", 64), Bytes: 100}
	e.SourceSnapshotRef, e.ReleaseSnapshotRef = e.ApprovalRef, e.ApprovalRef
	sig, err := signJSON(e, fixtureCurrencyKey)
	require.NoError(t, err)
	e.Signature = sig
	return &e
}

func fixtureModel(t *testing.T, result verdict) *model.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer fake-nim-fixture", r.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, model.DefaultModelID, payload["model"])
		raw, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"model": model.DefaultModelID, "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(raw)}}}})
	}))
	t.Cleanup(server.Close)
	client, err := model.NewClient(model.Config{BaseURL: server.URL, ModelID: model.DefaultModelID, APIKey: "fake-nim-fixture", MaxTokens: model.DefaultMaxTokens})
	require.NoError(t, err)
	client.MaxAttempts = 1
	return client
}

func TestExactQuoteAndPinpointOverrideModelAgreement(t *testing.T) {
	c := Citation{SourceID: "source:mcr3215", SourceVersion: "sha256:" + strings.Repeat("a", 64), Pinpoint: "MCR 3.215(E)(4)", Claim: fixtureQuote}
	sha := strings.Repeat("b", 64)
	snapshot := Snapshot{Citation: c, Index: 0, PrimaryURL: fixtureURL, FinalURL: fixtureURL, Raw: ArtifactRef{URI: "b2://fixtures/validation/snapshot", SHA256: "sha256:" + sha, VersionID: "v1"}, FetchedAt: fixtureNow, Status: Fetched, Currency: signedCurrency(t, c, sha)}
	for _, test := range []struct {
		name, quote, pin, status string
		clear                    bool
		want, code               string
	}{
		{"exact supported claim", fixtureQuote, c.Pinpoint, Verified, true, Verified, ""},
		{"provisional current text", fixtureQuote, c.Pinpoint, Verified, false, Verified, "CURRENCY_NOT_CLEARED"},
		{"fabricated quote", "A signature automatically waives the hearing.", c.Pinpoint, Verified, true, Blocked, "EXACT_QUOTE_NOT_AT_PINPOINT"},
		{"quote on different subsection", "A different deadline.", c.Pinpoint, Verified, true, Blocked, "EXACT_QUOTE_NOT_AT_PINPOINT"},
		{"changed pinpoint", fixtureQuote, "MCR 3.215(E)(5)", Verified, true, Blocked, "MODEL_CITATION_MISMATCH"},
		{"overstated claim", fixtureQuote, c.Pinpoint, Conflicted, true, Conflicted, "CLAIM_CONFLICTED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := snapshot
			if !test.clear {
				s.Currency = nil
			}
			v := ClaimVerifier{Model: fixtureModel(t, verdict{Status: test.status, SourceID: c.SourceID, SourceVersion: c.SourceVersion, Pinpoint: test.pin, Quote: test.quote}), CurrencyKey: fixtureCurrencyKey, Now: func() time.Time { return fixtureNow }}
			check := v.Verify(t.Context(), s, Extracted{Pages: []string{fixtureText}, InputSHA256: sha, VersionID: "v1", Extractor: "html.beautifulsoup4", ExtractorVersion: "fixture"})
			require.Equal(t, test.want, check.Status)
			require.Equal(t, test.code, check.FailureCode)
			if check.Status == Verified {
				require.True(t, rawDigest(check.QuoteSHA256))
			}
			require.Equal(t, test.clear, check.CurrencyStatus == Cleared)
		})
	}
}

func TestPinpointPagesAndMissingLocations(t *testing.T) {
	text, err := LocatePassage([]string{"first page", "second page"}, "p. 2")
	require.NoError(t, err)
	require.Equal(t, "second page", text)
	_, err = LocatePassage([]string{"first page"}, "page:2")
	require.Error(t, err)
	_, err = LocatePassage([]string{fixtureText}, "MCR 3.215(E)(99)")
	require.Error(t, err)
	_, err = LocatePassage([]string{fixtureText}, "MCR 3.216(E)(4)")
	require.Error(t, err)
	passage, err := LocatePassage([]string{fixtureText}, "MCR 3.215(E)(4)")
	require.NoError(t, err)
	require.Contains(t, passage, fixtureQuote)
	require.NotContains(t, passage, "different deadline")
}

func TestCurrencyCannotBeClearedByFlagsOldEvidenceOrDifferentSnapshot(t *testing.T) {
	c := Citation{SourceID: "source:mcr3215", SourceVersion: "sha256:" + strings.Repeat("a", 64)}
	sha := strings.Repeat("b", 64)
	s := Snapshot{Citation: c, PrimaryURL: fixtureURL, Raw: ArtifactRef{SHA256: "sha256:" + sha}, Currency: signedCurrency(t, c, sha)}
	require.True(t, currencyCleared(s, fixtureNow, fixtureCurrencyKey))
	require.False(t, currencyCleared(s, fixtureNow, nil))
	s.Currency.Status = "provisional"
	require.False(t, currencyCleared(s, fixtureNow, fixtureCurrencyKey))
	s.Currency = signedCurrency(t, c, sha)
	s.Raw.SHA256 = "sha256:" + strings.Repeat("f", 64)
	require.False(t, currencyCleared(s, fixtureNow, fixtureCurrencyKey))
	s.Raw.SHA256 = "sha256:" + sha
	require.False(t, currencyCleared(s, fixtureNow.Add(2*time.Hour), fixtureCurrencyKey))
}
