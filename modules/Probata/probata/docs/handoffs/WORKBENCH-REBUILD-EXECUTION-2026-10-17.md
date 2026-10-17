# Workbench rebuild — implementation tracker

Byline: ChatGPT, 2026-10-17. Scope: the owner's approved operator-experience rebuild, not a backend replacement.

## Current accounting

The full rebuild is **not complete**. PR #15 was an incremental enhancement, not completion of the approved redesign. The default/expanded/mobile workspace screenshots are explicitly synthetic interaction references; they are not integrated product pages or production acceptance.

The present implementation branch is `codex/workbench-unit-workspace-20261017`, based on deployed Workbench revision `7eb2981bfb72b2f1341c56aad14a4d6052878769`. The main branch has independent changes; do not replace it with the deployed branch or discard concurrent work. The unrelated knowledge-reconciler PR #16 is not part of this slice.

The attempted Codex helper stopped at the account usage limit before coding. Changes in this branch were subsequently implemented directly. There is no unattended coding worker promised by this record.

## Work packages still required

An item is not complete merely because its backend exists, its screen compiles, a mockup looks right, or a deployment is healthy. A connected operator journey is required. These are implementation tasks from the approved conversation, not new approval requests or estimates.

| ID | Work package | Current work / remaining work | User-visible completion test |
|---|---|---|---|
| WB-01 | Application flow and common selection | Selected-source inspection is recomposed in this branch. The broader shell, navigation, category-oriented discovery, persistent selection, reading position, mobile and accessibility work remains. | Find a unit, inspect it, change representation, return and reopen without losing context. No global-search panel dominates opened content. |
| WB-02 | Organized library | Prior source filters are usable building blocks. Broader categories (human messaging, AI chats, contacts, documents, geo, media, work products, unclassified), ownership and version facets, source-family grouping and saved views remain. | Find material by meaning and category without sorting through unrelated raw filenames. |
| WB-03 | Complete units, directories and package links | This branch resolves registered Facebook JSON siblings under one qualified conversation directory. It does NOT inventory unimported originals, establish native conversation identity, or implement arbitrary/nested atomic preservation. Connect actual catalog memberships, directory ancestry, owner marks, manifests and missing-part coverage. | Open message_7.json and see every known part and associated file, including not-yet-imported members. Any directory at any depth can be an atomic unit; nested membership survives. |
| WB-04 | SBV messaging and attachment reader | This branch uses existing SBV message bubbles with per-member source references and one unit stream. Complete Facebook donor adaptation, platform-specific replies/reactions/edits/calls, inline images/audio/video/documents, attachment galleries and source-reference resolution remain. | Read across part boundaries, open each attachment, distinguish missing bytes from an unresolved/unimported reference, and trace every item to its source. |
| WB-05 | PostgreSQL normalization inspection | This branch supplies actual normalized-record readback, full selected-record payload and latest-per-source generation inventory. Original-to-normalized field comparison, raw/member coverage, losses/skips and correctness validation remain. | Compare a source field to its stored PG value and account for all members without mistaking equal counts or a success receipt for correct normalization. |
| WB-06 | Expandable editable inspector and identities | Compact card plus expanded PG/detail panel reuse existing metadata/context review/foreshadowing and identifier tools. First-/third-party correction, scoped participant/contact/alias/entity reassignment, custom metadata, bulk edits and downstream refresh indication still need full in-place integration. Existing global-alias edits must not masquerade as one-message corrections. | Change the right association at the chosen scope, save, reload, verify persistence, and leave unrelated sources unchanged. |
| WB-07 | Lineage reconciliation | Retained source versions are accessible in this branch. Cross-file original/derived families, old agent/script work, repeat imports, historical extraction comparisons, projection generations, preferred versions and uncertain/orphaned links remain. | Identify several derivatives of one original and resolve their relationships without deleting originals or collapsing independent occurrences. |
| WB-08 | Processing controls, profiles and Tool Runner | Existing extraction selectors/results are reused per native conversation. Version-pinned whole-unit actions, approved type/schema profiles, known-schema matching, mapping/transformation previews, repair/retry, and contextual integration of the independent Tool Runner remain. | See what can be extracted and what has been extracted; run the chosen existing tool on the displayed scope and inspect the actual resulting artifacts. |
| WB-09 | Chunk and Weaviate inspection | Dedicated source-aligned chunk browser and indexed-object inspector remain. Use the agreed RAGFlow/Retab/Extend and Weavit/turbopuffer donors as appropriate, without adopting a replacement vector database. | Open a full chunk, inspect boundaries/overlap/generation, highlight source messages and confirm its actual indexed representation. |
| WB-10 | Advanced search and scope | This branch adds literal whole-unit PG search before pagination. Complete global federation across existing index/Super Index, Weaviate keyword/vector/hybrid, structured/metadata, media and graph retrieval remains, with actual supported filters and coverage. | Search Case Vault, imported content or both; retain provenance, facet scope, versions and partial-source failures. Within-unit search does not require Super Index. |
| WB-11 | AI-chat context lane | Keep existing parsers and direct-Weaviate context ingestion. Complete roles/branches/tool-output/source-package viewing, searchable readback, generation/chunk visibility and recovery of historical imports remain. | Import and retrieve a complete AI chat as context, without routing it through human-message normalization or evidence promotion. |
| WB-12 | Entities, events, timeline and graph contributions | Existing extraction/context-review primitives are reused, but full association/correction and source-linked event/schedule/timeline contributions remain. Neo4j and SurrealDB require distinct actual destination readbacks; neither is replaced or silently populated by viewing. | Follow a message to its extracted entity/event and any genuine graph contribution, then back to the original. Save a correction with clear provenance and affected versions. |
| WB-13 | Other source-specific viewers | Documents/OCR (Retab/Extend), structured tables/databases, images/media (PhotoPrism/Immich/other agreed donors), audio/video/transcripts, geodata/maps, archives and unknown sources remain separate capability tracks. Retab is not a global donor ranking. | Category selection opens the correct functional reader, with source-grounded outputs, attachments and full metadata instead of a generic JSON card. |
| WB-14 | Activity, integration and acceptance | Preserve Temporal/Go execution, actual attempts, errors, retries and outputs. Complete route/state/accessibility tests, real package browser acceptance on VPS, API readback and release integration remain. | Run the actual user journey through the deployed revision, including persisted corrections and missing/partial states. CI, screenshots and container health alone do not close this row. |

## Immediate implementation order

The next coherent release target is **Library → complete source unit → SBV conversation/attachments → expandable record/identity inspector → extraction/chunks → readback**. WB-01 through WB-06 and the necessary parts of WB-07/WB-08 are connected work, not separate substitute demos. Unit membership and attachment resolution are first dependencies, because a polished reader on a partial or wrongly grouped source is still wrong.

Parallel work can cover authoritative catalog/attachment adapters, the common workspace composition, scoped correction/identity contracts, and independent fixture/browser verification. Keep path ownership explicit. Do not spawn workers and leave their status unverified.

After the initial complete messaging path, reuse the same selection/inspection contracts for AI chats and the other category-specific viewers. Finish full search and lineage reconciliation through their existing owning stores; do not invent another index to avoid integration.

## What this branch actually adds

- Read-only API routes under `/api/imported/sources/{source_id}/unit`, `/unit/messages` and `/unit/records/{record_id}`.
- Registered Facebook sibling grouping based on the **same exact qualified conversation directory and recorded format**. No grouping by person name, message body or basename across directories.
- Inventory of each exact source key's latest retained version and latest normalization generation. Older generations remain in source history; selecting the latest is a stated display policy, not an owner preference or correctness attestation.
- A chronological, cursor-paged message read across the selected generation set, with literal text filtering before pagination and explicit member/conversation narrowing. Cursor fingerprints prevent mixing changed scopes or generations. Unknown timestamps remain unknown.
- Actual selected PG record fields, source/version/generation/ordinal and readback time, including a separately marked historical record when requested.
- A separate selected-unit page composition: member navigation, SBV-centered working content and a compact inspector expanding below. Global discovery/search is not mounted in this state.
- Existing context-review/foreshadowing, full file-metadata correction and identifier editing components are reused when a valid source operation exists. Missing operation links do not turn into fake saved edits.
- Existing native conversation extraction targets are resolved from the complete unit inventory, not just loaded message pages. Their outputs load on expansion. Existing extraction dispatch is NOT newly claimed to pin an entire unit to the displayed generation set.

### Deliberate limits, not completed features

The grouping covers **registered imports**, not a complete physical export or permanent catalog unit. Unimported JSON parts, externally referenced media, nested owner-defined atomics and full manifest correlation are still WB-03/WB-04. Attachment counts/reference fields alone are not image/audio/video playback. This branch must not be represented as satisfying the owner's complete seven-part-plus-media acceptance example.

The selected normalized payload is real stored data, not a validation report. Raw-original comparison and parser/schema correctness are still WB-05. Some reused editors operate on an existing source operation or a global registry alias; local source/mention-specific correction needs WB-06 rather than relabeling those operations.

No new graph projections, embeddings, source processing jobs, schema changes, case-data mutations or production deployment were executed while developing this slice. Tests use synthetic records.

## Donor reuse accounting

Existing SBV viewing code is actually reused. RAGFlow layout influence and SBV ports are already documented in the Workbench's `THIRD_PARTY_NOTICES.md`. The other agreed donors remain capability-specific integration work; their names in this tracker are not claims that their code or UI is integrated.

## Perspective-correct annotation source of truth

[As-Lived, Hindsight, and Foreshadowing — owner-approved use case](https://github.com/Cursedpotential/propria/blob/68f369eb738ef3fe27966ef49a1173ea18047d68/modules/Probata/probata/docs/AS-LIVED-HINDSIGHT-USE-CASE.md) is the controlling explanation. Do not restart that design discussion, introduce routine ingestion blockers, or call the downstream walk complete because the reviewer can set a flag. Source messages remain eligible when knowable; later annotations must not leak through ranking, graph links, derived text or agent memory.

## Verification and handoff

Exact test results and source commit/PR are recorded with the implementation review. Windows browser execution is prohibited; the desktop runs only synthetic API and browser-free frontend checks. Real database/package/attachment and VPS browser acceptance remain outstanding until recorded against the exact release.

The historical 31 complaint families remain a required reconciliation input. This task tracker preserves the approved scope, but it does not claim a complete 31-row complaint audit or closure where the underlying ledger has not been independently recovered.


### Local validation receipt ? 2026-10-17

- TypeScript typecheck: passed.
- ESLint: no errors (the full existing frontend reports its existing React-refresh warnings).
- Vite production build: passed; existing large-bundle warning remains.
- Browser-free frontend suite: **213 passed, 4 browser journeys skipped**, zero failures; includes four new synthetic selected-unit render/scope tests.
- Storybook build: passed.
- Full API suite: **936 passed, 2 skipped, 2 failed**. All eight added unit tests pass. Remaining failures are existing `test_structure.py` checks for backward imports and oversized unrelated Python files; they were not suppressed or fixed by this UI slice.
- Local runtime: Node 26 / Python 3.13. CI uses its configured Node/Python versions and must be read independently; this receipt does not predeclare its result.
- No live SQL/package readback or browser journey was performed. The tests substitute synthetic PG results; they do not verify the owner's actual seven-part export or attachment availability.

Deep links preserve the selected record for exact PG inspection, but seeking the message stream to an unloaded/historical record and restoring its scroll position still require WB-01/WB-05. Do not treat record-inspector access as complete source-context navigation acceptance.
