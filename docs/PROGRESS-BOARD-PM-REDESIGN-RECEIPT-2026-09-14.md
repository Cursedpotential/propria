> Updated 2026-09-24: palette references now reflect the owner-directed Probata color alignment; earlier verification dates below do not verify this update.

# Receipt: Propria progress board redesigned as a project-management app

> _Byline: Claude Code · Sonnet · 2026-09-14 — owner direction 08:07 EDT ("more project-management
> app style, with widgets and tracking boards"; design rule 08:22 "UIs conform to our design
> system, not the other way around"); supervisor correction 08:1x EDT (trim disclaimers to one
> small freshness flag, no banners)._

- Status: **Active** — deployed and live-verified in a real browser on 2026-09-14.
- Scope: `progress-board/public/*` only, on ovh-app (Coolify service `homv6zeg4ay2r2puxtzakf83`).
  `server.mjs` and its API contract were read, not edited.

## What changed

Live route: `https://homepage.tilapia-skilift.ts.net/progress/`. Deployed files (all four paths
`server.mjs` serves from `public/`):

- `public/index.html` — KPI strip, a Board/Trackers tab pair, kanban + lane-swimlane toggle,
  lane/owner/status filters and search.
- `public/app.js` — wires the KPI strip, tabs, filters and the new tracker table to the existing
  `/api/board` and `/api/health-report` polling.
- `public/view.mjs` — extends the prior Codex helpers with `kpiCounts`, `laneTotalsWidget`,
  `healthSummary`, `renderBoardMarkup` (flat or per-lane swimlanes), `trackerRows` /
  `filterTrackerRows` / `trackerTableRows`, priority/due badges on task cards, and a
  `statusChipClass` heuristic for the tracker's status chips.
- `public/style.css` — rebuilt on the Propria Probata graphite/indigo token contract. `server.mjs` only
  serves the four fixed filenames above (no route for a fifth `tokens.css`), so the generated
  token block (`resources/design/tokens.json`, sha256
  `4d1ed7b7a62cb04289c48570fd49a28f7493855413416ca20a088134699473e`, contract 1.0.0) is vendored
  verbatim at the top of `style.css` with a provenance comment instead of a separate `<link>`.

Backup of the pre-redesign `public/` (Codex, 2026-09-12): `progress-board/public.bak-20260914-pm-redesign/`
on ovh-app, alongside the original repo directory (a copy, nothing deleted).

### Disclaimer trim (supervisor correction, same session)

The first deployed pass carried the prior build's multi-sentence widget notes and a caveat-laden
footer. Per the owner's no-disclaimer-banners rule, every such block was cut to a single small
freshness/status flag: the connection banner reads `Connected · read <age>` instead of a full
sentence, the KPI strip reads `Read <age>`, each widget shows only a timestamp instead of an
explanatory paragraph, and the footer is one line (`Times use America/New_York.`). Removed
entirely: the "Done" column's completion-caveat paragraph, the "not assumed complete" qualifier on
the other-states disclosure, and the two-sentence storage/health explanations. Source `detail` text
recorded in Surreal by other workers is untouched — only this page's own template chrome changed.

## Live verification

Checked in the Browser pane against the real deployed page and its real `/api/board` +
`/api/health-report` responses (not a proxy signal):

- KPI strip counts matched the API-derived task set exactly: 23 backlog/ready, 6 in progress,
  2 blocked, 5 done, 0 overdue, 0 due within 7 days, summing to the 36-task snapshot the toolbar
  also reported. Health summary `26/44 services responding` matched
  `reachable(23 responding + 3 TCP open) / 44 total` from `/api/health-report`.
- Board tab: lane/owner/status filters and search populate and filter correctly; the "Group by
  lane" toggle switches to per-lane swimlane kanbans and back.
- Trackers & widgets tab: the tracker table rendered 30 of 30 tracked items (24 surfaces + 6
  non-task observations) with status chips, and the six widgets (task distribution, work by lane,
  endpoint reachability, storage, workers, catalog, migration) all rendered from live data.
- Console and network: no console errors and no CSP violations across both tabs, before and after
  the disclaimer-trim redeploy; every asset (`/`, `/style.css`, `/app.js`, `/view.mjs`) and both
  API calls returned HTTP 200.
- Found and fixed one real layout bug during verification: on a narrow viewport the `.task-top`
  flex row squeezed the status chip until its text wrapped one letter per line. Fixed with
  `white-space:nowrap`/`flex-shrink:0` on `.chip` and `flex-wrap:wrap` on `.task-top`; re-verified
  at a mobile-width viewport with the chip readable on its own line.

## Known rough edges / next steps

- The `todo` table is currently empty (Surreal), so priority/due badges are implemented and wired
  but have not yet rendered against a real non-null value; they will as soon as `todo` rows exist.
- `portal_observation` staleness (newest rows 2026-09-13 02:48 UTC) is unchanged by this work; the
  owner-known fix is restarting the observation refresh automation, which this task did not touch.
- Backup cleanup: `public.bak-20260914-pm-redesign/` can be removed once the redesign is confirmed
  stable for a few days.
