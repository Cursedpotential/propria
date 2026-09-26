# Advocatio: document workflow and outstanding delivery

Date: 2026-09-24. Author: Codex. This report reconciles the owner's current direction, recovered commitments, inspected implementation, and recorded integration receipts. It does not replace the 52-requirement register. Implementation recommendations below remain distinguishable from implemented capabilities.

Implementation update later September 24: the durable document service, styled DOCX templates, Collabora bridge and writing workspace have now been implemented and locally tested. Deployment is blocked by the deliberately stopped Docker engine on ovh-app. See the [implementation receipt](../receipts/2026-09-24-office-workspace-implementation.md) for exact proof and remaining gates. The current-state sections below describe the baseline that motivated this work, not a claim that these new components are absent.

## 1. Finding and required correction

The document workspace is incomplete. The original requirement was a real place to write and edit documents, supported by templates, official forms, suggested edits and native tracked changes. The current application supplies section textareas, coded outline templates, comparison views and an office-to-PDF service. These are useful components, but they do not fulfill that requirement.

Repeated titles and policy prose made the interface noisier while obscuring this missing functionality. Removing that prose improves usability; it does not close the functional gap. A template selector on the current Motion writer page is an immediate entry-point repair, not delivery of LibreOffice integration.

The owner has now explicitly clarified another required flow: begin with rough writing, then apply an appropriate PDF or office template later, using AI proposals and human review. This does not require the owner to decide the document's final structure before getting their thoughts down.

## 2. What was actually agreed

The preserved [original Category 5 handoff](../../resources/build-kit/artifacts%20(3)/HANDOFF%20%E2%80%94%20Category%205%20%20Contract%20%26%20Document%20Analysis.md), lines 13–24, requires real browser editing, native tracked changes, interactive PDF forms, programmatic field filling and office conversions. Lines 39–40 call for actual Genesee/FOC/circuit forms. Collabora and OnlyOffice were candidates; capability approval must not be confused with a final vendor selection.

The [requirements register](../planning/2026-09-13-advocatio-reconciliation/REQUIREMENTS.md), R13–R20, preserves methods for drafting and existing-document review, four separate review dimensions, accept/reject/edit proposals, court-language translation with original wording retained, and the distinction between interactive office editing and background rendering. R49 requires compact flags instead of repeated disclaimers.

The [rediscovery register](../planning/2026-09-13-advocatio-reconciliation/REDISCUSSION.md), lines 3–9, explicitly says HOLD and unimplemented do not mean the owner rejected a feature. It retains full office editing and AcroForm filling.

The [earlier gap report](../GAP_SHEET_AND_REMEDIATION_PLAN.md), lines 217–220, already identified missing editing, conversion and forms. Its conversion finding is now historical. The [external tools register](../planning/2026-09-13-advocatio-reconciliation/continuation/EXTERNAL-TOOLS-REGISTER.md), lines 37–48 and its later receipt, records a working LibreOffice renderer while continuing to list the editor, PDF filling and official forms library as absent.

No exact earlier phrase for “retroactively apply a template” was recovered in the bounded search. The owner's September 24 instruction establishes that requirement directly; existing-document review and proposed changes were already documented. Do not claim a verbatim historical agreement that was not found.

## 3. Implementation now, and what it does not yet provide

| Component inspected | Present behavior | Outstanding work |
|---|---|---|
| `web/src/components/DraftEditor.tsx` | Edits a section heading and plain-text body; saves through the draft API | Full document, formatting, pagination, footnotes, native comments/revisions, autosave/recovery |
| `TemplateForm.tsx`, `api/legal_workspace/domain/templates.py` | Instantiates coded document outlines as draft sections | Versioned ODT/DOCX template files; field mapping; applying a template to existing writing |
| `ContractWorkbench.tsx` | Reads/compares saved sections; review notes are component state | Durable notes and passage-linked proposals; uploaded-document review; meaningful version comparison |
| `services/renderer.py`, `api/document_routes.py` | Gotenberg LibreOffice conversion produces PDF derivatives and hashes | Interactive editing session and editor save integration; independent round-trip proof |
| `services/docx_export.py` | DOCX assembly exists | Proven office-template fidelity and preservation of tracked changes/comments through edits/exports |
| `PdfPane.tsx` | PDF-oriented preview/redaction surface | Field discovery, actual AcroForm input, mapped values, saved fillable output and official form catalog |
| Draft review/release | Section-level review and immutable release concepts | Individual edit decisions, stale-base conflicts, aggregate document revisions and rendered revision identity |
| Existing template library | Working outlines | Official Michigan/Genesee/FOC forms, applicability metadata, downloadable source templates and version retirement |

Source inspection establishes the missing editor. The September 21 conversion receipt records a synthetic DOCX-to-PDF success through the deployed application; this report does not reclassify that historical receipt as a fresh conversion test today.

## 4. Recommended everyday workflow

### One document workspace

Use **Documents** as the umbrella. The main area is the document; the optional right panel holds review, sources, template mapping and the assistant. The document title appears once. Keep save state and unresolved-review counts compact. Editing, reviewing and preparing output are actions within the same document, rather than unrelated screens each repeating the title and policy explanation.

Entry actions:

1. **Write a document** opens a blank editable document immediately. A template and citations are not prerequisites to save rough work.
2. **Open a document** imports an existing ODT/DOCX or working PDF and preserves the original bytes and filename.
3. **Use a template** opens the selected office template with appropriate fields/styles.
4. **Fill a court form** opens the original official PDF with a readable field panel and page preview.

The phone view should make opening, reading, adding a document, entering short text, filling fields and deciding proposals easy. A full desktop page-layout editor can remain available without forcing every phone interaction through miniature desktop controls. Both views address the same document and revision IDs.

### Write first, organize later

The owner writes or pastes freely. Save the original wording as a revision. An explicit **Apply template** action then selects the intended document type and version. The review panel previews a proposed structured document and lists unresolved mapping choices.

AI maps passages into sections or fields, proposes necessary rewording, and identifies omissions. It does not silently discard duplicate-looking text, compress a meaningful allegation into a different claim, invent dates, or introduce requested relief that was never selected. Unmapped material stays visible in an “Unplaced text” tray until the owner decides what to do with it.

Separate structural moves from changes of meaning. A paragraph moved under Facts can be accepted independently of a proposed rewrite. Formatting changes can be grouped; substantive changes remain individually reviewable. Preserve source links, notes, footnotes and references when material moves. Accepted changes create a new document revision; rejected changes remain recorded without changing the text.

Example: a rough account describes dates, denied parenting contact and a desired response. Applying a motion template proposes a caption, factual sequence and requested-relief section. Unknown caption fields remain empty. The account is retained, statements needing evidence get passage-level flags, and any proposed relief is visibly a proposal. The user can continue typing while review runs; results based on an older revision require reconciliation before application.

### Start with a template

Templates are actual versioned ODT/DOCX files, not only arrays of headings. They carry styles, margins, page numbering, caption layout, signature blocks and optional sections. Field bindings have stable identities independent of their visible labels. A reusable template may have accompanying drafting guidance without treating guidance as document text.

Prefill known case fields from selected case records with visible provenance. Do not silently copy dates, signatures or factual claims from an unrelated prior draft. After instantiation, ordinary writing remains possible; filling fields must not be the only way to edit an office document.

### Fill an official PDF

Keep the original PDF as the layout authority. Discover AcroForm fields, map clear labels, allow direct field edits and show the original page while filling. Offer saved fillable output and a separately identified flattened rendition when needed. Preserve field values and checkboxes across save/reopen and download.

Handle overlong answers explicitly: show overflow and offer an attachment page with a link back to the field. Never shrink or truncate text invisibly. Forms without interactive fields need a separately reviewed placement map; do not pretend every PDF has native fields. Signatures remain an explicit user action, separate from prefilling a name.

LibreOffice Writer controls and PDF form controls are not identical, and only some controls transfer through PDF export. This is why official PDF filling should have its own adapter rather than assuming a Writer round trip preserves everything. [LibreOffice Writer Guide, Forms](https://books.libreoffice.org/en/WG262/WG26218-Forms.html).

## 5. Full office integration architecture

Recommend evaluating **Collabora Online first**, because the owner specifically wants a LibreOffice-based interactive experience. Keep OnlyOffice as the original comparison candidate, not an unannounced replacement. Verify current deployment terms, compatibility and resource requirements before selecting a pinned production edition/image. The first gate is a real document round trip, not an embedded editor screenshot.

Retain Gotenberg for background rendering. The interactive editor is a separate service with a document-storage integration. Use the supported editor protocol, scoped authorization, version checks and file locks; do not give the browser a storage administrator credential. Collabora integration documentation is the implementation reference: [SDK manual](https://sdk.collaboraonline.com/CO-SDK-manual.pdf). Exact protocol details must be checked against the selected release before implementation.

The legal document service owns work-product originals and revisions. The editor opens and saves the same document revision family; PDF outputs record the input revision and renderer version. Accepted evidence remains in Probata and is linked by immutable source/version/span. Importing an evidence file should route through evidence intake; importing a work-product draft belongs here. Those two choices need plain labels in the import workflow.

Keep the frontend editor adapter in shared React/TypeScript. Reuse the common design contract, identity and API contracts; do not let a proprietary editor session dictate the legal domain model. Existing Next serving and Python services remain migration components while the approved unified-surface plan proceeds. A shared palette is not completed stack convergence.

Proposed persistent records:

| Record | Required content |
|---|---|
| Document | Matter, title, type, original format, current revision, access scope |
| Revision | Parent, bytes/hash, editor identity, timestamp, edit origin, selected template version |
| Template/form version | Original file, issuer/source, jurisdiction, date/version, fields, styles, mappings, supported output formats |
| Proposal | Base revision, exact target anchor, original/proposed text, move/wording/field/format category, explanation, method/source versions, disposition |
| Review finding | One of four dimensions, passage/field, criterion, support, reviewer/checker, time, stale state |
| Rendition | Input revision, converter/version, output bytes/hash, format and rendering checks |

The four dimensions are structure/method, legal sources/citations, factual support, and substantive reasoning/relief. Show compact counts and expandable findings. None should be collapsed into one universal “validated” badge.

Native office tracked changes and the application proposal ledger must be coordinated. Each accepted proposal must identify its resulting document revision. Merely highlighting different strings in a web panel is not proof that native revision marks survive download and reopening in LibreOffice.

## 6. Reliability and completion requirements

- Autosave with clear Saving/Saved/Save failed states, explicit retry and recoverable unsaved text. Never display Saved before durable acknowledgement.
- Concurrent browser/agent edits use base revisions and locks/conflict handling. An outdated AI result cannot overwrite intervening typing.
- A lost editor connection must preserve recoverable work and explain reconnection without blocking access to the last saved revision.
- Released versions remain immutable; subsequent edits create descendants. Template upgrades do not silently alter existing documents.
- Keyboard operation, readable mobile forms, accessible labels, meaningful focus movement and resizeable side panels are part of acceptance.
- Reference inspector opens the actual pinned source; compact support flags distinguish absent, partial, unreviewed and contrary support.
- Work-product imports keep the original; every transform produces a related derivative. Existing evidence custody is not duplicated.

The acceptance fixture pack must include a formatted ODT and DOCX with caption, tables, footnotes, comments, tracked edits and page numbering; an AcroForm with text/checkbox/repeated fields; a nonfillable PDF; an overflowing answer; contradictory/unsupported rough text; an interrupted save; and competing edits. Test editing, reopening and export in both application and desktop LibreOffice. Compare appearance and retained structure, not just whether a file opens. PDFs need page inspection and text checks as well as hashes; repeated conversion is not assumed byte-deterministic.

## 7. Whole-workdesk obligations retained

This matrix is a requirement coverage map, not a claim that every subsystem was runtime-audited today. Historical comparisons must be refreshed per slice before changing implementation.

| Scope / requirement IDs | Current evidence | Required deliverable and proof |
|---|---|---|
| Canonical routing, common sources, ownership R01–07 | Independent repo/build kit preserved; source-package boundary in code | Same source identities across phone, agents and workdesk; provenance and explicit correction relationships |
| Entire legal corpus R08–12 | Reference/capability inventories and existing browse/search surfaces | Reconcile all baseline items and earlier iterations; read back citations, versions, cheat sheets and source links; vLex originals still pending |
| Methods and document review R13–20 | Outline drafts, section review and conversion | Complete writing/form/template/proposal workflow described above; contextual method library and four independent checks |
| Context and strategy R21–24 | Private-note/critique and analysis surfaces | Durable strategy memos, opposing claims/risks/responses, personal account distinctions, linked evidence-analysis provenance and corroboration tasks |
| Timelines and support R25–31 | Docket-oriented calendar/timeline; recovered Timesketch/visualization plans | Relationship/contact/claims/court views over shared events; exact evidence links, partial/contrary support and reproducible gaps reports |
| Digital firm R32–36 | Role routing/traces exist | Original prompt recovery, managing/senior associate, red team, clerk, librarian/researcher and exhibit clerk; persistent handoffs, budgets and review records |
| Outside tools R37–38, R42, R50 | Preserved handoffs/donor inventory and integration register | Explicit adopt/adapt/defer decision per contribution, implemented adapters and functional tests; FreeEed/KAPE extraction belongs with evidence/intake |
| Human interface R39–40, R49 | Shared palette/task navigation; current copy cleanup | Dense mouse/keyboard workspace; one heading, accurate help, compact flags, useful empty-state actions; no phase-tab maze |
| Convergence R41, R44, R46 | Language/stack planning and common design tokens | Shared React/TypeScript contracts and staged migration with retirement criteria; no second conflicting source store |
| Toolkit/MCP R45, R47–48, R51 | Capability map and registrations are partial evidence | Every toolkit capability embedded/callable or explicitly unresolved; authenticated registration/discovery/manual calls, results and scoped agent use; current remote-store readback |
| Deployment R52 | Existing named legal service and Coolify route | Follow port registry; verify actual service/health/deployment commit without inventing new exposed ports |

The 383-entry capability map is not 383 working integrations. A connected MCP server is not proof that all toolkit resources are imported, accessible or usable from this app.

Other recovered obligations must stay visible: hearing binders; witness examination/objections/impeachment; post-hearing order comparison; discovery responses, objections, deficiencies and meet-and-confer; saved worksheets and filters; inbox triage; monitoring thresholds; editable playbooks and durable scheduled-run history; scenario comparison and reviewer packets; metadata/redaction/Bates reports; OCR/layout extraction; and treatise export. Their provenance/dispositions remain in REDISCUSSION and the external-tools register.

Timesketch, react-calendar-timeline, vis-timeline and Evidence.dev remain specifically retained integrations. NeoDash, Surrealist, React Flow, CopilotKit/AG-UI and Kepler/Leaflet are named candidates requiring bounded disposition, not automatically approved installations. Their adapters must consume common records instead of creating new factual authority.

## 8. Seven bounded implementation tasks

| Task | Dependencies and ownership | Deliverable / acceptance |
|---|---|---|
| 1. Durable document foundation | Backend document/revision service; no editor UI edits | Create blank document, import work product, save/reopen revisions, detect stale writes; service restart retains bytes/metadata; explicit API contract |
| 2. Interactive office vertical slice | Task 1; separate deployment/editor-adapter owner | Actual browser editing of ODT/DOCX, durable save, reopen and download; selected version/service/credentials recorded; representative formatting round trip |
| 3. Office template library | Task 1; template service/catalog owner | Real versioned files, field/style maps, new document from template, preserved original; sourced initial templates and completeness register |
| 4. Official PDF form lane | Task 1; PDF adapter and catalog owner | Verified official form sources, field mapping/editing, save/reopen, fillable and flattened output, overflow handling; independent from office UI |
| 5. Reviewable template application | Tasks 1, 3; proposal service owner, shared schema agreed first | Rough text to proposed structure; all input accounted for; accept/reject/edit, stale-base handling and persisted decisions; no live provider required for deterministic fixtures |
| 6. Complete document review workspace | Tasks 2, 4, 5; frontend owner | Same document identity across editing/forms/review; court-language proposals, sources/method panel, four checks, mobile flow, accessible controls; native tracked-change fidelity demonstrated |
| 7. Package and release integration | Task 6; render/export owner | Correct revision to ODT/DOCX/PDF, binder/exhibits and release manifest; immutable release; user-readable support gaps and complete receipts |

Tasks 2, 3 and 4 can be assigned in parallel after Task 1's contract is settled. Task 5 can begin with deterministic proposals without waiting for a model connection. Keep file ownership explicit; no two agents rewrite the same workspace component. Use inexpensive readers/builders for bounded work and focused higher-capability review for architecture, fidelity and conflicts.

Each task must leave changed files, tests, remaining gaps, source pointers, deployment status and exact next action in its receipt. Stop calling a task complete at “route exists,” “library installed,” “schema added,” or “PDF generated.” The acceptance behavior must be demonstrated.

## 9. Immediate changes in this delivery

The current patch removes duplicated shell/page titles, static policy chips and repeated disclaimer prose across the application. It gives pages action-oriented descriptions, uses readable privacy/provider states, labels draft fields, and places the existing outline-creation action on the writing page. These changes preserve backend review and authorization behavior. The privacy scanner is described accurately as a keyword scan with category/excerpt results.

The current outlines are labeled as outlines in the drafting entry point. Full office editing, official forms and retroactive template application remain open and are not represented as delivered by this patch.

Evidence import remains outstanding: provide an explicit import action choosing work-product upload versus evidence intake, routed to the correct owner. Discovery requests (RFA/RFP/interrogatories/subpoenas directed to the other party) stay distinct from internal investigation requests to the evidence platform.

## 10. Next execution handoff

Start Task 1 with the current document routes/storage and released-version rules. Reuse existing bytes and revision mechanisms where correct; do not build a parallel store. Produce an API/record diff and passing create/import/save/reopen/conflict tests. Then Task 2 must prove real editable office documents before marking LibreOffice integration complete. The broad workdesk requirements and external integrations remain tracked in their existing registers; this document expands the writing workflow rather than shrinking that scope.

Outstanding supplied materials: original digital-firm prompts and vLex examples. Outstanding technical proof: selected interactive editor edition, full round-trip behavior, official form catalog coverage and currently deployed corpus/tool coverage. These dependencies do not block durable blank-document editing or deterministic proposal mechanics.
