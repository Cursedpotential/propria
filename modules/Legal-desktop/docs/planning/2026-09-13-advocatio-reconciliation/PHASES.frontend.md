# Frontend phases

Proposed implementation sequence; no phase was executed during this planning audit. Read REQUIREMENTS.md, STACK.md, LANGUAGE-CONVERGENCE.md, SPLIT.md and GOTCHAS.md first. Phase 0 establishes actual commands/toolchain and scopes concurrent work. Frontend and backend coordinate only at the contract gates in SPLIT.md; phase numbers describe related outcomes, not permission to edit the other lane.

Latest additions: R48–R50 add legal skill/ZIP discovery and the compact-flag UX; apply STATUS-AND-FLAGS.md to every phase. R51 adds the generic MCP client, server registrations and manual calls to F2. R52 makes the new port/DNS standard part of F0/F8 integration. F2 also requires the MCP-CLIENT-AND-PORTS.md acceptance slice, beyond the single-tool criterion below.

## F0 — Recover and prove the baseline

**Requirements:** R01, R37, R40–R47. **Gotchas:** S1, S5, A2, A4, A5.

**Work:** Inventory existing routes, auth/proxy/streaming and donor surface dependencies; establish current web build and a minimal common-stack parity harness without replacing the app.

**Exit criterion:** A current-build receipt and a common-client browser fixture are saved; every server-side Next dependency has an owner and migration disposition, and Windows native proof is separately marked pass or unresolved.

**Dependency:** Read-only recovery complete; preserve current dirty work.

## F1 — Shared identity, revisions and contracts

**Requirements:** R02–R07, R16, R41, R44, R47. **Gotchas:** S1, S2, S5.

**Work:** Build shared shell, source inspector, accessible status components and contract fixtures using the agreed TypeScript client baseline.

**Exit criterion:** A reference and its exact source version display in the shell and simple-view fixture; denied, stale and missing-source states are visibly distinct and keyboard reachable.

**Dependency:** Phase 0 receipt and agreed auth/source/revision contract.

## F2 — Complete reference and toolkit bridge

**Requirements:** R08–R15, R37, R45, R47. **Gotchas:** S4, A5.

**Work:** Build reference/topic search, source/version detail, method help and tool invocation/result views; expose scoped simple-surface reads.

**Exit criterion:** A user finds a legal reference, reads its method guidance, opens its cited original and invokes one permitted toolkit capability through the shared interface; pending verification is never shown as validated.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F3 — Claims, chronology and visualizations

**Requirements:** R25–R31, R37, R44. **Gotchas:** S3, A3, A4.

**Work:** Build linked timeline/claim views, support inspector and gap reports; use retained visual engines by concrete view and add a scoped Timesketch integration slice.

**Exit criterion:** The same event opens the same claim/source across list and timeline views; one partly supported event remains partial, an uncertain date remains uncertain, and gap-report selection reveals the unsupported proposition.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F4 — Draft proposals and court-language translation

**Requirements:** R13–R20. **Gotchas:** S2, S3, S4, A1.

**Work:** Build original/proposal diff, accept/reject/edit controls, contextual methods and four independent review panels; evaluate rich-editor revision roundtrip.

**Exit criterion:** The user accepts one wording change and rejects another, sees the preserved original and separate factual/legal findings, and cannot apply a stale proposal silently.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F5 — Private context, strategy and analytical follow-up

**Requirements:** R21–R24, R29. **Gotchas:** S2, S3, S5.

**Work:** Build context, risk/anticipated-accusation, defense/strategy and analysis-follow-up views with clear status and source links.

**Exit criterion:** A reported event, anticipated accusation and AI analysis remain visibly distinct from accepted evidence, and a private record is absent from an unauthorized simple-surface view.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F6 — Office, forms and release derivatives

**Requirements:** R17, R20, R36, R42. **Gotchas:** S2, S6, A1.

**Work:** Deliver selected interactive editing/form workflow, tracked changes and render/redact/Bates/scrub previews with explicit job state.

**Exit criterion:** A representative document/form can be edited and reviewed, converted and reopened with agreed layout/revision fidelity; failures are visible and original-versus-derivative identity is clear.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F7 — Digital firm and fixed workflows

**Requirements:** R13–R16, R22–R24, R32–R36, R45. **Gotchas:** S2, S3, S5.

**Work:** Port recovered firm roles and managing/senior/red-team/clerk/librarian/researcher/exhibit-clerk handoffs into readable run/review views.

**Exit criterion:** A task passes through managing associate, specialist/research, senior review and red-team challenge with visible disagreements and human disposition; original prompt lineage is linked.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.

## F8 — Complete convergence and operational acceptance

**Requirements:** R02, R37–R47 and retained handoff items. **Gotchas:** S1–S6, A1–A6, B1.

**Work:** Complete remaining agreed ports/adapters and common-surface navigation; verify phone, desktop, keyboard and accessibility; retire transitional routes only after parity.

**Exit criterion:** Each capability in the discussion, toolkit and retained integration registers has working proof or an explicit deferred decision; the common surface reaches all accepted modules without a hidden parallel authored state.

**Dependency:** Previous phase's relevant contract and acceptance receipt; independent mock/fixture work may proceed earlier.
