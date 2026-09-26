# Current Advocatio source reconciliation — 2026-09-13

This is a static current-source audit for planning. It describes implementations observed on disk, not proven runtime behavior. No application, database, model, browser session, service, or test suite was started. Only this directory's new planning outputs were written. Existing concurrent edits were preserved.

## Scope and provenance

Canonical Git root verified: `E:/AI_Workspace/Projects/Propria/Legal-desktop`. Read current root AGENTS.md, AGENT_MEMORY.md and web/AGENTS.md. Applied Smart Explore: installed compatibility entrypoint resolved canonical Search skill with expected SHA-256 `ED5AB3E3176052AA8596FEA32599E94EAD2C339121541DB090E347B4690876D4`; structural search found draft symbols, outline mapped TerminalShell and unfold showed Workspace.update_draft. Followed with bounded file/line inspection. Search may maintain its external code index; application state was not accessed.

Inspected current `api/legal_workspace`, `web/src`, test sources, `web/smoke`, dependency manifests and relevant routing/config declarations. Excluded private data, runtime databases and donor/build-kit code. “Not found” below means not found in that searched current-code scope, not globally absent. A registry memory search supplied child-repository/search-scoping guidance only; substantive findings were freshly verified in source.

Dirty state: the scoped `api tests web config sql` status had **46 entries**, including existing API/config/auth, web pages/forms/shell/API-client and tests changes plus untracked auth/BFF/proxy/vendor/smoke items. Root guidance/manifests and reconciliation resources were also already dirty/untracked. No staging, cleanup, source edits, or reversion occurred. These findings describe the current working tree, not a clean commit.

## Stack and actual persistence

- `web/package.json:12`: Next ^16.3.1, React/React DOM 19.2.3, AI SDK ^7.0.68, OpenAI/React AI adapters, cmdk and lucide. This is a Next App Router web app; source inspected here does not establish a packaged native desktop runtime.
- `pyproject.toml:7`: Python >=3.12,<4, FastAPI, Pydantic v2, SQLAlchemy 2, psycopg, HTTPX, APScheduler, python-docx, pikepdf and pypdf. Dependency ranges are declarations, not installed-version proof.
- `api/legal_workspace/services/workspace.py:194` passes a workspace directory into table/store operations. `api/legal_workspace/db/engine.py:29` chooses local SQLite when store_dir is supplied; database URL support in the engine alone is not proof this workspace path uses deployed PostgreSQL.
- `Workspace._write` at `services/workspace.py:221` saves via WorkspaceStore; JSON state/events are conditional debug mirrors. Root AGENTS still says every mutation writes JSON/JSONL and is stale on that point.
- **High-priority persistence gap:** `db/store.py:609` replaces dependent aggregate rows, `:645` commits their deletion before reconstruction, `:778` writes work-product version 1, and `:1045` emits aggregate version 1. A subsequent reconstruction failure can leave missing state despite the final session rollback. This is a source-derived risk, not an induced failure. No destructive code was executed.
- `tests/test_persist.py:22` defines a local reload scenario and expects no default JSON mirrors; `tests/test_work_product_versions.py:15` tests a pure version helper, not end-to-end persistence of an edit history. No tests were run in this audit.

## Capability coverage

### CA-01 — Reference and case-law snapshots: partial

Observed: Curated authority records contain proposition, pinpoint, source_path and snapshot_hash. Seed pins are packet labels, not observed content-addressed snapshot captures. Search returns results without saving. All citation tabs call the parser.

Evidence: `api/legal_workspace/domain/authority_library.py:15`; `api/legal_workspace/domain/authority_library.py:40`; `api/legal_workspace/api/source_routes.py:26`; `web/src/components/CitationParse.tsx:43`.

Gap candidate: Versioned source capture with URL/date/content hash and explicit currency/history review; avoid claiming Snapshot check captures a snapshot.

Existing test source: tests/test_authorities.py:10; tests/test_research.py:36. Reviewed as source or located by exact test symbol; not executed.

### CA-02 — Shared versioned resources and skills: not_found_in_searched_code

Observed: No skill/resource-version domain, route or execution binding found in api, web/src, config, tests searches. Templates and routed prompts are adjacent implementations.

Evidence: `api/legal_workspace/domain/templates.py:25`; `api/legal_workspace/services/workspace.py:456`; `api/legal_workspace/services/routing.py:47`.

Gap candidate: Shared resource/version manifest and pinned skill dependencies per run/draft, with applicability and update review.

Existing test source: tests/test_templates.py:25. Reviewed as source or located by exact test symbol; not executed.

### CA-03 — Skill-guided drafting and review: partial

Observed: Templates render outlines. Factor draft creation uses entered text and copied factor citations. Agent runs invoke the supplied prompt and model; no skill-guided drafting pipeline or review binding found.

Evidence: `api/legal_workspace/services/workspace.py:456`; `api/legal_workspace/services/workspace.py:470`; `api/legal_workspace/services/workspace.py:593`; `web/src/components/DraftEditor.tsx:23`.

Gap candidate: Drafting session tied to selected skill version, source/context scope, proposed output, independent review and owner decision.

Existing test source: tests/test_templates.py:25; tests/test_draft_edit.py. Reviewed as source or located by exact test symbol; not executed.

### CA-04 — Independent structure, citation, factual and substantive checks: partial

Observed: Citation gate validates package/assertion version and nonempty locator; support map labels every non-instruction paragraph supported when any section citations pass. It does not match paragraph claims to evidence. Authority validator exists, but release-candidate path calls factual validation only.

Evidence: `api/legal_workspace/services/citation_gate.py:25`; `api/legal_workspace/services/citation_gate.py:46`; `api/legal_workspace/domain/support_map.py:56`; `api/legal_workspace/api/citation_routes.py:25`; `api/legal_workspace/services/workspace.py:1071`.

Gap candidate: Separate version-pinned check records; claim-to-span entailment and contradiction review; legal proposition/authority review; independent substantive review with unresolved blockers.

Existing test source: tests/test_citation_gate.py:65; tests/test_support_map.py:11. Reviewed as source or located by exact test symbol; not executed.

### CA-05 — Accept/reject proposed versioned edits: partial

Observed: Owner may approve/reject/request changes on a section hash. Editing a section already in a release manifest forks a new ID; otherwise edits overwrite its text. No change-level proposal/diff accept-reject contract found. DB save reconstructs work-product version=1.

Evidence: `web/src/components/ReviewForm.tsx:80`; `api/legal_workspace/services/workspace.py:493`; `api/legal_workspace/services/workspace.py:1015`; `api/legal_workspace/domain/work_product.py:47`; `api/legal_workspace/db/store.py:778`.

Gap candidate: Immutable revision chain and proposal base-version/hash; per-change accept/reject and rationale; stale proposal detection and preserved rejected suggestions.

Existing test source: tests/test_work_product_versions.py:15; tests/test_review_release.py:108. Reviewed as source or located by exact test symbol; not executed.

### CA-06 — Original voice to court-language translator: not_found_in_searched_code

Observed: Original-voice, translator, translation and related searches found no dedicated paired-original/rewrite model, route or UI in current source. General notes, draft editor and assistant are adjacent capabilities.

Evidence: `web/src/components/DraftEditor.tsx:18`; `api/legal_workspace/domain/strategy.py:25`; `api/legal_workspace/services/workspace.py:593`.

Gap candidate: Keep original immutable, propose court-language wording beside it, show factual additions/omissions and require owner acceptance.

Existing test source: No specific test found in bounded test search. Reviewed as source or located by exact test symbol; not executed.

### CA-07 — Personal, relationship and case context plus strategy: partial

Observed: Typed private theories, strategy, directions, scratch drafts, ideas and chat extracts exist. Saved surface context and live UI context reach assistant. No dedicated personal/relationship history or provenance-scoped context record found.

Evidence: `api/legal_workspace/domain/strategy.py:16`; `api/legal_workspace/domain/strategy.py:25`; `api/legal_workspace/services/workspace.py:299`; `web/src/components/TerminalShell.tsx:471`.

Gap candidate: Structured private context with who/when/source/uncertainty and scope controls; preserve separation from accepted evidence.

Existing test source: tests/test_strategy.py; tests/test_surface_context.py:1. Reviewed as source or located by exact test symbol; not executed.

### CA-08 — Risk, anticipated accusation and red-team records: partial

Observed: Red-team lenses and findings support severity, claim, why and non-court-finding flag. Target references and prompts persist. No separately tracked anticipated accusation/risk with mitigation/evidence/disposition lifecycle found.

Evidence: `api/legal_workspace/domain/redteam.py:18`; `api/legal_workspace/domain/redteam.py:28`; `api/legal_workspace/api/main.py:383`.

Gap candidate: Risk/accusation register connected to issues, rebuttal evidence, missing proof, mitigation, ownership and resolution.

Existing test source: tests/test_redteam_todos.py:11. Reviewed as source or located by exact test symbol; not executed.

### CA-09 — Overall timeline versus docket/calendar: partial

Observed: Timeline and calendar use identical docket-events endpoint. Timeline explicitly says court events, not evidence-platform factual timeline. Missing-proof requests carry linked issue, factor, contradiction and assertion IDs.

Evidence: `web/src/app/timeline/page.tsx:22`; `web/src/app/calendar/page.tsx:38`; `api/legal_workspace/domain/investigation.py:41`; `api/legal_workspace/api/main.py:533`.

Gap candidate: A read-only projection of accepted overall factual timeline with claim evidence states and gap reports; keep court dates distinct and avoid a second evidence truth store.

Existing test source: tests/test_calendar.py; tests/test_investigation.py. Reviewed as source or located by exact test symbol; not executed.

### CA-10 — Accepted analysis imports distinct from facts: partial

Observed: Approved-source package imports are present; package item contract represents assertion references, not accepted analysis findings. Agent output is agent_hypothesis/private/nonexportable. Analysis queue displays scheduled jobs. No distinct accepted-analysis import contract found.

Evidence: `api/legal_workspace/contracts/source_package.py:22`; `api/legal_workspace/services/source_package.py:24`; `api/legal_workspace/domain/agents.py:60`; `web/src/app/analysis-queue/page.tsx:9`.

Gap candidate: Analysis package with originating run/model/source/version, human acceptance and epistemic class; acceptance must not establish its claims as facts.

Existing test source: tests/test_source_package.py; tests/test_agents.py:38. Reviewed as source or located by exact test symbol; not executed.

### CA-11 — Firm roles and provider routing: partial

Observed: Eight roles: intake, research, element_mapper, drafter, citation, redteam, discovery, filing_checker. Runtime table selects models/surfaces; forbidden actions and confidential provider restrictions are represented. Requested managing/associate/senior-reviewer/clerk/librarian/exhibit-clerk hierarchy is not implemented as that role model.

Evidence: `api/legal_workspace/domain/agents.py:18`; `api/legal_workspace/services/routing.py:47`; `api/legal_workspace/services/workspace.py:593`; `web/src/components/RoutingEditor.tsx`.

Gap candidate: Map requested firm roles to explicit responsibilities, run contracts, escalation and separate author/reviewer assignments; preserve user-controlled provider routing.

Existing test source: tests/test_agents.py:20; tests/test_routing.py; tests/test_confidential.py. Reviewed as source or located by exact test symbol; not executed.

### CA-12 — Exhibits and released versions: partial

Observed: Exhibit candidates/annotations and owner-triggered Bates assignment exist for approved source items. Release candidate records include hash and exclusions, owner hash approval gates, JSON/Markdown/DOCX exports; state remains release_candidate and filed=false.

Evidence: `api/legal_workspace/services/workspace.py:779`; `api/legal_workspace/api/main.py:512`; `api/legal_workspace/services/workspace.py:1058`; `api/legal_workspace/services/workspace.py:1129`.

Gap candidate: Versioned exhibit assembly and attachment manifests linked to exact document revision; distinguish candidate/final approved/released states, prove immutable export behavior.

Existing test source: tests/test_exhibits.py:66; tests/test_exhibits.py:150; tests/test_review_release.py:108. Reviewed as source or located by exact test symbol; not executed.

### CA-13 — Dense plain-language mouse and keyboard UX: present_with_limits

Observed: Plain page labels, phase grouping, link navigation, clickable command suggestions, Enter/Escape/Tab, Ctrl/Cmd-K, F1 assistant, Ctrl/Cmd-backslash split and clickable Ask/split buttons exist. Agreement compare displays texts; negotiation note is unsaved useState. API catalog can override local labels.

Evidence: `web/src/lib/surfaces.ts:40`; `web/src/components/TerminalShell.tsx:91`; `web/src/components/TerminalShell.tsx:244`; `web/src/components/TerminalShell.tsx:537`; `web/src/components/ContractWorkbench.tsx:16`.

Gap candidate: Retain navigation affordances, integrate evidence/review/proposals in dense work surface, remove technical vocabulary from product explanations, verify focus and keyboard workflows in browser.

Existing test source: web/smoke/design-contract.contract.test.mjs:27. Reviewed as source or located by exact test symbol; not executed.

### CA-14 — Persistence and audit integrity: partial_high_risk

Observed: Workspace passes store_dir, selecting SQLite. JSON is debug-only. Save deletes aggregate rows and commits before rebuilding; failure after commit can leave missing state. Version=1 and aggregate_version=1 undermine complete revision audit. No DB opened or mutations executed.

Evidence: `api/legal_workspace/services/workspace.py:194`; `api/legal_workspace/services/workspace.py:221`; `api/legal_workspace/db/engine.py:29`; `api/legal_workspace/db/store.py:609`; `api/legal_workspace/db/store.py:645`; `api/legal_workspace/db/store.py:778`.

Gap candidate: Replace destructive aggregate rewrite with transactional versioned persistence; preserve history and verify failure rollback/concurrent edits before expanded drafting/review workflows.

Existing test source: tests/test_persist.py:22; tests/test_work_product_versions.py:15. Reviewed as source or located by exact test symbol; not executed.

## Interaction and UI reconciliation

TerminalShell is a real navigation shell, not merely a picture: Sidebar renders anchors, CommandLine resolves page names and routes, and header buttons manage an assistant pane and pinned split view. Relevant symbols: `Sidebar:91`, `CommandLine:244`, `TerminalShell:438`, `onKey:537`. It uses phase-priority grouping, persists phase/pin UI preferences, and sends saved/live surface context to the assistant. `web/src/lib/surfaces.ts:40` has descriptive labels including Case dashboard, Motion writer, Your review, Court dates, Missing evidence, and My private notes. `:81` lets backend catalog labels replace local ones, so a complete no-acronym guarantee needs both catalogs inspected and a browser check.

The user’s dense mouse-plus-keyboard direction fits this shell, but it does not establish completed deeper workflows. `ContractWorkbench.tsx:85` is side-by-side text, not semantic diff or change acceptance; `:104` holds negotiation notes in component state without a save path. `DraftEditor.tsx:23` sends a complete heading/body PUT. `ReviewForm.tsx:80` offers whole-section verdicts. A future work surface should join source evidence, original/rewrite/proposal, independent check records and owner decisions while preserving the existing navigation affordances.

The citation “Snapshot check” tab is specifically misleading relative to behavior: `CitationParse.tsx:115` says it records hash/date, while `:43` invokes the same structure parser for every tab; `api/citation_routes.py:25` only parses citation text. Authority seed strings such as `packet:M2:mcl-722.23` are provenance labels, not verified captured-page digests. The plan should distinguish a reference pointer, an archived authority version, citation formatting, holding support, and subsequent-history review.

## Visual artifacts

No current-app screenshot or mockup image was found in bounded nonprivate application inventories, including a hidden/no-ignore check of docs and web/smoke excluding original-context, generated dependency and output trees. No new screenshot was taken because no app/browser/service was run. The existing `docs/reviews/2026-09-13-reconciliation/comparison.html` is an artifact found by inventory, not a current UI screenshot.

Historical `docs/reviews/2026-09-13-reconciliation/path-comparison.csv:4380` names donor `donors/legal-terminal-master/legal-terminal-master/docs/screenshots/web-01-home.png`, with related donor chat/palette images. These are historical inventory paths and must not be represented as verified-current application captures. Build-kit content was intentionally outside this current-source lane; the parent mockup/archive lane can resolve retained donor files.

## Recommended planning order

1. Resolve persistence transaction/history integrity and document actual SQLite versus PostgreSQL routing.
2. Define versioned resources, original voice, accepted analysis, contextual records, draft revisions and proposals before adding UI controls.
3. Separate each review/check into a version-pinned record; never treat citation presence as proof that an entire section is supported.
4. Build one integrated workflow: accepted source/context + chosen skill version -> proposed draft/rewrite -> independent checks -> owner decisions -> immutable review copy.
5. Extend risk/anticipated-accusation tracking and overall accepted factual timeline projections, preserving separate court-date semantics and missing-proof requests.
6. Expand requested firm roles with independent author/reviewer assignments and provider-routing contracts.
7. Verify dense work surfaces using real mouse and keyboard browser journeys, including reload, stale edit, failed save, missing evidence and rejected proposal cases.

This audit supplies gap candidates, not implementation authorization or legal conclusions. Detailed machine-readable coverage is in `coverage.json`.


## Follow-up: intended Timesketch fork and graph/timeline dependencies

The parent reported the user remembers a Google timeline project, probably Timesketch, and several graph/timeline visualization donors. A fresh bounded dependency/import/name search of web/package.json, web/src, api/legal_workspace, pyproject.toml, config and uv.lock found no Timesketch/Timescale integration or declared visualization dependency from this set: React Flow/xyflow, Cytoscape, Sigma, vis-timeline, vis-network, D3, ECharts, Plotly, Mermaid, NetworkX or PyVis. This is current-app integration evidence only; retained donor archives are outside this lane and may contain the intended projects/fork.

The current implementation is direct React markup: web/src/app/timeline/page.tsx:47 draws a border and article dots; calendar/page.tsx:78 uses a seven-column CSS grid. Both fetch /v1/docket-events. api/legal_workspace/services/workspace.py:349 also supplies the same docket context for both surfaces. It would be incorrect to credit this current screen as integration of the intended external event-timeline/fork project.
