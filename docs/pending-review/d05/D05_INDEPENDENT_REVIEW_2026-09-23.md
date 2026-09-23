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
