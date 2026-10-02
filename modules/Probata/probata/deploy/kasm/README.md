# Kasm Workspaces CE on ovh-files

> _Byline: Claude Code · Opus 5.5 · 2026-10-02. Portal item P-1; brief: `docs/planning/2026-09-27-TODO.md` "Brief for #1"._

Kasm Workspaces Community Edition hosts the owner's **Devbox** desktop, a throwaway **Sandbox** desktop for
agents, and a **Guacamole** (RDP) connection to the devbox, all in the browser.

## What is where

| Thing | Where | Managed by |
|---|---|---|
| Kasm services (`kasm_proxy`, `kasm_api`, `kasm_manager`, `kasm_agent`, `kasm_db`, `kasm_guac`, `kasm_rdp_gateway`, …) | `/opt/kasm` on ovh-files | **Kasm's installer**, run once by `install_kasm.sh`. This is the one recorded exception to "every container is declared in Coolify". |
| Admin and user credentials | `/data/probata/secrets/kasm/kasm.env` (0600) and the desktop's `~/.secrets/kasm.env` | `install_kasm.sh` generates them; never printed |
| The `probata-devbox` image the Devbox workspace runs | built by Coolify app `devbox` (`pd3xc78ahqkfswq12bpfqgy1`) from `deploy/docker/devbox/Dockerfile` | Coolify (deploy explicitly; run `deploy/devbox/pre_redeploy_check.py` first) |
| Devbox home | `/data/probata/volumes/devbox/home` = Kasm persistent profile path | host volume |
| Workspace definitions | `workspaces/devbox.json`, `workspaces/sandbox.json` | tracked here; entered in Kasm's admin UI |

## Ports and exposure

- Kasm web: `8443` (Coolify's Traefik owns 443). Kasm's RDP gateway: `3389`, which is why the devbox's own xrdp moved to `13389` on 2026-10-02.
- Docker publishes both on all interfaces. The host's `DOCKER-USER` chain drops everything that arrives on the public interface `ens3`, so they are reachable only on the tailnet and private networks.
- Tailnet name (step 4, not done yet): Tailscale Service `svc:kasm` → `https://kasm.tilapia-skilift.ts.net`, `tailscale serve --service=svc:kasm --https=443 https+insecure://100.91.190.107:8443`. The name `kasm` was held by the devbox's Tailscale sidecar until 2026-10-02; the stale device must be removed first.

## Install (once)

```bash
ssh -i ~/.ssh/ovh root@100.91.190.107 'bash -s' < modules/Probata/probata/deploy/kasm/install_kasm.sh
```

It stops if `/opt/kasm` exists, if 3389 or 8443 is taken, or if the host has no swap. It pins the 1.19.0 bundle by sha256.

## Register the workspaces

Kasm admin → Workspaces → Add Workspace, one per JSON file. Copy `docker_run_config_override` into **Docker Run Config Override**, `volume_mappings` into **Volume Mappings**, and set Cores, Memory and Persistent Profile Path from the same file. Keys starting with `_` are notes.

- **Devbox** starts at 2 CPU / 4 GB (owner 2026-10-02). It shares its home with the Coolify devbox container until that container's desktop is retired (brief step 5).
- **Guacamole**: a Server workspace, RDP to `100.91.190.107:13389` (the devbox's xrdp). Credentials are the devbox's own; not stored here.

## The Claude Code listener (reserved, not built)

The legal work desk's "consult/escalate to Claude" backend is **not** a Kasm workspace. Kasm sessions start
and stop on demand; the listener must always be up. It is reserved as a second service `devbox-claude` in
`deploy/devbox.yaml` (same image, same home volume, tailnet-only, Tailscale Service `svc:devbox-claude`),
running the unmodified `claude -p` with the owner's own signed-in subscription. See the comment there.
