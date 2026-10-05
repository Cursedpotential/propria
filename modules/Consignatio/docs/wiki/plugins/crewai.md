---
title: "crewai"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# crewai

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

CrewAI framework: architecture choice, project scaffolding, agent and task design, and official-docs lookup. One progressive-disclosure entry skill (`crewai:crewai`) routing to 4 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/crewai`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `crew-ask-docs`

(crewai) Query the official CrewAI documentation for answers. Use when the user has a CrewAI question that isn't fully covered by the getting-started, design-agent, design-task skills — e.g., specific API details, configuration options, advanced features, troubleshooting errors, enterprise features, tool references, or anything where the latest docs are the best source of truth.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/crewai/skills/ask-docs/SKILL.md:1>) · SHA-256 `eb69183a1c2a67a27d689dfc96b4a60dad1462df7e97a5742e1944104605e678`

### `crewai`

(crewai) CrewAI framework: architecture choice, project scaffolding, agent and task design, and official-docs lookup. Entry point / router — read this first, then load one member from references/. Triggers: crewai, crew, flow, kickoff, agents.yaml, tasks.yaml, CrewBase. Members: ask-docs, design-agent, design-task, getting-started.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/crewai/skills/crewai/SKILL.md:1>) · SHA-256 `58b1642567286b72796602d056794f65b00fb7799464db4d1122080b5045ccf5`

### `crew-design-agent`

(crewai) CrewAI agent design and configuration. Use when creating, configuring, or debugging crewAI agents — choosing role/goal/backstory, selecting LLMs, assigning tools, tuning max_iter/max_rpm/max_execution_time, enabling planning/code execution/delegation, setting up knowledge sources, using guardrails, or configuring agents in YAML vs code.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/crewai/skills/design-agent/SKILL.md:1>) · SHA-256 `7d6b5dc3bb36b1e2231e78447a1727675b149b3b847347517837bc4b91f7d87e`

### `crew-design-task`

(crewai) CrewAI task design and configuration. Use when creating, configuring, or debugging crewAI tasks — writing descriptions and expected_output, setting up task dependencies with context, configuring output formats (output_pydantic, output_json, output_file), using guardrails for validation, enabling human_input, async execution, markdown formatting, or debugging task execution issues.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/crewai/skills/design-task/SKILL.md:1>) · SHA-256 `7e593c9dcdb460abed84eedad130138ae3846675c750bfa16d3b2a47f57433f5`

### `crew-getting-started`

(crewai) CrewAI architecture decisions and project scaffolding. Use when starting a new crewAI project, choosing between LLM.call() vs Agent.kickoff() vs Crew.kickoff() vs Flow, scaffolding with 'crewai create flow', setting up YAML config (agents.yaml, tasks.yaml), wiring @CrewBase crew.py, writing Flow main.py with @start/@listen, building experimental conversational Flows with handle_turn()/chat(), or using {variable} interpolation.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/crewai/skills/getting-started/SKILL.md:1>) · SHA-256 `f00140741e79c2dd815519e244c4c210c125dd387b2c60acfcc714128a5c6be9`

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
