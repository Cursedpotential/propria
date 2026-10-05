---
title: "search"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# search

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Propria Search (Smart Explore): tree-sitter/DuckDB structural search, CCC semantic code search, and selectable recall across Docstore, Codex, Claude, CNF, .remember and memsearch memory. One plugin for Claude Code and Codex, with the same MCP tools in both.

Source: `E:/AI_Workspace/plugins/plugins/search`. Version: `1.2.2`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `customize-search`

Install or update a pinned Search plugin instance for a target codebase and register its code-index, result, and optional recall-store profile.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/search/skills/customize-search/SKILL.md:1>) · SHA-256 `8a87d7e1727d3e3a73c8dabee05407030ff5593d96d1efac87cbe85a32f066a6`

### `smart-explore`

Canonical Propria structural code search, direct CCC semantic search and index controls, selected-store recall, conflict discovery, and reconciliation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/search/skills/smart-explore/SKILL.md:1>) · SHA-256 `035b04199f1fefaaa057132a575910377516b4ee549f36e557f639758301d31e`

## Agents

No entries found in the inspected declarations.

## Cli Entries

### `search.cmd`



```text
& "E:/AI_Workspace/plugins/plugins/search/search.cmd" --help
```

Validation: launcher present; help/runtime proof is recorded separately.

Source: [search.cmd:1](<E:/AI_Workspace/plugins/plugins/search/search.cmd:1>) · SHA-256 `18029033b044e8fb0259a97ce639c2851d879d3fb26732a47a25bb873a934341`

## Scripts

### `mcp_server.py`

Dependency-free MCP stdio facade for the canonical Propria Search engine.

Byline: Claude Code · Opus 5.5 · 2026-09-27 — newline-delimited JSON-RPC framing (the MCP stdio
transport), replacing Content-Length headers that no MCP client sends; answers ping.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [mcp_server.py:1](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:1>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `skills/customize-search/scripts/bootstrap.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/search/skills/customize-search/scripts/bootstrap.py" --help
```

Declared arguments: `--approval-file`, `--approve`, `--index-store`, `--name`, `--plan-file`, `--plan-id`, `--registry-home`, `--result-sink`, `--runtime-env`, `--source`, `--stores`, `--target`, `--update`, `mode`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| mode | — | positional | ['scan', 'plan', 'approve', 'apply', 'verify', 'report'] | — |
| --target | Path | False | — | — |
| --source | Path | False | — | — |
| --index-store | Path | False | — | — |
| --result-sink | Path | False | — | — |
| --runtime-env | Path | False | — | — |
| --stores | — | False | — | — |
| --name | — | False | — | — |
| --registry-home | Path | False | — | — |
| --plan-file | Path | False | — | — |
| --approval-file | Path | False | — | — |
| --plan-id | — | False | — | — |
| --approve | store_true | False | — | — |
| --update | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [bootstrap.py:1](<E:/AI_Workspace/plugins/plugins/search/skills/customize-search/scripts/bootstrap.py:1>) · SHA-256 `4ed0e3c687dcaf013a3901f84c320be46012617cdb9490ddbecf7f8e61712fe7`

### `smart_explore.py`

smart_explore.py — standalone, persistent, AST-based code exploration.

A self-contained rebuild of the claude-mem `smart-explore` skill with the
claude-mem MCP dependency removed. Parsing is done locally with tree-sitter
(via tree-sitter-language-pack), and the symbol index is persisted in a
per-project DuckDB file so it survives across sessions and only re-parses
changed files.

Subcommands:
  index   <path>                     build/refresh the symbol index for a dir
  indexes                            list all indexes in the central store (alias: list)
  search  <query> [--path P] [--max N] [--file-pattern G]
  outline <file>
  unfold  <file> <symbol>
  refs    <symbol> [--path P]        find usages/call sites across the indexed tree
          [--lsp [--timeout S]]      type-resolved via a language server when one is
                                     available (see `lsp`); falls back to lexical scan
  lsp                                show LSP server availability per language
  imports [--file F | --module M]    list a file's imports, or who imports a module
  changed [--since 24h|7d]           symbols added/removed/modified since a cutoff
  prune   [--yes]                    quarantine indexes whose project path no longer exists

Every subcommand accepts --json for machine-readable output and --db to
target an explicit index file.

Index location is ownership-scoped. Propria paths use
E:/AI_Workspace/Projects/Propria/.runtime/search/smart-explore/indexes;
all other paths use C:/Users/<user>/.smart-explore/indexes. Every agent/CLI tool
shares the index for the same path. Override a single index with --db only for
isolated diagnostics.

> Byline: Claude Code · Opus 4.8 · 2026-06-21
> Byline: Claude Code · Fable 5 · 2026-07-28 (cross-tool: central index store, moved to ~/.agents/skills)
> Byline: Claude Code · Opus 5.5 · 2026-09-27 (git-aware file walk that honors .gitignore and skips
  worktree/quarantine copies; outline/unfold/imports --file index one file; set-based bulk writes
  in one transaction; pruning of files a full walk no longer yields. Full Propria index 663 s -> 33 s)

```text
python "E:/AI_Workspace/plugins/plugins/search/smart_explore.py" --help
```

Declared arguments: `--db`, `--file`, `--file-path`, `--file-pattern`, `--format`, `--json`, `--lang`, `--limit`, `--lsp`, `--max`, `--mode`, `--module`, `--node-type`, `--offset`, `--output`, `--output-dir`, `--path`, `--query`, `--since`, `--store`, `--stores`, `--text`, `--timeout`, `--yes`, `file`, `packet`, `path`, `query`, `symbol`

Declared subcommands:

| Command | Source help |
|---|---|
| `index` | — |
| `indexes` | list all indexes in the central store |
| `search` | — |
| `outline` | — |
| `unfold` | — |
| `refs` | find usages/call sites of a symbol |
| `lsp` | show LSP server availability per language |
| `imports` | list a file's imports, or who imports a module |
| `changed` | symbols added/removed/modified since a cutoff |
| `prune` | quarantine indexes whose project path no longer exists |
| `stores` | inventory selectable code, docs, and memory stores |
| `semantic` | direct natural-language CCC code search |
| `ccc-grep` | structural grep via CCC |
| `recall` | query selected structural, semantic, docs, and memory stores |
| `conflicts` | discover cross-store conflicts with provenance |
| `decisions` | find governing decisions and final contracts |
| `reconcile` | run or inspect an adjudication packet |
| `run` | — |
| `repair` | — |
| `status` | — |
| `graph-query` | query nodes and incident edges in a reconciliation graph |
| `graph-preview` | preview graph counts, samples, and valid next actions |
| `export` | export a persisted reconciliation packet |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --db | — | False | — | explicit path to the index .duckdb file |
| --json | store_true | False | — | machine-readable JSON output |
| path | — | positional | — | — |
| --file-pattern | — | False | — | — |
| query | — | positional | — | — |
| --path | — | False | — | — |
| --max | int | False | — | — |
| --file-pattern | — | False | — | — |
| file | — | positional | — | — |
| file | — | positional | — | — |
| symbol | — | positional | — | — |
| symbol | — | positional | — | — |
| --path | — | False | — | — |
| --max | int | False | — | — |
| --file-pattern | — | False | — | — |
| --lsp | store_true | False | — | type-resolved references via a language server (falls back to lexical) |
| --timeout | float | False | — | LSP request timeout in seconds |
| --file | — | False | — | — |
| --module | — | False | — | — |
| --path | — | False | — | — |
| --max | int | False | — | — |
| --since | — | False | — | — |
| --path | — | False | — | — |
| --max | int | False | — | — |
| --yes | store_true | False | — | move into to_be_deleted (default: dry run) |
| --path | — | False | — | — |
| query | — | positional | — | — |
| --path | — | False | — | — |
| --lang | append | False | — | — |
| --file-path | — | False | — | — |
| --offset | int | False | — | — |
| --limit | int | False | — | — |
| --path | — | False | — | — |
| --query | — | True | — | — |
| --output-dir | — | False | — | — |
| --output-dir | — | False | — | — |
| packet | — | positional | — | — |
| packet | — | positional | — | — |
| --node-type | — | False | — | — |
| --store | — | False | — | — |
| --text | — | False | — | — |
| --limit | int | False | — | — |
| packet | — | positional | — | — |
| --limit | int | False | — | — |
| packet | — | positional | — | — |
| --format | — | False | ['json', 'md'] | — |
| --output | — | False | — | — |
| --path | — | False | — | — |
| query | — | positional | — | — |
| --path | — | False | — | — |
| --mode | — | False | ['auto', 'all', 'selected'] | — |
| --stores | append | False | — | comma separated; repeatable |
| --limit | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [smart_explore.py:1](<E:/AI_Workspace/plugins/plugins/search/smart_explore.py:1>) · SHA-256 `eeb7e025eb4d16c240b9f11ebdbb225e4f75ab9c8807b1db3c84fe85ab459c72`

## Mcp Tools

### `structural_search`

Tree-sitter structural symbol search backed by the Smart Explore DuckDB index.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:20](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:20>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `semantic_code_search`

Natural-language CCC code search with language/path filters and pagination.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:21](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:21>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `code_index_refresh`

Run the separate project-local CCC code index refresh.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:22](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:22>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `code_index_status`

Read project-local CCC code index status.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:23](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:23>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `code_index_doctor`

Run CCC health diagnostics.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:24](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:24>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `structural_grep`

Run CCC structural grep by example.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:25](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:25>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `selected_store_recall`

Query optional structural, semantic, docs and memory stores with per-store provenance.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:26](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:26>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `conflict_discovery`

Discover conflicting cross-store statements and provenance.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:27](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:27>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `decisions_final_contracts`

Find governing decisions and final/canonical contracts across selected stores.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:28](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:28>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_run`

Persist a provenance-rich adjudication packet and agent repair loop.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:29](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:29>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_repair`

Trigger a bounded agent repair packet when attribution or conflicts are dirty.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:30](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:30>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_status`

Read a persisted reconciliation packet status.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:31](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:31>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_graph_query`

Query packet graph nodes and incident edges by type, store, or text.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:32](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:32>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_graph_preview`

Preview packet graph counts, representative nodes, edges, and next actions.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:33](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:33>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `reconcile_export`

Export a packet as JSON or Markdown.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:34](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:34>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

### `store_inventory`

Report requested/available adapter identity for every selectable store.

Validation: source catalog; invocation not tested.

Source: [mcp_server.py:35](<E:/AI_Workspace/plugins/plugins/search/mcp_server.py:35>) · SHA-256 `d93aeeb60c88e3a32cbff3f402d0ebb12baa6737804f93de971acd9848c7820b`

## Mcp Servers

### `propria-search`

stdio

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/search/.mcp.json:1>) · SHA-256 `9d07e8eb5b8ead372b9a6b0e8a0fe8c38ac0d0f2b7a62118269e3a7996e13eb9`

### `propria-search`

stdio

Validation: configured; health not inferred.

Source: [mcp.json:1](<E:/AI_Workspace/plugins/plugins/search/.codex-plugin/mcp.json:1>) · SHA-256 `4c55ce4f69cfe5dedda514bc6af68ad7ca6359e6aa6f59ecfc1be112be5ce6a9`


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
