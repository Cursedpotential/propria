# Vestigia (formerly TraceIQ) — Agent Entry Point

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Rewritten for the monorepo layout; the earlier
> independent-repository text (Codex · GPT-5, 2026-08-27/29) is in git history._

Vestigia is the geodata product of Propria (D-140, 2026-09-05: "the Geo space"; old name TraceIQ,
both names valid in recall stores per D-142; canon `modules/Probata/probata/docs/NAMING.md`).

## Where things are

- This folder lives in the Propria monorepo at `modules/vestigia-geodata_processor/`, imported by
  `git subtree` on 2026-09-26. The commit boundary is the monorepo root: `git rev-parse
  --show-toplevel` must print `E:/AI_Workspace/Projects/Propria`. Stage by explicit path only.
- `traceiq-rebuild/` is the live application (database schema, ops scripts, UI, reports). It is
  tracked here like any other folder; read its `AGENTS.md` before working in it.
- The numbered folders (`00_Documentation` … `05_Installers_Zips`, `Utilities/`) are the legacy
  TraceIQ pipeline and tools: reference material, not the current build.
- The current plan: `traceiq-rebuild/docs/plans/2026-10-02-continuation-plan.md`.
- `REPOSITORY_RECONCILIATION.md` records the 2026-08-29 two-repository reconciliation. It is
  history; the two repositories no longer exist as such.

## Data that never enters git

Location evidence stays on disk and ignored: `Timeline.json*`, `raw_api_responses/`,
`TraceIQ_Backups/`, `TraceIQ_Evidence/`, `traaceiq_mess/`. Do not open, summarize or quote their
contents unless the owner places them in scope. `00_Documentation/Secrets_Work_Area/` is sensitive.

## Rules

- Closest `AGENTS.md` wins for subtree-specific instructions.
- No secrets, credentials or evidence content in instructions, memory routers, commits or reports.
- Never hard-delete owner data; approved removals go to `to_be_deleted/`, and only the owner
  deletes there.
