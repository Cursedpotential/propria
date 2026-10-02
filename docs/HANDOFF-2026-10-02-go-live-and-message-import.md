# HANDOFF — go-live identity, data-table guards, message import (2026-10-02)

> _Byline: Claude Code · Opus 5.5 · 2026-10-02_
STATUS: PARTIAL
BUILD_STATUS: UNKNOWN — the engine's `go test ./...` passed at `e6c36f88` (the `tsnet-front` binary link ran out of memory on the desktop, which is a desktop limit, not a code failure). The Workbench and Python suites were not run this session.

## Verified-live state (do not re-derive)

| Thing | State |
|---|---|
| Live case (platform DB, ovh-files container `probata-db-w10gg3an43jvry4y79n6sxi1-122959880896`) | Minted 2026-10-02 02:20:56 EDT. Matter `01a0f751-e07b-75cc-9ad5-63ad9449a8ba` "Salem v. Kinzel (custody)". Court case `01a0f751-e07b-76a1-a738-eb3e3aa3e68c` (2025-53985-DC, Judge Dawn M. Weier). Matthew S. Salem `01a0f751-e07b-76b6-afcb-63acfbba373e` (user/plaintiff); Katrina Kinzel `01a0f751-e07b-76c7-8c0f-65692ad656b8` (co_parent/defendant). Partition `live`; one owner receipt (payload `sql/bootstrap/case_registry_live_identity_20261001.json`, commit `5643178c`, sha256 `b39561e9…`, 4,532 B). |
| TEST purge | Done in the same transaction (`ops.reset_test_data` plus removal of the DEV identity). The format catalog was kept. |
| Append-only guards | 18 triggers on data tables removed (owner: nothing is immutable until evidence). The 11 on audit trails stay: `registry.identity_change`, `identifier_triage`, `analysis.review_decision`, `task_event`, `ops.tool_call_ledger`, `public.change_log`, `session_summaries`, and evidence. |
| Aliases | `registry.entity_alias` is a plain editable table; every change is logged in `identity_change`. Seeded with 39 confirmed rows (Matt 27 phones + 5 names, Katrina 4 phones + 3 names). The Case Bible's `raw_duck.msg_identity_20260924` is now an fdw view over the registry. |
| Third-party option A | Live: `third_party_conversation.source_version_id`, INSERT grants plus a column UPDATE (started_at/ended_at/message_count), no DELETE. |
| Engine probe | Requires the real receipt in every mode; the dev flag no longer changes identity (`e6c36f88`). proffer-starter and proffer-worker redeployed 06:49Z with 0 restarts. |
| Workbench | `PROFFER_REAL_MATTER_ID` / `PROFFER_REAL_COURT_CASE_ID` set (commit `76620f30`, deploy `bmdn793cdv7mzndechehiu1x`, container up 12:12:54Z). The live-mode lookup resolves the case from inside the container. Not verified through the browser: the route sits behind Authentik and answers 403 on the bare port. |
| Temporal | History archival on for namespace `default` → `/data/probata/volumes/temporal-archive` (bind mount in `deploy/temporal/compose.temporal.yaml`). Hourly add-only copy to `b2:salem-data/propria/temporal-archive` (systemd `temporal-archive-sync.timer`); the first run copied 4 files. Live retention is 30 days. |
| Desktop mounts | X: (B2), V:, Y: (R2) via task `RcloneR2DualMounts`. A 15-minute self-heal trigger was added. |
| Import | 87 source versions, **0 preview decisions, 0 working.message, 0 third_party_message** (08:05 EDT). First preview `PLATHR6PyU4fhZm6zCZh9W7jKiji1NRH` waits for the owner. |

## Findings / work done

- **Flow analysis**, in `modules/Probata/probata/docs/reference/2026-10-02-message-flow-as-built.md`:
  - Weaviate is written first and nothing updates it afterwards; the change-feed outbox has triggers but no consumer.
  - The 366,912 `MsgEvents20260918` objects all come from the Case Bible scripts, so Proffer runs add a second copy.
  - No CocoIndex, LlamaIndex or LangGraph sits in the message flow; the last two are decided (D-143/144) but not built.
  - The timeline projection's Python worker is not deployed.
  - Nothing writes to SurrealDB.
  - A Parquet lake exists for the catalog only (`consignatio/_system/lake/2026-09-27/`).
  - AI chats are not kept apart from messages.
- **Overnight-import agent (stopped at the session end).** Commits on main:
  - `57baffc9`: auto-approve clean runs;
  - `d1cb113a`: Facebook Messenger JSON parser;
  - `ade084fa`/`8879fdb3`: one-message chunks are ndjson; re-runs skip active runs;
  - `9e64c99c`: a signal applies clean_checks to a parked run.
  - It loaded the casevault listing into the catalog and placed the first file at `casevault/SourceCorpus/messaging/sms-backup-restore/8102689630/sms-2024-11-24.xml` (105 thread chunks).
- **Codex audit** (`~/.codex/.chatgpt-projects/…/reports/2026-10-02-0755-Propria-status-refresh.md`) flagged:
  - the placement script accepts equal sizes when a SHA-1 is missing (weak verification);
  - 19 failed `ops.workflow_run` rows.

## UNRESOLVED (mandatory)

- **First real import not committed.** WHY: it waits for the owner's approval of preview `PLATHR6…`. Agent `overnight-import-2` is running and owns the rest: hash-check fix, the 19 failed runs, the clean_checks signal to the other 104 chunks, the remaining placements, the SMS, calls and Facebook batches, then salvaging the 12 unclosed 2026 SMS backups.
- **dev/live rename.** The Workbench code and UI still say TEST/REAL. WHY: the case-identity agent stopped mid-task. APPROACH: one rename across the BFF, engine and UI, with no compatibility layer.
- **Workbench live mode is not browser-verified.** WHY: Authentik sits in front. SHORTCOMING: only an in-container lookup proved it.
- **The Facebook copy (544 objects) started by the first agent** has no verification receipt yet; `overnight-import-2` checks it.
- **Flow decisions A–F** (`2026-10-02-message-flow-as-built.md`): unanswered.

## Pending owner decisions

- **A. Keeping Weaviate, Surreal and Neo4j in step after a commit or edit.** Recommended: one change-feed worker over the existing outbox. Alternatives: re-run the search stage, or go back to Postgres-first.
- **B. Case Bible vs Proffer objects in `MsgEvents`.** Recommended: keep both, filter by origin, and retire the Case Bible copy of each file once Proffer has imported it.
- **C. Extractor.** Recommended: Go + kimi-k3 now; LlamaIndex/LangGraph later per D-144.
- **D. Timeline projection.** Recommended: deploy the existing Python `evidence-pipeline` worker.
- **E. Entities, events and timeline to the catalog.** Recommended: Parquet through `lake_publish` as one Temporal activity into casevault `DerivedKnowledge`.
- **F. AI chats kept apart.** Recommended: they skip the first-party commit and never get the promote-to-evidence action.

## Next steps (work in order)

1. The owner approves preview `PLATHR6PyU4fhZm6zCZh9W7jKiji1NRH` in Workbench Review (live mode).
2. `overnight-import-2` reads that run back, signals clean_checks to the other 104 chunks, and runs the batches. Watch its milestones.
3. The dev/live rename (new agent; Workbench BFF + UI + engine).
4. Owner decisions A–F, then build the change feed, Parquet export and send-to-Surreal.
5. Entity and event extraction on the imported messages (Review → Extract entities), with the NIM key checked on the worker.

## Owner working-style contract

- Structured replies: bullets, labeled blocks, white space, answer first. KISS in everything.
- Confirm before design changes; implement decided things without asking again; check the result of a command the owner ran before giving another.
- Never hard-delete (quarantine); audit trails stay locked; nothing is immutable until evidence.
- Modes are "dev" and "live". The home root is `b2:salem-data/consignatio/casevault/`.
- Same-bucket moves are server-side; cloud-to-cloud transfers run on the VPS, never through the desktop.
- Byline every artifact; verify before claiming done.
