import test from "node:test";
import assert from "node:assert/strict";
import worker, {testHooks} from "./index.js";

const TOKEN = "unit-test-secret";
const backend = "https://internal.example/reconcile";
const notifier = "https://n8n.example/hook";

function env() {
  return {
    RECONCILER_TRIGGER_TOKEN: TOKEN,
    RECONCILER_BACKEND_URL: backend,
    RECONCILER_BACKEND_TOKEN: "backend-secret",
    N8N_ALERT_WEBHOOK_URL: notifier,
    N8N_ALERT_TOKEN: "n8n-secret"
  };
}
function request(path, body, token = TOKEN) {
  return new Request("https://edge.example" + path, {
    method: "POST",
    headers: {"authorization": "Bearer " + token, "content-type": "application/json"},
    body: JSON.stringify(body)
  });
}
function receipt(overrides = {}) {
  return {schema: "propria-reconciliation-result/v1", run_id: "run-test-1",
    state: "review_pending", scanned: 5, findings: 1,
    review_queue: [{review_ref: "review-99", severity: "high", kind: "stale_conflicting_memory",
      scope: "propria", status: "queued", newly_created: true}],
    ...overrides};
}
function withFetch(fake, action) {
  const previous = globalThis.fetch;
  globalThis.fetch = fake;
  return Promise.resolve().then(action).finally(() => {globalThis.fetch = previous;});
}

test("read-only health exposes no credentials or data", async () => {
  const response = await worker.fetch(new Request("https://edge.example/health"), env());
  assert.equal(response.status, 200);
  assert.equal((await response.json()).approval_authority, false);
});

test("events require an authorized caller and source/version pins", async () => {
  await withFetch(async () => {throw Error("backend must not be called");}, async () => {
    const forbidden = await worker.fetch(request("/events", {
      type: "memory.changed", source: "memsearch", record_ref: "one", revision: "2",
      scope: "propria"
    }, "bad-token"), env());
    assert.equal(forbidden.status, 401);
    const bad = await worker.fetch(request("/events", {
      type: "memory.changed", source: "memsearch", record_ref: "one", revision: "2",
      scope: "propria", content: "private record contents must stay in source"
    }), env());
    assert.equal(bad.status, 400);
  });
});

test("persisted new high-severity HITL case creates a correlated n8n alert", async () => {
  const calls = [];
  await withFetch(async (url, options) => {
    calls.push({url: String(url), body: JSON.parse(options.body)});
    if (String(url) === backend) {
      return Response.json(receipt());
    }
    if (String(url) === notifier) {
      return Response.json({accepted: true, review_ref: "review-99"});
    }
    throw Error("unexpected endpoint");
  }, async () => {
    const response = await worker.fetch(request("/events", {
      type: "memory.changed", source: "engram", record_ref: "conversation:45", revision: "8",
      scope: "propria"
    }), env());
    const result = await response.json();
    assert.equal(result.status, "completed");
    assert.equal(result.alerts_sent, 1);
    assert.equal(result.review_queue[0].review_ref, "review-99");
    assert.equal(calls.length, 2);
    assert.equal(calls[0].body.source, "engram");
    assert.equal(calls[1].body.schema, "propria-knowledge-review-alert/v1");
    assert.ok(!JSON.stringify(calls[1].body).includes("private text"));
    assert.equal(calls[1].body.review_ref, "review-99");
  });
});

test("low severity, decided and older queued cases do not flood alerts", () => {
  const result = testHooks.sanitizeReceipt(receipt({review_queue: [
    {review_ref:"one",severity:"low",status:"queued",newly_created:true},
    {review_ref:"two",severity:"high",status:"queued",newly_created:false},
    {review_ref:"three",severity:"critical",status:"decided",newly_created:true}
  ]}));
  assert.equal(testHooks.alertCandidates(result).length, 0);
});

test("a backend suggestion lacking a persisted review ref is not treated as an alert", async () => {
  const calls = [];
  await withFetch(async (url) => {
    calls.push(String(url));
    return Response.json(receipt({review_queue: [{severity: "critical",
      kind: "conflict", status: "queued", newly_created: true}]}));
  }, async () => {
    const response = await worker.fetch(request("/runs", {scope: "propria"}), env());
    const result = await response.json();
    assert.equal(response.status, 503);
    assert.equal(result.status, "blocked");
    assert.deepEqual(calls, [backend]);
  });
});

test("unacknowledged notifications are reported, not called successful", async () => {
  await withFetch(async (url) => Response.json(String(url) === backend
    ? receipt()
    : {accepted: false, review_ref: "review-99"}), async () => {
    const response = await worker.fetch(request("/runs", {scope: "propria"}), env());
    const result = await response.json();
    assert.equal(result.status, "review_queued_alert_failed");
    assert.equal(result.alerts_sent, 0);
    assert.equal(result.alert_failures[0].review_ref, "review-99");
  });
});

test("a missing backend cannot create a false success result", async () => {
  const e = env();
  delete e.RECONCILER_BACKEND_URL;
  const response = await worker.fetch(request("/runs", {scope:"propria"}), e);
  assert.equal(response.status, 503);
  assert.equal((await response.json()).status, "blocked");
});

test("daily cron triggers the same backend path without claiming approvals", async () => {
  const promises = [];
  let seen = null;
  await withFetch(async (url, options) => {
    seen = JSON.parse(options.body);
    return Response.json(receipt({review_queue: []}));
  }, async () => {
    await worker.scheduled({scheduledTime: Date.UTC(2026, 9, 17, 12, 0, 0)}, env(), {
      waitUntil(promise) {promises.push(promise);}
    });
    await Promise.all(promises);
  });
  assert.equal(seen.kind, "scheduled");
  assert.equal(seen.scope, "propria");
  assert.ok(seen.scheduled_at.startsWith("2026-10-17T12:00:00"));
});
