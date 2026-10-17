/**
 * Propria Knowledge Reconciler: schedule/manual/change triggers and human-review alert relay.
 *
 * Inputs: authenticated reference-only events, daily Cron Trigger, approved backend responses.
 * Outputs: compact run receipts, pending-review alerts referencing existing Workbench cases.
 * Side effects: triggers an existing reconciliation backend and n8n notifications. Never writes
 * memories, documents, rules, PostgreSQL decisions or evidence. Choose this rather than
 * introducing another database/review queue or automatically deciding ambiguous conflicts.
 *
 * Backend contract: POST RECONCILER_BACKEND_URL must persist candidate review items through the
 * existing PostgreSQL/HITL workflow BEFORE returning newly_created queue receipts. The Worker
 * never assumes an item exists merely because an upstream analyzer suggested it.
 */

const MAX_BODY_BYTES = 16384;
const MAX_RECEIPT_BYTES = 262144;
const MAX_ALERTS_PER_RUN = 20;
const SOURCES = new Set([
  "memsearch", "engram", "cnf", "docstore", "propria-search",
  "superindex", "casebible", "workbench", "ccc"
]);
const TYPES = new Set(["memory.changed", "memory.created", "memory.retired",
  "document.changed", "document.created", "record.changed", "record.created"]);
const LEVELS = new Set(["low", "medium", "high", "critical"]);
const DECISIONS = new Set(["queued", "deferred", "decided"]);

function reply(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
      "x-content-type-options": "nosniff"
    }
  });
}

function string(value, limit = 512) {
  return typeof value === "string" ? value.trim().slice(0, limit) : "";
}

function fixedHttps(value, label) {
  if (!value) return null;
  const url = new URL(value);
  if (url.protocol !== "https:" || url.username || url.password ||
      url.hash || url.search || !url.hostname) {
    throw new Error(label + " must be a fixed HTTPS endpoint without credentials or query");
  }
  return url.toString();
}

async function authenticate(request, env) {
  // This API is an administrative ingress; never accept an unauthenticated change event.
  if (!env.RECONCILER_TRIGGER_TOKEN) return false;
  const header = request.headers.get("authorization") || "";
  if (!header.startsWith("Bearer ")) return false;
  const actual = new TextEncoder().encode(header.slice(7));
  const expected = new TextEncoder().encode(env.RECONCILER_TRIGGER_TOKEN);
  let mismatch = actual.length ^ expected.length;
  for (let i = 0; i < Math.max(actual.length, expected.length); i++) {
    mismatch |= (actual[i] || 0) ^ (expected[i] || 0);
  }
  return mismatch === 0;
}

async function parseBody(request) {
  const length = Number(request.headers.get("content-length") || "0");
  if (length > MAX_BODY_BYTES) throw new Error("body_too_large");
  const body = await request.text();
  if (new TextEncoder().encode(body).length > MAX_BODY_BYTES) throw new Error("body_too_large");
  try {
    const value = JSON.parse(body);
    if (!value || typeof value !== "object" || Array.isArray(value)) throw Error();
    return value;
  } catch {
    throw new Error("invalid_json_object");
  }
}

function eventPayload(input) {
  if (!TYPES.has(input.type) || !SOURCES.has(input.source))
    throw new Error("invalid_event_type_or_source");
  const ref = string(input.record_ref, 1024);
  const revision = string(input.revision, 256);
  const scope = string(input.scope, 120);
  if (!ref || !revision || !scope) throw new Error("event_requires_record_ref_revision_scope");
  if (Object.hasOwn(input, "content") || Object.hasOwn(input, "raw")) {
    throw new Error("source_content_not_allowed_in_trigger");
  }
  return {type: input.type, source: input.source, record_ref: ref,
    revision, scope, requested_by: "source_event"};
}

function runPayload(input) {
  const scope = string(input.scope, 120) || "propria";
  if (scope !== "propria" && !scope.startsWith("propria/"))
    throw new Error("invalid_propria_scope");
  const source = string(input.source, 60);
  if (source && !SOURCES.has(source)) throw new Error("invalid_source");
  return {scope, source: source || null, requested_by: "manual"};
}

async function readJsonResponse(response, label) {
  const text = await response.text();
  if (new TextEncoder().encode(text).length > MAX_RECEIPT_BYTES)
    throw new Error(label + "_response_too_large");
  if (!response.ok) throw new Error(label + "_http_" + response.status);
  let value;
  try { value = JSON.parse(text); } catch { throw new Error(label + "_invalid_json"); }
  if (!value || typeof value !== "object" || Array.isArray(value))
    throw new Error(label + "_invalid_result");
  return value;
}

function sanitizeReceipt(result) {
  if (result.schema !== "propria-reconciliation-result/v1")
    throw new Error("backend_receipt_contract_mismatch");
  const runId = string(result.run_id, 128);
  if (!runId || !Array.isArray(result.review_queue))
    throw new Error("backend_missing_run_or_review_queue");
  if (result.review_queue.length > 1000)
    throw new Error("backend_review_queue_overflow");
  const reviewQueue = result.review_queue.map(entry => {
    if (!entry || typeof entry !== "object") throw new Error("invalid_review_receipt");
    const reviewRef = string(entry.review_ref, 512);
    const status = string(entry.status, 30);
    const severity = string(entry.severity, 30);
    if (!reviewRef || !DECISIONS.has(status) || !LEVELS.has(severity))
      throw new Error("invalid_review_receipt");
    return {
      review_ref: reviewRef, status, severity,
      kind: string(entry.kind, 80),
      scope: string(entry.scope, 120),
      newly_created: entry.newly_created === true,
      // No source body, evidence excerpts, credentials, or LLM reasoning in an alert.
      review_path: "/activity"
    };
  });
  return {
    run_id: runId,
    state: string(result.state, 80) || "processed",
    counts: {
      scanned: Number.isSafeInteger(result.scanned) ? result.scanned : null,
      findings: Number.isSafeInteger(result.findings) ? result.findings : null,
      queued: reviewQueue.filter(x => x.status === "queued").length
    },
    review_queue: reviewQueue,
    source_mutations: 0
  };
}

function alertCandidates(receipt) {
  // Only genuinely new, persisted high-severity cases generate an immediate alert.
  // Older/unreviewed material stays queryable but does not flood the owner inbox.
  return receipt.review_queue.filter(x =>
    x.status === "queued" && x.newly_created &&
    (x.severity === "high" || x.severity === "critical")
  ).slice(0, MAX_ALERTS_PER_RUN);
}

async function postJson(url, token, body, label) {
  const endpoint = fixedHttps(url, label);
  if (!endpoint || !token) throw new Error(label + "_not_configured");
  const response = await fetch(endpoint, {
    method: "POST",
    headers: {"content-type": "application/json",
      "authorization": "Bearer " + token},
    body: JSON.stringify(body),
    redirect: "error",
    signal: AbortSignal.timeout(20000)
  });
  return readJsonResponse(response, label);
}

async function dispatch(env, input) {
  // Backend must have persisted review items through the established PG/Temporal/Workbench
  // authority chain, returning verified references. The edge Worker has no approval power.
  let value;
  try {
    value = await postJson(env.RECONCILER_BACKEND_URL,
      env.RECONCILER_BACKEND_TOKEN,
      {schema: "propria-reconciliation-trigger/v1", ...input},
      "reconciler_backend");
  } catch (error) {
    return {status: "blocked", reason: String(error.message).slice(0,120),
      review_queue: [], alerts_sent: 0};
  }
  let receipt;
  try { receipt = sanitizeReceipt(value); } catch (error) {
    return {status: "blocked", reason: String(error.message).slice(0,120),
      review_queue: [], alerts_sent: 0};
  }
  const candidates = alertCandidates(receipt);
  if (!candidates.length) {
    return {status: "completed", ...receipt, alerts_sent: 0};
  }
  if (!env.N8N_ALERT_WEBHOOK_URL || !env.N8N_ALERT_TOKEN) {
    return {status: "review_queued_alert_blocked", ...receipt, alerts_sent: 0,
      alert_blocker: "n8n alert endpoint or credential unavailable"};
  }
  let alertsSent = 0;
  const failures = [];
  for (const item of candidates) {
    try {
      // n8n DISPATCHES only: does not approve, archive, delete, or change records.
      await postJson(env.N8N_ALERT_WEBHOOK_URL, env.N8N_ALERT_TOKEN, {
        schema: "propria-knowledge-review-alert/v1",
        run_id: receipt.run_id, review_ref: item.review_ref,
        kind: item.kind, severity: item.severity, scope: item.scope,
        workbench_path: item.review_path
      }, "n8n_notification");
      alertsSent++;
    } catch (error) {
      failures.push({review_ref: item.review_ref, reason: String(error.message).slice(0,90)});
    }
  }
  return {
    status: failures.length ? "review_queued_alert_failed" : "completed",
    ...receipt, alerts_sent: alertsSent,
    alert_failures: failures,
    alert_truncated: receipt.review_queue.filter(x => x.status === "queued" &&
      x.newly_created && (x.severity === "high" || x.severity === "critical")).length > MAX_ALERTS_PER_RUN
  };
}

async function run(request, env, kind) {
  if (!(await authenticate(request, env))) return reply({error: "unauthorized"}, 401);
  if (request.method !== "POST") return reply({error: "method_not_allowed"}, 405);
  let parsed, input;
  try {
    parsed = await parseBody(request);
    input = kind === "event" ? eventPayload(parsed) : runPayload(parsed);
  } catch (error) {
    return reply({error: String(error.message).slice(0,120)}, 400);
  }
  const result = await dispatch(env, {kind, ...input});
  return reply(result, result.status === "blocked" ? 503 : 200);
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (request.method === "GET" && url.pathname === "/health") {
      return reply({ok: true, service: "propria-knowledge-reconciler",
        capability: "triggers_review_alerts", approval_authority: false});
    }
    if (url.pathname === "/events") return run(request, env, "event");
    if (url.pathname === "/runs") return run(request, env, "manual");
    return reply({error: "not_found"}, 404);
  },
  async scheduled(event, env, ctx) {
    ctx.waitUntil(dispatch(env, {
      kind: "scheduled", scope: "propria", requested_by: "daily_cron",
      scheduled_at: new Date(event.scheduledTime).toISOString()
    }).then(result => {
      if (result.status !== "completed") console.error("Reconciliation scheduled run status:", result.status,
        "run:", result.run_id || "none");
    }).catch(error => console.error("Reconciliation scheduled dispatch failed:", error.name)));
  }
};

export const testHooks = {sanitizeReceipt, alertCandidates, eventPayload, runPayload, dispatch};
