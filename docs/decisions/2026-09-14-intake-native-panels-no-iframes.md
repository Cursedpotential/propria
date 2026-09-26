---
title: "Intake surfaces are native Xplorer panels — the docked-iframe panel was the wrong tool (not a ban on iframes)"
date: 2026-09-14
status: decided
owner: Matt Salem
domains: [intake, consignatio, workbench, docs]
tags: [decision, owner-directive, intake, xplorer, ui-components, architecture]
---

# Intake surfaces are native Xplorer panels — the docked-iframe panel was the wrong tool (not a ban on iframes)

> _Byline: Claude Code · Fable 5.1 · 2026-09-14 22:05 EDT — recorded from the owner's words, session intake-56_

## Decision

Every Intake surface (review/metadata, search method selector and provenance, jobs, chat) is a **native panel inside the Xplorer client application** (`Intake/xplorer-copilot-buildkit/xplorer-copilot/apps/client/src`), registered like the engine's own panels and reading the engine's own selection and navigation state. **A panel that must share the engine's live selection and write edits back into the same state is written as native code in the engine, not as a separate document in an iframe with a postMessage bridge.** This is a tool-fit judgment for this surface, not a blanket rule: portals embed apps in frames because that is what portals do, and isolation cases (sandboxed previews of untrusted content) may still call for a frame. Decide per case and record why. Deployed builds never render sample data; sample sets exist only for Storybook and tests.

## Owner's words (2026-09-14 21:59–22:02 EDT)

- "that's a metadata viewing window that's not integrated"
- "that's a half-ass window smashed into a frame"
- "even without the sample that implementation is useless, it's disconnected, it's not usable"
- "it needs to be A PART OF THE APP, not a half-ass afterthought attachment"
- "IT HAS AN IFRAME IN AN IFRAME IN AN IFRAME"
- "there's a time and place for an iframe, like PORTALS. This isn't a portal. IT'S AN APP!"
- 22:08: "No iframes in this particular instance deployed this way. That doesn't mean no iframes ever. If I need to dig a hole, I use a shovel, not a fork. That doesn't mean forks are off the table."

## What this supersedes

The 2026-09-14 morning slice recorded in `Consignatio/Intake/docs/RECEIPT-2026-09-14-XPLORE-CB-CO-WORKSPACE.md`: progress-board shell iframe → Xplorer → `ReviewDockPanel.tsx` iframe of the separate review app (`../metadata/`) fed by `src/data/sampleReviewSet.ts`. That implementation is rejected and is being replaced in place.

## Consequences

- The portal (homepage / progress board) keeps framing the Intake app; that layer is legitimate. The header strip in the `intake-preview` wrapper is portal chrome and may stay or move into the app; it is not the defect.
- Backend access goes through one adapter module in the client to `casebible-corpus serve`; the renderer never holds database credentials (Intake AGENTS.md).
- Unavailable engine or missing catalog row: one small flag on the item, never a banner (owner rule 2026-09-13).
- Definition of done: verified live in a browser with a real selected file and a real search result, evidence in the receipt.
