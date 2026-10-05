---
title: "claude-reflect"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# claude-reflect

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Private fork of bayramannakov/claude-reflect 1.5.0 (MIT): /reflect writes learnings to ~/.claude/AGENTS.md and ./AGENTS.md, never CLAUDE.md (owner 2026-09-27).

Source: `E:/AI_Workspace/plugins/plugins/claude-reflect`. Version: `1.5.0-propria.2`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `reflect`

Reflect on session corrections and update AGENTS.md (with human review)

```text
/claude-reflect:reflect
```

Source: [reflect.md:1](<E:/AI_Workspace/plugins/plugins/claude-reflect/commands/reflect.md:1>) · SHA-256 `d22bde116ac8ca1de6f5ac0249a651e9446b1e7bf94487ac9ce496808e4e6364`

### `skip-reflect`

Discard queued learnings without processing

```text
/claude-reflect:skip-reflect
```

Source: [skip-reflect.md:1](<E:/AI_Workspace/plugins/plugins/claude-reflect/commands/skip-reflect.md:1>) · SHA-256 `a8ae7b434d2aaa2bc10d16d7b9ee82f113aae32f60bace9c13820f5c0c690de5`

### `view-queue`

View the learnings queue without processing

```text
/claude-reflect:view-queue
```

Source: [view-queue.md:1](<E:/AI_Workspace/plugins/plugins/claude-reflect/commands/view-queue.md:1>) · SHA-256 `9894822198dca7b9addfae91af0f696606714ff89acfd705c5b39f4d580142ce`

## Skills

No entries found in the inspected declarations.

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

No entries found in the inspected declarations.

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
