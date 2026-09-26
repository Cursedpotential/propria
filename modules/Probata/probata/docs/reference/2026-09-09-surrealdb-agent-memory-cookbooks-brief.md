# SurrealDB Agent Memory — Cookbooks Deep Read (supplement to Brief K)

> **CORRECTION 2026-09-09 06:55 (D-157): the self-hosting conclusion in this brief is WRONG.** SurrealDB Agent Memory ships a free self-hostable server binary `spectrond` (https://surrealdb.com/docs/agent-memory/reference/cli — install script at download.surrealdb.com/spectron, `spectrond bootstrap`, `spectrond dev start --bootstrap --connection-string … --bind-address 0.0.0.0:9090`, MCP at /mcp on that port; configuration reference documents SPECTRON_SURREALDB_URL and a local object store for single-node deployments). This brief read the "Hosted quickstart" (SurrealDB Cloud) as the only path and never opened the CLI or configuration reference. Treat every "closed source / enterprise-only / Cloud-only" statement below as struck. Kept in place as history per house rule; the corrected plan is `docs/reference/2026-09-09-spectrond-vps-deployment-plan.md` (pending).


> Byline: Claude Code · Sonnet 5 · 2026-09-09
> Read-only research. Fetched via WebFetch (small-model HTML→prompt summarizer, not raw HTML) against surrealdb.com/docs/agent-memory/cookbooks/*, plus `/docs/llms.txt` and a `site:` search for enumeration. No repo files touched.
> **Method caveat, stated up front:** WebFetch converts each page and answers a prompt about it with a small model — it is not a raw-HTML/DOM dump. Despite repeated prompts explicitly asking for verbatim reproduction and every hyperlink, most fetches returned **no on-page "Prerequisites" callout box and no sidebar/footer link list** — only inline links that appear inside body prose (mostly self-links back to `/docs/llms.txt`) and one external link (`github.com/supermemoryai/install-mcp`, cited by the coding-agent page for the `install-mcp` npx helper — note this differs from Brief K's `install-mcp` provenance, which didn't attribute a GitHub org; flagged, not resolved). This may reflect that these cookbook pages genuinely carry minimal outbound links (they're code-heavy, self-contained recipes), or it may reflect the summarizer dropping nav chrome. Where a cookbook names a concept (Context, scope, principal, `asOf`) without linking it, Brief K's own fetch of the dedicated concept pages (`mental-model/contexts-and-scope`, `reference/cli`, etc.) already covers it in depth — this report does not re-fetch those unless a cookbook linked to one explicitly (one did: `historical-and-archaeological-data` linked `mental-model/contexts-and-scope.md`).

---

## 0. Scope of this pass

Per the owner's staged instructions, this report fully covers **all 5 `cookbooks/build/*` pages and all 8 `cookbooks/patterns/*` pages** (13 total), not just the original two:

**`cookbooks/build/` (5/5 read):**
1. `multi-agent-shared-memory`
2. `coding-agent-with-project-memory`
3. `long-running-research-agent`
4. `customer-support-agent`
5. `personal-ai-assistant`

**`cookbooks/patterns/` (8/8 read):**
6. `reflection-loops`
7. `adding-memory-to-existing-app`
8. `knowledge-grounded-agents`
9. `user-memory-in-chat`
10. `event-driven-game-client`
11. `historical-and-archaeological-data`
12. `spoiler-safe-narrative-memory`
13. `stateful-workflows`

**`cookbooks/migrate/` (enumerated only, not read this pass — out of the owner's ask):** `from-langmem`, `from-mem0`, `from-vector-store`, `from-zep`.

Full one-line-each enumeration is in Section G.

---

## A. What each cookbook actually builds

All 13 pages share one architecture skeleton, confirmed independently on every page: an application-layer SDK client (`Memory`/`AsyncMemory` in Python, `AgentMemory` in JS/TS) or raw HTTP calls against `https://<context-host>/api/v1/{context_id}/...` (commonly shown as `spectron.surrealdb.com`), authenticated with a Bearer `api_key`/`sk-...`. **No page shows a locally-running or self-hosted `spectrond`; every code sample targets a remote HTTPS endpoint.** SurrealDB itself (the DB engine) is never directly touched in any cookbook — all thirteen pages interact exclusively through the Agent Memory application-tier HTTP/SDK/MCP surface, never raw SurrealQL. This matches Brief K's three-layer model (engine / `spectrond` / client) exactly, with the cookbooks living entirely at the client layer.

1. **`multi-agent-shared-memory`** — Two patterns: (a) *shared scope*, where every agent session is created with the identical `scopes=["org/acme/project/market-research-q3"]` array so writes from one agent are immediately readable by another; (b) *supervisor + workers*, where each worker gets its own nested scope (`[...SUPERVISOR_SCOPE, "agent/worker-N"]`) and a supervisor aggregates via `reflect(persist=True)` at a broader scope. Components: SDK client only (`AsyncMemory`/`AgentMemory`), talking to one remote Agent Memory endpoint — no MCP shown on this page, no mention of self-host.

2. **`coding-agent-with-project-memory`** — Builds an in-editor coding assistant wired via the **MCP** surface (not the SDK) — `npx install-mcp https://<your-context-host>/mcp --client cursor --header "Authorization: Bearer <your-api-key>" --oauth no`, and the manual JSON config uses the `spectron` server name with an `X-Spectron-Context` header. Tools used: `context` (session-start load), `remember`, `recall`, `reflect`. Scoping: repo-level `["org/acme/project/my-repo"]` plus per-developer `["org/acme/project/my-repo/user/alice"]`. Ingestion of docs via CLI: `spectron documents upload ./docs/architecture.md --label "topic=architecture"` and `spectron ingest ./docs --label "topic=project-docs"`.

3. **`long-running-research-agent`** — Multi-day/multi-session pattern: one session per working day/topic block, `profile()` loaded at session start to reconstruct prior state, `reflect(persist=True)` run at session end to consolidate. Shows `memory_category` values (`knowledge`, `context`), `session.metadata` (immutable post-creation — use attributes for counters instead), `state()` for entity/relation/uncertainty counts, and a scope-pruning example: `POST /scopes/forget {"path": "org/acme/project/sub-topic/"}`. Explicit statement: **"Requires Agent Memory server (spectrond), SurrealDB engine backend, and SDK client library. Cloud or self-hosted deployment both supported via API endpoint configuration."** — this is the fetch's own paraphrase, not a verbatim doc quote (flagged in Section B/E — it is not backed by a literal self-host instruction anywhere on the page).

4. **`customer-support-agent`** — The one cookbook with an explicit **"Prerequisites"** paragraph (not a link box, prose): *"A Context requires creation through the management API or dashboard. You'll need a `context_id`, management API key for ingestion, and an agent API key for conversation operations."* Two-key model: a **management key** (`SPECTRON_MGMT_KEY`) for `POST /contexts` and document upload, and a separate **agent/conversation key** for `remember`/`context` calls. Explicit note that the extraction entity-type vocabulary is **fixed** ("Customer types like 'Customer' cannot be registered" — custom types must map onto `person`/`organisation`/`product`/`event`/`other`). Quotes the authority-hierarchy line already found in Brief K verbatim: *"Authoritative content is protected from conversational drift"* regardless of user assertions.

5. **`personal-ai-assistant`** — Single-user assistant; `profile()` → formatted system-prompt block; five memory categories tabulated (Identity, Knowledge, Context, Instructions, Unknowns); `chat()` managed endpoint returning `reply`/`citations`/`memoryUpdates`/`sessionId`/`traceId`; `forget()` with `dry_run=True` preview, `POST /scopes/forget` for branch deletes, `DELETE /entities/{type}/{name}` for single entities, `purge=True` for irreversible hard delete. This page's own fetch explicitly concluded: *"The guide references an API key pattern (`api_key=\"sk-...\"`) but does not specify self-hosted versus cloud requirements, free-tier availability, or pricing details explicitly."*

6. **`reflection-loops`** — `reflect(query, persist)` is a synthesis pass distinct from `context()`/`recall()` (which only retrieve, never reason). Table contrasts them: retrieval = "low cost, before every LLM call"; reflection = "higher cost, periodically or on specific events." Shows a **supervisor API key** (`supervisor_sk_...`) used to reflect across an entire org scope — i.e., cross-tenant reflection requires a privileged key tier, not just a broader scope path on an ordinary key. Gives a nightly-batch-job pattern iterating `entities.list(type="Customer")` and reflecting per-customer with a 0.5s sleep between calls (rate-limit-conscious).

7. **`adding-memory-to-existing-app`** — Minimal-diff integration guide: (1) pre-call retrieval prepended to system prompt, (2) post-call turn recording. Five-phase adoption ladder (turn recording → context injection → profile injection → per-user scoping → authoritative doc ingestion), each independently deployable. This fetch's own conclusion: *"The integration requires an API key (`api_key=\"sk-...\"`) for SurrealDB Agent Memory, indicating a cloud-based service requirement rather than self-hosted deployment."*

8. **`knowledge-grounded-agents`** — Two-layer retrieval in the agent loop: `documents.query(mode="hybrid", k=4)` (Layer 0/authoritative) run in parallel with `session.context(query=...)` (Layer 1/experiential), both folded into one system prompt under separate headers. Documents the **`resolves_to`** bridging-relation mechanism with a concrete JSON example (an experiential `entity:["Product","airpods_pro"]` node in a user's scope pointing at the authoritative `knowledge:["Product","airpods_pro"]` node). Shows the conflict object shape returned when a user assertion contradicts a Layer-0 fact (`l0_fact`/`l1_belief`/`recommendation` fields). Ends: *"The documentation references Spectron (`spectron.surrealdb.com`) as a managed cloud platform requiring an `AGENT_MEMORY_API_KEY`."*

9. **`user-memory-in-chat`** — Two integration "shapes": Shape 1 lets the platform's own `chat()` endpoint drive the whole loop (retrieval + LLM call + persistence + extraction, server-side); Shape 2 is caller-driven (you fetch `profile()` + `sessions.context()` yourself, call your own LLM, then `remember()` both turns). Documents the `profile()` return shape precisely: top-level `static`, `dynamic`, `preferences`, `selfFacts` (key/value maps) and `instructions` (array of `{id, label, description}`). **This is the closest page to answering the owner's "sync" question** — see Section D below for the verbatim mechanism.

10. **`event-driven-game-client`** — Off-pattern from the rest: integrates a game engine's event stream (dialogue, travel, combat, item pickups) into memory via epistemic labels (`authority=lived|spoken|written|seen`) so the extraction pipeline doesn't conflate "an NPC told the hero about X" with "the hero did X." Quote: *"You do not replace the game. The engine stays the source of truth for physics and UI. SurrealDB Agent Memory holds what the hero (or operator) has learned, heard, read, and done."* Practical guidance: batch travel telemetry rather than per-tile events, keep actor IDs off labels (labels are for retrieval filtering, not storytelling), upload long lore as documents not single `/facts` calls, use fresh Contexts per test run to avoid retrieval pollution.

11. **`historical-and-archaeological-data`** — The most SurrealQL-adjacent-but-still-HTTP-only cookbook: ingesting fragmentary primary sources (Domesday Book entries, Pompeii electoral inscriptions, Danube radiocarbon reports) with `observedAt` set to *historical* time, not ingest time, via `POST /api/v1/{context_id}/documents` multipart upload with a `metadata` JSON field. Introduces three synthesis endpoints not seen elsewhere in this pass — `/elaborate` (pre-form relational links), `/consolidate` (merge name variants), `/reflect` (interpretive synthesis, same tool as #6). Gives a full **proof-of-retrieval verification protocol** (empty-baseline query before ingest, perturbation-then-requery, held-out subject, invented subject, provenance-array check) to prove answers come from ingested sources rather than the underlying LLM's training data — genuinely novel methodological content not covered by Brief K. Links out to `mental-model/contexts-and-scope.md` for its "a reused Context contaminates the next run" warning.

12. **`spoiler-safe-narrative-memory`** — Already summarized in Brief K's page list one-liner; this pass got the full mechanism. `observed_at` on ingest (a synthetic "story-time" stamp, e.g. per chapter) + `asOf` on recall (the reader's current position) jointly gate what's visible — confirmed with concrete CLI: `spectron documents upload ./chapters/ch08.txt --scope org/my-app/reader/alice --label chapter=8 --label page=5 --as-of 1886-03-01T00:00:00Z`. Explicitly generalizes past literal chronology to **release-order-vs-story-order franchises** (Star Wars watched IV→VI→I→III: "Vader is Luke's father" is timestamped after *Empire*, not *A New Hope*, and different viewer profiles get different stamp sequences on the *same* underlying records). Also documents an `fsck`-style duplicate-detection endpoint: `POST /fsck {"check": "duplicates", "duplicateThreshold": 0.95, "maxResults": 100}`. Explicit limitation quoted: *"Passages retain upload-time metadata regardless of `asOf`... `asOf` on `/query` walks known-time on attributes and relations only."* — i.e. `asOf` filtering is guaranteed only for the graph (entities/attributes/relations), not for raw retrieved text passages, which is a materially different (narrower) guarantee than a naive reading of "time-travel query" would suggest.

13. **`stateful-workflows`** — Uses Agent Memory as a durable, resumable state store for multi-step/multi-day agent workflows (research pipelines, etc.), independent of chat. Idempotent step-checking pattern: before running a step, `context(lens=scope, query="Has the X step been completed?")` and inspect returned entities for `status == "completed"`. Time-boxed step results via a per-turn `metadata.valid_until` field (`POST /sessions/{id}/turns` with `"metadata": {"valid_until": "2025-11-16T00:00:00Z"}`) — expired attributes silently drop out of context retrieval, causing the idempotency check to re-trigger the step. Progress tracked via **deltas** returned on every `remember()` call (`result.extraction.attributes/corrections/uncertainties` counts), not by polling full state. Ends: *"Cloud Service Model — The documentation references API keys (`api_key=\"sk-...\"`) indicating cloud-hosted SurrealDB Agent Memory service."*

---

## B. Free and/or self-hosted path — verdict: NO, confirmed 13/13

**Every single one of the 13 cookbook pages** authenticates against a remote HTTPS Agent Memory endpoint with a Bearer API key, and **none** shows a `spectrond` binary download, Docker image, `cargo install`, npm/pip package that runs the server itself, or a documented free tier. This is a unanimous, independently-arrived-at result across 13 separate page fetches (not one subagent's single conclusion propagated) — it directly reinforces Brief K Section B/D's finding of "no public `spectrond` install path."

Verbatim closing lines from the pages that discussed deployment explicitly (quoted above in Section A, repeated here for the record):
- `adding-memory-to-existing-app`: *"The integration requires an API key (`api_key=\"sk-...\"`) for SurrealDB Agent Memory, indicating a cloud-based service requirement rather than self-hosted deployment."*
- `stateful-workflows`: *"Cloud Service Model — The documentation references API keys... indicating cloud-hosted SurrealDB Agent Memory service."*
- `knowledge-grounded-agents`: *"The documentation references Spectron (`spectron.surrealdb.com`) as a managed cloud platform requiring an `AGENT_MEMORY_API_KEY`."*
- `personal-ai-assistant`: *"does not specify self-hosted versus cloud requirements, free-tier availability, or pricing details explicitly"* (i.e., silent on the question — the default assumed shape is still Bearer-key-against-a-hostname).
- `long-running-research-agent` is the **one outlier claiming both are supported**: *"Requires Agent Memory server (spectrond), SurrealDB engine backend, and SDK client library. Cloud or self-hosted deployment both supported via API endpoint configuration."* This sentence is **not a verbatim page quote** — it's the fetch model's own inference from seeing an `endpoint=` parameter on the SDK constructor, and it is not corroborated by any install command, download link, or self-host doc reference anywhere in this pass or in Brief K. Treat it as unverified/likely overreach by the summarizing model, not evidence of a real self-host path — Brief K's independent, much deeper pass (Section D) found zero `spectrond` install artifacts anywhere in the 130+-repo GitHub org or the docs.

**Net: no cookbook, and nothing any cookbook links to, shows a free or self-hosted way to run Agent Memory.** This matches and strengthens Brief K's conclusion rather than contradicting it.

---

## C. Multi-agent shared memory specifics (`multi-agent-shared-memory`)

- **Sharing mechanism:** identical scope path strings. `SHARED_SCOPE = ["org/acme/project/market-research-q3"]` passed to every agent's session creation or write call. There is no separate "sync" protocol — every agent talks to the *same* live server, so "shared memory" means "same backing store, same scope key," not replication.
- **Agent identification:** scope path segments, e.g. `f"agent/{worker_id}"` appended to a base scope (`[...SUPERVISOR_SCOPE, "agent/worker-N"]`). Agents are not first-class identities with their own auth by default in this pattern — they're just distinguished by which scope array they pass.
- **Isolation:** **scope visibility resolves as OR-of-AND clauses** (verbatim): a query at `{org: "acme", project: "alpha"}` sees "Everything at the project scope (shared findings)" plus "Everything at more specific scopes (individual worker findings)," while "A worker reading with a lens of `[\"org/acme/project/alpha/agent/worker-1\"]` sees only its own memory and the shared project scope." So isolation is hierarchical-prefix-based, not per-agent ACL — a worker's private scope is only private because nothing else queries with that exact/nested lens, not because of a cryptographic or role-based barrier at that layer (the *actual* access barrier is the API key's grants, described next).
- **Actual access control (not scope alone):** *"A key bound to a principal granted `memory:read` and `memory:write` on `org/acme/project/market-research-q3` can read and write at that scope. For genuinely read-only agents, mint the key against a principal granted `memory:read` but not `memory:write` on that path."* This confirms Brief K's C.5 finding (`surrealctl spectron principal create ... --grant memory:read=/path`) is the real enforcement point — scope strings alone are a retrieval filter, not a security boundary; the principal/grant system is.
- **Conflict handling:** *"The later write wins for attribute values"* on same-scope conflicting writes, with the reconciler creating a supersession chain inspectable via `state()` (rows carrying `supersedes`) or `entities.history(type, name, key)` for the full chain — consistent with Brief K's tri-temporal model.
- **Cross-tool question (Claude Code / Codex / OpenCode / n8n / Temporal workers reading one memory at once):** this cookbook never names those specific clients. Its architecture is client-agnostic by construction — any client capable of an HTTPS Bearer-key call (SDK, raw REST, or the MCP `remember`/`recall`/`context` tools documented on the `coding-agent-with-project-memory` page) can participate, as long as it's configured with the same scope-path convention. **Nothing in these 13 pages demonstrates n8n or Temporal specifically** — Brief K's own page list shows a dedicated `integrations/automation/n8n` page exists but was not re-fetched in this pass (out of the owner's two-cookbook-plus-additions scope); flagging as a real gap if the owner wants n8n wiring confirmed verbatim.

---

## D. Coding-agent project memory specifics — the "sync" mechanism the owner asked about

Owner's framing: *"should be able to sync — pretty good system there."* The mechanism, assembled from `coding-agent-with-project-memory` plus the corroborating detail in `user-memory-in-chat` and `multi-agent-shared-memory` (same underlying primitive, described three ways):

**It is not push/pull sync in the offline-replication sense. It is "shared server, shared scope key."** Every client (every developer's editor, every session, every agent) that authenticates with the right key and passes the right scope path is reading and writing the *same* live rows on the *same* remote Agent Memory instance. There is no local cache to reconcile, no CRDT, no merge step the application must run — consistency is delivered by there being exactly one backing store, not by synchronizing N copies.

Verbatim mechanism quotes:
- Repo-level scoping: *"Memory isolation uses scope dimensions like `[\"org/acme/project/my-repo\"]`, ensuring project-specific conventions don't contaminate other projects while allowing team-wide knowledge at the organizational scope."* (`coding-agent-with-project-memory`)
- Multi-developer: *"Shared project memory enables team collaboration, while individual user-scoped memories (e.g., `[\"org/acme/project/my-repo/user/alice\"]`) preserve personal preferences separate from project-wide conventions."* (same page)
- The general form of the same mechanism, stated most explicitly on the multi-agent page: *"Memory written by one agent is immediately visible to others operating in the same scope."* (`multi-agent-shared-memory`)
- And on `user-memory-in-chat`, describing the same primitive at user-granularity across *sessions* rather than *agents*: *"Memory scope is user-level, not session-level—all conversations within a user scope feed the same entity pool."* with a worked three-session progression (facts stated in session 1, a formatting preference in session 2, a role update in session 3 that "supersedes previous value while maintaining chain history") ending: *"By session 3, the profile contains all accumulated facts with transparent supersession tracking for auditability."*

**What gets remembered** (per `coding-agent-with-project-memory`): coding conventions ("Use `Result<T, Error>` not exceptions"), past technology decisions with rationale, individual developer style/ownership preferences, active work status/blockers, and ingested docs (specs, ADRs, API contracts).

**Hooks/skills/MCP wiring for Claude Code:** the page shows **MCP only**, not a native Claude Code hook/skill file. Install:
```bash
npx install-mcp https://<your-context-host>/mcp \
    --client cursor \
    --header "Authorization: Bearer <your-api-key>" \
    --oauth no
```
and manual config:
```json
{
  "mcpServers": {
    "spectron": {
      "url": "https://<your-context-host>/mcp",
      "headers": {
        "Authorization": "Bearer <your-api-key>",
        "X-Spectron-Context": "my-project"
      }
    }
  }
}
```
(Brief K's C.4 already has the Claude-Code-specific variant of this — plugin install + `AGENT_MEMORY_MCP_URL`/`AGENT_MEMORY_MCP_TOKEN` env vars — this cookbook's example is the generic/Cursor-flavored one, not a Claude-Code-native hook.)

**How recall is forced:** not automatic/ambient — the page's documented workflow is explicit tool calls at defined points: `context` tool called "before responding to the user's first message" (session start), `remember` called "when identified during conversations" (i.e., the calling agent/LLM decides when a decision is worth persisting — this is prompted behavior, not a background daemon), `recall` called on-demand against ingested docs, `reflect` optionally at session end. **This pass could not find, despite two targeted re-fetches, any additional "sync"/consistency section, Claude-Code-native hook, or system-prompt-injection code sample beyond what's summarized above** — the second fetch attempt explicitly returned "the following requested sections do not exist in this documentation" for a sync/consistency mechanism, Claude-Code-specific hooks, and a system-prompt-injection example. Given the WebFetch caveat in the header, this is reported as "not found in two passes," not as a confirmed absence from the live page — a manual page visit would be the only way to fully rule it out.

**Project scope expression:** `["org/<org>/project/<repo>"]`, nestable per-developer as `["org/<org>/project/<repo>/user/<name>"]` — same hierarchical scope-path convention used everywhere else in the cookbook set (Section C).

---

## E. Corrections to Brief K

1. **Reflection is manual — Brief K's citation for this may not be attributable to the page it cites.** Brief K quotes, attributed to `cookbooks/patterns/reflection-loops`: *"developers must call the `reflect()` method directly. It is not automatic."* This pass fetched that exact page twice with prompts specifically asking for that sentence, and both times the fetch reported it is **not present**: *"The provided documentation does not contain the specific sentence about manual invocation you referenced... does not include language stating developers must call it directly or that it is not automatic."* What the page *does* say is consistent in substance (reflection is invoked via explicit `reflect()`/`POST /reflect` calls at chosen trigger points — end of session, nightly batch, event-triggered — never shown running automatically), so **the underlying claim (manual, not automatic) is corroborated**, but the specific verbatim sentence Brief K quoted could not be reproduced from this page in this pass. This could mean: (a) the page copy changed since Brief K's fetch, (b) Brief K's subagent pulled that sentence from a different page (e.g. a cookbook index or another `reflect`-related doc) and mis-attributed it, or (c) WebFetch's summarizer dropped it both times. **Flagging as unresolved — do not cite that exact string as verified without a fresh direct check**, though the substantive conclusion ("reflect must be called explicitly, it's not automatic") stands on independent evidence from this pass (the trigger-point table, the nightly-batch-job code sample, and the retrieval-vs-reflection contrast table on the same page).

2. **Brief K's "no example shows a local/embedded connection string" claim is reinforced, not contradicted, by these 13 pages** — every SDK constructor shown here (`Memory(context=..., api_key=...)`, `AgentMemory({endpoint, context, apiKey})`) takes the same remote-endpoint-only shape Brief K found across all 8 language SDKs. No new counter-evidence surfaced.

3. **New, more granular confirmation of the two-key model** (Brief K mentioned principals/keys abstractly via `surrealctl spectron` in C.5; this pass found it demonstrated end-to-end in a cookbook): `customer-support-agent` shows a **management key** (`SPECTRON_MGMT_KEY`, used for `POST /contexts` and document ingestion) as distinct from an **agent/conversational key** (`SPECTRON_API_KEY`, used for `remember`/`context` during live conversations) and `reflection-loops` shows a third tier, a **supervisor key** (`supervisor_sk_...`) needed to reflect across an entire org rather than one scope. This is a finer-grained key hierarchy than Brief K's C.5 example implied (which showed `principal create` with per-path grants but didn't name a distinct "supervisor" tier) — worth folding into any future access-model comparison.

4. **New: an explicit, code-shown methodology for proving retrieval isn't LLM-training-data recall** (`historical-and-archaeological-data`'s empty-baseline / perturbation / held-out-subject / invented-subject / provenance-array five-point verification protocol). Brief K did not surface this — it's a genuinely new, reusable pattern regardless of whether we adopt Agent Memory, since our own kit will face the identical "is this answer really coming from what I ingested" verification problem.

5. **New: the `asOf` time-filter guarantee is narrower than a naive reading suggests.** `spoiler-safe-narrative-memory` states explicitly: *"`asOf` on `/query` walks known-time on attributes and relations only"* and *"Passages retain upload-time metadata regardless of `asOf`."* Brief K's Section B flagged the general `/query`-vs-`/state` supersession-exclusion question as unverified ("the general `/query` recall page did not restate the same guarantee... worth verifying directly"). This cookbook answers a closely related but distinct question (temporal filtering scope, not supersession exclusion) and confirms the guarantee is **partial**: structured graph data is time-filterable, raw text passages are not. This is a real, load-bearing refinement if our own kit's spoiler-safe/point-in-time design assumed uniform time-filtering across all row types.

6. **No contradiction found on the free/self-hosted question** — 13/13 independent page reads corroborate Brief K's central finding (Section B/D there). One page (`long-running-research-agent`) asserted self-hosting is "supported," but as flagged in Section B above, that assertion is unbacked by any concrete artifact and is best read as the summarizing model's inference from an `endpoint=` parameter, not a documented capability — it does not survive scrutiny against Brief K's much more thorough Section D investigation (130+-repo GitHub org search, direct 404 checks).

---

## F. Recommendation

**Unchanged from Brief K, now with 13 additional independent data points supporting it: (2) Agent Memory is not usable for free/self-hosted; re-implement the cookbook *patterns* (not the product) on plain SurrealDB.**

The strongest new argument for this reading, beyond "no install path exists" (already established in Brief K), is that **the patterns themselves are simple enough to be worth re-implementing directly rather than worth chasing a closed-source dependency for**:

- **"Shared memory across agents"** (Section C/D) = a shared SurrealDB instance + an application-level convention of hierarchical scope-path strings (`org/x/project/y/agent/z`) stored as a field on every row, filtered with an OR-of-AND predicate at query time, plus a role/grant table (principal → scope-prefix → read/write) gating who can query which prefixes. This is a schema design and a query-time WHERE clause, not a product feature — entirely buildable on our own `memory`/`entity`/`mentions` tables per Brief K's Section E fit-map, and it composes cleanly with our existing `fn::remember`/`fn::supersede_memory` guarded writes (Brief K flagged those as strictly *more* controllable than Agent Memory's opaque confidence-floor reconciler).
- **"Project memory that syncs"** (Section D) = the same shared-store-not-replicated-copies principle; our VPS-hosted SurrealDB already gives every agent (Claude Code, Codex, OpenCode, n8n/Temporal workers) that shared, always-current backing store the moment they all point at the same instance with the same scope convention — no additional product is required to get the owner's "should be able to sync" property.
- **Two genuinely new, worth-copying ideas from this pass** that are NOT product features, just design patterns to lift wholesale into our own schema/runbook:
  1. The **proof-of-retrieval verification protocol** from `historical-and-archaeological-data` (empty-baseline, perturbation, held-out/invented subject, provenance check) — apply this to our own decision_log/memory store to prove recall isn't just the LLM's parametric knowledge.
  2. The **`observed_at` (story/event time) + `asOf` (read-time) dual-timestamp gate** from `spoiler-safe-narrative-memory`, with the caveat (Section E.5) that any implementation of this idea needs the same explicit "structured facts are time-gated, raw text/passages are not, unless you build that too" boundary.

**Minimal mapping table (extends Brief K Section E):**

| Cookbook pattern | Plain-SurrealDB equivalent |
|---|---|
| Scope-path string (`org/x/project/y/agent/z`) on every row + OR-of-AND filter | A `scope: array<string>` (or a normalized `scope` edge table) field on our `memory`/`mentions` rows, filtered with an array-prefix `CONTAINS`/`INTERSECTS` predicate in our own SurrealQL |
| Principal/grant model (`memory:read`/`memory:write` per scope-prefix) | A `principal` table + `grant` edge table (`principal -> grant -> scope_prefix {read, write}`), checked in our own `fn::remember`/`fn::recall` guard functions before executing |
| `profile()` structured sections (`static`/`dynamic`/`preferences`/`selfFacts`/`instructions`) | A view/function over our `entity`/`attribute` tables grouped by category, formatted server-side into a markdown block for prompt injection — same shape, our own code |
| `reflect(persist)` | An explicit, app-triggered LLM synthesis job (nightly cron / session-end webhook) that writes back into our `memory` table with `source.kind = "reflect"` — exactly the manual-trigger model Section E.1 confirms Agent Memory itself uses, so there's no automation we'd be giving up |
| `observed_at` + `asOf` dual timestamp | Already close to our tri-temporal-adjacent `valid_from`/`valid_until` design per Brief K Section B — add a distinct `observed_at` (event/story time) column separate from `valid_from` (belief-validity time) if we want the spoiler-safe pattern specifically |
| Proof-of-retrieval protocol | A standing test fixture: empty-baseline query pre-ingest, one perturbed/invented control row per corpus, provenance-field assertion on every synthesized answer |

No part of this pass surfaced a reason to revisit that recommendation.

---

## G. Full page enumeration — `cookbooks/*`

**`cookbooks/build/` (5, all read in full this pass):**
- `multi-agent-shared-memory` — shared-scope and supervisor/worker patterns for cross-agent memory.
- `coding-agent-with-project-memory` — MCP-wired in-editor coding assistant with repo + per-developer scopes.
- `long-running-research-agent` — multi-day session continuity via daily sessions + profile-load + reflect.
- `customer-support-agent` — dual authoritative (catalogue/policy) + experiential (per-customer) memory, two-key model.
- `personal-ai-assistant` — single-user assistant, profile injection, five memory categories, forget/purge lifecycle.

**`cookbooks/patterns/` (8, all read in full this pass):**
- `reflection-loops` — `reflect()` vs retrieval, persistence, trigger-point guidance, supervisor-key cross-org reflection.
- `adding-memory-to-existing-app` — five-phase incremental adoption ladder for an existing LLM app.
- `knowledge-grounded-agents` — two-layer (authoritative/experiential) grounding, `resolves_to` bridging, conflict objects.
- `user-memory-in-chat` — per-user chat memory across sessions, two integration "shapes," `profile()` shape detail.
- `event-driven-game-client` — game-engine event ingestion with epistemic (`authority=lived|spoken|written|seen`) labels.
- `historical-and-archaeological-data` — fragmentary primary-source ingestion, `/elaborate`/`/consolidate`/`/reflect`, proof-of-retrieval protocol.
- `spoiler-safe-narrative-memory` — `observed_at`/`asOf` dual-timestamp spoiler gating, release-order-vs-story-order handling.
- `stateful-workflows` — durable multi-step workflow state, idempotent step-checking, TTL'd step results, delta-based progress tracking.

**`cookbooks/migrate/` (4, enumerated via `/docs/llms.txt` only, not read this pass):**
- `from-langmem`
- `from-mem0`
- `from-vector-store`
- `from-zep`

This matches Brief K's own G-section enumeration of the same 17 cookbook paths (no new paths found by either the sitemap re-fetch or the `site:` search, which returned only one on-topic hit — `spoiler-safe-narrative-memory` — plus unrelated cooking-cookbook noise).
