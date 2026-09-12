# Surface Design Adoption Register

> _Byline: Codex · GPT-5 · 2026-09-12_

**Status:** OPEN — the shared contract/package is verified and imported; product adoption is not assigned.

**Contract:** [`../SURFACE-DESIGN-CONTRACT.md`](../SURFACE-DESIGN-CONTRACT.md)

**Canonical package:** [`../resources/design/`](../resources/design/) at pinned source commit `c6da141`

This register prevents the verified shared package from being mistaken for completed product adoption or being lost when the design-contract task closes. A row changes state only with the evidence named in its acceptance gate. Chat status, an active process, a generated sample, or a local build is not completion.

## Dependency order

1. Keep the imported package and contract durable on the Propria remote.
2. Assign one implementation owner per independently owned product lane.
3. Vendor a pinned package copy into each product repository; never add a relative runtime dependency on the Propria router directory.
4. Implement and verify product-local adoption. Probata, Consignatio, advocatio Legal Workdesk, and Family Court Console may proceed in parallel after assignment.
5. Perform one cross-surface rendered reconciliation after all intended product lanes provide verified candidates.
6. Deploy and live-prove each product independently. Do not turn a cross-surface design review into a bundled release.

## Work register

| ID | Product lane | Exact surface boundary | Assignee | State | Earliest start | Acceptance gate |
|---|---|---|---|---|---|---|
| SDA-00 | Propria shared contract | `SURFACE-DESIGN-CONTRACT.md` and `resources/design/` | Propria migration owner | VERIFIED | complete | Imported manifest state; canonical target exists; compatibility junction resolves to it; 15-file package is tracked, including authoritative `tokens.json`; `npm.cmd run verify`; remote commit contains the package and contract. |
| SDA-01 | Probata | General/primary **Evidence Operations Desk**; Advanced/gated **Modular Service Cockpit** | `Rename important component` (`01a0960d-eb7a-7d91-a28b-338f88e2d5ae`) | IN PROGRESS — FUNCTIONAL RECOVERY PRECEDES VISUAL ADOPTION | 2026-09-12 | Prove one executable vertical slice before styling around failures: explicit TEST DATA/REAL MATTERS selector and selected matter; source filter/selection; useful preview and reliable Back; custody; intended parser; normalized records; Go chunk Activity; PostgreSQL persistence; actual record/chunk/database read-back; durable receipt. Scope switches must clear or refresh every matter-bound selection, preview, record, chunk, receipt and action. Then adopt the pinned package without replacing approved structural mockups; keep General/Advanced independent of theme and Advanced gated; run lint/build/smoke/Storybook/accessibility and authenticated live proof. |
| SDA-02 | Consignatio Intake | Xplorer sorts, organizes, compares and deduplicates vault material. Its adjacent preview/chat may use only ephemeral, in-app selected file/group metadata. Evidence-candidate review remains a separate stage-two surface. | `Add Intake and Probata preview` (`01a09620-155b-7b01-8ab3-da15cfd419a9`) | ASSIGNED — WAITING FOR CURRENT HEARTBEAT TO EXIT | 2026-09-12 | Separate pinned copies and receipts for the stage-one Xplorer runtime and stage-two metadata/review app; product-local adapters; existing Xplorer split/selection/preview/chat preserved; tests prove selected file/group metadata stays ephemeral and Xplorer-local; no Probata evidence, Docstore, legal, acceptance or cross-system context/authority; focus, resizers, reflow and degraded states; native click-through and live local selection-to-chat proof; independent deployment receipts. |
| SDA-03 | advocatio | **advocatio Legal Workdesk** | `Consolidate design contracts` (`01a09663-e9bb-72c2-bfce-c0714cba8d66`) | IN PROGRESS | 2026-09-12 | Pinned vendored package; legal typography and confidential states preserved; source currency, STOP/review, privacy, two-clock chronology, versioned `LegalSourcePackage`, staleness/revocation, draft/review/release and filing-readiness semantics tested; no evidence mutation; authenticated live-flow proof after independent deployment. |
| SDA-04 | Family Court Console | Separate from the advocatio Legal Workdesk | `Fix family court dark mode` (`01a09653-64ee-7c33-b6e7-bd3236b618af`) | IN PROGRESS | 2026-09-12 | Pinned vendored package; product identity remains separate; theme/token mapping, keyboard/focus/contrast/reflow checks, bounded unavailable/retry states, and independent deployment/live-route proof. |
| SDA-05 | Cross-surface reconciliation | Shared shell geometry, semantic colors, focus, context strip, cards/buttons/badges, result/provenance anatomy and bounded failure states | `Consolidate design contracts` (`01a09663-e9bb-72c2-bfce-c0714cba8d66`) | BLOCKED | after selected SDA-01..04 implementations pass local gates | Rendered comparison of actual product candidates in light/dark themes and applicable experience tiers; product names and authority boundaries correct; accessibility checks; unresolved differences recorded rather than hidden. |
| SDA-06 | Per-product release | Each adopted product independently | Each assigned product owner | BLOCKED | after its implementation and SDA-05 review | Exact revision deployed; direct route and shell route reachable; authentication and context scope verified; source/status freshness visible; governed receipts/read-back verified; rollback boundary recorded. |

## Assignment rule

An assignee is a named task or owner with explicit repository scope. Change **UNASSIGNED** only when that task exists and has accepted the lane. Process existence, an idle historical task, or a suggestion that a task “could” do it is not assignment.

Every assignment update records the task title/ID, repository, accepted file boundary, start date, and latest evidence. If work stops or is superseded, preserve the prior entry and add the successor; do not silently erase history.

## Shared implementation requirements

- `tokens.json` is the only editable token source. Do not hand-edit generated `tokens.css`.
- Run `npm.cmd run build:tokens` after token changes and `npm.cmd run verify` before accepting package parity.
- Centrally adjustable tokens do not authorize a shared runtime or framework migration.
- Shared design language does not merge product authority, data ownership, authentication, deployments, or release semantics.
- Contract/package verification is not adoption. Adoption is not browser parity. Browser parity is not deployment. Deployment is not authenticated live-workflow proof.
- Preserve unrelated dirty work. Stage and commit only the implementing lane's explicit allowlist.
- Never permanently delete files. Quarantine removals under the owning repository's `to_be_deleted/` directory for owner-only deletion.

## Active Probata functional blocker

SDA-01 cannot honestly begin as a paint-only exercise. Live verification on 2026-09-12 found an HTTP-healthy shell with an empty tool catalog, no monitored-actions capability route, failed recent ingest runs, and no executable Atomic Tools path. Active contracts disagree about allowed ingest lanes, while the Proffer path still produces non-empty Python chunks for a store that intentionally rejects them because chunking moved to the Go Temporal Activity.

The required Matter Selector is a safety boundary, not a preference. It must explicitly distinguish **TEST DATA** from **REAL MATTERS**, keep the active mode and selected matter unmistakable, prevent test data from being mistaken for or written into a real matter, and invalidate all matter-bound UI and action state when scope changes. SDA-01 remains in functional recovery until the full selected-matter-to-persisted-readback slice passes process-level tests and live acceptance proof.

## Cross-product authority boundaries

- Probata/PostgreSQL remains canonical for evidence, custody, governed review decisions, projection generations and receipts.
- The advocatio Legal Workdesk consumes issuer-verified, manifest-hashed, version-pinned `LegalSourcePackage` references and cannot write evidence.
- Family Court Console remains separate from the advocatio Legal Workdesk.
- Consignatio Intake prepares and organizes source material; its shared technology does not make it the evidence or legal-work authority. Xplorer selection state is ephemeral, in-app file/group metadata only and never carries Probata evidence context, Docstore context, legal context, evidence acceptance or cross-system authority.
- Browser identifiers are navigation hints, never authorization. Every eventual cross-surface launch requires audience-bound, expiring, single-use exchange and authoritative server-side scope validation.

## Current next action

SDA-01, SDA-02, SDA-03 and SDA-04 were assigned to explicit repository-scoped tasks on 2026-09-12. The tasks have each received the shared peer map and may coordinate directly without editing another lane's files. The next central action is to collect their local verification receipts, move each row independently to verification, and begin SDA-05 only after actual rendered candidates exist.
