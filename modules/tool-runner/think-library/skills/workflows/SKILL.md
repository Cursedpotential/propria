---
name: workflows
description: 'Run one of six integrated repeatable workflows: evaluate, decide, plan, debug, verify-claim, or debias. Each composes thinking models into one process and returns one workflow deliverable.'
metadata:
  short-description: Integrated repeatable thinking workflows
  think-kind: workflow-family
  think-public: 'true'
  think-tags: workflow,evaluate,decide,plan,debug,verify,debias
  think-registry: ../../skill-registry.json
---
# think · workflows

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Six repeatable **integrated processes**. These are not collections of independent framework reports. Each workflow uses several model operations as stages in one reasoning path and produces the workflow's single final deliverable.

## Persistent control

Keep `../thinking-model-router/references/reflex/SKILL.md` active during execution. Reflex handles sequentiality, downstream effects, scope, verification, and adaptive escalation without adding a separate report.

## Depth behavior inside workflows

Each referenced model contains Micro / Reflex, Focused, and Full branches. A workflow step should use the **smallest depth sufficient for that step**:

- **Micro** by default for supporting lenses.
- **Focused** when the step reveals material uncertainty or a consequential issue.
- **Full** only when the full model is needed or explicitly requested.

Even when a model escalates to Full, its result is fed back into the workflow. Do not emit its standalone deliverable unless the user asked for it or the workflow explicitly requires that artifact.

## Workflow selection

| workflow | use when | file |
|---|---|---|
| `evaluate` | review, completion claim, readiness, quality judgment | `references/evaluate.md` |
| `decide` | choose between options or make a consequential choice | `references/decide.md` |
| `plan` | sequence work, roadmap, kickoff, implementation plan | `references/plan.md` |
| `debug` | bug, regression, unexplained failure, root cause | `references/debug.md` |
| `verify-claim` | fact/citation/number needs evidence status | `references/verify-claim.md` |
| `debias` | fairness, interpretation, motive/character judgment | `references/debias.md` |

If no workflow matches the whole task, use `../thinking-model-router/SKILL.md` instead of forcing one.

## Integrated-workflow invariant

A workflow stage consumes the state produced by the prior stage and changes what happens next. Model names are implementation lenses, not separate assignments. The final response should be the workflow's own output unless the user requests otherwise.

## Paths

All cross-family model paths written inside workflow reference files are resolved relative to this `skills/workflows/` skill directory.
