-- Byline: Claude Code · Opus 5.5 · 2026-09-24
-- File citations for every normalized message and every Opus observation (owner 07:26: "I want a file name citation
-- on every single record").
-- chat_message_files_20260924: one row per representative message (chat_message_norm_20260924.representative_key)
--   with every source file any of its renderings came from (raw_duck.chat_event_provenance_20260918.catalog_rel,
--   falling back to vault_key), so a merged message cites all the backups and exports that hold it.
-- chat_bout_observations_cited_20260924 (view): each observation in chat_bout_labels_20260924 exploded to its cited
--   messages (bout ordinal -> dedup_key via chat_bout_messages_20260924) with that message's files.
-- Rebuild: truncate and rebuild the table; the view is replaced.

create table if not exists raw_duck.chat_message_files_20260924 (
  representative_key text primary key,
  conv_key           text not null,
  n_files            int not null,
  files              text[] not null,
  sha1s              text[] not null
);

begin;
truncate raw_duck.chat_message_files_20260924;
insert into raw_duck.chat_message_files_20260924 (representative_key, conv_key, n_files, files, sha1s)
select n.representative_key, min(n.conv_key),
       count(distinct coalesce(p.catalog_rel, p.vault_key)),
       array_agg(distinct coalesce(p.catalog_rel, p.vault_key) order by coalesce(p.catalog_rel, p.vault_key)),
       array_remove(array_agg(distinct p.sha1), null)
from raw_duck.chat_message_norm_20260924 n
join raw_duck.chat_event_provenance_20260918 p on p.dedup_key = n.dedup_key
group by n.representative_key;

create or replace view raw_duck.chat_bout_observations_cited_20260924 as
select l.bout_id, l.pass, l.model, obs.ordinality as observation_no,
       obs.o ->> 'category' as category, (obs.o ->> 'from_guide')::boolean as from_guide, obs.o ->> 'who' as who,
       obs.o ->> 'confidence' as confidence, (obs.o ->> 'child_related')::boolean as child_related,
       obs.o ->> 'note' as note, obs.o ->> 'quote' as quote,
       idx.i::int as message_ordinal, bm.dedup_key, f.files, f.n_files
from raw_duck.chat_bout_labels_20260924 l
cross join lateral jsonb_array_elements(coalesce(l.output -> 'patterns', '[]'::jsonb)) with ordinality as obs(o, ordinality)
cross join lateral jsonb_array_elements_text(obs.o -> 'message_is') as idx(i)
left join raw_duck.chat_bout_messages_20260924 bm on bm.bout_id = l.bout_id and bm.ordinal = idx.i::int
left join raw_duck.chat_message_files_20260924 f on f.representative_key = bm.dedup_key;

select conv_key, count(*) as messages, round(avg(n_files), 1) as avg_files, max(n_files) as max_files,
       count(*) filter (where n_files = 0) as uncited
from raw_duck.chat_message_files_20260924 group by 1 order by 1;
select 'normalized messages with no file citation' as check, count(*)
from raw_duck.chat_message_norm_20260924 n
where n.is_representative and not exists (select 1 from raw_duck.chat_message_files_20260924 f where f.representative_key = n.dedup_key);
commit;
