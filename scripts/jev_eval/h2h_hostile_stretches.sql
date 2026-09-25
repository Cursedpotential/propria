-- Byline: Claude Code · Opus 5.5 · 2026-09-25.
-- Longest Opus "hostile" stretch per 2024 bout (pass bout-tone-v1), with who drove it and how much of the bout Opus
-- marked hostile. Input to build_h2h_items.py (owner 00:31: the head-to-head had no clearly hostile stretch in it, "it
-- was easily the worst year of my entire life ... this isn't a mix").
-- Run on ovh-files (catalog PG), output kept on the devbox, never in git:
--   docker exec -i fgz1n7useplhk0t91uk7k1aw psql -U postgres -d casebible -At < h2h_hostile_stretches.sql > hostile.jsonl
select json_build_object('bout_id', bout_id, 'day', day, 'from_i', from_i, 'to_i', to_i, 'n', n, 'driver', driver,
                         'bout_hostile_msgs', bout_hostile, 'bout_msgs', bout_msgs)
from (
  select l.bout_id, b.day,
         (st->>'from_i')::int as from_i, (st->>'to_i')::int as to_i,
         (st->>'to_i')::int - (st->>'from_i')::int + 1 as n,
         st->>'driver' as driver, b.n_messages as bout_msgs,
         sum((st->>'to_i')::int - (st->>'from_i')::int + 1) over (partition by l.bout_id) as bout_hostile,
         row_number() over (partition by l.bout_id
                            order by (st->>'to_i')::int - (st->>'from_i')::int desc, (st->>'from_i')::int) as rn
  from raw_duck.msg_bout_labels_20260924 l
  join raw_duck.msg_bouts_20260924 b using (bout_id)
  cross join lateral jsonb_array_elements(l.output -> 'stretches') st
  where l.pass = 'bout-tone-v1' and l.ok and st->>'tone' = 'hostile' and l.bout_id like 'c2024-%'
) x
where rn = 1
order by bout_hostile desc, bout_id;
