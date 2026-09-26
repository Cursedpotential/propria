# Decision: every decision starts at the router root and runs the cross-platform preflight; decisions, docs and ADRs are recorded in the Probata Docstore

> _Byline: Claude Code · Fable 5.1 · 2026-09-14 08:57 EDT — owner rulings 08:36–08:57 EDT in the Fable supervisor session (intake-4c), recorded per the owner's order "decisions docs and adrs always go to probata docstore"._

- Status: **Accepted** (owner statements, 2026-09-14)
- Scope: every Propria project and every working directory, on both Claude and Codex
- Extends: the 2026-09-12 universal-Docstore decision in `Propria/AGENTS.md`
- Router text: `E:\AI_Workspace\AGENTS.md` § "Decision preflight", pointers in `Projects\AGENTS.md` and `Propria\AGENTS.md`

## Owner statements (verbatim)

- 08:36 "Ensure when making decisions that you read the recent Codex conversations over the last week." · "and memsearch cnf and remember" · "/smart-explore not smart search" · "nav up to main repo" · 08:45 "move up to mono root propria".
- 08:55 "I have started every chat the same way: check Codex, use these tools. And a pointer maybe that regardless of what the project is and where they are working, all decisions need to be checked against and started at the router root."
- 08:57 "decisions docs and adrs always go to probata docstore."
- Same rule to Codex, 2026-09-13 05:07–05:10: "dont stop there check the entire week and check codex", "full week both platforms", "NEVER STOP AT THE FIRST RESULT AND CALL IT A DAY".

## Decision

1. A design, dispatch or scope decision starts at the router root `E:\AI_Workspace\AGENTS.md`, whatever the project or cwd, and is checked against every history lane before it is made:
   - Codex: the last week of `C:\Users\matts\.codex\sessions\<yyyy>\<mm>\<dd>\*.jsonl` (owner text = `response_item` → `message` → `role: user` → `input_text`; Codex conclusion = `event_msg` → `task_complete.last_agent_message`; `session_meta.parent_thread_id` marks a sub-agent thread).
   - Claude: `/read-memories` (DuckDB over `~/.claude/projects/*/*.jsonl` and the OpenCode SQLite); `/smart-explore` for code structure.
   - Memory stores: memsearch shared collection `agent_session_memory_nemotron3`, CNF `.cnf/memories/project_memory.json`, `.remember/`, per-project auto-memory.
   - Docstore: `fn::current_decisions` and `fn::docs_search` (both `adr` and `document(doc_type="decision")`); an empty result is a finding.
2. Never stop at the first result. The newest owner-approved statement wins; Codex and Claude history count equally.
3. Propria work starts at the mono root `E:\AI_Workspace\Projects\Propria`.
4. Decisions, docs and ADRs are always recorded in the Probata Docstore through its governed functions (`fn::docs_register` / `fn::docs_new_version` / `fn::decision_amend`), with the source file kept in the owning repository. A file that is not registered is drift.

## Consequences

- Each chat's first action on a decision is the preflight, not a search of its own platform only.
- Supervisors brief agents from the merged lanes (the 2026-09-14 Codex-week extract lives in the supervisor session scratchpad; rebuild from the field map above when needed).
- Finding 2026-09-14: `fn::current_decisions` returned empty for consignatio, probata and propria, so the `adr` table holds no live rows for these projects; decisions exist only as `document(doc_type="decision")` rows.
