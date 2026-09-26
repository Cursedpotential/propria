-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Normalized speaker and duplicate-rendering map for the Matt <-> Katrina conversations, ALL years, built in the catalog
-- so the platform tables are filled from here later and nothing is re-derived (owner 07:18-07:20: "build it into the
-- catalog and then we can extract it from the catalog into the tables").
-- One row per catalog message (raw_duck.comm_events_20260918.dedup_key):
--   speaker           Matt | Katrina, from the rule for that source (never a model)
--   custody_party     third_party_acquired (texts from Katrina's device) | first_party (Matt's own Facebook export)
--   representative_key the message kept when several renderings of one message exist (same source, speaker, second,
--                      text); every other rendering points at it
-- Rules:
--   sms_her_phone  SMS Backup & Restore, conversation 8102959302 (her phones). Speaker comes from the ADDRESSED-TO
--                  number, not the sender field: one rendering stores sender 'Matthew' on messages she sent to Matt
--                  (3,696 in 2024 were wrongly credited to Matt). Addressed to 8102689630 or 'owner' -> Matt sent it;
--                  addressed to 8102959302, 'Matthew' or 'Matt Salem' -> Katrina sent it. Calls excluded.
--   fb_messenger   Facebook Messenger thread 'Katrina Kinzel'; speaker from the export's sender name.
-- Rebuild: deletes and rebuilds these two sources.

create table if not exists raw_duck.msg_norm_20260924 (
  dedup_key          text primary key,
  source             text not null,
  custody_party      text not null,
  speaker            text not null,
  event_ts_utc       timestamptz not null,
  representative_key text not null,
  is_representative  boolean not null,
  rule               text not null,
  built_at           timestamptz not null default now()
);
create index if not exists chat_message_norm_20260924_rep on raw_duck.msg_norm_20260924 (representative_key);

begin;
delete from raw_duck.msg_norm_20260924 where source in ('sms_her_phone', 'fb_messenger');

with k as (
  select e.dedup_key, e.event_ts_utc, e.body, e.n_sources, 'sms_her_phone'::text as source,
         'third_party_acquired'::text as custody_party,
    case when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = '8102689630'
              or coalesce(e.recipients, '') = 'owner' then 'Matt'
         when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = '8102959302'
              or coalesce(e.recipients, '') in ('Matthew', 'Matt Salem') then 'Katrina' end as speaker,
    'sms-her-phone-addressed-to-v1' as rule
  from raw_duck.comm_events_20260918 e
  where e.source_format = 'sms_backup_xml' and e.conversation_id = '8102959302'
    and e.event_kind is distinct from 'call' and e.event_ts_utc is not null
  union all
  select e.dedup_key, e.event_ts_utc, e.body, e.n_sources, 'fb_messenger', 'first_party',
    case when e.sender = 'Matt Salem' then 'Matt' when e.sender = 'Katrina Kinzel' then 'Katrina' end,
    'fb-sender-name-v1'
  from raw_duck.comm_events_20260918 e
  where e.source_format = 'fb_messenger_json' and e.conversation_title = 'Katrina Kinzel' and e.event_ts_utc is not null
),
r as (
  select k.*, first_value(dedup_key) over (
           partition by source, speaker, date_trunc('second', event_ts_utc), md5(coalesce(body, ''))
           order by n_sources desc, dedup_key) as representative_key
  from k where speaker is not null
)
insert into raw_duck.msg_norm_20260924 (dedup_key, source, custody_party, speaker, event_ts_utc,
                                                representative_key, is_representative, rule)
select dedup_key, source, custody_party, speaker, event_ts_utc, representative_key, dedup_key = representative_key, rule
from r;

select source, extract(year from event_ts_utc at time zone 'America/Detroit')::int as year,
       count(*) as renderings, count(*) filter (where is_representative) as messages,
       count(*) filter (where is_representative and speaker = 'Matt') as matt,
       count(*) filter (where is_representative and speaker = 'Katrina') as katrina
from raw_duck.msg_norm_20260924 group by 1, 2 order by 1, 2;
select 'unassigned speaker (not loaded)' as check, count(*) from raw_duck.comm_events_20260918 e
where ((e.source_format = 'sms_backup_xml' and e.conversation_id = '8102959302' and e.event_kind is distinct from 'call')
    or (e.source_format = 'fb_messenger_json' and e.conversation_title = 'Katrina Kinzel'))
  and not exists (select 1 from raw_duck.msg_norm_20260924 n where n.dedup_key = e.dedup_key);
commit;
