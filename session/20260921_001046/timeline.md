# Timeline

> _Byline: Claude Code · Fable 5.1 · 2026-09-21. Terse; the dated detail is in Probata `docs/planning/2026-09-20-TODO.md`._

## 2026-09-20 23:28–23:40 EDT
- Owner decided: thread oldest-first; More menu; extraction before edit API; media in two modes (SBV base64 before processing, derived media folder after).
- Built the message-ledger dashboard from real aggregates; rendered headlessly in Chrome, fixed number formatting/wrapping; sent as a local file.
- Agent `derive-activity-routing` finished: found it duplicates main's `derive_sms_threads_activity` (`2817d82`). Not merged. Logged (`77f6537`).
- Owner "b2?" on Intake: B2 roots are live but only under the Direct storage tab; Indexed catalog tab lists original occurrences. Not changed (other session's file, catalog off limits).

## 23:50–00:06 EDT
- Owner: derived output in a separate top-level vault directory mirroring the source tree; automatic derive is "the whole point"; batching by folder, one at a time.
- Dispatched Opus agent `derive-reconcile` (reconcile + `DERIVED_ROOTS_JSON` + start-batch). Lane claimed in the log (`78d8023`).
- Relayed the other session's 23:48 one-viewport ruling to `sbv-port`; spec amended.
- Wrong-chat paste (Legal Work Desk handoff) — owner said disregard; nothing done.
- Vault top-level names given; no numbers; not built; "make it configurable" (`d8c1be1`).

## 00:06–00:18 EDT
- Owner pasted the `DerivedKnowledge/messaging/` design: readable Markdown layer, not parser output. Saved verbatim under `docs/transcripts/`; log corrected (`29a72a2`). Agent told to keep vault names out of code/examples.
- CNF stores consolidated losslessly (backups `.bak-<stamp>` beside each file).
- Owner: cross-platform merged conversations later (`a64e2f5`).
- Owner "yes go": `execution_path` CHECK widened live — dry-run with ROLLBACK, applied, read back as `platform_runtime` (15 rows, 0 invalid). Receipt pushed (`f9d4091`).
- `/session-memory` checkpoint written.
