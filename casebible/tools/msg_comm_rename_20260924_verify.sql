-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Checks after msg_comm_rename_20260924.sql: every rename recorded, no chat_ object left, record_kind covers every row,
-- and the views that read the renamed tables still resolve.
select kind, count(*) from raw_duck.catalog_renames_20260924 group by 1 order by 1;
select coalesce(record_kind, '(unclassified)') as record_kind, count(*) from raw_duck.comm_events_20260918 group by 1 order by 1;
select count(*) as msg_events from raw_duck.msg_events_20260918;
select count(*) as ai_chat_events from raw_duck.ai_chat_events_20260918;
select count(*) as timeline_rows from raw_duck.timeline_20260918;
select count(*) as cited_observations from raw_duck.msg_bout_observations_cited_20260924;
select count(*) as msg_bouts, count(*) filter (where exists (select 1 from raw_duck.msg_bout_labels_20260924 l where l.bout_id = b.bout_id)) as labelled
  from raw_duck.msg_bouts_20260924 b;
select 'LEFT OVER: ' || relname from pg_class where relnamespace = 'raw_duck'::regnamespace and relname ~ '^chat_'
union all select 'LEFT OVER: ' || conname from pg_constraint where connamespace = 'raw_duck'::regnamespace and conname ~ '^chat_';
select has_table_privilege('metabase_ro', 'raw_duck.msg_events_20260918', 'select') as ro_msg_events,
       has_table_privilege('metabase_ro', 'raw_duck.msg_bouts_20260924', 'select') as ro_msg_bouts;
