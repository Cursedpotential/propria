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

## SBV (`modules/forks/sbv`) — ported viewing front end

- **Project:** SBV (`modules/forks/sbv`, canonical remote `Cursedpotential/sbv-forensic`
  — a donor project, not a fork; see `Probata/probata/AGENTS.md`)
- **License:** MIT, Copyright (c) 2025 lowcarbdev
- **Referenced paths (frontend, `modules/forks/sbv/frontend/src/`):**
  `components/ConversationList.jsx`, `components/MessageThread.jsx`,
  `components/Calls.jsx`, `components/Search.jsx`, `components/DateFilter.jsx`,
  `components/MediaGrid.jsx`/`.css`, `components/MediaCarousel.jsx`/`.css`,
  `components/LazyMedia.jsx`/`.css`, `components/VCardPreview.jsx`,
  `utils/vcfParser.js`
- **What was taken:** the message-thread reading behavior (chat bubbles,
  incoming/outgoing alignment, group-conversation sender labels), the calls
  history card layout, the date-range filter shape, the media grid/carousel
  interaction pattern, and the vCard parsing algorithm and preview layout —
  ported into `src/components/sbv/message-thread-view.tsx`,
  `message-bubble.tsx`, `attachment-preview.tsx`, `media-only-grid.tsx`,
  `media-only-carousel.tsx`, `date-range-filter.tsx`, `vcard-preview.tsx`,
  `calls-table.tsx`, and `src/lib/sbv/vcf-parser.ts`. Each ported file carries
  its own header comment naming the exact origin path.
- **What was adapted:** SBV's own backend (`/api/messages`, `/api/calls`,
  `/api/media`, `/api/search`, axios, react-router) is never called from this
  app — every request goes through this Workbench's own governed
  `ProfferPreview*` API. Bootstrap classes and inline SVGs were replaced with
  this app's Tailwind/shadcn tokens. `react-datepicker` and `date-fns` were not
  added as dependencies (not already present in this app, and no new npm
  package may be added for this port); `date-range-filter.tsx` uses native
  `<input type="date">` instead. Media rendering is bounded by this BFF's
  actual capability: post-ingest attachments load from the real
  `GET /api/proffer/previews/{handle}/media/{sha256}` endpoint the same way
  SBV's `LazyMedia`/`MediaGrid`/`MediaCarousel` load their own `/api/media`,
  with a metadata-only fallback card (never a fabricated URL) when that file is
  not yet available — see `attachment-preview.tsx` for the exact boundary and
  the one still-open gap (pre-ingest inline base64 media, which no field in
  this app's `ProfferPreviewAttachment` contract currently carries).
- **What was left out:** `Login.jsx`, `ProtectedRoute.jsx`, `Upload.jsx`,
  `SettingsModal.jsx`, `ChangePasswordModal.jsx`, `ThemeToggle.jsx`,
  `Activity.jsx`, `EvidenceImports.jsx`, `PrintView.jsx`/`.css`,
  `Summary.jsx`/`.css`, and SBV's own auth/backend routes — none of these
  apply to a read-only projection over this app's own API.
- **Where it applies:** `src/components/sbv/message-thread-view.tsx` and its
  sibling `message-bubble.tsx`, `attachment-preview.tsx`, `media-only-*.tsx`,
  `date-range-filter.tsx`, `vcard-preview.tsx`, `calls-table.tsx`, and
  `src/lib/sbv/vcf-parser.ts`.

### MIT License text

```
MIT License

Copyright (c) 2025 lowcarbdev

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
