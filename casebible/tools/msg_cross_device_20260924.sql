-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Her phone vs Matt's side (owner 09:17: "find the overlap between Katrina's phone and my phone, and the messages that
-- don't exist between the two of us in her phone that she deleted. And which ones they are"; 09:18: several numbers
-- and many burners because she kept blocking him; 09:19: 2023-24 first).
-- READ-ONLY on every source table; writes three report tables. Nothing is merged, deduplicated or deleted: a message
-- found on both devices is CORROBORATION (owner 09:28-09:36) and is recorded as a link between the two copies.
--
-- Sides
--   her phone  extraction attempt :'her_attempt' of her SMS Backup & Restore file (custodian Katrina, reader
--              elt_smsbackuprestore_v2, MMS direction from msg_box). ALL her message threads are searched, not only
--              numbers already known to be Matt's, because his burners are mostly unknown.
--   his side   extraction attempt :'attempt' (custodian Matt or unresolved) on threads with any of Katrina's numbers.
--              Copies of the same export are grouped by content_key + dup_occurrence for the comparison; every copy
--              stays cited. Speaker: sent/placed = Matt, received = Katrina.
-- Match (basis on every link): same text (raw_duck.msg_text_norm: case, quotes, attachment marks, whitespace; non-empty), |Δt| <= 5 min, and opposite
--   ends of the same message: his sent <-> her received, his received <-> her sent. On a thread whose number is not
--   yet a known Matt number the text must be at least 20 characters, so "ok" to someone else never counts.
--   Her times are UTC to the millisecond; his are whatever the reader established (Google Voice: stated offsets;
--   iMessage HTML export: UTC minutes, reader v3, cross-checked; unknown zone: read as America/Detroit).
-- Outputs
--   msg_corroboration_20260924      one row per linked pair (best = mutual nearest in time)
--   msg_matt_lines_on_her_phone_20260924   her phone's numbers that carry Matt's own messages, by content
--   msg_cross_device_gaps_20260924  messages on one side and absent on the other:
--     side_missing = 'her_phone': on his side, not on her phone, inside her phone's history window (its first to last
--       message of any thread). category:
--         katrina_sent  she sent it (it reached his phone) but her phone no longer has it: deleted from her phone
--         matt_sent     he sent it; not on her phone: deleted by her, OR never delivered (e.g. while blocked), OR
--                       delivered to a line/thread not in this backup. Not proof of deletion on its own.
--         no_text       attachment-only / empty; text matching cannot decide
--         other_line_of_hers  sent to/from one of her numbers that is not this phone's line (its MMS state the line);
--                       never on this phone, so not a deletion from it
--     side_missing = 'his_side': on her phone (threads with Matt's known or content-proven numbers), not in his
--       extracted copies, inside his side's date range.
-- Known limit: a repeated short text inside the same 5 minutes counts as present if any copy matches.
-- Run with: psql -v attempt=<his side's attempt_id> -v her_attempt=<her phone's attempt_id> -f -

create table if not exists raw_duck.msg_corroboration_20260924 (
  attempt_id      text not null,
  her_dedup_key   text not null,
  his_content_key text not null,
  his_occurrence  int not null,
  speaker         text not null,
  her_number      text,                   -- the thread's number on her phone
  her_ts_utc      timestamptz,
  his_ts_utc      timestamptz,
  delta_seconds   int,
  best            boolean not null,
  basis           text not null,
  primary key (attempt_id, her_dedup_key, his_content_key, his_occurrence)
);
alter table raw_duck.msg_corroboration_20260924 add column if not exists her_number text;

create table if not exists raw_duck.msg_matt_lines_on_her_phone_20260924 (
  attempt_id      text not null,
  her_number      text not null,
  her_contact     text,                   -- the name her phone saved for it
  linked_messages int not null,           -- his messages found on that thread by content (>= 20 chars)
  first_utc       timestamptz,
  last_utc        timestamptz,
  identity_before text,                   -- what raw_duck.msg_identity_20260924 said before this run
  primary key (attempt_id, her_number)
);

create table if not exists raw_duck.msg_cross_device_gaps_20260924 (
  attempt_id     text not null,
  side_missing   text not null,
  category       text not null,
  speaker        text,
  ts_utc         timestamptz,
  ts_basis       text,
  body           text,
  cited_files    text[] not null,
  ref_key        text not null
);

-- One text key for "the same words", used here and reusable for "pull it from every file it was said in" (owner 09:37):
-- lower-cased; curly quotes/apostrophes made straight; the object-replacement mark and attachment file names that some
-- exports (iMessage HTML) write into the text removed; every run of whitespace (newlines included) collapsed to one
-- space, then trimmed. Empty result = attachment-only. (First version trimmed before collapsing, so a trailing newline
-- on one phone's copy, e.g. "I'm sorry 😞\n", kept two identical messages apart.)
create or replace function raw_duck.msg_text_norm(t text) returns text language sql immutable parallel safe as $$
  select btrim(lower(regexp_replace(regexp_replace(regexp_replace(
           translate(coalesce(t, ''), chr(8217) || chr(8216) || chr(8220) || chr(8221), $q$''""$q$),
           chr(65532), '', 'g'),
           '\S+\.(jpe?g|png|gif|heic|heif|mov|mp4|m4a|amr|3gp|vcf|pdf|webp)(\s|$)', ' ', 'gi'), '\s+', ' ', 'g')))
$$;

begin;
delete from raw_duck.msg_corroboration_20260924 where attempt_id = :'attempt';
delete from raw_duck.msg_matt_lines_on_her_phone_20260924 where attempt_id = :'attempt';
delete from raw_duck.msg_cross_device_gaps_20260924 where attempt_id = :'attempt';

create temp table mattnums on commit drop as
  select distinct identifier from raw_duck.msg_identity_20260924 where person = 'Matt' and kind = 'phone';

create temp table her on commit drop as
select r.dedup_key, r.event_ts_utc as ts, r.body, r.direction, r.counterparty_phone as num, r.contact_name,
       raw_duck.msg_text_norm(r.body) as bn,
       case r.direction when 'sent' then 'Katrina' when 'received' then 'Matt' end as speaker_if_matt,
       array['salem-data/' || coalesce(mv.new_key, r.vault_key)] as files
from raw_duck.msg_extract_rows_20260924 r
left join raw_duck.vault_moves_20260924 mv on mv.old_key = r.vault_key
where r.attempt_id = :'her_attempt' and r.custodian = 'Katrina' and r.event_kind = 'message'
  and r.direction in ('sent', 'received');
alter table her add column bh text;
update her set bh = md5(bn);
create index on her (bh);

create temp table his on commit drop as
with katnums as (select distinct identifier from raw_duck.msg_identity_20260924 where person = 'Katrina' and kind = 'phone')
select r.content_key, r.dup_occurrence,
       min(coalesce(r.event_ts_utc, r.sort_ts at time zone 'America/Detroit')) as ts,
       min(case when r.event_ts_utc is not null then r.tz_status else 'local_minute_read_as_America/Detroit' end) as ts_basis,
       min(r.body) as body,
       raw_duck.msg_text_norm(min(r.body)) as bn,
       min(case when r.direction in ('sent', 'placed') then 'Matt' when r.direction = 'received' then 'Katrina' end) as speaker,
       min(r.counterparty_phone) as her_line,
       array_agg(distinct 'salem-data/' || r.vault_key) as files
from raw_duck.msg_extract_rows_20260924 r
where r.attempt_id = :'attempt' and r.event_kind = 'message'
  and r.counterparty_phone in (select identifier from katnums)
  and coalesce(r.custodian, 'Matt') = 'Matt'
group by 1, 2;
alter table his add column bh text;
update his set bh = md5(bn);
create index on his (bh);

insert into raw_duck.msg_corroboration_20260924
  (attempt_id, her_dedup_key, his_content_key, his_occurrence, speaker, her_number, her_ts_utc, his_ts_utc,
   delta_seconds, best, basis)
select :'attempt', h.dedup_key, m.content_key, m.dup_occurrence, m.speaker, h.num, h.ts, m.ts,
       extract(epoch from (h.ts - m.ts))::int,
       row_number() over (partition by h.dedup_key order by abs(extract(epoch from (h.ts - m.ts)))) = 1
         and row_number() over (partition by m.content_key, m.dup_occurrence order by abs(extract(epoch from (h.ts - m.ts)))) = 1,
       'same text + opposite ends + |dt|<=5min; his time basis ' || m.ts_basis
       || case when h.num in (select identifier from mattnums) then '' else '; thread number not yet known as Matt''s (text >= 20 chars)' end
from her h join his m on m.bh = h.bh and h.bn <> ''
 and h.speaker_if_matt = m.speaker
 and h.ts between m.ts - interval '5 minutes' and m.ts + interval '5 minutes'
 and (h.num in (select identifier from mattnums) or length(h.bn) >= 20)
on conflict do nothing;

insert into raw_duck.msg_matt_lines_on_her_phone_20260924
select :'attempt', c.her_number, max(h.contact_name), count(distinct c.her_dedup_key), min(c.her_ts_utc), max(c.her_ts_utc),
       (select string_agg(distinct i.person || ':' || i.status, ',') from raw_duck.msg_identity_20260924 i where i.identifier = c.her_number)
from raw_duck.msg_corroboration_20260924 c
join her h on h.dedup_key = c.her_dedup_key
where c.attempt_id = :'attempt' and c.her_number is not null
group by c.her_number;

-- The line(s) of the phone the backup came from: the device holder's own number stated in its MMS addressing. A message
-- to another of her numbers (e.g. 810-353-3592 from Dec 2024) was never on this phone's line, so it is 'other_line_of_hers',
-- not a deletion.
create temp table her_lines on commit drop as
  select distinct owner_line from raw_duck.msg_extract_rows_20260924
  where attempt_id = :'her_attempt' and owner_line is not null;

-- Matt's threads on her phone = known Matt numbers + numbers that carry his messages by content (>= 3 linked).
create temp table her_matt on commit drop as
select h.* from her h
where h.num in (select identifier from mattnums)
   or h.num in (select her_number from raw_duck.msg_matt_lines_on_her_phone_20260924
                where attempt_id = :'attempt' and linked_messages >= 3);

insert into raw_duck.msg_cross_device_gaps_20260924
select :'attempt', 'her_phone',
       case when m.her_line not in (select owner_line from her_lines) then 'other_line_of_hers'
            when m.bn = '' then 'no_text' when m.speaker = 'Katrina' then 'katrina_sent' else 'matt_sent' end,
       m.speaker, m.ts, m.ts_basis, m.body, m.files, m.content_key || '#' || m.dup_occurrence
from his m, (select min(ts) a, max(ts) b from her) w
where m.ts between w.a and w.b
  and not exists (select 1 from raw_duck.msg_corroboration_20260924 c
                  where c.attempt_id = :'attempt' and c.his_content_key = m.content_key and c.his_occurrence = m.dup_occurrence);

insert into raw_duck.msg_cross_device_gaps_20260924
select :'attempt', 'his_side',
       case when h.bn = '' then 'no_text' when h.speaker_if_matt = 'Katrina' then 'katrina_sent' else 'matt_sent' end,
       h.speaker_if_matt, h.ts, 'utc_known', h.body, h.files, h.dedup_key
from her_matt h, (select min(ts) a, max(ts) b from his) w
where h.ts between w.a and w.b
  and not exists (select 1 from raw_duck.msg_corroboration_20260924 c
                  where c.attempt_id = :'attempt' and c.her_dedup_key = h.dedup_key);

select 'her phone: all messages' as what, count(*) as n, min(ts)::date as first, max(ts)::date as last from her
union all select 'her phone: Matt threads (known + content-proven numbers)', count(*), min(ts)::date, max(ts)::date from her_matt
union all select 'his side: messages with her numbers (copies grouped)', count(*), min(ts)::date, max(ts)::date from his
union all select 'linked pairs (best)', count(*), min(her_ts_utc)::date, max(her_ts_utc)::date
          from raw_duck.msg_corroboration_20260924 where attempt_id = :'attempt' and best;
select her_number, her_contact, linked_messages, first_utc::date, last_utc::date, identity_before
from raw_duck.msg_matt_lines_on_her_phone_20260924 where attempt_id = :'attempt' order by linked_messages desc;
select side_missing, category, count(*) as n, min(ts_utc)::date as first, max(ts_utc)::date as last
from raw_duck.msg_cross_device_gaps_20260924 where attempt_id = :'attempt' group by 1, 2 order by 1, 2;
commit;
