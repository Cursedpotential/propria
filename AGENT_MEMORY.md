---
scope: E:/AI_Workspace/Projects/Propria
status: current
verified_at: 2026-09-20
superseded_by: null
authority:
  - Projects/Propria/AGENTS.md
  - Projects/Propria/docs/monorepo-migration-manifest.json
  - Projects/Propria/modules/Probata/probata/AGENTS.md
  - Projects/Propria/modules/Consignatio/AGENTS.md
watches:
  - Projects/Propria/docs/monorepo-migration-manifest.json
  - Projects/Propria/docs/docstore-source-registry.json
contains_secrets: false
---

# Propria Memory Router

## Current checkout routing — verified 2026-09-20

This dated routing supersedes historical placement text below. Propria owns the
root governance, `modules/FL-MCP/`, and the preserved shared `scripts/` and
`plugins/` source. The other product repositories retain independent histories:

| Product | Current path | Commit boundary |
|---|---|---|
| Indicia Probata | `modules/Probata/probata/` | Independent repository |
| Consignatio / Intake | `modules/Consignatio/` | Independent repository |
| Advocatio legal workdesk | `modules/Legal-desktop/` | Independent repository; `master` |
| Vestigia | `modules/vestigia-geodata_processor/` | Independent outer repository |
| TraceIQ Rebuild | `modules/vestigia-geodata_processor/traceiq-rebuild/` | Independent nested repository; `master` |
| Family Court Workbench | `modules/FL-MCP/` | Propria root repository |
| claude-context code-search MCP (private fork of zilliztech/claude-context, added 2026-09-26) | `modules/claude-context/` | Independent repository; `master`, remote `Cursedpotential/claude-context`; notes in `propria/README.md` |

Read the selected module's local instructions and verify its Git root before
staging. The existing `docs/` junctions and source registry remain the Docstore
routing surface. `docs/.docstore/` contains private credentials and database
state, never publishable source.

Root `scripts/` and `plugins/` preserve moved source, including edits. This is a
source preservation cutover only: Probata's deployment-owned copies remain in
its repository, and may contain newer integrated work. Do not overwrite either
copy or repoint deployment/installed plugins until their differences and build
contexts are reconciled. See `docs/ROOT-SOURCE-RECONCILIATION-2026-09-20.md`.


<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 2 | Platform: Codex / win32 | Changes: record monorepo-root transition | Context: explicit owner correction -->

## Historical repository topology decision — 2026-09-12

Propria is now the intended monorepo root. The target top-level source layout is
`projects/` plus `resources/`; runtime/data material stays outside versioned source.
~~Consignatio remains the canonical independent Git repository at `Consignatio/`;
the tracked `projects/consignatio/` tree is a duplicate imported overlay awaiting
reconciliation. Probata and other nested Git roots retain their current boundaries
until an independently verified import is recorded in the migration manifest.~~ **Corrected 2026-09-18 (Claude Code · Opus 5; owner: Propria is the project, no separate `projects/` home):** the 09-12 `projects/consignatio` overlay is quarantined in `Consignatio/to_be_deleted/2026-09-17-projects-consignatio-overlay-from-2026-09-12/`; Consignatio stays at `Propria/Consignatio/` (a same-day move into `projects/` was reversed).

This decision supersedes the prior statement that Propria is only a router. It
does not erase the old repository histories or authorize a dirty in-place move.

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

## Historical transitional routing — verified 2026-09-13

This section supersedes historical paths and representation claims below.

- Vault and Intake: `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md`.
  This independent Git repository is canonical. ~~`projects/consignatio/` is a
  duplicate overlay pending reconciliation and must not be treated as authority.~~ **Corrected 2026-09-18 (Claude Code · Opus 5; owner: Propria is the project, no separate `projects/` home):** the 09-12 `projects/consignatio` overlay is quarantined in `Consignatio/to_be_deleted/2026-09-17-projects-consignatio-overlay-from-2026-09-12/`; Consignatio stays at `Propria/Consignatio/` (a same-day move into `projects/` was reversed).
- Probata: `Probata/probata/AGENTS.md`, `Probata/probata/AGENT_MEMORY.md`.
- Advocatio legal workdesk: Legal-desktop/AGENTS.md and Legal-desktop/AGENT_MEMORY.md.
  Owner-designated independent canonical repository; original material is resources/build-kit/.
- Other Probata modules: descend through Probata's own current router.
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
| Strategy, legal research, drafting, review, release preparation — **advocatio** | `Legal-desktop/` | independent Git root at Legal-desktop/ |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | independent canonical Git root at `Consignatio/` (~~duplicate overlay at `projects/consignatio/`~~ quarantined 2026-09-18) |
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
