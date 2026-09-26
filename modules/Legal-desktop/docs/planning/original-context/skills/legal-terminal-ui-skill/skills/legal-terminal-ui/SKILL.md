---
name: legal-terminal-ui
description: Guides UI/UX design for legal-terminal, a Bloomberg Terminal-style dark-mode legal case management platform. Use when designing layouts, dashboards, panels, color systems, data tables, charts, AI chat panels, database explorer views, or ingestion status views — runs a guided interview, live research, and produces a design handoff document per module.
license: Proprietary
---

# Legal Terminal UI/UX

## Overview

legal-terminal follows a "concealed complexity" design philosophy borrowed from the Bloomberg Terminal: dense, information-rich views for power users, with complexity managed by the system rather than exposed as clutter. Dark-mode-first, keyboard-navigable, built around modular dockable panels — including AI chat/agent panels and database explorer panels.

## Step 1: Interview + Live Research (mandatory for new modules)

Run `scripts/design_interview.py` to collect structured answers and generate a `research_brief`. Skip only if `assets/design-decisions.json` already has a `final_recommendation` for this exact module.

After the script runs:
1. Read `research_brief.suggested_queries`.
2. Use `search_web` to research CURRENT component libraries, frameworks, and color/theme directions matching the stated tech stack and aesthetic. Treat `references/component-libraries.md` and `references/color-schemes.md` as starting hypotheses only.
3. Synthesize 3-4 live options with concrete tradeoffs (maintenance, styling fit, licensing, accessibility).
4. Present options to the user; let them pick or request more research.
5. Write the chosen option into `final_recommendation` in `assets/design-decisions.json`.

## Step 2: Generate the Design Handoff (mandatory once final_recommendation is set)

As soon as `final_recommendation` is written for a module, run `scripts/generate_handoff.py --module <module_name>`. This reads the interview answers + final recommendation and produces `handoffs/<module_name>-handoff.md` — a standalone, developer-ready spec containing:
- Module purpose and target user role
- Chosen component library + rationale
- Chosen color scheme rendered as concrete design tokens (hex values, elevation scale, status colors)
- Layout pattern to apply
- Data-viz/AI-chat/database requirements flagged during the interview
- A pre-ship checklist scoped to this module

Do not skip this step — the interview and research are only useful once compiled into a handoff artifact a developer (or Claude, in a later session) can build directly from without re-reading the whole interview transcript.

## Step 3: Apply Core Principles When Building

1. Conceal complexity — dense defaults, advanced options one interaction away, never deleted.
2. Progressive disclosure — layer novice guidance on top of expert density on the same screen.
3. Dark-first token system — see `references/dark-mode-tokens.md`.
4. Status must always be visible and actionable.
5. Every visualization matches its data type; every table/chart is exportable as CSV/JSON.
6. Legal relationships (case ↔ document ↔ party) are visualized, not just listed.
7. AI interactions and database views follow the same provenance and status-visibility rules as the rest of the platform.

## Step 4: Build

1. Open `handoffs/<module_name>-handoff.md` — this is the single source of truth for the module, not the raw JSON.
2. Apply the layout pattern from `references/layout-patterns.md`.
3. Apply dark-mode token rules from `references/dark-mode-tokens.md`.
4. If flagged: apply `references/data-viz-ingestion.md`, `references/legal-domain-ux.md`, `references/ai-chat-interaction.md`, and/or `references/database-productivity-ux.md`.
5. Run `scripts/contrast_check.py` on any new color pairs.
6. Review against the checklist embedded in the handoff doc.

## Resources

- `scripts/design_interview.py` — interactive interview + generates a live-research brief, writes `assets/design-decisions.json`
- `scripts/generate_handoff.py` — compiles a module's final_recommendation into `handoffs/<module>-handoff.md`
- `scripts/contrast_check.py` — WCAG 2.2 contrast ratio checker for color pairs
- `references/component-libraries.md` — starting-point library options (verify live via search_web)
- `references/color-schemes.md` — starting-point dark palette directions (verify live via search_web)
- `references/dark-mode-tokens.md` — elevation, contrast, token naming rules
- `references/layout-patterns.md` — panel/workspace architecture, command bar, saved views
- `references/data-viz-ingestion.md` — chart selection, pipeline/status monitoring UX
- `references/legal-domain-ux.md` — legal-specific workflow and accessibility requirements
- `references/ai-chat-interaction.md` — AI chat, agent transparency, trust/control UX
- `references/database-productivity-ux.md` — database viewing, multi-database integration, productivity workspace UX
- `references/checklist.md` — general pre-ship review checklist (module-specific version is embedded per handoff)
- `assets/design-decisions.json` — persisted interview answers + live-research recommendations
- `handoffs/` — one compiled design handoff doc per module (the actual deliverable developers build from)

Load only the reference file(s) relevant to the current task rather than reading all of them.
