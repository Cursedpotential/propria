# One case identity, independent Dev/Live policy

Byline: Codex, orchestrator, 2026-10-06.

## Scope

- Dev and Live use the same approved matter, court, person and source identities.
- Fresh omitted mode defaults Live; durable missing mode stays unknown.
- Canonical values are DEV/LIVE. Legacy input aliases and stored SQL labels are
  narrow rollout boundaries, not alternative case identities.
- Dev canonical mutations are denied before persistence/dispatch until a
  disposable data workspace exists. This change does not implement rollback,
  a general feature-flag registry, or an isolated Dev workspace.
- Authentication, evidence gates and retention remain independent and unchanged.
- Preserve concurrent upstream R2 retirement, ZIP toolkit placement, AI work-product
  placement, private dispatch and original-link work.

## Verification checkpoints

- Legal consumer: 27 passing tests, independently rerun; mocked proof only.
- Fresh existing-VPS Workbench packet pinned to `d41be78f`: **729 passed, zero
  failures**; focused transport/policy 232 passed. This closes the earlier 22
  failures in source-pinned proof, not by waiving them. Independent review
  accepts its dedicated bounded transport. Published final CI rerun is pending.
- Workbench web lint zero errors/26 existing warnings; typecheck/build passed.
  Browser-free smoke **144 passed/four browser-dependent skips**; Storybook
  passed. Skipped browser journeys do not establish actual user-facing clicks.
- Fresh parent Python chunk/promotion/scope proof: **399 passed, five existing
  skips**, lint/format/helper/CLI typecheck pass. Async HTTPX scope read owns a
  total five-second deadline, no proxy/redirect, raw 64 KiB cap and pre-body
  encoded-response denial. Exact old-label flag replay preserves original rows.
- Actual registered source court is checked before promotion receipt/replay/write.
- Bounded authenticated `/case-identity/scope` now returns the approved registry
  pair and actual court parent without people/count/history queries. It enforces
  a two-second total deadline before pool acquisition; both consumers correlate
  the court's parent. Source-pinned focused scope/store/API/starter proof passed.
- Earlier engine packet passed identity, repair, repositories, runtime API,
  Proffer, worker wiring, Activities, extraction and Temporal/starter packages.
  The new combined source includes upstream `cf83d7a7`. Whole-engine run found
  only an upstream stale optional-stage-count test; a narrow test-only correction
  preserves all prior assertions and adds the twelfth stage. Fresh full test,
  build and vet are running. Malformed/padded durable receipt IDs now deny without
  panic or replacing source pointers, accepted by independent review.
  Synthetic SDK replay is bounded first-command proof, not a production-history audit.

## Remaining acceptance gates

- Green published final Workbench CI.
- Final Python raw-body independent review and whole-engine test/build/vet receipt.
- Provision verified neutral case configuration; reuse the existing private
  service credential without rotation or user-auth changes.
- Merge only after acceptance, then deploy affected services manually through
  Coolify. Automatic deployment remains disabled.
- Verify served default/Dev/Live IDs and actual user-facing controls. The server
  browser is denied by Workbench's existing tailnet user boundary; the owner has
  been asked to approve a read-only existing-browser exception or self-check.

This is a draft integration checkpoint, not a completed deployment claim.
