#!/usr/bin/env bash
# Seed Probata registry identifiers from the Case Bible catalog's raw_duck.msg_identity_20260924.
#
# Byline: Claude Code · Opus 5.5 · 2026-10-01
# Owner order 2026-10-01 07:57: registry is the ONE identity store. This moves the owner-confirmed
# person <-> identifier rows (2026-09-24, Matt 32 / Katrina 7) into registry.entity_alias, every raw
# spelling kept, with its status, period, basis and original date. After it, the catalog reads registry
# (identity_registry_fdw_20261001.sql) and the old table is renamed *_retired_20261001, never deleted.
#
# Runs ON ovh-files (the two PostgreSQL containers are there):
#   bash 2026-10-01-seed-registry-identity.sh            # dry run: everything inside a transaction, then ROLLBACK
#   APPLY=1 bash 2026-10-01-seed-registry-identity.sh    # same, then COMMIT
# Idempotent: each row's idempotency_key is derived from (person, identifier, raw_value); a re-run inserts nothing.
# Refuses to run unless both persons exist in registry (they are minted by the go-live identity step).
set -euo pipefail

CATALOG_CONTAINER="${CATALOG_CONTAINER:-fgz1n7useplhk0t91uk7k1aw}"
PLATFORM_CONTAINER="${PLATFORM_CONTAINER:-$(docker ps --format '{{.Names}}' | grep '^probata-db-' | head -1)}"
MATT_ID="${MATT_ID:-01a0f751-e07b-76b6-afcb-63acfbba373e}"
KATRINA_ID="${KATRINA_ID:-01a0f751-e07b-76c7-8c0f-65692ad656b8}"
END="ROLLBACK"
[ "${APPLY:-0}" = "1" ] && END="COMMIT"

# The source: the catalog table, or its retired copy once the catalog reads registry (a re-run then inserts nothing).
SOURCE_TABLE="${SOURCE_TABLE:-$(docker exec "$CATALOG_CONTAINER" psql -U postgres -d casebible -Atc \
  "SELECT CASE WHEN to_regclass('raw_duck.msg_identity_20260924_retired_20261001') IS NOT NULL
               THEN 'raw_duck.msg_identity_20260924_retired_20261001' ELSE 'raw_duck.msg_identity_20260924' END")}"
echo "source: ${SOURCE_TABLE}"

{
cat <<SQL
\set ON_ERROR_STOP 1
BEGIN;
CREATE TEMP TABLE seed_identity (person text, identifier text, raw_value text, kind text, status text, period text, basis text, added_at timestamptz) ON COMMIT DROP;
\copy seed_identity FROM STDIN WITH (FORMAT csv)
SQL
docker exec "$CATALOG_CONTAINER" psql -U postgres -d casebible -v ON_ERROR_STOP=1 -Atc \
  "\copy (SELECT person, identifier, raw_value, kind, status, period, basis, added_at FROM ${SOURCE_TABLE} ORDER BY person, kind, identifier, raw_value) TO STDOUT WITH (FORMAT csv)"
cat <<SQL
\.
DO \$\$
BEGIN
  IF (SELECT count(*) FROM registry.person WHERE id IN ('${MATT_ID}', '${KATRINA_ID}')) <> 2 THEN
    RAISE EXCEPTION 'both persons must exist in registry before the seed (go-live identity step)';
  END IF;
  IF EXISTS (SELECT 1 FROM seed_identity WHERE person NOT IN ('Matt', 'Katrina')) THEN
    RAISE EXCEPTION 'the catalog holds identities for a person this seed does not map';
  END IF;
END \$\$;

-- The catalog label each person is filtered by (raw_duck.msg_identity_20260924.person), logged as a version.
WITH labels(id, short_name) AS (VALUES ('${MATT_ID}'::uuid, 'Matt'), ('${KATRINA_ID}'::uuid, 'Katrina')),
before AS (
  SELECT p.id, l.short_name, to_jsonb(p) AS state FROM registry.person p JOIN labels l ON l.id = p.id
  WHERE p.short_name IS DISTINCT FROM l.short_name
), updated AS (
  UPDATE registry.person p SET short_name = b.short_name FROM before b WHERE p.id = b.id RETURNING p.id, to_jsonb(p) AS state
)
INSERT INTO registry.identity_change (id, subject_table, subject_id, before_state, after_state, change_reason, recorded_by, recorded_by_uid, idempotency_key)
SELECT uuidv7(), 'registry.person', u.id, jsonb_build_object('short_name', b.state->'short_name'), jsonb_build_object('short_name', u.state->'short_name'),
       'seed: the label raw_duck.msg_identity_20260924 used for this person', 'seed:2026-10-01-case-identity', 'seed', 'seed:short_name:' || u.id
FROM updated u JOIN before b ON b.id = u.id
ON CONFLICT (idempotency_key) DO NOTHING;

INSERT INTO registry.entity_alias (id, entity_id, alias_text, alias_kind, status, period, basis, change_reason, recorded_by,
                                   idempotency_key, created_at, provenance)
SELECT uuidv7(),
       CASE s.person WHEN 'Matt' THEN '${MATT_ID}'::uuid ELSE '${KATRINA_ID}'::uuid END,
       s.raw_value,
       CASE WHEN s.kind IN ('name', 'phone', 'email', 'account', 'other') THEN s.kind ELSE 'other' END,
       s.status, NULLIF(s.period, ''), s.basis,
       'seeded from raw_duck.msg_identity_20260924 (Case Bible catalog), owner-confirmed 2026-09-23/24',
       'seed:raw_duck.msg_identity_20260924',
       'seed:msg_identity_20260924:' || md5(s.person || chr(31) || s.identifier || chr(31) || s.raw_value),
       s.added_at,
       ARRAY[ROW('postgres', s.person || '|' || s.identifier || '|' || s.raw_value, 'casebible.raw_duck.msg_identity_20260924')::ai.source_ref]
FROM seed_identity s
WHERE NOT EXISTS (SELECT 1 FROM registry.entity_alias a
                  WHERE a.entity_id = CASE s.person WHEN 'Matt' THEN '${MATT_ID}'::uuid ELSE '${KATRINA_ID}'::uuid END
                    AND lower(a.alias_text::text) = lower(s.raw_value))
ON CONFLICT (idempotency_key) DO NOTHING;

-- Read-back: every catalog row has exactly one registry row with the same person, raw spelling and key.
SELECT s.person, count(*) AS catalog_rows, count(a.id) AS registry_rows,
       count(*) FILTER (WHERE a.normalized = s.identifier) AS same_key,
       count(*) FILTER (WHERE a.status = s.status AND coalesce(a.period, '') = coalesce(s.period, '') AND a.basis = s.basis) AS same_facts
FROM seed_identity s
LEFT JOIN registry.entity_alias_current a
  ON a.entity_id = CASE s.person WHEN 'Matt' THEN '${MATT_ID}'::uuid ELSE '${KATRINA_ID}'::uuid END
 AND a.alias_text::text = s.raw_value
GROUP BY s.person ORDER BY s.person;
SELECT person, kind, status, count(*) FROM registry.vw_case_identifier GROUP BY 1, 2, 3 ORDER BY 1, 2, 3;
${END};
SQL
} | docker exec -i "$PLATFORM_CONTAINER" psql -U ai -d platform -At -F ' | '

echo "seed finished with ${END}"
