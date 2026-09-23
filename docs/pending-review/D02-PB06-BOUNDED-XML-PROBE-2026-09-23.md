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

## Independent review and CI hold — 2026-09-23

> Byline: Codex · GPT-6 · independent D02 PR #29 reviewer. Reviewed feature commit `e81daf381fb637bfcefd2e9b0e62292b72a6e15f` against `origin/main@382acf6a8f1f919b6f3abc02de40f946629d8461` in an isolated worktree.

- **Verdict:** The Python probe now opens `.xml` in binary mode and reads at most 4,096 bytes before lowercasing and checking the existing markers. It does not parse XML, resolve entities, or issue network requests. Routing for SMS Backup & Restore markers and ordinary generic XML is unchanged from the parent commit. A marker beginning beyond byte 4,096 remains outside the probe's detection window by design. This is local allocation proof, not an RSS measurement or a claim about downstream parser security.
- **Existing selection limitation:** The unchanged substring predicate can route otherwise valid generic XML to the SMS parser if a comment or CDATA section contains a literal `<sms ` marker. The same predicate is present on `main@382acf6`; this PR neither introduces nor repairs that ambiguity. The regression cases establish no new misclassification for ordinary generic XML, not correctness for every valid XML document. A separate parser-selection contract and test are needed before claiming that broader property.
- **Additional regression coverage:** Empty `.xml`, a truncated marker, small SMS XML, generic valid XML, and a non-XML extension now exercise routing and whether the probe opens the file. The 64 MiB sparse fixture still asserts a single bounded read and rejects `Path.read_bytes`. Focused plus related tests: `uv run --extra dev python -m pytest -q tests/test_proffer_xml_probe.py tests/test_format_router.py tests/test_proffer_repair_workflow_contract.py` — 19 passed. Ruff 0.15.17 check and format on the two D02 code/test files, plus `git diff --check`, passed.
- **Baseline test drift reproduced:** On untouched current `main` with the same local Python environment, `test_ingest_port.py` and `test_format_engine_override.py` produced 15 failures, 22 passes, and 1 teardown error. Fourteen failures reject obsolete platform/evidence lanes under the current context-only `IngestRequest`; one test tries to mutate frozen `FunctionTool.fn` and its teardown fails for the same reason. These tests are outside this PB06 slice.
- **Full-suite ceiling:** The local default unit collection could not begin: after supplying the missing `sqlparse` dependency in the isolated environment, collection still reports 10 errors from retired/missing SQL migration files and related imports (first missing path: `sql/0036_context_import_foundation.sql`). Installing the full `requirements.txt` on Windows also fails because `uvloop==0.22.1` does not support Windows. No full-suite pass is claimed.
- **Secret scan:** Official Gitleaks v8.30.0 Windows x64 binary was verified against its published SHA-256 (`54fe94f644b832dd08e8c3a5915efb3bfa862386d59fb27ca0792cb687a83573`). Both the staged D02-only follow-up and `origin/main..HEAD` feature history scanned with `--redact=100`; each reported zero leaks. The final staged receipt bytes are rescanned before commit.
- **Exact GitHub CI failure:** Validate/Python on PR #29 fails at `ruff format --config pyproject.toml --check server tests`, before pytest. Ruff 0.15.17 names only `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`; the exact same two failures reproduce on untouched `main@382acf6` locally. Those files are outside D02 ownership and were not changed in this PR. The required live integration job is skipped because Validate fails. CI HOLD remains until the owning lane repairs the baseline and this PR is retested. Keep PR #29 draft and unmerged.
- **Release boundary:** No deployment, intended-image readback, live-source run, XML parser/entity safety audit, RSS budget measurement, or D12 acceptance was performed. PB06 should not be marked fully closed on this receipt alone.
