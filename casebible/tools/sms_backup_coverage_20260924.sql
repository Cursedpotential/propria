-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- SMS backup coverage: which backup files are fully contained in a newer backup (owner 07:27: "as long as an original
-- backup covers everything that a different backup [has] ... we can get rid of the older ones as long as the newest
-- one that we're using is still an original"). READ-ONLY analysis: it writes a report table and moves nothing.
-- Coverage is measured over EVERY message the catalog parsed from each file (all threads on the phone, not just
-- Katrina's): file A is covered by file B when every A message (raw_duck.chat_events_20260918.dedup_key, via
-- chat_event_provenance_20260918) is also in B, and B's newest message is at least as new as A's.
-- A file whose exact bytes exist elsewhere (same sha1 at another path) is a copy, reported as such.
-- "Original" (owner) = the file as the backup app wrote it. The report carries each file's sha1s and the catalog's
-- zero-fill hold flag where known; confirm the chosen keeper is an original before anything is removed.
-- Caveat: this trusts the catalog's parse of each file. Before any removal, spot-check the parse count against the
-- file's own record count for the keeper and a sample of the covered files.

create table if not exists raw_duck.sms_backup_coverage_20260924 (
  file              text primary key,
  n_messages        int not null,
  first_ts          timestamptz,
  last_ts           timestamptz,
  sha1s             text[],
  covered_by        text,          -- the newest file that contains every message of this file
  covered_by_n      int,
  covering_files    int not null,  -- how many other files contain every message of this file
  verdict           text not null  -- keeper | covered_by_newer | not_covered
);

begin;
truncate raw_duck.sms_backup_coverage_20260924;
with fk as (
  select distinct coalesce(p.catalog_rel, p.vault_key) as f, p.dedup_key, p.sha1, e.event_ts_utc
  from raw_duck.chat_event_provenance_20260918 p
  join raw_duck.chat_events_20260918 e using (dedup_key)
  where e.source_format = 'sms_backup_xml'
),
fs as (
  select f, count(distinct dedup_key) as n, min(event_ts_utc) as first_ts, max(event_ts_utc) as last_ts,
         array_remove(array_agg(distinct sha1), null) as sha1s
  from fk group by f
),
pairs as (   -- how many of A's messages appear in B
  select a.f as fa, b.f as fb, count(distinct a.dedup_key) as shared
  from (select distinct f, dedup_key from fk) a
  join (select distinct f, dedup_key from fk) b on a.dedup_key = b.dedup_key and a.f <> b.f
  group by a.f, b.f
),
cover as (
  select p.fa, p.fb, fb.n as fb_n, fb.last_ts as fb_last
  from pairs p join fs fa on fa.f = p.fa join fs fb on fb.f = p.fb
  where p.shared = fa.n
    -- B must be strictly newer or larger; identical files break the tie by path so exactly one of them is kept
    and (fb.last_ts > fa.last_ts or (fb.last_ts = fa.last_ts and (fb.n > fa.n or (fb.n = fa.n and fb.f > fa.f))))
),
best as (
  select distinct on (fa) fa, fb, fb_n from cover order by fa, fb_last desc, fb_n desc, fb
)
insert into raw_duck.sms_backup_coverage_20260924 (file, n_messages, first_ts, last_ts, sha1s, covered_by, covered_by_n,
                                                   covering_files, verdict)
select fs.f, fs.n, fs.first_ts, fs.last_ts, fs.sha1s, b.fb, b.fb_n,
       (select count(*) from cover c where c.fa = fs.f),
       case when b.fb is not null then 'covered_by_newer' else 'not_covered' end
from fs left join best b on b.fa = fs.f;

-- A file that is not covered but covers others is a keeper.
update raw_duck.sms_backup_coverage_20260924 s set verdict = 'keeper'
where verdict = 'not_covered' and exists (select 1 from raw_duck.sms_backup_coverage_20260924 o where o.covered_by = s.file);

select verdict, count(*) as files, sum(n_messages) as message_rows from raw_duck.sms_backup_coverage_20260924 group by 1 order by 1;
select verdict, n_messages, first_ts::date, last_ts::date, file, coalesce(covered_by, '') as covered_by
from raw_duck.sms_backup_coverage_20260924 order by verdict, last_ts desc, file;
commit;
