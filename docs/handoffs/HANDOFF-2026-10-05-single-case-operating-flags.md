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
- Workbench: `codex/single-case-workbench-20261005`; recovery agent
  `single_case_workbench_recovery_20261005`, Workbench source/tests only. Original
  agent's model failed capacity twice; its edits were preserved in this worktree.
- Legal consumer: `codex/single-case-legal-consumer-20261005`; bounded config,
  read-through, compose and tests. Agent reported 27 tests passing, then capacity
  interrupted final lint/commit. Parent independently ran the 27 tests (all
  passed in 0.96s), checked the diff and committed `e3f1c92a`; integrated as
  `062184fa`. Independent review found no bounded Legal blocker. This remains
  mocked consumer proof, not served-surface verification.
- Promotion-review server: agent `single_case_promotion_server_20261005`,
  `server/api/inspect_routes.py` and focused tests only. Committed `52e7237d`,
  integrated as `0a89ec22`; 12 focused tests, full inspect suite 64 passed/4
  skipped with ephemeral SQLAlchemy asyncio extra, Ruff passed. Parent review
  subsequently found same-case IDs alone still do not prove durable Live mode;
  initial admission-event validation and rejection tests are being added.
- Python chunk queue: `single_case_python_chunk_guard_20261005`, separate clean
  worktree from fresh `c87aa233`; owns Python chunk publish/removal guards,
  coordinated request dataclasses and thin starter, plus focused tests only.
  Go owns matching request fields. No Live publish/removal has been invoked.
- Independent review: `single_case_independent_review_20261005`, read-only;
  temporarily interrupted to free a builder slot. Resume after a builder finishes
  for final combined review. Reported gaps are not accepted as resolved merely
  because a builder started changing them.
- Parent integration: `codex/single-case-integration-20261005`, root instructions,
  current decision/supersession, deployment manifest and serial integration.

Initial engine/Workbench/Legal lanes began at verified `origin/main` commit
`982cd942cf2324cf80a9fbd85631f271b16858cf`; promotion-server starts at refreshed
`741a654ddcd89b05082a73ed3fe9e243b7d05250`. Parent integration merged that fresh
origin (R2 acquisition retirement and toolkit ZIP source changes) as `9834e35b`.
Root policy/neutral manifest commit is `0cbdaed9`. Shared main is concurrently dirty,
including unrelated Family Court and engine Activity changes. Never reset,
clean, stash, overwrite or stage those changes.

Parent refreshed and merged upstream `c87aa233` (new independently tracked AI
work-product placement Activity/worker registration), integrated as `4613b111`.
The neutral exec API manifest is committed as `2555815f`; the actual Coolify
app uses `/deploy/exec.yaml` and has no PROFFER identity env keys as of this
read-only check. Neutral pair configuration is therefore a rollout prerequisite.

## Agreed protocol

Canonical `DEV`/`LIVE`, default Live. `operating_mode` is explicit in
`POST /reference-import/start` and `GET /reference-import/operations/{handle}`.
Batch start/child input also carries `operating_mode` explicitly.
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
- Durable owner decision is saved and independently read back: Docstore
  `note:single_case_operating_flags_20261005`, document generation 1/hash
  `d767660aa05c36d53ac8ee5f2e99f476a8d2e832ee7fe521d953a4880f6b78d3`,
  critical owner-decision flag revision 1, active across eight domains. The
  captured source document remains unapproved/unindexed; do not confuse the
  critical owner flag with source-revision approval or semantic indexing.
  Do not rewrite built-in memory files.
- Push scoped integration and merge safely; automatic Coolify deployment stays
  off. Manually deploy affected starter, worker, Workbench and Legal services.
- Read-only live probes through supported doors: same approved IDs for Dev,
  Live and default; unknown mode rejection; explicit operation mode readback.
  Prove Dev zero writes/dispatch in unit tests before any risky route probe.
- Save deployment/surface receipt; distinguish completion from local proof.
- Disposable Dev data workspace remains a separate owed feature, not done.

## Open review/verification checkpoints

- Durable missing mode must not default Live; fresh HTTP request omission may.
- Workflow/batch direct entry must admit the exact approved pair before Activity.
- Old Temporal histories need version/replay-compatible command preservation and
  an execution fence on scheduled/retrying write Activities. Prove replay plus
  zero Activity body calls for unknown/Dev mode; fresh workflow tests alone are
  insufficient. No mass cancellation, purge or identity-derived inference.
- Live extraction commit check must accept canonical LIVE; stale REAL fixtures
  must not hide a regression. Header edits reject unrelated court children.
- Case store/direct entry and every Case UI mutation retain explicit policy;
  mobile and stale dialogs cannot silently force Live after an explicit Dev flag.
- Neutral Workbench config is validated against the engine's authoritative
  header; do not duplicate approved UUID constants in Python or trust arbitrary
  configured UUIDs as approval.
- Python promotion-review flag API must accept canonical wire values and stop
  selecting identity by mode. This content-review flag is not a feature flag.
- Python conversation-chunk publishers and non-dry-run removal need their own
  explicit durable Live + approved-scope fence before heavy imports/body calls.
  Go guards do not protect this separate queue. Historical missing mode stays
  unknown; safe read/dry-run Activities remain separate. Neutral case envs must
  be supplied to the Python worker before any deployment. No purge/cancellation
  is authorized.
- Engine provisional checkpoint `25d241aa` is not accepted. Existing-VPS pass 1
  passed caseidentity and extraction/commitcheck but found a repairplan test
  mode mismatch, a stale repairplan.Environment field causing runtimeapi test
  compilation failure and Proffer failures requiring focused diagnosis. Some
  other failures were test-archive omissions (SQL/deploy/fixture files), not
  application defects; correct the source packet before claiming test results.

## Verified server test runtime

On 2026-10-05, existing ovh-files Coolify devbox container
`devbox-pd3xc78ahqkfswq12bpfqgy1-145003736857` exposes
`/usr/local/go/bin/go`, Go 1.27.1 linux/amd64. Host bind
`/data/probata/volumes/devbox/root` is container `/root`; `/data/test_data` is not
mounted. Use a fresh source-only test directory under
`/root/single-case-flags-20261005/`, tracked-source archive and bounded concurrency.
Never set integration-test DB DSNs, create a parallel stack or restart the host.

## Tool recall evidence and limitation

Named read-memories query found the owner's October 2 feature-flag intent.
Docstore recalled historical D-126 and D-127/D-128; current owner supersedes
identity switching. Federation covered Claude/Codex, CNF, .remember and
memsearch with bounded/partial coverage; direct Docstore recall was separate.
Smart Explore mapped the identity selectors. CCC semantic query was blocked by
an unclean protected-run lock, not bypassed; no successful semantic result is
claimed. No source-data reset, schema rebuild, auth expansion or host restart
is authorized by this handoff.
