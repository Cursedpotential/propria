---
scope: E:/AI_Workspace/Projects/Propria
status: current
verified_at: 2026-10-17
verification_scope: Repository routing at 68f369e; no desktop or runtime health check
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

## Current checkout routing — single repository since 2026-09-26

Propria is one Git repository. The product and application-fork histories were
imported with `git subtree` on 2026-09-26; their retired child Git directories
are historical preservation copies. Every route below commits at
`E:/AI_Workspace/Projects/Propria`. Read the root `AGENTS.md` current routing
table for the complete list, including SBV, Timesketch, custom modules, Xplorer
and the source-only build crew.

| Product | Current path | Commit boundary |
|---|---|---|
| Indicia Probata | `modules/Probata/probata/` | Propria root repository |
| Consignatio / Intake | `modules/Consignatio/` | Propria root repository |
| Advocatio legal workdesk | `modules/Legal-desktop/` | Propria root repository |
| Vestigia | `modules/vestigia-geodata_processor/` | Propria root repository |
| Vestigia app (formerly TraceIQ Rebuild) | `modules/vestigia-geodata_processor/vestigia/` | Propria root repository |
| Family Court Workbench | `modules/FL-MCP/` | Propria root repository |

Read the selected module's local instructions and verify its Git root before
staging. The existing `docs/` junctions and source registry remain the Docstore
routing surface. `docs/.docstore/` contains private credentials and database
state, never publishable source.

Shared development tools `memsearch`, `claude-context` and `cocoindex-code`
remain outside this monorepo under `E:/AI_Workspace/plugins/forks/`. The root
`AGENTS.md` records deployment source routing and its explicit exceptions.

_Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17. Authority: root
`AGENTS.md` (2026-09-26 import contract) and
`modules/vestigia-geodata_processor/AGENTS.md` (2026-10-02), read at `68f369e`._

## System boundaries

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

## Historical routing — 2026-09-12 through 2026-09-23

The proposed `projects/` / `resources/` layout, duplicate Consignatio overlay,
independent child-repository commit instructions and source-copy-only status
were superseded by the completed 2026-09-26 import. Their original wording and
bylines remain in Git history; dated receipts remain in
`docs/ROOT-SOURCE-RECONCILIATION-2026-09-20.md` and the migration manifest's
`historical_snapshots`. These receipts do not define current checkout or
deployment state. The former `the-platform-workspace` tree is also historical.

> _Byline: Codex · GPT-5 · 2026-08-27; repository reconciliation refreshed 2026-08-29._

## Vertical load order

1. Read this workspace `AGENTS.md` and `AGENT_MEMORY.md`.
2. Select exactly one active project for the task.
3. Descend through that project's `AGENTS.md` and `AGENT_MEMORY.md` hierarchy.
4. Load exact-file `.agent-memory/<filename>.md` only for files actually in scope.

Do not start all products or load their memories together. If a module file is
absent from a checkout, verify that checkout against the current repository
tree rather than reconstructing it from pre-import boundary maps.

## Project router

| Work | Descend into | Commit from |
|---|---|---|
| Evidence, custody, ingestion, parsing, analysis, platform operations — **Indicia Probata** | `modules/Probata/probata/` | Propria root |
| Strategy, legal research, drafting, review, release preparation — **advocatio** | `modules/Legal-desktop/` | Propria root |
| Vault, corpus preparation, search and Intake desktop | `modules/Consignatio/` | Propria root |
| Root governance and migration records | this directory | this Propria Git root with an explicit path allowlist |

Never let an opened chat directory decide the commit root. Before staging, run
`git rev-parse --show-toplevel`, verify the Propria root, and stage only explicit
paths in scope. Other sessions' changes in the shared index remain theirs.

Current import state is recorded in `docs/monorepo-migration-manifest.json`.

The Evidence Platform is canonical for evidence. Legal Workspace consumes accepted
`LegalSourcePackage` data and never becomes a second writable evidence store.
