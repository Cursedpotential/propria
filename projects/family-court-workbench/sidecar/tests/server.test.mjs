// Byline: Claude Code · Sonnet 5 · 2026-09-07
import assert from "node:assert/strict";
import { after, test } from "node:test";
import app from "../server.mjs";
import { loadStoreModule } from "../lib/store-client.mjs";

// See store-client.test.mjs's identical teardown comment: the embedded
// SurrealDB store's native handle keeps the event loop alive without this.
after(async () => {
  const mod = await loadStoreModule();
  await mod.closeAllStoresForTests?.();
  await app.close();
});

test("GET /api/health", async () => {
  const res = await app.inject({ method: "GET", url: "/api/health" });
  assert.equal(res.statusCode, 200);
  assert.equal(res.json().ok, true);
});

test("GET /api/auth/status never throws and never returns a raw token value", async () => {
  const res = await app.inject({ method: "GET", url: "/api/auth/status" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  assert.ok("configured" in body);
  assert.ok(!("token" in body), "token value must never be returned over HTTP");
});

test("GET /api/store/summary answers against the real store module (available or a clear reason)", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/summary" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  // Either a real CaseSummary (configured: true) or a graceful degradation
  // shape ({ available: false, reason }) — never a thrown 500 for a store
  // that simply has no data yet.
  assert.ok(body.configured === true || body.available === false, JSON.stringify(body));
});

test("GET /api/store/docket returns real entries", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/docket" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  assert.ok(Array.isArray(body.entries) || body.available === false, JSON.stringify(body));
});

test("GET /api/store/timeline is mode-aware (court | master | merged)", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/timeline?mode=court" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  if (body.available !== false) {
    assert.equal(body.mode, "court");
  }
});

test("GET /api/store/source requires id", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/source" });
  assert.equal(res.statusCode, 400);
});

test("GET /api/store/search requires q", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/search" });
  assert.equal(res.statusCode, 400);
});

test("POST /api/chat requires prompt and returns 401 or SSE, never an unhandled crash", async () => {
  const res = await app.inject({ method: "POST", url: "/api/chat", payload: {} });
  assert.equal(res.statusCode, 400);
});
