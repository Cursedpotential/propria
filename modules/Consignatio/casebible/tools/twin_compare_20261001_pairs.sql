-- Byline: Claude Code · Opus 5.5 · 2026-10-01
-- Pairs for twin_compare_20261001.py: media files over 1 MB where two copies share the exact name and size but not the
-- bytes (verification_20260930 flag altered_twin). One row per pair: the group's first visible copy against each other
-- distinct content that is visible on B2. Read only. Log: docs/URGENT-TODO.md, 2026-10-01 entry.
create temp table src as
  select regexp_replace(coalesce(payload -> 'source_record' ->> 'path', ''), '^.*/', '') as name,
         nullif(payload -> 'source_record' ->> 'size', '')::bigint as size,
         nullif(payload -> 'source_record' ->> 'sha1', '') as sha1
  from raw_duck.reconcile_occurrences_20260920;
create temp table zero_content as
  select distinct size, hash as sha1 from raw_duck.corrupt_recovery where store = 'b2' and length(hash) = 40;
create temp table groups as
  select name, size, array_agg(distinct sha1) as sha1s
  from src
  where sha1 is not null and size > 1000000
    and lower(substring(name from '\.([A-Za-z0-9]{1,5})$')) in
        ('jpg','jpeg','png','heic','heif','gif','webp','bmp','tif','tiff','mp4','mov','m4v','3gp','avi','mkv','mp3','m4a','wav','pdf','docx')
    and not exists (select 1 from zero_content z where z.size = src.size and z.sha1 = src.sha1)
  group by name, size having count(distinct sha1) > 1;
create temp table members as
  select g.name, g.size, s.sha1,
         (select v.object_key from catalog_reconcile.object_versions v
           where v.visible and v.action = 'upload' and v.sha1 = s.sha1 and v.size = g.size
             and v.object_key not like 'consignatio/intake/_quarantine/%'
           order by v.object_key limit 1) as object_key
  from groups g cross join lateral unnest(g.sha1s) s(sha1);
create temp table ranked as
  select m.*, row_number() over (partition by name, size order by object_key nulls last, sha1) as rn,
         count(object_key) over (partition by name, size) as on_b2
  from members m;
\copy (select a.name || '|' || a.size as group_id, a.size, a.object_key as key_a, b.object_key as key_b from ranked a join ranked b on b.name = a.name and b.size = a.size and b.rn > 1 where a.rn = 1 and a.object_key is not null and b.object_key is not null order by a.size desc) to stdout with (format csv, delimiter E'\t', header true)
select '#summary groups', count(*), 'with >=2 copies on B2', count(*) filter (where n >= 2), 'with <2 on B2', count(*) filter (where n < 2)
  from (select name, size, max(on_b2) as n from ranked group by 1, 2) x;
