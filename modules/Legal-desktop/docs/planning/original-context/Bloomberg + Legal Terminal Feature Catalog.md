# Feature Catalog: Real Bloomberg Terminal + legal-terminal Widgets

Purpose: a pick-list. Part 1 covers the actual Bloomberg Terminal (the product legal-terminal says it's inspired by) so you can borrow ideas legal-terminal *didn't* implement. Part 2 is a granular, code-level breakdown of every interactive element in legal-terminal, panel by panel, pulled directly from its source (not just its README). Mark what you want; everything gets rebuilt from scratch in Legal OS — no code copied, per your last message.

---

## Scope Note: Single-User, Single-Case (Self-Represented)

This catalog was built from software designed for multi-client law firms (legal-terminal's own copy calls it a "matter workbench for **high-volume legal teams**"; Bloomberg is built for trading desks with many securities and many users). Legal OS is for one person, one active high-conflict custody matter, self-represented. Below is my honest pass at what that scope difference actually changes — I looked for firm-scale bloat (client intake, billing, multi-matter management, staff permissions) specifically so you wouldn't have to.

**What I did NOT find in legal-terminal or legal-mcp (the two repos analyzed so far):** client intake forms, time tracking/billing, invoicing, multi-matter/case list management, or staff role/permissions. Neither repo actually has this — they're narrowly research/contract/citation/privilege tools plus a UI shell, already close to matter-agnostic. So there was very little firm-scale cruft to cut from *these two* sources. The bulk of any real risk on this front is more likely to live in `custodyguide_v1complete` and the other local archives you haven't handed me yet — I'll apply this same single-case filter explicitly when those get reviewed and call out anything I cut there too.

**What I did cut or rescope from this catalog, and why:**

| Item | Where | Cut / rescope | Why |
|---|---|---|---|
| `HR` trigger category | TRIG rules (contract/privilege/litigation/**hr**/general) | Removed | Inbox rule category for employment-related mail — no reason to triage HR mail in a custody case. If you disagree (e.g. you want a category for something else entirely, like "school/medical records"), easy to swap in. |
| "High-volume legal teams" framing / multi-client contract "library" browsing UX | HOME hero copy, CTRX's chip-picker-over-many-templates pattern | Rescoped, not removed | The *feature* (view/compare/negotiate a contract) stays — you'll have real documents (settlement proposals, parenting-time agreements, prior orders). What changes is the UX assumption: instead of a browsable library of many clients' many templates, it's a short, fixed list of *your* case documents. Same tool, smaller shelf. |
| Docket Watch as an open-ended, grow-forever watchlist product | WTCH panel design intent | Rescoped, not removed | Bloomberg's Monitor/Launchpad concept and legal-terminal's Docket Watch pitch are built to track arbitrarily many companies/people for many clients. For you it's a fixed, small watch list — your case number, the opposing party, maybe your own name — not a general product feature to build out. |
| Instant Bloomberg (chat/messaging network) | Bloomberg Part 1 | Already flagged not applicable | Built for cross-institution collaboration between many users; you're the only user. Noted only as a pattern in case you ever add a paralegal or co-counsel later. |
| Multi-monitor **Panel key** routing across a 4-screen trading desk | Bloomberg Part 1 | Kept as optional, low-priority | Still plausible if you run this on a multi-monitor setup, but it's a nice-to-have, not core — deprioritized rather than cut. |

**What I deliberately did NOT cut**, because it's still valuable solo, not just at firm scale:
- AI-provider privilege/trust grid (CONF + PRIV) — arguably *more* important for you than a firm, since you don't have an ethics/IT department vetting AI tools for you.
- Workflow Builder + Automations — a firm-scale idea (codified playbooks) that's just as useful for one person doing the same category of task repeatedly (e.g., every time a filing arrives from opposing counsel).
- Audit Log — valuable pro se record-keeping regardless of scale.
- PACER/CourtListener integration and the PACER fee-risk warnings — the fee-burn risk is *per account*, not per client, so it applies just as much to you.

If you disagree with any cut above, say so and I'll put it back — nothing here is destructive, this file is just a planning artifact.

---

## Part 1 — The Real Bloomberg Terminal (the actual inspiration)

Bloomberg Terminal is a 40+ year old financial data/trading system. Its UX philosophy — not its finance content — is what legal-terminal borrowed. Sourced from [Bloomberg's own product page](https://professional.bloomberg.com/products/bloomberg-terminal/), [Wikipedia](https://en.wikipedia.org/wiki/Bloomberg_Terminal), [Investopedia](https://www.investopedia.com/articles/professionaleducation/11/bloomberg-terminal.asp), [CFI's function/shortcut list](https://corporatefinanceinstitute.com/resources/equities/bloomberg-functions-shortcuts-list/), and several university library guides ([Imperial College](https://library-guides.imperial.ac.uk/bloomberg/getting-started), [NYU Law](https://nyulaw.libguides.com/c.php?g=1342741&p=9945524), [University of Delaware](https://my.lerner.udel.edu/wp-content/uploads/BB-Getting-Started-in-Launchpad.pdf)).

### Core interaction model
- [ ] **Function-code command bar with `<GO>`** — type a short mnemonic (e.g. `DES`, `GP`, `HP`) then hit a dedicated GO key to execute; the same pattern legal-terminal copies with `PREC`, `CITE`, etc. ([CFI](https://corporatefinanceinstitute.com/resources/equities/bloomberg-functions-shortcuts-list/))
- [ ] **Keyword fallback search** — if you don't know the mnemonic, type a plain-language keyword into the command line and it resolves to the right function ([Fin-CA library guide](https://fin-ca.libguides.com/c.php?g=727352&p=5215645)) — legal-terminal has this partly via the Ctrl+K palette but not from the raw command bar.
- [ ] **Dedicated colored hardware action keys** — HELP (green, context help), MENU (green, back up the function hierarchy), SEARCH (green, full-database keyword search), CANCEL (red, abort/exit) ([Imperial College guide](https://library-guides.imperial.ac.uk/bloomberg/getting-started)). Software analogue for Legal OS: persistent Help/Back/Cancel affordances in every module, not just F-keys.
- [ ] **Market Sector keys** (yellow) — one-touch jump into a domain context (govt bonds, equities, currencies, etc.) ([Investopedia](https://www.investopedia.com/articles/professionaleducation/11/bloomberg-terminal.asp)). Legal analogue: one-touch jump into a *case phase* context (Discovery, Motions, Hearing Prep, Trial) that reconfigures which panels/shortcuts are surfaced.
- [ ] **`LAST` command** — instantly review/re-run your last several functions ([NYU guide](https://guides.nyu.edu/bloombergguide/popular-commands)) — legal-terminal has command history in the palette but no single "repeat last" shortcut.
- [ ] **`GRAB`** — capture the current screen and email it as an image attachment directly from any function ([NYU guide](https://guides.nyu.edu/bloombergguide/popular-commands)). Legal analogue: one-key "export this panel" (screenshot or PDF) from anywhere, not just DOCA's export button.

### Launchpad (customizable multi-monitor dashboard) — legal-terminal does NOT have this
- [ ] **Launchpad** — a fully customizable desktop of drag-and-drop "components" (monitors, charts, news feeds, TV) arranged into a personal dashboard layout, saved and reloaded ([Bloomberg product page](https://professional.bloomberg.com/products/bloomberg-terminal/); [UConn Launchpad guide](https://finance-business.media.uconn.edu/wp-content/uploads/sites/723/2016/10/Bloomberg-Launch-Pad.pdf)).
- [ ] **Monitor** — a live watchlist of tracked entities (securities) that stays open and auto-updates ([Lippincott Library — "Monitor: A watch list of securities"](https://lippincottlibrary.wordpress.com/2013/03/11/bloomberg-launchpad-part-one/)). This is the general concept behind what legal-terminal's Docket Watch *pitches* but never builds — worth having as a first-class, always-on dashboard component in Legal OS from day one (e.g. a live monitor of opposing-party filings, upcoming deadlines, discovery responses due) instead of a "Soon" placeholder.
- [ ] **Color-coding + price/threshold alerts on individual watched items** ([UD Launchpad guide](https://my.lerner.udel.edu/wp-content/uploads/BB-Getting-Started-in-Launchpad.pdf)) — legal analogue: color-code case items by urgency/deadline proximity, with threshold alerts ("7 days until response due").
- [ ] **Panel key / multi-monitor routing** — send a function to a specific physical monitor in a 2-4 screen desk setup ([Imperial College guide](https://library-guides.imperial.ac.uk/bloomberg/getting-started)). Legal analogue: pop a panel out to a second window/monitor (beyond legal-terminal's single in-app split view).

### Messaging & collaboration — not applicable for single-user, noting for completeness
- [ ] **Instant Bloomberg (IB)** — a closed, cross-institution real-time chat/messaging network built into the terminal ([Bloomberg's own IB/Worksheets/Launchpad guide](https://www.bloomberg.com/professional/insights/technology/bloomberg-terminal-essentials-ib-worksheets-launchpad/)). Not relevant for a single-user Legal OS, but if you ever add co-counsel or a paralegal, worth remembering as a pattern (in-app threaded messaging tied to specific case items).

### Research/data & workflow patterns (translate the *shape*, not the finance content)
- [ ] **Worksheets** — persistent, named, re-runnable data grids you build once and refresh anytime ([Bloomberg IB/Worksheets guide](https://www.bloomberg.com/professional/insights/technology/bloomberg-terminal-essentials-ib-worksheets-launchpad/)). Legal analogue: a saved, re-runnable "case worksheet" (e.g., a live table of every deadline + status, or every witness + contact info + last-updated).
- [ ] **`DES` (Description) as a universal "give me the profile" pattern** — every entity type gets a canonical summary screen ([multiple library guides](https://libguides.hkust.edu.hk/c.php?g=208028&p=6948362)). Legal analogue: every case, party, exhibit, or witness gets one canonical "profile" screen, reachable the same way regardless of type.
- [ ] **`COMP` — compare 2-3 things side by side** ([UD Lerner functions list](https://lerner.udel.edu/seeing-opportunity/bloomberg-functions-list/)) — legal-terminal already does this for contracts (`CTRX compare`); Bloomberg's pattern generalizes it to *any* entity type (compare two witnesses' statements, two draft filings, two case theories).
- [ ] **`EQS`-style saved screening/query builder** — build and save a reusable filter/query, not just one-off searches ([Fin-CA guide](https://fin-ca.libguides.com/c.php?g=727352&p=5215645)) — legal analogue: saved research queries or saved document filters you can re-run.
- [ ] **Historical/time-series views (`HP`, `GP`)** — everything has a "how did this change over time" view ([Wall Street Oasis cheat sheet](https://www.wallstreetoasis.com/resources/data/bloomberg/bloomberg-functions-shortcuts-list)). Legal analogue: a timeline view of a case, exhibit, or negotiation position over time — not just current-state.
- [ ] **Excel/API integration (`XLTP`, `DAPI`, `FLDS`)** — pull terminal data into spreadsheets via formulas ([Fin-CA guide](https://fin-ca.libguides.com/c.php?g=727352&p=5215645)). Legal analogue: export any table (deadlines, exhibits, timeline) to a spreadsheet or a documented local API.
- [ ] **News aggregation with saved topic feeds (`N`, `TOP`, `READ`)** ([Columbia guide](https://guides.library.columbia.edu/bloomberg/basic)) — legal analogue: a saved feed of new filings/docket activity for your matter (this overlaps with Docket Watch — Bloomberg's version is just always-on and not a "Soon" feature).

---

## Part 2 — legal-terminal: Every Widget, Control, and Function (from source, not just docs)

Legend: `[panel/file]` — direct source reference for verification.

### Global shell
- [ ] Hybrid collapsible sidebar with 5 workflow groups (Assistant, Research, Contracts, Drafting, Operations), icon-only collapsed state, group labels hidden when collapsed `[Sidebar.tsx]`
- [ ] Sidebar footer: light/dark theme toggle (persisted to `localStorage`) `[Sidebar.tsx, terminalStore.ts]`
- [ ] Sidebar footer: Confidential Mode toggle with amber "on" visual state + inline "Settings" deep-link `[Sidebar.tsx]`
- [ ] Version string shown in sidebar footer (`v0.2.0-pre.3`) `[Sidebar.tsx]`
- [ ] Mobile responsive mode: sidebar becomes an overlay with backdrop-click-to-close, hamburger menu button appears in command row `[Sidebar.tsx, CommandLine.tsx]`
- [ ] Command bar: typed mnemonic + argument parsing (`PREC breach of contract` → navigates to PREC with query pre-filled) `[CommandLine.tsx]`
- [ ] Command bar: live autocomplete dropdown filtered by mnemonic prefix OR description substring match `[CommandLine.tsx]`
- [ ] Command bar: Tab-completes to the top suggestion; Escape clears; Enter executes `[CommandLine.tsx]`
- [ ] Command bar: 16 registered mnemonics, each mapped to a specific backing MCP tool name shown in the suggestion row (self-documenting) `[CommandLine.tsx]`
- [ ] Ctrl+K palette (via `cmdk`): separate "Recent" group (last 5 commands, replayable) and "Modules" group (searchable by mnemonic, description, or tool name) `[CommandPalette.tsx]`

### HOME
- [ ] 4 quick-action groups (Research / Contracts / Privilege & Risk / Operations), each a clickable pre-filled command shortcut `[HomePanel.tsx]`
- [ ] Live/mock runtime status pill + "Command driven" badge `[HomePanel.tsx]`
- [ ] 4 metric tiles: Authority Library count, Contract Files count, Analysis Queue count, Audit Trail count `[HomePanel.tsx]`
- [ ] Recent activity table (last 10 of 30 tracked events) with relative "time ago" formatting `[HomePanel.tsx]`
- [ ] "Ask the Paralegal" primary CTA with F1 badge `[HomePanel.tsx]`

### CHAT — Paralegal
- [ ] Rule-based intent router (not a real LLM) that pattern-matches: greetings, citation regex detection, brief/motion keywords, contract-related keywords (with a "is this a question, not a doc-review request" guard), privilege/AI-provider keywords (auto-detects which AI provider was named and infers a plausible file), else falls through to general research `[ChatPanel.tsx]`
- [ ] 5 distinct structured attachment card types rendered inline in chat: case cards, statute cards, citation card, contract list cards, brief outline card — each clickable to deep-link into the matching module with context pre-loaded `[ChatPanel.tsx]`
- [ ] 4 suggested-prompt starter cards on empty state, each with an icon and example query `[ChatPanel.tsx]`
- [ ] Animated 3-dot "thinking" indicator with staggered animation delays + artificial 500-1000ms latency before responding (so the indicator is visible even against instant mock data) `[ChatPanel.tsx]`
- [ ] Composer: auto-growing textarea (1-4 rows), Enter-to-send / Shift+Enter-for-newline, disabled send button until non-empty `[ChatPanel.tsx]`
- [ ] Persistent conversation state in its own store (survives navigating to other panels and back) + explicit "New conversation" clear action `[ChatPanel.tsx, chatStore]`
- [ ] Standing disclaimer footer under the composer ("Verify all authority before filing") `[ChatPanel.tsx]`

### PREC — Precedent & Case Search
- [ ] Master-detail layout: result list (numbered, with relevance bar) + full detail pane `[PrecPanel.tsx]`
- [ ] Per-result relevance visualization (`RelevanceBar`) driven by a numeric `relevance_score` `[PrecPanel.tsx]`
- [ ] Zero-result vs. never-searched empty states are visually distinct, with the latter showing clickable example queries `[PrecPanel.tsx]`
- [ ] Detail pane: holding (blockquote-styled), summary, topic tag chips, and a "Cites" list of clickable related-case links that re-run the search `[PrecPanel.tsx]`
- [ ] Cross-module deep links from detail pane: "Validate citation →" (to CITE) and "Brief outline →" (to BRF), both pre-filled `[PrecPanel.tsx]`

### STAT — Statute Viewer
- [ ] Free-text lookup by section ID or citation fragment, plus a row of one-click "quick link" buttons for common statutes `[StatPanel.tsx]`
- [ ] Full statute display: title, citation, jurisdiction, enacted date, last-amended date, full text block, legislative history narrative, topic tags `[StatPanel.tsx]`
- [ ] Explicit not-found state distinct from the loading and empty states `[StatPanel.tsx]`

### CITE — Citation Console
- [ ] 3 modes as tabs: Validate / Normalize / Verify integrity, each calling a distinct backend tool `[CitePanel.tsx]`
- [ ] Verdict banner (Valid/Invalid pill) + integrity status line (verified / not-found / mismatch, each its own color and icon glyph) `[CitePanel.tsx]`
- [ ] Parsed-components table: input, normalized form, reporter code, reporter full name `[CitePanel.tsx]`
- [ ] Issues list (only rendered when present) `[CitePanel.tsx]`
- [ ] Session history sidebar (up to 20 entries) — clicking a past entry reruns it and re-promotes it to the top `[CitePanel.tsx]`

### CTRX — Contract Workbench
- [ ] Fixture-library chip picker showing type, risk-level color dot, and label per contract, with an active "Viewing" indicator `[CtrxPanel.tsx]`
- [ ] Contract banner: title, type tag, risk badge, all party names joined, governing law, term — shown persistently above the tab content `[CtrxPanel.tsx]`
- [ ] 3-tab workspace: **Analyze** (clause list sorted by risk severity + missing-clause callouts + selected-clause detail with suggested alternative language), **Compare** (pick any two contracts from dropdowns, run a keyed diff with a change description per differing field), **Negotiate** (choose buyer/seller/mutual role, generate a per-clause accept/negotiate/reject recommendation with color-coded position label, rationale, and fallback language for anything not "accept") `[CtrxPanel.tsx]`
- [ ] Missing-clause detection surfaced in both Analyze and Negotiate views `[CtrxPanel.tsx]`
- [ ] Auto-loads alternative phrasing suggestions whenever a new clause is selected `[CtrxPanel.tsx]`

### DOCA — Document Analyzer
- [ ] Real file upload zone (drag/drop, multi-browse, folder picker) plus a curated "example files" quick-pick row and a manual filename entry field `[DocaPanel.tsx]`
- [ ] Uploaded files render as removable chips; clicking a chip re-triggers analysis; uploading auto-analyzes the first new file `[DocaPanel.tsx]`
- [ ] Results: risk badge + filename, a full metadata table (parties, dates, governing law, term, liability cap, payment terms — with an explicit "null — confirm manually" fallback for missing fields), and a clause-by-clause risk list `[DocaPanel.tsx]`
- [ ] "Queue for background analysis" as an alternate action to immediate analysis `[DocaPanel.tsx]`
- [ ] Export risk report button (stub in this repo; backed by `export_analysis_report` in legal-mcp) `[DocaPanel.tsx]`

### PRIV — Privilege Risk Check
- [ ] "Assess all providers" one-click batch check across every configured AI provider, run in parallel `[PrivPanel.tsx]`
- [ ] Full comparison table: provider, computed risk level, ZDR (zero data retention) yes/no, "no training" yes/no, HIPAA-eligible yes/no, and a plain-language verdict sentence per provider `[PrivPanel.tsx]`
- [ ] Click-to-assess a single unassessed provider directly from its table row `[PrivPanel.tsx]`
- [ ] Detected privilege indicators list (e.g. specific phrases/markers found in the doc) shown below the table, sourced from the highest-risk result `[PrivPanel.tsx]`
- [ ] Explicit legal citation baked into the empty state and results (*US v. Heppner*, ABA Model Rule 1.6) `[PrivPanel.tsx]`

### BRF — Brief Builder
- [ ] Case-type chip selector + optional free-text "key facts" field `[BrfPanel.tsx]`
- [ ] Generated output: numbered section outline, a structured 4-part IRAC block (Issue/Rule/Analysis/Conclusion) rendered as its own card, and a separate italicized issue-statement card `[BrfPanel.tsx]`

### JOBS — Analysis Queue
- [ ] Upload zone + manual filename entry, both routed to the same "queue" action `[JobsPanel.tsx]`
- [ ] Auto-refreshing table (polls every 2 seconds) with a manual refresh button `[JobsPanel.tsx]`
- [ ] Status summary strip showing live counts for complete/processing/queued/error `[JobsPanel.tsx]`
- [ ] Full job table: ID, file (truncated with ellipsis), status chip, queued time, completed time, risk level, flag count `[JobsPanel.tsx]`
- [ ] Row-click opens a side detail panel with every job field plus an error message row when applicable `[JobsPanel.tsx]`

### WKFL — Workflows & Builder
- [ ] **Browse tab:** list of system playbooks with mnemonic badges; detail pane shows the full numbered tool-call sequence (tool name + description per step); "Run workflow" only appears when the playbook has executable steps, otherwise a "Open {MODULE} →" deep-link is shown instead `[WkflPanel.tsx]`
- [ ] Run results panel shows per-step success/failure coloring and a summary line per tool call `[WkflPanel.tsx]`
- [ ] **Builder tab:** create/select custom workflows; per-step tool picker (drawn from a live tool catalog) with dynamically-rendered parameter inputs based on each tool's declared param type (plain text, `select` with options, or a special `contract` picker reusing the same contract fixture list as CTRX); add/remove steps freely; Save, Test run, and Delete actions `[WkflPanel.tsx]`
- [ ] Mnemonic-to-panel routing table so a playbook's target mnemonic (e.g. `NEGO`, `META`) resolves to the correct concrete panel (`CTRX`, `DOCA`) even when they don't match 1:1 `[WkflPanel.tsx]`

### AUTM — Automations
- [ ] Automation list with inline enable/disable checkbox (doesn't require opening the detail view) and last-run status chip `[AutmPanel.tsx]`
- [ ] Editor: name, workflow source (system playbook vs. custom workflow) with the workflow dropdown re-populating accordingly, and a non-executable warning when the selected workflow has no runnable steps `[AutmPanel.tsx]`
- [ ] 4 schedule types with type-specific fields: **Daily** (time picker), **Weekly** (day-of-week + time), **Once** (datetime picker), **Event** (dropdown of 5 event types: job_complete, document_upload, contract_selected, email_received, app_open) `[AutmPanel.tsx]`
- [ ] Real client-side scheduler: computes next-run timestamps per schedule type, ticks every 30 seconds, and additionally listens on an event bus for the 4 event triggers plus fires an `app_open` event automatically 500ms after load `[automationScheduler.ts]`
- [ ] Event automations support optional payload filters (filename substring filter, category-ID filter) so they only fire for matching events `[automationScheduler.ts]`
- [ ] "Run now" manual trigger independent of the schedule, with its own result panel `[AutmPanel.tsx]`

### TRIG — Triggers + Paralegal Inbox
- [ ] **Inbox tab:** simulated inbound message table (from, subject, category badge, attachment list, status chip) with per-row Process/Dismiss actions and a "Paralegal" deep-link per message; "Simulate inbound" supports picking a specific seed message or a random one `[TrigPanel.tsx]`
- [ ] **POP3 Config tab:** address, display name, host, port, username, password (masked), TLS checkbox, "enable listening (simulated)" checkbox, Save + Test connection actions with a transient status message `[TrigPanel.tsx]`
- [ ] **Category Rules tab:** category list with live rule counts; rule editor with subject-keyword list (comma-separated), from-domain list, linked automation (or "prompt only"), a templated agent-prompt field supporting `{{subject}} {{from}} {{category}} {{filename}} {{attachmentCount}}` tokens, and an "auto-run in Paralegal" checkbox `[TrigPanel.tsx]`
- [ ] Real categorization engine: keyword/domain-based auto-categorization into contract/privilege/litigation/hr/general on message arrival `[triggerRouter.ts]`
- [ ] Real rule-matching engine: matches by category + optional from-domain suffix + optional subject-keyword substring + optional minimum-attachment-count, first match wins, then runs the linked automation or workflow and optionally seeds a Paralegal chat prompt from the rendered template `[triggerRouter.ts]`

### AUDT — Audit Log
- [ ] Category filter chips generated dynamically from whatever categories are present in the log (search/analysis/validation/review/metadata/system/contract/privilege), each with its own color dot `[AudtPanel.tsx]`
- [ ] Full log table: timestamp, tool name (with category color dot), user, truncated JSON input preview, success/error status, duration in ms `[AudtPanel.tsx]`

### LIVE — Integration Status
- [ ] Standing PACER billing-advisory banner citing the $30/quarter fee-waiver risk `[LivePanel.tsx]`
- [ ] Server config table: transport type, port, enabled tool categories as chips `[LivePanel.tsx]`
- [ ] Per-integration cards (CourtListener, PACER) each showing enabled/disabled state, the exact env var needed to enable it, and (for PACER) the full block of required env vars `[LivePanel.tsx]`

### WTCH — Docket Watch (preview only, explicitly "under consideration")
- [ ] "Under consideration" banner with an "▲ I want this" vote button (local-only, not persisted to a backend) `[DktwPanel.tsx]`
- [ ] Product-pitch section: 3-step "how it works" explainer (add entity → nightly PACER scan → alert) with icons `[DktwPanel.tsx]`
- [ ] Alert-channel chips: Email + Webhook marked active, RSS + In-app marked "SOON" `[DktwPanel.tsx]`
- [ ] Fully interactive mock watchlist: add entity (name + type: company/person/matter#), remove, and toggle active/paused — table shows added date, next-check time, new-case count, status `[DktwPanel.tsx]`
- [ ] Cost/requirements disclosure footer (PACER credentials + $0.10/page cost) `[DktwPanel.tsx]`

### CONF — Settings
- [ ] **Privacy tab:** hero Confidential Mode card with lock/unlock icon and one-click enable/disable; a checklist of 6 enforced rules when active (e.g. "all inference routed through local Ollama endpoint only," "ABA Model Rule 1.6 confidentiality obligations enforced"); editable Ollama endpoint + model fields; a 6-provider AI trust-level grid (Ollama, Azure OpenAI, Google Vertex AI, OpenAI API, Anthropic API, OpenRouter) each showing risk tier, ZDR support, no-train support, HIPAA eligibility, and a verdict sentence, with non-local providers visually dimmed while Confidential Mode is active `[ConfPanel.tsx]`
- [ ] **General tab:** default startup panel picker, analysis-queue refresh interval (2s/5s/10s/manual), light/dark theme selector, "start with sidebar collapsed" checkbox `[ConfPanel.tsx]`
- [ ] **Integrations tab:** live MCP server URL, CourtListener API token (masked), PACER username fields `[ConfPanel.tsx]`
- [ ] **Notifications tab:** email digest toggle, webhook URL field, and a per-category (contract/privilege/litigation/hr/general) notification checkbox grid `[ConfPanel.tsx]`
- [ ] Transient "Settings saved" toast on every save action `[ConfPanel.tsx]`

---

## What I'd flag as worth prioritizing

If you want my read on which items punch above their weight for a **single-user litigation** Legal OS specifically (not just "matches Bloomberg" or "matches legal-terminal"):

- The **AI-provider privilege/trust-level grid** (CONF + PRIV) is the single most valuable pattern in either source — it's a genuinely useful, non-generic feature for your actual situation.
- **Triggers + rule engine** (TRIG) generalizes well beyond email — the categorize → match-rule → run-automation → seed-a-prompt pipeline is a solid backbone for "something happened, do something automatically" across your whole case.
- **Workflow Builder** (WKFL) — a user-editable tool-call sequence builder — is a strong pattern for codifying your own repeatable litigation playbooks (e.g., "new opposing filing arrives → do X, Y, Z").
- Bloomberg's actual **Monitor/Launchpad** concept (Part 1) is arguably a better foundation for your case-dashboard than legal-terminal's "Soon" Docket Watch mockup — worth building the *real* always-on watchlist concept from day one rather than copying legal-terminal's placeholder-only version.
- **Audit Log** (AUDT) is cheap to build and valuable given you're self-representing — a full local record of every research/analysis action taken on your matter.

Let me know which of these (from either part) you want carried into the feature list, and I'll fold your picks into the master feature spec once the local repos are in.
