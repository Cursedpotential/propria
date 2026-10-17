---
name: strategy
description: Build coherent strategy across objectives, tradeoffs, competition, innovation, product direction, and execution. Use directly or route to a strategy model at micro, focused, or full depth.
metadata:
  short-description: Strategy, competition, innovation, product direction, and coherent choices
  think-kind: family
  think-public: 'true'
  think-family: strategy
  think-tags: strategy,competition,innovation,product,market,execution,tradeoffs
  think-registry: ../../skill-registry.json
---
# think · strategy

## Invocation input

Host-provided invocation arguments: `$ARGUMENTS`

If that placeholder is unexpanded, ignore it. Otherwise treat the arguments as direct user intent. Parse an exact model/family/workflow name when present and an optional `micro`, `focused`, or `full` depth. Never silently replace an explicitly named model with another one; route or explain the mismatch instead.


Strategy, competition, innovation, product direction, and coherent choices. This is a **public family entry skill**. The underlying models are progressively loaded from `references/` and are not separately exposed as top-level plugin skills.

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
| `good-strategy-bad-strategy` | Craft coherent strategy through diagnosis of core challenge, guiding policy, and coordinated actions when facing complex problems requiring more than goals and aspirations | `references/good-strategy-bad-strategy/SKILL.md` |
| `playing-to-win` | Five-question strategy cascade framework for making integrated strategic choices from winning aspiration to management systems, from P&G's transformation | `references/playing-to-win/SKILL.md` |
| `porters-five-forces` | Systematically assess the competitive intensity and attractiveness of an industry by analyzing five fundamental forces that shape competition and profitability | `references/porters-five-forces/SKILL.md` |
| `three-horizons-framework` | Balance current business performance with emerging opportunities and future innovation by allocating resources across three time horizons of growth | `references/three-horizons-framework/SKILL.md` |
| `pest-analysis` | Generate PEST analysis plans with strategic questions and Perplexity AI prompts. Use when user mentions PEST analysis, strategic planning, market analysis, external factors, environmental scanning, or needs research questions for political economic social technological factors. | `references/pest-analysis/SKILL.md` |
| `pest-analysis-execute` | Execute PEST analysis by processing research results from Perplexity AI prompts. Create synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions PEST results, analysis execution, research synthesis, or after generating PEST analysis plans. | `references/pest-analysis-execute/SKILL.md` |
| `pest-analysis-synthesis` | Create PEST synthesis tables with strategic implications, impact and likelihood ratings. Use when user mentions synthesis table, strategic recommendations, opportunity threat analysis, or after completing PEST research. | `references/pest-analysis-synthesis/SKILL.md` |
| `pest-analysis-critical` | Critically evaluate PEST synthesis tables for biases, gaps, and oversights. Use when user mentions critical review, bias assessment, gap analysis, or after completing PEST synthesis. | `references/pest-analysis-critical/SKILL.md` |
| `pest-analysis-strategy` | Transform PEST analysis into actionable strategy: export to CSV, save to Obsidian, create SWOT, define strategic actions, prioritize, and build work plan. Use after completing PEST critical assessment. | `references/pest-analysis-strategy/SKILL.md` |
| `thinking-effectuation` | Start with means, not goals; co-create with partners; leverage contingencies. Use for startup strategy, innovation projects, and uncertain/novel domains where planning is unreliable. | `references/thinking-effectuation/SKILL.md` |
| `thinking-jobs-to-be-done` | Understand what 'job' users hire your product to do, focusing on progress users seek rather than features. Use for product development, feature prioritization, user research, and market positioning. | `references/thinking-jobs-to-be-done/SKILL.md` |
| `thinking-triz` | Apply TRIZ (Theory of Inventive Problem Solving) methodology to resolve technical contradictions and find innovative solutions. Use for engineering design, breaking through impossible constraints, and systematic innovation. | `references/thinking-triz/SKILL.md` |

## Manual selection

Manual invocation remains available without exposing every model globally. Name the exact model after the family entry skill, for example `thinking-systems focused`. If the requested model is outside this family, route to the router rather than silently substituting.

## Escalation

If a micro pass exposes a material unresolved issue, use that same model's Focused branch. If focused analysis remains insufficient or the user asks for the formal process/deliverable, use Full.
