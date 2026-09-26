# Category 7 — Workflow & automation engine

> _Byline: Grok · grok-4.6 · 2026-08-18_
> Source: `Legal-desktop/artifacts (3)/HANDOFF — Category 7  Workflow & Automation Engine.md`

## Scheduler (question 1)

**APScheduler inside `legal-api` (AsyncIOScheduler, jobstore =
~~Postgres `legal_core.automation_job`~~).**
*(corrected 2026-08-18: local runtime is MemoryJobStore +
`data/workspace` JSONL. `legal_core.automation_job` is the later PG
store — Type 1 HOLD, schema in `sql/0002_legal_automations.sql`, not
applied.)*

| Option | Verdict |
|---|---|
| APScheduler | Same language as FastAPI/Agno; cron + date + interval; no Redis. [docs](https://apscheduler.readthedocs.io/) |
| node-cron | Dies with the Node process; wrong side of the stack |
| BullMQ + Redis | Real queue, new dependency. Overkill for one user |
| n8n cron for *everything* | Couples internal playbooks to n8n uptime and a second UI |

legal-terminal’s `automationScheduler.ts` is a **client-side timer**
and is explicitly rejected (deep analysis §C.1; FEATURES.md gap).

n8n (already on the fleet) owns **outbound** only: email, SMS, push,
third-party webhooks.

## n8n trigger pattern (question 2)

Legal OS POSTs to an n8n **webhook** node with a shared secret
header (`X-Legal-Webhook-Secret`). n8n never calls back into
legal-api without HMAC.

Worked example:

```
event email_received (or owner-tagged FOC mail)
  → legal-api event bus
  → if playbook step == "notify"
       HTTP POST https://n8n.<svc>/webhook/legal-notify
       { "matter": "…", "title": "FOC notice", "when": "…" }
  → n8n sends the push/email
```

n8n webhook docs: [docs.n8n.io webhook](https://docs.n8n.io/integrations/builtin/core-nodes/n8n-nodes-base.webhook/).

## Event bus (question 3)

Hand-roll a tiny in-process bus on the FastAPI process (publish to
a table + notify), same spirit as legal-terminal `eventBus.ts` but
**server-side**. No Redis pub/sub.

Events from the catalog, rescoped: `job_complete`,
`document_upload` (work product, not raw evidence),
`package_imported`, `deadline_changed`, `app_open` (optional).
`email_received` only after a real IMAP/n8n inbound (TRIG rescope).

## Sequence runner (question 4)

Hand-rolled sequential runner: list of registered Legal OS tools,
per-step pass/fail, test-run tab. **No Temporal / Prefect / Dagster.**
Scale is one user and a handful of playbooks (opposing filing
arrives; FOC hearing prep). Anything that outgrows this goes to n8n.

Playbooks seed from legal-mcp-toolkit SKILL workflows **and**
custody-packet checklists, not NDA/MSA sequences.

## Persistence (question 5)

Yes — `legal_core` / `legal_audit` on the shared PG18 cluster
(Cat 1). Tables: `playbook`, `playbook_step`, `automation`,
`automation_run`.

## Module tree

```
api/legal_workspace/services/automation/
  scheduler.py
  bus.py
  runner.py
  n8n_webhook.py                 # HOLD — not built
sql/0002_legal_automations.sql   # later, not applied
```

## Non-goals

No new integration platform. No heavyweight orchestrator. No
client-side-only scheduling.
