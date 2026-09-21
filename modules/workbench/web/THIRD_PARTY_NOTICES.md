# Third-party notices — Workbench browser application

> _Byline: Claude Code · Opus 5 · 2026-09-20_

This file records third-party material whose design or source influenced code in
`modules/workbench/web`. Runtime dependencies and their licenses are declared in
`package.json` and `package-lock.json`; this file covers borrowed *source and design*.

## RAGFlow — layout pattern only, no code copied

- **Project:** [infiniflow/ragflow](https://github.com/infiniflow/ragflow)
- **License:** Apache License 2.0
- **Referenced path:**
  `web/src/pages/chunk/parsed-result/add-knowledge/components/knowledge-chunk/index.tsx`
- **What was taken:** the *shape* of a review screen — a toolbar above a
  virtualized list inside a resizable panel group, with detail columns driven by the
  current selection rather than by a per-row "details" action, and panels given
  explicit `id`/`order` so a conditionally mounted middle column keeps its position.
- **What was not taken:** no RAGFlow source code, styles, types, or assets are copied
  into this repository. The implementation is written against this project's own
  stack (React 19, TanStack Query, Glide Data Grid, `react-resizable-panels`) and its
  own `ProfferPreview*` API contracts.
- **Where it applies:** `src/components/sbv/message-browser.tsx` and its sibling
  `message-browser-*.tsx` / `message-*-panel.tsx` components.

Because no Apache-2.0 source text was copied, no Apache header is carried in those
files; the attribution above is recorded here instead.
