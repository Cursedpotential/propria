# Advocatio: working roadmap and model handoff

Updated 2026-09-13. Start here when continuing with another model. The detailed research is already saved; retrieve the relevant report rather than repeating the inventory.

Task A completed 2026-09-20 with three explicitly authorized cheaper parallel
agents. Current verified repository: `E:\AI_Workspace\Projects\Propria\modules\Legal-desktop`.
Use [the consolidated prompt](continuation/A-START.md), [completion receipt](continuation/A-receipt.md)
and [383-row checklist](continuation/A-capability-port-map.csv) for that work.

## What we are building

One legal-work module within the unified Propria surface, with a simpler phone/reference view using the same underlying sources. It brings together the complete Family Court Toolkit, legal references and skills, an MCP client with manual tool calls, case context and strategy, evidence-linked timelines, reviewed document creation and the multidisciplinary digital firm.

The common frontend uses the settled TanStack/Storybook/Glide direction, with Tauri for desktop. The recovered migration recommendation is React/TypeScript/Vite with TanStack Router/Query; exact package adoption is still to be recorded. Existing services can remain callable while their features are progressively brought into the unified surface. Compact flags replace repeated disclaimer blocks.

## What we have and what we still need

| Workstream | Already available | Still needed |
|---|---|---|
| Existing application | Canonical independent repository, working-tree source, navigation/split/assistant foundations, strategy, drafting, source imports, exhibits and review concepts | Current build proof; dependable transactions/revision history; common-stack migration and real workflow acceptance. |
| Legal reference collection | Full toolkit source, 193 source-ledger records, 23 reference payloads, current and earlier variants, original resource batch | Confirm remote population by IDs/hashes; reconcile differences; shared versioned access and item-level source checks. |
| Toolkit and MCP | Full plugin has 27 handlers, 36 skill files, nine agents, commands/procedures; eight reduced-console tools connected here and one read call proved | Complete callable capability coverage; generic server registration, discovery, manual calls, resource/prompt access and scoped agent use. |
| Remote services | Surreal-first store code, deployment/migration handoffs, new service-port registry | Current named-service route and database readback; Advocatio/Toolkit allocation; gateway/tool parity. Historical endpoint refusal does not identify current service state. |
| Skills and reusable code | Full toolkit, additional legal plugin discovery, substantive Legal MCP ZIP review, FreeEed reports | Select useful methods/functions, repair evidenced defects, bind them to tasks and review criteria; preserve origin/version. |
| Timelines and claims | Begun Timesketch fork with curation handoffs; two retained timeline libraries; current docket screens | Shared event/claim/support contract, source inspector, partial/contrary support, gap reports and actual fork integration. |
| Documents | Current outlines/drafts/review, donor parsing/report code and complete office handoffs | Proposed changes and immutable revisions, original-to-court-language translator, four independent review dimensions, interactive forms/editor, LibreOffice and verified derivatives. |
| Personal context and strategy | Private notes/red-team foundations and source material pointers | Structured relationship/case history, vulnerabilities, anticipated accusations, defenses, decisions/playbooks and analysis follow-up. |
| Digital firm | Existing role/provider machinery and owner's described operating model | Original Gemini/multimodel prompt series; exact role handoffs and testable workflows, including exhibit clerk. |
| Other integrations | Handoff-backed Timesketch, timeline engines, Evidence.dev and named graph/report/tool candidates | Per-component port/call/adapt/defer decision and proof in the common surface. |

Evidence and exact paths: [current app](inputs/current-app/REPORT.md), [toolkit/store](inputs/stack/TOOLKIT-CAPABILITIES.md), [handoffs](inputs/handoffs/HANDOFF-RECOVERY.md), [Legal MCP code](inputs/handoffs/LEGAL-MCP-PACK-REVIEW.md), [additional skills](inputs/current-app/LEGAL-CAPABILITY-DISCOVERY.md). These reports are the retrieval index, not an instruction to read every source again.

## How we get there

| Order | Milestone | What proves it is done |
|---|---|---|
| 1 | Establish current baseline and existing remote toolkit coverage | Build/test receipt, actual storage/auth map, remote/local reconciliation or precise unresolved fields, named-service/port disposition. |
| 2 | Stabilize source identity and persistence | Save failure preserves previous state; revisions survive reload; stale edits conflict; shared source/version contract is concrete. |
| 3 | Deliver references, skills and callable tools | Same pinned reference opens in full/simple views; register a server and call a tool manually and through an agent. Then extend verified coverage to the full toolkit. |
| 4 | Connect claims and timelines | Events, claims and exact sources link across views; partial support stays partial; missing-evidence reports are reproducible; Timesketch bridge works. |
| 5 | Deliver document proposals and private strategy | Preserve original language; adopt/reject individual changes; display separate review findings; save context, risks and analysis follow-up. |
| 6 | Complete office and firm workflows | Real form/edit/render/derivative proof, hearing/discovery outputs and recovered firm handoffs with attributable results. |
| 7 | Finish staged convergence and operational integration | Each accepted capability has an integrated/callable route and proof; remaining bridges have explicit next steps; service cutovers follow the new registry. |

Full criteria: [coordinated phases](PHASES.md). Interface contracts: [SPLIT.md](SPLIT.md) and [MCP client/ports](MCP-CLIENT-AND-PORTS.md). Known source-level defects: [GOTCHAS.md](GOTCHAS.md). The milestone order is a dependency sequence, not a promise that all seven fit into one week.

## Bounded work another model can do next

Each task is a single session-sized assignment. Use one task at a time. The first four are useful preparation work that does not require a deployed service or a new architecture decision.

| Task | State | Read only what is needed | Produce / completion test |
|---|---|---|---|
| A — Turn toolkit inventory into a port checklist | Complete 2026-09-20; [383/383 verified](continuation/A-verification.json), [receipt](continuation/A-receipt.md) | `inputs/stack/toolkit-capabilities.json`, toolkit report | `continuation/A-capability-port-map.csv`: every existing inventory row keeps its ID or origin key and gets a proposed shared-resource, method, callable-tool, code-adaptation or operational-support destination. Mark unknown route explicitly. Input/output coverage reconciles; no capability silently dropped. |
| B — Map reusable skills to actual work | Ready; independent of A | Additional-skill JSON/report, toolkit methods, Legal MCP prompt catalog; targeted skill files only | `continuation/B-method-map.csv`: drafting, review, source research, translation, strategy, exhibit and hearing/discovery method mappings, with version/path, output and executable-versus-guidance distinction. Identify overlap; preserve differing variants. |
| C — Specify and test the MCP contract shape | Ready; reuse A if available | MCP client plan, `SPLIT.md`, current registered tool schemas and relevant handlers | `continuation/C-mcp-contract.yaml` plus synthetic examples for connection/discovery/manual call/result/error. Validate schema/fixture consistency. No registrations, credentials or external tool calls needed. |
| D — Write support/proposal acceptance examples | Ready; independent of A–C | Requirements R16–R19 and R25–R29, current support/proposal findings | `continuation/D-review-fixtures.json`: synthetic examples for partial support, unrelated citation, conflicting evidence, uncertain date, unsupported translation addition and stale proposal. Each has an expected result and exact criterion. Parse and check fixture completeness. |
| E — Prove the existing application baseline | Ready for an implementation session after checking concurrent ownership | Current app report; effective AGENTS chain; actual manifests/tests | `continuation/E-baseline.md`: exact commit/dirty scope, build and isolated relevant test commands/results, auth/proxy/storage mapping, precise blockers. No broad dependency upgrades or fixes bundled into this task. |
| F — Reconcile remote Toolkit population | Pending a verified read-only connection | Toolkit/store report, current port standard and relevant deployment handoff | `continuation/F-remote-reconciliation.md`: configured route, live route proof, source/reference counts and IDs/hash differences using read-only queries. If connection is absent, record exactly what is missing and move to an independent task. Do not use the loader's schema-initializing dry-run. |
| G — First targeted persistence correction | After E; implementation scope must be available | GOTCHAS S1/S2 and exact store/revision tests | Fix one atomic-save defect in `api/legal_workspace/db/` with an isolated failure-injection test showing previous state survives. Preserve concurrent work. Record changed paths and test results; do not combine with schema migration, frontend port or remote deployment. |

Suggested order for a week of smaller-model sessions: A, B, C, D, then E; F when its connection is available; G after the baseline and file ownership are clear. Stop after each task's output and verification. Resume from its receipt rather than loading the whole planning package. The remaining frontend/backend prompts are available when those inputs are ready.

## What still needs an input or a decision

- **Owner files:** original digital-firm prompts and vLex examples. Their absence does not block A–G.
- **Current service evidence:** remote Toolkit population/route and registered product code. Follow the existing new port scheme; never allocate by guessing a free number.
- **Implementation selections:** exact common frontend package baseline; interactive editor/native revision bridge; staged Timesketch presentation convergence. Prepare options from recovered handoffs rather than reopening settled overall direction.
- **Substantive source acceptance:** resource corrections, currentness and factual support are per-item review work. Another model can collect evidence and propose findings; use the separate review dimensions already specified.

## Copy this into another model

```text
Continue Advocatio from this roadmap:
E:\AI_Workspace\Projects\Propria\modules\Legal-desktop\docs\planning\2026-09-13-advocatio-reconciliation\ROADMAP.md

Canonical repository: E:\AI_Workspace\Projects\Propria\modules\Legal-desktop.
Read the applicable AGENTS instructions and current dirty status. Start with the
first unfinished Ready task A–E, or a task I explicitly name. Read only that
task's linked inputs and the source passages needed to finish it. Existing
reports cover the initial research; do not repeat broad searches or audits.

Complete one bounded task, verify its output, update its State cell in ROADMAP.md
with the result link, and leave a short continuation/<task>-receipt.md containing:
what changed, exact paths, verification, unresolved inputs and next ready task.
Do not rewrite the entire plan. Use cheaper capable agents for independent
sections with explicit, non-overlapping file ownership; the coordinator merges
and verifies shared outputs. This is explicitly authorized by the owner.
For a blocker, record it once and finish useful independent work
within the selected task; do not repeatedly retry or invent missing evidence.

Preserve existing changes and original sources. No permanent deletion. No
deployment or external registration as part of A–E. Follow the new port/DNS
standard for any later bring-up. Keep UI status as compact flags. Use synthetic
fixtures for tests. Do not expose credentials or private case material in reports.

The target remains the complete unified surface and toolkit capability set;
callable services are valid intermediate steps. Report only what was actually
completed and the exact next step.
```

## Current handoff state

Planning and source reconciliation are complete for the reviewed scope. Task A is complete with 383 inventory entries mapped and verified; B–G remain pending. No application implementation milestone is marked complete by this roadmap. Start with **Task B** unless the owner selects another task. Nothing in this document schedules autonomous work or commits to a model's performance or cost.
