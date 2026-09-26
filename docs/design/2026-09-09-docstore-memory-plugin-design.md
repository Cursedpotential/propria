---
type: blueprint
domain: [docs, memory]
status: proposed
supersedes: []
---

# The docstore + memory plugin — two SurrealDB MCPs, one plugin, progressive disclosure (2026-09-09)

> _Byline: Claude Code · Fable 5.1 · 2026-09-09 06:35 EDT_
> _Owner directive 06:28: "two Surreal MCPs, one for docs, one for agent memory. All document recalls or stores or creations or ADR updates get exposed as a tool or a skill or a plugin. Maybe make a plugin with both MCPs attached with all of the tools, with progressive disclosure. Show me you know how to use your thinking skills and work that process through to a good idea." Rulings in force: D-154 (memory = self-hosted SurrealDB on the VPS, all agents), D-155 (docs local, embedded, in place), D-156 (seven types, domain tags, status field, hooks repointed, ingest-driven execution)._

**STATUS: DESIGN — for owner review before the plugin is built. Steps 1 (local engine) and 2 (schema) of the execution sequence are independent of this document and proceed.**

## 0 · Route (thinking-model router)

Domain: architecture. Problem type: decide + create. Router defaults: **Reversibility** first, then **Systems**. Owner-mandated additions: **Graph thinking**, **Socratic**. The "all the tools" versus "small context" tension is a textbook contradiction, so **TRIZ** for that one. Close with a **pre-mortem**. Each pass below produces an artifact, not a name-drop.

## 1 · Reversibility — which choices are one-way doors

| Choice | Type | Consequence |
|---|---|---|
| Embedding model and dimension (baked into every stored vector and the HNSW index) | **One-way** | Choose once. `nvidia/nemotron-3-embed-1b`, 2048 dims, is the live embedder `ccc` already uses on this desktop and the key is present. Changing it later means a full re-embed of both stores. |
| The seven `doc_type` values and the five `status` values (schema ASSERT lists reject anything else) | **One-way in practice** | Ruled by D-156. Adding a type later is an ALTER, dropping one is a data migration. |
| Two stores instead of one (docs local, memory remote) | **One-way** by ruling (D-154, D-155) and by nature: a memory row citing a doc id crosses a database boundary that cannot be joined | Design consequence: memory rows carry the document's stable id and source path as strings, never a live record link. |
| MCP transport (native HTTP `/mcp` on each server) | **Two-way** | Both are the same `surreal start` binary; the plugin holds only two URLs. A stdio launcher or a hosted endpoint could replace either without touching skills. |
| Plugin packaging (Claude Code plugin; Codex `config.toml` mirror) | **Two-way** | Skills and hooks are files; agents are Markdown. |
| Which VPS instance hosts memory (`surreal-case`) | **Two-way** | A namespace move is an export and import. |

Decision: spend care on the first three, move fast on the rest.

## 2 · Systems — the loop the plugin must break

The documents disaster has one generator: writes that bypass any registry. Three feedback loops must close inside the plugin, or the store becomes one more stale copy.

```
[write a doc] --(no registration)--> [store unaware] --> [agent reads stale] --> [writes another doc] --> loop
```

Balancing loops the plugin installs:

1. **Write gate.** A `PostToolUse` hook on `Write|Edit` under `docs/**` injects one line: "this file is unregistered until `docs_register` or `docs_new_version` runs." The librarian skill's definition of done is a store id, not a file.
2. **Read gate.** `UserPromptSubmit` injects a three-line reminder: search the store before reading files; hits carry `status`; a `superseded` hit is not an answer.
3. **Presence gate.** `SessionStart` preflight pings both `/health` endpoints and prints failure loudly. No silent filesystem fallback, because that fallback is the old loop.

Leverage point: the store's `supersedes` edge is the definition of done for any new version. The old hooks that spawn snapshot and compact files are retired, and the handoff skill writes the handoff into the store and mirrors it to `docs/handoff/` (D-156 item 4).

## 3 · Graph — what the tools actually are

Documents are nodes; the value is in edges. Every tool is an edge operation or a node lookup. Nothing else earns a tool.

| Edge / node op | SurrealQL function (docs store) | Skill that discloses it |
|---|---|---|
| find nodes by meaning + keywords, scoped by type, domain, status | `fn::docs_search(query, vec, type?, domain?, status?)` (RRF over HNSW + BM25) | `docs` |
| read one node with its status and successors | `fn::docs_get(id)` | `docs` |
| create a node (register a file: type, domain, status, path) | `fn::docs_register(path, type, domains, status, body)` | `docs-write` |
| new version in place: create node, `new -supersedes-> old`, flip old to `superseded`, regenerate mirror path | `fn::docs_new_version(old_id, body)` | `docs-write` |
| amend a decision: append-only row or banner, `decides` edge to the docs it closes | `fn::decision_amend(adr_or_dnumber, banner, closes[])` | `decisions` |
| open / close a todo item, with the `decides` or `implements` edge that closes it | `fn::todo_open(item)`, `fn::todo_close(id, evidence)` | `todo` |
| write a handoff node and mirror it | `fn::handoff_write(body)` | `handoff` |
| what is open, what is current, what is stale | `fn::open_work()`, `fn::current_decisions()`, `fn::stale_candidates()` | `docs`, `reconcile` |
| provenance of a node | `fn::provenance(id)` | `docs` |

| Memory op | SurrealQL function (memory store, VPS) | Skill |
|---|---|---|
| remember a claim, guarded against near-duplicates, scoped | `fn::remember({kind, claim, scope, evidence, confidence})` | `memory` |
| recall claims by meaning + keywords, scoped | `fn::recall(query, vec, scope)` | `memory` |
| supersede or retract a claim (never overwrite, never delete) | `fn::supersede_memory(old, new)`, `fn::forget(id, reason)` | `memory` |
| consolidate a session's episodes into durable claims | `fn::reflect(scope, since)` | `memory` (reflection-loop pattern from the cookbooks) |
| scope = path string `probata/<domain>/<agent>` (the cookbooks' sharing mechanism, reproduced) | field on every row; index on it | all |

The MCP servers themselves expose only SurrealDB's generic tools (`query`, `run`, `select`, `use`, `info`, and so on). The plugin never adds a custom MCP server: every domain operation is a `DEFINE FUNCTION` in the store, invoked through the native `run` tool. That keeps the MCP layer off-the-shelf and puts the whole vocabulary in the schema, where the ASSERT lists enforce it.

## 4 · Socratic — who calls what, and why each piece exists

| Question | Answer |
|---|---|
| Who writes documents? | Only the **librarian** (Sonnet) through `docs-write`. Other agents hand it text and get back an id. |
| Who writes memory? | Any agent through `memory`, but `fn::remember` refuses near-duplicates unless told to supersede. |
| Who reconciles? | The **reconciler** (Opus, the ceiling per the owner's rule), interactively, batch-adjudicated, never batch-writing. Runs the D-156 ingest and the stale-candidate passes. |
| Why two MCP servers and not one with two databases? | Because they are two machines by ruling. One MCP server fronts one SurrealDB process. |
| Why `run` on named functions instead of raw `query`? | Raw SurrealQL from an agent is the KNN-post-filter trap (empty results with status OK) and the freelance-write trap. Functions carry the scope filters inside the KNN predicate and are the only thing the reader role may execute. |
| Why does Codex get the same thing? | Same two HTTP endpoints in `config.toml`; the skills become prompt files; there are no hooks, so Codex is weaker on enforcement and the docs say so. |
| Why do n8n and Temporal workers get in? | They speak HTTP to `/sql` or `/rpc` with a worker principal; they never need MCP. |
| What does the owner see? | One status line per session start: both stores up, index ready, N unverified documents, M open items. |

## 5 · TRIZ — the contradiction, and progressive disclosure as its resolution

Contradiction: **complete coverage** (every operation is a tool, the owner's requirement) versus **small context** (an agent that loads 40 tool schemas and 8 skill bodies every turn is slower and dumber). TRIZ principle 1, segmentation, and principle 15, dynamics: split the surface into layers that load only when reached.

| Layer | What loads | When | Size |
|---|---|---|---|
| 0 | Three lines from `SessionStart`: both stores' health, the six skill names, "search before you read." | every session | ~60 tokens |
| 1 | Skill descriptions only (`docs`, `docs-write`, `decisions`, `todo`, `handoff`, `memory`, `reconcile`) | every turn, by the harness | ~200 tokens |
| 2 | One `SKILL.md` body: the two or three function calls that skill wraps, exact `run` syntax, the definition of done | when the skill matches | ≤ 60 lines each |
| 3 | `references/` under the skill: full function signatures, the schema for that type, worked examples, gotchas | when the skill body points there | on demand |
| 4 | The MCP servers' generic tool schemas | deferred by the harness's tool search; never eagerly loaded | 0 until used |

Only the raw `query` tool is denied to the main thread by the plugin's settings; everything else stays reachable, one layer at a time. That is the whole of "all the tools with progressive disclosure": the tools exist in the schema, the skills are the table of contents, and the harness's deferral does the rest.

## 6 · The plugin, concretely

```
plugins/docstore/                      (name provisional; component tier, lowercase, owner may rename)
  .claude-plugin/plugin.json
  .mcp.json            docs   -> http://127.0.0.1:8462/mcp   (basic auth from .docstore/.env)
                       memory -> http://100.91.190.107:8471/mcp (basic auth; needs SURREAL_MCP_ALLOWED_HOSTS on surreal-case)
  settings.json        deny raw `query` for the main thread; librarian not forced as main thread (kit's default rejected: too strong)
  hooks/hooks.json     SessionStart preflight · UserPromptSubmit read-gate · PostToolUse(Write|Edit docs/**) write-gate · PreCompact -> handoff skill
  bin/                 preflight.sh · remind-retrieve.sh · flag-doc-write.sh · compact-handoff.sh
  skills/              docs · docs-write · decisions · todo · handoff · memory · reconcile   (each SKILL.md ≤ 60 lines + references/)
  agents/              docstore-librarian (sonnet) · docstore-reconciler (opus) · memory-curator (sonnet)
codex/                 AGENTS.md section + config.toml [mcp_servers.docs] [mcp_servers.memory] + prompts/
```

What is reused off the shelf: the SurrealDB binary and its native MCP on both ends; the kit's schema files (adapted), functions, hooks scripts, agent definitions, dedupe reference. What is written new: the D-156 type and status lists, the domain-tag field, the scope field and the reflect function on the memory side, the seven skill bodies, and the ingest mapping loader (the kit's CocoIndex pipeline cannot run here; the library is absent and its collector wiring was never finished).

## 7 · Pre-mortem — it is October and the plugin made things worse

1. **Two stores drifted: memory claims cite documents that were superseded.** Fix in design: memory rows store the doc id and source path as strings, and `fn::recall` on the memory side is always followed by `fn::docs_get` on the doc side in the `memory` skill's instructions; a stale doc turns the claim into a `stale_candidate`.
2. **Agents freelanced raw SurrealQL and got empty KNN results with status OK.** Fix: raw `query` denied to the main thread; reader principal can only execute functions; every function keeps its scope filter inside the KNN predicate, verified once with `EXPLAIN FULL` at build.
3. **The VPS was unreachable and agents silently fell back to files.** Fix: preflight fails loudly, the `memory` skill says "memory unavailable, do not invent" and nothing else.
4. **Hooks became noise.** Fix: each hook prints at most three lines, and the read-gate line is suppressed when the last tool call was already a store search.
5. **The tool count grew anyway.** Fix: no custom MCP server ever; a new operation is a new `DEFINE FUNCTION` plus a line in an existing skill, or it does not exist.
6. **Codex agents wrote documents without registering them.** Fix: Codex gets the write skill as a prompt and the librarian role as the only write path; the reconciler's stale-candidate pass catches the rest weekly.
7. **The redeploy of `surreal-case` to set the allowed-hosts variable disrupted case work.** Fix: that redeploy is its own owner-gated step, done once, verified with a single `/health` and `/mcp` initialize before any agent config points at it.

## 8 · Build order (each step has a check; nothing proceeds on a failed check)

1. Local engine up — **done 06:30**: `127.0.0.1:8462`, health 200, version 3.2.0, store at `probata/.docstore/`.
2. Schema bound to D-156 and applied to the local store: seven types, five statuses, `domains` array field, `EMBED_DIM=2048`, functions from §3. Check: `INFO FOR DB` lists every table, index, function; `EXPLAIN FULL` on `fn::docs_search` shows the scope predicate inside the KNN scan.
3. Mapping table from the inventory (546 rows). Check: row count equals inventory rows minus the 15 quarantined.
4. Ingest with the mapping loader. Check: per-type and per-domain counts match the mapping; any domain under 1 percent of chunks gets a recall spot-check.
5. Regenerate the seven-folder mirror from the store with frontmatter; dead-link count before and after. Check: count did not rise; `INDEX.md` is generated.
6. Memory namespace and schema on `surreal-case`; `SURREAL_MCP_ALLOWED_HOSTS` set (owner-gated redeploy). Check: `/mcp` initialize from this desktop with the tailnet Host succeeds.
7. Plugin skeleton, hooks, skills, agents; `claude plugin validate`. Check: `/mcp` lists both servers; each skill fires by name; a full write-register-search loop works on both stores.
8. Retire the two old hooks; the handoff skill writes to the store. Check: a forced compaction produces no file under `docs/` root.

## 9 · Owner decisions in this design (not yet ruled)

- Plugin name (provisional `docstore`).
- The librarian is **not** forced as the main-thread agent (the kit's default). Recommendation: keep it as a callable agent; force nothing.
- Reconciler on Opus (judgment-heavy), librarian and curator on Sonnet.
- Codex parity: HTTP MCP entries if the installed Codex supports `url`; otherwise a stdio bridge. Verified at step 7.
