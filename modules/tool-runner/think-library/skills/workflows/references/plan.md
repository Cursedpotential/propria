# Workflow: plan

> **Integrated execution:** Treat the named thinking models below as stages/lenses inside this one workflow. Start each supporting lens at Micro unless the step needs more depth; escalate that same model to Focused/Full only when warranted. Feed its result into the next stage. Do not emit separate model reports or abandon the workflow. Keep Reflex active throughout.


> _Byline: Claude Code · Fable 5.1 · 2026-09-07._

## Purpose

Turn "plan this" into a plan that names its own bottleneck and its own buffers, instead of a list
of steps that all look equally important and equally certain. A plan that doesn't say which part
is the constraint isn't a plan, it's a wish list with dates on it.

## When it fires

Trigger phrases: "plan this", kickoff, roadmap, "how should we sequence this," starting a
multi-step piece of work with more than a couple of dependent stages.

## Roles

Single actor. No author/judge split.

## Steps

1. **First principles.** Load
   `../reason/references/thinking-first-principles/SKILL.md`.
   Decompose the goal into its fundamental requirements and constraints — the things that are true
   regardless of which approach gets picked — before proposing any sequence.

2. **Theory of constraints (the bottleneck).** Load
   `../systems/references/thinking-theory-of-constraints/SKILL.md`.
   Name **the** single bottleneck — the one resource, dependency, or unknown that caps how fast
   everything else can go. Improvements anywhere else don't matter until this is addressed.

3. **Fermi sizing.** Load
   `../forecast/references/thinking-fermi-estimation/SKILL.md`.
   Produce an order-of-magnitude estimate of effort/time/cost by decomposing into estimable
   factors — not a single guessed number with no visible arithmetic.

4. **Margin of safety.** Load
   `../decide/references/thinking-margin-of-safety/SKILL.md`. State
   the buffer explicitly — as a number or a percentage, against the Fermi estimate — and what it's
   protecting against (unknown unknowns, not just "things always take longer").

5. **Pre-mortem.** Load
   `../decide/references/thinking-pre-mortem/SKILL.md`. It is months
   later, the plan failed, name the top causes before a single step is executed.

6. **Sequence with human gates.** Order the work into stages. Every stage that produces something
   irreversible, costs real money, or touches production gets an explicit **go/no-go gate** a human
   must clear before the next stage starts — this is not optional and is not satisfied by "the
   agent will ask if something seems off."

## GATE

Invalid — say so explicitly — if:

- No single bottleneck is named (zero, or more than one "the" bottleneck without a stated reason
  the plan genuinely has co-equal constraints, is a smell: theory of constraints says there is
  always exactly one binding constraint at a time).
- No margin-of-safety buffer is stated as a number/percentage.
- The sequenced plan has zero human gates on stages that are costly, irreversible, or
  production-facing.
- The Fermi estimate (step 3) is a single number with no visible decomposition into factors.

## OUTPUT TEMPLATE

```markdown
# Plan: <the goal>

## Fundamentals (first principles)
- <requirement/constraint that's true regardless of approach>
- <requirement/constraint 2>

## The Constraint
Bottleneck: <the one thing capping speed>
Why it's the constraint, not something else: <one line>

## Fermi Sizing
| Factor | Estimate | Basis |
|---|---|---|
| <factor 1> | <order-of-magnitude> | <how estimated> |
| <factor 2> | <order-of-magnitude> | <how estimated> |
**Total estimate:** <range>

## Margin of Safety
Buffer: <number/percentage> on top of the Fermi estimate
Protecting against: <what unknown unknown this covers>

## Pre-Mortem (top causes of failure, ranked)
1. <cause>
2. <cause>
3. <cause>

## Sequenced Plan
| # | Stage | Depends on | Human gate? |
|---|---|---|---|
| 1 | <stage> | — | <gate condition, or "none — cheap/reversible"> |
| 2 | <stage> | 1 | <gate condition> |
```

## Worked example (synthetic)

**Goal:** migrate a legacy PHP monolith's authentication module from session cookies to OAuth2.

```markdown
# Plan: migrate auth module from session cookies to OAuth2

## Fundamentals (first principles)
- Existing sessions must keep working until every client has migrated — no flag day.
- The identity provider choice is independent of the migration mechanics; pick it once, don't
  re-litigate it mid-migration.
- Every endpoint currently trusts a session cookie implicitly; each one needs an explicit,
  auditable authorization check regardless of which token scheme replaces the cookie.

## The Constraint
Bottleneck: the number of endpoints (140+) that implicitly trust the session cookie and have no
existing authorization test coverage.
Why it's the constraint, not something else: OAuth2 library integration is a known, bounded task;
auditing 140 endpoints one at a time for what they actually assume about the caller is not bounded
until someone does it, and nothing else in the plan can be parallelized safely without it.

## Fermi Sizing
| Factor | Estimate | Basis |
|---|---|---|
| Endpoint audit | ~140 endpoints x 30min each = ~70 hours | sampled 10 endpoints, averaged 32min |
| OAuth2 library integration | ~40 hours | comparable integration done on a sibling service in 35h |
| Dual-auth transition period tooling | ~24 hours | needs a shim accepting both cookie and token |
**Total estimate:** 130–170 hours (roughly 4-5 engineer-weeks)

## Margin of Safety
Buffer: +40% on top of the 170-hour high end -> plan for ~240 hours.
Protecting against: endpoints whose implicit trust assumptions are worse than the sampled 10
suggested (e.g. internal-only endpoints nobody expected external callers to reach).

## Pre-Mortem (top causes of failure, ranked)
1. An endpoint outside the audited 140 also relies on the session cookie (an internal cron-triggered
   webhook, say) and breaks silently when cookies are eventually removed.
2. The dual-auth shim has a bug that accepts an expired token as if it were a valid cookie session,
   introduced during the transition window and missed because it's "temporary" code.
3. The chosen identity provider's rate limits are hit during the transition period because every
   request now round-trips to it instead of checking a local session store.

## Sequenced Plan
| # | Stage | Depends on | Human gate? |
|---|---|---|---|
| 1 | Audit all 140 endpoints for auth assumptions | — | none — read-only, reversible |
| 2 | Pick + integrate OAuth2 library against a staging IdP | 1 | none — staging only |
| 3 | Build dual-auth shim (accepts cookie OR token) | 2 | Yes — security review of the shim before it touches any traffic |
| 4 | Roll dual-auth to 5% of production traffic | 3 | Yes — owner sign-off, rollback plan confirmed |
| 5 | Ramp to 100%, then remove cookie path | 4 | Yes — owner sign-off after 2 weeks with zero auth-related incidents |
```
