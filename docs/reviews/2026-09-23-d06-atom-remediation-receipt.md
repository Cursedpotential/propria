# D06 mention atom remediation receipt

Byline: Codex GPT-6, 2026-09-23; Unicode coordinate-contract remediation 2026-09-24.
PR: [Probata #32](https://github.com/Cursedpotential/probata/pull/32).

## Scope and result

This bounded `SOURCE_TESTED` amendment changes only the D06 mention atom contract and its adversarial tests. Stable mention identity uses source occurrence, source version, exact turn spans, and the within-source mention key. `record_class` is a candidate observation and a disagreement now creates one review conflict with all observations retained. Span order and reconciliation output are canonical under input reversal. Blank turn, run, and window IDs fail validation.

Provenance origin and AI origin remain explicit. Redelivery still increases delivery count while one run/window counts as one observation. Distinct source occurrences that share a provenance origin remain separate atoms; this contract emits no independent-corroboration or truth count.

## Exact-head review remediation

The reviewed head `3c1967512407a4b22e87cfc5efc0d831167b6782` left the public
`SourceSpan` coordinate system ambiguous across runtimes. Implementation commit
`6895a444b64d75e1b8155e92b4b3d3c1538ae451` corrects that contract:

- `char_start` and `char_end` are end-exclusive Unicode code-point indices into the exact decoded source-version turn text, matching Python `str` indexing.
- A non-BMP code point counts as one position. Each combining code point counts separately; spans are not grapheme-cluster indices.
- `normalization="none"`: producers and consumers must not apply NFC, NFD, or another Unicode normalization before resolving the span.
- The fixed `coordinate_unit="unicode_code_point"` and `normalization="none"` fields are serialized, schema-constrained, and included in the canonical span representation that determines `atom_id`. Alternate UTF-16 or normalized-text policies fail validation.
- Synthetic regressions cover a non-BMP emoji, a decomposed combining-mark sequence, rejection of alternate policies, serialization, round-trip identity, and one exact deterministic atom ID.

The model still does not receive or reopen source content. Source-version digest resolution,
turn lookup, bounds verification, and exact excerpt readback therefore remain mandatory downstream
work rather than claims made by this pure contract.

## Verification

- `uv run --extra dev python -m pytest -q tests/test_work_product_atoms.py tests/test_content_chunk_context_thread_contracts.py`: 47 passed.
- `uv run --extra dev ruff check server/work_product tests/test_work_product_atoms.py`: passed.
- `uv run --extra dev ruff format --check server/work_product tests/test_work_product_atoms.py`: passed.
- `uv run --extra dev mypy server/work_product/atoms.py`: passed.
- Import/schema assertion: the public JSON schema fixes `coordinate_unit` to `unicode_code_point`; passed.
- `git diff --check`: passed.
- Official Gitleaks 8.30.0 Windows x64 archive matched its published SHA-256, `54fe94f644b832dd08e8c3a5915efb3bfa862386d59fb27ca0792cb687a83573`. Redacted file scans of `server/work_product/__init__.py`, `server/work_product/atoms.py`, and `tests/test_work_product_atoms.py` returned no leaks.
- The same official Gitleaks 8.30.0 binary scanned the complete `origin/main..HEAD` remediation range with redaction and returned zero findings.

## Review and release boundary

The PR's pre-amendment GitHub `validate (3.12)` jobs failed at repo-wide Ruff format on `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`. Those files are outside D06 ownership. Keep PR #32 draft/HOLD until that gate is resolved by its file owner and independent review accepts the bounded diff.

After the remediation push, both new `validate (3.12)` jobs failed at `Format with ruff` (GitHub runs `35920130762` and `35920123305`). The same repo-wide command run locally on the new head reported exactly those two files and `322 files already formatted`. PR #32 was converted to draft. Its other queued/running jobs are not treated as release proof.

No source bytes were reopened, model extraction was run, live system was tested, or merge/deploy was performed. D02 source-version/provenance resolution and span readback, governed run receipts, composite and interpretation identity, and full D06 acceptance remain open.
