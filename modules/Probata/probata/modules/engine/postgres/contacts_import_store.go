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
	"encoding/json"
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

// catalogSQL builds the one query both file kinds use, over the catalog's current listing of every provider
// and bucket (raw_duck.bucket_objects_current: provider, bucket, key, size, sha1, md5, modtime, listed_at).
// predicate selects the objects. Objects are grouped into one entry per distinct content:
//   - the content id is the SHA-1 when the row has one (Backblaze rows do; Cloudflare R2 rows carry none), else
//     the SHA-1 of a row with the same name and size, else the name and size themselves, so the same export
//     listed in several buckets and in both providers becomes ONE file;
//   - the first location is Backblaze, then casebible-raw, then casebible-sorted, then anything else; the rest
//     are its alternates, tried when the first object is gone (the catalog lists objects that have since moved);
//   - quarantined zero-filled copies are never offered, because their bytes are not the content.
//
// The recency the grouping reports is the newest object modification time, not the listing time (every row
// is listed in the same snapshot, so the listing time cannot tell one export from another).
func catalogSQL(predicate string) string {
	return `
WITH c AS (
  SELECT provider, bucket, key, size, nullif(lower(sha1), '') AS sha1, coalesce(modtime, listed_at) AS stamp,
         lower(regexp_replace(key, '^.*/', '')) AS base
  FROM raw_duck.bucket_objects_current
  WHERE (` + predicate + `)
    AND key NOT ILIKE '%/_quarantine/%' AND key NOT ILIKE '%zero-filled%'
), h AS (
  SELECT c.*, coalesce(c.sha1, (SELECT x.sha1 FROM c x WHERE x.sha1 IS NOT NULL AND x.size = c.size AND x.base = c.base LIMIT 1)) AS known_sha1
  FROM c
), g AS (
  SELECT h.*, coalesce(known_sha1, 'size:' || size::text || ':' || base) AS cid,
         CASE WHEN provider = 'b2' THEN 0 WHEN bucket = 'casebible-raw' THEN 1 WHEN bucket = 'casebible-sorted' THEN 2 ELSE 3 END AS pref
  FROM h
)
SELECT coalesce(max(known_sha1), ''), max(size)::bigint, coalesce(to_char(max(stamp) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
       (jsonb_agg(jsonb_build_object('provider', provider, 'bucket', bucket, 'key', key) ORDER BY pref, stamp DESC NULLS LAST, key))::text
FROM g GROUP BY cid ORDER BY max(stamp) DESC NULLS LAST, cid`
}

const (
	// looseContactPredicate selects vCards, contact CSV/JSON, and Facebook/Instagram imported/synced contact
	// lists. Call-recorder notes ("Unknown contact"), sync settings, block lists and chat exports are not contact
	// lists and are left out.
	looseContactPredicate = `(key ~* '\.vcf$'
      OR key ~* '(^|/)[^/]*contacts?[^/]*\.(csv|json)$'
      OR key ~* '(facebook|instagram|meta)[^[:space:]]*/[^[:space:]]*(imported_contacts|synced_contacts)[^/]*\.json$'
      OR key ~* '/(imported_contacts|synced_contacts)[^/]*\.json$')
    AND key !~* '(unknown contact|cube acr|contacts_sync_settings|blocked|/chats/)'`
	// zipContactPredicate selects archives whose name suggests a Google Takeout, a Facebook/Instagram download or a
	// phone export, which may hold contact files inside.
	zipContactPredicate = `key ~* '\.zip$' AND key ~* '(takeout|contact|google|facebook|instagram|meta|phone|katrina|sms|export|backup|vcard)'`
)

// scanCatalogFiles runs catalogSQL and turns each row into a contacts.File with its fallback locations.
func (c *ContactsCatalog) scanCatalogFiles(ctx context.Context, predicate string) ([]contacts.File, error) {
	rows, err := c.db.Query(ctx, catalogSQL(predicate))
	if err != nil {
		return nil, errors.New("contacts catalog: query unavailable")
	}
	defer rows.Close()
	var files []contacts.File
	for rows.Next() {
		var sum, stamp, sources string
		var file contacts.File
		if err := rows.Scan(&sum, &file.Size, &stamp, &sources); err != nil {
			return nil, errors.New("contacts catalog: unreadable row")
		}
		var places []contacts.Source
		if err := json.Unmarshal([]byte(sources), &places); err != nil || len(places) == 0 {
			return nil, errors.New("contacts catalog: unreadable locations")
		}
		file.Provider, file.Bucket, file.Key = places[0].Provider, places[0].Bucket, places[0].Key
		file.SHA1, file.ListedAt = sum, stamp
		if len(places) > 1 {
			file.Alternates = places[1:min(len(places), 7)]
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

// ContactFiles implements activities.ContactsCatalog: the loose contact files across every provider and
// bucket the current listing holds, one per distinct content, with fallback locations.
func (c *ContactsCatalog) ContactFiles(ctx context.Context) ([]contacts.File, error) {
	return c.scanCatalogFiles(ctx, looseContactPredicate)
}

// ZipFiles implements activities.ContactsCatalog: the archives that may hold contact files inside.
func (c *ContactsCatalog) ZipFiles(ctx context.Context) ([]contacts.File, error) {
	return c.scanCatalogFiles(ctx, zipContactPredicate)
}

// OwnerEntity returns the perspective person (the case's one person with role "user"), or "" if there is none.
// The owner's own numbers and emails appear on many contact cards ("Me", shared family lines) and must never
// chain unrelated contacts into one cluster.
func (r *ContactsRegistry) OwnerEntity(ctx context.Context) (string, error) {
	rows, err := r.db.Query(ctx, `SELECT p.id::text FROM registry.person p JOIN registry.entity e ON e.id = p.id
		WHERE p.role_in_case = 'user' AND e.merged_into_id IS NULL`)
	if err != nil {
		return "", errors.New("contacts registry: owner lookup unavailable")
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) > 1 {
		return "", errors.New("contacts registry: more than one person has the role user")
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
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
