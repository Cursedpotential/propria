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

# Advocatio — the legal workdesk (legal workbench) — Agent Orientation

<!-- Title corrected by Claude Code · Fable 5.1 · 2026-09-14 09:03 EDT (owner: "router is missing legal workbench work desk"). Was "Legal-Workspace — Agent Orientation". -->

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). ~~Directory: `probata/modules/advocatio/` (old name kept as a junction).~~ **Corrected 2026-09-14 (Claude Code · Fable 5.1): the directory is this repository, `E:\AI_Workspace\Projects\Propria\Legal-desktop`, per the owner routing decision of 2026-09-13 at the top of this file; the `probata/modules/advocatio-legal_workbench` location is retired.** GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_

**This is the legal-practice sibling of `Agno-MCP-Platform/`.**
Agno remains the evidence truth system. This repository begins after evidence
has been reviewed and accepted.

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
  writes `data/workspace/state.json` and appends `events.jsonl`
  before the API returns. Restart must reload the same Matter.
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
  content stream with pikepdf. Not Agno evidence. Not court-safe.
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

## Stack (locked)

- Web: **Next.js 16 App Router + React 19** — best Vercel AI SDK fit
  (RSC for Matter/FILE/RELS, client for CHAT/command line).
  UI **shape** is legal-terminal-master (sidebar, › command line,
  status bar, dark graphite). Do not switch this app to Vite.
- Chat/streaming: Vercel AI SDK on Next (`web/src/app/api/chat`).
  Domain mutations stay on Python.
- API: Python 3.12, FastAPI, Pydantic v2, uv only
- Data: PostgreSQL 18 schemas `legal_core` / `legal_research` /
  `legal_work_product` / `legal_release` / `legal_audit` on the existing cluster
- Object bytes: R2/S3-compatible (`legal/` prefix)
- Models: existing Portkey gateway (`model-gateway`)
- Agents: Agno adapter behind neutral interfaces

## Layout

```
api/legal_workspace/   Python package (contracts → config → repos → services → api)
web/                   Next.js Matter command center
sql/                   numbered raw SQL, same convention as Agno
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

## Service names (never tailnet IPs)

`legal-workspace` `legal-web` `legal-api` `legal-agents` `legal-renderer`
`legal-postgres` `evidence-platform` `model-gateway`

## This case

This remains a single-matter workspace; do not generalize it into a multi-client library. Historical
planning documents refer to `../Legal-desktop/` donor and handoff directories, but those directories
are not present in the live workspace as of 2026-08-29. Do not infer, recreate, or claim access to
those source packets. Use the reviewed material already retained under `docs/planning/` and record a
blocker if an absent donor is specifically required.

Sister repo `../Agno-MCP-Platform/` is CocoIndex-indexed. Search it with
`ccc search` **from that directory** — do not `ccc init` here and do
not re-crawl. Notes: `docs/planning/AGNO-INDEX-NOTES.md`.

---

> _Sprint-mode policy REMOVED 2026-08-25 on owner order ("you're grounded — remove it entirely"). Confirm-and-discuss-before-changing is back in force._
