---
priority: critical
authority: owner_decision
status: active
domains: [docs, intake, consignatio, probata, infra]
docstore_subject: note:ccc_intake_docstore_boundaries
---

# CCC, Intake and Docstore — owner-defined boundaries

<!-- Updated by: Codex | Date: 2026-09-12 | Rev: 3 | Platform: Codex / win32 | Changes: record Dragonfly availability and cross-task approval check | Context: owner requires current approved-use coordination -->

Authority: the owner's explicit clarification and instruction to record it on 2026-09-12. This is the shared boundary reference, not a new application or runtime. Local repositories repeat the essential definitions so their instructions remain useful independently.

| System | Purpose | Scope |
|---|---|---|
| CCC | Index and search each codebase | CocoIndex Code only; project-local development indexes |
| Intake | Explore, search, reconstruct and organize files with a human and agent | Multifaceted CocoIndex application, Weaviate advanced search, SurrealDB filesystem/atomic-unit relationships, multiple tools/libraries and patterns from nearly all CocoIndex examples |
| Docstore | Index, retrieve and manage project documentation | CocoIndex + SurrealDB; documentation tools, resources and Surrealist/Studio graph access |

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
Carbon-Linen-Seal palette still requires rendered owner approval and is not yet an accepted product
color decision.

Do not share or silently repoint app/environment identity, tracking databases, worker locks, configuration/credentials or target tables/collections between these systems. Endpoint sharing, if explicitly designed, does not merge ownership. Human-facing command names must disambiguate the system; never run codebase ccc indexing as Docstore or Intake ingestion.

Docstore retains its existing ProbataDocStore app and probata-docstore environment unless a separately verified migration changes them. Its current Markdown planning-helper limits do not limit Intake eligibility. Docstore does not acquire Intake's entire multimodal stack simply because both use CocoIndex.

Intake's filesystem relationship graph is also separate from downstream evidence-analysis data.

## Current source locations

- Intake: Consignatio/Intake; backend: Consignatio/Intake/backend.
- Docstore pipeline: Probata/probata/scripts/docstore; plugin source: Probata/probata/plugins/docstore.
- CCC: each initialized codebase's own index; not either product's ingestion command.

Historical plans and receipts remain historical. Apply this clarification when interpreting conflicting scope language; continue using verification receipts to determine actual implementation status.
