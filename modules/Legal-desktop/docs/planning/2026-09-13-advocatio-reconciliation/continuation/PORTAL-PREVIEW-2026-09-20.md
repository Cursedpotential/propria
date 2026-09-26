# Advocatio portal preview

Owner requested the preview in the portal and visible progress on 2026-09-20.

Live preview: https://legal.tilapia-skilift.ts.net/
Progress board: https://homepage.tilapia-skilift.ts.net/progress/

The existing Coolify application `legal-workspace`
(`gvghzivfmctev8dloetfssnj`) already serves the frontend and healthy API on ovh-app.
The named HTTPS preview returned 200. No app rebuild or new deployment was needed
for this portal correction. Task A changed planning artifacts, not deployed UI.

## Corrections

- `/data/dashboards/homepage/services.yaml`: Advocatio href and monitor changed
  from the old raw host address to the working named HTTPS service.
- `/data/dashboards/homepage-public/services.yaml`: Advocatio href and monitor
  changed from `legal.int.mitechconsult.com`, which returned an Authentik 404,
  to the working named HTTPS service. Existing tailnet access is required.
- Both tile descriptions now show the 383/383 toolkit map completion and next
  workflow integration step. Only the Advocatio tile was edited.
- `/data/dashboards/progress-board/surfaces.json`: matching surface receives the
  working URL, compact progress status and actual check timestamp.
- Docstore `portal_observation:advocatio_task_a_20260920` records Task A as done,
  links commit `743a183`, and identifies B/F as follow-up work.
- Existing `portal_observation:portal_surface_c6a9f65c553738e4b061` records the
  successful preview check separately from application feature completion.

All three remote configuration files have adjacent timestamped `.bak-...-advocatio-preview`
copies. No container or host restart, source-data modification, service-port
allocation or new server registration occurred. The existing named-service route
was retained; legacy backend binds were not changed.

## Verification

Preview HTTPS and portal progress page: HTTP 200. Homepage `/api/services`
returns the updated HTTPS target and `383 of 383` description. Progress
`/api/board` returns the completed Task A observation with its commit link.
These are live HTTP/configuration/feed checks; no browser interaction or
end-to-end legal workflow test is claimed.

Remote configuration is the deployed source for this portal; the old local
dashboards mirror differs and was not copied over it. This receipt preserves
the scoped change in Advocatio's repository. The completed Task A checklist is
available under this directory; visible product feature work remains next.
