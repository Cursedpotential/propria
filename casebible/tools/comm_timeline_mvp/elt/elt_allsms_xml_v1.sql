-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- elt_allsms_xml_v1 — the "<allsms count=…>" SMS export (one <sms address= time= date= type= body= read=
-- service_center= name= /> per text; SMS only, no MMS) -> event contract v2, DuckDB only (read_text + RE2).
-- Added 2026-09-24 (owner 09:47: no tool -> add one; 13:48: "I should have several XML files from her phone"). Found in
-- vault Takeout/salemnma/Drive/sms_20250218024754.xml: Katrina's phone ("T-Mobile: Hi KATRINA, your bill …"),
-- 15,549 texts, made 12 minutes before her SMS Backup & Restore backup of 2025-02-18.
-- The exporting app does not escape quotes inside bodies, so the file is NOT valid XML and no XML parser can read it.
-- This reader splits on '<sms address="' and takes the attributes by their fixed order; the body is everything between
-- ' body="' and the record's LAST '" read="' (greedy), so an unescaped quote inside a message cannot cut it short.
-- date is epoch milliseconds (UTC). type 1 = received, 2 = sent (Android). 'owner' = the phone's holder (custodian).
-- The file states no line of its own, so owner_line is NULL.
with recs as (
  select generate_subscripts(parts, 1) as i, unnest(parts) as chunk
  from (select string_split(content, '<sms address="') as parts from read_text('{{SRC}}'))
),
p as (
  select i,
    regexp_extract(chunk, '^([^"]*)"', 1) as address,
    regexp_extract(chunk, ' time="([^"]*)"', 1) as time_txt,
    try_cast(regexp_extract(chunk, ' date="([0-9]+)"', 1) as bigint) as date_ms,
    regexp_extract(chunk, ' type="([0-9]+)"', 1) as type,
    regexp_extract(chunk, '(?s) body="(.*)" read="', 1) as body_raw,
    regexp_extract(chunk, '(?s)" read="[^"]*" service_center="[^"]*" name="([^"]*)" */>', 1) as name
  from recs where i > 1
)
select
  i - 2 as record_index,
  case when date_ms > 0 then to_timestamp(date_ms / 1000.0) end as event_ts_utc,
  null::timestamp as sort_ts,
  'date=' || coalesce(date_ms::varchar, '') || '; time=' || coalesce(time_txt, '') as ts_original,
  'date(epoch_ms,UTC)' as ts_field,
  case when date_ms > 0 then 'utc_known' else 'missing' end as tz_status,
  'message' as event_kind,
  case when address like '%~%'
       then array_to_string(list_sort(list_distinct(list_transform(string_split(address, '~'), x -> norm_phone(x)))), ',')
       else coalesce(norm_phone(address), nullif(name, '')) end as conversation_id,
  coalesce(nullif(html_unescape(name), ''), address) as conversation_title,
  list_concat(['owner'], coalesce(list_filter(list_transform(string_split(address, '~'), x -> norm_phone(x)), x -> x is not null), [])) as participants,
  case when type in ('2', '4', '5', '6') then 'owner' else coalesce(norm_phone(address), address) end as sender,
  case when type in ('2', '4', '5', '6') then list_filter([norm_phone(address)], x -> x is not null) else ['owner'] end as recipients,
  case type when '1' then 'received' when '2' then 'sent' when '3' then 'draft' when '4' then 'outbox'
       when '5' then 'failed' when '6' then 'queued' else 'unknown' end as direction,
  case when address like '%~%' then null else norm_phone(address) end as counterparty_phone,
  nullif(html_unescape(name), '') as contact_name,
  html_unescape(body_raw) as body,
  null::varchar as attachments,
  null::varchar as member_path,
  null::varchar as owner_line
from p;
