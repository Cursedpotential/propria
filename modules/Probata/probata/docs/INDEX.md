# Documentation Index — Current Entry Points

> **Reconciled 2026-10-17 — ChatGPT · GPT-6 Astra.** This index routes current
> accepted requirements separately from dated implementation and deployment records.
> It preserves the owner's October 17 as-lived/hindsight/foreshadowing reference.
> The old August "current live" paragraph, stale dirty-worktree claim, resolved
> six-question review hold and superseded migration-status summary have been removed.
> [Reconciliation sources and scope](../../../../docs/ARCHITECTURE-REFERENCE-RECONCILIATION-2026-10-17.md).

Read the latest scoped owner conversation and current Docstore decisions before a
lane's packet. An accepted design does not prove implementation or deployment.
Use [root AGENTS.md](../../../../AGENTS.md) for the single repository and current
module routes. Current context processing follows the October 8 owner correction:
all sources enter context without hashing, custody or pre-storage/search approval
gates; AI conversation bodies remain external context.

## Start here

| Need | Document | Authority and validation boundary |
|---|---|---|
| Product purpose and operator experience | [PURPOSE.md](PURPOSE.md) | Dated owner statement, with the current context-processing amendment |
| Accepted architecture and decision routing | [PROJECT_CANON.md](PROJECT_CANON.md) | Current responsibility map; newer scoped owner decisions control |
| Intended as-lived/hindsight walk, foreshadowing and perspective-correct review | [AS-LIVED-HINDSIGHT-USE-CASE.md](AS-LIVED-HINDSIGHT-USE-CASE.md) | Owner-approved explanation, October 17; not implementation proof |
| Current architecture corrections and source record | [Architecture reference reconciliation](../../../../docs/ARCHITECTURE-REFERENCE-RECONCILIATION-2026-10-17.md) | Existing owner decisions reconciled; documentation-only validation |
| Decision history and supersession | [DECISION_LOG.md](DECISION_LOG.md), [ADR index](adr/README.md) | Read decision date, scope and subsequent amendments |
| Naming | [NAMING.md](NAMING.md) | Product/component naming record |
| One case identity and Dev / Live behavior | [October 5 contract](../../../../docs/decisions/2026-10-05-single-case-operating-flags.md) | Owner contract; deployment status comes from current receipts |
| Source, module and commit ownership | [Root AGENTS.md](../../../../AGENTS.md), [September 26 import](../../../../docs/decisions/2026-09-26-monorepo-import.md), [module AGENTS.md](../AGENTS.md) | One Propria repository with imported modules |
| Indexing and data boundaries | [SYSTEM-BOUNDARIES.md](../../../../SYSTEM-BOUNDARIES.md) | CCC, Intake and Docstore remain separate |
| Forward work | Selected current lane packet, [BUILD_PLAN.md](BUILD_PLAN.md), [handoffs](handoffs/) | Reconcile dated plan entries with later owner decisions and current code |
| Code placement and conventions | [REPO_STRUCTURE.md](REPO_STRUCTURE.md), [CONVENTIONS.md](CONVENTIONS.md) | Apply alongside the current root/module contracts |
| Coordination history | [COORDINATION.md](COORDINATION.md) | Append-only, dated record |

## Current accepted roles

Temporal is the durable execution spine; n8n is the visual integration layer;
the existing Go engine invokes bounded tools through individual Activities.
ContextForge federates MCP tools and Portkey routes models. AgentOS is retired;
Agno remains only the bounded disabled library adapter described by D-107.

PostgreSQL retains canonical state and metadata, with the source-configured
pg_duckdb/PostGIS/pgvector extensions. Weaviate serves governed search, Neo4j
serves the independent Semantica/SAT graph projections, and SurrealDB serves
governed temporal/walk analysis. Graphiti is retired. LlamaIndex and LangGraph
have the accepted roles in D-143/D-144. Current context-source rules and each
store's scope are in [PROJECT_CANON.md](PROJECT_CANON.md#4-current-stack).

The source configuration, registered tools and a service's current health are
separate evidence. This index makes no claim that every accepted capability is
implemented or that old activation holds still describe today's deployment.

## Dated implementation and review records

These links preserve traceability. They do not establish the current next task,
current live health, or a continuing hold without newer corroboration.

| Record | Historical scope |
|---|---|
| [MASTER-TODO-2026-08-18.md](MASTER-TODO-2026-08-18.md) | August production-resume ledger |
| [OWNER-REVIEW-2026-08-18-verified-todo-audit.md](OWNER-REVIEW-2026-08-18-verified-todo-audit.md) | August code/DB verification and gaps |
| [Evidence Desk handoff](handoffs/HANDOFF-2026-08-18-evidence-operations-desk-mvp.md) | August MVP packet |
| [Derived-document ingest handoff](handoffs/HANDOFF-2026-08-29-derived-document-ingest-wiring.md) | August work-product wiring; later context decisions control |
| [AgentOS retirement handoff](handoffs/HANDOFF-2026-08-29-agno-role-dissection.md) | D-107 retirement/cutover record |
| [Temporal adoption plan](plans/TEMPORAL-INTEGRATION-PLAN-2026-08-23.md) | D-067 adoption sequence |
| [INGESTION-READINESS-2026-08-23.md](INGESTION-READINESS-2026-08-23.md) | Readiness observed on its date |
| [Cross-repo evidence audit](reviews/2026-08-23-cross-repo-evidence-audit/) | Dated audit; superseded framework and capture-time custody rules are not current |
| [Awaiting-verification index](awaiting-verification/README.md) | Unverified historical records |
| [R0–R14 packets](awaiting-verification/handoffs/) | Dated review packets |

The historical `docs/archive/` directory was removed in the September 8
restructure; do not follow its retired link or invent a replacement archive policy.
Original decision records and Git history preserve the superseded material.
