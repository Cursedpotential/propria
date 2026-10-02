---
title: The devbox on ovh-files: what it has, when to use it
date: 2026-09-23
status: current
tags: [devbox, ovh-files, claude-code, agent-sdk, jobs, infrastructure, reference]
---

# The devbox on ovh-files

> _Byline: Claude Code · Opus 5.5 · 2026-09-23. Written on owner order 14:34 ("make a note somewhere that that box exists and when and why to use it and what it has"). Facts checked live on 2026-09-23. Re-check versions before relying on them._
> _Byline: Claude Code · Fable 5.1 · 2026-09-26. "How to reach it" rewritten for the new tailnet name `svc:devbox` and the public route `devbox.int`, both verified live 2026-09-27 04:00Z._
> _Byline: Claude Code · Opus 5.5 · 2026-10-02. "Where work lives" replaces "What persists" (the whole home has been the host volume since the 2026-09-28 deploy, and that deploy destroyed work kept outside it); the pre-redeploy guard; RDP moved to 13389; the ttyd terminal; the reserved Claude Code listener._

## What it is

A persistent Linux desktop and shell box (Kasm Ubuntu) that already has Claude Code, Python and Node. It runs on **ovh-files** as the Coolify app `devbox` (uuid `pd3xc78ahqkfswq12bpfqgy1`, project `agno-platform`).

- Container: the one named `devbox-pd3xc78ahqkfswq12bpfqgy1-<deploy stamp>`; find it with `docker ps --filter name=devbox-pd3x`. Image `probata-devbox:latest`, built by Coolify from `modules/Probata/probata/deploy/docker/devbox/Dockerfile` (compose `deploy/devbox.yaml`). Coolify auto-deploy is off on purpose: deploy explicitly, after the guard below.
- User: `kasm-user` (uid 1000).

## What it has (checked 2026-10-02, image built by Coolify deploy `4rstpgrt9hqzcsxyh3zvziw6`)

| Tool | Version / state |
|---|---|
| Claude Code | 2.1.287 at `/usr/bin/claude`; signed in with the owner's own subscription once he runs `/login` |
| Python | 3.12.3, with `uv`; shared venv `/opt/venv` (duckdb, httpx, pyarrow, pypdf, pypdfium2, pytz) |
| Node | 22.23.3 |
| Also present | `git`, `gh`, `opencode`, `codex` 0.160.0, `duckdb`, Docker CLI 29.8.2 (no daemon), `psql` 16.15, `rclone`, `jq`, `ttyd` 1.7.7, Synaptic, Chrome |
| Not present | Tailscale (the box is reached through the host's Tailscale Services) |
| Resources | Capped at 4 CPUs / 8 GB by `deploy/devbox.yaml`; host ovh-files has 8 cores, ~22 GB RAM |

## Where work lives

The whole home `/home/kasm-user` is the host folder `/data/probata/volumes/devbox/home`. It survives redeploys.

- **Agents work as `kasm-user`, in `~/work/<job>`** (host `/data/probata/volumes/devbox/home/work/<job>`). Use `docker exec -u kasm-user …`. A job's venv, logs and outputs go in its own folder there.
- `/root` is the host folder `/data/probata/volumes/devbox/root` (since 2026-10-02), so a `docker exec` without `-u` does not lose what it writes either.
- `/home/linuxbrew` is `/data/probata/volumes/devbox/linuxbrew`; `~/desktop` is the owner's desktop share `/mnt/desktop-share`.
- **Everything else is the container layer, and a redeploy destroys it**: `/tmp`, `/opt`, `/usr`, `/etc`, `/var`. `/tmp` is scratch only. A tool installed at runtime (`apt`, `npm -g`, `uv tool`) must also go into the Dockerfile, or it is gone after the next deploy.
- The 2026-09-28 deploy destroyed `.npm`, `.duckdb`, the old `.claude.json` and two agent work folders (`browser-journeys-*`), because they sat outside the volume of that time. `.claude.json` came back from `~/.claude/backups/` on 2026-10-02; the rest is gone.

**Before every redeploy, run the guard** (on ovh-files, from the desktop; nothing runs locally):

```bash
ssh -i ~/.ssh/ovh root@100.91.190.107 python3 - < modules/Probata/probata/deploy/devbox/pre_redeploy_check.py
```

It lists the running container's writable-layer changes (`docker diff`), sets aside named runtime churn (sockets, Kasm's self-extracting service binaries, bytecode caches), copies everything else into `~/rescued/<UTC stamp>/<container>/files/` with a manifest and a list of runtime-installed tools, and checks every copied file by sha256. Deploy only after it prints `SAFE TO REDEPLOY: yes`. `--dry-run` classifies without copying.

## How to reach it

- Shell: `ssh -i ~/.ssh/ovh root@100.91.190.107`, then `docker exec -it -u kasm-user $(docker ps -q --filter name=devbox-pd3x) bash`.
- Claude Code in the browser (ttyd, since 2026-10-02): `http://100.91.190.107:7681`, tailnet only. It opens `claude` in `~/work` inside a tmux session that survives a closed tab. Claude Code is signed in with the owner's own subscription through Anthropic's login; the credentials stay in `~/.claude` on the host volume.
- Desktop (Kasm, its own login: user `kasm_user`, password `VNC_PW`):
  - Tailnet: `https://devbox.tilapia-skilift.ts.net`, Tailscale Service `svc:devbox` on ovh-files. The short name `https://devbox.mitechconsult.com` redirects there on tailnet devices. Config: `modules/Probata/probata/deploy/tailscale/devbox-serve.hujson`.
  - Public: `https://devbox.int.mitechconsult.com`, behind Authentik first, then Kasm's login. Router `devbox-public` in `modules/Consignatio/docs/receipts/portal/propria-public-portal.yaml`.
  - Direct: `https://100.91.190.107:6901` (self-signed certificate).
- RDP on `100.91.190.107:13389` (moved off 3389 on 2026-10-02, which Kasm Workspaces needs for its own RDP gateway); Syncthing `:8384` / `:22000`. Tailnet IP only.
- Reserved, not built: the legal work desk's "consult Claude" listener will be a second service `devbox-claude` in `deploy/devbox.yaml` (same image, same home volume, tailnet only, Tailscale Service `svc:devbox-claude`). See the comment there.

## When and why to use it

- **Use it for server-side jobs that need Claude Code, the Claude Agent SDK, Python or Node, instead of building a new image.** Owner, 2026-09-23 14:33: reuse the existing boxes; nothing runs on the owner's desktop.
- **Run long jobs detached** with `docker exec -d …`, so they survive the session that started them. Logs go to a file under `~/work/<job>/`.
- **Pass secrets at run time** with `docker exec --env-file /data/probata/secrets/<job>/<file>.env`. Never store them in the devbox home.
- **Give each job its own folder and its own Python environment** (`uv venv ~/work/<job>/.venv`). Do not install into the system Python.

Current use: the Jev Tier-1 eval. Its work dir is `~/jev-eval` (host `/data/probata/volumes/devbox/home/jev-eval`), from before the `~/work` convention; its code lives in `modules/Probata/probata/scripts/jev_eval/`.

## The other boxes (not for these jobs)

| Box | Where | Has | Use |
|---|---|---|---|
| `opencode-server` | ovh-files, `https://opencode.tilapia-skilift.ts.net` | OpenCode headless server; Python 3.14.7 and Node 26 through mise; **no Claude Code** | Delegated agents on OpenCode (NIM and other models) |
| `desktop-…` (Kasm) | ovh-app, `svc:desk` | Python 3.8 only | A remote desktop; not a job runner |
