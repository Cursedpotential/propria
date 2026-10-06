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
- Fresh existing-VPS Workbench packet pinned to `d902ef2d`: **676 passed, zero
  failures**; focused acceptance 253 passed. This closes the earlier 22 failures
  in source-pinned proof, not by waiving them. Published CI rerun is pending.
- Workbench web lint zero errors/26 existing warnings; typecheck/build passed.
  Fresh browser-free smoke/Storybook final receipt is pending. Earlier browser
  journeys were skipped and do not establish actual user-facing clicks.
- Python chunk/promotion/scope agent checkpoint: **352 passed, five existing
  skips**, lint/format/helper typecheck pass. Parent rerun against current
  upstream is pending. Exact old-label flag replay preserves original rows.
- Actual registered source court is checked before promotion receipt/replay/write.
- Bounded authenticated `/case-identity/scope` now returns the approved registry
  pair and actual court parent without people/count/history queries. It enforces
  a two-second total deadline before pool acquisition; both consumers correlate
  the court's parent. Focused engine proof is running.
- Earlier engine packet passed identity, repair, repositories, runtime API,
  Proffer, worker wiring, Activities, extraction and Temporal/starter packages.
  The new combined source includes upstream `cf83d7a7` and needs a fresh full run.
  Synthetic SDK replay is bounded first-command proof, not a production-history audit.

## Remaining acceptance gates

- Green published Workbench CI and final fresh web receipts.
- Independent combined review and final source-pinned test receipt.
- Provision verified neutral case configuration; reuse the existing private
  service credential without rotation or user-auth changes.
- Merge only after acceptance, then deploy affected services manually through
  Coolify. Automatic deployment remains disabled.
- Verify served default/Dev/Live IDs and actual user-facing controls. The server
  browser is denied by Workbench's existing tailnet user boundary; the owner has
  been asked to approve a read-only existing-browser exception or self-check.

This is a draft integration checkpoint, not a completed deployment claim.
