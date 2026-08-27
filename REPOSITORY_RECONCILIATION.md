# TraceIQ Repository Reconciliation Hold

> _Byline: Codex · GPT-5 · 2026-08-27._

**Status: FAIL-CLOSED HOLD — NOT RECONCILED**

## Verified boundary

- Repository root: `E:\AI_Workspace\Projects\traceIQ`
- Current branch: `main`, tracking `origin/main`
- `traceiq-rebuild/` is an independent nested repository and is outside this repository's commit
  scope.

## Observed condition

The restored outer worktree reports a very large set of tracked paths as deleted while also
containing substantial untracked material. No conclusion has been made about whether those paths
were intentionally relocated, are recoverable equivalents, or represent divergent generations.
The status must not be normalized mechanically.

## Hold rules

Until an owner-approved reconciliation plan clears this hold:

- Do not run checkout, restore, reset, clean, stash, or deletion commands.
- Do not stage or commit any reported deletion.
- Do not use `git add .`, `git add -A`, wildcard staging, or bulk path staging.
- Do not move, rename, overwrite, or bulk-copy tracked or untracked material.
- Do not absorb, stage, or rewrite the history of `traceiq-rebuild/`.
- Do not infer equivalence from matching filenames; use content hashes and provenance in a later
  read-only inventory.

Read-only inspection is allowed. The owner-authorized additive governance set (`AGENTS.md`,
`AGENT_MEMORY.md`, `CLAUDE.md`, and this hold document) may be staged and committed by exact
allowlist. No reported deletion or other pre-existing worktree change may enter that commit.

## Required reconciliation before release

1. Capture an owner-reviewed inventory of tracked-missing, present-untracked, ignored, and nested
   repository paths without exposing evidence contents.
2. Classify each path by custody, provenance, sensitivity, and intended repository ownership.
3. Hash-compare candidate equivalents and document conflicts without overwriting either side.
4. Obtain an explicit owner decision for restoration, archival, migration, or intentional removal.
5. Stage only the approved allowlist and verify the staged diff from this exact Git root.
6. Update this document with the resulting commit identifiers and remove the hold only after the
   live worktree and remote relationship are reverified.
