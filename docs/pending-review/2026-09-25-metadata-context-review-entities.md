---
title: Deep metadata screen, context review (incl. hindsight-only foreshadowing), entity extraction with aliases
date: 2026-09-25
status: OWNER REQUIREMENTS 2026-09-25 19:13 EDT — build dispatched ("and action")
domains: [probata, workbench, engine]
tags: [metadata, exif, docx, ocr, context-review, overlay, foreshadowing, hindsight, knowledge-horizon, entities, aliases, requirements]
---

# Metadata screen, context review, entity extraction

> _Byline: Claude Code · Opus 5.5 · 2026-09-25._

Owner 2026-09-25 19:13: "Don't forget the metadata review and correction … the in-depth file metadata you showed me … XML files and docx files and all the information in there, and the information stored inside of screenshots … I need to click on it and see a screen [with] all the metadata. Context review: adding or correcting who the messages are to or about, whether or not it's about the child, whether or not it's relevant — checkbox for relevance. Also add a checkbox for foreshadowing … an internal flag … since I know it was significant in hindsight, but we don't want to taint the table … a system flag called foreshadowing. Also entity extraction is becoming important, with aliases." Then: "and action."

## 1. Deep metadata screen

**Scope correction (owner 19:15): "XML and DOCX were an example. It will include ALL metadata for ALL files, including any available accompanying sidecar."** So: every embedded metadata field exiftool / the format's own parser can read, for every file type (office docs, PDFs, images, video, audio, archives, email, XML/JSON exports, text), PLUS every sidecar that accompanies the file — Google Takeout `<file>.json` (and `<file>.supplemental-metadata.json` variants), `.xmp`, Apple `.aae`, `*.sidecar.md` / `*.EXTRACTION.md` (owner analyses — show, never flag or move), and any same-stem companion the catalog links. Sidecar values are shown next to the file's own values with their source named; a conflict between them is flagged, never silently merged.

Click any file (Sources or Review) → a screen with ALL of its metadata: container/format internals (DOCX core/app properties, revision/author/created/modified; XML headers/declared app/version/backup counts), image EXIF (device make/model, software, original capture time with its named source and conflict flag, GPS, dimensions), OCR text for screenshots, hashes, catalog provenance. Reuse the existing extractor (legal workdesk metadata/original-time/OCR routes on exiftool + tesseract; Intake `image_facts` / `original_time`) — shared engine, not a second copy.

## 2. Context review (corrections are OVERLAYS — source values stay immutable)
Per message/record, attributed + timestamped, append-only revisions:
- who it is TO and who it is ABOUT (add/correct; linked to entities once §3 exists);
- **about the child** (yes/no/unsure);
- **relevant** (checkbox);
- **foreshadowing** (checkbox) — **HINDSIGHT-ONLY system flag.** It records that the owner now knows, in hindsight, this was significant. It must NEVER reach the as-lived / ignorant view: stored as a separate overlay carrying an explicit hindsight horizon, excluded by the horizon PRE-filter in every store (Postgres, Weaviate, Surreal), never on the authored spine, never a column on the message table. This is the project's core knowledge-horizon rule (AGENTS.md "WHY THIS EXISTS": one leaked future fact silently spoils the ignorant walk).

## 3. Entity extraction with aliases
Populate the EXISTING `registry.entity` / `registry.entity_alias` / `working.entity_mention` / `working.entity_resolution`: extract people (and places/orgs) from normalized messages, resolve aliases (spellings, nicknames, phone numbers, handles → one entity), keep mentions linked to their source records, owner can merge/split/rename. Extraction via the DuckDB ELT → Weaviate → Surreal rule. Replaces the rejected per-person Weaviate columns (`catrina_class`, `katrina_ref_type`, `katrina_conf`). Entities feed §2's to/about.

## 4. Events (owner 2026-09-25 19:17)
"Make sure events are extracted also. If it was not auto-detected as an event, you'll be able to select it and say it's an event worth recalling, and commit it to … an event log, timeline log."
- Same propose → correct → commit pattern as entities, built by the `entities` agent.
- **Uses the EXISTING timeline schema — no new log:** proposals -> `timeline.event_candidate` (+ `_relative_time_anchor`, `_source_range`); committed -> `timeline.timeline_collection` / `timeline.timeline_member`; view -> the existing Timesketch projection (`timeline.timeline_projection_*`, `server/timeline/`).
- Auto-detected events (detected_by='auto') AND owner-marked events: "Mark as event worth recalling" on any message -> event_candidate (detected_by='owner'), dated by the SOURCE record's time, not by now.
- Knowledge horizon: an event carries occurred_at + its source's availability time; the as-lived view sees it only from `source_available_from`. "Worth recalling" is an owner provenance marker, NOT a hindsight flag (foreshadowing stays in §2's hindsight-only overlay).
