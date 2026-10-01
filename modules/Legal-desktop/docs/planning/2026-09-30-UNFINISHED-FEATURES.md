# Advocatio unfinished-feature delivery checklist — 2026-09-30

Advocatio is partially implemented. A reachable application, office service, reference import, or MCP registration does not complete the product. This checklist restores the original feature scope alongside the newer infrastructure ledger. It is a bounded reconciliation, not a fresh end-to-end audit of every feature.

## Evidence and current baseline

- September 30 deployment receipt: docs/receipts/2026-09-30-coolify-state-refresh.md. Drafts page, office status, and WOPI discovery responded successfully. Actual live editing/save/reopen remains unproved.
- September 24 document implementation: durable DOCX/ODT work documents, revisions, scoped WOPI sessions/locks, office templates and writing workspace. These supersede older claims that no interactive editor integration exists. They do not close forms, proposals, fidelity, recovery or phone workflows.
- September 27 toolkit checklist: 383 entries; 38 exercised successfully, 242 registered/stored but unexercised, 103 without hosted equivalents. Those are inventory rows, not unique user-visible features.
- September 30 central ledger records 323 references and a shared record contract; full collection correctness/completeness and human workflow proof remain separate.
- Original scope: docs/planning/2026-09-13-advocatio-reconciliation/REQUIREMENTS.md (R01–R52), ROADMAP.md, REDISCUSSION.md; original build-kit handoffs; September 24 DOCUMENT-WORKFLOW-AND-DELIVERY-GAPS.md.
- September 30 source check: domain/support_map.py still assigns supported to every non-instruction paragraph when section-level citation_count and citations_ok are truthy. Passage-level factual support is not established by that code.

## Open delivery areas

| Area | Present foundation | What still needs completion | Completion evidence |
|---|---|---|---|
| Everyday writing | Office adapter, work documents/revisions, template creation, import | Live edit/save/reopen; native comments/tracked changes; recovery, conflict and export fidelity; usable phone actions | Create, edit, save, reopen and download a document; compare text/formatting/comments/revisions and failed-save recovery |
| Official forms | PDF preview and document/rendering foundations | Official Michigan/Genesee/FOC form catalog, source/version metadata, field discovery, filling, overflow and saved editable/flattened output | Fill representative official forms, reopen values/checkboxes and verify no silent truncation |
| Proposed edits | Draft/review foundations | Durable passage-anchored accept/reject/edit proposals, original preservation, stale-base conflicts and resulting revision linkage | Mixed decisions survive reload; outdated proposals cannot overwrite later typing |
| Write first, apply template later | Blank writing and office template entry points | Map rough writing to selected template as proposals; retain unplaced text; separate structural moves from meaning changes | Full original retained; mappings reviewable; accepted output opens as an actual office document |
| Court-language translator | Toolkit methods and routed role foundations | Original/proposed wording together, observable behavior wording, source links, individual decisions | Rewrite preserves meaning and original; additions and unsupported claims identified; decisions persisted |
| Four independent checks | Existing citation/review concepts | Distinct structure/method, legal-source, factual-support and substantive-reasoning/relief findings, versioned and stale-aware | Separate outcomes on one document; no universal validated badge |
| Claims and evidence | Packages, citations, catalog and investigation foundations | Exact claim/passage-to-source links, partial/contradictory support, claims-without-evidence report; correct support-map false positives | A citation supporting one claim cannot support an unrelated claim; gaps/contradictions survive reload |
| Evidence workflow | Catalog and internal investigation routes | Clear import handoff to evidence intake, promotion linkage, actionable gap follow-up; discovery RFAs/RFPs plainly separate | Follow an import through its actual intake status; accepted item linked to a claim; separate discovery and investigation paths |
| Timelines and graphs | Local docket/calendar and retained fork plans | Shared relationship/claim/contact/court chronology, evidence indicators, filters, Timesketch adapter and named visualization dispositions | Same event/source/version across views; unsupported events visible; no second authored evidence store |
| Personal context and strategy | Private notes/strategy/red-team foundations | Structured case/relationship context, vulnerabilities, anticipated accusations, defenses, saved strategy decisions/playbooks and analysis follow-up | Private context retained and retrievable; theories remain distinguishable from accepted facts |
| Full toolkit and methods | Shared records, references, live MCP tools | Close all 103 hosted gaps; exercise remaining callable rows appropriately; browse packs/widgets; contextual method help and executable workflow bindings | Each inventory item has explicit disposition and representative proof through intended human/agent surface |
| MCP client and external sources | ContextForge gateway/tool registration | Full registration/edit/test/enable/disable, discovery, schema-driven manual calls, history and scoped agent use; CourtListener integration | Register a server, discover/call a tool, inspect a persisted result, revoke access and observe enforcement |
| Digital firm | Role/provider routing foundations | Recovered multidisciplinary workflow and role handoffs: managing/senior associate, red team, researcher/librarian, document clerk and exhibit clerk | One bounded case task passes through roles with source/method/run provenance and reviewable outputs |
| Shared quick surface and convergence | Shared legal-record contract and palette | Phone-friendly read/add/reference/calendar flows, shared source/detail behavior and selected framework/client migration | Same identifiers/versions on both surfaces and agent reads; responsive interaction proof beyond shared colors |
| Other integrations | Handoff inventories and preserved donor/fork code | Explicit adoption/adapter/disposition and delivered use cases for retained graph/report/analysis/screen-sharing candidates | Named capability and working common-surface use case, or explicit documented disposition |
| Operational foundation | Coolify, persistent SQLite, shared store | PostgreSQL contract reconciliation, identity/auth leftovers, promotion adapter, historical security claims recheck | Current live receipts; safe data migration proof; service lifecycle managed through Coolify |

## Execution order and parallel ownership

1. Correct factual-support semantics and define exact support contracts. Own support-map/domain/service/tests only; UI changes coordinated separately. This precedes claims-gap accuracy and factual-review completion.
2. Prove real office editing/save/reopen and document fidelity. Own office adapter/editor smoke proof; coordinate shared document route changes.
3. Deliver durable proposals and court-language review on work-document revisions, then template application. Depends on stable revision/anchor contracts; does not require all timelines or toolkit ports.
4. Build official PDF forms independently of proposal UI. Own form adapter/catalog/routes and form surface; coordinate shared document storage interface.
5. Finish toolkit/method/callable coverage and MCP client workflows. Own toolkit/client methods area; do not copy shared authoritative records.
6. Build claims/timelines/context/strategy around the accepted-source and support contracts.
7. Complete digital-firm handoffs and common-surface convergence around demonstrated workflows.

Use bounded agents with explicit file ownership. Serialize edits to shared routers, config, document contracts and Git index. Cheap readers can inspect inventories; implementation and review use sufficient capability for the risk. No blanket no-agent rule.

Do not treat incomplete work as needing renewed permission merely because a historical checklist says owner selection: wholesale toolkit coverage and the product workflows above were already requested. Additional original firm prompts, vLex examples and selection of personal source records remain actual pending inputs, not reasons to halt unrelated implementation.

## Status and scope of this update

No application feature is marked completed by this document. No application code was changed in this reconciliation. The immediate next engineering target is the support-map false positive; office live workflow proof can run in an independent lane. Existing older TODO entries are historical until reverified; this checklist supplements rather than deletes them.

