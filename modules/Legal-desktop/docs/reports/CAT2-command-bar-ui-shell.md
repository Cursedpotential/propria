# Category 2 — Command bar & UI shell

> _Byline: Grok · grok-4.6 · 2026-08-18_

## Framework (primary)

**Use Next.js 16.3.x + React 19.2.3 + Tailwind 4 + shadcn 3.8.x.**

The zip asked Vite vs Next. The approved build guide already locked
Next to stay compatible with Agno’s Workbench
(`workbench/web/package.json`: next `^16.3.1`, react `19.2.3`).
Agno is **not** an Nx/Turborepo monorepo — two Next apps with npm
lockfiles is the existing pattern. Do not introduce Vite as a second
app framework.

Tradeoff: SSR is unused for the command bar, but Matter Home,
deadlines, and release views benefit from RSC. Client islands cover
the editor and split panes.

legal-terminal-master remains the **shape** donor (dense graphite +
cmdk). ~~mnemonics~~ **corrected 2026-08-18:** ticker codes deleted.
Public identity is English `path` + `label`. Rewrite under `web/src`.

## Other recommendations

1. **Split view:** `react-resizable-panels` 4.12.x
   ([npm](https://www.npmjs.com/package/react-resizable-panels)).
   Keyboard: `Escape` closes the pinned pane; `Ctrl+\` toggles.
2. ~~**Mnemonic parser** (`CMD + rest`, `FCTR`/`ISSUE`/…)~~
   **corrected 2026-08-18:** type an English page name. Tab completes
   the first matching label. Resolve via `GET /v1/routing/resolve?q=`.
   No fuse.js.
3. **State:** Zustand for shell (active view, split, history,
   confidential flag). Next does not replace that. Server state for
   Matter/package stays in FastAPI.
4. **Export panel:** Phase 1 server-side (WeasyPrint or LibreOffice)
   rather than `html2canvas` (font/CSS fidelity fails on legal
   tables). Client PNG can wait.
5. **Fonts:** self-host `@fontsource/playfair-display`,
   `@fontsource/inter`, `@fontsource/ibm-plex-mono` (SIL OFL).
   No Google Fonts CDN.
   [Playfair OFL](https://github.com/clauseggers/Playfair/blob/master/OFL.txt).
6. **Case-phase switcher:** Bloomberg “yellow sector keys” analogue —
   Discovery / Motions / Hearing / Trial chips that reorder the
   mnemonic suggestion list. Lowest priority.

## Shell tree

```
web/src/
  app/page.tsx              Matter Home (exists)
  components/shell/
    Sidebar.tsx
    CommandLine.tsx
    CommandPalette.tsx      cmdk
    SplitWorkspace.tsx
    StatusPill.tsx
  store/shell-store.ts
  lib/surfaces.ts
```

## Non-goals

No hardware Help/Menu/Search/Cancel keys. No repeat-last shortcut.
No TUI.
