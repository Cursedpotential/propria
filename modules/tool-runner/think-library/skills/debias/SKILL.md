---
name: debias
description: Check bias, framing, assumptions, perspective, and map-versus-territory errors. Use to challenge a read before trusting it, or route to a specific debias model at micro, focused, or full depth.
metadata:
  short-description: Bias, perspective, and map-versus-territory checks
  think-kind: family
  think-public: 'true'
  think-family: debias
  think-tags: bias,debias,perspective,assumptions,evidence,judgment
  think-registry: ../../skill-registry.json
---
# think · debias

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Bias, perspective, and map-versus-territory checks. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

## Operating contract

1. Keep the persistent Reflex loop active by loading `../thinking-model-router/references/reflex/SKILL.md` when it is not already active.
2. If the user names one of this family's models exactly, load that model and honor any requested depth (`micro`, `focused`, `full`).
3. Otherwise select the smallest model or coordinated set that fits the task. If the task crosses families or selection is unclear, hand routing to `../thinking-model-router/SKILL.md`.
4. Each underlying model contains its own **Micro / Reflex**, **Focused**, and **Full** branches. Start micro unless the user or task clearly requires more depth.
5. Multiple models must form one process with explicit roles/order. Do not produce independent framework reports unless requested.
6. A nested model returns its findings to the current process; it does not replace the parent task.

## Members (12)

| model | use it when | load |
|---|---|---|
| `thinking-debiasing` | Systematic checklist to identify and counteract cognitive biases in decision-making. Use before major decisions, when evaluating recommendations, or when stakes are high. | `references/thinking-debiasing/SKILL.md` |
| `thinking-dual-process` | Apply Kahneman's Dual-Process Theory to recognize when to trust intuition vs engage deliberate analysis. Use for high-stakes decisions, error-prone contexts, or when balancing speed vs accuracy. | `references/thinking-dual-process/SKILL.md` |
| `thinking-map-territory` | Recognize limits of mental models and diagrams. Use when models diverge from reality, debugging expectation mismatches, or questioning abstraction accuracy. | `references/thinking-map-territory/SKILL.md` |
| `thinking-circle-of-competence` | Know the boundaries of your expertise and operate within them. Use when evaluating opportunities, making decisions outside your domain, or assessing when to defer to experts. | `references/thinking-circle-of-competence/SKILL.md` |
| `anchoring` | First information disproportionately influences judgments and decisions, even when the initial value is arbitrary or irrelevant | `references/anchoring/SKILL.md` |
| `availability-heuristic` | Correct probability and frequency judgments distorted by mentally available examples by seeking base rate data before relying on vivid anecdotes | `references/availability-heuristic/SKILL.md` |
| `confirmation-bias` | Combat the tendency to seek evidence supporting existing beliefs by actively searching for disconfirming data and treating opinions as testable hypotheses | `references/confirmation-bias/SKILL.md` |
| `dunning-kruger-effect` | Low-skill individuals overestimate their competence because they lack the metacognitive ability to recognize their own incompetence | `references/dunning-kruger-effect/SKILL.md` |
| `fundamental-attribution-error` | Over-attributing others' behavior to their character while under-weighting situational factors that shape their actions | `references/fundamental-attribution-error/SKILL.md` |
| `halo-effect` | One positive trait creates a global positive impression, influencing judgments about unrelated characteristics | `references/halo-effect/SKILL.md` |
| `hindsight-bias` | The tendency to perceive past events as more predictable than they actually were, leading to distorted learning and overconfidence | `references/hindsight-bias/SKILL.md` |
| `sunk-cost-fallacy` | Continuing an endeavor due to previously invested resources (time, money, effort) rather than current and future costs and benefits, escalating commitment to failing courses of action | `references/sunk-cost-fallacy/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
