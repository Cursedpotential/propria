-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- elt_imessage_html_v2 — bubble-style iMessage export pages (div.bubble.from-me / .from-them, each followed by
-- div.meta "Sender - YYYY-MM-DD hh:mm AM"; the vault's "imessage export <number> <years>" pages keep the bubbles
-- inside a JavaScript template string, which this reader handles the same way) -> event contract v2.
-- Engine: read_text + RE2. Timestamps are phone-local with no zone -> local_unknown_tz, sort_ts only.
-- Changes from v1 (bugs found 2026-09-24):
--   * v1 made the other party's number from ALL digits of the page title, so "imessage export 8102689630 2023-2024"
--     became 3020232024 and the thread never matched her number. v2: the number written in the title
--     (phone_in_text), else the number on the other side's own bubbles.
--   * numbers normalized with norm_phone (norm_phone_v1.sql); the other side's sender is its normalized number.
--   * owner_line column added (NULL: this export does not say which of his lines sent a message).
with pg as (
  select content,
         trim(html_unescape(regexp_extract(content, '<title>(.*?)</title>', 1))) as conv
  from read_text('{{SRC}}')
),
b as (
  select generate_subscripts(parts, 1) as i, unnest(parts) as chunk, conv
  from (select string_split_regex(content, '<div class=[''"]bubble') as parts, conv from pg)
),
r as (
  select i, chunk, conv,
    chunk like ' from-me%' as is_me,
    trim(html_unescape(regexp_replace(regexp_extract(chunk, '^[^>]*>(.*?)</div>', 1), '<[^>]*>', ' ', 'g'))) as body,
    trim(regexp_extract(chunk, 'class=.meta.>(.*?) - [0-9]{4}-', 1)) as who,
    trim(regexp_extract(chunk, 'class=.meta.>.*? - ([0-9]{4}-[0-9]{2}-[0-9]{2} [^<]*)</div>', 1)) as ts_raw
  from b where i > 1
),
k as (
  select r.*,
    coalesce(phone_in_text(conv),
             max(case when not is_me and regexp_matches(who, '^[+0-9() .-]+$') then norm_phone(who) end) over ()) as other
  from r
)
select
  i - 2 as record_index,
  null::timestamptz as event_ts_utc,
  coalesce(try_strptime(ts_raw, '%Y-%m-%d %I:%M %p'), try_strptime(ts_raw, '%Y-%m-%d %H:%M'),
           try_strptime(ts_raw, '%Y-%m-%d %I:%M:%S %p')) as sort_ts,
  ts_raw as ts_original,
  'div.meta(local,no_tz)' as ts_field,
  case when coalesce(try_strptime(ts_raw, '%Y-%m-%d %I:%M %p'), try_strptime(ts_raw, '%Y-%m-%d %H:%M'),
                     try_strptime(ts_raw, '%Y-%m-%d %I:%M:%S %p')) is null then 'unparsed' else 'local_unknown_tz' end as tz_status,
  'message' as event_kind,
  conv as conversation_id, conv as conversation_title,
  list_filter(['owner', other], x -> x is not null) as participants,
  case when is_me then 'owner' when regexp_matches(who, '^[+0-9() .-]+$') then norm_phone(who) else who end as sender,
  case when is_me then list_filter([other], x -> x is not null) else ['owner'] end as recipients,
  case when is_me then 'sent' else 'received' end as direction,
  other as counterparty_phone,
  case when not is_me and not regexp_matches(who, '^[+0-9() .-]+$') then who end as contact_name,
  body,
  nullif(to_json(regexp_extract_all(chunk, 'class=.attachment-img. src=.([^''"]+).', 1))::varchar, '[]') as attachments,
  null::varchar as member_path,
  null::varchar as owner_line
from k
where ts_raw <> '' or body <> '';
