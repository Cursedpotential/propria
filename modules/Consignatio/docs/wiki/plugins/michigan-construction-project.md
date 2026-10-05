---
title: "michigan-construction-project"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# michigan-construction-project

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Michigan construction lifecycle guidance with progressive disclosure and a Fable-to-Opus-to-Sonnet review chain.

Source: `E:/AI_Workspace/plugins/plugins/michigan-construction-project`. Version: `0.2.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `start-michigan-construction-project`

This skill should be used whenever the user says "let's start a construction project", "start a Michigan build", "help me build a house", "plan a renovation", "resume my construction project", asks Fable to coordinate Opus and Sonnet on a construction deliverable, asks what comes next in a Michigan construction project, or needs coordinated guidance spanning site, feasibility, design, code, permits, procurement, construction, inspections, occupancy, closeout, or operations. It maintains the project phase, progressively loads only the references needed for the current decision, and routes material deliverables through Opus planning, Sonnet execution, Opus review, and Fable final judgment.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/michigan-construction-project/skills/start-michigan-construction-project/SKILL.md:1>) · SHA-256 `706832cf249f40523c89711c8152c95faea9b0a495ee3d90f09644b9cd7db6d0`

## Agents

### `opus-construction-director`

Use this agent when Fable needs an overarching plan or an independent return review for a material Michigan construction deliverable. Typical triggers include planning a phase deliverable before Sonnet works, reviewing Sonnet's completed work against predeclared criteria, and resolving whether deficient work should be revised or rejected. See "When to invoke" in the agent body for worked scenarios. Do not invoke for routine intake, simple status, or clerical formatting.

Source: [opus-construction-director.md:1](<E:/AI_Workspace/plugins/plugins/michigan-construction-project/agents/opus-construction-director.md:1>) · SHA-256 `b6eaae6206d78364ce2f1d133219f1d84c1636565b650ed618928f1dd4b2d74b`

### `sonnet-construction-worker`

Use this agent when Fable supplies a bounded SONNET_WORK_ORDER authored by the Opus construction director. Typical triggers include producing one phase artifact, researching one defined Michigan authority question, updating specified project records, and making one correction requested by Opus. See "When to invoke" in the agent body for worked scenarios. Do not invoke without an explicit work-order ID and acceptance criteria.

Source: [sonnet-construction-worker.md:1](<E:/AI_Workspace/plugins/plugins/michigan-construction-project/agents/sonnet-construction-worker.md:1>) · SHA-256 `5a5e523d0389a7237a0e0cec143aca1a57e63b4cd74912d3ba86206caa903c48`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `scripts/project_router.py`

Validate and route a Michigan construction project state file.

```text
python "E:/AI_Workspace/plugins/plugins/michigan-construction-project/scripts/project_router.py" --help
```

Declared arguments: `--check`, `state`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| state | Path | positional | — | Path to construction-project.json |
| --check | store_true | False | — | Validate without emitting the route |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [project_router.py:1](<E:/AI_Workspace/plugins/plugins/michigan-construction-project/scripts/project_router.py:1>) · SHA-256 `9bf9ae26a4acaccecc0e14494bbf9def8f9b77929fe5b3a67209c587efaaad08`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
