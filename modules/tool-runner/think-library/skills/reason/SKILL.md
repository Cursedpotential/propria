---
name: reason
description: Test claims, assumptions, explanations, evidence, and first principles. Use for general reasoning or to route to a specific reasoning model at micro, focused, or full depth.
metadata:
  short-description: General reasoning, claims, assumptions, and explanation
  think-kind: family
  think-public: 'true'
  think-family: reason
  think-tags: reason,logic,evidence,assumptions,claims,first-principles,scientific
  think-registry: ../../skill-registry.json
---
# think · reason

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


General reasoning, claims, assumptions, and explanation. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

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
| `thinking-occams-razor` | Apply parsimony principle to prefer simpler explanations with fewer assumptions. Use for hypothesis selection in debugging, architecture decisions, and choosing between competing approaches. | `references/thinking-occams-razor/SKILL.md` |
| `hanlons-razor` | Never attribute to malice that which is adequately explained by incompetence | `references/hanlons-razor/SKILL.md` |
| `hitchens-razor` | What can be asserted without evidence can be dismissed without evidence | `references/hitchens-razor/SKILL.md` |
| `sagan-standard` | Calibrate evidence requirements proportional to claim implausibility when evaluating extraordinary assertions | `references/sagan-standard/SKILL.md` |
| `humes-guillotine` | You cannot derive an 'ought' from an 'is' without bridging logic | `references/humes-guillotine/SKILL.md` |
| `poppers-falsifiability` | Scientific theories must be testable and capable of being proven false | `references/poppers-falsifiability/SKILL.md` |
| `chestertons-fence` | Before removing something, understand why it was put there in the first place—reforms without understanding cause unintended consequences | `references/chestertons-fence/SKILL.md` |
| `thinking-scientific-method` | Hypothesis → Prediction → Test → Revise with explicit falsification. Use for debugging, feature experimentation, performance investigation, and A/B testing design. | `references/thinking-scientific-method/SKILL.md` |
| `thinking-socratic` | Systematic questioning framework to deepen understanding, challenge assumptions, and uncover hidden beliefs. Use for requirements gathering, debugging, coaching, and critical analysis. | `references/thinking-socratic/SKILL.md` |
| `thinking-thought-experiment` | Test ideas through hypothetical scenarios when empirical testing is impractical. Use for architecture evaluation, edge case analysis, ethics considerations, and strategy development. | `references/thinking-thought-experiment/SKILL.md` |
| `thinking-first-principles` | Break complex problems into fundamental truths by questioning assumptions and rebuilding from irreducible components. Use for innovation, challenging status quo, or when conventional solutions fail. | `references/thinking-first-principles/SKILL.md` |
| `thinking-steel-manning` | Argue against the strongest version of opposing positions, not the weakest. Use for design reviews, evaluating alternatives, conflict resolution, and decision validation. | `references/thinking-steel-manning/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
