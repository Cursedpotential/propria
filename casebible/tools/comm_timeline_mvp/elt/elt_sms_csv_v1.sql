-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- elt_sms_csv_v1 — phone SMS export as CSV with the columns address, readable_date, type (Sent/Received/...), body,
-- read, contact_name (e.g. vault SMS-KK-3592/8103533592.csv) -> event contract v2, DuckDB only (read_csv, all text).
-- Added 2026-09-24 (owner 09:47: no tool -> add one). readable_date is the phone's local wall clock with no zone ->
-- local_unknown_tz, sort_ts only. 'owner' = the holder of the phone that made the export (the custodian).
-- Rows whose date does not parse are kept with tz_status 'unparsed' (the runner reports a file where most rows fail
-- as suspect, for the repair toolkit), never dropped.
select
  row_number() over () - 1 as record_index,
  null::timestamptz as event_ts_utc,
  coalesce(try_strptime(readable_date, '%Y-%m-%d %H:%M:%S'), try_strptime(readable_date, '%Y-%m-%d %H:%M'),
           try_strptime(readable_date, '%m/%d/%Y %I:%M:%S %p'), try_strptime(readable_date, '%m/%d/%Y %H:%M')) as sort_ts,
  readable_date as ts_original,
  'readable_date(local,no_tz)' as ts_field,
  case when coalesce(try_strptime(readable_date, '%Y-%m-%d %H:%M:%S'), try_strptime(readable_date, '%Y-%m-%d %H:%M'),
                     try_strptime(readable_date, '%m/%d/%Y %I:%M:%S %p'), try_strptime(readable_date, '%m/%d/%Y %H:%M')) is null
       then 'unparsed' else 'local_unknown_tz' end as tz_status,
  'message' as event_kind,
  coalesce(norm_phone(address), nullif(trim(contact_name), '')) as conversation_id,
  coalesce(nullif(trim(contact_name), ''), address) as conversation_title,
  list_filter(['owner', norm_phone(address)], x -> x is not null) as participants,
  case when lower(type) in ('sent', 'outbox', 'queued', 'failed', '2') then 'owner' else coalesce(norm_phone(address), contact_name) end as sender,
  case when lower(type) in ('sent', 'outbox', 'queued', 'failed', '2') then list_filter([norm_phone(address)], x -> x is not null) else ['owner'] end as recipients,
  case lower(type) when 'sent' then 'sent' when '2' then 'sent' when 'received' then 'received' when 'inbox' then 'received'
       when '1' then 'received' else lower(type) end as direction,
  norm_phone(address) as counterparty_phone,
  nullif(trim(contact_name), '') as contact_name,
  body,
  null::varchar as attachments,
  null::varchar as member_path,
  null::varchar as owner_line
from read_csv('{{SRC}}', header = true, all_varchar = true, quote = '"', escape = '"', strict_mode = false);
