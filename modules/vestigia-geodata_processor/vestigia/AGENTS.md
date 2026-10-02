# Vestigia application — Agent Entry Point

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Rewritten for the monorepo layout; the earlier
> independent-repository text (Codex · GPT-5, 2026-08-27) is in git history._

Product name per D-140. The folder was `traceiq-rebuild` until 2026-10-02 (owner order); older
records use that path. The database is still named `traceiq` and the variables `TRACEIQ_*`.

## Repository

Part of the Propria monorepo since the 2026-09-26 subtree import. `git rev-parse --show-toplevel`
must print `E:/AI_Workspace/Projects/Propria`. Stage explicit paths only; never `git add -A`.
New worktrees go under `Propria/_worktrees/`.

## Where things are

- **Plan:** `docs/plans/2026-10-02-continuation-plan.md` (phases, open owner decisions V-1…V-8).
- **Decisions:** `docs/adr/README.md`.
- **Database:** `traceiq` on `probata-db` (ovh-files `100.91.190.107:5432`). Shape in
  `docs/SCHEMA.md`; executable history in `db/migrations/` and `db/transformations/`.
- **Owner direction 2026-10-02:** PostgreSQL is the source-of-truth data holder only; all analysis
  moves to SurrealDB on a mirror, once the data is right (see the plan).
- **Credentials:** never in tracked files. Ops scripts read `TRACEIQ_DSN_KV` (via `ops/db_env.py`),
  the UI reads `TRACEIQ_DSN`; values in `~/.secrets/traceiq-db.env`, names in `.env.example`.
- **Ops:** `ops/` (ingest, validation, providers); validation reports `docs/VALIDATION_REPORT*.md`
  and `ops/*REPORT.md`.
- **UI:** `ui/` (Next.js, its own `AGENTS.md` and `README.md`). **Reports:** `reports/README.md`.

Read `AGENT_MEMORY.md` next. Tool memories (`.claude/`, `.remember/`, `.memsearch/`) are context,
never authority over source, ADRs, data custody or git state.

## Safety

- Never delete evidence or data. Approved repository removals go to `to_be_deleted/`; only the
  owner deletes there. Stop and ask before any evidence or data move or migration.
- No secrets, credentials or evidence content in source, instructions, memory, logs, commits or
  reports.
- Every verification claim says whether it is static, local, integration or live.
- Nothing runs on the owner's desktop: services and long jobs belong on the VPSs via Coolify.
