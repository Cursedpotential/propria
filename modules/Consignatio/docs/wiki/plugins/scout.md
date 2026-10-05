---
title: "scout"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# scout

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Local headless unified capability discovery and selection using DuckDB.

Source: `E:/AI_Workspace/plugins/plugins/scout`. Version: `2.0.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `capability-scout`

Discover and select the smallest sufficient set of local skills, agents, plugins and observed runtime tools. Use for capability selection when a task needs it; do not restart discovery on continuation messages.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/scout/skills/capability-scout/SKILL.md:1>) · SHA-256 `5b2c086662aa83693d09caa2cbc1e5bda601989fd79ab92256c5d3317903d979`

## Agents

No entries found in the inspected declarations.

## Cli Entries

### `scout`



```text
scout --help
```

Validation: entry point declared; installation/PATH not inferred.

Source: [pyproject.toml:1](<E:/AI_Workspace/plugins/plugins/scout/pyproject.toml:1>) · SHA-256 `013c8dd859da14b44224b71f828e60a22683b7fab8da96e347b56fad0e82807a`

## Scripts

### `scout/cli.py`

Missing module docstring.

```text
python "E:/AI_Workspace/plugins/plugins/scout/scout/cli.py" --help
```

Declared arguments: `--cwd`, `--db`, `--depth`, `--format`, `--home`, `--host`, `--host-home`, `--id`, `--kind`, `--limit`, `--max-size`, `--need`, `--plugin-inventory`, `--plugin-root`, `--project-root`, `--query`, `--rebuild`, `--registry`, `--root`, `--runtime`, `--session-id`, `--stdin`, `--threshold`, `action`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| action | — | positional | ('index', 'discover', 'select', 'inspect', 'validate-registry') | — |
| --host | — | True | ('claude', 'codex') | — |
| --cwd | Path | False | — | — |
| --home | Path | False | — | — |
| --host-home | Path | False | — | Override CLAUDE_CONFIG_DIR or CODEX_HOME |
| --project-root | Path | False | — | — |
| --root | Path | False | — | Additional local capability root containing skills/agents/commands |
| --plugin-root | Path | False | — | Explicit enabled development plugin root |
| --plugin-inventory | Path | False | — | Observed Codex plugin list --json; install/version evidence, not runtime tool evidence |
| --registry | Path | False | — | — |
| --runtime | Path | False | — | Fresh session-attested runtime inventory JSON |
| --session-id | — | False | — | — |
| --db | Path | False | — | — |
| --query | — | False | — | — |
| --stdin | store_true | False | — | — |
| --need | append | False | — | Independent requirement; repeat for multi-capability tasks |
| --depth | — | False | ('auto', 'quick', 'deep') | — |
| --threshold | float | False | — | — |
| --limit | int | False | — | — |
| --max-size | int | False | — | — |
| --kind | append | False | ('skill', 'agent', 'command', 'tool', 'plugin', 'mcp') | — |
| --rebuild | store_true | False | — | — |
| --format | — | False | ('json', 'markdown') | — |
| --id | — | False | — | Inspect a graph node and its incident edges |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cli.py:1](<E:/AI_Workspace/plugins/plugins/scout/scout/cli.py:1>) · SHA-256 `e75f2b3618d2dca8c1d0afe988127cc10af60f7cd5e5e2baa188b25bb2d06181`

### `scripts/capture-runtime.py`

Normalize an observed tool list, not static MCP configuration, into a snapshot.

```text
python "E:/AI_Workspace/plugins/plugins/scout/scripts/capture-runtime.py" --help
```

Declared arguments: `--complete-tools`, `--cwd`, `--host`, `--session-id`, `--ttl`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --host | — | True | ('claude', 'codex') | — |
| --session-id | — | True | — | — |
| --cwd | Path | False | — | — |
| --complete-tools | store_true | False | — | Assert input is the complete advertised tool inventory |
| --ttl | int | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [capture-runtime.py:1](<E:/AI_Workspace/plugins/plugins/scout/scripts/capture-runtime.py:1>) · SHA-256 `8d2bf2c4d7f51317cf9a5896936106a8e20fd23c9cfe72f8fcf8798af031e4c7`

### `scripts/scout.py`

Run from an unpacked plugin without installation; dependencies are still required.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [scout.py:1](<E:/AI_Workspace/plugins/plugins/scout/scripts/scout.py:1>) · SHA-256 `98b50794d2a735d41f3cf3270f3d99645408031301c5f1d7be664985efb5c70d`

### `scripts/validate.py`

Reproducible static/package validation and optional behavioral test suite.

```text
python "E:/AI_Workspace/plugins/plugins/scout/scripts/validate.py" --help
```

Declared arguments: `--tests`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --tests | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [validate.py:1](<E:/AI_Workspace/plugins/plugins/scout/scripts/validate.py:1>) · SHA-256 `aa39ea301ccea816c8ac55cec970b6ede0557f8561c0ac10d28a5f0831d42d8b`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
