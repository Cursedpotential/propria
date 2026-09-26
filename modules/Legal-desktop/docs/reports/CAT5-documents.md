# Category 5 — Contract & document analysis

> _Byline: Grok · grok-4.6 · 2026-08-18_
> Source: `Legal-desktop/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md`

## A. In-app workspace — Collabora vs OnlyOffice vs custom

| | Collabora Online (CODE) | OnlyOffice Docs CE | TipTap / Lexical |
|---|---|---|---|
| Track changes | Native (LO) | Native | Build yourself |
| WOPI | Yes | Yes | N/A |
| Extra Docker | Yes (heavy) | Yes (heavy) | No |
| Coolify tax | Another 8080-collision risk | Same | None |
| Single-user value | High fidelity, high ops | Same | Enough for MD drafts |

**Recommendation: Markdown-first custom editor (TipTap) for v1 Drafting
Studio.** Collabora/OnlyOffice are real office suites meant for
Nextcloud-style multi-user editing. That ops cost is Type 1 on this
fleet (new service, WOPI, RAM). The custody packet drafts and
templates are already Markdown.

**See-the-work (2026-08-18):** Chrome DevTools MCP sees the Next.js
shell (DOM/a11y + screenshot). Owner PDFs open same-origin at `/doc`
so DevTools/F1 can see them. Collabora CODE / OnlyOffice and the
LibreOffice `legal-renderer` sidecar remain the **second** surface
when Type 1 Docker is available — a WOPI iframe is often cross-origin,
so DevTools on the parent tab will not read the document canvas.
~~That sidecar is still HOLD.~~ **Corrected 2026-09-21 (Claude Code · Fable 5.1): the
LibreOffice `legal-renderer` sidecar is live on ovh-app (Gotenberg 8.37.0); receipt in
`docs/planning/2026-09-13-advocatio-reconciliation/continuation/EXTERNAL-TOOLS-REGISTER.md`.
Collabora/OnlyOffice remain undecided.** Do not run it on this
desktop (no Docker CLI).

Phase 2 option: OnlyOffice *if* native DOCX redlines become
unavoidable. Do not run both.

PDF view/fill: **pdf.js** for viewing
([Mozilla](https://mozilla.github.io/pdf.js/)); **pdf-lib** (JS) or
**pypdf** (Python) for AcroForm fill. `pdfme` only if we design new
fillable templates. `react-pdf-viewer` is a wrapper — optional.

Vercel AI SDK (handoff note): already locked with Next in Cat 2 /
build guide. Does not change the editor choice.

## B. Conversion

| Job | Pick | License | Why |
|---|---|---|---|
| Programmatic PDF fill | `pypdf` on `legal-api` | BSD | Same language as services; [pypdf](https://pypdf.readthedocs.io/) |
| Markdown → PDF | WeasyPrint | BSD | No Chromium; [weasyprint.org](https://weasyprint.org/) |
| PDF → Markdown | Docling + OCR extra | MIT | Layout + optional OCR; [docling](https://github.com/docling-project/docling) |
| JSON → PDF | Jinja2 → WeasyPrint | BSD | Motion cover sheets, auth reports |
| DOCX assemble | `python-docx` | MIT | Already in legal-mcp |
| DOCX ⇄ PDF | LibreOffice sidecar `legal-renderer` | MPL | `soffice --headless --convert-to` |
| Treatise EPUB | Calibre CLI later | GPL | Not the interactive workspace |

Scanned exhibits: Docling OCR or Tesseract. Raw photo *ingest* stays
Agno.

## C. Real redaction

Detect with Microsoft Presidio
([docs](https://microsoft.github.io/presidio/)) plus
custody-guide `redaction_helper.py` patterns.
Remove with **pikepdf** content-stream edit — not a black box.
[pikepdf](https://pikepdf.readthedocs.io/) (MPL).

## D. Bates / exhibit stamp

No dedicated library needed. Thin wrapper on `pypdf` page overlay:
`{prefix}-{n:06d}`. Prefix from the Matter display name. Spec lives
next to custody-guide `exhibit_indexer.py`.

## E. Metadata + authentication report

`exiftool` for read + scrub (one binary, two modes).
Validate evidence hashes via Agno `POST /v1/verify/{sha256}`
(Agno interface analysis). Owner-produced files hash into
`legal_work_product.work_product_version.content_hash`.

### Legal-research hold (not decided here)

FRE 902(13)–(14) is federal. Michigan analogue is **MRE 902**
(self-authentication). Whether a generated “Certificate of
Authentication” actually supports a Genesee filing is a **separate
legal-research pass** against the archived
`michigan-rules-of-evidence` primary in the custody packet. Do not
print a form that implies the rule is settled.

## F. Michigan / Genesee forms

Official SCAO forms: [courts.michigan.gov forms](https://www.courts.michigan.gov/SCAO-forms/).
Genesee local: clerk / FOC pages — treat as PROVISIONAL until the
owner confirms the current packet (custody-guide rule).
Preload the forms the custody packet already names (motion, FOC
objection, proof of service, fee waiver). Link the official PDF;
do not ship a competing form.

## G. Self-custody hashing

**Parallel lightweight table in legal schemas**, not Agno custody
tables. Agno custody is for ingested evidence. Owner-produced drafts
are work product. Cross-link by hash if the owner later files a
copy into Agno as an exhibit.

## Module tree

```
api/legal_workspace/services/documents/
  editor_export.py      # MD → DOCX/PDF via renderer
  fill_pdf.py
  redact.py
  bates.py
  metadata.py
deploy/legal-renderer/  # LibreOffice only
web/src/components/editor/
```

## Non-goals

No raw evidence ingestion. No multi-user collaborative editing
infra. No Collabora in Phase 0/1.
