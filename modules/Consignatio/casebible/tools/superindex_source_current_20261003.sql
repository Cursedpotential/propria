-- Coco Super Index source view: one row per object in the CURRENT bucket listings, every bucket.
-- Byline: Claude Code · Sonnet 5.5 · 2026-10-02
--
-- Why a view and not another dated table: the Super Index runs on a schedule and must see new listings without
-- anyone rebuilding a table. raw_duck.bucket_objects_current is the newest listing of each (provider, bucket)
-- (loaded by the catalog loader; 2.19 M rows over 8 buckets on 2026-10-02). raw_duck.vault_index_source_20260918
-- is the 2026-09-22 snapshot that carries every recorded occurrence (original path, name, modtime, native hash,
-- disposition) of a B2 salem-data object; it is joined, never rewritten. An object listed after that snapshot has
-- no occurrence rows yet: occurrence_count = 0 and occurrences = '[]'. Nothing is merged or invented.
--
-- Read-only grant needed for the index's catalog role (metabase_ro already reads raw_duck):
--   GRANT SELECT ON raw_duck.superindex_source_current TO metabase_ro;
--
-- Apply (one view, nothing else written):
--   ssh -i ~/.ssh/ovh root@100.91.190.107 "docker exec -i casebible-pg-oli1nf8wmj5atb9o7oylj4um-131418412167 \
--     psql -U postgres -d casebible -v ON_ERROR_STOP=1" < superindex_source_current_20261003.sql
-- Verify: the count must equal bucket_objects_current (2,186,449 on 2026-10-02) and the salem-data rows with an
-- occurrence snapshot must be 508,152 or fewer (objects removed since 09-22 drop out).

create or replace view raw_duck.superindex_source_current as
select o.provider,
       o.bucket,
       o.key,
       o.size,
       o.sha1,
       o.md5,
       o.modtime,
       o.listed_at,
       coalesce(v.name, regexp_replace(o.key, '^.*/', '')) as name,
       coalesce(v.occurrence_count, 0) as occurrence_count,
       coalesce(v.occurrences, '[]'::jsonb) as occurrences
from raw_duck.bucket_objects_current o
left join raw_duck.vault_index_source_20260918 v
       on o.provider = 'b2' and o.bucket = 'salem-data' and v.key = o.key;

grant select on raw_duck.superindex_source_current to metabase_ro;

select count(*) as objects, count(*) filter (where occurrence_count > 0) as with_occurrences
from raw_duck.superindex_source_current;
select (select count(*) from raw_duck.bucket_objects_current) as current_listing_objects;
