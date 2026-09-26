# Consignatio — Level 2 architecture and population guide

**Planning revision:** 2026-09-21 · **Status:** design/scaffold; not deployed.

This is a consolidated continuation of the owner-approved nine-domain model. It includes the added `DerivedKnowledge/messaging/` section. Existing agreed domain names, context-first lifecycle, B2 storage and catalog integration are carried forward. New population details and previously unfinished EvidenceVault subfolders are proposals in this planning revision, not newly approved global agent rules.

## Location and authority

All paths in this guide are **logical paths below the configured B2 vault prefix**. Neither a bucket name nor a B2 prefix is invented here. The Windows path `E:\AI\_Workspace\Projects\Propria\modules\Consignatio` identifies the application module, not a local evidence-storage root.

The existing **catalog** supplies authoritative identities, locations, state and source relationships according to its actual implementation. A local/Obsidian rendering is a navigation surface, not a required storage backend and not a replacement catalog. No API names, table names, completion field names or storage credentials are prescribed.

## Common domain envelope

Every top-level domain has `INDEX.md`, `Dashboard.md`, `_Incoming/`, `AGENTS.md` and `MANIFEST.json` in this draft scaffold.

- **INDEX.md:** purpose, contents, population process, exclusions and usable navigation.
- **Dashboard.md:** catalog-backed operational view when integrated; it must show scope and freshness. This scaffold shows **not connected**, never fabricated live counts or a zero-backlog claim.
- **_Incoming/:** coarse-sort landing/review area. `EvidenceVault/_Incoming/` is promotion staging only; `Archive/_Incoming/` is disposition review only.
- **AGENTS.md:** a descriptive, local planning guide linking to the design. It does not activate automation, install new behavioral rules or override existing owner instructions.
- **MANIFEST.json:** explicitly a **planning-only scaffold descriptor**, not a live catalog or evidence-package manifest. Its keys are not an integration contract.

All substantive Level-2 sections also have an `INDEX.md` explaining how they are populated. Added working dashboards are navigation documents, not live jobs.

## Population flow

1. Inspect the B2 holding corpus using the existing catalog and scoped inspection. Do not guess deep destinations from ambiguous filenames.
2. Route unknown material to Triage; recovery-origin material can pass through Recovered. A known domain can land directly in that domain's `_Incoming/`; neither Triage nor Recovered is a compulsory stop for every item.
3. Fine-sort into its canonical content home on B2, preserving source/package identity and recorded old-to-new relationships. This scaffold does not execute moves.
4. Process from the established home. Source-specific extraction, normalization, attachments, chunks and run artifacts remain associated with that source under the existing processing conventions.
5. Materialize readable conversations, dossiers, timelines, graphs and analytical reports in DerivedKnowledge. Preserve source links and distinguish generated material from human notes.
6. Explicit promotion can select eligible source or derived context directly; it does **not** require that every source pass through DerivedKnowledge first. Promotion creates a separate immutable evidence copy on B2 through the existing governed operation.
7. The Legal Work Desk discovers completed promotions through the catalog and resolves their B2 objects. Working annotations and exhibit assembly do not alter sealed evidence.

## Context, processing and evidence are different axes

An item can be processed, normalized, reviewed or highly relevant and still remain **context**. A familiar filename, a `facts/` folder, a polished PDF, an exhibit number or an analysis result does not create evidence status. Internal promotion and any external court status are distinct records.

Original collected bytes are not rewritten to add metadata, repair text, normalize times or make Markdown. Corrections create documented derived versions. Moving an object into its structured home is not permission to alter its content. No blanket freeze of the entire B2 holding mess is reinstated by this guide.

## Naming and navigation

Retain the agreed PascalCase domain names, lower-kebab-case content subfolders and exact `_Incoming/` spelling. Archive mirrors retain the corresponding domain names. Use readable, collision-safe note names and vault-root-qualified wikilinks so repeated `INDEX.md` filenames are unambiguous. Preserve native package filenames inside preserved native trees; do not rename source files merely to match the presentation convention.

## What is not implemented here

This package contains navigation files, explanations and planning descriptors only. It has no B2 connection, catalog adapter, parser, materializer, promotion operation, watcher, sync configuration, scheduled job, live dashboard data, corpus scan or migration script. Existing service contracts and all Level-3 collection internals remain to be bound to the real implementation.


## 1. Triage

Global classification uncertainty: material whose appropriate owning domain is not yet known.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `unclassified/` | Rough discovery from the B2 holding corpus or a submitted item whose owning domain is unresolved. | Readable item/package review notes, original-location references, classification candidates and unresolved questions. | Once the owning domain is known, route to its _Incoming or established home; do not treat uncertainty as a permanent content type. |
| `bundles/` | Mixed or unidentified archives and export packages discovered during coarse sorting. | Package-level inventories and review notes; the native package remains intact while its contents are understood. | Recognized exports route to SourceCorpus/exports or KnowledgeBase/ai-chats. Do not scatter the native package into unrelated folders. |
| `documents/` | Documents whose operational role is unclear after a light first pass. | Document review records and their associated source objects awaiting classification. | A document is not routed to CaseManagement merely because it mentions court. Distinguish operational case material, collected sources and reference material. |
| `rescued-review/` | Triage references to salvaged material needing a classification decision. | Review views linking to the recovered item and any recovery limitations. | Recovered owns the recovery artifact until a canonical content home is chosen; do not create a second recovery copy here. |
| `carve-review/` | Classification questions about outputs from carving/recovery runs. | Review notes identifying fragments, candidate types and relationships to a recovery run. | The actual carved output remains under Recovered until routed; do not imply completeness or reliable dates from a recovered filename. |
| `conflicts/` | Conflicting proposed destinations, package identities, filenames or provenance associations. | Decision records presenting alternatives and the evidence for each routing choice. | This is an organizational conflict queue, not a replacement for substantive contradictions in DerivedKnowledge/claims. |
| `duplicates-review/` | Existing catalog duplicate/overlap findings and explicit review requests. | Comparison reports and disposition records linked to every relevant source instance. | No silent merge or deletion; byte similarity does not authorize discarding independent provenance or distinct native exports. |

### Dashboard population

Read the existing catalog/processing/review state for: Unclassified packages, Routing conflicts, Duplicate reviews, Items ready for domain routing. Report scope and last refresh. The scaffold itself has no live data.

## 2. Recovered

Recovery-origin material and recovery records, with damage and reconstruction history kept visible.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `salvaged/` | Recovery tools or existing salvage collections producing readable objects. | Recovered source objects, acquisition/recovery references and completeness notes. | A readable recovery is still context; move into its content home only with preserved origin and recorded routing. |
| `carved/` | Carving outputs associated with a specific recovery run. | Recovered fragments or objects, offsets when available, and parser/recovery notes. | Never manufacture missing structure, dates or authors to make a fragment appear complete. |
| `reconstructed/` | Explicit reconstruction processes operating on recovered fragments. | Reconstructed versions plus explanations of inputs, methods and unresolved gaps. | Keep reconstruction distinguishable from the collected or recovered bytes used to create it. |
| `partial/` | Recovery results known to be incomplete or truncated. | Partial objects, readable descriptions of missing ranges and available verified content. | Incomplete content remains discoverable; do not silently drop it or present it as a complete export. |
| `orphaned/` | Recovered files or attachments whose parent, account or source package is unresolved. | Objects and matching candidates with unresolved association notes. | A plausible match is not a confirmed parent relationship. |
| `recovery-records/` | Recovery tools, reviewed run logs and the existing catalog. | Readable run summaries, methods, input/output references and disposition histories. | Retain these references after recovered material reaches its canonical home; a normal export extraction is not a recovery run. |

### Dashboard population

Read the existing catalog/processing/review state for: Recovery runs awaiting review, Partial or orphaned items, Reconstruction limitations, Items ready for canonical placement. Report scope and last refresh. The scaffold itself has no live data.

## 3. SourceCorpus

The canonical structured home for collected human communications, exports, media, geospatial and other source material. Everything remains context.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `exports/` | Recognized platform, account or device export packages routed from coarse sorting. | Complete native packages and source-associated inventories, extracted representations and processing records. | Keep native packages intact. Expose contained messaging, email or geo material through links and derived views rather than duplicating the entire package into each section. |
| `messaging/` | Native human-to-human messaging exports acquired independently or routed after inspection. | SMS, RCS, iMessage and platform-native conversation collections, attachment associations and local processing artifacts. | Readable cross-source conversation documents belong in DerivedKnowledge/messaging. AI chats belong in KnowledgeBase/ai-chats. |
| `email/` | Mailbox exports, individual messages and other collected native email sources. | Native mailbox/message objects, headers, attachments and source-associated parsing results. | Case-running correspondence belongs in CaseManagement/correspondence when that is its canonical operational role; link rather than create an unmanaged duplicate. |
| `screenshots/` | Screenshot acquisitions and reviewed classification of existing image collections. | Unmodified screenshot objects, capture/source references and source-associated transcription or inspection artifacts when produced. | A screenshot of a conversation is a separate source from the native message export; link their relationship without conflating them. |
| `media/` | Photo/video collections acquired outside an inseparable export package. | Photographs, video and their available original metadata; associated previews and processing records. | Attachments that are part of a native conversation package remain associated with that package; source identity survives rendering or transcoding. |
| `audio/` | Acquired recordings and audio collections. | Original recordings, available metadata and source-associated transcripts or other processing outputs. | The transcript is a derived representation, not the original recording. Telephony event logs have their own section. |
| `telephony/` | Phone/carrier exports and collected call or voicemail records. | Call logs, carrier records, voicemail metadata and source-associated normalized records. | Audio files can retain their established source home with linked references; this section does not require a second copy of every recording. |
| `geospatial/` | GPS, location-history, device-location and mapping exports routed as collected sources. | Native tracks, points, location exports and associated parsing/normalization artifacts. | Trips, stops, inferred places and reviewable maps belong in DerivedKnowledge/geospatial. A location product inside Takeout may remain in that native package and be indexed here. |
| `device-data/` | Device-level acquisitions and records not more specifically owned elsewhere. | Device dumps, application data and system artifacts with source-associated processing records. | Route already-distinct messaging, telephony or geo sources to their specific homes; preserve an atomic device acquisition intact. |
| `financial/` | Collected statements, payment records and financial source exports. | Unmodified financial sources plus associated extracted tables and inspection artifacts. | Case-specific support calculations, worksheets and declarations belong in CaseManagement/financials; do not overwrite collected records with calculations. |
| `public-record/` | Collected public records or research acquisitions concerning people, organizations or events. | The actual retrieved documents, captures and source details. | General legal authority/reference belongs in KnowledgeBase/legal. Entity dossiers assembled from these records belong in DerivedKnowledge/entities. |
| `records-data/` | Structured records, datasets and logs with no better specific home. | Source CSV/JSON/log collections and source-associated processing artifacts. | This is not a new database dump destination or a reason to absorb messaging and geospatial into a generic records folder. |
| `documents/` | Collected source documents that lack a more specific content or operational home. | Native documents and any source-associated extracted text or readable renditions. | Operational case documents route to CaseManagement; research/reference route to KnowledgeBase. Unknown roles remain in _Incoming. |

### Dashboard population

Read the existing catalog/processing/review state for: Incoming source collections, Processing status by source, Unavailable attachments or failed parsers, Source collections with readable projections. Report scope and last refresh. The scaffold itself has no live data.

## 4. KnowledgeBase

AI chats, research, reference, notes and personal-history context, distinct from primary collected sources and operational case work.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `ai-chats/` | Native AI-platform exports, saved AI conversations and recognized re-exports. | Complete AI chats grouped by platform/source; readable versions and source-associated processing outputs. | Preserve user statements, assistant suggestions and quoted third-party content distinctly. Extracted entities/events/claims link downstream; AI-generated text is not independent corroboration. |
| `research/` | Research sessions, saved investigations and literature/document review work. | Research notes, questions, findings and their references. | Case-specific strategy/drafting belongs in CaseManagement/work-product. Acquired primary records retain their source identity and home. |
| `legal/` | Legal reference imports, research outputs and reviewed authority collections. | Authorities, reference notes and verification/treatment information actually available. | Keep issuing authority distinct from research commentary. Do not mark citations current or verified from import alone; this plan supplies no legal conclusions. |
| `notes/` | Human-authored general notes and captured contextual thoughts. | Editable notes, contextual annotations and links to relevant material. | Keep these separate from generated views so regeneration does not overwrite personal writing. |
| `docs/` | Imported or authored non-code explanatory documentation. | Procedures, explanations and supporting documents used to understand the corpus or workflows. | The coding/application wiki belongs in Code/wiki; a file is not placed here merely because its extension is Markdown. |
| `reference/` | Stable reference imports and supporting explanatory material. | Reference guides, glossaries and background resources that are neither a specific research session nor case operations. | Executable taxonomy/ontology definitions retain the agreed ontology or authoritative project home and are linked rather than copied casually. |
| `personal-history/` | The owner's recollections, interview notes and personal-history source material. | Attributed accounts, recollections and contextual life-history documents. | Preserve what was recalled, when it was recorded and uncertainty. Later recollection does not become contemporaneous knowledge in an as-lived view automatically. |

### Dashboard population

Read the existing catalog/processing/review state for: Incoming AI chats/reference, Unprocessed chat collections, Reference review status, Human notes and new research. Report scope and last refresh. The scaffold itself has no live data.

## 5. DerivedKnowledge

Usable documents and navigable views of structured knowledge. Underlying machine records remain in the existing authoritative stores; these views remain context.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `messaging/` | Processed human-to-human message records and the source/record relationships already maintained by the system. | Complete readable conversations, participant indexes, cross-platform and chronological views, quality reports and requested exports. | Preserve message content and context; paginate rather than summarize away material. Detailed substructure is carried forward below. AI-chat originals remain in KnowledgeBase. |
| `entities/` | Identity resolution across source records, reviewed identifiers and human corrections. | Readable dossiers for people, accounts, addresses, households, organizations and devices; candidate identities and alias histories. | Canonical identity is not verified biography. Ambiguous merges remain visible. Raw contact exports remain sources rather than new entity assertions. |
| `claims/` | Attributed assertions extracted from messages, documents, interviews or AI-chat context. | Claim pages showing the assertion, speaker, date/context, support, contradictions and uncertainty. | A person or model asserting something is not proof it happened; retain the exact source context and distinguish direct quotation from paraphrase. |
| `facts/` | The existing structured fact model and its recorded review/correction decisions. | Readable factual-record views with source anchors, status, qualifications and competing accounts. | The folder name grants no truth or evidence status. Do not create an automatic claim-to-fact promotion rule in this scaffold. |
| `events/` | Processed records and existing event extraction/reconciliation outputs. | Event notes showing what was recorded, when it occurred or was reported, participants and source links. | Distinguish an alleged event from a recorded communication about it. Do not substitute file or ingest time for occurrence time. |
| `timelines/` | Existing event/claim/fact records and approved temporal-view generation. | Bounded chronologies, topic timelines and the established as-lived and hindsight views. | As-lived uses the intended temporal knowledge boundary, not just date sorting; later revelations must not leak backward. Hindsight identifies its batch/snapshot and scope. |
| `relationships/` | Resolved links in the canonical relationship model and reviewed corrections. | Readable relationship histories, household summaries, communication associations and relationship-specific review notes. | Do not create a Markdown file for each graph edge. Mere co-occurrence does not establish a personal relationship. |
| `graphs/` | The existing graph stores, graph-generation outputs and selected view/export processes. | Explained entity, communication, temporal and provenance network views with links to useful graph artifacts. | Keep node/edge semantics, source references and graph version visible. Machine graph storage remains where implemented, not inside a mandatory vault DB. |
| `patterns/` | Explicit pattern-analysis runs against known input records and human review of results. | Reports describing the finding, observed instances, counterexamples, scope, method and uncertainty. | A pattern is an analytical output, not a diagnosis, established motive or automatic evidence promotion. Do not hide neutral/contrary instances. |
| `ontologies/` | Imported existing ontology assets, owner-authored/reconciled definitions and generated ontology documentation. | Native ontology/vocabulary/taxonomy assets or references to their authoritative homes, mappings, change history and readable concept/hierarchy guides. | Imported/authored definitions are not automatically model-derived. Preserve existing ontology lineage and distinguish candidate definitions from accepted versions; do not auto-union old ontologies. |
| `geospatial/` | Processed location records, the existing trip/stop/leg pipeline and reviewed spatial analyses. | Readable trip/place/route notes, stop and cluster reports, device-separation reviews, maps and useful spatial exports. | Keep source points and processing artifacts associated with their source; avoid a note per point. A device location does not by itself prove a person's presence. |
| `observations/` | Agent passes, chronological walks and human inspection of a defined dataset. | Dated observations describing what was noticed, the inspected scope and links to supporting and contradictory records. | An observation can be tentative and later revised. Preserve it without silently upgrading it into a fact or pattern. |
| `models/` | Saved analytical model artifacts, configurations and materialized explanations from existing workflows. | Model cards, feature definitions, evaluation notes and reusable analytical specifications or references. | Distinguish analytical models from ontologies and implementation schemas. Large checkpoints or executable source stay in their existing authoritative homes unless explicitly relocated. |

### Messaging detail carried forward

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `conversations/` | Processed message records, grouped by native conversation/participants and bounded time ranges. | Complete readable transcripts, stable per-message references and links to attachments and original context. | Split long transcripts into linked parts without omitting routine or inconvenient content; no one-note-per-message explosion. |
| `participant-views/` | Conversation membership and resolved identities. | Person/group navigation pages linking to their conversations and entity dossiers. | No duplicated identity database or repeated full transcript bodies just to create an index. |
| `cross-platform/` | Reviewed identity associations and timestamps across communication platforms. | Combined reading views that identify every message's platform and source. | Preserve uncertain identities/order and original timestamp semantics; do not merge separate claims on the basis of similarity alone. |
| `chronological-views/` | Selected processed communications ordered using known temporal information. | Bounded daily/monthly/date-range communication chronologies or index-and-link views. | These show communications, not every life event; a case timeline lives in timelines/. |
| `quality-reports/` | Existing parse, attachment, overlap, identity and timestamp validation outputs. | Reports of unavailable attachments, failed parsing, conflicts and demonstrated coverage limitations. | Distinguish a proven gap from a period for which no source was supplied. |
| `exports/` | Explicit, scoped rendering/export selections. | Requested readable or machine-reusable conversation deliverables with scope and source references. | Formats such as Markdown, HTML, PDF, CSV or JSON are outputs, not evidence status; do not generate every format for every source by default. |

### Dashboard population

Read the existing catalog/processing/review state for: Materialization freshness and scope, Message/identity reconciliation issues, Unreviewed claims and observations, Ontology and geospatial outputs. Report scope and last refresh. The scaffold itself has no live data.

## 6. CaseManagement

Operational case-running and editable case work, including filings, received material, preparation and exhibit assembly.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `filings/` | Recorded filing events or verified imports of papers actually filed by the user/representative. | The submitted version, available filing receipt/acknowledgment and links to its preparation workspace. | An exported or final-looking draft is not a filed document. Keep later edits in working material, not silently inside the filed copy. |
| `received/` | Documents received from another party, the court or another case participant. | Received pleadings, notices, service material and available receipt/provenance details. | Orders get their dedicated home. Preserve received bytes and do not infer valid service or procedural effect from a filename. |
| `orders/` | Identified court orders/rulings imported from their actual sources. | Order documents, readable references and recorded supersession/relationship information. | Separate operative orders from proposed orders; importing an order still leaves it context under this system's lifecycle. |
| `drafts/` | Human or tool-assisted authoring sessions and unfinished document work. | Editable drafts, revision context and review notes. | Regenerating derived knowledge must not overwrite drafts; no automatic filing or evidence promotion. |
| `motions/` | Creation and maintenance of a motion-specific workspace. | Strategy notes, research links, draft links, supporting-item selections and filed-version references. | The motion workspace is an organizing context, not another mandatory copy of the final filing or each evidence artifact. |
| `discovery/` | Discovery preparation, requests/responses, productions and related reviews. | Requests, responses, production indexes, tracking notes and linked source/produced material. | Keep received/produced identities and provenance; cross-link canonical documents instead of redistributing one source into many unmanaged copies. |
| `correspondence/` | Case-running email, letters and other correspondence captured through the established workflow. | Correspondence documents and navigable communication records relating to operation of the case. | Bulk personal message archives stay in SourceCorpus and readable transcripts in DerivedKnowledge/messaging. A communication may be linked from both without losing one canonical home. |
| `hearings/` | Hearing preparation, attendance notes and subsequently acquired hearing records. | Preparation sheets, questions, outcome notes and links to notices, orders, recordings and transcripts. | Distinguish participant notes from official transcripts and orders; do not turn an inferred outcome into a recorded ruling. |
| `calendar/` | Recorded case dates, confirmed scheduling information and reviewed task/deadline inputs. | Readable calendars, scheduling notes and links to the source for each significant date. | This scaffold does not compute legal deadlines or create calendar events. Unconfirmed dates stay labeled and live integrations remain existing-system concerns. |
| `exhibits/` | Deliberate exhibit selection and document-assembly work. | Exhibit lists, proposed numbering, working/redacted copies and presentation bundles linked to original or promoted artifacts. | Assembly is not promotion. Edits belong to working copies; promoted evidence remains immutable. This folder does not imply court admission. |
| `financials/` | Case-specific financial preparation and reviewed calculations using source records. | Support worksheets, income summaries, financial declarations and related calculation work product. | Collected bank/carrier/payment sources remain in SourceCorpus/financial or their existing operational homes; derived calculations do not replace them. |
| `parenting-time/` | User-entered logs, scheduling records and case-specific review of communications/events. | Schedule records, exchange logs, make-up-time tracking and links to underlying conversations or documents. | Entries carry their source and whether they are firsthand, reported or inferred; this scaffold asserts no case facts. |
| `work-product/` | Human reasoning, drafting and reviewed outputs adopted for case preparation. | Case analysis, strategy, narrative work, chronologies, preparation reports and working review decisions. | This is editable case work, not a mirror of all generated DerivedKnowledge. Preserve human authorship/corrections through later materialization runs. |

### Dashboard population

Read the existing catalog/processing/review state for: Incoming operational documents, Draft/review work, Recorded hearing and scheduling information, Exhibit assembly status. Report scope and last refresh. The scaffold itself has no live data.

## 7. EvidenceVault

A separate domain for explicitly promoted immutable copies and their navigation/review surfaces. Completed promotion, not file type, creates evidence status.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `promoted/` | Only the established explicit promotion operation, after its actual completion checks. | Separately copied evidence artifacts/packages on B2, linked to the originating context and completed promotion record. | No direct coarse sorting into this section. Sealed content is immutable; exact B2 keys, package internals and completion fields are intentionally unspecified. |
| `collections/` | Selection of already-completed promotions using their existing catalog identities. | Readable lists and named sets of evidence references for navigation and review. | These are editable navigation views, not extra evidence copies. Exhibit assembly itself belongs in CaseManagement/exhibits. |
| `promotion-records/` | The existing promotion/catalog audit records exposed as readable views or authorized exports. | Promotion histories, original-to-promoted relationships and recorded supersession references. | Not a new promotion ledger or schema. Completed records are not silently rewritten; record corrections through the actual governed mechanism. |
| `integrity-reports/` | Explicit integrity checks against the selected promoted objects and manifests. | Readable check results with checked scope, artifact/version references, methods and failures where recorded. | A missing report is not a successful check. Reports may be added/refreshed outside sealed payloads without modifying promoted bytes. |

### Dashboard population

Read the existing catalog/processing/review state for: Completed promotions, Pending or failed promotions, Integrity checks and unresolved failures, Evidence collections. Report scope and last refresh. The scaffold itself has no live data.

## 8. Code

Coding/application documentation and technical project assets, respecting the authoritative repositories and existing execution/configuration boundaries.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `wiki/` | Project documentation surfaced from its authoritative repository or authored for the coding/application wiki. | Architecture explanations, developer guides, tooling reference and implementation navigation. | Not KnowledgeBase/docs. A surfaced repo file is not automatically a writable second source of truth. |
| `source/` | Known first-party project source references or authorized source snapshots. | First-party source collections or navigable references to the canonical repositories. | Do not copy or move a live repository merely to fill the scaffold; preserve existing repository ownership and paths. |
| `snippets/` | Deliberately saved code fragments, examples and extracted reusable utilities. | Small reusable snippets with origin and applicability notes. | A snippet is not a production deployment or an automatically trusted executable. |
| `preferences/` | User-approved coding/editor conventions and existing configuration documentation. | Saved preferences and references to settings with their actual scope recorded. | No AI-created global behavioral rules. A local correction does not become a cross-project policy through this folder. |
| `apps-utilities/` | Existing standalone application/tool assets and their documentation. | Utility collections, usage notes, build references and known installation/source relationships. | Do not install, launch or duplicate tools as a side effect of building navigation. |
| `_third-party/` | Third-party source/vendor imports already held or deliberately recorded. | Unmodified upstream material, licenses and provenance/version references. | Keep upstream source distinct from local forks and executable environment state. |
| `forked-3rd-party/` | Known modified forks and their project metadata. | Fork references/snapshots, upstream relationships and documented local changes. | Do not merge or overwrite the upstream copy automatically. |
| `ai-platform/` | Existing AI-platform, agent, skill, plugin, command and integration project assets. | Navigable platform documentation and references to implementation/configuration in their actual homes. | Document actual scope and loaded/configured state only when inspected; this scaffold enables no plugin, MCP, indexer or agent rule. |
| `to-review/` | Technical assets whose origin, relevance, completeness or intended ownership needs review. | Review notes and linked code/tool candidates. | Aging items do not authorize cleanup or execution; code review status is not evidence status. |
| `project-iterations/` | Preserved earlier iterations, release snapshots and design evolution records. | Version histories, migration notes and links to superseded project material. | Retain lineage; a recent timestamp does not prove architectural authority. Active canonical repositories remain active, not moved here by a template. |

### Dashboard population

Read the existing catalog/processing/review state for: Technical items awaiting review, Documentation freshness, Authoritative repository references, Iteration/supersession review. Report scope and last refresh. The scaffold itself has no live data.

## 9. Archive

Intentional retention of classified inactive material, mirroring content domains without becoming a second unstructured dump.

| Section | Populated by | Contents | Boundary |
|---|---|---|---|
| `SourceCorpus/` | A reviewed lifecycle decision that a classified source collection is inactive. | A mirrored inactive path or catalog-backed historical view, with prior-home and source references preserved. | Archive is not a permanent home for unclassified material and does not authorize deletion or break provenance links. |
| `KnowledgeBase/` | Reviewed retirement/supersession of contextual reference or knowledge material. | Historical notes, reference editions and AI-chat collections or their catalog-backed retained locations. | Preserve versions and prior references; archived context is still context. |
| `DerivedKnowledge/` | Superseded materialization snapshots or reviewed retirement of generated views. | Older documents, reports, model/ontology versions and snapshot information. | Do not retain a new full binary copy on every run by default; use the existing version/retention mechanism. |
| `CaseManagement/` | Reviewed retirement of case-work versions or inactive operational material. | Inactive drafts, historical work product and retained operational snapshots. | Preserve filed/received identities and links; archive does not retrospectively change their recorded status. |
| `EvidenceVault/` | Catalog lifecycle views of inactive or superseded completed promotions. | An archive-facing index of the already-sealed evidence objects and their recorded statuses. | Do not move, re-key, overwrite or duplicate sealed evidence simply to make an Archive mirror. Physical objects remain at their catalog-resolved preserved locations. |
| `Code/` | Reviewed retirement of project snapshots, utilities or technical reference. | Inactive project iterations and technical material with authoritative-source references. | Do not move active source repositories or follow symlinks into unrelated working projects. |

### Dashboard population

Read the existing catalog/processing/review state for: Pending disposition review, Inactive collections by domain, Unresolved historical links, Retained/superseded evidence views. Report scope and last refresh. The scaffold itself has no live data.

## End-to-end examples

**SMS export:** SourceCorpus/messaging (or its intact export package) → source-associated parsed messages → DerivedKnowledge/messaging/conversations → linked claims/events/observations → optional explicit promotion of a selected source or derived artifact → EvidenceVault.

**AI chat:** KnowledgeBase/ai-chats → attributed user/assistant content and source-associated processing → linked DerivedKnowledge views. The chat is not independent verification of its contents.

**Location export:** SourceCorpus/geospatial or an intact SourceCorpus/exports package → existing normalized spatial records → DerivedKnowledge/geospatial trip/stop/map views → optional explicit promotion.

**Ontology import:** existing ontology artifact routed/referenced in DerivedKnowledge/ontologies → preserved definition/version → readable documentation and mappings. Its presence does not automatically activate it in the processing system.

**Filed paper:** a recorded filing version in CaseManagement/filings, linked from its motion/draft workspace. It remains context unless separately promoted.

**Exhibit assembly:** CaseManagement/exhibits resolves selected context/promoted artifacts through the catalog; working edits are separate. A completed promotion retains its sealed bytes.
