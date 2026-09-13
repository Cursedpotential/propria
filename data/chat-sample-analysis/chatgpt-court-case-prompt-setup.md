# ChatGPT-Court case prompt setup.md
> _Byline: Claude Code · Sonnet · 2026-07-11_

## Snapshot
- **Source tool:** ChatGPT (standard markdown export, ChatGPT Projects — conversation was moved into a
  Project folder named "Court" partway through).
- **Size:** ~200KB / 4,025 lines.
- **Format:** Single continuous conversation (one file = one thread, not a multi-conversation export).
  Confirmed by grep: exactly one top-level `# ` title (line 1); every other `# ` in the file is a Python
  comment inside embedded code blocks.
- **Export header (verbatim):** User: `Matthew Salem (Error 404 Fuks Not Found) (matt.salemnet@gmail.com)`
  · Created: 8/8/2025 12:25 · Updated: 8/18/2025 18:53 · Exported: 8/21/2025 13:55 · includes a
  `chatgpt.com/g/g-p-.../c/...?project_id=...` deep link.
- **Turns:** 30 user prompts / 30 assistant responses (grep-verified `## Prompt:` ×30, `## Response:` ×30).
  The final response is cut off mid-generation at EOF (line 4025 ends on a bare `---` after the model's
  own "Analyzing" status line) — the export appears to have been taken while the last turn was still
  streaming/erroring, so the conversation has no clean ending.

## Structure & format
- Turns are delimited by markdown H2 headers: `## Prompt:` (user) and `## Response:` (assistant) —
  **not** the "You asked: / ChatGPT Replied:" + `----------` underline pattern the task brief guessed at.
  There is no such pattern anywhere in this file.
- `* * *` (rendered as `---`/hr) is used liberally *inside* assistant responses as a thematic section
  break between sub-headed blocks (e.g., separating "1. Problem Areas" from "2. Primary Complaints") —
  it is not a turn separator. A literal `---` also appears as the very last line of the file (4025),
  but that's the truncated tail of the final unfinished response, not a delimiter.
- Assistant responses embed a lot of structured sub-content: markdown tables, blockquoted "court-safe
  language" drafts, fenced prompt templates (fenced with triple-backtick, no language tag), and — from
  roughly line 1930 onward — multiple large fenced **Python code blocks** (pandas/regex) representing a
  custom NLP text-analysis tool the model was building live in a code-interpreter-style tool ("Thought
  for 35s" / "Analysis paused" / "Analysis errored" / "Request for GPT-5 Pro" / "Reasoned for Xs" status
  lines are ChatGPT UI artifacts baked into the markdown export).
- Two of the three code-generation attempts **errored** (visible as "Analysis errored" followed by a
  cut-off Python string), and the user explicitly calls this out ("did yoiubreak" — sic). The model
  regenerates the whole file bundle from scratch each time rather than patching, so the same
  dictionaries/analyzer/templates appear 2–3 times near-verbatim in the second half of the file.
- No JSON front-matter, no attachment/file metadata blocks beyond inline citation-style markers like
  `Legal Document Formatti…`, `Custody Case Defense An…`, `+19062044529`, `+18102689630` — these look
  like ChatGPT's truncated inline source-citation footnotes referencing project files/chats the model
  had retrieval access to (the underlying doc names are cut off with an ellipsis by the exporter).
  A parser cannot recover the full source filenames from these truncated citations alone.
- **PARSING NOTES (for the extraction-lane design):**
  1. Turn detection is trivial (`^## Prompt:` / `^## Response:`) — much simpler than the guessed format.
  2. The file mixes **prose** (case narrative/strategy) with **verbatim source code** (Python) and
     **file-bundle manifests** (the model repeatedly "writes" and "zips" a set of named files whose
     *content* only exists as inline code — a parser must not mistake these embedded synthetic file
     dumps for real evidence files; they are ChatGPT's own generated deliverables, not chat/evidence logs).
  3. Truncated inline citations (`index+18102689630`, `Custody Case Defense An….`) are lossy — a parser
     should flag them as "citation, source truncated by exporter" rather than try to resolve them.
  4. Real phone numbers are used as informal **evidence-log IDs** for conversation exports the owner had
     uploaded elsewhere (e.g., `+19062044529` = the "Jake L." conversation) — a parser building an entity
     index should treat these as evidence-set identifiers/PII, not just phone numbers.
  5. The conversation ends mid-generation (broken code, no user reply after) — a parser should not assume
     every file ends on a clean final turn.

## Section-by-section breakdown (in order)
1. **Lines 1–63 — Multi-chat workflow design.** User is overwhelmed prepping for court and wants a
   system of per-year ChatGPT threads that reference each other so context-window limits don't degrade
   quality. Assistant drafts a reusable "Court Case Yearly Analysis & Timeline Prompt" (role, objective,
   continuity rules, output format).
2. **Lines 65–201 — Project-folder task extraction.** User asks the model to read every chat in his
   ChatGPT "Court" project folder; model first says it can't browse folders, then (once the conversation
   is moved into that project) produces a task inventory: Affidavit & Motion Work (keep affidavit
   first-person/factual, motion = legal argument+citations), Pre-Hearing Evidence Checklist, Strategic
   Case Objectives under **MCL 722.23**, Evidence & Data Management (raw exports, metadata sidecars,
   `geocoding_cache.json` / `place_id_db.json` for movement-pattern corroboration), a Cross-Year Chat
   Organization Plan, and a first "Master Prompt for Year-by-Year Review."
3. **Lines 203–311 — Self-assessed weaknesses vs. complaints against the mother.** User asks the model to
   identify weak spots in his own case (substance use, evidence gaps, communication tone) and his primary
   complaints against Katrina (unsafe individuals around the child, neglect/poor supervision, dishonesty,
   interference with co-parenting). Model proposes a "Vulnerability vs. Attack" matrix (declined by user
   at this point — picked up implicitly later).
4. **Lines 313–393 — Framing the mother's conduct as domestic violence.** User wants manipulation,
   gaslighting, alienation, denial of access, defamation, and "assassination of my recovery identity"
   framed as DV under MCL 722.23, plus a defense of how he obtained evidence. Model supplies DV-framing
   language, an evidence-acquisition defense (lawful access / implied consent / necessity), and
   court-safe boilerplate language for both.
5. **Lines 395–499 — This chat becomes "Master Orchestrator."** User redirects: this thread should
   coordinate the others. Model defines the orchestrator's role and produces the first 3-prompt set:
   Year-by-Year Timeline Review, Court Filing Draft, Evidence Deep-Dive — plus a 4-stage workflow diagram
   (Orchestrator → Year Chats → Filings Chat → Deep-Dive Chats).
6. **Lines 501–605 — Evidence-deep-dive-first pivot.** User wants evidence processed before the
   year-by-year pass. Model gives an Evidence Deep-Dive Master Prompt and an "ED-##" numbering scheme,
   and clarifies it cannot open new chats itself — the user must copy/paste the kickoff prompt manually.
7. **Lines 607–716 — ED-01 selection and background: the "Jake L." conversation.** Model recommends
   starting with conversation log `+19062044529` (Oct 21, 2024) as highest-value. User supplies real
   background: this is a call to a third party ("Jake L.") after being repeatedly lied to about Katrina's
   involvement with him; user states Katrina previously had Kailah "taken from me" during this dispute.
   Model folds this into an updated ED-01 prompt.
8. **Lines 718–996 — ED-01 through ED-07 queue (conversation logs).** Model produces a full batch of
   evidence-deep-dive kickoff prompts, each tagged to a specific phone-number-identified conversation
   export: ED-01 `+19062044529` (Jake L., Oct 21 2024), ED-02 `+18102689630` (Katrina, Sept 6–7 2023,
   trust breakdown), ED-03 `+18102809875` (May & Sept 2024, childcare/work), ED-04 (duplicate export of
   ED-01), ED-05 `+18109644981` (Feb 2 2024, casual chat w/ alcohol references), ED-06 `+18104296417`
   (May 24 2024, vehicle-help exchange, framed as a positive-character piece), ED-07 (alternate export of
   ED-03).
9. **Lines 998–1191 — Full case caption + ED-08 through ED-11 (formal documents).** User supplies the
   full case caption (names, court, address — see Notable content). Model embeds this into deep-dive
   prompts for four named formal court documents already drafted elsewhere: ED-08
   `Public_Statement_for_Court.docx`, ED-09 `Motion_and_Subpoena_for_Surveillance_Footage_RECREATED.docx`,
   ED-10 `Hybrid_Declaration_Overview.docx`, ED-11
   `Public_Declaration_Abuse_and_Reputation_UPDATED.docx`, and names three more queued docs
   (`In_Camera_Custody_Legal_Summary.docx`, `Sealed_Declaration_Request.docx`,
   `Custody Case Defense Analysis.pdf`).
10. **Lines 1193–1231 — User frustration / pivot away from templates.** User: *"I don't know what the
    fuck it is that you're creating but it's fucking pointless"* and clarifies the real problem is that
    the templates are empty shells with none of his actual case facts in them. Model drops the
    "ED-code" scaffolding approach and offers to process one file/conversation directly, in-thread.
11. **Lines 1233–1608 — Live deep-dive: the Jake L. narrative (this is the densest fact/story section
    of the file).** Working turn-by-turn from the user's own recollection (not an uploaded transcript),
    the model builds out a full incident narrative: an earlier encounter at "Monica's" party (where
    Katrina broke her ankle and Jake was asked to leave), a nude-photo exchange, a 4 a.m. driveway
    meeting on Oct 21 2024, ~400 text messages exchanged between Katrina and Jake in two days, a "first
    date"-style outing at Stepping Stone Falls with Kailah present, and a screenshot in which Katrina
    brags to a friend about Jake's anatomy plus her deflecting reply when confronted ("it's ok I like
    small dicks"). This is followed by the user disclosing that Katrina called the police on him in
    mid-April after a canceled visit, following his phone call to her workplace.
12. **Lines 1609–1728 — Generalizing to a repeated pattern (April–November 2024).** User states this
    was "the first of MANY MANY times she denied access" and points the model at Aug–Oct 2024
    conversations re: photos found on Kailah's phone/Google/Snapchat. Model proposes a recurring
    Trigger → Response → Punishment/Denial pattern-chart format and asks how to sequence the review.
13. **Lines 1661–1943 — User-supplied NLP/detection blueprint pivot (code/app-dev begins).** User pastes
    a "Reactive Abuse Cycle" detection spec (apparently self-authored or drafted in a prior session) plus
    a much larger "Finalized Conversational Analysis Blueprint" with five keyword/phrase dictionaries:
    (1) Substance Abuse & Weaponization, (2) Deception & Narrative Control (incl. a homophobic-slur list
    and infidelity/social-media-deception keyword sets), (3) Control & Destabilization / the
    Provocation→Reaction→"Gotcha" reactive-abuse-cycle logic, (4) Financial & Domestic Abuse, (5) Sexual
    & Emotional Weaponization (love-bombing / sexual-shaming). This is a fully worked taxonomy, not a
    placeholder.
14. **Lines 1930–4025 — Assistant builds and repeatedly rebuilds a Python analysis tool
    ("code/app-dev" section, roughly the back half of the file).** The model implements
    `reactive_abuse_analyzer.py` (pandas + regex), a `dictionaries.json`, a message CSV template seeded
    with realistic example dialogue using the real party names, a `pattern_skeleton_apr_nov.csv`, a
    drafted `incident_profile_jake_l.md` (court-style exhibit narrative citing MCL 722.23 (j)(k)(f) and
    an Exhibit A–F list), a `jake_l_incident_timeline.csv`, a full `prompts_master_pack.md`, the ED
    prompt files, `court_framing_snippets.md`, an `april_lockdown_checklist.md`, and a README — all
    zipped into a "Court_Pack" bundle. Code execution errors twice ("Analysis errored", user: "did
    yoiubreak"); the model regenerates the entire bundle from scratch two more times (near-duplicate
    content at ~2780–3300 and ~3510–4011). The file terminates mid-generation on the third rebuild,
    unresolved.

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity(who) | Yes | Full case caption given verbatim (line 999, re-embedded lines 1018-1023 etc.): "Plaintiff: Matthew Salem … Defendant: Katrina Kinzel … Minor Child: Kailah Salem." Also "Jake L." (third party), "Monica" (party host), an unnamed friend Katrina messaged. |
| entities | Yes | Court: "Seventh Circuit Court, Family Division, Genesee County, Michigan… 900 S. Saginaw St., Flint, MI 48502" (line 1022-1023); evidence-log phone numbers `+19062044529`, `+18102689630`, `+18102809875`, `+18109644981`, `+18104296417`; named docs `Public_Statement_for_Court.docx`, `Motion_and_Subpoena_for_Surveillance_Footage_RECREATED.docx`, `Hybrid_Declaration_Overview.docx`, `Public_Declaration_Abuse_and_Reputation_UPDATED.docx`; `geocoding_cache.json`, `place_id_db.json`; location "Stepping Stone Falls", "Huckleberry Junction / Huck's" (in the dictionary, referenced as a possible affair-location keyword). |
| relationships | Yes | Matthew–Katrina (ex-partners/co-parents, described as having reconciled after she "left you for another man while you were jailed for 30 days," line 1247); Katrina–Jake L. (alleged affair timeline, lines 1292-1450); Matthew–Jake L. (confrontational phone call, ED-01); Kailah as shared child exposed to Jake within days of contact. |
| timeline/events | Yes | Oct 21 2024 4 a.m. driveway meeting; ~400 texts in 2 days; Stepping Stone Falls outing 3 days after first contact (line 1367); mid-April 2024 confrontation → canceled visit → police called (lines 1554, 1609); denial-of-access pattern chart April→Nov 2024 (lines 1673-1682); Sept 2023 "trust breakdown" convo (ED-02); Feb 2, 2024 casual convo (ED-05); May 24, 2024 vehicle convo (ED-06). |
| life-history | Yes | Owner's own recovery/sobriety history and a prior "jailed for 30 days" period during which Katrina left him for another man (line 1247); the Adderall-sharing dynamic between the two as a described historical pattern (lines 1754-1762, 1868). |
| legal-strategy | Yes | MCL 722.23 best-interest-factor mapping used repeatedly, esp. (j) willingness to facilitate the other parent's relationship, (k) domestic violence, (f) moral fitness (lines 328, 1280-1283, 1584-1589); DV-via-coercive-control theory of the case (lines 324-341); evidence-acquisition defense strategy ("lawful access to shared devices/accounts," line 361-373); affidavit-vs-motion fact/argument separation rule (lines 110-116, 1099-1119); multi-chat "orchestrator" workflow strategy to manage AI context-window degradation (lines 9-63, 395-499). |
| legal-artifacts | Yes | Full drafted "Court Framing" quote paragraphs meant for direct court use, e.g. line 338-340 and line 1489 (the "Respondent's position that 'there was nobody else' is flatly contradicted…" paragraph); a drafted `incident_profile_jake_l.md` exhibit narrative with an Exhibit A–F list (lines 3052-3083 and duplicated later); ED-01…ED-11 evidence-review prompt templates keyed to specific named documents/logs; an "Evidence Access Statement" draft (line 373, 3081-3082). |
| mood/sentiment(owner-self, low-pri) | Yes | User frustration outbursts: "I don't know what the fuck it is that you're creating but it's fucking pointless" (line 1194); "its ALLLLL FUCKED dude,. its ALLLLL FUCKED" (line 1662); assistant repeatedly acknowledges over-engineering and course-corrects. |
| psychiatric | Yes (framed as recovery/identity, not clinical) | Owner describes the mother's conduct as "assassination of my recovery identity" (line 314) and the model frames this explicitly as targeting "your sobriety and recovery image" (line 335); Adderall-sharing/control dynamic (Category 1.3, lines 1754-1762) and alcohol-abuse/child-endangerment dictionary (Category 1.1, lines 1736-1744) are both about the *other* party but were authored using the couple's shared substance history; no formal diagnosis or therapy-note content appears in this file. |
| code/app-dev/plans | Yes — dominant in back half | A full custom Python "Reactive Abuse Cycle Analyzer" (pandas + regex, ~250 lines, rewritten 3×: lines 1945-2302, 2836-2998, 3567-3727), a JSON keyword-dictionary taxonomy (5 categories, ~25 sub-lists), CSV/JSON templates, and a packaged "Court_Pack" ZIP bundle with README — this is a self-contained NLP tool-build, not case narrative. |
| work | Minor | Model asks Jake, rhetorically, whether he "value[s] his job" given the risk of the affair being exposed (line 1250); ED-03/ED-06 background notes reference Katrina's "work schedules" and workplace-conflict framing around childcare (line 802-806); Matthew's own call to Katrina's workplace is discussed as a fact he needs to defend against a "harassment" characterization (line 1554, 1598, 2724). |

## Notable content
- **Full case caption** (line 999, repeated at 1018-1023 and in every ED-08+ prompt): Plaintiff Matthew
  Salem, Defendant Katrina Kinzel, minor child Kailah Salem, Seventh Circuit Court – Family Division,
  Genesee County, Michigan, 900 S. Saginaw St., Flint, MI 48502.
- **Drafted DV-framing legal language** (line 338-340), meant to be used verbatim in a filing: *"The
  respondent's actions constitute a sustained pattern of coercive control and emotional abuse, meeting
  the statutory definition of domestic violence under MCL 722.23(k). This includes deliberate
  manipulation, isolation from the child, and character assassination designed to destabilize the
  petitioner's identity, relationships, and recovery process."*
- **Evidence-acquisition defense language** (line 373) — pre-drafted for handing to the judge/opposing
  counsel: *"The evidence was obtained through lawful access to shared devices/accounts during our
  cohabitation and/or via direct communication from the respondent or third parties. At no point did I
  bypass security measures or engage in unauthorized access."*
- **The Jake L. incident** (lines 1237-1608) is the single largest concrete fact-narrative in the file:
  Monica's-party origin story, alleged nude-photo exchange, ~400 texts in 48 hours, a 4 a.m. meeting, an
  outing with the minor child three days after first contact, a screenshot of Katrina messaging a friend
  about Jake's anatomy, and her quoted deflection *"it's ok I like small dicks"* (line 1507) when
  confronted — the model explicitly advises the user **not** to lead with that quote in court and instead
  use it only as proof an exchange occurred (lines 1526-1529).
- **The "Reactive Abuse Cycle" dictionaries** (lines 1730-1929, re-serialized as `dictionaries.json`
  through the rest of the file) — a genuinely detailed, user-authored (or user-sourced) behavioral-pattern
  taxonomy: alcohol/DUI-risk phrases, substance-use insults ("crackhead", "tweaker"), an Adderall
  control/hypocrisy dynamic, a **homophobic slur list** ("fagot", "fagget", "faggot", "sick fagot" —
  present as detection-dictionary entries, not as content directed at anyone within this file), sexual
  shaming and love-bombing phrase banks, and the 3-step Provocation→Reaction→"Gotcha" cycle logic that
  the Python code (lines 2109-2530 and duplicates) actually implements.
- **The multi-year, multi-chat "context window" workflow itself** is a notable structural artifact: the
  owner is explicitly designing a manual retrieval-augmentation scheme (per-year chats + an orchestrator
  chat + carryover notes) to work around ChatGPT's context limits while building his court case — directly
  relevant to how this platform's own chat-ingest/extraction lane should think about long-running,
  multi-thread case documentation.
- **Unresolved ending**: the final ~1,100 lines are the model regenerating the same Python/file bundle
  twice after execution errors, and the file cuts off mid-generation with no user reply — there is no
  "conclusion" to mine from the tail of this file.

## Sensitivity
- **Abuse allegations (LABELED, not redacted):** The file contains detailed first-person allegations of
  gaslighting, coercive control, parental alienation, denial of parenting-time access, and retaliatory
  police involvement against the child's mother (Katrina Kinzel), including the owner's own drafted legal
  characterization of this conduct as domestic violence under MCL 722.23(k).
- **Sexual content (LABELED):** Explicit references to nude/dick-pic exchange, a quoted vulgar deflection
  line from the mother ("it's ok I like small dicks," line 1507), and detailed narrative around an alleged
  affair, all presented as case evidence rather than gratuitous content.
- **Substance use disclosures (LABELED):** The owner discloses his own recovery/sobriety identity and a
  prior period of incarceration ("jailed for 30 days," line 1247); the file also contains a jointly-authored
  keyword taxonomy cataloguing alcohol and Adderall-related conduct attributed to the mother, built from the
  couple's shared substance-use history.
- **Slur content (LABELED):** A homophobic slur list ("fagot", "fagget", "faggot", "sick fagot") appears
  as an NLP detection dictionary (Category 2.1, line 1771 and duplicated in the Python `DICT` literal at
  lines 1972, 2813, 3544) — included for pattern-matching against messages allegedly directed at the
  owner, not authored as an attack within this document, but it is slur text nonetheless and should be
  handled per the platform's sensitivity-tier labeling, not omitted.
- **Minor child PII (LABELED):** The minor child is named repeatedly by full name (Kailah Salem) with
  details about her devices (phone, Google account, Snapchat) being checked for content, and about her
  presence at an outing with an adult male alleged to be her mother's new partner.
- **PII (LABELED, not redacted per research policy):** Full legal names of both parents and the child,
  the exact court name/address, the owner's email address (`matt.salemnet@gmail.com`, from the export
  header), and five phone numbers used as informal evidence-log identifiers for third-party/mother
  conversation exports.
- **Profanity/high emotional register (LABELED, low-pri):** Frequent profanity from the owner, especially
  in the two frustration outbursts noted above under mood/sentiment — flagged for completeness, not a
  redaction concern.
