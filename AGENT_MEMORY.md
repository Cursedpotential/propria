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
Consignatio remains the canonical independent Git repository at `Consignatio/`;
the tracked `projects/consignatio/` tree is a duplicate imported overlay awaiting
reconciliation. Probata and other nested Git roots retain their current boundaries
until an independently verified import is recorded in the migration manifest.

This decision supersedes the prior statement that Propria is only a router. It
does not erase the old repository histories or authorize a dirty in-place move.

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

## Transitional routing — verified 2026-09-13

This section supersedes historical paths and representation claims below.

- Vault and Intake: `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md`.
  This independent Git repository is canonical. `projects/consignatio/` is a
  duplicate overlay pending reconciliation and must not be treated as authority.
- Probata: `Probata/probata/AGENTS.md`, `Probata/probata/AGENT_MEMORY.md`.
- Legal/other Probata modules: descend through Probata's own current router.
- Root governance and migration changes belong to the Propria root repository.
- Follow each lane's exact manifest state. Imported lanes commit from this root;
  source lanes remain in their child repository; a pending cutover keeps its
  protected source available for reconciliation.

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
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | independent canonical Git root at `Consignatio/`; do not author against the duplicate `projects/consignatio/` overlay |
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
