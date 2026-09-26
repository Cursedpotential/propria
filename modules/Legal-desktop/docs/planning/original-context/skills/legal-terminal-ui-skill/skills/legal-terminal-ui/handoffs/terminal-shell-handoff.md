# Design Handoff: terminal-shell

_Generated 2026-08-18T16:09:48.192387+00:00 from interview + live research._

## Module Brief

- **Purpose:** Bloomberg-style matter command center chrome: sidebar, › command line, command palette, status bar, and split workspace wrapping Matter Home and mnemonic routes. Chat is a module at /chat, not the landing page. One high-conflict Michigan custody matter. Primary operator is a self-represented litigant (not a lawyer). Agno is evidence truth; this app never clones evidence.

- **Primary user role:** Self-represented litigant (Michigan family court / Genesee 7th Circuit). Not a lawyer. Must remain usable at expert density.

- **Density:** Maximal density (Bloomberg-style)

- **Tech stack:** Next.js 16.3 App Router + React 19.2 + TypeScript. No Vite. cmdk already installed. Tailwind 4 + shadcn cited in CAT2 but not yet installed. FastAPI legal-api sibling on :8010.

- **Aesthetic direction:** Bloomberg terminal / legal-terminal-master dark graphite, concealed complexity, IBM Plex Mono + Inter + Playfair Display (self-hosted OFL). Progressive disclosure: keep the density, put a plain-English layer on top.

- **Data/viz/interaction needs:** Combination: dense tables + AI chat/agent panel + database/source explorer + matter-document-party relationships + status/ingestion chrome. Keyboard-first command bar.

- **Must-haves/constraints:** WCAG 2.2 AA; keyboard-first; self-hosted fonts only (no Google CDN); MIT/Apache/OFL licenses; keep existing CSS tokens in globals.css; Confidential Mode persist + no consumer-model fallback; CHAT is /chat not landing; never clone Agno evidence; no Vite; no TUI; export tables as CSV/JSON later; cmdk already in tree; every mnemonic has a plain-English name + one-line Michigan explanation; do not Barney-down — treat the user as a capable adult who is not a lawyer.

## Final Recommendation (from live research)

```json
{
  "chosen_at": "2026-08-18T16:20:00+00:00",
  "byline": "Grok \u00b7 grok-4.6 \u00b7 2026-08-18",
  "library": {
    "name": "Existing graphite CSS tokens + cmdk; adopt shadcn/ui + Tailwind 4 incrementally",
    "why": "CAT2 already locked Next 16 + shadcn + Tailwind 4. The running app already has legal-terminal-master tokens, cmdk, and a working shell. A big-bang Tailwind preflight rewrite would trash that. termcn is an Ink TUI (CAT2: no TUI). Blueprint v6 is React-19-ready but Sass/CSS-in-JS and fights the existing token file. Fortress is a paid finance template, not a library.",
    "rejected": [
      {
        "name": "termcn (shadcn-labs/termcn)",
        "why": "Ink + OpenTUI. CLI/TUI only. CAT2 forbids a TUI for this app."
      },
      {
        "name": "Palantir Blueprint v6",
        "why": "Excellent dense tables (Apache-2.0, React 19 as of v6.16, 2026-06-09). Wrong styling engine for a Tailwind-bound Next app that already has graphite tokens."
      },
      {
        "name": "Fortress / glitchcn / cyberpunk kits",
        "why": "Paid template or neon scanlines. Wrong license posture and wrong legal aesthetic."
      },
      {
        "name": "Classic amber-on-black Bloomberg clone",
        "why": "Nostalgic, worse for long legal reading. Confidential gold already occupies amber. Keep graphite."
      }
    ],
    "install_later": "When the next primitive is needed (Tooltip, Dialog, DataTable): npm i tailwindcss @tailwindcss/postcss; import utilities only (no preflight); npx shadcn@latest add tooltip dialog. Map shadcn CSS vars onto the existing --bg/--text/--accent tokens."
  },
  "color_scheme": {
    "name": "Legal-terminal graphite (already in globals.css)",
    "direction": "Keep current tokens. Do not switch to slate/navy SaaS or amber-on-black.",
    "tokens": {
      "bg": "#0e1114",
      "bg-panel": "#141820",
      "bg-panel2": "#1a202a",
      "bg-hover": "#222938",
      "bg-selected": "#223040",
      "bg-inset": "#11151c",
      "surface-raised": "#171c25",
      "text": "#d4d8df",
      "text-dim": "#8a9099",
      "text-muted": "#4d5563",
      "text-heading": "#e8eaf0",
      "accent": "#6f9cbd",
      "accent-dim": "#466f90",
      "risk-critical": "#c25b5b",
      "risk-high": "#c2854f",
      "risk-medium": "#b3a44f",
      "risk-low": "#6fa370",
      "confidential": "#c8a03c"
    },
    "rules": [
      "Never use color alone for status \u2014 pair with a word.",
      "Amber/gold is Confidential Mode, not 'warning' unless labeled.",
      "No pure #000 background, no pure #fff text.",
      "WCAG 2.2 AA: body text 4.5:1."
    ]
  },
  "layout": {
    "pattern": "Persistent left sidebar + \u203a command line + module header + split workspace + status bar",
    "landing": "/",
    "chat_route": "/chat",
    "keyboard": "\u203a line, Tab complete, Ctrl/Cmd+K palette, Ctrl+\\ split, Esc closes split"
  },
  "srl_layer": {
    "audience": "Michigan self-represented litigant, Genesee County / 7th Circuit, high-conflict custody. Capable adult, not a lawyer. Do not Barney-down.",
    "rule": "Same screen stays dense. Every control also has a plain-English name and a one-line 'what this is' explanation. Legal terms stay; they get a gloss, they are not deleted.",
    "group_renames": {
      "Matter": "Your case",
      "Research": "Law & cases",
      "Drafting": "Papers",
      "Operations": "Evidence & calendar",
      "Private": "Advanced"
    },
    "advanced_disclosure": "STRAT / TEAM / AGNT / PRIV start folded under Advanced. One click opens them. Never deleted.",
    "not_court_safe": "Always visible as: 'Workbench only \u2014 do not file this screen with the court.'",
    "examples": {
      "FCTR": "Best-interest factors \u2014 the 12 things a Michigan judge must weigh (MCL 722.23).",
      "FOC": "Friend of the Court \u2014 the county office that investigates custody and support.",
      "RELS": "Release candidate \u2014 the version you review before anything is treated as ready to file.",
      "PRIV": "Privilege first-pass \u2014 flags attorney-client / work-product text. Not a legal conclusion.",
      "CONF": "Confidential mode \u2014 blocks consumer AI (ChatGPT/Claude). Stays on until you turn it off."
    }
  }
}
```

## Core Principles (apply to every screen)

- Conceal complexity: dense defaults, advanced options one interaction away, never deleted.
- Progressive disclosure: layer novice guidance on top of expert density on the same screen.
- Status must always be visible and actionable.
- Every visualization matches its data type; every table/chart is exportable as CSV/JSON.
- Provenance (source, timestamp, version) shown wherever data is displayed.

## Guidance: Dark Mode Tokens

Design dark mode as its own deliberate system, not an inverted light theme.

## Elevation Scale
Define 4-5 grey/navy steps from deepest background to highest surface, each slightly lighter than the one below (no drop shadows for elevation): bg-base, bg-surface, bg-elevated, bg-overlay.
Never use pure black (#000) for bg-base or pure white for text.

## Accent Colors
Desaturate light-mode accent/brand colors by roughly 20 points before using in dark mode.

## Status Color Mapping
Red: alerts/overdue. Amber: warnings. Green: healthy/complete. Blue: neutral info.
Status color must always be paired with an icon or text label — never rely on color alone.

## Contrast Requirements (WCAG 2.2)
Normal text: minimum 4.5:1. Large text/headings: minimum 3:1. Verify with scripts/contrast_check.py.

## Token Implementation
Build as design tokens (CSS custom properties), named semantically (--color-status-danger, --bg-surface-1).

## Guidance: Layout Patterns

## Multi-Panel Workspace
Modular, dockable panels (case list, document viewer, research pane, command input) mirroring Bloomberg's multi-window function panes.

## Saved Workspace Views
Support saved layouts per role, savable/restorable instantly.

## Persistent Command Bar
Keyboard-accessible command/search bar for jumping between functions without leaving the keyboard.

## Spatial Grouping
Group related workflows spatially (research → drafting → filing).

## Tabs and Accordions
Use only for stable, bounded sections. Never hide frequently used actions in a dropdown.

## Information Hierarchy
Surface 4-6 top-level KPIs in the primary zone; push secondary metrics into drill-downs.

## Component Consistency
One set of card/table/chart/filter/badge components reused identically everywhere; 8px spacing grid.

## Guidance: Data Viz Ingestion

Bar: categorical. Line: trends. Tables: precise/dense values. Scatter: relationships.
Strip decorative chrome — every visual element should carry information.
Tables: monospace numerals, right-aligned numbers, frozen headers, zebra striping.
Ingestion pipelines need a dedicated monitoring surface (source → transform → load) with drill-down.
Status (queued/processing/indexed/failed) must be persistently visible without navigating away.
Pair every anomaly with a direct action (retry/investigate/resolve).
Every chart/table must support CSV/JSON export.
Health should be readable in under 3 seconds via position/size/color alone.

## Guidance: Legal Domain Ux

Auto-fill repetitive case/client data, reusable filing templates, plain jargon-light labels.
Visualize case-to-document, party-to-filing, citation networks as graphs/timelines, not flat lists.
Inline validation, autosave indicators, contextual tooltips for high-stakes filing/deadline flows.
Accessibility mandatory: keyboard nav, screen-reader labels, resizable text, no disruptive animation.
Show data provenance (source, timestamp, version) for every case fact/document reference.
Provide a simplified touch-friendly fallback view even in a desktop-first product.
Usability-test with real attorneys/paralegals early and often.

## Guidance: Ai Chat Interaction

Treat AI chat as a dockable panel, not a modal — coexists with case list/document viewer.
Persistent collapsible chat rail (right-dock default); expandable to full panel for deep sessions.
Scope chat context explicitly with a visible context chip (this case / this doc / this query).
Distinguish AI-generated content visually from human/source content at all times.
Stream responses; show clear generating/done states.
Surface citations/sources inline within AI answers, linking back to source docs.
Show a collapsible "steps taken" trace for tool-use/agent actions — concealed by default, expandable.
Show real-time status for long-running agent tasks; give an explicit stop/interrupt control.
Require explicit user confirmation before any AI action modifies a case record or filing.
Maintain a visible, filterable AI action history/log per case for audit.
Let the persistent command bar double as an AI command entry point (slash-style quick actions).

## Guidance: Database Productivity Ux

Dedicated data-explorer panel: schema/table tree + dense results grid, dockable like any other panel.
Support both a visual query builder and a raw query console (SQL/SurrealQL), togglable without losing state.
Results grids follow the same dense-table conventions: monospace numerals, frozen headers, CSV/JSON export.
Vector/semantic search results show similarity scores and let users drill back to the source document.
Show query execution metadata (row count, latency, source database) in a persistent status strip.
Label per-field/per-panel data source when blending multiple backends.
Provide a connections panel with live per-database health (connected/degraded/offline) + last-sync time.
Let users save cross-database queries as first-class, shareable workspace objects.
Paginate/virtualize large result sets; debounce and visibly indicate searching states.

## Pre-Ship Checklist for This Module

- [ ] Was the design interview run and final_recommendation set before this screen was built?
- [ ] Does the layout conceal complexity behind sensible defaults?
- [ ] Are panels modular, dockable, and savable as named workspace views?
- [ ] Is the color system built from dark-first tokens with proper elevation and desaturated accents?
- [ ] Have all color pairs passed scripts/contrast_check.py (WCAG 2.2)?
- [ ] Is status color always paired with an icon or label?
- [ ] Does every chart match its data type, with tables as first-class dense components?
- [ ] Is ingestion/processing status persistently visible with drill-down and retry actions?
- [ ] Can users export any visualized/tabular data as CSV/JSON?
- [ ] Are legal relationships visualized, not just listed?
- [ ] Is keyboard-first navigation supported for core terminal functions?
- [ ] Does data provenance show for case facts and documents?
- [ ] If AI chat/agent features are present, is there a confirmation step before any write action?
- [ ] If database views are present, is per-source provenance labeled?
