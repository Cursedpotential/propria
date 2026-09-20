---
title: "OpenDataLoader PDF — tool reference (owner: add as a tool, look into)"
date: 2026-09-14
tags: [tool, pdf, extraction, ocr, rag, intake, docstore, reference, owner-directive]
---

# OpenDataLoader PDF — tool reference

> _Byline: Claude Code · Fable 5.1 · 2026-09-14 23:40 EDT — owner 23:39: "add opendataloader as a tool, look into." Facts below were read from the GitHub repo, opendataloader.org docs and PyPI on 2026-09-14; verify before relying on version-specific claims._

## What it is
- **PDF → AI-ready structured data**: Markdown, JSON with bounding boxes and semantic types (heading/paragraph/table/list/image/caption, page number, font), HTML, plain text, annotated PDF, and Tagged PDF (auto-tagging for accessibility).
- Repo `github.com/opendataloader-project/opendataloader-pdf`, homepage `opendataloader.org`. **Apache-2.0** core. Backed by Hancom with Dual Lab (veraPDF developers). Created 2025-05; ~28.9k stars, ~2.76k forks, ~80 open issues; PyPI `opendataloader-pdf` (v2.3.0 seen), npm `@opendataloader/pdf`, Maven `org.opendataloader:opendataloader-pdf-core`; LangChain integration `langchain-opendataloader-pdf`.
- **Runtime**: Java 11+ engine; Python 3.10+ / Node 20+ wrappers spawn a JVM per `convert()` call — batch all files in one call. No GPU needed. Runs fully local.
- **Modes**: *Fast* (deterministic, rule-based, XY-Cut++ reading order, ~60+ pages/s CPU) and *Hybrid* (`pip install "opendataloader-pdf[hybrid]"`, run `opendataloader-pdf-hybrid --port 5002 [--force-ocr --ocr-lang ..]`, client `--hybrid docling-fast [--hybrid-mode full]`) which routes complex pages (borderless tables, scans/OCR, formulas, chart descriptions) to a Docling-based backend; other OCR engines selectable (`--ocr-engine tesseract|rapidocr`, per issue #460). Known gotcha: `auto` triage does not detect image-only scanned pages — use `hybrid_mode='full'`; OCR on CPU can OOM on huge scans (split page ranges).
- **Not covered**: Word/Excel/PowerPoint (keep python-docx etc.); embedded file metadata (EXIF/XMP) is not its job — that stays with ExifTool.
- Enterprise add-ons (PDF/UA export, accessibility studio) are paid; not needed.

## Where it fits in Propria
- **Intake backend extraction** (`Consignatio/Intake/backend/src/casebible_index/extractors.py`, `.pdf` → today `pypdf` text layer only): replace/augment with OpenDataLoader JSON (reading order, tables, bounding boxes for citations) and hybrid OCR for scanned court filings and Disk Drill recoveries. Output JSON + Markdown become sidecars keyed by content hash (owner's sidecar requirement, 2026-09-14).
- **Case Bible / legal-desktop**: court filings, FOC forms, benchbooks — tables and reading order matter for citation; Tagged-PDF output is a bonus for accessibility.
- **Docstore**: not needed (markdown-native), except for PDFs that get converted into docs.
- ~~**Alternatives already known**: MinerU, Docling, Marker, Unstructured, Apache Tika.~~ **Corrected 2026-09-14 23:45 EDT (owner: "we did test a bunch of PDF libraries, like six"):** the prior test is `projects/consignatio/repair-tool-kit/FINDINGS.md` (2026-09-11, Codex lane): a seven-reader bake-off on a real evidence transcript — poppler `pdftotext`, `pypdf` and pdfium (`pypdfium2`) decoded the ZapfDingbats emoji glyph (33/33 non-ASCII recovered); pdf.js, pdfminer.six and pdfplumber returned the raw byte `n` (kept as raw-byte reporters); mutool/MuPDF mapped it to `I` and is **actively rejected** for text extraction. That file rules until re-tested. OpenDataLoader is therefore evaluated on a different axis (reading order, tables, bounding boxes, OCR) and must ALSO pass the same ZapfDingbats bake-off on the same transcript before it is used as a text reader. Docling is already the planned complex-PDF route in Intake (MASTER-TODO Phase 7, DOCUMENT-HANDLING-AND-DEDUPE.md); OpenDataLoader's hybrid mode is Docling-backed, so it may be the packaging of that plan rather than a competitor.

## Next step
Bounded live test on ovh-files (container, Java 17 + Python 3.12): run fast and hybrid modes over the gate sample PDFs and 10 real court PDFs from B2; compare against pypdf output; record speed, table fidelity, OCR on a scanned page, JSON size. Receipt to `Consignatio/Intake/docs/OPENDATALOADER-SPIKE-2026-09-14.md`.

## Bake-off result (2026-09-15 00:26 EDT)
Run on the exact FINDINGS.md transcript (sha256 match) inside the spike container: **pypdf control = `■` U+25A0, 33/33** (reproduces FINDINGS.md). **OpenDataLoader fast mode (text and markdown) = the glyph is absent: no codepoint, 0/33.** ZapfDingbats never appears in its JSON font list across all 24 pages; the run ends at "pooped" with no placeholder. That is a fourth category below mutool's wrong-but-present: **silent, traceless omission** of symbol-font runs (likely treated as decorative). Consequence: OpenDataLoader is **not a text reader for evidence PDFs**; it may serve as a layout/table/OCR engine only, with the text layer taken from a passing reader (poppler/pypdf/pdfium). Untested follow-up: `--content-safety-off`. Details: `Consignatio/Intake/docs/OPENDATALOADER-SPIKE-2026-09-14.md` § Bake-off parity with FINDINGS.md.
