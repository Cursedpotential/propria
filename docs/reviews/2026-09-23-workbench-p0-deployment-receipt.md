# Workbench P0 deployment and runtime correction — 2026-09-23

> Byline: Codex · GPT-6 · 2026-09-23. Live checks and scoped Coolify deployment. This record distinguishes completed checks from pending acceptance.

## Checkpoint before the Serve-peer correction

- Repository `main` was a strict fast-forward from `origin/main` at `e4966d0` to `200d03cc0054d7eb41b411caba596918f40696f5`; the 13 reviewed commits were pushed normally. The separate `modules/forks/sbv` gitlink status and two untracked JEV scripts were preserved.
- Coolify Workbench application: `xjbuo6drbwjfby75lalk8bk7`, branch `main`, compose `/deploy/workbench.yaml`. Prior deployment: `kia7conxu13321a245nldih9` at `9158ffc6580b52bc972efd555198ba3fea112bca`. New deployment `gbxrbkfvuec6bps9vipqdbje` finished at `200d03cc0054d7eb41b411caba596918f40696f5`.
- The running edge network has `coolify-proxy=10.201.0.2`, Authentik `10.201.0.3`, Workbench `10.201.0.4`. Public `TRAEFIK_PROXY_CIDR=10.201.0.2/32` is exact and unchanged. Unauthenticated public Workbench redirected to Authentik (HTTP 302).
- The new Workbench container is attached to `propria-edge` and its application network; its old `probata` network attachment did not survive the Coolify deployment. A Tailnet `/sources` request returned HTTP 403 `Untrusted proxy`. A synchronized read-only packet capture of that request observed `10.201.0.1:* -> 10.201.0.4:8020`; no headers or payload were captured. The running Tailnet Serve-peer trust was still `192.168.112.1/32`.
- R2 credential entitlement for `casebible-sorted` and `nexus` remains unproved and was not changed. The earlier diagnostic receipt found `NotEntitled` for both buckets; B2 remained available. Coolify briefly reported healthy during startup, before Docker health settled.

## Planned change and rollback

Change only the Coolify Workbench production environment key `WORKBENCH_TAILSCALE_SERVE_PROXY_CIDRS` from `192.168.112.1/32` to the observed exact peer `10.201.0.1/32`, then redeploy the same Git revision. Keep `TRAEFIK_PROXY_CIDR=10.201.0.2/32`, all Authentik/portal configuration, and the object-store mapping untouched. Recheck the Tailnet route, untrusted/header-spoof denial, public Authentik redirect, container health, and B2 listing.

Rollback value: `WORKBENCH_TAILSCALE_SERVE_PROXY_CIDRS=192.168.112.1/32` through Coolify followed by a redeploy. That value is known to deny the current Tailnet route; use it only if the new exact-peer setting causes a worse regression. Prior app deployment ID and current deployment ID are recorded above.

## Post-change verification

- Coolify production environment readback showed `WORKBENCH_TAILSCALE_SERVE_PROXY_CIDRS=10.201.0.1/32` and unchanged `TRAEFIK_PROXY_CIDR=10.201.0.2/32`. Coolify deployment `p2s6gdtk1leadcaoyo2w7e2x` finished at the same source SHA `200d03cc0054d7eb41b411caba596918f40696f5`.
- From this enrolled Tailnet client, `https://workbench.tilapia-skilift.ts.net/` and `/sources` returned HTTP 200 after deployment. The earlier HTTP 403 `Untrusted proxy` was resolved. This is a live route check, not a full interactive browser review.
- The TEST-mode B2 vault browser API returned 200 rows on page one (160 objects, 40 prefixes) and a continuation token. Page two returned 200 additional rows (174 objects, 26 prefixes), reaching 400 rows without exposing object names. Both pages remained truncated with a continuation. `b2-bucket` root listing also succeeded. REAL-mode listing returned HTTP 503 with `REAL matter identity is not configured`; no REAL-mode claim is made.
- Public unauthenticated `/`, `/docs`, `/openapi.json`, and `/api/health/deps` redirected to Authentik (HTTP 302). Supplying forged Tailscale or Authentik identity headers to the public route still redirected (HTTP 302). The raw Coolify sslip hostname returned HTTP 404. Direct loopback `/` and `/docs` without identity returned HTTP 403. An authenticated public browser session was unavailable for a full user-surface round trip; the forward-auth redirect and proxy topology alone do not prove that flow.
- Once startup settled, Docker reported the new Workbench container unhealthy. Its loopback `/health` JSON returned `status=degraded`, `lancedb=true`, `object_store=false`. Coolify's brief `running:healthy` immediately after deployment was a startup observation and is not the final health result. Staged-upload write/read-back was not attempted while the required R2 bucket entitlement is absent.
- The only R2-named host credential document found under `/data/probata/secrets` was the already mounted `casebible-r2.json`; the prior receipt's read-only probes established that it was not entitled to list/head `casebible-sorted` or `nexus`. No approved alternate credential or policy evidence established both compatibility read access and scoped `nexus/workbench/staging/` read/write. No credential value, mapping, privilege, or storage object was changed. R2 entitlement and a scoped staging drill remain open with the credential owner.
- Exact-peer trust relies on the VPS host bridge. A process already running on that host can potentially appear as the trusted Serve peer; this check did not establish protection from a compromised host-local process. That boundary remains for host-level review.
