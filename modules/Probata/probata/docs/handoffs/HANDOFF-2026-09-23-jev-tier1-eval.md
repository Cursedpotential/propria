---
title: "HANDOFF: Jev Tier-1 classifier evaluation (raw API, real messages)"
date: 2026-09-23
source: owner upload to Claude Code session b9db6641 (09:27 EDT), saved verbatim below the rule
status: active, plan-first, owner checkpoints required
tags: [handoff, jev, typesafe, classifier, tier-1, evaluation, case-bible]
---

> _Byline: saved by Claude Code · Opus 5.5 · 2026-09-23. Text below is the owner-supplied handoff, unedited._

> **Owner decisions, 2026-09-23 14:15–14:17 EDT (added by Claude Code · Opus 5.5):**
> - **Opus runs on the owner's long-lived Claude Code token, not an API key.** `CLAUDE_CODE_OAUTH_TOKEN` in `~/.secrets/anthropic.env` via the Claude Agent SDK (`claude-agent-sdk` 0.2.158, installed on the desktop Python 2026-09-23). Smoke test 14:16: `claude-opus-5-5` answered, subscription rate limit `allowed`. Labels use `output_format={type: json_schema}` and read `ResultMessage.structured_output`. The SDK exposes **no temperature setting**, so the handoff's "temperature 0" cannot be applied. Proposed substitute: log model + session id per label, and re-label a random 10% to measure Opus self-consistency.
> - **New checkpoint after Phase 3:** before any Jev comparison (Phase 4+), the owner gets a report of Opus's classifications (every message, every tag, rationales) to review. Jev runs only after the owner accepts the Opus labels.

> **Owner decision, 2026-09-23 09:41 EDT (added by Claude Code · Opus 5.5):** TypeSafe has paused new sign-ups, so there is no `TYPESAFE_API_KEY`. The eval runs on **OpenRouter only**: Phase 4 keeps cells **A** (target only) and **B** (target + ≤5 prior); cells C and D and the provider-parity scoring are dropped. Fact check by the prep agent (docs/handoffs/JEV-EVAL-PREP-2026-09-23.md): `OPENROUTER_API_KEY` is present; `ANTHROPIC_API_KEY` is missing (Opus reference labels need a key, or must run inside Claude Code itself; this is for the owner to decide).

> **Owner rule, 2026-09-23 09:30 EDT (added by Claude Code · Opus 5.5):** "Opus needs to do the classification and any reasoning work and validating the results." Reference labels, planning, scoring interpretation, disagreement analysis, the wording A/B and the final verdict are all done by Opus (`claude-opus-5-5`). Cheaper models (Sonnet subagents) may only gather facts: key presence, docs checks and data inventory.

# HANDOFF: Jev Tier-1 Classifier Evaluation (Raw API, Real Messages)

**Owner:** Matt (decision-maker). **Executor:** Claude Code.
**Scope:** Prove whether TypeSafe Jev works as the cheap Tier-1 decision classifier for the Case Bible / forensic evidence pipeline (Salem v. Kinzel). Raw API calls only. **No n8n, no Agno integration, no production wiring** until Matt reviews the scorecard.

---

## 0. Operating rules (non-negotiable)

1. **Plan mode first.** Present the plan for each phase and wait for Matt's go before executing it. Stop at every CHECKPOINT.
2. **Evidence is read-only.** Never modify, move, rename, or delete source exports. All outputs go to a sandbox work dir (proposed: `./jev-eval/`). Confirm the path with Matt before creating it.
3. **No silent decisions.** If something is ambiguous, ask one question and wait.
4. **Glass box.** Log every API request and response (raw JSON) with timestamp, provider, model, and question-set version. Nothing gets summarized away.
5. **Never fabricate.** If a parser can't read a message, record it as unparsed. Don't guess.
6. **Deterministic stays deterministic.** Sender, direction, timestamps, platform, thread, dates, amounts, and counts are computed in code, never by Jev.
7. **Secrets** come from environment variables only. Never print or log keys.

---

## 1. Jev API facts (verified Sept 23, 2026)

| | TypeSafe direct | OpenRouter |
|---|---|---|
| Endpoint | `POST https://api.typesafe.ai/v1/systemone` | `POST https://openrouter.ai/api/v1/systemone` |
| Auth | `Authorization: Bearer $TYPESAFE_API_KEY` | `Authorization: Bearer $OPENROUTER_API_KEY` |
| Pinned model ID | `jev-1.13.0` | `typesafe/jev-1.13` |
| Context | 64k/request (state + longest question ≤32k) | 32K listed. **Size all state to fit 32K so both paths match.** |
| Price | $0.042/M input, output free | same |
| Rate limit | 1,200 req/min, 250k tok/s | per OpenRouter account |

- **Not chat completions.** OpenRouter runs Jev on its Decisions API. OpenAI/chat SDKs do not work with it.
- **Python SDK:** `pip install typesafe-sdk`. It reads `TYPESAFE_BASE_URL` and `TYPESAFE_API_KEY`. For OpenRouter, set `TYPESAFE_BASE_URL=https://openrouter.ai/api` and put the OpenRouter key in `TYPESAFE_API_KEY`. Raw `httpx`/`requests` works too; prefer raw HTTP for this eval so the full response JSON is captured.
- **Smoke test CLI:** `pip install jev-cli`, then `jev auth test --provider openrouter`. The official provider is the default.

**Request shape**
```json
{
  "model": "jev-1.13.0",
  "state": "<string or JSON object>",
  "questions": {
    "q_id": {
      "type": "noul",
      "instructions": "Self-contained question text",
      "criteria": { "true": "what counts as yes", "false": "what counts as no" }
    },
    "register": {
      "type": "choice",
      "instructions": "...",
      "criteria": { "factual": "...", "opinion": "...", "emotional": "...", "mixed": "..." }
    }
  }
}
```

**Response**
- Noul: `{ "type": "noul", "noul": 0.98 }`, where the number is the probability of yes.
- Choice: returns the pick, a probability for every option, and a `confidence` value.
- Usage block: input tokens (billed) and output tokens (not billed).

**Hard constraints the API enforces**
- `state: null` → 422 error. Send `""` or an object instead.
- Choice questions take 1–255 labels; score questions take 2–10 levels.
- **The question ID is never sent to the model.** Instructions must stand alone.
- All questions in one call see the same state and are answered in parallel. Extra questions are nearly free, so send the full tag set in one call per message.

**Known Jev weak spots (design around them)**
- Dates, numbers, and counting.
- Adversarial or manipulative phrasing.
- Non-English text.
- Overlapping labels.
- Irrelevant context lowers accuracy.
- It can be wrong at confidence 1.00. A random audit sample is mandatory.

---

## 2. Tag set v0 (question-set version `tags-v0`)

The state always contains a `target_message`. Every instruction refers to "the target message."

### Gate
| id | instructions | true | false |
|---|---|---|---|
| `case_relevant` | Does the target message relate to the child, parenting time, custody, co-parenting, money for the child, or court proceedings? | Touches any of those subjects | Unrelated small talk, spam, or other topics |

### Content tags (Noul, multi-label)
| id | instructions | true | false |
|---|---|---|---|
| `parenting_time_denial` | Does the target message refuse, cancel, shorten, delay, or put conditions on a parent's time or contact with the child? | Parenting time or contact is blocked, reduced, delayed, or conditioned | Time or contact proceeds, or isn't discussed |
| `child_referenced` | Does the target message mention or discuss the child (Kailah)? | The child is named, nicknamed, or clearly the subject | No reference to the child |
| `info_gatekeeping` | Does the target message withhold or refuse information about the child's school, health, location, or activities? | Information is refused, deflected, or conditioned | Information is shared, or no such request exists |
| `hostility_threat` | Does the target message contain insults, threats, intimidation, or demeaning language? | Any insult, threat, intimidation, or contempt | Neutral or civil, even if disagreeing |
| `parent_disparagement` | Does the target message criticize or run down a parent's character or fitness as a parent? | Negative claims about a parent's character or parenting | No character or fitness attack |
| `financial` | Is the target message about child support, payments, expenses, or money? | Money is a subject | Money isn't mentioned |
| `logistics` | Is the target message about scheduling, pickup, drop-off, or exchange arrangements? | Times, places, or arrangements are discussed | No scheduling content |
| `legal_reference` | Does the target message mention court, attorneys, orders, filings, police, or the Friend of the Court? | Any legal-system reference | None |
| `third_party` | Does the target message involve a third party in the child's life, such as a partner, relative, or Mike Joubran? | A non-parent person is involved or discussed | Only the two parents and the child |
| `child_wellbeing` | Does the target message address the child's health, safety, school, or emotional state? | Wellbeing is a subject | Not addressed |
| `cooperation_offer` | Does the target message offer flexibility, a compromise, or extra time? | A cooperative offer is made | No offer |
| `admission` | Does the sender of the target message admit, concede, or apologize for something? | Admission, concession, or apology | None |

### Experimental flags (evaluated separately; used for escalation only, never as final tags)
| id | instructions | true | false |
|---|---|---|---|
| `reframes_prior_event` | Does the target message deny or recast something that happened earlier in the conversation? | Contradicts or reinterprets a prior event | No such reframing |
| `blame_shift` | Does the target message shift responsibility for a problem onto the other person? | Responsibility is redirected | No redirection |

### Choice
| id | instructions | options |
|---|---|---|
| `register` | What kind of statement is the target message primarily? | `factual`: verifiable facts or logistics; `opinion`: judgments or beliefs; `emotional`: feelings or venting; `mixed`: substantial blend |

---

## 3. Phases

### Phase 0: Environment & smoke test
- Confirm `OPENROUTER_API_KEY`, `TYPESAFE_API_KEY`, and `ANTHROPIC_API_KEY` exist in the environment. Report present or missing only; never print values.
- Make one identical call to each provider using a synthetic state (not case data) with 2 noul questions and 1 choice question. Save the raw responses.
- Verify the response shapes match Section 1, both providers return per-option probabilities and confidence, and usage is reported.
- **CHECKPOINT:** report shapes, latency, and any mismatch between providers.

### Phase 1: Locate & inventory exports (read-only)
- Ask Matt for the export locations first. Search the filesystem only if he says to.
- Inventory by platform (SMS XML, Facebook HTML, iMessage PDF, WhatsApp, other) with file counts, message counts, date ranges, and parse status.
- Reuse existing parsers or DuckDB/Postgres tables if present. Ask before writing new parsers.
- **CHECKPOINT:** inventory table.

### Phase 2: Build the 300-message sample
- Stratify by platform, year, sender, and thread activity. Include both quiet and high-conflict stretches.
- Normalized schema: `msg_id, source_file, source_sha256, platform, thread_id, ts_utc, sender_label, direction (outgoing/incoming, from Matt's side), text, prior_context[≤5 msgs]`.
- Store at `./jev-eval/sample.parquet` and in DuckDB. Record the sampling seed.
- **CHECKPOINT:** spread table, plus 10 random rows for eyeball QA.

### Phase 3: Opus reference labels
- Model: `claude-opus-5-5`, temperature 0, tool use with a strict schema: one boolean per tag, including the experimental flags, plus a `register` enum and a one-line rationale per true tag.
- Use the exact tag wording and criteria from Section 2, and the same state (target plus context) Jev gets.
- Output: `./jev-eval/labels_opus.parquet`.
- These are **reference labels**, not ground truth. Matt audits the disagreements in Phase 6.

### Phase 4: Jev run matrix
Run all 300 messages through each of the 4 cells, one call per message with all 16 questions:

| cell | provider | context |
|---|---|---|
| A | OpenRouter | target only |
| B | OpenRouter | target + ≤5 prior (marked) |
| C | TypeSafe direct | target only |
| D | TypeSafe direct | target + ≤5 prior (marked) |

- State format: `{"target_message": {"sender": ..., "ts": ..., "text": ...}, "prior_messages": [...]}`. In target-only cells, `prior_messages` is omitted.
- Concurrency: ≤10 parallel requests, with retry and backoff on 429/5xx.
- Log raw responses to `./jev-eval/raw/{cell}/{msg_id}.json`.
- Estimated cost: about 1,200 calls × ~2k tokens ≈ **$0.10**. Opus labeling costs a few dollars.

### Phase 5: Scoring (per tag, per cell)
- Agreement with Opus: accuracy, precision, recall, and F1.
- Calibration: ECE, a reliability table (10 bins), and a confidence histogram.
- **Threshold sweep:** for each tag, find the lowest auto-accept threshold that hits ≥95% agreement on auto-accepted items (make the target configurable), and report the coverage at that threshold. Do the same for the auto-reject side.
- **High-confidence misses:** list every disagreement where Jev is at ≥0.90 or ≤0.10.
- Provider parity: A vs C and B vs D, as the share of identical decisions and the mean absolute probability difference.
- Context effect: A vs B and C vs D, per tag.
- Latency p50/p95 per provider, and actual cost from usage.
- Gate analysis: how many content tags fire when `case_relevant` < 0.5. This shows what the gate would drop.

### Phase 6: Wording A/B
- Take the 3 weakest tags by F1. Write 2 alternate wordings of each, with changes to both the instructions and the criteria.
- Re-run on the best cell only, then score.
- **CHECKPOINT:** show Matt the variants before running them.

### Phase 7: Deliverables
1. `./jev-eval/scorecard.html`: a single self-contained HTML report with a summary verdict (does Jev work as Tier 1, and for which tags), per-tag tables, the threshold recommendations, the provider pick, and the context-mode pick.
2. `./jev-eval/disagreements.csv`: every Jev/Opus disagreement with the message text, both labels, Jev's probability, and the Opus rationale, for Matt's audit.
3. `./jev-eval/audit_sample.csv`: a random 2% of Jev high-confidence decisions that agree with Opus, to check for shared blind spots.
4. `./jev-eval/decisions_ledger.parquet` in the proposed production ledger schema (below).
5. `./jev-eval/RUNLOG.md`: every step, every checkpoint decision, and every anomaly.

---

## 4. Proposed production ledger schema (prototype only)
`decision_id, msg_id, question_set_version, question_id, question_text_sha256, provider, model_id, context_mode, noul_prob | choice_label, choice_probs(json), confidence, tier_resolved (jev|escalated|human), latency_ms, input_tokens, request_ts, raw_response_path`

---

## 5. Out of scope / explicit rejections
- n8n workflows, Agno tools, and Postgres production tables: **not yet**.
- `jev-latest` or any unpinned alias.
- Chat-completions endpoints for Jev.
- Asking Jev about dates, amounts, counts, or sender identity.
- Writing anything into the evidence vault or source export folders.

## 6. Open items Code must resolve with Matt before Phase 1
1. Export locations (paths and drives).
2. Work-dir location for `./jev-eval/`.
3. Spend caps on the OpenRouter and TypeSafe keys (recommend $5 each for this eval).
4. Whether existing parsed DuckDB/Postgres message tables should be the sample source instead of the raw exports.
