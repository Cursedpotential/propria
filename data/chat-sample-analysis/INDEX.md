# Chat Sample Analysis — INDEX

> _Byline: Claude Code · Opus 4.8 · 2026-07-11_
> Per-file breakdowns of the owner's AI-chat sample set
> (`C:\Users\matts\OneDrive\Desktop\chat sample`). Discovery material feeding the
> chat-ingest + multi-facet **extraction lane** design (see auto-memory
> `resume-20260711-agno-memory-research.md` → EXTRACTION LANE; DECISION_LOG D-034).
> **Nothing decided here** — this maps what's actually IN the chats so the facet
> design is grounded in real data, not on paper (owner rule: sample before deciding).

## Purpose
The AI chats are the owner's **story** (working his custody case with AI), NOT
evidence — we mine them to build a map that points at the real evidence to pull.
Each breakdown reads the file **in full** and reports structure, section-by-section
flow, and which extraction facets appear.

## Per-file breakdown template (every agent follows this)
```
# <filename>
> _Byline: Claude Code · Sonnet · 2026-07-11_
## Snapshot
- Source tool / export style · size · format (md/json/docx/folder) · era/date · # turns or conversations
## Structure & format
- How turns/roles are delimited · metadata/front-matter · embedded artifacts (code, tables, drafted docs)
- PARSING NOTES: what a parser must handle for THIS format specifically
## Section-by-section breakdown (in order)
- Ordered walk of the major segments/topics — what each covers, how the conversation flows/progresses
## Facets present (extraction-lane map)
| facet | present? | examples / notes |
Facets: identity(who) · entities · relationships · timeline/events · life-history ·
legal-strategy · legal-artifacts · mood/sentiment(owner-self, low-pri) · psychiatric ·
code/app-dev/plans · work
## Notable content
- Key evidence pointers · drafted legal artifacts · named entities · dates · decisions
## Sensitivity
- Flag personal/abuse/psychiatric/PII (LABEL, don't redact — research policy)
```

## File → breakdown map
| # | Source file | Breakdown doc | Agent |
|---|---|---|---|
| 1 | conversations.json (native export, 1.4M) | `chatgpt-native-export.md` | A1 (DuckDB) |
| 2 | ChatGPT-Legal AI - Custody III.md (792K) | `chatgpt-legal-ai-custody-iii.md` | A2 |
| 3 | ChatGPT-Document consolidation plan.md (520K) | `chatgpt-document-consolidation-plan.md` | A3 |
| 4 | Scholar GPT (1).md (348K) + ChatGPT-Court case prompt setup.md (200K) | `scholar-gpt.md`, `chatgpt-court-case-prompt-setup.md` | A4 |
| 5 | AI Lawyer (4).md (220K) + ChatGPT-Condense project description.md (36K) + AI_Chat_File_Processing_Applet.md (20K) | `ai-lawyer.md`, `chatgpt-condense-project-description.md`, `ai-chat-file-processing-applet.md` | A5 |
| 6 | Gemini ×4 (Call Log Analyzer, Timeline Analysis, Tracking Relationship Sentiment, Emotional Manipulation) | `gemini-*.md` | A6 |
| 7 | Research_Plan_Ready.md (92K) + Navigating_Toxic_Relationship_Dynamics.md (88K) + Abusive_Relationship.md (16K) | `research-plan.md`, `navigating-toxic-relationship.md`, `abusive-relationship.md` | A7 |
| 8 | Reverse_Geocode_App.md (76K) + Explaining_Formal_Closing_Statement.md (13K) + ALEX_DENNIS_SUMMARY.md (20K) | `reverse-geocode-app.md`, `explaining-formal-closing.md`, `alex-dennis-summary.md` | A8 |
| 9 | Comprehensive Plan…/ (folder, 172K) + In_Camera_Talking_Points_Snapchat.docx (108K) | `comprehensive-plan-folder.md`, `in-camera-talking-points-snapchat.md` | A9 |
| — | conversations.zip (568K) | SKIP — duplicate of conversations.json | — |

## STATUS (2026-07-11)
**20 / 20 breakdowns written. None pending.** `chatgpt-native-export.md` (conversations.json,
DuckDB structural pass) and `in-camera-talking-points-snapchat.md` (the .docx work-product,
zipfile+regex extraction) were completed 2026-07-11 after the session-limit interruption noted
below closed out the last two gaps. Cross-file synthesis below is now final over the full set.

## Cross-file synthesis (from A2–A9 reports; owner reviews)

### Format / delimiter inventory (the parser must handle ALL of these)
1. **Native ChatGPT export** (`conversations.json`): `mapping`→node→`message.author.role` + `message.content.parts`, parent/child tree. *(schema pending A1 verification.)*
2. **ChatGPT single-conv md**: `## Prompt:` / `## Response:` with **nested Deep-Research + code-interpreter sub-turns** (python fences, `Analyzed`/`Result` lines) — route code-interpreter separately from prose.
3. **Custom-GPT md** (AI Lawyer/Scholar): `You asked:` / `ChatGPT Replied:` + per-turn **disclaimer/ad boilerplate** to strip.
4. **Gemini md**: `**You:**` / `**Gemini:**` + `---`; **Deep-Research & Audio-Overview turns are STUBS** (report/audio body NOT in export).
5. **Dated-export md** (`Title-YYYY-MM-DD-HH-MM-SS.md`): `---` used **both** between turns **and** between unrelated concatenated topics → NOT a reliable turn delimiter; one file often holds several unrelated conversations.

### Corpus-level findings
- **Duplication is real**: `navigating-toxic` ≈90% overlaps `reverse-geocode`; several files concatenate 4+ unrelated topics → **dedup + topic-splitting needed at ingest**, not just per-file parsing.
- **Story vs work-product**: `alex-dennis-summary`, `comprehensive-plan-folder`, `in-camera-*.docx` are **work-products/legal artifacts**, not transcripts — different handling.
- **Prior art for THIS platform**: multiple chats show the owner already prototyping chat-vectorization/compression applets + a pandas SMS/CSV→chronological-tagged-CSV pipeline → direct precedent for the ingest/extraction design.
- **Real entities present**: owner **Matt**, partner **Katrina**; case **Salem v. Kinzel** (Genesee County); child ~5yo.
- **Sensitivity + legal flags** (labeled, NOT redacted — research policy): psychiatric self-disclosure (bipolar/depression/anxiety, family alcoholism); sexual-trauma content; threatened-violence exchange; **"hacked" evidence** (admissibility implications — surface to owner); child-endangerment allegations.
- ⚠️ `chatgpt-document-consolidation-plan.md` was written by a subagent that tripped a security-policy warning (likely from summarizing hack/threat content, not misbehavior) — spot-check before relying on it.

## Facet coverage heat-map (all 20 breakdowns)

Derived from each breakdown's "Facets present" table. **strong** = the breakdown rated the
facet heavy/dominant/very-high/core-subject/primary; **present** = yes/moderate/light/weak/
partial/minor/implied/adjacent; **absent** = none/no/unknown. Counts sum to 20 per row.

| facet | strong | present | absent | best exemplar files |
|---|---|---|---|---|
| identity (who) | 3 | 15 | 2 | `chatgpt-court-case-prompt-setup.md` (full case caption verbatim), `alex-dennis-summary.md`, `abusive-relationship.md` (only file with real first names Matt/Katrina) |
| entities | 2 | 14 | 4 | `chatgpt-court-case-prompt-setup.md` (address, phone numbers, named docs), `alex-dennis-summary.md` (courts, agencies, socials) |
| relationships | 8 | 7 | 5 | `alex-dennis-summary.md`, `chatgpt-court-case-prompt-setup.md`, `gemini-tracking-relationship-sentiment.md` (core subject of the whole file) |
| timeline/events | 6 | 8 | 6 | `chatgpt-document-consolidation-plan.md` (rebuilt end-to-end 5×), `scholar-gpt.md`, `alex-dennis-summary.md` |
| life-history | 1 | 10 | 9 | `gemini-tracking-relationship-sentiment.md` (explicit 7–8yr frame — only "strong" rating), `chatgpt-legal-ai-custody-iii.md`, `scholar-gpt.md` |
| **legal-strategy** | 6 | 11 | 3 | `chatgpt-legal-ai-custody-iii.md`, `chatgpt-court-case-prompt-setup.md`, `scholar-gpt.md` |
| **legal-artifacts** | 6 | 7 | 7 | `chatgpt-legal-ai-custody-iii.md` (~25+ drafted items), `ai-lawyer.md` (~20 named .docx filings), `scholar-gpt.md` (5 real uploaded court filings) |
| mood/sentiment (owner-self, low-pri) | 5 | 12 | 3 | `chatgpt-legal-ai-custody-iii.md`, `gemini-tracking-relationship-sentiment.md` (primary subject of the file), `abusive-relationship.md` |
| psychiatric | 6 | 6 | 8 | `scholar-gpt.md`, `chatgpt-legal-ai-custody-iii.md`, `abusive-relationship.md` |
| code/app-dev/plans | 12 | 2 | 6 | `chatgpt-native-export.md` (#19 = direct prior art for this platform), `ai-chat-file-processing-applet.md` (direct precedent), `gemini-call-log-analyzer.md` (Neo4j schema) |
| work | 1 | 8 | 11 | `navigating-toxic-relationship.md` (largest single content block in the file), `ai-lawyer.md`, `chatgpt-court-case-prompt-setup.md` |

**Reading the map**: `legal-strategy`/`legal-artifacts` and `code/app-dev/plans` are the two most
consistently populated facets (17/20 and 14/20 files respectively at present-or-better) — the
corpus is dominated by drafted legal work-product and app-dev prototyping, exactly as the corpus-
level findings above describe. `work` and `life-history` are the sparsest (most files are
custody/relationship-scoped, not general biography or employment history). `psychiatric` and
`mood/sentiment` cluster heavily in the ChatGPT-legal-strategy files and the two Gemini
relationship-modeling files — the ones with the most raw, unfiltered owner voice.

## Deliverable complete

All 20 per-file breakdowns + this INDEX are done. **Next step is an owner discussion**, not more
extraction: (1) the stage-1 tag scheme for the extraction lane (this heat-map + the facet notes
above are the grounding data), and (2) wiring `cb_chat_dedup` into the case-bible organizer — the
tool is already built at `~/.claude/local-plugins/case-bible/tools/cb_chat_dedup.py`
(report-only, gate-passed) and is a direct fit for the corpus-level duplication finding above
(`navigating-toxic` ≈90% overlaps `reverse-geocode`, etc.).
