package postgres

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/aicontextsource"
	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// NativeReadFixtureForTest writes a retained synthetic exact-source bundle and its quote-free saved row.
// Inputs: explicit VPS fixture root. Outputs: native candidate and immutable review digest.
// Effects: synthetic external files only, preserved for owner cleanup; no live data access.
func NativeReadFixtureForTest(t *testing.T) (service.AICandidate, service.AIReviewRow, string) {
	t.Helper()
	base := os.Getenv("AI_NATIVE_REPAIR_TEST_ROOT")
	if base == "" {
		t.Skip("requires explicit retained VPS fixture root")
	}
	root, err := os.MkdirTemp(base, "native-read-*")
	require.NoError(t, err)
	t.Setenv("AI_CONTEXT_ROOT", root)
	ref := func(name string) string {
		return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(root, name))}).String()
	}
	write := func(name string, raw []byte) { require.NoError(t, os.WriteFile(filepath.Join(root, name), raw, 0600)) }
	original := []byte(`{"turns":[{"text":"A🧭B"}]}`)
	write("original.json", original)
	provider := "exact-provider-version"
	identity := aicontextsource.Identity{SourceRef: "b2://fixture/native.json", SourceVersionID: "11111111-1111-4111-8111-111111111111", ProviderVersionID: &provider, SourceFormat: "chatgpt", SourceSHA256: aicontextsource.Hash(original)}
	prepared := aicontextsource.Prepared{ContractVersion: "ai-context-v1", Stage: "prepared", Source: identity, OriginalRef: ref("original.json"), OriginalSHA256: identity.SourceSHA256}
	raw, err := json.Marshal(prepared)
	require.NoError(t, err)
	write("prepared.json", raw)
	raw, err = json.Marshal(map[string]any{"contract_version": "ai-context-v1", "stage": "candidates", "source": identity, "prepared_ref": ref("prepared.json"), "candidates": []any{map[string]any{"kind": "entity", "name": "Fixture Person", "entity_type": "person", "quote": "🧭", "status": "unreviewed_context_candidate", "confidence": 0.8, "occurrences": []any{map[string]any{"native_json_pointer": "/turns/0/text", "start": 1, "end": 2, "unit": "unicode_codepoint"}}}}})
	require.NoError(t, err)
	write("candidates.json", raw)
	full := service.AICandidate{AISourcePin: service.AISourcePin{SourceVersionID: identity.SourceVersionID, SourceRef: identity.SourceRef, VersionID: &provider, SourceSHA256: identity.SourceSHA256, PreparedRef: ref("prepared.json")}, NativeJSONPointer: "/turns/0/text", SourceSpan: service.AISourceSpan{Start: 1, End: 2, SHA256: aicontextsource.Hash([]byte("🧭"))}, SpanUnit: "unicode_codepoint", Kind: "entity", ReportedKind: "entity", ReviewDomain: "ai_chat_content", Name: "Fixture Person", EntityType: "person", EvidenceQuote: "🧭", EvidenceBundleRef: ref("candidates.json"), EvidenceBundleSHA256: aicontextsource.Hash(raw), Confidence: 0.8}
	require.NoError(t, service.ValidateAICandidate(full))
	saved, err := aiCandidateForPersistence(full)
	require.NoError(t, err)
	return saved, nativeReadRow(t, saved), root
}

// nativeReadRow projects a saved typed candidate into the exact store read shape without altering its digest.
// Inputs: PostgreSQL-safe candidate. Outputs: synthetic review row. Effects: none.
func nativeReadRow(t *testing.T, c service.AICandidate) service.AIReviewRow {
	t.Helper()
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	digest := service.AICandidateDigest(c)
	return service.AIReviewRow{ID: "22222222-2222-4222-8222-222222222222", Kind: c.Kind, ReportedKind: c.ReportedKind, ReviewDomain: c.ReviewDomain, ReviewState: "pending", ContentSHA256: hex.EncodeToString(digest[:]), Candidate: raw, ReviewDecision: json.RawMessage(`{}`)}
}

// TestNativeCandidateReadHydratesQuoteWithoutChangingStoredDigest proves exact Unicode quote recovery and page-local caching.
// Inputs: 200 reads of one quote-free saved row. Outputs: verified response quote and unchanged decision digest.
// Effects: one external candidates read and one prepared/original read; no PostgreSQL write seam exists.
func TestNativeCandidateReadHydratesQuoteWithoutChangingStoredDigest(t *testing.T) {
	saved, row, _ := NativeReadFixtureForTest(t)
	originalJSON := append([]byte(nil), row.Candidate...)
	page := newNativeCandidateReadPage()
	bundleReads, preparedReads := 0, 0
	page.readFile = func(ref, base string) ([]byte, string, error) {
		bundleReads++
		return aicontextsource.ReadFile(ref, base)
	}
	page.readPrepared = func(ref string) (aicontextsource.Prepared, []byte, error) {
		preparedReads++
		return aicontextsource.ReadPrepared(ref)
	}
	for i := 0; i < 200; i++ {
		hydrated, err := page.hydrate(saved.SourceVersionID, row)
		require.NoError(t, err)
		var response service.AICandidate
		require.NoError(t, json.Unmarshal(hydrated.Candidate, &response))
		require.Equal(t, "🧭", response.EvidenceQuote)
		require.Equal(t, row.ContentSHA256, hydrated.ContentSHA256)
		projection, err := aiCandidateForPersistence(response)
		require.NoError(t, err)
		require.Equal(t, saved, projection)
	}
	require.Equal(t, 1, bundleReads)
	require.Equal(t, 1, preparedReads)
	require.Equal(t, originalJSON, []byte(row.Candidate))
	require.NotContains(t, string(row.Candidate), "evidence_quote")
	legacy := row
	legacy.Candidate = json.RawMessage(`{"source_version_id":"old","evidence_quote":"preserved legacy excerpt"}`)
	unchanged, err := page.hydrate("old", legacy)
	require.NoError(t, err)
	require.Equal(t, legacy, unchanged)
}

// TestNativeCandidateReadRefusesChangedHashPinsAndOriginalSpan proves fail-closed hydration without trusting filenames.
// Inputs: independently mutated synthetic source, bundle, span and saved digest. Outputs: refusal with no quote response.
// Effects: retained fixtures only; no database or provider activity.
func TestNativeCandidateReadRefusesChangedHashPinsAndOriginalSpan(t *testing.T) {
	for _, kind := range []string{"bundle-hash", "bundle-missing", "provider", "source-version", "source-ref", "prepared-ref", "pointer", "span", "span-hash", "stored-digest", "bundle-tamper", "original-tamper", "bundle-version"} {
		t.Run(kind, func(t *testing.T) {
			candidate, row, root := NativeReadFixtureForTest(t)
			switch kind {
			case "bundle-hash":
				candidate.EvidenceBundleSHA256 = strings.Repeat("a", 64)
			case "bundle-missing":
				candidate.EvidenceBundleRef = strings.Replace(candidate.EvidenceBundleRef, "candidates.json", "missing/candidates.json", 1)
			case "provider":
				different := "other-provider"
				candidate.VersionID = &different
			case "source-version":
				candidate.SourceVersionID = "33333333-3333-4333-8333-333333333333"
			case "source-ref":
				candidate.SourceRef = "b2://fixture/different.json"
			case "prepared-ref":
				candidate.PreparedRef = strings.Replace(candidate.PreparedRef, "prepared.json", "missing/prepared.json", 1)
			case "pointer":
				candidate.NativeJSONPointer = "/turns/1/text"
			case "span":
				candidate.SourceSpan.Start = 0
				candidate.SourceSpan.End = 1
			case "span-hash":
				candidate.SourceSpan.SHA256 = strings.Repeat("b", 64)
			case "stored-digest":
				row.ContentSHA256 = strings.Repeat("c", 64)
			case "bundle-tamper":
				require.NoError(t, os.WriteFile(filepath.Join(root, "candidates.json"), []byte(`{}`), 0600))
			case "original-tamper":
				require.NoError(t, os.WriteFile(filepath.Join(root, "original.json"), []byte(`{"changed":true}`), 0600))
			case "bundle-version":
				raw, err := os.ReadFile(filepath.Join(root, "candidates.json"))
				require.NoError(t, err)
				raw = []byte(strings.ReplaceAll(string(raw), candidate.SourceVersionID, "33333333-3333-4333-8333-333333333333"))
				require.NoError(t, os.WriteFile(filepath.Join(root, "candidates.json"), raw, 0600))
				candidate.EvidenceBundleSHA256 = aicontextsource.Hash(raw)
			}
			if kind != "stored-digest" {
				row = nativeReadRow(t, candidate)
			}
			_, err := newNativeCandidateReadPage().hydrate(candidate.SourceVersionID, row)
			require.Error(t, err)
		})
	}
}

type nativeCandidateReadDB struct {
	DB
	rows                 []service.AIReviewRow
	reads, writeAttempts int
}

// BeginTx records and rejects any transaction attempted by the read-only fixture.
// Inputs: unused SQL transaction parameters. Outputs: error. Effects: increments the write-attempt counter.
func (d *nativeCandidateReadDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	d.writeAttempts++
	return nil, errors.New("read must never open a transaction")
}

// Query returns synthetic saved candidate rows only for a SELECT statement.
// Inputs: store SQL. Outputs: fixed rows or a refusal. Effects: records reads and unexpected writes.
func (d *nativeCandidateReadDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	d.reads++
	if !strings.HasPrefix(sql, "SELECT ") {
		d.writeAttempts++
		return nil, errors.New("read issued a write statement")
	}
	rows := [][]any{}
	for _, row := range d.rows {
		rows = append(rows, []any{row.ID, row.Kind, row.ReportedKind, row.ReviewDomain, row.ReviewState, row.ContentSHA256, row.DecisionID, row.Candidate, row.ReviewDecision})
	}
	return &reviewFakeRows{rows: rows}, nil
}

// TestListNativeCandidatesReturnsVerifiedAPIProjectionWithoutDatabaseWrites exercises the actual store read hook.
// Inputs: quote-free SQL rows plus immutable synthetic external evidence. Outputs: API-shaped quote with stable saved digest.
// Effects: one SELECT and retained local reads; any transaction or SQL write attempt fails the test.
func TestListNativeCandidatesReturnsVerifiedAPIProjectionWithoutDatabaseWrites(t *testing.T) {
	saved, row, _ := NativeReadFixtureForTest(t)
	db := &nativeCandidateReadDB{rows: []service.AIReviewRow{row}}
	store := &EntityExtractionStore{db: db}
	response, err := store.ListAICandidates(context.Background(), saved.SourceVersionID, "", 201)
	require.NoError(t, err)
	require.Len(t, response, 1)
	raw, err := json.Marshal(map[string]any{"candidates": response})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"evidence_quote":"🧭"`)
	require.Equal(t, row.ContentSHA256, response[0].ContentSHA256)
	require.Equal(t, 1, db.reads)
	require.Zero(t, db.writeAttempts)
	require.Equal(t, row, db.rows[0])
}
