---
title: "surrealdb"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# surrealdb

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

SurrealDB for Propria: SurrealDB's official agent skills (synced from surrealdb/agent-skills), a deployments skill for our own instances (surreal-docs, surreal-case, surreal-intake), and SurrealDB's own MCP tools for those instances served through ContextForge.

Source: `E:/AI_Workspace/plugins/plugins/surrealdb`. Version: `1.0.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `surql-formatter`

Use after editing or writing any `.surql` file, or when the user asks Claude to format, lint, or check SurrealQL files.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surql-formatter/SKILL.md:1>) · SHA-256 `ec5cc9ed5f9cbd2f484106ba1922f8d9bcb6bc27c87f4090bef88d9c10fcd736`

### `surrealdb`

(surrealdb) SurrealDB for Propria: the official SurrealDB skills (SurrealQL, functions, performance, vector search, CLI, SurrealKit, JS and Python SDKs, docs lookup, .surql formatting) plus our own deployments (surreal-docs, surreal-case, surreal-intake): how to reach each one, what each holds, who may write, and the SurrealQL tools served through ContextForge. Entry point / router: read this first, then load one member. Triggers: surrealdb, surrealql, surql, surreal, surrealkit, hnsw, record id, docstore database, case store, agent memory database, intake graph. Members: surrealdb-deployments, surrealql, surrealql-functions, surrealql-performance, surrealdb-vector, surrealdb-cli, surrealkit, surrealdb-js, surrealdb-python, surrealdb-docs, surrealdb-agent-memory-docs, surql-formatter.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb/SKILL.md:1>) · SHA-256 `ed40a5678584507dead03dbce1a05e2136112868d1114ec77f95a292518644de`

### `surrealdb-agent-memory-docs`

Look up official SurrealDB Agent Memory (formerly Spectron) documentation from surrealdb.com as markdown: quickstarts, mental model, ingest, sessions, recall, reasoning, SDKs, framework integrations, MCP server, REST API, CLI, and configuration. Use when building with or answering questions about SurrealDB Agent Memory. Read-only; sends only search terms and doc paths.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-agent-memory-docs/SKILL.md:1>) · SHA-256 `24c95f3a6f846a6ee00283a53a0d9bc2fe8d251a79d9d47e18a718f0cec0b19f`

### `surrealdb-cli`

Use the `surreal` command-line binary to run a SurrealDB server (in-memory, RocksDB, SurrealKV, or TiKV), open an interactive or piped SQL REPL, import and export databases, check server readiness, upgrade the binary, repair storage, and manage SurrealML models. Use this skill whenever users run, connect to, back up, restore, or operate SurrealDB from the terminal. Triggers: surreal start, surreal sql, surreal import, surreal export, surreal isready, surreal upgrade, surreal fix, surreal ml, SurrealDB CLI, surreal server.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-cli/SKILL.md:1>) · SHA-256 `b6c2a781893e9f7da4191aaedbc769d76834c693220d9d47162f3c174e32d0af`

### `surrealdb-deployments`

(surrealdb) OUR SurrealDB instances and how each one is used: surreal-docs (Docstore documentation index), surreal-case (Family Court Toolkit case store plus the shared agent memory), surreal-intake (Consignatio Intake graph) and the Surrealist admin page. Addresses, namespaces and databases, where credentials live, who writes each store, the access rules, the SurrealQL tools served through ContextForge, and the sq.py query helper. Use before connecting to, querying, changing, backing up or debugging any of our SurrealDB stores.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-deployments/SKILL.md:1>) · SHA-256 `a4b73b9d9aef886d4b829f079bbfab1f3f9670804980b13df71f255133b5284e`

### `surrealdb-docs`

Look up official SurrealDB documentation (SurrealQL, SDKs, CLI, deployment) from surrealdb.com as markdown. Read-only; sends only search terms and doc paths.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-docs/SKILL.md:1>) · SHA-256 `eac86bcefd2e52fc7bd5fc227b6887bee11d19d05e6e1e13d5ad91a202709ba3`

### `surrealdb-js`

Using SurrealDB from JavaScript and TypeScript with the official surrealdb SDK, covering connecting (WebSocket/HTTP and embedded engines), authentication, CRUD, parameterized queries, and live queries. Use when connecting to SurrealDB from Node, Deno, Bun, or the browser, using the surrealdb npm package, or performing CRUD and real-time operations from JS/TS code. Triggers: surrealdb JS, surrealdb.js, new Surreal(), db.query, db.create, db.live, TypeScript SDK, npm i surrealdb.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-js/SKILL.md:1>) · SHA-256 `0f4cc947dbc2609559e30cf2fbefe756a474900e3c9d5f93566feafb060fe0b8`

### `surrealdb-python`

Using SurrealDB with the Python SDK, covering both client/server mode (WebSocket) and embedded mode (in-memory or file-based). Use when connecting to SurrealDB from Python, using the surrealdb Python package, running SurrealDB embedded without a server, or performing CRUD operations from Python code. Triggers: surrealdb Python, Surreal(), AsyncSurreal(), Python SDK, embedded SurrealDB, mem://, file://.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-python/SKILL.md:1>) · SHA-256 `11b9cbf1b6aed48204e7abb84c852c85fa9c6f56b272a644947e5443c080ec9b`

### `surrealdb-vector`

Vector search with SurrealDB using HNSW indexes, KNN queries, and similarity scoring. Use when creating vector indexes, querying vectors with KNN distance operators, building semantic search or RAG pipelines, tuning HNSW parameters (EFC, M, M0, distance function, type), or implementing recommendation systems with SurrealDB. Triggers: HNSW, vector, embedding, KNN, cosine, euclidean, semantic search, RAG, vector::distance.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealdb-vector/SKILL.md:1>) · SHA-256 `aad0bdd2379f57b5c6e14cdc8d527e1495a0ffd550bef93ec46f031b4f3e7127`

### `surrealkit`

Use SurrealKit, SurrealDB's schema-management and migration CLI, to scaffold projects from templates, sync schema in development, plan and execute production rollouts (with rollback), statically check SurrealQL without a database, generate JSON/TypeScript types, and write declarative TOML test suites for schemas, permissions, and API endpoints. Use this skill whenever users set up, migrate, type, check, or test a SurrealDB schema with SurrealKit.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealkit/SKILL.md:1>) · SHA-256 `b458fc32775999f5fd42c0c72e83f00298b7f1f190b43f2567538a70f07162ca`

### `surrealql`

Generate and modify SurrealQL queries to interact with SurrealDB databases. This includes creating and retrieving records, designing and managing schemas, establishing and querying graph relationships, performing live (real-time) queries, and leveraging all unique SurrealQL features for advanced database workflows. Use this skill whenever users need to write, adapt, or troubleshoot SurrealQL statements.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealql/SKILL.md:1>) · SHA-256 `424dc7e88c0f1d8803ce0d0aa78b17780d223556d075768a3e94d834a2537068`

### `surrealql-functions`

Discover and use SurrealDB's built-in SurrealQL functions with accurate, version-current signatures by running the SurrealQL language server (LSP) and tree-sitter grammar, with a namespace catalog linking every function group to its docs. Use when looking up which built-in function to use, its exact signature/arguments, getting editor completions/hover/signature-help for .surql files, or confirming a function exists in the installed SurrealDB version. Triggers: SurrealQL function, built-in function, function signature, string::, array::, math::, type::, LSP, language server, completions, surrealql-language-server.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealql-functions/SKILL.md:1>) · SHA-256 `615ca035334e76737e469265a00508be1f7e1dadd1ca772b851d80c05f111e5f`

### `surrealql-performance`

Optimize SurrealDB performance through record ID and key design, indexing strategy, and computed/derived fields. Use when queries are slow, when designing record IDs for locality and range scans, choosing between standard/unique/full-text/vector indexes, verifying index usage with EXPLAIN, or deciding whether to precompute values with computed fields, views, or events. Triggers: slow query, performance, record id design, range scan, DEFINE INDEX, EXPLAIN, computed field, FUTURE, DEFINE TABLE AS SELECT.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/surrealdb/skills/surrealql-performance/SKILL.md:1>) · SHA-256 `68657d54cc5100ceae5d31c70744ddd6781e559e445b4e0c50e7a4f75b671082`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `scripts/sync_upstream.py`

Refresh the official SurrealDB skills in this plugin from SurrealDB's own repositories.

Byline: Claude Code · Fable 5.1 · 2026-09-30

Sources (both published by SurrealDB Ltd.):
  surrealdb/agent-skills      skills/<name>/                       every skill there
  surrealdb/ai-claude-plugin  plugins/surrealdb/skills/<name>/     only the skills in EXTRA below

Each synced skill is copied byte for byte (SKILL.md plus references/) and gets a .sync-source.json
recording the repository and commit. The skills in OURS are written here and are never touched.
Nothing is deleted: a file that upstream dropped is reported, and moving it out is a manual step.

    python scripts/sync_upstream.py            # clone both repositories and sync
    python scripts/sync_upstream.py --check    # report drift only; exit 1 if anything differs
    python scripts/sync_upstream.py --agent-skills <dir> --claude-plugin <dir>   # use local checkouts

```text
python "E:/AI_Workspace/plugins/plugins/surrealdb/scripts/sync_upstream.py" --help
```

Declared arguments: `--agent-skills`, `--check`, `--claude-plugin`, `--ref`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --check | store_true | False | — | — |
| --ref | — | False | — | — |
| --agent-skills | — | False | — | local checkout of surrealdb/agent-skills |
| --claude-plugin | — | False | — | local checkout of surrealdb/ai-claude-plugin |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [sync_upstream.py:1](<E:/AI_Workspace/plugins/plugins/surrealdb/scripts/sync_upstream.py:1>) · SHA-256 `3a412307508547027c3e1b8cb176d81ab52965d8142f6910c44cd1b3c187ee70`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

### `surreal`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/surrealdb/.mcp.json:1>) · SHA-256 `5b467b6c4ea828fd693f5b3fccaee9635e67224279af844d5567cb891aa048c4`


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `docs-info`
- `docs-list`
- `docs-query`
- `docs-run`
- `docs-select`
- `mem-info`
- `mem-list`
- `mem-query`
- `mem-run`
- `mem-select`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
