# Propria portal (Homepage)

> _Byline: Claude Code · Opus 5.5 · 2026-09-26._

The owner's start page, in two instances built from this folder:

| Instance | Address | Reached through | Tiles |
|---|---|---|---|
| tailnet | https://homepage.tilapia-skilift.ts.net | tailscale serve `svc:homepage` → `100.72.169.40:3010` | everything, `*.tilapia-skilift.ts.net` links, no login |
| public | https://homepage.int.mitechconsult.com | Traefik file route `homepage-public` → `100.72.169.40:3012`, behind the one Authentik login | user surfaces only, `*.int.mitechconsult.com` links |

Both run as one Coolify application, `propria-portal` (compose `../portal.yaml`), on ovh-app. The
configuration is baked into the image; nothing is edited on the host.

## Files

| Path | What it is |
|---|---|
| `Dockerfile` | Homepage v1.13.2, pinned by digest; `INSTANCE=tailnet` or `public` picks the services file |
| `shared/` | identical in both instances: `settings.yaml` (layout, title from `HOMEPAGE_VAR_PORTAL_TITLE`), `widgets.yaml`, `bookmarks.yaml`, `custom.css`, `custom.js`, `docker.yaml` (no Docker integration) |
| `tailnet/services.yaml`, `public/services.yaml` | the tiles of each instance |
| `coolify_app.py` | the Coolify record: create, sync watch paths, status, deploy |
| `shoot.sh`, `shoot.mjs` | headless-Chrome screenshots, run inside the Probata devbox on ovh-files |

## Layout

Owner, 2026-09-26: the left third holds every widget, stacked; the right two thirds hold the app
buttons. Groups named under `layout` in `shared/settings.yaml` render in Homepage's
`#layout-groups` (the buttons, in that order). The `Live board` group and the `Reference`
bookmarks are left out of `layout`, so Homepage renders them in `#services` and `#bookmarks`;
`custom.css` places those under the header widgets in the left column. At 1200 px and wider the
page is a two-column grid; when the window is at least 960 px tall the button column is sticky,
so it stays in view while the longer widget column scrolls. Narrower, everything stacks, buttons
first. Every button section uses four columns, so every tile has the same width.

Rules the files keep: widgets are native Homepage widgets (`customapi` plus the two ApexCharts cards
in `custom.js`), never an iframe; widget `url:` values and site monitors are fetched by the
container over the host's tailnet address and are not links; the public file lists no
infrastructure or admin console and no link to a `ts.net` name.

## Changing the portal

1. Edit the files here in a worktree.
2. Preview before committing: `./shoot.sh preview <out-dir> <label>` renders this checkout's config
   with a throwaway Homepage process inside the devbox and writes PNGs plus layout numbers.
3. Commit, push to `main`, then `python coolify_app.py deploy`.
4. Check the live result: `./shoot.sh live <out-dir> <label>`.

A fresh container re-renders its index page from the healthcheck (`/api/revalidate`, see
`../portal.yaml`); without that, a browser that never visited would get Homepage's build-time page.

## Not here

- The progress board behind `/progress` (widgets' data, the lane, health and intake pages;
  `/progress/family-court/` redirects to the hosted Family Law Toolkit) is a separate Coolify service, `propria-progress-board`, whose source is still only on
  ovh-app in `/data/dashboards/progress-board/`.
- The portal editor (code-server, `portal-edit`) edits the retired host copy in `/data/dashboards`,
  which no longer feeds the portal. Keep, repoint or retire it is an open owner decision.
