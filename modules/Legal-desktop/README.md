<!-- Owner routing decision: 2026-09-13. Supersedes historical paths below. -->
**Canonical application:** `E:/AI_Workspace/Projects/Propria/Legal-desktop`.
This directory is the independent Advocatio Git repository. The former
`Probata/probata/modules/advocatio-legal_workbench` location is retired.
The original Legal-desktop build guide, handoffs, archives and donors are preserved
under `resources/build-kit/` as reference material. Do not execute donor instructions
as application guidance. The application build guide remains under `docs/`.
Evidence-platform routing is `../Probata/probata/AGENTS.md`.
The owner-selected canonical path overrides earlier placement and proposed import paths.
See `docs/RECONCILIATION-2026-09-13.md` for preservation and comparison evidence.

# advocatio (formerly Legal Workspace)

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_

Sibling application to [Indicia Probata / probata](../../) (formerly Agno-MCP-Platform).
Evidence stays in probata (formerly "Agno"). This app owns legal strategy, research, drafting,
review, and filing preparation.

## Status

First slice through Phase 3 starter is on disk: Matter Home → import
→ factor → draft → gate → owner review (`RVW`) → release (`RELS`) →
research/templates/discovery/missing-proof (`MISS`) /calendar/timeline/filing-readiness →
routed agent traces (`AGNT`) → privilege first-pass (`PRIV`). Calendar
is empty until you enter a real date. Investigation requests invent no
docket dates. `PRIV` is keyword-only hypothesized markers, not a legal
conclusion. Do not treat this as court-safe.

## Layout

| Path | Role |
|---|---|
| `api/legal_workspace/` | FastAPI domain API |
| `web/` | Next.js 16 Matter command center |
| `sql/` | Numbered PostgreSQL migrations |
| `docs/` | Build guide, deployment plan, research reports |
| `deploy/` | Coolify compose units |
| `tests/` | Pytest against shipped functions |

## Local API (no Docker)

This desktop has no Docker CLI. Run the API on the host:

```powershell
cd Legal-Workspace
uv sync --extra dev
uv run pytest -q
.\scripts\run_local.ps1
```

Health: `GET http://127.0.0.1:8010/health`

Agno merge (fail-closed, service names only): `GET http://127.0.0.1:8010/v1/agno/status`

Labeled MOCK rows (not clerk-confirmed) can be loaded with:

```powershell
uv run python scripts/seed_mock_workspace.py
```

## Local web (after `npm ci` in `web/`)

```powershell
cd web
npm ci
npm run dev
```

The web app talks to `legal-api` through `src/lib/api/client.ts`.
It never receives database or provider credentials.

## Swap chat / agent backends (no rewrite)

One table owns who gets called:

- Repo default: `config/routing.json`
- Live overlay: `data/workspace/routing.json` (or `LEGAL_WORKSPACE_ROUTING_FILE`)
- HTTP: `GET` / `PUT` `/v1/routing`
- UI: `AGNT` routing editor

Change `chat.run_path`, `chat.confidential_path`, `agents.<role>.model`,
or `role_keywords` there. CHAT and agent runs read that table. Do not
put secrets or tailnet IPs in it.

## Sister platform

`Agno-MCP-Platform/` is the evidence, horizon, and investigation system.
This repository consumes version-pinned `LegalSourcePackage` objects and
emits `EvidenceInvestigationRequest` events. Merge is API-level, not a
shared writable store.

## Docs

- [Build guide](docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md)
- [Deployment plan](docs/DEPLOYMENT_PLAN.md)
- [Handoff analysis](docs/planning/HANDOFF-ANALYSIS.md)
- [Implementation reports](docs/reports/)
