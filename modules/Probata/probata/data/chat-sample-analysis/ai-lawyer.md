# AI Lawyer (4).md
> _Byline: Claude Code · Sonnet · 2026-07-11_

## Snapshot
- Source tool / export style: custom ChatGPT GPT called **"AI Lawyer"** (branded, ad-supported —
  footer links to StartupTechLaw.com on every turn), captured via a Chrome extension
  (`chrome-extension://oibghjgooccojibfacdonaoipegckdeg`) that also injects
  "Create Google Docs / Word Docs / PDF Docs" export-button labels into the scrape.
- Size: 220 KB · 3,944 lines · single continuous conversation, no session breaks.
- Format: Markdown, `.md`.
- Era/date: undated in-file (no title/timestamp header at all — unusual vs. the other two
  files in this batch, which both have export metadata blocks).
- Turns: ~30 user/assistant exchanges, but individual assistant turns are very long
  (multi-hundred-line drafts), so total content is dense.

## Structure & format
- Delimiter pattern: `You asked:\n----------\n<user text>\n\n---\n\nChatGPT Replied:\n----------------\n<assistant text>\n\n---`.
  This is a **distinct pattern from the other two files in this batch** — neither the
  `**User:**/**Response:**` header-block style (file 2) nor the `**You:**/**Gemini:**`
  bold-inline style (file 3).
- Every single assistant turn opens with an identical boilerplate **NOTICE OF LEGAL
  DISCLAIMER** paragraph (custom-GPT system framing, not user content) and closes with
  an identical **"Visit www.StartupTechLaw.com"** ad + three export-button image/alt-text
  lines. Both are pure noise for extraction and should be stripped by a parser.
- Embedded artifacts: no code. Several user turns are **lists of Google Drive file links**
  (pasted from a "share multiple files" UI action) — each entry is `[filename.docx\n\nGoogle Docs\n...]\n(drive URL)`, representing `.docx` files the user uploaded to the custom GPT's
  knowledge/context (not embedded content — the actual document text is never inline,
  only filenames + Drive URLs). One turn also references "Hybrid Litigation Strategy
  Blueprint" and "Case Summary Overview" as "Unable to display visualization" — the GPT
  attempted to render a chart/diagram that didn't survive export.
- PARSING NOTES:
  - Strip the disclaimer header and ad footer from every assistant turn before
    extraction — they're ~9 lines of fixed boilerplate repeated ~30 times (~15% of file
    bytes are pure noise).
  - The Google-Drive-link turns are **pointers to external documents, not content** — a
    parser should capture them as a filename inventory (evidence of a drafted-document
    corpus that exists outside this chat, on Google Drive) rather than try to extract
    prose from them.
  - No timestamps anywhere in the file — turn ordering is the only sequencing signal.
  - Heavy dictation-artifact text (run-on, punctuation-free, transcription errors like
    "Jackson County" later self-corrected by the user to "Lapeer County," "adjacent"
    misheard for a county name) — this is voice-dictated input to a mobile/desktop STT,
    not typed. A parser doing entity/date extraction needs fuzzy correction awareness.

## Section-by-section breakdown (in order)
1. **Session-resume prompt + GPT's status sync** (opening) — user pastes a long
   "continue where we left off" prompt describing a hybrid public/sealed custody
   litigation strategy already in progress; GPT replies with a recap of "prior" drafted
   declarations, motions (in-camera review, psych eval), FOIA/subpoena tools, witness
   strategy, and a filing roadmap — establishing the custom GPT's persistent
   "case memory" framing.
2. **Document inventory dump** — two large batches of Google Drive links (~20 `.docx`
   filenames total: motions, preservation letters, FOIA requests, declarations,
   affidavits) pasted by the user as the accumulated work product from a prior session.
3. **Filing roadmap** — GPT produces a 6-step sequence (public declarations → in-camera
   motion → confidential evidence → subpoena/FOIA → hearing prep → organization) with a
   summary table, explicitly citing the Public vs. Sealed/In-Camera split strategy.
4. **Consolidation request** — user explains the doc set is chaotic ("brain vomit"),
   asks GPT to merge duplicate iterations into one clean set per document type plus a
   case roadmap and evidence list.
5. **Master timeline construction (long arc)** — the conversation's core: user dictates
   raw personal history in long unstructured bursts; GPT restructures each into
   court-safe / therapy-safe bullet-point timeline sections, iterating through:
   custody/legal status clarification (no order exists — putative-father rights only),
   the "weaponization of the child," public defamation, financial/housing sabotage,
   the narcissistic-abuse-cycle framing (later explicitly told to avoid the word
   "narcissist" in court and use behavior-description instead), the ex's self-acknowledged
   possible BPD (with owner's own DSM-5 covert-NPD hypothesis raised then set aside per
   GPT's advice), owner's own mental-health history (bipolar, depression, suicidal
   ideation, mother's death by suicide/alcoholism, loss of relationship with 4 older
   children), "identity assassination" framing, alcohol as a specific trauma trigger vs.
   drug use as a separate acknowledged vulnerability, gaslighting/gatekeeping cycle tied
   to the ex starting new relationships, sexualized-photo evidence (daughter photographed
   asleep on/with the ex's new partners) framed as double-standard/hypocrisy evidence,
   and a legal-vulnerability-weaponization episode (ex allegedly using a child-support
   warrant — corrected mid-stream from "Jackson/adjacent County" to **Lapeer County** —
   plus police-call threats to control access).
6. **In-camera motion drafting/merging** — multiple iterative rewrites of a
   "Motion and Declaration for In Camera Review & Sealing," each revision folding in new
   raw disclosures (calculated/staged emotional outbursts allegedly discovered by
   reading the ex's phone, the "package deal" new-partner-integration concern, delayed
   discovery of the sleeping-photo evidence).
7. **Closing stretch** — user considers moving venting to a separate "vent chat" (citing
   this custom GPT's lack of cross-chat memory vs. standard ChatGPT), then asks the GPT
   to produce a **portable "stage-setting" master prompt** summarizing tone, case status,
   and gap-list questions — designed to seed a fresh chat without re-explaining
   everything. File ends here (mid-flow, not a natural conversation end).

## Facets present (extraction-lane map)
| facet | present? | examples / notes |
|---|---|---|
| identity (who) | yes | Owner (unnamed in this file; putative father via affidavit of parentage, no custody order); ex-partner "Katrina" (named repeatedly); daughter (unnamed except one likely-name slip "Kailah") |
| entities | yes | Katrina (ex/mother), Michael (ex's new partner), Adam (past parallel-relationship figure), Lapeer County (child-support warrant), MCL 722.23 (MI best-interest statute, cited by name), StartupTechLaw.com (GPT's ad sponsor) |
| relationships | yes | Co-parent conflict dynamic; daughter used as leverage; owner's unnamed BPD-diagnosed friend consulted for insight into Katrina |
| timeline/events | yes | Relationship start → alcoholism escalation "mid/end 2021" → BPD self-disclosure "late 2022/early 2023" → sleeping-photo incident "June 2023," discovered "~1 month before chat" → recent Lapeer County warrant episode "~1 month before chat" |
| life-history | yes | Owner: bipolar/depression/suicidal ideation, mother's suicide (alcoholism-linked), loss of relationship with 4 older children tied to owner's own past mental-health crisis |
| legal-strategy | yes | Public-vs-sealed hybrid filing strategy; explicit coaching to avoid the word "narcissist" in court; how to reference undisclosed sensitive evidence without triggering premature discovery/objection; framing self-acknowledged BPD as "her own admission," not a diagnosis by the owner |
| legal-artifacts | yes | ~20 named `.docx` filings referenced (not embedded) via Drive links: Motion_and_Subpoena_for_Surveillance_Footage, multiple Preservation_Letters, 3 FOIA_Request docs, Custody_Motion_Revised, Affidavit_Psych_Eval_Request, In_Camera_Request_Statement, Narrative_Summary_Supporting_Evidence, Hardship_Statement_LegalAid (+updated), Motion_for_In_Camera_Review_and_Sealing (2 versions), Public_Declaration_Request_for_In_Camera_Review, Hybrid_Declaration_Overview, Sealed_Declaration_Request, Public_Statement_for_Court, Confidential_In_Camera_Statement, Declaration_Patterned_Deception_Emotional_Manipulation, In_Camera_Talking_Points_Snapchat_Evidence, In_Camera_Declaration_Sexual_Trauma (+updated), Public_Declaration_Abuse_and_Reputation (+2 updated versions), LexisNexis_Family_Law_Cheat_Sheet, In_Camera_Custody_Legal_Summary; also full drafted Declaration text produced in-chat |
| mood/sentiment (owner-self, low-pri) | yes | Extremely raw/profane throughout; explicit venting acknowledged by both parties; owner explicitly asks GPT to be a "sounding board" at points |
| psychiatric | yes (extensive) | Owner: bipolar, depression, suicidal ideation, generational trauma from mother's suicide; discussion of ex's self-reported possible BPD and owner's own (set-aside) covert-NPD hypothesis; explicit court-strategy coaching around diagnostic labels |
| code/app-dev/plans | no | None — pure legal-strategy/narrative content |
| work | yes | Owner's electrician job and a pizza-delivery/restaurant job referenced as casualties of relationship conflict (job loss/re-hire tied to crisis episodes) |

## Notable content
- **Digital evidence provenance concern**: owner states he obtained "calculated
  outburst" evidence by "getting into her phone" — flagged as a fact the extraction
  pipeline should surface, since unauthorized phone access could itself carry legal risk
  the owner may want reviewed by counsel (not analyzed further here — no redaction, just
  a pointer for the evidence-routing map).
- **Sexualized/child-adjacent photo evidence**: repeated references to sexualized or
  suggestive photos involving the ex holding the sleeping daughter, and to the daughter
  photographed sleeping on/with a new partner (Michael) days into that relationship —
  these are described as intended in-camera-only exhibits, explicitly not for public
  filing, per the owner's own stated hybrid strategy.
- **Named legal citation**: MCL 722.23 (Michigan best-interest-of-the-child factors)
  cited directly by the GPT — useful anchor for cross-referencing against the platform's
  existing `mcl-factor-mapper` skill logic.
- **Correction-of-record moment**: mid-conversation the owner corrects the county name
  for a child-support warrant from a misheard "Jackson/adjacent County" to
  **Lapeer County**, and explicitly instructs "do not rewrite" the prior text — a good
  test case for how an extraction pipeline should handle user corrections that shouldn't
  trigger full re-summarization.
- **Meta-signal for the ingest design**: the owner explicitly complains about this
  custom GPT's context-window/no-cross-chat-memory limits near the end, and drafts a
  portable "stage-setting" prompt to carry context into a fresh chat — this is a live
  example of the exact problem the chat-ingest extraction lane is meant to solve.

## Sensitivity
- **Psychiatric**: LABEL — extensive, includes suicidal ideation history (self and, via
  suicide, the owner's mother), bipolar/depression, discussion of the ex's self-reported
  BPD and an NPD hypothesis the owner explicitly set aside for court purposes.
- **Sexual/child-adjacent**: LABEL — sexualized-photo allegations and an
  "In_Camera_Declaration_Sexual_Trauma" filing are referenced/discussed at a
  strategy level (not graphically depicted in this chat itself); flagged given the
  child is directly involved in the described photos.
- **Substance use**: LABEL — owner's own drug-use vulnerability and alcohol trauma
  trigger, and the ex's alcoholism, both discussed candidly for court-prep purposes.
- **PII**: LABEL — ex-partner named ("Katrina") throughout, a likely daughter name-slip
  ("Kailah"), a specific county (Lapeer) tied to a legal record (child-support warrant),
  new-partner first name ("Michael"). No redaction performed per research policy — real
  values stay in place for internal research use.
- **Profanity/raw language**: LABEL — pervasive throughout dictated passages; not
  sanitized, per owner's own explicit instruction to the GPT to preserve raw tone.
