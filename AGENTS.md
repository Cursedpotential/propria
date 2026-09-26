# Propria — Monorepo Root Contract

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


<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 3 | Platform: Codex / win32 | Changes: establish universal Docstore use | Context: explicit owner clarification -->

**Owner decision — 2026-09-12 (repository topology):**
`E:\AI_Workspace\Projects\Propria` is the top of the Propria monorepo. Owned
applications and product modules converge under `projects/`; shared, reusable,
reference, and vendor-governance material converges under `resources/`. Large
corpora, databases, caches, generated indexes, secrets, and runtime state are not
monorepo source and must remain ignored or externally mounted with manifests.

Existing child Git repositories must retain their histories and local-only work
until a staged import is independently verified. ~~Consignatio remains an explicit
independent canonical repository; its `projects/consignatio` import is a duplicate
overlay pending reconciliation, not an authority transfer.~~ **Corrected 2026-09-18 (Claude Code · Opus 5; owner: Propria is the project, no separate `projects/` home):** the 09-12 `projects/consignatio` overlay is quarantined in `Consignatio/to_be_deleted/2026-09-17-projects-consignatio-overlay-from-2026-09-12/`; Consignatio stays at `Propria/Consignatio/` (a same-day move into `projects/` was reversed). An upstream fork may
remain an explicit external dependency when its update/provenance lifecycle
requires that boundary.

See [the staged migration plan](docs/MONOREPO-MIGRATION-PLAN-2026-09-12.md).

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

**Owner decision — 2026-09-12 (universal Docstore):** Probata hosts
Docstore, but Docstore serves the entire Propria monorepo. All project
documentation will be progressively registered and indexed there. Every agent and
project must be able to use its first-class semantic search, bounded recall,
resource retrieval, note/decision, revision, flag and freshness-verification
tools. Before updating documentation or recording a note, query Docstore for
related current decisions; write through its governed tools and read back the
result. Source documents keep their owning repository and authority. Universal
Docstore access does not merge Docstore with CCC or Intake.

See [CCC / Intake / Docstore boundaries](SYSTEM-BOUNDARIES.md) for indexing eligibility, duplicate provenance and the human-agent organizing workflow.

**Owner rule — 2026-09-14 (decision preflight):** every decision, in any child
project or working directory, starts at the router root and runs the preflight
in `E:\AI_Workspace\AGENTS.md`: last week of Codex sessions, `/read-memories`,
memsearch, CNF, `.remember`, Docstore `fn::current_decisions`. Never stop at the
first result. <!-- added by Claude Code · Fable 5.1 · 2026-09-14 -->

> _Byline: Codex · GPT-5 · 2026-08-27; repository reconciliation refreshed 2026-08-29._

This directory currently contains several former independent product repositories
and historical/reference material. During migration, select the intended lane and
descend into its local `AGENTS.md` and `AGENT_MEMORY.md`. Those local contracts
remain authoritative until their lane is imported and the root manifest marks the
cutover complete.

These child references resolve in the configured live workspace. The parent repository does not
currently hydrate every child on a fresh clone; use `../REPOSITORY_BOUNDARIES.md` for canonical origins
and verified representation state.

## Active products

| Work | Repository | Next instructions | Commit root |
|---|---|---|---|
| Evidence custody, ingestion, parsing, knowledge horizons, analysis, and operations — **Indicia Probata** | `Probata/probata/` | `Probata/probata/AGENTS.md` | verify inside `Probata/probata/` |
| Legal research, strategy, drafting, review, and release preparation — **advocatio** (the legal workdesk / legal workbench; GitHub repo still `Legal-Workspace`) | `Legal-desktop/` | `Legal-desktop/AGENTS.md`, `Legal-desktop/AGENT_MEMORY.md` | independent Git root at `Legal-desktop/` |
| Simple record / reference / calendar surface usable from a phone and by agents — **Family Court Workbench** | `FL-MCP/` (**2026-09-18, owner: Propria is the project, no `projects/` folder** — moved back from ~~`projects/family-court-workbench/`~~) | its local `AGENTS.md` | Propria root |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md` | independent canonical Git root at `Consignatio/` (~~`projects/consignatio/` duplicate overlay~~ quarantined 2026-09-18) |

Current paths above were reconciled against the migration manifest on 2026-09-13.
Owner decision, 2026-09-13: Legal-desktop/ is the canonical independent Advocatio application. Its original build kit is preserved under resources/build-kit/. This is an explicit exception to the proposed projects/ layout.
The retired workspace layout is historical evidence and is not a current route.
The former `milvus-coolify/` path is absent here; do not recreate it from old routing.

The Evidence Platform is canonical for evidence. Legal Workspace consumes accepted,
versioned `LegalSourcePackage` data and never becomes a second writable evidence store.

## Reference-only areas

- `Agno-MCP-Platform-agno - alpha/` is a parts bin, not an active build.
- `dev-resources/` and `Legal-desktop/resources/build-kit/` are references/parts bins. Port bounded useful material only;
  never revive an archived iteration wholesale.
- Never open or ingest `dev-resources/Archives/OTHER_RESOURCES_TO_SORT/Secrets/`.
- Ignore `*.xxh3` checksum files during discovery.
- Historical workspace handoffs and v8.1 guides are seeds, not current product truth.

## Repository rules

1. Run `git rev-parse --show-toplevel` from the target path before staging or committing.
2. Until a project is marked imported in the root migration manifest, product
   changes commit in its existing child repository. After cutover, they commit in
   this root repository under `projects/<project>/`.
3. Root governance and migration files commit from this directory with an explicit
   path allowlist.
4. A dirty child or changed parent pointer is not authorization to stage that pointer.
5. Never reset, clean, stash, overwrite, or hard-delete concurrent work. Quarantine files under the
   owning repository's `to_be_deleted/` directory; only the owner deletes from quarantine.
6. Create Propria-owned migration worktrees under `E:\AI_Workspace\Projects\Propria\_worktrees\`.
   That directory is machine-local and ignored. During the transition, a child repository may link
   a worktree there without changing its Git boundary. Move an existing linked worktree only with
   `git worktree move`, after its owner is paused and its branch, dirty state, `.git` pointer, common
   directory, and representative hashes have been captured.

Read `AGENT_MEMORY.md` for progressive routing. The current workspace boundary
map is `../REPOSITORY_BOUNDARIES.md`; the migration manifest governs each lane's
transition state.

> _Sprint mode was removed by owner order on 2026-08-25. Confirm and discuss before changing._
