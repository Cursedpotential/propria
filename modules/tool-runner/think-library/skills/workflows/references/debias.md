# Workflow: debias

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

Turn "am I being fair" into a structured split between what the evidence actually supports and
what the narrative wants to conclude, instead of a single confident read that quietly picked the
story up front and back-filled the reasoning. Built for high-stakes character or situation
judgments where getting it wrong has real consequences for a real person.

## When it fires

Trigger phrases: "am I being fair", "read this person", "is this biased", judging someone's
intent, character, or motive from limited or ambiguous evidence — anything where the honest answer
might be "the evidence doesn't actually support the conclusion I'd reach on gut feel."

## Roles

Single actor. No author/judge split.

## Steps

1. **Debiasing checklist.** Load
   `../debias/references/thinking-debiasing/SKILL.md`. Run the
   general sweep before the specific lenses below — name which biases are plausible risks in this
   specific read before checking for them one at a time.

2. **Fundamental attribution error.** Load
   `../debias/references/fundamental-attribution-error/SKILL.md`.
   Check whether the read attributes behavior to character ("they're inconsiderate") when a
   situational explanation ("they were in a meeting with a dead phone") is at least as consistent
   with the same evidence.

3. **Confirmation bias.** Load
   `../debias/references/confirmation-bias/SKILL.md`. Actively go
   looking for evidence that *disconfirms* the read, not just re-list evidence that supports it.
   This step fails if it produces zero disconfirming candidates even after actually looking.

4. **Halo effect.** Load
   `../debias/references/halo-effect/SKILL.md`. Check whether one
   trait (good or bad) is coloring judgment of unrelated behavior — "they're generally reliable, so
   this missed deadline must have a good excuse" or the negative mirror of that.

5. **Anchoring.** Load
   `../debias/references/anchoring/SKILL.md`. Check whether the
   first piece of information received about this situation is doing more work in the conclusion
   than the evidence that came in afterward actually warrants.

6. **Outside view.** Load
   `../forecast/references/outside-view-inside-view/SKILL.md`.
   Set the specific story aside and ask: in situations like this one, generically, what's the base
   rate for the charitable explanation versus the uncharitable one?

## GATE

Invalid — say so explicitly — if:

- The "evidence" column and the "story wants" column end up saying the same thing (that means the
  bias check wasn't actually engaged — the two columns exist specifically to be capable of
  disagreeing).
- Step 3 (confirmation bias) produces zero actively-sought disconfirming facts — "I looked and
  found nothing" is only valid if the search is shown, not asserted.
- Fewer than all six lenses are actually touched (a debias run that skips halo or anchoring because
  they "don't seem relevant" must say so explicitly with a one-line reason, not silently omit them).
- The net read has no confidence/hedge attached — treating the debiased conclusion as certain
  defeats the purpose of the workflow.

## OUTPUT TEMPLATE

```markdown
# Debias: <the situation being read>

## Situation
<what happened, stated as observably as possible, before interpretation>

## Evidence Column (what's actually supported)
- <fact, with source: message/log/direct observation>
- <fact>

## Story Column (what the narrative wants to conclude)
- <the uncharitable/charitable interpretation being pulled toward>
- <why it's tempting — which bias is doing the pulling>

## Bias Flags
| Lens | Fired? | Note |
|---|---|---|
| Fundamental attribution error | yes/no | <situational explanation considered> |
| Confirmation bias | yes/no | <disconfirming facts actively sought> |
| Halo effect | yes/no | <unrelated trait bleeding into this judgment> |
| Anchoring | yes/no | <first-information weight check> |
| Outside view | yes/no | <base rate for situations like this> |

## Net Read
<the revised, hedged conclusion — not certainty>
Confidence: <low/medium/high>
```

## Worked example (synthetic, no real people)

**Situation:** a coworker went silent in a video meeting for several minutes and didn't respond to
messages during that window; the read forming is "they're checked out and don't take this seriously."

```markdown
# Debias: coworker's silence during the planning meeting

## Situation
During a 45-minute planning call, coworker's video and mic were on but they did not speak or react
for approximately 8 minutes, and did not respond to two direct chat messages during that window.
They rejoined the conversation normally afterward with no explanation offered.

## Evidence Column (what's actually supported)
- 8 minutes of no verbal/chat response, confirmed from the call recording timestamp.
- They resumed participating normally afterward, contributing a relevant point.
- No pattern of this in the prior 6 meetings (checked calendar/call history).

## Story Column (what the narrative wants to conclude)
- "They're checked out and don't take this project seriously."
- Tempting because: it's the first explanation that comes to mind for silence, and it lines up
  with a mild frustration already present about this project's pace (anchoring on that frustration).

## Bias Flags
| Lens | Fired? | Note |
|---|---|---|
| Fundamental attribution error | yes | Situational explanations (connectivity, a side interruption, muted-but-thought-unmuted) fit the same 8-minute window equally well and weren't considered first |
| Confirmation bias | yes | Initially only recalled evidence for "checked out"; actively checking, found they've had zero other silences in 6 prior meetings, which cuts the other way |
| Halo effect | no | No unrelated trait (good or bad) about this person is visibly coloring the read |
| Anchoring | yes | Existing frustration about project pace is doing real work in reaching for the least charitable read of an ambiguous 8 minutes |
| Outside view | yes | Base rate: unexplained short silences on video calls are usually connectivity/interruption, not disengagement, absent a pattern — and there is no pattern here |

## Net Read
The evidence supports "something interrupted them for 8 minutes and they didn't announce it" more
than it supports "they don't take this seriously" — the latter is being pulled in by pre-existing
frustration (anchoring) and a first-guess attribution to character rather than situation. Worth a
low-key, non-accusatory check-in ("everything okay on your end mid-call? froze up on my screen")
rather than treating disengagement as established.
Confidence: medium — the base rate favors the charitable read, but this is genuinely unverified
either way without asking.
```
