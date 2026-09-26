# Legal-Workspace Gap Sheet & Remediation Plan

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Claude Code · Kimi K2.7 · 2026-08-18_  
> _Scope: Reconcile as-built `Legal-Workspace` against `docs/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md` and the 7 category handoffs. Produce a prioritized plan to bring it to spec._

---

## 0. Progress update

- ✅ **Git repository created** on GitHub: `https://github.com/Cursedpotential/Legal-Workspace`
- ✅ **Persistence refactor completed**: SQLite is the runtime source of truth, backed by SQLAlchemy models that mirror the PostgreSQL schemas (`legal_core`, `legal_research`, `legal_work_product`, `legal_audit`). JSON files are kept only as a debug/audit mirror. All **134** tests pass.
- ✅ **Plain-English route/label rename completed**: all 31 UI surfaces now use plain-English paths and labels. 4-letter abbreviations (`/drft`, `/rvw`, `/fctr`, `/agnt`, etc.) and legal jargon removed from routes, labels, and help text.
- 🔄 **Monorepo workspace wiring** — root `package.json` already lists `Legal-Workspace/web` as a workspace member, but there is no `turbo.json` or shared-package boundary yet. Awaiting owner decision before relocating the folder.
- ❌ **Functional depth** — most features are still stubs or mocked. Research is hardcoded to CourtListener, document workspace has no real editor, workflow steps do not execute real tools, agent roles route to `unevaluated-manual` by default.

---

## 1. Executive summary

Grok built a **standalone-looking FastAPI + Next.js app** that is far wider than it is deep. The directory is **nested inside `the-platform-workspace`** with partial monorepo wiring, and originally violated the owner’s plain-English requirement by using cryptic abbreviation paths and legal jargon throughout labels and help text.

The app is currently a **surface-heavy scaffold** with many page stubs, mocked or unevaluated agent routing, and no verified end-to-end flow. It is **not court-safe and not production-ready**.

Three granular audits were completed:

1. **Backend/persistence audit** — confirmed the SQLite/SQLAlchemy persistence layer is wired and tests pass; flagged the remaining JSON debug writes, missing issue/factor tables, and the need for repository-layer separation.
2. **UI/UX route audit** — mapped every abbreviation path and jargon label to a plain-English replacement.
3. **Feature inventory vs. spec** — compared every surface against the build guide and all 7 category handoffs, identifying stubs and missing features.

---

## 2. Confirmed as-built state

### 2.1 Repository / packaging

| What is on disk | What spec/owner wants |
|---|---|
| `Legal-Workspace/` folder inside `the-platform-workspace` | A proper workspace member of the monorepo |
| Standalone `pyproject.toml` | Monorepo package under `apps/legal-workspace` or `packages/legal-*` |
| Standalone `web/package.json` already listed in root workspaces | Workspace dependency on shared packages |
| Root `package.json` exists but no `turbo.json`, no `pnpm-workspace.yaml` | pnpm workspaces + Turborepo pipeline |
| Grok bylines on `README.md`, `AGENTS.md`, `compose.yaml`, `pyproject.toml`, many source files | Claude-Code-managed bylines per house rule |

### 2.2 Web frontend

- **Next.js 16 App Router, React 19, TypeScript 5** — matches spec.
- **31 page routes**, now using plain-English paths (`/assistant`, `/case-search`, `/laws`, `/drafts`, `/review`, `/final-copy`, `/evidence-requests`, etc.).
- **Command palette, terminal shell, split workspace, dark mode, status bar, case phase switcher** — present; sidebar labels and help text are now plain English.
- **No shadcn/ui, Radix, or Tailwind 4** evidence in `package.json`.
- **No generated TypeScript client from OpenAPI**.
- **CORS only allows `localhost:3010`** — will break real deploys.

### 2.3 Python API

- **FastAPI 0.141, Pydantic v2, uv** — matches spec.
- **Single-file `main.py` is very large** (imports many domain modules).
- **In-memory scheduler (APScheduler MemoryJobStore)** — spec wants durable PG-backed scheduling.
- **SQLite is the runtime store**; PostgreSQL migration path exists via `DATABASE_URL`.
- **Dependencies present for:** `sqlalchemy`, `psycopg`, `eyecite`, `pikepdf`, `pypdf`.
- **Dependencies missing for key spec requirements:** `pgvector`, `presidio`, Collabora/OnlyOffice/TipTap/Lexical, `unstructured`, `docling`, `pandoc`, `weasyprint`, `soffice` wrapper.

### 2.4 Data / persistence

| Spec requirement | As-built |
|---|---|
| PostgreSQL schemas: `legal_core`, `legal_research`, `legal_work_product`, `legal_release`, `legal_audit` | ✅ SQLAlchemy models in `api/legal_workspace/db/models.py` mirror these schemas; SQLite used locally with `legal_*` table prefixes |
| Pydantic contracts → config → repositories → services → API | ✅ Contracts exist; `WorkspaceStore` in `api/legal_workspace/db/store.py` is the coarse repository-like layer |
| Event outbox with idempotent consumers | `legal_audit.event_outbox` table exists; **audit row is written on every save**; idempotent consumer not yet implemented |
| Append-only audit with payload hashes | ✅ Every save writes an audit event with SHA-256 payload hash |
| Every mutation writes DB before API returns | ✅ SQLite is now the source of truth; JSON files kept only as debug backup |
| Swap to PostgreSQL later | ✅ Engine is URL-driven; changing `DATABASE_URL` switches to Postgres with no code changes |

**Known debt:** `WorkspaceStore.save()` currently deletes *all* rows in each child table because the app is single-Matter and Matter identity can change during Agno projection. This must be replaced with per-Matter deletes before multi-Matter support.

### 2.5 Domain coverage

The app has page stubs for nearly every spec surface, but most are **not functionally complete**:

- **Matter Home**: hardcoded to a single Genesee County custody matter.
- **Issue tree / Questions the judge decides**: single hardcoded issue shown.
- **What the judge must consider**: page exists; factor notes editable.
- **Motion writer**: per-section editor exists; no full-document preview/export/numbering.
- **Your review / Final review copy**: forms exist; release manifest + DOCX export works server-side; UI download wiring is partial.
- **Research tools**: CourtListener integration present; no `pgvector`/BM25 hybrid search; no configurable sources.
- **Document workspace**: no actual editor (Collabora/OnlyOffice/TipTap/Lexical) integrated; PDF viewer is browser-only.
- **Redaction**: `pikepdf` dependency present; functional redaction workflow unverified.
- **Bates/exhibit stamping**: functions exist; not fully wired to exhibit flow.
- **Evidence requests**: form exists; not linked to real missing-proof workflow.
- **Court dates**: manual entry only.
- **Agent runs**: routing table exists, but every agent uses `unevaluated-manual` model and live invoke is gated by `LEGAL_WORKSPACE_INVOKE_MODELS=false`.
- **Confidential Mode / provider grid**: UI exists; real ZDR provider verification not done.

---

## 3. Critical gaps (must fix before anything else)

### 3.1 Plain-English requirement — DONE ✅

All 31 UI surfaces now use plain-English paths and labels. Examples:

| Old path | New path | Old label | New label |
|---|---|---|---|
| `/chat` | `/assistant` | Paralegal | Ask the assistant |
| `/prec` | `/case-search` | Precedent Search | Case search |
| `/stat` | `/laws` | Statutes | Laws |
| `/cite` | `/citation-check` | Citations | Citation check |
| `/rqst` | `/open-questions` | Research questions | Open questions |
| `/issue` | `/questions` | Issue tree | Questions the judge decides |
| `/fctr` | `/custody-factors` | Best-interest factors | What the judge must consider |
| `/ctrx` | `/agreements` | Contract Workbench | Agreement review |
| `/doc` | `/documents` | Document Analyzer | Document viewer |
| `/priv` | `/confidentiality-check` | Privilege Check | Confidentiality check |
| `/drft` | `/drafts` | Brief Builder | Motion writer |
| `/tmpl` | `/templates` | Motion outlines | Starting templates |
| `/rvw` | `/review` | Owner review | Your review |
| `/rels` | `/final-copy` | Release candidate | Final review copy |
| `/file` | `/filing-checklist` | Filing checklist | Filing readiness checklist |
| `/disc` | `/evidence-requests` | Discovery | Evidence requests |
| `/exh` | `/evidence` | Exhibit list | Evidence list |
| `/miss` | `/missing-evidence` | Missing evidence | Missing evidence |
| `/todo` | `/tasks` | Tasks | Your tasks |
| `/cal` | `/calendar` | Docket Watch | Court dates |
| `/agnt` | `/assistant-log` | Agent log | Assistant activity log |
| `/strat` | `/private-notes` | Private strategy | My private notes |
| `/team` | `/challenge-draft` | Red team | Devil's advocate review |
| `/audt` | `/activity-log` | Audit Log | Activity log |
| `/autm` | `/scheduled-jobs` | Automations | Scheduled jobs |
| `/trig` | `/notices` | Triggers | Inbound notices |
| `/wkfl` | `/playbooks` | Workflows | Playbooks |
| `/live` | `/external-sources` | Integrations | External sources |
| `/jobs` | `/analysis-queue` | Analysis Queue | Analysis queue |
| `/timl` | `/timeline` | Timeline | Timeline |
| `/auth` | **removed** | — | — |

Help text was also rewritten to remove “citator,” “Shepardize,” “MCL 722.23,” “interrogatories,” “RFPs,” “RFAs,” “FOC,” and similar jargon.

### 3.2 Monorepo integration is partial

- Root `package.json` lists `Legal-Workspace/web` as a workspace.
- No `turbo.json`, no shared packages, no unified `dev`/`build`/`test` pipeline.
- Cannot run `pnpm dev` / `turbo build` / `turbo test` from `the-platform-workspace` root.

### 3.3 Functional depth is mostly stubs

The app has the right surfaces but not the right behavior behind them. The biggest missing pieces:

- Real document viewer/editor.
- Configurable research sources + hybrid search.
- Real agent model routing when `LEGAL_WORKSPACE_INVOKE_MODELS=true`.
- Postgres-backed scheduler + n8n boundary.
- Hybrid privilege classifier (Presidio + LLM escalation).

### 3.4 Auth / identity / boundary

- No authentication or authorization layer visible.
- `Matter` and `CourtCase` are hardcoded/mock projections.
- No verified `LegalSourcePackage` import from the Evidence Platform.
- No `EvidenceInvestigationRequest` outbound API verified.

### 3.5 Byline / attribution drift

Grok bylines still remain on many core files. They should be updated to Claude-Code bylines as files are touched.

---

## 4. Structural gaps

| Area | Gap | Severity |
|---|---|---|
| Monorepo tooling | No `turbo.json`, no shared packages, no root pnpm workspace config | HIGH |
| Shared packages | No `@repo/ui`, `@repo/tsconfig`, `@repo/eslint-config` | MEDIUM |
| OpenAPI client generation | No generated TS client from FastAPI | MEDIUM |
| DB layer | SQLite works; repository separation incomplete; issue/factor tables missing | MEDIUM |
| Migration discipline | `sql/0001_legal_os_bootstrap.sql` is marked HOLD; `sql/0002_legal_automations.sql` is also HOLD | MEDIUM |
| Event system | Outbox table present but no producer/consumer implementation | MEDIUM |
| Secret management | Passwords/default credentials in compose and `.env.example` | CRITICAL |
| CORS / security | Dev-only localhost CORS; no auth middleware | CRITICAL |
| Tests | **134 tests pass**; coverage gate and web E2E tests not in place | MEDIUM |
| CI/CD | No GitHub Actions, no pre-commit, no lint gate | MEDIUM |

---

## 5. Functional gaps by category

### Category 1 — Persistence & Settings
- ✅ Settings config exists
- ✅ SQLite source of truth wired
- ❌ Issue/factor tables not modeled
- ❌ Per-Matter deletes not implemented (single-Matter hack deletes whole child tables)
- ❌ No SurrealDB evidence-grounding client
- ❌ No read-only Agno REST fallback

### Category 2 — Command Bar & UI Shell
- ✅ Terminal shell, command palette, split view, status pill, case phase switcher present
- ✅ Plain-English labels applied
- ❌ No verified Tailwind 4 / shadcn / Radix stack
- ❌ No mobile overlay mode verified

### Category 3 — Chat / AI Agent Orchestration
- ✅ Chat route, agent routing table, F1 summon concept present
- ❌ All agents default to `unevaluated-manual` — not real models
- ❌ No evidence-grounded responses verified
- ❌ No automatic context injection from active panel verified
- ❌ No deep-research loop

### Category 4 — Research Tools
- ✅ Pages for search, laws, citations, open questions
- ✅ `eyecite` dependency present
- ❌ No `pgvector` + BM25 hybrid search
- ❌ No configurable/pluggable research sources
- ❌ No rate-limiting / cost-warning UI

### Category 5 — Contract & Document Analysis
- ❌ No in-app document editor (Collabora, OnlyOffice, TipTap, Lexical)
- ❌ No DOCX/PDF conversion pipeline beyond server-side DOCX release export
- ❌ No metadata scrub/view/validate/print workflow
- ❌ No Michigan forms template library
- ❌ Redaction function unverified end-to-end
- ❌ Bates stamping function unverified end-to-end

### Category 6 — Privilege, Privacy & LLM Routing
- ✅ Confidential Mode toggle, provider grid UI
- ❌ No Presidio PII detection
- ❌ No verified ZDR provider configuration
- ❌ Live model invoke gated off; no real routing proven

### Category 7 — Workflow & Automation Engine
- ✅ APScheduler in API
- ❌ MemoryJobStore instead of Postgres
- ❌ No n8n delegation boundary
- ❌ Workflow builder is a stub
- ❌ No durable event-triggered automations

---

## 6. Remediation plan

### Phase A — Foundation (in progress / next)

1. ✅ Plain-English route/label rename.
2. 🔄 **Monorepo workspace structure:**
   - Add `turbo.json` with `build`, `dev`, `lint`, `test`, `type-check` pipelines.
   - Add shared packages (`@repo/tsconfig`, `@repo/eslint-config`) or decide against them.
   - Wire Python `test`/`lint`/`type-check` into the Turbo pipeline.
   - Owner decision needed on whether to relocate `Legal-Workspace` to `apps/legal-workspace`.
3. **Complete SQLite repository layer:**
   - Add issue/factor tables to `db/models.py` and `sql/0001_legal_os_bootstrap.sql`.
   - Replace whole-table deletes with per-Matter deletes.
   - Remove JSON debug writes or gate them behind a `LEGAL_WORKSPACE_DEBUG_JSON=1` env var.
4. **Verify and tighten security:**
   - Remove hardcoded passwords from `compose.yaml` and `.env.example`.
   - Validate required secrets at startup.
   - Replace dev-only CORS with Tailscale-name allowlist.

### Phase B — First vertical slice (usable MVP)

5. **Release export wiring:** connect the server-side DOCX export to a UI download button; add manifest preview.
6. **Document workspace stub → real viewer:** integrate `pdf.js` or `react-pdf-viewer`; add text-extraction preview.
7. **Configurable research sources:** allow JSON upload of `SourceConfig`; keep CourtListener as default; add cost warning.
8. **Real agent routing when enabled:** when `LEGAL_WORKSPACE_INVOKE_MODELS=true`, route through Portkey/model-gateway with cost/audit recording.
9. **Hybrid privilege scan:** keyword first pass + LLM escalation + redaction preview.
10. **Hearing/deadline calculation:** rule-based deadline calculation with trigger/authority/timezone.

### Phase C — Expand to spec

11. Research tools (`pgvector`/BM25, hybrid search, general web search).
12. In-browser document editor + full format-conversion pipeline.
13. Discovery + missing-proof workflow.
14. Calendar/deadlines with real calculation + explanation.
15. Agent roles with real model routing and cost/audit records.
16. Confidential Mode + verified ZDR provider grid.
17. Workflow engine with PG-backed scheduler and n8n boundary.

---

## 7. Immediate next-action recommendation

With plain English done and tests green, the next highest-leverage fixes are:

1. **Monorepo wiring** — add `turbo.json` and unify `build`/`test` from the platform root.
2. **Release export UI wiring** — make the final review copy downloadable from the UI.
3. **Document viewer stub** — swap the iframe-only PDF viewer for `pdf.js` + text extraction.
4. **Configurable research sources** — allow JSON upload of a `SourceConfig` and surface it in settings.

These four turn the scaffold into something that actually does work the owner can see and use.

---

## 8. Open questions for owner

Only one decision is genuinely blocked on you:

1. **Workspace relocation:** Should `Legal-Workspace` stay where it is as a workspace member, or be moved to `apps/legal-workspace` (or `packages/legal-workspace`)?

Everything else can proceed autonomously against the spec once that is settled.
