# Architecture reference reconciliation — 2026-10-17

_Byline: ChatGPT · GPT-6 Astra · 2026-10-17._

## Why this correction exists

The owner requested removal of stale architecture and infrastructure references,
then explicitly included memsearch, claude-never-forgets and Propria Docstore.
Earlier assistant summaries had presented superseded stacks as current, while
active repository routers and the project canon contradicted later owner decisions.

This document records the sources and scope of that cleanup. It reconciles existing
decisions; it is not a new architecture selection. Source/configuration evidence,
accepted requirements and live verification remain distinct.

## Current sources used

Repository source was read at
[68f369eb738ef3fe27966ef49a1173ea18047d68](https://github.com/Cursedpotential/propria/commit/68f369eb738ef3fe27966ef49a1173ea18047d68).
The cleanup retains the owner's
[October 17 as-lived/hindsight/foreshadowing use case](../modules/Probata/probata/docs/AS-LIVED-HINDSIGHT-USE-CASE.md).

| Source | Established correction |
|---|---|
| [Root AGENTS.md](../AGENTS.md), [September 26 import](decisions/2026-09-26-monorepo-import.md), and the existing Vestigia October 2 routing note | One Git root at `E:/AI_Workspace/Projects/Propria`; application source under `modules/`; old product remotes are archives. |
| [Probata DECISION_LOG.md](../modules/Probata/probata/docs/DECISION_LOG.md), D-070/D-095 | Graphiti retired; it is not a baseline, dependency or bake-off gate. Neo4j has separate surviving roles. |
| Same decision log, D-073/D-080/D-107/D-151 | SurrealDB is the accepted governed temporal/analytical engine. Only the legacy Agno operational adapter was retired. |
| Same decision log, D-107/D-130 | AgentOS retired; Temporal, n8n, the existing Go engine, ContextForge and Portkey have distinct responsibilities. Agno is a bounded disabled library adapter. |
| Same decision log, D-143/D-144 | LlamaIndex and LangGraph are included for their defined retrieval, indexing, stitching, extraction and bounded assembly roles. |
| Same decision log, D-080/D-158; [PostgreSQL Dockerfile](../modules/Probata/probata/deploy/docker/postgres/Dockerfile) | PostgreSQL 18 plus configured pg_duckdb/PostGIS/pgvector; Weaviate search and governed graph projections have distinct roles. Source configuration does not prove current extension installation. |
| [October 5 operating-flags decision](decisions/2026-10-05-single-case-operating-flags.md) | Dev and Live retain one approved case identity. Prior identity-switching/reset instructions do not authorize a current rebuild or data rollback. |
| [SYSTEM-BOUNDARIES.md](../SYSTEM-BOUNDARIES.md) and current Docstore flags | CCC, Intake, documentation retrieval, agent memory and case/corpus search retain separate ownership and indexing boundaries. |

The owner's direct correction in this cleanup is that Supabase, Qdrant and Chroma
are not the current Propria stack. SQLite is provisional/local to particular apps.
Generic third-party adapter support and dated historical research do not establish
deployment of any of those products.

## Governed Docstore records read back

The following active owner records were returned by the governed flag read on
October 17. Record IDs and revisions make their scope traceable without treating
the word "active" as proof of implementation.

| Subject / flag | Revision | Applicable scope |
|---|---:|---|
| `note:ai_chat_content_extraction_20261006` / `docstore_flag:2e89a207550d9514c5200e8dfe0fdab3f6081c8824d53d597aeae0b8394e3566` | 9 | October 8: every source enters context without hashing, custody or pre-storage/search approval gates. AI conversations always remain context; bodies/messages stay outside PG and human-SMS normalization. Preserve complete exports and native turn/source locators. |
| `note:context_tool_selection_20261008` / `docstore_flag:bfc8e2f75d7dbce55c4d81c67c4d2f098adbe7e211819311f76fb012d1065a22` | 1 | Existing Go engine invokes tools one at a time as tracked Temporal Activities; DuckDB first where capable, Chonkie chunking, bounded Python gaps/hybrid work. |
| `note:coolify_validation_route_20261008` / `docstore_flag:0c315822212786f907a0585d80f11c6fe59423c6d357e5ebc1f084d7daae1f37` | 2 | Existing Coolify delivery and Go/Temporal execution; local/static checks are not production acceptance. |
| `note:single_case_operating_flags_20261005` / `docstore_flag:cc4ae1b4cfbe9759d33ba95a9dce54275d738343b7109ba6ddcbdf675e1b5d66` | 1 | One case identity and independent operating/feature controls; no shared-Live rollback. |
| `note:family_court_convergence_20261004` / `docstore_flag:413e1ed6cc4896a7c42a693f653fd740805ab486efcd14b6a6000cd39aa05613` | 1 | Shared Surreal legal library and platform-wide source traceability; current convergence is distinct from a read-only adapter stage. |
| `note:case_corpus_search_isolation_20261011` / `docstore_flag:79e033f2b8e8cc2c49a6bc7051663d5716d5c2e0f897c681797955abe4525d37` | 1 | Case/corpus search excludes Docstore and development results. Context_Files remains stopped. |

The original source-mirror canon was also read independently from GitHub:
`probata/PROJECT_CANON.md`, raw SHA-256
`19360bd27d1f15876bd44bee3057922ce179bd7064137b9d62edb46f857f0e87`.
Its October 11 amendments establish newer-owner-decision precedence and correct
the whole-product Surreal retirement misreading. Those corrections are retained
in the reconciled canon. The mirror's older INDEX lacked the October 17 use-case
entry already on GitHub; the reconciled index preserves that newer entry.

## Active references corrected

The current root README, root/module agent instructions and memories, migration
manifest and import decision now route through the single monorepo. The manifest
retains the old source/import checkpoint data explicitly as historical snapshots.

Probata's canon and documentation index remove the obsolete July fleet inventory,
old provider-priority/access recipes, retired AgentOS/Graphiti runtime and UI
directions, contradictory storage assertions, obsolete framework exclusions and
stale deployment/activation status. Accepted analytical requirements remain
linked to their decision records and the complete October 17 use case.

The September 22 purpose statement remains a dated owner record, with an explicit
October 8 scope amendment so its hash/custody wording does not gate current context
ingestion. Consignatio and Advocatio instructions use their imported module paths;
the legal-library description follows the October 4 convergence decision.

Deployment-file edits in this cleanup are comments/path guidance. The memsearch
wrapper's old managed-Zilliz label is corrected to the actual shared-journal/Milvus
role. No runtime instruction is changed by those comment/docstring edits.

## Memory and retrieval verification boundary

Each retrieval store must be updated through its own source and write lifecycle.
A corrected GitHub document or a new memory note is not proof that old embeddings
have been replaced.

- **Docstore:** governed source planning/application preserves exact hashes and
  quarantines replaced source bytes; source application is separate from indexing.
  A full-source reconciliation must preserve all seven registered roots. Selected
  paths are verification targets, not permission to retire omitted roots.
- **Memsearch:** the October 17 status read returned server `1.0.2`,
  memsearch `0.4.19+propria8`, collection `agent_session_memory_nemotron3`,
  80,865 rows, NVIDIA `nemotron-3-embed-1b`, and Milvus at the service's internal
  endpoint. This is a current agent-memory use of Milvus, distinct from its retired
  platform-wide vector role.
- **Memsearch source updates:** the maintained
  [fork at c4d52aac](https://github.com/Cursedpotential/memsearch/tree/c4d52aacbb98f4da6767e585073af23957d88721)
  keeps journals and the indexer on the desktop. Its hosted MCP exposes search,
  expand, recall and status only. Indexing exact corrected original file paths
  replaces obsolete chunks for those sources; indexing downloaded copies under
  different absolute paths would create duplicate sources.
- **CNF:** the installed plugin owns its local project JSON and locked/atomic
  storage routines. Source inspection and a verified local write/readback are
  required; a memsearch or Docstore update does not update CNF.

This repository change certifies documentation/source reconciliation only.
Per-store mutation receipts and post-cleanup queries determine whether each memory
or index has been refreshed. It does not certify a new fleet inventory, production
deployment or PostgreSQL extension probe.
