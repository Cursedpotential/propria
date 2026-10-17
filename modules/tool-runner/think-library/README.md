# think

Dual-compatible Claude Code + Codex thinking package.

Version **2.0.1** restores direct Claude slash-command access to every underlying thinking model while preserving the 10-skill category/router surface and progressive disclosure.

## Public surface: 10 skills

- `debias`
- `forecast`
- `reason`
- `diagnose`
- `systems`
- `incentives`
- `decide`
- `strategy`
- `workflows`
- `thinking-model-router`

The 84 individual thinking models are **not** top-level `skills/` directories. They remain under the appropriate family `references/<model>/SKILL.md` for progressive disclosure. Each contains three execution branches: **Micro / Reflex**, **Focused**, and **Full**.

For Claude Code, every underlying model also has a **manual-only slash command** in `commands/`. This restores exact direct invocation without flattening the skill/category layout.

## Dual-host manifests

This package intentionally carries **both** host manifests:

- `.claude-plugin/plugin.json` — Claude Code plugin metadata.
- `.codex-plugin/plugin.json` — Codex plugin metadata and `skills` root.

Only direct children of `skills/` are intended as the public surface. Underlying models remain under each family's `references/` tree and are loaded through progressive disclosure. Do not relocate them back to `skills/<model>/SKILL.md` unless you intentionally want them public.

## Direct model commands

Claude Code now exposes one manual-only slash command for every underlying model. Examples:

- `/think:thinking-first-principles focused [task]`
- `/think:thinking-second-order micro [task]`
- `/think:thinking-systems full [task]`
- `/think:pre-mortem` is **not** invented as an alias; use the model's real command name, e.g. `/think:thinking-pre-mortem`.

The one name collision is handled explicitly: `/think:incentives` remains the **family** entry skill, while the underlying model named `incentives` is `/think:incentives-model`.

Each direct command loads:

1. `skills/thinking-model-router/references/reflex/SKILL.md` as the persistent execution-control loop; and
2. the exact selected model resource.

Depth syntax is `[micro|focused|full] [task]`. If depth is omitted, the model begins with its **Micro / Reflex** branch and may escalate according to its own criteria. If task text is omitted, the command applies the model to the current task/conversation context. Direct commands are marked `disable-model-invocation: true`, so they exist for explicit user control rather than becoming another automatic routing surface.

The original family/router form still works and remains useful when you want routing rather than an exact model:

- Claude: `/think:systems thinking-feedback-loops focused`
- Claude: `/think:thinking-model-router thinking-second-order full`
- Codex: `$systems thinking-feedback-loops focused`
- Codex: `$thinking-model-router thinking-second-order full`

This keeps the category structure while restoring exact one-command access to all 84 models.

## Router + Reflex

`thinking-model-router` owns model selection, combination, fallback routing, and depth orchestration. Combination is not a separate public skill.

Its `references/reflex/SKILL.md` is the persistent execution-control layer: observe/orient/sequence, second-order/system effects, scope, lightweight pre-mortem, act, verify, reflect, update, and selective escalation. Every direct Claude model command explicitly loads this Reflex resource before the selected model, so direct invocation does not bypass the persistent loop.

## Discovery metadata

- `skill-registry.json` — rich machine-readable discovery index
- `skill-routing.schema.json` — reusable JSON Schema
- `ROUTING-METADATA-ADOPTION.md` — instructions for adopting the convention in other plugins/scouts
- `DIRECT-COMMANDS.md` — all 84 direct Claude model commands grouped by family

Host correctness never depends on custom metadata; `name` and `description` remain the native discovery contract.

## Validate

```bash
python scripts/validate_package.py
python skills/thinking-model-router/scripts/pick.py "task description"
```
