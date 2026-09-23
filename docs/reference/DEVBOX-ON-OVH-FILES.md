---
title: The devbox on ovh-files: what it has, when to use it
date: 2026-09-23
status: current
tags: [devbox, ovh-files, claude-code, agent-sdk, jobs, infrastructure, reference]
---

# The devbox on ovh-files

> _Byline: Claude Code · Opus 5.5 · 2026-09-23. Written on owner order 14:34 ("make a note somewhere that that box exists and when and why to use it and what it has"). Facts checked live on 2026-09-23. Re-check versions before relying on them._

## What it is

A persistent Linux desktop and shell box (Kasm Ubuntu) that already has Claude Code, Python and Node. It runs on **ovh-files** as the Coolify app `devbox` (uuid `pd3xc78ahqkfswq12bpfqgy1`, project `agno-platform`).

- Container: `devbox-pd3xc78ahqkfswq12bpfqgy1-150427235321`. Image `probata-devbox:latest`, built 2026-09-08; the v2 rebuild has never built successfully (see `modules/Probata/probata/docs/handoffs/HANDOFF-2026-09-10-cloud-services-tailscale-devbox.md`).
- User: `kasm-user` (uid 1000).

## What it has (checked 2026-09-23)

| Tool | Version / state |
|---|---|
| Claude Code | 2.1.263 at `/usr/bin/claude` |
| Python | 3.12.3, with `pip3` and `uv` |
| Node | 22.23.2 |
| Also present | `git`, `gh`, `opencode`, `duckdb`, `rclone`, `jq` |
| Not present | Docker CLI, `psql`, Codex, Tailscale |
| Resources | 8 CPUs, ~22 GB RAM (host shared). Root disk was 89% used (22 GB free) on 2026-09-23 |

## What persists

- `/home/kasm-user/persist` is the host folder `/data/probata/volumes/devbox/home`. It survives redeploys. Put work here.
- `/home/kasm-user/desktop` is the host folder `/mnt/desktop-share`.
- Everything else inside the container is lost on the next redeploy.

## How to reach it

- Shell: `ssh -i ~/.ssh/ovh root@100.91.190.107`, then `docker exec -it -u kasm-user devbox-pd3xc78ahqkfswq12bpfqgy1-150427235321 bash`.
- Desktop: Kasm on `https://100.91.190.107:6901` (Kasm login); RDP on `100.91.190.107:3389`; Syncthing `:8384` / `:22000`. Tailnet IP only. No `svc:` HTTPS name yet (`svc:kasm` is pending in the handoff above).

## When and why to use it

- **Use it for server-side jobs that need Claude Code, the Claude Agent SDK, Python or Node, instead of building a new image.** Owner, 2026-09-23 14:33: reuse the existing boxes; nothing runs on the owner's desktop.
- **Run long jobs detached** with `docker exec -d …`, so they survive the session that started them. Logs go to a file under `persist/`.
- **Pass secrets at run time** with `docker exec --env-file /data/probata/secrets/<job>/<file>.env`. Never store them in the devbox home.
- **Give each job its own folder and its own Python environment** (`python3 -m venv persist/<job>/.venv`). Do not install into the system Python.

Current use: the Jev Tier-1 eval. Its work dir is `persist/jev-eval` (host `/data/probata/volumes/devbox/home/jev-eval`); its code lives in `modules/Probata/probata/scripts/jev_eval/`.

## The other boxes (not for these jobs)

| Box | Where | Has | Use |
|---|---|---|---|
| `opencode-server` | ovh-files, `https://opencode.tilapia-skilift.ts.net` | OpenCode headless server; Python 3.14.7 and Node 26 through mise; **no Claude Code** | Delegated agents on OpenCode (NIM and other models) |
| `desktop-…` (Kasm) | ovh-app, `svc:desk` | Python 3.8 only | A remote desktop; not a job runner |
