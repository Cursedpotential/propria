# Kasm Workspaces proof — 2026-10-02 (P-1 step 6)

> _Byline: Claude Code · Opus 5.5 · 2026-10-02._

Headless Chrome ran inside the Probata devbox on ovh-files. Nothing ran on the owner's desktop. The script is `deploy/kasm/proof/kasm_proof.sh` with `kasm_proof.mjs`. The sessions it opened were ended afterwards (`end_sessions.py`).

| File | What it shows |
|---|---|
| `01-login.png` | `https://kasm.tilapia-skilift.ts.net/#/login`, Kasm's sign-in page on the tailnet name (svc:kasm). |
| `02-workspaces.png` | Signed in as `msalem`, an admin, so Kasm opens the admin dashboard. Its image list holds Devbox, Devbox (RDP) and Sandbox. |
| `03-devbox-session.png` | A launched Devbox workspace: the XFCE desktop streaming in the browser. |
| `04-rdp-session.png` | A launched Devbox (RDP) workspace: the same desktop over RDP through Kasm's Guacamole proxy. |
| `05-public-signed-out.png` | `https://kasm.int.mitechconsult.com` signed out: the Authentik login, the expected result. |
| `06-devbox-session-synaptic.png` | The desktop of a Kasm Devbox session with Synaptic open, captured by ImageMagick `import`. The same session reported `synaptic`, `claude` and `ttyd` on its PATH. |
| `proof-run2.json`, `proof-run5.json` | Per-step URL, title and what the signed-in page listed. |

**Persistence (`deploy/kasm/session_proof.py Devbox`):**
- A marker file written in session 1 had the same sha256 in session 2, which ran in a new container. Result: PERSISTED.
- The marker was purged afterwards.

**Sandbox:** one session launched (`kasmweb/core-ubuntu-noble:1.17.0`, no mounts) and was ended.

**Devbox (RDP), proven on the clean image** (devbox deploy `dni65rfrjh28z9vcocnsefwo`, commit `14a0493c`, with no manual changes in the container):
- `04-rdp-session.png` is the devbox's XFCE desktop over RDP through Kasm's Guacamole proxy.
- Three fixes made it work:
  - `pam_systemd` is commented out, because it waited for a logind the container does not have;
  - the image ships its own `/etc/xrdp/startwm.sh`, with an ICE authority file private to the RDP session;
  - `custom_startup.sh` starts a system D-Bus.
- The `03-devbox-session.png` and `proof-final.json` files are from the same final run.

**Shared home:** a Kasm Devbox session has `DEVBOX_KASM_SESSION=1` and runs neither Syncthing nor the memsearch loop. The Coolify devbox container runs both (2 Syncthing processes, 1 memsearch loop), checked live.
