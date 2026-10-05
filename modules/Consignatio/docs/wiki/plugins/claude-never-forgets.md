---
title: "claude-never-forgets"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# claude-never-forgets

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Local fork of claude-never-forgets (persistent project memory). Cleanup threshold 60 + lossless consolidation; fetch-only upstream tracking — see UPSTREAM.md.

Source: `E:/AI_Workspace/plugins/plugins/claude-never-forgets`. Version: `1.0.2`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `cnf-consolidate`

Dispatch lossless CNF background consolidation on demand without waiting in the foreground

```text
/claude-never-forgets:cnf-consolidate
```

Source: [cnf-consolidate.md:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/commands/cnf-consolidate.md:1>) · SHA-256 `e320dd47cdb9d20cbf1ba08bd30103468a67a55c2392858b9b735f054ad2686c`

### `cnf-forget`

CNF session-recall lane — remove an entry from .cnf/memories/project_memory.json (NOT the built-in auto-memory MEMORY.md)

```text
/claude-never-forgets:cnf-forget
```

Arguments: `['what to forget']`

Source: [cnf-forget.md:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/commands/cnf-forget.md:1>) · SHA-256 `84f4983f8bbbb08b84e41ca46d5a5c4f8dd394a697761725c4bd3a312c26f897`

### `cnf-memories`

CNF session-recall lane — show all entries in .cnf/memories/project_memory.json (NOT the built-in auto-memory MEMORY.md)

```text
/claude-never-forgets:cnf-memories
```

Source: [cnf-memories.md:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/commands/cnf-memories.md:1>) · SHA-256 `489f13d1fcf7c6b29a6afa36997c9b880dec014a114864ce0ab9c8731d3aba5c`

### `cnf-remember`

CNF session-recall lane — manually add something to .cnf/memories/project_memory.json (NOT the built-in auto-memory MEMORY.md)

```text
/claude-never-forgets:cnf-remember
```

Arguments: `['what to remember']`

Source: [cnf-remember.md:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/commands/cnf-remember.md:1>) · SHA-256 `b76583ce115dcc1cf750b613a9bcaab9a6e49adfa115aed646f3b45d93f27498`

## Skills

### `cnf-recall`

CNF session-recall lane (NOT the built-in Claude Code auto-memory). Read and apply realtime-captured session memories from .cnf/memories/project_memory.json for context-aware assistance. Use when recalling session-level decisions, corrections, or preferences captured this session or recently. Trigger with phrases like "remember when", "like before", or "what was our decision about".
For DURABLE curated facts (decisions, infra, preferences) use the built-in auto-memory (MEMORY.md + frontmatter .md files), not this skill.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/skills/cnf-recall/SKILL.md:1>) · SHA-256 `fd9fd4652a9650f6c624ec41760cf6f06c241b802402737bcd69f81a67bfb020`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `hooks/cleanup_worker.py`

Group CNF records losslessly using a detached, tool-free cloud classifier.

Inputs: --cwd and optional --force; project-local realtime CNF records.
Outputs: cleanup status JSON and thematic records with exact original provenance.
Side effects: one configured cloud request, quarantine backup and atomic CNF update.
Use behind stop_cleanup.dispatch; inference never owns the memory write lock.
Byline: Codex · GPT-6 · 2026-10-04.

```text
python "E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/cleanup_worker.py" --help
```

Declared arguments: `--cwd`, `--dispatch`, `--force`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --cwd | — | True | — | — |
| --force | store_true | False | — | — |
| --dispatch | store_true | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cleanup_worker.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/cleanup_worker.py:1>) · SHA-256 `67984b2a7f24df07e899ebea0d2cedda8c86ecaa87557e32c0ad3eebe12e5823`

### `hooks/session_start.py`

Load consolidated CNF topics plus recent captures into session recall.

Inputs: SessionStart JSON containing cwd. Outputs: scoped recall context.
Side effects: memory reads only; source records stay on disk.
Choose for session recall, not capture or consolidation.
Byline: Codex · GPT-6 · 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [session_start.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/session_start.py:1>) · SHA-256 `3b377e2d62c05675efb3186266d3a10973fec6a485ed2ca24bea5902671204e6`

### `hooks/stop_cleanup.py`

Dispatch detached CNF cleanup without interrupting the foreground agent.

Inputs: Stop-hook JSON containing cwd and stop_hook_active.
Outputs: {} immediately after launching; never a blocking decision.
Side effects: launches cleanup_worker.py with CNF_BACKGROUND=1 and detached stdio.
Use for automatic maintenance; the worker checks twenty entries without cooldown.
Byline: Codex · GPT-6 · 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [stop_cleanup.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/stop_cleanup.py:1>) · SHA-256 `fadd02d06930baefde253af69382381188b36cf31fea46d781110ff79040e6a7`

### `hooks/storage.py`

Provide repository-scoped CNF paths, process locks and atomic JSON storage.

Inputs: a working directory or explicit CNF file path.
Outputs: paths, parsed JSON objects and lock contexts.
Side effects: persistent lock files and atomic replacements; never unlinks files.
Use this shared guard for capture and cleanup rather than unlocked writes.
Byline: Codex · GPT-6 · 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [storage.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/storage.py:1>) · SHA-256 `a473e26932a36a4f1ec8174c6a0fb5e5b5ae46f0b64d33841f9ee7beddd77c98`

### `hooks/tool_rejected.py`

Capture complete owner rejection feedback using the shared CNF storage guard.

Inputs: failed-tool hook JSON. Outputs: an empty hook response.
Side effects: atomic correction append; background classifier events are ignored.
Choose for explicit rejection feedback, not routine tool failures or cleanup.
Byline: Codex · GPT-6 · 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [tool_rejected.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/tool_rejected.py:1>) · SHA-256 `490a789eb30cd8f59a38cddd93d19564c20fe7928cc7d6f445e3798f81176d8b`

### `hooks/user_prompt.py`

Capture full owner prompts with atomic CNF updates and background guards.

Inputs: UserPromptSubmit JSON and the optional existing Claude transcript.
Outputs: an empty hook response. Side effects: locked CNF memory replacement.
Use for realtime capture; cleanup_worker groups captured records separately.
Byline: Codex · GPT-6 · 2026-10-04.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [user_prompt.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/hooks/user_prompt.py:1>) · SHA-256 `42df908d1b43c5bfe2196260541b49cd0d468db0b578d7149340da999ca860da`

### `skills/cnf-recall/scripts/manage-memory.py`

manage-memory.py - Manage project memories for Claude Never Forgets

Supports:
- Adding new memories
- Listing all memories
- Removing memories
- Searching memories

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [manage-memory.py:1](<E:/AI_Workspace/plugins/plugins/claude-never-forgets/skills/cnf-recall/scripts/manage-memory.py:1>) · SHA-256 `6d6b85257187181832c06269c189945ae9a99c1ba1c7ffb813621c393e46fbb3`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
