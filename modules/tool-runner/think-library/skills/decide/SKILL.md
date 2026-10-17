---
name: decide
description: Improve decisions using reversibility, risk, pre-mortems, red teams, prioritization, and adaptive decision loops. Use directly or route to a decision model at micro, focused, or full depth.
metadata:
  short-description: Decision quality, reversibility, risk, prioritization, and adversarial checks
  think-kind: family
  think-public: 'true'
  think-family: decide
  think-tags: decision,reversibility,risk,premortem,red-team,priority,ooda
  think-registry: ../../skill-registry.json
---
# think · decide

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Decision quality, reversibility, risk, prioritization, and adversarial checks. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

## Operating contract

1. Keep the persistent Reflex loop active by loading `../thinking-model-router/references/reflex/SKILL.md` when it is not already active.
2. If the user names one of this family's models exactly, load that model and honor any requested depth (`micro`, `focused`, `full`).
3. Otherwise select the smallest model or coordinated set that fits the task. If the task crosses families or selection is unclear, hand routing to `../thinking-model-router/SKILL.md`.
4. Each underlying model contains its own **Micro / Reflex**, **Focused**, and **Full** branches. Start micro unless the user or task clearly requires more depth.
5. Multiple models must form one process with explicit roles/order. Do not produce independent framework reports unless requested.
6. A nested model returns its findings to the current process; it does not replace the parent task.

## Members (14)

| model | use it when | load |
|---|---|---|
| `thinking-reversibility` | Classify decisions by reversibility and match decision process to decision type. Use for technology choices, architecture decisions, process changes, and hiring decisions. | `references/thinking-reversibility/SKILL.md` |
| `thinking-regret-minimization` | Project to your future self and ask what you would regret not doing. Use for career decisions, strategic pivots, risk-taking choices, and life-changing decisions. | `references/thinking-regret-minimization/SKILL.md` |
| `thinking-inversion` | Approach problems backward by identifying paths to failure, then systematically avoiding them. Use for risk identification, planning, and avoiding obvious mistakes. | `references/thinking-inversion/SKILL.md` |
| `thinking-pre-mortem` | Imagine a project has failed and work backward to identify why. Use at project kickoffs, before major decisions, or when optimism may be obscuring risks. | `references/thinking-pre-mortem/SKILL.md` |
| `thinking-red-team` | Deliberately attack your own plans, systems, and assumptions to find weaknesses before adversaries or reality does. Use for security review, architecture validation, plan stress-testing, and pre-launch preparation. | `references/thinking-red-team/SKILL.md` |
| `thinking-margin-of-safety` | Build in buffers for unknown unknowns and don't optimize to the edge. Use for capacity planning, deadline estimation, architecture design, and risk management. | `references/thinking-margin-of-safety/SKILL.md` |
| `thinking-via-negativa` | Improve by removal rather than addition. Focus on what to stop doing, eliminate the negative, and subtract complexity. Use for system simplification, process improvement, and feature prioritization. | `references/thinking-via-negativa/SKILL.md` |
| `schelling-fence` | Strategic precommitment to defend an arbitrary line on a slippery slope. Use when you face a continuum from acceptable to unacceptable and need a conspicuous, defensible boundary. | `references/schelling-fence/SKILL.md` |
| `thinking-bounded-rationality` | Apply Herbert Simon's Bounded Rationality and satisficing to make good-enough decisions under real-world constraints. Use for design decisions under time pressure, recognizing cognitive limits, and setting appropriate stopping criteria. | `references/thinking-bounded-rationality/SKILL.md` |
| `thinking-ooda` | Rapid decision-making loop for dynamic situations. Use for incident response, competitive scenarios, time-sensitive decisions, and situations requiring quick adaptation. | `references/thinking-ooda/SKILL.md` |
| `thinking-cynefin` | Classify problems by complexity domain (clear, complicated, complex, chaotic) and match approach to domain. Use for choosing methodologies, problem framing, and process design. | `references/thinking-cynefin/SKILL.md` |
| `thinking-lindy-effect` | For non-perishable things, future life expectancy is proportional to current age. Use for technology selection, evaluating frameworks/libraries, and predicting tool longevity. | `references/thinking-lindy-effect/SKILL.md` |
| `eisenhower-matrix` | Prioritize tasks by urgency and importance—focus on important but not urgent work to prevent firefighting | `references/eisenhower-matrix/SKILL.md` |
| `rice-prioritization` | Scoring framework using Reach, Impact, Confidence, and Effort to objectively prioritize product features and projects | `references/rice-prioritization/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
