# Single-case operating flags: active handoff

Byline: Codex, orchestrator, 2026-10-05. Status: implementation in progress;
not merged, pushed, deployed or verified complete.

## Authority and invariant

Owner explicitly instructed fixing mode-dependent case identities. Use one
approved case for Dev and Live, Live by default, separate feature flags. Do not
restore the superseded D-126 placeholder/reset policy. The canonical decision
is `docs/decisions/2026-10-05-single-case-operating-flags.md`.

## Current owned lanes

- Engine: `codex/single-case-engine-20261005` worktree under `_worktrees/`;
  agent `single_case_engine_build_20261005`, engine source/tests only.
- Workbench: `codex/single-case-workbench-20261005`; agent
  `single_case_workbench_fix_20261005`, Workbench source/tests only. Capacity
  interrupted one turn; edits preserved and agent resumed.
- Legal consumer: `codex/single-case-legal-consumer-20261005`; bounded config,
  read-through, compose and tests. Agent reported 27 tests passing, then capacity
  interrupted final lint/commit. Parent must independently verify and commit.
- Parent integration: `codex/single-case-integration-20261005`, root instructions,
  current decision/supersession, deployment manifest and serial integration.

All began at verified `origin/main` commit
`982cd942cf2324cf80a9fbd85631f271b16858cf`. Shared main is concurrently dirty,
including unrelated Family Court and engine Activity changes. Never reset,
clean, stash, overwrite or stage those changes.

## Agreed protocol

Canonical `DEV`/`LIVE`, default Live. `operating_mode` is explicit in
`POST /reference-import/start` and `GET /reference-import/operations/{handle}`.
Engine persists it in initial preview-event detail and Temporal input without
schema changes. Old/unverifiable bindings fail mutation authorization closed;
mode is never inferred from matter ID. Dev canonical mutations fail before
dispatch/persistence until isolated workspace implementation exists.

## Remaining checklist

- Finish/commit each scoped builder, inspect allowlists and independent tests.
- Audit mode propagation on every Case page write, not just header edits.
- Integrate serially in the parent worktree; preserve fresh origin/main changes.
- Neutral Workbench env names and compose: `PROFFER_MATTER_ID` /
  `PROFFER_COURT_CASE_ID`; preserve approved pair and independent auth settings.
- Durable owner decision in Docstore through governed capture/flag tools,
  independently read back. Do not rewrite built-in memory files.
- Push scoped integration and merge safely; automatic Coolify deployment stays
  off. Manually deploy affected starter, worker, Workbench and Legal services.
- Read-only live probes through supported doors: same approved IDs for Dev,
  Live and default; unknown mode rejection; explicit operation mode readback.
  Prove Dev zero writes/dispatch in unit tests before any risky route probe.
- Save deployment/surface receipt; distinguish completion from local proof.
- Disposable Dev data workspace remains a separate owed feature, not done.

## Tool recall evidence and limitation

Named read-memories query found the owner's October 2 feature-flag intent.
Docstore recalled historical D-126 and D-127/D-128; current owner supersedes
identity switching. Federation covered Claude/Codex, CNF, .remember and
memsearch with bounded/partial coverage; direct Docstore recall was separate.
Smart Explore mapped the identity selectors. CCC semantic query was blocked by
an unclean protected-run lock, not bypassed; no successful semantic result is
claimed. No source-data reset, schema rebuild, auth expansion or host restart
is authorized by this handoff.
