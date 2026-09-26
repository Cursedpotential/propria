-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- elt_google_voice_html_v2 — Google Takeout "Voice/Calls" hChatLog HTML -> event contract v2, DuckDB only.
-- Engine: read_text + RE2 (fixed microformat: div.message / abbr.dt / cite.sender / a.tel / q).
-- Block "text" = SMS threads exported as chat logs; block "call" = placed/received/missed/voicemail pages.
-- Timestamps are ISO-8601 WITH offset in abbr@title -> utc_known.
-- Changes from v1 (bugs found 2026-09-24 while extracting the owner's 2023-24 side):
--   * v1 took counterparty_phone from the tel link of EACH message, so on the owner's own messages ("Me") the
--     "other party" was the owner's own Voice line. v2: the other party is the thread's non-"Me" number(s); a thread
--     with more than one other number is a group (counterparty NULL, everyone in participants).
--   * v1 read the thread name from <title>, which Takeout leaves EMPTY; v2 falls back to the file name
--     ("+18102689630 - Text - 2023-09-02T13_51_32Z").
--   * numbers go through norm_phone (norm_phone_v1.sql), never "last 10 digits".
--   * new column owner_line: the owner's own number on that message (the tel link next to "Me"), so the lines he
--     used (Voice, Fi, burners) are recorded per message. NULL on messages he received.
--   * sender is 'owner' for "Me", else the sender's normalized number (or name when there is no number).
--   * a thread holding only the owner's messages names the other party only in the file name; v2 uses it.

-- @block text
with f as (
  select content,
         coalesce(nullif(trim(html_unescape(regexp_replace(regexp_extract(content, '<title>(.*?)</title>', 1), '\s+', ' ', 'g'))), ''),
                  regexp_extract('{{SRC}}', '([^/]+)\.[A-Za-z]+$', 1)) as conv
  from read_text('{{SRC}}')
),
m as (
  select generate_subscripts(parts, 1) as i, unnest(parts) as chunk, conv,
         -- a thread holding only the owner's own messages names the other party only in its file name
         case when len(others0) = 0 then list_filter([phone_in_text(conv)], x -> x is not null) else others0 end as others
  from (
    select string_split(content, '<div class="message">') as parts, conv,
           -- every tel link in the file that is not labelled "Me" = the other party / parties
           list_distinct(list_filter(list_transform(
             regexp_extract_all(content, 'href="tel:([^"]+)"><(?:span|abbr) class="fn"[^>]*>(?:[^<]*)<'),
             x -> case when regexp_matches(x, 'class="fn"[^>]*>Me<') then null
                       else norm_phone(regexp_extract(x, 'href="tel:([^"]+)"', 1)) end), x -> x is not null)) as others0
    from f
  )
),
r as (
  select *,
    regexp_extract(chunk, '<abbr class="dt" title="([^"]+)"', 1) as ts_raw,
    norm_phone(regexp_extract(chunk, 'href="tel:([^"]+)"', 1)) as tel,
    coalesce(nullif(regexp_extract(chunk, 'class="fn"[^>]*>([^<]*)</', 1), ''), '') = 'Me' as is_me,
    html_unescape(nullif(regexp_extract(chunk, 'class="fn"[^>]*>([^<]*)</', 1), '')) as fn
  from m where chunk like '%<abbr class="dt"%'
)
select
  i - 1 as record_index,
  strptime(ts_raw, '%Y-%m-%dT%H:%M:%S.%f%z') as event_ts_utc,
  null::timestamp as sort_ts,
  ts_raw as ts_original,
  'abbr.dt@title(ISO,offset)' as ts_field,
  case when ts_raw <> '' then 'utc_known' else 'missing' end as tz_status,
  'message' as event_kind,
  conv as conversation_id, conv as conversation_title,
  list_concat(['owner'], others) as participants,
  case when is_me then 'owner' else coalesce(tel, fn) end as sender,
  case when is_me then others else ['owner'] end as recipients,
  case when is_me then 'sent' else 'received' end as direction,
  case when len(others) = 1 then others[1] end as counterparty_phone,
  case when not is_me then fn end as contact_name,
  trim(html_unescape(regexp_replace(regexp_extract(chunk, '<q>(.*?)</q>', 1), '<[^>]*>', ' ', 'g'))) as body,
  nullif(to_json(regexp_extract_all(chunk, '<img src="([^"]+)"', 1))::varchar, '[]') as attachments,
  null::varchar as member_path,
  case when is_me then tel end as owner_line
from r;

-- @block call
with f as (
  select content,
         coalesce(nullif(trim(html_unescape(regexp_replace(regexp_extract(content, '<title>(.*?)</title>', 1), '\s+', ' ', 'g'))), ''),
                  regexp_extract('{{SRC}}', '([^/]+)\.[A-Za-z]+$', 1)) as conv,
         list_distinct(list_filter(list_transform(
           regexp_extract_all(content, 'href="tel:([^"]+)"><(?:span|abbr) class="fn"[^>]*>(?:[^<]*)<'),
           x -> case when regexp_matches(x, 'class="fn"[^>]*>Me<') then null
                     else norm_phone(regexp_extract(x, 'href="tel:([^"]+)"', 1)) end), x -> x is not null)) as others0,
         list_distinct(list_filter(list_transform(
           regexp_extract_all(content, 'href="tel:([^"]+)"><(?:span|abbr) class="fn"[^>]*>Me<'),
           x -> norm_phone(regexp_extract(x, 'href="tel:([^"]+)"', 1))), x -> x is not null)) as mine
  from read_text('{{SRC}}')
),
fc as (select *, case when len(others0) = 0 then list_filter([phone_in_text(conv)], x -> x is not null) else others0 end as others
       from f)
select
  0 as record_index,
  strptime(regexp_extract(content, 'class="published" title="([^"]+)"', 1), '%Y-%m-%dT%H:%M:%S.%f%z') as event_ts_utc,
  null::timestamp as sort_ts,
  regexp_extract(content, 'class="published" title="([^"]+)"', 1) as ts_original,
  'abbr.published@title(ISO,offset)' as ts_field,
  case when regexp_extract(content, 'class="published" title="([^"]+)"', 1) <> '' then 'utc_known' else 'missing' end as tz_status,
  'call' as event_kind,
  conv as conversation_id, conv as conversation_title,
  list_concat(['owner'], others) as participants,
  case when lower(regexp_extract(conv, '(Placed|Received|Missed|Voicemail|Recorded)', 1)) = 'placed' then 'owner'
       else coalesce(others[1], html_unescape(nullif(regexp_extract(content, 'class="fn"[^>]*>([^<]*)</', 1), ''))) end as sender,
  null::varchar[] as recipients,
  lower(regexp_extract(conv, '(Placed|Received|Missed|Voicemail|Recorded)', 1)) as direction,
  case when len(others) = 1 then others[1] end as counterparty_phone,
  null::varchar as contact_name,
  trim(html_unescape(regexp_replace(
       coalesce(nullif(regexp_extract(content, 'class="full-text"[^>]*>(.*?)</span>', 1), ''),
                nullif(regexp_extract(content, '<div class="haudio">(.*?)</div>', 1), ''),
                conv), '<[^>]*>', ' ', 'g'))) as body,
  nullif(regexp_extract(content, 'class="duration" title="([^"]+)"', 1), '') as attachments,
  null::varchar as member_path,
  mine[1] as owner_line
from fc
where content not like '%<div class="message">%';
