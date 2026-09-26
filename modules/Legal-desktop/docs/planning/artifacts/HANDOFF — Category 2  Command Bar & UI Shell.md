# HANDOFF — Legal OS Category 2: Command Bar & UI Shell — Implementation Research

Status: rough feature set agreed with owner (2026-08-17); this handoff requests in-depth research to scaffold the actual implementation. Research only — no code, no scaffolding yet.

## Context

Legal OS is a single-user, single-case, self-represented-litigant practice-management app, Bloomberg-terminal-style keyboard-first UI, sibling package to the existing "Agno MCP Platform" (Next.js/FastAPI Workbench) in one monorepo. This category covers the app shell: navigation, command input, layout, and visual design language — not any individual research/contract/tool panel (those are separate categories).

## Feature set agreed (do not re-litigate — build research around this)

**In scope for v1:**
- Collapsible sidebar with grouped navigation (icon-only collapsed state, mobile overlay mode)
- Mnemonic command bar: typed shortcuts (e.g. `PREC breach of contract`) with live autocomplete/suggestion dropdown and Tab-complete — this is the primary navigation input, not a secondary affordance
- **Split view — high priority.** Pin a second panel beside the primary one from any panel's header, close with an ×. Owner explicitly called this "super handy" — treat as a core v1 feature, not a stretch goal.
- Status pill in the shell chrome showing real connection status to Agno's data sources (Postgres `legal_os` schema, SurrealDB analytical surface, Agno's `/v1/*` REST API) — not a mock/live toggle, a genuine health indicator
- Confidential Mode toggle in the sidebar footer (visual posture indicator — the actual inference-routing behavior it controls is a separate category)
- Export-this-panel action (export the current panel's content to PDF or image) — lower priority than the above, but in scope
- Case-phase switcher — a one-touch control that switches which shortcuts/quick-actions are surfaced based on case phase (Discovery / Motions / Hearing Prep / Trial) — lower priority than the above, but in scope
- Dark-mode-only visual design: graphite/slate palette, Playfair Display (headings/case names), Inter (UI chrome/body text), IBM Plex Mono (data cells/citations/status bar), desaturated CRITICAL/HIGH/MEDIUM/LOW risk-severity badge system. **No light/dark toggle — dark is the only theme.**
- **Ctrl+K command palette, reinstated (2026-08-17).** Owner reconsidered once told this is low-effort via `cmdk` (small, well-maintained React library, same one legal-terminal itself uses) — back in scope for v1. Overlay listing all modules/mnemonics/descriptions plus a "Recent" group of the last 5 commands, Escape to close, selecting an item navigates and logs the command (same behavior legal-terminal implements).

**Explicitly cut — do not build, do not research:**
- Persistent hardware-style Help/Menu/Search/Cancel action-key affordances
- "Repeat last command" shortcut

**Deferred, not v1:**
- TUI (terminal UI) companion — explicitly deferred to v2. Do not scaffold or research TUI frameworks (e.g. Textual) as part of this pass.

**Open and explicitly why this handoff exists:**
- Frontend framework: **React + Vite + Tailwind + Zustand** (matching legal-terminal's shape) vs. **Next.js** (matching Agno's own Workbench, for monorepo build-tool/component-sharing consistency). Owner has not decided and wants real tradeoffs before deciding — this is the single most important output of this research pass.

## Research questions to answer (with citations/links)

1. **Framework decision (primary deliverable of this research pass).** Compare React+Vite+Tailwind+Zustand vs. Next.js specifically for: (a) building a keyboard-first, client-heavy SPA with no real SSR/SEO need, (b) sharing UI components/design tokens with Agno's existing Next.js Workbench in the same monorepo (e.g. via a `packages/shared-ui` workspace package — is this easier if both apps use Next.js, or does it not matter much with a well-isolated component package?), (c) build/dev tooling overhead in a monorepo already using whatever Agno currently uses (check Agno's `package.json`/`pnpm-workspace.yaml`/`turbo.json` if present, to see what monorepo tooling — pnpm workspaces, Turborepo, Nx — is already in play, so Legal OS doesn't introduce a second one). Give a clear recommendation with tradeoffs, not just "either works."
2. **Split-view implementation approach.** Survey how comparable multi-pane terminal/IDE-style web UIs implement resizable/pinned split panes (e.g. libraries like `react-resizable-panels`, `allotment`, or a hand-rolled CSS grid approach like legal-terminal itself likely uses). Recommend one, with a note on keyboard-accessibility (can a pinned panel be closed/swapped without a mouse, matching the keyboard-first philosophy).
3. **Mnemonic command bar parsing.** This needs simple, robust tokenizing/autocomplete logic (mnemonic + trailing argument string, prefix-matched suggestions) — confirm whether a library is warranted at all (e.g. `fuse.js` for fuzzy matching against mnemonic+description text) or whether hand-rolled string matching (as legal-terminal itself does) is sufficient and preferable for something this small. Recommend, with rationale.
4. **State management.** Confirm Zustand is still the right choice for shell-level state (active view, sidebar collapsed state, split-view state, command history) regardless of the framework decision above — or flag if the Next.js path changes this recommendation (e.g. would Next.js's own patterns push toward React Context/Server state instead).
5. **Export-to-PDF/image.** Research current, actively-maintained libraries for exporting an arbitrary DOM panel to PDF or PNG client-side (e.g. `html2canvas` + `jsPDF`, or a server-rendered alternative). Note any known reliability issues with exporting complex CSS layouts (fonts, monospace tables) since this will be exporting data-dense legal panels.
6. **Font licensing/self-hosting.** Confirm licensing terms for self-hosting Playfair Display, Inter, and IBM Plex Mono (all should be open/free — Google Fonts originated all three — but confirm exact license, e.g. SIL Open Font License, and recommend a self-hosting approach (e.g. `@fontsource/*` packages) rather than a Google Fonts CDN call, consistent with the offline-first/local-data posture already established for this app.
7. **Case-phase switcher UX pattern.** Light research only — look for any existing UX pattern (not necessarily code) for a "context/mode switcher that reconfigures which shortcuts are surfaced" so the design has real precedent rather than being invented from scratch. Keep this one brief; it's the lowest-priority item in this category.

## Deliverable

A single markdown report, saved to the workspace, with:
- A clear framework recommendation (question 1) with tradeoffs stated explicitly, not hedged
- One recommendation per remaining research question, each with source citations (real URLs, current library versions)
- A proposed shell component/file structure sketch (directory tree, not full code) reflecting the chosen framework
- Explicit non-goals restated: no Help/Menu/Search/Cancel keys, no repeat-last shortcut, no TUI (Ctrl+K palette is back in scope — see feature set above)
