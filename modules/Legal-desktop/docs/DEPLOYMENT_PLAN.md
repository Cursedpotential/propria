# Deployment plan

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_
> Status: written plan. **Not deployed.** Coolify writes are Type 1.

## Answer first

Deploy Legal Workspace as **new Coolify applications on OVH-1** (the Agno
box), on the existing `agno` docker network + a local `legal` network.
Reuse the live PostgreSQL 18 cluster via an isolated role and schemas.
Reuse Portkey (`model-gateway`) and R2. Do **not** stand up a second
evidence database.

This desktop has **no Docker CLI**. Compose is yaml-validated only.
Boot verification happens on the VPS after an owner Type 1 sign-off.

## Topology

```
browser
  → legal-workspace (Traefik / Tailscale service, HTTPS)
       → legal-web   (Next standalone :3000)
       → legal-api   (FastAPI :8010)
            → legal-postgres schemas on existing PG18
            → R2 prefix legal/
            → evidence-platform (Agno API)
            → model-gateway (Portkey)
            → legal-renderer (Phase 1, LibreOffice sidecar)
            → legal-agents (Phase 3)
```

Tailscale service names (never IPs):

| Name | Role |
|---|---|
| `legal-workspace` | browser entry |
| `legal-web` | Next |
| `legal-api` | FastAPI |
| `legal-postgres` | existing PG, new role |
| `legal-renderer` | ~~later~~ **live 2026-09-21** (Gotenberg 8.37.0 in `compose.yaml`, no published port; Claude Code · Fable 5.1) |
| `legal-agents` | later |
| `evidence-platform` | Agno |
| `model-gateway` | Portkey |

Public DNS (when exposed): `legal.mitechconsult.com` → OVH-1 public IP,
DNS-only, Let's Encrypt HTTP-01. Until then, tailnet only.

## Data plane

**Decision (Cat 1, grounded in Agno blueprint):** same PG18 host,
isolated schemas, dedicated role `legal_os_app`. Object bytes in R2
under `legal/`. No Surreal writes.

Migration: `sql/0001_legal_os_bootstrap.sql` applied **inside a
transaction, counted, rolled back** on a scratch DB first. Applying it
to the live Agno volume is Type 1.

Alternative rejected: a second Postgres container (`compose.yaml`
includes `legal-postgres` only for a disposable local/dev replica).
Production should point `DATABASE_URL` at the existing `agentos-db`
service name once Coolify wiring exists.

## Coolify units (one file per app, Agno S10 convention)

| App | File | Notes |
|---|---|---|
| legal-api | `deploy/Dockerfile.api` + root `compose.yaml` service | health `GET /health` |
| legal-web | `deploy/Dockerfile.web` | `output: "standalone"` |
| ~~(later)~~ legal-renderer | ~~not created~~ **running since 2026-09-21** as a service of the `legal-workspace` Coolify app | LibreOffice headless behind Gotenberg |

Coolify 4.1.2: create via `POST /applications/private-github-app` with
`github_app_uuid`. Env bulk = `PATCH .../envs/bulk` body `{"data":[...]}`.
Coolify-proxy owns host **8080** on every node — do not publish
legal-api on 8080.

Bind-mount host dirs, never named volumes.

## Auth and egress

- Same-origin browser → `legal-workspace` only.
- Browser never receives DB, R2, or provider credentials.
- Confidential Mode: Presidio (or regex first pass) then Portkey
  route to a verified-tier model. If none configured → hard-block.
- PACER stays off. CourtListener optional token in Coolify env.

## Phased rollout

### Phase 0 (this repo) — local contracts

- Scaffold, pytest, yaml-valid compose.
- No Coolify app, no live SQL apply.

### Phase 1 — usable drafting MVP

1. Owner sign-off on schema apply (Type 1).
2. Create Coolify apps; env from gitignored secrets.
3. Import one `LegalSourcePackage`.
4. Matter Home + FCTR + one draft section + citation gate.
5. Owner review (`RVW`) + deterministic release candidate (`RELS`).
6. Local DOCX beside the JSON/Markdown manifest. PDF waits for
   the renderer sidecar (Type 1). Calendar / historic timeline
   is Phase 2, not this deploy.

### Phase 2–4

Research adapters, deadlines from custody-packet calculator,
agents, calendar — see build guide §10.

## Rollback

- Coolify: previous deployment.
- Schema: additive only in 0001; later migrations must ship
  `NNNN_name.down.sql` or be transaction-safe.
- R2: never delete; quarantine prefix `legal/_quarantine/`.

## Verification (named checks)

| Check | When it becomes PASS |
|---|---|
| `uv run pytest` | local, this slice |
| `yaml.safe_load(compose.yaml)` | local, this slice |
| `GET legal-api/health` on VPS | after Type 1 deploy |
| Import package + citation gate on live data | after Type 1 deploy |

`BUILD_STATUS` for **deploy** is UNKNOWN until the VPS checks run.
