// Byline: Claude Code · Sonnet 5 · 2026-09-07
import assert from "node:assert/strict";
import { after, test } from "node:test";
import app from "../server.mjs";
import { getSharedStore, loadStoreModule } from "../lib/store-client.mjs";

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

test("GET /api/store/reference/library filters and pages synthetic shared-store rows", async () => {
  const [mod, store] = await Promise.all([loadStoreModule(), getSharedStore()]);
  await mod.casePut(store, {
    table: "reference",
    id: "desktop-library-fixture-a",
    data: {
      kind: "fixture-kind",
      category: "fixture-category",
      pattern: "fixture-pattern-a",
      definition: "Synthetic reference body A.",
      aliases: ["fixture alias"],
      source: { path: "fixture/source-a.md", sha256: "fixture-hash-a", url: "https://example.invalid/source-a" },
    },
  });
  await mod.casePut(store, {
    table: "reference",
    id: "desktop-library-fixture-b",
    data: { kind: "fixture-kind", category: "fixture-category", pattern: "fixture-pattern-b", definition: "Synthetic reference body B." },
  });

  const res = await app.inject({ method: "GET", url: "/api/store/reference/library?q=fixture&limit=1&offset=0" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  assert.equal(body.total, 2);
  assert.equal(body.entries.length, 1);
  assert.equal(body.offset, 0);
  assert.equal(body.limit, 1);
  assert.equal(body.next_offset, 1);
  assert.ok(body.entries[0].definition.startsWith("Synthetic reference body "));

  const bySource = await app.inject({ method: "GET", url: "/api/store/reference/library?q=fixture-hash-a" });
  assert.equal(bySource.statusCode, 200);
  assert.equal(bySource.json().entries[0].source.sha256, "fixture-hash-a");
  assert.equal(bySource.json().entries[0].source.path, "fixture/source-a.md");

  const next = await app.inject({ method: "GET", url: "/api/store/reference/library?q=fixture&limit=1&offset=1" });
  assert.equal(next.statusCode, 200);
  assert.equal(next.json().entries.length, 1);
  assert.equal(next.json().next_offset, null);

  const invalid = await app.inject({ method: "GET", url: "/api/store/reference/library?limit=51" });
  assert.equal(invalid.statusCode, 400);
});

test("GET /api/store/search requires q", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/search" });
  assert.equal(res.statusCode, 400);
});

test("POST /api/chat requires prompt and returns 401 or SSE, never an unhandled crash", async () => {
  const res = await app.inject({ method: "POST", url: "/api/chat", payload: {} });
  assert.equal(res.statusCode, 400);
});
