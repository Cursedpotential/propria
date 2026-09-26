# Legal OS — Feature Decision Matrix

Every feature identified across the [Bloomberg + Legal Terminal Feature Catalog](./bloomberg-and-legal-terminal-feature-catalog.md) and the [Deep Analysis of legal-terminal + legal-mcp](./legal-terminal-and-legal-mcp-deep-analysis.md), sorted into **Keep** (build as-is), **Rescope** (build the underlying idea, but adapted for single-case/self-represented use), or **Exclude** (skip for now). This is a planning artifact, not a final decision — push back on anything and I'll move it.

**Summary:** 74 Keep · 28 Rescope · 10 Exclude (112 items total)

---

## Global Shell

| Feature | Decision | Note |
|---|---|---|
| Collapsible sidebar w/ workflow groups | Keep | Rename groups to your actual case phases |
| Light/dark theme toggle | Keep | |
| Confidential Mode toggle (sidebar footer) | Keep | Core to your privilege-safety posture |
| Mobile-responsive sidebar/overlay | Keep | Useful if you check things from a phone during a hearing recess |
| Mnemonic command bar w/ parsing + autocomplete + Tab-complete | Keep | This is the core "shape" you asked to borrow |
| Ctrl+K command palette (Recent + Modules) | Keep | |
| Version string in footer | Keep | Trivial, no reason to cut |
| Bloomberg: dedicated Help/Menu/Search/Cancel action-key pattern | Rescope | Softwareize as a persistent Help/Back/Cancel affordance in every module rather than hardware-style keys |
| Bloomberg: `LAST` (repeat/review last commands) | Rescope | Fold into command history — add a one-key "repeat last" |
| Bloomberg: `GRAB` (export screen as image/email) | Rescope | Build as a "export this panel" (PDF/image) button available everywhere, not just DOCA |
| Bloomberg: yellow Market Sector keys (jump into a domain context) | Rescope | Legal analogue: one-touch case-phase switch (Discovery / Motions / Hearing Prep / Trial) that reconfigures which shortcuts surface |
| Bloomberg: multi-monitor Panel-key routing | Exclude | Trading-desk-scale feature; revisit only if you actually run a multi-monitor setup |
| Bloomberg: Instant Bloomberg (cross-user messaging) | Exclude | No second user to message |

## HOME

| Feature | Decision | Note |
|---|---|---|
| Quick-action groups (Research / Contracts / Privilege / Operations) | Keep | Recategorize actions to your actual recurring tasks |
| Metric tiles (counts) | Keep | Rescope metrics to: open deadlines, case documents, exhibits, audit entries |
| Recent activity feed | Keep | |
| "Ask the Paralegal" primary CTA | Keep | |
| Live/mock backend status pill | Rescope | Not mock-vs-live — becomes a real **data-source connection status pill**: shows which of your actual knowledge bases/databases from the Agno MCP Platform (Postgres evidence store, Weaviate vector store, Neo4j graph, etc.) are currently reachable. Same UI real estate, different meaning: "connected to your real data" instead of "demo fixtures vs. a live server toggle." |
| "Matter workbench for high-volume legal teams" framing/copy | Rescope | Same panel, personal-litigant copy instead of firm marketing copy |

## CHAT (Paralegal)

| Feature | Decision | Note |
|---|---|---|
| Structured attachment cards (case/statute/citation/contract/brief) w/ deep links | Keep | |
| Suggested-prompt empty state | Keep | Rescope prompt examples to custody-case scenarios |
| Thinking indicator | Keep | |
| Auto-growing composer, Enter/Shift+Enter | Keep | |
| Persistent conversation store + "New conversation" | Keep | |
| Disclaimer footer | Keep | |
| Rule-based keyword-matching "intent router" | Rescope | Replace with a real agent/LLM call — legal-terminal's version is a regex fake for demo purposes; you'll have an actual backend |
| Artificial 500-1000ms response delay | Exclude | Was purely to make the fake demo feel real; a real backend has real latency |

## PREC — Precedent & Case Search

| Feature | Decision | Note |
|---|---|---|
| Master-detail search layout | Keep | |
| Relevance bar | Keep | |
| Empty states w/ example queries | Keep | Rescope examples to Michigan family-law issues |
| "Cites" cross-link list | Keep | |
| Deep links to Citation Console / Brief Builder | Keep | |

## STAT — Statute Viewer

| Feature | Decision | Note |
|---|---|---|
| Free-text lookup + quick-link buttons | Keep | Rescope quick links to Michigan Child Custody Act, MCL 722.23 factors, FOC-relevant statutes |
| Full statute display (text, history, topics) | Keep | |

## CITE — Citation Console

| Feature | Decision | Note |
|---|---|---|
| Validate / Normalize / Verify-integrity tabs | Keep | |
| Verdict banner + parsed-components table | Keep | |
| Session history sidebar | Keep | |

## CTRX — Contract/Agreement Workbench

| Feature | Decision | Note |
|---|---|---|
| Analyze / Compare / Negotiate tabs | Keep | **Negotiate tab is high-value** — directly applicable to custody settlement negotiation |
| Per-clause risk list + alternative-language suggestions | Keep | |
| Missing-clause/provision detection | Keep | Rescope detection logic from NDA/MSA clauses to parenting-plan provisions (holiday schedule, decision-making authority, relocation clause, etc.) |
| Fixture-library chip picker (browse many templates) | Rescope | Shrink to a short, fixed list of *your* actual case documents, not a browsable template library |
| Contract banner (title/parties/governing law/term) | Rescope | Relabel fields for orders/agreements (parties, court, entry date, term) instead of commercial-contract fields |

## DOCA — Document Analyzer

| Feature | Decision | Note |
|---|---|---|
| Upload zone (drag/drop, folder picker) | Keep | |
| Uploaded-file chips, auto-analyze on upload | Keep | |
| Queue for background analysis | Keep | |
| Export risk report | Keep | |
| Metadata table (parties, dates, governing law, liability cap, payment terms) | Rescope | Replace commercial-contract fields with custody-relevant metadata: parties, filing date, order type, custody/parenting-time terms, next-review date |
| Clause risk list | Rescope | Repurpose risk heuristics for custody-order/parenting-plan language instead of commercial clauses |

## PRIV — Privilege Risk Check

| Feature | Decision | Note |
|---|---|---|
| Assess-all-providers batch check | Keep | Called out earlier as your single highest-value feature |
| Provider comparison table (risk/ZDR/no-train/HIPAA/verdict) | Keep | |
| Click-to-assess single provider | Keep | |
| Detected privilege-indicator list | Keep | |
| *Heppner* / ABA Rule 1.6 grounding | Keep | |
| 6-provider default list (Ollama, Azure, Vertex, OpenAI, Anthropic, OpenRouter) | Rescope | Trim/reorder to the providers you actually use (Claude, GPT, local Ollama on your VPS) rather than all 6 |

## BRF — Brief/Motion Builder

| Feature | Decision | Note |
|---|---|---|
| IRAC argument structure + issue-statement card | Keep | |
| Case-type chip selector (contract dispute, employment law, etc.) | Rescope | Replace with your actual motion types: motion to change custody, parenting-time enforcement, FOC objection, motion for temporary orders |

## JOBS — Analysis Queue

| Feature | Decision | Note |
|---|---|---|
| Upload + manual queue entry | Keep | |
| Auto-refreshing table + status strip | Keep | |
| Job detail side panel | Keep | |

## WKFL — Workflows & Builder

| Feature | Decision | Note |
|---|---|---|
| Browse system playbooks w/ tool-call sequence detail | Keep | Rescope playbooks to your recurring processes (e.g. "opposing filing arrives," "prep for FOC hearing") |
| Run results panel | Keep | |
| Builder tab (custom tool-sequence editor) | Keep | High value — codify your own repeatable workflows |
| Mnemonic-to-panel routing table | Keep | Internal plumbing, carries over regardless of content |

## AUTM — Automations

| Feature | Decision | Note |
|---|---|---|
| Automation list w/ inline enable + last-run status | Keep | |
| 4 schedule types (daily/weekly/once/event) | Keep | |
| Real scheduler (tick + event bus) | Keep | |
| Event payload filters | Keep | |
| Run-now manual trigger | Keep | |

## TRIG — Triggers + Inbox

| Feature | Decision | Note |
|---|---|---|
| Inbox table w/ category badges + Process/Dismiss | Keep | Real use: opposing-counsel emails, FOC/court notices |
| Category rules engine (keyword/domain matching → automation) | Keep | |
| Category set: contract / privilege / litigation / general | Keep | |
| Category: HR | Exclude | No employment mail to triage in a custody matter |
| POP3 config, simulated only | Rescope | Needs to become a *real* IMAP/POP3 (or forwarding-rule) connection to your actual inbox — legal-terminal's version never actually connects to anything |
| "Simulate inbound" demo injector | Exclude | Demo-only affordance, no purpose once it's wired to a real inbox |

## AUDT — Audit Log

| Feature | Decision | Note |
|---|---|---|
| Category filter chips | Keep | |
| Full invocation log table | Keep | Valuable pro se record-keeping |

## LIVE — Integration Status

| Feature | Decision | Note |
|---|---|---|
| PACER billing advisory banner | Keep | Real money risk, keep visible |
| Per-integration cards (CourtListener, PACER) w/ enable instructions | Keep | |
| Raw server-config table (transport/port/enabled categories) | Rescope | Useful for you as the operator/developer, but move to a diagnostics/dev view rather than a primary nav panel |

## WTCH / Docket Watch → becomes a real Case Monitor

| Feature | Decision | Note |
|---|---|---|
| "Under consideration" banner + "I want this" vote button | Exclude | Vote button is meaningless with one user — you already know you want it |
| Product-pitch/explainer copy | Exclude | Marketing copy for a feature you're building for yourself, not selling |
| Interactive watchlist (add/remove/pause entities) | Rescope | Becomes the **real, always-on Case Monitor** (Bloomberg's actual Monitor concept) — not a mockup: a small fixed watch list (your case number, opposing party) with real nightly PACER/CourtListener checks |
| Alert channels (email, webhook, RSS-soon, in-app-soon) | Rescope | Fold into the Notifications tab in Settings rather than a separate feature list |
| Cost/requirements disclosure (PACER $0.10/page) | Keep | Real cost, keep visible |
| Bloomberg: color-coding + threshold alerts on watched items | Keep | Color-code by deadline proximity |
| Bloomberg: Launchpad (drag-drop customizable dashboard) | Rescope | Build a simpler fixed dashboard layout rather than a full drag-and-drop customization system — revisit later if you want it |

## CONF — Settings

| Feature | Decision | Note |
|---|---|---|
| Confidential Mode hero card + enforced-rules checklist | Keep | |
| AI-provider trust-level grid | Keep | (same provider trim as PRIV above) |
| General tab (default panel, refresh interval, theme, sidebar default) | Keep | |
| Notifications tab (email digest, webhook, per-category) | Keep | Absorbs Docket Watch's alert-channel settings |
| Integrations tab: "Live MCP server URL" field | Rescope | Same reasoning as the status pill above — becomes real connection config for the data sources/knowledge bases Legal OS pulls from (the sister platform's Postgres/Weaviate/Neo4j endpoints), not a mock↔live demo switch |
| Integrations tab: CourtListener token / PACER username fields | Keep | Real credentials you'll actually configure |
| "Settings saved" toast | Keep | |

## Bloomberg-only concepts not yet built anywhere

| Feature | Decision | Note |
|---|---|---|
| `DES`-style universal "canonical profile" screen per entity type | Keep | Every case, party, exhibit, witness gets one profile screen reached the same way |
| `COMP` generalized 2-3-way comparison | Keep | Generalize CTRX's compare beyond contracts (e.g., compare two draft filings, two case theories) |
| Worksheets (persistent, named, re-runnable data grids) | Keep | "Case worksheets" — a live deadlines table, a witness/contact table |
| `EQS`-style saved/reusable query builder | Rescope | Saved research queries or saved document filters, simplified vs. Bloomberg's full screener |
| Historical/time-series view (`HP`/`GP`) | Rescope | A timeline view of the case or of a negotiation position over time |
| Excel/API export (`XLTP`/`DAPI`/`FLDS`) | Rescope | Simple CSV/spreadsheet export, not a full formula-driven API |
| News feed with saved topic feeds | Rescope | Overlaps with the Case Monitor above — one feature, not two |

## legal-mcp Toolset (27 tools, 8 categories)

| Category | Decision | Note |
|---|---|---|
| Research (`search_precedents`, `search_case_law`, `extract_statute`, `research_legal_issue`) | Keep | Retarget content focus to Michigan family law |
| Citation (`validate_citation`, `normalize_citation`, `check_demo_database`) | Keep | |
| Contract (`compare_contracts`, `analyze_clauses`, `extract_clauses`, `suggest_clause_alternatives`, `generate_negotiation_guide`, `deep_analyze_clause`) | Keep | Retarget clause library to custody/parenting-plan provisions |
| Document (`analyze_document`, `compare_documents`, `export_analysis_report`, `extract_contract_metadata`) | Keep | Retarget metadata schema (see DOCA above) |
| Privilege (`check_privilege_risk`) | Keep | |
| Brief (`generate_brief_outline`, `create_argument_structure`, `generate_issue_statement`) | Keep | Retarget case-type list to family-court motions |
| Analysis Queue (4 tools) | Keep | |
| Integrations (`integration_status`, `search_live_case_law`) | Keep | |
| Demo mode + 13 bundled commercial contract templates (NDA, MSA, HIPAA BAA, etc.) | Exclude | Not your content — useful only as a parity-testing fixture set during development, not user-facing |
| Feature-flag system (`LEGAL_MCP_ENABLE_*` per category) | Keep | Still useful even solo — lets you disable categories you don't use |

---

Once you've reacted to this (agree / flip anything), this becomes the actual scope for the Legal OS feature spec — and the input for the overlap matrix once the sister repo (Agno MCP Platform) and your local custom repos are folded in.
