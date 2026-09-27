---
tags: [probata, librechat, receipt, screenshot, proof]
---

# LibreChat signed-out pages on the tailnet name (2026-09-27 04:39 UTC)

> _Byline: Claude Code · Fable 5.1 · 2026-09-26 (EDT). Log entry: `docs/planning/2026-09-20-TODO.md` (LibreChat recreated as Coolify apps)._

Taken by the tracked script `deploy/docker/librechat/shoot_login.sh` (git blob as of `e6694df5`), fed to the Probata devbox on ovh-files and run there as `kasm-user` with its headless Chrome, never on the owner's desktop:

`ssh ovh-files 'docker exec -i -u kasm-user <devbox> bash -s -- /var/tmp/lc-shots-20260927 https://librechat.tilapia-skilift.ts.net' < deploy/docker/librechat/shoot_login.sh`

| File | Page | Read from the rendered DOM | sha256 |
|---|---|---|---|
| `librechat-login.png` (1400×900) | `/login` | `<title>LibreChat</title>`; `Welcome back`, `Email address`, `Continue` ×2, `Sign up` | `e110771f5e95ed6ea01be5c2f53df26ed9c144165985062276c56724131c679c` |
| `librechat-register.png` (1400×900) | `/register` | `<title>LibreChat</title>`; `Create your account`, `Continue` | `2f7ee8f07209e693090694eafdbc01eeab3fdcf0c127d34554abae28abb86514` |

No signed-in screenshot exists: the agent does not create accounts or sign in with passwords on anything but a local development host, so the owner's first sign-up is the first signed-in session.
