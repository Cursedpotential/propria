---
scope: .
status: current
verified_at: 2026-10-17
verification_scope: Repository routing and cited owner decisions at 68f369e; no live service verification
superseded_by: null
authority:
  - ../../../AGENTS.md
  - AGENTS.md
  - docs/INDEX.md
  - docs/PROJECT_CANON.md
  - docs/DECISION_LOG.md
  - docs/MEMORY_ARCHITECTURE.md
watches:
  - AGENTS.md
  - docs/INDEX.md
  - docs/PROJECT_CANON.md
  - docs/DECISION_LOG.md
contains_secrets: false
---

# Repository Agent Memory

> **Owner directive, 2026-09-24 07:18–07:20 (Claude Code · Opus 5.5).** ~~Any processing … is kept and propagated,
> never redone.~~ *Corrected 07:20 (owner: "a bit strong for what I said").* **Don't throw away work we'd have to do
> again.** Be efficient: carry usable work products from one task into the next, and look ahead to what later stages
> will need. Ad-hoc SQL over the catalog is often exactly that work (extraction, normalization, sender/direction
> fixes, deduplication, chunking, labels), so build it into the catalog and fill the platform tables from there later
> (owner: "keep it in a temp table for the moment"; "build it into the catalog and then we can extract it from the
> catalog into the tables"). Record the target beside each staged result. First register:
> `docs/handoffs/JEV-EVAL-OPTIONS-2026-09-23.md` § "Propagation register".

> _Byline: Codex · GPT-5 · 2026-08-27; navigation verification refreshed 2026-08-29._
> _Byline: Claude Code · Fable 5.1 · 2026-09-05 — naming canon sweep D-137..D-141; no UIW/proffer rows existed in this router. See `docs/NAMING.md`._
> _Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17; root import contract
> (2026-09-26), root operating-mode contract (2026-10-05), D-107/D-143/D-144/D-151
> and source tree read at `68f369e`._

This file is a router, not an encyclopedia. Read only the branches relevant to the current task.

## Durable owner preferences

- **Owner directive:** lead with the result, use labeled sections and plain English, and use an
  HTML/diagram/table when complex relationships materially benefit from a visual surface.
- **Owner directive:** finish one functional vertical slice before adding another feature; never
  substitute an unapproved surface or disconnected stub without explicit disclosure.
- **Owner directive:** persist material findings, decisions, implementation, and verification in
  project documentation; chat-only results are not a handoff.
- **Owner directive:** preserve concurrent work, use bounded ownership, and never hard-delete.
- **Owner directive:** edit and validate source in this checkout; commit and push; Coolify builds
  and deploys on the VPS. Do not create a duplicate local infrastructure stack.

## Current cross-tree corrections

- SurrealDB remains the governed temporal-graph, walk, and analysis target. Only the retired legacy
  operational adapter/instance is parked. See `AGENTS.md` and ADR-0056.
- LanceDB is not part of the current platform stack. **Milvus = memsearch only;
  Weaviate = project vector store projection** (owner ruling 2026-09-03). This
  role distinction is not a current health result. Follow the root's memsearch
  routing and check current status when using it.
- PostgreSQL remains canonical. Timesketch, search, graph, RAG, and operator surfaces are governed
  views or projections, not replacement authority.
- D-107 retires AgentOS; Agno remains only a disabled, bounded adapter. Temporal,
  n8n, ContextForge, Portkey and plain FastAPI have the roles recorded there.
  LlamaIndex/LangGraph are selected under D-143/D-144; their role is not a new
  canonical writer. See `docs/DECISION_LOG.md`.
- Current Dev/Live behavior follows the Propria root's 2026-10-05 single-case
  contract. Historical wipe/re-ingest notes do not authorize resetting shared
  Live data or erasing concurrent work.

## Repository boundary router

- Module path: `modules/Probata/probata/` from the Propria root. Current paths
  are recorded in the root `AGENTS.md` routing table.
- Repository status: imported on 2026-09-26. Probata, SBV, Timesketch and custom
  modules share the `E:/AI_Workspace/Projects/Propria` Git root. Old child Git
  directories and remotes are historical copies.
- Canonical linked-worktree root: `E:/AI_Workspace/Projects/Propria/_worktrees/`.
- Codex-managed exception: `C:/Users/matts/.codex/worktrees/`.
- Consignatio is the sibling module at `modules/Consignatio/` from the monorepo
  root. Read its own routers before Consignatio work; Probata's module contract
  does not replace its local authority.

## Path router

| Path in scope | Read next |
|---|---|
| `docs/**` | `docs/AGENT_MEMORY.md` |
| `server/**` | `server/AGENT_MEMORY.md` plus the closest nested memory |
| `modules/engine/**` | `modules/engine/AGENT_MEMORY.md` |
| `server/contracts/**` | `server/contracts/AGENTS.md` |
| `sql/**` | `sql/AGENT_MEMORY.md` |
| `deploy/**` | `deploy/AGENT_MEMORY.md` |
| `deploy/docker/**` | `deploy/docker/AGENT_MEMORY.md`; for n8n also `deploy/docker/n8n/AGENT_MEMORY.md` |
| Monorepo root: `modules/Legal-desktop/**` and `modules/vestigia-geodata_processor/**` | each sibling module's `AGENTS.md` / `AGENT_MEMORY.md` |
| `modules/workbench/**` | `modules/workbench/AGENT_MEMORY.md` plus the closest nested memory |
| `tests/**` | `tests/AGENT_MEMORY.md` |
| `knowledge/**` | `knowledge/AGENT_MEMORY.md` |
| `server/vendored/**` | `server/vendored/` project READMEs |
| `modules/forks/**` | `modules/forks/AGENT_MEMORY.md`, then the selected SBV or Timesketch source instructions |
| `modules/custom/**` | the selected module's local instructions (`llm_probe`, `llm_probe_ui`, `pilot`, `tool-skills`) |

Pre-restructure paths and the retired independent-repository routing remain in
Git history. The current SBV/Timesketch sources are under `modules/forks/`, and
the probe modules are under `modules/custom/`; all commit at the Propria root.

Format and precedence: `docs/agent-memory/README.md`.

## Geo lane is PARKED (D-121, 2026-08-31) — note for every future session
The location/GPS lane (stay_point, gps_track, geocode_*, home_base,
waypoint_device_split, vehicle, geofence, geocode_audit) is REAL, owner-ruled,
and deliberately OUT of the live database until ingest is proven on lesser
data. Complete one-file restore: `sql/parked/geo_lane_parked_20260831.sql`
(restore-proven against a clean target). Do NOT recreate these ad hoc; do NOT
treat their absence as a gap. Owner: "why am I gonna bring in the key to the
application I've been working on even longer than this, just to have it
fucked up too."

## Git on this repo
~~Desktop Commander ONLY~~ **Corrected 2026-09-01 (owner):** sessions with real
shell access (Claude Code desktop) run git directly. The DC-only rule applied
only to sandboxed Local-Agent-Mode sessions that could not unlink .git locks or
push (proven 2026-08-31, commit 15a1d87). Use DC only from such a sandbox.
