-- Jev Tier-1 eval: 300-message sample of Matt <-> Katrina messages (sample v1, seed 20260923)
-- Byline: Claude Code · Opus 5.5 · 2026-09-23. Handoff: docs/handoffs/HANDOFF-2026-09-23-jev-tier1-eval.md
-- Source of truth: catalog raw_duck.chat_events_20260918 (casebible PG, ovh-files). Read-only.
-- Sender/direction are computed here in SQL, never by a model (handoff rule 6).
--
-- Strata (owner 2026-09-23 14:20-14:23; the 810-869 number is dropped by owner order):
--   A_fb             Facebook Messenger thread "Katrina Kinzel" (2018-08 .. 2025-08)
--   B_sms_2021_22    SMS on the owner's phone with Katrina at 810-295-9303 (hers), 2021-2022
--   C_sms_her_phone  SMS from Katrina's phones (268-9630 era): conversation keyed by the owner's
--                    number 810-295-9302, so sender 'owner' = KATRINA here (owner-confirmed)
--   D_sms_2025_26    SMS with Katrina at 810-353-3592
-- Allocation: A 110, B 80, C 60, D 50 = 300. Each stratum is half outgoing (Matt) and half incoming
-- (Katrina); a third of each half comes from busy days (top 10% daily volume in that stratum) and the
-- rest from quiet days. Years are interleaved within each cell. A short cell is reported, not refilled.
-- Excluded: calls, rows without a timestamp, and bodies shorter than 2 characters (attachment-only).
-- Duplicate renderings of one message (same side, same second, same text) collapse to one row.

with base as (
  select e.*,
    case
      when e.source_format = 'fb_messenger_json' and e.conversation_title = 'Katrina Kinzel' then 'A_fb'
      when e.source_format = 'sms_backup_xml'
           and right(regexp_replace(coalesce(e.counterparty_phone, ''), '\D', '', 'g'), 10) = '8102959303'
           and extract(year from e.event_ts_utc) between 2021 and 2022 then 'B_sms_2021_22'
      when e.source_format = 'sms_backup_xml' and e.conversation_id = '8102959302' then 'C_sms_her_phone'
      when e.source_format = 'sms_backup_xml'
           and right(regexp_replace(coalesce(e.counterparty_phone, ''), '\D', '', 'g'), 10) = '8103533592' then 'D_sms_2025_26'
    end as src
  from raw_duck.chat_events_20260918 e
  where e.event_kind is distinct from 'call' and e.event_ts_utc is not null
),
k as (
  select base.*,
    case
      when src = 'A_fb' and sender = 'Matt Salem' then 'outgoing'
      when src = 'A_fb' and sender = 'Katrina Kinzel' then 'incoming'
      when src in ('B_sms_2021_22', 'D_sms_2025_26') and sender = 'owner' then 'outgoing'
      when src = 'B_sms_2021_22' and sender = 'Katrina Kinzel' then 'incoming'
      when src = 'D_sms_2025_26' and sender in ('Katrina', 'Katrina Kinzel') then 'incoming'
      when src = 'C_sms_her_phone' and sender in ('Matthew', '+18102959302', 'Matt Salem') then 'outgoing'
      when src = 'C_sms_her_phone' and sender = 'owner' then 'incoming'
    end as direction_matt
  from base where src is not null
),
d as (
  select distinct on (src, direction_matt, date_trunc('second', event_ts_utc), md5(coalesce(body, ''))) *
  from k where direction_matt is not null
  order by src, direction_matt, date_trunc('second', event_ts_utc), md5(coalesce(body, '')), n_sources desc, dedup_key
),
n as materialized (
  select d.*, row_number() over (partition by src order by event_ts_utc, dedup_key) as rn,
         (event_ts_utc at time zone 'UTC')::date as day
  from d
),
dayvol as (select src, day, count(*) as c from n group by 1, 2),
busy as (select src, percentile_cont(0.9) within group (order by c) as p90 from dayvol group by 1),
eligible as (
  select n.*, (dv.c >= b.p90) as busy_day, extract(year from n.event_ts_utc)::int as yr
  from n join dayvol dv using (src, day) join busy b using (src)
  where length(trim(coalesce(n.body, ''))) >= 2
),
ranked as (
  select e.*, row_number() over (partition by src, direction_matt, busy_day, yr order by md5(dedup_key || '20260923')) as r_in_year
  from eligible e
),
ordered as (
  select r.*, row_number() over (partition by src, direction_matt, busy_day order by r_in_year, md5(dedup_key || '20260923')) as r_cell
  from ranked r
),
alloc(src, quota) as (values ('A_fb', 110), ('B_sms_2021_22', 80), ('C_sms_her_phone', 60), ('D_sms_2025_26', 50)),
cellq as (
  select a.src, v.dir, w.bz,
         (case when w.bz then round(a.quota / 2.0 / 3) else a.quota / 2 - round(a.quota / 2.0 / 3) end)::int as q
  from alloc a cross join (values ('outgoing'), ('incoming')) v(dir) cross join (values (true), (false)) w(bz)
),
pick as (
  select o.* from ordered o join cellq c on c.src = o.src and c.dir = o.direction_matt and c.bz = o.busy_day
  where o.r_cell <= c.q
),
prov as (
  select distinct on (v.dedup_key) v.dedup_key, v.vault_key, v.sha1, v.catalog_rel, v.member_path, v.record_index
  from raw_duck.chat_event_provenance_20260918 v
  where v.dedup_key in (select dedup_key from pick)
  order by v.dedup_key, (v.sha1 is null), v.vault_key
)
select row_to_json(x) from (
  select p.dedup_key as msg_id, p.src as stratum, p.source_format as platform, p.conversation_id as thread_id,
         p.event_ts_utc as ts_utc, p.tz_status, p.direction_matt as direction,
         case when p.direction_matt = 'outgoing' then 'Matt' else 'Katrina' end as sender_label,
         p.sender as sender_raw, p.body as text, p.yr as year, p.busy_day, p.n_sources,
         pv.vault_key as source_file, pv.sha1 as source_sha1, pv.catalog_rel, pv.member_path, pv.record_index,
         (select coalesce(json_agg(json_build_object(
                    'sender', case when q.direction_matt = 'outgoing' then 'Matt' else 'Katrina' end,
                    'ts', q.event_ts_utc,
                    'text', coalesce(nullif(trim(q.body), ''), '[attachment or empty]')) order by q.rn), '[]'::json)
            from n q where q.src = p.src and q.rn between p.rn - 5 and p.rn - 1) as prior_context,
         'sample-v1-seed-20260923' as sample_version
  from pick p left join prov pv on pv.dedup_key = p.dedup_key
  order by p.src, p.direction_matt, p.busy_day, p.r_cell
) x;
