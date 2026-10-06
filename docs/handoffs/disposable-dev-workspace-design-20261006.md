# Disposable Dev workspace and independent flags: proposed next slice

Byline: Codex Architect agent, GPT-6.1-Sol, 2026-10-06; persisted by Codex orchestrator.
Read-only design against merged source `21b746fb`. Not approved or implemented.

## Recommendation

Use a server-owned, materialized frozen projection of the same approved case
plus a workspace-scoped overlay. Begin with edits to existing case-header and
person fields, keeping canonical Dev writes blocked. Exit revokes the workspace
and refreshes current Live; it never restores a snapshot over Live, switches
case identity, changes the global database URL, or discards concurrent Live work.

The workspace is a data-isolation coordinate, not a new case or authentication
substitute. Existing authenticated ownership remains required through the
current trusted door. Same matter/court/person/source/version/hash coordinates
are preserved. No original is rewritten or substituted.

## Source-grounded constraints

Paths below are relative to `modules/Probata/probata/` in the merged repository.

- `modules/engine/caseidentity/identity.go:20-40` retains canonical Live admission;
  workspace writes need a distinct narrow adapter, not an exception to this rule.
- `modules/engine/caseidentity/types.go:68-73` and Workbench
  `api/app/runtime/operating_mode.py:19-28` default only fresh omitted requests
  to Live. Missing durable context stays unknown.
- `api/app/service/case_identity.py:105-117` and `proffer_start.py:24-32` reject
  Dev before canonical dispatch. Do not broadly remove those guards.
- `modules/engine/postgres/case_identity_store.go:211-257` reads the registry
  projection in one read-only RepeatableRead transaction with an eight-second
  statement limit. Reuse this bounded seam to materialize a baseline; do not
  retain a browser-lifetime transaction.
- BFF `api/app/service/case_identity.py:84-89` appends catalog data from another
  store. The full response is not one cross-store snapshot. Explicitly capture
  per-store versions/timestamps or exclude that store from frozen coverage.
- Initial Proffer receipt scope/mode lives in
  `postgres/proffer_preview_store.go:383-425,508-566`; write admission is in
  `postgres/preview_write_admission.go:14-28`. Future workspace context must be
  durable there when those actions become supported, never inferred from IDs.
- `api/app/service/matter_mode.py:26-27,82-99` and
  `preview_mode_recovery.py:21-67` are recovery caches, not workspace authority.
- `web/src/lib/fixed-case-context.tsx:10-16,38-53` and
  `components/intake/matter-mode-selector.tsx:14-31` currently hold mode rather
  than a server lease. Entry/exit needs an asynchronous lifecycle.
- `web/src/components/case/case-identity-screen.tsx:441-449` caches by mode;
  add workspace ID/generation so two Dev sessions cannot share cached state.
- Starter `temporal/cmd/starter/main.go:74-102` and worker
  `profferworker/worker.go:331,517,577-655` share pools across repositories.
  Ambient DSN switching would redirect concurrent Live work.
- `profferworker/operating_guard.go:29-105` fences delayed/retrying Activities;
  future isolated Activities must independently admit/revalidate their workspace
  before body execution and publication.
- `proffer/context_chunks.go:45-46,204` calls the separate Python publication
  queue; `server/temporal/chunk_activities.py:63,77,91,156,209` writes fixed
  projection targets. A PostgreSQL clone alone cannot isolate the pipeline.
- File-backed marks in `api/app/repo/source_unit_marks.py:26-27,46-52` and
  `runtime/source_unit_marks.py:39-42` have no workspace coordinate. Deny them
  or add a deliberate isolated adapter before claiming general coverage.
- Imported caches in `api/app/service/imported.py:94-112,169-170,223-226`
  use global keys; included views need workspace-aware keys/invalidation.
- `runtimeapi/source_ref.go:34-39,75-80` and `temporal/httpapi.go:46,82,249`
  retain existing auth/source-admission coupling. Inventory and separately
  review it; a flag registry must not silently widen or repurpose auth.

Historical diagnostic constants/comments are not evidence of active identity
selection. The current probe selects the authoritative pair. Do not revive
historical placeholder/reset instructions while adding isolation.

## Alternatives and why the first slice is bounded

1. **Frozen projection plus overlay:** best fit for a small reviewable first
   slice. No copied authority or alternate case tables. Every supported action
   needs an explicit overlay adapter; this does not cover arbitrary ingestion.
2. **Isolated cloned database:** same IDs can be preserved in a physical copy,
   but credentials/grants, sensitive data copies, queue/object/search/catalog
   isolation, running-work metadata and cleanup all need separate treatment.
   The current owner decision rejects parallel deployment/case-table design;
   a clone is conditional on an explicit new owner choice, not approved now.
3. **Workspace/version coordinates across every store:** broad potential
   coverage, but extensive key/query/repository/projection changes and high leak
   risk from any unscoped query. Not a small migration or the first slice.

A long open transaction followed by rollback is not a disposable-workspace
solution: external publications are not rolled back, and exported PostgreSQL
snapshots depend on the exporting transaction remaining open. Materialize the
bounded baseline, then close its transaction.

## Proposed durable lifecycle

Keep case scope, operating mode, workspace context, capability flags, and
authentication independent. Proposed workspace metadata:

- Opaque workspace ID, exact approved matter/court IDs and authenticated owner.
- Materialized baseline reference/hash/schema version/capture time, per-store
  coverage manifest, original source/version/hash references.
- Generation/fence token, status (`creating`, `active`, `closing`, `discarded`,
  `failed`), expiry/heartbeat, supported capabilities/configuration revision.
- Activation/closure receipts, closure reason, explicit idempotency scope.

`Live -> create baseline -> activate lease -> Dev overlay -> revoke lease -> current Live refresh`

Entry is idempotent and activates only a complete validated baseline. Editing
requires explicit Dev, an active matching owner/generation, an implemented
capability and existing typed validators. Append before/after, actor, reason,
target ID, idempotency key and expected overlay revision only in workspace state;
do not call canonical registry write methods. Workspace ID/generation must be
part of idempotency; existing canonical Actor.StoredKey is insufficient alone.

Reads use the frozen baseline plus that workspace's changes, not cache-miss
refills from Live presented as frozen. Missing/incomplete baselines fail closed.
Exit atomically revokes admission before acknowledging discard, then clears
Dev caches/dialogs/streams and fetches current Live. There is no implicit
publish, merge, writeback or Live restore step.

Close and edit serialize status admission plus overlay append in one transaction:
an edit commits before closure and is subsequently hidden, or is denied after
closure. Old-generation retries cannot resurrect state. Crash recovery admits
only a complete active durable receipt; discarded/expired/incomplete leases
cannot resume. A fresh Dev entry gets a fresh baseline from then-current Live.

Discard means absent from active views/authority, not permanent deletion. Keep
originals and audit/closure receipts. Quarantine eligible workspace-only files
under the owning runtime's approved policy; physical destruction is the owner's
action. Derived Dev work must retain workspace/parent provenance and cannot
become accepted evidence merely because a capability flag is enabled.

## Independent flag registry

Start with typed code-owned definitions and existing per-service configuration
adapters. A shared machine-consumed schema belongs in `modules/contracts/`
only with actual consumers. Dynamic PostgreSQL values or an external flag
service add authority/availability/privacy work; neither is implemented here.

Definitions need key, type, safe default, owner, allowed scope, side-effect class,
implementation/readiness state, validator and tested on/off paths. Mode is not
an all-dev-features switch. Workspace availability and each implemented overlay
action have separate gates. Promotion/content-review flags are not feature flags;
Temporal version markers are replay mechanisms, not runtime feature toggles.
Record resolved behavior-affecting configuration in new receipts. Do not read
mutable ambient flags inside replaying workflows; Activities revalidate current
revocation/safety. Unknown keys/invalid values reject; absent values use explicit
safe defaults. Authentication/evidence/retention settings remain independent.

## Ranked implementation slices

1. Contracts, registry definitions, durable lifecycle and frozen Case reads:
   proposed engine `devworkspace/`, PostgreSQL adapter/HTTP handler, BFF workspace
   types/service/routes and browser workspace client/context. Compose in
   `temporal/cmd/starter/case_identity_composition.go`; reuse the bounded registry
   read without changing Live. Distinguish explicit workspace routes from
   canonical routes in BFF operating policy.
2. Actual disposable header edits and edits to existing people, separate overlay
   adapters and `web/src/components/case/identity-dialogs.tsx`; preserve all IDs.
   Defer creating/merging people, identifier changes and arbitrary review actions
   until their semantics are implemented.
3. Additional review overlays via workspace-specific repositories; canonical
   `postgres/context_review_store.go:266-287` and
   `source_metadata_store.go:388-419` keep their Live fences.
4. Explicit pipeline/queue isolation: request/receipt/child-workflow context,
   worker resource selection, workspace object/bundle roots, Python publication
   adapters, separate projection targets, catalog/file-backed marks/tool effects.

**New durable storage is a prerequisite, not authorized by the core repair.**
Do not overload Proffer initial events, auth config, ops audit rows or ingest JSON
as an accidental workspace database. Dedicated lifecycle/overlay storage needs
a separately reviewed schema/grant/forward-only application plan. Do not apply
bootstrap rebuild rules to the active shared Live database or invent numbered
migrations contrary to current repository instructions.

## Acceptance before any disposable edit is enabled

- Exact case/court/parent/person/source identity equality across modes/default.
- Fresh omitted mode Live; old durable missing context unknown.
- Two Dev sessions have separate workspaces, caches and idempotency.
- Dev edits change only workspace state; canonical write methods never run.
- Concurrent post-capture Live changes remain absent from frozen Dev, appear
  on exit, and survive discard.
- Close/edit races, duplicates, lost responses, retries, expiry, corrupt manifests
  and restart recovery cannot leak or resurrect authority.
- Stale dialogs/tabs/generations cannot write after closure.
- All independent flag combinations leave identity/auth/evidence/retention intact.
- Unsupported actions reject before I/O, persistence or dispatch, including
  upload/promotion/publication and file-backed marks.
- Actual served entry/edit/exit behavior is checked; builds/tests are not clicks.

Pipeline isolation additionally needs old-history replay without altered commands,
unknown scheduled/retrying Activity denial, workspace context through child/Python
queues, and independent destination-isolation proof for every side-effect store.
No mass cancellation, purge or identity-derived admission repair.

## Material owner choices before implementation

- Bounded disposable Case page first, or full ingestion/analysis isolation now?
- Freeze registry plus separately timestamped catalog, or explicitly exclude
  catalog/search from first-slice coverage? No cross-store snapshot exists today.
- Per-browser/session workspace versus a shared owner workspace (per session
  recommended so one tab's exit does not discard another tab's work).
- Exit/expiry policy, flag-disable revocation semantics and recoverable retention.
- Approval of the dedicated persistence seam and safe forward-only schema/grant
  plan; the prior core identity repair approval does not authorize it.

No new choice is needed to preserve originals/concurrent Live writes, default
Live, retain unknown durable mode, keep canonical Dev denial or leave auth intact.
