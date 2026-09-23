# D06 mention atom remediation receipt

Byline: Codex GPT-6, 2026-09-23. PR: [Probata #32](https://github.com/Cursedpotential/probata/pull/32).

## Scope and result

This bounded `SOURCE_TESTED` amendment changes only the D06 mention atom contract and its adversarial tests. Stable mention identity uses source occurrence, source version, exact turn spans, and the within-source mention key. `record_class` is a candidate observation and a disagreement now creates one review conflict with all observations retained. Span order and reconciliation output are canonical under input reversal. Blank turn, run, and window IDs fail validation.

Provenance origin and AI origin remain explicit. Redelivery still increases delivery count while one run/window counts as one observation. Distinct source occurrences that share a provenance origin remain separate atoms; this contract emits no independent-corroboration or truth count.

## Verification

- `uv run --extra dev python -m pytest -q tests/test_work_product_atoms.py tests/test_content_chunk_context_thread_contracts.py`: 43 passed.
- `uv run --extra dev ruff check server/work_product tests/test_work_product_atoms.py`: passed.
- `uv run --extra dev ruff format --check server/work_product tests/test_work_product_atoms.py`: passed.
- `git diff --check`: passed.
- Official Gitleaks 8.30.0 Windows x64 archive matched its published SHA-256, `54fe94f644b832dd08e8c3a5915efb3bfa862386d59fb27ca0792cb687a83573`. Redacted file scans of `server/work_product/__init__.py`, `server/work_product/atoms.py`, and `tests/test_work_product_atoms.py` returned no leaks.

## Review and release boundary

The PR's pre-amendment GitHub `validate (3.12)` jobs failed at repo-wide Ruff format on `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`. Those files are outside D06 ownership. Keep PR #32 draft/HOLD until that gate is resolved by its file owner and independent review accepts the bounded diff.

No source bytes were reopened, model extraction was run, live system was tested, or merge/deploy was performed. D02 source-version/provenance resolution and span readback, governed run receipts, composite and interpretation identity, and full D06 acceptance remain open.
