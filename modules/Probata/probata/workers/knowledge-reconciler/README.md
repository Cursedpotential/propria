# Propria Knowledge Reconciler — n8n-led workflow

**Implementation source committed, NOT LIVE, as of 2026-10-17.**

The owner selected a deliberately simple structure: **Cloudflare Worker triggers ONE n8n workflow; n8n owns the entire end-to-end reconciliation; existing Propria PostgreSQL stores human reviews; Workbench Activity is the human interface.** Temporal is NOT required on the ordinary path. No new independent index, evidence queue, or agent state machine.

## End-to-end diagram

    Cloudflare scheduled/manual/change event
       -> POST n8n /webhook/propria/knowledge/reconcile
          -> Validate source identity, scope, revision
          -> One Platform source-snapshot request (fan-in across native source adapters)
          -> n8n Code: compare within subject, duplicates, explicit supersession, expired actives
          -> One canonical PostgreSQL review-batch write
          -> Verify persisted review refs and duplicate suppression
          -> IF new high/critical: owner email via n8n SMTP credential
          -> Respond with compact verified receipt
       -> Workbench Activity: Knowledge review card -> inspect -> approve plan/reject/defer
          -> Native memory/document writer stage remains separately gated

The n8n workflow is an actual editable 11-node workflow, not an n8n wrapper around dozens of atomic parser calls. It is stored at:

    modules/Probata/probata/deploy/docker/n8n/workflows/knowledge-reconciler/wf-knowledge-reconcile.json

n8n uses real Webhook, Code, HTTP Request, IF, Send Email and Respond nodes. The same workflow is called from all triggers. SMTP errors are surfaced without erasing a queued review. A repeated fingerprint returns newly_created=false, so it cannot generate another urgent alert.

## Where each part is implemented

| Part | Source | Responsibility |
|---|---|---|
| Cloudflare Worker | workers/knowledge-reconciler/src/index.js | Daily cron, on-demand POST /runs, signed source-change POST /events |
| n8n whole workflow | deploy/docker/n8n/workflows/knowledge-reconciler/wf-knowledge-reconcile.json | Retrieve, compare, propose, persist, notify and return |
| Platform API | server/api/knowledge_reconcile_routes.py | Authorized source snapshots and canonical review persistence |
| PostgreSQL | sql/drafts/0101_knowledge_reconciliation_review.sql | Idempotent versioned review case and append-only human decision |
| Workbench API | modules/workbench/api/app/runtime/knowledge_review.py | Authenticated review read and owner-guarded decision proxy |
| Workbench UI | modules/workbench/web/src/components/activity/knowledge-review-*.tsx | Same Activity inbox, review card, version-pinned source references, decision |
| Native source adapters | Existing source-specific readers, configured in Platform env | Search original stores; report missing/offline sources honestly |

The current source adapter interface is intentionally not allowed to improvise credentials, read directly from arbitrary host paths, or silently substitute one memory store for another. Native adapters must implement the fixed source-snapshot contract before going live. In particular, Engram, local CNF, and desktop CCC are NOT made reachable by committing this Worker. Desktop adapters need the authenticated Tailscale bridge and the correct device/account. Hosted Memsearch and Docstore need their authorized native adapters; search availability alone is insufficient for a full daily inventory.

## Security and authority

- **One native source of truth per domain.** Memsearch Markdown journals; Engram conversations; CNF local JSON; Docstore revisions; Superindex/Case Bible source catalogs; canonical Platform PostgreSQL. None are merged or rewritten into a second replica.
- **No unreviewed global rules.** A per-record correction never becomes a platform-wide policy without explicit owner approval and scope.
- **HITL in existing Workbench Activity.** Decisions are version-checked and the reviewer comes from authenticated Workbench request state. n8n cannot approve the proposal or submit an owner decision.
- **An approval is not application.** Approve_plan means the human accepted a proposed change; source_applied=false. A separate authorized native writer must apply the approved revision and produce a receipt before status is described as cleaned.
- **Source originals and evidence stay immutable.** References and version hashes are preserved; historical content stays queryable for hindsight, with as-lived horizon controls unaffected.
- **Compact results, not destructive source filtering.** In the n8n Code node, duplicate greetings, superficial tool-call lines and embedded system noise are removed only from the comparison projection. The original memory text is not modified. Native source adapters should use their installed DuckDB or jq-based presentation packs when available, but a Cloudflare Worker does not execute these host binaries.
- **Record-only events.** Worker triggers contain IDs, scope and revisions, not raw case data. Cloudflare cron is at 12:00 UTC daily (8 AM EDT / 7 AM EST); adjust only if the owner changes cadence.
- **No credential values in git.** MCP, PostgreSQL, n8n and SMTP secrets live in their respective provider/host credential systems. n8n Community blocks node access to environment variables; the workflow uses fixed nonsecret tailnet API URLs and managed credential references.

## What must be connected for activation

1. Apply the audited, additive SQL from sql/drafts/0101_knowledge_reconciliation_review.sql to the **existing Platform PostgreSQL** after checking the current schema, grants and rollback path. The older context-review foundation is not assumed live.
2. Publish the Platform API route on the correct existing service. Confirm the n8n machine identity can invoke POST /v1/knowledge-reconcile/snapshot and POST /v1/knowledge-reconcile/reviews, but not a human decision. The checked-in workflow targets the established nonpublic Platform tailnet address at 100.72.169.40:8000; verify reachability from the running n8n container before activation.
3. Configure Platform KNOWLEDGE_RECONCILE_SOURCES_JSON as a JSON list of authorized adapters with store, fixed URL, and token_file; mount the token files read-only. Each native adapter returns a bounded array of record IDs, revisions, subject keys, text for comparison, explicit supersession metadata, and whether there are more results. Missing adapters remain reported unavailable, never silently skipped. Event producers need to POST the corresponding reference-only change event after a committed update.
4. Import the checked-in workflow in the **existing n8n** instance. Keep active=false until URL, input schema, credentials and host routes are verified. Bind: N8N_KNOWLEDGE_TRIGGER (Webhook Header Auth, matching Worker N8N_RECONCILER_TOKEN), PROPRIA_KNOWLEDGE_SOURCE_READER, PROPRIA_KNOWLEDGE_REVIEW_WRITER (Platform service bearer), and PROPRIA_OWNER_SMTP (owner-authorized SMTP account). Replace the two REPLACE_WITH_* notification addresses in the Email node with verified sender/recipient details. Do not reuse a case/court SMTP identity without owner authorization.
5. Configure **Workbench** `KNOWLEDGE_REVIEW_OWNER_UIDS` with the exact Authentik subject UID(s) actually authorized to make decisions; unset denies decisions. Configure both Workbench and Platform `KNOWLEDGE_REVIEW_DECISION_TOKEN_FILE` with a managed matching reviewer-only capability secret, never exposed to n8n or browser JavaScript. Configure **Platform only** `KNOWLEDGE_REVIEW_DECISION_DB_DSN_FILE` with a mounted DSN for a separate DB role authorized by the administrator to assume the `knowledge_review_adjudicator` NOLOGIN capability. Do **not** grant that capability to `platform_runtime` or n8n. Owner decisions require exact expected revision, rationale and receipt; deploy Workbench API/UI together.
6. Set Cloudflare Worker secrets N8N_RECONCILER_WEBHOOK_URL, N8N_RECONCILER_TOKEN, RECONCILER_TRIGGER_TOKEN. The Worker calls the n8n workflow over HTTPS. Do not expose the n8n container's private tailnet IP directly from Cloudflare; use its already-authenticated public HTTPS ingress.
7. After live tests with synthetic data and explicit owner-approved sources, activate n8n webhook and deploy Worker; verify one recorded run, one persisted review ID, one notification acknowledgement, and one owner decision. Test duplicate delivery, offline CNF/CCC and no unexpected source writes.

## Contracts

POST /v1/knowledge-reconcile/snapshot accepts the validated Cloudflare/n8n trigger and returns:

    {
      "schema":"propria-knowledge-snapshot/v1",
      "coverage":[{"store":"memsearch","state":"queried","reason":""}],
      "records":[{
        "store":"memsearch","id":"stable-native-id","revision":"42",
        "scope":"propria","subject_key":"tool availability",
        "text":"the bounded source claim","state":"active",
        "source_hash":"sha256 value or source-native digest",
        "supersedes":["memsearch:earlier-record"]
      }]
    }

The native reader is capped at 100 per store; a truncated cursor must be explicitly marked partial. A zero-match result from one source is not proof all historical records are absent. POST /v1/knowledge-reconcile/reviews accepts the bounded batch of n8n proposals (no free-form automatic archival) and returns:

    {
      "schema":"propria-knowledge-review-write/v1",
      "run_id":"n8n-knowledge-123",
      "review_queue":[{
        "review_ref":"verified-uuid","status":"queued",
        "severity":"high","kind":"possible_topic_conflict",
        "newly_created":true
      }],
      "source_mutations":0
    }

n8n returns a separate compact workflow receipt with status, run_id, review_queue, coverage and alerts. The Cloudflare Worker trusts only that contract and reports blocked on a mismatch.

The Workbench GET /api/knowledge-review/inbox uses its existing authenticated Platform API credential and displays source-ref metadata. GET /api/knowledge-review/{uuid} loads the original revision pins; POST /decision records approve_plan, reject, or defer with a nonblank reason and expected version.

**One important limitation:** the current PR does not yet include native-source apply/retire adapters. Approval is recorded, but the proposed source mutation is held pending owner-authorized application. The code does not automatically archive or rewrite memories simply because a classifier called them stale.

## Verification

Run from the repo checkout, using the installed supported runtimes:

    cd modules/Probata/probata/workers/knowledge-reconciler
    npm test
    npm run check

    node ../../deploy/docker/n8n/workflows/knowledge-reconciler/test-workflow.mjs

    cd ../..
    python -m pytest tests/test_knowledge_reconcile_routes.py
    python -m pytest modules/workbench/api/tests/test_knowledge_review.py

Do not claim end-to-end success from unit tests, Coolify healthy state, or an n8n workflow definition. An actual signed HTTP request must traverse Worker -> n8n -> Platform database -> Workbench authenticated review and result in the correct human-visible review item. Native-source writes need their own separately checked receipt.

This source branch is a reviewable implementation candidate; it is not a request to auto-create new rules, delete memories, or deploy to production untested.
