# Workflow: decide

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

Turn "which one" into a decision record instead of a vibe. Forces the reversibility class onto
the table before the options get argued, and ends with named tripwires so the decision can be
reopened on evidence instead of quietly re-litigated whenever someone feels like it.

## When it fires

Trigger phrases: "which one", "should we", "A vs B", "pick between", any fork with 2+ genuinely
live options where picking wrong is costly enough to be worth 10 minutes of structure.

## Roles

Single actor. No author/judge split — the workflow's discipline comes from forcing every option
through the same six lenses, not from adversarial roles.

## Steps

1. **Reversibility class.** Load
   `../decide/references/thinking-reversibility/SKILL.md`. Classify
   the decision Type 1 (hard/costly to reverse) or Type 2 (cheap to reverse) *before* comparing
   options — this sets how much of the rest of the workflow is worth doing.

2. **Opportunity cost.** Load
   `../incentives/references/thinking-opportunity-cost/SKILL.md`.
   For each option, name the best alternative given up — not "we lose flexibility" in the abstract,
   the actual next-best option foregone.

3. **Second-order effects.** Load
   `../incentives/references/thinking-second-order/SKILL.md`. For
   each option, name at least two consequences beyond the immediate one — what happens *after* the
   first effect lands.

4. **Inversion (paths to failure).** Load
   `../decide/references/thinking-inversion/SKILL.md`. For each
   option, list the concrete ways it fails — work backward from "this was the wrong call" to why.

5. **Kepner-Tregoe grid.** Load
   `../diagnose/references/thinking-kepner-tregoe/SKILL.md`. Build a
   weighted criteria table scoring every option against the same criteria — weights assigned before
   scores are filled in, not after (assigning weights after seeing scores is how KT grids get
   reverse-engineered to a foregone conclusion).

6. **Regret check.** Load
   `../decide/references/thinking-regret-minimization/SKILL.md`. Ask
   what the version of you five years out would regret *not* having tried, for each live option.

7. **Decision + tripwires.** Synthesize: state the choice, restate its reversibility class, name
   the cost given up (from step 2), and — this is the part usually skipped — name explicit
   tripwires: observable conditions that mean "reopen this decision," stated in advance, not
   invented after something goes wrong.

## GATE

Invalid — say so explicitly — if:

- Fewer than two options were actually compared (a "decision" with one option is a plan, use the
  `plan` workflow instead).
- The reversibility class (step 1) is never stated, or is asserted without the one-line rationale
  for why it's Type 1 or Type 2.
- The Kepner-Tregoe grid (step 5) has scores but no weights, or the weights were visibly chosen
  after the scores to justify a pre-decided winner.
- Zero tripwires are named in the final decision (a decision with no reopening condition is a
  decision nobody can ever revisit on new evidence — that is itself a failure mode, not neutrality).

## OUTPUT TEMPLATE

```markdown
# Decide: <the fork>

## Options
1. <option A>
2. <option B>
3. <option N, if any>

## Reversibility
Class: Type 1 (hard to reverse) / Type 2 (cheap to reverse)
Why: <one line>

## Opportunity Cost (per option)
| Option | Best alternative given up |
|---|---|
| A | <specific alternative> |
| B | <specific alternative> |

## Second-Order Effects (per option, >= 2 each)
| Option | Effect 1 | Effect 2 |
|---|---|---|
| A | <after the first effect, then...> | <...> |
| B | <...> | <...> |

## Paths to Failure (inversion, per option)
| Option | How it fails |
|---|---|
| A | <concrete failure mode> |
| B | <concrete failure mode> |

## Kepner-Tregoe Grid
| Criterion (weight) | Option A | Option B |
|---|---|---|
| <criterion 1> (<weight>) | <score> | <score> |
| <criterion 2> (<weight>) | <score> | <score> |
| **Weighted total** | <total> | <total> |

## Regret Check
5 years out, I'd regret not trying: <option, and why>

## Decision
**Choice:** <option>
**Reversibility class:** Type 1 / Type 2
**Cost given up:** <from opportunity cost step>

## Tripwires (reopen this decision if...)
- <observable condition 1>
- <observable condition 2>
```

## Worked example (synthetic)

**Fork:** migrate a service's data layer via a big-bang cutover to a new schema, or run dual-writes
to old and new schemas until confidence is high, then cut over reads.

```markdown
# Decide: schema migration strategy for the orders service

## Options
1. Big-bang cutover (stop writes, migrate, resume on new schema)
2. Dual-write (write both schemas, backfill, cut reads over gradually)

## Reversibility
Class: Type 1 for big-bang (a bad migration under load is hard to unwind mid-cutover); Type 2 for
dual-write (each stage can be paused or rolled back independently).
Why: big-bang has one irreversible window; dual-write's stages are each individually reversible.

## Opportunity Cost (per option)
| Option | Best alternative given up |
|---|---|
| Big-bang | The two engineering-weeks it would take to build dual-write tooling, spent elsewhere |
| Dual-write | The one maintenance-window simplicity of big-bang; ~3 extra weeks of dual-schema code |

## Second-Order Effects (per option, >= 2 each)
| Option | Effect 1 | Effect 2 |
|---|---|---|
| Big-bang | Requires a maintenance window customers notice | A failed cutover mid-window has no partial-rollback path, forcing a full restore from backup |
| Dual-write | Backfill job competes for DB I/O with production traffic for weeks | Dual-write bugs (writing to one schema but not the other) can silently diverge the two datasets |

## Paths to Failure (inversion, per option)
| Option | How it fails |
|---|---|
| Big-bang | Migration script has an untested edge case on production data volume; window overruns; no rollback |
| Dual-write | Divergence between schemas goes undetected because nobody wrote a reconciliation check |

## Kepner-Tregoe Grid
| Criterion (weight) | Big-bang | Dual-write |
|---|---|---|
| Time to ship (0.2) | 9 | 4 |
| Risk if wrong (0.4) | 2 | 7 |
| Ops complexity during (0.2) | 8 | 4 |
| Rollback granularity (0.2) | 1 | 8 |
| **Weighted total** | 4.4 | 6.2 |

## Regret Check
5 years out, I'd regret not trying dual-write more than I'd regret the extra weeks — a
schema-corruption incident from a rushed big-bang is the kind of thing that gets remembered.

## Decision
**Choice:** Dual-write with a reconciliation job before the read cutover.
**Reversibility class:** Type 2 (staged, each stage independently reversible).
**Cost given up:** the simplicity and shorter timeline of a single maintenance window.

## Tripwires (reopen this decision if...)
- The reconciliation job finds >0.1% divergence between schemas after two weeks of dual-write.
- The backfill job's I/O contention measurably degrades production query latency (p95 up >20%).
```
