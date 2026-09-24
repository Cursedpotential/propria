-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- Storybook fixture for the Workbench mood strip (modules/workbench/web/src/platform-ui/conversations/
-- __fixtures__/her-phone-2024-tone.json): every 2024 bout from her phone with its bout-tone-v1 reading, no message text.
-- Run on ovh-files: docker exec -i <pg container> psql -U postgres -d casebible -At < mood_strip_fixture.sql
-- Sender names come from each message's `who`; the component reads `senders` as a name -> count map and hardcodes nobody.
select json_agg(json_build_object(
  'id', b.bout_id,
  'start', to_char(b.start_local, 'YYYY-MM-DD"T"HH24:MI'),
  'end', to_char(b.end_local, 'YYYY-MM-DD"T"HH24:MI'),
  'messages', b.n_messages,
  'senders', (select jsonb_object_agg(t.who, t.n)
              from (select m.who, count(*) as n from raw_duck.msg_bout_messages_20260924 m
                    where m.bout_id = b.bout_id group by m.who) t),
  'stretches', (select json_agg(json_build_array(s->>'tone', (s->>'to_i')::int - (s->>'from_i')::int + 1, s->>'driver')
                                order by (s->>'from_i')::int)
                from jsonb_array_elements(l.output->'stretches') s),
  'shifts', jsonb_array_length(l.output->'shifts'),
  'abrupt', (select count(*) from jsonb_array_elements(l.output->'shifts') x where x->>'speed' = 'abrupt')
) order by b.start_local)
from raw_duck.msg_bouts_20260924 b
join raw_duck.msg_bout_labels_20260924 l on l.bout_id = b.bout_id and l.pass = 'bout-tone-v1'
where b.source = 'sms_her_phone';
