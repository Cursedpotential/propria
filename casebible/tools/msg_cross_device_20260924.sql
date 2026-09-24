-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Her phone vs Matt's side, 2023-24 (owner 09:17: "find the overlap between Katrina's phone and my phone, and the
-- messages that don't exist between the two of us in her phone that she deleted. And which ones they are").
-- READ-ONLY on every source table; writes two report tables. Nothing is merged, deduplicated or deleted: a message
-- found on both devices is CORROBORATION (owner 09:28-09:36) and is recorded as a link between the two copies.
--
-- Sides
--   her phone  Katrina's SMS Backup & Restore file sms-20250218025955.xml (acquired third-party), rows from the 09-18
--              discovery index (comm_events_20260918 + comm_event_provenance_20260918) on threads with any of Matt's
--              numbers (raw_duck.msg_identity_20260924, confirmed + candidate). Speaker: the speaker table
--              (msg_norm_20260924, source sms_her_phone) where built, else the device's own sent/received flag
--              (sent = Katrina). MMS with neither stay unassigned and are counted, never guessed.
--   his side   extraction attempt :'attempt' (msg_extract_rows_20260924) on threads with any of Katrina's numbers,
--              custodian Matt or unresolved. Speaker: sent/placed = Matt, received = Katrina. Copies of the same export
--              (e.g. "index copy.html" and the renamed html) are grouped by content_key + dup_occurrence for the
--              comparison, and every copy stays cited.
-- Match (basis recorded on every link): same speaker, same text (lower-cased, whitespace collapsed), and |Δt| <= 5 min.
--   Her times are UTC to the millisecond. The iMessage export writes local wall-clock minutes with no zone; they are
--   read as America/Detroit (DST-aware), so Δt includes up to 59 s of truncation plus delivery delay.
-- Gaps (only inside her backup's coverage window for his threads, first to last message on her phone, because
-- before/after that window her phone holds nothing to compare):
--   side_missing = 'her_phone'  on his side, not on her phone. category:
--       katrina_sent   she sent it (it reached his phone) but her phone no longer has it: deleted from her phone
--       matt_sent      he sent it; not on her phone: deleted by her, OR never delivered (e.g. while she had him
--                      blocked), OR sent from a line whose thread is not in her backup. Not proof of deletion alone.
--       no_text        attachment-only / empty on his side; text matching cannot decide, listed separately
--   side_missing = 'his_side'   on her phone, not in his extracted copies (his export may be incomplete or he deleted).
-- Known limit: a repeated short text ("ok") inside the same 5 minutes counts as present if any copy matches.

create table if not exists raw_duck.msg_corroboration_20260924 (
  attempt_id      text not null,
  her_dedup_key   text not null,          -- comm_events_20260918 row on her phone
  his_content_key text not null,          -- his message (all copies share it with the same dup_occurrence)
  his_occurrence  int not null,
  speaker         text not null,
  her_ts_utc      timestamptz,
  his_ts_utc      timestamptz,            -- local minute read as America/Detroit where the source has no zone
  delta_seconds   int,
  best            boolean not null,       -- mutual nearest pair
  basis           text not null,
  primary key (attempt_id, her_dedup_key, his_content_key, his_occurrence)
);

create table if not exists raw_duck.msg_cross_device_gaps_20260924 (
  attempt_id     text not null,
  side_missing   text not null,           -- her_phone | his_side
  category       text not null,
  speaker        text,
  ts_utc         timestamptz,
  ts_basis       text,                    -- utc_known | local_minute_read_as_America/Detroit
  body           text,
  cited_files    text[] not null,         -- current B2 keys (salem-data/...) of every copy on the side that HAS it
  ref_key        text not null            -- her dedup_key, or his content_key#occurrence
);

begin;
delete from raw_duck.msg_corroboration_20260924 where attempt_id = :'attempt';
delete from raw_duck.msg_cross_device_gaps_20260924 where attempt_id = :'attempt';

create temp table her on commit drop as
with mattnums as (select distinct identifier from raw_duck.msg_identity_20260924 where person = 'Matt' and kind = 'phone'),
f as (
  select distinct p.dedup_key, coalesce(mv.new_key, p.vault_key) as cur_key
  from raw_duck.comm_event_provenance_20260918 p
  left join raw_duck.vault_moves_20260924 mv on mv.old_key = p.vault_key
  where p.vault_key = 'consignatio/vault/v1/Evidence/Call data/SMS Backup & Restore Data/sms-20250218025955.xml'
)
select e.dedup_key, e.event_ts_utc as ts, e.body,
       lower(regexp_replace(btrim(coalesce(e.body, '')), '\s+', ' ', 'g')) as bn,
       coalesce(n.speaker, case e.direction when 'sent' then 'Katrina' when 'received' then 'Matt' end) as speaker,
       array_agg(distinct f.cur_key) as files
from f
join raw_duck.comm_events_20260918 e on e.dedup_key = f.dedup_key
left join raw_duck.msg_norm_20260924 n on n.dedup_key = e.dedup_key and n.source = 'sms_her_phone'
where e.event_kind = 'message' and raw_duck.norm_phone(e.counterparty_phone) in (select identifier from mattnums)
  and (n.dedup_key is null or n.is_representative)
group by 1, 2, 3, 4, 5;

create temp table his on commit drop as
with katnums as (select distinct identifier from raw_duck.msg_identity_20260924 where person = 'Katrina' and kind = 'phone')
select r.content_key, r.dup_occurrence,
       min(coalesce(r.event_ts_utc, r.sort_ts at time zone 'America/Detroit')) as ts,
       min(case when r.event_ts_utc is not null then 'utc_known' else 'local_minute_read_as_America/Detroit' end) as ts_basis,
       min(r.body) as body,
       lower(regexp_replace(btrim(coalesce(min(r.body), '')), '\s+', ' ', 'g')) as bn,
       min(case when r.direction in ('sent', 'placed') then 'Matt' when r.direction = 'received' then 'Katrina' end) as speaker,
       array_agg(distinct 'salem-data/' || r.vault_key) as files
from raw_duck.msg_extract_rows_20260924 r
where r.attempt_id = :'attempt' and r.event_kind = 'message'
  and r.counterparty_phone in (select identifier from katnums)
  and coalesce(r.custodian, 'Matt') = 'Matt'
group by 1, 2;

insert into raw_duck.msg_corroboration_20260924
select :'attempt', h.dedup_key, m.content_key, m.dup_occurrence, h.speaker, h.ts, m.ts,
       extract(epoch from (h.ts - m.ts))::int,
       row_number() over (partition by h.dedup_key order by abs(extract(epoch from (h.ts - m.ts)))) = 1
         and row_number() over (partition by m.content_key, m.dup_occurrence order by abs(extract(epoch from (h.ts - m.ts)))) = 1,
       'same speaker + same text + |dt|<=5min; his time basis ' || m.ts_basis
from her h join his m on m.speaker = h.speaker and m.bn = h.bn and h.bn <> ''
 and h.ts between m.ts - interval '5 minutes' and m.ts + interval '5 minutes'
on conflict do nothing;

insert into raw_duck.msg_cross_device_gaps_20260924
select :'attempt', 'her_phone',
       case when m.bn = '' then 'no_text' when m.speaker = 'Katrina' then 'katrina_sent' else 'matt_sent' end,
       m.speaker, m.ts, m.ts_basis, m.body, m.files, m.content_key || '#' || m.dup_occurrence
from his m, (select min(ts) a, max(ts) b from her) w
where m.ts between w.a and w.b
  and not exists (select 1 from raw_duck.msg_corroboration_20260924 c
                  where c.attempt_id = :'attempt' and c.his_content_key = m.content_key and c.his_occurrence = m.dup_occurrence);

insert into raw_duck.msg_cross_device_gaps_20260924
select :'attempt', 'his_side',
       case when h.bn = '' then 'no_text' when h.speaker is null then 'speaker_unassigned'
            when h.speaker = 'Katrina' then 'katrina_sent' else 'matt_sent' end,
       h.speaker, h.ts, 'utc_known', h.body, (select array_agg('salem-data/' || x) from unnest(h.files) x), h.dedup_key
from her h, (select min(ts) a, max(ts) b from his) w
where h.ts between w.a and w.b
  and not exists (select 1 from raw_duck.msg_corroboration_20260924 c
                  where c.attempt_id = :'attempt' and c.her_dedup_key = h.dedup_key);

select 'her phone messages (his threads)' as what, count(*) as n, min(ts)::date as first, max(ts)::date as last from her
union all select 'his side messages (her threads)', count(*), min(ts)::date, max(ts)::date from his
union all select 'corroborated pairs (best)', count(*), null, null from raw_duck.msg_corroboration_20260924 where attempt_id = :'attempt' and best;
select side_missing, category, count(*) as n, min(ts_utc)::date as first, max(ts_utc)::date as last
from raw_duck.msg_cross_device_gaps_20260924 where attempt_id = :'attempt' group by 1, 2 order by 1, 2;
commit;
