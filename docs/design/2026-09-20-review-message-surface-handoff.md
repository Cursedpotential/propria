---
tags: [probata, workbench, review, design, handoff, sbv, messages]
status: draft — owner has not ratified
---

# Handoff Spec: Review — message source view

> _Byline: Claude Code · Fable 5.1 · 2026-09-20. Source: owner's screenshot and words on the live page (23:10–23:16 EDT), the deployed code, `src/app/globals.css`, and the SBV front end in `modules/forks/sbv/frontend/`._

## Overview

One person reviews eight years of his own messages before they are published as context. He opens a run, reads the conversation, checks where it came from, and approves or rejects it. Today the page makes him scroll past a banner and eleven tabs to reach raw JSON (owner: "god awful … i have to scroll way too much, and i have to read in the json").

**The rule this spec enforces:** the first thing on screen is the conversation, readable like a phone thread; everything else is one click away and never in the way.

Scope: the Review page for a **messaging** source (SMS/MMS thread chunk, call log). Non-messaging sources keep today's layout.

Out of scope (capability does not exist — do not mock it): entity extraction, claim candidates, relationships/graph content, editing or correcting a record. Those tabs get a one-line empty state.

## Layout

Desktop-first; the owner works on a 1920×1080 display with the 248 px app sidebar open.

```
┌ page header (existing) ───────────────────────────────────────────────┐
├ run strip  — 1 line, 32 px ──────────────────────────────────────────┤
├ Sources and proposals — collapsible, 1 line per run, max 224 px ──────┤
├ view bar — 40 px: [Thread][Table][Records][Attachments][More ▾]  search│
├───────────────────────────────────────────────────────────────────────┤
│ conversation (fills remaining height, own scroll)  │ detail (resizable)│
│                                                    │  message          │
│                                                    │  provenance       │
└───────────────────────────────────────────────────────────────────────┘
  decision bar — sticky bottom, 48 px: status · Approve · Reject
```

- The content region is `height: calc(100dvh - header - strip - viewbar - decisionbar)`; the **page body must not scroll** while a messaging source is open. Only the conversation and the detail panel scroll.
- Two resizable panels (`react-resizable-panels`, already a dependency): conversation `minSize 45`, detail `defaultSize 32, minSize 22, collapsible`. Persist sizes with `autoSaveId="review-thread"`.
- "Sources and proposals" (already shipped as `review-resource-list.tsx`) gains a collapse toggle; it auto-collapses to its header once a run is selected.

## Design Tokens Used

All values are the existing variables in `src/app/globals.css`; use the Tailwind names, never hex.

| Token (Tailwind) | Light | Dark | Usage |
|---|---|---|---|
| `bg-background` | `#f5f3ee` | `#1d252c` | page |
| `bg-card` / `.platform-panel` | `#fffefb` | `#242e36` | panels, incoming bubble |
| `border-border` | `#d5d1c9` | `#43505a` | panel and row rules |
| `bg-primary` / `text-primary-foreground` | `#4051b9` / `#fff` | `#8591f0` / `#111820` | outgoing bubble, active view tab underline |
| `bg-accent` / `text-accent-foreground` | `#e9ecfb` / `#2f3d9c` | `#313a66` / `#dfe3ff` | selected message, selected run, search hit background |
| `text-muted-foreground` | `#687078` | `#b1b8bd` | timestamps, meta, empty states |
| `text-destructive` | `#b5433b` | `#e06e65` | failed run dot, reject |
| `--chart-3` | amber | amber | TEST-mode dot only (replaces the amber banner) |
| `font-sans` | Instrument Sans | — | all reading text |
| `font-mono` | IBM Plex Mono | — | ids, hashes, `.platform-kicker` labels |
| `rounded-md` (`--radius` 0.25rem − 2px) | — | — | chips, inputs |
| bubble radius | `rounded-2xl` with the tail corner `rounded-sm` | — | thread bubbles only |
| spacing | Tailwind 4 px scale: `gap-1` within a group, `gap-3` between groups, panel padding `px-4 py-3` | | |

Type scale: body `text-sm` (14 px) for message text — not smaller; meta `text-[11px]`; section labels `.platform-kicker`.

## Components

| Component | Source | Notes |
|---|---|---|
| `RunStrip` | new, replaces the amber "TEST operation destination" card | one line: `● TEST` dot + matter name · source file name · lifecycle chip. Matter id, court case id, attempt, request id live in a popover opened from an `i` button (copy buttons inside). |
| `ReviewResourceList` | shipped `review-resource-list.tsx` | add collapse; otherwise unchanged. |
| `ViewBar` | new | primary views as tabs; `More ▾` menu holds Overview, Chunks, Entities, Relationships, Graph, Lineage, Warnings, Attempts / runs. A view whose data is empty shows a muted count `0` in the menu. No views are deleted. |
| `ThreadView` | **port of SBV `MessageThread.jsx`** (826 lines, MIT © 2025 lowcarbdev) | default view. Bubbles: outgoing right/`bg-primary`, incoming left/`bg-card` with border. Max bubble width `min(68ch, 72%)`. Consecutive messages from one sender within 5 min group under one timestamp. Day separator: centered `.platform-kicker` rule, sticky at top while scrolling that day. |
| `ThreadSearch` + `DateFilter` | port of SBV `Search.jsx`, `DateFilter.jsx` | lives in the view bar, right side. Server-side (`q`, `from`, `to`, `has_attachments`, `sender`). Shows `12 of 927`. Enter / F3 next hit, Shift+Enter previous; hits highlighted with `bg-accent` and the thread scrolls the hit to center. |
| `AttachmentStrip` | port of SBV `LazyMedia.jsx`/`MediaGrid.jsx` layout | under the bubble: thumbnail slot 96×96, name, type, size. ~~Binary preview is not wired~~ **Corrected 2026-09-20 23:31 (owner):** two modes — before processing SBV decodes the base64 part from the backup; after ingest the bytes stream from `<key>.derived/media/<sha256><ext>`. Click opens the SBV carousel. |
| `CallsView` | port of SBV `Calls.jsx` | for call records: direction icon, number/name, local time, duration `1h 02m 03s`; missed calls `text-destructive`. |
| `TableView` | shipped Glide browser (`message-browser*.tsx`) | the dense mode; same query, filters and selection as the thread. |
| `RecordsView` | replaces the JSON cards | Glide grid: `#`, type, time, sender → recipients, body (single line), attachments. Raw JSON only inside the detail panel's `Raw` disclosure, collapsed by default, monospace, copy button. |
| `DetailPanel` | shipped `message-detail-panel.tsx` + `message-source-panel.tsx`, merged into one panel with two sections | follows selection in every view. Holds the single "Potential future use" action (one form for the selected record — never one per record). |
| `DecisionBar` | existing approve/reject controls, moved | sticky bottom; disabled with the reason as its tooltip when approval is locked (replaces the yellow paragraph). |

## States and Interactions

| Element | State | Behavior |
|---|---|---|
| Message bubble | hover | `ring-1 ring-border`; timestamp shows full local date-time in a tooltip |
| Message bubble | selected (click, or ↑/↓) | `ring-2 ring-ring` + `bg-accent` tint on incoming, 2 px `ring-ring` on outgoing; detail panel updates; **no "Details" button exists anywhere** |
| Thread | scroll near bottom | loads the next page (forward cursor); appended messages never move what is on screen |
| Thread | open | lands on the **oldest** message and reads forward (owner 2026-09-20 23:28; ~~newest, like a phone~~); `Home` jumps to oldest, `End` to newest |
| Search field | typing | 300 ms debounce; spinner inside the field; `Esc` clears and returns focus to the thread |
| View tab | active | 2 px `border-primary` underline; others `text-muted-foreground` |
| Run row | selected | `bg-accent`; list collapses |
| Approve / Reject | locked | disabled; tooltip states the one missing requirement |
| Approve / Reject | submitting | spinner in the button, both disabled |
| Raw disclosure | open | 240 px max height, own scroll |

## Responsive Behavior

| Breakpoint | Changes |
|---|---|
| ≥ 1280 px | two panels side by side (default) |
| 1024–1279 px | detail panel starts collapsed; opens as an overlay sheet from the right (360 px) on selection |
| < 1024 px | single column; detail is a bottom sheet (60 % height); view bar scrolls horizontally; Glide Table view is replaced by the thread (tables are unusable on a phone) |

## Edge Cases

- **Empty view** (Entities, Chunks, Graph…): one muted line, e.g. "No entities — this pipeline does not extract them yet." No scaffolding, no disabled controls.
- **Zero search hits**: "No messages match" + a `Clear` link; the thread stays visible underneath, dimmed 40 %.
- **Very long message** (> 1,200 chars): clamp to 12 lines with `Show all`; never truncate inside the detail panel.
- **Group thread**: sender name above each incoming group (`text-[11px] font-medium`); color the name from `--chart-1…5` by participant index.
- **`self`**: render as "You"; never show the literal `self`.
- **Missing timestamp**: bubble shows "time unknown" and sorts by ordinal; one small flag on that bubble.
- **Attachment named but no bytes in the backup** (`attachment_references` with `source_reported_missing`): dashed 96×96 slot, "missing from backup", original file name kept.
- **Loading**: 6 skeleton bubbles alternating sides; the view bar and run strip render immediately.
- **Error**: inline in the conversation area with a `Retry` button; the rest of the page stays usable. Never a full-page error.
- **Slow connection**: pages of 200; show "Loading older messages…" pinned at the top while fetching.
- **927 → 100,000 messages**: the thread must virtualize (`@tanstack/react-virtual`, variable row heights measured on render); never render more than ~60 bubbles.

## Animation / Motion

| Element | Trigger | Animation | Duration | Easing |
|---|---|---|---|---|
| Detail panel | collapse / expand | width | 160 ms | `ease-out` |
| Resource list | collapse | height | 160 ms | `ease-out` |
| Search hit | jump to hit | scroll + 600 ms `bg-accent` pulse on the bubble | 200 ms scroll | `ease-in-out` |
| New page of messages | prepend | none (anchor scroll; movement here is disorienting) | — | — |

All motion is removed under `prefers-reduced-motion` (`motion-reduce:` utilities, as the codebase already does for spinners).

## Accessibility Notes

- Focus order: run strip → resource list toggle → view tabs → search → conversation → detail panel → decision bar.
- Conversation is `role="log"` with `aria-label="Conversation with <participants>"`, `aria-live="off"` (history, not live). Each message is a `role="article"` with an accessible name "You, 2:46 PM: <first 80 chars>".
- Roving tabindex inside the conversation: ↑/↓ move between messages, `Enter` focuses the detail panel, `Esc` returns. `/` focuses search.
- View tabs are a real `role="tablist"`; `More` is a `menu` button.
- Day separators are `role="separator"` with the full date as the label.
- Contrast (computed, WCAG 2.x): outgoing bubble text `#fff` on `#4051b9` = 6.78:1 (light), `#111820` on `#8591f0` = 6.22:1 (dark); meta text `--muted-foreground` on `--card` = 4.98:1 (light) and 6.89:1 (dark); `--accent-foreground` on `--accent` = 7.87:1. All pass AA for 14 px text; light-theme meta text is the tightest at 4.98 — do not use it below 11 px or on `--muted`. Search highlight must not rely on color alone: add a 2 px `ring-ring` outline.
- Every icon-only button has an `aria-label`; the TEST dot has the text "TEST" beside it.

## Implementation notes

- Stack: React 19 + Vite + TanStack Router/Query, Tailwind tokens above, Glide `6.0.4-alpha24` (owner-approved 2026-09-20), `react-resizable-panels`. Add `@tanstack/react-virtual` for the thread (check the version the xplorer-copilot buildkit already pins).
- Data: `GET /api/proffer/previews/{handle}/messages` (cursor paging; server-side search params are being added on `feat/preview-search-and-calls`), `/content` for package facts and normalized records.
- Ported SBV files keep a header: origin path + "MIT, Copyright (c) 2025 lowcarbdev"; SBV is listed in `modules/workbench/web/THIRD_PARTY_NOTICES.md`.
- Contracts to update with the change: `smoke/proffer-operator-surface.contract.test.mjs` (landing view, the supersession of `platform-message-viewer.tsx`).

## Owner decisions (2026-09-20 23:28–23:31 EDT)

1. Thread opens on the **oldest** message and reads forward in time. Paging appends the next page near the bottom; `Home` = oldest, `End` = newest.
2. Extra tabs go in the `More ▾` menu ("more for now" — keep it cheap to change).
3. After this surface: the **extraction stage** (entities, claim candidates) comes first — there is nothing to add to or correct until it exists. The edit/correction API follows it.
4. Media preview is wired in two modes, both rendered the SBV way: **before processing**, SBV decodes the base64 parts inside the backup; **after ingest**, media streams from the run's adjacent `<key>.derived/media/` folder and the thread reads the new `<key>.derived/threads/` chunks (`GET /api/proffer/previews/{handle}/media/{sha256}`).
