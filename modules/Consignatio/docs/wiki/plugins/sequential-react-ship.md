---
title: "sequential-react-ship"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# sequential-react-ship

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

thinkharder

Source: `E:/AI_Workspace/plugins/plugins/sequential-react-ship`. Version: `1.0.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `sequential-react-ship`

Runs a task of work as numbered ReAct thoughts with HITL profiles and execution-group skills. Use when the user asks to ship, deploy, implement, or run a dev/deploy cycle; when a Type 1, 1.5, or 2 door needs classification; or when sequential thinking plus ReAct is requested.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/sequential-react-ship/SKILL.md:1>) · SHA-256 `19927c531de06318ac67e9f7617004e3510156664bb30edef81b29b85156ea7d`

### `ship-close`

Closes and hands off a task for sequential-react-ship: verification gate, ship-critic review, calibration, red-team thought experiment, map-debt register, and handoff. Use at the CLOSE checkpoint.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-close/SKILL.md:1>) · SHA-256 `79f9739c2aafa797d457e7ece8e69123fb02c54017016a85af9a952393cd1e8f`

### `ship-gates`

Runs pre-action gates for sequential-react-ship: systems map, Cynefin domain, reversibility analysis, and setup thought experiment. Use at ORIENT and CLASSIFY checkpoints and when classifying door type before any Act.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-gates/SKILL.md:1>) · SHA-256 `a789f04e54aa3ee6cd43702a275272fd19bb11ecea7991fa32d2b3eb900897ac`

### `ship-planning`

Adds planning enrichments to sequential-react-ship: second-order thinking, margin of safety, Occam's razor, and consequence-tracing thought experiments. Use at the PLAN checkpoint for non-trivial Type 2, Type 1.5, and piloted Type 1 tasks.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-planning/SKILL.md:1>) · SHA-256 `a726d646a46177f0318f49542681cbac0d27c843642fc6564abc52a600ed0959`

### `ship-revision`

Handles recovery and recalibration for sequential-react-ship when territory falsifies the map. Use at the REVISE checkpoint after an Observation contradicts a Thought or when the door type changes.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-revision/SKILL.md:1>) · SHA-256 `a492dbfdc032875ed3e972ad3a1f9105b81d6df28e8e7163757ec164e921f97c`

### `ship-risk`

Runs failure and risk analysis for sequential-react-ship: pre-mortem, systems failure loops, Five Whys Plus root-cause analysis, and inversion. Use at the PREMORTEM checkpoint before the first Act of a non-trivial task.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-risk/SKILL.md:1>) · SHA-256 `144dea373f967449909508af46e61e76d32f7e7d56139e6fb6308589b75cd7a0`

### `ship-spine`

Provides the always-on ReAct spine for sequential-react-ship: one Thought, one Action, one Observation per cycle with verification and Bayesian update. Use on every ReAct cycle and when the conductor asks for the always-on execution discipline.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-spine/SKILL.md:1>) · SHA-256 `885cf0e7cc6612b8a7e0d72fbb0cfd518341c303792f1387e0f691b67e110492`

### `ship-verify`

Emits post-action verification stamps for sequential-react-ship: verification-before-completion, map-territory check, and Bayesian confidence update. Use when explicit evidence chains are needed after every Action.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/skills/ship-verify/SKILL.md:1>) · SHA-256 `f6ddaf471c1351210484dcada8b20cdb070f3fa112db6fa5d3215a3d640d42b9`

## Agents

### `ship-critic`

Use this agent when a task is about to close, when a Type 1 door is being considered, or when the user asks for an independent review of a ship/deploy decision. Validates that claims are backed by territory, that BUILD_STATUS was earned by actual runs, and that no Type 1 door was walked through without a HITL packet.

Source: [ship-critic.md:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/agents/ship-critic.md:1>) · SHA-256 `f53f831c68b9a2fad2fe88951be41c3c9cdbd11808883aea5c2b0c48b5fb5823`

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `scripts/ship-store.py`

ship-store.py — durable persistence for the Sequential ReAct Ship Cycle.

Stores every thinking artifact in a local SQLite database under the plugin's
.think/ directory. Records survive session compacts, restarts, and crashes.

Usage:
  init                       Create tables and return the db path.
  write --json '{...}'       Insert one ship event (strict validation).
  read --task-id <id>       Read all events for a task, ordered.
  read --last <n>            Read the last n events across all tasks.
  status                     Print db path, task count, event count.

```text
python "E:/AI_Workspace/plugins/plugins/sequential-react-ship/scripts/ship-store.py" --help
```

Declared arguments: `--json`, `--last`, `--task-id`

Declared subcommands:

| Command | Source help |
|---|---|
| `init` | Initialize the database |
| `write` | Write one event |
| `read` | Read events |
| `status` | Print database status |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --json | — | True | — | JSON payload |
| --task-id | — | False | — | Read events for a task |
| --last | int | False | — | Read last N events globally |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [ship-store.py:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/scripts/ship-store.py:1>) · SHA-256 `7c726c0f34769b17f540f81ecb30ffd2b80d31a29b5dc7b0c98bbde275771a21`

### `scripts/ship-validate-skills.py`

Validate sequential-react-ship plugin files against Claude Skill authoring best practices.

> _Byline: Claude Code · Fable 5 · 2026-08-19_

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [ship-validate-skills.py:1](<E:/AI_Workspace/plugins/plugins/sequential-react-ship/scripts/ship-validate-skills.py:1>) · SHA-256 `1ef62899f0d27e5b199ff9f607d3be619476f5a3cc94a4216c6bf03e478b4e4d`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
