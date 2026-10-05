---
title: "codebase"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# codebase

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Codebase comprehension and documentation tools: map/onboard a repo, analyze structure and debt, pack a repo for AI, design deep modules, improve architecture. 7 first-class skills (`codebase:<name>`), plus an index entry skill.

Source: `E:/AI_Workspace/plugins/plugins/codebase`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `cb-acquire-codebase-knowledge`

(codebase) Use this skill when the user explicitly asks to map, document, or onboard into an existing codebase. Trigger for prompts like 'map this codebase', 'document this architecture', 'onboard me to this repo', or 'create codebase docs'. Do not trigger for routine feature implementation, bug fixes, or narrow code edits unless the user asks for repository-level discovery.

Arguments: `Optional: specific area to focus on, e.g. "architecture only", "testing and concerns"`

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/acquire-codebase-knowledge/SKILL.md:1>) · SHA-256 `e54545ac68370ac59f89cd7f8c5cf4ecbc2dd1b19e8b15a8f7ed0f1f1916fd46`

### `codebase`

(codebase) Codebase comprehension and documentation tools: map/onboard a repo, analyze structure and debt, pack a repo for AI, design deep modules, improve architecture. Entry point / router — read this first, then load one member from references/. Triggers: map this codebase, onboard, analyze repository, repomix, pack repo, architecture improvement, deep modules, explore codebase. Members: acquire-codebase-knowledge, repository-analyzer, improve-codebase-architecture, codebase-design, repomix, explore-codebase.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/codebase/SKILL.md:1>) · SHA-256 `35e2b4ce4284f7b4feb4a2730513c22d450909e7b0d9ad7d2a56048fd5a66692`

### `cb-codebase-design`

(codebase) Shared vocabulary for designing deep modules. Use when the user wants to design or improve a module's interface, find deepening opportunities, decide where a seam goes, make code more testable or AI-navigable, or when another skill needs the deep-module vocabulary.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/codebase-design/SKILL.md:1>) · SHA-256 `85fcded6ea80046d074a342ffe60020d9c0b88cb91979589cb6aefccf91a387b`

### `cb-explore-codebase`

(codebase) RLM-style large-codebase comprehension — build a mental map of any codebase by dispatching sub-agents to explore regions without bloating main context

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/explore-codebase/SKILL.md:1>) · SHA-256 `c82dc98ebbf50f9199dc87c7a90cbf7d40f66c033cc245fdf455a438bb2ac61a`

### `cb-improve-codebase-architecture`

(codebase) Scan a codebase for deepening opportunities, present them as a visual HTML report, then grill through whichever one you pick.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/improve-codebase-architecture/SKILL.md:1>) · SHA-256 `228ae044b7eb06f814c575bd1b5771598ead14f2bfe0eeffec8a85f5b8d3b1f8`

### `cb-repomix`

(codebase) Package entire code repositories into single AI-friendly files using Repomix. Capabilities include pack codebases with customizable include/exclude patterns, generate multiple output formats (XML, Markdown, plain text), preserve file structure and context, optimize for AI consumption with token counting, filter by file types and directories, add custom headers and summaries. Use when packaging codebases for AI analysis, creating repository snapshots for LLM context, analyzing third-party libraries, preparing for security audits, generating documentation context, or evaluating unfamiliar codebases.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/repomix/SKILL.md:1>) · SHA-256 `eed09d1bc0c62c9d231e6a8b4ff89b7dbcbd31ec2d0ac67856caf8c2cb64b42a`

### `cb-repository-analyzer`

(codebase) Analyzes codebases to generate comprehensive documentation including structure, languages, frameworks, dependencies, design patterns, and technical debt. Use when user says 'analyze repository', 'understand codebase', 'document project', or when exploring unfamiliar code.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/repository-analyzer/SKILL.md:1>) · SHA-256 `4dfb82ba1dc348038d1c1f7964dba78d9e05d784ca2910e27e7a4bda1d85af27`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/acquire-codebase-knowledge/scripts/scan.py`

scan.py — Collect project discovery information for the acquire-codebase-knowledge skill.
Run from the project root directory.

Usage: python3 scan.py [OPTIONS]

Options:
  --output FILE   Write output to FILE instead of stdout
  --help          Show this message and exit

Exit codes:
  0  Success
  1  Usage error

```text
python "E:/AI_Workspace/plugins/plugins/codebase/skills/acquire-codebase-knowledge/scripts/scan.py" --help
```

Declared arguments: `--output`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --output | str | False | — | Write output to FILE instead of stdout |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [scan.py:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/acquire-codebase-knowledge/scripts/scan.py:1>) · SHA-256 `193d5b9f73a71d6ecc2f39f13f33d96be49d675abe83f5857ddbdc4ae21b7bd9`

### `skills/repomix/scripts/repomix_batch.py`

Batch process multiple repositories using Repomix.

This script processes multiple repositories (local or remote) using the repomix CLI tool.
Supports configuration through environment variables loaded from multiple .env file locations.

```text
python "E:/AI_Workspace/plugins/plugins/codebase/skills/repomix/scripts/repomix_batch.py" --help
```

Declared arguments: `--file`, `--ignore`, `--include`, `--no-security-check`, `--output-dir`, `--remote`, `--remove-comments`, `--style`, `--verbose`, `-f`, `-o`, `-v`, `repos`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| repos | — | positional | — | Repository paths or URLs to process |
| -f, --file | — | False | — | JSON file containing repository configurations |
| --style | — | False | ['xml', 'markdown', 'json', 'plain'] | Output format (default: xml) |
| -o, --output-dir | — | False | — | Output directory (default: repomix-output) |
| --remove-comments | store_true | False | — | Remove comments from source files |
| --include | — | False | — | Include pattern (glob) |
| --ignore | — | False | — | Ignore pattern (glob) |
| --no-security-check | store_true | False | — | Disable security checks |
| -v, --verbose | store_true | False | — | Verbose output |
| --remote | store_true | False | — | Treat all repos as remote URLs |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [repomix_batch.py:1](<E:/AI_Workspace/plugins/plugins/codebase/skills/repomix/scripts/repomix_batch.py:1>) · SHA-256 `e112b15a553b94416e5c264f56c230e7c5da925bcd12d5207692d83f6bc2ee0f`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
