---
title: "portkey"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# portkey

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Portkey AI Gateway: Python and TypeScript SDKs and the LiteLLM-to-Portkey migration playbook. One progressive-disclosure entry skill (`portkey:portkey`) routing to 3 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/portkey`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `pk-migrate-litellm-to-portkey`

(portkey) Migrate Python applications from LiteLLM to Portkey AI Gateway. Covers litellm.completion, Router, Proxy, fallbacks, caching, retries, callbacks, embeddings, and image generation. Use when the user wants to replace LiteLLM with Portkey or switch from LiteLLM to Portkey.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/portkey/skills/migrate-litellm-to-portkey/SKILL.md:1>) · SHA-256 `7f97ab4a4b076b1670337d5d24d6113481dfa6053eabe94f4a0af820b393e2b5`

### `portkey`

(portkey) Portkey AI Gateway: Python and TypeScript SDKs and the LiteLLM-to-Portkey migration playbook. Entry point / router — read this first, then load one member from references/. Triggers: portkey, gateway, litellm migration, fallbacks, virtual keys. Members: portkey-python-sdk, portkey-typescript-sdk, migrate-litellm-to-portkey.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/portkey/skills/portkey/SKILL.md:1>) · SHA-256 `7ec9aa1d74e6d0349c1d61bb453153e410b440064cb0e72a25008d6e96991240`

### `pk-portkey-python-sdk`

(portkey) Complete reference for the Portkey AI Gateway Python SDK with unified API access to 200+ LLMs, automatic fallbacks, caching, and full observability. Use when building Python applications that need LLM integration with production-grade reliability.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/portkey/skills/portkey-python-sdk/SKILL.md:1>) · SHA-256 `af73ba67061cf54c50da8e2738597689d30d6e944ce96f17782e0e32c73fc677`

### `pk-portkey-typescript-sdk`

(portkey) Integrate Portkey AI Gateway into TypeScript/JavaScript applications. Use when building LLM apps with observability, caching, fallbacks, load balancing, or routing across 200+ LLM providers.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/portkey/skills/portkey-typescript-sdk/SKILL.md:1>) · SHA-256 `474c7adbbfeccc976b756996660ee20458b029935759af410f3d7a57b3f04b04`

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
