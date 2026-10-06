# Single-case operating flags: active handoff

Byline: Codex, orchestrator, 2026-10-06. Resumed on explicit owner instruction.
Draft PR #2 is open. Not merged into main, deployed or verified complete.

## Current checkpoint — 2026-10-06 resume

This checkpoint supersedes the older progress and acceptance snapshots below.
The three interrupted builder lanes stopped on account usage-limit errors.
Current account inspection permits ordinary work; all three were resumed.

- Source integration tip `bf183d1f` preserves current upstream `cf83d7a7`.
  The sole merge conflict was independent imports in
  `postgres/first_party_context_store.go`; both were retained. No shared-checkout
  changes were staged, reset, stashed or overwritten.
- Actual source-court protection `ed7501c1` is integrated as `59d0e781`.
- Workbench canonical-upload/R2-new-acquisition denial `9260ca0e`, corrected B2
  fixture `0e7b225b`, narrow comments `ea97cdcf` and bounded scope consumer
  `d902ef2d` are integrated as `59f1df72`, `f6c13b4c`, `b3b2f9e6` and `ddf19944`.
  The fresh source-pinned Workbench suite reports **676 passed, zero failures**;
  focused suite 253 passed. Web lint has zero errors/26 existing warnings;
  build/typecheck pass. Final smoke/Storybook receipt is being recovered.
- Python bounded scope consumer `5f2d1836` is integrated as `b11ee5a3`.
  Agent's combined synthetic suite reports **352 passed, five existing skips**;
  lint, formatting and bounded helper typecheck pass. Parent rerun against the
  newer upstream source is pending.
- Engine bounded scope provider `60953696` is integrated as `bf183d1f`.
  It adds authenticated `/case-identity/scope`, the existing approved registry
  pair and actual court parent, no aggregate reads, and a total two-second
  deadline before pool acquisition. Focused source-pinned proof is running;
  parent will rerun the combined engine including current upstream changes.
- An independent read-only release reviewer owns combined source review.
- All six affected Coolify applications were reread: repository
  `Cursedpotential/propria`, branch `main`, auto-deploy disabled. No deployment
  or environment change has been triggered by this lane.

Still required: final combined source proof, green published CI, independent
review, current-main reconciliation, neutral configuration validation, manual
Coolify deployments, served default/Dev/Live identity and denial readbacks,
and actual user-facing click verification. The last full Case aggregate read
returned 503; this has not yet been refreshed in the resumed run. A healthy
service is not proof that the full Case page works.

Disposable Dev data isolation and a general feature-flag registry remain owed;
neither is implemented by these admission guards. The existing-desktop-browser
exception or owner-operated click check remains an unanswered owner choice.

## Historical checkpoint — 2026-10-05 15:00 UTC

Parent tip `6bce74fc` preserves upstream `460826ad`, including Claude's working
library, original-file aliases and sync reconciliation. Shared checkout changes
were neither staged nor overwritten. PR: https://github.com/Cursedpotential/propria/pull/2.

- Engine `6be314ef` integrated as `5f6485d7`: direct extraction corrections and
  event marking require durable Live admission before any store call; context
  review, foreshadowing and metadata overlays require the exact initial receipt
  before beginning a transaction and recheck scope within it. Binding and repair
  anchor readers reject court/receipt conflicts. Pre-import source context has
  its own explicit Live spec and exact-pair guard before database access; a
  nonexistent preview is not required and no later authority is inferred from
  its ID or request digest.
- Python `497378ba` / `69517c91` integrated as `7b2bd4cd` / `03031f6b`: fresh
  authenticated authoritative header verification before chunk/promotion writes;
  neutral configured pair must match the approved header. Exact legacy stored
  flag-note replay is accepted only for the same key and full metadata with the
  sole old/new operating-label difference. No duplicate insert or historical
  rewrite is authorized.
- Independently rerun combined Python suite: **331 passed, 5 existing skips**.
- Source-pinned existing-VPS packet `engine-pass3` from `5c7dbc8a` passed
  caseidentity, repairplan, postgres, runtimeapi, proffer, profferworker,
  extraction commitcheck/entities/events/flow/librarysync/model/service and
  Temporal/starter packages. The engine tree is unchanged between that packet
  and `6bce74fc`. Libraryvalidation initially rejected a symlink interpreter;
  rerun with a regular-file interpreter and the same pinned lxml/pypdf libraries
  **passed**. Whole-suite combined acceptance, including Activities, is pending.
- GitHub combined Workbench API run `37325292197` reported **620 passed,
  22 failed**. This supersedes the isolated older 627-pass snapshot. Twenty-one
  failures expose stale R2 acquisition expectations plus an active legacy upload
  path; one is the pre-existing 300-line module-cap failure. They are not waived.
- Separate owners are reconciling fresh UI uploads onto the existing canonical
  Proffer stream, denying new retired-storage staging before I/O, preserving old
  records, and updating positive B2/Live fixtures without weakening guards;
  refactoring imported-read modules without changing behavior or the cap; and
  making retained synthetic Go fixture paths portable on the VPS.

## Current remaining acceptance gates

1. Integrate the bounded Workbench upload/CI and portable-fixture commits.
2. Run combined source-pinned suites; require Workbench CI green and independent
   final review of direct guards, authoritative verification and rollout replay.
3. Provision the verified neutral pair for Workbench, exec API and Python worker;
   reuse the existing matching service credential. No credential rotation or
   auth-policy change is needed. Automatic deployment stays disabled.
4. Merge accepted source, manually deploy affected services through Coolify, and
   verify served default/Dev/Live IDs, unknown-mode rejection and operation
   receipt readback. No runtime deployment has been triggered by this lane.
5. Actual user-facing clicks remain a separate gate. The existing server browser
   is denied by Workbench's user boundary. Owner choice is pending: approve a
   read-only existing-desktop-browser check or perform that check personally.
6. Quarantine only this lane's owned source exports; preserve other agents'
   worktrees and the concurrently dirty main checkout. No permanent deletion.

The disposable Dev-data workspace and general feature-flag registry remain owed
features. This bounded repair must not be presented as implementing either.

## Authority and invariant

Owner explicitly instructed fixing mode-dependent case identities. Use one
approved case for Dev and Live, Live by default, separate feature flags. Do not
restore the superseded D-126 placeholder/reset policy. The canonical decision
is `docs/decisions/2026-10-05-single-case-operating-flags.md`.

## Earlier lane/progress snapshot — retained for provenance, not current status

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

## Earlier checklist — use the current acceptance gates above

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

## Review invariants and earlier verification checkpoints

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
