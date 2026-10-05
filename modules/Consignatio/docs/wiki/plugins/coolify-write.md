---
title: "coolify-write"
type: tool-reference
status: SOURCE_VERIFIED
date: 2026-10-04
generated_by: "Codex / GPT-6"
revision: 1
tags: [propria, tools, wiki]
---

# coolify-write

> _Byline: Codex · GPT-6 · 2026-10-04 — generated from the cited sources._

Coolify ops — 43 tools over the Coolify v4 REST API through ContextForge (hosted server built from this plugin): read, app/database/service/project create + guarded delete, envs, deploy/start/stop/restart, logs, deployments, plus a coolify_api passthrough. Full parity with the retired coolify-cli.cjs skill.

Source: `E:/AI_Workspace/plugins/plugins/coolify-write`. Version: `1.4.1`.
Registered: `True`. Installed manifests: .claude.

## How to invoke it

Slash commands run inside an agent app. Terminal commands run in PowerShell. MCP tools require an attached server and are called by the agent or an MCP client. A skill is an instruction package, not a standalone executable.

Descriptions below are extracted from source metadata/docstrings. This is discovery evidence, not a claim every service invocation passed.

## Commands

No entries found in the inspected declarations.

## Skills

### `coolify-write`

Manage Coolify infrastructure via API — deploy, start, stop, restart applications; create and manage databases (8 engines), services, and projects; set environment variables; read build and runtime logs; manage servers, teams, private keys, GitHub apps, and backups. Use when the user wants to deploy to Coolify, check why a Coolify app is down or a build failed, create or delete Coolify resources, or inspect Coolify servers and projects.

Source: [SKILL.md:1](<E:/AI_Workspace/plugins/plugins/coolify-write/skills/coolify-write/SKILL.md:1>) · SHA-256 `ba17c42c36172c3f91fdbdd966bdd74fe24509dd9a44ffe50ae1e3846099bb5b`

## Agents

No entries found in the inspected declarations.

## Cli Entries

No entries found in the inspected declarations.

## Scripts

### `scripts/api-shape-probe.py`

api-shape-probe.py — dev utility: probe Coolify REST read endpoints and print
response shapes (keys, list lengths, port fields). Token is read from the env
file pointed at by COOLIFY_ENV_FILE (or the default path below); it is NEVER
printed. Use to confirm response shapes before wiring the MCP server tools.

Run:
  python api-shape-probe.py            # probes /applications, /servers, /applications/{uuid}
  python api-shape-probe.py <uuid>     # probes a specific application uuid

> Byline: Claude Code · glm-5.2:cloud (via Ollama) · 2026-07-05

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [api-shape-probe.py:1](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/api-shape-probe.py:1>) · SHA-256 `d34dac7e5157a2d552fa9b6c7a76ff710a1195ac20b64c69c144666ed53f4134`

### `scripts/server.py`

coolify MCP server (consolidated) — read cluster + write cluster.

Read cluster mirrors the hosted Coolify MCP tool names exactly so existing
agent calls keep working after the ~/.claude.json cutover:
  get_infrastructure_overview, list_servers, get_server, list_projects,
  list_applications, get_application, list_databases, get_database,
  list_services, get_service.

Write cluster (5 tools):
  create_application, update_application, upsert_application_envs,
  delete_application, check_port_collision.

Diagnostics (1 tool, added 2026-10-02):
  github_app_webhook (the GitHub App's webhook URL and recent deliveries).

Lifecycle cluster (7 tools):
  deploy_application, restart_application, stop_application, start_application,
  get_deployment, list_deployments_for_app, get_application_logs.

Parity cluster (18 tools, added 2026-08-23) — closes the gap against the older
coolify-cli.cjs skill:
  create_database, start/stop/restart/delete_database,
  create_service, start/stop/restart/delete_service,
  list_application_envs, delete_application_env,
  create_project, update_project, delete_project,
  list_deployments, cancel_deployment,
  coolify_api (generic REST passthrough for teams, private keys, github apps,
  database backups, and server administration — see CLI-PARITY.md).

Transport-agnostic: stdio (default) or streamable-HTTP, selected by
MCP_TRANSPORT env var. This file is the ONE source for both: the desktop runs it
over stdio, and the hosted `coolify-mcp` app on ovh-app (behind ContextForge) is
built from this same folder (Dockerfile, compose.hosted.yaml) with MCP_TRANSPORT=http.

Token handling: reads COOLIFY_API_TOKEN + base URL from the env file at
COOLIFY_ENV_FILE (default C:/Users/matts/.secrets/coolify-ionos-api.env); the
same names in the process environment win over the file, which is how the hosted
container gets them. The token is NEVER logged and NEVER copied into ~/.claude.json.

> Byline: Claude Code · glm-5.2:cloud (via Ollama) · 2026-07-05
> Hosting patches merged back from the old hosted copy: Claude Code · Fable 5.1 · 2026-10-01
> github_app_webhook and the auto-deploy switch: Claude Code · Opus 5.5 · 2026-10-02

Validation: source inspected; execute only when the documented owning runtime permits.

Source: [server.py:1](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

## Mcp Tools

### `get_infrastructure_overview`

Coolify version, all servers, projects with resource counts, and aggregates.

Discovery
---------
Start here. Single call returns the whole topology — servers, projects,
and counts of applications/databases/services — so you can pick the right
UUID before calling the per-resource tools.

When to use
-----------
- First call in a session to map the fleet.
- Before any write op, to confirm the target server/project/uuid exists.
- NOT for: drilling into one resource (use get_application/get_server/etc).

Returns
-------
dict
    {
      coolify_version: str,
      servers: list[dict],          # name, uuid, ip, is_reachable
      projects: list[dict],         # name, uuid (per-project app count
                                    # needs an /environments call — drill via
                                    # list_applications for per-app detail)
      counts: {applications, databases, services, servers},
    }

Validation: source declaration; invocation not tested.

Source: [server.py:290](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:290>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_servers`

List servers owned by the authenticated team. Returns summary (uuid, name, ip, is_reachable).

Discovery
---------
Use get_infrastructure_overview for a one-shot fleet map, or this for
paginated server listing. Pass the returned uuid to get_server for details.

| Parameter | Declared type |
|---|---|
| `page` | int |
| `per_page` | int |

Validation: source declaration; invocation not tested.

Source: [server.py:339](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:339>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_server`

Full details for one server by UUID.

Parameters
----------
uuid : str
    Server UUID. Get it from list_servers or get_infrastructure_overview.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:351](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:351>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_projects`

List projects (summary: uuid, name, description). full=True returns whole records.

| Parameter | Declared type |
|---|---|
| `page` | int |
| `per_page` | int |
| `full` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:386](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:386>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_applications`

List applications (summary: uuid, name, status, fqdn, git_repository).

Parameters
----------
tag : str, optional
    Filter by tag name.
full : bool
    Return whole records (redacted) instead of the summary.

| Parameter | Declared type |
|---|---|
| `page` | int |
| `per_page` | int |
| `tag` | str \| None |
| `full` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:393](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:393>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_application`

Full details for one application by UUID (93-field record: status, env, compose, ports, health...).

Do / Don't
----------
Do: pass the exact uuid from list_applications.
Don't: guess or wildcard — there is no fuzzy match; a wrong uuid 404s.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:412](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:412>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_databases`

List standalone databases (summary: uuid, name, status, type). full=True for whole records.

| Parameter | Declared type |
|---|---|
| `page` | int |
| `per_page` | int |
| `full` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:424](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:424>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_database`

Full details for one standalone database by UUID.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:431](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:431>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_services`

List services (multi-container stacks; summary: uuid, name, status).

The raw record carries docker_compose_raw with every environment value in plain text, so the
summary is the default. full=True returns the whole record, redacted.

| Parameter | Declared type |
|---|---|
| `page` | int |
| `per_page` | int |
| `full` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:437](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:437>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_service`

Full details for one service (multi-container stack) by UUID. Secret values are redacted.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:448](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:448>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_service_env`

Environment variables a service declares in its compose, grouped by compose service.

Added 2026-09-30 (Claude Code · Opus 5) because redacting get_service closed the only route
to these values, and one job legitimately needs them: migrating them into a secrets manager.
Applications have list_application_envs; services keep their environment inside
docker_compose_raw, which get_service masks.

This returns the env pairs and NOTHING else -- no compose document, no webhook secrets, no
image, volume or network detail -- so reading a value here does not also expose the rest of
the stack the way an unredacted get_service would.

Parameters
----------
uuid : str     # service UUID (list_services)
reveal : bool  # False (default) gives names with value LENGTHS, enough to inventory;
               # True gives the values, and is the only way to read them through this plugin.

Returns
-------
dict  {uuid, name, services: {<compose service>: {<NAME>: value | "<N chars>"}}, revealed}

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `reveal` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:454](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:454>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `set_service_image`

Point one container of a Coolify service at a new image tag, without exposing its compose.

Reads the service's compose, changes ONLY services.<service_name>.image, writes it back, then
re-reads it and checks that nothing else changed. If the read-back differs anywhere other
than that one field, the original compose is restored. Secret values never appear in the
result. Follow with a deploy of the service to start the new image.

Parameters
----------
uuid : str          # service UUID (get_service / list_services)
service_name : str  # the compose service key, e.g. "docstore"
image : str         # new image reference, e.g. "propria-docstore:0.8.1-r7"
confirm : bool      # must be True to write; False reports what would change

Returns
-------
dict  {uuid, service, old_image, new_image, written, verified, restored?}

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `service_name` | str |
| `image` | str |
| `confirm` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:605](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:605>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `update_application`

Repoint an existing application's build inputs (PATCH /applications/{uuid}).

Added 2026-09-28 (Claude Code · Opus 5): repointing an app at a different compose file or
base directory had no tool, so it would have meant a raw coolify_api call. Changing an app in
place is how a service gets replaced without standing up a parallel stack.
is_auto_deploy_enabled added 2026-10-02 (Claude Code · Opus 5.5).

Only the fields you pass are touched. Writes nothing unless confirm=True; then it re-reads the
application and checks every field it set actually took. If any did not, the previous values
are written back and the result says so, so a half-applied repoint never survives silently.

Parameters
----------
uuid : str                       # application UUID (list_applications)
base_directory : str, optional   # build context root, e.g. "/" for the monorepo root
docker_compose_location : str, optional  # compose path RELATIVE TO base_directory
watch_paths : str, optional      # newline-separated globs, matched against repository root
is_auto_deploy_enabled : bool, optional
    Whether a push deploys the app. Off on every app by owner rule (2026-09-20, reaffirmed
    2026-10-02): deploys are explicit. Read back from the record's `settings`.
confirm : bool                   # must be True to write; False reports the diff only

Returns
-------
dict  {uuid, changes: {field: {from, to}}, written, verified, restored?}

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `name` | str \| None |
| `description` | str \| None |
| `base_directory` | str \| None |
| `docker_compose_location` | str \| None |
| `watch_paths` | str \| None |
| `git_repository` | str \| None |
| `git_branch` | str \| None |
| `is_auto_deploy_enabled` | bool \| None |
| `confirm` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:667](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:667>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `create_application`

Create a new Coolify application (default: docker-compose from a git repo).

Mirrors the proven data-tier pattern: repo Cursedpotential/mcp-platform-agno-mcp,
branch main, github_app_id 2, compose location "/".

When to use
-----------
- Splitting a bundled compose app into independent Coolify apps.
- Adding a new service from a git-hosted compose file.
NOT for: standalone databases (use Coolify UI) or importing images.

Parameters
----------
project_uuid : str  # from list_projects
server_uuid : str   # from list_servers
name : str          # human label, also becomes the default fqdn slug
git_repository : str
    Meaning depends on `type`:
      dockercompose / public -> "owner/repo" (or full URL for public)
      dockerfile             -> the Dockerfile path or inline content
      dockerimage            -> the image reference, e.g. "nginx:latest"
git_branch : str
docker_compose_location : str  # path within repo to compose.yaml; "/" = root
github_app_id : int  # Coolify GitHub App source id (2 on this instance)
type : str
    Build type. One of:
      "dockercompose" (default) — private GitHub App + compose file
      "public"                  — public git repo, nixpacks build
      "dockerfile"              — build from a Dockerfile
      "dockerimage"             — deploy a prebuilt image, no build
description : str

Do / Don't
----------
Do: confirm project_uuid + server_uuid via get_infrastructure_overview first.
Do: run check_port_collision after create, before starting, to avoid binding wars.
Don't: pass a placeholder repo — Coolify will accept the create but the deploy will fail.

Returns
-------
dict — the created application record (uuid, name, status). Use the uuid
with upsert_application_envs to set runtime env, then trigger a deploy.
Auto-deploy is switched off right after creation (owner rule: deploys are explicit);
`is_auto_deploy_enabled` in the result is the read-back.

| Parameter | Declared type |
|---|---|
| `project_uuid` | str |
| `server_uuid` | str |
| `name` | str |
| `git_repository` | str |
| `git_branch` | str |
| `docker_compose_location` | str |
| `github_app_id` | int |
| `type` | str |
| `description` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:762](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:762>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `upsert_application_envs`

Set/update environment variables on an application (bulk upsert).

Tries PATCH /applications/{uuid}/envs/bulk first; falls back to per-key
POST /applications/{uuid}/envs if the bulk endpoint is unavailable.

Parameters
----------
application_uuid : str
envs : dict[str, str]   # {KEY: "value", ...} — keys not present are left unchanged
is_runtime : bool       # available at runtime (default True)
is_buildtime : bool     # available at build time (default True)

Do / Don't
----------
Do: group all env for one app into a single call (bulk path is one round-trip).
Do: remember Coolify bakes env VALUES into the rendered compose at deploy —
    changing env requires a redeploy of the app for it to take effect.
Don't: put the token or other secrets here as literal values you then log —
    this tool never logs values, but the agent's transcript might.

Returns
-------
dict — {method: "bulk"|"per-key", updated: int, failed: list}

| Parameter | Declared type |
|---|---|
| `application_uuid` | str |
| `envs` | dict |
| `is_runtime` | bool |
| `is_buildtime` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:883](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:883>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `delete_application`

Delete an application by UUID. DESTRUCTIVE — requires confirm_name to match.

The confirm_name guard prevents accidental deletion when an agent passes a
stale or wrong uuid. The name must match the application's current name.

Do / Don't
----------
Do: pass confirm_name = the app's exact current name (from get_application).
Do: snapshot any needed config/env FIRST — delete is irreversible.
Don't: pass a uuid you haven't just re-read in this session.
Don't: ever pass a wildcard or empty string for either argument.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `confirm_name` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:955](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:955>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `check_port_collision`

Report which applications bind a given host port (read-only, safe).

Primary verification tool before a cutover deploy. Parses each
application's `docker_compose_raw` (the reliable source for compose apps,
since `ports_mappings` is None for compose apps) and `ports_mappings`
(for dockerfile apps) to find host-port bindings.

Parameters
----------
port : int          # the host port to check, e.g. 5432
server_uuid : str, optional  # if given, restrict to apps on that server.
    NOTE: Coolify's /applications list does not expose a clean server_uuid
    field, so server filtering is best-effort via the app's network/destination.
    When in doubt, leave None and inspect the returned app names.

When to use
-----------
- Before deploying a new app that binds a known port — confirm nothing
  already holds it.
- Before a split cutover — confirm the bundled app still holds the port
  (so you know to stop it before starting the new app).

Returns
-------
dict
    {
      port: int,
      collides: bool,
      apps: [{uuid, name, status, ports: [int]}],  # apps that bind this port
      next_actions: list[str],
    }

| Parameter | Declared type |
|---|---|
| `port` | int |
| `server_uuid` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:989](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:989>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `deploy_application`

Trigger a deployment for an application (or database/service) by UUID.

Pre-deploy guard (2026-10-02)
-----------------------------
If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
A guarded call returns {"guard": {...}, "coolify": <Coolify's answer>}; an unguarded call returns
Coolify's answer unchanged.

Uses POST /deploy?uuid=&force= (Coolify 4.3 answers 405 to GET; a JSON body {"uuid","force"} is also
accepted by the API; GET+query is used here since it needs no body).

Verification (owner ruling 2026-09-08)
--------------------------------------
A finished deployment is NOT a working app. After the deployment reaches
`finished`, load the user-facing URL/port (expect 200, or 401 when auth is
on), confirm the container is `healthy`, and for tailscale sidecars hit the
served hostname. Report "deployed, not yet verified" until that is done.
Also: relative binds of repo files in the compose become EMPTY DIRECTORIES
(Coolify keeps no checkout beside the rendered compose) — use absolute host
paths for mounted config files.

When to use
-----------
- After upsert_application_envs, to bake new env values into the running
  container (Coolify renders env into compose at deploy time — env
  changes do NOT reach the container until a redeploy).
- After changing git_branch/docker_compose_location via the Coolify UI.
- Redeploying the current commit (force=False reuses cache where possible;
  force=True does a clean rebuild).

Parameters
----------
uuid : str    # application (or db/service) UUID
force : bool  # force rebuild without cache (default False)

Returns
-------
dict — {"message": "...", "deployment_uuid": "..."}. Pass deployment_uuid
to get_deployment to poll status/logs.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `force` | bool |
| `check_only` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1166](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1166>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `restart_application`

Restart a running application by UUID (stop + start; rebuilds from current image).

Pre-deploy guard (2026-10-02)
-----------------------------
If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
A guarded call returns {"guard": {...}, "coolify": <Coolify's answer>}; an unguarded call returns
Coolify's answer unchanged.

Do / Don't
----------
Do: use this for a quick container bounce (e.g. picking up a restarted
    dependency) when you do NOT need a fresh build.
Don't: use this expecting new env values to apply from a compose baked at
    an earlier deploy — use deploy_application for that.

Returns
-------
dict — {"message": "Restart request queued.", "deployment_uuid": "..."}

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `check_only` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1214](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1214>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `stop_application`

Stop a running application by UUID.

Pre-deploy guard (2026-10-02)
-----------------------------
If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
A guarded call returns {"guard": {...}, "coolify": <Coolify's answer>}; an unguarded call returns
Coolify's answer unchanged.

Parameters
----------
uuid : str
docker_cleanup : bool  # prune networks/volumes after stop (API default True)

Do / Don't
----------
Do: check_port_collision first if you're stopping one app to free a port
    for another (this IS the "stop the old app" step in that workflow).
Don't: docker-stop the container manually outside Coolify — Coolify owns
    the container lifecycle and will fight a manually-stopped container
    on its next reconciliation pass, leaving it in a confused state.

Returns
-------
dict — {"message": "Application stopping request queued."}

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `docker_cleanup` | bool |
| `check_only` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1240](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1240>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `start_application`

Start a stopped application by UUID (deploys and starts containers).

Pre-deploy guard (2026-10-02)
-----------------------------
If guards.json lists this uuid (today: the devbox), its guard runs first over SSH. When it does not
report safe, nothing is sent to Coolify and the tool errors with GUARD_REFUSED plus the guard's
output. check_only=True runs the guard (or reports that none applies) and never calls Coolify.
A guarded call returns {"guard": {...}, "coolify": <Coolify's answer>}; an unguarded call returns
Coolify's answer unchanged.

Parameters
----------
uuid : str
force : bool           # force rebuild
instant_deploy : bool  # skip the deploy queue

Do / Don't
----------
Don't: use `docker start <container>` directly on a Coolify-managed
container — Coolify tracks desired state itself; starting outside the API
creates an orphan container Coolify doesn't know is running (verified
local lesson — leads to port conflicts and duplicate containers on the
next Coolify-triggered deploy). Always start/stop through this tool or
the Coolify UI.

Returns
-------
dict — {"message": "...", "deployment_uuid": "..."} (may be None if the
app was already running).

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `force` | bool |
| `instant_deploy` | bool |
| `check_only` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1273](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1273>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_deployment`

Get one deployment's status and logs by deployment UUID, with errors pre-extracted.

The raw `logs` field from the API is a JSON-encoded STRING of an array of
log-line objects (not a plain string, not pre-parsed JSON) — this tool
parses it and surfaces the tail + any error-looking lines so you don't
have to re-parse it yourself every call.

Parameters
----------
deployment_uuid : str  # from deploy_application / restart_application /
                        # start_application response, or list_deployments_for_app

Requires
--------
The API token needs `read:sensitive` permission to see the `logs` field —
without it, Coolify strips logs from the response (removeSensitiveData()).

Returns
-------
dict
    {
      deployment_uuid, status, application_id, server_id,
      log_line_count: int,
      last_lines: list[str],       # tail of the log (up to 40 lines)
      error_lines: list[str],      # lines containing error/fail/exception (case-insens.)
      raw_logs_available: bool,    # False if token lacks read:sensitive
    }

| Parameter | Declared type |
|---|---|
| `deployment_uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1312](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1312>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_deployments_for_app`

List past/current deployments for one application by app UUID.

GET /deployments/applications/{uuid} — separate from GET /deployments
(which only lists deployments currently queued/in-progress across the
whole team, not scoped to one app).

Parameters
----------
uuid : str   # application UUID (not deployment_uuid)
skip : int   # pagination offset (default 0)
take : int   # page size (default 10)
full : bool  # return whole records, including the build logs

Summary by default: each record carries the entire build log, so ten deployments came back
at 2.5 MB. Use get_deployment(deployment_uuid) for one deployment's logs.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `skip` | int |
| `take` | int |
| `full` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1374](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1374>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `get_application_logs`

Get recent runtime container logs for an application by UUID.

This is CONTAINER STDOUT/STDERR (GET /applications/{uuid}/logs), distinct
from deployment build logs (get_deployment). Use this for "why is my app
crashing at runtime"; use get_deployment for "why did the build fail."

Parameters
----------
uuid : str
lines : int  # number of lines from the end of the log (API default 100)

Requires
--------
Token needs `read:sensitive` permission — logs may contain secrets.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `lines` | int |

Validation: source declaration; invocation not tested.

Source: [server.py:1404](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1404>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `create_database`

Create a standalone database. One tool covers all 8 Coolify engine types.

POST /databases/{db_type} — replaces the old CLI's eight separate
`databases create-*` commands.

Parameters
----------
db_type : str
    One of: postgresql, mysql, mariadb, mongodb, redis, keydb,
    clickhouse, dragonfly.
options : dict, optional
    Engine-specific fields passed through verbatim, e.g.
    {"postgres_user": "admin", "postgres_password": "...",
     "postgres_db": "myapp"} for postgresql, or {"redis_password": "..."}
    for redis. Omitted keys get Coolify's generated defaults.

Do / Don't
----------
Do: confirm project_uuid + server_uuid via get_infrastructure_overview first.
Do: run check_port_collision if you plan to expose the DB on a host port.
Don't: hand-write a password you then log — the transcript keeps it.

Returns
-------
dict — created database record (uuid, name). Start it with start_database.

| Parameter | Declared type |
|---|---|
| `project_uuid` | str |
| `server_uuid` | str |
| `db_type` | str |
| `name` | str |
| `environment_name` | str |
| `description` | str |
| `options` | dict \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1469](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1469>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `start_database`

Start a stopped standalone database by UUID. POST /databases/{uuid}/start.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1524](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1524>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `stop_database`

Stop a running standalone database by UUID. POST /databases/{uuid}/stop.

Do / Don't
----------
Don't: `docker stop` the container directly — same orphan problem as
applications; Coolify owns the lifecycle.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1530](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1530>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `restart_database`

Restart a standalone database by UUID. POST /databases/{uuid}/restart.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1542](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1542>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `delete_database`

Delete a standalone database by UUID. DESTRUCTIVE — requires confirm_name.

Same guard as delete_application: the name is re-read and must match.

Do / Don't
----------
Do: take a backup first (coolify_api POST /databases/{uuid}/backups/{backup_uuid}/execute)
    — deleting a database destroys its volume.
Don't: pass a uuid you haven't re-read this session.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `confirm_name` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1548](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1548>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `create_service`

Create a multi-container service from a raw docker-compose document.

POST /services. Distinct from create_application: a *service* is a compose
stack Coolify manages as one unit; an *application* is git-backed and
rebuilt on push.

Parameters
----------
docker_compose_raw : str
    The compose file content as a YAML string (not a path, not a dict).

Do / Don't
----------
Do: parse-check the compose locally before sending — Coolify accepts the
    create and only fails at deploy time on a malformed document.
Do: check_port_collision for every host port the compose binds.
Don't: bind host port 8080 — coolify-proxy owns it on every node.

| Parameter | Declared type |
|---|---|
| `project_uuid` | str |
| `server_uuid` | str |
| `name` | str |
| `docker_compose_raw` | str |
| `environment_name` | str |
| `description` | str |
| `instant_deploy` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1567](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1567>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `start_service`

Start a stopped service (compose stack) by UUID. POST /services/{uuid}/start.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1607](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1607>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `stop_service`

Stop a running service (compose stack) by UUID. POST /services/{uuid}/stop.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1613](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1613>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `restart_service`

Restart a service (compose stack) by UUID. POST /services/{uuid}/restart.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1619](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1619>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `delete_service`

Delete a service (compose stack) by UUID. DESTRUCTIVE — requires confirm_name.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `confirm_name` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1625](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1625>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_application_envs`

List environment variables on an application (key, value, uuid, flags).

GET /applications/{uuid}/envs. Use this to find an env's `uuid` before
delete_application_env, and to audit what is actually set versus what the
compose expects.

Requires
--------
Token needs `read:sensitive` to see VALUES; without it keys come back with
values stripped.

| Parameter | Declared type |
|---|---|
| `uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1635](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1635>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `delete_application_env`

Delete one environment variable from an application by its env UUID.

DELETE /applications/{uuid}/envs/{env_uuid}.

Do / Don't
----------
Do: get env_uuid from list_application_envs immediately before deleting.
Do: redeploy afterwards — a removed env stays baked into the running
    container until the next deploy renders a fresh compose.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `env_uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1651](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1651>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `create_project`

Create a new project (the container for applications/databases/services).

POST /projects. Returns the record including the uuid you pass as
project_uuid to create_application / create_database / create_service.

| Parameter | Declared type |
|---|---|
| `name` | str |
| `description` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1673](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1673>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `update_project`

Rename a project or change its description. PATCH /projects/{uuid}.

Only the fields you pass are sent; omitted fields are left unchanged.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `name` | str \| None |
| `description` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1683](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1683>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `delete_project`

Delete a project by UUID. DESTRUCTIVE — requires confirm_name.

Do / Don't
----------
Do: list what the project still holds first (list_applications /
    list_databases / list_services) — deleting a project takes its
    resources with it.

| Parameter | Declared type |
|---|---|
| `uuid` | str |
| `confirm_name` | str \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1701](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1701>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `list_deployments`

List deployments currently queued or running across the whole team.

GET /deployments — team-wide and in-flight only. For one app's deployment
HISTORY (including finished ones) use list_deployments_for_app instead.

Validation: source declaration; invocation not tested.

Source: [server.py:1718](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1718>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `cancel_deployment`

Cancel a queued or in-progress deployment by deployment UUID.

POST /deployments/{uuid}/cancel. Endpoint taken from the v4 OpenAPI spec
and NOT verified live against this instance — if it 404s, the deployment
can still be stopped by stop_application on the target app.

Parameters
----------
deployment_uuid : str  # from list_deployments or a deploy_* response

| Parameter | Declared type |
|---|---|
| `deployment_uuid` | str |

Validation: source declaration; invocation not tested.

Source: [server.py:1728](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1728>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `github_app_webhook`

The GitHub App's webhook, as GitHub sees it: target URL, subscribed events, recent deliveries.

Added 2026-10-02 (Claude Code · Opus 5.5) to find why a push does not start a deployment.
Push-to-deploy needs, in order: GitHub sends a `push` delivery to this URL, Coolify answers it,
and the app has auto-deploy on and a watch path matching a changed file. This tool shows the
first two; get_application shows the rest (`settings.is_auto_deploy_enabled`, `watch_paths`).

Read-only. The App's private key and webhook secret are never returned.

Parameters
----------
github_app_id : int        # Coolify's id for the GitHub App source (the apps' `source_id`; 2)
repository : str, optional # "owner/name" or "name": only deliveries for that repository
limit : int                # deliveries to fetch from GitHub before filtering (max 100)
delivery_id : int, optional
    One delivery in full: its push ref and changed files, and what Coolify answered.

Returns
-------
dict  {app, hook, deliveries: [{id, delivered_at, event, status, status_code, repository}],
       delivery?: {event, ref, changed_files, response_status, response_body}}

| Parameter | Declared type |
|---|---|
| `github_app_id` | int |
| `repository` | str \| None |
| `limit` | int |
| `delivery_id` | int \| None |

Validation: source declaration; invocation not tested.

Source: [server.py:1788](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1788>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

### `coolify_api`

Call any Coolify v4 REST endpoint directly. Escape hatch for the long tail.

Covers everything without a dedicated tool: teams, private keys, GitHub
apps, database backups, and server administration. See
references/CLI-PARITY.md for the exact method+path of every such operation.

Parameters
----------
method : str   # GET, POST, PATCH, PUT, DELETE
path : str     # path under the API base, e.g. "/teams" or
               # "/databases/abc-123/backups". Leading slash optional;
               # do NOT include the /api/v1 prefix (it is in the base URL).
body : dict, optional    # JSON request body for POST/PATCH/PUT
params : dict, optional  # query string parameters
confirm_destructive : bool
    Required True for any non-GET/HEAD method. This is a deliberate
    speed bump: the passthrough can delete anything the token can reach
    and has no per-resource name guard like delete_application does.

Examples
--------
List teams:            method="GET",  path="/teams"
Current team:          method="GET",  path="/teams/current"
List private keys:     method="GET",  path="/security/keys"
List GitHub apps:      method="GET",  path="/github-apps"
List db backups:       method="GET",  path="/databases/{uuid}/backups"
Validate a server:     method="GET",  path="/servers/{uuid}/validate"
Server resources:      method="GET",  path="/servers/{uuid}/resources"
Create a private key:  method="POST", path="/security/keys",
                       body={"name": "...", "private_key": "..."},
                       confirm_destructive=True

Do / Don't
----------
Do: prefer a dedicated tool when one exists — they carry guards, retries,
    and response post-processing this raw path does not.
Don't: paste a secret value into `body` and then echo the result — the
    transcript keeps both.

Returns
-------
The decoded JSON response, or None for 204/empty bodies.

| Parameter | Declared type |
|---|---|
| `method` | str |
| `path` | str |
| `body` | dict \| None |
| `params` | dict \| None |
| `confirm_destructive` | bool |

Validation: source declaration; invocation not tested.

Source: [server.py:1874](<E:/AI_Workspace/plugins/plugins/coolify-write/scripts/server.py:1874>) · SHA-256 `c58941b72021a3c3d236fc929cbe23ca1249e3d940dc4bb7ac38897a4392b566`

## Mcp Servers

### `coolify`

http

Validation: configured; health not inferred.

Source: [.mcp.json:1](<E:/AI_Workspace/plugins/plugins/coolify-write/.mcp.json:1>) · SHA-256 `d5253e81c9d468d7e1952575d99a22d1d9b3cd7923f3c7d8c0e835269c238395`


## Existing hosted MCP exposure

The live ContextForge server associated with this plugin exposes the following names. Complete argument schemas and descriptions are in [[Code/wiki/contextforge-tools]]. This is registry discovery, not invocation proof.

- `coolify-write-cancel-deployment`
- `coolify-write-check-port-collision`
- `coolify-write-coolify-api`
- `coolify-write-create-application`
- `coolify-write-create-database`
- `coolify-write-create-project`
- `coolify-write-create-service`
- `coolify-write-delete-application`
- `coolify-write-delete-application-env`
- `coolify-write-delete-database`
- `coolify-write-delete-project`
- `coolify-write-delete-service`
- `coolify-write-deploy-application`
- `coolify-write-get-application`
- `coolify-write-get-application-logs`
- `coolify-write-get-database`
- `coolify-write-get-deployment`
- `coolify-write-get-infrastructure-overview`
- `coolify-write-get-server`
- `coolify-write-get-service`
- `coolify-write-get-service-env`
- `coolify-write-github-app-webhook`
- `coolify-write-list-application-envs`
- `coolify-write-list-applications`
- `coolify-write-list-databases`
- `coolify-write-list-deployments`
- `coolify-write-list-deployments-for-app`
- `coolify-write-list-projects`
- `coolify-write-list-servers`
- `coolify-write-list-services`
- `coolify-write-restart-application`
- `coolify-write-restart-database`
- `coolify-write-restart-service`
- `coolify-write-set-service-image`
- `coolify-write-start-application`
- `coolify-write-start-database`
- `coolify-write-start-service`
- `coolify-write-stop-application`
- `coolify-write-stop-database`
- `coolify-write-stop-service`
- `coolify-write-update-application`
- `coolify-write-update-project`
- `coolify-write-upsert-application-envs`

Back to [[Code/wiki/plugin-inventory|Plugin inventory]].
