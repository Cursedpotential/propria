# Frontend task prompts

These prompts are future work instructions, not evidence of execution. Repository root: `E:\AI_Workspace\Projects\Propria\Legal-desktop`. Read effective AGENTS instructions and current dirty status before editing. You are not alone in the codebase: preserve others' changes, own only the named lane and use explicit file allowlists. Never permanently delete; quarantine only after parity and ownership checks. Existing public/private data is not a test fixture.

Frontend Phase F0 must establish a `test:acceptance` script and record its runner; commands below using it are a proposed verification contract, not a currently existing script. Backend per-phase test files named below are new acceptance suites to create during that phase, not tests claimed to exist now. Do not repeat broad tests after a passing phase unless new changes justify it.

## F0

```text
Implement F0 — Recover and prove the baseline from PHASES.frontend.md. Read GOTCHAS.md S1, S5, A2, A4, A5 and requirements R01, R37, R40–R47.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F0/ only; no backend or sibling source edits.
Inventory existing routes, auth/proxy/streaming and donor surface dependencies; establish current web build and a minimal common-stack parity harness without replacing the app.
Completion bar: A current-build receipt and a common-client browser fixture are saved; every server-side Next dependency has an owner and migration disposition, and Windows native proof is separately marked pass or unresolved.
Verification: From web/, inspect package.json, then run npm run build and npx tsc --noEmit on the preserved current implementation. Record environment failures accurately; create an isolated synthetic browser acceptance harness and actual test:acceptance command before migration.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F1

```text
Implement F1 — Shared identity, revisions and contracts from PHASES.frontend.md. Read GOTCHAS.md S1, S2, S5 and requirements R02–R07, R16, R41, R44, R47.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F1/ only; no backend or sibling source edits.
Build shared shell, source inspector, accessible status components and contract fixtures using the agreed TypeScript client baseline.
Completion bar: A reference and its exact source version display in the shell and simple-view fixture; denied, stale and missing-source states are visibly distinct and keyboard reachable.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F1 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F2

```text
Implement F2 — Complete reference and toolkit bridge from PHASES.frontend.md. Read GOTCHAS.md S4, A5 and requirements R08–R15, R37, R45, R47.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F2/ only; no backend or sibling source edits.
Build reference/topic search, source/version detail, method help and tool invocation/result views; expose scoped simple-surface reads.
Completion bar: A user finds a legal reference, reads its method guidance, opens its cited original and invokes one permitted toolkit capability through the shared interface; pending verification is never shown as validated.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F2 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F3

```text
Implement F3 — Claims, chronology and visualizations from PHASES.frontend.md. Read GOTCHAS.md S3, A3, A4 and requirements R25–R31, R37, R44.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F3/ only; no backend or sibling source edits.
Build linked timeline/claim views, support inspector and gap reports; use retained visual engines by concrete view and add a scoped Timesketch integration slice.
Completion bar: The same event opens the same claim/source across list and timeline views; one partly supported event remains partial, an uncertain date remains uncertain, and gap-report selection reveals the unsupported proposition.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F3 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F4

```text
Implement F4 — Draft proposals and court-language translation from PHASES.frontend.md. Read GOTCHAS.md S2, S3, S4, A1 and requirements R13–R20.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F4/ only; no backend or sibling source edits.
Build original/proposal diff, accept/reject/edit controls, contextual methods and four independent review panels; evaluate rich-editor revision roundtrip.
Completion bar: The user accepts one wording change and rejects another, sees the preserved original and separate factual/legal findings, and cannot apply a stale proposal silently.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F4 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F5

```text
Implement F5 — Private context, strategy and analytical follow-up from PHASES.frontend.md. Read GOTCHAS.md S2, S3, S5 and requirements R21–R24, R29.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F5/ only; no backend or sibling source edits.
Build context, risk/anticipated-accusation, defense/strategy and analysis-follow-up views with clear status and source links.
Completion bar: A reported event, anticipated accusation and AI analysis remain visibly distinct from accepted evidence, and a private record is absent from an unauthorized simple-surface view.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F5 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F6

```text
Implement F6 — Office, forms and release derivatives from PHASES.frontend.md. Read GOTCHAS.md S2, S6, A1 and requirements R17, R20, R36, R42.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F6/ only; no backend or sibling source edits.
Deliver selected interactive editing/form workflow, tracked changes and render/redact/Bates/scrub previews with explicit job state.
Completion bar: A representative document/form can be edited and reviewed, converted and reopened with agreed layout/revision fidelity; failures are visible and original-versus-derivative identity is clear.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F6 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F7

```text
Implement F7 — Digital firm and fixed workflows from PHASES.frontend.md. Read GOTCHAS.md S2, S3, S5 and requirements R13–R16, R22–R24, R32–R36, R45.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F7/ only; no backend or sibling source edits.
Port recovered firm roles and managing/senior/red-team/clerk/librarian/researcher/exhibit-clerk handoffs into readable run/review views.
Completion bar: A task passes through managing associate, specialist/research, senior review and red-team challenge with visible disagreements and human disposition; original prompt lineage is linked.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F7 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```

## F8

```text
Implement F8 — Complete convergence and operational acceptance from PHASES.frontend.md. Read GOTCHAS.md S1–S6, A1–A6, B1 and requirements R02, R37–R47 and retained handoff items.
Ownership: web/ and new docs/planning/2026-09-13-advocatio-reconciliation/verification/F8/ only; no backend or sibling source edits.
Complete remaining agreed ports/adapters and common-surface navigation; verify phone, desktop, keyboard and accessibility; retire transitional routes only after parity.
Completion bar: Each capability in the discussion, toolkit and retained integration registers has working proof or an explicit deferred decision; the common surface reaches all accepted modules without a hidden parallel authored state.
Verification: From web/, run npx tsc --noEmit, npm run build, and the F0-established npm run test:acceptance -- --grep F8 against synthetic phase fixtures. Do not declare a browser or native outcome from static compilation alone.
Persist the exact result, source/version and unresolved dependencies in the phase verification directory. Do not deploy, transmit legal work or silently drop a held capability to pass the phase.
```
