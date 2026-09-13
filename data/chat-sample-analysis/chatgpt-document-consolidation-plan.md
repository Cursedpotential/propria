# ChatGPT-Document consolidation plan.md
> _Byline: Claude Code · Sonnet 5 · 2026-07-11_

## Snapshot
- Source: ChatGPT export → Markdown, single conversation, run inside a custom
  GPT project named **"court"** (URL slug `g-p-6880e45d99a88191b8d8b15d09c70b12-court`).
- Metadata block at top of file: `User: Matthew Salem (Error 404 Fuks Not Found)
  (matt.salemnet@gmail.com)`, `Created: 8/2/2025 20:56`, `Updated: 8/18/2025
  19:04`, `Exported: 8/21/2025 13:54`, plus the original `chatgpt.com` link.
- Size: 520 KB / **3,704 lines**. One continuous conversation, no sub-files.
- Era: content spans **2017/2018 → August 2025** (the case's real-world
  timeline), authored live between **Aug 2 – Aug 18, 2025**.
- Turn count: **56 Prompt/Response pairs** (grepped `## Prompt:` / `##
  Response:` — lines 9, 40, 50, 62, 351, 354 … 3651, 3665). Several Responses
  are themselves ChatGPT "Deep Research" runs (`Research completed in Xm · Y
  sources · Z searches`) that each generate a multi-thousand-word structured
  sub-document, so turn count understates content volume by roughly 5-10x.
- Ends mid-task (line 3704, unfinished): owner has just pasted a contacts CSV
  and GPT is proposing a name/number-to-message entity-tagging pass that never
  completes in this file.

## Structure & format
- **Turn delimiter:** `## Prompt:` and `## Response:` as literal Markdown H2
  headers, strictly alternating. This is the ChatGPT-web-export-to-markdown
  convention (same as the other files in this sample set) — reliable regex:
  `^## (Prompt|Response):$`.
- **No per-turn metadata** (no timestamps, no model-name) — only the one
  file-level metadata block at the top has Created/Updated/Exported dates.
- **Deep Research sub-turns:** several Responses open with `Research completed
  in Nm · N sources[· N searches]` immediately followed by a full report using
  Markdown **Setext headers** (`Title\n====` / `Title\n----`) and numbered
  `### N. **Bold Title**` subsections — a visually distinct "report" register
  nested inside a normal Response turn. A parser must NOT treat these as new
  turns; they're structurally still inside one `## Response:` block.
- **Code-interpreter sub-turns:** later in the conversation (from ~line 1379
  onward) Responses contain fenced ` ```python ` blocks followed by bare
  status lines — `Analyzed`, `Result`, `Analysis paused`, `Analysis errored`,
  `undefined` — which are ChatGPT's code-interpreter tool-call/tool-result
  pairs flattened into the markdown export with **no code fences around the
  output** and no indication of what "undefined" actually returned (the
  original UI must have rendered images/files/tables that didn't survive
  export). PARSING NOTE: any extraction pipeline needs a distinct rule for
  "```python ... ``` / Analyzed / Result / undefined" quads — they are not
  prose and should route to a code/tool-use lane, not the narrative lane.
- **Inline citations:** two distinct citation styles coexist:
  1. `[Google Drive](https://drive.google.com/file/d/...)` /
     `[Google Drive](https://docs.google.com/document/d/...)` — links to the
     owner's own uploaded source documents (declarations, motions, chat
     exports). These are the dominant citation type and are effectively
     **evidence provenance pointers** — extraction should treat them as
     "cites owner-uploaded doc X" edges, not as external authority.
  2. Real external URLs (`michiganlegalhelp.org`, `stangelawfirm.com`,
     `attorneyatlawmagazine.com`, `walterslawgroup.com`) — genuine open-web
     research citations from the Deep Research tool, concentrated in the
     "cell tower data" section (~line 779-807).
- **Emoji-prefixed section headers** appear from roughly line 1398 onward
  (✅ 🔍 ⚖️ 📦 🛑 🔐 🧩 💣 🔒 🧠) as the GPT persona shifts into a more
  informal "advocate/strategist" register mid-conversation, distinct from the
  earlier formal-legal-memo register.
- **Embedded artifacts:** Markdown tables (timeline tables, evidence-index
  tables, a Kristen/support-network tracking table), one raw pasted CSV/table
  snippet from an SMS export, and drafted legal-document excerpts
  (declaration paragraphs, motion language) rendered as blockquotes.

## Section-by-section breakdown (in order)
1. **Setup (L1-79):** Owner uploads 7 prior ChatGPT export files (Summary
   Consolidation, Custody Case Opposing Party, Custody Case Strategy, Legal AI
   Custody II/III, Legal Assistance Overview, Personality Disorders Outcomes)
   and asks — in raw, profane, emotionally unfiltered language — for one
   master document ahead of a Legal Aid phone consult. GPT asks clarifying
   questions (structure, tone, deadline); owner answers (Tuesday TRO filing
   deadline, tone toned down but not sanitized).
2. **First master-document generation (L81-350):** A full "Custody and TRO
   Master Case Document" — Timeline of Events (2018-2025), Case Summary,
   Evidence Index (Attachments A-E: 2019 text log, 2018-2024 FB Messenger
   export, Sept 2023 Google Voice texts, Nov 2024 WhatsApp export, phone call
   log incl. a 911 call), Behavioral & Legal Analysis mapped to MCL 722.23
   best-interest factors, a Filing & Strategy Plan (TRO/PPO, emergency
   custody, permanent modification), and Supporting Document Notes.
3. **Iterative regeneration cycles (L351-900):** Owner repeatedly uploads more
   files ("bunch more documents… double check you got all of them") and GPT
   re-runs Deep Research passes, each producing a near-duplicate but
   expanding master packet (adds relocation threat, medical/educational
   neglect — delayed vaccines, dropped autism evaluation, ER visit hidden from
   father — and school emergency-contact removal).
4. **Strategy pivot after ex parte denial (L702-900):** Owner reports the ex
   parte TRO was denied; strategy shifts to an expedited noticed hearing.
   Introduces a **cell-tower-location-data plan** to prove the mother's
   cohabitation/relocation risk, citing real external sources (Stange Law
   Firm, Walters Law Group, Attorney at Law Magazine) on obtaining carrier
   call-detail records via subpoena.
5. **Table-format master packet (L840-1200):** Another full regeneration,
   this time as Markdown tables — Timeline table, Case Background, Evidence
   Index (adds Public Declaration, an **In-Camera Declaration of Sexual
   Trauma**, the actual filed Emergency Ex Parte Motion draft, a "Strategic
   Case Analysis and Litigation Plan" memo, and AI-assisted research notes
   like "Proving Narcissistic Co-Parent Abuse"), Legal Analysis (5 numbered
   issues citing MCL 722.23/722.31, MCR 3.207, *Ireland v. Smith*), and a
   5-motion Strategic Motion Plan.
6. **Export/tooling thread (L1373-1500):** Owner asks for the master doc as
   clean Markdown, then Google Docs + CSV (Notion/Airtable) + a proposed
   folder layout (`/declarations`, `/text_messages`, `/draft_motions`). First
   clear code-interpreter usage (`pathlib.Path` file read/preview).
7. **Deeper 2017-2025 narrative rebuild (L1500-1700):** New pass adds far more
   explicit detail — the "Jake" affair discovery (Oct 25, 2024, 1-3:30 AM text
   barrage), a Dec 7, 2024 exchange in which the mother threatens to "beat the
   living shit out of" the father and he consents ("if you need to beat my
   ass… so be it"), a Feb 2025 "mailbox incident" (alleged drunk driving with
   the child), and earlier 2023 incidents (mother allegedly breaks her ankle
   at a party with drugs present, then two weeks later falls down stairs
   holding the child, chipping the child's tooth).
8. **Live-dictated raw narrative + sensitive-evidence handling (L1704-2100):**
   Owner free-associates a long, expletive-heavy account of May 2024 (police
   sent to his grandmother's house, the Jake/Adam/Dustin overlap, a detail
   about the mother invoking his late mother's memory/song, a hotel night
   followed by what he calls a "drug run" — an alleged Molly/Adderall
   purchase with the child in the car and a photo of pills "in her mouth").
   GPT reframes each disclosure into court-safe declaration language in
   real time. This section also directly confronts **evidentiary risk**: the
   owner admits key photos were "kind of hacked" (pulled from the child's
   phone, logged into the mother's account); GPT walks through MRE
   1101/family-court admissibility flexibility, recommends in-camera review
   instead of public filing, and drafts a declaration paragraph justifying
   the access as protective, not retaliatory.
9. **Owner's own substance-use framing (L1966-2064):** Owner discloses a
   **bipolar disorder diagnosis** and a family history of alcoholism (his
   mother's), frames his own substance use as functional/coping vs. the
   mother's as reckless/endangering, and GPT drafts declaration language
   distinguishing the two for the custody motion.
10. **"Isolation of support network" arc (L2066-2300):** A detailed account of
    "Kristen" (owner's ex-sister-in-law/best friend) being triangulated,
    weaponized, and cut off — including a stillbirth detail and an owner
    quote allegedly showing the mother later admitted the conflict was staged
    ("I need my bitches"). GPT proposes this as a standalone legal-strategy
    section plus a reusable table (Support Person → Tactic → Timeline →
    Outcome) that generalizes the pattern.
11. **Repeated full chronological rebuilds, 2019-2024 (L2400-3020):** GPT
    regenerates the year-by-year "Case Narrative" multiple times in "live
    build mode," with the owner correcting/adding facts turn by turn
    (introduces "Adam #1/#2," clarifies two different "Jakes," the 2019
    "flaccid dick" smear backstory, month-by-month 2023 detail including a
    "Monica/pie incident"). Code-interpreter used again: extracts a zipped
    case-law PDF (*Hallman v. Mendoza*, 2016 Mich. Cir. LEXIS 2049) and
    generates an actual **.docx via `python-docx`** (`Document()`,
    `doc.add_heading`, `doc.save(...)`) for a "Custody Case Master Summary &
    Strategy Packet." Owner reports losing the file because the ChatGPT
    sandbox link expired before he could download it; GPT commits to always
    printing full text inline going forward, not just a download link.
12. **Raw data-processing finale (L3104-3704):** Owner pastes raw phone-export
    tables directly into the chat, then uploads three files — `sms_export
    Matt & Katrina.csv` (from the mother's phone, 2024), `convo_dustin.csv`,
    and `convo_matt.csv`/`.tsv` (his own "burner" thread, used to get around
    being blocked). GPT uses **pandas** to load/normalize all three
    (handling delimiter and encoding mismatches across retries), merges the
    two Matt↔Katrina threads into one chronological CSV
    (`Matt_Katrina_AllConvos.csv`), then writes a **keyword/regex auto-tagger**
    (`tag_message()`, tagging `court_threat`, `alienation_excuse`,
    `love_bombing`, `rage_attack`, `gaslighting`, `self_victimization`) to
    produce `Matt_Katrina_AllConvos_Tagged.csv` for Notion import. The file
    ends mid-task with a contacts-list upload for a planned
    name/number-to-message entity-tagging pass (Adam, Jake, Dustin, Kristen,
    Miles, Monica, Anaya) that is never completed in this conversation.

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity (who) | yes | Matthew Salem (owner/petitioner; GPT project persona name "Error 404 Fuks Not Found"); Katrina Kinzel (respondent/mother); daughter Kailah (DOB given inconsistently across passes — 8/2020 in one master doc, 1/31/2020 in later ones); ~10 named third parties (Adam ×2 distinct people, Jake ×2 distinct people, Dustin Church, Kristen, Monica, Anaya, Miles, Michael/"Mikey," Nikki, Jade, Austin) |
| entities | yes | phone numbers incl. owner's "burner" numbers used to evade blocking; Google Drive/Docs doc-IDs cited as evidence provenance; case caption *Salem v. Kinzel* (Genesee County); statutes MCL 722.23 / 722.27a / 722.31, MCR 3.207; case law *Ireland v. Smith*, *Hallman v. Mendoza* |
| relationships | yes | central to the whole file — romantic/co-parent history, alleged infidelity web (Adam, Jake, Michael, Dustin), and a full "isolation of support network" analysis of the Kristen relationship |
| timeline/events | yes (dominant) | rebuilt end-to-end at least 5 separate times at increasing granularity (year-level → month-level → day-level for hot spots like May 2024 and Dec 7, 2024) |
| life-history | yes | 7.5-year relationship arc from ~2018 to present; owner's own history (mother's alcoholism and death, a song tied to her final apology letter) |
| legal-strategy | yes (dominant) | TRO→PPO framing debate, ex parte→expedited-hearing pivot after denial, cell-tower-data evidentiary tactic, in-camera review strategy for sensitive/"hacked" evidence, best-interest-factor-by-factor argument mapping |
| legal-artifacts | yes | drafted/referenced: Public Declaration, In-Camera Declaration of Sexual Trauma, Emergency Ex Parte Motion (filed, denied), proposed orders, lettered evidence index (Attachments A-I across passes) |
| mood/sentiment (owner-self, low-pri) | yes | owner's voice is raw, profane, emotionally dysregulated throughout, especially in dictated passages (L38, L1705, L2592 "oh my motherfucking God…give me some fucking documents") |
| psychiatric | yes | owner discloses **bipolar disorder** diagnosis + family alcoholism history + self-described "mental breakdown" (Feb 2025) + trauma/codependency framing of his own appeasement behavior (L1622-1635); repeatedly attributes narcissistic/personality-disorder and Borderline-type "idealization↔devaluation" traits to Katrina (L1574) |
| code/app-dev/plans | yes — the strongest example in the sample | Concrete ChatGPT code-interpreter turns: `pathlib.Path` file export (L1379-1388); zip extraction of a case-law PDF (L2182-2212); **.docx generation via `python-docx`** (L2307-2378); **pandas** CSV/TSV load-and-merge across 3 phone-export files with delimiter/encoding retry logic (L3252-3524); a hand-rolled **keyword/regex auto-tagger** function classifying message sentiment/behavior categories (L3566-3608); plus explicit document-handling strategy discussion (Notion/Airtable CSV exports, folder layout for attachments, `timeline.json` mention at L2462) and a load-bearing workflow-friction note — owner lost a generated file because "the sandbox closes" before download (L2408) |
| work | minimal | only as evidence of stability (18-hour workdays cited to support custody argument), no professional/work-product content |

## Notable content
- **Evidence pointers:** Sept 9, 2023 Google Voice text barrage (16 messages,
  threatening to withhold the child), Nov 21, 2024 WhatsApp confrontation,
  Dec 7, 2024 violence-threat exchange, Feb 7, 2025 911 call, three raw
  phone-export files merged/tagged at the very end of the file.
- **Drafted legal artifacts referenced/summarized:** Emergency Ex Parte Motion
  (filed, denied), Public Declaration, **In-Camera Declaration of Sexual
  Trauma**, Strategic Case Analysis and Litigation Plan memo.
- **Key dates:** 8/2/2025 (conversation start) through 8/18/2025 (last
  update); real-world case events run 2018-2025; TRO filing deadlines cited
  as Feb 18, 2025 and Aug 5, 2025 in different passes (the document was
  regenerated across multiple real filing attempts).
- **Decisions/pivots:** ex parte TRO denied → pivot to expedited hearing +
  standalone TRO; plan to seek cell-tower subpoena in person at the hearing;
  explicit decision to keep certain evidence to in-camera-only review rather
  than public filing because of how it was obtained.

## Sensitivity
LABEL, do not redact (per research policy). This file contains, in the
owner's own words:
- **Explicit sexual content and a declaration specifically about sexual
  trauma** (In-Camera Declaration of Sexual Trauma; a "flaccid dick" smear
  narrative; nude-photo-exchange details).
- **Psychiatric self-disclosure:** owner's bipolar diagnosis, family
  alcoholism/death history, self-described "mental breakdown," and
  substance-use admissions (his own, framed as functional/coping, and
  extensive allegations against Katrina).
- **Domestic-conflict/threat-of-violence content:** the Dec 7, 2024 exchange
  in which the mother threatens to have someone "beat the living shit out of"
  the father and he consents.
- **Potentially improperly-obtained evidence:** owner admits certain photos
  were "kind of hacked" from the child's phone/mother's account; the file
  itself documents GPT's legal-risk coaching around this.
- **Minor-child safety allegations:** alleged substance purchase/use with the
  child in the vehicle, an alleged injury to the child (chipped tooth) during
  a fall.
- **Pervasive profanity** throughout, especially in owner-dictated passages.
