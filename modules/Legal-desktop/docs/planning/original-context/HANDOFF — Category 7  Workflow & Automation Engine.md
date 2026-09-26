# HANDOFF — Legal OS Category 7: Workflow & Automation Engine — Implementation Research

Status: rough feature set agreed with owner (2026-08-17); this handoff requests in-depth research to scaffold the actual implementation. Research only — no code, no scheduler/service activation.

## Context

Legal OS is a single-user, single-case, self-represented-litigant practice-management app, sibling package to "Agno MCP Platform" in one monorepo, deployed on the owner's own VPS. Agno's existing `compose.yaml` already runs an **n8n** service — this is a load-bearing fact for this category's architecture, not a minor detail.

## Feature set agreed (do not re-litigate — build research around this)

1. **Ownership boundary: Legal OS's own lightweight engine vs. n8n.** Legal OS builds and runs its own small sequential tool-call runner for playbooks composed entirely of *Legal OS's own tools* (e.g., research → analyze → draft sequences, matching the WKFL panel's original browse/builder concept). **Anything that needs to reach outside Legal OS** — sending a text/email, posting to an external service, hitting a third-party webhook — **delegates to the existing n8n instance** rather than Legal OS growing its own integration platform. Do not scaffold new outbound-integration code in Legal OS for things n8n already does well.
2. **Real server-side scheduling — this is a gap-fix, not a nice-to-have.** legal-terminal's own docs admit its automation scheduler is a client-side JS timer that only runs "while the tab is open." Legal OS must run scheduling as a genuine backend service so scheduled/event-triggered automations fire regardless of whether the UI is open. Since the whole app runs on the owner's own VPS, there's no infrastructure obstacle to this — it should just be built correctly from the start.
3. **Automation types carried over from the original catalog:** daily / weekly / once / event-triggered (job_complete, document_upload, contract_selected, email_received, app_open), automation list with inline enable/disable + last-run status, run-now manual trigger, event payload filters (e.g. only fire for a specific filename pattern or category).
4. **Workflow Builder carried over:** browse pre-built playbooks with visible tool-call sequences, a builder tab for composing custom sequences from a live tool catalog, test-run with per-step pass/fail results.

## Research questions to answer (with citations/links)

1. **Server-side scheduler runtime and library.** Compare: (a) **APScheduler** (Python, matches Agno's backend language) vs. (b) **node-cron** or **BullMQ** (Node, matches the Vercel AI SDK-adjacent frontend stack, BullMQ specifically adds a real job queue with retries/persistence via Redis, which node-cron alone doesn't) vs. (c) **delegating all time-based triggering to n8n's own cron-trigger nodes**, with Legal OS only handling the purely internal event-based triggers via its own lightweight in-process event bus. Recommend one, with the tradeoff on operational simplicity vs. capability stated explicitly (a job queue like BullMQ adds Redis as a new dependency — is that worth it for a single-user app's automation volume, or is it overkill).
2. **n8n integration mechanics.** Research how Legal OS should trigger an n8n workflow from its own automation/event system — n8n's webhook-trigger nodes are the obvious mechanism, but confirm current best practice (authentication on the webhook call, payload shape conventions) rather than assuming a bare HTTP POST is sufficient.
3. **In-process event bus for internal triggers.** Confirm whether a full pub/sub library is warranted (e.g. Node's built-in `EventEmitter`, or Python equivalents) or whether this is simple enough to hand-roll, matching legal-terminal's own tiny `eventBus.ts` pattern in spirit but moved server-side.
4. **Tool-call sequence runner for the Workflow Builder.** Confirm a hand-rolled sequential runner (array of steps, each calling a registered tool, collecting per-step results) remains the right call for Legal OS's own internal playbooks, given the scale is "one user, a handful of playbooks" — explicitly rule out adopting a heavier orchestration engine (Temporal, Prefect, Dagster) for this specific piece, and state why in the report (likely: operational overhead not justified at this scale, especially with n8n already available for anything that outgrows this).
5. **Persistence for automation/workflow definitions and run history.** Confirm these live in the `legal_os` Postgres schema per the Category 1 decision (automation definitions, schedules, run logs) rather than needing their own store.

## Deliverable

A single markdown report, saved to the workspace, with:
- A clear scheduler/runtime recommendation (question 1) with tradeoffs stated explicitly
- A concrete n8n-trigger integration pattern (question 2), with a worked example (e.g., "email_received event → n8n webhook → n8n sends a push notification")
- One recommendation per remaining research question, each with source citations where applicable
- A proposed module structure sketch (directory tree, not full code) for the scheduler/event-bus/workflow-runner pieces
- Explicit non-goals restated: no new integration platform inside Legal OS (use n8n), no heavyweight workflow-orchestration engine for internal playbooks, no client-side-only scheduling
