// Byline: Claude Code · Sonnet · 2026-10-02
//
// The two SQL boundaries of ContactsImportWorkflow: the Case Bible catalog (read-only) and the registry
// sweeps that find numbers no person carries and fill NULL entity columns. Registry rows are never written
// here: people, aliases and placeholders go through CaseIdentityStore, and these queries only read, or
// fill NULL entity columns of imported rows (platform_runtime holds column-level UPDATE on exactly those,
// sql/bootstrap/placeholders_20261002.sql).
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Cursedpotential/probata/engine/contacts"
)

// ContactsCatalog reads the contact exports out of raw_duck.b2_objects.
type ContactsCatalog struct{ db DB }

// NewContactsCatalog requires a database (the Case Bible catalog, not the platform database).
func NewContactsCatalog(db DB) (*ContactsCatalog, error) {
	if db == nil {
		return nil, errors.New("contacts catalog requires a database")
	}
	return &ContactsCatalog{db: db}, nil
}

// The catalog's current listing of every bucket (raw_duck.bucket_objects_current; raw_duck.b2_objects is
// stale). Its column names are read from information_schema at run time and matched against these
// candidates, so the query never guesses a column; if a needed one is missing the step fails and says
// which columns the table has.
const contactsCatalogTable = "bucket_objects_current"

var contactColumnCandidates = map[string][]string{
	"bucket": {"bucket", "bucket_name"},
	"key":    {"key", "object_key", "object_name", "path", "name"},
	"size":   {"size", "size_bytes", "bytes", "content_length"},
	"sha1":   {"sha1", "content_sha1", "sha1_hex"},
	"sha256": {"sha256", "content_sha256", "sha256_hex"},
	"listed": {"listed_at", "snapshot_at", "last_listed_at", "last_modified"},
}

func pickColumn(have map[string]bool, role string) string {
	for _, candidate := range contactColumnCandidates[role] {
		if have[candidate] {
			return candidate
		}
	}
	return ""
}

// ContactFiles implements activities.ContactsCatalog: vCards, contact CSV/JSON, and Facebook/Instagram
// imported/synced contacts across every bucket the current listing holds, one object per content hash
// (the most recently listed).
func (c *ContactsCatalog) ContactFiles(ctx context.Context) ([]contacts.File, error) {
	columnRows, err := c.db.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema = 'raw_duck' AND table_name = $1`, contactsCatalogTable)
	if err != nil {
		return nil, errors.New("contacts catalog: column lookup unavailable")
	}
	have := map[string]bool{}
	var names []string
	for columnRows.Next() {
		var name string
		if err := columnRows.Scan(&name); err != nil {
			columnRows.Close()
			return nil, err
		}
		have[name] = true
		names = append(names, name)
	}
	columnRows.Close()
	if err := columnRows.Err(); err != nil {
		return nil, err
	}
	key, size, listed := pickColumn(have, "key"), pickColumn(have, "size"), pickColumn(have, "listed")
	bucket, sha1, sha256 := pickColumn(have, "bucket"), pickColumn(have, "sha1"), pickColumn(have, "sha256")
	if key == "" || (sha1 == "" && sha256 == "") {
		return nil, fmt.Errorf("contacts catalog: raw_duck.%s has columns %v; a key column and a sha1 or sha256 column are required", contactsCatalogTable, names)
	}
	// Every identifier below comes from information_schema and matched an allow-listed candidate name.
	quote := func(column string) string { return `"` + column + `"` }
	or := func(column, fallback string) string {
		if column == "" {
			return fallback
		}
		return quote(column)
	}
	hash := or(sha1, "NULL")
	hashKind := "sha1"
	if sha1 == "" {
		hash, hashKind = quote(sha256), "sha256"
	}
	listedExpr := "''"
	order := hash
	if listed != "" {
		listedExpr = `to_char(` + quote(listed) + ` AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
		order = hash + ", " + quote(listed) + " DESC"
	}
	statement := `SELECT DISTINCT ON (` + hash + `) ` + or(bucket, "''") + `::text, ` + quote(key) + `::text, ` + or(size, "0") + `::bigint, lower(` + hash + `::text), ` + listedExpr + `
FROM raw_duck.` + contactsCatalogTable + `
WHERE ` + hash + ` IS NOT NULL AND ` + hash + `::text <> '' AND (
      ` + quote(key) + ` ~* '\.vcf$'
   OR ` + quote(key) + ` ~* '(^|/)[^/]*contact[^/]*\.(csv|json)$'
   OR ` + quote(key) + ` ~* '(facebook|instagram|meta)[^[:space:]]*/[^[:space:]]*(imported_contacts|synced_contacts)[^/]*\.json$'
   OR ` + quote(key) + ` ~* '/(imported_contacts|synced_contacts)[^/]*\.json$'
)
ORDER BY ` + order
	rows, err := c.db.Query(ctx, statement)
	if err != nil {
		return nil, errors.New("contacts catalog: query unavailable")
	}
	defer rows.Close()
	var files []contacts.File
	for rows.Next() {
		var file contacts.File
		var sum string
		if err := rows.Scan(&file.Bucket, &file.Key, &file.Size, &sum, &file.ListedAt); err != nil {
			return nil, errors.New("contacts catalog: unreadable row")
		}
		if hashKind == "sha1" {
			file.SHA1 = sum
		} else {
			file.SHA256 = sum
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

// ContactsRegistry is the platform-database side of the placeholder and re-link steps.
type ContactsRegistry struct{ db DB }

// NewContactsRegistry requires the platform database.
func NewContactsRegistry(db DB) (*ContactsRegistry, error) {
	if db == nil {
		return nil, errors.New("contacts registry requires a database")
	}
	return &ContactsRegistry{db: db}, nil
}

// UnlinkedNumbers lists phone numbers on imported rows with a NULL entity column that NO non-retired identifier
// carries, most frequent first.
func (r *ContactsRegistry) UnlinkedNumbers(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
SELECT number FROM (
  SELECT registry.norm_identifier(coalesce(nullif(from_e164, ''), from_raw, '')) AS number FROM working.call_log WHERE from_entity_id IS NULL
  UNION ALL
  SELECT registry.norm_identifier(coalesce(nullif(to_e164, ''), to_raw, '')) FROM working.call_log WHERE to_entity_id IS NULL
  UNION ALL
  SELECT registry.norm_identifier(coalesce(nullif(participant_e164, ''), participant_raw, '')) FROM working.message_participant WHERE entity_id IS NULL
  UNION ALL
  SELECT registry.norm_identifier(coalesce(nullif(participant_e164, ''), participant_raw, '')) FROM working.third_party_message_participant WHERE entity_id IS NULL
) u
WHERE number ~ '^[2-9][0-9]{9}$'
  AND NOT EXISTS (SELECT 1 FROM registry.entity_alias a WHERE a.status <> 'retired' AND registry.norm_identifier(a.alias_text::text) = u.number)
GROUP BY number ORDER BY count(*) DESC, number`)
	if err != nil {
		return nil, errors.New("contacts registry: unlinked-number query unavailable")
	}
	defer rows.Close()
	var numbers []string
	for rows.Next() {
		var number string
		if err := rows.Scan(&number); err != nil {
			return nil, err
		}
		numbers = append(numbers, number)
	}
	return numbers, rows.Err()
}

// CarriedEntities maps each number to the (unmerged) person that carries it, oldest identifier first.
func (r *ContactsRegistry) CarriedEntities(ctx context.Context, numbers []string) (map[string]string, error) {
	out := map[string]string{}
	if len(numbers) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
SELECT DISTINCT ON (n) n, entity_id FROM (
  SELECT registry.norm_identifier(a.alias_text::text) AS n, a.entity_id::text AS entity_id, a.created_at
  FROM registry.entity_alias a JOIN registry.entity e ON e.id = a.entity_id
  WHERE a.status <> 'retired' AND e.merged_into_id IS NULL AND registry.norm_identifier(a.alias_text::text) = ANY($1::text[])
) x ORDER BY n, created_at`, numbers)
	if err != nil {
		return nil, errors.New("contacts registry: lookup unavailable")
	}
	defer rows.Close()
	for rows.Next() {
		var number, entity string
		if err := rows.Scan(&number, &entity); err != nil {
			return nil, err
		}
		out[number] = entity
	}
	return out, rows.Err()
}

const relinkMapSQL = `(SELECT DISTINCT ON (n) n, entity_id FROM (
  SELECT registry.norm_identifier(a.alias_text::text) AS n, a.entity_id, a.created_at
  FROM registry.entity_alias a JOIN registry.entity e ON e.id = a.entity_id
  WHERE a.alias_kind = 'phone' AND a.status <> 'retired' AND e.merged_into_id IS NULL
) x WHERE n ~ '^[2-9][0-9]{9}$' ORDER BY n, created_at) m`

// RelinkSweep fills every NULL entity column whose number a person now carries. dryRun rolls back.
func (r *ContactsRegistry) RelinkSweep(ctx context.Context, dryRun bool) (map[string]int64, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, errors.New("contacts registry: transaction unavailable")
	}
	finished := false
	defer func() {
		if !finished {
			cleanup, cancel := boundedCleanup(ctx)
			defer cancel()
			_ = tx.Rollback(cleanup)
		}
	}()
	counts := map[string]int64{}
	for _, step := range []struct{ name, sql string }{
		{"call_from", `UPDATE working.call_log c SET from_entity_id = m.entity_id FROM ` + relinkMapSQL +
			` WHERE c.from_entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(c.from_e164, ''), c.from_raw, '')) = m.n`},
		{"call_to", `UPDATE working.call_log c SET to_entity_id = m.entity_id FROM ` + relinkMapSQL +
			` WHERE c.to_entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(c.to_e164, ''), c.to_raw, '')) = m.n`},
		{"message_participants", `UPDATE working.message_participant x SET entity_id = m.entity_id FROM ` + relinkMapSQL +
			` WHERE x.entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(x.participant_e164, ''), x.participant_raw, '')) = m.n`},
		{"third_party_participants", `UPDATE working.third_party_message_participant x SET entity_id = m.entity_id FROM ` + relinkMapSQL +
			` WHERE x.entity_id IS NULL AND registry.norm_identifier(coalesce(nullif(x.participant_e164, ''), x.participant_raw, '')) = m.n`},
	} {
		tag, err := tx.Exec(ctx, step.sql)
		if err != nil {
			return nil, fmt.Errorf("contacts registry: re-link %s failed", step.name)
		}
		counts["linked_"+step.name] = tag.RowsAffected()
	}
	// What is still empty after the sweep: the numbers no person carries (short codes, unusable values).
	var remaining int64
	if err := tx.QueryRow(ctx, `SELECT
	  (SELECT count(*) FROM working.call_log WHERE from_entity_id IS NULL) + (SELECT count(*) FROM working.call_log WHERE to_entity_id IS NULL)
	  + (SELECT count(*) FROM working.message_participant WHERE entity_id IS NULL)
	  + (SELECT count(*) FROM working.third_party_message_participant WHERE entity_id IS NULL)`).Scan(&remaining); err != nil {
		return nil, errors.New("contacts registry: remaining-row count unavailable")
	}
	counts["still_without_an_entity"] = remaining
	finished = true
	if dryRun {
		cleanup, cancel := boundedCleanup(ctx)
		defer cancel()
		_ = tx.Rollback(cleanup)
		return counts, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("contacts registry: commit failed")
	}
	return counts, nil
}
