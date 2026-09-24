<!-- Updated by: Codex (W18 enrichment-fault review remediation) | Date: 2026-09-24 | Rev: 2 | Platform: Codex Desktop / win32 | Changes: exact-head review remediation and verification | Context: PR #37 review found mutable-payload, public-invariant, and malformed-JSON classification gaps -->

# W18 Null/Truncated Enrichment Contract Receipt

## Decision

**DRAFT / HOLD — bounded source contract only; not integrated or deployed.**

This slice establishes a framework-neutral fail-closed boundary for W18/N08. It does not claim that the historical Intake enrichment runner, snapshot route, PostgreSQL ledger, or deployed search surface uses the contract.

## Source boundary

- Repository: `Cursedpotential/probata`
- Base: `origin/main@382acf6a8f1f919b6f3abc02de40f946629d8461`
- Branch: `codex/w18-enrichment-fault-20260923`
- W18 source: `.reconciliation/2026-09-23-orchestration-input/agents/W18/W18_PARSER_FAULT_READINESS_2026-09-23.md`
- Preserved finding: N08 / `AC-F-N08`

## Proven behavior

- Every attempt receipt retains the exact source id/version/hash plus inventory and extracted-text references.
- Exact provider output receives a SHA-256 identity even when empty, malformed, truncated, timed out, cancelled, or unsupported.
- A nullable field accepted by the caller's schema remains `null`; the contract does not invent a value.
- A null required field becomes a bounded `schema_validation` failure.
- Provider length termination and structurally truncated JSON become `partial/truncated_output`, never `completed`.
- Structurally closed malformed JSON is `failed/invalid_json`, not a truncated partial outcome.
- Malformed nontruncated JSON, provider errors, timeouts, and cancellation cannot publish a snapshot.
- Only schema-validated output can create a new immutable snapshot.
- Publication recomputes the canonical payload digest, so caller mutation after evaluation cannot publish under a stale validated hash.
- The public publication-decision model binds the active snapshot to the receipt's exact source, attempt, model, configuration, raw-output, and payload identities.
- A partial or failed attempt leaves the prior validated snapshot active.
- Restart recovery selects the highest nonconflicting validated sequence for the same source and fails closed on conflicting sequence claims.

## Files

- `server/contracts/enrichment.py`
- `tests/test_enrichment_fault_contract.py`
- `docs/reviews/2026-09-23-w18-enrichment-fault-contract.md`

## Verification

Observed in the isolated worktree:

- Focused plus adjacent contract suites after review remediation: **55 passed** (`test_enrichment_fault_contract.py`, `test_semantica_phase1_worker.py`, and `test_content_chunk_context_thread_contracts.py`).
- Focused Ruff lint: pass.
- Repository-wide `ruff check server tests`: pass.
- Focused Ruff format check: pass.
- Targeted mypy for `server/contracts/enrichment.py`: pass with no issues.
- `git diff --check`: pass.
- Gitleaks 8.30.0 against each of the three new files: zero findings.
- Whole-tree Gitleaks remains unsuitable as a PR-owned gate: it reports 75 historical findings outside this three-file slice.
- Repository-wide format check remains red only on two unchanged externally owned files: `tests/test_authentik_deploy_contract.py` and `tests/test_docker_user_firewall_contract.py`.
- A broader adjacent run including unchanged `tests/test_ingest_port.py` produced 49 passes and 14 failures. All 14 fail while constructing the pre-existing `IngestRequest`: the base contract permits only the `context` lane while those unchanged tests request `platform` or `evidence`. This PR does not edit that separate baseline.
- CocoIndex semantic search was attempted, but the existing global CCC safety-fault guard blocked it before search. Repository discovery therefore used scoped `rg` plus direct source inspection; the CCC fault record was not altered.
- Independent review of head `2c5c880fb2cf72849df0d3be76a5af0c2959378a` found three issues: mutable caller payload after evaluation, an incomplete public decision invariant, and closed malformed JSON misclassified as truncated. All three have focused regressions and were remediated in `ad8109859eb44e869d16ffb72990321591578de3`.

- Initial implementation commit: `cf0f05327c8336706a1f8c298532d37956866015`
- Exact-head review remediation commit: `ad8109859eb44e869d16ffb72990321591578de3`
- Draft/HOLD pull request: <https://github.com/Cursedpotential/probata/pull/37>

The final exact PR head is reported by the orchestrator handoff because a commit cannot contain its own hash. No integration test, provider call, database write, live source read, deployment, or production readback was performed.

## Open activation gates

1. Choose the actual Intake/Probata enrichment producer and durable receipt/snapshot store.
2. Bind this contract or an independently reviewed equivalent to that producer.
3. Persist receipts and validated snapshots atomically with explicit activation history.
4. Prove a process restart reads the last validated snapshot from the durable store.
5. Prove inventory and extracted text remain searchable when enrichment is partial or failed.
6. Run an authorized mock-provider integration test for truncation, nullability, timeout, and retry behavior.
7. Deploy through the owning Coolify application and perform live readback before closing N08/W18.
