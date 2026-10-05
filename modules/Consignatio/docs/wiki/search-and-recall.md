---
title: Search and recall
date: 2026-10-04
status: source-and-help-verified
byline: Documentation agent GPT-6.1
tags:
  - search
  - memory
  - code
---

# Search and recall

Choose the search scope first: **code**, **project documentation**, **agent memory**, or **source corpus**. Propria Search exposes code exploration and selectable recall, while `memsearch` and CCC also have their own terminal commands. [S1, S2, S6]

Related notes: [[Code/wiki/README]], [[Code/wiki/plugin-inventory]], [[Code/wiki/case-bible-search]], [[Code/wiki/search-and-recall]].

Contents: [Choose a scope](#choose-a-scope) · [Launch](#launch) · [Search CLI reference](#search-cli-reference) · [Recall and provenance](#recall-and-provenance) · [MCP tools](#mcp-tools) · [Memory CLI](#memory-cli) · [CCC CLI](#ccc-cli) · [Troubleshooting](#troubleshooting) · [Sources and validation](#sources-and-validation).

## Choose a scope

These systems preserve separate ownership and index identities. [S1; S6: `inventory`, `_query_store`; Consignatio `AGENTS.md`, three-system boundary]

| Question | Tool or store |
|---|---|
| Where is a code symbol or implementation? | Smart Explore structural search; Tree-sitter parses code and DuckDB stores symbols. |
| Which code implements this behavior? | CCC semantic search, backed by its project-local CocoIndex Code index. |
| What is the current project guide or decision? | Docstore through its configured adapter; inspect governing sources and revisions. |
| What did an agent previously learn or run? | Standalone `memsearch search`, then `expand` and `transcript`; or explicitly selected memory stores in Propria Search. |
| What do documents, messages or chats say? | Case Bible content reader in [[Code/wiki/case-bible-search]]. |

An agent slash command, such as `/smart-explore`, is plugin instruction syntax. It is not a terminal command. Terminal users invoke `search.cmd`, `memsearch`, or `ccc`; MCP clients use named tools. [S1: CLI/MCP sections; S2: `main`; S3: `TOOLS`]

## Launch

The supported Search launcher uses the plugin's locked `uv` project: Python 3.11 or later, DuckDB 1.5.5 and Tree-sitter language pack 1.9.1. It finds `uv` on PATH, then checks its configured fallback. `UV_PROJECT_ENVIRONMENT` controls the runtime environment location. The launcher can create or synchronize that environment; arbitrary system-Python invocation bypasses the declared runtime. [S4: lines 1–11; S5; S1: CLI]

Use the absolute launcher in PowerShell:

```powershell
$search = 'E:/AI_Workspace/plugins/plugins/search/search.cmd'
& $search --help
& $search semantic --help
& $search reconcile run --help
```

On 2026-10-04, `Get-Command` resolved `memsearch.exe` and `ccc.exe` in `C:/Users/matts/.local/bin`, but did not resolve `search.cmd`. Use `search.cmd --help` only when its folder is already on PATH, or `./search.cmd --help` from its folder. The absolute launcher above passed local help checks.

**Write effects matter.** Smart Explore `search`, `refs`, `imports`, `changed`, `outline` and `unfold` refresh derived index state. Propria code paths route to `.runtime/search/smart-explore/indexes`; other paths route to the user Smart Explore store. `--db` selects an explicit diagnostic database. No indexing or search refresh was run for this page. [S2: `cmd_search`, `cmd_refs`, `cmd_imports`, `cmd_changed`, `cmd_outline`, `cmd_unfold`, runtime constants]

## Search CLI reference

All top-level subcommands accept `-h`/`--help`, `--db DB` and `--json`. Supply them after the top-level command. For nested reconciliation, place `--db` and `--json` before `run`, `repair`, or `status`. Common flags are accepted by the parser; `--db` does not redirect CCC or memory stores. Several adapter commands already emit JSON or relay another CLI's output. [S2: `main`, `_ccc_command`, `cmd_recall`]

The complete command and option list follows local help output, with defaults checked in `main`. `--path` defaults to the current directory wherever listed. [S2: lines 1265–1372]

| Command | Arguments and additional options | Purpose / effects |
|---|---|---|
| `index` | `path`; `--file-pattern G` | Build/refresh structural index; writes generated state. |
| `indexes` (alias `list`) | None | List central indexes. |
| `search` | `query`; `--path P`, `--max N` (20), `--file-pattern G` | Structural symbol search; refreshes index. |
| `outline` | `file` | List symbols; indexes the selected file. |
| `unfold` | `file symbol` | Show matching symbol source; indexes the selected file. |
| `refs` | `symbol`; `--path P`, `--max N` (30), `--file-pattern G`, `--lsp`, `--timeout S` (60) | References; refreshes index. Optional language server resolves types, with lexical fallback. |
| `lsp` | None | Report language-server availability. |
| `imports` | `--file F`, `--module M`, `--path P`, `--max N` (40) | File imports, module importers, or import overview; refreshes selected file or project. |
| `changed` | `--since T` (`24h`), `--path P`, `--max N` (100) | Refresh then show symbol changes; accepts integer minutes/hours/days, such as `90m`, `24h`, `7d`. |
| `prune` | `--yes` | Default preview; `--yes` moves stale indexes to quarantine. |
| `stores` | `--path P` | Inventory selectable adapters and identities. |
| `semantic` | `query`; `--path P`, repeatable `--lang L`, `--file-path G`, `--offset N` (0), `--limit N` (10) | Delegate to CCC semantic code search. |
| `ccc-index` | `--path P` | Refresh separate CCC index; writes generated state. |
| `ccc-status` | `--path P` | Delegate to CCC status. |
| `ccc-doctor` | `--path P` | Delegate to CCC health diagnostics. |
| `ccc-grep` | `--path P`, required `--query Q` | Delegate to CCC structural grep. |
| `recall` | `query`; recall options below | Query selected stores; selected code adapters may refresh state. |
| `conflicts` | `query`; recall options below | Discover conflicting statements with provenance. |
| `decisions` | `query`; recall options below | Return detected decisions and contracts. |
| `reconcile run` | `query`; recall options; `--output-dir D` | Write JSON/Markdown adjudication packet. |
| `reconcile repair` | `query`; recall options; `--output-dir D` | Write packet and report repair requested/status; does not itself implement agent fixes. |
| `reconcile status` | `packet` | Read saved packet status. |
| `graph-query` | `packet`; `--node-type T`, `--store S`, `--text Q`, `--limit N` (50) | Read packet graph nodes and incident edges. |
| `graph-preview` | `packet`; `--limit N` (10) | Read graph counts, samples, backlog and next actions. |
| `export` | `packet`; `--format json\|md` (`md`), `--output F` | Print export; writes a file when `--output` is provided. |

Shared recall options are `--path P`, `--mode auto|all|selected` (default `auto`), repeatable `--stores S` (comma-separated lists accepted), and `--limit N` (20). Selected mode requires a nonempty valid store list. [S2: `add_recall_options`; S6: `select_stores`]

For code exploration, these are parser-checked recipes. They were not executed because structural search updates generated state:

```powershell
& $search search "search" --path E:/AI_Workspace/Projects/Propria/modules/Consignatio --max 10 --json
& $search semantic "filter search results" --path E:/AI_Workspace/Projects/Propria/modules/Consignatio --lang python --limit 5
& $search outline E:/AI_Workspace/plugins/plugins/search/smart_explore.py
& $search unfold E:/AI_Workspace/plugins/plugins/search/smart_explore.py cmd_recall
```

## Recall and provenance

Selectable names are `smart_explore`, `ccc`, `docstore`, `codex_memory`, `claude_memory`, `cnf`, `remember`, and `memsearch`. `all` requests every store. `auto` uses an explicit supplied list, or all names when no list is supplied. Availability and query success are recorded separately. [S6: `STORE_NAMES`, `select_stores`, `recall`]

For a focused documentation and memory question, select the stores explicitly:

```powershell
& $search recall "source traceability" --path E:/AI_Workspace/Projects/Propria --mode selected --stores docstore,codex_memory --limit 10
& $search decisions "documentation ownership" --mode selected --stores docstore,claude_memory
```

These are parser-checked examples, not live adapter proofs. Docstore requires `PROPRIA_DOCSTORE_ADAPTER`, a JSON-stdio executable adapter. Search also reads `AI_WORKSPACE_ROOT` for memory routing. CCC availability depends on its executable, project settings and recorded health. Memory availability checks include configured roots; the `memsearch` entry checks its executable, `~/.memsearch/config.toml`, and referenced `env:NAME` credentials. Do not publish configuration values. [S6: `_roots`, `inventory`, `_query_store`]

**The selectable `memsearch` store uses filesystem text search in this implementation.** It does not call the standalone semantic `memsearch search` command. Other memory stores also use lexical filesystem retrieval; Smart Explore, CCC and Docstore have explicit query branches. [S6: `_query_store`, `_filesystem_search`]

Results normalize store, source URI/path, available line range and revision, hash, score, excerpt and provenance. Duplicate excerpts retain occurrences across stores. A generated `content_hash` can identify normalized result data rather than the original source file. Check its origin before treating it as a source-byte hash. Conflict and decision discovery remain retrieval aids; verify authority against the cited original. [S6: `_normalize`, `_dedupe`, `discover_conflicts`, `recall`]

`reconcile run` and `repair` persist packets under `.search-reconcile/packets` by default. `repair` sets status for subsequent agent action; its handler does not invoke a builder. Packet inspection and export can then use an existing path. [S2: `cmd_reconcile`, `cmd_export`; S6: `persist_packet`]

## MCP tools

The source declares **16 MCP tools**. This mapping is derived from `TOOLS` and `call`, not a claim that a live client has registered every tool. [S3: lines 19–35, `call`]

| MCP tool | CLI operation |
|---|---|
| `structural_search` | `search` |
| `semantic_code_search` | `semantic` |
| `code_index_refresh` | `ccc-index` |
| `code_index_status` | `ccc-status` |
| `code_index_doctor` | `ccc-doctor` |
| `structural_grep` | `ccc-grep` |
| `store_inventory` | `stores` |
| `selected_store_recall` | `recall` |
| `conflict_discovery` | `conflicts` |
| `decisions_final_contracts` | `decisions` |
| `reconcile_run` | `reconcile run` |
| `reconcile_repair` | `reconcile repair` |
| `reconcile_status` | `reconcile status` |
| `reconcile_graph_query` | `graph-query` |
| `reconcile_graph_preview` | `graph-preview` |
| `reconcile_export` | `export` |

## Memory CLI

Standalone `memsearch` searches indexed Markdown memory and supports progressive recall. Local root and subcommand help passed on 2026-10-04. Use hashes and transcript paths returned by your own results; the placeholders below are not literal inputs. [M1: `search`, `expand`, transcript command; verified help]

```powershell
memsearch --help
memsearch search "documentation ownership" --top-k 5 --json-output
memsearch expand <chunk-hash> --lines 20 --json-output
memsearch transcript <transcript-path> --turn <turn-id> --context 2
```

The read-oriented interface is:

| Command | Arguments / options from local help |
|---|---|
| Root | `--version`, `--help`. |
| `search QUERY` | `-k/--top-k`, `--source-prefix`, `--reranker-model`, `-j/--json-output`; connection/embedding options below. |
| `expand CHUNK_HASH` | `--section/--no-section`, `-n/--lines`, `-j/--json-output`; connection/embedding options. |
| `transcript PATH` | `-t/--turn`, `-c/--context`, `-j/--json-output`. |
| `stats` | `-c/--collection`, `--milvus-uri`, `--milvus-token`. |
| `config` | `get`, `list`, `init`, `set`; `init` and `set` change configuration. |

Connection/embedding options are `--milvus-token`, `--milvus-uri`, `-c/--collection`, `--api-key`, `--base-url`, `--batch-size`, `-m/--model`, and `-p/--provider`. They are option names, not instructions to expose credential values. All subcommands provide `--help`. [M1: `_common_options`; local help]

The remaining root commands are `index PATHS...`, `watch PATHS...`, `compact`, `summarize`, `skills` and `reset`. Index/watch write index state; watch runs continuously. Compact writes summaries; summarize uses a configured LLM and requires `--plugin`, with optional `--agent-name`. Skills includes `add`, `distill`, `install`, `list`, `status`; capture/distillation/installation writes artifacts. Reset drops indexed data. These write/lifecycle commands were inspected only with `--help`, not run. [M1: corresponding command handlers; local help]

Memory recall should preserve the chunk source and journal anchor, then inspect original turns when exact commands or decisions matter. Successful help does not verify Milvus connectivity, embedding credentials, collection freshness or transcript availability. [M1: `search`, `expand`, transcript help]

## CCC CLI

CCC searches code using a project-local index. Run it from the intended project so settings and index identity resolve correctly. Search wrapper `--path P` selects the project working directory; standalone `ccc search --path G` filters file paths. [S2: `_ccc_command`; K1: `require_project_root`, `search`; local help]

These recipes are checked against local help:

```powershell
ccc --help
ccc search "filter search results" --lang python --limit 5 --json
ccc grep 'print(...)' . --lang python --no-color
```

They were not executed beyond help. CCC search requires existing global and project settings and uses its daemon search path; `--refresh` explicitly refreshes before searching. Structural grep needs no index or daemon, but scans supported code files under the requested path. [K1: `search`, `grep`, `require_project_root`; local help]

| Command | Additional options / behavior from local help |
|---|---|
| Root | `--install-completion`, `--show-completion`, `--help`. Installing completion writes shell setup. |
| `init` | `--litellm-model`, `-f/--force`; initializes configuration. |
| `index` | Refreshes code index. |
| `search QUERY...` | `--lang`, `--path` file glob, `--offset` (0), `--limit` (10), `--refresh`, `--json`. |
| `grep PATTERN [PATH]` | Path defaults to `.`; `--lang`, `--path` file glob, `--no-color`. |
| `status` | Show project status. |
| `doctor` | `-v/--verbose` for full diagnostic exceptions. |
| `reset` | `--all`, `-f/--force`; resets databases and optionally removes settings. Do not use for routine recall. |
| `mcp` | Start the stdio MCP server. |
| `version` | Print CLI version. |
| `daemon` | `status`, `restart`, `stop` manage the CCC daemon process. |

All commands accept `--help`. This page records reset and daemon controls for discoverability; none was executed. Configuration identity includes `.cocoindex_code/settings.yml` and `target_sqlite.db`. Provider/model credentials belong in configured environment references; avoid printing their values. [K1; S1: CCC identity; S6: `inventory`]

## Troubleshooting

| Symptom | Next check |
|---|---|
| `search.cmd` not found | Use the absolute launcher; inspect PATH without changing it. |
| Launcher cannot find `uv` | Check the existing runtime installation and launcher prerequisites. |
| A store is listed but recall fails | Inspect per-store `requested`, `available`, `queried`, `skipped`, `error`, adapter identity, duration and result count. Availability is not query proof. |
| Docs recall unavailable | Check whether `PROPRIA_DOCSTORE_ADAPTER` names the governed adapter. |
| Semantic memory results differ from selectable recall | They use different retrieval paths: standalone semantic index versus lexical memory-file search. |
| Code search changes runtime files | Structural commands refresh generated state; CCC also uses daemon runtime state. |
| No relevant hit | Recheck scope, project root, index freshness and filter/window settings before changing infrastructure. |

These checks derive from launcher handling and adapter/CLI code. No repair, refresh, reset, changefeed operation, or service lifecycle action formed part of this documentation validation. [S2, S4, S6, M1, K1]

## Sources and validation

External source root: `E:/AI_Workspace/plugins/`. Source paths below are relative to that root. Function names are stable locators; hashes identify the reviewed snapshot on **2026-10-04**.

| ID | Source / locator | SHA256 |
|---|---|---|
| S1 | `plugins/search/README.md`, CLI, MCP tools, identity | `20cd486f34c105879c5053341e4bb34758803677a82cdc0802bd62cfa2ba8ead` |
| S2 | `plugins/search/smart_explore.py:1265`, `main`; `cmd_search`, `_ccc_command`, `cmd_reconcile`, `cmd_export` | `eeb7e025eb4d16c240b9f11ebdbb225e4f75ab9c8807b1db3c84fe85ab459c72` |
| S3 | `plugins/search/mcp_server.py:19`, `TOOLS`; `:50` `call` | `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b` |
| S4 | `plugins/search/search.cmd:1`, launcher | `18029033b044e8fb0259a97ce639c2851d879d3fb26732a47a25bb873a934341` |
| S5 | `plugins/search/pyproject.toml:1`, runtime dependencies | `6a1c1462937096cb531688b8fd0aa532e2d4ab2750a5f9434804f93472556e0e` |
| S6 | `plugins/search/reconciliation.py:28`, `inventory`; `:55` `select_stores`; `:69` `_normalize`; `:106` `_query_store`; `:167` `persist_packet` | `5552983d9dfe4166c033c33014437cc8178724d28c9b4045b0d143b595fe28ac` |
| M1 | `forks/memsearch/src/memsearch/cli.py:344`, `search`; `:427` `expand`; `config_group`, `skills_group` and command handlers | `3caf611fb3995c429e1da68fc24a2d72c952f875a70b80c9b6503b55566f023c` |
| K1 | `forks/cocoindex-code/src/cocoindex_code/cli.py:729`, `search`; `:792` `grep`; `require_project_root`, lifecycle handlers | `a5f7d5098baa1f0c02b0277eb21914c130b834260207cd5f336de7bf01791c34` |

**Executed validation:** `Get-Command` for launcher resolution; Search root help, every canonical top-level subcommand help, and `reconcile run/repair/status --help`; memsearch root help plus `search`, `expand`, `transcript`, `stats`, `config`, `index`, `watch`, `compact`, `summarize`, `skills`, `reset` help; CCC root help plus every top-level command help. Every help batch exited successfully. Nested config/skills/daemon operations were listed by group help, not exercised.

**Not established:** live adapter registration, successful backend queries, complete/fresh indexes, relevance, or working credentials. Non-help examples were checked against actual parsers/help; they were not executed, preserving the read-only validation scope. Parent owns Docstore preflight and publication to B2 `salem-data/consignatio/casevault/Code/wiki/`. This is the existing CaseVault hierarchy; the old OneDrive vault is a legacy reference. The existing `Code/INDEX` routes to `Code/wiki/INDEX`; parent preserves and extends that scaffold. Recovery material belongs to the existing `Recovered/` domain; this documentation task moves no recovery files and creates no new domain.
