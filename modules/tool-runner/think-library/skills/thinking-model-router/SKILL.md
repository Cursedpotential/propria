---
name: thinking-model-router
description: Route a task to the smallest useful combination of think families/models, choose micro/focused/full depth, and orchestrate them as one process. Use when the right thinking approach is unclear or mixed.
metadata:
  short-description: Route and orchestrate thinking models
  think-kind: router
  think-public: 'true'
  think-tags: thinking,routing,orchestration,model-selection,combination,skill-discovery
  think-registry: ../../skill-registry.json
---
# Think Model Router / Orchestrator

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Use this as the default entry point when the appropriate thinking family/model is unclear, when multiple lenses may be needed, or when the user names an underlying model directly. It owns **selection + combination + depth orchestration**; those are not separate public skills.

## First: activate Reflex

Read `references/reflex/SKILL.md` and keep its execution-control loop active while this task is being worked. Reflex normally does not create a separate deliverable.

## Manual exact-model invocation

If the user names an underlying model or exact registry ID, honor that selection unless it is unavailable or obviously incapable of answering the request. The user may also specify `micro`, `focused`, or `full`. If depth is omitted, start at micro and let the model's branch logic escalate.

Examples of equivalent intent:

- Claude: `/think:thinking-model-router thinking-second-order focused`
- Codex: `$thinking-model-router thinking-second-order focused`
- Natural language: “Use second-order thinking on this; go full if the quick pass finds something material.”

The model itself lives under its family `references/<model>/SKILL.md`; manual selection does not require exposing it as another top-level skill.

## Routing process

1. State the actual question or decision blocking progress.
2. Check whether one of the six fixed workflows (`workflows`) already matches the whole task. If yes, route there instead of rebuilding it ad hoc.
3. Otherwise choose the primary family: `debias`, `forecast`, `reason`, `diagnose`, `systems`, `incentives`, `decide`, or `strategy`.
4. Use `../../skill-registry.json` for cheap discovery. Rank by task meaning, description, tags, aliases, and family—not keyword overlap alone.
5. Select the **smallest sufficient model set**. One model is preferred when it covers the problem. Add a supporting model only when it answers a specific missing question or blind spot.
6. Assign each selected model a role and starting depth before loading it.
7. Compose the selected operations into **one coherent reasoning path** ordered by the task. Do not run independent reports and concatenate them.
8. During execution, Reflex may call abbreviated supporting models or escalate a selected model when evidence warrants it.
9. Return one integrated result unless the user explicitly asks to see separate framework outputs.

## Combination design

A combination is a dependency-aware process, not “model soup.” Before loading multiple models, define:

- **Primary question** — what the combination must resolve.
- **Role of each model** — what unique question it answers.
- **Order** — why this model's output becomes the next model's input.
- **Depth** — micro/focused/full independently for each model.
- **Stop condition** — when enough reasoning has been done.

Valid patterns include sequential, nested, adversarial, and iterative combinations. Parallel independent analyses are appropriate only when genuine independence is useful; synthesize them before acting.

### Example

For an unfamiliar architectural change:

`reason/scientific-method (focused: establish behavior)` → `systems (focused: dependency/blast-radius map)` → `decide/reversibility (micro: commitment level)` → `incentives/second-order (micro: downstream consequences)` → implementation/verification under Reflex.

That is one process. Do not emit four separate framework reports.

## Family map

| Family | Primary question |
|---|---|
| `reason` | What is true, what follows, and what assumptions need testing? |
| `diagnose` | Why did this happen / what is the root cause? |
| `systems` | How do interacting components, feedback, constraints, or dependencies behave? |
| `forecast` | What is likely, how uncertain are we, and what is the plausible range? |
| `incentives` | How will actors/tradeoffs/downstream consequences respond? |
| `decide` | What should we do given stakes, reversibility, alternatives, and risk? |
| `strategy` | What coherent direction and coordinated choices achieve the objective? |
| `debias` | What distortion, framing, or cognitive bias could be driving the read? |

## When nothing fits cleanly

Read `references/decision-matrix.md`. Do not force the task into a category merely because a skill exists. If the library lacks a needed capability, use ordinary reasoning and identify the gap as a potential future skill. A gap does not authorize inventing a persistent rule.

## Re-routing

Re-route when new evidence changes the problem type, the selected model yields no useful discrimination, or Reflex reflection identifies a missed dimension. Preserve useful work already completed; switching models is not permission to restart the whole task from zero.

## Discovery hygiene

Do not load all 84 model files. Use the registry, family entrypoint, or exact model name first, then read only the selected model files.
