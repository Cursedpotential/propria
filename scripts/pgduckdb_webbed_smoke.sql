-- Byline: Claude Code · Fable 5.1 · 2026-09-06
-- Smoke check for the DuckDB `webbed` community extension inside pg_duckdb
-- (read_xml / read_html for extract-only templates, D-149 item 9).
-- Run against the platform DB as a superuser-capable role:
--   psql -U ai -d platform -f scripts/pgduckdb_webbed_smoke.sql
-- Expects /var/lib/postgresql/webbed_smoke.xml to exist in the PG container (see the runbook
-- note in docs/reviews/2026-09-06-webbed-install.md).

\echo == 1. pg_duckdb alive
select * from duckdb.raw_query($$select 1 as ok$$);

\echo == 2. community extensions enabled?
show duckdb.allow_community_extensions;

\echo == 3. webbed installed / loaded?
select * from duckdb.raw_query($$
  select extension_name, installed, loaded
  from duckdb_extensions() where extension_name = 'webbed'
$$);

\echo == 4. read_xml over a tiny SMS-shaped document
select * from duckdb.raw_query($$
  select address, date, body from read_xml('/var/lib/postgresql/webbed_smoke.xml')
$$);

\echo == 5. row count
select * from duckdb.raw_query($$
  select count(*) as n from read_xml('/var/lib/postgresql/webbed_smoke.xml')
$$);
