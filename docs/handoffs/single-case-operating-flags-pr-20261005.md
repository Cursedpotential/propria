# One case identity, independent Dev/Live policy

Byline: Codex, orchestrator, 2026-10-05.

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
- Workbench API: 627 passed, one documented baseline file-size check failure
  (imported_pg.py 406 lines; imported.py 830 versus upstream 828).
- Existing-VPS web gates: lint zero errors/26 warnings, typecheck/build passed,
  browser-free smoke 140 passed/four browser journeys skipped, Storybook passed.
- Python chunk/promotion combined checkpoint: 220 passed/five skipped. Additional
  authoritative approval verification and exact receipt-scope tests are in review.
- Engine: source integrated; combined VPS suite and final direct-service guard
  review remain pending. Synthetic SDK replay is bounded first-command proof,
  not a complete production-history audit.

## Remaining acceptance gates

- Finish authoritative header verification and final direct-service guards.
- Independent combined review and final source-pinned test receipt.
- Provision verified neutral case configuration; reuse the existing private
  service credential without rotation or user-auth changes.
- Merge only after acceptance, then deploy affected services manually through
  Coolify. Automatic deployment remains disabled.
- Verify served default/Dev/Live IDs and actual user-facing controls. The server
  browser is denied by Workbench's existing tailnet user boundary; the owner has
  been asked to approve a read-only existing-browser exception or self-check.

This is a draft integration checkpoint, not a completed deployment claim.
