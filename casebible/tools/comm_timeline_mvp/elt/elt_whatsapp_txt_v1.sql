-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- elt_whatsapp_txt_v1 — WhatsApp "Export chat" text (iOS style "[M/D/YY, h:mm:ss AM] Name: text", Android style
-- "M/D/YY, h:mm AM - Name: text"; continuation lines belong to the message above) -> event contract v2, DuckDB only.
-- Added 2026-09-24 (owner 09:47: "if you try the toolkit and there's no tool, then we have to find one and add it"):
-- the 09-18 runner had no WhatsApp reader, so "WhatsApp Chat - Katrina Kinzel/_chat.txt" (Nov 2024 on) was skipped.
-- Timestamps are phone-local with no zone -> local_unknown_tz, sort_ts only. WhatsApp exports carry no phone numbers.
-- Speaker: the export is named for the other party ("WhatsApp Chat - <Name>", folder or file); lines from that name
-- are the other party, every other name is the device holder ('owner' = the custodian), e.g. his profile "Salem".
-- WhatsApp's own notices (encryption banner, omitted media, deleted-message stubs) are kept verbatim; omitted media is
-- also recorded in attachments.
with src as (
  select replace(replace(replace(content, chr(13), ''), chr(8206), ''), chr(8239), ' ') as content,
         trim(coalesce(nullif(regexp_extract('{{SRC}}', 'WhatsApp Chat (?:with |- )([^/]+?)(?:/[^/]*|\.txt)$', 1), ''),
                       regexp_extract('{{SRC}}', '([^/]+)\.[A-Za-z]+$', 1))) as other
  from read_text('{{SRC}}')
),
lines as (
  select generate_subscripts(l, 1) as ln, unnest(l) as line, other
  from (select string_split(content, chr(10)) as l, other from src)
),
marked as (
  select *,
    coalesce(nullif(regexp_extract(line, '^\[([0-9]{1,2}/[0-9]{1,2}/[0-9]{2,4}, [0-9]{1,2}:[0-9]{2}(?::[0-9]{2})? ?[APap]?[Mm]?)\] ', 1), ''),
             nullif(regexp_extract(line, '^([0-9]{1,2}/[0-9]{1,2}/[0-9]{2,4}, [0-9]{1,2}:[0-9]{2}(?::[0-9]{2})? ?[APap]?[Mm]?) - ', 1), '')) as ts_raw,
    coalesce(nullif(regexp_extract(line, '^\[[^\]]+\] ([^:]{1,80}): ', 1), ''),
             nullif(regexp_extract(line, '^[0-9/]+, [^-]+ - ([^:]{1,80}): ', 1), '')) as who,
    coalesce(nullif(regexp_extract(line, '^\[[^\]]+\] [^:]{1,80}: (.*)$', 1), ''),
             regexp_extract(line, '^[0-9/]+, [^-]+ - [^:]{1,80}: (.*)$', 1)) as first_text
  from lines
),
grouped as (
  select *, sum(case when ts_raw is not null and who is not null then 1 else 0 end)
              over (order by ln rows unbounded preceding) as msg_no
  from marked
),
msgs as (
  select msg_no, min(other) as other, min(ln) as ln,
    max(ts_raw) filter (where ts_raw is not null and who is not null) as ts_raw,
    max(who) filter (where ts_raw is not null and who is not null) as who,
    trim(string_agg(case when ts_raw is not null and who is not null then first_text else line end, chr(10) order by ln)) as body
  from grouped where msg_no > 0 group by msg_no
),
k as (
  select *, lower(trim(who)) <> lower(trim(other)) as is_me,
    coalesce(try_strptime(upper(ts_raw), '%m/%d/%y, %I:%M:%S %p'), try_strptime(upper(ts_raw), '%m/%d/%y, %I:%M %p'),
             try_strptime(upper(ts_raw), '%m/%d/%Y, %I:%M:%S %p'), try_strptime(upper(ts_raw), '%m/%d/%Y, %I:%M %p'),
             try_strptime(ts_raw, '%m/%d/%y, %H:%M:%S'), try_strptime(ts_raw, '%m/%d/%y, %H:%M'),
             try_strptime(ts_raw, '%d/%m/%Y, %H:%M:%S'), try_strptime(ts_raw, '%d/%m/%Y, %H:%M')) as ts
  from msgs
)
select
  row_number() over (order by ln) - 1 as record_index,
  null::timestamptz as event_ts_utc,
  ts as sort_ts,
  ts_raw as ts_original,
  'line_prefix(local,no_tz)' as ts_field,
  case when ts is null then 'unparsed' else 'local_unknown_tz' end as tz_status,
  case when regexp_matches(body, '^(Messages and calls are end-to-end encrypted|This message was deleted|You deleted this message|Missed (voice|video) call|(Voice|Video) call(,|$))')
       then 'notice' else 'message' end as event_kind,
  'whatsapp:' || lower(other) as conversation_id, other as conversation_title,
  ['owner', other] as participants,
  case when is_me then 'owner' else who end as sender,
  case when is_me then [other] else ['owner'] end as recipients,
  case when is_me then 'sent' else 'received' end as direction,
  null::varchar as counterparty_phone,
  case when not is_me then who else null end as contact_name,
  body,
  nullif(regexp_extract(body, '((?:image|video|audio|sticker|GIF|document) omitted|<attached: [^>]+>)', 1), '') as attachments,
  null::varchar as member_path,
  null::varchar as owner_line
from k;
