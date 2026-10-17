---
name: incentives
description: Analyze second-order consequences, incentives, tradeoffs, strategic interaction, opportunity cost, and value over time. Use directly or route to an incentives model at micro, focused, or full depth.
metadata:
  short-description: Consequences, incentives, tradeoffs, value over time, and game theory
  think-kind: family
  think-public: 'true'
  think-family: incentives
  think-tags: incentives,second-order,tradeoffs,game-theory,opportunity-cost,consequences
  think-registry: ../../skill-registry.json
---
# think · incentives

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Consequences, incentives, tradeoffs, value over time, and game theory. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

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
| `thinking-second-order` | Think beyond immediate consequences to second and third-order effects. Use for strategic decisions, policy changes, and avoiding unintended consequences. | `references/thinking-second-order/SKILL.md` |
| `thinking-opportunity-cost` | Evaluate decisions by what you give up, not just what you gain. Use for resource allocation, prioritization, build vs. buy choices, and technical debt evaluation. | `references/thinking-opportunity-cost/SKILL.md` |
| `goodharts-law` | Anticipate metric gaming and unintended optimization by recognizing that when a measure becomes a target, it ceases to be a good measure | `references/goodharts-law/SKILL.md` |
| `incentives` | Predict and shape behavior by understanding the hidden rewards and punishments that drive human action | `references/incentives/SKILL.md` |
| `principal-agent-problem` | Conflicts arising when one party (agent) makes decisions on behalf of another (principal), but their interests diverge and information is asymmetric | `references/principal-agent-problem/SKILL.md` |
| `mechanism-design` | Reverse game theory that engineers rules and incentive structures to achieve desired outcomes when participants have private information and self-interested motivations | `references/mechanism-design/SKILL.md` |
| `time-value-of-money` | Value money received today higher than the same amount in the future because present money can be invested to earn returns—use Net Present Value to compare cash flows across time and make rational investment decisions | `references/time-value-of-money/SKILL.md` |
| `compound-interest` | Leverage exponential growth by reinvesting gains to earn returns on returns, whether in finance, skills, relationships, or habits—small consistent improvements compound into transformational outcomes | `references/compound-interest/SKILL.md` |
| `nash-equilibrium` | Identify stable strategic outcomes where no player benefits from unilaterally changing their strategy - predict where competitive dynamics settle | `references/nash-equilibrium/SKILL.md` |
| `prisoners-dilemma` | Recognize situations where individual rational choices lead to collectively worse outcomes - core model for cooperation vs. defection dynamics | `references/prisoners-dilemma/SKILL.md` |
| `tit-for-tat` | Start new business relationships with trust and investment, but immediately match any breach. Return to cooperation as soon as they do. Make the pattern obvious through transparency. | `references/tit-for-tat/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
