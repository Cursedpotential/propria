# SurrealDB Agent Memory — Expert Brief

> **CORRECTION 2026-09-09 06:55 (D-157): the self-hosting conclusion in this brief is WRONG.** SurrealDB Agent Memory ships a free self-hostable server binary `spectrond` (https://surrealdb.com/docs/agent-memory/reference/cli — install script at download.surrealdb.com/spectron, `spectrond bootstrap`, `spectrond dev start --bootstrap --connection-string … --bind-address 0.0.0.0:9090`, MCP at /mcp on that port; configuration reference documents SPECTRON_SURREALDB_URL and a local object store for single-node deployments). This brief read the "Hosted quickstart" (SurrealDB Cloud) as the only path and never opened the CLI or configuration reference. Treat every "closed source / enterprise-only / Cloud-only" statement below as struck. Kept in place as history per house rule; the corrected plan is `docs/reference/2026-09-09-spectrond-vps-deployment-plan.md` (pending).


> Byline: Claude Code · Sonnet 5 · 2026-09-09
> Read-only research brief. Sources: surrealdb.com/docs/agent-memory/* (full site-map enumerated via `/docs/llms.txt`, ~110 pages), surrealdb.com marketing/blog pages, github.com/surrealdb/surrealmcp, github.com/surrealdb/agent-memory. Compiled from six parallel subagent fetches (sonnet model) covering every page listed in Section G; each subagent was instructed to quote verbatim and flag fetch failures. No repository files were edited as part of this research.

---

## A. What SurrealDB "agent memory" actually is today

**Naming.** The product is documented under the name **"SurrealDB Agent Memory."** Its internal/former codename **"Spectron"** survives everywhere in the actual surface area: the CLI binary is literally called `spectron`, the server binary is `spectrond`, HTTP headers are `X-Spectron-Context` / `X-Spectron-On-Behalf-Of`, env vars are `SPECTRON_*`, error types are `Spectron*Error`, and third-party package names are `spectron-crewai`, `spectron-hermes`, `@surrealdb/spectron-vercel-ai`, etc. The Respan observability integration page states outright that "Spectron" was the former project name for Agent Memory. No version number for the Agent Memory product itself was found on any page (unlike SurrealDB the database, which is versioned — SurrealDB 3.0 was announced via blog in Feb 2026, and 3.2.0 is the desktop's installed winget version per our constraints).

**Built-in vs. separate binary — the load-bearing fact for our deployment.** SurrealDB Agent Memory is explicitly **NOT** a feature of the `surreal` server binary and is **NOT** embeddable in-process. Two independent pages state this almost verbatim:

> "There is no supported in-process library that runs extraction and recall inside your binary without the SurrealDB Agent Memory server." — `/docs/agent-memory/integrations/surfaces/embedded-library`

> "There is no supported in-process API that runs extraction and recall inside your application binary without the SurrealDB Agent Memory server." — `/docs/agent-memory/quickstarts/embedded`

The architecture is: **`spectrond`** (the Agent Memory application-tier server, described as "a horizontally scalable HTTP service in front of SurrealDB" — `integrations/surfaces/embedded-library`) sits in front of a **separate, already-running SurrealDB database instance** that it uses as its storage backend (config requires `SPECTRON_SURREALDB_URL`/`_USER`/`_PASS` — `reference/configuration`). `spectrond` serves the REST API, the MCP endpoint, and the filesystem view, all on **one host, default port 9090** (`integrations/surfaces/rest`). A separate client CLI, `spectron`, talks to a running `spectrond` over HTTP.

This means "Agent Memory" is a **three-layer stack**, not a database feature:
1. **SurrealDB** (the database engine) — open source, self-hostable, is what actually stores the data (graph/vector/document/relational tables).
2. **`spectrond`** (the Agent Memory application tier) — a **separate closed-source Rust binary** that implements extraction, reconciliation, retrieval ranking, the REST/MCP/chat surfaces, and talks to #1 as its backing store.
3. **`spectron`** — the CLI/client for #2.

Per the marketing page (`surrealdb.com/agent-memory`), quoted exactly:

> "SurrealDB, the database engine underneath, is open source and free to self-host. The Agent Memory layer on top is closed source and ships as a single Rust binary with no Python runtime and no sidecars."

And:

> "Self-hosted and air-gapped deployments, with local model inference, are available for qualifying enterprise engagements."

This is a critical, load-bearing distinction for our constraints: **SurrealDB itself (the DB engine) is free/open/self-hostable with no license gate** (this is what the winget-installed 3.2.0 binary is). **The Agent Memory layer (`spectrond`) is a closed-source, separately-distributed binary**, and full self-hosting of it ("self-hosted and air-gapped") is described as an **enterprise-tier offering**, not something documented as freely downloadable like the CLI (`spectron`) itself appears to be (install script found: `curl -fsSL https://download.surrealdb.com/spectron/install.sh | sh`, per `reference/cli` — that installs the **client CLI**, `spectron`, not necessarily an unrestricted `spectrond` server binary for self-hosting; the docs did not show an equivalent public download/install command for `spectrond` itself in the pages fetched). Pricing tiers found on the marketing page: Sandbox (free, 3M tokens one-off), Agent Memory Lite ($30/mo, 1M tokens/mo), Standard ($300/mo, 10M/mo), Plus ($1,100/mo, 40M/mo), Enterprise (custom, includes self-hosted/air-gapped).

**Distinct from `surreal mcp`.** Separately, core SurrealDB (the plain database) has its own **built-in `surreal mcp` CLI subcommand** that starts an MCP server over stdio for raw SurrealQL/database access — this is a completely different, unrelated MCP surface from Agent Memory's own `/mcp` HTTP endpoint. (Full detail pending the MCP-cluster subagent; see Section C.)

**What "eight pillars, six categories" means as a product claim.** The docs frame the whole system around eight architectural "pillars" (Authoritative, Experiential, Reconciliation, Elaboration, Reflection, Consolidation, Calibration, Collective — `architecture/eight-pillars-and-categories`) and six experiential "memory categories" (Episodic, Identity, Knowledge, Context, Instructions, Uncertainty). Note a **documented inconsistency** the research surfaced: the glossary page (`reference/glossary`) defines "Memory category" as an **API-enforced enum of only three values** — `identity`, `knowledge`, `context` — while three other pages (`architecture/eight-pillars-and-categories`, `mental-model/memory-categories`, `memory-and-knowledge`) all describe **six** categories including episodic, instructions, and uncertainty. This is a real doc-drift issue worth flagging if we ever rely on the "category" enum in integration code.

---

## B. The memory-and-knowledge model

### Two-layer (authoritative / experiential) architecture

Everything lives in **one SurrealDB graph** — not two databases. Records carry a `source.kind` field (`document`, `upsert`, `turn`, `reflect`, `elaboration`, `consolidation`, …) that tags which "stream" produced them, and a reconciler applies different trust defaults per stream:

- **Authoritative** — "manuals, policies, product data, structured uploads" (`source.kind = "document"` or `"upsert"`), higher default trust, ingested via document upload or trusted-triple `POST /facts` with `infer:"triples"`.
- **Experiential** — "what people and agents said" plus everything the system derives from it (reflection, elaboration, consolidation), lower default trust, ingested via `remember()`/turns.

> "Curated knowledge and conversational knowledge are distinguished by `source.kind` and trust policy, not by hiding one half in a separate database you cannot join transactionally." — `welcome/accuracy-promise`

On conflict, **authoritative wins and is never silently modified**; the conflicting experiential belief is stored with provenance intact, and the disagreement is surfaced as an `uncertainty` record rather than resolved silently (`reasoning/authority-hierarchy`, `cookbooks/build/customer-support-agent`: "Authoritative content is protected from conversational drift regardless of how many users assert conflicting information.").

### Entities, facts, episodes — the actual data model

From `reference/data-model-and-schema` (three-plane layout: `spectron/metadata` control plane, `spectron/_jobqueue` async jobs, one `(namespace, database)` pair per **Context** for actual data):

- `document` — uploaded/authoritative source, states `queued → extracting → chunking → embedding → keywording → ready/failed`.
- `knowledge_chunk` — vector-embedded passage (3072-dim, Gemini embedding model), with simhash-based near-duplicate suppression.
- `keyword` — RAKE-extracted keyphrases linked to documents via `knowledge_has_keyword` edges.
- `session` / `turn` — conversation containers and individual messages (`role`: user/assistant/system/tool). This is your "episode" analog — the raw ordered transcript is the **Episodic** memory category.
- `entity` — typed node (`Person/alice`, `Product/airpods_pro`, …), keyed by a **composite `[normalised_type, normalised_name]`** — an exact-match lookup, **not** fuzzy/embedding-based matching. Fixed type enum: `person, organisation, project, location, topic, product, policy, concept, event, agent, service, other`.
- `attribute` — key/value on an entity, carries confidence, temporal validity, and supersession chain pointers.
- `relates_to` — entity-entity edges with labels and temporal bounds.
- `instruction` — behavioral directive (active/inactive), applied at prompt-assembly time, not generic retrieval.
- `uncertainty` — explicit "we don't know / sources conflict" record.
- `memory_chunk` — turn-derived embedded text segment (used by hybrid retrieval alongside `knowledge_chunk`).
- `decision_trace` / `retrieval_trace` / `response_trace` — the "trace layer" (see below).

### Supersession / versioning semantics (tri-temporal model)

This is the most load-bearing section for comparison with our own `active|superseded|retracted` design. SurrealDB Agent Memory tracks **three separate clocks simultaneously**, quoted exactly from `architecture/tri-temporal-model`:

> "System time — SurrealDB MVCC history... Known time — When SurrealDB Agent Memory first recorded a belief; `as_of` walks supersession chains... Valid time — When the fact held in the real world, via `valid_from`/`valid_until`."

Mechanically:
- Every active attribute/relation has `valid_from` and `valid_until` (`null` = still valid).
- **Supersession is automatic**, triggered as a side effect of the reconciler during extraction — not an application-called function. On a same-source conflict, if the new extraction's confidence is ≥ a configurable floor (default **0.7**) and ≥ the existing row's confidence, the old row is closed (`valid_until` set) and a new row opens (`valid_from` set), with the two linked in a chain (`supersedes`/`supersededBy`). If confidence is below floor, **both persist and an `uncertainty` record is created** instead of overwriting.
- Cross-provenance conflicts (experiential contradicting authoritative) **never modify the authoritative record** — the experiential belief is stored separately with provenance, and a conflict/`uncertainty` object is surfaced.
- History is queryable per-attribute: `GET /entities/{type}/{name}/history/{key}` returns the full chain; `POST /query` (and other reads) accept an `asOf` parameter for point-in-time reconstruction.
- **Corrections are described as immutable/irrevocable**: "Corrections accumulate and are never deleted. The full supersession chain for any attribute is always queryable... the audit trail for any fact is complete and irrevocable." (`sessions/state-and-diffs`)
- **`/state` explicitly excludes superseded facts by default**: "always returns the current, canonical view: superseded attributes are not included." This exclusion is confirmed explicitly only for `/state`; the general `/query` recall page did not restate the same guarantee in the pages fetched — worth verifying directly if our design depends on ranked search also excluding superseded rows (it very likely does, since the reconciler updates `valid_until` on the canonical row, but the docs corpus fetched did not say so in as many words for `/query` specifically).

**Retraction / deletion is a separate lifecycle stage from supersession**, and maps closer to (but is richer than) our `retracted` status. `operations/forget` and several cookbook pages describe a **four-state lifecycle**, not three:

- **active** (current, returned by all reads)
- **superseded** (closed by a newer same-source value; `valid_until` set; still in history)
- **retired** (our closest analog to `retracted` — explicit application call to `POST /forget`, default behavior: soft-delete via `valid_until = now`, excluded from `/query`, `/profile`, `/state`, and chat context going forward, but still visible in history/audit)
- **purged** (a distinct, harder tier we do not have an analog for — `POST /forget {"purge": true}` "permanently removes the matched entities, their fact history" — irreversible)

`forget` supports `dryRun: true` to preview matches before committing (both REST and `spectron forget --dry-run`).

**No first-class `decision_log`/event-log construct exists.** The closest analogs are: (a) the inline `uncertainties`/`conflict` object returned from `/query` and `/documents.query` on some conflicts, and (b) the **trace layer** (`decision_trace`, `retrieval_trace`, `response_trace` — see below), which records what the reconciler considered/decided on every write and what a ranked read returned, but is framed as an internal feedback signal for ranking/consolidation, not as a queryable "why did this change" audit log comparable to a purpose-built `decision_log` table. It is graph-resident (`decision_trace` nodes have edges to the entities/attributes they touched), so it is *walkable*, but it is not documented as a stable, versioned, application-facing event schema the way our `decision_log` presumably is.

### Retrieval (hybrid, RRF, vector index, filters)

`retrieve/hybrid-search` is the single most load-bearing retrieval page. Exact quotes:

> "Reciprocal rank fusion (RRF) of vector and BM25 results. Both retrieval passes run independently, and their ranked lists are merged into a single ranking."

> "score(d) = Σ 1 / (k + rank_i(d))" — with default smoothing constant **k = 60** (configurable per-query as `rrf_k`).

> "Pure HNSW (hierarchical navigable small world) approximate nearest-neighbour search over dense embeddings." Embedding space is **3072-dim** (Gemini embedding model, fixed, not swappable per `SPECTRON_MODEL_EMBEDDING` config — "must be `gemini-embedding-2`... deployment-fixed").

> "Filters are applied before scoring, so they do not affect the ranking of results that pass through." — i.e. **pre-filter, not post-filter**, but this is described in API-parameter terms (`scope`, `labels`, `lens`), not as raw SurrealQL syntax.

An additional `hybrid_graph` mode reranks the initial hybrid candidates using graph-density signals from named edge types (`knowledge_has_keyword`, `section_match`, `document_link`, `document_summary`).

**No raw SurrealQL KNN-operator syntax (e.g. a `<|K,EF|>` literal-K/EF clause) or `DEFINE INDEX ... HNSW` statement with explicit M/EFC build parameters was found anywhere in the fetched Agent Memory doc set.** Everything vector/HNSW-related is exposed through the higher-level HTTP API (`k`, `rrf_k`, `mode`) rather than raw SurrealQL — this is a real gap relative to what our own docstore kit assumes about controlling KNN literals/EF directly; **if we need literal K/EF control, that lives at the SurrealDB-core SurrealQL layer, not the Agent Memory HTTP API**, and would require bypassing Agent Memory and talking to SurrealDB directly with our own `DEFINE INDEX ... HNSW` + `<|K,EF|>` queries (which is exactly our docstore kit's own approach).

BM25 (`retrieve/keywords-and-bm25`) runs through a named `spectron_analyzer` (blank + class tokenizers, case-normalization + English stemming) — invoked via `mode="bm25"` or `mode="hybrid"` in the API. **No raw `DEFINE ANALYZER`/`DEFINE INDEX ... BM25` SurrealQL was shown**, and no BM25 k1/b coefficients are exposed as configurable.

Query-size limits: `k` (answer size) defaults to 10, capped at 50 per deployment (`SPECTRON_MAX_QUERY_K`); candidate pool size for hybrid retrieval defaults to 256 (`SPECTRON_RETRIEVAL_POOL_SIZE`).

Retrieval overall is described as a **four-tier cost ladder** (`architecture/coherence-retrieval-and-tiers`): Tier 1 = direct entity/attribute key lookup (no embeddings/LLM), Tier 2 = semantic response cache (reuse prior answers above 0.95 cosine similarity, invalidated automatically when a cited fact is superseded — "no manual flush verb"), Tier 3 = fused hybrid retrieval (vector+BM25+graph) over the bounded candidate pool + LLM synthesis, Tier 4 = broader sweep when Tier 3 confidence is below a 0.40 floor.

### What's automatic vs. what the app must do

**Automatic (no application call required):**
- Extraction on every `remember()`/turn write (escalates through heuristics → fast LLM → strong LLM stages as needed; synchronous/blocking on `POST /sessions/{id}/turns`, per `sessions/adding-turns`: "blocks until extraction is complete").
- Reconciliation/supersession as a side effect of extraction.
- Cache invalidation on the semantic response cache (Tier 2) when a cited fact changes.
- Entity/relation graph construction, document chunking/embedding/keywording on upload.
- Cross-layer linking (`resolves_to` between experiential mentions and authoritative catalogue entities), when normalization matches.
- Ontology-grounding feedback (recently-used entity/relation/attribute vocabulary fed back into the extraction prompt to reduce vocabulary fragmentation) — but there is **no API for supplying our own controlled vocabulary**.

**Manual (application/operator must call):**
- `POST /reflect` — on-demand LLM synthesis, optionally `persist: true` to write results back as knowledge-category attributes. Explicitly not automatic: "developers must call the `reflect()` method directly. It is not automatic." (`cookbooks/patterns/reflection-loops`)
- `POST /forget` — retire or purge; requires a `memory:forget` grant.
- `POST /consolidate` — memory pooling/merging.
- Entity resolution across name variants — **not automatic**: "Name variants are separate entities" and `same_as` relations must be created manually (`cookbooks/patterns/spoiler-safe-narrative-memory`); the system does not fuzzy-merge entity names on its own.
- Choosing which model runs each pipeline stage (extraction/reconciliation/synthesis/elaboration-consolidation are independently configurable; embedding model is fixed).
- Scope/Context/principal/grant provisioning (`spectron scopes create`, `spectrond contexts create`, `spectrond principals create`, key minting/rotation) — all explicit setup steps, not automatic.

---

## C. MCP server

There are, confirmed across 25 pages/repos, **three genuinely distinct MCP surfaces** carrying the SurrealDB name. Conflating them was the single biggest risk in the kit we inherited (which assumes `surrealdb/surrealmcp:latest` via Docker) — that image is **one specific, now-archived generation** of surface #2 below, not "the" SurrealDB MCP server.

### C.0 — The three surfaces, disambiguated

| # | Surface | What it exposes | Transport | Install | Status |
|---|---|---|---|---|---|
| 1 | **`surreal mcp`** (built into the core `surreal` binary, SurrealDB 3.1+) | Raw SurrealQL/record CRUD tools (query, select, create, insert, upsert, update, delete, relate, namespace/db switching) against a local embedded or file/rocksdb-backed instance | **stdio only** | Already on your machine — it's a subcommand of the `surreal` binary you installed via winget | Current, actively documented, this is "Embedded MCP" |
| 2 | **`surrealmcp`** standalone binary (`github.com/surrealdb/surrealmcp`) | Same DB CRUD tools **plus** SurrealDB Cloud instance/org management | stdio, HTTP, or Unix socket | `cargo install --path .` (Docker-free) or `docker run surrealdb/surrealmcp:latest` | **Archived.** README now reads: "SurrealMCP is now a part of SurrealDB and SurrealDB Cloud." This is what the buildkit's Docker-based kit assumed — it is legacy. |
| 3a | **`https://mcp.surrealdb.com`** (current hosted, unified MCP) | DB CRUD tools + SurrealDB Cloud management + **Agent Memory context management** (create/configure Spectron contexts) | HTTP, browser-login or Bearer PAT | Nothing to install — it's a URL | Current, actively promoted successor to #2 |
| 3b | **Agent-Memory-only MCP** (`https://<your-context-host>/mcp`, or a self-hosted `spectrond`'s own `/mcp`) | Exactly 7 memory tools: `remember`, `recall`, `context`, `reflect`, `forget`, `upload`, `inspect` | HTTP | Nothing to install client-side; server side requires a running `spectrond` (no public Docker-free — or Docker — install path was found anywhere in the docs or the `surrealdb` GitHub org) | Current |

Critically: **`github.com/surrealdb/agent-memory` does not exist** — confirmed as a genuine 404 both by direct fetch and by `gh repo list surrealdb --limit 200` (130+ org repos enumerated, none named `agent-memory` or bare `spectron`; only downstream framework-integration repos like `spectron-openclaw`, `spectron-hermes`, `spectron-google-adk`). **No public install path — Docker or otherwise — exists for the `spectrond` Agent Memory server binary itself.** This is the single most important gap for our "memory on VPS" plan: whatever runs Agent Memory on the VPS today, its install/run recipe is not published; only its *client-facing* HTTP/MCP contract is documented.

### C.1 — Surface #1: `surreal mcp` (built-in, stdio, Docker-free, local)

Exact syntax (`/docs/reference/cli/surrealdb-cli/commands/mcp`):
```bash
surreal mcp [OPTIONS] [PATH]
```
`[PATH]` defaults to `"memory"` (in-memory, non-persistent). For a persistent local docs store:
```bash
surreal mcp --user root --pass secret --ns main --db main rocksdb://path/to/data
```
Flags: `--ns <NAMESPACE>`, `--db <DATABASE>`, `-u/--username`, `-p/--password`.

Explicit security caveat quoted from the fetch: **"lacks per-call HTTP authentication, so avoid exposing to untrusted users"** — it runs every tool call with **owner-level (root) credentials**, and the given `-u`/`-p` only take effect if no root user exists yet. Treat it as single-trusted-operator-only, which matches our desktop use case.

Claude Code / Codex / Cursor / VS Code / Claude Desktop config is identical in shape everywhere — spawn the binary as a stdio child process (`/docs/build/ai-agents/mcp/embedded`):

Claude Code / Claude Desktop (`.mcp.json` / `claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "surrealdb": {
      "command": "surreal",
      "args": ["mcp", "--user", "root", "--pass", "secret",
          "--ns", "main", "--db", "main", "memory"]
    }
  }
}
```
VS Code (`.vscode/mcp.json`):
```json
{
  "servers": {
    "surrealdb": {
      "type": "stdio",
      "command": "surreal",
      "args": ["mcp", "--user", "root", "--pass", "secret",
          "--ns", "main", "--db", "main", "memory"]
    }
  }
}
```
No Codex-specific example for this surface was found in any of the 25 pages — a documented gap; the same `[mcp_servers.<name>]` TOML shape used for surface #3b (below) applies, just swap `url`/`bearer_token_env_var` for a `command`/`args` stdio launcher (Codex's `config.toml` supports both stdio and HTTP `mcp_servers` entries per its own schema, even though SurrealDB's docs don't show it explicitly for this surface).

**Note on our winget install:** `surreal` is present but not on PATH — the `command` field above must be the **absolute path** to `surreal.exe` (or PATH must be fixed) for any of these configs to launch successfully.

### C.2 — Surface #2: standalone `surrealmcp` (archived — legacy, do not adopt for new work)

Confirmed via `gh api` against the last pre-archival README (commit `6b82d699`), byte-exact (not WebFetch-summarized):

Docker-free install:
```bash
cargo install --path .
```
Docker install (what the inherited buildkit assumed):
```bash
docker run --rm -i --pull always surrealdb/surrealmcp:latest start
```
Usage (three transports):
```bash
surrealmcp start                                          # stdio (default)
surrealmcp start --bind-address 127.0.0.1:8000             # HTTP
surrealmcp start --socket-path /tmp/surrealmcp.sock        # Unix socket
```
Full non-Docker config example:
```bash
surrealmcp start \
  --endpoint ws://localhost:8000/rpc \
  --ns mynamespace \
  --db mydatabase \
  --user root \
  --pass root
```
Env vars: `SURREALDB_URL`, `SURREALDB_NS`, `SURREALDB_DB`, `SURREALDB_USER`, `SURREALDB_PASS`, `SURREAL_MCP_BIND_ADDRESS`, `SURREAL_MCP_SERVER_URL`, `SURREAL_CLOUD_AUTH_SERVER`, `SURREAL_MCP_EXPECTED_AUDIENCE`, `SURREAL_MCP_RATE_LIMIT_RPS`, `SURREAL_MCP_RATE_LIMIT_BURST`, `SURREAL_MCP_AUTH_REQUIRED`, `SURREAL_MCP_CLOUD_ACCESS_TOKEN`, `SURREAL_MCP_CLOUD_REFRESH_TOKEN`.

License: **Business Source License (BSL 1.1)**, not pure open source. **This repo is archived and its README now redirects to `mcp.surrealdb.com`** — treat as historical reference only; do not build new automation on it. Since the buildkit's Docker assumption points here, **the direct non-Docker substitute is `cargo install --path .` against the frozen final commit, but this is not a forward-maintained path** — prefer surface #1 (embedded, current) or #3a (hosted, current) instead.

### C.3 — Surface #3a: hosted unified MCP (`https://mcp.surrealdb.com`)

```json
{
  "mcpServers": {
    "surrealdb": { "url": "https://mcp.surrealdb.com" }
  }
}
```
Auth: browser sign-in via Surreal ID by default, or a personal access token (from `account.surrealdb.com/tokens`) via header:
```json
{
  "mcpServers": {
    "surrealdb": {
      "url": "https://mcp.surrealdb.com",
      "headers": { "Authorization": "Bearer <your-token>" }
    }
  }
}
```
Claude Code: `claude mcp add --transport http surrealdb https://mcp.surrealdb.com` (add `--header "Authorization: Bearer <token>"` for PAT auth); `claude mcp remove surrealdb` to remove.
Version requirement for direct SurrealQL execution against your own instance through this surface: **"it must be on SurrealDB 3.1 or later"** (per `/docs/build/ai-agents/mcp`) — though the SurrealMCP blog post separately states **"DB ≥ 3.2.3"** for query execution, a minor inconsistency across SurrealDB's own pages worth flagging, not resolving. Our installed 3.2.0 satisfies either reading.
No Docker, no local process — this is pure remote HTTP, nothing to self-host.

### C.4 — Surface #3b: Agent-Memory-only MCP (the "Spectron" tools)

Tool list, verbatim from `integrations/mcp-server/tools-reference`:

| Tool | Params | Purpose |
|---|---|---|
| `remember` | `text` (required), `session_id`, `scope`, `labels`, `infer`, `context_id` | Store a fact/exchange, auto-classify + reconcile |
| `recall` | `query` (required), `k`, `mode`, `lens`, `labels`, `context_id` | Ranked unified search over facts + document passages |
| `context` | `query` (required), `lens`, `labels`, `context_id` | Assemble a markdown context block for prompt injection |
| `reflect` | `query` (required), `persist`, `context_id` | On-demand synthesis, optional persistence |
| `forget` | `query` (required), `purge`, `context_id` | Soft-delete (default) or permanently erase (`purge`) |
| `upload` | `bytes_base64` (required), `title`, `source`, `mime_type`, `filename`, `scopes`, `labels`, `context_id` | Upload a document for async processing |
| `inspect` | `ref` (required), `context_id` | Look up an entity/trace/document by typed reference |

`context_id` is optional on every call since one API key is bound to exactly one Context.

**Install via helper (writes config for most clients):**
```bash
npx install-mcp https://<your-context-host>/mcp \
  --client cursor \
  --header "Authorization: Bearer <your-api-key>" \
  --oauth no
```
`--oauth no` because "SurrealDB Agent Memory uses a static Bearer key, not OAuth." Supported `--client` values via this helper: cursor, vscode, windsurf, zed, opencode.

**Manual config, generic shape:**
```json
{
  "mcpServers": {
    "spectron": {
      "url": "https://<your-context-host>/mcp",
      "headers": { "Authorization": "Bearer <your-api-key>" }
    }
  }
}
```
Gotcha: Windsurf and Antigravity use `"serverUrl"` instead of `"url"`.

**Claude Code — two paths:**
1. Env-var + plugin:
   ```bash
   export AGENT_MEMORY_MCP_URL="https://<your-spectron-instance>/mcp"
   export AGENT_MEMORY_MCP_TOKEN="<your-api-key>"
   /plugin marketplace add surrealdb/ai-claude-plugin
   /plugin install spectron@surrealdb
   ```
   Verify: `claude mcp list`. Gotcha: "`AGENT_MEMORY_MCP_URL` must be set or the server will not connect."
2. Claude Desktop config file (macOS `~/Library/Application Support/Claude/claude_desktop_config.json`, Windows `%APPDATA%\Claude\claude_desktop_config.json`):
   ```json
   {
     "mcpServers": {
       "spectron": {
         "type": "http",
         "url": "https://<your-spectron-instance>/mcp",
         "headers": { "Authorization": "Bearer <your-api-key>" }
       }
     }
   }
   ```

**Codex CLI** (`~/.codex/config.toml` or project `.codex/config.toml`):
```toml
[mcp_servers.spectron]
url = "https://<your-context-host>/mcp"
bearer_token_env_var = "AGENT_MEMORY_API_KEY"
http_headers = { "X-Spectron-Context" = "acme-prod" }
```
```bash
export AGENT_MEMORY_API_KEY="<your-api-key>"
```
"For SurrealDB Cloud, obtain the context host from Studio's API keys section; for self-hosted setups, use your server's base URL plus `/mcp`." Verify: `codex mcp list`.

**Other clients (config paths/shapes, all confirmed from their dedicated pages):**
- **Cursor** — `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project); same JSON shape as the generic template, plus `X-Spectron-Context` header.
- **VS Code** — `.vscode/mcp.json`, `"servers"` key (not `"mcpServers"`), `"type": "http"`; a team-safe variant uses `"inputs"` with `promptString` so the API key is never committed.
- **Windsurf** — `~/.codeium/windsurf/mcp_config.json` (macOS/Linux) or `%USERPROFILE%\.codeium\windsurf\mcp_config.json` (Windows); uses `"serverUrl"`.
- **OpenCode** — `opencode.json`, `"mcp"` key, `"type": "remote"`, header value templated as `"Bearer {env:AGENT_MEMORY_API_KEY}"`.
- **Zed** — `~/.config/zed/settings.json`, `"context_servers"` key; legacy Zed versions without remote-URL support must bridge via `npx -y mcp-remote <url> --header "Authorization: Bearer <key>"`.
- **JetBrains (AI Assistant plugin: IntelliJ/PyCharm/WebStorm/etc.)** — Settings → Tools → AI Assistant → MCP, same generic JSON.
- **JetBrains Air** (standalone IDE) — Settings → AI | MCP Servers; **gotcha explicitly called out: "Air reads the bearer token from the configuration file itself, and tokens passed through environment variables are not supported,"** and the `install-mcp` npx helper does not support this client — manual config only.
- **Antigravity** — `~/.gemini/config/mcp_config.json`, uses `"serverUrl"` like Windsurf.

### C.5 — `surrealctl spectron` (admin CLI, not a server-runner)

This is a management CLI for Agent Memory contexts/keys/principals — **not** an install path for a self-hosted server:
```bash
surrealctl spectron context create research --region aws-euw1
surrealctl spectron key create "agent runtime" --context research > /run/secrets/spectron-key
surrealctl spectron principal create "research agent" --kind agent \
    --grant memory:read=/projects --grant memory:write=/projects/acme --context research
surrealctl spectron scoped-key create "read-only worker" \
    --principal "research agent" --grant memory:read=/projects/acme --context research
```
Per the fetch: this "does not run a self-hosted Spectron/Agent Memory server without Docker" — it only administers an *already-running* one (hosted or otherwise).

### C.6 — Limitations/gotchas the docs themselves call out (MCP-specific)

- `surreal mcp` has **no per-call auth** — root-equivalent for every tool call, explicitly flagged as unsuitable for untrusted exposure.
- The Agent-Memory MCP (`/mcp` on `spectrond`) uses a **static Bearer key only, no OAuth** (`--oauth no` in the installer is deliberate, not a workaround).
- Config **key names differ by client** for the same concept: `"url"` (most clients) vs. `"serverUrl"` (Windsurf, Antigravity) vs. `"servers"`/`"type":"http"` (VS Code) vs. TOML `[mcp_servers.<name>]` with `bearer_token_env_var` (Codex).
- JetBrains Air cannot read the token from an env var — file-only, a real secret-hygiene wrinkle for that one client.
- Version-number inconsistency across SurrealDB's own pages: one page says the hosted MCP needs SurrealDB **"3.1 or later"** to run direct SurrealQL, another (the SurrealMCP blog) says **"3.2.3"**. Our installed 3.2.0 clears the lower bar but not necessarily the higher one — worth a live check if we route raw-SurrealQL MCP calls through the hosted surface rather than local `surreal mcp`.
- No Codex-specific config example exists for either the hosted unified MCP (`mcp.surrealdb.com`) or the local embedded `surreal mcp` — only for the Agent-Memory-only surface. Both are inferable from Codex's own generic `mcp_servers` TOML schema, but SurrealDB's docs don't spell them out.

---

## D. Embedded/local mode

**Agent Memory cannot run as a pure embedded/in-process library.** Running it locally without Docker still requires running **two separate server processes**:

1. A running **SurrealDB** instance (`surreal start`), using a file-backed storage engine locally — `surrealkv://` or `rocksdb://`. Per `/docs/manage/self-hosted/configuration` and `/deployment-models`: RocksDB is "recommended for production on-disk data" ("High-performance LSM-tree key-value store optimised for high write throughput, SSD storage, and predictable persistence"); SurrealKV is explicitly **beta** ("SurrealDB's own LSM-backed storage engine... comparatively small configuration surface," "focused on embedded and local-first scenarios") — explicit guidance: **"For conservative production on-disk server deployments today, prefer RocksDB."** (Neither page's fetch surfaced the literal `surreal start` flag syntax — e.g. exact `--user`/`--pass`/`--bind` invocation — that is a documented gap to close with a direct CLI `--help` check, not a doc-fetch.)
2. A separate **`spectrond`** Agent Memory server process, configured via `SPECTRON_SURREALDB_URL`/`_USER`/`_PASS` to point at #1, exposing its own HTTP/MCP surface on port 9090.

None of the SDK pages (Python, JS/TS, Go, Kotlin, Swift, Dart, Elixir, Haskell) showed a local/embedded connection example — every SDK client constructor takes a **single remote `endpoint` + `context` + `api_key` triple** (e.g. `Memory(context=..., endpoint="https://api.memory.example", api_key=...)`), and none demonstrated `surrealkv://`, `rocksdb://`, `file://`, or `mem://` style local connection strings. That vocabulary belongs to plain SurrealDB, not to any Agent Memory SDK.

**No documented mechanism lets one agent/SDK client address both a local and a remote instance through a single configuration.** Each SDK client is bound to exactly one `endpoint`/`context`/`api_key`. Practically, "docs local, memory on VPS" means: **two separate connections** — one plain SurrealDB client/MCP pointed at the local file-backed instance for document/docstore work, and one Agent Memory SDK/MCP client pointed at the VPS `spectrond` endpoint for memory — not one unified server or one namespace-switching config.

The **filesystem-view** MCP surface (`integrations/surfaces/filesystem-view`) is a read-only virtual filesystem exposed over MCP `resources/list`/`resources/read` (`spectron://fs/documents/...`, `/entities/...`, `/sessions/...`), toggleable via `SPECTRON_FS_VIEW=false`, explicitly marked experimental: "Do not rely on the path conventions described here in production systems."

**Confirmed after the full MCP-cluster pass: no `spectrond` install/run command exists anywhere in the docs or the public `surrealdb` GitHub org (130+ repos enumerated via `gh repo list`, none named `agent-memory` or bare `spectron`; `github.com/surrealdb/agent-memory` is a genuine 404).** The docs describe deploying `spectrond` ("Deploy spectrond and call the HTTP API or generated client") but never show how — no binary download, no `cargo install`, no `docker run`, no winget/brew/npm package. This is a real, confirmed documentation gap, not something we failed to find. Practical implication: **we cannot self-host the Agent Memory application tier from public documentation as it stands today** — only its *client-facing contract* (REST + MCP + SDKs, all requiring a Bearer key against a `<context-host>`) is public. Whatever `data-surreal-phase1-t0-r1` or `surreal-case` on the VPS actually are, they are almost certainly **plain SurrealDB instances**, not a self-hosted `spectrond` Agent Memory tier — confirm this before assuming "memory on the VPS" can mean "point Agent Memory's SDK at the VPS."

### D.1 — What running locally without Docker actually looks like

For the **docs-local** half (plain SurrealDB, no Agent Memory layer at all — this is what our docstore kit already targets), the confirmed Docker-free command, verbatim from `/docs/build/ai-agents/mcp/embedded` (shown there for `surreal mcp` but the same flag set applies to a plain `surreal start`):

```bash
surreal mcp --user root --pass secret --ns main --db main memory
```
or, for persistence:
```bash
surreal mcp --user root --pass secret --ns main --db main rocksdb://path/to/data
```

`surrealkv://<path>` is the equivalent form for the `surrealkv` engine. Storage-engine guidance (`/docs/manage/self-hosted/{configuration,deployment-models}`): **RocksDB is recommended** ("High-performance LSM-tree key-value store optimised for high write throughput, SSD storage, and predictable persistence... For conservative production on-disk server deployments today, prefer RocksDB"); **SurrealKV is explicitly beta** ("SurrealDB's own LSM-backed storage engine... comparatively small configuration surface," positioned for "embedded and local-first scenarios"). Given our desktop is a single, low-concurrency dev box, SurrealKV's simplicity is attractive, but the docs' own recommendation leans RocksDB for anything we'd call "production," even locally.

**Neither this fetch pass nor the earlier one surfaced the exact `surreal start` (as opposed to `surreal mcp`) flag syntax** (bind address, TLS flags, etc.) — that's a one-command gap to close locally with `surreal start --help` / `surreal mcp --help` rather than another doc fetch, since the flags are shared between subcommands per the CLI reference page.

For the **memory-on-VPS** half, since no `spectrond` install path is published, the realistic options are: (a) treat the VPS as **plain SurrealDB only** and build our own docstore kit's memory semantics on it directly (bypassing Agent Memory entirely — see Section E), or (b) sign up for SurrealDB's **hosted Agent Memory Cloud** product and point the SDK/MCP at that hosted `<context-host>` instead of self-hosting anything (this contradicts the "memory may live on a VPS" framing but is the only path the docs actually support today), or (c) contact SurrealDB about the **enterprise self-hosted/air-gapped tier**, which is the only place a `spectrond` install artifact is mentioned at all ("Self-hosted and air-gapped deployments... available for qualifying enterprise engagements").

---

## E. Fit map — our kit vs. what SurrealDB actually ships

Note: this session could not locate our docstore kit's SurrealQL source files or the ADR-0056..0058 documents on disk (searched `E:/AI_Workspace` and the `probata` working directories; no `ADR-005[678]`, `fn::remember`, `fn::supersede_memory`, or `decision_log` matches — the probata search also timed out on the full-tree grep). This map is therefore built from the kit's description as given in this task's brief, cross-referenced against the documented SurrealDB Agent Memory behavior above. If the kit's actual SurrealQL differs from the description, re-run this comparison against the real source.

| Our kit element | Replace / Wrap / Leave-to-us | Why, with doc evidence |
|---|---|---|
| **`document` / `chunk` tables** | **Wrap-able, if we adopt Agent Memory** | Agent Memory's `document` + `knowledge_chunk` tables (`reference/data-model-and-schema`) do the same job — async ingest, chunking, embedding, keywording, dedup by content hash. But adopting it means giving up our own SurrealQL control over chunking/embedding params: everything is exposed only via `POST /documents` + `ingestion_profile` config, not `DEFINE INDEX`/`DEFINE ANALYZER` we write ourselves. |
| **`entity` / `mentions` tables** | **Wrap-able, with a caveat** | Their `entity` table is keyed by exact composite `[normalised_type, normalised_name]` — **no fuzzy/embedding entity resolution at write time**: "Name variants are separate entities" (`cookbooks/patterns/spoiler-safe-narrative-memory`); `same_as` links must be created manually. If our `mentions` design does its own resolution/linking logic, that is **not** replaced — it is exactly the gap Agent Memory admits it doesn't fill. |
| **`memory` table with `status: active\|superseded\|retracted`** | **Leave to us / partially analogous, not equivalent** | Agent Memory's lifecycle is **four states**, not three: active → superseded (automatic, reconciler-driven on same-source conflict) → retired (≈ our `retracted`; soft, excluded from reads, history kept) → purged (hard delete, no analog in our three-state model). Crucially, **their supersession is automatic** (a side effect of extraction/reconciliation, confidence-floor gated at 0.7 by default), not an application-called guarded write. Our explicit `fn::supersede_memory` gives us something they don't expose as a callable primitive — direct application control over *when* a supersession happens, rather than trusting an opaque confidence-floor heuristic. |
| **`fn::remember` guarded write** | **Loosely wrapped by `POST /facts` / `remember()`, but semantics differ** | Their `/facts` endpoint does run "every write... through the reconciler (calibration, supersession, uncertainty on conflict)" automatically — similar *intent* to a guarded write. But it is a fixed pipeline (heuristics → fast LLM → strong LLM escalation) we cannot swap out for our own guard logic; the only levers are `infer` mode (`full`/`triples`/`preview`/`none`) and per-stage model selection. If `fn::remember`'s "guard" encodes business rules specific to our domain, that logic has no home in their pipeline and must stay ours. |
| **`decision_log` events table** | **Leave to us — no equivalent exists** | This is the clearest non-overlap. SurrealDB has a `decision_trace` graph node type (`reference/data-model-and-schema`, `architecture/traces-and-evolution`) that records what the reconciler considered per write, but it is explicitly framed as an internal **ranking/consolidation feedback signal**, not a stable, queryable, application-facing audit log schema — no page documents a guaranteed field contract for `decision_trace` the way an application-owned `decision_log` table would have. If we need a stable, versioned "why did this change" event log for our own tooling/compliance, we must keep building and owning it — do not assume `decision_trace` is a drop-in substitute. |
| **HNSW + BM25 hybrid retrieval (raw SurrealQL, our own `DEFINE INDEX`)** | **Not replaceable if we need SurrealQL-level control; Agent Memory wraps a version of it but hides the levers** | Agent Memory's hybrid search *is* HNSW + BM25 fused via RRF (`score(d) = Σ 1/(k+rank_i(d))`, default `k=60`) — conceptually identical to what we'd hand-build. But **no page exposes raw SurrealQL KNN operator syntax (literal K/EF values), no `DEFINE INDEX ... HNSW` with M/EFC build params, and no BM25 k1/b tuning** — everything is API-parameter only (`mode`, `k`, `rrf_k`). If our kit's design depends on literal K/EF control inside a KNN clause (as the task brief flags as a known contradiction risk), **that control point does not exist in Agent Memory at all** — it only exists if we talk to SurrealDB directly with our own SurrealQL, which is exactly what our kit already does. This is the single strongest argument for keeping our own hybrid-search implementation rather than adopting Agent Memory's. |
| **Post-filter vs. filter-inside-KNN concern** | **Resolved differently than expected — worth knowing either way** | Agent Memory states explicitly: "Filters are applied before scoring, so they do not affect the ranking of results that pass through" (`retrieve/hybrid-search`) — i.e., they claim **pre-filtering**, not the classic post-filter-after-KNN pitfall. But this is asserted, not demonstrated with SurrealQL — we cannot verify it is implemented as a true in-KNN filter (SurrealDB's `<|K,EF|>` operator with a `WHERE` inside the operator) versus a candidate-pool pre-filter before the ANN pass (which is a different, also-valid technique but has different recall trade-offs at low K). If our kit's own concern was specifically about SurrealQL's `<|K,EF|>` filter placement, that's an orthogonal, lower-level question this page doesn't answer either way. |
| **Changefeeds** | **No overlap found** | None of the ~110 Agent Memory pages fetched mentioned SurrealDB changefeeds (`DEFINE TABLE ... CHANGEFEED`) in relation to Agent Memory's supersession/versioning mechanism — versioning is implemented at the **row level** (`valid_from`/`valid_until` + chain pointers), not via changefeed-driven event capture. If ADR-0056..0058's "governed analytical projection" design leans on changefeeds as the mechanism for building a projection/read-model, that is entirely our own construction — Agent Memory does not use or expose changefeeds anywhere in its documented architecture. |
| **ADR-0056/57/58 — "SurrealDB as governed analytical projection + walk-memory runtime, PostgreSQL authority"** | **Leave to us — architecturally incompatible with adopting Agent Memory wholesale** | Our ADRs presuppose **PostgreSQL is the authority** and SurrealDB is a downstream, governed projection. Agent Memory's whole design assumes the *opposite* relationship: **SurrealDB (via `spectrond`) is the primary, authoritative store** for memory — there is no documented "project from an external system of record into Agent Memory" pattern; ingestion is either document upload or direct `remember()`/fact writes, always treated as first-class, not a projection target. Adopting Agent Memory as our memory layer would mean **inverting our ADR's authority model** for the memory subsystem specifically (SurrealDB/Spectron becomes authoritative for memory, Postgres stays authoritative for everything else) — a real architectural decision, not a drop-in. If we want to preserve Postgres-as-authority even for memory, we are necessarily in "leave to us" territory for the whole product, using at most raw SurrealDB (not Agent Memory) as our own projection target — which is what our kit already does. |

**Overall read:** Agent Memory is a legitimate, fairly sophisticated product for the "give an agent a memory API and don't think about the internals" use case, but it is **not a wrapper we can drop under our existing kit** — it is a **parallel, opinionated implementation of the same problem space**, closed-source at the application-tier, with no exposed SurrealQL-level control and no public self-host path. Every piece of our kit that depends on us controlling SurrealQL directly (HNSW/BM25 tuning, explicit supersession functions, a stable decision-log schema, Postgres-authoritative projection) is **left to us** either because Agent Memory doesn't expose the control point at all, or because adopting it would require inverting our ADR's authority model. The pieces that are genuinely **wrap-able** (raw document ingest, basic entity/fact storage) are also the least differentiated parts of our design — the ones a shared closed-source service is least risky to depend on and easiest to swap back out of later.

---

## F. Recommended setup: "docs local, memory on VPS"

Given Section D's confirmed finding (no public `spectrond` install path) and Section E's fit-map conclusion, there are two honest options. Both are laid out; pick based on whether "memory on the VPS" must mean self-hosted Agent Memory specifically, or can mean "our own SurrealDB-backed memory design, on the VPS."

### Option 1 — Recommended: skip Agent Memory, use plain SurrealDB on both ends

This matches our ADR-0056..0058 authority model and our existing docstore kit, and needs nothing undocumented.

1. **Local desktop (docs):** run `surreal.exe` (already installed via winget, just needs to be on PATH or referenced by absolute path) as a file-backed local instance:
   ```bash
   surreal start --user root --pass <local-pass> --bind 127.0.0.1:8000 rocksdb://C:/path/to/docs-data
   ```
   (RocksDB per SurrealDB's own recommendation for anything beyond throwaway dev; SurrealKV is beta and fine to evaluate in parallel but not the default choice per the docs.) Confirm exact `surreal start` flags with `surreal start --help` since no page gave the literal flag list.
2. **Local MCP for docs, Docker-free, stdio:**
   ```json
   { "mcpServers": { "surrealdb-docs": {
       "command": "C:/path/to/surreal.exe",
       "args": ["mcp", "--user", "root", "--pass", "<local-pass>", "--ns", "docs", "--db", "docs", "rocksdb://C:/path/to/docs-data"]
   }}}
   ```
   in Claude Code's `.mcp.json` and the equivalent Codex `[mcp_servers.surrealdb-docs]` stdio entry.
3. **VPS (memory):** run a second SurrealDB instance (or reuse `data-surreal-phase1-t0-r1` if its namespace/db can be dedicated to memory) with our own docstore kit's schema (`document`/`chunk`/`entity`/`mentions`/`memory` tables, `fn::remember`, `fn::supersede_memory`, `decision_log`, our HNSW+BM25 hybrid via raw `DEFINE INDEX`) — i.e., build our own "Agent Memory," not SurrealDB's, on that instance.
4. **Remote MCP for memory:** since our own memory instance is plain SurrealDB (not `spectrond`), we cannot use the Agent-Memory-only MCP surface (C.4) — instead expose it either via the **hosted unified MCP** pattern is not applicable (that's SurrealDB Cloud-specific) or, more simply, run `surreal mcp` **on the VPS itself**, bound appropriately, and reach it from the desktop over the tailnet as an HTTP/stdio bridge, OR use the standalone (archived-but-functional) `surrealmcp` binary's HTTP mode (`surrealmcp start --bind-address 0.0.0.0:8000`) if we want a proper networked MCP endpoint rather than SSH-tunneled stdio, accepting it is unmaintained. A cleaner option: don't put MCP on the VPS at all — keep both MCP servers local (stdio to a locally-installed `surreal` binary that opens a **remote WebSocket connection** to the VPS instance, i.e. `surreal mcp` connecting to `ws://100.91.190.107:8000/rpc` instead of a local file path — this is directly supported since `surreal mcp`'s `[PATH]` argument accepts remote endpoints, not just local paths, per the standalone `surrealmcp`'s own `--endpoint ws://...` precedent, though this specific point should be verified with `surreal mcp --help` since the fetched pages only showed local-path examples).
5. **Result:** two MCP server entries in Claude Code / Codex — `surrealdb-docs` (local, stdio, file-backed) and `surrealdb-memory` (either local-stdio-to-remote-DB or a VPS-hosted MCP endpoint reached over the tailnet) — both using surface #1 (`surreal mcp`), never surface #2 (archived) or #3b (undocumented self-host).

### Option 2 — If self-hosted "Spectron" Agent Memory specifically is required

Not achievable from public documentation today. The only paths that actually work are: (a) SurrealDB's **hosted** Agent Memory Cloud (`quickstarts/hosted` — `SPECTRON_URL`/`SPECTRON_CONTEXT_ID`/`SPECTRON_API_KEY`, a `sk-ctx-...` key, contexts provisioned via their Cloud API) — this means the VPS isn't actually hosting memory, SurrealDB's cloud is; or (b) contact SurrealDB sales about the enterprise self-hosted/air-gapped tier, which is the only place a `spectrond` distribution is mentioned. Do not spend engineering time hunting for a hidden Docker image or binary — three independent fetches (marketing page, GitHub org listing, direct 404 check on the plausible repo name) confirm it isn't publicly published.

### Ten-item gotcha list

1. **"SurrealDB Agent Memory" ≠ a SurrealDB feature.** It's a separate, closed-source Rust application tier (`spectrond`) that requires its own running SurrealDB as a backend — not something `surreal start` gains by upgrading versions.
2. **`github.com/surrealdb/agent-memory` does not exist.** Don't waste time looking for it again; confirmed 404 twice plus a full org repo listing.
3. **The Docker image in our inherited kit (`surrealdb/surrealmcp:latest`) is from an *archived* repo.** It still runs, but SurrealDB's own docs now route everyone to `mcp.surrealdb.com` or the built-in `surreal mcp` instead — treat the kit's Docker assumption as outdated, not just Docker-specific.
4. **Three MCP surfaces share overlapping names and config shapes** (`url` vs `serverUrl`, `mcpServers` vs `servers`, `X-Spectron-Context` header) — a config copy-pasted from the wrong client's doc page will silently fail to authenticate rather than error clearly in some clients.
5. **`surreal mcp` runs every call as root with no per-call auth** — fine for our single-operator desktop, not something to expose beyond localhost/tailnet without our own auth layer in front.
6. **Vector/BM25 tuning knobs (K/EF, HNSW M/EFC, BM25 k1/b) are not exposed by Agent Memory's API at all** — only by raw SurrealQL against plain SurrealDB. If we need that control, we are by definition not using the Agent Memory product for that piece.
7. **SurrealKV is still beta**; SurrealDB's own guidance says prefer RocksDB "for conservative production on-disk server deployments today" — don't pick SurrealKV by default without a specific reason (e.g., wanting its smaller config surface for a throwaway local dev store).
8. **No SDK (any of 8 languages checked) supports a local/embedded connection string** — every Agent Memory SDK client is constructed with a single remote `endpoint`+`context`+`api_key` triple. "Local Agent Memory" via an SDK is not a thing; only `surreal mcp`/plain SurrealDB SDKs connect locally.
9. **Supersession in Agent Memory is automatic and confidence-gated (default floor 0.7), not application-triggered** — if two of our systems both write similar facts, "the later write wins" only when confidence clears the floor; otherwise both persist and an `uncertainty` record appears instead. This is a materially different failure mode than an explicit `fn::supersede_memory` call.
10. **Version-number and category-count inconsistencies exist within SurrealDB's own docs** (3.1 vs. 3.2.3 minimum for hosted-MCP SurrealQL execution; three-value vs. six-value "memory category" enum between the glossary and three other pages) — don't treat any single page as fully authoritative; cross-check load-bearing numbers against a second page or a live probe before depending on them.

---

## G. URLs read, with what each covers

Confirmed fetched successfully (no 404s reported by any subagent) across all clusters:

**Welcome/architecture/mental-model (18):** welcome/what-is-surrealdb-agent-memory, welcome/how-it-works, welcome/why-agentic-memory, welcome/accuracy-promise, architecture/principles-and-goals, architecture/eight-pillars-and-categories, architecture/coherence-retrieval-and-tiers, architecture/tri-temporal-model, architecture/traces-and-evolution, architecture/surface-security-and-models, architecture/glossary, mental-model/two-layer-architecture, mental-model/memory-categories, mental-model/memory-lifecycle, mental-model/contexts-and-scope, mental-model/sessions-and-turns, mental-model/provenance-and-traceability, memory-and-knowledge.

**Reasoning/reference (13):** reasoning/extraction-pipeline, reasoning/reconciliation-and-supersession, reasoning/authority-hierarchy, reasoning/cross-layer-linking, reasoning/temporal-validity, reasoning/instructions-and-uncertainties, reference/data-model-and-schema, reference/configuration, reference/cli, reference/management-api, reference/errors, reference/glossary, reference/agents, reference/rest-api.

**Ingest/sessions/retrieve (13):** ingest/authoritative/uploading-documents, ingest/authoritative/bulk-import, ingest/authoritative/multimodal-content, ingest/authoritative/knowledge-nodes, ingest/experiential/remember, sessions/chat-sessions, sessions/creating-sessions, sessions/adding-turns, sessions/state-and-diffs, retrieve/recall, retrieve/hybrid-search, retrieve/keywords-and-bm25, retrieve/graph-traversal.

**Quickstarts/surfaces/SDKs/self-hosted (20):** quickstarts/embedded, quickstarts/hosted, quickstarts/surrealdb-cloud, integrations/surfaces/embedded-library, integrations/surfaces/filesystem-view, integrations/surfaces/rest, integrations/sdks/python, integrations/sdks/javascript-and-typescript, integrations/sdks/go, integrations/sdks/kotlin, integrations/sdks/swift, integrations/sdks/dart, integrations/sdks/elixir, integrations/sdks/haskell, reference/sdk-javascript, reference/sdk-python, reference/sdk-kotlin, reference/sdk-swift, /docs/manage/self-hosted/configuration, /docs/manage/self-hosted/deployment-models.

**Operations/tuning/cookbooks (23):** operations/reflect, operations/forget, operations/profiles, tuning/models-per-stage, tuning/caching-and-invalidation, tuning/ontology-grounding, cookbooks/build/coding-agent-with-project-memory, cookbooks/build/customer-support-agent, cookbooks/build/long-running-research-agent, cookbooks/build/multi-agent-shared-memory, cookbooks/build/personal-ai-assistant, cookbooks/migrate/from-langmem, cookbooks/migrate/from-mem0, cookbooks/migrate/from-vector-store, cookbooks/migrate/from-zep, cookbooks/patterns/adding-memory-to-existing-app, cookbooks/patterns/event-driven-game-client, cookbooks/patterns/historical-and-archaeological-data, cookbooks/patterns/knowledge-grounded-agents, cookbooks/patterns/reflection-loops, cookbooks/patterns/spoiler-safe-narrative-memory, cookbooks/patterns/stateful-workflows, cookbooks/patterns/user-memory-in-chat.

**Framework/AI-SDK/automation/observability/voice + marketing (33):** integrations/frameworks/{agno,autogen,camel-ai,crewai,eve,google-adk,hermes,langchain,llamaindex,mastra,openai-agents,openclaw,pydantic-ai,strands-agents}, integrations/ai-sdks/{cloudflare-workers-ai,tanstack-ai,vercel-ai-sdk}, integrations/automation/{n8n,zo-computer}, integrations/observability/{agentops,respan}, integrations/voice/{elevenlabs,gradium,livekit}, cookbooks/patterns/spoiler-safe-narrative-memory (dup), surrealdb.com/agent-memory, surrealdb.com/use-cases/agent-memory, surrealdb.com/blog/agent-memory-is-a-filesystem-building-one-on-surrealdb, surrealdb.com/blog/introducing-surrealdb-3-0--the-future-of-ai-agent-memory, surrealdb.com/blog/announcing-the-mastra-integration-for-surrealdb-and-agent-memory, /docs/build/ai-agents/agent-skills, /docs/build/ai-agents/ai-frameworks, /docs/build/ai-agents (index).

**MCP cluster (25, all fetched — no 404s except where noted):**
- `integrations/mcp-server/install` — only install method for the Agent-Memory MCP surface is the `install-mcp` npx helper or manual JSON; no binary/cargo/npm/brew/winget/Docker for this surface specifically.
- `integrations/mcp-server/tools-reference` — the 7 Agent Memory MCP tools (remember/recall/context/reflect/forget/upload/inspect) with full param lists.
- `integrations/mcp-server/coding-assistants/claude-desktop-and-code` — Claude Code plugin + env-var path, Claude Desktop JSON config.
- `integrations/mcp-server/coding-assistants/codex` — Codex `config.toml` `[mcp_servers.spectron]` block.
- `integrations/mcp-server/coding-assistants/cursor` — Cursor `~/.cursor/mcp.json`, cloud vs self-hosted URL swap.
- `integrations/mcp-server/coding-assistants/vscode` — VS Code `.vscode/mcp.json`, plus a team-safe `promptString` input variant.
- `integrations/mcp-server/coding-assistants/windsurf` — Windsurf config path + `"serverUrl"` key quirk.
- `integrations/mcp-server/coding-assistants/opencode` — OpenCode `opencode.json`, `"type":"remote"`, templated env header.
- `integrations/mcp-server/coding-assistants/zed` — Zed `settings.json` `"context_servers"`, plus legacy `mcp-remote` bridge.
- `integrations/mcp-server/coding-assistants/jetbrains` — JetBrains AI Assistant plugin MCP settings.
- `integrations/mcp-server/coding-assistants/jetbrains-air` — standalone Air IDE; env-var tokens NOT supported, file-only.
- `integrations/mcp-server/coding-assistants/antigravity` — `~/.gemini/config/mcp_config.json`, `"serverUrl"` key.
- `/docs/reference/cli/surrealdb-cli/commands/mcp` — the built-in `surreal mcp` subcommand, exact syntax/flags, stdio-only, root-auth caveat.
- `/docs/reference/cli/surrealctl/commands/spectron` — admin CLI for Agent Memory contexts/keys/principals; not a server-runner.
- `/docs/build/ai-agents/mcp` — the hosted unified MCP (`mcp.surrealdb.com`) overview, auth modes, version requirement.
- `/docs/build/ai-agents/mcp/claude` — Claude Code/Desktop config for the hosted unified MCP.
- `/docs/build/ai-agents/mcp/cursor` — Cursor config for the hosted unified MCP, secret-hygiene gotcha.
- `/docs/build/ai-agents/mcp/embedded` — **the Docker-free local `surreal mcp` path**, exact commands, Cursor/VS Code/Claude Desktop configs.
- `/docs/build/ai-agents/mcp/examples` — example natural-language prompt catalogue, including Agent Memory prompts.
- `surrealdb.com/mcp` — marketing/landing page distinguishing hosted MCP, embedded MCP, and Agent Memory as three offerings.
- `github.com/surrealdb/surrealmcp` — **archived** repo; byte-exact README (via `gh api` against a pre-archival commit) with the full Docker-free `cargo install --path .` path, all env vars, three transports.
- `github.com/surrealdb/agent-memory` — **confirmed 404, does not exist.**
- `surrealdb.com/blog/surrealmcp-a-managed-mcp-server-for-ai-agents` — announces the hosted-MCP pivot away from per-instance config.
- `surrealdb.com/blog/introducing-surrealmcp` — older announcement of the original (now-archived) SurrealMCP, Docker-only examples.
- `/docs/spectron` — naming/branding page: "Spectron" is the retained internal name across binaries/env vars/headers even though docs/SDKs say "Agent Memory."

**Pages NOT fetched at all:** none identified. `/docs/llms.txt` was fetched first specifically to enumerate the complete `/docs/agent-memory/*` sitemap (110 paths) plus related MCP/agents docs, and every path it listed was subsequently fetched by one of the six research clusters. The only two URLs that returned a genuine 404 were `github.com/surrealdb/agent-memory` (confirmed non-existent, not a fetch failure) — no SurrealDB-hosted doc page failed to load.
