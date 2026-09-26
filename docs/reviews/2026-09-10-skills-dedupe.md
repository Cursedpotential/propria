# Skills dedupe — ~/.agents excluded, everything Claude uses brought into ~/.claude

> _Byline: Claude Code · Opus 5 · 2026-09-10. Owner order 11:16: "fix all the duplicate skills, inside and outside of plugins … exclude .agents … bring it over to .claude."_

## Result

| Measure | Before | After |
|---|---|---|
| Skills Claude Code loads (user `~/.claude/skills` + probata project + enabled plugins) | 534 | 514 |
| Duplicated skill names | 25 | 7 (all different content, listed below) |
| Symlinks/junctions in `~/.claude/skills` pointing into `~/.agents` | 26 | 0 |

## What was done

1. **Links replaced with real folders.** 3 working links (`hf-cli`, `lancedb`, `zilliz`) were replaced by real copies of their `~/.agents/skills` targets.
2. **Dangling links removed from view.** 23 junctions pointed at `~/.agents/skills/<name>` folders that no longer exist, so they loaded nothing (antigravity-bridge, claim-chart, clarity-gate, context7, coolify, database-schema-designer, deeppapernote, defuddle, draft, flashcards, graph-database-expert, i-have-adhd-and-47-tabs, legal-writing, n8n-agents-official, n8n-node-configuration-official, n8n-workflow-patterns, opencode-delegate, openrouter-typescript-sdk, repo-analyzer, surrealdb-expert, surrealdb-memory, using-n8n-skills-official, vercel-connect). `context7` and `defuddle` still exist as plugins. All 26 links are kept in `~/.claude/backups/skills-links-hold-20260910/` with `log.txt`.
3. **Skills that lived only in `~/.agents` brought over.** 12 with no plugin twin were copied into `~/.claude/skills`, file counts verified equal: cf-add-integration, clean-architecture, clean-code, cocoindex, developing-llamaindex-systems, duck-time-travel, mineru, nemo-retriever, sequential-thinking, smart-explore, textual-tui, voyage-cli. Not copied, because a loaded plugin already ships them: behavioral-pattern-analyzer (family-court-toolkit), tailscale (ops), weaviate and weaviate-cookbooks (weaviate plugin, byte-identical).
4. **Clear-cut duplicates resolved (newest wins).**
   - `prompt-lookup`, `skill-lookup`: user standalone copies moved to `~/.claude/backups/skills-dupes-hold-20260910/`; the prompts.chat plugin copies stay loaded. Note: the held `skill-lookup` copy (2026-06-10) is slightly newer than the plugin's (2026-05-17).
   - `cocoindex`: the newest copy (downloaded from cocoindex.io 2026-09-10 into the probata project) is now the single user-level skill; the older user copy (2026-09-03) is in the same hold, and the project copy is in `probata/to_be_deleted/2026-09-10-skills-dupes/`.
5. **`~/.agents/skills` left intact.** OpenCode reads it through `~/.opencode/skills -> ../.agents/skills`, so nothing there was moved. Claude Code does not scan `~/.agents/skills` on its own; it only saw those skills through the 26 links.

Nothing was deleted.

## Duplicates that remain — different content, need an owner ruling

| Name | Copies | Difference |
|---|---|---|
| `sequential-thinking` | user (owner-installed 2026-09-08, named in global CLAUDE.md) · plugin `specialized-tools` (2026-05-17) | Keep user copy; disabling the plugin also removes `mermaidjs-v11` |
| `handoff` | user HANDOFF v2 file skill (2026-07-31) · plugin `docstore` (2026-09-09, writes to the SurrealDB docs store) | docstore is the newer convention; hold the user copy? |
| `read-memories` | user dual-source version (2026-09-07) · plugin `duckdb-skills` upstream (2026-08-23) | Owner-patched vs upstream; different behaviour |
| `recall` | user unified recall (2026-09-07) · plugin `recall-skill` vault recall (2026-07-28) | Two different tools sharing a name |
| `memory` | plugin `claude-never-forgets` · plugin `docstore` | Different skills; namespaced, no collision in use |
| `query` | plugin `duckdb-skills` · plugin `docstore` | Different skills; namespaced |
| `hyperfocus` | 3 copies inside the one `hyperfocus` plugin (root, `skills/`, `plugins/hyperfocus/skills/`) | Plugin packaging; fix upstream or in the plugin cache |

Concurrent change seen during the run: another session wrote `surrealdb-*` and `weaviate*` skill folders into `probata/.claude/skills/` at about 11:25; they were not touched.
