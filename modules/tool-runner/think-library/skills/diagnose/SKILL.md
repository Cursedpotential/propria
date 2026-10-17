---
name: diagnose
description: Find root causes and analyze failures without jumping to the first explanation. Use for debugging, incidents, process failures, or routing to a diagnosis model at micro, focused, or full depth.
metadata:
  short-description: Root-cause, failure analysis, and structured diagnosis
  think-kind: family
  think-public: 'true'
  think-family: diagnose
  think-tags: diagnose,debug,root-cause,failure,causal,problem-analysis
  think-registry: ../../skill-registry.json
---
# think · diagnose

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Root-cause, failure analysis, and structured diagnosis. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

## Operating contract

1. Keep the persistent Reflex loop active by loading `../thinking-model-router/references/reflex/SKILL.md` when it is not already active.
2. If the user names one of this family's models exactly, load that model and honor any requested depth (`micro`, `focused`, `full`).
3. Otherwise select the smallest model or coordinated set that fits the task. If the task crosses families or selection is unclear, hand routing to `../thinking-model-router/SKILL.md`.
4. Each underlying model contains its own **Micro / Reflex**, **Focused**, and **Full** branches. Start micro unless the user or task clearly requires more depth.
5. Multiple models must form one process with explicit roles/order. Do not produce independent framework reports unless requested.
6. A nested model returns its findings to the current process; it does not replace the parent task.

## Members (5)

| model | use it when | load |
|---|---|---|
| `thinking-five-whys-plus` | Enhanced root cause analysis with explicit bias guards and stopping criteria. Use for incident post-mortems, bug investigations, and process failures where standard 5 Whys might mislead. | `references/thinking-five-whys-plus/SKILL.md` |
| `fishbone-diagram` | Visually map all potential causes of a problem across structured categories to systematically identify root causes through team collaboration | `references/fishbone-diagram/SKILL.md` |
| `current-reality-tree` | A logical diagram that maps cause-effect relationships between undesirable effects to identify root causes that drive most organizational problems | `references/current-reality-tree/SKILL.md` |
| `fault-tree-analysis` | Top-down deductive methodology for analyzing system failures by mapping logical relationships between faults, subsystems, and redundant safety elements using Boolean logic diagrams | `references/fault-tree-analysis/SKILL.md` |
| `thinking-kepner-tregoe` | Systematic rational process for complex problem analysis, decision making, and risk assessment. Use for high-stakes engineering decisions, root cause analysis beyond 5 Whys, and multi-factor evaluations requiring structured criteria. | `references/thinking-kepner-tregoe/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
