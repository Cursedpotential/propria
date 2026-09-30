# Family Law Toolkit capability checklist — 2026-09-27

> _Byline: Claude Code · Opus 5.5 · 2026-09-27_

This checklist runs the 383-entry inventory
(`../inputs/stack/toolkit-capabilities.json`, stable ids from `A-capability-port-map.csv`)
against what is live today. Every row is in
[`TOOLKIT-CAPABILITY-CHECKLIST-2026-09-27.csv`](TOOLKIT-CAPABILITY-CHECKLIST-2026-09-27.csv)
with its status and evidence.

## Status definitions

- **done**: a live call succeeded through the deployed `family-court-console`
  (Coolify `sokv65ibdq2y8xdaqmd6p4rq`, commit `d8c6350a`), probed over MCP inside its
  container on 2026-09-27.
- **callable**: registered or stored live, but not exercised by the probe. This covers
  write tools, which were not called against production data, and store rows that open
  through `case_record`.
- **missing**: no hosted equivalent exists.

## Totals

| Status | Rows |
|---|---|
| done | 38 |
| callable | 242 |
| missing | 103 |

## By kind

| Kind | done | callable | missing | Notes |
|---|---|---|---|---|
| external_source_record | | 193 | | All 193 ledger ids exist as `source:<id>` in surreal-case. |
| event_context_pack | | 17 | | All 17 exist as `reference:<id>`. |
| mcp_tool | 32 | 3 | | The 3 callable rows are `case_put`, `case_export` and `case_import` (writes). Codex 2.0.0 duplicates count against the same live tool names. |
| mcp_widget | | 16 | | The console serves all 8 `ui://` widgets to MCP App hosts. There is no hosted web host. |
| mcp_resource | 4 | | | |
| mcp_prompt | | 4 | | |
| mcp_server | 2 | | 1 | CourtListener is not hosted. |
| resource_pack | | 9 | | Content ships in the console image. There is no browsing surface. |
| sidecar_route, runtime_adapter | | | 18 | FL-MCP desktop sidecar, not hosted. |
| skill, agent, command, procedure | | | 66 | Instruction artifacts with no hosted runner. |
| utility/maintenance script, hook, assessment | | | 18 | |

## Shared records (first acceptance slice)

- **Contract.** The contract is `propria.legal-record.v1`. The store computes the version
  as `sha256` over SurrealDB's own sorted string form of the record.
- **Toolkit side.** The toolkit exposes it as the read-only MCP tool `case_record`.
- **Advocatio side.** Advocatio reads it at `/v1/toolkit/records/{ref}` and on the
  `/toolkit` page. It runs the same query text (`services/family_court_toolkit.py`).
- **Live proof on the toolkit side.** `case_record source:00-how-to-use-references` returned
  version `sha256:1e8346004048b0b33337e4be912a5c08c85f70c8408fa372fd7751c2196ca28d`, the
  same value the store computed directly before the deploy.

**Not yet proven:**

- **Advocatio needs credentials.** It needs a read-only (VIEWER) user on `fct/case`, plus
  `FAMILY_COURT_TOOLKIT_STORE_URL`, `_USER` and `_PASS` on the Coolify app `legal-workspace`
  (`gvghzivfmctev8dloetfssnj`). Until the owner provisions them, `/toolkit` reports
  "not configured". Provisioning the user was refused by the agent's permission classifier.
- **No cheat sheet is in the store.** The custody-guide `CHEAT-SHEET.md` exists only as a
  content file. Loading it as `reference:cheat-sheet-custody-guide` was refused by the
  classifier as a production write.
- **No case documents exist.** surreal-case has 0 filings, drafts, exhibits and notes, so
  the case-document slice has nothing real to open. The synthetic add-and-remove proof also
  needs a production write.
- **No hosted web surface.** The mobile web host (`/`, `/api/*` on the console) was refused
  by the classifier.
