---
title: "cf"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# cf

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Cloudflare developer platform: Workers, Durable Objects, Agents SDK, Sandbox SDK, Wrangler, Zero Trust (Cloudflare One), Email, Turnstile, web performance; plus a REST/Wrangler CLI tool and Cloudflare's hosted MCP servers. One progressive-disclosure entry skill (`cf:cf`) routing to 11 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/cf`. Version: `1.1.0`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

### `build-agent`

Build an AI agent on Cloudflare using the Agents SDK

```text
/cf:build-agent
```

Arguments: `['agent-description']`

Source: [build-agent.md:1](<E:/AI_Workspace/plugins/plugins/cf/commands/build-agent.md:1>) · SHA-256 `426baaf5068f297de04a996e6077eb4f5a145d52ed827d929998e1230f59e2c7`

### `build-mcp`

Build a remote MCP server on Cloudflare using McpAgent

```text
/cf:build-mcp
```

Arguments: `['mcp-description']`

Source: [build-mcp.md:1](<E:/AI_Workspace/plugins/plugins/cf/commands/build-mcp.md:1>) · SHA-256 `5a9187b6e253b663838c9354b1dd6b9e71c369554cb3c385dac76c81019b9015`

## Skills

### `cf-agents-sdk`

(cf) Build AI agents on Cloudflare Workers using the Agents SDK. Load when creating stateful agents, durable workflows, real-time WebSocket apps, scheduled tasks, MCP servers, chat applications, voice agents, or browser automation. Covers Agent class, state management, callable RPC, Workflows, durable execution, queues, retries, observability, and React hooks. Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/agents-sdk/SKILL.md:1>) · SHA-256 `9393c614f697c43bed7bba65ada26c33245ad02be5ac2273441a52b1b5ef6d35`

### `cf`

(cf) Cloudflare developer platform: Workers, Durable Objects, Agents SDK, Sandbox SDK, Wrangler, Zero Trust (Cloudflare One), Email, Turnstile, web performance; plus a REST/Wrangler CLI tool and Cloudflare's hosted MCP servers. Entry point / router — read this first, then load one member from references/. Triggers: cloudflare, workers, wrangler, durable objects, r2, pages, zero trust, cloudflare one, turnstile, agents sdk, dns, tunnel. Members: cloudflare, wrangler, workers-best-practices, durable-objects, agents-sdk, sandbox-sdk, cloudflare-one, cloudflare-one-migrations, cloudflare-email-service, turnstile-spin, web-perf.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cf/SKILL.md:1>) · SHA-256 `c0b15a8fa20101a512685523bc9b0af6223da4b67f66fead22ed8a45741db192`

### `cf-cloudflare`

(cf) Comprehensive Cloudflare platform skill covering Workers, Pages, storage (KV, D1, R2), AI (Workers AI, Vectorize, Agents SDK), feature flags (Flagship), networking (Tunnel, Spectrum), security (WAF, DDoS), and infrastructure-as-code (Terraform, Pulumi). Use for any Cloudflare development task. Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cloudflare/SKILL.md:1>) · SHA-256 `adb482dea1ac86afcf62cc0de5580178126fc553be3bc3312993302c06a5bd0d`

### `cf-cloudflare-email-service`

(cf) Send and receive transactional emails with Cloudflare Email Service (Email Sending + Email Routing). Use when building email sending (Workers binding or REST API), email routing, Agents SDK email handling, or integrating email into any app — Workers, Node.js, Python, Go, etc. Also use for email deliverability, SPF/DKIM/DMARC, wrangler email setup, MCP email tools, or when a coding agent needs to send emails. Even for simple requests like 'add email to my Worker' — this skill has critical config details.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cloudflare-email-service/SKILL.md:1>) · SHA-256 `5f8d6719a681add2f968a9a1c18695797858d0b8ebe92bb6e76dafc275047507`

### `cf-cloudflare-one`

(cf) Guides Cloudflare One Zero Trust and SASE work across Access, Gateway, WARP, Tunnel, Cloudflare WAN, DLP, CASB, device posture, and identity. Use when designing, configuring, troubleshooting, or reviewing Cloudflare One deployments. Retrieval-first: use current Cloudflare docs/API schemas instead of embedded product docs.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cloudflare-one/SKILL.md:1>) · SHA-256 `0a7fafa8f8fc8d9a79885b1fe9e1bffc53657a0073b3683201333a5f3d864f9b`

### `cf-cloudflare-one-migrations`

(cf) Plans migrations from Zscaler ZIA/ZPA, Palo Alto, legacy VPN, SWG, or SASE stacks to Cloudflare One. Use for migration assessments, policy mapping, rollout plans, and parity/gap analysis.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cloudflare-one-migrations/SKILL.md:1>) · SHA-256 `44e2e954aa107165057fe96a63a863cf12cdf698085a52259b30dad1db6daaca`

### `cf-durable-objects`

(cf) Create and review Cloudflare Durable Objects. Use when building stateful coordination (chat rooms, multiplayer games, booking systems), implementing RPC methods, SQLite storage, alarms, WebSockets, or reviewing DO code for best practices. Covers Workers integration, wrangler config, and testing with Vitest. Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/durable-objects/SKILL.md:1>) · SHA-256 `8316cc248890bba086c8b0b936413b126ba4b7ef783d82c4aefe60cbf368209a`

### `cf-sandbox-sdk`

(cf) Build sandboxed applications for secure code execution. Load when building AI code execution, code interpreters, CI/CD systems, interactive dev environments, or executing untrusted code. Covers Sandbox SDK lifecycle, commands, files, code interpreter, and preview URLs. Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/sandbox-sdk/SKILL.md:1>) · SHA-256 `3f395546a68d31c0877573499aeca0525359666fd587480495ddac35141deb8b`

### `cf-turnstile-spin`

(cf) Set up Cloudflare Turnstile end-to-end in a project — scan the codebase, create the widget via the Cloudflare API, deploy the managed siteverify Worker, write the frontend snippets, validate, and persist the skill. Load this when a user asks to add Turnstile, set up CAPTCHA, protect a form from bots, or fix a Turnstile integration. Mirrors developers.cloudflare.com/turnstile/spin.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/turnstile-spin/SKILL.md:1>) · SHA-256 `54f9430f105710cb8b4179dec9e28c89f6e67dc601e21ccf7468564104ed7a9e`

### `cf-web-perf`

(cf) Analyzes web performance using Chrome DevTools MCP. Measures Core Web Vitals (LCP, INP, CLS) and supplementary metrics (FCP, TBT, Speed Index), identifies render-blocking resources, network dependency chains, layout shifts, caching issues, and accessibility gaps. Use when asked to audit, profile, debug, or optimize page load performance, Lighthouse scores, or site speed. Biases towards retrieval from current documentation over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/web-perf/SKILL.md:1>) · SHA-256 `ac6fd1ed1f2895f2702fd3ad2d50eaeb303aff8c3def13c3c5d591783826d0ea`

### `cf-workers-best-practices`

(cf) Reviews and authors Cloudflare Workers code against production best practices. Load when writing new Workers, reviewing Worker code, configuring wrangler.jsonc, or checking for common Workers anti-patterns (streaming, floating promises, global state, secrets, bindings, observability). Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/workers-best-practices/SKILL.md:1>) · SHA-256 `663e9ce9cecd28740b2301850dab0c9f4210b1655d3eaad754aa15658ba07bfc`

### `cf-wrangler`

(cf) Cloudflare Workers CLI for deploying, developing, and managing Workers, KV, R2, D1, Vectorize, Hyperdrive, Workers AI, Containers, Queues, Workflows, Pipelines, and Secrets Store. Load before running wrangler commands to ensure correct syntax and best practices. Biases towards retrieval from Cloudflare docs over pre-trained knowledge.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/cf/skills/wrangler/SKILL.md:1>) · SHA-256 `504e739643e5a3655c664050f9858ab0bad6c55df26baea2269d1ff8351b3fb9`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/cf/scripts/cf_api.py`

cf_api.py — reusable Cloudflare REST v4 + Wrangler helper for the `cf` plugin.

Byline: Claude Code · Fable 5.1 · 2026-09-07

Credentials: CLOUDFLARE_API_TOKEN / CLOUDFLARE_ACCOUNT_ID from the environment, else parsed from
~/.secrets/cloudflare.env with a tolerant `KEY = value` regex (the file is NEVER sourced, values are
NEVER printed). Every command prints compact JSON to stdout; errors go to stderr with exit 1.

  cf_api.py whoami                      verify the token; prints token status + account id (masked)
  cf_api.py accounts                    accounts visible to the token
  cf_api.py zones [--name X]            zones (id, name, status, plan)
  cf_api.py dns <zone-name-or-id> [--type A] [--name host]
  cf_api.py r2-buckets                  R2 buckets in the account
  cf_api.py workers                     Worker scripts in the account
  cf_api.py pages                       Pages projects
  cf_api.py tunnels                     Cloudflare Tunnel list
  cf_api.py raw <METHOD> </client/v4/path> [--json '{...}'] [--param k=v ...]
  cf_api.py wrangler <args...>          run wrangler with the token injected (non-interactive), e.g. wrangler whoami

Paths starting with `/accounts/` or `/zones/` are prefixed with https://api.cloudflare.com/client/v4.
`{account}` inside a raw path is replaced with the account id.

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [cf_api.py:1](<E:/AI_Workspace/plugins/plugins/cf/skills/cf/scripts/cf_api.py:1>) · SHA-256 `c30b26c33c4b88bec85b0ee303c0c454eb8d2db0393d65cd1b033029be6407e8`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
