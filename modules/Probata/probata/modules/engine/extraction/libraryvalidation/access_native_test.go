// Byline: Codex · GPT-6.1 · 2026-10-04.
package libraryvalidation

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/surrealsink"
	"github.com/stretchr/testify/require"
)

func TestRecordAccessAgainstIsolatedSurreal324(t *testing.T) {
	endpoint := os.Getenv("TOOLKIT_VALIDATION_SCOPE_TEST_URL")
	if endpoint == "" {
		t.Skip("explicit isolated 3.2.4 fixture required")
	}
	require.Equal(t, "http://127.0.0.1:18045", endpoint, "never connect this probe to production")
	for _, role := range []struct{ access, principal, user string }{{ValidatorAccess, ValidatorPrincipal, "fixture-validator"}, {CurrencyAccess, CurrencyPrincipal, "fixture-currency"}} {
		client, err := NewAccessSQLClient(AccessConfig{DB: surrealsink.Config{URL: endpoint, Namespace: "toolkit_scope_fixture", Database: "case", User: role.user, Password: "synthetic-record-password", AuthLevel: "database"}, Access: role.access, Principal: role.principal}, nil)
		require.NoError(t, err)
		results, err := client.Query(t.Context(), "RETURN <string> $auth; SELECT * FROM source;", nil)
		require.NoError(t, err)
		var principal string
		require.NoError(t, json.Unmarshal(results[0], &principal))
		require.Equal(t, role.principal, principal)
		var rows []map[string]any
		require.NoError(t, json.Unmarshal(results[1], &rows))
		require.Len(t, rows, 1)
		literalResults, err := client.Query(t.Context(), "RETURN [$source, $digest, $when];", map[string]any{"source": "source:official", "digest": "sha256:" + strings.Repeat("a", 64), "when": "2026-10-04T00:00:00Z"})
		require.NoError(t, err)
		require.Len(t, literalResults, 1)
		var literals []string
		require.NoError(t, json.Unmarshal(literalResults[0], &literals))
		require.Equal(t, []string{"source:official", "sha256:" + strings.Repeat("a", 64), "2026-10-04T00:00:00Z"}, literals)
		var record map[string]any
		if role.access == ValidatorAccess {
			saved, err := client.Query(t.Context(), "SELECT * FROM ONLY type::record('library_validation',$key);", map[string]any{"key": "11111111-2222-3333-4444-555555555555"})
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(saved[0], &record))
			delete(record, "id") // Receipt input has no database-managed id field.
			_, err = client.Query(t.Context(), commitSQL, map[string]any{"key": "11111111-2222-3333-4444-555555555555", "receipt": record})
			require.NoError(t, err, "actual scoped Go receipt writer must commit idempotently")
		} else {
			saved, err := client.Query(t.Context(), "SELECT * FROM ONLY library_currency:official;", nil)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(saved[0], &record))
			delete(record, "id")
			_, err = client.Query(t.Context(), currencyCommitSQL, map[string]any{"evidence": record})
			require.NoError(t, err, "actual scoped Go currency writer must commit idempotently")
		}
	}
}
