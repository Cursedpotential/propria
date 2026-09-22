---
title: D-159 — The goal is six steps; every surface is measured against them
date: 2026-09-22
status: active
authority: owner_decision
domains: [probata, workbench, intake, consignatio, docs]
tags: [decision, owner-decision, purpose, six-steps, review, sources, intake, surfaces]
---

# D-159 — The goal is six steps; every surface is measured against them

> _Byline: Claude Code · Fable 5.1 · 2026-09-22 19:54 EDT. Owner ruling, recorded the same hour._

## Ruling

The goal of this application is six steps, performed by one person:

1. Open a file or folder, through an index.
2. Verify whether it is relevant.
3. Make sure it has a hash, so it can be trusted later.
4. Pick a parser or extractor and get it into context and into the processing and analysis platforms.
5. Preview the result after the parse and extract, to make sure the machine did it right.
6. Fill in gaps and missing context.

Sorting (the Vault / Consignatio / Intake work) exists only because step 1 could not be done on the corpus as it stood. The owner expected all six to be done by now.

## Rule

Before adding anything to a surface, name which step it serves. If it serves none, it does not go on the surface.

- **Sources** owns steps 1–4: browse by folder and index; see name, size, kind and unit membership; preview before anything runs; the hash visible; detected format and one Process action.
- **Review** is steps 5–6 plus the accept/reject decision: the file's preview, full screen; photos inline; anything missing flagged on the item; the gap and its leads; the one or two next actions in plain words.
- Run status, receipts, package identity, storage destination, lineage, attempts and tool catalogs are never primary content on an operator surface. They are a drawer or their own screen.

## Why

Review had descended from the July 20 Operator Console brief (observability: see every stage, receipt and store) and that brief had become the page's primary content; the messages — the thing step 5 is about — were a 40 px strip under a tab (owner, 19:39: "utterly garbage… the whole process needs to be re-thought"). The purpose had to be reconstructed from four months of session logs; this decision makes it a stated rule so that never happens again.

## Record

`docs/PURPOSE.md` holds the dated trail from 2026-06-01 to today. Root `AGENTS.md` carries the six steps as a binding section. `modules/workbench/AGENT_MEMORY.md` carries them at the top of the surface router.
