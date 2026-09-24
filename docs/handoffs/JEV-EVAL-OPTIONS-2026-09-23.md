---
title: Jev eval: other ways to structure it (options for the owner talk)
date: 2026-09-23
status: options for discussion, nothing decided, nothing built
tags: [jev, classifier, evaluation, incidents, extraction, options, owner-decision-pending]
---

# Jev eval: other ways to structure it

> _Byline: Claude Code · Opus 5.5 · 2026-09-23 17:40 EDT. Owner 17:17: single messages are not the way; results were inaccurate; the categories don't fit. Owner 17:18: "use some thinking processes and explore other options". Models used, in order: 5 Whys (why it missed) → Jobs to be Done (what the cheap tier is for) → First Principles (rebuild the unit and the categories) → Pre-mortem (stress-test the lead option)._

## The short version

- **Why it missed:** the eval was shaped around Jev's input format (one state, yes/no questions), not around what you need found.
- **What you need found:** your issues are **patterns across time**. A promise, then a reversal. An event, then its denial. Contact, then a condition put on it. A single "Nope" or "The gig is up" can't hold those.
- **The design already exists:** your own drafts (the Corpus Extraction Recipe and the Synthesis Spec) describe it:
  1. a cheap pass pulls out every *mention*, neutrally;
  2. a middle pass groups mentions into *incidents*;
  3. an expensive pass interprets them.
- **Jev can't extract anything.** It only answers questions with probabilities. So in that design Jev can only be the **gate**: a cheap first look that decides which conversations get read closely.
- **Lead option (A):** judge **conversation sessions** instead of single messages. Jev gates the sessions, Opus pulls out incidents from the ones that pass, and you name the categories after you see what the incidents actually are.

## What we have (measured)

| Source | Messages | Sessions (new after 6 h of silence) | Typical session | Largest |
|---|---:|---:|---:|---:|
| Facebook 2018–25 | 67,377 | 1,435 | 19 messages | 1,108 |
| Texts 2021–22 | 41,017 | 545 | 39 | 1,365 |
| Texts 2024, her phone | 23,032 (includes duplicate renderings) | 112 | 41 | 3,088 |
| Texts 2025–26 | 4,259 | 83 | 13 | 682 |

Long sessions are cut into windows of about 40 messages with 5 overlapping, as your recipe says. That comes to roughly 3,000–4,000 windows in all.

## Why the first attempt missed (5 Whys)

1. The labels were wrong or empty: 160 of 300 messages had no tag, and half the messages are under 34 characters.
2. **Why?** A single message rarely carries the meaning; the exchange does. Opus leaned on context for 9 "case-relevant" calls because the rules forced a per-message answer.
3. **Why per message?** Jev takes one state and a list of questions, and the handoff built the eval around that input.
4. **Why don't the categories fit?** They are generic family-law tags written by whoever drafted the handoff, not the issues **you** have been describing for months.
5. **Root cause:** the eval was designed from the tool outward. The job should come first.

*Steelman for single messages:* they are the exact, citable anchor, the thing you'd quote in a filing. Every option below keeps message IDs as the anchor. Only the unit of **judgment** changes.

## What the cheap tier is for (Jobs to be Done)

The job is **"point me to the stretches of conversation that matter for the case, and say what kind of thing happened there, so I and Opus read the real thing."** That is triage, not labelling. It maps to your six steps: step 2 (is it relevant?) and the gap-finding in step 6.

## The options

| | Unit judged | Categories come from | Jev's role | Opus's role | Cost and time (rough) |
|---|---|---|---|---|---|
| **A: sessions → incidents (lead)** | Conversation session or window | **You name them after seeing the incident clusters** | Gate: "does anything here matter?" (recall first) | Pulls neutral incident records from gated windows, groups them into incidents, interprets | Jev ≈ $1 for every window; Opus on maybe a third of windows, a few days of subscription time |
| **B: your taxonomy first** | Session | **Your own words first**: the review page's definition box, plus what you've already written in your AI chats about the case | Gate + coarse issue flags | Checks the gated sessions against your list | Needs an hour of your time up front; then as A |
| **C: Opus reads everything** | Window | Emerge at grouping | None | Pulls incidents from every window | Most complete; roughly 3–4k Opus calls, about a week of subscription limits |
| **D: contradiction first (the "delta")** | Claim | Built in: what she said vs. what was true vs. when you found out | Gate | Pulls claims, then lines them up against later facts and records | Closest to the platform's purpose; needs A's incident records underneath it first |
| **E: keep single messages, fix the labels** | Message, always read with its session | You rewrite the tag list | Per-message labeller | Reference labels | Smallest change, but it's the structure you just rejected |

**Default recommendation: A, with B feeding the names and D as the later lens.** A turns your recipe into something that runs on messages instead of AI chats. The categories stop being guessed up front: you see the incidents first, then name the kinds. D (the gaslighting delta) needs A's incident records to exist first. Jev's test becomes simple: **does its gate catch the sessions Opus and you say matter, and how much does it save?**

## What could go wrong with A (pre-mortem)

- **An incident spans sessions** (a fight at night that picks up in the morning). The grouping pass works across sessions, and the windows overlap, so it gets rejoined.
- **The gate misses subtle sessions** (sarcasm, coded talk). Set the threshold for recall, and have Opus audit a random 2–5% of the sessions the gate turned away.
- **Opus extraction costs too much.** Pilot on one period first. The 2024 texts from her phone are where hostility clustered: 112 sessions.
- **Review burden.** An incident list is far shorter than 300 messages, and each incident shows the actual messages.
- **Characterization creep.** Your recipe's neutrality rules stay: records describe conduct and never label it "manipulative"; interpretation comes later.
- **Jev turns out not to be needed.** That's a valid result. The eval exists to answer that question.

## Questions for tonight

1. Is **"find the stretches that matter, then pull incidents out of them"** the job, or is it something else?
2. Unit: **session or window** (lead), or a **day**, or a whole **incident**?
3. Categories: **named after seeing clusters** (A), **your list first** (B), or both?
4. Opus reading windows is extraction. Does your rule "Opus does the classification and reasoning" cover it, or may a cheaper model do the first neutral pass, the way the recipe splits it?
5. Pilot period: the **2024 texts from her phone** (112 sessions), or another stretch you know well enough to judge?

## Owner direction, 2026-09-23 21:06–21:08 EDT (voice, cleaned)

- Judge a larger snippet, **not the whole conversation**. Use the index's chunks, which should be **conversation-shaped** once messages are ingested and indexed, as the narrowing constraint.
- Inside that, capture the **natural bouts of conversation within the day**, because "there's going to be rapid shifts throughout the day, and it can be important to catch those."
- All of this belongs in the **Case Bible super index (CocoIndex)**; anything not there gets pushed through it.

### What exists today (checked 21:10)

- **Super index** (`Consignatio/Intake/backend/src/casebible_index`): files are cut by CocoIndex's splitter at 2,400 characters with 300 overlap. Chat messages are indexed one object per message (`ChatEvents20260918`). **There is no conversation-shaped chunk.**
- **Probata engine:** message "chunks" are 64 MB storage files per thread (`derive/smsthreads`); the only registered chunker is the Markdown one. **Nothing conversation-shaped here either.**

### Bouts within a day, measured (Katrina pool, days in America/Detroit)

| A new bout starts after silence of | Bouts | Per day | Typical bout | 90th pct | Single-message bouts |
|---|---:|---:|---:|---:|---:|
| 15 min | 11,532 | 8.6 | 3 | 24 | 2,828 |
| 30 min | 8,283 | 5.7 | 5 | 35 | 1,679 |
| 60 min | 5,735 | 3.7 | 7 | 53 | 988 |
| 2 h | 3,855 | 2.4 | 10 | 82 | 565 |

A silence split alone won't catch a shift **inside** a fast back-and-forth (friendly at 2:10, hostile by 2:25). That needs a content signal.

## Owner decisions and facts, 2026-09-24 03:18 EDT

- **"Try A and A":** a new bout after **30 minutes** of silence, within a day (America/Detroit); **tone** marks rapid shifts inside a bout. Pilot: the 2024 texts from Katrina's phone.
- **Case fact:** there was no child before **January 2020**. Messages before then can be case-related, since they establish patterns and behavior, but they cannot be child-related. Every labelling prompt must carry this whenever pre-2020 messages are included (Facebook 2018–2019).

## Owner direction, 2026-09-24 04:37 EDT: context versus hindsight

- Came from the owner's review of bout `c2024-b0274` (Sep 26), marked "missed a shift". The shift in her tone is nearly imperceptible. The owner hears it because he knows the patterns, and it only shows against the rest of the day.
- The owner judged the context-blind label **probably correct as it stands**. The contrast seen when looking back matters more: "it's going to be more important to see the hindsight comparison, and the contrast when looking back."
- The work is to strike the right balance of context versus hindsight. This is the project's knowledge-horizon mechanism (`AGENTS.md`, "WHY THIS EXISTS"), applied to tone:
  - an **as-lived** read sees only what came before;
  - a **hindsight** read sees the whole day and what came after;
  - the **difference** between the two is the signal.
- **Deferred (owner 04:38: "We aren't doing that part yet").** Nothing is built for it, and the question below waits until the owner raises it.
- **Owner context for the hindsight pass, 06:38.** The as-lived and discovery passes never see it; the hindsight pass reads each bout against it.
  - **Jul 27 2024 (c2024-b0142):** the discovery pass flagged Matt's "monitoring". The owner says monitoring has to be put into context, and that can be done down the road.
  - **Owner's account:** by then Katrina had been cheating fairly consistently for about 4.5 years that he knew of, three times with two different men.
  - **The recurring pattern he describes:** when he doesn't answer his phone for 30 seconds, or when he is at home or at his grandmother's, she swears at him and accuses him of sleeping with someone, even though she has his location. He has sent photos of himself in his room at his house to prove where he is. His reading is that this happens when she is out doing something herself.
  - **For the hindsight pass:** read Matt's monitoring against this known history. Check the accusations against shared location and the proof photos (location-sharing records, photos sent in the thread, dates of the infidelity he knew about).
- **Owner context, 07:12: blocked, and kept from his daughter.** The owner was blocked for long periods, during which Katrina would not let him see his daughter. (An "Oregon" in the dictation was a voice-typing error, owner 07:13; it has nothing to do with the case.) Upsetting exchanges and his yelling are often child-related and must be read that way. What was built for it:
  - Prompt v2.1 flags `child_related` on every observation and adds `contact_blocked_mentioned`. This stays neutral and within the chunk.
  - The timeline will show gaps in contact computed from the data, so blocked periods sit next to the bursts around them.
  - The hindsight pass reads the rest.
- **Owner direction, 06:40: flagging Matt's own behaviour is the point.** In the owner's account, she pushes, manufactures situations and is hypocritical, then points at him as the bad guy and will use that in court. Every place Matt's messages look like monitoring, anger or pressure is therefore an **exposure**. The discovery pass finds them; the hindsight pass answers each one: why it was reasonable, or how it reacts to the history and to what she did. This matches `claim_assertion` kind `exposure` in `docs/design/CLAIM-AND-ASSERTION-CANDIDATES-2026-08-29.md` (an adverse fact and how it is answered).
- **Owner direction, 06:42–06:43: every exposure flags a search for its context.** The search covers other sources, other media and other time periods, for whatever led to the action, reaction or comment. The owner expects that Katrina may have elicited it on purpose, or at least in practice as the expected reaction to what was already there. The context **may not be in a nearby time window**: it can be six months earlier. So the search is by subject across all history, not a date window. That is semantic search over the super index, plus the catalog, read with hindsight. Each exposure without context found becomes something the owner actively goes out to find. This is the "gap" or evidence-need seam in the 08-29 design.
- **Owner ruling, 06:39: no PII mitigation until court documents are being created.** It adds confusion for the owner and for the models, and he is the only user. Case names, facts and context go into the project docs and tables as they are. ~~An earlier copy was kept out of Git on the server (`persist/jev-eval/context/owner-context-for-hindsight.md`)~~; this doc is now the one place for it.
- Open (owner): how much the as-lived read sees. Options:
  - A. the bout alone (what bout-tone-v1 did);
  - B. earlier bouts that same day (the proposed default);
  - C. everything before it.
