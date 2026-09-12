---
priority: critical
authority: owner_direction_and_existing_accepted_contracts
status: accepted_for_initial_adoption
contract_version: 1.0.0
updated: 2026-09-12
---

# Propria shared surface design contract

<!-- Created by: Codex | Date: 2026-09-12 | Platform: Codex / win32 -->

This contract gives Propria's owner-facing applications one recognizable operating language without
turning them into one codebase, one runtime, or one data authority. It applies to the Propria home
portal, project progress board, Consignatio Intake, the Indicia Probata Workbench, and the
**advocatio Legal Workdesk**.

The interaction and authority rules below consolidate current accepted decisions and verified
implementation receipts. The **Carbon-Linen-Seal** palette responds to the owner's 2026-09-12
feedback that the current portal and Intake presentation is too blue, too bland, and lacks contrast.
The owner accepted this direction for initial adoption on 2026-09-12, with the explicit requirement
that color and CSS remain a centrally editable template. Acceptance does not freeze individual
values: a versioned token change can adjust the entire family without changing each product by hand.

## Contract precedence

This visual contract does not replace domain contracts. If wording conflicts, use this order:

1. Canonical custody, evidence, legal-work and application ADRs or decisions.
2. [CCC, Intake and Docstore boundaries](SYSTEM-BOUNDARIES.md).
3. [Shared result presentation and document revision contract](RESULT-PRESENTATION-CONTRACT.md).
4. This shared surface contract.
5. Product-local component and styling conventions.
6. Historical mockups and design donors.

The shared layer owns semantic names, visible state meanings, typography roles, accessibility floors,
and cross-surface context handoff. Each product retains its runtime, credentials, storage, release
cycle, domain components and direct fallback route.

## Surface map

| Surface | Primary job | Authority boundary | Required visual relationship |
|---|---|---|---|
| Propria home portal | One front door and current surface health | Navigation and health only | Shared shell tokens; never imply that a link or iframe merges applications |
| Project progress board | Show actual work, verification and blockers | Reads project receipts/status; does not invent completion | Real project-management board plus independent source-labelled graphs and widgets |
| Consignatio Intake | Find, group, compare, decide and move files | Stage-one filesystem organization; not evidence acceptance | Highest-density working surface; two independently navigable Explorer panes with adjacent preview/metadata/chat |
| Indicia Probata Workbench | Intake preview, governed review, evidence operations and status | PostgreSQL and governed receipts remain canonical | Shared shell and result semantics; stage-two review is visibly distinct from Intake organization |
| advocatio Legal Workdesk | Research, theory, drafting, review and release preparation | Consumes immutable, version-pinned `LegalSourcePackage`; cannot write evidence | Shared shell semantics with a quieter document-reading/editor treatment |
| Engineering consoles | Temporal, n8n, database and infrastructure administration | Their own bounded administrative authority | Status summary and deliberate deep link; do not style or embed them as ordinary product pages |

## General and Advanced Workbench surfaces

The owner-approved Probata structure has two related surfaces. These are experience tiers inside the
same bounded Workbench, not separate evidence stores, not light-versus-dark themes, and not a split
between Probata and the advocatio Legal Workdesk.

| Tier | Canonical Probata donor | Purpose | Default composition |
|---|---|---|---|
| **General** | **Evidence Operations Desk** in `modules/workbench/design-mockups/evidence-operations-desk/` | Daily intake, sorting, review, search, receipts, provenance and ordinary chronology | Graphite navigation chassis, warm-paper working canvas, dense primary table/work area, persistent right custody inspector |
| **Advanced** | **Modular Service Cockpit** in `modules/workbench/design-mockups/modular-service-cockpit/` | Deep forensic analysis and governed service tools | Persistent context spine, compact rail, configurable module grid, visible service boundaries and audit metadata |

Both tiers use the same semantic color meanings and typography roles. The Advanced surface may be
denser and more modular, but it does not introduce a competing brand palette. Theme (`light` or
`dark`) and experience tier (`general` or `advanced`) are independent axes.

General is the default entry point. Advanced is disclosed deliberately from Probata when its target
tools have passed deployment, authority and round-trip proof. Timesketch, deep temporal inspection,
graph/map/schema projections, workflow diagnostics and governed curation belong there. A surface
switch preserves matter, court case, knowledge partition, selection/revision where compatible, and
correlation context; it never upgrades authority or silently widens scope.

The portal, Intake and advocatio Legal Workdesk are General-first applications. They may deep-link into a qualified
Probata Advanced instrument with a scoped context ticket, but they do not duplicate the Modular
Service Cockpit or relabel themselves as the Advanced surface.

## Probata and legal alignment

Probata, the advocatio Legal Workdesk and the separate Family Court Console should look like related workspaces without
pretending to be one application. Align these parts now:

- App shell proportions: compact header/search, bounded left navigation, primary work canvas and an
  optional right inspector or assistant.
- The same Carbon-Linen-Seal semantic roles, focus treatment, typography roles, spacing, radii,
  hairline borders and restrained shadows.
- The context strip vocabulary for matter, court case, selection/source, revision/package, run,
  authority and freshness.
- Card, badge, button, input, unavailable notice, result row, provenance/source block and expandable
  detail anatomy.
- Loading, bounded unavailable/error and retry behavior. A disconnected service must not collapse the
  rest of the shell or become a fabricated empty state.

Keep these legal meanings distinct even when components are visually shared:

- attorney review required, publication blocked, provisional, verified primary, stale, superseded,
  conflicted and STOP AND VERIFY;
- authority level, source currency/checked date, direct-support and conflict posture;
- occurred-at versus known-at chronology, plus explicit court and master timeline lanes;
- child-name suppression, confidential treatment, filing/deadline disclaimers and release approval;
- advocatio Legal Workdesk draft/review/released states and immutable `LegalSourcePackage` version binding.

The separate **Family Court Console** is not the advocatio Legal Workdesk and must not be relabelled
as it. It can share
this visual contract and later integrate through an explicitly clamped read-only adapter. Mixed-mode
tools that can mutate a case store are not safe merely because they also provide a read operation.
The current route and tool proof boundary is recorded in
[design-contract/CALLABILITY.md](design-contract/CALLABILITY.md).

## Non-negotiable experience boundaries

### One front door, bounded applications

The accepted ADR-0061 model remains in force: Workbench can carry a versioned `OperatorContext` and
launch bounded applications, but shared browser state is never authorization. Each receiving
application exchanges an audience-bound, expiring, single-use ticket server-side, establishes its own
session, and revalidates scope. Direct product routes remain available when the shell is unavailable.

### Intake is the stage-one filesystem workstation

Intake's primary view is not a candidate-review queue and not two links to independent applications.
It is one working context with:

- Two independently navigable Explorer panes, each retaining its own path/tab identity and selection.
- Natural file and group selection. The active pane feeds the toolbar, preview and assistant context.
- Navigation clearing only the affected pane's selection; switching focus must not erase the other
  pane's group.
- Persistent preview and metadata with adjacent selection-aware AI chat. The current Intake-only
  preview-plus-chat vertical split and resizer are the implementation donor; do not rebuild them as a
  second docking system.
- Chat conversation and draft surviving preview/chat toggles and sidebar collapse.
- Metadata-only selection manifests that preserve the complete group without automatically reading
  file bytes. An active preview may read the selected file through its explicit preview path.
- The right-rail **AI Chat** action opening the assistant. The top-bar **New Chat** action creates a
  chat file and must not be relabelled as the assistant.
- Search showing source identity, exact path, snippet, score interpretation, coverage and explicit
  unavailability. Empty or partial coverage never means the file does not exist.

Ordinary browse, copy, move and rename operations do not wait for indexing, extraction,
classification, or evidence approval. Proposed destructive actions remain outside this contract and
must follow repository and product safety rules.

### Evidence review is stage two

Indicia Probata owns candidate review, custody-aware preview, accept/reject decisions, processing
status and receipts. A source, a machine proposal and a human decision are separate layers. Selection,
search relevance, successful parsing, duplicate content or a polished preview never imply acceptance.

### Legal work remains version-bound

The advocatio Legal Workdesk shows the matter/package/revision in view, distinguishes research from assertions and drafts,
and keeps draft, review, approved-for-release, stale and released states explicit. An upstream source
revision or revocation visibly marks dependent work as requiring revalidation; it never silently
updates or re-approves legal work.

## Shared context model

Use one visible context strip pattern, adapted to each surface. Show only fields that are meaningful,
but do not change their meanings:

| Field | Meaning | Examples |
|---|---|---|
| `surface` | Current bounded application | Intake, Evidence operations, Legal work |
| `scope` | Human-recognizable working scope | source root, matter, court case, package |
| `selection` | Current selected item/group | pane + tab/path identity for Intake; governed IDs downstream |
| `revision` | Version that the view or decision actually references | source version, projection generation, document revision, package version |
| `run` | Correlated asynchronous work | job/run/correlation ID and state |
| `authority` | Whether the view is source, proposal, pending decision or accepted record | never inferred from color alone |
| `freshness` | Last successful read or receipt time | stale and unknown are first-class states |

The context strip is compact and persistent. IDs and hashes can be copied or expanded, but the default
view uses plain labels and bounded excerpts.

## Semantic token contract

Machine-readable values are in [design-contract/tokens.json](design-contract/tokens.json), portable
CSS variables are in [design-contract/tokens.css](design-contract/tokens.css), and
[design-contract/verify.mjs](design-contract/verify.mjs) checks token parity, package/adaptor presence
and the contract's core contrast pairs. [The package README](design-contract/README.md) is the adoption
guide. Products map their local variables through the supplied adapters to the `--pr-*` semantic
names; they do not import another product's CSS or create a shared runtime dependency.

### Carbon-Linen-Seal palette

| Semantic role | Light | Dark | Use |
|---|---:|---:|---|
| Canvas | `#F3F0E8` | `#161A18` | Page/workspace background |
| Surface | `#FFFDF8` | `#202622` | Primary panel, card and editor |
| Surface muted | `#E7E1D5` | `#2A312C` | Recessed controls, lanes and secondary panels |
| Ink | `#171A1C` | `#F5F1E8` | Primary text |
| Ink muted | `#5D625F` | `#B8B1A5` | Secondary text; never critical facts alone |
| Border | `#C6BEB0` | `#49534D` | Ordinary separation |
| Border strong | `#81786A` | `#717C74` | Interactive boundaries and selected groups |
| Shell | `#171C19` | `#101412` | Navigation shell |
| Shell surface | `#232B27` | `#1B211D` | Active or nested shell area |
| Shell text | `#F4F0E8` | `#F5F1E8` | Shell copy and icons |
| Action / seal | `#9F303B` | `#F27479` | Primary action, active route, deliberate commitment |
| Action text | `#FFFFFF` | `#211011` | Text/icon on a solid action background |
| Action hover | `#7B222D` | `#FF9A9F` | Hover/pressed emphasis |
| Action soft | `#F3E1E5` | `#4A252B` | Selected or active background |
| Focus / brass | `#7D5200` | `#F0B45A` | Keyboard focus and current target |
| Positive / verified | `#247047` | `#6CC392` | Verified success only |
| Caution | `#9A5A12` | `#E6B55D` | Pending, incomplete, stale or attention |
| Destructive | `#B42318` | `#FF8377` | Failure or genuinely destructive action |
| Information | `#376F72` | `#82BDC0` | Neutral information and charts; blue/teal is not the brand action |

Rules:

- Primary actions use seal red, not blue. Blue/teal is reserved for information where it adds
  meaning.
- Focus is brass and always includes an outline, not a color-only change.
- Success green means verified completion, never merely submitted, transported or locally built.
- Caution covers pending, partial, stale and unknown only when a text/icon label states which one.
- Destructive red is not reused for ordinary navigation, decoration or a generic chart series.
- Light and dark schemes carry the same semantic meanings. Theme changes do not change authority.

### Typography

| Role | Contract | Current compatibility |
|---|---|---|
| Workflow UI | `Instrument Sans`, then `Segoe UI`, sans-serif | Intake and Probata already use Instrument Sans; portal migrates away from Calibri |
| Data | `IBM Plex Mono`, then `Cascadia Code`, monospace | Paths, IDs, hashes, timestamps, snippets and logs only |
| Document | `Source Serif 4`, then `Playfair Display`, Georgia, serif | advocatio Legal Workdesk may retain Playfair as its document-display alias until a deliberate migration |

Use sentence case. Avoid all-caps eyebrows and labels except established legal abbreviations or a
literal source value. A monospace face signals machine-verifiable data; it is not generic decoration.
Body text targets 45-78 characters per line. Dense tables may be wider but keep headings and help copy
readable.

### Geometry and density

- Spacing scale: `4, 8, 12, 16, 24, 32, 48px`.
- Radii: `4px` controls, `6px` panels, `8px` dialogs. Pills are reserved for compact status badges or
  segmented choices.
- Target size: 36px minimum in dense desktop tools; 44px wherever the layout permits and for touch.
- Focus outline: 3px, 2px offset, never clipped by overflow.
- Divider hit area: 8px minimum even when the visible rule is 1-2px. Every resizer supports keyboard
  arrows and announces its orientation/value.
- Use one quiet panel shadow and one dialog/overlay shadow. Borders carry most hierarchy.
- Motion answers an action or state change. Respect reduced motion; do not stagger page-load cards.

## Shared component behavior

### Navigation

Navigation says what the owner can do, not how a backend is named. Active state combines position,
contrast and text/icon treatment. Do not expose disconnected destinations, empty stubs, or engineering
consoles as if they were finished product routes.

### Results and tables

Use the shared result-presentation contract: typed pageable rows, bounded excerpts, explicit omitted
counts, full-detail retrieval, stable IDs, provenance, score meaning, lifecycle, freshness, warnings,
errors and missing-data status. Do not collapse corroborating occurrences or call an excerpt a summary.

### Actions and receipts

Action names remain stable through the flow: **Move** produces **Moved** or **Move failed**; **Approve
revision** produces a receipt naming that revision. Submitted, queued, running, persisted and verified
are different states. Optimistic UI may show pending but cannot style itself as accepted.

### Empty, partial, stale and unavailable

Every non-happy state gives the next useful action and names the boundary:

- **Empty**: query succeeded and returned zero within known coverage.
- **Partial**: some sources, rows or fields are omitted; say how many when known.
- **Stale**: last good value remains visible with its timestamp and refresh/revalidation path.
- **Unavailable**: source or capability could not be checked; do not replace it with zero.
- **Unknown**: the system never established the fact.

### Chat

Chat context is visible, editable and bounded. Show selected-item count and up to five readable names,
then an explicit remainder count; retain the full manifest behind it. Source text and filenames are
untrusted data, not instructions. The assistant can propose grouping, movement, investigation or legal
work, but the relevant human gate and authoritative backend decide what becomes accepted.

## Portal and progress-board contract

The portal is a launch and health surface, not a substitute application. Each destination shows
purpose, current availability, last verified time and a direct route. An iframe preview is labelled as
a preview and must not be presented as integrated authority.

The progress board uses real project-management structure:

- Separate Backlog, Ready, In progress, Blocked, Verification and Done meanings. Do not compress all
  work into three vague columns.
- Work items show owner/lane, source, last update, dependency/blocker and verification boundary.
- Done requires the applicable receipt. Local build, deployed, reachable, authenticated and live-flow
  verified are separate fields.
- Independent widgets or graphs cover delivery flow, verification coverage, deployment health and
  blockers. Each widget declares its source, scope, time range, last refresh and unknown/partial data.
- Never publish one blended percent that combines unrelated repositories, story points, tests and
  deployments. If a percentage is used, define its denominator beside it.
- Graph series use semantic colors only where the meaning is consistent; otherwise use a neutral
  categorical sequence derived from ink, seal, brass, verify and information tokens.

## Stack adapters observed 2026-09-12

These are current implementation facts, not an upgrade mandate and not shared runtime dependencies.

| Surface | Current stack | Adapter rule |
|---|---|---|
| Intake metadata/review app | React `18.3.1`, Vite `8.2.2`, TypeScript `7.0.2`, Tauri `2.11.x`, Storybook `10.5.10`, Vitest `4.1.11`, Glide Data Grid `6.0.3` | Map existing `--workspace`, `--surface`, `--ink`, `--indigo` and status variables to `--pr-*`; preserve `.theme-dark` compatibility |
| Intake native Explorer | Existing Xplorer fork on branch `feat/acp-copilot` | Reuse existing split, selection, right-sidebar, preview and chat components; theme through the fork's local adapter, not a replacement shell |
| Probata Workbench | React `19.2.3`, Vite `8.2.2`, TypeScript `5.x`, Tailwind `4`, shadcn `3.8.4`, Radix UI `1.4.3`, Storybook `10.5.10` | Expose `--pr-*` through Tailwind `@theme inline`; preserve `.dark` theme behavior and TanStack Router |
| advocatio Legal Workdesk | Next `16.3.1`, React `19.2.3`, AI SDK `7.x`, cmdk `1.1.1`, Lucide `0.544.x`, local Inter/IBM Plex Mono/Playfair fonts | Alias current legal tokens to `--pr-*`; preserve document typography and confidential-state semantics |
| Progress board | Node `22+`, server-rendered static assets | Render semantic tokens directly; widgets remain independent and source-labelled |

Do not migrate Intake to React 19, Probata back to Next.js, or advocatio Legal Workdesk to Vite merely to share the
design contract. Shared meaning is the integration seam.

## Accessibility floor

- Text and UI contrast target WCAG 2.2 AA. Validate exact rendered foreground/background pairs; token
  names alone are not proof.
- Every interactive element is keyboard reachable with visible focus. Pane activation, selection,
  resizers, preview/chat toggles and command palettes receive explicit keyboard tests.
- Status never relies on color alone. Include text and, where useful, a consistent icon.
- Reading order follows the visual order at reflow. Narrow layouts may stack surfaces, but they preserve
  pane identity, selection and active context.
- Respect `prefers-reduced-motion`, 200% zoom and Windows high-contrast/forced-colors modes.
- Use real semantic controls and headings. Tooltips do not carry instructions required to complete a
  task.

## Adoption and verification gates

1. **Owner visual direction:** SATISFIED FOR INITIAL ADOPTION — the owner accepted the
   Carbon-Linen-Seal sample on 2026-09-12 and clarified that the approved Probata General/Advanced
   mockups remain the structural donors. Cross-surface rendered comparison is still required before
   declaring visual parity complete.
2. **Token adoption:** map product-local variables to `--pr-*` without importing another product's
   runtime or replacing its components.
3. **Contract tests:** verify semantic token presence, theme parity, status labels and context-strip
   fields in each application.
4. **Interaction tests:** preserve Intake's two-pane selection and simultaneous preview/chat behavior;
   preserve Workbench review gates and advocatio Legal Workdesk revision/staleness behavior.
5. **Accessibility:** run automated checks plus keyboard, focus, zoom, reflow and contrast checks.
6. **Live proof:** verify the real direct route and shell route, current revision, auth/context scope,
   source/status freshness and receipts. A local build or screenshot is not deployment proof.

### Current evidence boundary

- Source inspection confirms the existing Probata and Intake metadata apps share a graphite/warm-paper
  token ancestry, while the current portal is heavily blue and advocatio Legal Workdesk uses a separate deep-blue
  legal shell.
- The owner rejected the current bland/blue/low-contrast direction; this contract removes blue as the
  primary brand/action color.
- The owner accepted the Carbon-Linen-Seal direction for now and required a centrally adjustable CSS
  template. The preserved Probata Evidence Operations Desk and Modular Service Cockpit—not the later
  generic sample—remain the approved structural source for General and Advanced surfaces.
- Intake's selection/chat receipt records component and source tests, but native click-through and an
  actual selection-to-live-chat exchange remain unverified.
- This Codex session could start the local Intake and Probata Vite servers, but no browser surface was
  available to capture or inspect screenshots. The owner's direction acceptance is not a rendered
  cross-product verification claim.
