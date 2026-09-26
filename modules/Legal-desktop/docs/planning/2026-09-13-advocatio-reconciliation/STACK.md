# Stack reconciliation

Date: 2026-09-13. Decision evidence: [exact conversation and manifest recovery](inputs/stack/STACK-RECOVERY.md). Implementation evidence: [current app audit](inputs/current-app/REPORT.md). Application runtime acceptance and migration remain future work; the bounded successful MCP call is recorded separately in MCP-CLIENT-AND-PORTS.md.

## Settled direction versus proposed mechanics

The recovered owner decision is **frontend unification**, beyond matching visual tokens. On September 12, 2026 (afternoon Eastern), the owner said the apps were supposed to share a stack with Tauri added for desktop, rejected fractured frameworks, and explicitly settled **TanStack, Storybook and Glide**. The initial recollection of 'today' in the September 13 discussion is corrected by the timestamped source, not used to discard the decision.

The ensuing assistant recommendation was **React + TypeScript + Vite + TanStack Router/Query + Storybook + Glide + Propria design tokens**, with Tauri wrapping the common client. That exact Vite/Router/Query selection instead of TanStack Start is a documented recommendation; a separate owner approval of every package detail was not found. Use it as the migration planning baseline, while keeping this distinction visible. Do not re-open the settled TanStack/Storybook/Glide direction as a fresh framework shopping exercise.

**Latest owner clarification in this task:** Advocatio is part of one unified surface and the underlying languages must be brought together. This expands reconciliation beyond frontend visual consistency. See [LANGUAGE-CONVERGENCE.md](LANGUAGE-CONVERGENCE.md): every existing runtime/framework gets an explicit migration disposition. Earlier instructions to retain a donor's framework are historical constraints to reconcile, not permanent exemptions from the current direction. The user has not named one language for every processing engine; do not invent a wholesale Python/Java rewrite decision.

| Layer | Observed current state | Reconciled target/disposition |
|---|---|---|
| Advocatio frontend | Next `^16.3.1`, React `19.2.3`, AI SDK; no declared TanStack/Storybook/Glide in current manifest | Retain working feature/domain behavior as migration source; migrate to shared frontend after a route/auth/streaming parity slice. Old 'Next locked' prose is superseded as target direction. |
| Family Court simple surface | React `19.2.3`, Vite, TanStack Router/Query/Table/Virtual, Storybook, Tauri, Glide `^6.0.3`, Claude Agent SDK sidecar | Shared client conventions and common source reads; do not copy its full analysis/tools into phone reference mode or transplant its sidecar into all products. |
| Canonical Consignatio | React `18.3.1`, Vite, Tauri, Storybook, TanStack Store/Table, Glide `6.0.3` | Related migration participant, independent ownership. No source changes authorized here. |
| Glide experiment | Historical isolated `6.0.4-alpha24` + React `19.2.3` harness exists; prior report records browser editing/paste and builds passing | Owner authorized testing alpha. This does not prove product adoption. Native Tauri packaging was inconclusive with `kernel32.lib`; rerun preflight on intended toolchain. |
| Advocatio backend | Python/FastAPI/Pydantic/SQLAlchemy; current workspace path selects SQLite | Preserve legal domain API ownership. Original deployment plan intends PostgreSQL legal schemas. Atomicity and real version persistence require repair before sensitive imports; a DB engine option is not proof of actual PG use. |
| Evidence | Probata owns source custody, accepted factual assertions, spans, factual chronology and analysis | Legal consumes versioned packages/projections; shared frontend does not merge evidence authority or databases. |
| Timesketch | Maintained sibling fork found under Probata `modules/forks/timesketch`; earlier accepted ADR preserves upstream Vue/OpenSearch boundary | Real fork commitment remains, but its Vue presentation conflicts with latest convergence direction. Determine common-surface port/adaptation and maintained engine/upstream boundary explicitly. Neither an adapter-only downgrade nor an automatic separate Vue app satisfies reconciliation. |
| Graph/timeline renderers | Sibling workbench dependency declarations/lockfile entries; no current imports found in the bounded workbench or Advocatio search | Retain both declared development options, react-calendar-timeline and vis-timeline. Choose an engine per concrete interaction, allowing both where useful; no forced winner or replacement of Timesketch. |
| Office/document editing | Current Python document/PDF libraries; historical Markdown/TipTap-first plan with office surface explicitly held | LibreOffice requirement retained. Headless rendering and interactive office editing are distinct deliverables; exact editor/office bridge remains open. |
| Agent orchestration | Current routing config and provider adapters; historical firm prompt design still to be supplied | Roles separate from providers. No universal SDK decision recovered; preserve API/auth and model/data boundaries rather than forcing Claude sidecar everywhere. |

Manifest versions above describe local declarations, not current upstream latest, installed state or deployed services. Do not infer dependency compatibility from declaration alone.

## Migration proof before broad port

Inventory current Next routes, server-side API proxy/auth and streaming dependencies. Port one source-linked reference/detail workflow into the common client with equivalent authorization and source/version behavior. Build Storybook fixtures for loading, denied, stale, missing-source and partial-support states. Exercise Glide selection/edit/paste with browser interaction tests, not component snapshots alone. Package a minimal Tauri client on Windows only after the web slice passes. Keep the independent Python API; protect secrets formerly held in Next server routes during the port.

## Primary-source integration notes

LibreOffice documents headless operation, conversion filters/output directories and a configurable user-profile path; these establish a rendering/conversion interface, not a browser editor. Plan isolated job profiles and output paths and verify revision/format fidelity. [LibreOffice command-line documentation](https://help.libreoffice.org/latest/en-GB/text/shared/guide/start_parameters.html).

LibreOffice separately documents recording and displaying edits; therefore native tracked-change interoperability deserves its own fixtures rather than being inferred from successful PDF conversion. [LibreOffice tracked changes](https://help.libreoffice.org/latest/en-US/text/shared/guide/redlining.html).

Timesketch's upstream describes collaborative forensic timeline analysis. The exact local adaptation and accepted fork scope come from the local ADR and handoffs, not from upstream marketing or the remembered name 'Timescale'. [Timesketch upstream](https://github.com/google/timesketch).

## Remaining narrow decisions

1. Record the exact common package/version baseline and whether the proposed Router/Query SPA baseline is adopted; do not call this unsettled overall unification.
2. Specify interactive document editor/LibreOffice bridge and tracked-change model, with a fidelity spike.
3. Reconcile the older maintained Timesketch/Vue decision with the latest unified-surface/language direction, including the scope of a React presentation port and the preserved upstream engine.
4. Confirm production persistence and auth topology from live deployment evidence before migration; current static audit proves neither.

Docstore: the decision was recovered from actual session evidence, but no governed write/readback tool was available. Local recovery is durable; universal registration remains outstanding.
