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
