<!-- Byline: Codex · 2026-10-04 · Family Court convergence owner decisions. -->
## Family Court library convergence — owner decision, 2026-10-04

The validated Family Court library must be imported into the shared working
SurrealDB store (`surreal-case`, namespace `fct`, database `case`) and used by the
phone surface, desktop, Advocatio and platform. A read-only `/v1/toolkit` bridge
is the existing implementation, not the finished integration. Additions and edits
must reach the same records across those surfaces; do not create another library
database or persist a competing copy in Advocatio's SQLite store.

Preserve the original iterations and their substantive validation corrections.
Case Bible retains durable source originals under
`b2:salem-data/consignatio/casevault/`; continuous Case Bible synchronization is a
later slice. Every imported claim and reference needs traceable source identity,
version/hash, applicable pinpoint and validation evidence. Revalidate each library
update before propagating it. Report specific unresolved items rather than adding
generic warnings throughout the interface.

Use the settled React/TypeScript, TanStack, Storybook, Glide and Tauri stack and
the shared graphite/indigo design contract for the convergence. Preserve the
existing implementation while making bounded changes; this is not authority to
rewrite every workdesk subsystem or replace evidence-custody storage. Bulk source
inspection and processing run on the VPS as tracked Temporal Activities.

Current decision record: Docstore `note:family_court_convergence_20261004`.
These owner decisions supersede the read-only toolkit/library and stack statements
below for this convergence lane. Root `AGENTS.md` governs the current monorepo
boundary and `modules/Legal-desktop/` routing.

<!-- Routing reconciled 2026-10-17 against the completed 2026-09-26 root import. -->
**Canonical module:** `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`.
Advocatio belongs to the single `Cursedpotential/propria` repository. Its commit
boundary is the Propria root; applicable product instructions remain in this module.
The [root contract](../../AGENTS.md) and
[import decision](../../docs/decisions/2026-09-26-monorepo-import.md) govern current routing.
Original build guides, handoffs, archives and donors under `resources/build-kit/`
remain reference material. The application build guide stays under `docs/`.
Evidence-platform routing is `../Probata/probata/AGENTS.md`.

_Routing reconciliation: ChatGPT · GPT-6 Astra · 2026-10-17; existing owner decisions._

# Advocatio — the legal workdesk (legal workbench) — Agent Orientation

<!-- Title corrected by Claude Code · Fable 5.1 · 2026-09-14 09:03 EDT (owner: "router is missing legal workbench work desk"). Was "Legal-Workspace — Agent Orientation". -->

> _Naming: D-138, 2026-09-05. This product is Advocatio; the evidence-record
> product is Indicia Probata. Current naming authority:
> `../Probata/probata/docs/NAMING.md`. Historical aliases remain searchable (D-142)._

> _Byline: Grok · grok-4.6 · 2026-08-18_

**Advocatio is the legal-work module alongside Indicia Probata.**
Indicia Probata owns the evidence record. This module consumes reviewed and accepted
evidence through the governed integration contract.

## Read first

1. `docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md` — product contract
2. `docs/DEPLOYMENT_PLAN.md` — Coolify / Tailscale / compose plan
3. `docs/planning/HANDOFF-ANALYSIS.md` — what the zip handoffs decided
4. `docs/reports/` — Category 1–7 implementation research

## Boundary (do not cross)

- Do **not** ingest, hash, normalize, or establish evidence here.
- Cite accepted facts through `LegalSourcePackage` + exact spans.
- Missing proof → `EvidenceInvestigationRequest` back to the Evidence Platform
  (`MISS`, `POST /v1/investigations`, event `legal.evidence_request.created.v1`).
- Released documents are immutable. Edits create a new version.
- Agents cannot approve, file, serve, email, or transmit.
- **Never keep domain state only in process memory.** Every mutation
  is saved to SQLite (`WorkspaceStore.save`) before the API returns;
  `state.json` / `events.jsonl` are written only when
  `LEGAL_WORKSPACE_DEBUG_JSON=true`. Restart must reload the same Matter.
- **Strategy store (`STRAT`)** is private scratch: theories, directions,
  chat extracts, unfinished drafts. `court_safe=false`, not exportable,
  never an established fact. Only persist packet material that is
  complete and applicable.
- **Red team (`TEAM`)** stores adversarial evaluations of drafts/theories
  (opposing counsel, judge, FOC, missing proof). Same private lane.
- **Todos (`TODO`)** are owner tasks, not computed legal deadlines.
- **Owner review (`RVW`)** is the only path that can approve a draft.
  Agents cannot approve, file, or release.
- **Release candidates (`RELS`)** are deterministic manifests, not
  filings. Strategy, red-team, todos, and review rationale stay out.
- **Research (`AUTH` / `RQST`)** is snapshot-pinned Michigan authority
  plus owner research questions. Not a citator. Currency flags mark
  cited drafts `revalidation_required`.
- **Templates (`TMPL`)** are working outlines for this postjudgment
  matter, not official SCAO forms. Instantiation creates a private
  draft. Dates stay blank.
- **Discovery (`DISC`)** is request text linked to an issue and missing
  proof. Not served until the owner records an actual date.
- **Missing proof (`MISS`)** writes an `EvidenceInvestigationRequest`
  for Agno. Not a docket. Invents no dates. `court_safe=false`.
- **Calendar / historic timeline (`CAL` / `TIML`)** is a local docket.
  No Google account, no PACER, no Coolify. Owner types a clerk-set
  date. Empty is correct until then. Not Agno's authored factual
  timeline.
- **Filing readiness (`FILE`)** is a checklist only. It cannot file,
  serve, or email. Agents cannot verify checks.
- **Agent runs (`AGNT`)** are routed traces for eight narrow roles.
  Approve/file/serve intents persist as blocked. Live model calls
  stay off unless `LEGAL_WORKSPACE_INVOKE_MODELS=true`. Fail closed
  if `model-gateway` is unreachable. Output is never court-safe.
  Who gets called lives in `config/routing.json` (overlay
  `data/workspace/routing.json` or `LEGAL_WORKSPACE_ROUTING_FILE`).
  Edit the JSON or `PUT /v1/routing` — do not rewrite Next/FastAPI
  to remount a role or chat backend.
- **Exhibits (`EXH`)** are candidates from imported approved
  `LegalSourcePackage` items only. Annotations persist. No evidence
  bytes. No seeded Bates numbers.
- **Redaction (`POST /v1/redactions`)** edits an owner-produced PDF
  content stream with pikepdf. This does not modify Probata evidence. Not court-safe.
- **Confidential Mode** persists on `state.confidential_mode` via
  `GET/PUT /v1/confidential`. Chrome cache is localStorage only.
- **Privilege first-pass (`PRIV`)** is keyword-only hypothesized
  markers (attorney-client, work product, strategy, medical,
  child-identifying). Never a legal conclusion. `court_safe=false`.
  The scan itself does not route.
- **Provider grid / Confidential Mode** is the CAT6 cited terms
  table (`GET /v1/providers`). Eligible chat ids: `ollama-cloud`
  (primary), `openrouter-zdr` (secondary), `venice-private`
  (optional, never third-party relay). `nim-hosted` is embed/rerank
  only. Consumer Claude/ChatGPT are blocked. No local Ollama as a
  trust posture. No PACER. `invoke_chat(..., confidential=True)`
  hard-blocks anything else (`confidential_blocked`). Live invoke
  stays off unless `LEGAL_WORKSPACE_INVOKE_MODELS=true`.
- **Automations (APScheduler in legal-api)** use MemoryJobStore
  locally. `legal_core.automation_job` is later PG (Type 1 HOLD).
  Playbook steps are structural labels. No invented docket dates,
  no PACER, no n8n outbound (HOLD), no tab-timer automations.
  `court_safe=false`. Mutations go to `data/workspace` JSONL.

## Implementation and current convergence contract

- Web: **Next.js 16 App Router + React 19** — best Vercel AI SDK fit
  (RSC for Matter/FILE/RELS, client for CHAT/command line).
  UI **shape** is legal-terminal-master (sidebar, › command line,
  status bar, dark graphite). Do not switch this app to Vite.
- Chat/streaming: Vercel AI SDK on Next (`web/src/app/api/chat`).
  Domain mutations stay on Python.
- API: Python 3.12, FastAPI, Pydantic v2, uv only
- Data: SQLite in the bind mount (`/var/lib/legal-workspace/workspace` →
  `/data/workspace/legal.sqlite`) through `WorkspaceStore`
  (`api/legal_workspace/services/workspace.py`, `db/engine.py`). Postgres is not wired:
  `DATABASE_URL` is unset in `compose.yaml`, whose DATABASE note explains why.
  This SQLite ownership applies to workdesk state. The October 4 convergence
  decision above places the shared Family Court library in `surreal-case`
  (`fct/case`); the existing read-only `/v1/toolkit` bridge is an implementation
  step, not the final library boundary.
- Object bytes: R2/S3-compatible (`legal/` prefix)
- Models: existing Portkey gateway (`model-gateway`)
- Agents: Agno adapter behind neutral interfaces

## Layout

```
api/legal_workspace/   Python package (contracts → config → repos → services → api)
web/                   Next.js Matter command center
sql/                   workdesk SQL sources; inspect the current local database contract
tests/                 pytest against shipped functions
docs/                  canon, deployment, research reports
deploy/                one Coolify compose file per service
donors live in         ../Legal-desktop/donors/  (inspiration only, never import)
```

## First vertical slice

Matter Home → import `LegalSourcePackage` → one issue → one MCL 722.23
factor → one evidence-linked draft section → citation gate → owner review
→ deterministic release candidate + manifest.

Do not start with a generic chat screen.

## Service discovery

Use the current deployment configuration and service inventory for endpoints.
The August deployment plan and its proposed service names are dated history;
they do not establish which services are currently deployed.

## This case

This remains a single-matter workspace; do not generalize it into a multi-client library. Historical
planning documents refer to `../Legal-desktop/` donor and handoff directories, but those directories
are not present in the live workspace as of 2026-08-29. Do not infer, recreate, or claim access to
those source packets. Use the reviewed material already retained under `docs/planning/` and record a
blocker if an absent donor is specifically required.

For Probata source, follow `../Probata/probata/AGENTS.md` and the current root
indexing routes. `docs/planning/AGNO-INDEX-NOTES.md` records the historical
Agno-MCP-Platform path; it is not a current repository or index location.

---

> _Sprint-mode policy REMOVED 2026-08-25 on owner order ("you're grounded — remove it entirely"). Confirm-and-discuss-before-changing is back in force._
