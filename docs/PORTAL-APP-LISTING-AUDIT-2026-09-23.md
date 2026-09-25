# Shared portal app-listing audit — 2026-09-23

> Byline: Codex · GPT-6 · 2026-09-23 08:20 EDT. Read-only live-source and unauthenticated route audit. No portal files, application code, credentials, deployment, or Authentik configuration were changed.

## Planned change and checkpoint

The owner requires one shared, Authentik-protected multi-app portal off Tailnet; direct Tailnet access has Tailscale as its sole barrier. Each user-facing application surface should have exactly one launcher entry, and launcher links should not lead to an API, storage engine, administration, debug, or service endpoint. Probata and Intake are P0. This audit locates the live listing source and identifies repairs for a separately coordinated portal release. It does not authorize a live edit.

Checkpoint: `/data/dashboards/homepage/services.yaml` and `/data/dashboards/homepage-public/services.yaml` on `ovh-app` are the mounted Homepage listing sources. The older local `E:/AI_Workspace/Projects/dashboards` folder is unversioned and is not the deployment source. Both live files contain one named Intake workspace entry and one named Probata workbench entry. The remote `/data/dashboards` tree is also unversioned; no canonical Git revision currently controls these mounted listings.

## Source and deployment evidence

| Live source or runtime | Evidence observed 2026-09-23 | Limit |
|---|---|---|
| Tailnet Homepage listing | `/data/dashboards/homepage/services.yaml`; SHA-256 `20425a5e499c98303b438c6bce56e824431c4f28ecf4f82139cfbfd90bc1b25f` | Plain Compose bind mounts this directory at `/app/config`; not Git controlled. |
| Public Homepage listing | `/data/dashboards/homepage-public/services.yaml`; SHA-256 `4147237528c86b01ab3116c6b6a0d703b4ddb01e8b744c05cb5d3324a6f535bb` | `homepage-public` container bind mounts this directory at `/app/config`; not Git controlled. |
| Progress/Intake preview server | `/data/dashboards/progress-board/server.mjs`; SHA-256 `3a7d65438900988cf00558954b342dd337dc71d5be5900bcc95c84cbc45145c2` | Coolify-managed container has read-only source mount. Its running image and mounted file are separate evidence from a Git release. |
| Intake release pointer | `/data/dashboards/progress-board/intake-build/current.json`; SHA-256 `07e89b85f46f29719932459584ec1315dd9cfe715d8f6673b932d64d7ec32e9c` | Names a 2026-09-19 hosted Intake UI build; does not prove integrated workflow execution. |

The listing schema is Homepage's `services.yaml`: an array of named groups containing named cards, each optionally holding `href`, `description`, `icon`, `siteMonitor`, or a widget with a server-side `url`. Both active files parsed with the installed Homepage `js-yaml` module (7 tailnet groups, 8 public groups). This is syntax validation, not a browser, authorization, or target-function test.

## P0 card results

| Card | Listing and route | Access evidence | Disposition |
|---|---|---|---|
| Consignatio / Case Bible Intake | One card in each Homepage. Tailnet points to the same-origin Progress Board `/progress/intake/` path; public points to the public Progress Board `/intake/` path. | Tailnet route returned HTTP 200 HTML. Public route returned HTTP 302 into Authentik before content. The served page is an Intake shell embedding the versioned Xplorer build in an iframe, with a fallback direct Xplorer link. | P0 card exists, but it currently identifies a preview shell. Full, current Intake workstation navigation and real selection-to-action behavior were not established; therefore this is not a verified complete P0 app listing. |
| Probata workbench | One card in each Homepage. Each points to its lane's Workbench hostname root, not to a raw API. | Public route returned HTTP 302 into Authentik before content. Tailnet Tailscale Serve maps the approved service hostname to a local Workbench proxy and advertises its application capability. The Tailnet hostname and local origin both returned HTTP 403 JSON `Untrusted proxy` from `uvicorn`. The Workbench container was `unhealthy` at observation time. | P0 card exists and targets the intended app root, but working Tailnet access is unverified. The observed 403 is produced by the Workbench proxy-trust layer after routing; it is not an Authentik challenge or a Tailscale Serve denial. An enrolled user's identity path may differ from this server-side probe. |

Unauthenticated redirects were followed only far enough to identify the Authentik login flow. No credentials were used and no authenticated content was requested. The public Homepage itself also returned HTTP 302 to Authentik. No direct-origin bypass, approved-user login, logout, session-expiry, asset, WebSocket, or real Tailnet-user test was completed.

## Current listing inventory and exceptions

The tailnet file has 31 cards in 7 groups; the public file has 32 cards in 8 groups. Many are dashboard widgets or status views, not distinct apps. The user-facing workspace cards are Intake preview, Probata Workbench, Family Court read-only preview, and Advocatio legal preview. Tailnet also lists FileFlows as a media-processing tool; public does not. Other cards represent project progress/health, OpenList and Filestash storage views, database browsers/explorers, operations consoles, developer tools, and a public Secrets group.

High-priority exceptions in the public listing:

- The Advocatio card uses a Tailnet-only hostname despite appearing on the public Homepage. This is a wrong-lane target; an off-Tailnet authenticated user cannot be assumed to reach it.
- Administration and internal tooling cards are listed under Storage and databases, Operations, Preview pipeline, and Secrets. Candidates for removal from the user launcher include database browsers, deployment/workflow consoles, model and MCP gateways, the portal editor, developer sessions/probe playground, and secrets management. Their presence as clickable entries conflicts with the owner's user-surface-only rule. Whether any is deliberately approved as a distinct user app needs an explicit surface inventory, not an assumption from HTTP reachability.
- `PostgreSQL administration` and `Weaviate` are two cards with the same Workbench `/schemas` target. The labels imply different surfaces but do not provide unique destinations. Project progress, live-board widgets, and health cards also repeatedly link to the same progress/health pages; those should be modeled as widgets or one page entry rather than multiple app listings.
- The public file includes private service addresses under widget `url` fields and health probes under `siteMonitor`; its clickable P0 `href` values do not point to raw APIs. Homepage custom API widgets fetch on the server side, but this audit did not inspect browser payloads to prove those internal addresses cannot be disclosed through rendered/config APIs. Raw API and health endpoints must not become launcher `href` values.
- The no-link Surrealist and LibreChat placeholders are not functioning app listings. FileFlows is Tailnet-only, and no current user-surface register proved whether a public card is required. Completeness against *every* product surface remains open.

## Proposed source-of-truth reconciliation

1. Assign the portal's Git owner and bring the two reviewed Homepage listing files, Progress Board routing, and deployment manifest into that portal-owned repository. Preserve each remote file and hash as a dated baseline; do not overwrite the live mounts from the stale local mirror. Keep deployment-specific routes as explicit tailnet/public projections of a reviewed surface manifest.
2. Define one manifest row per user-facing surface with product owner, display name, full-app versus preview state, Tailnet route, public route, public Authentik application/policy, and verification receipt. Generate or validate the two Homepage files from this manifest so the same surface appears once per intended lane.
3. Resolve P0 first: identify Intake's actual complete workstation route and promote the card only after source/release/functional proof; diagnose Workbench's proxy trust with an enrolled Tailnet identity and the current unhealthy deployment, then confirm the app root works without a second login. Keep the existing public Authentik gate while checking approved-user behavior.
4. Remove or relocate internal/admin/debug cards from the user launcher after product-owner review, correct Advocatio's public-lane destination, and consolidate repeated `schemas` and progress/health links. Review the resulting diff and validate both Homepage configs in their containers before any separately authorized rollout.

## Boundary and remaining checks

No clear portal-owned P0 source typo was proven: both P0 cards exist and point to app-shaped routes. The defects are preview completeness, unproven Workbench access, the legal wrong-lane link, internal/admin listing policy, duplication, and lack of Git-controlled live source. Nothing was patched or deployed. The exact live checks still needed are authenticated public click-through for each approved user surface, enrolled Tailnet click-through for Probata and Intake, Intake real workflow proof, Workbench proxy/health diagnosis, browser payload inspection for internal address exposure, direct-origin bypass checks, and a current product-by-product surface register.

Docstore was queried for related current decisions before this receipt. Its indexing/sync write was intentionally not run in this read-only live audit; this file remains a local Propria-root document until the coordinated documentation sync reads it back.
