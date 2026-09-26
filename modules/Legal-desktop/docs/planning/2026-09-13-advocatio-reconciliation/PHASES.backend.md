# Backend phases

Proposed implementation sequence; no phase was executed during this planning audit. Read REQUIREMENTS.md, STACK.md, LANGUAGE-CONVERGENCE.md, SPLIT.md and GOTCHAS.md first. Phase 0 establishes actual commands/toolchain and scopes concurrent work. Frontend and backend coordinate only at the contract gates in SPLIT.md; phase numbers describe related outcomes, not permission to edit the other lane.

Latest additions: R48–R50 extend B0/B2 inventories to extra legal skills and the fully opened Legal MCP pack. R51 requires the MCP client/server-registry/manual-invocation contract in B2; R52 requires registered port/DNS routing in B0/B8. Read GOTCHAS A7/A8 before remote toolkit reconciliation. B2 also requires the MCP-CLIENT-AND-PORTS.md acceptance slice, beyond the single-tool criterion below.

## B0 — Recover and prove the baseline

**Requirements:** R01, R37, R40–R47. **Gotchas:** S1, S5, A2, A4, A5.

**Work:** Recover remote toolkit/MCP/store handoffs and existing migration receipts; inspect current persistence/auth boundaries and preserve a pre-change source/state inventory.

**Exit criterion:** The plan names local and remote toolkit roots, configured versus historically migrated versus live-verified state, current storage selection and all blocking unknowns; no duplicate import is started.

**Dependency:** Read-only recovery complete; preserve current dirty work.

## B1 — Shared identity, revisions and contracts

**Requirements:** R02–R07, R16, R41, R44, R47. **Gotchas:** S1, S2, S5.

**Work:** Implement atomic version persistence and versioned resource/source/error contracts; retain separate case scope and accepted-evidence ownership.

**Exit criterion:** An injected save failure preserves the previous aggregate, concurrent stale mutation returns a conflict, and reload retains two document revisions plus review provenance.

**Dependency:** Phase 0 receipt and agreed auth/source/revision contract.

## B2 — Complete reference and toolkit bridge

**Requirements:** R08–R15, R37, R45, R47. **Gotchas:** S4, A5.

**Work:** Reconcile complete current toolkit capability/content inventory against existing remote store, preserve baseline manifests, implement missing read/tool adapters and item-level promotion review.

**Exit criterion:** Every inventoried item has a disposition; local/remote counts and IDs reconcile or list explicit gaps, and one tool/MCP call returns a real versioned result with access/error checks and no duplicate authored corpus.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B3 — Claims, chronology and visualizations

**Requirements:** R25–R31, R37, R44. **Gotchas:** S3, A3, A4.

**Work:** Implement bounded event/claim/support projections, gap query, investigation request and version-aware Timesketch/Probata bridge using accepted fork handoffs.

**Exit criterion:** Fixtures with two claims and one supporting span never mark both supported; contrary links and scheduled/actual distinctions survive query/export, and unauthorized upstream edits are rejected.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B4 — Draft proposals and court-language translation

**Requirements:** R13–R20. **Gotchas:** S2, S3, S4, A1.

**Work:** Implement method-pinned drafting/translation proposals, atomic adoption, immutable revision chain, review records and dependency staleness.

**Exit criterion:** A translation fixture does not add unsupported facts; accepting a proposal creates a new revision, rejects a stale base, retains rejected changes and does not approve release.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B5 — Private context, strategy and analytical follow-up

**Requirements:** R21–R24, R29. **Gotchas:** S2, S3, S5.

**Work:** Implement private record revisions, basis/contrary links, strategy decisions/playbooks and accepted analysis-reference import with follow-up tasks.

**Exit criterion:** An imported AI finding retains its original provenance and limitations; legal follow-up never changes source acceptance, and private records fail unauthorized reads.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B6 — Office, forms and release derivatives

**Requirements:** R17, R20, R36, R42. **Gotchas:** S2, S6, A1.

**Work:** Implement bounded LibreOffice/render/document jobs, true-redaction verification, Bates/metadata manifests and immutable release gates; retain independently selected editing bridge.

**Exit criterion:** Fixture conversions produce verified outputs and failure receipts; redacted export reveals no removed text through extraction/revisions, original hashes stay unchanged, and release requires correct version-specific checks.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B7 — Digital firm and fixed workflows

**Requirements:** R13–R16, R22–R24, R32–R36, R45. **Gotchas:** S2, S3, S5.

**Work:** Implement versioned role/method/model configuration, bounded context handoffs, fixed workflow runs, cost/output provenance and callable toolkit integration.

**Exit criterion:** A run records role/provider/model/source/method versions, rejects out-of-scope tool use and never grants an agent approval/filing authority; absent original prompt inputs remain explicit rather than replaced silently.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## B8 — Complete convergence and operational acceptance

**Requirements:** R02, R37–R47 and retained handoff items. **Gotchas:** S1–S6, A1–A6, B1.

**Work:** Verify remaining external integrations/schedules/callable resources, deployment-ready auth/data/job contracts and restore/correction paths; register/read back documentation when Docstore is available.

**Exit criterion:** A capability-by-capability receipt ties version, scope, test and artifact to each accepted item; remote availability/deployment are only marked proven from real readback, and all unresolved holds have an owner and next action.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.
