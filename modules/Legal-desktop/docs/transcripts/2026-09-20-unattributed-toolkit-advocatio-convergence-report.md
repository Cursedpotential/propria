---
title: Agent status report — Family Law Toolkit and Advocatio as two surfaces of one legal system
date_shared: 2026-09-20
shared_by: owner (pasted into a Claude Code session at the Propria root, 22:46 EDT)
source_service: not stated by the owner; text matches the session that authored commits a233171..b08e7b8 in this repository
capture: verbatim as pasted; the first character was cut off in the paste ("es." is "Yes.")
tags: [transcript, advocatio, family-law-toolkit, convergence, live-bringup, capability-map]
related:
  - docs/planning/2026-09-13-advocatio-reconciliation/continuation/SINGLE-WORKDESK-CONVERGENCE.md
  - docs/planning/2026-09-13-advocatio-reconciliation/continuation/LIVE-BRINGUP-2026-09-20.md
---

> _Byline: Claude Code · Fable 5.1 · 2026-09-20 (saved verbatim; no facts added here — the planning documents above hold the record)_

## Transcript

es. The Family Law Toolkit and Advocatio are two surfaces of the same legal system. Your clarification is now saved in the convergence plan and published to the document store.

| Surface | Intended role |
|---|---|
| Family Law Toolkit | Mobile-friendly access to view and add documents, read cheat sheets, browse law and sources, and retrieve custom and case-specific information. |
| Advocatio workdesk | Deeper analysis, drafting, strategy, evidence-linked claims, review, and document preparation using that same information. |

The integration scope includes all toolkit data, sources, features, and custom material. Each capability needs an implemented destination or a callable operation. Shared document and source identifiers, versions, and case records will let both surfaces stay synchronized without entering the same information twice.

The stack and design refactor remains part of that work: shared React/TypeScript conventions, common API contracts, Propria design components, and responsive layouts that can eventually live within one unified surface. Your decision to use one current workdesk version does not cancel the toolkit's mobile role.

There is also concrete progress on getting the current workdesk working:

- Fixed and deployed the API packaging defect that prevented matter data from loading.
- Verified the dashboard, drafts, templates, laws, calendar, and assistant pages load successfully.
- Verified the application's matter, drafts, and templates API reads return successfully, and the browser shows "Available: workspace backend."
- Published four planning and verification documents to the document store and independently checked their stored hashes.
- Committed and pushed the changes; the working tree is clean.

You can open the repaired Advocatio workdesk.

The full toolkit integration is still outstanding. The 383-capability map tells us what needs to come across; it is not a claim that everything is already connected. The next integration proof is to open the same cheat sheet, legal source, and case document in both surfaces with matching IDs and versions, then trace a synthetic document submission through shared intake into the workdesk.

The clarified contract is saved in [Single Workdesk Convergence](E:/AI_Workspace/Projects/Propria/modules/Legal-desktop/docs/planning/2026-09-13-advocatio-reconciliation/continuation/SINGLE-WORKDESK-CONVERGENCE.md), with deployment results in the [live verification receipt](E:/AI_Workspace/Projects/Propria/modules/Legal-desktop/docs/planning/2026-09-13-advocatio-reconciliation/continuation/LIVE-BRINGUP-2026-09-20.md).
