<!-- Updated by: Codex (D02 Sources/Catalog) | Date: 2026-09-23 | Rev: 2 | Platform: Codex / win32 | Changes: record PB06 implementation and local verification | Context: preserve the boundary between a bounded probe fix and full D02 acceptance -->

# D02 PB06 bounded XML probe

## Planned change

FILE: `server/proffer/service.py` and `tests/test_proffer_xml_probe.py`
TYPE: AUGMENT / ADD
RISK: MEDIUM
WHAT: Bound XML signature inspection to the first 4,096 bytes and test routing.
WHY: The current `path.read_bytes()[:4096]` allocates the entire source before slicing.
APPROACH: Use a binary stream prefix read and synthetic sparse/adversarial fixtures with downstream parsing mocked.
SHORTCOMINGS: This addresses PB06 local probe allocation only; global admission and deployed behavior remain separate.
BEFORE: `head = path.read_bytes()[:4096].lower()`
AFTER: `with path.open("rb") as source: head = source.read(4096).lower()`

## Execution receipt

- Base: Probata `origin/main@382acf6a8f1f919b6f3abc02de40f946629d8461`, isolated branch `codex/d02-pb06-bounded-xml`.
- Changed behavior: XML signature inspection opens the file in binary mode and reads at most 4,096 bytes. Existing `<smses`, `<sms `, and `<mms ` routing remains unchanged.
- Focused local proof: `uv run --extra dev python -m pytest -q tests/test_proffer_xml_probe.py` — 5 passed. A 64 MiB sparse fixture guards the read size and forbids `Path.read_bytes`; an external-entity string is treated as literal prefix data with downstream parsing mocked. No network or XML entity resolution is performed by this probe.
- Broader related proof: `test_format_router.py`, `test_proffer_repair_workflow_contract.py`, and the new probe tests — 14 passed. `ruff check`, `ruff format --check`, and `git diff --check` passed for the changed files.
- Existing test drift: `test_ingest_port.py` constructs platform/evidence requests rejected by the current context-only `IngestRequest` contract (19 failures in the attempted combined run). `test_format_engine_override.py` tries to monkeypatch the frozen `FunctionTool.fn` field (one failure plus teardown error; the attempted combined related run had 30 passes). These tests and contracts were not changed by this slice.
- Secret scan: official Gitleaks v8.30.0, `git --staged --redact=100` against the three explicit staged paths, 0 findings. The final staged bytes are rescanned after this receipt update.
- Proof ceiling: `LOCAL_EXECUTED` for PB06's Python prefix allocation and routing only. No deployed image, full ingestion, bounded live source, aggregate scratch admission, XML parser security, or global D02 acceptance is claimed.

## Handoff

PB06's local bounded-reader defect is repaired and regression-tested. D12 should independently assess the candidate/deployed image before changing its acceptance state. W05, W14, W34, PB02, PB03, and NF-INTAKE-03/05/08 retain their prior open or unverified states.
