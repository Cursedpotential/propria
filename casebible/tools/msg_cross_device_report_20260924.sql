-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Readable export of raw_duck.msg_cross_device_gaps_20260924 for the owner: every message on Matt's side that is not on
-- Katrina's phone (and the reverse), one row each, in time order, with the file(s) where it still exists. READ-ONLY.
-- Message text is his evidence: the output goes to the owner, never into git.
-- period: before 2024-06-27 her phone has no thread with him at all (its history starts 2023-12-21 for every other
--         contact), so those rows are "whole conversation absent"; from 2024-06-27 the thread exists and a missing row
--         is an individual message.
-- Run with: psql -v attempt=<his side's attempt_id> -v her_attempt=<her phone's attempt_id> -f -  > report.csv
copy (
  select
    g.side_missing as missing_from,
    case g.category
      when 'katrina_sent' then 'Katrina wrote it'
      when 'matt_sent' then 'Matt wrote it'
      when 'no_text' then 'attachment only'
      when 'other_line_of_hers' then 'to/from her other number (not this phone)'
      else g.category end as what,
    case when g.side_missing = 'her_phone' and g.ts_utc < timestamptz '2024-06-27 00:00 America/Detroit'
         then 'whole conversation absent from her phone before 2024-06-27'
         when g.side_missing = 'her_phone' then 'individual message (thread exists on her phone)'
         else 'on her phone, not in his exported copies' end as period,
    to_char(g.ts_utc at time zone 'America/Detroit', 'YYYY-MM-DD HH24:MI:SS') as local_time_detroit,
    g.ts_basis,
    g.body,
    (select string_agg(left(r.body, 80) || ' @' || to_char(r.event_ts_utc at time zone 'America/Detroit', 'HH24:MI:SS'), ' | '
                       order by abs(extract(epoch from r.event_ts_utc - g.ts_utc)))
       from (select * from raw_duck.msg_extract_rows_20260924 r
             where r.attempt_id = :'her_attempt' and g.side_missing = 'her_phone'
               and r.event_ts_utc between g.ts_utc - interval '3 minutes' and g.ts_utc + interval '3 minutes'
               and r.direction = case when g.category = 'katrina_sent' then 'sent' else 'received' end
             order by abs(extract(epoch from r.event_ts_utc - g.ts_utc)) limit 2) r) as her_phone_nearby,
    array_to_string(g.cited_files, ' | ') as still_exists_in
  from raw_duck.msg_cross_device_gaps_20260924 g
  where g.attempt_id = :'attempt'
  order by g.side_missing, g.ts_utc
) to stdout with (format csv, header true);
