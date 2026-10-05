---
title: "n8n"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# n8n

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Our n8n server (host/creds resolved at run time from N8N_API_URL / ~/.secrets/n8n-ovh2.env, never hardcoded): REST API tool for workflows/executions/import/export/webhooks, server facts and workflow inventory, plus the n8n CLI skill and n8n source-repo conventions. One progressive-disclosure entry skill (`n8n:n8n`) routing to 2 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/n8n`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `n8n`

(n8n) Our n8n server (host/creds resolved at run time from N8N_API_URL / ~/.secrets/n8n-ovh2.env, never hardcoded): REST API tool for workflows/executions/import/export/webhooks, server facts and workflow inventory, plus the n8n CLI skill and n8n source-repo conventions. Entry point / router — read this first, then load one member from references/. Triggers: n8n, workflow, execution, webhook, n8n-cli, activate workflow, n8n api. Members: n8n-cli, n8n-conventions.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/n8n/skills/n8n/SKILL.md:1>) · SHA-256 `d57b6a1ab71173b9a819f34663855cc08e9b6deb394d81ee733ed5fe810d27e9`

### `n8n-cli`

(n8n) Use the n8n CLI to manage workflows, credentials, executions, and more on an n8n instance. Use when the user asks to interact with n8n, automate workflows, manage credentials, or operate their instance from the command line.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/n8n/skills/n8n-cli/SKILL.md:1>) · SHA-256 `dffe9effc2786584bdde912ecbcf8eb5fbed66444ad6252e413bd4191a7f3fa4`

### `n8n-n8n-conventions`

(n8n) Quick reference for n8n patterns. Full docs /AGENTS.md

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/n8n/skills/n8n-conventions/SKILL.md:1>) · SHA-256 `62556e9dac03b1cd0764cccc61d937ffb189a497f237dd0579bbc03154cf9a18`

### `n8n-our-server`

(n8n) Where to find the truth about OUR n8n server (host, URL, credentials, boundary rules) without copying it here — every fact is resolved live or from its owning document.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/n8n/skills/our-server/SKILL.md:1>) · SHA-256 `71e023bca2a453a90f38254ba8d3c0f7c0e7fcf099383451a7cb8285c979f901`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/n8n/scripts/n8n_api.py`

n8n_api.py — reusable n8n public REST API (v1) helper for OUR server (host resolved from N8N_API_URL; never hardcoded).

Byline: Claude Code · Fable 5.1 · 2026-09-07

Credentials: N8N_API_URL / N8N_API_KEY from the environment, else parsed from ~/.secrets/n8n-ovh2.env with a
tolerant `KEY = value` regex (never sourced, never printed). Output is compact JSON on stdout; errors → stderr, exit 1.

  n8n_api.py facts                               LIVE server facts: host (from N8N_API_URL), healthz, version, workflow counts, creds source
  n8n_api.py health                              /healthz + API reachability (masked key id)
  n8n_api.py workflows [--active] [--tag T]      id, name, active, updatedAt, tags, node count
  n8n_api.py workflow <id|name> [--nodes]        one workflow; --nodes lists node name/type/(webhook path)
  n8n_api.py activate <id|name> | deactivate <id|name>
  n8n_api.py executions [--workflow <id|name>] [--status error|success|waiting|running] [--limit 20]
  n8n_api.py execution <id>                      status, timing, and the failing node + message if any
  n8n_api.py export <id|name> <file.json>        write the workflow JSON (nodes/connections/settings) to disk
  n8n_api.py import <file.json> [--name N]       create a workflow from a JSON file (does NOT activate)
  n8n_api.py update <id|name> <file.json>        replace nodes/connections/settings of an existing workflow
  n8n_api.py webhook <path> [--json '{...}'] [--test]   POST to https://<host>/webhook/<path> (or /webhook-test/)
  n8n_api.py tags | credentials                  list tags / credential names+types (never secrets)
  n8n_api.py raw <METHOD> </api/v1/path> [--json '{...}'] [--param k=v]

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [n8n_api.py:1](<E:/AI_Workspace/plugins/plugins/n8n/skills/n8n/scripts/n8n_api.py:1>) · SHA-256 `4cb8b2244e324d39e239542a6fb82aec2b54c71b5f5f373095b4e2e86e66abeb`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `n8n-add-data-table-column`
- `n8n-add-data-table-rows`
- `n8n-archive-workflow`
- `n8n-call-agent`
- `n8n-create-agent`
- `n8n-create-data-table`
- `n8n-create-folder`
- `n8n-create-workflow-from-code`
- `n8n-delete-agent`
- `n8n-delete-data-table-column`
- `n8n-discover-agent-assets`
- `n8n-execute-workflow`
- `n8n-explore-node-resources`
- `n8n-get-agent`
- `n8n-get-agent-builder-reference`
- `n8n-get-data-table-rows`
- `n8n-get-node-types`
- `n8n-get-workflow-best-practices`
- `n8n-get-workflow-details`
- `n8n-get-workflow-execution`
- `n8n-get-workflow-history`
- `n8n-get-workflow-sdk-reference`
- `n8n-get-workflow-version`
- `n8n-get-workflow-versions-diff`
- `n8n-list-agent-versions`
- `n8n-list-credentials`
- `n8n-list-n8n-connect-services`
- `n8n-list-workflow-tags`
- `n8n-move-workflows-to-folder`
- `n8n-mutate-agent`
- `n8n-prepare-workflow-pin-data`
- `n8n-publish-agent`
- `n8n-publish-workflow`
- `n8n-rename-data-table`
- `n8n-rename-data-table-column`
- `n8n-restore-workflow-version`
- `n8n-revert-agent`
- `n8n-search-agents`
- `n8n-search-data-tables`
- `n8n-search-folders`
- `n8n-search-nodes`
- `n8n-search-projects`
- `n8n-search-workflow-executions`
- `n8n-search-workflows`
- `n8n-test-workflow`
- `n8n-unpublish-agent`
- `n8n-unpublish-workflow`
- `n8n-update-agent-integration`
- `n8n-update-folder`
- `n8n-update-workflow`
- `n8n-validate-agent`
- `n8n-validate-node-config`
- `n8n-validate-workflow`
- `n8n-verify-agent-mcp-server`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
