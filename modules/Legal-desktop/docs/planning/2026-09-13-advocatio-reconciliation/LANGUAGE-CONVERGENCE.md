# One surface: language and framework convergence

The owner explicitly wants Advocatio to become part of **one unified surface**, including bringing underlying languages together. This is a current requirement. Shared colors, iframe links and a directory of unrelated applications are insufficient to claim completion.

The owner subsequently clarified that **a complete port need not happen immediately**. Shared callable services and temporary bridges are acceptable stages when they have an explicit convergence target. First recover the existing remote Family Court Toolkit database/MCP work; it may already provide the needed bridge and must not be duplicated by a new import plan.

The recovered settled client choices are TanStack, Storybook, Glide and Tauri for desktop. The recorded migration recommendation uses TypeScript/React/Vite with TanStack Router/Query. Exact package adoption is tracked in [STACK.md](STACK.md).

## Proposed convergence plan

| Area | Current/historical languages and frameworks | Proposed disposition for review | Proof needed |
|---|---|---|---|
| Owned application presentation | Advocatio Next/React/TypeScript; Family Court and Intake Vite/React/TypeScript with different versions/state usage | One React/TypeScript client architecture, route conventions, shared components/tokens, typed API clients and Storybook states. Port Advocatio's useful screens and Next server dependencies deliberately. | Same component/contracts in integrated surface; navigation, auth, focus, source links and model streaming parity. |
| Desktop container | Tauri/Rust native wrapper | Shared desktop shell/native capability boundary around common web client; product differences become modules and permissions. | Windows package proof plus web/phone operation; native-only actions unavailable safely on phone. |
| Simple reference surface | Family Court client and sidecar | A simpler projection/module of common resources and records. Reuse shared client conventions, avoid a separate authored legal corpus. | Same resource/event versions across desktop, phone and agent reads; private strategy correctly scoped. |
| Timesketch fork | Upstream Python application and Vue frontend; preserved OpenSearch boundary in earlier accepted ADR | Recover fork features first. Specify which interaction components become React/TypeScript in unified surface and which maintained engine/API code remains. Compare actual port cost/upstream compatibility; record any temporary bridge as transitional. | Governed chronology/curation features reach common surface with source IDs and acceptance boundary intact; no claim that an external link alone completes integration. |
| Timeline/graph libraries | React calendar timeline, vis timeline/data, additional graph/report/geospatial candidates | Shared TypeScript adapters and source/event types. Different visualization engines may serve different views without different product application frameworks. | Multiple renderers select the same event/claim and open the same source inspector; no duplicated authoritative timeline state. |
| Advocatio legal domain | Python/FastAPI/Pydantic/SQLAlchemy | Retain during staged migration while inventorying overlap with sidecars and common platform services. Generate/check shared typed contracts. Any language consolidation of service logic needs an explicit module plan and behavior-preserving tests. | No duplicated legal rule implementations across TypeScript/Python; contract parity and immutable version semantics. |
| Existing Family Court sidecar | Fastify/JavaScript and Claude Agent SDK | Inventory responsibilities; move shared product-neutral capabilities behind common contracts instead of copying this sidecar into every module. Decide which remaining local-only operations justify a native bridge. | Single clear owner per method/run/resource operation; no universal provider assumption. |
| Office and processing engines | LibreOffice native/UNO; PDF/Python tooling; Java FreeEed donor | Consume bounded engine capabilities through shared job/result contracts where justified. Port reusable UX/process ideas into common client. Do not adopt every donor runtime because a package was supplied. | Real tool output/fidelity and provenance; explicit donor-by-donor adopt/adapt/reject decision. |
| Frozen reports/admin tools | Evidence.dev and historical NeoDash/Surrealist candidates | Recover handoff commitments; integrate results/navigation/identity into common surface or adapt needed components. Admin-only applications do not define normal user workflows. | Report/graph provenance and scope; explicit temporary external-tool status when no port exists. |

This is an implementation recommendation, not a claim that the owner approved retaining every backend language. Conversely, 'bring languages together' is not evidence that all mature native/Python/Java engine code should be rewritten immediately. The concrete deliverable is a module-level disposition with file/module ownership, cost, risk, target language and acceptance test.

## Required convergence artifacts before porting

1. Route/component/server-function map of each owned client and selected donor.
2. Handoff-backed feature list, including features already started in forks.
3. Decision for each module: migrate to common TypeScript client, move shared service logic, retain bounded engine, temporary bridge, defer, or reject with reason.
4. Dependency and provider inventory; one common client baseline, no scattered copies of methods, schemas or legal rules.
5. Migration slices with parity checks and reversible cutover. Preserve original histories and dirty work; no permanent deletion.

An external project's existence, library declaration or local fork checkout is not integration proof. Runtime language consistency, source identity and user workflow continuity must each be demonstrated.
