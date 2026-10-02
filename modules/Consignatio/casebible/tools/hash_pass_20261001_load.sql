-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- Load the hash pass (hash_pass_20261001.py export -> /tmp/hash_pass_20261001.tsv inside the catalog container) and
-- match every computed SHA-256 against the sources' own SHA-256 (Google Drive's provider hash, our D:/F: disk hasher).
-- Additive: one new table raw_duck.hash_pass_20261001. Log: docs/LOG.md, 2026-10-01 entry.
\set ON_ERROR_STOP on
begin;
create table raw_duck.hash_pass_20261001 (
  object_key text primary key, size_expected bigint, size_read bigint,
  sha256 text, sha1 text, md5 text, status text, err text, loaded_at timestamptz not null default now());
comment on table raw_duck.hash_pass_20261001 is
  'SHA-256/SHA-1/MD5 of B2 vault objects that carry no B2 SHA-1 or match their sources by size only, computed by streaming each object once (casebible/tools/hash_pass_20261001.py, 2026-10-01). Log docs/LOG.md 2026-10-01.';
\copy raw_duck.hash_pass_20261001 (object_key, size_expected, size_read, sha256, sha1, md5, status, err) from '/tmp/hash_pass_20261001.tsv' with (format text, header true, null '')
select status, count(*), round(sum(size_read) / 1e9, 1) as gb from raw_duck.hash_pass_20261001 group by 1;

-- does any source copy carry the same SHA-256 (or SHA-1, or MD5) for the same size?
create temp table m as
  select h.object_key, h.size_read,
         exists (select 1 from raw_duck.source_occurrences s where s.native_hash_kind = 'sha256' and lower(s.native_hash) = h.sha256 and s.size = h.size_read) as sha256_match,
         exists (select 1 from raw_duck.source_occurrences s where lower(s.md5) = h.md5 and s.size = h.size_read) as md5_match
  from raw_duck.hash_pass_20261001 h where h.status = 'ok';
select case when sha256_match then 'source sha256 = B2 bytes'
            when md5_match then 'source md5 = B2 bytes'
            else 'no source hash matches' end as proof, count(*), round(sum(size_read) / 1e9, 1) as gb
  from m group by 1 order by 2 desc;
select 'unmatched, largest' as t, round(size_read / 1e9, 2) as gb, object_key from m where not sha256_match and not md5_match order by size_read desc limit 15;
select 'failed reads' as t, size_expected, object_key from raw_duck.hash_pass_20261001 where status <> 'ok' order by size_expected desc;
commit;
