# Workbench Sources → Activity → Read delivery

> Byline: Codex · GPT-6 · 2026-10-06. Scope: Probata Workbench, not the Homepage portal or Advocatio legal desk.

Authority: the ratified `docs/pending-review/2026-09-21-intake-review-module-rethink.md` and the owner's 2026-10-06 instruction to implement the complete workflow. Existing case identity, custody, source data and operator gates remain authoritative.

## Delivered code

- Sources is the entry point. Folder navigation has durable URLs; local files use the existing sealed acquisition endpoint. One Process action records accepted requests, retains their identities during partial retry, and exposes the returned Activity attempt/batch link.
- Activity reads the existing Proffer operation ledger and batch records. Refresh retains older loaded pages, new operations stay first, cancellation requires the exact mode-bound snapshot, and retry/decision links open the corresponding Read attempt.
- Read combines imported sources, whole-conversation selection, existing extraction tools, content search, messages and cited extracted context. Message links retain source/thread/search context. Preview navigation keeps pending parser/repair state mounted.
- Old `/intake`, `/review`, `/evidence/preview` and `/conversations` links redirect while preserving identifiers and query context. Case remains a separate supporting page.
- Sources content/meaning search uses the existing Intake API at `http://100.91.190.107:8765`, configured through Coolify's `INTAKE_DISCOVERY_INDEX_URL`. Bounded live keyword and hybrid probes each returned one result from `IntakeCorpus`; post-deploy Workbench probes each returned 100 results. Coverage remains `unknown`. No new index was created.
- Read message search uses `ProfferChunks20261002`. The deployed compose had selected the older raw-event collection, which lacks the chunk fields the reader queries. Live schema probes confirmed that mismatch; the compose now matches the existing reader and publisher contract.
- The existing extraction reader lacked SELECT on `working.extraction_run`, `working.candidate_entity` and `working.candidate_event`. Applied only those grants from the already-tracked `sql/bootstrap/workbench_extractions_20261002.sql`; verified SELECT true and INSERT/UPDATE/DELETE/TRUNCATE false on all three. The existing thread extractions endpoint then returned 200. No data rows changed.
- Conversation headings use returned participant labels rather than internal thread keys. Empty original records remain intact and show a compact explanation; original IDs and citation links are preserved.

## Validation and release

- Code through `d11c42a2` pushed to `main`; auto-deploy remains disabled.
- Production TypeScript/Vite build and Storybook build passed. ESLint: zero errors, 26 existing warnings.
- Combined web suite: 162 passed, four browser checks skipped on Windows by policy. Focused search/link checks passed after the last change.
- First manual Coolify deployment `yqfopjuznf1xmknugb6s9swo` finished and application `xjbuo6drbwjfby75lalk8bk7` was healthy. Browser interactions passed 12/13 journeys and exposed the Read search failure; runtime checks additionally exposed missing extraction grants. Both failures were investigated and corrected, rather than counting page loads as proof.
- Follow-up: four focused API tests and 16 Read tests passed; production build and audit self-test passed. Second manual deployment `hyfr4imenjulq9wn8p1cmrpw` finished and was healthy. Browser pass two verified search-to-exact-message and extracted context; 13/14 journeys passed, with an intermittent Case identity 503.
- Three subsequent Case identity reads succeeded in 7.38–8.31 seconds. The engine's whole-page read includes expensive aggregate queries under an eight-second per-statement timeout; this is a source-level explanation to investigate, not proof of the exact statement behind the observed failure. Added an explicit Retry action to the Case error state.
- Third manual deployment `w97lhiccaj5wrjrcopxsc0d8` finished and was healthy. This carries the Case Retry action. The audit now additionally clicks into Sources folders, reloads the durable location, returns to root, searches indexed content and opens the returned excerpt/locator. Final browser outcome is recorded below.
- Final browser pass, 2026-10-06 EDT (2026-10-07 UTC): **15 of 16 checks passed**. Sources folder navigation/reload/root return, indexed search/excerpt, Activity filters/details, Read source/thread/message citations, indexed search-to-exact-message, extracted context, preview return, legacy links and read-only network guard passed. Case navigation failed again with registry unavailability. This is partial verification, not a clean full-app pass.
- Read-only timing of the exact tracked engine queries, using current identifiers: `caseCountsSQL` 7.06 s (56 aggregate rows), `caseUnknownsSQL` 1.26 s (100 rows). The counts query is close to the engine's eight-second statement limit; no timeout setting, engine query, schema or source data was changed. The failing request's exact server exception was not captured, so timing pressure remains the evidenced likely cause rather than a proven exception trace.
- All browser request-guard/auth/interception/runtime-error counters were zero. Screenshots and exact user-data-bearing output are retained only under the private worktree's untracked `web/.verification/live-20261006{,-r2,-r3}/` and the existing devbox audit directories; none is committed or published.
- `deploy/workbench-audit/workflow.mjs` performs read-only Chrome interactions in the existing devbox on ovh-files; no desktop browser. It blocks job/gate/data mutations and writes redacted outcomes plus private screenshots.

## Specific remaining gaps

- Relationship search cannot safely derive a graph record from the current search response: occurrence identity depends on a snapshot absent from the hit. The former UI silently substituted hybrid search. It now reports that the graph link is unavailable instead of returning unrelated results as relationships.
- Imported document/media content has no generalized reader contract: imported-source rows lack the root/key/etag needed by the existing original-byte endpoint. Existing Sources previews and processing previews remain available. No bucket, identifier or citation mapping was invented.
- Read's chunk index currently contains ten conversation chunks, versus 233,486 raw event objects in the separate older collection. All ten first-message locators and all three referenced source versions resolve in PostgreSQL. These are different indexing units, not a coverage percentage. The UI describes indexed-chunk search and does not claim an empty result means no original messages match.
- Case's full identity read has shown intermittent unavailability. Retry is now available, but query performance is not claimed repaired by that UI change.
- These UI changes do not establish full corpus indexing, complete AI-chat extraction, disposable Dev workspaces, an independent feature-flag registry, or successful processing of every source. Those backend tasks retain their own verification requirements.
- No genuine records were changed by validation. No ephemeral test rows were created; none require cleanup. Browser validation blocks processing, cancellation and decision mutations; those paths received local contract tests, not a live end-to-end mutation claim.
