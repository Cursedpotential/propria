# HANDOFF — Legal OS Category 5: Contract & Document Analysis — Implementation Research

Status: rough feature set agreed with owner (2026-08-17); this handoff requests in-depth research to scaffold the actual implementation. Research only — no code, no service activation.

## Context

Legal OS is a single-user, single-case, self-represented-litigant practice-management app for a Michigan family-law (custody) matter, sibling package to "Agno MCP Platform" in one monorepo. Agno owns raw evidence intake/custody/hashing (per the existing ownership boundary — do not duplicate ingestion). This category covers everything Legal OS itself does with documents: contract/agreement clause analysis (CTRX), an actual in-app document workspace (drafting/viewing/filling), format conversion in both directions, and a metadata/authentication capability. Owner explicitly wants big/real libraries here, not toy implementations, and wants 2-3 named options per sub-capability rather than a single forced choice, so the deliverable should preserve genuine tradeoffs.

Owner is also planning to use the **Vercel AI SDK** for the AI-facing parts of the app — this is Next.js-native (though it has a framework-agnostic core), and is a live data point for Category 2's still-open frontend-framework decision. Note this connection in the deliverable but do not resolve Category 2 here.

## Feature set agreed (do not re-litigate — build research around this)

### A. In-app document workspace (the actual editing/working surface, not just conversion)
- Real in-browser document editing is required, not just format conversion utilities running in the background.
- Candidates to properly compare, not guess between: **Collabora Online (CODE)** and **OnlyOffice Docs (Community Edition)** — both self-hosted, Dockerized, WOPI-capable office suites with native track-changes/redlining, the same category of tech Nextcloud uses for in-browser editing. A lighter **custom editor (TipTap or Lexical)** remains a fallback option if full office-document fidelity isn't worth the operational overhead of running a separate WOPI-integrated service.
- Native track-changes (from Collabora/OnlyOffice) satisfies the redlining requirement for pulling suggested clause language into an actual draft — no separate redline system needed if either is chosen.
- In-browser PDF viewing + interactive AcroForm filling: candidates are `pdf.js` (the engine, Mozilla), `react-pdf-viewer` (community wrapper with forms/annotation/search plugins — likely most "batteries included"), and `pdfme` (open source, bundles a visual form/template designer if the owner wants to design custom fillable templates, not just fill existing court forms).

### B. Format conversion (both directions)
- **PDF form filling (programmatic):** `pdf-lib` (JS, pure, no external binary), `pypdf` (Python, actively maintained PyPDF2 successor), `pdftk`/`pdftk-java` (CLI, FDF-based, callable from either language)
- **Markdown → PDF:** `md-to-pdf` (Node/Puppeteer), WeasyPrint (Python, HTML/CSS→PDF, no browser engine), Pandoc (CLI, MD→PDF via LaTeX or HTML, also covers many other conversions in one tool)
- **PDF → Markdown / structured extraction:** Docling (Python, IBM open-source, layout-aware, handles tables), `marker` (Python, ML-based, strong on complex layouts), `unstructured` (Python, widely used in RAG pipelines) — **all three assume clean text-native PDFs; this category also requires OCR** (Tesseract, or Docling's built-in OCR pipeline option) for scanned/photographed court documents and exhibits, which is a real gap the owner confirmed needs covering, not an edge case to skip.
- **JSON → PDF/Markdown (templating):** Jinja2 (Python, reuses the MD/HTML→PDF pipeline), Handlebars/Nunjucks (Node equivalent), `@react-pdf/renderer` (React components driven directly by JSON props, skips the Markdown/HTML intermediate step for structured templates like motion cover sheets)
- **Office format conversion (DOCX/ODT ⇄ PDF):** headless LibreOffice (`soffice --headless --convert-to`, run as a Docker sidecar), `unoconv` (thinner CLI wrapper on the same engine), `python-docx` (already a legal-mcp dependency) for direct DOCX assembly with LibreOffice only handling final PDF export
- **Long-form/e-book conversion (distinct use case — the owner's custody treatise/self-help course project, not the interactive workspace):** Calibre's `ebook-convert` CLI (HTML/Markdown/DOCX → EPUB/MOBI/PDF), Pandoc again (can also produce EPUB from Markdown, worth comparing), Calibre's Python conversion API for deeper control if the CLI proves insufficient

### C. Real redaction, not visual-only
- Must permanently remove content, not just draw a black box over it (a well-known legal-tech failure mode where "redacted" text remains extractable). Candidate: `pikepdf` for true content removal. Research current best-practice approach and whether a second library is warranted for the detection side (e.g. Microsoft Presidio for PII detection to flag what needs redacting, paired with `pikepdf` for the actual removal).

### D. Bates numbering / exhibit stamping
- Standard legal practice: sequential exhibit page numbering (e.g. `SMITH-000123`) for filings/discovery organization. Research existing open-source approaches (this is often a thin layer over `pypdf`/`pdf-lib` page-stamping — confirm whether a dedicated library exists or whether this is simple enough to be a small custom utility on top of the PDF libraries already selected in section B).

### E. Metadata: scrub, view, validate, and print (Evidence Authentication Report)
- **`exiftool`** covers both directions — reading comprehensive metadata (any file type) and scrubbing it before filing/sharing. One tool, two modes; don't add a second dependency for the scrub side.
- **Validation:** compare a document's current hash against Agno's recorded custody hash via its `/v1/verify/{sha256}` endpoint (already identified in the Agno interface analysis), and surface any metadata inconsistencies (timestamps, EXIF anomalies).
- **Print/export as a court-presentable exhibit:** feed the extracted metadata + custody chain data into the JSON→PDF templating pipeline (section B) as a "Certificate of Authentication"/"Metadata Report" template.
- Research question: many jurisdictions (federal courts under FRE 902(13)-(14)) allow self-authentication of electronic evidence via a certification describing a hash-verification process. **Confirm Michigan's actual equivalent rule** (likely an MRE 902 analogue) so the generated report's format/content genuinely supports that certification path rather than just looking official. This is a legal-research question, flag clearly if it needs a separate legal-research pass rather than a technical one.

### F. Michigan forms as a template library
- The PDF-form-filling capability (section B) needs real content behind it: pre-loaded actual Genesee County/FOC/circuit court forms relevant to a custody matter, not just generic "can fill any PDF form" capability. Research where official fillable versions of relevant Michigan family-court forms are published (courts.michigan.gov, Genesee County court site) and whether they're already fillable AcroForms or need to be made so.

### G. Self-custody hashing for owner-produced documents
- Documents Legal OS itself produces (drafts, generated reports, filled forms) should get the same SHA-256-hash-and-record treatment Agno applies to ingested evidence, for the owner's own integrity record ("this is exactly what I filed on this date"). Research whether this should write into Agno's existing custody tables (via its API) or a parallel lightweight table in the `legal_os` schema — tie this to the Category 1 persistence decisions already made.

## Explicit boundary — restate, don't cross
Raw, messy evidence intake (photographed exhibits, texts, emails as ingested) stays Agno's job per the existing ownership boundary. This category covers documents the owner produces (drafts, filled forms, generated reports) plus already-reviewed/promoted evidence coming from Agno — not raw ingestion.

## Deliverable

A single markdown report, saved to the workspace, with:
- A clear Collabora-vs-OnlyOffice comparison (section A) with a recommendation, including realistic operational overhead (Docker services, WOPI integration effort) for a single-user deployment
- One recommendation per remaining sub-capability (B through G), each with source citations (real URLs, current library versions, license type)
- A proposed module/service structure sketch (directory tree, not full code) showing how these pieces compose (e.g., is document conversion one shared service both the main app and any Node-based AI SDK layer call into?)
- A clearly-flagged note on the Michigan evidence-authentication rule question (section E) as needing legal research, separate from the technical recommendation
- Explicit non-goals restated: no raw evidence ingestion (stays Agno's), no real-time multi-user collaborative editing infrastructure beyond what Collabora/OnlyOffice provide natively
