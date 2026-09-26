// Byline: Claude Code · Opus 5 · 2026-09-20

package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/parser"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/stretchr/testify/require"
)

// TestSMSBackupSignatureRoutesOnlyToDerive proves the routing change: the
// smsbackuprestore_xml signature yields exactly one candidate and it is the
// derive path. The DuckDB sms_xml_v1 template is NOT offered as an
// alternative — it fails on every real backup, and the owner's standing rule
// is one registry entry per signature with no fallback ladder.
func TestSMSBackupSignatureRoutesOnlyToDerive(t *testing.T) {
	decoder := parser.Capability{ParserID: "sbv_sms_backup_restore", ParserVersion: "1.4.0"}
	candidates := handlerCandidatesForDetectedFormat("smsbackuprestore_xml", decoder)

	require.Len(t, candidates, 1, "a signature registers exactly one handler")
	chosen := candidates[0]
	require.Equal(t, proffer.HandlerPathDerive, chosen.ExecutionPath)
	require.Equal(t, activities.DeriveHandlerID, chosen.HandlerID)
	require.Equal(t, activities.DeriveHandlerVersion, chosen.HandlerVersion)
	require.NotEmpty(t, chosen.CompatibilityRef)
	require.NotEmpty(t, chosen.Reason)
	require.NotEqual(t, activities.StructuredELTParserID, chosen.HandlerID,
		"the DuckDB sms_xml_v1 template must not be routed to: it cannot read a real backup")
}

// TestDeriveRoutingLeavesOtherSignaturesUnchanged guards the blast radius:
// only the one signature moved.
func TestDeriveRoutingLeavesOtherSignaturesUnchanged(t *testing.T) {
	decoder := parser.Capability{ParserID: "sbv_csv", ParserVersion: "1.4.0"}
	for detected, wantPath := range map[string]string{
		"csv": "duckdb", "ndjson": "duckdb", "chatgpt_official_json": "duckdb",
		"messages_transcript": "duckdb", "callsbackuprestore_xml": "decoder",
		"pdf": "decoder", "archive": "decoder", "text": "decoder",
	} {
		candidates := handlerCandidatesForDetectedFormat(detected, decoder)
		require.Len(t, candidates, 1, detected)
		require.Equal(t, wantPath, string(candidates[0].ExecutionPath), detected)
	}
}

// TestExecutionPathConstraintMatchesTheRoutedPaths proves the snapshot's
// CHECK admits every execution path the router can emit. The snapshot is the
// database (D-142): this passing does NOT mean the live database has the
// constraint — as of 2026-09-20 it still reads ('decoder','duckdb'), verified
// read-only, and scripts/2026-09-20-derive-execution-path.sql is the
// deliberately unapplied change that closes the gap.
func TestExecutionPathConstraintMatchesTheRoutedPaths(t *testing.T) {
	snapshot, err := os.ReadFile(filepath.Join("..", "..", "..", "sql", "bootstrap", "schema_snapshot_20260907.sql"))
	require.NoError(t, err)
	const constraintName = "CONSTRAINT handler_compatibility_execution_path_check"
	start := strings.Index(string(snapshot), constraintName)
	require.NotEqual(t, -1, start)
	constraint := string(snapshot[start:])
	if end := strings.IndexByte(constraint, '\n'); end >= 0 {
		constraint = constraint[:end]
	}
	for _, path := range []proffer.HandlerExecutionPath{
		proffer.HandlerPathDecoder, proffer.HandlerPathDuckDB, proffer.HandlerPathDerive,
	} {
		require.Contains(t, constraint, "'"+string(path)+"'::text", string(path))
	}
	require.Equal(t, 6, strings.Count(constraint, "'"),
		"execution_path CHECK contains an unexpected or missing value")
}
