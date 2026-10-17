---
name: systems
description: Analyze interacting components, dependencies, feedback, constraints, bottlenecks, and leverage. Use for cross-component problems or route to a systems model at micro, focused, or full depth.
metadata:
  short-description: Systems, feedback, constraints, dependencies, and leverage
  think-kind: family
  think-public: 'true'
  think-family: systems
  think-tags: systems,feedback,constraints,dependencies,leverage,architecture
  think-registry: ../../skill-registry.json
---
# think · systems

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Systems, feedback, constraints, dependencies, and leverage. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

## Operating contract

1. Keep the persistent Reflex loop active by loading `../thinking-model-router/references/reflex/SKILL.md` when it is not already active.
2. If the user names one of this family's models exactly, load that model and honor any requested depth (`micro`, `focused`, `full`).
3. Otherwise select the smallest model or coordinated set that fits the task. If the task crosses families or selection is unclear, hand routing to `../thinking-model-router/SKILL.md`.
4. Each underlying model contains its own **Micro / Reflex**, **Focused**, and **Full** branches. Start micro unless the user or task clearly requires more depth.
5. Multiple models must form one process with explicit roles/order. Do not produce independent framework reports unless requested.
6. A nested model returns its findings to the current process; it does not replace the parent task.

## Members (7)

| model | use it when | load |
|---|---|---|
| `thinking-systems` | Analyze problems as interconnected systems with feedback loops, emergent behavior, and non-linear effects. Use for debugging complex systems, architecture decisions, and understanding unexpected behavior. | `references/thinking-systems/SKILL.md` |
| `thinking-feedback-loops` | Analyze systems using Donella Meadows' feedback loop framework to identify reinforcing loops, balancing loops, delays, and leverage points. Use for organizational dynamics, product growth design, debugging runaway or oscillating systems, and finding high-impact interventions. | `references/thinking-feedback-loops/SKILL.md` |
| `thinking-archetypes` | Recognize Senge's Systems Archetypes to diagnose recurring organizational and technical problems, identify why fixes keep failing, and design interventions that address root structure. | `references/thinking-archetypes/SKILL.md` |
| `thinking-leverage-points` | Identify where small changes can have large effects using Donella Meadows' hierarchy of system intervention points. Use for strategic decisions, system optimization, and choosing where to focus engineering effort. | `references/thinking-leverage-points/SKILL.md` |
| `thinking-theory-of-constraints` | Identify and manage the bottleneck; improvements elsewhere don't matter until the constraint is addressed. Use for performance optimization, process improvement, and resource allocation. | `references/thinking-theory-of-constraints/SKILL.md` |
| `causal-loop-diagrams` | Map cause-and-effect relationships with reinforcing/balancing feedback loops to visualize system structure and predict behavior patterns | `references/causal-loop-diagrams/SKILL.md` |
| `graph-thinking` | Apply graph-based thinking to visualize complex relationships and solve problems non-linearly. Use when mapping dependencies, analyzing systems, exploring interconnected concepts, or designing architectures. | `references/graph-thinking/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
