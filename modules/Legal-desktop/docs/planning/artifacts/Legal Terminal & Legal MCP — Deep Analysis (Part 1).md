# Deep Analysis — Part 1: Public Donor Repos

> _Naming (D-138, 2026-09-05; applied 2026-09-06): this product is **advocatio** (formerly Legal-Workspace / Legal Workspace); the evidence platform it consumes is **Indicia Probata** / `probata` (formerly Agno-MCP-Platform). Directory: `probata/modules/advocatio/` (old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


Scope: [genego-io/legal-terminal](https://github.com/genego-io/legal-terminal) (shape donor) and [agentic-ops/legal-mcp](https://github.com/agentic-ops/legal-mcp) (primary toolset). Your customized local repos (custodyguide, your legal-mcp fork, and the other archives in `Sources and skills\`) are out of scope here — those need the other agent with local filesystem access. Sister-repo mapping (`mcp-platform-agno-mvp`) and the overlap matrix come after this.

---

## A. Repo Inventory

| Repo | Role | Language/Stack | License | Size | Status |
|---|---|---|---|---|---|
| [legal-terminal](https://github.com/genego-io/legal-terminal) | Shape/UX donor | React+TS+Vite+Tailwind (web) / Python+Textual (TUI) | **Proprietary — All Rights Reserved** | 134 files | v0.2, actively maintained (updated today) |
| [legal-mcp](https://github.com/agentic-ops/legal-mcp) | Primary toolset donor | Python 3.10+, FastMCP / official `mcp` SDK | **AGPL-3.0** | 76 files | V1, "implemented and tested" |

**Relationship between the two:** legal-terminal is a thin UI shell built entirely on top of legal-mcp's 27 tools — it ships with a `MockClient` fixture layer so it runs standalone with no backend, plus a documented (but not fully implemented) path to a `LiveClient` that would call a running legal-mcp server over SSE. They are separate repos by the same author (Edwin Genego), explicitly *not* a submodule relationship.

---

## B. Per-Repo Profile

### legal-terminal

- **Purpose:** Bloomberg-terminal-style, keyboard-first legal workstation UI. It is a *demo/showcase* of what a UI on top of legal-mcp could look like — no independent backend logic of its own.
- **Stack:** `webterm/` (React 18, Vite, TypeScript, Tailwind, Zustand for state, `cmdk` for command palette, `lucide-react` icons); `tui/` (Python, Textual ≥0.84, Rich ≥13).
- **License — important:** "All rights reserved," viewing permitted for personal evaluation only; copying, modification, derivative works, or any commercial/non-commercial reuse requires Edwin Genego's written permission ([LICENSE](https://github.com/genego-io/legal-terminal/blob/main/LICENSE)). This is materially stricter than legal-mcp's AGPL-3.0. See the **Licensing Note** below.
- **Structure:** `webterm/src/{mcp,store,components,panels}`, `tui/legal_term/{screens,widgets}`, shared `fixtures/*.json` synced between both frontends via `scripts/sync-fixtures.mjs`.
- **Testing:** 66 Vitest tests (web), 39 pytest tests (TUI).
- **Deployability:** Railway-ready (`railway.toml`, `Dockerfile`), runs in mock mode with zero backend by default; live demo hosted at [legal-terminal.up.railway.app](https://legal-terminal.up.railway.app/).
- **Explicitly NOT in this repo** (per its own `FEATURES.md`): real POP3/IMAP polling, server-side automation scheduler, cross-device sync, the legal-mcp server itself, live PACER/CourtListener data, Ollama inference routing, authentication, and TUI parity for several web-only panels.

### legal-mcp

- **Purpose:** A Model Context Protocol (MCP) server exposing legal-research and paralegal tools to any MCP-compatible AI client (Claude Desktop, Cursor, Codex CLI, etc.) — precedent/case-law search, statute lookup, citation validation, contract clause analysis, document risk analysis, privilege checking, and brief scaffolding.
- **Stack:** Python 3.10+, `mcp[cli]>=1.28`, FastMCP, Pydantic ≥2.11, httpx, `python-docx` for `.docx` ingestion/export. Transports: stdio, SSE, streamable-HTTP.
- **License:** AGPL-3.0 — free to fork, self-host, and modify privately; network-copyleft only triggers if you run a *modified* version as a network service reachable by others. For your single-user, personal, non-distributed use case, this is fully permissive.
- **Structure:** predictable, agent-friendly layout — `tools/` (27 tools, 8 categories), `resources/` (`legal://` URIs), `prompts/` (8 templates), `integrations/` (CourtListener + PACER, both opt-in/disabled by default), `data/` (offline JSON seed data), `tests/` (unit + integration), `.agents/skills/legal-mcp-toolkit/` (a portable `SKILL.md` — directly reusable in your own agent setup).
- **Feature flags:** every tool category (`LEGAL_MCP_ENABLE_RESEARCH`, `_CITATION`, `_CONTRACT`, `_DOCUMENT`, `_PRIVILEGE`, `_BRIEF`, `_ANALYSIS_QUEUE`, `_INTEGRATIONS`) can be toggled independently; `LEGAL_MCP_DEMO_MODE` gates all bundled sample data.
- **Privacy design worth stealing directly:** documented trust-boundary diagram (you → AI client → inference provider vs. MCP server → local data vs. opt-in live sources), a recommended-posture table by data sensitivity (privileged/work product → self-hosted or ZDR-contracted cloud only), and a hard rule that credentials are never echoed back in tool output.
- **Notable safety detail relevant to your case work:** cites *United States v. Heppner* (S.D.N.Y. Feb. 2026) — held consumer AI outputs are not privileged because there's no reasonable expectation of confidentiality under standard consumer ToS. Its `check_privilege_risk` tool and workflow rule #5 in the SKILL.md operationalize this directly. Given you're handling your own litigation matter, this pattern is worth adopting as-is in Legal OS.
- **Testing:** pytest unit + integration suite; live integrations tested via `httpx.MockTransport` (zero real network calls in CI).

---

## C. Full Feature Inventory

### C.1 — legal-terminal (UI/UX shape reference)

**High-level categories:**
1. Navigation & layout shell
2. Command interface (keyboard-first)
3. Conversational assistant (chat-over-tools)
4. Domain modules (16 total panels)
5. Backend/connection management
6. Privacy/confidentiality posture (UI-level)
7. Document upload
8. Status & activity surfaces
9. Visual design system

**Detailed breakdown:**

| Category | Feature | Detail |
|---|---|---|
| Navigation | Hybrid sidebar | Grouped by workflow (Assistant / Research / Contracts / Drafting / Operations); collapses to icon rail; active-module highlight |
| Navigation | Home dashboard | Quick-action cards, backend status, "Ask the Paralegal" entry, 30-event recent-activity feed |
| Navigation | Full-viewport modules | Consistent header (mnemonic badge, title, backing MCP tool subtitle), rich empty states with clickable examples |
| Navigation | Split view | Pin a 2nd module beside the primary one; close with × |
| Command UX | Mnemonic command bar | Bloomberg-style typed commands (`PREC breach of contract`, `CITE 2022 Cal.App.4th 1234`); tab-complete; 50-command history |
| Command UX | Command palette | Ctrl+K fuzzy search (`cmdk`) over all modules + recent commands |
| Command UX | F-key shortcuts | F1–F10 map directly to modules (web); F1–F7 (TUI) |
| Assistant | Paralegal chat | Intent router (not a separate LLM) over `research_legal_issue`, `validate_citation`, `generate_brief_outline`, `list_contracts`; structured clickable result cards (case/statute/citation/contract/brief); persistent conversation store; suggested prompts on empty state |
| Domain module | `PREC` Precedent & Case Search | Master-detail layout, example queries, case detail pane |
| Domain module | `STAT` Statute Viewer | Statute text + quick links |
| Domain module | `CITE` Citation Console | History sidebar, parsed-component display |
| Domain module | `CTRX` Contract Workbench | Analyze / Compare / Negotiate tabs; clause risk list; alternative-language suggestions |
| Domain module | `DOCA` Document Analyzer | Upload zone, example files, metadata + clause-risk table |
| Domain module | `PRIV` Privilege Risk Check | All-providers comparison table, document selector |
| Domain module | `BRF` Brief Builder | Outline generation with example prompts |
| Domain module | `JOBS` Analysis Queue | Upload zone, auto-refresh job table, job detail side panel |
| Domain module | `WKFL` Workflows | Browse system playbooks (from fixtures) + a **builder** (pick tools from MCP catalog, set params, save, test-run) |
| Domain module | `AUTM` Automations | Bind a workflow to a schedule (once/daily/weekly) or an event (`job_complete`, `document_upload`, `contract_selected`, `email_received`, `app_open`); runs only while tab is open |
| Domain module | `TRIG` Triggers + Paralegal Inbox | Inbound message queue with category badges (Contract/Privilege/Litigation/HR/General); simulated POP3 config; rule-based auto-routing to automations/prompts; "simulate inbound" demo injector |
| Domain module | `AUDT` Audit Log | Filterable tool-invocation log |
| Domain module | `LIVE` Integrations | CourtListener/PACER/server config status |
| Domain module | `WTCH` Docket Watch (**preview only, not built**) | Product pitch for continuous PACER docket monitoring vs. one-time search; 3-step explainer; alert channel spec (email/webhook active, RSS/in-app "soon"); interactive mock watchlist; vote button; PACER cost-requirement notice |
| Domain module | `CONF` Settings | Tabs: Privacy (Confidential Mode toggle + rules checklist + Ollama endpoint config + provider trust cards), General (startup panel/refresh interval/theme/sidebar default), Integrations (live MCP URL, CourtListener token, PACER username placeholders), Notifications (email digest, webhook URL, per-category toggles) |
| Connection mgmt | Mock↔live toggle | Status-bar pill; proxy-based client singleton swaps implementation with no reload; `VITE_MCP_URL` sets build-time default |
| Connection mgmt | Mock data layer | Shared JSON fixtures as canonical source; one sync script generates both TS and Python-consumable data; full offline demo |
| Privacy | Confidential Mode | One-click sidebar toggle; amber visual posture (border/badge/lock icon) across sidebar + status bar; **UI posture only today** — no actual inference re-routing wired yet |
| Document handling | Upload zone | Drag-drop, multi-file browse, folder picker (`webkitdirectory`); accepts `.pdf/.docx/.doc/.txt/.md`; client-side text read for text formats; used in DOCA (auto-analyze on click) and JOBS (auto-queue) |
| Status/activity | Status bar | Confidential badge, mock/live pill, integration placeholders, current view, latest activity event, clock |
| Status/activity | Activity feed | Last-30-event log across searches, chat responses, mode switches, uploads, queue ops |
| Visual design | Theme system | Graphite/slate dark theme; Playfair Display (headings/case names), Inter (chrome/body), IBM Plex Mono (data/citations/status); desaturated CRITICAL/HIGH/MEDIUM/LOW risk badges; consistent empty-state pattern (icon + title + description + clickable example) |

### C.2 — legal-mcp (primary tool/backend reference)

**High-level categories:** Research, Citation, Contract, Document, Privilege, Brief, Analysis Queue, Integrations — plus Resources, Prompts, and cross-cutting infrastructure (feature flags, audit, privacy guidance, agent-skill packaging).

**Detailed tool inventory (27 tools):**

| Category | Tool | What it does |
|---|---|---|
| Research | `search_precedents` | Keyword-ranked precedent search over local case data |
| Research | `search_case_law` | Case-law search with relevance ranking + summaries |
| Research | `extract_statute` | Statute text with optional legislative context |
| Research | `research_legal_issue` | Multi-source research across local cases, statutes, and CourtListener |
| Citation | `validate_citation` | Validates structure/reporter format (not existence or good-law status) |
| Citation | `normalize_citation` | Normalizes spacing + Bluebook-style abbreviations |
| Citation | `check_demo_database` | Checks citation against opt-in demo data only |
| Contract | `compare_contracts` | Clause-level differ with risk flags |
| Contract | `analyze_clauses` | Rule-based clause risk analysis incl. `missing_clauses` |
| Contract | `extract_clauses` | Template-filtered clause extraction |
| Contract | `suggest_clause_alternatives` | Curated alternative phrasings for risky clauses |
| Contract | `generate_negotiation_guide` | Per-clause accept/negotiate/reject guide + fallback language |
| Contract | `deep_analyze_clause` | Keyword heuristics + optional MCP LLM sampling for deeper reasoning |
| Document | `analyze_document` | Risk analysis for real `.docx`/`.txt` files |
| Document | `compare_documents` | Clause-level diff for real files |
| Document | `export_analysis_report` | Exports a formatted `.docx` risk report |
| Document | `extract_contract_metadata` | Structured extraction: parties, dates, governing law, term, liability cap, payment terms |
| Privilege | `check_privilege_risk` | Assesses AI-routing risk for privileged material (cites *Heppner*, ABA Rule 1.6) |
| Brief | `generate_brief_outline` | Outline from a brief framework by case type |
| Brief | `create_argument_structure` | IRAC-style argument scaffold |
| Brief | `generate_issue_statement` | Issue-statement framework from facts + law |
| Analysis Queue | `queue_document_analysis` | Queues a doc for background risk analysis, returns job ID |
| Analysis Queue | `get_analysis_status` | Job status (queued/complete/error) |
| Analysis Queue | `get_analysis_result` | Retrieves completed job result |
| Analysis Queue | `list_analysis_jobs` | Lists all jobs + statuses/timestamps |
| Integrations | `integration_status` | Reports which live sources are enabled/configured (never echoes credentials) |
| Integrations | `search_live_case_law` | Queries CourtListener/RECAP or PACER when enabled |

**Resources (`legal://` URIs):** server-config, case-database, statute-library, contract-templates, brief-frameworks, citation-standards, integrations (static) + per-case/statute/contract/brief dynamic templates.

**Prompts (8):** `precedent_analysis`, `statutory_interpretation`, `brief_construction`, `argument_development`, `contract_review`, `clause_comparison`, `citation_validation`, `authority_integration`.

**Cross-cutting infrastructure:**
- Per-category feature flags (`LEGAL_MCP_ENABLE_*`) — entire tool categories can be switched off.
- Demo-mode gate (`LEGAL_MCP_DEMO_MODE`) — 13 sample contract templates (NDA, MSA, DPA, HIPAA BAA, ToU, Privacy Policy, Advisor Agreement, CA Offer Letter, Post-Money SAFE, Cookie Notice), 18 cases, 9 statutes — all disabled unless explicitly enabled.
- Audit logging (`utils.audit`) on every tool/resource call.
- Documented privacy trust-boundary model + provider-by-sensitivity recommendation table (self-hosted/ZDR for privileged material).
- Opt-in live integrations: CourtListener/RECAP (free) and PACER (paid, fee-risk warnings baked into the docs).
- A portable `SKILL.md` (`legal-mcp-toolkit`) encoding 8 workflow patterns (contract risk triage, negotiation prep, metadata lookup, privilege-safe AI review, legal research, brief drafting, citation cleanup, batch analysis) as ready-to-use tool-call sequences — directly reusable as an agent skill in your own stack, license permitting (AGPL, personal use = no issue).

---

## Licensing Note & Decision

- **legal-mcp (AGPL-3.0):** Safe to fork, self-host, and modify privately for your personal, single-user, non-distributed Legal OS. No obligation to publish changes since you are not running it as a public network service. You can freely import its tool logic, prompts, and even the `SKILL.md` methodology layer.
- **legal-terminal (Proprietary/All Rights Reserved):** Its LICENSE requires permission for copying/derivative works, stricter than "personal use is fine."
- **Decision (confirmed 2026-08-17):** legal-terminal is itself explicitly "inspired by" Bloomberg Terminal's density/workflow philosophy — it is not the origin of this UX pattern, just one implementation of it. We're treating it the same way it treats Bloomberg: as inspiration for a **from-scratch rebuild**, not a source of copied code. Its panel taxonomy, mnemonic scheme, and module list (Section C.1 above) are the target *feature list* for Legal OS's UI — every one of those features gets reimplemented independently in our own stack, no component code, CSS, copy text, or fixtures copied. This fully resolves the licensing concern: functional/organizational ideas (what panels exist, what a command bar does) get thin copyright protection anyway, and we're not touching the protected expression (their actual code) at all.

---

## What's next

Once you (or the other agent) finish analyzing your local custom repos (custodyguide, your legal-mcp fork, and the rest of `Sources and skills\`), the next step per the build guide is:
1. Map the sister repo (`mcp-platform-agno-mvp`) capabilities.
2. Build the overlap matrix (Feature | Found In | Decision: reuse/adapt/build new/defer to Agno platform/exclude).
3. Stop for your review/pruning pass before any scaffold gets designed.
