-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Texts deleted from a phone between backups (owner 09:15: "yeah, sure"). READ-ONLY on the evidence; writes one
-- report table. A text counts as deleted when it appears in a backup, and a LATER backup of the same phone, whose
-- date range covers the text's timestamp, no longer has it.
--   same phone  = the later backup shares at least half of the earlier backup's messages
--   last_seen   = the newest backup that still contains the text
--   gone_by     = the earliest later same-phone backup (after last_seen) whose range covers the text but lacks it
-- Each file is cited at its CURRENT B2 location: moved backups resolve through raw_duck.vault_moves_20260924
-- (quarantine), everything else is at its original vault key. Nothing in B2 is read or changed by this script.
-- Sources: comm_events_20260918 / comm_event_provenance_20260918 (renamed from chat_* by another session,
-- 2026-09-24), sms_backup_coverage_20260924, sms_backup_headcheck_20260924.

create table if not exists raw_duck.sms_vanished_messages_20260924 (
  dedup_key          text primary key,
  event_ts_utc       timestamptz,
  counterparty_phone text,
  conversation_title text,
  sender             text,
  recipients         text,
  body               text,
  last_seen_file     text,        -- current B2 key
  last_seen_backup   timestamptz,
  last_seen_quarantined boolean,
  gone_by_file       text,        -- current B2 key
  gone_by_backup     timestamptz,
  gone_by_quarantined boolean
);

begin;
truncate raw_duck.sms_vanished_messages_20260924;
with fk as (
  select distinct coalesce(p.catalog_rel, p.vault_key) as f, p.dedup_key
  from raw_duck.comm_event_provenance_20260918 p
  join raw_duck.comm_events_20260918 e using (dedup_key)
  where e.source_format = 'sms_backup_xml'
),
loc as (
  select c.file as f, c.first_ts, c.n_messages as n, h.backup_date as bd,
         coalesce(m.new_key, h.vault_key) as now_key, m.new_key is not null as quarantined
  from raw_duck.sms_backup_coverage_20260924 c
  join raw_duck.sms_backup_headcheck_20260924 h using (file)
  left join raw_duck.vault_moves_20260924 m on m.old_key = h.vault_key
),
ov as (
  select a.f as fa, b.f as fb, count(*) as shared
  from fk a join fk b on a.dedup_key = b.dedup_key and a.f <> b.f
  group by 1, 2
),
same_phone as (   -- later backups of the same phone
  select ov.fa, ov.fb from ov join loc la on la.f = ov.fa join loc lb on lb.f = ov.fb
  where lb.bd > la.bd and ov.shared >= 0.5 * la.n
),
last_seen as (    -- newest backup still holding each text
  select distinct on (fk.dedup_key) fk.dedup_key, fk.f, l.bd, l.now_key, l.quarantined
  from fk join loc l on l.f = fk.f
  order by fk.dedup_key, l.bd desc, fk.f
),
gone as (
  select distinct on (s.dedup_key) s.dedup_key, s.f as last_f, s.bd as last_bd, s.now_key as last_key,
         s.quarantined as last_q, lb.now_key as gone_key, lb.bd as gone_bd, lb.quarantined as gone_q
  from last_seen s
  join raw_duck.comm_events_20260918 e on e.dedup_key = s.dedup_key
  join same_phone sp on sp.fa = s.f
  join loc lb on lb.f = sp.fb
  where e.event_ts_utc >= lb.first_ts
    and not exists (select 1 from fk x where x.f = sp.fb and x.dedup_key = s.dedup_key)
  order by s.dedup_key, lb.bd
)
insert into raw_duck.sms_vanished_messages_20260924
select g.dedup_key, e.event_ts_utc, e.counterparty_phone, e.conversation_title, e.sender, e.recipients, e.body,
       g.last_key, g.last_bd, g.last_q, g.gone_key, g.gone_bd, g.gone_q
from gone g join raw_duck.comm_events_20260918 e on e.dedup_key = g.dedup_key;

select count(*) as deleted_texts, count(distinct coalesce(conversation_title, counterparty_phone)) as conversations,
       min(event_ts_utc)::date as oldest, max(event_ts_utc)::date as newest
from raw_duck.sms_vanished_messages_20260924;
select coalesce(conversation_title, counterparty_phone) as conversation, right(regexp_replace(coalesce(counterparty_phone, ''), '\D', '', 'g'), 10) as number,
       count(*) as deleted, min(event_ts_utc)::date as first, max(event_ts_utc)::date as last
from raw_duck.sms_vanished_messages_20260924 group by 1, 2 order by 3 desc limit 25;
select last_seen_backup::date as last_seen, gone_by_backup::date as gone_by, count(*) as texts
from raw_duck.sms_vanished_messages_20260924 group by 1, 2 order by 1, 2;
commit;
