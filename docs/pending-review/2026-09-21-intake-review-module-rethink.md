---
title: Intake + Review — module rethink (what it is for, what goes, what stays)
date: 2026-09-21
status: PROPOSAL — owner has not ratified; nothing built from it
domains: [probata, workbench]
tags: [workbench, intake, review, rethink, design, pending-review, jobs-to-be-done]
---

# Intake + Review — module rethink

> _Byline: Claude Code · Fable 5.1 · 2026-09-21 08:45 EDT._

Owner 08:32: "The review screen is closer … but it doesn't function very well … the Go Tools menu … nothing listed. When I select a decoder, it doesn't do anything. It locks up … Intake is still just broken as shit. I don't even know what it's for. Now you need to rethink this entire module." And: "use it on this end and just use it."

## What I saw (live pages, headless Chrome on the desktop, 1600×1000, 08:32)

First time this session looked at the rendered pages instead of API responses.

**Intake**
1. Opens on "Indexed catalog": three folders (gdrive, local, onedrive) that are where files USED to live. B2 — where everything is — is behind a second tab.
2. Five paragraphs of caveats before any file ("coverage is not established…", "Index coverage: unknown…", "require separate verification").
3. A six-step rail, a "Fixed case" card, a "Source integrity" card, a "Workflow receipt" card and a yellow "read-only" warning are all on screen before a source is chosen. None helps choose one.
4. Nothing says what the page is for or what happens after a file is picked.

**Review**
1. The run list cuts every name to one or two letters ("s…", "8…", "b2…") because the status text takes the row.
2. It opens on "Overview": a grid of `n/a` / `pending` system fields. The messages — the reason to be here — are behind "Source records" or "More".
3. TEST/REAL is on screen three times, plus "TEST resource selected", plus a "TEST OPERATION DESTINATION" bar.
4. "The Proffer preview event stream is unavailable" banner; an empty "Next valid actions" box.
5. "Go-managed structured extraction and tools" (the Go tools menu): owner reports nothing listed and a lock-up on selecting a decoder. Not reproduced yet (needs clicks; headless covers load-state only).

## What the module is for (the job)

One person has ~2 TB of his own records in B2. He needs to **get them into a form he can read and search, in bulk, and then read them**. Everything the pipeline does in between (hashing, handler choice, normalization, receipts) is machinery. Today the screens are organised around the machinery; the rethink organises them around the two things he does.

| He does | Today | Rethought |
|---|---|---|
| Bring files in | "Intake": one file at a time, declare a format, watch six checkpoints, answer gates | **Sources** |
| See what is happening | spread over Intake rail, Review list, banners | **Activity** |
| Read and decide | "Review": system fields first, messages behind tabs | **Read** |

## Proposal

**1. Sources (replaces Intake).** A file browser over B2. Folder tree left, files centre, **metadata panel right** (owner 2026-09-22 09:00: "i need the meta data in view and someway to signify its a unit of some kind"). Every row carries one state mark: not processed · decoded · in context · failed. Select files or a folder → one button, **Process**. Format is detected, never declared.

- **Metadata panel (always open, follows selection):** name, size, modified, sha256 (from the catalog when it has one, else "not hashed"), detected format, decode state and counts (messages / media / rejected once decoded), which runs touched it, and the catalog's provenance for it (original location, occurrence count). Below that, the preview: the Messages view for a backup, the picture / PDF / text for anything else.
- **Units are marked in the tree and the list.** A folder that is a recognised unit (Takeout set, CubeACR folder, Obsidian vault, git repo, Snapchat export, a hand-marked folder — the catalog's `unit_type` and the bulk-intake unit registry, requirement 4) shows a unit badge with its kind on the folder row, and its member files show a small "part of <unit>" mark. Selecting the folder selects the unit; the metadata panel then shows the unit's own facts (kind, parts present / missing, total size, account for a Takeout). Marking a folder as a unit by hand happens from the same panel (requirement 4: "select dir from tree and mark as unit"; a Takeout unit stays a supervised proposal, requirement 7).

- **Search is built in FROM STEP ONE — it ships with Sources, not after it** (owner 2026-09-22 09:06 "from step one"), across every store (owner 2026-09-22 09:06; restates 2026-09-20 20:01 "it's supposed to use the index and Surreal and the graph and Weaviate when available — that's why we gave these tools").** One search box on Sources with modes: names and paths (catalog), file contents (the corpus CocoIndex super-index), meaning (Weaviate), relationships (SurrealDB graph). Results are B2 files or units, each with its state mark, and selecting a result is the same as selecting it in the tree. A mode that is not reachable shows one small flag on the mode, not a paragraph. The same modes serve Read. What blocks it today (2026-09-20 finding): the CocoIndex backend's catalog query points at a relation missing from the live database, and Probata bypasses the discovery layer — both must be fixed for this, not worked around.

**2. Activity.** One line per run or batch: name (full file name), state, counts, time. A run only asks for a click when it has an exception — damaged file (repair choice), no parser, missing payloads. Clean files go through without a click (owner requirement 6, 2026-09-20). Batches show progress and "retry failed".

**3. Read (replaces Review).** Conversations left, messages centre, detail right — the layout the owner approved 23:48, with messages as the landing view, not Overview. Search across every conversation, not one run. Flag / send on from the detail panel. Approve/reject stays, as one bar.

**Moves out of the operator's way** (kept, one click away, under a per-run "Technical details" drawer): package identity and hashes, D-158 storage destination, repair/preprocessing detail, checkpoints, receipts, the Go tools catalog. The Atomic/Go tools page becomes a developer page, not a tab in the daily flow.

**Shown once:** TEST/REAL (top bar only). **Removed:** caveat paragraphs (one small flag on the item instead — owner rule `one-flag-no-disclaimers`), the event-stream banner, empty boxes.

## Options for the owner

- **A (default): rebuild around the three screens above**, reusing what works today — the B2 browser, the Messages viewer, the Glide message browser, search, media streaming, the batch engine. Roughly: Sources first (it is the front door and the most broken), then Read, then Activity.
- **B: keep Intake and Review as pages, strip and re-order them** (hide the machinery, messages first, names readable, B2 default). Faster, but the one-file-at-a-time shape and the split between "pick" and "see progress" stay.
- **C: make the Xplorer-based Intake the front door** and point Probata's Sources at it (owner direction 2026-09-16: Intake = one co-workspace on the Xplorer engine). Biggest change; the Probata pages shrink to Activity + Read.

## Before building either way

Fix what is simply broken, regardless of option: run names readable; Review lands on messages; TEST/REAL once; B2 the default tab; reproduce and fix the Go tools lock-up (or take the panel off the page).
