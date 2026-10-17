# Portable Skill Discovery & Routing Metadata — Adoption Reference

## Purpose

This file defines the metadata convention used by `think` so a lightweight scout/router can discover skills cheaply without forcing Claude Code, Codex, or another Agent Skills host to understand custom routing fields. It is intended as the reference for future plugin adoption.

## Compatibility model

Use two layers:

1. **Native `SKILL.md` frontmatter** for host discovery. Keep `name` and `description` authoritative. Optional `metadata` may contain a few **string-valued hints** only. Do not make correct execution depend on a host understanding custom metadata.
2. **External `skill-registry.json`** for rich discovery, routing, aliases, semantic groups, depth modes, relationships, and manual-invocation hints. Hosts may ignore this file completely without breaking the plugin.

`skill-routing.schema.json` validates the registry shape.

## Recommended frontmatter hints

```yaml
---
name: systems
description: Systems, feedback, constraints, dependencies, and leverage. Route within this family when interactions or blast radius matter.
metadata:
  short-description: Systems, feedback, constraints, dependencies, and leverage
  think-kind: family
  think-public: "true"
  think-family: systems
  think-tags: systems,feedback,constraints,dependencies,leverage
  think-registry: ../../skill-registry.json
---
```

Keep metadata values strings for portability. Rich arrays/objects belong in the registry.

## Registry fields

- `id` — globally stable plugin-scoped identifier, e.g. `think:systems:thinking-feedback-loops`.
- `name` — skill/model name.
- `kind` — router, family, workflow-family, workflow, model, core, or other.
- `family` — semantic parent when applicable.
- `path` — package-relative source path.
- `description` — concise capability/use-case description.
- `tags` — semantic discovery terms; not a replacement for description matching.
- `aliases` — exact/manual names and common human variants.
- `depths` / `default_depth` — progressive execution modes.
- `reflex_roles` — optional roles a model can serve in persistent Reflex.
- `manual_invocation` — host-specific examples only; semantic routing must not depend on this syntax.

## Scout behavior

For a new task:

1. Honor an exact user-named skill/model first.
2. Search/rank registry entries by task meaning, descriptions, aliases, tags, and family. Keyword overlap alone is only a shortlist signal.
3. Prefer one clear match. Recommend a combination only when each additional model answers a distinct missing question.
4. If confidence is low, read only the most relevant family/router entrypoint; do not bulk-load every model.
5. If nothing fits, route to the fallback decision matrix or ordinary reasoning and identify a library gap.
6. Return recommendation + rationale + suggested depth. Do not silently execute an unrelated model merely because it ranked highest.

## Future plugin adoption procedure

When adding another plugin to the scout index:

1. Inventory its public skills and supporting/nested resources.
2. Preserve the plugin's native frontmatter; add only portable string-valued metadata hints when useful.
3. Add registry entries with stable plugin-scoped IDs.
4. Add semantic tags/aliases based on what the skill actually does and when it applies.
5. Mark whether an entry is public/discoverable or a nested resource through `kind` and path; do not expose a resource merely to index it.
6. Add depth/reflex metadata only when the skill really supports those modes.
7. Validate paths and uniqueness against the schema and package validator.
8. Test at least: exact manual selection, ambiguous task, no-fit task, multi-model task, and low-risk task that should remain micro.

## Governance: rules are user-authorized

The scout/router may discover recurring patterns and **suggest** a candidate persistent rule, but it does not have authority to create one. A user correction must not be generalized into a global rule automatically. Only the user may approve persistence and scope. This rule-authority boundary was explicitly established by the owner of this plugin.

For candidate rules, report:

- proposed wording;
- demonstrated problem it addresses;
- proposed scope;
- likely side effects/overreach risk;
- whether a temporary local adaptation already solves the immediate issue.

Do not write the candidate into persistent policy until the user approves it and its scope.

## Versioning

Increment `schema_version` only for registry-shape compatibility changes. Plugin versions may change independently. Consumers should ignore unknown registry fields so future additive metadata does not break older scouts.

## Host manifests and exposure

For a dual Claude Code + Codex plugin, keep the canonical skill content shared and use thin host manifests:

- Claude Code: `.claude-plugin/plugin.json`
- Codex: `.codex-plugin/plugin.json` with the `skills` root pointing at `./skills/`

Treat `skills/<public-entry>/SKILL.md` as the public/discoverable layer. Put progressively disclosed implementation models beneath the owning public skill's `references/` tree. Do not duplicate the same reasoning process into host-specific command files.

## Correction and rule-governance metadata

Discovery metadata may describe a skill's purpose, tags, aliases, family, depth, relationships, and Reflex roles. It must **not** silently convert observations or user corrections into durable behavioral policy. A scout/router may propose a candidate rule, including rationale and proposed scope, but persistent rule adoption requires explicit user authorization.

Local correction is not global metadata. Do not add a new registry-wide prohibition merely because one task required a local correction.
