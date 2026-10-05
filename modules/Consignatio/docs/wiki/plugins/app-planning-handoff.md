---
title: "app-planning-handoff"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# app-planning-handoff

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Six coordinated skills for producing research-backed application build kits: stack research, frontend/backend separation, ranked gotchas, phased prompts, and scaffold generation. Serves Claude Code and Codex from one plugin folder.

Source: `E:/AI_Workspace/plugins/plugins/app-planning-handoff`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `app-frontend-backend-split`

Sub-skill of app-planning-handoff. Use to determine whether an application has separable frontend and backend tiers, and if so, split planning, phases, prompts, and scaffolding into two independent workstreams with an explicit contract between them. Invoked as Stage 2 of the app planning pipeline, but can be used standalone when a user explicitly says an app should be "split" into frontend/backend.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-frontend-backend-split/SKILL.md:1>) · SHA-256 `01d8c2c0751a852073f97dd09606d6744bace03189b0c1fa7c079c5fb95ce043`

### `app-gotcha-audit`

Sub-skill of app-planning-handoff. Use to compile a ranked list of concrete, library-specific failure modes and traps for a decided technology stack — never generic advice like "write tests" or "handle errors." Invoked as Stage 3 of the app planning pipeline, but usable standalone when a user has a stack already and just wants "what should I watch out for" or "what will bite me building this."

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-gotcha-audit/SKILL.md:1>) · SHA-256 `d4606b3b8c5a5a79f8e8272eda14cf1abdd82b416cac0c3c7c2e2e1b2ddcda85`

### `app-phase-planning`

Sub-skill of app-planning-handoff. Use to turn a decided stack (and split, if applicable) into a numbered, exit-criteria-gated sequence of build phases, plus one copy-paste task prompt per phase for driving a coding agent through the plan. Invoked as Stage 4 of the app planning pipeline, but usable standalone when the user already has a stack/architecture and just wants "break this into phases" or "give me prompts to build this."

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-phase-planning/SKILL.md:1>) · SHA-256 `6ca2b9b217811d6470f46be8316aad5e824117d60773a2c325bff62f1ca92dd3`

### `app-planning-handoff`

Plan, architect, scope, or create a build guide and development handoff for a complete application, tool, service, or platform. Use for web, mobile, CLI, desktop, API, background service, browser extension, game, and full-stack platform planning. Coordinates five focused planning skills and assembles their outputs. Do not use for ordinary bug fixes, narrow features, or single-file scripts.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-planning-handoff/SKILL.md:1>) · SHA-256 `a61df0e9ce32bb5005b40577741024d0d309d019b28129d50dc55872cd3a6a06`

### `app-research-stack`

Sub-skill of app-planning-handoff. Use to research and decide the technology stack for a new application — real, current library/framework/API/platform choices with sourced trade-offs, not guessed-from-memory defaults. Invoked as Stage 1 of the app planning pipeline, but can also be used standalone when the user only wants "what stack should I use for X" without a full build kit.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-research-stack/SKILL.md:1>) · SHA-256 `3c7ce0806b9d77fa1cf9c44876e1885a170ad70570659587ea55594bd2f07948`

### `app-scaffold-generator`

Generate the actual stub file and folder structure for a planned application using the chosen stack's conventions, with TODO markers naming the implementing phase and appropriately scoped AGENTS.md operating rules. Use as Stage 5 of app-planning-handoff or standalone when the stack and architecture are already decided. For an existing repository, audit and plan only the required integration delta.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/app-planning-handoff/skills/app-scaffold-generator/SKILL.md:1>) · SHA-256 `a2d43d18bec7cd3f2c99e7be03c2dceefb7d20de1f59a0f27c03721e64062ead`

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
