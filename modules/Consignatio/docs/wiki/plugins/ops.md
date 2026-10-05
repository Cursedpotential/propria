---
title: "ops"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# ops

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Infra and workstation operations: Docker and Compose, SSH, gcloud, Doppler secrets and PyPI publishing, env/PATH hygiene, disk cleanup, OAuth PKCE bootstrap, Playwright CLI, Conductor workflows, terminal export, modern CLI tooling. One progressive-disclosure entry skill (`ops:ops`) routing to 18 member skills loaded on demand.

Source: `E:/AI_Workspace/plugins/plugins/ops`. Version: `1.1.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `ops-cli-ninja`

(ops) Master CLI navigation and code exploration using modern command-line tools. Use this skill when navigating repositories, searching for files/text/code patterns, or working with structured data (JSON/YAML/XML). Provides guidance on fd (file finding), rg (text search), ast-grep (code structure), fzf (interactive selection), jq (JSON), and yq (YAML/XML).

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/cli-ninja/SKILL.md:1>) · SHA-256 `3e0a01aff4aeeac381a289694ebb8d82faf63e6e7b732d92c5e5534f62237787`

### `ops-conductor`

(ops) Create, run, monitor, and manage Conductor workflows and tasks. Use when the user wants to define workflows, start executions, check status, pause/resume/terminate/retry workflows, or signal tasks. Uses the `conductor` CLI or falls back to bundled REST API script. Requires CONDUCTOR_SERVER_URL.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/conductor/SKILL.md:1>) · SHA-256 `e622db713bb4677771961ddd6141d092265fc2ff964c3eee33fa8e819a057d9c`

### `ops-developing-with-docker`

(ops) Debugging-first guidance for professional Docker development across CLI, Compose, Docker Desktop, and Rancher Desktop. Use when asked to 'debug Docker', 'troubleshoot containers', 'fix Docker networking', 'resolve volume permissions', or 'Docker Compose issues', and when explaining cross-platform runtime behavior (Linux, macOS, Windows/WSL2) or Docker runtime architecture.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/developing-with-docker/SKILL.md:1>) · SHA-256 `d428449fa3d0e3a9bfe53e4e95669aa087ca61cdb3bb3a8379467543bf739d4c`

### `ops-disk-hygiene`

(ops) macOS disk cleanup, cache pruning, stale file detection, and Downloads triage. TRIGGERS - disk space, cleanup, disk usage, stale files, cache clean, brew cleanup, forgotten files, Downloads cleanup, free space, storage, dust, dua, gdu, ncdu.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/disk-hygiene/SKILL.md:1>) · SHA-256 `0f2662729962b0efdc841b70ab06716138b06209086ccc924273f6000a558dae`

### `ops-docker-management`

(ops) Manage Docker containers, images, volumes, networks, and Compose stacks — lifecycle ops, debugging, cleanup, and Dockerfile optimization.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/Docker/SKILL.md:1>) · SHA-256 `d6992ef668abb3b236b1b108f84f6644b30e3d73ec51a6da55131e5bcbc743f1`

### `ops-docker-compose`

(ops) Docker Compose V2 and Compose Specification expertise for writing correct compose.yaml and docker-compose.yml files. Use when the user mentions Docker Compose, compose.yaml, docker-compose.yml, docker compose CLI, multi-container apps, profiles, healthchecks, depends_on, networks, volumes, secrets, build contexts, or avoiding deprecated V1 patterns.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/docker-compose/SKILL.md:1>) · SHA-256 `f94933877bb543fff1c2f3a724b1619520b88fe988c28bb0999fb9ea8f73d963`

### `ops-docker-expert`

(ops) Docker Expert. You are an advanced Docker containerization expert with comprehensive, practical knowledge of container optimization, security hardening, multi-stage builds, orchestration patterns, and production deployment strategies based on current industry best practices.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/docker-expert/SKILL.md:1>) · SHA-256 `2586beed08d75a4ce84546bb6d08c0b1d9f521b0dc2ca2c1b5090f458ab30ceb`

### `ops-doppler-secret-validation`

(ops) Validate and test Doppler secrets. TRIGGERS - add to Doppler, store secret, validate token, test credentials.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/doppler-secret-validation/SKILL.md:1>) · SHA-256 `533fb68e9524388e19bc2000491605db3304b426c52ca8882dc983c6ebb72dad`

### `ops-doppler-workflows`

(ops) Doppler credential and publishing workflows. TRIGGERS - PyPI publish, AWS credentials, Doppler secrets.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/doppler-workflows/SKILL.md:1>) · SHA-256 `898927713d8644a9d435af6968c7e38c09bdd7b5f71da5f96cedb9d701061521`

### `ops-env-inventory`

(ops) Inventory Windows user environment variables and cross-reference agent configuration and local secret sources.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/env-inventory/SKILL.md:1>) · SHA-256 `2555b3ee5c9cb0f7546dcefff91393a8fe158469aac449dc1943f28677482ff8`

### `ops-mastering-gcloud-commands`

(ops) Expert-level Google Cloud CLI (gcloud) skill for managing GCP resources. Use when working with 'gcloud commands', 'cloud run deploy', 'alloydb', 'cloud sql', 'workload identity federation', 'iam permissions', 'vpc networking', 'secret manager', or 'artifact registry'. Covers installation, authentication, IAM, Cloud Run, Cloud Storage, VPC, AlloyDB, Firebase, and CI/CD integration with GitHub Actions and Cloud Build.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/mastering-gcloud-commands/SKILL.md:1>) · SHA-256 `94a02f8808fc6e585584bc139fe26f3284082e91d244bbdfd2a8cc4c47fdc2a6`

### `ops-oauth-pkce-helper`

(ops) Run an OAuth2 Authorization Code + PKCE (S256) browser login for any provider and emit agent-parseable TOKEN:/REFRESH:/EXPIRES: markers, including rclone-shaped token JSON. Use when the user says oauth, pkce, rclone authorize, onedrive token, gdrive token, google drive oauth, microsoft oauth, browser login flow, or needs to bootstrap rclone OneDrive/GDrive remotes without rclone's own browser dance. Presets for Microsoft OneDrive and Google Drive included.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/oauth-pkce-helper/SKILL.md:1>) · SHA-256 `7f1ecef5b9ab29da931d1455c69cc2ac8d9b77b12f0ee9db148c014075bbd913`

### `ops`

(ops) Infra and workstation operations: Docker and Compose, SSH, gcloud, Doppler secrets and PyPI publishing, env/PATH hygiene, disk cleanup, OAuth PKCE bootstrap, Playwright CLI, Conductor workflows, terminal export, modern CLI tooling. Entry point / router — read this first, then load one member from references/. Triggers: docker, compose, ssh, gcloud, doppler, secrets, PATH, env vars, disk space, oauth, playwright, conductor, fd rg jq, terminal export. Members: Docker, docker-compose, docker-expert, developing-with-docker, ssh, mastering-gcloud-commands, doppler-secret-validation, doppler-workflows, pypi-doppler, env-inventory, source-command-env-inventory, path-rationalization, disk-hygiene, oauth-pkce-helper, playwright-cli, conductor, cli-ninja, terminal-print.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/ops/SKILL.md:1>) · SHA-256 `528931e30e698cecec5c0a887ce31353b9e97470ffd844a02ca255e0f7d9d29b`

### `ops-path-rationalization`

(ops) Audit and clean shell PATH pollution in .bashrc, .zshrc, .zshenv, .profile. Use when PATH has junk, wrong binary resolves, temp dirs in PATH, or duplicates.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/path-rationalization/SKILL.md:1>) · SHA-256 `c7da1cd32a8d381824f8dea3097e874f13a458b2ee94e55860092d63aeb7e699`

### `ops-playwright-cli`

(ops) Automates browser interactions for web testing, form filling, screenshots, and data extraction. Use when the user needs to navigate websites, interact with web pages, fill forms, take screenshots, test web applications, or extract information from web pages.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/playwright-cli/SKILL.md:1>) · SHA-256 `053676ed0a873526b8546b60ee598d9e6c78b88ad48ea5040bbde9bc8b39d9c3`

### `ops-pypi-doppler`

(ops) LOCAL-ONLY PyPI publishing with Doppler credentials. TRIGGERS - publish to PyPI, pypi upload, local publish. NEVER use in CI/CD.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/pypi-doppler/SKILL.md:1>) · SHA-256 `35c530cb5303b8fc85b054a352c374ef0d9bf4edeca1a3e3137dec6cbf013e8c`

### `ops-source-command-env-inventory`

(ops) List all User environment variables with cross-references to Codex, OpenCode, and .secrets sources

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/source-command-env-inventory/SKILL.md:1>) · SHA-256 `33581d51d2b4224b424ac3eb741159031c226587d1dbda5c13623c5c49fc3305`

### `ops-ssh`

(ops) SSH remote access - connections, tunnels, keys, file transfers. Use when connecting to servers, managing SSH keys, setting up port forwarding, or transferring files with scp/rsync.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/ssh/SKILL.md:1>) · SHA-256 `ce4e99e2c9ce50aff73e96e16fc4a29e54e6274d80bd7f6a04f26b673eb2eb4f`

### `ops-tailscale`

(ops) Guide for installing, configuring, and managing Tailscale, headscale, and the Tailscale product family. Covers core mesh VPN (exit nodes, subnet routers, access controls, SSH, MagicDNS), Docker and Kubernetes integration, CI/CD pipelines, ephemeral nodes, infrastructure access, site-to-site networking, app connectors, Aperture (AI/LLM gateway for governance, cost control, usage visibility), device posture, MDM, SCIM provisioning, SSH and kubectl session recording, tsrecorder, Taildrop, Tailscale Serve, Funnel, and building Go applications that embed Tailscale via the tsnet library. Use when someone asks about Tailscale networking, mesh VPN, VPN replacement, containers, Kubernetes operator, CI/CD runners, device management, session recording, audit logging, LLM API access, AI cost control, file sharing, exposing internal services, or writing a Go program that joins a tailnet as its own device — even when they describe the scenario without naming the product.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/tailscale/SKILL.md:1>) · SHA-256 `94b4d2d81a6cb2ad5fb833c903afb3650fced1a7299beddcba4c70d86b8da684`

### `ops-terminal-print`

(ops) Export terminal output (clipboard or a file) to a Markdown file, optionally an HTML page for Ctrl+P printing. Windows-native; the old macOS/iTerm2 print path is kept as a reference only. TRIGGERS - export terminal, terminal to md, print terminal, terminal PDF, print session output.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/ops/skills/terminal-print/SKILL.md:1>) · SHA-256 `507debd59094cb8c71a67c61a835d8c220a5d398014ac2b9207db42791df362e`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `skills/conductor/scripts/conductor_api.py`

Conductor REST API fallback — stdlib only, no third-party packages.

Use when the `conductor` CLI is not installed.
Requires CONDUCTOR_SERVER_URL env var. CONDUCTOR_AUTH_TOKEN is optional.

```text
python "E:/AI_Workspace/plugins/plugins/ops/skills/conductor/scripts/conductor_api.py" --help
```

Declared arguments: `--correlation-id`, `--count`, `--file`, `--id`, `--include-tasks`, `--input`, `--input-file`, `--name`, `--output`, `--query`, `--reason`, `--size`, `--sort`, `--status`, `--task-ref`, `--task-type`, `--version`, `--workflow-id`

Declared subcommands:

| Command | Source help |
|---|---|
| `list-workflows` | List all workflow definitions |
| `get-workflow` | Get a workflow definition |
| `create-workflow` | Create a workflow definition from JSON file |
| `update-workflow` | Update a workflow definition from JSON file |
| `delete-workflow` | Delete a workflow definition |
| `start-workflow` | Start a workflow execution |
| `get-execution` | Get workflow execution status |
| `search-workflows` | Search workflow executions |
| `pause-workflow` | Pause a running workflow |
| `resume-workflow` | Resume a paused workflow |
| `terminate-workflow` | Terminate a workflow |
| `restart-workflow` | Restart a completed workflow |
| `retry-workflow` | Retry the last failed task |
| `signal-task` | Signal a task (async) |
| `signal-task-sync` | Signal a task (sync, returns workflow) |
| `poll-task` | Poll for tasks of a given type |
| `queue-size` | Get task queue size |

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --name | — | True | — | — |
| --version | — | False | — | — |
| --file | — | True | — | — |
| --file | — | True | — | — |
| --name | — | True | — | — |
| --version | — | True | — | — |
| --name | — | True | — | — |
| --version | — | False | — | — |
| --correlation-id | — | False | — | — |
| --input | — | False | — | Inline JSON input |
| --input-file | — | False | — | Path to JSON input file |
| --id | — | True | — | — |
| --include-tasks | store_true | False | — | — |
| --status | — | False | — | — |
| --query | — | False | — | — |
| --size | int | False | — | — |
| --sort | — | False | — | — |
| --id | — | True | — | — |
| --id | — | True | — | — |
| --id | — | True | — | — |
| --reason | — | False | — | — |
| --id | — | True | — | — |
| --id | — | True | — | — |
| --workflow-id | — | True | — | — |
| --task-ref | — | True | — | — |
| --status | — | True | — | — |
| --output | — | False | — | JSON output to pass to the task |
| --workflow-id | — | True | — | — |
| --task-ref | — | True | — | — |
| --status | — | True | — | — |
| --output | — | False | — | JSON output to pass to the task |
| --task-type | — | True | — | — |
| --count | int | False | — | — |
| --task-type | — | False | — | — |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [conductor_api.py:1](<E:/AI_Workspace/plugins/plugins/ops/skills/conductor/scripts/conductor_api.py:1>) · SHA-256 `11c7039cf81fc2ad88b3bb8ed5ea82d4842936d417ea21b0939d840e5db1b582`

### `skills/oauth-pkce-helper/scripts/pkce_auth.py`

oauth-pkce-helper — generic OAuth2 Authorization Code + PKCE (S256) browser login.

Adapted from the confidence plugin's onboard-confidence/auth.py (Auth0 PKCE flow),
generalized for any provider. Python 3.10+ stdlib only. Windows-safe.

Usage:
    python pkce_auth.py --provider onedrive --client-id <APP_ID>
    python pkce_auth.py --provider gdrive --client-id <ID> --client-secret-env GDRIVE_CLIENT_SECRET
    python pkce_auth.py --authorize-url URL --token-url URL --client-id ID         --scopes "scope1 scope2" [--port 53682] [--redirect-path /]         [--extra-param key=value ...] [--rclone-json] [--timeout 300]
    python pkce_auth.py --self-test        # verify PKCE S256 against RFC 7636 vector, no network

Config precedence: CLI args > environment > providers.json preset.
Environment fallbacks: OAUTH_CLIENT_ID, OAUTH_AUTHORIZE_URL, OAUTH_TOKEN_URL,
    OAUTH_SCOPES, OAUTH_PORT. Client secret is env-only (--client-secret-env NAME,
    default OAUTH_CLIENT_SECRET) and is NEVER printed or echoed.

Agent-parseable stdout markers:
    WAITING_FOR_LOGIN            browser opened, localhost callback armed
    TOKEN:<access_token>         success
    REFRESH:<refresh_token>      refresh token (if granted)
    EXPIRES:<seconds>            expires_in from the token response
    EXPIRY:<rfc3339>             computed absolute expiry (local tz, RFC 3339)
    RCLONE_TOKEN:<json>          (--rclone-json) token JSON in rclone's expected shape
    AUTH_ERROR:<msg> / TOKEN_ERROR:<msg> / CONFIG_ERROR:<msg>

Exit codes: 0 = success, 1 = error.

```text
python "E:/AI_Workspace/plugins/plugins/ops/skills/oauth-pkce-helper/scripts/pkce_auth.py" --help
```

Declared arguments: `--authorize-url`, `--client-id`, `--client-secret-env`, `--extra-param`, `--no-browser`, `--port`, `--provider`, `--rclone-json`, `--redirect-path`, `--scopes`, `--self-test`, `--timeout`, `--token-url`

Argument contracts (subcommand ownership follows the cited parser; not every flag belongs to every command):

| Argument | Type/action | Required | Choices | Meaning |
|---|---|---|---|---|
| --provider | — | False | — | preset name from providers.json (e.g. onedrive, gdrive) |
| --client-id | — | False | — | OAuth client id (or env OAUTH_CLIENT_ID) |
| --client-secret-env | — | False | — | name of env var holding the client secret, if the provider needs one (default OAUTH_CLIENT_SECRET; value is never printed) |
| --authorize-url | — | False | — | authorization endpoint URL |
| --token-url | — | False | — | token endpoint URL |
| --scopes | — | False | — | space-separated scopes, e.g. "Files.Read.All offline_access" |
| --port | int | False | — | localhost callback port (default: preset value or 53682) |
| --redirect-path | — | False | — | callback path, e.g. / or /callback (default: preset value or /) |
| --extra-param | append | False | — | extra authorize-request query param (repeatable), e.g. --extra-param access_type=offline |
| --timeout | int | False | — | seconds to wait for the browser callback (default 300) |
| --rclone-json | store_true | False | — | also print RCLONE_TOKEN:<json> in rclone's token format |
| --no-browser | store_true | False | — | print the authorize URL instead of opening a browser |
| --self-test | store_true | False | — | verify PKCE S256 generation against the RFC 7636 test vector and exit |

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [pkce_auth.py:1](<E:/AI_Workspace/plugins/plugins/ops/skills/oauth-pkce-helper/scripts/pkce_auth.py:1>) · SHA-256 `04e864ad6d19de473ae9bfbf4b766d28f3618fb0b1577a8c12d15abf7b1161d2`

## Mcp Tools

No entries found in the inspected declarations.

## Mcp Servers

No entries found in the inspected declarations.


Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
