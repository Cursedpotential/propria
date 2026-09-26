# Concrete implementation risks

Planning triage: S = data loss or misleading authority/support; A = substantial integration/rework; B = bounded usability defect. Existing findings are static source-derived risks, not exercised failures. Read [current audit](inputs/current-app/REPORT.md) for file/line evidence and [split contract](SPLIT.md) for proposed mitigations.

## S1 — Workspace replacement commits deletion before reconstruction

`api/legal_workspace/db/store.py:609–645` deletes dependent aggregate rows and commits, then rebuilds. A later failure cannot roll that committed deletion back. Before importing personal/legal collections, replace this with an atomic unit of work or version-preserving insert path, and inject a reconstruction failure in an isolated database to prove old state survives. Never exercise the failure against the owner's workspace.

## S2 — Version fields do not establish version history

The same store writes work-product `version=1` at `:778` and audit aggregate version 1 at `:1045`; ordinary draft edit overwrites text. A proposal referring only to a document ID can apply to changed text. Persist immutable revisions and expected-version decisions together; prove stale proposals conflict and a released revision survives later edits/reload.

## S3 — Section citations overstate factual support

`domain/support_map.py:69` marks non-instruction paragraphs supported from valid citations at section level. Citation gate checks package/assertion references and locator presence, not whether the cited material supports each proposition. Two claims, one cited source, must produce separate results; a reference-valid but unrelated span must not pass factual support. Keep substantive legal review independent.

## S4 — Authority labels imitate snapshot verification

`authority_library.py` seeds labels such as `packet:M2:mcl-722.23`; `CitationParse.tsx` uses the parser for the 'Snapshot check' tab. Those do not demonstrate captured bytes/hash/date, current law or subsequent-history review. Model actual capture evidence and uncertainty; rename/qualify the current claim before exposing it as verified. A resource importer must not promote packet labels into content hashes.

## S5 — Next-to-SPA migration can move server secrets into browser code

Current Advocatio includes API proxy/auth and AI SDK server-side behavior. Replacing Next routing without accounting for those server boundaries can break protected calls or expose server-only assumptions. Inventory every route/stream/proxy in F0/B0, retain API-side authorization and credentials, and exercise denied/matter-crossing/expired-session fixtures before parity approval. Common UI stack does not imply common credentials or provider SDK.

## S6 — Redaction overlays and translated text can preserve unwanted content

Historical handoffs require true redaction and court-safe derivatives. A black rectangle over PDF content or a neutral visible paragraph with the original retained in export revisions/comments is insufficient. Test text extraction, hidden layers/metadata, comments and revision export on the derivative; preserve original privately. Do not claim the current PDF dependencies implement this full pipeline.

## A1 — LibreOffice headless conversion is not interactive editing

The official parameters expose headless conversion and output/profile control, not a browser document editor. A job may succeed while losing layout or native revision semantics needed by the workflow. Separate rendering from editing; use explicit filters, per-job output/profile isolation and representative DOCX/ODT/PDF fixtures covering tracked changes, comments, captions, footnotes, tables and pagination. [LibreOffice parameters](https://help.libreoffice.org/latest/en-GB/text/shared/guide/start_parameters.html); [tracked changes](https://help.libreoffice.org/latest/en-US/text/shared/guide/redlining.html).

## A2 — A Glide web proof does not prove Tauri packaging

The recovered alpha experiment reports successful web/grid interactions but unresolved Windows native packaging at `kernel32.lib`. Reuse the exact recorded harness/version for preflight, then verify the intended Windows SDK/toolchain and native build separately. Storybook snapshots do not exercise canvas selection, keyboard editing or TSV paste. See [stack recovery](inputs/stack/STACK-RECOVERY.md).

## A3 — Timeline screen name masks missing factual chronology

Current `timeline/page.tsx` and calendar both fetch docket events. Adding a new visual library to that endpoint does not create relationship/parenting/claim chronology or source support. Establish EventView and claim/support contracts first; test scheduled versus actual events, uncertain dates, multiple tracks and knowledge horizon. Preserve advanced Timesketch fork integration as a distinct dependency.

## A4 — Timesketch fork and unified first-party client have different lifecycle decisions

Recovered accepted fork direction preserves upstream Vue/OpenSearch; current frontend unification selects a common first-party stack. Replacing fork internals without an explicit reconciliation would discard a maintained-upstream decision. Implement a bounded projection/curation interface and shell navigation contract; confirm whether embedding, linked launch or component adaptation fits the accepted fork. Do not silently create a second writable timeline truth.

## A5 — Full plugin import can confuse current source with older corrected variants

The reviewed resource batch has exact duplicates, empty files, multiple editions and claimed corrections. A filename/date-driven import can lose a valid item or promote obsolete legal guidance. Preserve file identities and relationships; reconcile a per-item manifest against baseline counts; make authority verification a separate recorded pass. FreeEed's almost-empty complete pack is not equivalent to its populated source checkout.

## A6 — Local UI state can look like saved case work

Current `ContractWorkbench.tsx:104` negotiation notes use component state. Leaving the page can discard apparent work. Mark provisional state and persist explicit revisions with save/error feedback before treating this as a strategy record. Draft comparison is not tracked changes or a saved negotiation lifecycle.

## B1 — Dense terminal navigation can hide ordinary actions

Existing shell already has mouse links and keyboard interaction. Preserve it while removing mnemonic-first assumptions from donor material. Full labels and visible actions must remain usable without knowing a command code; mobile help must work with tap, and keyboard focus must reach the same controls. Dense means more useful information per view, not compressed jargon.

## A7 — Toolkit loader dry-run can still initialize the database

The toolkit audit found `loadContent` calls `getStore` before the dry-run branch, and `getStore` initializes schema. Do not use that loader as a read-only population probe. Inspect migration receipts and use an explicitly read-only query/client to reconcile remote counts. Successful local fallback reads cannot prove remote population. See [toolkit inventory](inputs/stack/TOOLKIT-CAPABILITIES.md).

## A8 — Connected reduced console does not expose the full toolkit

The full plugin registers 27 handlers while this session exposes eight from the reduced console. Plugin manifest, runtime package and server constant also carry different version strings. A connection success or one version label cannot establish completeness. Reconcile tool identities/schemas and execute a representative call per capability group; track the complete inventory and current route separately.
