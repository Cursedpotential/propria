> Updated 2026-09-24: palette references now reflect the owner-directed Probata color alignment; earlier verification dates below do not verify this update.

---
priority: critical
authority: owner_decision
status: active
domains: [docs, intake, consignatio, probata, infra]
docstore_subject: note:ccc_intake_docstore_boundaries
---

# CCC, Intake and Docstore — owner-defined boundaries

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 4 | Platform: Codex / win32 | Changes: establish Docstore as the universal Propria documentation memory and tool plane | Context: explicit owner clarification -->

Authority: the owner's explicit clarification and instruction to record it on 2026-09-12. This is the shared boundary reference, not a new application or runtime. Local repositories repeat the essential definitions so their instructions remain useful independently.

| System | Purpose | Scope |
|---|---|---|
| CCC | Index and search each codebase | CocoIndex Code only; project-local development indexes |
| Intake | Explore, search, reconstruct and organize files with a human and agent | Multifaceted CocoIndex application, Weaviate advanced search, SurrealDB filesystem/atomic-unit relationships, multiple tools/libraries and patterns from nearly all CocoIndex examples |
| Docstore | Search, recall, record and govern documentation for the entire Propria monorepo | Probata-hosted CocoIndex + SurrealDB service; universal documentation tools, resources, notes, decisions, revision history, flags and Surrealist/Studio graph access |

## Docstore's universal scope

Probata currently hosts Docstore, but Docstore is not limited to Probata's product
documentation. It is the shared documentation semantic-search, recall, note and
decision plane for **all Propria projects and agents**. Every project's documents
will be registered and migrated into its indexed corpus over time while the source
files retain their owning repository, path, provenance and authority.

The universally available Docstore capability set must include:

- semantic and structured documentation search;
- bounded, context-efficient recall with source and revision provenance;
- note and decision creation and revision-safe updates;
- separate priority, authority and status flags, including approved-revision state;
- document/resource retrieval and relationship/graph inspection;
- index health, source freshness and CocoIndex CDC verification tools; and
- clean result shaping for agent context, with full-detail drill-down retained.

All agents must query Docstore for related current decisions before changing
project documentation or recording a new note, then write through the governed
note/decision tools and read the result back. A chat message or local file alone
does not satisfy durable note persistence. Agent integrations must expose these
capabilities as first-class discoverable tools, skills and resources rather than
requiring knowledge of a private script path. Registration and federation may be
implemented through the approved shared tool gateway, but must preserve Docstore's
own application identity, credentials, tracking state, locks and write authority.

Progressive migration into Docstore does not make Docstore the owner of each
source document and does not merge product authority. CCC remains codebase-only
development search. Intake remains filesystem reconstruction and organization.
Neither CCC nor Intake is a substitute for universal project-document recall.

## Intake's workflow and eligibility

Find → group → compare → decide → move; include or exclude material from later processing. Dual-pane navigation, previews, metadata and selection-aware chat support the same working context. Record human grouping and destination decisions as provenance-bearing examples for later agent proposals.

Index readable unique content before organization or evidence selection. Unknown or unsupported formats still get inventory metadata and an explicit extraction status. Empty, broken and unhydrated entries stay visible; do not hydrate placeholders incidentally.

Exact byte duplicates can share extraction/embedding work, but retain every occurrence, original location, available metadata and provenance. Duplicate detection is not deletion authority. Differing formats, independent exports/devices, screenshots and corroborating copies remain distinct and linked rather than silently discarded.

Being searchable is not evidence acceptance. Downstream exclusion need not remove an item from the organizing index. Intake is not a candidate-review queue.

Intake's intended processing includes OCR, speech-to-text (STT), video transcription/processing (VTT), advanced small-language-model extraction/classification, multiple vector representations, sidecars, cross-store hashes, nested atomic units and archival/data-lake outputs. Handler choices follow the owner's agreed pipeline decisions. These are requirements, not a claim all handlers are live.

## Isolation

### Additional shared stack resource: Dragonfly

Owner update, 2026-09-12: **Dragonfly is added to the stack and will be available as a resource** for Docs/Docstore, Case Bible, Intake and future applications. This is owner-reported stack availability, not a live connectivity or deployment verification by this agent.

No integration role or connection is assigned by this note. Before an application uses it, establish endpoint, credentials, application/environment key isolation, retention/eviction requirements and failure behavior. Existing stores and ownership remain unchanged; adding Dragonfly does not migrate human flags, document revisions, approvals or CocoIndex tracking state. The shared clean-result contract also applies to any Dragonfly-backed tool or application surface.

Owner follow-up: when Dragonfly or another shared stack resource becomes applicable to a concrete feature, notify the owner with the proposed use and inspect the relevant other task's latest updates for **current approved use** before integrating. Availability is not role approval. Do not infer approval from an old plan or another agent's proposal. If the owning task cannot be identified or its current approval is unclear, ask the owner rather than guessing. This is an on-demand coordination check, not a background monitor.

The owner-approved [shared result presentation and document revision contract](RESULT-PRESENTATION-CONTRACT.md) applies to Docs, Case Bible, Intake and all future applications: compact context-efficient results with preserved provenance/diagnostics, full-detail drill-down, and separate current-index, revision-history, human-flag and approval responsibilities.

The [Propria shared surface design contract](SURFACE-DESIGN-CONTRACT.md) aligns owner-facing semantics,
context, typography, accessibility and visual tokens without merging application runtimes or authority.
Its Intake layout rules consolidate the current native split/selection/chat receipts. Its proposed
Probata graphite/indigo palette follows the 2026-09-24 owner direction; source alignment alone does not prove product
color decision.

Do not share or silently repoint app/environment identity, tracking databases, worker locks, configuration/credentials or target tables/collections between these systems. Endpoint sharing, if explicitly designed, does not merge ownership. Human-facing command names must disambiguate the system; never run codebase ccc indexing as Docstore or Intake ingestion.

Docstore retains its existing ProbataDocStore app and probata-docstore environment unless a separately verified migration changes them. "Probata-hosted" identifies its implementation home, not its corpus boundary: the corpus is progressively project-wide. Its current Markdown planning-helper limits do not limit Intake eligibility. Docstore does not acquire Intake's entire multimodal stack simply because both use CocoIndex.

Intake's filesystem relationship graph is also separate from downstream evidence-analysis data.

## Current source locations

- Intake: Consignatio/Intake; backend: Consignatio/Intake/backend.
- Docstore pipeline: Probata/probata/scripts/docstore; plugin source: Probata/probata/plugins/docstore.
- CCC: each initialized codebase's own index; not either product's ingestion command.

Historical plans and receipts remain historical. Apply this clarification when interpreting conflicting scope language; continue using verification receipts to determine actual implementation status.
