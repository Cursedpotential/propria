---
scope: E:/AI_Workspace/Projects/Propria
status: current
verified_at: 2026-09-13
superseded_by: null
authority:
  - Projects/Propria/AGENTS.md
  - Projects/Propria/docs/monorepo-migration-manifest.json
  - Projects/Propria/Probata/probata/AGENTS.md
  - Projects/Propria/Consignatio/AGENTS.md
watches:
  - Projects/Propria/docs/monorepo-migration-manifest.json
  - Projects/Propria/docs/docstore-source-registry.json
contains_secrets: false
---

# Propria Memory Router

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 2 | Platform: Codex / win32 | Changes: record monorepo-root transition | Context: explicit owner correction -->

## Current repository topology decision — 2026-09-12

Propria is now the intended monorepo root. The target top-level source layout is
`projects/` plus `resources/`; runtime/data material stays outside versioned source.
The present `Consignatio/`, `Probata/`, and nested Git roots are migration sources.
Do not infer that a lane has crossed the boundary until the staged import is
verified and recorded in `docs/monorepo-migration-manifest.json`.

This decision supersedes the prior statement that Propria is only a router. It
does not erase the old repository histories or authorize a dirty in-place move.

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

## Transitional routing — 2026-09-10 paths pending monorepo import

This section supersedes historical paths and representation claims below.

- Vault and Intake: `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md`.
- Probata: `Probata/probata/AGENTS.md`, `Probata/probata/AGENT_MEMORY.md`.
- Legal/other Probata modules: descend through Probata's own current router.
- Root governance and migration changes belong to the Propria root repository.
- Product changes remain in their current child repository until its manifest
  state changes from `source` to `imported`.

Do not start all products or load their memories together. Do not infer current
Git representation from the target layout; check the migration manifest and
`git rev-parse --show-toplevel` before staging.

> _Byline: Codex · GPT-5 · 2026-08-27; repository reconciliation refreshed 2026-08-29._

## Vertical load order

1. Read this workspace `AGENTS.md` and `AGENT_MEMORY.md`.
2. Select exactly one active project for the task.
3. Descend through that project's `AGENTS.md` and `AGENT_MEMORY.md` hierarchy.
4. Load exact-file `.agent-memory/<filename>.md` only for files actually in scope.

This load order describes the configured live workspace. If a child file is absent after cloning the
parent, consult `../REPOSITORY_BOUNDARIES.md`; do not manufacture a replacement or treat an empty
gitlink directory as the product source.

## Project router

| Work | Descend into | Commit from |
|---|---|---|
| Evidence, custody, ingestion, parsing, analysis, platform operations — **Indicia Probata** | `Probata/probata/` | `Probata/probata/` |
| Strategy, legal research, drafting, review, release preparation — **advocatio** | `Probata/probata/modules/advocatio-legal_workbench/` | verify inside that child boundary |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | `Consignatio/` |
| Root governance and migration records | this directory | this Propria Git root with an explicit path allowlist |

Never let an opened chat directory decide the commit root. Before staging, run
`git rev-parse --show-toplevel`, compare it with the target project's declared boundary above, and
stage only explicit paths owned by that project. A parent gitlink changing because a child is dirty
is not authorization to stage or commit the gitlink.

The current transitional representation is recorded in
`docs/monorepo-migration-manifest.json`. Do not use the retired
`the-platform-workspace` tree to resolve current paths.

The Evidence Platform is canonical for evidence. Legal Workspace consumes accepted
`LegalSourcePackage` data and never becomes a second writable evidence store.

> _Naming amendment: Claude Code · Fable 5.1 · 2026-09-06 — product canon D-137..D-150 (`probata/docs/NAMING.md`). Evidence Platform = **Indicia Probata** (`probata`, formerly `Agno-MCP-Platform`); Legal Workspace = **advocatio** (nested at `probata/modules/advocatio/`, formerly `modules/Legal-Workspace/`); TraceIQ = **vestigia** (nested at `probata/modules/vestigia/`, formerly `modules/traceIQ/`). The directory rename landed 2026-09-06; `Agno-MCP-Platform/` is now a junction to `probata/`. Old paths stay as junctions for a week; both names remain valid in recall stores (D-142)._

> _Byline amendment: Claude Code · Fable 5.1 · 2026-09-06 — directory renames D-137..D-142: `Agno-MCP-Platform/` → `probata/` (Indicia Probata), nested `modules/traceIQ/` → `modules/vestigia/`, `modules/Legal-Workspace/` → `modules/advocatio/`. Old paths remain as junctions for one week; recall stores keep both names (D-142)._
