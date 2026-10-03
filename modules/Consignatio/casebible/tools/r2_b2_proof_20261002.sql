-- Byline: Claude Code · Opus 5.5 · 2026-10-02
-- R2 -> B2 "nothing lost" proof, step 1 (catalog only, no bucket reads).
-- Owner 2026-10-02 19:06 EDT: "I wanna know 100% that we're not losing anything when I let that bucket or that account
-- go"; 19:16: hash only the ones we don't have a proper SHA for. Log: modules/Consignatio/docs/LOG.md.
--
-- Every R2 object (raw_duck.bucket_objects_current, provider r2) is classified:
--   zero_byte        size 0: nothing to lose
--   safe_md5_sha1    its MD5+size is linked to a SHA-1 (see bridge) and that SHA-1+size exists on B2 now (any key)
--   needs_hash       no usable MD5 (multipart ETag), or no MD5->SHA-1 link, or the linked SHA-1 is not on B2 now:
--                    goes to the casebible-r2-hasher Worker (SHA-1 of the R2 bytes), then step 2 re-checks against B2
-- The MD5->SHA-1 bridge comes only from rows where one party hashed the same bytes both ways:
--   a) b2_content (md5 of the bytes copied to B2) -> vault_keep_v7 (that copy's kept vault key) -> the 09-16 vault
--      listing's SHA-1 for that key (B2-computed) and the 09-18 vault index
--   b) source_occurrences rows that carry both an md5 and a provider/disk SHA-1 (native_hash_kind = 'sha1')
-- A bridge MD5 that maps to two different SHA-1s for the same size is dropped (never trusted).
\set ON_ERROR_STOP 1
begin;
drop table if exists raw_duck.md5_sha1_bridge_20261002;
create table raw_duck.md5_sha1_bridge_20261002 as
with pairs as (
  select c.md5, c.size, v.sha1 from raw_duck.b2_content c
  join raw_duck.vault_keep_v7 k on k.canonical_key = c.b2_key and k.size = c.size
  join raw_duck.vault_objects_20260916_r4 v on v.key = k.dest_key and v.size = c.size
  where c.md5 ~ '^[0-9a-f]{32}$' and v.sha1 ~ '^[0-9a-f]{40}$'
  union
  select c.md5, c.size, v.sha1 from raw_duck.b2_content c
  join raw_duck.vault_keep_v7 k on k.canonical_key = c.b2_key and k.size = c.size
  join raw_duck.vault_index_source_20260918 v on v.key = k.dest_key and v.size = c.size
  where c.md5 ~ '^[0-9a-f]{32}$' and v.sha1 ~ '^[0-9a-f]{40}$'
  union
  select lower(o.md5), o.size, lower(o.native_hash) from raw_duck.source_occurrences o
  where o.native_hash_kind = 'sha1' and o.md5 ~* '^[0-9a-f]{32}$' and o.native_hash ~* '^[0-9a-f]{40}$'
)
select md5, size, min(sha1) as sha1 from pairs group by md5, size having count(distinct sha1) = 1;
create index on raw_duck.md5_sha1_bridge_20261002 (md5, size);

drop table if exists raw_duck.r2_b2_proof_20261002;
create table raw_duck.r2_b2_proof_20261002 as
with b2 as (select distinct sha1, size from raw_duck.bucket_objects_current
            where provider = 'b2' and bucket = 'salem-data' and sha1 is not null)
select r.bucket, r.key, r.size, r.md5, br.sha1 as bridged_sha1,
       case when r.size = 0 then 'zero_byte'
            when b2.sha1 is not null then 'safe_md5_sha1'
            else 'needs_hash' end as status,
       null::text as r2_sha1, null::text as r2_sha256, null::text as proof_detail
from raw_duck.bucket_objects_current r
left join raw_duck.md5_sha1_bridge_20261002 br on br.md5 = r.md5 and br.size = r.size
left join b2 on b2.sha1 = br.sha1 and b2.size = r.size
where r.provider = 'r2';
alter table raw_duck.r2_b2_proof_20261002 add primary key (bucket, key);
comment on table raw_duck.r2_b2_proof_20261002 is
  'R2 -> B2 nothing-lost proof (2026-10-02): one row per R2 object; status zero_byte | safe_md5_sha1 | needs_hash | safe_sha1 | missing_on_b2. Built by casebible/tools/r2_b2_proof_20261002.sql; needs_hash rows are hashed by the casebible-r2-hasher Worker.';
commit;
select (select count(*) from raw_duck.md5_sha1_bridge_20261002) as bridge_rows;
select bucket, status, count(*), pg_size_pretty(sum(size)) from raw_duck.r2_b2_proof_20261002 group by 1, 2 order by 1, 2;
