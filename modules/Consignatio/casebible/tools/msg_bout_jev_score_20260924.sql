-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Score Jev (pass bout-q-v1) against the Opus reference labels (pass bout-tone-v1) in raw_duck.msg_bout_labels_20260924.
-- READ-ONLY. Owner ~13:15 "send it" (Jev run, 645 bouts / 660 windows, $0.08).
-- Opus main tone = the tone of the stretch covering the most messages; Opus "has X" = any stretch has tone X.
-- Jev main tone = output.main_tone (the piece covering the most messages); Jev yes/no = any piece's noul >= 0.5.
create temp view opus as
select bout_id,
  (select s->>'tone' from jsonb_array_elements(output->'stretches') s
    order by (s->>'to_i')::int - (s->>'from_i')::int desc, (s->>'from_i')::int limit 1) as main,
  (select array_agg(distinct s->>'tone') from jsonb_array_elements(output->'stretches') s) as tones
from raw_duck.msg_bout_labels_20260924 where pass = 'bout-tone-v1' and ok;

create temp view jev as
select bout_id, output->>'main_tone' as main, output
from raw_duck.msg_bout_labels_20260924 where pass = 'bout-q-v1' and ok;

create temp view yesno as
select o.bout_id, q,
  case q when 'any_warm' then o.tones && array['friendly', 'affectionate']
         when 'any_conflict' then o.tones && array['tense', 'hostile']
         when 'any_hostile' then o.tones && array['hostile']
         when 'any_distress' then o.tones && array['distressed']
         when 'any_conciliatory' then o.tones && array['conciliatory']
         when 'tone_changes' then cardinality(o.tones) > 1 end as opus_yes,
  (select bool_or(((w->'answers'->q)->>'noul')::float >= 0.5) from jsonb_array_elements(j.output->'windows') w) as jev_yes
from opus o join jev j using (bout_id),
     unnest(array['any_warm', 'any_conflict', 'any_hostile', 'any_distress', 'any_conciliatory', 'tone_changes']) q;

select 'main tone' as measure, count(*) as bouts, count(*) filter (where o.main = j.main) as agree,
       round(100.0 * count(*) filter (where o.main = j.main) / count(*), 1) as pct, null::bigint as opus_yes,
       null::bigint as jev_yes, null::bigint as both_yes
from opus o join jev j using (bout_id)
union all
select q, count(*), count(*) filter (where opus_yes = jev_yes), round(100.0 * count(*) filter (where opus_yes = jev_yes) / count(*), 1),
       count(*) filter (where opus_yes), count(*) filter (where jev_yes), count(*) filter (where opus_yes and jev_yes)
from yesno group by q
order by 1;

-- Main-tone confusion: rows = Opus's main tone; jev = what Jev said for those bouts, most common first.
select opus_main, sum(c) as bouts, string_agg(jev_main || ' ' || c, ', ' order by c desc) as jev_said
from (select o.main as opus_main, j.main as jev_main, count(*) as c from opus o join jev j using (bout_id) group by 1, 2) x
group by opus_main order by bouts desc;

-- The owner's 5 reviewed bouts (Bout Review page, 2026-09-24): b0142, b0407, b0592, b0007 marked Right; b0274 Missed.
select o.bout_id, o.main as opus_main, j.main as jev_main
from opus o join jev j using (bout_id)
where o.bout_id in ('c2024-b0142', 'c2024-b0407', 'c2024-b0592', 'c2024-b0007', 'c2024-b0274') order by 1;
