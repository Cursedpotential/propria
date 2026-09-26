# Comprehensive Plan for Custody Case Document (folder)
> _Byline: Claude Code · Sonnet · 2026-07-11_

**This is a WORK-PRODUCT, not a chat transcript.** It is a single Gemini-authored
planning document (an evidence-organization plan for the custody case), not a
conversation export. It happens to survive on disk as a **folder of raw OOXML
fragments** rather than a normal `.docx` file — see Parsing Notes.

## Snapshot
- Source: a single Google Docs/Word document titled "Comprehensive Plan for Custody
  Case Document Management," authored collaboratively with Gemini (the doc refers to
  itself in third person: "for me (Gemini, in assisting with scripts and analysis)").
- Size: 172 KB folder total (`word/document.xml` 76,949 bytes + `word/numbering.xml`
  91,692 bytes). No other files/subfolders exist under it.
- Format: **not a chat, not a valid `.docx` package** — it's the unzipped `word/`
  subtree of an OOXML document with only two of the normal ~15 parts present
  (`document.xml`, `numbering.xml`). Missing `[Content_Types].xml`, `_rels/`,
  `docProps/core.xml` (so no author/created/modified timestamps recoverable), styles,
  fonts, theme. Word/python-docx cannot open this as-is; it was parsed directly by
  walking `document.xml`'s body XML.
- Era/date: undated (no `docProps/core.xml` survived). Content is plan/scaffolding,
  not evidence, so no in-body incident dates either.
- "Turns": none — single continuous document, one author's plan (with Gemini credited
  as co-drafter), organized under 5 numbered top-level headings.

## Structure & format
- Standard Word heading styles (`Heading1` title, `Heading2` for the 5 major
  sections) plus multi-level numbered/bulleted lists (`w:numPr` / `numId` / `ilvl`)
  for procedural steps — up to 4 levels deep (top-level steps → sub-steps →
  sub-sub-steps).
- One large **ASCII folder-tree diagram** is embedded as a single run of plain text
  inside one paragraph (uses box-drawing-style characters `├──`, `│`, `└──` and folder
  emoji `📁`/`📄`), not a real Word list or table — it's literal text art describing
  the target Google Drive folder structure.
- No tables, no images, no tracked changes, no comments in the surviving parts.
- **PARSING NOTES (folder-of-fragments case):**
  - This is what remains after someone extracted a `.docx` zip archive and only kept
    `word/document.xml` + `word/numbering.xml` (likely a partial/failed extraction,
    or a deliberate "just grab the text parts" copy) — the folder is literally a
    docx's internal ZIP structure exploded on disk, missing the ZIP container itself
    and most sibling parts.
  - A parser must NOT assume `.docx`-shaped input is always a valid zip; needs a
    fallback path that walks a bare `word/document.xml` with `ElementTree` under the
    `http://schemas.openxmlformats.org/wordprocessingml/2006/main` namespace,
    pulling text from `w:t` runs inside `w:p` paragraphs (and `w:tbl`/`w:tr`/`w:tc`
    for tables, unused here).
  - No metadata recoverable (no author/date) — a parser should flag such fragments as
    **metadata-incomplete** rather than fail silently or fabricate a timestamp.
  - `numbering.xml` carries only list-formatting definitions (abstractNum/num level
    definitions, bullet glyphs, fonts) — zero narrative content; safe to ignore for
    text extraction but needed if list markers must be rendered faithfully.
  - The embedded ASCII folder-tree is a single unbroken text run — a naive line-based
    parser will see it as one giant paragraph; if the tree structure itself matters
    downstream, it needs its own indent-aware sub-parser rather than generic sentence
    splitting.

## Section-by-section breakdown (in order)
1. **Title + intro** — "Comprehensive Plan for Custody Case Document Management."
   States purpose: consolidate all steps/strategies discussed for organizing
   sensitive court-case communications in Google Drive; a reference doc for the
   owner, for Gemini (as the AI assisting with scripts/analysis), and for anyone
   else who needs to understand the system later.
2. **§1 Project Goal & Overview** — Case framed as "[Your Name] vs [Mother's
   Name]" (placeholders, not filled in). Primary purpose: systematically collect,
   organize, analyze, and cross-reference evidence of relationship/co-parenting
   dynamics — specifically communication inconsistencies, infidelity evidence, and
   behavior patterns — for custody court proceedings. Lists key tools: Google Drive,
   OCR (Perplexity / Drive OCR / dedicated software), Google Apps Script (automation),
   Gemini (script generation, parsing, summarizing, cross-referencing), and a
   spreadsheet program for structuring parsed data into CSVs.
3. **§2 Google Drive Folder Structure** — Full proposed hierarchy as ASCII tree:
   `00 - INGEST` (unsorted landing zone) → `01 - Direct Communications with [Mother]`
   (texts/messenger/email/call logs, each bucketed by chronological range) →
   `02 - Communications with Third Parties` (`02.01 Infidelity Evidence`,
   `02.02 Third Party Communications`, `02.03 Social Media Posts`) →
   `03 - Child-Related Communications & Evidence` (co-parenting app logs, school/
   medical records, well-being evidence) → `04 - Chronology, Journal & Analysis`
   (personal journal, key-event timeline, inconsistency log, AI analysis reports,
   potential exhibits) → `05 - Financial Documents` → `06 - Legal Documents &
   Discovery` (filings, discovery, attorney correspondence, case law) →
   `07 - Exhibits for Court`.
4. **§3 Workflow Plan: Ingest, Process, Organize, Analyze** — 5-step numbered
   workflow: (1) Ingest raw screenshots/OCR output into `00 - INGEST` with temp
   names; (2) Process & structure via a planned Google Apps Script that prompts for
   metadata (date, platform, sender, recipient, keywords) then renames/moves files,
   plus manual OCR cleanup and CSV structuring (Date/Time/Sender/Recipient/Message
   Content columns); (3) Ongoing organization — maintain journal, event timeline,
   and inconsistency log (explicitly: "systematic record of contradictory statements
   made by the mother... what she told you vs. what she told others," each entry
   linked to supporting evidence); (4) AI-powered analysis via Gemini — summarize,
   find recurring themes/patterns, cross-reference across files, draft evidence
   summaries; (5) Final court prep — collaborate with legal counsel to select/
   number exhibits as PDFs.
5. **§4 Standardized Naming Conventions** — Format spec:
   `YYYY-MM-DD_Platform_Sender_Recipient_KeywordsOrTopic.FileExtension`, with a
   controlled vocabulary for `Platform` (SMS, WhatsApp, FBMessenger, InstaDM,
   Email, CallLog, Voicemail, OurFamilyWiz, Doc, PDF, Image) and example keyword
   tags including `Infidelity`, `ChildCustodyIssue`, `FinancialDispute`, `Threat`,
   `Agreement`, `Inconsistency`, `Gaslighting`, `Manipulation`. Gives 6 concrete
   filename examples (all with placeholder names).
6. **§5 Next Steps for Automation & Implementation** — Concrete next actions:
   manually create the `00 - INGEST` folder, open the Google Apps Script editor,
   name the project "Custody Case File Organizer," await Gemini-provided starter
   script to auto-build the folder tree or list files in `00 - INGEST` for
   renaming/moving.
7. **Closing line** — Encouragement/support note: "this is an emotionally
   demanding process... I am here to support you through each step."

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity(who) | placeholder-only | "[Your Name]" / "[Mother's Name]" never filled in anywhere in the surviving text — genuinely anonymized-by-template, not owner-redacted |
| entities | minimal | Generic tool/platform entities only: Google Drive, Google Apps Script, Gemini, Google Sheets, OurFamilyWizard/TalkingParents (named as an example co-parenting app) |
| relationships | implied | Owner ↔ "Mother" (co-parent, adversary in custody matter) ↔ unnamed third parties (infidelity-evidence subjects) — structural only, no names |
| timeline/events | none in-body | Doc is a forward-looking system design, not an incident log; the `04.02 Key Event Timeline` folder it proposes is itself empty (future work) |
| life-history | none | — |
| legal-strategy | yes | Core purpose IS legal-strategy: evidence taxonomy explicitly organized around "infidelity," "inconsistency," "gaslighting," "manipulation," "threat" tags, and an "Inconsistency Log" designed to catch contradictory statements for cross-examination/court use |
| legal-artifacts | yes (meta) | The document itself is a work-product (an evidence-management SOP/plan), not a filed legal document; it also specifies a target folder for finished `Exhibits for Court` and references a companion `README_ProjectOverview_and_Nomenclature.docx` (not present in this sample) |
| mood/sentiment (owner-self, low-pri) | yes | Closing line acknowledges the process is "emotionally demanding" and offers support — mild but explicit |
| psychiatric | none | — |
| code/app-dev/plans | yes | Central facet — this is fundamentally an app-dev/automation plan (Google Apps Script for file ingest/rename/move, planned CSV parsing pipeline) layered on top of the evidence-org plan |
| work | none | — |

## Notable content
- **Evidence taxonomy** worth reusing for the platform's own facet/tag design:
  Infidelity, ChildCustodyIssue, FinancialDispute, Threat, Agreement, Inconsistency,
  Gaslighting, Manipulation — these are the owner's own self-authored evidence-tag
  vocabulary from mid-2026, independently converging with the platform's planned
  facet map (legal-strategy, entities, relationships).
  - Confirms the owner had, in a prior tool (Gemini/Google Drive), designed almost
    exactly the evidence-organization system this platform is now meant to
    supersede/automate — directly relevant prior art for the extraction-lane design.
- References an **"Inconsistency Log"** concept (04.03) — a structured record
  cross-referencing what the co-parent told the owner vs. what she told third
  parties, each entry linked to source evidence file paths. This is a reusable
  pattern for the platform's relationship/timeline facets.
- References a **planned companion document** not present in this sample:
  `README_ProjectOverview_and_Nomenclature.docx`.
- No named individuals, no dates, no case number, no court — all identity fields
  are unfilled placeholders in the source text itself.

## Sensitivity
- LOW-MODERATE. No explicit abuse/DV/psychiatric narrative content survives in this
  fragment, but the taxonomy vocabulary itself (Infidelity, Threat, Gaslighting,
  Manipulation) signals the underlying case involves alleged infidelity and coercive/
  manipulative behavior by the co-parent — LABEL as custody-case strategy/evidence-
  management planning material, adjacent to abuse-pattern documentation. Placeholder
  names mean no third-party PII is exposed in this file itself.
