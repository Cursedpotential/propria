---
title: "propria-toolbox"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# propria-toolbox

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Propria toolbox: every atomic tool behind the tool gateway and every ContextForge virtual server, listed in a generated catalog with how to reach and run them.

Source: `E:/AI_Workspace/plugins/plugins/propria-toolbox`. Version: `0.1.0`.
Registered: `True`. Installed manifests: not found in inspected manifests.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `toolbox`

Find and run Propria's atomic tools (parsers, extractors, repair and engine probes behind the tool gateway) and see which ContextForge virtual servers and tools exist. Use when the user asks what tools we have, which tool handles a file type, how to parse or repair a file, or how to run a tool on a b2://, r2:// or upload:// locator.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/skills/toolbox/SKILL.md:1>) · SHA-256 `3065e050a91bf081b6ea2aaf28f3df0f91c1767f78277b21c08260569d925c07`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `scripts/generate_catalog.py`

Generate skills/toolbox/references/catalog.md from the live toolbox.

Byline: Claude Code · Sonnet 5.5 · 2026-10-02

Sources (both read-only, both live):
  1. The tool gateway's GET /tools. Fetched by running curl ON ovh-app over ssh, with the bearer
     read from the token file there, so the token never reaches this machine or any file.
  2. ContextForge (/servers, /gateways, /tools). Bearer is CF_MCP_CLIENT_TOKEN, regex-parsed
     from ~/.secrets/contextforge.env (never sourced, never printed).

    python3 generate_catalog.py           # write references/catalog.md (left untouched if nothing changed)
    python3 generate_catalog.py --check   # exit 1 when the live catalog differs from the file

The output is sorted and stable: an unchanged catalog gives a zero diff. The "Generated" line is
kept from the existing file when the body is unchanged, and refreshed only when the body changes.
Descriptions are never edited here: they come from the code docstrings through the registry and
the gateway (docstring -> registry -> gateway -> this file).

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [generate_catalog.py:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/scripts/generate_catalog.py:1>) · SHA-256 `d0c7c58c23fe54e21c505ff2b0755aa358454a06169611f00ad06c19e744dcdb`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

### `atomic-tools`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/propria-toolbox/.mcp.json:1>) · SHA-256 `80ec450e7a35631957da4ce1e59e0da120c61cb08d3feb3b832c6650f34b63c7`


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `atomic-tools-atomic-tools`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
