# Propria — Monorepo Root Contract

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 2 | Platform: Codex / win32 | Changes: establish Propria as monorepo root | Context: explicit owner correction -->

**Owner decision — 2026-09-12 (repository topology):**
`E:\AI_Workspace\Projects\Propria` is the top of the Propria monorepo. Owned
applications and product modules converge under `projects/`; shared, reusable,
reference, and vendor-governance material converges under `resources/`. Large
corpora, databases, caches, generated indexes, secrets, and runtime state are not
monorepo source and must remain ignored or externally mounted with manifests.

The existing child Git repositories are migration sources, not permission to
flatten dirty trees in place. Preserve their histories and local-only work until
the staged import is independently verified. An upstream fork may remain an
explicit external dependency only when its update/provenance lifecycle requires
that boundary; owned product code should converge into the root repository.

See [the staged migration plan](docs/MONOREPO-MIGRATION-PLAN-2026-09-12.md).

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 1 | Platform: Codex / win32 | Changes: record three-system boundary | Context: explicit owner clarification -->

**Owner decision — 2026-09-12:** CCC means project-local CocoIndex Code indexes only. Intake is the multifaceted CocoIndex-based filesystem workstation, using Weaviate for advanced search and SurrealDB for relationships, with multimodal tools/libraries (OCR, STT, video transcription/processing, advanced SLM extraction/classification). Docstore is CocoIndex + SurrealDB for project documentation. Keep their apps, tracking state, locks, configuration and target ownership isolated; shared technology is not a shared runtime. This defines scope, not proof every feature is implemented.

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
| Legal research, strategy, drafting, review, and release preparation — **advocatio** | `Probata/probata/modules/advocatio/` | descend through Probata's router | verify within selected module |
| Vault, corpus preparation, search and Intake desktop | `Consignatio/` | `Consignatio/AGENTS.md`, `Consignatio/AGENT_MEMORY.md` | `Consignatio/` |

Current paths above were checked after the owner relocation on 2026-09-10.
The old `the-platform-workspace/` and root-level `casebible/` routes are historical.
The former `milvus-coolify/` path is absent here; do not recreate it from old routing.
Historical naming amendments below are not current path-resolution instructions.

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

Read `AGENT_MEMORY.md` for progressive routing. Historical
`../REPOSITORY_BOUNDARIES.md` describes the former multi-repository representation;
the dated monorepo migration plan governs the new transition.

> _Sprint mode was removed by owner order on 2026-08-25. Confirm and discuss before changing._

> _Naming amendment: Claude Code · Fable 5.1 · 2026-09-06 — product canon D-137..D-150 (`probata/docs/NAMING.md`). Evidence Platform = **Indicia Probata** (`probata`, formerly `Agno-MCP-Platform`); Legal Workspace = **advocatio** (nested at `probata/modules/advocatio/`, formerly `modules/Legal-Workspace/`); TraceIQ = **vestigia** (nested at `probata/modules/vestigia/`, formerly `modules/traceIQ/`). The directory rename landed 2026-09-06; `Agno-MCP-Platform/` is now a junction to `probata/`. Old paths stay as junctions for a week; both names remain valid in recall stores (D-142)._

> _Byline amendment: Claude Code · Fable 5.1 · 2026-09-06 — directory renames D-137..D-142: `Agno-MCP-Platform/` → `probata/` (Indicia Probata), nested `modules/traceIQ/` → `modules/vestigia/`, `modules/Legal-Workspace/` → `modules/advocatio/`. Old paths remain as junctions for one week; recall stores keep both names (D-142)._
