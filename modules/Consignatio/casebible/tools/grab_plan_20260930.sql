-- Byline: Claude Code · Opus 5.5 · 2026-09-30
-- What must be fetched so every file has its best copy on B2, and every corrupt file is replaced, computed from the
-- catalog alone: no object-store, Drive, OneDrive, R2 or local reads (owner 2026-09-30 15:52: "using the catalog can you
-- figure out what we need to grab in order to satisfy the oldest real + most meta, ensuring any corrupt files are
-- replaced without doing a full move and read of everything again"). Log: docs/LOG.md, 2026-09-30 entry.
--
-- Inputs (read only):
--   catalog_reconcile.object_versions  09-20 B2 listing (generation 2c2ae40f), visible and noncurrent versions, sha1
--   catalog_reconcile.occurrences      every source occurrence with its B2 availability (visible / historical / none)
--   catalog_reconcile.r2_occurrences   every R2 object with its B2 link state
--   raw_duck.source_occurrences        disposition per occurrence (content_on_b2 | copied | zero_byte | junk_excluded | exported)
--   raw_duck.corrupt_recovery          09-14 corrupt set: 47,058 all-zero occurrences (identity = basename + size)
-- Outputs (new, additive):
--   raw_duck.best_copy_20260930   per visible content (sha1 + size): the copy whose metadata wins by the owner's
--                                 2026-09-13 rule. Bytes are identical within a content, so nothing is fetched for it.
--   raw_duck.grab_plan_20260930   one row per file that needs bytes moved, with where from and why.
--
-- Owner's grading rule (2026-09-13 11:46 + 12:40, grading_selection.sql): oldest REAL date wins; sentinel dates
-- (1970-01-01, 1979-12-31, 1980-01-01, anything before 1990) and batch stamps (one exact timestamp on >100 copies =
-- a copy event) are not real; then most complete metadata; filename least; still tied = keep both; zero-filled and
-- zero-byte copies are never candidates. Dates considered per copy: the recorded modtime, Drive btime and mtime,
-- OneDrive ModTime. Completeness = number of non-empty metadata fields the copy carries.
--
-- Known limit: a file with no hash is matched by basename + size, the same identity the 09-14 corrupt count used.
-- Such matches are labelled *_by_name_size so they are never mistaken for byte-verified ones.

begin;
set local work_mem = '256MB';

-- 0. Visible B2 objects outside the zero-filled quarantine, and the zero-filled contents we know by hash.
create temp table vis as
  select file_id, object_key, sha1, size, regexp_replace(object_key, '^.*/', '') as name
  from catalog_reconcile.object_versions
  where visible and action = 'upload' and size > 0
    and object_key not like 'consignatio/intake/_quarantine/%';
create index on vis (name, size);
create index on vis (sha1, size);
create temp table zero_content as
  select distinct size, hash as sha1 from raw_duck.corrupt_recovery where store = 'b2' and length(hash) = 40;
create temp table vis_good as
  select v.* from vis v where not exists (select 1 from zero_content z where z.size = v.size and z.sha1 = v.sha1);
create index on vis_good (name, size);

select 'visible objects (outside quarantine)', count(*), round(sum(size) / 1e9, 1) as gb from vis;
select 'visible objects that are known zero-filled', count(*), round(coalesce(sum(size), 0) / 1e9, 2) as gb
  from vis v where exists (select 1 from zero_content z where z.size = v.size and z.sha1 = v.sha1);

-- Occurrence disposition, one row per occurrence (source + path + size can repeat on Drive).
create temp table occ as
  select o.occurrence_id, o.source, o.source_path, o.size, o.availability, o.version_ids,
         o.recorded_modtime, coalesce(o.source_metadata, '{}'::jsonb) as md,
         regexp_replace(o.source_path, '^.*/', '') as name,
         (select s.disposition from raw_duck.source_occurrences s
           where s.source = o.source and s.path = o.source_path and s.size = o.size
           order by (s.disposition = 'zero_byte') desc limit 1) as disposition
  from catalog_reconcile.occurrences o;
create index on occ (name, size);

-- 1. Best copy per visible content (catalog only).
create temp table occ_vis as
  select c.*, v.sha1
  from occ c
  cross join lateral (select v.sha1 from jsonb_array_elements_text(c.version_ids) vid
                      join catalog_reconcile.object_versions v on v.file_id = vid limit 1) v
  where c.availability in ('visible_exact_path', 'visible_exact_elsewhere')
    and coalesce(c.disposition, '') not in ('zero_byte', 'junk_excluded')
    and c.size > 0;

create temp table cand_dates as
  select occurrence_id, d
  from occ_vis,
       lateral (values (nullif(recorded_modtime, '')), (md ->> 'btime'), (md ->> 'mtime'), (md ->> 'ModTime')) x(t),
       lateral (select case when t ~ '^\d{4}-\d{2}-\d{2}' then t::timestamptz end as d) y
  where d is not null;
create temp table batch_stamps as
  select d from cand_dates group by d having count(distinct occurrence_id) > 100;
create temp table oldest_real as
  select c.occurrence_id, min(c.d) as oldest_real
  from cand_dates c
  where c.d >= timestamptz '1990-01-01'
    and c.d::date not in (date '1970-01-01', date '1979-12-31', date '1980-01-01')
    and not exists (select 1 from batch_stamps b where b.d = c.d)
  group by c.occurrence_id;

create temp table scored as
  select v.occurrence_id, v.source, v.source_path, v.sha1, v.size, r.oldest_real,
         (select count(*) from jsonb_each_text(v.md) e
           where coalesce(e.value, '') not in ('', 'null', '[]', '{}')) as completeness
  from occ_vis v left join oldest_real r using (occurrence_id);

create temp table ranked as
  select s.*,
         rank() over (partition by sha1, size
                      order by (oldest_real is null), oldest_real, completeness desc) as pick_rank,
         count(*) over (partition by sha1, size) as copies
  from scored s;

create table raw_duck.best_copy_20260930 as
  select sha1, size, occurrence_id, source, source_path, oldest_real, completeness, copies,
         case when count(*) over (partition by sha1, size) = 1 then 'winner' else 'keep_both' end as disposition,
         'owner-2026-09-13' as rule_version, now() as graded_at
  from ranked where pick_rank = 1;
comment on table raw_duck.best_copy_20260930 is
  'Best copy per visible B2 content (sha1+size) by the owner''s 2026-09-13 rule: oldest real date, then most metadata; ties kept. Catalog only, bytes identical within a content. Script casebible/tools/grab_plan_20260930.sql; log docs/LOG.md 2026-09-30.';

select 'best copy: contents', count(distinct (sha1, size)),
       'winners', count(*) filter (where disposition = 'winner'),
       'keep-both contents', count(distinct (sha1, size)) filter (where disposition = 'keep_both'),
       'no real date on any copy', count(distinct (sha1, size)) filter (where oldest_real is null)
  from raw_duck.best_copy_20260930;
select 'best copy: winning source', source, count(distinct (sha1, size)) from raw_duck.best_copy_20260930 group by source order by 3 desc;

-- 2. What needs bytes moved.
create table raw_duck.grab_plan_20260930 (
  kind        text not null,   -- restore_b2_version | corrupt | no_hash_file | r2_only
  status      text not null,   -- what the catalog says must happen (see the summary)
  name        text,
  size        bigint,
  item        text not null,   -- the occurrence / corrupt copy / R2 object this row is about
  from_where  text,            -- where the good bytes are: b2 key, B2 version id, source path, or r2 path
  reason      text,
  planned_at  timestamptz not null default now()
);
comment on table raw_duck.grab_plan_20260930 is
  'Files that need bytes moved so every file has a good copy on B2 (catalog only). Script casebible/tools/grab_plan_20260930.sql; log docs/LOG.md 2026-09-30.';

-- 2a. Bytes that survive only as an old B2 version: restore on B2 (server-side copy), no re-pull from any source.
insert into raw_duck.grab_plan_20260930 (kind, status, name, size, item, from_where, reason)
select 'restore_b2_version',
       case when exists (select 1 from vis_good g where g.sha1 = v.sha1 and g.size = v.size) then 'already_visible'
            else 'restore' end,
       c.name, c.size, c.source || ':' || c.source_path, v.file_id,
       'occurrence bytes exist only as a noncurrent B2 version'
from occ c
cross join lateral (select v.file_id, v.sha1, v.size from jsonb_array_elements_text(c.version_ids) vid
                    join catalog_reconcile.object_versions v on v.file_id = vid limit 1) v
where c.availability = 'historical_exact';

-- 2b. Real files the catalog cannot tie to B2 bytes by hash (large multipart uploads carry no SHA-1).
insert into raw_duck.grab_plan_20260930 (kind, status, name, size, item, from_where, reason)
select 'no_hash_file',
       case when g.object_key is not null then 'present_by_name_size' else 'grab_from_source' end,
       c.name, c.size, c.source || ':' || c.source_path,
       coalesce(g.object_key, c.source || ':' || c.source_path),
       'no usable hash; disposition ' || coalesce(c.disposition, 'unknown')
from occ c
left join lateral (select g.object_key from vis_good g where g.name = c.name and g.size = c.size limit 1) g on true
where c.availability = 'no_usable_identity'
  and coalesce(c.disposition, '') in ('content_on_b2', 'copied', 'to_copy')
  and c.size > 0;

-- 2c. Corrupt (all-zero) copies: is a good copy on B2 now, only in a source, only in R2, or nowhere?
create temp table corrupt_ids as
  select name, size, count(*) as corrupt_copies, min(store || ':' || coalesce(nullif(container, '') || '/', '') || path) as example
  from raw_duck.corrupt_recovery group by name, size;
create temp table good_occ as
  select c.name, c.size, c.availability, c.source || ':' || c.source_path as where_
  from occ c
  where coalesce(c.disposition, '') in ('content_on_b2', 'copied', 'exported') and c.size > 0;
create index on good_occ (name, size);
create temp table r2_by_name as
  select regexp_replace(source_path, '^.*/', '') as name, nullif(source_record ->> 'size', '')::bigint as size,
         availability, bucket || ':' || source_path as where_
  from catalog_reconcile.r2_occurrences;
create index on r2_by_name (name, size);

insert into raw_duck.grab_plan_20260930 (kind, status, name, size, item, from_where, reason)
select 'corrupt', r.status, i.name, i.size, i.example, r.from_where,
       i.corrupt_copies || ' all-zero copies'
from corrupt_ids i
cross join lateral (
  select * from (
    (select 1 as o, 'good_on_b2' as status, g.object_key as from_where
       from vis_good g where g.name = i.name and g.size = i.size limit 1)
    union all
    (select 2, 'good_on_b2_renamed', o.where_ from good_occ o
      where o.name = i.name and o.size = i.size and o.availability like 'visible%' limit 1)
    union all
    (select 3, 'restore_b2_version', o.where_ from good_occ o
      where o.name = i.name and o.size = i.size and o.availability = 'historical_exact' limit 1)
    union all
    (select 4, 'grab_from_source', o.where_ from good_occ o
      where o.name = i.name and o.size = i.size and o.availability = 'no_usable_identity' limit 1)
    union all
    (select 5, 'good_in_r2_on_b2', r.where_ from r2_by_name r
      where r.name = i.name and r.size = i.size and r.availability like '%catalog_link' limit 1)
    union all
    (select 6, 'grab_from_r2', r.where_ from r2_by_name r
      where r.name = i.name and r.size = i.size and r.availability = 'unresolved' limit 1)
    union all
    select 7, 'no_good_copy', null
  ) x order by o limit 1
) r;

-- 2d. R2 objects never tied to B2 bytes (R2 is being retired): present by name + size, or fetch from R2.
insert into raw_duck.grab_plan_20260930 (kind, status, name, size, item, from_where, reason)
select 'r2_only',
       case when g.object_key is not null then 'present_by_name_size' else 'grab_from_r2' end,
       r.name, r.size, r.where_, coalesce(g.object_key, r.where_),
       'R2 object with no hash link to B2'
from r2_by_name r
left join lateral (select g.object_key from vis_good g where g.name = r.name and g.size = r.size limit 1) g on true
where r.availability = 'unresolved';

-- 3. Summary for the owner.
select kind, status, count(*) as files, round(sum(size) / 1e9, 2) as gb
from raw_duck.grab_plan_20260930 group by 1, 2 order by 1, 2;
select 'grab_from_source by source', split_part(from_where, ':', 1) as source, count(*), round(sum(size) / 1e9, 2) as gb
from raw_duck.grab_plan_20260930 where status = 'grab_from_source' group by 2 order by 3 desc;

commit;
