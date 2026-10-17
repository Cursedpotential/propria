# Propria Knowledge Reconciler — Cloudflare trigger and HITL alerts

**Status:** source implementation / **NOT DEPLOYED** (2026-10-17). This is deliberately a small edge coordinator, **not** a new memory store, vector index, approval database, or rule engine. It uses the existing Propria Workbench Activity and canonical human-review workflow.

## Existing ownership contract

- **Sources of truth:** Memsearch Markdown journals, Engram, CNF, Propria Docstore, governed PostgreSQL records and Case Bible documents, each with its own native writer. Supersession must never delete an immutable original or silently rewrite third-party testimony/evidence.
- **Discovery / reconciliation:** Propria Search, using each real adapter and preserving exact source identity. Use DuckDB for bounded joins and filtering or `jq` for focused JSON selection at the backend presentation boundary. Do **not** remove source evidence or raw tool outputs to make displays shorter.
- **Approval:** the **existing PostgreSQL-backed review decision** (not a new Cloudflare D1/KV state machine).
- **Long human waits, retry and exact activity receipts:** Temporal.
- **Human-facing inbox:** existing Propria Workbench Activity route (`/activity`), with an additional read-only *Knowledge review* panel.
- **Alert delivery:** existing n8n. It sends only safe links, severity, scope and review references. n8n **does not approve** a decision.
- **Cloudflare Worker:** daily/on-demand/source-event trigger, bounded ingestion of **persisted review-case receipts**, and notification dispatch. No capability to approve, archive or edit a source.

The current implementation does not claim that the August `context` conflict-review migration is live; its old design receipt explicitly described an unverified local-only database migration. Do not bind the Worker to that schema until the **current live** canonical review API, owner identity and authorization are established. In particular, do not use the old Agno HITL system: Agno AgentOS was retired in platform decision D-107.

## Entrypoints

| Trigger | Method | Contract |
|---|---|---|
| Daily sweep | Cloudflare Cron (`0 12 * * *`, 12:00 UTC) | Submits a **reference-only** scheduled reconciliation task to the configured backend |
| On demand | `POST /runs` | `{"scope":"propria"}`, optional valid `"source"` |
| On change | `POST /events` | `{"type":"memory.changed","source":"memsearch","record_ref":"...","revision":"...","scope":"propria"}` |
| Process check | `GET /health` | Public minimal liveness only; no claims about backend/HITL connectivity |

Both POST endpoints require `Authorization: Bearer <RECONCILER_TRIGGER_TOKEN>`. Event types: `memory.created|changed|retired`, `document.created|changed`, `record.created|changed`. Sources: `memsearch`, `engram`, `cnf`, `docstore`, `propria-search`, `superindex`, `casebible`, `workbench`, `ccc`.

Events carry references and revisions, **not raw personal/case contents**. The backend must retrieve the referenced version under the correct authorization and inspect source records there.

## Deployment configuration

Build in `modules/Probata/probata/workers/knowledge-reconciler`. Run `npm test` (Node's built-in test runner), `npm run check`, then build/deploy using Wrangler **after** the backend and n8n routes are verified.

Configure these Worker secrets (no values in git):

| Variable | Purpose |
|---|---|
| `RECONCILER_TRIGGER_TOKEN` | Authorize caller-generated events/on-demand runs |
| `RECONCILER_BACKEND_URL` | Fixed HTTPS URL of the existing Propria reconciliation engine route |
| `RECONCILER_BACKEND_TOKEN` | Scoped service token for reconciliation submission |
| `N8N_ALERT_WEBHOOK_URL` | Fixed HTTPS n8n workflow endpoint which accepts notifications only |
| `N8N_ALERT_TOKEN` | Scoped notification token |

Cloudflare Workers do **not** inherently join the user's Tailscale network. Use an existing authenticated HTTPS ingress or Cloudflare Access/Tunnel service-token route for backend calls. Do not configure a raw `100.x` tailnet address in the Worker or enable Tailscale Funnel.

The backend must return a JSON object with this contract **only after persisting its HITL queue entries**:

```json
{
  "schema": "propria-reconciliation-result/v1",
  "run_id": "durable-run-id",
  "state": "review_pending",
  "scanned": 100,
  "findings": 2,
  "review_queue": [
    {
      "review_ref": "canonical-review-ref",
      "status": "queued",
      "severity": "high",
      "kind": "contradictory_owner_decision",
      "scope": "propria",
      "newly_created": true
    }
  ]
}
```

**No persisted `review_ref` = no alert.** This is the guard against inventing an owner's inbox item or reporting an analysis candidate as already queued. Duplicate scans must return `newly_created:false` for the same unresolved case (backend idempotency, pinned to source IDs/versions). The Worker sends immediate notifications only for **new high/critical** cases; medium/low issues remain visible in the authoritative inbox and should be digestible there, rather than firing an alert per extracted row. Worker returns explicit `blocked`, `review_queued_alert_blocked` or `review_queued_alert_failed` states when downstream systems cannot confirm success.

The n8n webhook must acknowledge each alert with `{"accepted":true,"review_ref":"canonical-review-ref"}`. It can deliver an email/chat/other owner-selected notification; **no external notification channel is configured by this source change**. Do not include source text, evidence excerpts, tool-call bodies, transcripts, credentials or model prompts in an alert.

## Workbench read-only Activity inbox

The existing Workbench API gains `GET /api/knowledge-review/inbox`, authenticated by its current Authentik/Tailscale middleware. Add **server-side** environment variables:

- `KNOWLEDGE_REVIEW_QUEUE_URL`: fixed route returning the canonical review queue from the already-owned backend.
- `KNOWLEDGE_REVIEW_QUEUE_TOKEN`: backend query-only service token.

The backend queue response must be `{"schema":"propria-knowledge-review-queue/v1","total":N,"items":[...],"next_cursor":null}`. Each bounded item contains `review_ref`, `kind`, `severity`, `status` (`queued` or `deferred`), `scope`, `summary`, `source_count`, and optionally a verified same-origin `review_href` path (`/review...` or `/activity...`). No externally supplied URL is accepted in a review link. This read-only panel does not implement approval: that requires the live validated owner decision endpoint, exact expected revision, recorded identity, rationale and receipt.

**Until a real backend provides the contract:** the Activity panel explicitly reports that it is not connected. It cannot display fake zero findings, approve a review, or claim that alert delivery is working.

## Decision rules — no accidental policy promotion

1. **Verified same-record version replacement**: detect from explicit `supersedes` refs/revisions; propose archival of prior active view. Historical bytes remain intact.
2. **Exact duplicates**: collapse only the **presented** results; all source occurrences remain traceable. Any durable retirement must use the owner's existing approved method.
3. **Conflicting statements / ungrounded recency**: generate an actionable review candidate with both sources and versions. **Never auto-pick a winner.**
4. **Important owner corrections / approved policy changes**: require actual owner approval and explicit scope. A correction to one memory does not become a global system rule.
5. **As-lived/hindsight**: never retroactively expose a later correction or foreshadowing marker inside an earlier knowledge-horizon view.
6. **Alert volume**: present the actionable subset, not every unreviewed extraction row. Deduplicate by underlying case ID and source versions, with daily review rather than an email per record.
7. **Output hygiene**: sources retain verbatim input. Returned result summaries should be bounded, DuckDB/`jq`-filtered when running on the backend, grouped by source/decision and free of greetings, redundant tool calls, progress narration and repeated fragments. The Worker uses bounded structured JSON because it is an edge HTTP dispatcher, not a host with a DuckDB/`jq` subprocess.

## Acceptance before production

- [ ] Verify the **current** authoritative PostgreSQL HITL review queue/decision API and permissions (older schema receipts are insufficient).
- [ ] Implement and test the reconciliation backend (native Engram/Memsearch/CNF/Docstore/Case Bible adapters); it must persist canonical review-case references and not accept AI text alone as authority.
- [ ] Wire native change events or CDC/outbox to `POST /events`; validate authorization and idempotency.
- [ ] Implement n8n notification workflow with authenticated correlated acknowledgements and owner-selected destination.
- [ ] Verify Workbench's queue endpoint, same-origin review deep links, and actual owner approval/reject/defer/modify controls through the original review authority.
- [ ] Run Node/Python unit tests, then safe integration tests with synthetic conflicting memories and an actual authenticated user in the Workbench.
- [ ] Verify action receipt, alert delivery, duplicate suppression, deferred state and no memory/evidence deletion.
- [ ] Only then deploy Worker and Workbench deliberately; pushes are not deployment approval.

Source changes are intentionally reviewable without altering existing documents, source stores, configured secrets, or live Workbench state.
