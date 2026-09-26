# Backend task prompts

These prompts are future work instructions, not evidence of execution. Repository root: `E:\AI_Workspace\Projects\Propria\Legal-desktop`. Read effective AGENTS instructions and current dirty status before editing. You are not alone in the codebase: preserve others' changes, own only the named lane and use explicit file allowlists. Never permanently delete; quarantine only after parity and ownership checks. Existing public/private data is not a test fixture.

Frontend Phase F0 must establish a `test:acceptance` script and record its runner; commands below using it are a proposed verification contract, not a currently existing script. Backend per-phase test files named below are new acceptance suites to create during that phase, not tests claimed to exist now. Do not repeat broad tests after a passing phase unless new changes justify it.

## B0

```text
Implement B0 — Recover and prove the baseline from PHASES.backend.md. Read GOTCHAS.md S1, S5, A2, A4, A5 and requirements R01, R37, R40–R47.
Ownership: api/legal_workspace/, tests/test_advocatio_baseline.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B0/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Recover remote toolkit/MCP/store handoffs and existing migration receipts; inspect current persistence/auth boundaries and preserve a pre-change source/state inventory.
Completion bar: The plan names local and remote toolkit roots, configured versus historically migrated versus live-verified state, current storage selection and all blocking unknowns; no duplicate import is started.
Verification: From the repository root, inspect pyproject.toml and current storage wiring; run uv run pytest tests/test_persist.py against isolated fixtures after verifying no owner database is targeted. Record configured/migrated/reachable toolkit evidence separately.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B1

```text
Implement B1 — Shared identity, revisions and contracts from PHASES.backend.md. Read GOTCHAS.md S1, S2, S5 and requirements R02–R07, R16, R41, R44, R47.
Ownership: api/legal_workspace/, tests/test_advocatio_revision_contract.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B1/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement atomic version persistence and versioned resource/source/error contracts; retain separate case scope and accepted-evidence ownership.
Completion bar: An injected save failure preserves the previous aggregate, concurrent stale mutation returns a conflict, and reload retains two document revisions plus review provenance.
Verification: From repository root, run uv run pytest tests/test_advocatio_revision_contract.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B2

```text
Implement B2 — Complete reference and toolkit bridge from PHASES.backend.md. Read GOTCHAS.md S4, A5 and requirements R08–R15, R37, R45, R47.
Ownership: api/legal_workspace/, tests/test_advocatio_toolkit_bridge.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B2/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Reconcile complete current toolkit capability/content inventory against existing remote store, preserve baseline manifests, implement missing read/tool adapters and item-level promotion review.
Completion bar: Every inventoried item has a disposition; local/remote counts and IDs reconcile or list explicit gaps, and one tool/MCP call returns a real versioned result with access/error checks and no duplicate authored corpus.
Verification: From repository root, run uv run pytest tests/test_advocatio_toolkit_bridge.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B3

```text
Implement B3 — Claims, chronology and visualizations from PHASES.backend.md. Read GOTCHAS.md S3, A3, A4 and requirements R25–R31, R37, R44.
Ownership: api/legal_workspace/, tests/test_advocatio_claim_timeline.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B3/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement bounded event/claim/support projections, gap query, investigation request and version-aware Timesketch/Probata bridge using accepted fork handoffs.
Completion bar: Fixtures with two claims and one supporting span never mark both supported; contrary links and scheduled/actual distinctions survive query/export, and unauthorized upstream edits are rejected.
Verification: From repository root, run uv run pytest tests/test_advocatio_claim_timeline.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B4

```text
Implement B4 — Draft proposals and court-language translation from PHASES.backend.md. Read GOTCHAS.md S2, S3, S4, A1 and requirements R13–R20.
Ownership: api/legal_workspace/, tests/test_advocatio_proposal_review.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B4/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement method-pinned drafting/translation proposals, atomic adoption, immutable revision chain, review records and dependency staleness.
Completion bar: A translation fixture does not add unsupported facts; accepting a proposal creates a new revision, rejects a stale base, retains rejected changes and does not approve release.
Verification: From repository root, run uv run pytest tests/test_advocatio_proposal_review.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B5

```text
Implement B5 — Private context, strategy and analytical follow-up from PHASES.backend.md. Read GOTCHAS.md S2, S3, S5 and requirements R21–R24, R29.
Ownership: api/legal_workspace/, tests/test_advocatio_private_strategy.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B5/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement private record revisions, basis/contrary links, strategy decisions/playbooks and accepted analysis-reference import with follow-up tasks.
Completion bar: An imported AI finding retains its original provenance and limitations; legal follow-up never changes source acceptance, and private records fail unauthorized reads.
Verification: From repository root, run uv run pytest tests/test_advocatio_private_strategy.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B6

```text
Implement B6 — Office, forms and release derivatives from PHASES.backend.md. Read GOTCHAS.md S2, S6, A1 and requirements R17, R20, R36, R42.
Ownership: api/legal_workspace/, tests/test_advocatio_document_jobs.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B6/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement bounded LibreOffice/render/document jobs, true-redaction verification, Bates/metadata manifests and immutable release gates; retain independently selected editing bridge.
Completion bar: Fixture conversions produce verified outputs and failure receipts; redacted export reveals no removed text through extraction/revisions, original hashes stay unchanged, and release requires correct version-specific checks.
Verification: From repository root, run uv run pytest tests/test_advocatio_document_jobs.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B7

```text
Implement B7 — Digital firm and fixed workflows from PHASES.backend.md. Read GOTCHAS.md S2, S3, S5 and requirements R13–R16, R22–R24, R32–R36, R45.
Ownership: api/legal_workspace/, tests/test_advocatio_firm_workflows.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B7/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Implement versioned role/method/model configuration, bounded context handoffs, fixed workflow runs, cost/output provenance and callable toolkit integration.
Completion bar: A run records role/provider/model/source/method versions, rejects out-of-scope tool use and never grants an agent approval/filing authority; absent original prompt inputs remain explicit rather than replaced silently.
Verification: From repository root, run uv run pytest tests/test_advocatio_firm_workflows.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## B8

```text
Implement B8 — Complete convergence and operational acceptance from PHASES.backend.md. Read GOTCHAS.md S1–S6, A1–A6, B1 and requirements R02, R37–R47 and retained handoff items.
Ownership: api/legal_workspace/, tests/test_advocatio_convergence.py and new docs/planning/2026-09-13-advocatio-reconciliation/verification/B8/ only. Any SQL migration or sibling adapter requires an explicit phase file allowlist first.
Verify remaining external integrations/schedules/callable resources, deployment-ready auth/data/job contracts and restore/correction paths; register/read back documentation when Docstore is available.
Completion bar: A capability-by-capability receipt ties version, scope, test and artifact to each accepted item; remote availability/deployment are only marked proven from real readback, and all unresolved holds have an owner and next action.
Verification: From repository root, run uv run pytest tests/test_advocatio_convergence.py and uv run ruff check on the exact changed Python paths. Add failure/conflict/unauthorized fixtures required by the phase; do not touch production state.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```
