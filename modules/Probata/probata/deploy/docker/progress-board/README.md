# Propria progress board — source

> _Byline: Claude Code · Opus 5.5 · 2026-09-26._

The Node service behind `/progress` on both portal instances and behind the portal's Live board
widgets. Compose: `deploy/progress-board.yaml`. Dependency: `pg` only (`package-lock.json`).

Brought into git on 2026-09-26 from the running host copy, `ovh-app:/data/dashboards/progress-board`,
where it had only ever existed (no repository, edited in place). Every file below was copied
byte-for-byte; the first 16 hex characters of each SHA-256 match the host copy at copy time.
Left on the host on purpose: `node_modules/` (rebuilt by `npm ci`), `intake-build/` and `data/`
(runtime data, mounted by the compose file), `_scratch/` and the `*.bak*` backups.

| SHA-256 (16) | File |
|---|---|
| `bed9599b86b9b06f` | `diagnose.mjs` |
| `a40228df06f230e4` | `health-endpoints.json` |
| `95ee77ddfdd54be1` | `health.mjs` |
| `130fa92e62de7960` | `intake-preview/app.js` |
| `547214e41149dcd7` | `intake-preview/index.html` |
| `d38ffc04413352e3` | `intake-preview/pr-intake-adapter.css` |
| `3d2488a12e0dd25b` | `intake-preview/pr-theme.css` |
| `4f24523240b4f9b9` | `intake-preview/pr-tokens.css` |
| `dcb700e80b9db108` | `intake-preview/style.css` |
| `e162bd1f03b26f8b` | `intake-preview/view.mjs` |
| `03f17a9f87f42324` | `openlist-bridge.mjs` |
| `d04703dfdf1e187c` | `outputs/live-service-storage-health.html` |
| `7846aafa80a0556a` | `package-lock.json` |
| `e28e235ac3cc640e` | `package.json` |
| `9ec9d40ff3c9c380` | `pg-catalog.mjs` |
| `d8292606cdeee1d2` | `provider-limits.mjs` |
| `28693a3fb04473d5` | `public/app.js` |
| `0caa1bc39ab38102` | `public/index.html` |
| `923b2d0565b27bb4` | `public/provider-settings.js` |
| `217a479364277bd2` | `public/style.css` |
| `d86c035bb724db67` | `public/task-actions.mjs` |
| `56fb1229fe77c8cc` | `public/vendor/apexcharts.min.js` (ApexCharts 3.54.1, MIT) |
| `f3b646fb58a877af` | `public/view.mjs` |
| `f3af98278b5af942` | `repair-queue.mjs` |
| `3a7d65438900988c` | `server.mjs` |
| `46f81e32df931bdb` | `surfaces.json` |

Before the running service is switched to an image built from here, compare the host copy with
this table again: the portal editor (code-server on `/data/dashboards`) can still change the host
copy in place.

The family-court preview (`family-court-preview/`) was retired on 2026-09-28: `/family-court` now
redirects to the hosted Family Law Toolkit (Claude Code · Opus 5.5).
