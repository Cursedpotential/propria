# Workflow: verify-claim

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

Settle a factual assertion, citation, or number into exactly one of four honest states instead of
smoothing it into confident prose. Adapted from the four-tier disposition used for legal drafting
in `family-court-toolkit`'s `custody-guide/GUARDRAILS.md` (VERIFIED / PROVISIONAL / CONFLICTED /
its fourth state there is UNVERIFIED — this workflow generalizes that state to **UNSUPPORTED** so
it reads correctly outside a legal-drafting context, while keeping the same discipline: a claim
that cannot be checked ships only with a visible marker, never silently folded into the rest of the
text as if it were settled).

## When it fires

Trigger phrases: "verify this claim", "citation", "is that true", "fact check", "check this
citation," any number, quote, or "fact" that will be relied on in a filing, a report, a commit
message, or anything sent to the owner as settled.

## Roles

Single actor, and prefer that this actor **not** be the one who first asserted the claim, when
that's practical — a claim's originator has already anchored on believing it (see `anchoring` in
the `debias` family if the originator is doing their own verification and wants to check that risk).

## Steps

1. **Sagan standard.** Load
   `../reason/references/sagan-standard/SKILL.md`. Calibrate how much
   evidence this claim actually needs before proceeding — a routine claim ("the function is defined
   on line 40") needs less than an extraordinary one ("this circuit split means the statute doesn't
   apply here"). State the evidence bar before looking for evidence, not after.

2. **Hitchens's razor screen.** Load
   `../reason/references/hitchens-razor/SKILL.md`. If the claim as
   currently stated has zero evidence behind it, that's an immediate `UNSUPPORTED` — do not proceed
   to the later steps hunting for a way to upgrade it; state the screen result first.

3. **Source hierarchy.** Rank whatever sources exist: a primary source (the statute/opinion/code/
   file itself) beats a secondary source (a summary, a blog post, a second-hand description) beats
   an unsourced assertion. State which tier the best available source for this claim actually sits
   at — this step has no lens file; it is the procedural core of the workflow.

4. **Calibration.** Load
   `../forecast/references/calibration/SKILL.md`. State a numeric
   confidence alongside the status label — a status of PROVISIONAL at 55% confidence and one at 90%
   confidence both belong in the ledger, but they are not the same claim strength.

5. **Status label.** Assign exactly one label — see STRICT MODE.

## STRICT MODE (always on for this workflow — no opt-out)

The status label in step 5 must be **exactly one** of:

- **VERIFIED** — checked directly against a primary source, that source's location/date recorded.
- **PROVISIONAL** — best available source is secondary, stale, or currently unconfirmable, and the
  claim ships anyway with a visible marker naming what's unconfirmed and who/what can confirm it.
- **CONFLICTED** — credible sources disagree; both positions are stated, no resolution is asserted.
- **UNSUPPORTED** — no adequate evidence exists for the claim as stated; it does not ship as prose,
  only as a flagged row.

No other word (e.g. "probably true," "likely," "confirmed," "unverified" as a synonym drift from
UNSUPPORTED) is a valid status in this workflow's output. Hedged prose in place of one of these four
words is a gate failure, not a softer answer.

## GATE

Invalid — say so explicitly — if:

- The status is anything other than the four words above.
- Status is VERIFIED but no primary source is cited (location + date/version).
- Status is CONFLICTED but only one position is actually stated.
- Status is PROVISIONAL but no resolution path (who/what would confirm it) is named.
- Status is UNSUPPORTED but the claim is nonetheless carried forward elsewhere as settled fact in
  the same output (an UNSUPPORTED claim must not silently reappear as if verified).

## OUTPUT TEMPLATE

```markdown
# Verify: <the claim, quoted exactly as asserted>

## Sagan Check
Evidence bar for a claim like this: <low/medium/high, one line why>

## Hitchens Screen
Any evidence at all offered for this claim? <yes/no>
If no -> status is UNSUPPORTED, skip to Status.

## Source(s) Consulted
| Source | Tier (primary/secondary/none) | Location/date |
|---|---|---|
| <source> | <tier> | <where, when> |

## Status
**VERIFIED / PROVISIONAL / CONFLICTED / UNSUPPORTED**
Confidence: <NN>%

## Resolution Path (PROVISIONAL / CONFLICTED / UNSUPPORTED only)
<who or what can move this to VERIFIED, and how>
```

## Worked example (synthetic, legal-research flavored — no real people or real jurisdictions)

**Claim as asserted:** "In the fictional State of Ashford, the statute of limitations for a breach
of a residential lease is three years from the date of breach."

```markdown
# Verify: "the statute of limitations for a residential lease breach in Ashford is 3 years"

## Sagan Check
Evidence bar for a claim like this: high — a filing deadline is exactly the kind of claim where
being wrong has a concrete, irreversible cost (a claim filed late gets dismissed), so this needs a
primary source, not a summary.

## Hitchens Screen
Any evidence at all offered for this claim? Yes — a secondary source (a legal-aid FAQ page) states
"three years," but the underlying statute section was not cited or checked.

## Source(s) Consulted
| Source | Tier (primary/secondary/none) | Location/date |
|---|---|---|
| Ashford legal-aid FAQ, "Lease Disputes" page | secondary | accessed today, no revision date shown |
| Ashford Civil Code (the actual limitations statute) | primary | not yet located/read |

## Status
**PROVISIONAL**
Confidence: 55%

## Resolution Path (PROVISIONAL / CONFLICTED / UNSUPPORTED only)
Locate and read the actual Ashford Civil Code limitations section for contract/lease claims (the
FAQ page does not cite it); confirm whether "three years" is the general contract limitations
period being applied here rather than a lease-specific rule, since those sometimes differ. Until
read, do not state "three years" as settled in any filing.
```
