<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# End-to-End Application Review

The repository ecosystem has a strong conceptual architecture and unusually explicit safety boundaries, but it **cannot yet credibly deliver the intended end-to-end outcome**. The central blocker is the absence of one production-proven chain from source selection through immutable processing, human review, exact approval, destination commit, read-back, legal-work-product use, and recoverable failure handling.

> **Evidence boundary:** This review directly examined project-workspace materials and current artifacts from [Legal-Workspace](https://github.com/Cursedpotential/Legal-Workspace), [Consignatio](https://github.com/Cursedpotential/Consignatio), [probata](https://github.com/Cursedpotential/probata), [sbv-forensic](https://github.com/Cursedpotential/sbv-forensic), and [xplorer-copilot-buildkit](https://github.com/Cursedpotential/xplorer-copilot-buildkit). The [Intake-desktop](https://github.com/Cursedpotential/Intake-desktop) implementation was assessed primarily through its authoritative handoff and repository map. The source of [propria](https://github.com/Cursedpotential/propria) and [traceiq-rebuild](https://github.com/Cursedpotential/traceiq-rebuild), complete GitHub issue/PR histories, and live deployed systems were not exhaustively inspected; findings involving those areas are marked **Unable to Verify** rather than treated as implemented.

## Executive assessment

### Intended outcome

The available materials describe an ecosystem rather than a single monolithic application:

- **Intake/Consignatio** is the local file workstation: browsing, organizing, searching, previewing, grouping, and AI-assisted interaction with a large collection before evidence decisions begin. Its authoritative handoff explicitly says it is not the evidence-selection system and should not gate ordinary file operations on evidence approval.
- **Proffer within Probata** prepares retained source resources into inspectable, immutable processing attempts and proposals. It is intended to expose records, chunks, entities, relationships, metadata, files, warnings, lineage, configuration, and execution receipts before an exact human approval.
- **Probata** is the evidence, provenance, custody, context, and investigation authority. Search projections and graph stores are downstream or reconstructible views rather than the canonical evidence authority.[^1_1]
- **Advocatio/Legal Workspace** is the legal workdesk. It begins after evidence has been accepted, consumes version-pinned `LegalSourcePackage` objects, supports research/drafting/review, and emits investigation requests when proof is missing.
- **SBV Forensic** is a specialized SMS/MMS/call-backup ingestion and review surface with per-user SQLite data, media handling, full-text search, authentication, and large XML import support.
- **Docstore/CocoIndex and Smart Explore** are intentionally separate documentation and code-index planes. They may be queried together by an agent, but retain separate freshness, management, and receipt boundaries.
- **The custody-guide and legal-reference materials** are supporting legal-information products, not authoritative runtime evidence stores. The last reviewed project-folder handoff still classified their public release as blocked pending correction of legal, integrity, currency, and packaging defects.[^1_2]


### Delivery judgment

**Current realistic status: partial vertical slices with substantial integration groundwork, not an end-to-end complete product.**

The module set is capable of delivering the intended outcome in principle, but only if the existing boundaries are retained and the missing contracts are completed. The most serious gaps are not missing technologies; they are missing **authoritative transitions** between already-selected technologies:

- No valid canonical proposal artifact has been proved for the retained real workbook package.
- No generic secured DuckDB proposal adapter was implemented in the latest Proffer gap ledger.
- The operator could not yet create a complete operation with explicit source, matter, mode, path, parser, repair, extraction, entity, relationship, and chunk choices.
- Immutable rerun, complete attempt comparison, exact-stage retry, checkpoint recovery, and authenticated cancellation remain incomplete.
- PostgreSQL, Weaviate, and Neo4j proposal commits with destination-drift checks, idempotency, receipts, and read-back were not proved for the proposal model.
- Intake’s native desktop existed and backend search proof passed, but native click-through, selection-to-chat behavior, real-corpus indexing, and multimodal processing remained unverified.
- Advocatio’s current documentation contains competing persistence descriptions—JSON/JSONL durability, PostgreSQL as the locked data stack, and a prior audit finding that SQLite/SQLAlchemy was the actual runtime authority.


### Major strengths

- **Clear evidence/legal separation:** Advocatio does not ingest or establish evidence and is required to cite accepted assertions through source-package references and exact spans.
- **Fail-closed interaction design:** The newer Probata operator surface exposes unavailable information instead of fabricating package, custody, Temporal, n8n, chunk, or graph state.
- **Immutable-attempt direction:** Proffer’s intended approval binds exact proposal content, external artifacts, and a destination plan rather than approving a mutable workflow state.
- **TEST/REAL isolation:** Mode is propagated through URLs, API checks, component identity, and handle validation rather than represented only by UI styling.
- **Explicit source precedence:** Recent handoffs identify which historical instructions have been superseded and warn against treating old plans or donor materials as active application instructions.
- **Repository preservation discipline:** Advocatio relocation preserved thousands of files and Git metadata without resetting, cleaning, or deleting the dirty working state.
- **Focused verification receipts:** Intake and Probata contain scoped test/build receipts and clearly distinguish source proof from deployment or native visual acceptance.


### Highest-risk gaps

- **Control-plane fragmentation:** Source identity, mode, matter, operation, attempt, package, chunk generation, approval, and destination coordinates are not consistently carried across all applications.
- **Authority ambiguity:** Several documents identify different canonical stores or checkouts.
- **No complete acceptance proof:** No receipt demonstrates the entire user outcome across Intake, Proffer, Probata, Advocatio, search stores, and legal presentation.
- **Incomplete failure recovery:** A user may see the failure but still lack an authenticated retry, cancel, rerun, or resume command.
- **Security boundary incompleteness:** No reviewed artifact establishes one shared identity envelope, least-privilege service policy, or cross-application audit identity.
- **Lifecycle incompleteness:** Retention, legal hold, reprocessing, supersession, deletion, cache invalidation, projection rebuilding, and destructive-operation approval are not expressed as one ecosystem contract.


## Intent baseline

### Product purpose

The system’s intended job is to turn disorganized files and communications into controlled, inspectable knowledge and evidence products without losing originals, provenance, user control, or the distinction between proposed, approved, evidentiary, and legal-work-product state.

The architecture divides this into four major stages:

1. **Organize:** Intake provides a file-workstation experience over local and mounted sources without requiring evidence classification.
2. **Prepare:** Proffer preserves source/package identity, runs explicit processing stages, creates immutable attempts, and exposes proposal contents before commit.
3. **Govern:** Probata controls context/evidence state, provenance, custody, promotion, retrieval projections, and investigation workflows.[^1_1]
4. **Use legally:** Advocatio consumes approved source packages for issues, factor analysis, drafting, citations, owner review, release candidates, and missing-proof requests.

### Target users

- The current system is primarily a **single-owner, single-matter workstation**, not a general multi-client legal SaaS product.
- The owner is expected to make consequential decisions: source selection, repair choices, processing overrides, approval, release, evidence promotion, and filing-readiness judgments.
- Agents are assistants, not authorities. They cannot approve, file, serve, email, transmit, or establish court-safe facts.
- Some components, notably SBV, expose multi-user authentication, but no reviewed artifact establishes that SBV’s user model is the ecosystem-wide tenancy model.


### Core workflows

- Browse and organize messy files.
- Select real source resources under an explicit matter and TEST/REAL mode.
- Preserve originals, members, attachments, metadata, and hashes.
- Process sources through visible and overridable stages.
- Review records, chunks, entities, relationships, warnings, lineage, configuration, and receipts.
- Freeze and approve one immutable attempt and destination plan.
- Commit selected approved projections to appropriate stores and verify read-back.
- Promote selected context material to evidence only through a separate governed action.
- Consume approved facts and exact spans in legal research, strategy, drafting, review, and release-candidate preparation.
- Return missing-proof requests to Probata rather than invent facts or dates.


### Non-goals

- Intake is not an evidence-admission or custody system.
- Proffer’s current context-preparation phase is not evidence promotion.
- Advocatio is not permitted to establish evidence, approve filings, serve documents, or transmit them.
- Release candidates are deterministic manifests, not completed court filings.
- SurrealDB is not an automatic proposal-commit destination; it is a later manual whole-case projection in the current design.
- Dragonfly cannot be an authoritative state store; it is intended only for reconstructible catalog, progress, event, layout, or coordination state.


### Ambiguities

- Whether “the application” means the two major applications identified by current Probata materials, the entire Propria ecosystem, or every listed repository.
- Whether Intake’s file-operation state, Proffer’s package state, Probata’s evidence state, and Advocatio’s legal state share one global resource identity.
- Whether Advocatio is currently JSON/JSONL-backed, SQLite-backed, PostgreSQL-backed, or operating in a transitional combination.
- Whether SBV is a source viewer only or a governed producer of source packages acceptable to Proffer.
- Whether TraceIQ is an authoritative analysis component, an optional derived view, or a separate experiment.
- Whether current legal-reference artifacts have resolved the previously documented publication blockers.


## Requirements traceability

| Goal or deliverable | Responsible modules | Status | Evidence and gap | Recommended ownership |
| :-- | :-- | --: | :-- | :-- |
| Local/mounted file browsing, organization, preview, selection, and AI assistance | Intake Desktop, Consignatio | Partially Covered | Desktop and scoped selection/search/chat behavior are documented, but native click-through and real-corpus proof remained unverified. | Intake owns file interaction; do not move this into Proffer. |
| Source preservation and package identity | Intake, Proffer, Probata | Partially Covered | The package model is extensively specified, but current operator projections historically lacked complete package identity and authoritative handoff proof. | Probata defines the package contract; Intake and SBV implement producers. |
| Explicit TEST/REAL and matter selection | Workbench, Proffer | Partially Covered | Mode isolation is implemented in the reviewed surface, but complete operation creation and durable mode filtering remain incomplete. | Workbench BFF plus Proffer command API. |
| Immutable processing attempts | Proffer | Partially Covered | Freeze/digest contracts exist, but no valid canonical v3 proposal was proved and rerun creation remained incomplete. | Proffer engine and durable orchestrator. |
| Precommit human review | Workbench Review | Partially Covered | Multi-tab operator view and truthfully unavailable states exist; canonical proposal loading and complete entity/relationship views remain incomplete. | Review UI consumes only browser-safe Proffer projections. |
| Entity and relationship refinement | Proffer, Review | Not Covered | Mention resolution, aliasing, link/unlink, merge/split, temporal qualification, and cross-attempt comparison were identified as incomplete. | Proffer domain and Review workspace. |
| Exact approval over immutable content | Proffer, Review | Partially Covered | Digest and selected-destination concepts exist; complete exact approval and downstream commits are not live-proved. | Proffer approval service. |
| Idempotent destination commit and read-back | Probata projection adapters | Not Covered | No proved independent PostgreSQL, Weaviate, and Neo4j commit/read-back flow for the proposal model. | Destination adapters behind Proffer commit orchestration. |
| Context retrieval | Weaviate, Docstore, Workbench | Partially Covered | Intake synthetic keyword/hybrid proof exists; governed proposal publication and integrated production retrieval remain incomplete. | Each index owns its projection and freshness receipt. |
| Code and documentation discovery | Smart Explore, CocoIndex/Docstore | Covered by design; integration partial | Separate indexing boundaries are explicit; combined agent querying must preserve separate freshness and source identity. | Keep separate; add a federated result envelope, not a merged store. |
| Evidence/custody authority | Probata | Partially Covered | Strong invariants exist, but reviewed Proffer materials deliberately stop at context preparation and do not prove later promotion end to end.[^1_1] | Probata evidence domain only. |
| Legal drafting and citation gates | Advocatio | Partially Covered | The first vertical slices, citation boundary, review gate, and release manifest are documented; persistence and production authority remain ambiguous. | Advocatio owns legal work product, never evidence truth. |
| Missing-proof feedback | Advocatio to Probata | Partially Covered | The `EvidenceInvestigationRequest` boundary is documented, but full delivery, authentication, deduplication, and resolution closure were not verified. | Shared versioned API/event contract. |
| SMS/MMS/call import and review | SBV Forensic | Covered standalone | XML import, media, search, authentication, and per-user stores are specified; governed export into Proffer is not established. | SBV owns source-specific decoding; Probata owns admissible package import. |
| AI model routing and tool boundaries | Intake, Advocatio, Probata | Partially Covered | Intake has bounded tool/model behavior; Advocatio has routing configuration and defaults live invocation off, but no cross-application evaluation or cost policy was found. | Per-app gateways under one shared AI governance contract. |
| Failure recovery | Proffer, Temporal, Workbench | Partially Covered | Failures are increasingly visible, but exact-stage retry, cancel, checkpoint resume, and immutable replacement-attempt commands remain missing. | Temporal-backed authenticated command layer. |
| Authentication and authorization | Each application | Unable to Verify | SBV has a local user/session model; other reviewed artifacts describe service boundaries but not one complete cross-app principal and authorization model. | Shared identity envelope with local policy enforcement. |
| Retention, deletion, supersession, and reprocessing | All modules | Not Covered coherently | Individual safeguards exist, but no reviewed ecosystem lifecycle state machine covers originals, proposals, projections, legal holds, caches, and releases. | Probata defines lifecycle semantics; every adapter implements them. |
| Deployment, rollback, backup, and disaster recovery | All services | Partially Covered | Local builds and scoped proofs exist, but several documents explicitly deny deployment proof. | Platform operations with service-specific runbooks and restore tests. |

## Module review

| Module | Purpose and ownership | Inputs and outputs | Current contribution | Main gaps and risks |
| :-- | :-- | :-- | :-- | :-- |
| **Consignatio umbrella** | Repository map, Intake planning, backend, repair tools, and legacy Case Bible materials | Plans, source, configuration, receipts | Establishes repository boundaries and preserves separate desktop/build-kit histories. | Parent clone does not include the desktop child; a fresh operator can obtain an incomplete application unless bootstrap validation checks every required repository. |
| **Intake Desktop** | Local/mounted file browsing, dual-pane interaction, selection, preview, chat, and indexed search | Filesystem paths, metadata, user selections, search requests | Native shell reportedly runs; pane selection, chat retention, search invocation, and remote model integration have focused tests. | No visual/native acceptance receipt, no real-corpus or multimodal proof, and incomplete source-package handoff. |
| **Intake backend/index** | CocoIndex-to-Weaviate processing and filesystem search API | Source roots, bounded extracted content, embeddings, search queries | Synthetic keyword/hybrid retrieval, deterministic unchanged-file behavior, scoped identity, and retirement behavior were proved. | Search authentication is not established for external exposure; real source scope, parser hardening, graph integration, and large-corpus behavior remain pending. |
| **Xplorer build kit** | Planning/scaffold for ACP-enabled Xplorer fork | Build guidance and stubs | Supplies phase plans, ACP reference, tool catalog, test strategy, and destructive-operation safeguards. | It is not runtime proof. Stubs or design documents must not be counted as implemented desktop behavior. |
| **Repair toolkit** | Bounded repair and derived-output operations | Original references, defect reports, approved repair parameters | Current Probata review materials recognize a guarded repair lane and allowlisted derived writers. | Repair reports, applicable tool identity, exact bounded inputs, derived-artifact hashes, validation, and authenticated decision receipts were incomplete in the browser contract. |
| **Proffer engine** | Source retention, processing orchestration, immutable attempts, proposal assembly, and approval | Source package and processing configuration | Strong proposal schema/digest design and explicit stage responsibilities. | No valid canonical proposal, generic adapter, complete operation creator, immutable rerun command, or destination commit proof. |
| **Workbench/Review** | Human operator interface and browser-safe command/read boundary | Proffer projections and authenticated commands | Seven-to-ten explicit views, server-projected valid actions, mode isolation, and truthful unavailable states. | Missing first-class entity workflow, complete attempt comparison, operational n8n/Temporal identity, and executable recovery commands. |
| **Probata context/evidence** | Context authority, evidence promotion, provenance, custody, investigation, and projections | Approved proposal outputs and later promotion decisions | Strong conceptual separation between context preparation and evidence promotion. | Promotion lifecycle was not reviewed as a current live end-to-end implementation; commit/read-back and custody transitions need current proof. |
| **Docstore/CocoIndex** | Governed documentation corpus and natural-language search | Governed documents and indexing receipts | A handoff ledger was written and read back through Docstore, although indexing was not automatically triggered. | Write/read-back does not prove index freshness; indexing requires a separate receipt. |
| **Smart Explore** | Structural code search and decision/conflict discovery | Source code and tree-sitter/DuckDB index | Intentionally separate from documentation indexing. | Federation must not erase source type, freshness, or index-management identity. |
| **SBV Forensic** | SMS/MMS/call import, review, media browsing, and full-text search | SMS Backup \& Restore XML | Mature standalone source-specific ingestion and viewing surface. | No reviewed versioned export contract into Proffer; SQLite BLOB storage and per-user identity must not become evidence authority by accident. |
| **Advocatio** | Legal strategy, research, drafting, review, release manifests, filing readiness, and missing-proof requests | Accepted `LegalSourcePackage` references and legal authority | Correct legal/evidence separation, owner review, deterministic release-candidate concept, and blocked destructive/legal actions. | Persistence authority is contradictory; production PostgreSQL migration, authentication, internal search, and complete evidence-platform integration require current verification. |
| **Legal-reference guide** | Michigan family-court legal information, routing, checklists, and research support | Primary authorities, forms, case law, source ledger | Useful safety gates and structured research process.[^1_2] | Last reviewed state was publication-blocked by legal-currentness, deadline, citation, manifest, and packaging issues; current remediation status is unverified. |
| **Propria umbrella** | Presumed ecosystem routing and shared project organization | Multiple child repositories | Referenced as the higher-level project boundary. | Source and current authority map were not directly verified in this review. |
| **TraceIQ Rebuild** | Intended role not established from the inspected artifacts | Unable to verify | None credited | Must not be assigned authoritative analytics, graph, timeline, or evidence responsibilities until its contract and current source are reviewed. |

## Cross-module review

### Source-to-context journey

The intended path is:

1. The operator browses or searches files in Intake.
2. Intake identifies a real source set without performing evidence admission.
3. A retained source package preserves originals, members, attachments, metadata, and hashes.
4. Proffer creates an operation and immutable attempt.
5. Processing stages produce records, chunks, entities, relationships, temporal expressions, warnings, lineage, and receipts.
6. Review exposes the entire proposal and its configuration.
7. The owner approves one frozen attempt and selected destination plan.
8. PostgreSQL, Weaviate, and Neo4j are committed independently and read back.
9. A later governed action may promote selected material into evidence.

The current break occurs between steps 2–8. Intake does not yet have a proved canonical source-package handoff, Proffer lacks a current valid real proposal and complete operation creator, and proposal destination commits are unproved.

### Evidence-to-legal journey

The intended legal path is:

1. Probata produces a version-pinned `LegalSourcePackage`.
2. Advocatio imports the package without copying evidence authority.
3. A user creates issues, factors, draft sections, and exact citations.
4. Citation gates validate assertion and span references against the imported package.
5. Owner review approves a draft.
6. A deterministic release manifest excludes private strategy/red-team material.
7. Missing proof returns to Probata as an investigation request.

This boundary is one of the strongest areas of the architecture. The earlier audit found that Advocatio used assertion and item references instead of duplicating raw evidence, and that its package verification depended on a fail-closed evidence-platform client. The remaining risk is operational: package versioning, integration authentication, persistence authority, request deduplication, and closure of investigation requests need current end-to-end proof.

### SMS journey

SBV can upload and incrementally parse SMS Backup \& Restore XML, persist messages and media, provide FTS5 search, and isolate users into separate SQLite stores. What is missing is a governed bridge that exports:

- Original XML identity and hash.
- Parser and parser-version identity.
- Message, MMS-part, attachment, and call-record locators.
- Parse warnings and unsupported fields.
- Import idempotency evidence.
- Exact source-member lineage.
- A signed or hashed package manifest acceptable to Proffer.

Without that bridge, SBV is a useful review application but not a verified participant in the source-to-context chain.

### AI and agent journey

Intake’s handoff describes bounded model iterations, bounded search snippets, provider configuration outside the renderer, and restrictions against automatic content reads merely to construct chat context. Advocatio routes narrow agent roles through configurable backends, blocks approve/file/serve intents, marks output non-court-safe, and leaves live invocation disabled unless explicitly enabled.

The missing ecosystem layer is a shared AI execution envelope containing:

- Actor and user intent.
- Matter, mode, resource, operation, and attempt scope.
- Model and provider identity.
- Prompt/skill/tool versions.
- Retrieved source identities and freshness.
- Tool calls and destructive-operation approvals.
- Token/cost/time budgets.
- Citation coverage.
- Safety decisions and fallbacks.
- Final disposition and evaluation result.


### Failure and recovery

The architecture increasingly represents failures honestly. The Workbench can show the failed stage, attempts, receipts, missing controls, and the fact that “start a new import” is not an exact-stage retry.

It still cannot consistently let the user:

- Cancel a running operation.
- Retry an exact failed stage.
- Resume from a durable checkpoint.
- Apply a specific allowlisted repair.
- Edit a processing profile and create a replacement immutable attempt.
- Compare complete attempts.
- Reconcile an ambiguous destination commit.
- Roll back or supersede a bad projection without affecting the retained original.


### Security boundaries

Confirmed controls include local-only or tailnet-oriented operation, renderer credential exclusion, fail-closed legal actions, per-user SBV stores, httpOnly SBV sessions, explicit no-public-exposure warnings, and source-only repository secret scans.

The largest unresolved security concern is not a known exploit; it is **policy fragmentation**. The reviewed materials do not establish one principal, role, authorization, audit, and delegation model spanning Intake, Proffer, Probata, SBV, Docstore, model gateways, and Advocatio.

### Data lifecycle

The ecosystem recognizes originals, retained packages, derived artifacts, attempts, chunks, projections, legal references, drafts, releases, and caches, but does not yet expose one complete lifecycle state machine.

A required lifecycle should distinguish:

- `discovered`
- `selected`
- `retained`
- `processing`
- `proposal_open`
- `proposal_frozen`
- `approved_for_context`
- `context_committed`
- `evidence_candidate`
- `evidence_approved`
- `superseded`
- `held`
- `deletion_requested`
- `projection_removed`
- `source_destroyed`
- `tombstoned`

Every transition should identify who authorized it, what immutable object it targets, whether children or projections are affected, and how recovery or legal hold prevents destruction.

## Prior-review reconciliation

| Prior finding | Current status | Evidence | Remaining work or new risk |
| :-- | --: | :-- | :-- |
| Advocatio must not duplicate or establish evidence | **Resolved/confirmed** | Current authority documents retain the boundary, and the earlier code audit found references rather than duplicated evidence stores. | Add continuous architecture tests preventing evidence bytes or custody tables from entering Advocatio. |
| Advocatio had no persisted file upload pipeline | **Superseded as a product gap** | The current boundary says Advocatio consumes accepted packages and should not ingest evidence. | The UI must not imply that its browser-local PDF viewer imports evidence. |
| Advocatio runtime schema diverged across PostgreSQL, SQLite, and ORM | **Unable to Verify; likely still material** | The earlier audit found live SQLAlchemy/SQLite drift, while current docs name PostgreSQL and JSON/JSONL durability. | Select and document one authority, migrate it, prove restart durability, and remove stale alternatives. |
| Advocatio lacked internal workspace search | **Unable to Verify** | Earlier audit found only CourtListener outbound search and no internal full-text/vector search. | Reinspect current code; if still absent, implement bounded legal-work-product search without copying evidence authority. |
| Advocatio canonical-directory ambiguity | **Resolved** | The 2026-09-13 reconciliation establishes `Legal-desktop` as canonical and retires the former Probata module path. | Refresh generated launchers and remote indexes that may still contain old absolute paths. |
| Proffer operator view concealed missing data and valid actions | **Partially resolved** | New operator surface exposes explicit tabs, server-computed valid actions, and unavailable reasons. | Implement the commands behind missing retry, repair, rerun, and comparison controls. |
| Chunk preview was not guaranteed before final review | **Resolved for the current non-messaging path** | Follow-up integration runs verification and chunking before preview and binds exact package/attempt/chunk references. | Bind queries directly to attempts for multi-attempt operation and prove the messaging path. |
| Parser/options changes could approve stale output | **Partially resolved** | Changes now produce `rerun_required` instead of approving stale output. | Implement the authenticated replacement-attempt command. |
| No immutable rerun/attempt comparison | **Unresolved** | Both current Proffer reviews retain this gap. | Add attempt-creation command, expected-state token, idempotency, complete receipts, and comparison projection. |
| No n8n/Temporal operational truth | **Unresolved** | Current UI exposes logical stages but not workflow version, activation, execution, node, or safe link identity. | Add browser-safe operational projections and deployed proof. |
| No canonical proposal artifact | **Unresolved** | The current gap ledger says no valid canonical v3 proposal exists for the retained 69-record package. | Build, freeze, bundle, verify, load, and visually inspect one canonical proposal. |
| Intake had only conceptual search integration | **Partially resolved** | Synthetic CocoIndex/NIM/Weaviate keyword and hybrid proof passed. | Complete native UI smoke and bounded real-source proof. |
| Intake native application was unproved | **Partially resolved** | Native process/build evidence exists, but click-through and visual acceptance were explicitly not verified. | Run repeatable native acceptance tests with screenshots and recorded outputs. |
| Semantica was entirely unwired | **Partially changed, still unresolved** | Earlier workspace analysis found no runtime wiring; current Proffer ledger treats Semantica as an unintegrated selectable capability lane.[^1_1] | Implement independent attempt identity, receipts, comparison, and explicit unavailable behavior. |
| Legal guide was complete and publication-ready | **Invalidated by prior audit; current remediation unverified** | The project-folder remediation report identified release-blocking legal, deadline, manifest, and packaging defects.[^1_2] | Re-run the release gate against the latest tree before exposing guide output as authoritative. |

## Issue registry

| ID | Affected area | Classification | Finding | Evidence | Impact | Severity | Recommendation | Dependencies | Validation |
| :-- | :-- | :-- | :-- | :-- | :-- | :-- | :-- | :-- | :-- |
| **ARCH-01** | All repositories | Architecture/scope | **Confirmed:** The system is represented across multiple independent repositories and child checkouts without one machine-verifiable ecosystem manifest. | Consignatio requires separate clones for its desktop and kit; Advocatio and Probata also have independent canonical-path decisions. | Incomplete clones, wrong branch use, stale instructions, and false completion claims. | High | Create a versioned ecosystem manifest listing every required repo, branch/tag, role, authority, API version, and bootstrap validation. Propria should own it. | Owner decision on ecosystem boundary. | A clean machine can bootstrap all required source and pass a manifest conformance command. |
| **ARCH-02** | Probata canonical checkout | Architecture/operations | **Confirmed:** Current Proffer work was isolated from a dirty, diverged canonical Probata checkout. | Gap G11 and ordered reconciliation plan. | Unique work can remain stranded; deployments may run a different lineage. | Critical | Reconcile through a clean worktree, preserve all dirty paths, merge reviewed changes, and record exact resulting commit lineage. | Owner-reviewed preservation plan. | Canonical checkout commit contains all accepted work; clean comparison reports no unexplained loss. |
| **FLOW-01** | Intake → Proffer → Probata | Functional/integration | **Confirmed:** There is no proved complete source-selection-to-context-commit workflow. | Intake lacks final handoff proof; Proffer lacks valid proposal, complete operation creator, and destination commit. | Core user outcome cannot be completed. | Critical | Define and implement the versioned `SourcePackageSubmission` handoff and one bounded reference E2E path. | ARCH-02, identity contract. | One real source completes selection, retention, proposal, review, approval, commit, and read-back. |
| **DATA-01** | Proffer proposal | Data | **Confirmed:** No valid canonical v3 proposal existed for the retained real workbook package. | Current Proffer ledger. | Review can display scaffolding without proving real proposal correctness. | Critical | Generate, freeze, bundle, hash, verify, and load the canonical proposal. | Canonical schema and retained package. | Go verifier, adapter validation, table counts, logical digest, byte digest, and UI inspection all agree. |
| **API-01** | Proffer Review API | Functional/security | **Confirmed:** Generic DuckDB discovery and secured read-only adapter were not implemented in the gap ledger. | G2. | Review cannot safely consume canonical proposals. | Critical | Implement root containment, symlink rejection, read-only DuckDB settings, schema validation, digest verification, and mode/matter isolation. | DATA-01. | Positive real-bundle tests and negative path, schema, mutable-state, digest, and symlink tests pass. |
| **UX-01** | Intake/Workbench operation creation | UX/functional | **Confirmed:** Users cannot yet create a complete visible operation with all required source and processing choices. | G3. | Users encounter hidden defaults and cannot reproduce or understand processing. | High | Build a first-class operation creator with explicit source, mode, matter, path, parser, repair, extraction, entity, relationship, and chunk configuration. | API-01 and typed configuration resources. | A user creates an operation without internal IDs; every chosen option appears in receipts. |
| **DATA-02** | Entity/relationship workspace | Data/UX | **Confirmed:** Entity resolution and relationship refinement remain incomplete. | G4. | Graph output can be opaque, inaccurate, or impossible to correct before commit. | High | Add mention review, aliasing, merge/split, link/unlink, temporal qualifiers, and cross-attempt diffs. | Immutable attempt model. | Every entity/relationship change creates a new revision or attempt and survives reload. |
| **OPS-01** | Proffer/Temporal | Reliability | **Confirmed:** Exact-stage retry, checkpoint resume, cancellation, and replacement-attempt commands are absent. | Current operator-surface gaps. | Visible failures remain operational dead ends. | Critical | Implement authenticated commands with expected-state tokens, idempotency keys, authorization, and append-only receipts. | Temporal command policy. | Fault injection proves retry/cancel/resume behavior without duplicate writes or identity confusion. |
| **OPS-02** | Temporal/n8n visibility | Operations/observability | **Confirmed:** Workflow, build, activation, execution, and node identities are not available in browser-safe projections. | Current source audit. | Operators cannot diagnose deployment or activity failures accurately. | High | Add safe operational coordinates and deep links without exposing payloads or secrets. | Security review and n8n metadata contract. | UI shows exact current workflow/version/execution; disabled and failed workflows are distinguished. |
| **DATA-03** | PostgreSQL, Weaviate, Neo4j | Data/reliability | **Confirmed:** Independent proposal commit, destination-drift checking, idempotency, receipts, and read-back are unproved. | G7. | Approval can diverge from committed state or produce partial silent success. | Critical | Implement prepare/approve/commit/reconcile state machines per destination. | DATA-01, API-01, exact approval. | Repeated commits are idempotent; induced partial failures reconcile to explicit terminal state. |
| **AI-01** | Semantica | AI/agent | **Confirmed:** Semantica remains an unintegrated optional processing lane. | Earlier runtime audit and current gap G8.[^1_1] | Silent fallback could create incomparable results or false feature claims. | High | Treat Semantica as a separately receipted attempt path with explicit availability and comparison. | Dependency compatibility decision. | Custom and Semantica attempts run independently and can be compared field by field. |
| **AI-02** | All model-enabled apps | AI/operations | **Likely risk:** No common model/tool execution envelope or cross-app evaluation standard was found. | Intake and Advocatio use separate routing and safety mechanisms. | Inconsistent cost, safety, citation, and audit behavior. | High | Define a shared AI run receipt while retaining app-specific policies. | Identity, resource-scope, and privacy decisions. | Every run exposes model, provider, prompt/skill version, tools, sources, cost, safety, and result evaluation. |
| **LEGAL-01** | Advocatio persistence | Architecture/data | **Confirmed documentation conflict:** JSON/JSONL, PostgreSQL, and historical SQLite/ORM authorities are all represented. | Current agent contract and prior code audit. | Restart loss, split-brain state, migration errors, and unverifiable legal work product. | Critical | Select one canonical runtime store, document migration and event semantics, and make all other stores projections or development fixtures. | Current code inspection and data inventory. | Restart, crash, migration, backup, and restore tests preserve one exact matter state. |
| **LEGAL-02** | Advocatio integration | Security/integration | **Unable to Verify:** The earlier direct evidence-platform call and missing shared gateway hardening have not been reconciled in reviewed current artifacts. | Prior audit identified the direct dependency. | Weak service authentication or inconsistent authorization. | High | Reinspect current client routing; require authenticated service identity and scoped package/hash APIs. | Platform identity policy. | Unauthorized and cross-matter requests fail; authorized package verification succeeds and is audited. |
| **INTAKE-01** | Intake Desktop | Testing/UX | **Confirmed:** Build/process state was treated separately from visual/native acceptance. | Intake handoff explicitly marks click-through unverified. | A runnable process may still contain broken user workflows. | High | Automate native acceptance for panes, selection, preview, chat, search, and failure states. | Stable test fixture and desktop build. | Repeatable test captures prove all core interactions and restart behavior. |
| **INTAKE-02** | Intake → Proffer | Architecture/data | **Observed absence:** No reviewed canonical handoff contract maps selected files into a retained source package and Proffer operation. | Intake and Proffer describe adjacent boundaries but not a proved exchange. | Manual scripts or hidden assumptions can break lineage. | Critical | Introduce a versioned package-submission API, not a new product module. | Shared resource identity. | Selection IDs and file hashes map exactly to package members and Review lineage. |
| **SBV-01** | SBV → Proffer | Data/integration | **Observed absence:** SBV’s specification defines local import and search but not a governed source-package export. | SBV README and technical specification. | Messages may be reviewed but not enter the governed pipeline with complete lineage. | High | Add deterministic export manifest and parser receipt compatible with the Proffer package contract. | FLOW-01 package schema. | Re-export is deterministic; every message/media item resolves to an original XML locator. |
| **SEC-01** | Cross-application access | Security | **Unable to Verify:** No single principal/role/delegation model spans the reviewed applications. | SBV has local sessions, while other components rely on different service and runtime boundaries. | User, service, agent, and automation authority can drift. | Critical | Define a shared signed identity envelope and local authorization matrix. | Owner policy for local-only versus shared use. | Cross-user, cross-matter, cross-mode, and agent-escalation tests fail closed. |
| **LIFE-01** | All data stores | Data/security | **Confirmed design gap:** No unified retention/deletion/reprocessing contract was found. | Current materials define preservation rules and limited deletion safeguards but not one lifecycle. | Orphaned derivatives, stale indexes, accidental destruction, or inability to honor deletion. | High | Define lifecycle transitions and projection invalidation rules before broad real-data adoption. | Legal-hold and backup policy. | A deletion/reprocess simulation reconciles every authoritative and derived store. |
| **TEST-01** | Entire ecosystem | Testing | **Confirmed evidence gap:** Reviewed receipts are scoped per component and explicitly disclaim deployment or full live proof. | Intake, Proffer, and Review receipts. | Component tests can pass while the user journey fails. | Critical | Create an ecosystem conformance suite using bounded synthetic fixtures and one approved real-source fixture. | Stable manifests and service versions. | CI and a deployment rehearsal produce one signed end-to-end receipt. |
| **DOC-01** | Plans, handoffs, donors | Documentation | **Confirmed:** Historical documents frequently remain preserved after decisions supersede them. | Intake and Advocatio contain explicit supersession warnings. | Agents may reintroduce rejected architecture or treat donor instructions as active. | High | Add machine-readable document status and supersession metadata; exclude historical/donor materials from active instruction retrieval. | Docstore schema. | Search results label authority, effective date, superseded-by link, and implementation evidence. |
| **LEGAL-CONTENT-01** | Custody guide | Legal/documentation | **Unable to Verify current remediation:** Last reviewed release state contained P0 legal and package-integrity blockers. | Project-folder remediation report and handoff.[^1_2] | Incorrect deadlines, forms, citations, or integrity claims could harm legal decision-making. | Critical | Keep publication blocked until current primary-source, semantic-citation, route, manifest, and cross-platform tests pass. | Current source tree and legal-currentness date. | Independent adversarial audit finds no unresolved P0 issues and package manifests reproduce exactly. |

## Missing capabilities

### Shared resource identity

A single resource should be traceable across every application without forcing the stores to merge. The identity envelope should carry:

- Ecosystem resource ID.
- Source occurrence ID.
- Original-byte hash.
- Source-system and path identity.
- Matter and TEST/REAL mode.
- Package, operation, attempt, and proposal IDs.
- Record, chunk, entity, relationship, and attachment IDs.
- Approval digest.
- Destination receipt IDs.
- Evidence-promotion and legal-package versions.

This should be a shared contract library and conformance test, not a shared writable database.

### Source-package exchange

Intake and SBV need producer contracts for Proffer. A package submission should be create-only or explicitly versioned and include:

- Original and member identities.
- Byte length and hashes.
- Filesystem and embedded metadata.
- Container/member hierarchy.
- Parser eligibility and warnings.
- Attachment relationships.
- Tool/version receipts.
- Privacy and privilege hints as hypotheses, not legal conclusions.
- Mode and matter.
- Idempotency coordinate.


### Durable command model

Every consequential mutation should include:

- Actor and authorization.
- Target state and expected current state.
- Idempotency key.
- Reason.
- Exact immutable inputs.
- Created attempt or version.
- Append-only receipt.
- Recovery and reconciliation state.

This should apply to repair, rerun, approval, commit, promotion, release, redaction, deletion, and reprocessing.

### Ecosystem observability

A common trace should correlate:

- UI request.
- Workbench BFF request.
- Proffer request and operation.
- Temporal workflow/run/activity.
- n8n workflow/version/execution/node.
- Tool invocation.
- Package and attempt.
- Destination transaction/read-back.
- Model/tool run.
- User-visible terminal result.

Payloads and sensitive content should remain outside operational traces unless explicitly allowed.

### Lifecycle control

The system needs explicit treatment for:

- Source moved or renamed after indexing.
- Source contents changed in place.
- Original unavailable during later evidence promotion.
- Parser version superseded.
- Proposal approved before an index-schema migration.
- Partial destination commit.
- Legal package citing superseded evidence.
- Deletion request while a legal hold exists.
- Reprocessing after corrections.
- Cache or vector-store loss.
- Restoring from backup into a newer schema.


## Architectural changes

### Preserve the product split

Do not merge Intake, Proffer, Probata, and Advocatio into one service or database. Their boundaries reflect materially different authority:

- Intake owns file interaction.
- Proffer owns proposed transformations and attempts.
- Probata owns context/evidence authority.
- Advocatio owns legal work product.
- SBV owns source-specific decoding and review.
- Search stores own rebuildable projections.

The fix is stronger versioned contracts and conformance testing, not consolidation into a larger monolith.

### Consolidate control contracts

Consolidate these cross-cutting definitions:

- Resource identity.
- Mode and matter scope.
- Package manifest.
- Attempt configuration.
- Approval digest.
- Service principal.
- Audit actor.
- Error envelope.
- Idempotency.
- Destination receipt.
- Lifecycle state.

Each application may retain its native implementation language, but generated schemas and contract tests should come from one canonical specification.

### Remove competing authorities

- Resolve Advocatio’s JSON/JSONL versus SQLite versus PostgreSQL authority.
- Reconcile Probata’s canonical checkout.
- Mark donor and historical documents non-authoritative.
- Treat build kits and scaffolds as planning inputs, never runtime modules.
- Keep Dragonfly, Weaviate, Neo4j, SurrealDB, DuckDB, and browser state subordinate to their declared authority roles.


### Add a conformance layer

A new runtime microservice is not required. What is required is an **ecosystem contract and test package** that can verify:

- Repository versions.
- API/schema compatibility.
- Identity propagation.
- Mode and matter isolation.
- Package immutability.
- Attempt reproducibility.
- Approval correctness.
- Destination idempotency.
- Read-back.
- Recovery.
- Audit linkage.


## Remediation roadmap

### Immediate blockers

1. **Reconcile the canonical Probata checkout.**
2. **Resolve Advocatio’s authoritative persistence model.**
3. **Create and verify one canonical real Proffer proposal.**
4. **Implement the secured generic DuckDB Review adapter.**
5. **Implement the first-class operation creator.**
6. **Implement immutable replacement-attempt creation and complete attempt comparison.**
7. **Implement exact approval plus destination-drift validation.**
8. **Implement independent PostgreSQL, Weaviate, and Neo4j commits with read-back.**
9. **Define the shared identity and service-authentication envelope.**
10. **Keep the legal guide publication-blocked until its latest state is re-audited.**

### MVP-critical work

- Implement Intake-to-Proffer source-package submission.
- Implement SBV-to-Proffer deterministic export.
- Complete entity and relationship review.
- Add repair selection and derived-artifact receipts.
- Add authenticated cancel, retry, and checkpoint recovery.
- Bind every chunk projection directly to its producing attempt.
- Run Intake native click-through acceptance.
- Prove one source-to-context and one evidence-to-legal workflow.
- Add missing-proof request delivery, deduplication, status, and closure.
- Prove restart durability for every authoritative store.
- Add ecosystem-level mode, matter, and user isolation tests.


### Post-MVP hardening

- Operational n8n and Temporal metadata.
- Backup and restore rehearsals.
- Migration rollback tests.
- Projection rebuild tests.
- Retention, legal hold, deletion, and tombstone handling.
- Centralized metrics, logs, alerts, and cost accounting.
- AI prompt/tool/version receipts and evaluation suites.
- Parser fuzzing and decompression-bomb defenses.
- Browser virtualization and paginated server-side capability queries.
- Secret rotation and service credential inventory.
- Security review of CSRF, session invalidation, rate limiting, and cross-origin policy.
- Signed build and deployment provenance.


### Longer-term enhancements

- Semantica dual-path comparison.
- SurrealDB whole-case projection after approved-resource boundaries are stable.
- Dragonfly reconstructible UI coordination.
- Multimodal and face-analysis policy after privacy and provider rules are settled.
- Cross-index agent search with explicit source freshness.
- TraceIQ integration after its authority and output contract are reviewed.
- Generalization beyond one matter only after the single-matter workflow is complete and proven.


## Open questions

- Is the product officially two applications, four authority layers, or one named ecosystem?
- Which repository and document contain the canonical ecosystem manifest?
- Has the Probata integration branch been merged into the canonical checkout?
- Which Advocatio store is authoritative today: JSON/JSONL, SQLite, or PostgreSQL?
- What exact object crosses from Intake into Proffer?
- Does an SBV import produce only a local review database, or is it expected to produce a governed source package?
- What is the canonical global resource identifier?
- Which service authenticates human users across applications?
- How are service and agent principals represented?
- Does TEST/REAL mode exist in every authoritative table and event, or only at selected boundaries?
- What permissions can an agent exercise over file move, rename, delete, repair, approval, and promotion?
- What event closes an `EvidenceInvestigationRequest`, and how is the resulting evidence package linked back to the original legal issue?
- What happens when approved content is later reprocessed with a new parser or corrected source?
- How are citations invalidated when an assertion, span, package, or authority becomes stale?
- What is the legal-hold and destruction policy for originals, proposals, caches, vectors, graph nodes, drafts, and releases?
- Are backups encrypted, tested, and independently restorable?
- Which deployment is currently canonical, and which exact commits does it run?
- Has Intake’s synthetic proof been followed by a bounded real-source proof?
- Has the current legal-reference package resolved the earlier P0 deadline, citation, form, and manifest defects?
- What role is TraceIQ intended to own, and is that role authoritative or derived?
- What measurable MVP acceptance threshold determines that the ecosystem is usable rather than merely buildable?

<span style="display:none">[^1_3][^1_4]</span>

<div align="center">⁂</div>

[^1_1]: HANDOFF-2026-08-09-parser-schema-triage.md

[^1_2]: CLAUDE-COPY-REMEDIATION-REPORT.md

[^1_3]: conversation_ingestion_system_design.md

[^1_4]: HANDOFF-2026-08-13-vlex-legal-completion.md

