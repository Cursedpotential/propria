# AGENTS.md — Universal Entry Point

> _Original byline: Claude Code · Sonnet 5 · 2026-08-09, with dated corrections
> through 2026-09-30 by Claude Code and Codex; full amendment history is in Git.
> Routing amendment: ChatGPT · GPT-6 Astra · 2026-10-17. Sources: Propria root
> `AGENTS.md` (2026-09-26 import and 2026-10-05 operating-mode contract),
> Vestigia's 2026-10-02 router, and the source tree at `68f369e`._

> **Read the Propria root `AGENTS.md` before this module entry point.**
> This file provides Probata context and navigation. **Closest file wins** — nested
> `AGENTS.md` files below override this one for their subtree; read the nested map
> before editing inside that directory.

Repository-local memory follows the same path hierarchy: after reading applicable
`AGENTS.md` files, read `AGENT_MEMORY.md` root-to-leaf and then an exact target's
`.agent-memory/<filename>.md` when present. `AGENT_MEMORY.md` is a sourced context router,
never authority over current canon, ADRs, decisions, or verified handoffs. Format and
precedence: `docs/agent-memory/README.md`.

## THE GOAL — six steps, every surface is measured against them

> _Owner ruling · 2026-09-22 19:54 (D-159). Full record: `docs/PURPOSE.md`. Binding on every surface, screen, API and workflow in this repository._

1. **Open** a file or folder, through an index.
2. **Verify** whether it is relevant.
3. **Make sure it has a hash**, so it can be trusted later.
4. **Pick a parser or extractor** and get it into context and the analysis platforms.
5. **Preview** the result to make sure the machine did it right.
6. **Fill in gaps** and missing context.

Sorting exists only because step 1 could not be done on the corpus as it stood. Before adding
anything to a surface, name which step it serves; if it serves none, it does not go on the surface.
Sources owns steps 1–4. Review is steps 5–6 plus the accept/reject decision — the file's preview,
full screen, nothing else. Run status, receipts, stores, lineage, attempts and tool catalogs are
never primary content on an operator surface; they are a drawer, or their own screen.

**Context-processing amendment (owner, 2026-10-08):** the goals above do not
make custody, hashing or approval a prerequisite for current context ingestion,
for any source type. Evidence admission is a separate later phase; AI
conversations remain context. Read the dated amendment in `docs/PURPOSE.md`
and the current scope in `docs/PROJECT_CANON.md`.

## Project

**Indicia Probata** (`probata`) is the source-processing and eventual
evidence-record product under **propria** (D-137/D-138; naming:
`docs/NAMING.md`). Current work opens, extracts, normalizes where appropriate,
searches and reviews context from all supported sources, including messaging,
documents, media, location records and complete export packages. The owner's
2026-10-08 amendment removes custody, hashing and approval prerequisites from
context ingestion. Evidence admission and court-ready export are separate later
work; AI conversations remain context.

Analysis over a bitemporal graph belongs to **Indagatio Veri** / `indagatio`
(D-139). The eventual AI Legal Team is **advocatio** (D-138). See
`docs/PURPOSE.md` and `docs/PROJECT_CANON.md` for current scope.

## Repository and worktree boundary

Probata lives at `modules/Probata/probata/` in the Propria monorepo. Its history
was imported on 2026-09-26, together with SBV, Timesketch and the custom modules.
The commit boundary is `E:/AI_Workspace/Projects/Propria`; verify it with
`git rev-parse --show-toplevel` and stage only explicit paths in scope. Current
paths and import state are in the root `AGENTS.md` and
`docs/monorepo-migration-manifest.json` at that root. The old child Git
directories and remotes are historical copies, not commit destinations.

Normal linked worktrees belong under
`E:/AI_Workspace/Projects/Propria/_worktrees/`. Codex-managed worktrees may
instead live under `C:/Users/matts/.codex/worktrees/`; this tool-managed
exception does not create a new repository boundary. Fetch and branch new
worktrees from the monorepo's `origin/main`, as the root contract requires.

Consignatio is a sibling module at `modules/Consignatio/` from the monorepo
root. Read its own routers for that work; Probata's module contract does not
replace Consignatio's local authority.

The root's 2026-10-05 single-case operating-mode contract also applies here.
Historical reset and re-ingest notes do not authorize restoring a snapshot over
shared Live data or erasing concurrent changes. Read
`../../../docs/decisions/2026-10-05-single-case-operating-flags.md` for that
owner decision.

## Standing Subagent Authorization

> _Owner directive · 2026-08-18._

**Always use subagents when work can be investigated, verified, or executed usefully in
parallel.** This is standing, explicit owner authorization for Codex, Claude Code, Gemini,
OpenCode, and any other agent host whose policy otherwise requires the user to request
subagents for each task. The root agent remains responsible for scope, integration, review,
and final verification. Trivial or inherently serial steps do not require artificial
delegation, but agents must not avoid useful subagents merely because the current prompt did
not repeat this authorization.

## The horizon-delta analysis agent — downstream, and NOT an intake input

> _Reframed 2026-09-26 (Claude Code · Opus 5) on owner correction. This section used to read
> "read this before proposing anything about storage, retrieval, memory, agents or schema,"
> and sessions duly applied it to intake and extraction work, where it does not belong. The
> owner, 2026-09-26: it is "an agent driven by query-time analysis through Surreal that we
> can't get to yet because you keep fucking up the intake and the ETL." Stop raising it during
> intake, extraction, ELT, projection or schema work._

**What it is.** A query-time analysis agent running over SurrealDB. It compares what was
knowable at a point in time against what is known now, and the **delta is the deliverable** —
"what you were led to believe vs what was true vs when you found out." Not a timeline.

**Where it sits.** Downstream of everything. It is gated behind working intake and a working
ETL, and it reads the analytical surface. It is not a store, a lane, a table or a destination,
and it does not shape how data is extracted or committed.

**What intake actually owes it — dates, and nothing more.** Carry the temporal columns on the
record and the agent does the rest at query time: `occurred_at` (when it happened),
`knowledge_time` (when it became known), and `disclosure_tier` / `ai.disclosure_horizon`
(`contemporaneous` / `hindsight` / `discovered`). Those columns already exist in the schema.
Owner, 2026-09-26: "that's all in a query later. It just needs proper dates."

**Two facts worth keeping for whoever builds the analysis layer** (they cost real debugging
time and are not obvious):

- **A horizon filter must be a PRE-filter in every store.** Vector search is the leak:
  embeddings have no sense of time, so a later document scores as similar as a contemporaneous
  one. Filtering after top-k silently shrinks k instead of excluding.
- **On Weaviate, use dict filters, never a `FilterExpr`.** A FilterExpr passes tests on other
  vector databases and applies **zero** filters in production.

Contamination is silent either way — a leaked later fact does not raise an error, it just makes
the analysis quietly worthless. That is a reason to get the filter right when the agent is
built, not a reason to constrain extraction now.

## ATOMICITY — every unit must be assignable to a Temporal Activity

> _Owner directive · 2026-09-02. Binding on every directory below this file.
> Reinforces the 2026-08-25 boundary ruling, ADR-0061, and D-077._

**Write every unit of work so it can be handed to one Temporal Activity, and never
conflate multiple processes into one unit.**

Owner, 2026-09-02: *"Everything needs to be modular so that it can be assigned to
Temporal activities. We can't be conflating or mixing a bunch of processes into one.
Yes, the engine can call individual ones, but it's going to be calling the Activity
more likely than 99.9% of the time."* And: *"Or to be added into an n8n node which
gets run as an activity, however that shape looks."*

Rules, in force everywhere:

1. **One unit does one thing.** A parser parses and does nothing else (owner,
   2026-08-29: *"they parse, they do nothing more"*). A chunker chunks. A hasher
   hashes. If a function does two of those, it is wrong and must be split before it
   is wired to anything.
2. **Hashing is its own Activity family and is never folded into parsing, chunking,
   or normalization.** Custody hashing is separate machinery with its own boundary
   (D-077, four hash moments; see `docs/reference/HASH-TAXONOMY-2026-08-29.md`).
3. **The Activity is the normal caller.** Direct in-process calls stay legitimate —
   but the overwhelmingly common path is invocation *as*, or from *within*, a
   Temporal Activity. Design signatures for that: bounded inputs, bounded outputs,
   no ambient state, no hidden I/O, deterministic given its inputs, safely
   retryable. An Activity may be retried; anything that breaks on a second identical
   call is a defect.
4. **Three call shapes, one unit.** The same unit must serve all of them without
   knowing which is in play: (a) called directly in-process; (b) invoked as a
   Temporal Activity; (c) **wrapped as an n8n node that is itself executed as, or
   from within, an Activity.** n8n owns the visual flow, Temporal owns durability,
   the unit owns one job. A unit that needs to know its caller has a boundary
   violation in it.
5. **Pass references, never payloads.** Source bytes and bundles move by locator
   (`upload://`, `r2://`, sealed `file://`), never through Temporal history, an n8n
   payload, or a PostgreSQL activity request.
6. **No orchestration inside a unit.** Sequencing, fan-out, retries, and human gates
   belong to the workflow (`modules/engine/proffer`; renamed from `uiw`, D-140) and to n8n's visual flow — never
   buried inside a parser, decoder, chunker, or repository method.
7. **New capability = new Activity, registered in the stage graph.** Do not widen an
   existing Activity to cover a second concern because it is convenient.

The test before adding or editing anything here: *could this be scheduled on its own,
retried, wrapped as an n8n node, and reasoned about in isolation?* If not, it is not
finished.

## Stack

PostgreSQL is the canonical source/control plane; Weaviate is the project search
projection. Neo4j carries the Semantica-originated semantic graph; SurrealDB is
the governed reconciled temporal-graph, walk and analysis target (D-073/D-080).
Temporal owns durability and n8n the visual business/agent flow. ContextForge is
the MCP gateway; Portkey is the sole model, embedding and reranking gateway;
plain FastAPI is the platform HTTP host (D-107). AgentOS is retired. Agno may
remain only as the disabled, bounded atomic-agent adapter described in D-107.
LlamaIndex and LangGraph are selected for the roles in D-143/D-144; they do not
replace Temporal/n8n orchestration or become writers of canonical truth.
These are the owner-ruled boundaries, not a claim that every implementation is
complete. Dependency versions come from `requirements.txt` and `pyproject.toml`.

The PostgreSQL image is configured for PG18 with `pg_duckdb`, PostGIS and
pgvector, plus the `pg_stat_statements` preload. Sources: accepted ADR-0013,
`deploy/docker/postgres/Dockerfile` and `deploy/data-pg.yaml`, read at `68f369e`.
This is source configuration; the installed extensions and working DuckDB/PG
integration require verification on the target database.

The 2026-09-03 role ruling is **Milvus for shared memsearch only; Weaviate for
the project vector projection**. It is not a perpetual service-health result.
Use the root's memsearch routing and a current status check when that service is
needed. The old Agno operational Surreal adapter/instance and Graphiti are
retired (D-095/D-107/D-151); those retirements do not retire SurrealDB's
analytical role. Historical exports remain governed by the preservation rules,
with their original locations recorded in the dated handoffs and Git history.

## Repository Layout

> Backend repacked under one `server/` boundary (ADR-0033); `server/tools/` is
> capability-sub-namespaced (ADR-0035). Progressive disclosure: this table
> tells you WHICH nested `AGENTS.md` to read before editing a subtree.

| Directory | What lives there | Nested map |
|---|---|---|
| `server/` | The whole backend (`server.*` imports), dependency direction | `server/AGENTS.md` |
| `server/contracts/` | Import-light `NormalizedRecord`/`RecordType` contract | `server/contracts/AGENTS.md` |
| `modules/contracts/` | **Future home** of cross-language contract schemas (JSON/YAML consumed by Go + Python + n8n). Owner ruling 2026-09-01: the old root `contracts/` (a never-populated placeholder) was removed; recreate HERE — not `docs/` (machine-consumed artifacts drift as docs), not hidden `.contracts/` (boundary definitions must stay reviewable) — only when H-02 lands the first real schema files, so the directory is born with content. | — |
| `server/evidence/` | The evidence spine: custody, store, workflows, cli | `server/evidence/AGENTS.md` |
| `server/tools/` | Cross-domain parser/extractor/gateway registry | `server/tools/AGENTS.md` |
| `server/agents/` | Agent/team constructors, providers, `@tool` wrappers | `server/agents/AGENTS.md` |
| `server/timeline/` | Canonical timeline membership and Timesketch projection | `server/timeline/AGENTS.md` |
| `server/api/`, `server/core/`, `server/analysis/`, `server/ingest/` | Entrypoint/config, DB session/model factory, behavioral analysis, ingest application service | see `server/AGENTS.md` |
| `server/case_management/`, `server/observability/`, `server/temporal/` | Case views/workflows, audit/telemetry, and Temporal integration | see `server/AGENTS.md` |
| `server/vendored/` | Third-party Python projects (chatminer, semantica) — not ours to lint | — |
| `modules/engine/` | The Go engine (own `go.mod`; ~~UIW~~ **proffer** (D-140) custody writer, parsers, chunkers, stagegraph) — **moved from root `engine/` in the 2026-09-01 owner restructure** | — |
| `modules/workbench/` | Operator Workbench — `api` (FastAPI) + `web`. Moved from root `workbench/` 2026-09-01. | — |
| `modules/forks/` | Imported application source: `timesketch` and `sbv`, both tracked in the Propria root since 2026-09-26. SBV's donor provenance remains D-131 (MIT, Copyright (c) 2025 lowcarbdev); the import does not change that provenance. | each project's README; SBV `UPSTREAM.md` |
| `modules/custom/` | Imported owner-authored modules: `llm_probe`, `llm_probe_ui`, `pilot`, `tool-skills`; tracked in the Propria root. | each module's local instructions |
| Monorepo root: `modules/Legal-desktop/` (**advocatio**, D-138) | Sibling product, sharing the Propria Git boundary. Consumes accepted `LegalSourcePackage` read-only; never a second writable evidence store. | its own `AGENTS.md` / `AGENT_MEMORY.md` |
| Monorepo root: `modules/vestigia-geodata_processor/` (**vestigia**, D-140) | Sibling product; the application is its `vestigia/` folder (renamed 2026-10-02). | its own `AGENTS.md` |
| `modules/apps/` | Owner's transient staging area during reorganizations — gitignored, contents move on; never reference it in code or docs | — |
| `sql/` | Schema source is `bootstrap/schema_snapshot_<date>.sql` (D-142 §3, D-152, D-153). Numbered migrations were retired to `sql/_stale/` on 2026-09-07. Historical rebuild instructions are subject to the root's 2026-10-05 single-case contract; they do not authorize resetting shared Live data. | `sql/bootstrap/README.md` and the root operating-mode decision |
| `deploy/docker/` | One folder per service image (`tools/`, `gateway/`, `postgres/`, ...) — moved from root `docker/` 2026-09-01; compose files in `deploy/` now resolve their `./docker/...` build contexts correctly per the compose spec | — |
| `docs/` | Canon, ADRs, decision log, plans, wiki | `docs/PROJECT_CANON.md` |
| `tests/` | The pytest suite | — |
| `scripts/` | format/validate/ingest/entrypoint | — |

## Development and verification topology

> _Owner clarification · 2026-08-26._

The source is edited in this checkout, then committed and pushed; Coolify builds and deploys the
containers on the VPS. Do not create a duplicate local application or infrastructure stack and do
not run local Docker/Podman/Compose services. CWD validation remains required: formatting, lint,
mypy/typecheck, unit tests, integration tests that exercise the real deployed services, application
build tests such as `next build`, and lockfile/dependency verification. Those checks do not replace
Coolify deployment and live VPS proof.

## Commands

| Task | Command |
|---|---|
| Lint | `uv run ruff check server tests` |
| Format check | `uv run ruff format --check server tests` |
| Typecheck | `uv run mypy server` |
| Test (default, unit) | `uv run pytest -q` |
| Test (one file) | `uv run pytest -q tests/test_<name>.py` |
| Integration tests (live services) — **REQUIRED before any "done"** | `uv run pytest -m integration` |
| Go build/test (engine) | from `modules/engine/`: `go build ./...` / `go vet ./...` / `go test ./...` |
| Go build/test (`modules/forks/sbv`) | from that dir: `go build -tags fts5 ./...` / `go test -tags fts5 ./...` (own `go.mod`, shared Propria Git root) |

The `fts5` build tag is **mandatory** for `modules/forks/sbv`. A plain `go test ./...`
fails every DB-backed test with `no such module: fts5` — that is a missing build
tag, not a code defect. Use the `modules/forks/sbv/Makefile` targets, which set it for you.

~~Integration tests are opt-in.~~ **Corrected 2026-08-20 (owner directive):** live
integration tests are **mandatory**, not opt-in. `pytest -q` alone never establishes that
something works — see the LIVE ONLY testing policy below. Unit runs are a fast local
smoke check only.

All Python is `uv`-managed — never invoke a bare `python`/`pip`/`pytest`.

### Test layout (owner consolidation ruling, 2026-09-01)

`tests/` holds the pytest suite (source, tracked). Generated pytest durable
reports write to **`tests/_reports/`** (gitignored) — test source and test
results share one parent. `build/` is packaging output only and holds no test
artifacts. Configured in `server/observability/pytest_reporter.py`; recorded in
`docs/CONVENTIONS.md` and ADR-0054 (amended).

### Scan & discovery tooling (owner directives, 2026-09-01)

- **DuckDB** (`duckdb` CLI, v1.5+ local) is the preferred engine for bulk
  scans: file sweeps via `read_text()` globs + `regexp_extract_all`, schema
  drift anti-joins against live `information_schema`, CSV/JSON crunching.
  Prefer it over hand-rolled Python iteration for set-shaped scans.
- **CocoIndex Code (`ccc`)** semantic index is maintained up to date on this
  repo — use it for repository-wide discovery, blast-radius analysis, and as a
  second net after mechanical grep sweeps (catches string-built SQL and
  f-string qualifications grep misses). Private dev assistance only — never
  cite it in product architecture or plans.
- **Git:** sessions with real shell access run git directly. Desktop Commander
  is only for sandboxed Local-Agent-Mode sessions that cannot unlink `.git`
  locks or push (see `AGENT_MEMORY.md`).

## Agent Topology

D-107 retired AgentOS and the Agno-native platform topology. Retained Agno
definitions are disabled adapters, not the durable workflow or control plane.
D-107 requires an explicit Temporal task ID, reference-only input/output
contract, scope, tool allowlist and tested approval path before one is callable.
Read `server/agents/AGENTS.md` and `server/agents/factory.py` for the retained
source roster; apply D-107 and the Activity boundary above when working on it.
The former Router / Platform Ops / Builder diagram remains in Git history.

## Model gateway and legacy provider settings

Portkey owns model routing under D-107. Inspect the active caller and gateway
configuration for current selection. The fallback settings in
`server/core/settings.py` describe the retained Agno adapter; they do not define
the platform's orchestration or authorize reactivating that adapter.

The dated 2026-09-25 correction (Claude Code · Opus 5.5) selected NVIDIA NIM
`moonshotai/kimi-k3` and made Ollama Cloud non-default. The owner's blanket ban
on `glm-5.1` remains. A dated model preference does not establish current
provider availability.

## Querying SurrealDB stores (every agent)

> _Byline: Claude Code · Opus 5 · 2026-09-10 — owner order: everybody runs the same query format.
> Scope corrected 2026-09-30 (Claude Code · Fable 5.1): `sq.py` reaches the Docstore store only._

Inspect the Docstore store (`surreal-docs`, namespace `probata`, database `docs`) through `scripts/docstore/sq.py`, never a one-off script that prints raw SDK objects.
It normalises `RecordID`, datetimes, embeddings and bodies, then prints a DuckDB table; `--sql` runs DuckDB SQL over the result as table `r`.

The other SurrealDB stores (the case store and the agent memory on `surreal-case`, the Intake graph on
`surreal-intake`) are not reachable with `sq.py`. The map of every instance, its access rules and the
SurrealQL tools are in the `surrealdb` plugin: `/surrealdb:surrealdb-deployments`.

```bash
C:/Users/matts/.local/bin/python3.exe scripts/docstore/sq.py "SELECT * FROM todo LIMIT 5;"
C:/Users/matts/.local/bin/python3.exe scripts/docstore/sq.py "SELECT doc_type, status FROM document;" --sql "SELECT doc_type, count(*) n FROM r GROUP BY ALL"
C:/Users/matts/.local/bin/python3.exe scripts/docstore/sq.py --target docs "INFO FOR DB;"
```

Full usage, flags and rules: the `propria-docstore` plugin's `query` skill (`/propria-docstore:query`).

## Further Reading

Before a non-trivial task, identify which of these are relevant and read them first:

- `docs/PROJECT_CANON.md` — vision, locked decisions, roadmap, gotchas
- `docs/REPO_STRUCTURE.md` — where every kind of file goes
- `docs/CONVENTIONS.md` — coding style, tool contract, docstring standards
- `docs/COORDINATION.md` — multi-chat lane ownership + live TODO ledger
- `docs/DECISION_LOG.md` — running append-only decision log (complements ADRs)
- `docs/adr/` — Architecture Decision Records (`docs/adr/README.md` = index)
- `docs/DEBT.md` — active stubs and known debt

Tool-specific setup (hooks, slash-commands) lives in that tool's own config, never here.

## Commit Attribution

AI commits carry: `Co-Authored-By: <agent name and model> <noreply@anthropic.com>`

## Deployment and custody learnings

_Derived from the dated claude-reflect notes; original export and bylines remain
in Git history. Retired AgentOS operation instructions were removed under D-107._

### Environment Setup
- Current host and database routing is in the Propria root `AGENTS.md` under
  Paths. Historical host moves do not establish the current address of a service.
- Tailnet services bind their configured `${BIND_IP}`. On-box probes and SSH
  tunnels must use the intended service's bind address; a loopback target does
  not reach a service bound only to its tailnet address.

### Deploy & Data-Tier Gotchas
- Coolify apps WITHOUT watch_paths redeploy on EVERY push to their branch — the whole data tier (pg/neo4j/graphiti/surreal/vector) bounced on every main merge until watch paths were scoped per-app (2026-07-21/22). New app = set watch_paths at creation, always.
- Milvus standalone boot "etcdserver: leader changed" panic loop (exit 134). ~~Cause: embedded etcd defaults + slow VPS disk; fixed via heartbeat 1000 / election 10000~~ ~~ungraceful stop mid-flush~~ **Corrected 2026-09-17 (log-proven):** with EMBEDDED etcd, Milvus's first metadata read goes out ~20 ms after etcd starts, before etcd has elected itself leader, and fails "leader changed" once it does — a startup race on every restart of a store with data (the "7 corruptions"). Fix in place: etcd runs as its own `memsearch-etcd` service and Milvus starts only when it is healthy. Never go back to `ETCD_USE_EMBED=true`.
- memsearch-milvus on the 3.0 nightly with `storageType: local`: JSON shredding stats load via a remote-style `files/` path -> "invalid local path" (segcore 2011) and the collection sticks at Loading. Keep `common.enabledJSONShredding: false` until on a release that fixes it.
- **memsearch-milvus is deployed by Coolify FROM GIT** (app renamed from `data-vector` 2026-09-17, compose `deploy/memsearch-milvus.yaml`). Never edit the rendered compose under `/data/coolify/applications/<uuid>/` on the host: a 2026-09-10 host-only edit was silently reverted by the next Coolify deploy, which mounted a dead volume (2026-09-17 outage). Commit + push, then deploy.
- Historical Agno workflow gotcha: `Step.on_error` defaulted to skip. This is
  relevant only when inspecting retained adapter code; D-107 retired Agno as
  platform orchestration.
- H3 construction history (owner-verified 2026-08-01): the SBV Go chain starts
  with an empty value and folds `sha256(previous + "<LF>" + H2)`; the Case Bible
  chain used genesis H1 and `sha256(previous_hex + h2_hex)`. Neither construction
  is invalid merely because it differs from the other. The 2026-08-02 decision
  resolved the ambiguous tag: new H3 rows carry `h3-chain-sbv-genesisempty-v1`;
  legacy `h3-chain-v1` rows are read-only and disambiguated by writer. See
  `modules/forks/sbv/CUSTODY.md`, `modules/forks/sbv/internal/custody.go`, and
  `docs/DECISION_LOG.md` (2026-08-02).

## Documentation lifecycle

Current truth is indexed by `docs/INDEX.md`; completed or superseded documents move under
`docs/archive/` in the same change. ADRs and append-only `docs/DECISION_LOG.md` remain in
place. Mockup/design history is never production truth.

## Owner result-persistence rule

> _Owner directive · 2026-08-26._

Always persist material investigation, audit, verification, decision, and implementation results
in project documentation. Chat-only reporting is not a durable handoff. If the final canonical
location is not yet known, record the result in a temporary TODO, `docs/pending-review/`, a session
handoff, or another clearly labeled durable project artifact, then reconcile or promote it later.
Never allow useful findings to disappear merely because a session ends or the permanent document
has not been selected yet.

## Owner delivery rule — production means production

> _Owner directive · 2026-08-18._

Every request to build, add, finish, or change a product capability means production
implementation, Coolify deployment, and live verification. Local, staged, preview, or
mockup work is never completion unless the owner explicitly says “mockup.” Use the
least-expensive subagent that can reliably complete each bounded task; escalate model or
specialist capability when task risk or complexity requires it. The root agent orchestrates,
records decisions, integrates results, and prevents duplicate work.

## Repository-wide discovery rule

Use CocoIndex Code (`ccc`) / CocoIndex v1 privately for repository-wide discovery, semantic
code indexing, and blast-radius analysis when available. Do not use CocoIndex v0 APIs. Do
not mention this development assistance in application architecture, RAG design,
implementation plans, TODOs, handoffs, or product-status reporting.

### Session Learnings 2026-08-02
- The design-phase wipe/re-ingest performed on 2026-08-02 is historical. The
  root's 2026-10-05 single-case contract governs current Dev/Live behavior and
  supersedes pre-launch resets; this note grants no current reset permission.
- Config accepted ≠ feature working: agno accepted `EntityMemoryConfig(mode=PROPOSE)` for months and silently did nothing. Verify features via docs + an observed write, never via config acceptance.
- Custody canon tags name the exact construction: new H3 rows carry `h3-chain-sbv-genesisempty-v1`; legacy `h3-chain-v1` rows are read-only and disambiguated by writer (see docs/DECISION_LOG.md 2026-08-02).
- Remove a worktree when its branch merges — 13 stale ones hid an unmerged security fix. Quarantine untracked remnants before `--force` removal.

### Session Learnings 2026-08-12
- The former root `compose.yaml` is now `deploy/compose.yaml`. Service-specific
  deployment definitions are under `deploy/`; follow the selected Coolify app's
  source and compose path recorded in the root contract and deployment config.
  Changes to a live service's definition remain production-facing edits.
- Owner vocabulary: "artifact" = created WORKS (AI chats, generated documents/code). Never name extraction output tables/columns `artifact_*` — fact-claims are `claim_candidate` (entities can be merged/deduped; claims accumulate and are NEVER rewritten). Locked in ADR-0052 ruling Q6 / D-054.
- Linked worktrees for this repository belong under `E:/AI_Workspace/Projects/Propria/_worktrees/`,
  except Codex-managed worktrees under `C:/Users/matts/.codex/worktrees/`. Always verify the
  worktree's Git common directory and branch before mutation.
- Engine-split routing is COVERAGE-based, never size-based (ADR-0052 ruling Q3): Go parses every format it has a decoder for, any size; Python serves uncovered formats or logged failure-fallback only. No byte thresholds anywhere in the router.

---

> _Sprint-mode policy REMOVED 2026-08-25 on owner order ("you're grounded — remove it entirely"). Confirm-and-discuss-before-changing is back in force._
