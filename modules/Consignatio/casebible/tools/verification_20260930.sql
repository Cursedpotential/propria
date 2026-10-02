-- Byline: Claude Code · Opus 5.5 · 2026-09-30
-- Per vault file: how strong the proof is that it is what it claims to be, and anything that looks off.
-- Owner 2026-09-30 22:21 EDT: "make sure the files are verifiably what they say they are, ready for court evaluation if
-- need be, and that there's no funny business with any of them." Catalog reads only; no object-store or source reads.
-- Log: docs/LOG.md, 2026-09-30 entries. Output: raw_duck.verification_20260930 (new, additive).
--
-- Proof levels (strongest first):
--   independent_sha1    the source's own SHA-1 (Google Drive / OneDrive provider hash, or our disk hasher on D:/F:)
--                       equals the SHA-1 B2 computed on upload: two independent parties hashed the same bytes.
--   b2_sha1_only        B2 holds a SHA-1, but no source occurrence carries its own SHA-1 to compare with.
--   no_hash             B2 holds no SHA-1 (multipart uploads); identity rests on name/size until a read pass hashes it.
-- Flags (anything that deserves a look before a file is relied on):
--   no_source_link      no recorded source occurrence points at this object (made by our own processes, or unlinked)
--   zero_filled         the bytes are a known all-zero payload
--   altered_twin        another copy with the same name AND the same size has DIFFERENT bytes somewhere in the sources
--                       (same-size edits, partial overwrites, or corruption; the strongest catalog sign of tampering)
--   date_after_capture  a recorded date is later than the moment the copy was catalogued (impossible for an original)
--   future_date         a recorded date is after today
--   no_real_date        every recorded date is a sentinel, pre-1990, or a batch stamp

begin;
set local work_mem = '256MB';

create temp table vobj as
  select file_id, object_key, sha1, size
  from catalog_reconcile.object_versions
  where visible and action = 'upload' and object_key like 'consignatio/vault/v1/%';
create index on vobj (file_id);

-- every source occurrence linked to a vault object, with the source's own hash and dates
create temp table link as
  select v.file_id, r.payload ->> 'source' as source,
         nullif(r.payload -> 'source_record' ->> 'sha1', '') as src_sha1,
         r.payload -> 'source_record' as rec,
         regexp_replace(coalesce(r.payload -> 'source_record' ->> 'path', ''), '^.*/', '') as name,
         nullif(r.payload -> 'source_record' ->> 'size', '')::bigint as size
  from raw_duck.reconcile_occurrences_20260920 r
  cross join lateral jsonb_array_elements_text(r.payload -> 'version_ids') vid
  join vobj v on v.file_id = vid
  where r.payload ->> 'state' like 'visible%';
create index on link (file_id);

create temp table zero_content as
  select distinct size, hash as sha1 from raw_duck.corrupt_recovery where store = 'b2' and length(hash) = 40;

-- altered twins: same basename + same size, more than one distinct source SHA-1 (zero-filled copies excluded)
create temp table src_named as
  select regexp_replace(coalesce(r.payload -> 'source_record' ->> 'path', ''), '^.*/', '') as name,
         nullif(r.payload -> 'source_record' ->> 'size', '')::bigint as size,
         nullif(r.payload -> 'source_record' ->> 'sha1', '') as sha1
  from raw_duck.reconcile_occurrences_20260920 r;
create temp table altered as
  select s.name, s.size
  from src_named s
  where s.sha1 is not null and s.size > 0
    and not exists (select 1 from zero_content z where z.size = s.size and z.sha1 = s.sha1)
  group by s.name, s.size having count(distinct s.sha1) > 1;
create index on altered (name, size);

-- dates per linked copy
create temp table ldates as
  select l.file_id, d, nullif(l.rec ->> 'recorded_at', '')::timestamptz as captured
  from link l,
       lateral (values (l.rec ->> 'modtime'), (l.rec -> 'metadata' ->> 'btime'), (l.rec -> 'metadata' ->> 'mtime'),
                       (l.rec -> 'metadata' ->> 'ModTime')) x(t),
       lateral (select case when t ~ '^\d{4}-\d{2}-\d{2}' then t::timestamptz end as d) y
  where d is not null;

create table raw_duck.verification_20260930 as
select v.object_key, v.sha1, v.size,
       case when v.sha1 is null or v.sha1 in ('', 'none') then 'no_hash'
            when exists (select 1 from link l where l.file_id = v.file_id and l.src_sha1 = v.sha1) then 'independent_sha1'
            else 'b2_sha1_only' end as proof,
       (select count(*) from link l where l.file_id = v.file_id) as source_copies,
       (select count(distinct l.source) from link l where l.file_id = v.file_id) as sources,
       array_remove(array[
         case when not exists (select 1 from link l where l.file_id = v.file_id) then 'no_source_link' end,
         case when exists (select 1 from zero_content z where z.size = v.size and z.sha1 = v.sha1) then 'zero_filled' end,
         case when exists (select 1 from altered a where a.name = regexp_replace(v.object_key, '^.*/', '') and a.size = v.size)
                or exists (select 1 from link l join altered a on a.name = l.name and a.size = l.size where l.file_id = v.file_id)
              then 'altered_twin' end,
         case when exists (select 1 from ldates d where d.file_id = v.file_id and d.captured is not null
                                                     and d.d > d.captured + interval '1 day') then 'date_after_capture' end,
         case when exists (select 1 from ldates d where d.file_id = v.file_id and d.d > now() + interval '1 day') then 'future_date' end,
         case when not exists (select 1 from raw_duck.best_copy_20260930 b where b.sha1 = v.sha1 and b.size = v.size and b.oldest_real is not null)
              then 'no_real_date' end
       ], null) as flags,
       now() as checked_at
from vobj v;
comment on table raw_duck.verification_20260930 is
  'Per vault object: proof level (independent_sha1 | b2_sha1_only | no_hash) and flags (no_source_link, zero_filled, altered_twin, date_after_capture, future_date, no_real_date). Catalog only. Script casebible/tools/verification_20260930.sql; log docs/LOG.md 2026-09-30.';

select proof, count(*) as files, round(sum(size) / 1e9, 1) as gb from raw_duck.verification_20260930 group by 1 order by 2 desc;
select f as flag, count(*) as files, round(sum(size) / 1e9, 1) as gb
  from raw_duck.verification_20260930, unnest(flags) f group by 1 order by 2 desc;
select 'clean: independent_sha1 and no flag except no_real_date', count(*), round(sum(size) / 1e9, 1)
  from raw_duck.verification_20260930
  where proof = 'independent_sha1' and coalesce(array_remove(flags, 'no_real_date'), '{}') = '{}';
select 'altered-twin groups in the sources (name+size, >1 distinct content)', count(*) from altered;

commit;
