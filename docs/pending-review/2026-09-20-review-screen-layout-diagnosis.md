---
title: Review screen layout — measured diagnosis and one-viewport proposal
date: 2026-09-20
status: proposal awaiting owner; nothing built
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
