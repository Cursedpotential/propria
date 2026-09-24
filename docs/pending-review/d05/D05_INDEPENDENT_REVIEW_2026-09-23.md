<!-- Byline: Codex D05 independent reviewer · GPT-6 · 2026-09-23; proof: local tests plus GitHub CI readback -->

# D05 independent review of PR #30

**Decision: HOLD.** Reviewed `Cursedpotential/probata` PR #30 from base `382acf6a8f1f919b6f3abc02de40f946629d8461` through candidate `f844050` on `codex/d05-exact-review-targets-20260923`. This is a bounded repair to W08/PB05 record and chunk annotation lookup. It does not close all W08 acceptance, the frozen proposal reader, entity correction, selected destinations, or live release proof.

## Independent finding and correction

At `9dc5496`, PostgreSQL `ContentTarget` checked a chunk against the newest sealed chunk generation for a **source version** but returned the preview snapshot's **normalized generation** as its attempt ID. Another attempt's newer chunk generation for that source could therefore be accepted under the older preview attempt. The same source-only selector was used to display Review chunks. Commit `f844050` binds both selectors to the snapshot's source version **and** normalized generation and retains the sealed/latest ordinal constraint. The existing chunk primary key and `content_chunk_generation_source_idx`/`content_chunk_generation_normalized_idx` support bounded lookup; the query uses bound UUID parameters and does not accept caller SQL. A regression test asserts all three coordinates reach the query. Live PostgreSQL planning and readback were not performed.

The new Go endpoint previously sent unexpected store error text to callers through `storeError`; commit `f844050` now returns a generic 503 for such errors while retaining the existing not-found/not-ready mapping. A test checks that an internal error string is absent from the response.

## Boundary checks

- The Go route requires the existing Tailnet/service-token check. The BFF requires TEST/REAL mode binding before lookup and the flag route requires an authenticated subject before submitting to the flag spine. D01 action authorization remains an external dependency; this review does not claim an independent action grant at the Go endpoint.
- Record lookup binds the record ID to the snapshot normalized generation. Chunk lookup now binds source version, normalized generation and sealed chunk generation. Both resolve independently of the 250-item presentation page. Unknown IDs and stale attempt IDs are rejected before the flag call. Entity scope is rejected because no governed entity reader exists.
- Target lookup and flag creation cross services, so there is no atomic compare-and-write across a concurrent snapshot change. The persisted flag retains its explicit attempt and handle metadata. This is a remaining concurrency/replay acceptance limit, not a demonstrated atomic current-at-write guarantee.
- The flag is reversible review metadata in the existing flag spine; this patch performs no evidence admission or promotion.

## Verification

| Gate | Result |
| --- | --- |
| `go test ./runtimeapi/... ./postgres/...` | Passed after `f844050` |
| `go vet ./runtimeapi/... ./postgres/...` | Passed |
| Focused Workbench flag and mode pytest suites, using the existing Probata virtual environment | 52 passed, 1 dependency deprecation warning |
| Ruff check on the three changed Workbench Python files | Passed |
| `git diff --check` | Passed |
| Official Gitleaks 8.30.0, redacted `origin/main..f844050` Git scan | Three candidate commits, zero findings; [empty report](D05_GITLEAKS_2026-09-23.json). Release ZIP SHA-256 `54fe94f644b832dd08e8c3a5915efb3bfa862386d59fb27ca0792cb687a83573` matches the prior D05 receipt. |
| Go race test | Not run: this Windows Go environment requires cgo for `-race`; no C compiler was found. |
| Live PostgreSQL/BFF/Go/sink/browser/deployment | Not run; no live acceptance claimed. |

GitHub PR checks at review time: Go engine and CodeQL passed. Both Validate runs failed at `uv run ruff format --config pyproject.toml --check server tests` on exactly `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`; neither file is in PR #30's diff. The mandatory live integration suite was skipped by the workflow. These two formatter files belong to another owner and were not edited here. PR #30 must remain open until required checks are genuinely green. No merge or deployment occurred in this review.

## Remediation re-review — atomic current-attempt flag admission

> _Byline: Codex · GPT-5 · 2026-09-24; implementation evidence is local only._

The independent review's read-then-write race is addressed in implementation commit
`0a8b742699f2807521613d015f1c0e697fcad490`, on the same PR branch. The exact lineage from
the reviewed base is:

`382acf6a8f1f919b6f3abc02de40f946629d8461` →
`a195c416d1aaf6be6886ad44539d5e6eba6be7ca` →
`9dc5496bbecb3dcb4fca4336e54bc5b3fac5231e` →
`f844050f9eab5a80fec85189ff2ab703f8acaf91` →
`fdd390d64122fdd411ad37aff370f0b2686fbde3` →
`0a8b742699f2807521613d015f1c0e697fcad490`.

The BFF retains its mode/handle check, but no longer does a separate current-target read
before the write. It submits the expected handle, attempt, target UUID/scope, mode, actor,
and reason to a dedicated existing-flag-spine endpoint. That endpoint runs in the shared
PostgreSQL transaction, acquires `pg_advisory_xact_lock(hashtextextended(preview_handle, 0))`,
reads the latest snapshot, verifies both the expected normalized attempt and exact current
record/chunk target, then inserts the flag before releasing the transaction lock. Go
snapshot publication and decision writes already acquire the same per-handle advisory lock.
Thus a snapshot advanced before admission causes a 409 and no insert; a competing advance
after admission waits until the flag transaction completes.

The flag remains reversible metadata in `analysis.corroboration_flag`, not an evidence
mutation. The table/API target-kind contract permits only `record`, `knowledge`, or `run`;
the preview handle is therefore stored as a `run` target, while notes preserve the exact
preview handle, TEST/REAL mode, record/chunk scope, target UUID, attempt ID, and actor.
Listing is additionally filtered by the preview handle. The response normalizer now accepts
the flag spine's actual `flag_id` response field, and the target-scoped listing query is
covered. Entity annotation remains explicitly unavailable.

Regression coverage includes a target attempt advanced before atomic admission (rejected
without insertion), an absent current target (rejected without insertion), and a successful
current target whose flag and provenance metadata are persisted. The API test double verifies
that the advisory lock precedes the latest-snapshot read and that the insert is not reached on
either denial path. This is not a live PostgreSQL concurrency test.

### Re-review verification

| Gate | Result |
| --- | --- |
| Workbench potential-promotion flag + mode-isolation suites | 53 passed, 1 existing dependency deprecation warning |
| Shared flag API / CRUD / atomic-admission tests | 12 passed, 1 existing dependency deprecation warning |
| `go test ./runtimeapi/... ./postgres/...` | Passed (cached) |
| Ruff lint and format check on seven changed Python files | Passed |
| Targeted Workbench mypy with imports skipped | No issues in five changed source/test files, including the explicit typed service-call tuple union |
| `git diff --check` | Passed |
| Official Gitleaks 8.30.0, scoped to seven changed Python files | Zero findings |
| Full `tests/test_inspect_routes.py` | 3 baseline failures from assertions for absent retired `sql/0007_curation_and_flags.sql`; all 12 flag-focused tests passed |
| Import-following mypy | Existing unrelated Workbench import/type errors remain, including the unchanged nullable parameter at `app/runtime/proffer.py:212`; existing annotation/indexing errors also remain elsewhere in `tests/test_inspect_routes.py`. The changed Workbench source/test set passes the import-skipping targeted check. |
| Live PostgreSQL, deployed BFF/Go, sink readback, browser, release | Not run; not claimed |

The atomicity finding is closed at the implementation/test boundary, not independently
verified against a live PostgreSQL service. Keep PR #30 open until the post-push CI state is
fresh and required checks pass. The original independent review and receipt entries above are
retained as historical evidence; the D05 receipt carries this repair's exact commit lineage
and expanded source/test scope.

## 2026-09-24 provenance remediation, awaiting independent re-review

> _Byline: Codex PR30 implementation agent · GPT-6 · 2026-09-24. This section is an implementation receipt, not an independent approval._

At code commit `380018a0addd4824bdf1ab2c414f9561683a117b`, the dedicated flag
endpoint requires a short-lived HMAC delegation from the Workbench BFF over the exact
actor, mode, handle, attempt, scope, target and claim. The BFF obtains the actor from
its authenticated request state. The Platform API checks the signature before touching
PostgreSQL and compares the request's mode with the current snapshot source version's
durable `matter_id` under the per-handle transaction lock. The shared key is a new
service-to-service secret at `/run/secrets/proffer-flag-delegation-key`; it must be
mounted in both services before this endpoint can be released. Platform API
`PROFFER_TEST_MATTER_ID` and `PROFFER_REAL_MATTER_ID` must match Workbench's configured
identities. Missing configuration fails closed.

Generic flag creation and note edits now reject the reserved Proffer contract; note
edits to existing governed rows lock the row and return 409. Ordinary flag status
and artifact updates remain available. Proffer listing reads one ordered server-side
snapshot, returns up to 2000 rows, and returns 409 if the bound is exceeded. It no
longer silently inherits the general flag list's 50-row default. The special INSERT
uses `ARRAY[]::text[]` for the PostgreSQL `evidence_wanted` column.

Focused shared API tests: 15 passed and the opt-in disposable PostgreSQL text-array
test skipped because `PROBATA_TEST_POSTGRES_DSN` was not configured. Workbench flag
and mode tests: 55 passed. Changed-file Ruff lint/format, targeted Workbench mypy,
Go focused tests and vet, and `git diff --check` passed. Verified Gitleaks 8.30.0
scanned all seven commits from `origin/main..380018a` with no findings. The
existing `settings.py` mypy network-type errors remain outside this change.

**Decision remains HOLD.** The new key and matter settings are not mounted on the
deployed services; there is no disposable PostgreSQL execution receipt, live BFF/Go
exchange, browser proof, or fresh independent review of this commit. The current
schema uses free-text notes for provenance; the API reservation prevents ordinary
route forgery and mutation, while direct privileged SQL writes remain outside the
route boundary. Shared Validate CI is red on unrelated formatter baseline files.
