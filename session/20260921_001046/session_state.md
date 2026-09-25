# Session State

> _Byline: Claude Code · Fable 5.1 · 2026-09-21._

- Session: 20260921_001046 (Claude session 77aa963a, started 2026-09-20 ~09:56 EDT)
- Repo: `E:/AI_Workspace/Projects/Propria` (router) — work repo is Probata: worktree `E:/AI_Workspace/Projects/Propria/_worktrees/probata-preview-mode-recovery`
- Branch: detached at `origin/main` (docs pushed as fast-forwards with `git push origin HEAD:main`)
- Updated: 2026-09-21 00:18 EDT

## Goal
Get the Probata Workbench (Intake -> Review) working end to end on real vault files from B2: a source runs through the proffer pipeline, and the owner can actually read his messages in Review. Owner: "stay on probata".

## Current Subtask
Supervising four Opus agents (search/calls/media endpoint; SBV-borrowing one-viewport Review; derive reconcile + derived-root config + batch-by-folder) and bringing their findings to the owner before any merge.

## Loaded Skills
- `interface-design`, `dataviz`, `data:build-dashboard` — used for the local message-ledger dashboard (done, sent).
- `design:design-handoff` — produced `docs/design/2026-09-20-review-message-surface-handoff.md` (amended with owner decisions).
- `karpathy-guidelines`, `think:thinking-model-router` — standing behaviour.
- `session-memory` — this checkpoint.

## Current Status
- Live and verified earlier: B2 DuckDB secret, DuckDB role gate, first 26/26 run, derive tool on 110 MB and 584 MB backups, Review pages validate, compact Sources list, message browser deployed, Docstore 0.8.1 healthy, Coolify deploys manual.
- Tonight: owner decisions recorded (oldest-first, More menu, extraction before edit API, two media modes, one-viewport layout, derived location = config, vault top-level names, `messaging/` = readable layer, cross-platform merges later); `execution_path` CHECK widened live (receipt `docs/pending-review/2026-09-21-derive-execution-path/`); dashboard built locally (not in git — phone numbers); CNF stores consolidated with backups.
- NOT done: nothing from the agents is merged/deployed; in-app browser cannot reach the tailnet, so UI is verified via API/deployed chunks only.

## Plan
- [ ] Review each agent branch on completion; report to owner; merge + deploy one app at a time on approval
- [ ] Reconcile main's schema snapshot with the live three-value CHECK at derive merge
- [ ] Fix own-number participant bug in derive; confirm the 5,000-record question with the owner
- [ ] Then: extraction stage design (options to owner), render step

## Assumptions
- "yes go" (00:09) was the approval for the DB constraint — stated to the owner; change is harmless if it meant something else.

## Blockers
- None technical. Merges wait on agent reports + owner review.
