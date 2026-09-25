# Files

> _Byline: Claude Code · Fable 5.1 · 2026-09-21. Paths relative to the Probata repo unless absolute._

## Changed (pushed to Probata `main` tonight)
- `docs/planning/2026-09-20-TODO.md` — THE log; entries 23:40 through 2026-09-21 00:16.
- `docs/design/2026-09-20-review-message-surface-handoff.md` — owner decisions, oldest-first, two media modes, one-viewport ruling.
- `docs/transcripts/2026-09-21-unknown-assistant-derivedknowledge-messaging-layout.md` — owner-pasted vault design, verbatim (source service unknown).
- `docs/pending-review/2026-09-21-derive-execution-path/` — receipt + applied SQL for the live CHECK change.

## Inspected
- `modules/workbench/api/app/types/source_roots.py` — `SOURCE_ROOTS_JSON` / `OBJECT_STORES_JSON`; live value has `b2-vault`, `b2-bucket`, three R2 roots.
- `modules/workbench/web/src/components/intake/discovery-explorer.tsx` — Indexed catalog vs Direct storage tabs (other session's work).
- `docs/decisions/2026-09-20-bulk-intake-owner-requirements.md` — bulk-intake requirements and build order (other session).
- `docs/pending-review/2026-09-20-review-screen-layout-diagnosis.md` — one-viewport Review ruling (other session).
- `modules/engine/activities/derive_sms_threads.go` (main) vs `activities/derive_structured_text.go` (branch) — the duplicate.

## Generated (local, NOT in git)
- `C:/Users/matts/AppData/Local/Temp/claude/E--AI-Workspace-Projects-Propria/77aa963a-4603-4612-8266-de053f052a48/scratchpad/message_ledger.html` — dashboard (contains phone numbers); builder `build_message_ledger.py`; data `%TEMP%/dash_data2.json`.
- `…/scratchpad/cnf_fold_20260921.py` — CNF consolidation script; backups `project_memory.json.bak-<stamp>` beside both CNF files.

## Agent worktrees (do not edit while agents run)
- `_worktrees/probata-preview-search-calls` — `feat/preview-search-and-calls`
- `_worktrees/probata-derive-sms-activity` — `feat/derive-sms-activity`
- `_worktrees/probata-review-message-browser` — has node_modules; used for tsc/lint/smoke
