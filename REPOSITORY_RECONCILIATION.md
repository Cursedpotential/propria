# TraceIQ Repository Reconciliation Resolution

> _Naming (D-140, 2026-09-05; applied 2026-09-06): this product is **vestigia** (formerly traceIQ / TraceIQ - Latin: footprints, tracks). Working copy: `probata/modules/vestigia/` (directory rename from `modules/traceIQ/` landed 2026-09-06; old name kept as a junction). GitHub repo name unchanged pending its own decision. Canon: `probata/docs/NAMING.md`. Historical text below is left verbatim; both names remain valid in recall stores (D-142)._


> _Byline: Codex · GPT-5 · 2026-08-27; resolved by Codex · GPT-5 · 2026-08-29._

**Status: RESOLVED — CHILD CLEAN/PUSHED; PARENT RAW GITLINK VERIFIED**

## Verified boundary

- Repository root: `E:\AI_Workspace\Projects\traceIQ`
- Current branch: `main`, tracking `origin/main`
- `traceiq-rebuild/` is an independent nested repository and is outside this repository's commit
  scope.

## Resolution evidence

- The worktree contained 2,088 missing tracked paths and 9,487 untracked paths with zero exact
  path collisions.
- The missing tracked paths were restored additively from the existing child `HEAD`; no present
  file was overwritten or deleted.
- Local-only tool state, timelines, backups, evidence JSON, debris, and the nested rebuild were
  explicitly ignored in child commit `d670b3de3f92f7547e876a7f1193573c01c70f4d`, pushed to
  `origin/main`.
- Fourteen formerly parent-owned local data files were SHA-256 manifested and remain physically
  present. The workspace parent replaced 15 mixed entries with one raw Gitlink in parent commit
  `9cfde017`.
- Full hashes and isolated-index proof are retained at
  `../repository-boundary-receipts/TRACEIQ-CONVERSION-2026-08-29.md` from the `Projects/` directory.

## Continuing rules

- Commit outer TraceIQ source only from this root with an explicit allowlist.
- Do not commit ignored private data or the nested repository.
- Never reset, clean, stash, overwrite, or hard-delete unrelated work.
- Work in `traceiq-rebuild/` only from its own repository root and instructions.
