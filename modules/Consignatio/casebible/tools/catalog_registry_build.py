#!/usr/bin/env python3
"""Build raw_duck.catalog_registry: one row per catalog relation and per outside object (bucket, Worker, ledger, folder).

Byline: Claude Code · Opus 5.5 · 2026-10-02.
Owner 2026-10-02 19:16 EDT "consolidate all of this somewhere so it stops being forgotten that it exists"; 19:30 option A:
a registry table in the catalog, the map at the top of docs/URGENT-TODO.md, stale tables moved aside. Log: docs/LOG.md.

Input: the read-only classification TSV (schema, table, est_rows, size, kind, covers, as_of, status, superseded_by,
made_by, evidence, suggested_name) and this file's EXTERNAL list. Output: a SQL script on stdout that, in ONE transaction,
  1. moves superseded tables that no live code reads into schema raw_duck_superseded (KEEP_IN_PLACE are still read),
  2. (re)creates raw_duck.catalog_registry and fills it,
  3. prefixes every relation's COMMENT with its registry status, and corrects the schema comments.
Usage: catalog_registry_build.py <registry.tsv> > build.sql ; then run it with psql -v ON_ERROR_STOP=1.
"""
import csv
import sys

# Superseded but still read by live code or a re-runnable pipeline (checked 2026-10-02 by a code search):
#   b2_objects  - Probata engine postgres/catalog_versions.go, tools/contacts_manifest.py, catalog_reconcile/run.py
#   vault_objects - catalog_reconcile/run.py; scrambled_survey_20261002_*.sql (today)
#   vault_content_v0 - catalog_reconcile/run.py
#   raw_duck_d.r2_files - name shared with live readers of raw_duck.r2_files; left for a separate check
KEEP_IN_PLACE = {("raw_duck", "b2_objects"), ("raw_duck", "vault_objects"), ("raw_duck", "vault_content_v0"),
                 ("raw_duck_d", "r2_files")}

EXTERNAL = [  # object_schema, object_name, object_type, kind, covers, as_of, status, superseded_by, made_by, evidence
    ("b2", "salem-data", "bucket", "storage", "Backblaze B2 bucket: the canonical home of the corpus (consignatio/intake, vault/v1, casevault, _system/lake, reconciliation, recovery, timeline_mvp_20260918)", "2026-10-02", "current", "", "", "listed whole 2026-10-02 into raw_duck.bucket_objects"),
    ("b2", "salem-data/consignatio/_system/lake/", "lake_folder", "export_snapshot", "Parquet copy of the catalog published beside the payloads (LATEST names the date); every object recorded in raw_duck.lake_publish_<date>", "2026-09-27", "current", "", "lake_publish_20260927.sh", "raw_duck.lake_publish_20260927"),
    ("r2", "casebible-quarantine", "bucket", "storage", "Cloudflare R2 bucket, retiring (owner 2026-09-28): older corpus copy", "2026-10-02", "historical", "b2:salem-data", "", "listing 2026-10-02 /data/consignatio/listings/r2-all-20261002/"),
    ("r2", "casebible-raw", "bucket", "storage", "Cloudflare R2 bucket, retiring: raw upload area of the 2026-06 corpus", "2026-10-02", "historical", "b2:salem-data", "", "listing 2026-10-02"),
    ("r2", "casebible-sorted", "bucket", "storage", "Cloudflare R2 bucket, retiring: 2026-06/07 sorted corpus", "2026-10-02", "historical", "b2:salem-data", "", "listing 2026-10-02"),
    ("r2", "casebible-hash-ledger", "bucket", "ledger", "SHA-256 ledger of casebible-sorted written by Worker casebible-sha256-backfill (sha256/v2/<keyhash>/<version>.json; _state, _deferred 460 large files, _errors 40)", "2026-08-17", "historical", "", "Worker casebible-sha256-backfill", "_state/sha256-backfill-v2.json status complete 2026-08-17"),
    ("r2", "casebible-lakehouse", "bucket", "export_snapshot", "R2 Data Catalog warehouse (Iceberg): 2026-06 enrichment tables; the enrichment table is also on B2 _system/lake/2026-09-27/enrichment.parquet", "2026-06-23", "superseded", "b2:salem-data/consignatio/_system/lake/", "cb_lakehouse.py", "lake_publish_20260927 row enrichment 15,106"),
    ("r2", "milvus-memsearch", "bucket", "storage", "Object storage of the memsearch Milvus (shared agent memory) - LIVE; must move before the R2 account is released", "2026-10-02", "current", "", "", "owner 2026-10-02: R2 to be released"),
    ("r2", "photos", "bucket", "storage", "Cloudflare R2 bucket 'photos' (2025)", "2026-10-02", "unknown", "", "", "listing 2026-10-02"),
    ("r2", "nexus", "bucket", "storage", "Cloudflare R2 bucket 'nexus' (2026-01)", "2026-10-02", "unknown", "", "", "listing 2026-10-02"),
    ("r2", "r2-explorer-bucket", "bucket", "storage", "Cloudflare R2 bucket of the r2-explorer template Worker (2026-08-08)", "2026-10-02", "unknown", "", "", "listing 2026-10-02"),
    ("cloudflare", "casebible-r2-hasher", "worker", "tool", "Read-only Worker: POST /hash {bucket, keys} streams R2 objects through SHA-1 + SHA-256 (R2 -> B2 nothing-lost proof); secret HASHER_TOKEN in ovh-files /data/consignatio/secrets/r2-hasher.env", "2026-10-02", "current", "", "casebible/tools/r2_hash_worker/", "5/5 test SHA-1 = B2 SHA-1"),
    ("cloudflare", "casebible-sha256-backfill", "worker", "tool", "Self-listing cron Worker that hashed casebible-sorted with SHA-256 into casebible-hash-ledger (complete 2026-08-17)", "2026-08-17", "superseded", "casebible-r2-hasher", "source not in git (bundled copy readable via the Cloudflare connector)", "ledger _state complete"),
    ("ovh-files", "/data/consignatio/listings/", "vps_folder", "listing", "Raw rclone listings that feed raw_duck.bucket_objects (b2-salem-data-<date>/, r2-all-<date>/)", "2026-10-02", "current", "", "bucket_objects_load.py", ""),
    ("ovh-files", "/data/consignatio/migrations/", "vps_folder", "receipt", "Per-run copy lists, rclone logs and verify receipts of the 2026-09 consolidation (inputs to the catalog, not curated)", "2026-09-16", "historical", "", "", "docs/LOG.md 2026-09-15 map"),
    ("ovh-files", "/data/consignatio/court-ready-20261001/", "vps_folder", "receipt", "2026-10-01 runs: R2 -> B2 copy (6,060 files, byte-checked), version restores, zero-file quarantine, hash pass, twin compare", "2026-10-01", "historical", "", "r2_to_b2_20261001.sh, b2_version_ops_20261001.py, hash_pass_20261001.py", "copy/r2.log"),
    ("repo", "modules/Consignatio/docs/receipts/", "repo_folder", "receipt", "Human-readable receipts, catalog exports, hash ledgers (payloads gitignored)", "2026-10-02", "current", "", "", ""),
    ("repo", "modules/Consignatio/docs/LOG.md", "repo_file", "receipt", "The one change log (dated history)", "2026-10-02", "current", "", "", "owner 2026-10-02 19:30"),
    ("repo", "modules/Consignatio/docs/URGENT-TODO.md", "repo_file", "plan", "Open items only; map at the top", "2026-10-02", "current", "", "", "owner 2026-10-02 19:18"),
    ("repo", "modules/Consignatio/docs/COMPLETED-TODO.md", "repo_file", "receipt", "Finished and superseded to-dos with proof", "2026-10-02", "current", "", "", "owner 2026-10-02 19:18"),
    ("weaviate", "IntakeCorpus", "search_index", "content_index", "Weaviate collection of the super index (file chunks); a small slice of the corpus", "2026-09-22", "current", "", "Intake/backend superindex", "37,857 objects 2026-10-02"),
    ("weaviate", "MsgEvents20260918 / ProfferMsgEvents20261002 / AiChatEvents20260918 / DocEvents20261001", "search_index", "content_index", "Weaviate event collections: messages 366,912 + Proffer messages 196,280 + AI chat 446; documents 0", "2026-10-02", "current", "", "", "counts 2026-10-02"),
]

NEW_RELATIONS = [  # created today; not in the classification TSV
    ("raw_duck", "bucket_objects", "table", "listing", "Every object of every bucket (B2 salem-data whole bucket; R2 buckets as loaded), one row per object per listing generation", "2026-10-02", "current", "", "bucket_objects_load.py", "B2 567,757 objects loaded 2026-10-02 23:07Z"),
    ("raw_duck", "bucket_objects_current", "view", "listing", "Newest listing generation of each (provider, bucket) in bucket_objects - START HERE for 'what exists in which bucket'", "2026-10-02", "current", "", "bucket_objects_load.py", ""),
    ("raw_duck", "catalog_registry", "table", "reference", "This registry: what every catalog relation and outside object is, as of when, current or superseded", "2026-10-02", "current", "", "catalog_registry_build.py", "owner 2026-10-02 19:30 option A"),
]

SCHEMA_COMMENTS = {
    "raw_duck": "The live Case Bible catalog (lakehouse working copy; published to B2 _system/lake/). Every relation is described in raw_duck.catalog_registry - check its status there before using it. Start with raw_duck.bucket_objects_current (what exists in which bucket), source_occurrences (where each file came from), vault_keep_v7/vault_delete_v7 (vault lineage).",
    "raw_duck_superseded": "Catalog tables replaced by newer ones, moved here 2026-10-02 so they are not used by mistake. Nothing deleted; raw_duck.catalog_registry names each one's replacement.",
    "inventory": "Atomic-unit detection (export packages, Takeout parts). Most tables empty; see raw_duck.catalog_registry.",
}
OLD_SUPERSEDED_NOTE = "SUPERSEDED BY inventory.*"


def q(v: str) -> str:
    return "'" + (v or "").replace("'", "''") + "'"


def main() -> int:
    rows = list(csv.DictReader(open(sys.argv[1], encoding="utf-8"), delimiter="\t"))
    move = [(r["schema"], r["table"]) for r in rows
            if r["status"] == "superseded" and (r["schema"], r["table"]) not in KEEP_IN_PLACE]
    out = ["\\set ON_ERROR_STOP 1", "begin;",
           "create schema if not exists raw_duck_superseded;"]
    for s, t in move:
        out.append(f"alter table {s}.\"{t}\" set schema raw_duck_superseded;")
    out.append("""
drop table if exists raw_duck.catalog_registry;
create table raw_duck.catalog_registry (
  object_schema text not null,   -- PG schema, or b2 / r2 / cloudflare / ovh-files / repo / weaviate for outside objects
  object_name   text not null,
  object_type   text not null,   -- table | view | foreign_table | bucket | worker | vps_folder | repo_file | ...
  kind          text,            -- listing | content_index | lineage | plan | receipt | analysis | export_snapshot | reference | working | tool | storage | ledger
  covers        text,
  as_of         text,
  status        text not null,   -- current | superseded | historical | intermediate | unknown
  superseded_by text,
  made_by       text,
  evidence      text,
  est_rows      bigint,
  size          text,
  moved_from    text,            -- original schema when moved to raw_duck_superseded on 2026-10-02
  registered_at timestamptz not null default now(),
  primary key (object_schema, object_name)
);""")
    vals = []
    for r in rows:
        s, t = r["schema"], r["table"]
        moved = (s, t) in move
        sup = r["superseded_by"]
        if t == "b2_objects" and s == "raw_duck":
            sup = "raw_duck.bucket_objects_current (all of B2, 2026-10-02); " + sup
        note = r["evidence"] + (f" | suggested name: {r['suggested_name']}" if r.get("suggested_name") else "")
        est = r["est_rows"] if r["est_rows"].lstrip("-").isdigit() else "null"
        vals.append("(" + ", ".join([q("raw_duck_superseded" if moved else s), q(t), "'relation'", q(r["kind"]),
                    q(r["covers"]), q(r["as_of"]), q(r["status"]), q(sup), q(r["made_by"]), q(note), est,
                    q(r["size"]), q(s if moved else "")]) + ")")
    for e in NEW_RELATIONS + EXTERNAL:
        vals.append("(" + ", ".join([q(x) for x in e] + ["null", "''", "''"]) + ")")
    out.append("insert into raw_duck.catalog_registry (object_schema, object_name, object_type, kind, covers, as_of, "
               "status, superseded_by, made_by, evidence, est_rows, size, moved_from) values\n" + ",\n".join(vals) + ";")
    # real relkind for PG relations
    out.append("""
update raw_duck.catalog_registry g set object_type = case c.relkind when 'r' then 'table' when 'p' then 'table'
  when 'v' then 'view' when 'm' then 'materialized_view' when 'f' then 'foreign_table' else c.relkind::text end
from pg_class c join pg_namespace n on n.oid = c.relnamespace
where n.nspname = g.object_schema and c.relname = g.object_name;
do $$
declare r record; old text; kw text;
begin
  for r in select g.*, c.oid, c.relkind from raw_duck.catalog_registry g
           join pg_namespace n on n.nspname = g.object_schema
           join pg_class c on c.relnamespace = n.oid and c.relname = g.object_name loop
    old := obj_description(r.oid, 'pg_class');
    if old like '[registry:%' then old := nullif(substr(old, position('] ' in old) + 2), ''); end if;
    old := regexp_replace(coalesce(old, ''), '^.*?\\| previous: ', '');
    kw := case r.relkind when 'v' then 'view' when 'm' then 'materialized view' when 'f' then 'foreign table' else 'table' end;
    execute format('comment on %s %I.%I is %L', kw, r.object_schema, r.object_name,
      format('[registry: %s, as of %s] %s%s. See raw_duck.catalog_registry.%s',
             upper(r.status), coalesce(nullif(r.as_of, ''), '?'), coalesce(r.covers, ''),
             case when coalesce(r.superseded_by, '') <> '' then ' Superseded by: ' || r.superseded_by else '' end,
             case when old <> '' then ' | previous: ' || old else '' end));
  end loop;
end $$;""")
    for sch, txt in SCHEMA_COMMENTS.items():
        out.append(f"comment on schema {sch} is {q(txt)};")
    out.append(f"""
do $$
declare s record;
begin
  for s in select nspname, obj_description(oid, 'pg_namespace') d from pg_namespace
           where obj_description(oid, 'pg_namespace') like '%{OLD_SUPERSEDED_NOTE}%' and nspname not in ('raw_duck', 'inventory') loop
    execute format('comment on schema %I is %L', s.nspname,
      replace(replace(s.d, 'SUPERSEDED BY inventory.*. DO NOT READ for analysis.', 'Historical import; see raw_duck.catalog_registry.'),
              'SUPERSEDED BY inventory.*', 'Historical import; see raw_duck.catalog_registry'));
  end loop;
end $$;
commit;
select status, count(*) from raw_duck.catalog_registry group by 1 order by 2 desc;
select count(*) as moved_to_raw_duck_superseded from pg_tables where schemaname = 'raw_duck_superseded';""")
    print("\n".join(out))
    return 0


if __name__ == "__main__":
    sys.exit(main())
