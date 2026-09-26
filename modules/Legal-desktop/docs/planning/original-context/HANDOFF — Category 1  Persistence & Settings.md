# HANDOFF — Legal OS Category 1: Persistence & Settings — Implementation Research

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


Status: rough feature set agreed with owner (2026-08-17); this handoff requests in-depth research to scaffold the actual implementation. Research only — no code, no schema activation, no infra changes.

## Context

Legal OS is a single-user, single-case, self-represented-litigant practice-management app (Bloomberg-terminal-style UI), being built as a sibling package to an existing platform, "Agno MCP Platform" (repo: mcp-platform-agno-mvp), inside one monorepo. Agno owns evidence custody, ingestion, and a bitemporal analysis pipeline that promotes reviewed "claims" into "established facts" + "promoted evidence subgraphs," landing in a governed SurrealDB analytical surface (Postgres remains canonical/authoritative underneath it; SurrealDB is a curated downstream projection). Legal OS owns the UI shell, legal research/contract/citation tools, and a new "AI Legal Team" (motion/strategy/discovery drafting).

## Feature set already agreed (do not re-litigate — build research around this)

1. Legal OS's own plain app state (settings, workflow definitions, saved research queries, drafts, UI preferences) lives in a **dedicated Postgres schema** (e.g. `legal_os`) inside the **same Postgres 18 cluster** Agno already runs (which includes `pg_duckdb`, `pgvector`, `PostGIS`) — not a separate database.
2. Migrations should follow **Agno's existing convention**: numbered raw SQL files under a `sql/` directory (their lineage runs `sql/0001` → `sql/0018+`, with `sql/_manual` and `sql/drafts` staging dirs) — not Alembic or a second migration tool, to keep one mental model and one CI check across the monorepo.
3. Data contracts should be **Pydantic v2 models** mirroring Agno's own pattern (see `server/contracts/case_management.py` and `server/case_management/repository.py`/`service.py` for the exact shape to match — thin service layer, repository does persistence, Pydantic models are the API/domain boundary).
4. Startup/secret config via `pydantic-settings` (env-sourced); user-editable runtime preferences (theme, refresh interval, notification settings) via a small `legal_os.app_settings` table — single-row config, since this is single-user (no per-user settings scoping needed).
5. **SurrealDB is the primary data source the AI Legal Team consumes from** for evidence grounding (established facts, promoted evidence subgraphs) once Agno's Surreal analytical surface is production-ready. As of 2026-08-17 it is at "core-gate pass" in Phase 1 (see Agno's `docs/HANDOFF-2026-08-17-R14-phase1-surreal-live-core-pass.md` and `docs/SURREAL-INVESTIGATION-BLUEPRINT-2026-08-15.md`) — not yet activated for production, no schema/target locked. The AI Legal Team's evidence-retrieval interface should be designed as a first-class SurrealDB client from day one, with a fallback path through Agno's plain REST API (`/v1/matters/*`, `/v1/knowledge/items`) if SurrealDB isn't reachable yet.
6. **Open question, explicitly deferred to this research pass:** should Legal OS's own document parking (uploaded case documents, generated drafts/briefs) live in SurrealDB (native document type) or Postgres/object storage (R2, already used by Agno)? Leaning consideration already on record: the Surreal blueprint's own §1 governing boundary states "object storage retains original bytes and custody bindings" — implying custody-bound originals belong in Postgres/R2, with only promoted extracts/citations in SurrealDB. **Confirm this against the actual blueprint text and Agno's implementation before recommending a final answer** — don't re-derive from first principles alone.

## Research questions to answer (with citations/links)

1. **SurrealDB Python SDK** — current stable client library, query patterns for reading a governed/curated projection (not writing to it), connection/auth patterns matching what Agno's own `tool-skills/graphiti-client` or their Surreal test harness already does (check `tests/test_surreal_investigation_phase0_contract.py` in the Agno repo for the client patterns they've already validated). Cite the actual SDK docs and version.
2. **Postgres schema-per-app patterns in a monorepo** — survey how similar multi-app-one-database setups handle schema ownership, migration ordering across apps that share a cluster, and connection-string/role separation (e.g., a `legal_os_app` Postgres role scoped to only the `legal_os` schema plus read grants on Agno's `case_management` tables). Cite real examples/blog posts/docs, not just general advice.
3. **pydantic-settings vs. alternatives** (e.g. `dynaconf`, `python-decouple`) for a single-user app's env+file config — recommend one with rationale, cite docs.
4. **Document storage precedent** — look at how comparable evidence/legal-tech or forensic platforms (open source if any exist) split "original document custody" vs. "derived/promoted analysis" storage. Also re-read Agno's own `docs/SURREAL-INVESTIGATION-BLUEPRINT-2026-08-15.md` §1 and §4 (canonical identity layers) directly and quote the exact boundary language before making a recommendation.
5. **Repository/service-layer pattern libraries** — is there a lightweight OSS pattern/library worth adopting for the repository-service split (matching Agno's own hand-rolled pattern), or is hand-rolling correct here (i.e., don't add a dependency just to formalize a pattern that's already three files)?

## Deliverable

A single markdown report, saved to the workspace, with:
- One recommendation per research question above, each with source citations (real URLs)
- A concrete `sql/000X_legal_os_bootstrap.sql`-style migration sketch (schema + `app_settings` table + any other Category-1 tables identified)
- A resolved answer (or a clearly-flagged remaining open question) on the document-parking decision, grounded in the actual Agno blueprint text
- Explicit non-goals: no code beyond illustrative SQL/Pydantic sketches, no infra changes, no credential handling decisions (flag those as a separate future pass)
