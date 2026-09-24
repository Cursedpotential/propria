-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Chat bouts as permanent catalog tables (owner 2026-09-24 06:35: write these chunks to a table permanently, so later
-- parsing, chunking and normalization can reference them; do as many jobs at once as possible).
-- A bout = the messages of one conversation within one America/Detroit day with no silence over 30 minutes
-- (owner 03:18, "try a and a"). Bouts are the parent unit: search chunks are message-aligned windows inside a bout,
-- and a hit opens its bout. The bridge table ties every bout to its catalog messages (comm_events_20260918.dedup_key).
-- Sources today (2024): the texts from Katrina's phone (SMS backup conversation 8102959302; sender taken from the
-- addressed-to number, see Probata scripts/jev_eval/build_bouts_v2.sql) and the Facebook Messenger thread
-- 'Katrina Kinzel'. Bout ids match the JSONL files the Opus passes read (c2024-b####, f2024-b####).
-- Rebuild is idempotent for a source: its rows are deleted and rebuilt. Labels from any pass land in chat_bout_labels;
-- their foreign key is ON DELETE RESTRICT, so a rebuild over labelled bouts fails instead of wiping labels (amended
-- 2026-09-24 07:18, see msg_bouts_custody_party_20260924.sql, which also adds custody_party).

create table if not exists raw_duck.msg_bouts_20260924 (
  bout_id          text primary key,
  source           text not null,          -- sms_her_phone | fb_messenger
  conversation_key text not null,
  day              date not null,
  start_local      timestamp not null,
  end_local        timestamp not null,
  n_messages       int not null,
  n_matt           int not null,
  n_katrina        int not null,
  n_chars          int not null,
  bout_rules       text not null,
  custody_party    text,                   -- third_party_acquired (her device) | first_party (Matt's own export)
  built_at         timestamptz not null default now(),
  build_script     text not null default 'Consignatio/casebible/tools/msg_bouts_20260924.sql'
);

create table if not exists raw_duck.msg_bout_messages_20260924 (
  bout_id    text not null references raw_duck.msg_bouts_20260924(bout_id) on delete cascade,
  ordinal    int not null,                  -- the message index `i` the Opus passes cite
  dedup_key  text not null,                 -- raw_duck.comm_events_20260918.dedup_key
  who        text not null,                 -- Matt | Katrina
  ts_utc     timestamptz not null,
  primary key (bout_id, ordinal)
);
create index if not exists chat_bout_messages_20260924_key on raw_duck.msg_bout_messages_20260924 (dedup_key);

create table if not exists raw_duck.msg_bout_labels_20260924 (
  bout_id        text not null references raw_duck.msg_bouts_20260924(bout_id) on delete restrict,
  pass           text not null,             -- e.g. bout-tone-v1, bout-discover-v1
  model          text not null,
  prompt_sha256  text not null,
  labelled_at    timestamptz,
  ok             boolean not null,
  output         jsonb,                     -- stretches, shifts, patterns, new_categories, summary
  raw_ref        text not null,             -- devbox persist/jev-eval/raw/<pass dir>/<bout_id>.json
  loaded_at      timestamptz not null default now(),
  primary key (bout_id, pass)
);

begin;
delete from raw_duck.msg_bouts_20260924 where source in ('sms_her_phone', 'fb_messenger') and day between '2024-01-01' and '2024-12-31';

with k as (
  select e.dedup_key, e.event_ts_utc, e.body, e.n_sources, 'sms_her_phone'::text as source, e.conversation_id as conversation_key,
    case when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = '8102689630'
              or coalesce(e.recipients, '') = 'owner' then 'Matt'
         when right(regexp_replace(coalesce(e.recipients, ''), '\D', '', 'g'), 10) = '8102959302'
              or coalesce(e.recipients, '') in ('Matthew', 'Matt Salem') then 'Katrina' end as who,
    'c2024' as prefix, 'bouts-v2-30min-detroit-addressed-to' as rules
  from raw_duck.comm_events_20260918 e
  where e.source_format = 'sms_backup_xml' and e.conversation_id = '8102959302'
    and e.event_kind is distinct from 'call' and e.event_ts_utc is not null
  union all
  select e.dedup_key, e.event_ts_utc, e.body, e.n_sources, 'fb_messenger', 'fb:Katrina Kinzel',
    case when e.sender = 'Matt Salem' then 'Matt' when e.sender = 'Katrina Kinzel' then 'Katrina' end,
    'f2024', 'bouts-fb-30min-detroit'
  from raw_duck.comm_events_20260918 e
  where e.source_format = 'fb_messenger_json' and e.conversation_title = 'Katrina Kinzel' and e.event_ts_utc is not null
),
d as (
  select distinct on (source, who, date_trunc('second', event_ts_utc), md5(coalesce(body, ''))) *
  from k where who is not null
  order by source, who, date_trunc('second', event_ts_utc), md5(coalesce(body, '')), n_sources desc, dedup_key
),
n as (
  select d.*, (event_ts_utc at time zone 'America/Detroit') as local_ts, (event_ts_utc at time zone 'America/Detroit')::date as day
  from d where extract(year from event_ts_utc at time zone 'America/Detroit') = 2024
),
g as (
  select n.*, case when lag(event_ts_utc) over w is null or lag(day) over w <> day
                        or event_ts_utc - lag(event_ts_utc) over w > interval '30 minutes' then 1 else 0 end as brk
  from n window w as (partition by source order by event_ts_utc, dedup_key)
),
b as (
  select g.*, sum(brk) over (partition by source order by event_ts_utc, dedup_key) as bout_no,
         row_number() over (partition by source order by event_ts_utc, dedup_key) as rn
  from g
),
m as (
  select b.*, prefix || '-b' || lpad(bout_no::text, 4, '0') as bout_id,
         (row_number() over (partition by source, bout_no order by event_ts_utc, dedup_key) - 1)::int as ordinal
  from b
),
ins_bouts as (
  insert into raw_duck.msg_bouts_20260924 (bout_id, source, conversation_key, day, start_local, end_local,
                                             n_messages, n_matt, n_katrina, n_chars, bout_rules, custody_party)
  select bout_id, min(source), min(conversation_key), min(day), min(local_ts), max(local_ts), count(*),
         count(*) filter (where who = 'Matt'), count(*) filter (where who = 'Katrina'),
         sum(length(coalesce(body, ''))), min(rules),
         case min(source) when 'sms_her_phone' then 'third_party_acquired' when 'fb_messenger' then 'first_party' end
  from m group by bout_id
  returning bout_id
)
insert into raw_duck.msg_bout_messages_20260924 (bout_id, ordinal, dedup_key, who, ts_utc)
select m.bout_id, m.ordinal, m.dedup_key, m.who, m.event_ts_utc
from m join ins_bouts using (bout_id);

select source, count(*) as bouts, sum(n_messages) as messages from raw_duck.msg_bouts_20260924 group by source order by source;
commit;
