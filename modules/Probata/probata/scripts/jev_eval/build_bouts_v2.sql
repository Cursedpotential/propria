-- Bout pilot v2: the 2024 texts from Katrina's phone, cut into bouts within each day.
-- v2 (2026-09-24 03:25): sender comes from WHO THE MESSAGE WAS ADDRESSED TO, not the sender field. In this backup
-- one rendering stores sender 'Matthew' on messages addressed to Matt's number (9302), i.e. sent BY Katrina.
-- Check: with the sender field, 225/645 bouts were two-sided and the speaker changed 21% of the time; addressed-to
-- gives 353 two-sided bouts and a 40% change rate (15,542 Matt / 7,490 Katrina).
-- Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 2026-09-23 21:06 (judge natural bouts within the day,
-- inside conversation-shaped chunks; catch rapid shifts) and 2026-09-24 03:18 ("try a and a": new bout after
-- 30 minutes of silence; tone marks the shifts inside a bout).
-- Source: catalog raw_duck.comm_events_20260918, conversation 8102959302 (her phones; 'owner' = Katrina,
-- owner-confirmed 2026-09-23 14:23). Read-only. Sender/direction are computed here, never by a model.
-- Days are America/Detroit. Duplicate renderings of one message (same side, same second, same text) collapse.
-- Output: one JSON object per bout with its messages in order.

with k as (
  select e.*,
    case when right(regexp_replace(coalesce(recipients, ''), '\D', '', 'g'), 10) = '8102689630'
              or coalesce(recipients, '') = 'owner' then 'Matt'        -- addressed to her: Matt sent it
         when right(regexp_replace(coalesce(recipients, ''), '\D', '', 'g'), 10) = '8102959302'
              or coalesce(recipients, '') in ('Matthew', 'Matt Salem') then 'Katrina'  -- addressed to Matt: she sent it
    end as who
  from raw_duck.comm_events_20260918 e
  where e.source_format = 'sms_backup_xml' and e.conversation_id = '8102959302'
    and e.event_kind is distinct from 'call' and e.event_ts_utc is not null
),
d as (
  select distinct on (who, date_trunc('second', event_ts_utc), md5(coalesce(body, ''))) *
  from k where who is not null
  order by who, date_trunc('second', event_ts_utc), md5(coalesce(body, '')), n_sources desc, dedup_key
),
n as (
  select d.*, (event_ts_utc at time zone 'America/Detroit') as local_ts,
         (event_ts_utc at time zone 'America/Detroit')::date as day
  from d where extract(year from event_ts_utc at time zone 'America/Detroit') = 2024
),
g as (
  select n.*,
    case when lag(event_ts_utc) over w is null or lag(day) over w <> day
              or event_ts_utc - lag(event_ts_utc) over w > interval '30 minutes' then 1 else 0 end as brk
  from n window w as (order by event_ts_utc, dedup_key)
),
b as (select g.*, sum(brk) over (order by event_ts_utc, dedup_key) as bout_no from g)
select row_to_json(x) from (
  select 'c2024-b' || lpad(bout_no::text, 4, '0') as bout_id, min(day) as day,
         min(local_ts) as start_local, max(local_ts) as end_local, count(*) as n_messages,
         count(*) filter (where who = 'Matt') as n_matt, count(*) filter (where who = 'Katrina') as n_katrina,
         json_agg(json_build_object(
           'i', 0, 'msg_id', dedup_key, 'ts_local', to_char(local_ts, 'HH24:MI'), 'who', who,
           'text', coalesce(nullif(trim(body), ''), '[attachment or empty]'),
           'attachment', coalesce(attachments, '') not in ('', '[]')) order by event_ts_utc, dedup_key) as messages,
         'bouts-v2-30min-detroit-addressed-to' as bout_rules
  from b group by bout_no order by bout_no
) x;
