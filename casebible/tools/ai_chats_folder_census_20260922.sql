-- Byline: Claude Code · Fable 5.1 · 2026-09-22
-- Read-only. Owner 2026-09-22 18:53: "a whole bunch of sidecars in the AI chats folder, but none of
-- the original AI chats." Census of every catalog path containing "AI_Chats" / "AI Chats":
-- per folder, how many files are sidecars/extractions vs originals.
with f as (
  select catalog_rel_example as p, size, vault_key
  from raw_duck.comm_candidates_20260918
  union
  select catalog_path, size, vault_key from raw_duck.ai_chat_probe_20260918
)
select regexp_replace(p, '/[^/]*$', '') as dir,
       count(*) as files,
       count(*) filter (where p ~* '\.(sidecar|EXTRACTION)\.md$') as sidecar_or_extraction,
       count(*) filter (where p !~* '\.(sidecar|EXTRACTION)\.md$') as other,
       (array_agg(regexp_replace(p, '^.*/', '') order by size desc) filter (where p !~* '\.(sidecar|EXTRACTION)\.md$'))[1:3] as other_examples
from f
where p ~* 'ai[_ ]chats'
group by 1
order by files desc
limit 30;
