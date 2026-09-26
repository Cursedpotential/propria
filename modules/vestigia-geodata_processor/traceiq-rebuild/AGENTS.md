# TraceIQ Rebuild — Agent Entry Point

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Codex · GPT-5 · 2026-08-27._

This file governs the independent repository at
`E:\AI_Workspace\Projects\traceIQ\traceiq-rebuild`. Before staging or committing, run
`git rev-parse --show-toplevel` from the target file's directory and require that exact root.
Stage only explicit owned files; never use `git add .` or `git add -A`.

## Repository boundary

The parent `traceIQ/` directory is a different, reconciled repository that intentionally ignores
this nested child. Never stage, commit, or otherwise mutate the parent repository while working here. Git
worktrees under `.claude/worktrees/` belong to this repository but are separate working trees;
always verify the active worktree and branch before acting.

## Authority and navigation

- `docs/adr/README.md` indexes accepted architecture decisions.
- `docs/SCHEMA.md` documents the database shape; numbered files under `db/migrations/` are the
  executable schema history.
- `docs/VALIDATION_REPORT.md` and `docs/VALIDATION_REPORT_data.md` record bounded validation.
- `ops/` contains ingestion, provider, and validation operations.
- `ui/` contains the Next.js workspace UI and has its own `AGENTS.md`; closest instructions win.
- `reports/README.md` indexes the reporting surface.

Read `AGENT_MEMORY.md` after this file. Treat tool-generated memories as context only, never as
authority over current source, ADRs, data custody, or Git state.

## Safety

- Preserve unrelated and untracked work; never reset, clean, stash, or overwrite it.
- Never delete evidence, data, or repository content. If an approved repository file must be removed,
  move it to this repository's `to_be_deleted/` boundary; only the owner permanently deletes there.
  Stop and ask before any evidence/data move or migration.
- Do not place secrets, credentials, private evidence content, or copied evidence text in source,
  instructions, memory routers, logs, commits, or reports.
- Verification claims must identify whether they are static, local, integration, or live.
