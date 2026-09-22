---
title: What this app is for
date: 2026-09-22
status: owner statement, verbatim; the measure every screen is held to
domains: [probata, workbench, intake, consignatio]
tags: [purpose, canon, owner-decision, review, sources, intake]
---

# What this app is for

> _Byline: Claude Code · Fable 5.1 · 2026-09-22 19:54 EDT. Recorded after the owner asked "what is this screen supposed to do?" and the answer had to be reconstructed from four months of logs. Nothing in this file is inferred; every line below the first section is the owner's own words with its date._

## The goal — owner, 2026-09-22 19:54 (verbatim, capitalisation his)

> OPEN FILE OR FOLDER (HOPEFULLY THROUGH AN INDEX)
> VERIFY IF ITS RELEVANT IN ANY WAY
> ENSURE IT HAS A HASH IT FOR LATER
> PICK A PARSER OR EXTRACTOR TO GET IT INTO CONTEXT AND INTO THE PROCESSING AND ANALYSIS PLATFORMS
> PREVIEW AFTER THE PARSE AND EXTRACT TO MAKE SURE THE MACHINE DID IT RIGHT
> FILL IN GAPS AND MISSING CONTEXT!!!
> THAT'S THE FUCKING GOAL. I ADDED SORTING CAUSE YOU ALSO CAN'T GET THAT RIGHT, SO WE NEED TO DO WHAT I HAD HOPED WOULD ALREADY BE DONE BY THIS POINT.

Six steps. One person. That is the whole operator surface:

| # | Step | What the screen must let him do |
|---|---|---|
| 1 | **Open** a file or folder, through an index | Browse the vault by folder; find by name or content; see name, size, kind, and whether it is a unit (a Takeout, a backup family). |
| 2 | **Verify relevance** | Look at it — messages, picture, PDF — before anything runs. Decide it matters or it doesn't. |
| 3 | **Ensure it has a hash** for later | The hash exists and is shown; he never has to ask whether custody was recorded. |
| 4 | **Pick a parser or extractor** and get it into context and the analysis platforms | Detected format, one Process action; a choice only when there really is more than one way. |
| 5 | **Preview after the parse** to make sure the machine did it right | The extracted messages/records, readable, full screen; photos inline; anything missing flagged on the item. |
| 6 | **Fill in gaps and missing context** | Where the file is short (missing payloads, truncated backups, unmatched sources), see the gap and the leads, and attach what fills it. |

Sorting (the Vault / Consignatio / Intake work) was added because step 1 could not be done on the corpus as it was. It is in service of the six steps, not a seventh product.

## Where the six steps came from (the record)

- **2026-06-01** — first message in the logs: the codebase "should match" `Agno_MCP_Platform_MVP_Handoff_Guide_v8`; "that is what this was supposed to be." The v8 purpose: an assistant layer over the existing MCP tool platform that ingests, normalizes, reviews and analyses with a human gate on every write, and helps build itself toward the Semantica evidence platform. Before that: the Google Takeout timeline reader ("nine months of code trying to decipher that", 07-03) and the mcp-tool-platform / TheBigOne mockups (07-04).
- **2026-06-20** — the job in plain words: "Brain-dump → timeline… a workflow that won't eat up all my usage before completing, that can scan through my dumps of directories with transcripts, identifying conversations and pulling that stuff out." "Anything can be consumed by the agents to further the platform or evidence searching or evidence presentation, timeline creation."
- **2026-06-25** — type first, then domain ("is it evidence or is it knowledge; once it's evidence, then by domain"); hashes at every level "before we actually do evidence"; the first normalized output rejected because "the user and AI messages just run together with no distinction."
- **2026-06-29** — "The Case Bible ultimately is the agent context for the application as well as the vault for the evidence and knowledge."
- **2026-07-07** — "Essentially what we are doing is creating a bitemporal graph RAG… normalize the conversation and messaging evidence into a real PG database, message by message."
- **2026-07-20** — Workbench v1 rejected: "what the fuck does 'promoted' even mean… what I was imagining was, rather than sending things blind, being able to actually drive the workflow, drive the tools." Operator Console requirements: verify parsing by eye, verify hashes, curate, flag "needs corroboration."
- **2026-08-02** — the point of the analysis: "it's just a permissions thing and which agents have hindsight… an agent completely ignorant to everything until we assemble everything; then an agent lives it the way I did." The delta is the gaslighting (canon §1).
- **2026-08-19** — "I want a unified surface, I want them to communicate, you're missing the whole point."
- **2026-08-29** — "The SBV GUI is supposed to be the front end, the preview window — a client that sits above and can call the Go agent and the workflows and view things as it goes through the pipeline."
- **2026-09-05/06** — names (D-137…D-150): propria (umbrella) · Indicia Probata / probata (evidence record: ingest, custody, normalize — this repo) · Indagatio Veri / indagatio (analysis: horizon walks, the delta, SurrealDB) · consignatio (the Vault / Case Bible + Intake) · advocatio (legal workbench) · vestigia (geo) · proffer (ingest lane, was UIW) · workbench (UI shell) · intake (ingest client) · admit (promotion, proposed). D-145: lifecycle is context → Surreal analysis → evidence; the owner's clicks are "send to Surreal" and "promote."
- **2026-09-12** — surfaces: Evidence Operations Desk (everyday), Modular Service Cockpit (advanced, gated).
- **2026-09-22 13:00** — Sources (replaces Intake): a B2 file browser and nothing else; tree left, files right; one state per file (not processed / decoded / in context / failed); select and press one Process; format detected, never declared; click a file to preview it (Messages view for a backup, picture/PDF otherwise); metadata in view; mark a file as part of a unit.
- **2026-09-22 19:39–19:47** — Review walked by clicking: "utterly garbage"; the messages got a 40 px strip, three horizontal scrollbars, filenames truncated to "sms-…", no sizes, no way to tell a whole backup from one thread chunk, a raw Temporal error as the next action, a two-line "More" menu. "The whole process needs to be re-thought."

## What went wrong, in one paragraph

Review descends from the July 20 console, whose brief was *observability*: see every stage, every receipt, every store. That brief became the page's primary content, and step 5 — look at what the machine got out — became a sub-tab under a tab. Each session added a panel; no one reset the page against June 20. Sources (13:00 today) already owns steps 1–4. Review is step 5 and 6 plus the decision: the file's preview, full screen, with gaps flagged and the one or two things he can do next in plain words. Nothing else belongs on it.

## Rule for every screen from now on

Before adding anything to a surface, name which of the six steps it serves. If it serves none, it is not on the surface.
