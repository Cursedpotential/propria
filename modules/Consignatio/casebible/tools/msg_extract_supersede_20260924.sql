-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Mark an extraction attempt superseded and take its rows out of the catalog staging tables, so a defective attempt
-- never becomes the working copy (owner rule: test data must never become canonical). The attempt's proposal bundle
-- in devbox (proposal.duckdb + manifest.json) is NOT touched: it stays as the record of what that attempt produced,
-- and the attempts row keeps its manifest plus the reason.
-- Run with: psql -v old=<attempt_id> -v new=<attempt_id> -v note='<why>' -f -
alter table raw_duck.msg_extract_attempts_20260924 add column if not exists superseded_by text;
alter table raw_duck.msg_extract_attempts_20260924 add column if not exists note text;
begin;
update raw_duck.msg_extract_attempts_20260924 set superseded_by = :'new', note = :'note' where attempt_id = :'old';
delete from raw_duck.msg_corroboration_20260924 where attempt_id = :'old';
delete from raw_duck.msg_cross_device_gaps_20260924 where attempt_id = :'old';
delete from raw_duck.msg_extract_warnings_20260924 where attempt_id = :'old';
delete from raw_duck.msg_extract_lineage_20260924 where attempt_id = :'old';
delete from raw_duck.msg_extract_rows_20260924 where attempt_id = :'old';
select attempt_id, superseded_by, note, manifest->'relations'->'proposed_records'->>'rows' as bundle_rows
from raw_duck.msg_extract_attempts_20260924 order by loaded_at;
commit;
