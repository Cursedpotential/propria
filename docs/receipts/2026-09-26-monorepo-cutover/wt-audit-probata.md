# Probata worktree audit — retirability for monorepo import

- Repo: `E:/AI_Workspace/Projects/Propria/modules/Probata/probata` (git root, remote `origin` = `Cursedpotential/probata.git`, branch `main`)
- Method: `git fetch origin` (confirmed up to date against GitHub), then `git worktree list --porcelain`, then per-worktree `status`/`rev-list`/`merge-base`/`branch -r --contains`/`log`. Read-only throughout - no commits, checkouts, merges, or worktree removals were performed.
- Snapshot locked at: **2026-09-26 18:30:20 EDT** (script run completed 18:29:56-18:30:20 EDT). This repo is **actively shared by other concurrent sessions** (see Volatility note below) - treat this table as a point-in-time snapshot, not a guarantee, and re-run the same checks immediately before actually removing `.git`.
- 27 linked worktrees found (plus the main checkout, skipped per instructions) = 28 total `git worktree list` entries, consistent with the "~27" estimate.

## Table

| path | branch | bucket | dirty | ahead | merged | pushed | last commit |
|---|---|---|---|---|---|---|---|
| `_worktrees/probata-ci-ruff-baseline-20260923` | `codex/ci-ruff-baseline-20260923` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-23 docs(workbench): record R2 key and REAL identity blockers |
| `_worktrees/probata-derive-sms-activity` | `feat/derive-sms-activity` | RETIRABLE | 0 | 0 | MERGED | yes, = origin | 2026-09-21 Merge remote-tracking branch "origin/main" into feat/derive-sms-activity |
| `_worktrees/probata-entities` | `feat/entities` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-25 fix(entities): move formatWhen out of the RecordPeek component file |
| `_worktrees/probata-integrate-20260926` | `integrate/repair-entities-20260926` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-26 docs(todo): 2026-09-26 decisions, merges, deploys, outage and fixes |
| `_worktrees/probata-metadata-context` | `feat/metadata-context` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-26 chore(review): align the detail-panel and engine package-map edits with the integration branch |
| `_worktrees/probata-no-glm` | `chore/no-glm` | RETIRABLE | 0 | 0 | MERGED | yes, = origin | 2026-09-25 feat(workbench): one INFO log line per model attempt (model, mode, prompt size, outcome, latency) |
| `_worktrees/probata-preview-mode-recovery` | *(detached @ eac18a51)* | RETIRABLE | 0 | 0 | MERGED | yes - commit is on origin/main itself + 12 other origin branches | 2026-09-22 test(workbench): cover the batch passthrough, unit marks and decode-state check |
| `_worktrees/probata-repair-engine` | `feat/repair-engine` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-25 fix(engine): bind after registration, re-entry link receipt, zero-fill guard |
| `_worktrees/probata-repair-workbench` | `feat/repair-workbench` | RETIRABLE | 0 | 0 | MERGED | no remote branch, but see note 1 | 2026-09-26 docs(todo): repair builder Workbench side built on feat/repair-workbench, not pushed |
| `_worktrees/probata-review-actions` | `ops/ovh-files-data-dir-20260926` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-26 ops(ovh-files): move /data onto the 500 GB disk as a bind mount (warm copy, short stop, verify, swap) |
| `_worktrees/probata-review-message-browser` | *(detached @ eb8d8b58)* | RETIRABLE | 0 | 0 | MERGED | yes - commit is on origin/main + 17 other origin branches | 2026-09-20 test(workbench-web): resource-list contract reads the new component; readback label kept on each row |
| `_worktrees/probata-workbench-resume-20260920` | `fix/workbench-operations-route` | RETIRABLE | 0 | 0 | MERGED | no remote branch (moot - merged) | 2026-09-20 docs(planning): bucket-wide B2 key, whole-bucket root, desktop rclone and .secrets updated, superseded key revoked |
| `_worktrees/probata-ci-baseline-unblock` | `codex/ci-baseline-unblock-20260923` | PUSH-THEN-RETIRE | 0 | 1 | UNMERGED | yes, = origin | 2026-09-23 style(ci): restore Ruff formatter baseline |
| `_worktrees/probata-d02-pb06-bounded-xml` | `codex/d02-pb06-bounded-xml` | PUSH-THEN-RETIRE | 0 | 2 | UNMERGED | yes, = origin | 2026-09-23 test(proffer): review bounded XML probe edge cases |
| `_worktrees/probata-d04-zip-inventory` | `codex/d04-zip-inventory-20260923` | PUSH-THEN-RETIRE | 0 | 4 | UNMERGED | yes, = origin | 2026-09-24 fix(d04): admit sealed ZIP objects and reject ZIP64 entries |
| `_worktrees/probata-d05-exact-review-targets` | `codex/d05-exact-review-targets-20260923` | PUSH-THEN-RETIRE | 0 | 11 | UNMERGED | yes, = origin | 2026-09-24 Record PR30 replay fix and exact source scan lineage |
| `_worktrees/probata-d06-workproduct-atoms` | `codex/d06-workproduct-atoms-20260923` | PUSH-THEN-RETIRE | 0 | 5 | UNMERGED | yes, = origin | 2026-09-24 docs(work-product): record Unicode span remediation |
| `_worktrees/probata-d07-walk-schedule` | `codex/d07-walk-schedule-20260923` | PUSH-THEN-RETIRE | 0 | 2 | UNMERGED | yes, = origin | 2026-09-23 docs(indagatio): record PR validation hold |
| `_worktrees/probata-d08-legal-source-package` | `codex/d08-legal-source-package-contract-20260923` | PUSH-THEN-RETIRE | 0 | 5 | UNMERGED | yes, = origin | 2026-09-24 Require Ed25519 signature wire length for legal packages |
| `_worktrees/probata-d11-context-export` | `codex/d11-context-export-contract-20260923` | PUSH-THEN-RETIRE | 0 | 3 | UNMERGED | yes, = origin | 2026-09-24 Narrow optional D11 manifest values for type safety |
| `_worktrees/probata-w18-enrichment-fault` | `codex/w18-enrichment-fault-20260923` | PUSH-THEN-RETIRE | 0 | 4 | UNMERGED | yes, = origin | 2026-09-24 docs: record PR 37 review remediation |
| `_worktrees/probata-d04-first-party-thread` | `codex/d04-first-party-thread-20260924` | NEEDS-WORK | 0 | 1 | UNMERGED | no remote branch | 2026-09-24 docs(d04): record first-party projection schema hold |
| `_worktrees/probata-drop-claudemd` | `docs/drop-claude-md-importer-20260926` | NEEDS-WORK | 0 | 1 | UNMERGED | no remote branch | 2026-09-26 docs(agents-md): drop the CLAUDE.md importer so AGENTS.md loads natively |
| `_worktrees/probata-openlist-consignatio` | `fix/openlist-consignatio-mounts` | NEEDS-WORK | 1 (1 modified, 0 untracked) | 0 | MERGED | no remote branch | 2026-09-25 docs(todo): entities hand-back - built scope, event-staging decision, follow-ups |
| `_worktrees/probata-preview-search-calls` | *(detached @ d3490220)* | NEEDS-WORK | 0 | 1 | UNMERGED | no - not on any remote branch | 2026-09-22 fix(workbench): real catalog unit joins, readable state/unit columns |
| `_worktrees/probata-recover-20260920` | *(detached @ 1c438deb)* | NEEDS-WORK | 650 (649 modified, 1 untracked) | 0 | MERGED | yes - commit on origin/main + 22 others (working tree is not) | 2026-09-15 docs(federation): record octopoda gateway/virtual-server addition in section 8 |
| `_worktrees/probata-sources-screen` | `feat/sources-screen` | NEEDS-WORK | 0 | 1 | UNMERGED | remote branch exists but is stale/behind local | 2026-09-22 fix(workbench): real catalog unit joins, readable state/unit columns |

No worktrees fell into **UNTRACKED-ONLY** (every dirty worktree had at least one modified tracked file, not untracked-only) or **BROKEN** (every path in the final snapshot existed and was a valid, listed worktree) at snapshot time.

Note 1: `probata-repair-workbench`'s own last commit says "not pushed," referring to the branch pointer - but `git merge-base --is-ancestor HEAD origin/main` confirms HEAD is a true ancestor of `origin/main` (its commits reached `origin/main` through another path, e.g. someone merged the work into local `main` and pushed `main` directly without ever pushing this branch). Verified mechanically, not assumed.

## Summary

**Counts (27 total):**
- RETIRABLE: 12
- PUSH-THEN-RETIRE: 9
- NEEDS-WORK: 6
- UNTRACKED-ONLY: 0
- BROKEN: 0

**NEEDS-WORK - every entry and what's at risk:**

1. **`probata-d04-first-party-thread`** (`codex/d04-first-party-thread-20260924`) - 1 commit ("docs(d04): record first-party projection schema hold") exists only in this worktree: not merged into `origin/main`, no `origin/<branch>` exists. Deleting it without pushing loses that commit.
2. **`probata-drop-claudemd`** (`docs/drop-claude-md-importer-20260926`) - 1 unpushed commit ("docs(agents-md): drop the CLAUDE.md importer..."), no remote branch. **Also observed changing live during this audit** (HEAD advanced from one commit to a new one between two checks a few minutes apart) - another session is actively committing here right now; re-check immediately before any action.
3. **`probata-openlist-consignatio`** (`fix/openlist-consignatio-mounts`) - branch itself is fully merged, but has 1 uncommitted modified tracked file, `deploy/openlist.yaml`, that would be silently destroyed if the worktree is removed.
4. **`probata-preview-search-calls`** (detached @ `d3490220`) - 1 commit ("fix(workbench): real catalog unit joins...") not on any remote branch. The identical commit is also the HEAD of branch `feat/sources-screen` (worktree `probata-sources-screen`, see #6) - so this content currently survives only because two different worktrees happen to hold the same unpushed commit; it is not safe on its own.
5. **`probata-recover-20260920`** (detached @ `1c438deb`) - the commit itself is safe (already on `origin/main` and 22 other branches), but the working tree carries **650 uncommitted changes (649 modified + 1 untracked)** across `.py`/`.md`/`.go`/`.tsx`/`.sh`/`.surql`/`.json` files, including a `.review_hold/` staging area with names like `naming-sweep-verify_*` - this looks like a large in-progress repair/consolidation run. This is the highest-value at-risk data found in the audit. **This worktree did not exist under this name at the start of the audit** - it appeared (replacing a worktree called `probata-smoke-browser-free` that vanished during the same window) while this audit was running, confirming another session is actively working in it right now. Do not touch without direct confirmation from whoever is driving it.
6. **`probata-sources-screen`** (`feat/sources-screen`) - local HEAD is 1 commit ahead of its own `origin/feat/sources-screen` (which is stale, pointing at an older commit `eac18a51`). That 1 commit is the same `d3490220` described in #4 - push `feat/sources-screen` (or cherry-pick the commit somewhere durable) before retiring either this worktree or `probata-preview-search-calls`.

**Other findings worth flagging (outside the strict per-worktree scope):**
- The **main checkout's local `main` branch is itself 6 commits ahead of `origin/main`** (unpushed). This audit correctly used `origin/main` throughout (per the task's exact instructions, and because it's the durable copy on GitHub), so no bucket above is affected - but before the monorepo import removes this repo's `.git`, local `main` should be pushed too, or those 6 commits are also at risk.
- **This repository is under active concurrent use by at least one other session** during the audit window: worktree `probata-drop-claudemd` gained a new commit, worktree `probata-smoke-browser-free` (3 dirty files) had its dirty state change three times and then the worktree itself disappeared, replaced by a new `probata-recover-20260920` worktree carrying 650 dirty files, all within about 5 minutes of wall-clock time. The table above reflects the last full snapshot (18:30:20 EDT); re-run the same checks immediately before actually executing the import.
- **`origin/main` itself moved during this audit.** A follow-up `git fetch origin main` at 18:35 EDT (after the table snapshot) picked up a brand-new commit `51424e7` "docs: recover 61 docs lost in the 2026-09-20 git consolidation," pushed at 18:32:44 EDT - i.e. after this audit's `origin/main` reference was fixed. This is consistent with the `.review_hold`/`naming-sweep` recovery work seen live in `probata-recover-20260920` above, and is further, concrete proof that both the worktree set and `origin/main` are live targets right now, not settled state. It does not change any bucket above (a later commit on `origin/main` can only turn UNMERGED into MERGED, never the reverse), but it means a re-run minutes from now will likely show more of the PUSH-THEN-RETIRE / NEEDS-WORK rows flip to RETIRABLE - re-run before trusting counts.
