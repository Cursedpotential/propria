---
title: Conversation analysis pipeline — runbook (manual today, programmatic later)
date: 2026-09-24
status: current method; every step below was run by hand on 2026-09-24 and is the spec for automating it
tags: [runbook, conversations, bouts, catalog, opus, extraction, normalization, chunking, jev-eval, reproducible]
---

# Conversation analysis pipeline — runbook

> _Byline: Claude Code · Opus 5.5 · 2026-09-24. Owner 07:45: "make sure you record what you're doing so it's
> repeatable … we're going to have to make it reproducible and programmatic later. What we're doing now is most of
> the work that needs to be done, just on a bigger scale." Run log: `RUNLOG.md`. Decisions and owner context:
> `docs/handoffs/JEV-EVAL-OPTIONS-2026-09-23.md`._

Each step lists the script, what it reads and writes, and what it becomes when automated (one unit = one future
Temporal Activity, per AGENTS.md "ATOMICITY"). Catalog = PG `casebible` (schema `raw_duck`, container
`fgz1n7useplhk0t91uk7k1aw` on ovh-files). Opus passes run in the ovh-files devbox (`Propria/docs/reference/DEVBOX-ON-OVH-FILES.md`),
work dir `persist/jev-eval`.

## How to run a SQL step

Tracked files only (the case-bible hook blocks scratch code against durable infrastructure). Pipe the tracked file:

```bash
ssh -i ~/.ssh/ovh root@100.91.190.107 "docker exec -i fgz1n7useplhk0t91uk7k1aw psql -U postgres -d casebible -v ON_ERROR_STOP=1 -f -" < <file>.sql
```

Test first: `sed 's/^commit;$/rollback;/' <file>.sql | ssh … -f -`, check the counts, then run for real.

## Steps

| # | Step | Script (repo) | Reads → writes | Future unit |
|---|---|---|---|---|
| 1 | **Register the conversation** (source format, match keys, speaker rule, the two people's identities, custody party, bout prefix) | Consignatio `casebible/tools/msg_extract_registry_20260924.sql` (the `insert … values` rows) | → `msg_conversation_registry_20260924` | config row; no code change per conversation |
| 2 | **Normalize: speaker + duplicate renderings** (speaker by rule, never a model; same side/second/text = one message) | same file | `comm_events_20260918` + registry → `msg_norm_20260924` | normalization Activity |
| 3 | **Chunk into day bouts** (America/Detroit day, new bout after >30 min silence; stable ids `<prefix><year>-b<nnnn>`) | same file | norm → `msg_bouts_20260924`, bridge `msg_bout_messages_20260924` | Go message-window chunker |
| 4 | **Custody party + label protection** | `msg_bouts_custody_party_20260924.sql` | bouts (`custody_party`); labels FK RESTRICT | part of source registration |
| 5 | **File citations** (every message → every source file that holds it) | `msg_files_20260924.sql` | norm + `comm_event_provenance_20260918` → `msg_files_20260924`, view `msg_bout_observations_cited_20260924` | provenance Activity |
| 6 | **Export bouts for a pass** (one JSON line per bout) | Probata `scripts/jev_eval/build_bouts_v2.sql`, `build_bouts_fb2024.sql` (per source; to be replaced by one export over `msg_bouts_20260924`) | catalog → devbox `bouts/*.jsonl` | export Activity |
| 7 | **Opus tone pass** (tone stretches + shifts, same view the owner validated) | `scripts/jev_eval/bouts_tone_opus.py` | bouts jsonl → `raw/bout_tone/<bout>.json` | model Activity, one bout per call |
| 8 | **Opus discovery pass** (conversation-level observations, open categories, child-related flag, no judgment of people) | `scripts/jev_eval/bouts_discover_opus.py` (prompt v3 under owner comparison with v1, v2 and v2.1; `--ids` / `--only` for samples) | bouts jsonl → `raw/bout_discover_v3/<bout>.json` | model Activity |
| 9 | **Load labels into the catalog** | `scripts/jev_eval/export_bout_labels.py` → `\copy raw_duck.msg_bout_labels_stage_20260924` → Consignatio `msg_bout_labels_upsert_20260924.sql` | raw json → `msg_bout_labels_20260924` | loader Activity |
| 10 | **Owner review pages** (tone review; prompt compare) | `scripts/jev_eval/bout_review_page.py`, `compare_discover_page.py`, `pick_review_set.py` | bouts + raw → private artifact with `db` answers | becomes the Workbench components below |
| 11 | **Backup coverage + head/tail check** (which backups a newer original fully contains; does the catalog hold every record) | Consignatio `sms_backup_coverage_20260924.sql`, `sms_backup_headcheck_20260924.sh` (`ssh root@ovh-files 'bash -s' < …`) | catalog + B2 heads/tails → `sms_backup_coverage_20260924`, `sms_backup_headcheck_20260924` | vault hygiene Activity |

Where each output belongs in the platform later: `docs/handoffs/JEV-EVAL-OPTIONS-2026-09-23.md` § "Propagation register".

## Rules the steps depend on (owner rulings, 2026-09-23/24)

- Speaker and direction come from data rules, never a model; unmatched rows stay unassigned and are counted.
- Bouts are the parent unit. Search chunks are message-aligned windows inside a bout; a hit opens its bout.
- The discovery prompt is neutral and narrow: only the chunk, no history, no judging either person, warm acts
  recorded as carefully as conflict. The owner's background is guidance for Claude only, never in a prompt.
- Every record carries a file citation.
- Keep reusable work: results land in catalog staging tables with their platform target recorded.
- No PII mitigation until court documents are being created.

## Surfaces (owner 07:39–07:45: TanStack + Storybook + Glide; charts by type)

| Piece | Library | Where |
|---|---|---|
| ~~Mood strip (bouts on a zoomable date axis, colored by tone)~~ **Corrected 2026-09-24 08:40 (owner: the vis-timeline build did not look like the bout review page):** Mood strip = the bout review page's chart, one row per day, each bout a block sized by its messages and split into tone stretches; sender names come from the data | ~~vis-timeline~~ plain React + CSS (flex rows, no chart library) | Workbench Storybook component `modules/workbench/web/src/platform-ui/conversations/mood-strip.tsx`; fixture from `scripts/jev_eval/mood_strip_fixture.sql` |
| Conversation view (messages, tone, sender, time, file citation, observations) | Glide Data Grid (build on `components/sbv/message-browser-grid.tsx`) | Workbench |
| Summary charts (tone mix by month, categories per person) | Recharts | Workbench |
| Frozen reports | Evidence.dev (`Consignatio/Intake/evidanceio`) | reports over the same catalog tables |

Library decision on record: Advocatio `docs/planning/2026-09-13-advocatio-reconciliation/STACK.md` and
`inputs/current-app/VISUALIZATION-RECOVERY.md` (keep react-calendar-timeline and vis-timeline, pick per interaction);
Propria `docs/reference/chart-visualization-libraries-2026-09-14.md` (library list).

## To automate (in order)

1. One export over `msg_bouts_20260924` replaces the per-source bout SQL (step 6).
2. Steps 2–5 and 9 as Activities, each on its own; step 1 stays a registry row.
3. Steps 7–8 as a model Activity per bout, resumable (the scripts already skip bouts with an `ok` result).
4. The review pages become Workbench components reading the catalog tables.
