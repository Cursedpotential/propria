# Jev Tier-1 eval: run log

> _Byline: Claude Code · Opus 5.5 · 2026-09-23. Handoff: `docs/handoffs/HANDOFF-2026-09-23-jev-tier1-eval.md`. Facts: `docs/handoffs/JEV-EVAL-PREP-2026-09-23.md`._
> Work dir with case data: **ovh-files `/data/probata/jev-eval/`** (owner 14:24: run on the server). Message text and raw responses stay there and never go into git. Code: `scripts/jev_eval/`.

| When (EDT) | Step | Result | Evidence |
|---|---|---|---|
| 14:15 | Opus auth | Long-lived `CLAUDE_CODE_OAUTH_TOKEN` via Claude Agent SDK; smoke `claude-opus-5-5` → `TOKEN_OK` | desktop `claude -p` run |
| 14:20–14:23 | Katrina pool | FB 67,377 · SMS 2021–22 (810-295-9303, hers) · 2024 her phone (conv 8102959302) · SMS 2025–26 (810-353-3592); 869 dropped | prep report addendum |
| 14:26 | Sample v1 | 300 rows; 5 Facebook system notices got in | `sample/sample_v1.jsonl` sha256 `f5e9e04b…0d35`; sql `db8b38dc…1a` |
| 14:45 | Sample v2 (current) | 300 rows, 291 unique texts; system notices + tapbacks excluded; 295/300 same as v1; every cell full; source_sha1 271/300 (29 missing in catalog provenance; SHA-256 not in catalog) | `sample/sample_v2.jsonl` sha256 `a07ddc18…6604`; sql `build_sample_v2.sql` sha256 `f936e1ae…2f19` |
| 14:29 | Phase 0 OpenRouter smoke (synthetic state) | HTTP 200, 752 ms; noul + choice shapes match handoff §1; served model `typesafe/jev-1.13-20260917`; cost $0.00002247 | `raw/phase0/smoke_20260923T182933Z.json` |
| 14:33 | Runtime | Reused the ovh-files **devbox** (Claude Code 2.1.263, Python 3.12, Node 22; owner 14:33) instead of a new image; work dir moved to devbox `persist/jev-eval` (host `/data/probata/volumes/devbox/home/jev-eval`); venv with claude-agent-sdk 0.2.158 (bundled CLI), pyarrow 25, duckdb 1.5.5; token passed at run time with `--env-file` from `/data/probata/secrets/jev-eval/claude.env` | `Propria/docs/reference/DEVBOX-ON-OVH-FILES.md` |
| 14:37–14:47 | **Phase 3 Opus labels** | 300/300 labelled + 30/30 repeat, 0 errors, 0 structural problems; `claude-opus-5-5`, prompt `a0dbb42296d7`, ~6–8 s/message at 4 in parallel. Claude Code also makes small Haiku housekeeping calls; every label is Opus. True counts: case_relevant 80, hostility 39, blame_shift 36, third_party 35, child_referenced 34, disparagement 29, financial 25, logistics 23, reframes 21, admission 13, wellbeing 10, legal 4, cooperation 4, **parenting_time_denial 0, info_gatekeeping 0**; 160 with no tag. Self-consistency on the repeat: 476/480 decisions identical (register 2, third_party 1, logistics 1) | `labels_opus.jsonl` sha256 `e0fe705c…74d9`; `labels_opus_repeat.jsonl` `56258c7e…c4`; `raw/opus/`, `raw/opus_repeat/` |
| 14:48 | Opus-label report v1 | For owner review before any Jev run | `reports/opus_labels_report_v1.html` sha256 `27d0de93…0249` |
| 15:15 | Owner review page | Owner 15:06: case_relevant "wrong entirely on every account"; asked for a page where he can type responses. Published a private claude.ai artifact **Opus Label Review** (`https://claude.ai/artifact/CWoTSa3hsUU4XrqQzmU3CD`, capability `db`). The owner flips labels, sets the register, comments, and writes tag definitions. Answers land in the artifact db (`reviews/m001..m300`, `notes/definitions`), read back with read_db | `scripts/jev_eval/review_page.py` |
| 17:17 | **PAUSED by owner** | "We got to talk about both the categories [and] the way the whole thing is structured. I don't think individual messages is really the way to go … it was really inaccurate … it doesn't [seem to be] capturing the issues; the categories don't necessarily fit very well. We're going to have to have a talk when I get home from work." No Jev runs and no new building until that talk. | owner message 17:17 EDT |
| 09-24 03:18 | **Restructure (owner)** | "Try A and A": new bout after 30 min of silence within a Detroit day; tone marks rapid shifts inside a bout. Pilot = 2024 texts from her phone. No child before Jan 2020. Healthy/loving stretches are flagged as carefully as conflict. | `docs/handoffs/JEV-EVAL-OPTIONS-2026-09-23.md` |
| 09-24 ~03:25 | Bouts v2 | Sender derived from recipients (the "Matthew" label had credited 3,696 of Katrina's messages to Matt); duplicate renderings collapsed; 645 bouts | `scripts/jev_eval/build_bouts_v2.sql`; `bouts/c2024_bouts_v2.jsonl` sha256 `1cb42f70…` |
| 09-24 03:22–03:52 | **Opus tone pass** | 645/645 bouts ok, 0 errors, 0 structural problems; `claude-opus-5-5`, `bout-tone-v1`. 283 bouts hold one tone; 1,594 shifts, 1,307 marked abrupt. Messages by tone: tense 8,211 · hostile 7,666 · neutral 2,372 · distressed 2,272 · friendly 1,549 · conciliatory 652 · affectionate 308 | `scripts/jev_eval/bouts_tone_opus.py`; `raw/bout_tone/` |
| 09-24 04:00 | Owner bout review page | Private artifact **Bout Review · 2024 Texts** (`https://claude.ai/artifact/2t8mQ3xVuV7whk45ZUqdDP`, capability `db`). Owner marks each bout Right / Wrong / Missed a shift + note; answers land in db `bouts/<bout_id>`; db reachable (read_db list returned empty, no error) | `scripts/jev_eval/bout_review_page.py` (97728eb); `reports/bout_review_v1.html` sha256 `de46938d…d879` |
| 09-24 04:39 | **Owner review, Start-here set (5 bouts)** | Saved in db: b0142 flare-up **right**; b0407 warm **right**; b0592 friendly-turns-sudden **right**; b0274 making-up **missed a shift** (the owner says it is overall right, but a subtle shift in her tone only shows against the rest of the day; led to the deferred context-versus-hindsight note in the options doc). b0007 hurt/pleading **right** (saved 04:40) | `scripts/jev_eval/pick_review_set.py` (03fa492); db `bouts/*` |
| 09-24 06:17–06:36 | **Discovery mode (owner 06:28–06:32)** | Same view, but categories are discovered, not locked: a guide list of behaviours plus free naming; reactions read against what came before (owner 06:31); a neutral prompt with no background on either person (owner 06:32). The owner's background was confirmed for Claude's guidance only and is not in the prompt. Facebook 2024 bouts built: 479 bouts from 7,169 messages. Smoke test on three owner-reviewed bouts: 3/3 ok, 0 structural problems, prompt `3d43778b60db`. Both people are read the same way; Jul 27 (b0142) flags Matt's monitoring, blow-up and blame-shift and Katrina's deflection and use of his reaction against him. New categories surfaced: minimizing_concern, minimizing_own_conduct, self_victimizing. 2024 source files: FB thread `katrinakinzel_1614217831996100` (Drive salem85 export 2025-02-21 + D:\Backup copy) and SMS vault `sms-20250218025955.xml`. **None from the owner's own phones for 2024.** | `scripts/jev_eval/bouts_discover_opus.py`, `build_bouts_fb2024.sql` (4f041fa); `bouts/f2024_bouts_v1.jsonl` sha256 `c4787e24…`; `raw/bout_discover/` |
| 09-24 06:40–06:50 | **Discovery v1 → v2, before/after test** | The v1 full run started 06:41 and was stopped at 06:45 after 122 Facebook bouts, plus the 3 smoke bouts, when the owner ruled "narrow". **These v1 results are flagged as the TEST** (owner 06:46), kept in `raw/bout_discover/` for comparison. v2 (`f53a4c5`, `9823c2c`) classifies only observable acts in the chunk: no looking backward or forward, no provocation or blame calls, no psychological reading (owner 06:45). One message can carry several categories (owner 06:48). v2 is re-running on the same 125 bouts into `raw/bout_discover_v2/`; the first 56 were made before the multi-category line and set aside in `raw/bout_discover_v2_partial_before_multitag/`. A compare page (`scripts/jev_eval/compare_discover_page.py`) shows v1 and v2 side by side for the owner's picks (db `compare/<bout_id>`). Catalog: bouts are permanent tables (Consignatio `4e54fd5`, 1,124 bouts, 30,198 message links); labels load through `export_bout_labels.py` + `chat_bout_labels_upsert_20260924.sql` (`b005035`, `f6b4121`), and bout-tone-v1 is loaded (645). | `logs/discover_v2_compare.log` |
| 09-24 03:40 | Precedent (Docstore) | This pilot is a throwaway test of the unit and the tone marks. The real bout cutter belongs inside the Case Bible CocoIndex super index (Codex correction 2026-09-18), with bouts as `working.extraction_window` rows (CLAIM-AND-ASSERTION-CANDIDATES-2026-08-29) and the Go chunker bridge (2026-09-06 ingest thinking pass) | Docstore records cited |

## Open checkpoints

**Agenda for the owner talk (evening 2026-09-23). Questions to settle, not proposals made:**
1. **Unit of analysis.** Is it a single message, a stretch of conversation (an exchange or episode), a day, or an incident? The owner says single messages are not the way to go.
2. **Categories.** What are the real issues the case needs surfaced? Build the list from the owner's own words, not the handoff's tags-v0.
3. **What Tier-1 is for.** Is it a cheap filter to find where the issues are (recall first), or a labeller whose output is used directly?
4. **What happens to what exists.** Sample v2, the Opus labels and the review page stay as evidence of what didn't work; the review page can still collect the owner's notes.

- [x] Owner approved the Phase 2 sample and said go for Phase 3 (14:32).
- [x] Phase 3 done; report v1 delivered 14:50.
- [ ] **Owner reviews the Opus-label report** before any Jev comparison (owner 14:17).
- [ ] Owner: 2 tags have zero positives in the random sample (parenting_time_denial, info_gatekeeping), so Jev cannot be scored on them. Proposed: an enriched add-on set for the rare tags, labelled by Opus and scored separately.
- [ ] Owner: 9 messages are `case_relevant` only because of their context ("Nope" answering a child-care question). Should the gate count context, or target words only?

## Known deviations from the handoff
- Opus runs through the Agent SDK, which has no temperature setting. Proposed substitute: re-label a random 10% to measure self-consistency.
- `source_sha256` is not in the catalog; `source_sha1` is carried instead (271/300) and the gap is reported, never filled in.
- TypeSafe direct is unavailable (sign-ups paused): cells A/B only.

## 2026-09-24 20:46–21:15 EDT — owner review of the disagreements; whole-block episodes across models

> _Byline: Claude Code · Opus 5.5 · 2026-09-24._

- **Owner review** of 29 Opus-vs-Jev disagreements (all in the "Opus neutral → Jev tense" group): Opus right 12, Jev right 7, notes only 10. The notes, in the owner's words, boiled down to four points:
  - **Chunking destroys context.** Sample notes: "there was more to this conversation" (×5), "why was one message evaluated", "by stripping out one message it destroyed the context".
  - **"Neutral" hides the good.** "hasn't marked one good thing"; "I almost want to take neutral away"; flag working together and planning to move in, for the later contrast.
  - **Upset about a third party is not hostility toward her.** b0149: anger at her brother.
  - **Her silences and blocks must be flagged.** b0291.

  Jev over-reads tension and Opus under-reads it.
- **New approach** (`scripts/jev_eval/block_review_llm.py`, `332e806`): a model reads a 100-message block whole and makes its own conversations by topic and flow. Labels are multi-label, each with who / at whom / intensity 1–3 / evidence indexes. The catch-all neutral is gone: "logistics" is used only when a stretch has no emotion at all. Unanswered runs are recorded.
  - **Prompt v2** (owner 20:52): a factual background plus a list of behaviours to watch for, applied to both people. It adds the labels leverage, deflecting, threat and checking, a `responds_to_i` field, and 6 worked examples written from the owner's notes (b0014, b0019, b0108, b0149, b0187, b0291). The examples hold case text, so they live in devbox `persist/jev-eval/prompts/block_examples_v2.json`, never git.
- **Models** (keys in root-only `/data/probata/secrets/jev-eval/*.env`; the script rotates across keys on 503/429):
  - Gemini **3.8 Flash**: ok. The first try got 503 "high demand"; the retry passed.
  - Gemini 3 Flash Preview: ok.
  - Gemini 3.7 Flash: 503.
  - Gemini **3.1 Pro**: no quota on any key (RESOURCE_EXHAUSTED). 2.5 Pro is gone (NOT_FOUND). **Pro needs a key on a project with billing enabled.**
  - `GEMINI_API_KEY_4` is denied outright (403).
  - **Gemma 4 31B IT**: ok, but slow (412 s).
  - **Nemotron 3.5 Lightning** (NIM): it rejects `nvext.guided_json` (switched to `response_format` json_schema) and returned 1 conversation that did not cover the block. Not usable for this task as prompted.
- **June 27 block** (100 messages, 06-27 09:27 → 06-29 13:33, prompt v1):
  - Gemini 3.8 Flash: 7 conversations, clean.
  - Gemma: 12 conversations, clean, and 3 unanswered runs.
  - Both mark cooperation, warmth and playfulness, and put the "Scott hitting child" incident at a third party.
  - Page: **Block Review** https://claude.ai/artifact/CnP8fihnfwg2EuvvxLoNg3 (db `block_reviews`).
- **July 1 block** (07-01 07:14 → 07-02 12:04), v1 vs v2 on the same three models: running. It holds none of the example bouts, so the comparison is fair.
