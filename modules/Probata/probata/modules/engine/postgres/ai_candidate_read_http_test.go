package postgres_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/extraction/service"
	"github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/runtimeapi"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type nativeReadHTTPDB struct {
	postgres.DB
	row           service.AIReviewRow
	reads, writes int
}

// BeginTx refuses unexpected SQL writes from the authenticated HTTP read test.
// Inputs: unused transaction parameters. Outputs: error. Effects: records a write attempt.
func (d *nativeReadHTTPDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	d.writes++
	return nil, errors.New("HTTP read attempted a write transaction")
}

// Query supplies one saved quote-free candidate to the real PostgreSQL review adapter.
// Inputs: SELECT statement. Outputs: synthetic rows. Effects: records reads and refuses writes.
func (d *nativeReadHTTPDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	d.reads++
	if !strings.HasPrefix(sql, "SELECT ") {
		d.writes++
		return nil, errors.New("HTTP read issued a write")
	}
	return &nativeReadHTTPRows{row: d.row}, nil
}

type nativeReadHTTPRows struct {
	pgx.Rows
	row      service.AIReviewRow
	consumed bool
}

// Close releases no resources because the synthetic row owns no external cursor.
// Inputs/outputs/effects: none; use only for this HTTP fixture.
func (r *nativeReadHTTPRows) Close() {}

// Err reports successful exhaustion of the synthetic row fixture.
// Inputs: none. Outputs: nil. Effects: none; use only for this HTTP fixture.
func (r *nativeReadHTTPRows) Err() error { return nil }

// Next exposes the single fixed candidate once.
// Inputs: none. Outputs: row availability. Effects: advances the memory-only cursor.
func (r *nativeReadHTTPRows) Next() bool {
	if r.consumed {
		return false
	}
	r.consumed = true
	return true
}

// Scan transfers exact fixed candidate columns into the real store's destinations.
// Inputs: expected scan destinations. Outputs: nil. Effects: memory assignments only.
func (r *nativeReadHTTPRows) Scan(dest ...any) error {
	values := []any{r.row.ID, r.row.Kind, r.row.ReportedKind, r.row.ReviewDomain, r.row.ReviewState, r.row.ContentSHA256, r.row.DecisionID, r.row.Candidate, r.row.ReviewDecision}
	for i, value := range values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type nativeReadHTTPStore struct {
	*postgres.EntityExtractionStore
	pin service.AISourcePin
}

// VerifyAISource isolates existing source admission from the quote-hydration HTTP contract.
// Inputs: exact synthetic source pin without a preview handle. Outputs: LIVE or refusal. Effects: none.
func (s nativeReadHTTPStore) VerifyAISource(_ context.Context, handle string, pin service.AISourcePin) (string, error) {
	if handle != "" || !reflect.DeepEqual(pin, s.pin) {
		return "", errors.New("unexpected authenticated HTTP source pin")
	}
	return "LIVE", nil
}

type nativeReadHTTPWorkflows struct{ runtimeapi.ExtractionWorkflows }

// TestNativeCandidateHTTPReadReturnsExternalQuoteAndSavedDecisionDigest exercises the existing authenticated GET route.
// Inputs: real PostgreSQL review adapter over read-only synthetic rows and immutable external fixtures; source admission is stubbed independently.
// Outputs: verified short quote and unchanged saved content SHA, or refusal after bundle tampering.
// Effects: one SELECT per request and local retained reads, with zero SQL transaction/write attempts or workflow dispatches.
func TestNativeCandidateHTTPReadReturnsExternalQuoteAndSavedDecisionDigest(t *testing.T) {
	candidate, row, root := postgres.NativeReadFixtureForTest(t)
	db := &nativeReadHTTPDB{row: row}
	store, err := postgres.NewEntityExtractionStore(db)
	require.NoError(t, err)
	token := strings.Repeat("s", 32)
	tokenPath := filepath.Join(root, "synthetic-service-token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(token), 0600))
	handler, err := runtimeapi.NewEntityExtractionHTTPHandler(nativeReadHTTPStore{EntityExtractionStore: store, pin: candidate.AISourcePin}, nativeReadHTTPWorkflows{}, tokenPath)
	require.NoError(t, err)
	query := url.Values{"matter_mode": {"LIVE"}, "source_version_id": {candidate.SourceVersionID}, "source_sha256": {candidate.SourceSHA256}, "source_ref": {candidate.SourceRef}, "prepared_ref": {candidate.PreparedRef}, "version_id": {*candidate.VersionID}}
	invoke := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/reference-import/ai-candidates?"+query.Encode(), nil)
		request.RemoteAddr = "100.64.1.9:3456"
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-authentik-uid", "synthetic-actor")
		request.Header.Set("X-authentik-username", "synthetic-owner")
		response := httptest.NewRecorder()
		handler.Routes().ServeHTTP(response, request)
		return response
	}
	response := invoke()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"evidence_quote":"🧭"`)
	require.Contains(t, response.Body.String(), `"content_sha256":"`+row.ContentSHA256+`"`)
	require.NoError(t, os.WriteFile(filepath.Join(root, "candidates.json"), []byte(`{}`), 0600))
	response = invoke()
	require.NotEqual(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), "🧭")
	require.Equal(t, 2, db.reads)
	require.Zero(t, db.writes)
	require.Equal(t, row, db.row)
}
