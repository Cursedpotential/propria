# Metabase deployment receipt — 2026-09-14

> _Byline: Claude Code · Sonnet 5 · 2026-09-14_

## Decision

Metabase is the owner's single data site for the Propria portal (owner
2026-09-14: "Metabase only" — replaces NocoDB and CloudBeaver). Deployed as
its own Coolify "Docker Compose" app on ovh-app, with the DuckDB driver
(motherduckdb 1.5.5.0) for future Parquet querying.

Product code **74**, portal family, canonical backend port **9074**,
service `svc:metabase`, registered in `deploy/service-port-registry.json`.

## Why not the stock `metabase/metabase` image

Verified live 2026-09-14: that image is Alpine 3.24.1 / musl / OpenJDK 25.
The DuckDB driver is a glibc-only native binding and will not load on musl.
Deployed instead on `eclipse-temurin:21-jre-jammy` running the plain
Metabase jar (v0.63.17) with the driver jar mounted via `MB_PLUGINS_DIR`
from a host bind mount — no custom registry build (owner: "raw and
unfinished is fine").

## What was done

- Host prep on ovh-app: `/data/probata/volumes/metabase/{app,plugins,pgdata,data}`,
  pinned jars staged and sha256-verified.
- `deploy/metabase.yaml` — Metabase (glibc JRE + DuckDB plugin) plus its own
  bind-mounted Postgres app DB (`metabase-db`), Coolify magic env for the DB
  password, health check on `/api/health`, Traefik labels pre-wired but
  `traefik.enable=false` (route not opened).
- `deploy/service-port-registry.json` — added the `metabase` product entry
  (code 74, port 9074, `svc:metabase`).
- `deploy/host/metabase-pg-readonly-setup.py` / `-verify.py` — provisioned a
  least-privilege `metabase_ro` Postgres role (CONNECT + USAGE on
  `raw_duck` + SELECT on all its tables, default privileges for future
  tables) on the `casebible` catalog container on ovh-files, for Metabase's
  read-only catalog-browsing connection. Password written to
  `/data/probata/secrets/metabase/pg-readonly` on ovh-app (mode 600), never
  printed or recorded elsewhere.
- `deploy/tailscale/metabase-serve.hujson` — host-local Tailscale Service
  config for `svc:metabase`, applied on ovh-app via `tailscale serve
  set-config` + `tailscale serve advertise`.
- Coolify compose app `metabase` created (uuid `zb0hi2bi26vnndyw9eozn737`),
  project `agno-platform`, environment `production`, server `ovh-app`,
  watch path `deploy/metabase.yaml`. Deployed successfully from commit
  `5d256d7` (later commit `1086249` added the PG provisioning scripts).

## Verified live

- `docker ps` on ovh-app: both `metabase` and `metabase-db` containers
  running; `metabase` container health = healthy.
- `curl http://100.72.169.40:9074/api/health` → `HTTP 200 {"status":"ok"}`.
- `curl http://100.72.169.40:9074/` and `/setup` → both `HTTP 200` (Metabase
  setup page is live).
- Metabase container logs: `Registered driver :duckdb (parents:
  [:sql-jdbc])`, loaded from `/plugins` — the DuckDB driver plugin loaded
  successfully.
- `information_schema.role_table_grants` on `casebible`: `metabase_ro` has
  `SELECT` on all 56 tables in `raw_duck`, nothing else.

## NOT done (deliberately, or blocked)

- **No Metabase admin account created** — owner does this on first visit
  (owner instruction: don't create it).
- **No database connections added inside Metabase** (the PG read-only
  connection, the DuckDB test) — Metabase requires the admin setup wizard
  to complete before any database can be added, and creating that account
  was explicitly out of scope.
- ~~**`https://metabase.tilapia-skilift.ts.net` does not resolve yet.**~~
  **Corrected 2026-09-14 10:30 EDT: done; see "Tailscale Service (verified live)" below.** The stale text is kept struck through. The 401 came from revoked key copies in `probata.env`/`Agno-MCP-Platform.env`; the key in `tailscale.env` works.
  `tailscale serve set-config` + `tailscale serve advertise svc:metabase`
  were applied on ovh-app and show correctly in `tailscale serve status`
  on that host, but `svc:metabase` does not yet appear in `tailscale
  service list` run from another tailnet member (ovh-files) — unlike the
  existing `svc:workbench`, which does resolve there. This indicates the
  tailnet's ACL/policy file needs the new service registered (a tailnet-
  wide policy change) before the VIP + MagicDNS record goes live tailnet-
  wide. The `TAILSCALE_API_KEY` in `~/.secrets/tailscale.env` returned
  `401 API token invalid` against `api.tailscale.com`, so this could not be
  completed via API either. **Needs the owner** (admin console, or a valid
  API token) to register `svc:metabase` in the tailnet policy. Until then,
  Metabase is reachable tailnet-wide only via the direct backend
  `http://100.72.169.40:9074`, not the friendly hostname.

  **Exact unblock procedure** (per `docs/handoffs/HANDOFF-2026-09-10-cloud-services-tailscale-devbox.md`,
  line 30 — "Tailscale Services need three steps, not one"): a Service must be
  (1) **registered** — `PUT /api/v2/tailnet/-/vip-services/svc:metabase`,
  (2) **advertised from the host** — `tailscale serve --service=svc:metabase
  --https=443 http://100.72.169.40:9074` (done here via the equivalent
  `set-config` + `advertise`), then (3) **approved** —
  `POST /api/v2/tailnet/-/services/svc:metabase/device/<deviceId>/approved
  {"approved":true}`. Step (1) was never done for `svc:metabase` (it does
  exist for `svc:workbench`, hence that one resolves). Attempted step 1 via
  API with `TAILSCALE_API_KEY` from `~/.secrets/tailscale.env`
  (`tskey-api-...`, 63 chars) against both `tailnet=-` and
  `tailnet=tilapia-skilift.ts.net` — both returned `401 {"message":"API
  token invalid"}`, so the key on file is expired/revoked. Needs either a
  fresh API token (owner generates one in the admin console) or the owner
  doing steps 1 and 3 directly in the admin console's Services page.

## Tailscale Service (verified live, 2026-09-14 10:30 EDT)

> _Byline: Claude Code · Opus 5 · 2026-09-14. Folded in from the hand-registered Docstore rows (the docs now go through the pipeline only)._

- **Register:** `PUT /api/v2/tailnet/-/vip-services/svc:metabase` returned HTTP 200. The body needs an explicit `"name":"svc:metabase"` (the shorthand body returns 400). Assigned addresses are `100.103.90.212` and `fd7a:115c:a1e0::8629:5ad5`.
- **Advertise:** ovh-app had a stray `http:443` handler, cleared first with `tailscale serve --http=443 off`. Then `tailscale serve --service=svc:metabase --https=443 http://100.72.169.40:9074` was applied.
- **Approve:** ovh-app device `2981889328716894`, `POST .../approved {"approved":true}`, returned 200 with `{"approved":true,"autoApproved":true}`. The existing `autoApprovers.services` wildcard (`autogroup:tagged` for `tag:docker`) already covers it, so the ACL was not changed.
- **Verify:** `curl -L https://metabase.tilapia-skilift.ts.net/api/health` returned 200 `{"status":"ok"}`, and `/` returned 200. The supervisor re-checked both.
- **Credential:** `TAILSCALE_API_KEY` in `~/.secrets/tailscale.env` works (tailnet `-`).
- **Repo file:** `deploy/tailscale/metabase-serve.hujson` was rewritten to the `services` nesting used by `workbench-serve.hujson`. It is still uncommitted.
- **Scope:** MagicDNS names are tailnet-only by design. Public access would be a separate Cloudflare DNS-only A record (`metabase.int.mitechconsult.com` → `40.160.5.19`) plus a Traefik route behind Authentik. That is not done; it waits for the owner's go-ahead.
