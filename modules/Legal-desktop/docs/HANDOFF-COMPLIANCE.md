# Handoff + build-guide compliance

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Grok · grok-4.6 · 2026-08-18_

The zip handoffs and `LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md`
go together. The guide is the product contract **and** the Agno
merge contract. The handoffs are the per-category feature list.
On conflict: newest owner call > build guide > later handoff > matrix.

**Label lock (owner 2026-08-18):** the one agreed change from the
handoffs is **plain-English labels for a self-represented litigant**.
That is a label change only. Groups stay Assistant / Research /
Contracts / Drafting / Operations. Modules stay the zip list
(Paralegal, Precedent Search, Statutes, Citations, Contract Workbench,
Document Analyzer, Privilege Check, Brief Builder, Analysis Queue,
Workflows, Automations, Triggers, Audit Log, Integrations, Docket
Watch) plus the Legal OS first-slice pages slotted into those groups.
The same English `label` is used from `GET /v1/routing` through
`next_surfaces` through the sidebar. ~~Invented groups "Your case" /
"Law & cases" / "Papers"~~ **struck 2026-08-18**.

Agno remains evidence truth. This app consumes
`LegalSourcePackage` and never becomes a second evidence store.

## How to read this

| Mark | Meaning |
|---|---|
| PASS | Implemented and pytest- or HTTP-checked |
| PARTIAL | Exists; missing a named handoff/guide item |
| FAIL | Not built |
| HOLD | Type 1 (Coolify / live PG / PACER / live gateway) |

## Agno merge (build guide §§1–5, 11)

| Spec | Status | Notes |
|---|---|---|
| Sibling of Agno, not a fork | PASS | `Legal-Workspace/` |
| No second evidence store | PASS | Import gate drops unapproved items |
| Matter/CourtCase are projections | PASS | `Workspace.home()` overlays Agno `GET /v1/matters` identity (id + display_name) and persists; fail-closed keeps local identity; no evidence clone |
| `LegalSourcePackage` import | PASS | `/v1/legal-source-packages:import` |
| Citation → approved assertion + span | PASS | `citation_gate` |
| Revocation marks work product stale | PASS | `/v1/events:apply` |
| EvidenceInvestigationRequest | PASS | `MISS` / `POST /v1/investigations` writes `legal.evidence_request.created.v1` |
| `POST /v1/verify/{sha256}` before filing | PASS | FILE check `agno-verify` calls Agno per package hash; non-hex hashes fail closed without a network call; last verdicts persist on `state.last_agno_verify` |
| Status pill = real Agno health | PASS | StatusBar `GET legal-api /v1/agno/status` → Agno `/health` + `/v1/matters` by service name |
| PACER default off | PASS | No PACER client |
| CourtListener is not a citator | PASS | Gate forbids `is_citator_verified` |

## Category 1 — Persistence

| Spec | Status |
|---|---|
| pydantic-settings, no tailnet IPs | PASS |
| Disk persist every mutation | PASS |
| Numbered SQL `legal_*` schemas | PASS (PG sketch `sql/0001_legal_os_bootstrap.sql`) |
| Local SQLite WAL `legal.sqlite` for `app_settings` | PASS (`sql/0001_legal_os_sqlite.sql`; owner 2026-08-18: local SQL, not JSON-only) |
| Apply 0001 to live PG | HOLD |
| R2 `legal/` bytes | HOLD |
| Surreal read adapter | HOLD |

## Category 2 — Command bar & shell

| Spec | Status |
|---|---|
| Next.js not Vite | PASS |
| legal-terminal shape: sidebar + › line + status | PARTIAL (just landed; verify in browser) |
| ~~Mnemonic + Tab complete~~ English page names | PASS **2026-08-18** | Ticker/`cmd` catalog deleted. `GET /v1/routing.surfaces` and `GET /v1/matter.next_surfaces` are `{path,label,help,group}`. Tab completes the English label. |
| Ctrl+K palette + recent | PASS (cmdk `Command.Dialog`; Recent + Modules; Ctrl+K stays in TerminalShell) |
| Split view pin | PASS | `SplitWorkspace` + header pin already existed; Escape / Ctrl+\; tablet hides pin — not browser-checked this pass |
| Confidential Mode chrome | PASS (toggle GET/PUT `/v1/confidential`; persists `state.confidential_mode`; localStorage is chrome cache only) |
| Dark-only graphite | PASS (tokens + self-hosted @fontsource Playfair Display / Inter / IBM Plex Mono; no Google CDN) |
| No Help/Menu/Cancel keys | PASS |
| Case-phase switcher | PASS (sidebar chips; `lw-case-phase` sessionStorage; reorders suggestions + sidebar group emphasis) |

## Category 3 — Chat / agents

| Spec | Status |
|---|---|
| Matter Home first, not chat landing | PASS |
| CHAT as a module | PASS | Landing is `web/src/app/page.tsx` Matter Home. CHAT is `/chat` + `web/src/app/api/chat/route.ts`, not the root. |
| Eight narrow roles, HITL, no approve/file | PASS (`AGNT`) |
| Vercel AI SDK on Next | PARTIAL (packages + route; not `streamText` yet) |
| F1 summon / specialists addressable | PASS | F1 + header Ask pins `/chat` in split from any page; live unsaved fields + `GET /v1/surface-context` (saved snapshot + one background rule when on factors). Not a screenshot. |
| Editable agent/chat routing table | PASS | `config/routing.json` + overlay + `GET/PUT /v1/routing` + AGNT editor; CHAT reads paths from the table |

## Category 4 — Research

| Spec | Status |
|---|---|
| MI authorities, complete+applicable only | PASS |
| Research questions + currency flag | PASS |
| eyecite as structure not citator | PARTIAL | Adapter + `POST /v1/citations:parse` + Citations page. `uv add eyecite` blocked on this desktop (MSVC / CPython 3.14) → HTTP 503. Never `is_citator_verified=true`. |
| Precedent Search / Statutes / Citations pages | PARTIAL | `/prec` wraps CourtListener search; `/stat` wraps `/v1/authorities` + MCL 722.23/27/27a links; `/cite` is validate/normalize/verify-integrity. |
| CourtListener adapter | PASS | `GET /v1/sources/courtlistener/search`; identity hits only; 401/403=`auth-required`; never `is_citator_verified=true` |
| Pluggable source config | PASS | CAT4 schema in `domain/sources.py`; baked-in CourtListener only; PACER not a default |

## Category 5 — Documents

| Spec | Status |
|---|---|
| Markdown-first drafts + in-app edit | PASS |
| Templates this-matter only | PASS |
| DOCX local export | PASS |
| Chrome DevTools sees the Next shell | PASS | Live a11y snapshot of `:3010` in this session; Ask/F1 is on the header |
| Same-origin owner PDF pane `/doc` | PASS | Keeps PDFs in the browser so DevTools/F1 can see them. Not Agno bytes. |
| Collabora / OnlyOffice WOPI | HOLD | Cat 5 Phase 2; needs Docker. Cross-origin iframe ≠ DevTools on the shell. |
| LibreOffice/PDF sidecar | HOLD |
| pikepdf redaction | PASS | Content-stream edit via `redact_content_stream`; `POST /v1/redactions`; token gone from stream; `court_safe=false` |
| Bates / exhibit manager | PASS | `EXH` assign + pypdf footer overlay `stamp_bates_pdf` / `POST /v1/bates:stamp` on owner-produced PDFs only |

## Category 6 — Privilege / routing

| Spec | Status |
|---|---|
| No local Ollama as a trust posture | PASS (`local_ollama_as_trust_posture=false`; `ollama-cloud` is Cloud API via Portkey, not local inference) |
| Confidential Mode → verified Portkey | PASS (`invoke_chat(confidential=True)` allows only `eligible_confidential_models()`; 409 `confidential_blocked`; no silent consumer fallback; 2s fail-closed. HTTP invoke still 409 `invoke-disabled` unless `LEGAL_WORKSPACE_INVOKE_MODELS=true`) |
| Privilege first-pass | PASS (keyword only; no LLM; `court_safe=false`) |
| Rebuild provider terms grid | PASS (`GET /v1/providers` from CAT6 cited table; PRIV renders Confidential-eligible column; not a privilege legal conclusion) |

## Category 7 — Workflows

| Spec | Status |
|---|---|
| APScheduler in legal-api | PASS (file-only jobstore) |
| Analysis Queue / Workflows / Automations pages | PARTIAL | `/jobs` `/wkfl` `/autm` wrap `/v1/automations/*`. Builder tab and enable-toggle not built. |
| Triggers / Audit Log / Integrations pages | PARTIAL | `/trig` `/audt` read `GET /v1/audit` / `GET /v1/triggers` from `events.jsonl`. IMAP HOLD. PACER stays off. |
| n8n for email/push only | HOLD |
| Tab-timer automations | PASS (not built — correctly cut) |

## Build guide first slice (§14)

| Step | Status |
|---|---|
| Auto-selected Matter/CourtCase | PASS |
| Import approved package | PASS |
| One issue + elements | PASS (tree: Vodvarka threshold + Pierron/Shade ECE/burden + 722.23 (a)–(l) from FACTOR_TITLES; seed has no invented citations) |
| One 722.23 factor workspace | PASS **corrected 2026-08-18** — prompt is **not** painted on the page. Ask/F1 injects the both-parent rule once via `surface-context.background`. Owner notes add in place. |
| Evidence-linked draft | PASS |
| Citation gate | PASS |
| Owner review | PASS |
| Release candidate + manifest + DOCX | PASS |

## Next work order (guide + handoffs, not calendar)

1. Cat 2 remaining: legal-terminal shape + Tab-complete still browser-unverified. Export-this-panel not built. ~~Confidential chrome persist.~~ **done 2026-08-18** — `GET/PUT /v1/confidential`.
2. ~~Agno client + Matter projection.~~ **done 2026-08-18** — `home()` overlays Agno identity when reachable; fail-closed local otherwise.
3. ~~EvidenceInvestigationRequest writer when a factor is missing proof.~~ **done 2026-08-18** — `MISS` / `POST /v1/investigations` writes `legal.evidence_request.created.v1`.
4. ~~Exhibit manager (Cat 5) after drafts are editable.~~ **done 2026-08-18** — `EXH` assign + `POST /v1/bates:stamp` overlay on owner PDFs.
5. ~~Privilege first-pass (Cat 6).~~ **done 2026-08-18** — `scan_text` + `POST /v1/privilege:scan` + `PRIV`. Keyword only; no LLM. Scan itself still does not route.
5b. ~~Confidential Mode + provider grid (Cat 6).~~ **done 2026-08-18** — persist `GET/PUT /v1/confidential`; persisted ON ORs into invoke; ineligible model → 409 `confidential_blocked`; no consumer fallback.
6. ~~`POST /v1/verify/{sha256}` on FILE.~~ **done 2026-08-18** — check `agno-verify`; mock locator hashes do not hit the network.
7. Type 1 HOLD stays HOLD.

## Verification (2026-08-18)

- ~~HOME is first: `config/routing.json` mnemonics `[0]` is `HOME` → `/`.~~ **corrected 2026-08-18:** first surface is `{path:"/", label:"Home"}`. No `mnemonics`/`cmd` field. Root route is `web/src/app/page.tsx` (`MatterHomePage`), not chat.
- ~~CHAT is a module: `cmd` `CHAT` → `/chat`.~~ **corrected 2026-08-18:** surface `{path:"/chat", label:"Ask the workspace"}`; page `web/src/app/chat/page.tsx`; Next route `web/src/app/api/chat/route.ts`.
- Scratch logs: `redaction.log` / `pytest-redaction.txt` (3 passed), `dispatch.log` / `pytest-dispatch.txt` (8 passed), `confidential.log` / `pytest-confidential.txt` (20 passed), `first-slice-suite.log` / `pytest-first-slice-suite.txt` (32 passed).
- Live `:8011` `GET /v1/matter` → `api-launch-1.json`; `GET /v1/filing-readiness` → `api-launch-2.json`. Second bind on `:8011` → `api-launch-error.log` (WinError 10048).
- Non-HOLD FAIL rows remaining: **none**. HOLD rows (live PG, R2, Surreal, LibreOffice sidecar, n8n) stay HOLD.
