# Extracted-package web review handoff — 2026-10-12

Byline: Codex · GPT-6 · 2026-10-12. Source implementation and validation receipt;
this is not a deployment or combined live-acceptance receipt.

## Ownership and exact base

- Worktree: `E:/AI_Workspace/Projects/Propria/_worktrees/extracted-package-web-20261012`.
- Branch: `codex/extracted-package-web-20261012`.
- Created from `origin/main` as required by the root contract, then based on exact
  requested web release `2b621fee49f9a79d0393e166537ebd96381a6b69`.
- Implementation validated through `e6f2047ff` before this receipt was added.
- All changed files are beneath `modules/Probata/probata/modules/workbench/web/`.
- Parent owns integration, release, deployment and SMS work. The backend worker owns
  the API, source readers, PostgreSQL review writer and migration. No backend,
  deployment, originals, shared checkout or existing chat files were edited here.

The latest owner correction controls: contextual agent chat/SDK integration is a
later TODO for the parent to persist. No chat pane or new chat implementation is in
this slice. Existing chat remains unchanged.

## Delivered behavior

The default `/read` entry shows actual backend Case Vault category headers and
packages/files underneath each header on the same screen. It uses persisted
domain/category/path/classification basis and visibly retains unresolved placement.
It does not infer categories from extensions, flatten packages, require a category
click, move originals, or invent a taxonomy. Category and package lists have explicit
server-cursor paging. Unavailable/unbound packages show the actual reason instead of
an invented reader link.

The navigation label is **Context review**; the `/read` route is retained. Existing
`resource=human-...` and native human preview links still open `HumanContextPreview`
by default, with an explicit full-package link. Existing context-workflow candidate
links retain `AICandidateReview`. Legacy processing previews, raw SMS, decoded SMS,
source/thread/search links and their working viewers are retained. Category links
open the new full workspace via `/read?package=<actual handle>`.

The full workspace uses the backend section inventory, including unavailable reasons
and counts. There is no source-type or section-ID whitelist: native human, AI chat,
screenshots, documents and locations use the same source-linked package contract.
Chunks, entities, events, timeline, locations, strategies, documents, records,
proposals, edges, embeddings, correction history and native artifact sections can all
render when returned by the backend. Available sections depend on retained artifacts;
this UI never pretends a missing extraction exists.

The layout reuses the RAGFlow-inspired `sbv/message-browser.tsx` split-panel pattern,
actual Glide grid and grid theme, the existing metadata correction editor, existing
entity alias rendering, existing native/SBV viewers and source-aware `/read`
navigation. The message-surface handoff's historical exclusions of editing/graph
work were superseded by this owner's explicit current scope; the donor was adapted
without replacing the SMS viewer.

Selecting an item drives the right editor. Complete nested text is readable without
grid-preview clipping. All returned fields, source/version/locator objects, metadata,
placement, complete item records and correction history remain inspectable. Source
account, source device, person and provenance-party/basis fields remain separate.
Actual edge endpoints/kinds/direction/source pins appear in the graph and full edge
inspector. Timeline cards display each native clock independently in source order.
Unknown clocks, unavailable readers and absent source locators remain explicit.

Review controls derive from advertised capabilities. The shared metadata editor
supports corrections; supplemental JSON supports additional metadata/findings/edge
fields; reassociation uses the actual entity registry and available association
fields. Accept/reject/commit appear only if advertised. Native immutable fields are
honored. Writes carry exact item/revision/source refs, owner reason and a stable
idempotency key. Failures retain drafts; saved correction heads overlay display
values while original fields remain in the complete record. Pending writes and dirty
drafts guard route changes and browser unload. No optimistic save claim is made.

TanStack Query mutations invalidate the selected package; placement edits also
invalidate category/package listings. Section queries retain at most four manually
requested pages and expire inactive contents after one minute. Earlier-page unloading
is visible with a first-page control. There are no scroll/load-all fetch loops.
An item outside loaded pages requires explicit paging or a section selection; no
uncontracted item-resolution endpoint is fabricated.

## Confirmed API contract and release dependencies

- `GET /api/context/categories?cursor=` returns actual category identities/counts.
- `GET /api/context/packages?category=&domain=&cursor=` returns intact packages,
  persisted placement/basis, native handles and availability/reason.
- `GET /api/context/packages/{handle}?section=&cursor=` returns source/package/version,
  revision, metadata, dynamic sections, items/correction heads/edges/capabilities and
  the next cursor. Default section is backend `allmetadata`.
- `POST /api/context/packages/{handle}/review` accepts
  `{operation,item_id,revision,fields,source_refs,reason}` and `Idempotency-Key`.
  Actual operations are `metadata|edit|accept|reject|commit`.
- Native human handles are existing `human-<source version UUID>`. AI handles are actual
  workflow IDs such as `context-source:...`, without an invented `ai:` wrapper.
- Backend item `fields` remains original; `correction.fields/source_refs/review_state`
  is the attributed current head. Initial correction revision can be the empty string.
  Native edge keys include `edge_id,kind,from_node_id,to_node_id,source_pins`.
- `placement` is a real allmetadata item using the advertised metadata capability.
  Derived outputs inherit/link to recorded source placement. No source relocation is
  implied by a placement correction.
- Backend live inspection found older persisted category labels and absent proposed
  placement tables. The frontend uses actual current labels; it does not remap them.

The backend worker reports that `workbench_reader` needs SELECT grants on native
source/correction/object/binding tables. Its migration also makes legacy coordinates
nullable for native correction rows. Parent must integrate/apply that migration;
no production DDL was applied by this web lane. The worker independently reports
focused BFF/Go/PostgreSQL tests and read-only production query-shape probes passing;
those reports are separate from this web receipt.

**Model retry is still an integration dependency.** `package-retry.tsx` invokes only
an advertised `retry_extraction` or `retry` capability with `method: POST`, a valid
same-origin `/api/` path and actual body/model configuration. It requires a reason,
retains failure drafts, guards pending navigation and invalidates only its package.
Supported model configuration is `models: string[] | {id,label}[]` or an actual fixed
`body.model`. The backend package contract currently supplies no retry capability,
so no retry button is rendered. Parent must supply/map the real working capability;
no extraction/model placeholder or fake call was added.

## Verification receipt

All builds, dependencies and browser checks ran in the existing OVH devbox
`devbox-pd3xc78ahqkfswq12bpfqgy1-145003736857` on ovh-files, as `kasm-user`.
No desktop browser, Docker invocation, build or heavy test was run locally.
Source archives and test profiles were retained; no files were deleted.

Validated source `e6f2047ff`:

| Check | Result |
| --- | --- |
| Changed TS/TSX plus Storybook configuration ESLint | Pass, zero errors and zero warnings |
| `npm run build` (`tsc --noEmit` plus Vite) | Pass; existing large-bundle warning remains |
| `STORYBOOK_DISABLE_TELEMETRY=1 npm run build-storybook` | Pass; six package stories registered |
| Remote Chrome actual Storybook checks | Pass, all six scenarios and accessibility checks |
| Entire source smoke suite | 176 total: 165 pass, 6 fail, 5 intentionally skipped browser tests |
| Untouched exact base: the two failing SMS/source test files | Same 6 failures, 10 passes |

The six passing actual-component browser scenarios are category headers with a
package underneath, complete package, reopened correction, save-and-reopen,
failed-save draft retention, and dirty navigation/stay. They run the real package
components, Query hooks and editor through bounded synthetic Storybook API responses.
They establish UI behavior, not live backend persistence or production acceptance.

The six baseline failures are five `source-perspective.contract.test.mjs` cases whose
isolated VM callback lacks `isSMSContextSource`, plus the
`sources-screen.contract.test.mjs` Process test's stale `processSelection` regex.
Both tests and their source targets remain unchanged from exact base. Parent owns
that SMS/source lane. Full-repository lint also previously failed on unchanged
`src/components/read/context-search.tsx:32` (`react-hooks/set-state-in-effect`);
changed-file lint passes. The native-reader stub and metadata-title test affected by
this implementation were updated and pass in the final suite.

Remote logs are retained under `/home/kasm-user/context-review-web-014128bd8/`:
`changed-lint-final.log`, `build-final.log`, `storybook-final.log`,
`story-runtime.log`, `smoke-final3.log`. Baseline comparison is retained under
`/home/kasm-user/context-review-base-2b621/base-smoke.log`.
The `014128bd8` directory name is the initial source-copy label; the final archive
unpacked into it is from validated `e6f2047ff`.

Browser verification is opt-in and refuses Windows:

```sh
CONTEXT_STORY_BROWSER=/usr/local/bin/google-chrome node --test smoke/context-package.stories.test.mjs
```

## Complete web file allowlist

Paths below are relative to this web directory. These are the only source/receipt
files changed by this lane:

- `CONTEXT-PACKAGE-REVIEW-HANDOFF-2026-10-12.md`
- `.storybook/main.ts`
- `src/app/read/page.tsx`
- `src/surfaces/primary/navigation.ts`
- `src/lib/context-package-client.ts`
- `src/lib/context-package-review.ts`
- `src/hooks/use-context-package.ts`
- `src/components/packages/category-landing.tsx`
- `src/components/packages/context-package-workspace.tsx`
- `src/components/packages/package-items-grid.tsx`
- `src/components/packages/package-content.tsx`
- `src/components/packages/package-item-editor.tsx`
- `src/components/packages/package-relations.tsx`
- `src/components/packages/package-retry.tsx`
- `src/components/packages/context-package.stories.tsx`
- `src/components/metadata/metadata-field-table.tsx`
- `src/components/entities/entity-alias-label.tsx`
- `src/components/entities/entity-card.tsx`
- `smoke/context-package.contract.test.mjs`
- `smoke/context-package.stories.test.mjs`
- `smoke/decoded-reader.contract.test.mjs`
- `smoke/proffer-operator-surface.contract.test.mjs`
- `smoke/review-overlays.contract.test.mjs`

## Parent integration acceptance still required

Integrate this branch with the backend worker, apply its required native-review
migration/grants through the parent release lane, and map the real retry capability.
Then validate on the existing Coolify dev environment: actual default category
headers and packages, native human and AI packages, complete documents/locators,
persisted placement changes, correction save/reopen, stale revision, failure/dirty
handling, decisions and retry. Record exact source versions and receipts. Existing
native/SMS readers must continue to work. No deployment, production mutation or
combined live acceptance is claimed by this handoff.

## Final backend receipt and clarified completion boundary

The backend worker subsequently pushed implementation commit
`a7dafa40fc17421663fc840df001377616ac8c88` on
`codex/backend-package-review-20261012`. Its instruction is to cherry-pick that
implementation commit only, excluding preparation commits `9c7796e55` and
`8435cbe80`. This web lane independently read the backend HEAD and review writer
code; it did not integrate or deploy them.

The required migration is
`modules/Probata/probata/modules/engine/postgres/source_package_review.sql`.
It provides nullable legacy coordinates for native review rows, native revision
uniqueness/supersedes linkage, and SELECT grants on five existing source tables for
`workbench_reader`. Parent release owns applying it.

The worker's final test report is 16 passing BFF tests, passing focused
humancontext/runtimeapi/starter/PostgreSQL tests, isolated PostgreSQL review
integration with four revisions, and passing read-only production SQL shapes.
The worker reports its isolated database was purged and no production schema or
deployment changes were applied. These are backend-worker receipts, separate from
the web verification above.

The implemented `commit` writer appends a committed assertion with the accepted
fields and exact source references to `SourceMetadataStore` correction history.
It does **not** bridge existing candidate edit/publication writers. The web button
therefore says **Commit reviewed assertion**, or uses an explicit backend capability
label, and describes its actual package-history effect. Do not claim existing
candidate publication integration from this action.

The graph UI exposes actual returned edges. Complete source-reader coverage beyond
the returned bundles, including all typed graph inventories in the reference
guides, is not established by this renderer or the backend tests. The worker
explicitly identifies reader coverage outside those bundles as unfinished.

The worker reports the queried persisted taxonomy currently contains exports (1),
messaging (774), telephony (8) and unresolved (3). This does not establish that
approved Case Vault roots are reconciled with that catalog. The UI faithfully
displays persisted categories; parent still owns placement/catalog reconciliation
and deployment. Model retry likewise remains a parent integration dependency.
These are open delivery boundaries, not completed by dynamic sections or controls.

The final backend immutable-field list also includes `raw_reply`, `reply` and
`replies`; the editor already consumes the actual descriptor list dynamically.
Chat/SDK work remains deferred and unchanged.
