---
title: Review screen layout — measured diagnosis and one-viewport proposal
date: 2026-09-20
status: BUILT AND VERIFIED LIVE 2026-09-21 02:45 EDT (owner 00:01: this session takes it); items 4-media and 5 still open
domains: [probata, workbench]
tags: [workbench, review, layout, density, design, pending-review]
---

# Review screen layout — measured diagnosis and one-viewport proposal

> _Byline: Claude Code · Fable 5.1 · 2026-09-20 23:50 EDT._

Owner 23:42 (with a screenshot of `/review?mode=TEST&resource=…`): "EVERYTHING ABOUT THIS IS TERRIBLE, THE SCREEN SPACE IS WASTED EVERYTHING IS EXPANDED SO FAR YOU HAVE TO SCROLL 4 TIMES TO SEE ANYTHING … need to go back to the drawing board."

## Measured (live page, 1281×720 viewport, `main.platform-workspace`)

Page height **3,615 px = 5.0 screens**.

| Section | Top (px) | Height (px) | Note |
|---|---:|---:|---|
| Sources and proposals | 295 | 255 | the list the owner picks from |
| Context processing | 565 | 145 | six checkpoints |
| Test operation destination | 788 | 132 | |
| **Context extraction package** | 1,010 | **2,512** | 70 % of the page |
| · four identity cards | 1,220 | 174 each | 3 of 4 say "unavailable" + a paragraph about the API |
| · Source repair and preprocessing | 1,736 | 430 | two more "unavailable" cards, 175 px each |
| · D-158 context storage destination | 2,186 | 352 | |
| · Go-managed structured extraction | 2,630 | 876 | |
| · · Tool catalog | 2,835 | 670 | three panes, all empty |
| **Next valid actions** | **3,538** | 126 | the only actionable block, last on the page |

## What is wrong

1. The one thing to act on is at pixel 3,538 of 3,615.
2. System internals are laid out as the page body; the extracted messages — what Review is for — are not on the page until six checkpoints pass.
3. Five cards spend 174–175 px each to say "unavailable" with an explanatory paragraph. Owner rule (auto-memory `one-flag-no-disclaimers`): unavailable = one small flag on the item, never a paragraph.
4. An empty tool catalog reserves 670 px.
5. Everything is expanded by default; nothing collapses; nothing is sticky.

## Proposal (wireframe shown to the owner in chat)

One viewport, no page scroll:

- **Top strip (one line, sticky):** source name · six checkpoints as six small marks · exception flags (e.g. "2 missing payloads") · the next valid actions as buttons (incl. retry).
- **Left rail:** sources list with counts and a failed marker (replaces the 255 px block).
- **Centre:** the messages, dense rows (time · sender · text · attachment/missing flag), search and filter on top. Glide Data Grid per `modules/workbench/web/AGENTS.md`.
- **Right rail, collapsed by default:** Source and package · Repair · Storage destination · Tools. An unavailable value is a small `n/a` flag on its row.
- Empty panes take no space.

## Coordination

Branch `feat/review-message-browser` (another session) is building the message browser and was merged toward `main` on 2026-09-20. This layout must be agreed with that lane, not built beside it.

## Owner decision — 2026-09-20 23:48 EDT

- On the one-viewport layout: "THAT WOULD BE MUCH BETTER".
- "AND THE STRAIGHT SBV PORTED VIEW AS OPTIONAL" — the SBV viewer's conversation view is an optional view of the same messages, not the default and not a separate app (auto-memory `work-surface-is-probata-workbench`: never deploy SBV as the surface).
- "BUT BORROW FROM IT" — the default view takes SBV's good parts.

### What SBV's viewer has (`modules/forks/sbv/frontend/src/components/`, inventory 23:50)

| SBV component | Lines | Borrow into |
|---|---:|---|
| `ConversationList.jsx` | 170 | left rail: threads with last message, time, count |
| `MessageThread.jsx` | 826 | the optional conversation view; its scroll-to-message, search highlight, lazy media and inline audio/video also serve the dense grid's detail pane |
| `MediaGrid.jsx` + `MediaCarousel.jsx` | 395 | a per-thread media tab — also where a missing payload shows as an empty slot with its flag |
| `Search.jsx` + `DateFilter.jsx` | 278 | toolbar: cross-thread search, jump to date |
| `PrintView.jsx` | 401 | court-ready print/export of a thread |
| `Calls.jsx`, `Activity.jsx`, `VCardPreview.jsx` | 716+ | calls log view, activity over time, contact cards |

### Already on `main` from the message-browser lane (same night)

`3d61972` three-panel message browser (`web/src/components/sbv/message-browser*.tsx`, `message-detail-panel.tsx`, `message-source-panel.tsx`), `f0fefd8` Messages mounted as the Review view for messaging sources, `967fd8d` Sources and proposals as a compact filterable one-line list. `feat/review-message-browser` has nothing ahead of `main`. The live page measured above at 23:45 did not show these yet (Workbench not redeployed, or the measured resource is not a messaging source — not checked).

### Still open against the approved layout

1. Actions + checkpoints + exception flags in a sticky top strip (today: "Next valid actions" at the bottom).
2. "Context extraction package", repair, storage destination and tools collapsed into a right rail; `unavailable` as a small flag, no paragraph.
3. Empty tool catalog takes no space.
4. A view switch on the centre pane: grid (default) · conversation (SBV-ported) · media.
5. Missing-payload flag per message and per thread (engine now emits `attachment_references` kind `mms_part_without_payload`).

## Built and measured live — 2026-09-21 (Claude Code · Fable 5.1)

Owner 00:01: "B: I take them … i think its done" (the other lane). Its finished branch `feat/preview-search-and-calls` (SBV front-end port, server-side search, calls table, calls-only publish) was unmerged and undeployed; merged first (`f94acef`), then the layout was built on top.

Same resource, same 1280×720 viewport, `main.platform-workspace`:

| Step | Commit | Page height | Screens |
|---|---|---:|---:|
| Before | — | 3,615 px | 5.0 |
| Sticky actions, compact rows, folded sections, Messages default, one-line checkpoints | `afddee1`-era | 1,574 px | 2.2 |
| Sources as a left rail, one-line header | `c16f42e` | 1,081 px | 1.5 |
| Browser sized to the window, one-row checkpoints, one-line event notice | latest | **728 px** | **1.01** |

Live facts after the last deploy: lands on **Messages**; search totals read "927 of 927"; actions strip at 270 px (was 3,538); checkpoints 48 px (was 145); page header 42 px (was 181); the merged branch already provides the **Conversation / Table** switch, filters (has attachments, date range) and server-side search. Checks each step: `tsc`, lint 0 errors, build, smoke 69/69.

### Still open

- **Media view** in the centre switch (grid · conversation · **media**). Attachments now publish (`b80d884`, other lane, verified live 01:56).
- **Missing-photo flag** per message and thread: the engine emits `attachment_references` kind `mms_part_without_payload`; the preview projection and the grid do not carry it yet.
- Package / repair / storage / tools live in the **Overview tab** (folded, compact rows), not a right rail. Moving them to a rail is a further step if the owner still wants it after using this.
- The live page shows "The Proffer preview event stream is unavailable" — the event stream is not connecting. Now a one-line notice; the cause is not investigated.
- At 1280×720 the scroll region still overflows by 133 px; on a 1080-high window it fits.
- `workbench/api` `test_file_size_limits` fails on `main` (runtime/proffer.py 363 lines, service/proffer.py 345; cap 300) — predates this work.
- npm's cache on this desktop is on `C:` (`AppData/Local/npm-cache`); `npm ci` for this checkout used it.
