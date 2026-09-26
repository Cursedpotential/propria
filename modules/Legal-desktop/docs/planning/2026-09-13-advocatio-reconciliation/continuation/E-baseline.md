# Advocatio current baseline — 2026-09-20

## Scope

Bounded local baseline for the canonical single-matter Advocatio workdesk at
`E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`. This records the
installed-environment backend and web checks requested for the shared stack.
It does not claim live data, deployment, remote-provider, or production proof.
This receipt does not modify source or dependency files; the coordinator's
`web/src/app/assistant/page.tsx` edit was present for the refreshed web build.

The repository instructions were read from `AGENTS.md` and `web/AGENTS.md`.
The observed local runtime is Python/FastAPI with `uv`, and Next.js App Router
with React 19. The shared target is governed by the current `STACK.md`; this
receipt does not restate older locked-stack prose. The web package declares
Next `^16.3.1`; the installed lock/runtime resolved Next `16.3.2`.

## Repository receipt

- Repository root: `E:/AI_Workspace/Projects/Propria/modules/Legal-desktop`
- Commit at baseline: `0149c0ca22ec66c360f1b7737fe69ad3f66cdd43`
- Dirty paths observed before/after checks:
  - `web/src/app/assistant/page.tsx` (coordinator source edit, present for the refreshed build)
  - `docs/planning/2026-09-13-advocatio-reconciliation/continuation/F0-route-parity.md` (untracked, pre-existing coordinator lane)
  - `docs/planning/2026-09-13-advocatio-reconciliation/continuation/MOCKUP-IDENTITY-2026-09-20.md` (untracked, pre-existing mockup lane)
- No other tracked modifications were introduced by this lane. Build output
  under `web/.next/` is ignored and did not add a dirty path.

## Backend

Installed environment was used; no install, upgrade, network operation, live
service, or deployment was performed.

Collection command:

```text
uv run --no-sync python -m pytest --collect-only -q
```

Result: **151 tests collected**. The direct script form
`uv run --no-sync pytest --collect-only -q` failed before collection with:
`uv trampoline failed to canonicalize script path`. Using the module form
avoids that `uv` launcher issue while using the same installed environment.

Full isolated/local suite:

```text
uv run --no-sync python -m pytest -q
```

Result: **150 passed, 1 xfailed, 2 warnings in 62.36s (0:01:02)**.

The xfail is `tests/test_calendar.py::test_calendar_delete_unknown_id_returns_404`.
Its explicit reason records the known dev SQLite schema drift: the local
`data/workspace/legal.sqlite` is missing ORM column
`legal_core_matter_ref.last_agno_verify`, so the well-formed unknown-event route
currently reaches an `OperationalError`; the test is intentionally retained as
an xfail pending the separately tracked database rebuild.

Read-only schema inspection confirms the mismatch. The SQLAlchemy model at
`api/legal_workspace/db/models.py` declares `last_agno_verify` as nullable JSON;
the local SQLite table has `matter_id`, `display_name`, `is_friendly_primary`,
`source_revision`, `synced_at`, `issue_tree`, and `factor_matrix`, but no
`last_agno_verify`. The checked-in PostgreSQL bootstrap SQL also predates this
column and does not provide a migration. This is directly relevant to local
matter reads through the ORM (including the calendar unknown-event lookup),
but does not prove the deployed PostgreSQL schema is missing it; no database was
changed during this inspection.

Warnings were:

- Starlette deprecation warning that `httpx` with `starlette.testclient` is
  deprecated and `httpx2` is expected in a future compatibility path.
- Python 3.16 deprecation warning from `pytest_asyncio`'s event-loop-policy
  access.

These checks are synthetic/local tests. They do not establish the deployed
service behavior or prove a live Indicia Probata connection.

## Web

Installed `web/node_modules` was present; no package install or upgrade was
performed.

TypeScript check:

```text
web/node_modules/.bin/tsc.cmd --noEmit
```

Result: **exit 0**.

Production build:

```text
npm run build
```

Result: **exit 0**. Next reported:

```text
Next.js 16.3.2 (Turbopack)
Compiled successfully in 14.3s
Finished TypeScript in 2.6s
Generating static pages ... 34/34
```

Refreshed after the coordinator's `assistant/page.tsx` edit on the same
installed dependencies: compile **4.1s**, TypeScript **2.5s**, static pages
**34/34 in 556ms**, overall process exit **0**. The earlier baseline build had
compiled in 14.3s; both runs passed.

The generated route inventory included the expected shared-stack surfaces,
including `/api/chat`, `/api/legal/[...path]`, the single-matter pages, and the
App Router middleware proxy. A preliminary `npm run build -- --no-lint` attempt
was rejected by this installed Next version (`unknown option '--no-lint'`); it
was not treated as an application failure, and the declared script was rerun
without that obsolete flag and passed.

## Current baseline conclusion

The current local single Advocatio workdesk passes the exercised checks on its
observed Python/FastAPI + Next App Router/React 19 runtime. The shared target is
governed by the current `STACK.md`; this receipt does not claim to supersede it.
Backend behavior is green apart from the explicitly documented stale-SQLite xfail;
web typecheck and production build are green.
Deployment and live-route parity remain outside this synthetic/local receipt.
