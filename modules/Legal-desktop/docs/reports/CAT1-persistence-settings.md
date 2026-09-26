# Category 1 — Persistence & Settings

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_
> Research report requested by the zip handoff. No live schema apply.

## Recommendations

### 1. SurrealDB Python SDK (read-only projection)

Use the official `surrealdb` async Python client when Agno’s
analytical surface is actually reachable.
Docs: [SurrealDB Python SDK](https://surrealdb.com/docs/sdk/python).
Agno already pins `surrealdb>=1.0.4,<2`.

Until that surface is production-ready, Legal Workspace reads
established facts from `LegalSourcePackage` + Evidence Platform REST
(`/v1/established-facts/{id}/versions/{v}`). Do not make Surreal a
startup hard-dependency.

### 2. Schema-per-app on one cluster

Follow PostgreSQL schema + role isolation
([Privileges](https://www.postgresql.org/docs/current/ddl-priv.html),
[Schemas](https://www.postgresql.org/docs/current/ddl-schemas.html)).

- Schemas: `legal_core`, `legal_research`, `legal_work_product`,
  `legal_release`, `legal_audit`
- Role: `legal_os_app` — DML on those schemas only
- Numbered SQL under `sql/`, same as Agno (no Alembic)

Sketch shipped: `sql/0001_legal_os_bootstrap.sql`.

~~Runtime settings lived only in `state.json` until 2026-08-18.~~
**Correction 2026-08-18 (owner):** local runtime is **SQLite WAL**
(`data/workspace/legal.sqlite`, `sql/0001_legal_os_sqlite.sql`) for
`legal_core_app_settings`. Live Postgres apply stays Type 1 HOLD.

### 3. Config

Use `pydantic-settings` v2
([docs](https://docs.pydantic.dev/latest/concepts/pydantic_settings/)).
Env + `.env` for secrets; single-row `legal_core.app_settings` for
theme/confidential_mode/case_phase. Matches Agno and the handoff.

Reject `dynaconf` / `python-decouple` — second config dialect in a
monorepo that already uses pydantic-settings.

### 4. Document parking (resolved)

Agno blueprint: object storage keeps original bytes and custody
bindings; curated projections are downstream.

**Decision:** R2 prefix `legal/` for owner-produced drafts and
release candidates. Postgres stores metadata + hashes. Surreal does
not store documents.

Self-custody hashes for *Legal-Workspace-produced* files live in
`legal_work_product.work_product_version.content_hash`. They are not
Evidence Platform established facts.

### 5. Repository / service split

Hand-roll, matching Agno (`contracts` → `repositories` → `services`
→ `api`). No extra ORM framework. SQLAlchemy can wait until more
than the bootstrap tables exist.

## CocoIndex (Agno territory, 2026-08-18)

`ccc search` from `Agno-MCP-Platform/` shows promotions start
`safe_for_legal_use=false` until HITL. Legal import must keep dropping
unapproved items. Details: `docs/planning/AGNO-INDEX-NOTES.md`.

## Non-goals

No production migration, no credential design beyond env names, no
second PG cluster on OVH-1 unless the existing host is full.
