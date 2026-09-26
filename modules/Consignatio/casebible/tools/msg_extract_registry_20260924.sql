-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Conversation extractor driven by a registry: add a conversation row, rerun this file, and every message in it gets a
-- normalized speaker, its duplicate renderings collapsed, and day bouts (the parent chunks), for all years.
-- Owner 07:21: "record the means in which you did it so that all of the other conversations ... can utilize that same
-- process as a programmatic parser or extractor." Owner 07:18-07:20: build it into the catalog; fill the platform
-- tables from the catalog later; don't throw away work we'd do again.
--
-- THE METHOD (the same for every conversation)
-- 1. Match: catalog messages (raw_duck.comm_events_20260918) belong to a registered conversation by source_format plus
--    any of conversation_id / conversation_title / counterparty number (last 10 digits). Calls are excluded.
-- 2. Speaker, by rule, never by a model:
--    addressed_to  the sender field in SMS Backup & Restore exports is unreliable (one rendering stores the contact name
--                  on messages sent TO that contact), so the speaker comes from who the message was ADDRESSED TO:
--                  addressed to one of Katrina's identities -> Matt sent it; addressed to one of Matt's -> Katrina.
--    sender_name   exports with trustworthy sender names (Facebook): the sender name decides.
--    Anything that matches neither side stays unassigned and is counted, never guessed.
-- 3. Duplicate renderings: messages with the same conversation, speaker, second and text are one message; the copy
--    seen in the most sources represents it (ties: lowest dedup_key).
-- 4. Bouts: within one America/Detroit day, a new bout after more than 30 minutes of silence (owner 03:18). Bout ids are
--    <prefix><year>-b<nnnn>, numbered within the conversation and year, so existing ids stay stable.
-- Outputs (catalog, staging for the platform tables, see Probata docs/handoffs/JEV-EVAL-OPTIONS-2026-09-23.md
-- "Propagation register"): msg_norm_20260924, msg_bouts_20260924, msg_bout_messages_20260924.
-- Labelled bouts are never deleted (labels FK is RESTRICT); bouts and bridge rows are upserted.

create table if not exists raw_duck.msg_conversation_registry_20260924 (
  conv_key            text primary key,         -- also the `source` value in the output tables
  description         text not null,
  source_format       text not null,
  conversation_id     text,
  conversation_title  text,
  counterparty_digits text,
  speaker_rule        text not null check (speaker_rule in ('addressed_to', 'sender_name')),
  matt_ids            text[] not null,          -- addressed_to: Katrina's identities (addressed to her = Matt sent it);
                                                -- sender_name: Matt's sender names
  katrina_ids         text[] not null,          -- addressed_to: Matt's identities; sender_name: Katrina's sender names
  custody_party       text not null check (custody_party in ('first_party', 'third_party_acquired')),
  bout_prefix         text not null unique,
  notes               text
);

insert into raw_duck.msg_conversation_registry_20260924 values
 ('sms_her_phone', 'Texts from Katrina''s phones (SMS Backup & Restore, vault sms-20250218025955.xml)',
  'sms_backup_xml', '8102959302', null, null, 'addressed_to',
  array['8102689630', 'owner'], array['8102959302', 'Matthew', 'Matt Salem'], 'third_party_acquired', 'c',
  'Her device, so acquired third-party (owner 07:16). 3,696 of her 2024 messages carry sender ''Matthew''.'),
 ('fb_messenger', 'Facebook Messenger thread ''Katrina Kinzel'' (Matt''s own export)',
  'fb_messenger_json', null, 'Katrina Kinzel', null, 'sender_name',
  array['Matt Salem'], array['Katrina Kinzel'], 'first_party', 'f', null),
 ('sms_9303', 'Texts with Katrina at 810-295-9303 (hers), on Matt''s phone backup',
  'sms_backup_xml', null, null, '8102959303', 'addressed_to',
  array['8102959303', 'Katrina Kinzel', 'Katrina'], array['8102959302', 'owner'], 'first_party', 's',
  'Owner 2026-09-23 14:23: 9302 is Matt''s, 9303 is hers. Group messages (e.g. ''Mike J, Kailyn'') stay unassigned.'),
 ('sms_3592', 'Texts with Katrina at 810-353-3592 (hers), on Matt''s phone backup',
  'sms_backup_xml', null, null, '8103533592', 'addressed_to',
  array['8103533592', 'Katrina', 'Katrina Kinzel'], array['8103535467', 'owner'], 'first_party', 'd',
  'Owner 2026-09-24 07:23: 810-353-5467 was Matt''s number, so messages addressed to it are Katrina''s.')
on conflict (conv_key) do update set description = excluded.description, source_format = excluded.source_format,
  conversation_id = excluded.conversation_id, conversation_title = excluded.conversation_title,
  counterparty_digits = excluded.counterparty_digits, speaker_rule = excluded.speaker_rule, matt_ids = excluded.matt_ids,
  katrina_ids = excluded.katrina_ids, custody_party = excluded.custody_party, bout_prefix = excluded.bout_prefix,
  notes = excluded.notes;

alter table raw_duck.msg_norm_20260924 add column if not exists conv_key text;

begin;
delete from raw_duck.msg_norm_20260924 n
 using raw_duck.msg_conversation_registry_20260924 r where n.source = r.conv_key;

with k as (
  select e.dedup_key, e.event_ts_utc, e.body, e.n_sources, r.conv_key, r.custody_party, r.speaker_rule,
    case r.speaker_rule
      when 'addressed_to' then
        case when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = any (r.matt_ids)
                  or coalesce(e.recipients, '') = any (r.matt_ids) then 'Matt'
             when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = any (r.katrina_ids)
                  or coalesce(e.recipients, '') = any (r.katrina_ids) then 'Katrina' end
      when 'sender_name' then
        case when e.sender = any (r.matt_ids) then 'Matt' when e.sender = any (r.katrina_ids) then 'Katrina' end
    end as speaker
  from raw_duck.comm_events_20260918 e
  join raw_duck.msg_conversation_registry_20260924 r
    on e.source_format = r.source_format
   and (r.conversation_id is null or e.conversation_id = r.conversation_id)
   and (r.conversation_title is null or e.conversation_title = r.conversation_title)
   and (r.counterparty_digits is null
        or right(regexp_replace(coalesce(e.counterparty_phone, ''), '\D', '', 'g'), 10) = r.counterparty_digits)
  where e.event_kind is distinct from 'call' and e.event_ts_utc is not null
),
r as (
  select k.*, first_value(dedup_key) over (
           partition by conv_key, speaker, date_trunc('second', event_ts_utc), md5(coalesce(body, ''))
           order by n_sources desc, dedup_key) as representative_key
  from k where speaker is not null
)
insert into raw_duck.msg_norm_20260924 (dedup_key, source, conv_key, custody_party, speaker, event_ts_utc,
                                                representative_key, is_representative, rule)
select dedup_key, conv_key, conv_key, custody_party, speaker, event_ts_utc, representative_key,
       dedup_key = representative_key, conv_key || ':' || speaker_rule || '-v1'
from r
on conflict (dedup_key) do nothing;

-- Bouts over the representative messages, per conversation and Detroit year.
with n as (
  select m.*, (m.event_ts_utc at time zone 'America/Detroit') as local_ts,
         (m.event_ts_utc at time zone 'America/Detroit')::date as day,
         extract(year from m.event_ts_utc at time zone 'America/Detroit')::int as yr,
         e.body, r.bout_prefix
  from raw_duck.msg_norm_20260924 m
  join raw_duck.msg_conversation_registry_20260924 r on r.conv_key = m.conv_key
  join raw_duck.comm_events_20260918 e on e.dedup_key = m.dedup_key
  where m.is_representative
),
g as (
  select n.*, case when lag(event_ts_utc) over w is null or lag(day) over w <> day
                        or event_ts_utc - lag(event_ts_utc) over w > interval '30 minutes' then 1 else 0 end as brk
  from n window w as (partition by conv_key, yr order by event_ts_utc, dedup_key)
),
b as (select g.*, sum(brk) over (partition by conv_key, yr order by event_ts_utc, dedup_key) as bout_no from g),
m as (
  select b.*, bout_prefix || yr || '-b' || lpad(bout_no::text, 4, '0') as bout_id,
         (row_number() over (partition by conv_key, yr, bout_no order by event_ts_utc, dedup_key) - 1)::int as ordinal
  from b
),
up as (
  insert into raw_duck.msg_bouts_20260924 (bout_id, source, conversation_key, day, start_local, end_local, n_messages,
                                             n_matt, n_katrina, n_chars, bout_rules, custody_party, build_script)
  select bout_id, min(conv_key), min(conv_key), min(day), min(local_ts), max(local_ts), count(*),
         count(*) filter (where speaker = 'Matt'), count(*) filter (where speaker = 'Katrina'),
         sum(length(coalesce(body, ''))), 'registry-30min-detroit-v1', min(custody_party),
         'Consignatio/casebible/tools/msg_extract_registry_20260924.sql'
  from m group by bout_id
  on conflict (bout_id) do nothing
  returning bout_id
)
insert into raw_duck.msg_bout_messages_20260924 (bout_id, ordinal, dedup_key, who, ts_utc)
select m.bout_id, m.ordinal, m.dedup_key, m.speaker, m.event_ts_utc from m
on conflict (bout_id, ordinal) do nothing;

select r.conv_key, count(n.*) as renderings, count(*) filter (where n.is_representative) as messages,
       count(*) filter (where n.is_representative and n.speaker = 'Matt') as matt,
       count(*) filter (where n.is_representative and n.speaker = 'Katrina') as katrina,
       min(n.event_ts_utc)::date as first, max(n.event_ts_utc)::date as last
from raw_duck.msg_conversation_registry_20260924 r left join raw_duck.msg_norm_20260924 n on n.conv_key = r.conv_key
group by 1 order by 1;
select source, count(*) as bouts, sum(n_messages) as messages from raw_duck.msg_bouts_20260924 group by 1 order by 1;
select r.conv_key, count(*) as unassigned_not_loaded
from raw_duck.comm_events_20260918 e
join raw_duck.msg_conversation_registry_20260924 r
  on e.source_format = r.source_format
 and (r.conversation_id is null or e.conversation_id = r.conversation_id)
 and (r.conversation_title is null or e.conversation_title = r.conversation_title)
 and (r.counterparty_digits is null
      or right(regexp_replace(coalesce(e.counterparty_phone, ''), '\D', '', 'g'), 10) = r.counterparty_digits)
where e.event_kind is distinct from 'call' and e.event_ts_utc is not null
  and not exists (select 1 from raw_duck.msg_norm_20260924 n where n.dedup_key = e.dedup_key)
group by 1 order by 1;
commit;
