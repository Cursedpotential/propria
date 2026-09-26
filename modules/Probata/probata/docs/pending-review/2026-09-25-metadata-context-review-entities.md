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

## Build record — §1 and §2 (branch `feat/metadata-context`)

> _Byline: Claude Code · Opus 5.5 · 2026-09-26. Built, unit-tested, NOT applied, NOT deployed, NOT live-proven._

**Metadata screen (§1).** Click the run's file name in Review, or "All metadata" on any message attachment. The screen shows what the platform recorded for that file: registration and retained original (SHA-256, size, sealed time, provenance), every `context.source_metadata` row of every class with its extractor and receipt, sidecars, custody hash receipts and retained members. Engine `GET /reference-import/previews/{handle}/metadata?subject_sha256=`; BFF `GET /api/proffer/previews/{handle}/metadata?mode=`. The BFF adds sidecars from two bounded lookups: the objects directly beside the file with the same stem (one delimited listing) and the catalog's record of the file's original folder (`raw_duck.intake_catalog_fs_20260917`, by vault key). A Takeout capture time or GPS value that disagrees with the file's own value is flagged on both fields, never merged.

**Corrections.** Any shown value can be corrected. Each correction is a new attributed row in `context.source_metadata_correction` (append-only; supersedes exactly the newest revision of that field on that file; "withdraw" is a revision too). The recorded value is never written. Engine `POST .../metadata/corrections`; SQL `scripts/2026-09-26-source-metadata-correction-overlay.sql`, the same block in the bootstrap snapshot.

**Context review (§2).** In the message detail panel: To, About, About the child (yes / no / unsure), Relevant, saved as one attributed revision in `context.record_context_review_revision`; Foreshadowing is a separate checkbox writing `context.record_foreshadowing_flag` (horizon `hindsight` on every row). Engine and BFF routes under `.../messages/{message_id}/context-review` and `.../foreshadowing`.

**Knowledge horizon.** Reads default to `as_lived`. PostgreSQL: the flag is its own table, never a column on the record or message; the views return no flag row when `app.horizon` is set; `context_reader` cannot read the base table; the engine's as-lived read plan contains no statement that names the flag table (unit-tested). The HTTP answers (engine and BFF) omit the `foreshadowing` member on an as-lived read even if a store returned it. Weaviate and SurrealDB: the flag is not projected to either; `modules/engine/contextreview/horizon_tripwire_test.go` fails when any code outside a named allowlist names the flag relations, so a future projection must be added there deliberately, after its own pre-filter.

**Extractor lookup (owner rules "look up the feature", "engines are one shared toolkit").** The engine already has the Activity for this, `ExtractEmbeddedMetadata`, but its extractor is a no-op, so no run has recorded EXIF, document properties or media tags; the screen shows one small flag, "embedded metadata not read". The tool-runtime image has pillow, pypdf, pikepdf, ffmpeg, poppler and tesseract but no exiftool. Two exiftool readers already exist outside this repository: the legal workdesk (`/v1/documents:metadata`, with original-time resolution and Takeout sidecar input) and Intake (`image_facts`, `original_time`). No new extractor was written.

**Owner decision needed: where the one embedded-metadata reader lives.**
- **A (default).** One exiftool reader in Probata's tool-runtime, reached through the tool gateway by locator (D-132); `ExtractEmbeddedMetadata` calls it and records rows; an on-demand read for runs already ingested and for attachments; the legal workdesk and Intake then call the same tool and retire their copies.
- **B.** Probata calls the legal workdesk's `/v1/documents:metadata`. It needs a service credential: the Workbench reaches legal-api on the same host through a Docker bridge, which legal-api's tailnet-owner lane does not admit.
- **C.** Keep one reader per surface (today's state: two copies).

**Open question for the owner.** To / about / relevant are recorded now, in hindsight. They are treated as ordinary corrections, visible to the as-lived view, because only foreshadowing was named hindsight-only. Should an ignorant-agent walk see owner review overlays at all?
