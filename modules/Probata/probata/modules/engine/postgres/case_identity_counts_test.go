// Byline: Codex · GPT-6.1-sol · 2026-10-06.
package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Cursedpotential/probata/engine/caseidentity"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const caseCountObservations = `SELECT * FROM (VALUES
 ('2025550101', 'entity-a', 'first_party_message', '2026-01-01'::timestamptz),
 ('2025550101', 'entity-a', 'first_party_message', '2026-01-02'::timestamptz),
 ('(202) 555-0101', 'entity-b', 'first_party_message', '2026-01-03'::timestamptz),
 (NULL, 'entity-a', 'first_party_message', NULL::timestamptz),
 ('ALPHA', 'entity-a', 'mention', '2026-01-04'::timestamptz),
 (' alpha ', 'entity-b', 'mention', '2026-01-02'::timestamptz),
 ('2025550101', NULL, 'call', '2026-01-01'::timestamptz),
 ('2025550101', NULL, 'call', '2026-01-01'::timestamptz),
 ('202.555.0101', NULL, 'call', NULL::timestamptz),
 ('noise', 'entity-out', 'first_party_message', '2026-01-01'::timestamptz),
 ('entity-a', 'entity-a', 'mention', '2026-01-01'::timestamptz)
) AS fixture(raw, entity, src, t)`

const caseUnknownObservations = `SELECT * FROM (VALUES
 ('ALPHA', '2026-01-01'::timestamptz), ('ALPHA', '2026-01-02'::timestamptz),
 (' alpha ', '2026-01-03'::timestamptz),
 ('2025550101', '2026-01-01'::timestamptz), ('(202) 555-0101', '2026-01-02'::timestamptz),
 ('2025550101', NULL::timestamptz), ('known', '2026-01-01'::timestamptz),
 ('dismissed', '2026-01-01'::timestamptz), ('retired', '2026-01-02'::timestamptz),
 ('null-date', NULL::timestamptz), (NULL, NULL::timestamptz),
 ('', '2026-01-01'::timestamptz), ('   ', '2026-01-01'::timestamptz)
) AS fixture(raw, t)`

// caseFixtureQueryer executes the production aggregations over synthetic VALUES instead of corpus tables.
// Inputs: read-only connection and substituted queries. Outputs: actual PostgreSQL rows; effects: SELECTs only.
// Choose for aggregation semantics tests without creating a schema, tables, indexes or evidence records.
type caseFixtureQueryer struct {
	*pgx.Conn
	counts, unknowns string
	queries          int
}

// Query substitutes only the two count reads and rejects any unexpected query.
// Inputs: production SQL and arguments. Outputs: synthetic-result rows; effects: one read-only SELECT.
// Choose through readCounts/readUnknowns to exercise their actual bigint/date scanners as well as SQL.
func (q *caseFixtureQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.queries++
	switch sql {
	case caseCountsSQL:
		return q.Conn.Query(ctx, q.counts, args...)
	case caseUnknownsSQL:
		return q.Conn.Query(ctx, q.unknowns, args...)
	default:
		return nil, errors.New("unexpected non-fixture query")
	}
}

// caseFixtureAggregation replaces unchanged observation readers while retaining the production aggregation/filter SQL.
// Inputs: production query and synthetic raw observations. Outputs: SELECT-only fixture SQL; effects: none.
// Choose to isolate the changed aggregation, with no dependency on real participant or identity records.
func caseFixtureAggregation(t *testing.T, sql, observations string) string {
	t.Helper()
	boundary := strings.Index(sql, "), grouped AS MATERIALIZED (")
	require.GreaterOrEqual(t, boundary, 0)
	return "WITH observed AS (" + observations + sql[boundary:]
}

// caseFixtureConnection opens an explicitly configured, bounded read-only PostgreSQL session for synthetic tests.
// Input: CASE_IDENTITY_READ_TEST_DSN. Output: connection/context; effects: connection and SELECTs, never database writes.
// Choose for opt-in PostgreSQL semantics proof; ordinary browser-free unit runs skip when the DSN is absent.
func caseFixtureConnection(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
	dsn := os.Getenv("CASE_IDENTITY_READ_TEST_DSN")
	if dsn == "" {
		t.Skip("CASE_IDENTITY_READ_TEST_DSN is not configured; synthetic SELECT-only PostgreSQL proof is opt-in")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("read-only fixture connection configuration is invalid")
	}
	config.ConnectTimeout = 2 * time.Second
	config.RuntimeParams["default_transaction_read_only"] = "on"
	config.RuntimeParams["statement_timeout"] = "8000"
	config.RuntimeParams["timezone"] = "UTC"
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	t.Cleanup(cancel)
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal("read-only fixture connection is unavailable")
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = conn.Close(cleanup)
	})
	var readOnly string
	require.NoError(t, conn.QueryRow(ctx, "SHOW transaction_read_only").Scan(&readOnly))
	require.Equal(t, "on", readOnly)
	return conn, ctx
}

// caseFixtureDate converts a fixed synthetic UTC day to the nullable timestamps scanned by pgx.
// Input: day or empty string. Output: timestamp in pgx's default local scan location or nil; effects: none.
// Choose for exact instant expectations across desktop/VPS timezones, including missing dates.
func caseFixtureDate(day string) *time.Time {
	if day == "" {
		return nil
	}
	value, err := time.Parse("2006-01-02", day)
	if err != nil {
		panic("invalid synthetic fixture date")
	}
	value = value.Local()
	return &value
}

// TestCaseCountsSQLPreservesWeightedEventsEntitiesAndDates proves the count aggregation on synthetic observations.
// Inputs: opt-in read-only PostgreSQL connection. Outputs: assertions; effects: SELECTs only.
// Choose to detect lost repeated events, endpoint multiplicity, entity origins or null-date handling.
func TestCaseCountsSQLPreservesWeightedEventsEntitiesAndDates(t *testing.T) {
	conn, ctx := caseFixtureConnection(t)
	q := &caseFixtureQueryer{Conn: conn, counts: caseFixtureAggregation(t, caseCountsSQL, caseCountObservations)}
	counts, err := readCounts(ctx, q, []string{"2025550101", "alpha", "entity-a", "2025550101"}, []string{"entity-a", "entity-b"})
	require.NoError(t, err)
	require.ElementsMatch(t, []caseidentity.Count{
		{Key: "2025550101", Source: "first_party_message", Events: 3, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-03")},
		{Key: "2025550101", Source: "call", Events: 3, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-01")},
		{Key: "alpha", Source: "mention", Events: 2, FirstAt: caseFixtureDate("2026-01-02"), LastAt: caseFixtureDate("2026-01-04")},
		{Key: "entity-a", Source: "mention", Events: 1, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-01")},
		{Key: "entity-a", Source: "first_party_message", Events: 3, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-02")},
		{Key: "entity-a", Source: "mention", Events: 2, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-04")},
		{Key: "entity-b", Source: "first_party_message", Events: 1, FirstAt: caseFixtureDate("2026-01-03"), LastAt: caseFixtureDate("2026-01-03")},
		{Key: "entity-b", Source: "mention", Events: 1, FirstAt: caseFixtureDate("2026-01-02"), LastAt: caseFixtureDate("2026-01-02")},
	}, counts)
	before := q.queries
	empty, err := readCounts(ctx, q, nil, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	require.Equal(t, before, q.queries, "no keys must avoid a corpus scan")
}

// TestCaseUnknownsSQLPreservesExclusionsRawSpellingRankingAndLimit proves the unknown queue on synthetic observations.
// Inputs: opt-in read-only PostgreSQL connection. Outputs: assertions; effects: SELECTs only.
// Choose to detect false zeroes, altered alias/dismissal filtering, raw representatives, ordering or limits.
func TestCaseUnknownsSQLPreservesExclusionsRawSpellingRankingAndLimit(t *testing.T) {
	conn, ctx := caseFixtureConnection(t)
	sql := caseFixtureAggregation(t, caseUnknownsSQL, caseUnknownObservations)
	sql = strings.Replace(sql, "WITH observed AS (", `WITH fixture_aliases(normalized,status) AS (
	 VALUES ('alpha','retired'), ('known','confirmed'), ('retired','retired')
	), fixture_dismissed(normalized) AS (VALUES ('dismissed')), observed AS (`, 1)
	sql = strings.ReplaceAll(sql, "registry.entity_alias_current", "fixture_aliases")
	sql = strings.ReplaceAll(sql, "registry.vw_identifier_dismissed", "fixture_dismissed")
	q := &caseFixtureQueryer{Conn: conn, unknowns: sql}
	unknowns, err := readUnknowns(ctx, q)
	require.NoError(t, err)
	require.Equal(t, []caseidentity.Unknown{
		{Normalized: "2025550101", RawValue: "(202) 555-0101", Events: 3, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-02")},
		{Normalized: "alpha", RawValue: " alpha ", Events: 3, FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-03")},
		{Normalized: "null-date", RawValue: "null-date", Events: 1},
		{Normalized: "retired", RawValue: "retired", Events: 1, FirstAt: caseFixtureDate("2026-01-02"), LastAt: caseFixtureDate("2026-01-02")},
	}, unknowns)
	rows, err := q.Query(ctx, caseUnknownsSQL, 1)
	require.NoError(t, err)
	limited, err := pgx.CollectRows(rows, pgx.RowToStructByPos[caseidentity.Unknown])
	require.NoError(t, err)
	require.Equal(t, unknowns[:1], limited)
}

// Synthetic source tables exercise every unchanged observation branch without accessing corpus rows.
// The duplicate-current row is defensive parity input; live uq_res_current prohibits that state.
const caseCountsSourceFixtures = `WITH
 fixture_message(id,ts_utc) AS (VALUES
  ('m1','2026-01-01'::timestamptz), ('m2','2026-01-02'::timestamptz),
  ('m3','2026-01-03'::timestamptz), ('m-null',NULL::timestamptz)
 ), fixture_message_participant(message_id,participant_e164,participant_raw,entity_id) AS (VALUES
  ('m1',NULL::text,'2025550101','entity-a'), ('m2','+12025550101','ignored','entity-a'),
  ('m3','','(202) 555-0101',NULL::text), ('m-null',NULL::text,NULL::text,'entity-a'),
  ('m-null',NULL::text,NULL::text,NULL::text)
 ), fixture_third_party_message(id,occurred_at) AS (VALUES
  ('third-1','2026-01-04'::timestamptz), ('third-2',NULL::timestamptz)
 ), fixture_third_party_message_participant(message_id,participant_e164,participant_raw,entity_id) AS (VALUES
  ('third-1',NULL::text,'ALPHA','entity-b'), ('third-2','',' alpha ','entity-b'),
  ('third-1',NULL::text,'ALPHA','entity-b')
 ), fixture_call_log(from_e164,from_raw,from_entity_id,to_e164,to_raw,to_entity_id,started_at) AS (VALUES
  ('+12025550101','ignored','entity-a','','(202) 555-0101',NULL::text,'2026-01-05'::timestamptz),
  (NULL::text,NULL::text,'entity-a',NULL::text,NULL::text,NULL::text,NULL::timestamptz)
 ), fixture_entity_mention(id,surface_text,created_at) AS (VALUES
  ('mention-1','ALPHA','2026-01-06'::timestamptz),
  ('mention-2',NULL::text,'2026-01-03'::timestamptz), ('mention-3',' alpha ','2026-01-02'::timestamptz)
 ), fixture_resolutions(mention_id,canonical_entity_id,sys_period,duplicate_current) AS (VALUES
  ('mention-1','entity-a',tstzrange('2026-01-01',NULL,'[)'),false),
  ('mention-1','entity-b',tstzrange('2026-01-01',NULL,'[)'),true),
  ('mention-1','entity-out',tstzrange('2025-01-01','2025-12-01','[)'),false),
  ('mention-1','entity-out',tstzrange('2025-12-01','2026-01-01','[)'),false),
  ('mention-2','entity-a',tstzrange('2026-01-01',NULL,'[)'),false),
  ('mention-3','entity-out',tstzrange('2025-01-01','2025-12-01','[)'),false)
 ), fixture_entity_resolution AS (
  SELECT mention_id,canonical_entity_id,sys_period FROM fixture_resolutions
  WHERE NOT duplicate_current OR $3::bool
 ), `

// caseFixtureSources replaces working tables with VALUES while retaining every production observation join.
// Inputs: complete count SELECT. Output: synthetic read-only SELECT; effects: none.
// Choose for full-branch parity, including aliases, endpoints and historical/current resolution multiplicity.
func caseFixtureSources(sql string) string {
	replacer := strings.NewReplacer(
		"working.message_participant", "fixture_message_participant",
		"working.third_party_message_participant", "fixture_third_party_message_participant",
		"working.third_party_message", "fixture_third_party_message",
		"working.message", "fixture_message",
		"working.call_log", "fixture_call_log",
		"working.entity_mention", "fixture_entity_mention",
		"working.entity_resolution", "fixture_entity_resolution",
	)
	return caseCountsSourceFixtures + strings.TrimPrefix(strings.TrimSpace(replacer.Replace(sql)), "WITH ")
}

// TestCaseCountsSQLFullObservationParity compares weighted aggregation with the original per-observation count semantics.
// Inputs: synthetic source tables in a read-only PostgreSQL session. Outputs: parity and exact-count assertions.
// Effects: SELECTs only. Choose to catch lost origins, duplicate endpoints, null keys or resolution-join changes.
func TestCaseCountsSQLFullObservationParity(t *testing.T) {
	conn, ctx := caseFixtureConnection(t)
	boundary := strings.Index(caseCountsSQL, "), grouped AS MATERIALIZED (")
	require.GreaterOrEqual(t, boundary, 0)
	reference := caseCountsSQL[:boundary] + `), keyed AS MATERIALIZED (
 SELECT registry.norm_identifier(raw) AS k, entity, src, t FROM observed
 ) SELECT k, src, count(*), min(t), max(t) FROM keyed WHERE k = ANY($1::text[]) GROUP BY k, src
 UNION ALL
 SELECT entity, src, count(*), min(t), max(t) FROM keyed WHERE entity = ANY($2::text[]) GROUP BY entity, src`
	for _, duplicateCurrent := range []bool{false, true} {
		args := []any{[]string{"2025550101", "alpha"}, []string{"entity-a", "entity-b"}, duplicateCurrent}
		beforeRows, err := conn.Query(ctx, caseFixtureSources(reference), args...)
		require.NoError(t, err)
		before, err := pgx.CollectRows(beforeRows, pgx.RowToStructByPos[caseidentity.Count])
		require.NoError(t, err)
		afterRows, err := conn.Query(ctx, caseFixtureSources(caseCountsSQL), args...)
		require.NoError(t, err)
		after, err := pgx.CollectRows(afterRows, pgx.RowToStructByPos[caseidentity.Count])
		require.NoError(t, err)
		require.ElementsMatch(t, before, after, "duplicate-current fixture: %v", duplicateCurrent)
		require.Contains(t, after, caseidentity.Count{Key: "2025550101", Source: "first_party_message", Events: 3,
			FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-03")})
		require.Contains(t, after, caseidentity.Count{Key: "2025550101", Source: "call", Events: 2,
			FirstAt: caseFixtureDate("2026-01-05"), LastAt: caseFixtureDate("2026-01-05")})
		require.Contains(t, after, caseidentity.Count{Key: "entity-a", Source: "first_party_message", Events: 3,
			FirstAt: caseFixtureDate("2026-01-01"), LastAt: caseFixtureDate("2026-01-02")})
		mentions := int64(2)
		if duplicateCurrent {
			mentions++
		}
		require.Contains(t, after, caseidentity.Count{Key: "alpha", Source: "mention", Events: mentions,
			FirstAt: caseFixtureDate("2026-01-02"), LastAt: caseFixtureDate("2026-01-06")})
		for _, count := range after {
			require.NotEqual(t, "entity-out", count.Key, "bounded historical resolutions are not current origins")
		}
	}
}
