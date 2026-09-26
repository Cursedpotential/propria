-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Upsert Opus bout-pass results into raw_duck.msg_bout_labels_20260924 (see msg_bouts_20260924.sql).
-- Feed: CSV from Probata scripts/jev_eval/export_bout_labels.py, copied into the stage table first:
--   \copy raw_duck.msg_bout_labels_stage_20260924 from stdin with (format csv)
-- Re-runnable: a newer result for the same (bout_id, pass) replaces the older one.

create table if not exists raw_duck.msg_bout_labels_stage_20260924 (
  bout_id text, pass text, model text, prompt_sha256 text, labelled_at text, ok text, output text, raw_ref text
);

begin;
insert into raw_duck.msg_bout_labels_20260924 (bout_id, pass, model, prompt_sha256, labelled_at, ok, output, raw_ref)
select s.bout_id, s.pass, s.model, s.prompt_sha256, nullif(s.labelled_at, '')::timestamptz, s.ok::boolean,
       nullif(s.output, '')::jsonb, s.raw_ref
from raw_duck.msg_bout_labels_stage_20260924 s
join raw_duck.msg_bouts_20260924 b using (bout_id)
on conflict (bout_id, pass) do update
  set model = excluded.model, prompt_sha256 = excluded.prompt_sha256, labelled_at = excluded.labelled_at,
      ok = excluded.ok, output = excluded.output, raw_ref = excluded.raw_ref, loaded_at = now();
select 'stage rows' as what, count(*) from raw_duck.msg_bout_labels_stage_20260924
union all select 'stage rows with no matching bout', count(*) from raw_duck.msg_bout_labels_stage_20260924 s
  where not exists (select 1 from raw_duck.msg_bouts_20260924 b where b.bout_id = s.bout_id);
truncate raw_duck.msg_bout_labels_stage_20260924;
select pass, count(*) as bouts, count(*) filter (where ok) as ok from raw_duck.msg_bout_labels_20260924 group by pass order by pass;
commit;
