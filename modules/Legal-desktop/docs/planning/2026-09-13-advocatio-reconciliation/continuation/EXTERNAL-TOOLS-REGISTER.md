---
title: Outside applications and libraries to integrate — working register
date: 2026-09-20
tags: [advocatio, integration, external-tools, libreoffice, pdf, ocr, register]
status: working
---

# Outside applications and libraries to integrate — working register

> _Byline: Claude Code · Fable 5.1 · 2026-09-20_

Owner order, 2026-09-20 22:55: find the discussions about outside tools and
libraries, list everything that is supposed to be integrated, start integrating.

This file is the status column only. The requirements, rationale and provenance
stay where they were recorded; nothing is restated here.

## Where the discussions are

| Record | What it holds |
|---|---|
| `docs/planning/artifacts/HANDOFF — Category 1…7` (2026-08-17) | The original owner-agreed feature sets. Category 5 is the document/PDF/office list. |
| `docs/reports/CAT1…CAT7-*.md` (2026-08-18) | One researched pick per capability, with licenses. |
| `../inputs/handoffs/HANDOFF-RECOVERY.md` + `rediscovery.json` (2026-09-13) | 35 recovered requirements (`DOC-01…12` are the document tools) and 20 named outside integrations. |
| `../TOOL-CATALOG.md`, `../REDISCUSSION.md` | Catalog and the items still needing a disposition. |
| Codex rollout 2026-09-13 04:11, owner 09:28 | "We are going to be bringing in like LibreOffice… handoffs… a bunch of different resources" — the only owner-typed mention in the Codex lane; it points at the handoffs above. |

Lanes searched: Docstore (advocatio, intake), repository, Codex sessions, Claude
session logs. Not searched: claude.ai web conversations (not on disk).

## Document, PDF and office tools (Category 5)

State is what the code and the running deployment show, not what a plan says.

| ID | Capability | Named tools (pick first) | State 2026-09-20 |
|---|---|---|---|
| DOC-02 | Office ⇄ PDF conversion | **LibreOffice headless** as `legal-renderer` sidecar; unoconv | **Integrated this session** — see receipt below. |
| DOC-09 | Bates / exhibit stamping | pypdf / pikepdf overlay | In code before today: `services/bates.py`, `POST /v1/bates:stamp`. |
| DOC-08 | True redaction | **pikepdf** removal; **Microsoft Presidio** detection | Removal in code (`services/redaction.py`). Presidio detection absent. |
| — | DOCX assembly | **python-docx** | In code (`services/docx_export.py`). |
| DOC-04 | Programmatic PDF form fill | **pypdf**; pdf-lib; pdftk-java | pypdf installed; no fill service or route. |
| DOC-05 | Markdown / JSON → PDF | **WeasyPrint** + **Jinja2**; md-to-pdf; Pandoc; @react-pdf/renderer | Absent. |
| DOC-06 | PDF → Markdown, with OCR | **Tesseract** (live); Docling; marker; unstructured | **Image OCR integrated 2026-09-21** (Tesseract 5.5). Still absent: PDF → Markdown/layout extraction (Docling), OCR of scanned PDFs, and feeding OCR text into Intake's vector index. |
| DOC-10 | Metadata read / scrub / report | **exiftool** (read, every file type), **pikepdf** (PDF scrub) | **Integrated 2026-09-21** — images, video, audio, office, PDF; original-time resolution and series ordering. Still absent: scrub for images, hash check against the evidence platform, printable report. |
| DOC-03 | PDF viewing + AcroForm fill in browser | **pdf.js**; react-pdf-viewer; pdfme | Absent from `web/package.json`. |
| DOC-01 | In-browser editor with tracked changes | TipTap (v1 pick); **Collabora Online**; **OnlyOffice Docs**; Lexical | Absent. Editor choice is REDISCUSSION decision 1 — owner call. |
| DOC-07 | Treatise / EPUB export | Calibre `ebook-convert`; Pandoc | Absent; recorded as later. |
| DOC-11 | Michigan / Genesee / FOC form library | Official SCAO PDFs | Absent. |
| DOC-12 | Work-product hashes | sha256 into work-product versions | Partly: render results now return a content hash. |

## Other named outside integrations (from `rediscovery.json`)

Timesketch fork · react-calendar-timeline · vis-timeline/vis-data · Evidence.dev ·
Claude Code history viewer · NeoDash · Surrealist · CopilotKit / AG-UI · React Flow ·
Kepler.gl / Leaflet · legal-mcp · legal-terminal (genego-io) · legal-terminal
(JuriSupport) · LexRAG · LIGHT-2 · THEMIS · Suna / Kortix · OpenLegalDataSkills ·
claude-power-skills · custody-guide bundle.
Also in the catalog: Family Law Toolkit (383 capabilities), CourtListener MCP,
eyecite (already wired: `services/eyecite_adapter.py`), FreeEed, KAPE, n8n,
APScheduler (already wired).

None of these was touched today. Their state is whatever `rediscovery.json` records.

## Order of work

1. DOC-02 LibreOffice renderer — done today.
2. DOC-10 PDF metadata — done 2026-09-21. DOC-05 Markdown/JSON → PDF is next.
3. DOC-04 form fill + DOC-11 forms, then DOC-03 viewer in the web client.
4. DOC-06 extraction/OCR after the ownership line with Intake is confirmed.
5. DOC-01 editor after the owner picks among the options.

## Choice made today, open to veto

Recorded as: **the `legal-renderer` sidecar is Gotenberg 8.37.0**, an off-the-shelf
container that puts an HTTP API in front of headless LibreOffice. Reason: the
alternative was a hand-written image plus a hand-written HTTP wrapper around
`soffice`. Situation-specific; swapping it is one compose service and one URL.

## Receipt — DOC-02

Verified live 2026-09-21 ~04:00 UTC against `https://legal.tilapia-skilift.ts.net` (commits `97e1370`, `169195d`):

- `legal-renderer` (Gotenberg 8.37.0, no published port) deployed with the app through Coolify.
- A synthetic DOCX posted to `/api/legal/v1/documents:convert` through the real same-origin bridge returned 200; the PDF fetched from `/v1/documents/renders/…` had the same sha256 the API reported, producer `LibreOffice 26.8.0.3`, and the marker text and table content were extractable.
- An unsupported file returned 400. The three probe files were removed from the server afterwards.
- **Bug found and fixed on the way:** the bridge signs percent-encoded paths while the API verified the decoded path, so every colon route (`/v1/bates:stamp`, `/v1/gateway:invoke`, `/v1/events:apply`, …) returned 401 through the bridge. Fixed in `api/auth.py` with a regression test.
- Not done: no web UI calls the route yet; local `tests/test_sqlite_packaging.py` cannot run on this desktop venv (no `pip` module) — unrelated to this change.

## ContextForge registration (owner rule, 2026-09-20 23:07: "every single tool gets registered in ContextForge")

~~Live ContextForge state 2026-09-21: 150 tools, all served through MCP gateways; none from `legal-api`; zero REST-type tools. The conversion route is **not yet registered**. Open owner decision — how workdesk tools are exposed (A: one MCP face + one gateway; B: per-route REST tools).~~

**Decided and done 2026-09-21 01:40 EDT — owner chose A.** Verified live (commit `fd48682`):

- `legal-api` serves an MCP endpoint at `/mcp/` (`api/legal_workspace/api/mcp_face.py`, fastmcp 4, stateless JSON) behind the same auth middleware as the HTTP API.
- New auth lane: bearer `LEGAL_MCP_GATEWAY_TOKEN` (off when unset; value in `~/.secrets/legal-workspace.env` and the Coolify app env, never in git).
- ContextForge gateway `advocatio` → `http://legal-api:8010/mcp/` (service name over the `probata` network), team visibility, status active, reachable.
- Tool `advocatio-convert-office-document-to-pdf` listed in ContextForge and called **through ContextForge's own `/mcp`** with a synthetic DOCX: LibreOffice PDF returned, stored file's sha256 matched the tool result; probe files removed.
- fastapi-mcp was tried first and dropped: no release since 2025-07, breaks on mcp 2.x, and recurses forever on the self-referential issue-tree schema.
- Observed: LibreOffice PDFs are **not byte-deterministic** (same DOCX → different sha256 each run; embedded timestamps). DOC-02 asks for deterministic rendering — open item.

**How every later tool gets registered:** add a `@mcp.tool` function in `mcp_face.py` over the service; after deploy, ContextForge picks it up on gateway refresh. No per-tool registration step.

## Receipt — DOC-10 (PDF metadata)

Verified live 2026-09-21 ~06:05 UTC (commit `f5e8fb3`), called **through ContextForge** (`advocatio-read-pdf-metadata`, `advocatio-scrub-pdf-metadata`; the gateway refresh added both tools automatically):

- Read of a synthetic LibreOffice PDF returned 10 authored fields (Author, Creator, CreatorTool, Producer, Create/Modify/Metadata dates, Date, Format, Language).
- Scrub removed 9 of them; exiftool re-read of the output shows only `Language` (the catalog's document-language tag, not authorship). Page text intact. Probe output removed from the server.
- HTTP twins: `POST /v1/documents:metadata`, `POST /v1/documents:scrub-metadata`. Uploads are processed in a temp folder and not kept; only the scrubbed copy is stored under `renders/`.

> **Correction 2026-09-21 (owner):** the first DOC-10 cut covered PDFs only. The list already said exiftool for any
> file type, and images — not PDFs — are the evidence. The PDF-only receipt above is superseded by the one below.

## Receipt — DOC-10 for every file type, original time, device (2026-09-21 ~06:20 UTC, commit `ee67000`)

All called **through ContextForge** on synthetic files; nothing kept on the server.

- `advocatio-read-file-metadata`: exiftool 13.25 installed in the `legal-api` image. A JPEG with EXIF returned capture
  time with offset, modify time, device make/model, editing software, signed GPS (43.0125, -83.6875), 38 fields, and
  a sha256 equal to the local file's.
- `original_time` (owner requirement: screenshots often show no date; order depends on the original timestamp):
  candidates come from EXIF capture time, a Google Takeout sidecar's `photoTakenTime`, embedded XMP/PNG/container
  creation times, device-generated filename stamps (`Screenshot_20240312-141502`, `PXL_…`, `IMG_…`), epoch-style
  names (`received_1710252000123`) and date-only names (`IMG-20240312-WA0007`, low confidence). Each carries source,
  confidence and timezone basis; disagreement over a day sets `conflict`. **Filesystem dates are never candidates**
  (uploads carry the upload's time; the corpus restore re-stamped files in batches).
- `advocatio-order-images-by-original-time`: four shots came back in true order; `IMG_4821.PNG` (no recoverable
  time) was reported as unresolved; mixed timezone basis was flagged.
- Summary also carries device serial, lens, host computer, unique image id and user comment when present.
- **Limit to know:** an iOS screenshot named `IMG_####.PNG` with no EXIF has no recoverable time in the file. Its
  time has to come from the catalog's source occurrence or a Takeout sidecar — pass the sidecar text to the tool.

## Receipt — DOC-06 image OCR (2026-09-21 ~06:12 UTC, commit `e75804c`)

- `advocatio-ocr-image` / `POST /v1/documents:ocr`: Tesseract 5.5.0 in the `legal-api` image. A synthetic chat
  screenshot came back word-for-word (29 words, mean confidence 95.5) with a pixel box and confidence per line.
  Layout modes: auto, block, sparse. Output is marked a derivative, never court-safe.

## Open: making screenshot text searchable (owner, 2026-09-21 02:03)

Intake's CocoIndex pipeline does not index images today: `backend/src/casebible_index/config.py` has no image
extensions and `extractors.py` says "OCR is intentionally deferred". Intake's own plan lists it as CBX-P8-003.
Two steps: (1) an OCR branch in Intake's extractor so screenshots become indexable text in Weaviate, kept apart from
native-export text; (2) an index run over the screenshots in B2 — a billable, long job that needs the owner's go
with counts from the catalog first.

**Scope from the catalog (read-only `catalog_reconcile.occurrences`, 2026-09-21 ~06:40 UTC):** 677,730 image
occurrence rows, 416.0 GB (jpg 388,657 · png 202,290 · heic 33,231 · gif 24,325 · jpeg 16,085 · webp 13,074).
Rows whose path names a screenshot: 26,208 (17.2 GB) → **14,948 unique files by hash, 9.04 GB**, plus 758 rows with
no hash. Sources: D-Backup 11,430 · OneDrive 10,532 · F-Disk-Drill 2,673 · gdrive/salemnet 1,118 · F-case 455.
Every one of those rows carries a `recorded_modtime` and `source_metadata` in the catalog — a further original-time
candidate for files like `IMG_####.PNG` that carry none themselves (lower confidence: the restore re-stamped batches).
Screenshots not named as such are not in this count.

## Receipt — buttons (2026-09-21 ~10:00 UTC, commit `6c937bd`)

Owner order 05:52 EDT. Sidebar → Contracts → **File tools** (`/file-tools`, `web/src/components/FileTools.tsx`):
Read metadata · Put images in time order · Read text (OCR, with layout choice) · Convert to PDF · Remove PDF metadata.
Takeout `.json` sidecars picked beside an image are paired and sent as an original-time source. Checked in a real
browser on the live site with three synthetic screenshots: time-order put them in the right order and flagged the
one with no recoverable time; OCR returned each one's text word for word; no console errors. Convert and Remove
buttons were not clicked in the browser (their routes were verified earlier the same night). Still without
buttons: Bates stamp and redaction (older routes; their outputs have no download route yet).

The Intake image-index design that follows from the owner's 05:52 direction (Weaviate MaxSim + SurrealDB, hosted
embedder, Tesseract fallback) lives in `Consignatio/Intake/docs/PROPOSAL-2026-09-21-IMAGE-INDEX.md`.

