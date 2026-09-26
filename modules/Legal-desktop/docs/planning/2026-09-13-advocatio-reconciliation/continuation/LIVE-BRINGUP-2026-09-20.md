# Live workdesk bring-up and convergence receipt

Date: 2026-09-20 (work continued across UTC midnight).

## Scope and owner direction

Keep one current Advocatio workdesk working while following the shared stack and
design refactor. The Family Law Toolkit remains the quick mobile companion for
documents, uploads, cheat sheets, legal sources and custom case information.
Its complete data and capability integration remains required. See
SINGLE-WORKDESK-CONVERGENCE.md for the clarified integration acceptance criteria.

## Verified baseline and first deployment

- Backend baseline: 150 passed, 1 xfailed; see E-baseline.md for the local stale
  SQLite exception. Production web build and TypeScript check passed.
- Assistant form repaired in a233171: /api/chat replaces absent /api/assistant;
  failed/non-JSON responses are displayed. Isolated browser success/error tests
  passed with no model provider calls. Shared design contract checks passed.
- Existing Coolify legal-workspace deployed a233171 successfully. The signed
  same-origin API bridge is present; /api/legal/health returns JSON 200.
- Matter read then returned 500. Runtime logs identified a missing packaged SQL
  resource: /usr/local/lib/python3.12/sql/0001_legal_os_sqlite.sql. Checkout tests
  had not exercised installed-package resource loading.
- Live SQLite backup before deployment passed integrity_check. No case records
  were edited in the browser or smoke tests. Credentials remain server-side.

## Remaining convergence work

F0-route-parity.md records the source-search/versioned-inspector migration slice.
The common-client implementation, Storybook state fixtures and public Authentik
SSR user-identity parity are still outstanding. The 383-capability inventory is
complete as a map; it does not mean all toolkit features/data are integrated.
Full shared-record mobile-to-workdesk parity must be demonstrated separately.

## Packaging repair and live verification

Packaging fix 4bc8d6d ships the schema as package data and reads it through importlib.resources. The wheel test checks canonical SQL byte equality and initializes settings from an isolated extracted wheel. Settings and packaging checks passed (3 tests); the retained-artifact follow-up 6d2440d passed its packaging test again. Coolify deployment mw98erzbk6d4k1kbno2kcedd targets 6d2440d. Final live checks are pending.

Final verification: Coolify finished deployment of 6d2440d at
2026-09-21T00:09:45Z. Browser checks of /, /drafts, /templates, /laws,
/calendar and /assistant returned HTTP 200 without application/API-unavailable
error text. Same-origin /api/legal/health, /v1/matter, /v1/drafts and
/v1/templates all returned JSON 200. No uncaught browser errors. The browser
explicitly waited for the hydrated status indicator 'Available: workspace backend'.
Live checks were read-only with zero provider calls and zero case mutations.
These checks establish the current tailnet workdesk bring-up, not complete
upstream evidence connectivity, public login or full toolkit integration.
