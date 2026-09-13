# Propria — Monorepo Root Contract

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 3 | Platform: Codex / win32 | Changes: establish universal Docstore use | Context: explicit owner clarification -->

**Owner decision — 2026-09-12 (repository topology):**
`E:\AI_Workspace\Projects\Propria` is the top of the Propria monorepo. Owned
applications and product modules converge under `projects/`; shared, reusable,
reference, and vendor-governance material converges under `resources/`. Large
corpora, databases, caches, generated indexes, secrets, and runtime state are not
monorepo source and must remain ignored or externally mounted with manifests.

Existing child Git repositories must retain their histories and local-only work
until a staged import is independently verified. Consignatio remains an explicit
independent canonical repository; its `projects/consignatio` import is a duplicate
overlay pending reconciliation, not an authority transfer. An upstream fork may
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
| Legal research, strategy, drafting, review, and release preparation — **advocatio** | `Probata/probata/modules/advocatio-legal_workbench/` | descend through Probata's router | verify within selected module |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md` | independent canonical Git root at `Consignatio/`; `projects/consignatio/` is a duplicate overlay pending reconciliation |

Current paths above were reconciled against the migration manifest on 2026-09-13.
The retired workspace layout is historical evidence and is not a current route.
The former `milvus-coolify/` path is absent here; do not recreate it from old routing.

The Evidence Platform is canonical for evidence. Legal Workspace consumes accepted,
versioned `LegalSourcePackage` data and never becomes a second writable evidence store.

## Reference-only areas

- `Agno-MCP-Platform-agno - alpha/` is a parts bin, not an active build.
- `dev-resources/` and `Legal-desktop/` are references/parts bins. Port bounded useful material only;
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
