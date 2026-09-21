---
title: DerivedKnowledge/messaging — vault layout design (pasted by the owner)
date: 2026-09-21
source_service: unknown (another assistant; pasted by the owner into the Probata Workbench chat at 00:06 EDT — service not stated)
captured_by: Claude Code · Fable 5.1 · session 77aa963a
status: owner-supplied design; the vault is NOT built yet (owner 00:05: "its not built yet make it configurable")
domains: [probata, consignatio, vault, messaging]
tags: [transcript, vault-layout, derivedknowledge, messaging, sourcecorpus, evidencevault, conversations, decision-input]
---

> _Byline: saved verbatim by Claude Code · Fable 5.1 · 2026-09-21. Nothing below the rule is edited. Context given by the owner just before it, same chat: the vault's top-level folders are Triage, Recovered, SourceCorpus, KnowledgeBase, DerivedKnowledge, CaseManagement, EvidenceVault, Code, Archive (no numbers in the names)._

---

**Yes—`messaging/` needs to be its own Level-2 section.** We had places for findings extracted **from** conversations, but were missing the usable, readable **conversations themselves**.

## Updated `DerivedKnowledge/`

```text
DerivedKnowledge/
├── INDEX.md
├── Dashboard.md
├── AGENTS.md
├── MANIFEST.json
├── _Incoming/
│
├── messaging/
├── entities/
├── claims/
├── facts/
├── events/
├── timelines/
├── relationships/
├── graphs/
├── patterns/
├── ontologies/
├── geospatial/
├── observations/
└── models/
```

## What `messaging/` contains

This is the **human-readable messaging layer**: complete conversation transcripts, linked conversation histories, and useful views of processed human-to-human communications.

It is **not just summaries**, and it is not another storage location for the original exports.

The distinction is:

| Location | What belongs there |
|---|---|
| `SourceCorpus/messaging/` | Native messaging sources in their canonical structured homes. Complete export bundles can remain under `SourceCorpus/exports/`. |
| Source-associated processing artifacts and existing data stores | Parsed messages, normalization results, attachment mappings, processing records, and other machine-oriented outputs. |
| **`DerivedKnowledge/messaging/`** | Readable Markdown conversation documents and navigable views generated from those processed records. |
| Other `DerivedKnowledge/` sections | Claims, events, entities, observations, and patterns extracted from or supported by those conversations. |
| `EvidenceVault/` | Separately copied, explicitly promoted, immutable artifacts—not an automatic destination for processed messages. |

**These are logical locations within the B2-backed structure. The catalog connects the sources, processed records, readable documents, and any promoted copies.**

## Proposed messaging substructure

```text
DerivedKnowledge/
└── messaging/
    ├── INDEX.md
    ├── Dashboard.md
    │
    ├── conversations/
    ├── participant-views/
    ├── cross-platform/
    ├── chronological-views/
    ├── quality-reports/
    └── exports/
```

### `conversations/` — the actual readable transcripts

**Populated from:** parsed and normalized message records, preserving their links to the original sources.

**Contains:** complete, readable conversation documents organized by recognizable participants, platform, and bounded time periods where needed.

An illustrative collection—not a prescribed filename contract—would look like:

```text
conversations/
└── Matt and Katrina/
    ├── INDEX.md
    ├── SMS/
    │   ├── INDEX.md
    │   ├── 2024-05.md
    │   └── 2024-06.md
    └── Messenger/
        ├── INDEX.md
        └── 2024-05.md
```

The documents retain message text, speaker attribution, timestamps, ordering, and available attachment/reaction/edit information. Each message needs a stable reference back to its processed record and source.

**Splitting a long conversation across documents is pagination, not permission to summarize or omit messages.** Navigation should connect the preceding and following portions.

### `participant-views/` — find conversations involving a person

**Populated from:** conversation membership and resolved participant identities.

**Contains:** navigation pages showing which conversations involve a person or group, the platforms represented, and the available date ranges.

These pages link to `DerivedKnowledge/entities/` for identity information and to the conversation documents for the actual communications. They do not create a competing set of person dossiers or duplicate every transcript.

### `cross-platform/` — follow an exchange across channels

**Populated from:** messages from multiple platforms, using resolved identities and available timestamps.

**Contains:** combined reading views that let you follow an exchange that moved from SMS to Messenger or another channel.

Every message still shows its originating platform and source reference. Identity uncertainty, conflicting timestamps, and uncertain ordering stay visible. Combining records into a view must not turn an inferred connection into an established fact.

### `chronological-views/` — read communications over time

**Populated from:** selected conversation records ordered by their recorded times.

**Contains:** daily, monthly, or date-range communication views across relevant threads.

This differs from `DerivedKnowledge/timelines/`: a messaging chronology shows **what was communicated and when**; a case timeline may combine communications with appointments, filings, location records, and other events.

These can be index-and-link views where reproducing the full message text would only create another copy.

### `quality-reports/` — show what is incomplete or uncertain

**Populated from:** parser results, reconciliation checks, attachment resolution, and identity/time validation.

**Contains:** readable reports of unresolved speakers, unavailable attachments, conflicting timestamps, overlapping exports, parsing failures, and other detected issues.

Reports distinguish **a gap actually demonstrated by the records** from a period for which the system simply has no supplied messages.

### `exports/` — reusable conversation deliverables

**Populated from:** a defined selection of processed messages or a conversation view.

**Contains:** requested Markdown, HTML, PDF, CSV, or JSON deliverables, with their selection scope and source references recorded.

An export remains **context**. Producing a polished PDF does not promote it to evidence.

---

## How the documents connect to the rest of the vault

```text
Original messaging source
        ↓ parse / normalize
Processed message records
        ↓ render
DerivedKnowledge/messaging/conversations/
        │
        ├── links to participants → entities/
        ├── links to assertions   → claims/
        ├── links to occurrences → events/
        ├── links to findings    → observations/ and patterns/
        └── links back to original sources through the catalog
```

**The conversation remains available in full; extracted findings sit alongside it through links rather than replacing it.** AI-chat originals still belong in `KnowledgeBase/ai-chats/`, separate from this human-messaging section.
