---
name: forecast
description: Estimate probabilities, ranges, uncertainty, calibration, and plausible outcomes. Use for forecasting, sizing uncertainty, or routing to a specific forecasting model at micro, focused, or full depth.
metadata:
  short-description: Probability, estimation, uncertainty, and calibration
  think-kind: family
  think-public: 'true'
  think-family: forecast
  think-tags: forecast,probability,uncertainty,estimation,calibration,risk
  think-registry: ../../skill-registry.json
---
# think · forecast

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Probability, estimation, uncertainty, and calibration. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

## Operating contract

1. Keep the persistent Reflex loop active by loading `../thinking-model-router/references/reflex/SKILL.md` when it is not already active.
2. If the user names one of this family's models exactly, load that model and honor any requested depth (`micro`, `focused`, `full`).
3. Otherwise select the smallest model or coordinated set that fits the task. If the task crosses families or selection is unclear, hand routing to `../thinking-model-router/SKILL.md`.
4. Each underlying model contains its own **Micro / Reflex**, **Focused**, and **Full** branches. Start micro unless the user or task clearly requires more depth.
5. Multiple models must form one process with explicit roles/order. Do not produce independent framework reports unless requested.
6. A nested model returns its findings to the current process; it does not replace the parent task.

## Members (11)

| model | use it when | load |
|---|---|---|
| `thinking-probabilistic` | Express confidence in ranges, update predictions with new information, and track calibration over time. Use for project estimation, risk assessment, and decision making under uncertainty. | `references/thinking-probabilistic/SKILL.md` |
| `thinking-bayesian` | Update beliefs systematically based on new evidence using probabilistic reasoning. Use when estimating probabilities, learning from data, or making decisions under uncertainty. | `references/thinking-bayesian/SKILL.md` |
| `thinking-fermi-estimation` | Make order-of-magnitude estimates for unknown quantities by decomposing into known or estimable factors. Use for capacity planning, cost estimation, market sizing, and technical feasibility assessment. | `references/thinking-fermi-estimation/SKILL.md` |
| `calibration` | Align predicted probabilities with actual outcomes - train yourself so things you say are 70% likely actually happen 70% of the time | `references/calibration/SKILL.md` |
| `superforecasting` | Make probabilistic predictions when strategic decisions require accurate risk assessment | `references/superforecasting/SKILL.md` |
| `tetlocks-10-commandments` | Improve forecast accuracy by following Philip Tetlock's 10 evidence-based principles from superforecaster research distinguishing top 2% predictors | `references/tetlocks-10-commandments/SKILL.md` |
| `thinking-in-bets` | Frame decisions as probability bets instead of certainties - separate decision quality from outcome quality to avoid resulting bias | `references/thinking-in-bets/SKILL.md` |
| `outside-view-inside-view` | Forecast by comparing to similar past cases (outside view) instead of analyzing unique project details (inside view) - cures planning fallacy | `references/outside-view-inside-view/SKILL.md` |
| `wisdom-of-crowds-aggregation` | Combine diverse independent estimates to produce more accurate predictions than any single expert, when conditions support collective intelligence | `references/wisdom-of-crowds-aggregation/SKILL.md` |
| `monte-carlo-simulation` | Probabilistic modeling technique that runs thousands of scenarios with random variable inputs to quantify uncertainty and assess risk distributions | `references/monte-carlo-simulation/SKILL.md` |
| `sensitivity-analysis` | Systematic testing of how changes in input variables (assumptions) affect output results, revealing which factors drive outcomes and where model risk is concentrated. | `references/sensitivity-analysis/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
