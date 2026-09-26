# Original handoff and integration recovery — 2026-09-13

This is a historical requirement recovery, not an implementation or legal-authority audit. All seven original category handoffs were read in full, including research questions and non-goals. The original deep analysis, Bloomberg catalog, feature matrix, and build guide were read directly. Later HANDOFF-ANALYSIS, DONOR-ANALYSIS, and selected category reports are separately identified. Source files were not changed.

## Findings to re-discuss first

1. **The office/document requirement was broader than drafting.** Category 5 required real browser editing, tracked changes, PDF viewing/filling, format conversions, OCR, true redaction, Bates stamps, metadata inspection/scrub/reporting, official forms, and work-product hashes. LibreOffice is explicitly named at original Category 5 line 24. The later CAT5 report selected Markdown-first/TipTap while retaining LibreOffice/office editing as a second surface on HOLD and saying “Do not skip it” (lines 22–32). That is an unresolved delivery obligation, not evidence the requirement disappeared.
2. **Timesketch is a real intended fork.** The user’s “timescale by Google” recollection matches local google/timesketch: accepted ADR-0060 explicitly chooses a maintained personal-case application fork, rejects adapter-only use, and requires governed individual/bulk curation. It lives in the sibling platform at `E:/AI_Workspace/Projects/Propria/Probata/probata/modules/forks/timesketch`. It was absent from the seven Legal-desktop handoffs, which is why reviewing only those would miss it.
3. **Chronology is not one generic timeline widget.** Timesketch has candidate-versus-approved authority and round-trip curation requirements. Separately, `react-calendar-timeline` and `vis-timeline` remain declared development dependencies for lighter primary-surface views. The August 30 direction keeps both options and says neither replaces Timesketch.
4. **Sources and automation were functional requirements.** Config-driven research sources with field mappings/separate tokens, hybrid internal search, bounded multi-hop research, actual background scheduling, editable playbooks, and n8n external integration boundaries deserve explicit decisions.
5. **The dense shell should survive without mnemonic dependence.** Keep split/pin views, grouped labeled navigation, master-detail research, side detail panels, direct cross-links, visible job/review/status tables and panel export. Current owner direction is plain language and mouse plus keyboard; original acronym-primary navigation is superseded.
6. **Later narrowing must be visible.** The feature matrix calls itself a proposal. Later reports classify several external apps as shape-only; the owner now recalls intended co-option/integration. Preserve each identity and reopen that disposition rather than claiming all resources were only inspiration.

## Evidence limits and precedence

Current owner direction and the canonical repository routing take precedence over historical instructions. Original handoffs are dated August 17 and mark research-only status, with capabilities agreed but many engine choices open. The older decision matrix explicitly says it is not a final decision (line 3). August 18 reports are later interpretations, even when they call choices “locked.” Neither plan text nor a listed dependency proves implemented or deployed behavior. All “today’s overlap” labels use the parent-provided rough discussion list and require final reconciliation.

The original Category 6 no-local-inference rule overrides Category 3’s stale local-Ollama routing reference. Category 2 dark-only/cut-repeat/cut-hardware-keys supersedes the earlier matrix; Ctrl+K was reinstated, while TUI remained v2. The guide requires fixed workflows before a generic builder and places external operational integrations in Phase 4. Historical PACER/provider/legal-authentication claims remain unresolved or later-held, never current legal advice or authorization.

## Requirement rediscovery catalog

### DOC-01 — A real in-browser document editor with native tracked changes and clause suggestions entering drafts

- **Original state:** agreed capability; tool choice open.
- **Rationale/resources:** Owner wanted a working document surface and real libraries, not background converters alone. Collabora Online CODE; OnlyOffice Docs CE; TipTap or Lexical fallback.
- **Current-discussion overlap:** Partial overlap: drafting/proposed changes; full office fidelity rediscovered.
- **Re-discuss:** Define whether office-native editing and redlines are required alongside Markdown, and what 'done' means for accepting/rejecting a clause.
- **Later disposition:** Later report recommends TipTap/Markdown v1; office suite and LibreOffice remain a second surface on HOLD, explicitly 'Do not skip it'; OnlyOffice conditional Phase 2. This is not evidence the original requirement was withdrawn.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:7–17](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:7>); [HANDOFF — Category 5  Contract & Document Analysis.md:51–55](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:51>); [CAT5-documents.md:16–32](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:16>).

### DOC-02 — LibreOffice office conversion and deterministic DOCX/PDF rendering

- **Original state:** agreed conversion capability; named engine candidate.
- **Rationale/resources:** Office work must interoperate with DOCX/ODT/PDF and export durable work products. Headless LibreOffice sidecar; unoconv; python-docx.
- **Current-discussion overlap:** Explicit user reassertion today; historical provenance recovered.
- **Re-discuss:** Restore explicit renderer requirement, define format-direction/fidelity acceptance and deployment boundary. Do not assume PDF-to-editable-DOCX roundtrip quality from the historical double arrow.
- **Later disposition:** Later report selects LibreOffice legal-renderer, still HOLD. Current user request makes omission unacceptable; no installation performed.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:19–25](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:19>); [CAT5-documents.md:24–32](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:24>); [CAT5-documents.md:44–52](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:44>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:577–585](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:577>).

### DOC-03 — PDF viewing and interactive AcroForm filling

- **Original state:** agreed capability; candidates open.
- **Rationale/resources:** Owner needs to read and complete actual court forms in the workspace. pdf.js; react-pdf-viewer; pdfme.
- **Current-discussion overlap:** Partial overlap: templates; interactive form editing rediscovered.
- **Re-discuss:** Separate viewing, existing-field filling, new form design, save and render validation; choose initial official forms.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:17–20](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:17>); [CAT5-documents.md:34–37](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:34>).

### DOC-04 — Programmatic PDF form filling

- **Original state:** agreed capability; candidates open.
- **Rationale/resources:** Template data should populate actual form fields. pdf-lib; pypdf; pdftk/pdftk-java.
- **Current-discussion overlap:** New detail beneath templates.
- **Re-discuss:** Define reviewed field mappings, blank unknown fields and output checks for filled/flattened forms.
- **Later disposition:** Later report picks pypdf; runtime unverified.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:20–20](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:20>); [CAT5-documents.md:44–46](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:44>).

### DOC-05 — Markdown and structured JSON to printable PDF/Markdown

- **Original state:** agreed capability; candidates open.
- **Rationale/resources:** Reusable cover sheets, motions and reports need one composable rendering path. md-to-pdf; WeasyPrint; Pandoc; Jinja2; Handlebars/Nunjucks; @react-pdf/renderer.
- **Current-discussion overlap:** Partial overlap: templates, drafting.
- **Re-discuss:** Choose one reviewed template/render pipeline and clarify layout fidelity, fonts, page breaks and provenance manifest.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:21–24](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:21>); [CAT5-documents.md:44–51](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:44>).

### DOC-06 — PDF to Markdown/structured extraction, including OCR

- **Original state:** agreed and explicitly identified gap.
- **Rationale/resources:** Scanned or photographed court documents cannot be ignored; extraction must handle tables/layout. Docling; marker; unstructured; Tesseract.
- **Current-discussion overlap:** New interoperability detail alongside evidence/documents.
- **Re-discuss:** Clarify work-product/accepted-source conversion boundary versus evidence-platform raw ingestion; preserve page/spans and surface OCR uncertainty. Historical library claims are unverified candidates.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:22–22](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:22>); [HANDOFF — Category 5  Contract & Document Analysis.md:45–46](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:45>); [CAT5-documents.md:48–55](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:48>).

### DOC-07 — Treatise/self-help-course EPUB and ebook conversion

- **Original state:** agreed distinct use case; outside interactive workspace.
- **Rationale/resources:** Owner's custody treatise/course project was named separately from day-to-day legal work. Calibre ebook-convert CLI/API; Pandoc EPUB.
- **Current-discussion overlap:** Rediscovered; no clear overlap in supplied current list.
- **Re-discuss:** Keep as a separately tracked publishing need or explicitly defer; do not silently drop or inflate core app scope.
- **Later disposition:** Later report labels Calibre CLI later.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:25–25](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:25>); [CAT5-documents.md:52–52](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:52>).

### DOC-08 — True derivative redaction with detection and extraction checks

- **Original state:** agreed capability; removal approach research.
- **Rationale/resources:** Black rectangles that leave text extractable are insufficient. pikepdf; Microsoft Presidio.
- **Current-discussion overlap:** Partial overlap: evidence/export; detection-plus-removal rediscovered.
- **Re-discuss:** Confirm detection/review/removal separation, original preservation, text/image/metadata leakage tests and output verification; a dependency alone does not prove secure redaction.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:27–28](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:27>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:482–488](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:482>); [CAT5-documents.md:57–63](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:57>).

### DOC-09 — Bates numbering and exhibit page stamping

- **Original state:** agreed capability; implementation open.
- **Rationale/resources:** Sequential page identities support filing/discovery organization. pypdf; pdf-lib; custody-guide exhibit_indexer.py (later report reference).
- **Current-discussion overlap:** Partial overlap: evidence/exhibits.
- **Re-discuss:** Define prefix, sequence, stable page references and relationship between original and stamped derivative.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:30–31](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:30>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:432–439](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:432>); [CAT5-documents.md:65–69](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:65>).

### DOC-10 — Metadata inspection, scrub, hash check and printable metadata report

- **Original state:** agreed technical capability; legal certification explicitly open.
- **Rationale/resources:** Owner needs file metadata/integrity inspection and a printable record before sharing. exiftool; evidence API /v1/verify/{sha256} historical candidate; JSON-to-PDF template.
- **Current-discussion overlap:** Rediscovered beyond evidence browsing.
- **Re-discuss:** Separate factual technical report from legal authentication/certification; verify real endpoint and current Michigan primary authority before any court claim.
- **Later disposition:** Later report retains separate legal research HOLD; do not label report an established certificate.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:33–37](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:33>); [CAT5-documents.md:71–85](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:71>).

### DOC-11 — Official Michigan/Genesee/FOC form library

- **Original state:** agreed content requirement.
- **Rationale/resources:** Generic capability to fill PDFs was explicitly insufficient without real relevant forms. courts.michigan.gov; Genesee clerk/FOC official forms.
- **Current-discussion overlap:** Partial overlap: references/templates; official forms versus outlines distinction.
- **Re-discuss:** Choose actual official revisions and record provenance/fillability/currentness; private outlines are a different product.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:39–40](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:39>); [CAT5-documents.md:87–94](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:87>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:413–427](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:413>).

### DOC-12 — Own work-product hashes and immutable version receipts

- **Original state:** agreed capability; storage decision open.
- **Rationale/resources:** Owner wanted a record of exactly what was produced/filed and when. SHA-256; legal_work_product version metadata; Evidence Platform API as historical alternative.
- **Current-discussion overlap:** Overlap: evidence/proposed changes; work-product identity needs explicit treatment.
- **Re-discuss:** Distinguish generated-copy integrity, approved release and actual filing receipt; never imply producing PDF means it was filed.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 5  Contract & Document Analysis.md:42–46](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 5  Contract & Document Analysis.md:42>); [CAT5-documents.md:96–101](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT5-documents.md:96>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:347–368](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:347>).

### UI-01 — Dense grouped navigation with a pinned second panel

- **Original state:** agreed, split view explicitly high priority.
- **Rationale/resources:** Owner called split view 'super handy'; compare research/document while working. legal-terminal shell inspiration; react-resizable-panels; allotment; CSS grid.
- **Current-discussion overlap:** Confirmed latest dense/plain language/mouse+keyboard UX.
- **Re-discuss:** Keep density and split view; label destinations in ordinary language and provide direct click controls plus optional keyboard search. Do not retain mnemonic recall as required navigation.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 2  Command Bar & UI Shell.md:11–20](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 2  Command Bar & UI Shell.md:11>); [HANDOFF — Category 2  Command Bar & UI Shell.md:35–35](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 2  Command Bar & UI Shell.md:35>).

### UI-02 — Panel export and case-phase shortcuts

- **Original state:** agreed lower-priority v1.
- **Rationale/resources:** Capture useful current panel and surface actions appropriate to discovery/motions/hearing/trial. PDF/image panel export; Case-phase switcher.
- **Current-discussion overlap:** Rediscovered shell capabilities.
- **Re-discuss:** Decide exportable content versus private material; make phase switch change visible suggestions without hiding core tools.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 2  Command Bar & UI Shell.md:17–18](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 2  Command Bar & UI Shell.md:17>); [HANDOFF — Category 2  Command Bar & UI Shell.md:38–40](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 2  Command Bar & UI Shell.md:38>).

### UI-03 — Truthful health and confidentiality indicators; personal shell settings

- **Original state:** agreed.
- **Rationale/resources:** Status must report actual source connectivity; dark-only replaced earlier theme toggle. pydantic-settings; app_settings; Graphite/Playfair/Inter/IBM Plex Mono historical design.
- **Current-discussion overlap:** Partial overlap: current UX.
- **Re-discuss:** Separate connection state, stale data and running jobs; today's design can reconsider original visual tokens without reviving fake mock/live toggles.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 1  Persistence & Settings.md:14–14](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 1  Persistence & Settings.md:14>); [HANDOFF — Category 2  Command Bar & UI Shell.md:15–20](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 2  Command Bar & UI Shell.md:15>); [Legal OS — Feature Decision Matrix.md:31–35](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:31>).

### AI-01 — Summon assistant beside current work with automatic visible context and persistent session

- **Original state:** agreed.
- **Rationale/resources:** Owner should not re-explain the visible document; chat must survive navigation in other pane. F1 historical shortcut; structured panel/document/selection context; global session state.
- **Current-discussion overlap:** Overlap: firm roles, assistant; context continuity may be missed.
- **Re-discuss:** Specify visible-context consent/preview, selected excerpt and source references; add a plain-language button with optional shortcut.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:11–14](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:11>); [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:23–24](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:23>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:733–734](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:733>).

### AI-02 — Action-capable chat, router and directly selectable specialists

- **Original state:** agreed capability; action tiering open.
- **Rationale/resources:** Assistant should queue work and fill workflow steps while retaining access to named specialists. Agno/AG2/LangGraph/hand-rolled candidates; DSPy prompt optimization; LlamaIndex retrieval; Portkey reuse.
- **Current-discussion overlap:** Overlap: firmroles/skills/proposedchanges.
- **Re-discuss:** Use human-readable role names; recover scoped action execution and preview/confirmation rules. No autonomous approve/file/send.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:13–17](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:13>); [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:21–29](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:21>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:499–529](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:499>).

### AI-03 — Bounded multi-hop deep research with citations and cost limits

- **Original state:** agreed added gap fix.
- **Rationale/resources:** Single search calls were explicitly inadequate for decomposing/refining a legal question. Typed Category 4 research tools; query decomposition/refine/stopping loop.
- **Current-discussion overlap:** Partial overlap: references/skills.
- **Re-discuss:** Make depth/cost budget, citations across hops, adverse results and stopping conditions visible; distinguish research tool capability from orchestration.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:16–16](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:16>); [HANDOFF — Category 3  Chat   AI Agent Orchestration.md:28–28](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 3  Chat   AI Agent Orchestration.md:28>); [HANDOFF — Category 4  Research Tools.md:21–21](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:21>).

### RES-01 — Add research sources through settings JSON plus separately entered credentials

- **Original state:** agreed core requirement; manifest details proposed.
- **Rationale/resources:** Owner wanted additional sources without code changes and a single panel result shape. Case/Statute/Citation canonical models; CourtListener/RECAP default; Midpage candidate; Airbyte manifest subset; JSONPath/jsonschema.
- **Current-discussion overlap:** Overlap: references; configurable source interoperability rediscovered.
- **Re-discuss:** Define endpoint/auth/mapping/pagination validation and credential boundary; current provider API availability is not established by these docs.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 4  Research Tools.md:11–16](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:11>); [HANDOFF — Category 4  Research Tools.md:27–32](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:27>); [CAT4-research-tools.md:109–119](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT4-research-tools.md:109>).

### RES-02 — Hybrid internal search, full-text documents, configured general web search

- **Original state:** agreed explicit gap fix; engine choice open.
- **Rationale/resources:** Sources alone do not specify retrieval across saved research and internal work product. pgvector + full-text/BM25 candidates; RRF; Brave/Perplexity examples; Weaviate reuse considered.
- **Current-discussion overlap:** Partial overlap: references/evidence.
- **Re-discuss:** Specify exact identifiers, lexical+semantic results, matter/sensitivity filters and relevance explanations; keep internal work product separate from evidence truth.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 4  Research Tools.md:17–21](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:17>); [HANDOFF — Category 4  Research Tools.md:33–34](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:33>); [CAT4-research-tools.md:11–37](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT4-research-tools.md:11>).

### RES-03 — Citation parsing/normalization plus separate integrity and currentness

- **Original state:** agreed library requirement.
- **Rationale/resources:** Real citation parser replaces donor regex; formatted citations cannot prove legal status. eyecite; authority snapshots; official primary sources.
- **Current-discussion overlap:** Overlap: references/court translator.
- **Re-discuss:** Show parse result separately from source existence, pinpoint support and currentness; no inherited citator/privilege claim.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 4  Research Tools.md:15–15](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 4  Research Tools.md:15>); [Legal Terminal & Legal MCP — Deep Analysis (Part 1).md:105–107](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal Terminal & Legal MCP — Deep Analysis (Part 1).md:105>); [CAT4-research-tools.md:93–107](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT4-research-tools.md:93>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:632–637](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:632>).

### PRIV-01 — Confidential preflight, verified-provider routing and hard block

- **Original state:** agreed; terms verification open.
- **Rationale/resources:** No local inference capability; no unsafe provider fallback. Presidio; Portkey; Ollama Cloud; NVIDIA NIM; Venice; OpenRouter ZDR.
- **Current-discussion overlap:** Rediscovered explicit privacy pipeline.
- **Re-discuss:** Restore preflight and provider-grid requirements; verify current terms separately, never copy old donor eligibility/guarantees.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:7–24](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:7>); [HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:28–33](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:28>).

### PRIV-02 — Hybrid privilege detector and all-provider comparison

- **Original state:** agreed rule-first/ambiguity-escalation design.
- **Rationale/resources:** Toy keyword-only checking was specifically rejected; owner valued the provider grid. Rules + low-cost classification candidate; provider trust grid.
- **Current-discussion overlap:** Partial overlap: roles/AI.
- **Re-discuss:** Clarify how ambiguous sensitive text can be classified without prior unauthorized egress; flags remain hypotheses, never legal conclusions.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:22–24](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:22>); [Legal OS — Feature Decision Matrix.md:101–106](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:101>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:525–533](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:525>).

### PRIV-03 — Manual high-sensitivity Colab runbook

- **Original state:** agreed manual documentation only.
- **Rationale/resources:** An ephemeral on-demand GPU session was proposed for exceptional documents. Google Colab Pro; open-weight model manual setup/teardown.
- **Current-discussion overlap:** Rediscovered historical optional resource.
- **Re-discuss:** Decide whether to retain the documented option under today's remote-only policy; do not call it a privacy guarantee or automate it.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:21–21](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:21>); [HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:34–44](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 6  Privilege, Privacy & LLM Routing.md:34>).

### WF-01 — Browse and compose internal playbooks with step parameters, test run and results

- **Original state:** agreed handoff; sequencing conflict with build guide.
- **Rationale/resources:** Repeatable legal work should be inspectable and owner-composable. legal-mcp toolkit playbooks; custody packet checklists; sequential internal tool runner.
- **Current-discussion overlap:** Overlap: skills/roles; editable workflow UI rediscovered.
- **Re-discuss:** Resolve timing: Category 7 says builder, original guide forbids generic designer before fixed legal workflows proven. Preserve capability, agree phased scope.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 7  Workflow & Automation Engine.md:11–14](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 7  Workflow & Automation Engine.md:11>); [HANDOFF — Category 7  Workflow & Automation Engine.md:21–22](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 7  Workflow & Automation Engine.md:21>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:690–701](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:690>); [CAT7-workflow-automation.md:58–66](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:58>).

### WF-02 — Backend schedules/events with run now, filters, enable/disable and history

- **Original state:** agreed; scheduler choice open.
- **Rationale/resources:** Automation must continue with UI closed. APScheduler; BullMQ/node-cron/n8n cron evaluated; daily/weekly/once/event.
- **Current-discussion overlap:** Rediscovered operations detail.
- **Re-discuss:** Specify durable schedules/run state and recovery verification, not merely server timer existence; clarify event sources.
- **Later disposition:** Later report corrects PostgreSQL claim to MemoryJobStore + JSONL and retains PostgreSQL HOLD; does not prove current runtime.
- **Provenance:** [HANDOFF — Category 7  Workflow & Automation Engine.md:12–22](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 7  Workflow & Automation Engine.md:12>); [CAT7-workflow-automation.md:8–13](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:8>); [CAT7-workflow-automation.md:53–56](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:53>).

### WF-03 — Existing n8n for authorized external integrations

- **Original state:** agreed division; activation research-only.
- **Rationale/resources:** Reuse existing integration engine for mail/text/webhooks rather than build a second platform. n8n webhook/auth boundary.
- **Current-discussion overlap:** Rediscovered integrations; no sending authorized in this task.
- **Re-discuss:** Keep notification/inbound/outbound distinction and owner-controlled communications; agent boundaries prohibit autonomous external delivery.
- **Later disposition:** Later report states n8n webhook HOLD; no integration called.
- **Provenance:** [HANDOFF — Category 7  Workflow & Automation Engine.md:7–19](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 7  Workflow & Automation Engine.md:7>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:515–524](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:515>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:603–609](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:603>); [CAT7-workflow-automation.md:25–42](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:25>); [CAT7-workflow-automation.md:81–82](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:81>).

### OPS-01 — Inbox triage with process/dismiss, rules, attachments and contextual assistant

- **Original state:** proposed matrix rescope, not final decision.
- **Rationale/resources:** Convert donor simulated inbox into real court/FOC/opposing-counsel mail handling. IMAP/POP3/forwarding candidate; category/domain/keyword rules; n8n.
- **Current-discussion overlap:** Rediscovered beyond generic communications.
- **Re-discuss:** Decide inbound ownership and evidence-intake routing; remove demo injector/HR category and never treat email_received as live without a source.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [Legal OS — Feature Decision Matrix.md:142–151](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:142>); [Bloomberg + Legal Terminal Feature Catalog.md:162–167](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Bloomberg + Legal Terminal Feature Catalog.md:162>); [CAT7-workflow-automation.md:53–56](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/reports/CAT7-workflow-automation.md:53>).

### OPS-02 — Case monitor and alert thresholds

- **Original state:** proposed matrix rescope; guide Phase 4.
- **Rationale/resources:** Small fixed watchlist instead of demo product pitch and unlimited docket product. CourtListener/PACER historical candidates; notifications settings.
- **Current-discussion overlap:** Partial overlap: timelines; live monitoring distinct.
- **Re-discuss:** Re-discuss actual court/source coverage and budgets; historical PACER idea is not authorization, present integration or current requirement.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [Legal OS — Feature Decision Matrix.md:168–178](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:168>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:603–609](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:603>); [HANDOFF-ANALYSIS.md:67–68](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/HANDOFF-ANALYSIS.md:67>).

### OPS-03 — Saved worksheets, reusable filters, entity profiles, generalized comparison and CSV export

- **Original state:** proposed matrix Keep/Rescope, not final decision.
- **Rationale/resources:** Bloomberg provided broader information-work patterns beyond acronym navigation. Named rerunnable grids; canonical party/witness/exhibit profiles; 2–3-way compare; CSV/spreadsheet export.
- **Current-discussion overlap:** Overlap: timelines/personalcontext/evidence; worksheet/compare/export details rediscovered.
- **Re-discuss:** Select first concrete worksheets and comparison types; use dense master-detail tables and ordinary labels, avoid sprawling generic dashboard customization.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [Legal OS — Feature Decision Matrix.md:192–202](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:192>); [Bloomberg + Legal Terminal Feature Catalog.md:45–60](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Bloomberg + Legal Terminal Feature Catalog.md:45>).

### LAW-01 — Agreement analysis/compare/negotiate with missing parenting-plan provisions

- **Original state:** proposed matrix Keep/Rescope; guide P1.
- **Rationale/resources:** Settlement proposals, parenting plans and prior orders replace commercial template libraries. legal-mcp contract tool concepts; per-clause alternatives/fallback language.
- **Current-discussion overlap:** Overlap: proposedchanges; negotiation workflow rediscovered.
- **Re-discuss:** Define owner goals, offer/counteroffer history and clause review; suggestions cannot choose concessions or outcomes.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [Legal OS — Feature Decision Matrix.md:76–95](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal OS — Feature Decision Matrix.md:76>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:466–472](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:466>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:518–519](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:518>).

### LAW-02 — Hearing/trial binder, witness examination, objections, impeachment and post-hearing comparison

- **Original state:** guide P1 high-value operations.
- **Rationale/resources:** Connect factor/element proof to hearing preparation and follow-up. verified citation binder; provenance manifest; requested relief; order-compliance tasks.
- **Current-discussion overlap:** Partial overlap: timelines/evidence/firmroles.
- **Re-discuss:** Recover concrete hearing products and workflows; separate historical timelines from evidence-platform authored factual timeline.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:474–480](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:474>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:40–47](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:40>).

### LAW-03 — Discovery response/objection/deficiency and meet-and-confer lifecycle

- **Original state:** guide P1.
- **Rationale/resources:** Discovery extends beyond generating request text. interrogatories/RFP/admissions/subpoenas; deficiency history; motion-to-compel readiness.
- **Current-discussion overlap:** Partial overlap: roles/proposedchanges.
- **Re-discuss:** Track lifecycle and missing-proof linkage with owner-controlled scope/service; avoid reducing this to text generation.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:459–464](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:459>).

### CORE-01 — Persistence, source packages, staleness propagation and review invalidation

- **Original state:** agreed/historical guide boundary.
- **Rationale/resources:** Keep canonical evidence ownership and legal work product distinct with durable state. shared PG18 schemas and numbered SQL; Pydantic v2; R2 bytes; read-only Surreal projection; versioned neutral APIs.
- **Current-discussion overlap:** Overlap: evidence/references/proposedchanges.
- **Re-discuss:** Treat original monorepo placement and direct Surreal-primary framing as superseded; verify contracts independently and keep source changes invalidating dependent approvals.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [HANDOFF — Category 1  Persistence & Settings.md:11–24](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/HANDOFF — Category 1  Persistence & Settings.md:11>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:238–306](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:238>); [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:450–455](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:450>); [HANDOFF-ANALYSIS.md:61–64](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/HANDOFF-ANALYSIS.md:61>).

### LATER-01 — Scenario comparison, legal-template regression, model comparison, lawyer review and isolated reusable research

- **Original state:** guide P2 later.
- **Rationale/resources:** Explicit advanced roadmap outside initial proof. reviewed gold documents; latency/cost/token comparisons; local-court revision watchers.
- **Current-discussion overlap:** Partial overlap: firmroles/references; later features rediscovered.
- **Re-discuss:** Retain named backlog items; do not turn them into silent MVP commitments or multi-client tenancy.
- **Later disposition:** No later resolution established in this document-only pass.
- **Provenance:** [LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:490–497](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/LEGAL-WORKSPACE-BUILD-GUIDE-2026-08-17.md:490>).

## Named external integration register

This register distinguishes intended application forks/adapters from candidates and visual inspiration. The latest user correction reopens any later shape-only narrowing. It does not automatically approve every archive for integration.

### google/timesketch

- **Purpose / commitment:** Personal-case chronology, dense event exploration, annotations, narrative preparation and individual/bulk curation. Accepted ADR-0060/D-084/D-085 real maintained application fork; adapter-only explicitly rejected.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Probata/probata/modules/forks/timesketch`
- **Observed evidence:** Live read-only checks: clean local Git HEAD 00eff7d; no remotes listed. UPSTREAM.md records tag 20260630 and upstream 10dd077c6fe3b5e74fd9e28cd3ac1ef6c7c85849. personal_case_authority files and importer TimelineProjector present; upstream analyzer registration gate present.
- **Disposition / hold:** Historical WP-E02 reports no end-to-end importer/OpenSearch proof, no HTTP mount and governed source resolver absent. Aug30 places Timesketch in advanced surface gated on deployment/PG round-trip proof. Current deployment not tested.
- **Provenance:** [0060-timesketch-personal-case-timeline-fork.md:3–40](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/adr/0060-timesketch-personal-case-timeline-fork.md:3>); [0060-timesketch-personal-case-timeline-fork.md:104–145](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/adr/0060-timesketch-personal-case-timeline-fork.md:104>); [README.md:3–24](<E:/AI_Workspace/Projects/Propria/Probata/probata/modules/forks/timesketch/personal_case_authority/README.md:3>); [__init__.py:22–35](<E:/AI_Workspace/Projects/Propria/Probata/probata/modules/forks/timesketch/timesketch/lib/analyzers/__init__.py:22>); [TIMESKETCH-WP-E02-IMPLEMENTATION-STATUS.md:118–145](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/reviews/2026-08-25-schema-audit/TIMESKETCH-WP-E02-IMPLEMENTATION-STATUS.md:118>).

### react-calendar-timeline

- **Purpose / commitment:** Lightweight primary-surface event visualization. Aug30 owner direction: retain as development-only option; not replacement for Timesketch.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Probata/probata/modules/workbench/web/package.json`
- **Observed evidence:** Declared ^0.30.0-beta.19 at line 44; no UI runtime audit in this lane.
- **Disposition / hold:** Choose engine per concrete visualization; no forced bake-off.
- **Provenance:** [package.json:44–44](<E:/AI_Workspace/Projects/Propria/Probata/probata/modules/workbench/web/package.json:44>); [WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:22–25](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/awaiting-verification/WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:22>).

### vis-timeline

- **Purpose / commitment:** Alternative lightweight interactive timeline. Aug30 owner direction: retain alongside react-calendar-timeline.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Probata/probata/modules/workbench/web/package.json`
- **Observed evidence:** Declared ^8.5.4 at line 53; no runtime verification.
- **Disposition / hold:** May coexist for different jobs; does not replace governed Timesketch.
- **Provenance:** [package.json:53–53](<E:/AI_Workspace/Projects/Propria/Probata/probata/modules/workbench/web/package.json:53>); [WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:113–116](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/awaiting-verification/WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:113>).

### Evidence.dev

- **Purpose / commitment:** SQL/Markdown reports over PG/DuckDB. Historical selection explicitly never reversed; project re-establishment owed.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Corrected September 2 note: not deployed for platform after project moved to TraceIQ.
- **Provenance:** [gui-integration-spec.md:111–111](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:111>).

### Claude Code history viewer

- **Purpose / commitment:** Session history browsing. Owner-requested; exact repository undecided.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Candidates d-kimuson/claude-code-viewer, InDate/claude-log-viewer, daaain/claude-code-log; no selected local fork verified.
- **Provenance:** [gui-integration-spec.md:112–112](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:112>).

### NeoDash (neo4j-labs)

- **Purpose / commitment:** Entity networks, relationship timelines, incident maps. Embed candidate.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Historical iframe /x/neodash/ idea; current adoption not verified.
- **Provenance:** [gui-integration-spec.md:113–113](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:113>).

### Surrealist

- **Purpose / commitment:** SurrealDB admin interface. Embed candidate.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Historical /x/surreal/ operator tool; not legal work-product editor.
- **Provenance:** [gui-integration-spec.md:114–114](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:114>).

### CopilotKit generative UI / AG-UI

- **Purpose / commitment:** Inline agent-generated charts/tables/timelines. Historical in-app integration design.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Current survival not verified; do not resurrect older architecture by implication.
- **Provenance:** [gui-integration-spec.md:115–115](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:115>).

### React Flow

- **Purpose / commitment:** Workflow-run directed graph visualization. Historical in-app G5 component candidate.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Specific graph library identified; no local dependency found in current app auditor's lane.
- **Provenance:** [gui-integration-spec.md:116–116](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:116>).

### Kepler.gl / Leaflet pattern

- **Purpose / commitment:** Geospatial PostGIS analysis. Historical map pattern candidate.
- **Local source/fork:** No exact retained source/fork established in this bounded pass.
- **Observed evidence:** Named source integration row; local implementation not assessed.
- **Disposition / hold:** Keep separate from chronology; source local fork not established.
- **Provenance:** [gui-integration-spec.md:117–117](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/planning/gui-integration-spec.md:117>).

### legal-mcp

- **Purpose / commitment:** Research/citation/document/clause/queue tool logic and workflow skills. Deep analysis calls primary toolset; later analysis says port as FastAPI adapters, not second MCP runtime.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/legal-mcp-main/legal-mcp-main`
- **Observed evidence:** Retained source pyproject.toml/README/tools exists; current app port fidelity owned by auditor.
- **Disposition / hold:** Replace toy regex/provider postures/demo knowledge; preserve tool jobs, not legal conclusions.
- **Provenance:** [DONOR-ANALYSIS.md:65–94](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:65>).

### legal-terminal-master (genego-io)

- **Purpose / commitment:** Dense Bloomberg-style shell, split panels, master-detail and job tables. From-scratch visual/function inspiration explicitly, not code co-option.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/legal-terminal-master/legal-terminal-master`
- **Observed evidence:** Original screenshots and source tree retained; no modification assessment.
- **Disposition / hold:** Current plain-language mouse+keyboard requirement overrides mnemonic-first interaction.
- **Provenance:** [Legal Terminal & Legal MCP — Deep Analysis (Part 1).md:143–147](<E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/artifacts (3)/Legal Terminal & Legal MCP — Deep Analysis (Part 1).md:143>).

### legal-terminal-main (JuriSupport)

- **Purpose / commitment:** Record/draft split, file mentions, hearing notes, PDF reading. Later donor analysis treats separate Korean Electron product as shape donor.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/legal-terminal-main/legal-terminal-main`
- **Observed evidence:** README/package.json and screenshots/app.png retained; differs from master donor.
- **Disposition / hold:** Later analysis excludes Korean-specific HWP/SSH/Electron assumptions. Latest user says some outside projects intended adoption; this interpretation should be re-discussed.
- **Provenance:** [DONOR-ANALYSIS.md:140–153](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:140>).

### LexRAG

- **Purpose / commitment:** Hybrid legal retrieval separate from chat. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/LexRAG-main/LexRAG-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:155–160](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:155>).

### LIGHT-2

- **Purpose / commitment:** Router/specialist agents, authority refresh and form workflow. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/LIGHT-2-main/LIGHT-2-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:162–171](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:162>).

### THEMIS

- **Purpose / commitment:** Skills per instrument/work-product kind. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/themis-main/themis-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:173–182](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:173>).

### Suna / Kortix

- **Purpose / commitment:** Agent operating system and durable tool/run traces. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/suna-main-docs/suna-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:184–187](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:184>).

### OpenLegalDataSkills

- **Purpose / commitment:** Case/statute/citation source tools and skills. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/OpenLegalDataSkills-main/OpenLegalDataSkills-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:189–192](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:189>).

### claude-power-skills

- **Purpose / commitment:** Reusable legal skill/prompt packages. Archive present; later Aug18 analysis classifies as shape/skill donor, not proven owner approval to narrow scope.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/claude-power-skills-main/claude-power-skills-main`
- **Observed evidence:** Top-level resource identity read; source fork modifications not diffed.
- **Disposition / hold:** Re-discuss later shape-only narrowing against user's recalled integration intent. No archive was run or installed.
- **Provenance:** [DONOR-ANALYSIS.md:189–192](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/DONOR-ANALYSIS.md:189>).

### custodyguide_v1complete_20260812

- **Purpose / commitment:** Case-specific guardrails, sources, templates and tools. Later analysis says this case's substance, not optional content pack.
- **Local source/fork:** `E:/AI_Workspace/Projects/Propria/Legal-desktop/resources/build-kit/donors/custodyguide_v1complete_20260812/Projects/custody-guide`
- **Observed evidence:** Retained original case packet; legal accuracy/currentness not reverified here.
- **Disposition / hold:** Keep legal primary-source and template-version checks; parent separate resource review owns detailed packet analysis.
- **Provenance:** [HANDOFF-ANALYSIS.md:70–85](<E:/AI_Workspace/Projects/Propria/Legal-desktop/docs/planning/HANDOFF-ANALYSIS.md:70>).

## Timesketch verification boundary

Read-only Git status returned no changes and HEAD `00eff7d` (“Initial commit: extracted from Agno-MCP-Platform (owner restructure 2026-09-01)”). `git remote` returned no remotes. The old UPSTREAM document describes a plain snapshot without Git; today's independent Git boundary reflects the later extraction. Do not mistake that stale packaging description for today's repository state.

The upstream pin is recorded in `UPSTREAM.md`; this pass did not fetch or diff upstream. Local modification evidence is concrete: `personal_case_authority/authority.py`, `fixtures.py`, and `importer.py` exist; `TimelineProjector` appears at importer.py:137; the upstream analyzer import gate appears at `timesketch/lib/analyzers/__init__.py:34`. Historical WP-E02 explicitly limits its reported verification: importer not executed against real Timesketch/OpenSearch, no HTTP mount, no populated governed-source resolver. Its rollback-scoped PostgreSQL tests are historical reported results, not tests rerun today. Deployment and real bulk-edit success remain unverified.

The retained README describes forensic timeline analysis at line 18 and says the code is Google-owned but not an official Google product at line 50. This supports identifying the user’s recollection as Timesketch, without converting it into a Google product endorsement.

## Donor layout recovery and screenshot paths

All paths below were found on disk; screenshots were identified, not visually rendered in this lane.

- `resources/build-kit/donors/legal-terminal-master/legal-terminal-master/docs/screenshots/`: web-01-home.png, web-02-chat-empty.png, web-03-chat-research.png, web-04-prec.png, web-05-ctrx.png, web-06-priv.png, web-07-doca.png, web-08-palette.png, web-10-docket-watch.png, web-12-conf-on.png.
- `resources/build-kit/donors/legal-terminal-main/legal-terminal-main/screenshots/app.png`: the separate JuriSupport application’s layout.
- `resources/build-kit/donors/LIGHT-2-main/LIGHT-2-main/frontend/src/assets/screen.png`: kiosk app asset; purpose requires visual inspection.
- `resources/build-kit/donors/LexRAG-main/LexRAG-main/marketing/assets/`: hero.png, evolution_core.png; marketing assets, not assumed product proof.

Keep the dense fixed dashboard, readable tables, persistent case/document context, side-by-side source/draft or source/assistant, short actual-document list and clear empty/error/stale states. Simplify the fixed single-case shelf, human-readable commands and small named worksheets. Do not restore artificial chat delays, fake live toggles, simulated inbox injection, demo docket votes, commercial fixture libraries, required acronym commands or finance content. Generalized comparisons and CSV export are recoverable proposed capabilities, not automatically in v1.

## Archive package map

The original archive directory is `resources/build-kit/Sources and skills/`. Names are inventory evidence, not approval to install. Unpacked peers sit under `resources/build-kit/donors/`; suna-main.zip maps to the docs-only suna-main-docs directory, and sequential-react-ship skill.zip maps to sequential-react-ship. The seven handoffs and companion original analyses also remain in `artifacts (3).zip`.


## Latest unified-surface correction and exact handoff locations

The owner now requires the applications to be part of one unified surface and to bring the underlying languages/frameworks together. This is a new controlling reconciliation constraint. The old Vue-retention and iframe/deep-link choices are recorded so they can be consciously revisited; they are not permanent exemptions from convergence.

- Timesketch original implementation handoff: [TIMESKETCH-FORK-CURATION-HANDOFF.md:76](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/reviews/2026-08-25-schema-audit/TIMESKETCH-FORK-CURATION-HANDOFF.md:76>) retains timeline/search, annotations, analyzer/aggregator extension interfaces and Vue shell. Lines 92–100 split work into TS-00..TS-08, including UI bulk selection/results at TS-05 and mandatory live deployment proof at TS-08.
- Fork foundation handoff: [TIMESKETCH-FORK-WP-E01-HANDOFF.md:17](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/reviews/2026-08-25-schema-audit/TIMESKETCH-FORK-WP-E01-HANDOFF.md:17>) records the pinned upstream revision; lines 28–34 enumerate the copied tree, analyzer gate, authority extension and smoke/Podman files.
- Later unified-surface decision: [ADR-0061:23](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/adr/0061-unified-operator-surface.md:23>) says one operating context while keeping domain/deploy boundaries; lines 129–131 choose same-origin proxy with bounded iframe/full-page launch, expressly no module federation. This is precisely the historical choice to revisit under today’s language-convergence instruction.
- Retained timeline-library owner direction: [WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:22](<E:/AI_Workspace/Projects/Propria/Probata/probata/docs/awaiting-verification/WORKBENCH-STORYBOOK-AND-SURFACE-REFACTOR-2026-08-30.md:22>) keeps both lightweight engines while retaining Timesketch separately. Their declarations were observed in the current sibling module package.json, not inferred from that report.
- Other project handoffs: no individual owner-approved handoff for every NeoDash/React Flow/Kepler/Surrealist/Evidence.dev entry was located in this bounded pass. The cited integration-spec rows are actual historical planning evidence but must not be upgraded into final integration commitments. The archived legal project packages’ later DONOR-ANALYSIS shape-only interpretations likewise do not settle the user’s remembered co-option intent.

The next design discussion should decide a common presentation/language strategy while preserving evidence, work-product, curation and review ownership. Recovering a maintained fork does not itself decide whether to preserve its Vue interface, port the interface, or expose its capabilities in a shared UI.

## Enumerated ZIP packages

- appellate-formatting.zip
- appellate-mandate.zip
- claude-power-skills-main.zip
- cold-start-interview.zip
- custodyguide_v1complete_20260812.zip
- decision-record-verification.zip
- duckdb-data-explorer.zip
- generate-hypothetical-scotus-ruling.zip
- hallucination-taxonomy.zip
- irac-practice.zip
- Legal guidance.zip
- legal-mcp-main.zip
- legal-terminal-main.zip
- legal-terminal-master.zip
- LexRAG-main.zip
- LIGHT-2-main.zip
- OpenLegalDataSkills-main.zip
- plain-english.zip
- precedent.zip
- quality-checker.zip
- research-start.zip
- sequential-react-ship skill.zip
- singularity-judicial-intelligence.zip
- suna-main.zip
- themis-main.zip
- trusted-sources.zip

## Verification receipt

2026-09-13: rediscovery.json parsed successfully: 35 requirements and 20 named integration records. All 124 citation references were checked for source-file existence and in-bounds line ranges; zero failures. This is citation-location validation, not independent proof of every historical claim. Original sources were read only. No donor installation, service execution, deployment, or legal-authority verification occurred. An initial oversized shell write failed before process creation; subsequent quoted-content writes were replaced with successful encoded writes and the final complete report was verified from its heading and size.
