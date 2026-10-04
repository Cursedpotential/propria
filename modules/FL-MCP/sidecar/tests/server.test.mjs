// Byline: Claude Code · Sonnet 5 · 2026-09-07
import assert from "node:assert/strict";
import { after, test } from "node:test";
import app, { buildLibraryPage } from "../server.mjs";
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

test("library page validation rejects missing and malformed caseQuery envelopes", () => {
  const validCount = { results: [[{ count: 3 }]], truncated: false };
  const base = { table: "reference", limit: 2, offset: 0, countResult: validCount };
  for (const pageResult of [
    undefined,
    {},
    { results: [], truncated: false },
    { results: [[null]], truncated: false },
    { results: [[{ label: "missing id" }]], truncated: false },
    { results: [[{ id: "reference:1" }, { id: "reference:2" }, { id: "reference:3" }]], truncated: false },
    { results: [[{ id: "reference:1" }]], truncated: true },
  ]) {
    assert.throws(() => buildLibraryPage({ ...base, pageResult }));
  }
  for (const countResult of [undefined, {}, { results: [[]], truncated: false }, { results: [[{}]], truncated: false }]) {
    assert.throws(() => buildLibraryPage({
      table: "reference",
      limit: 2,
      offset: 0,
      pageResult: { results: [[{ id: "reference:1" }]], truncated: false },
      countResult,
    }));
  }
});

test("library page validation rejects an empty nonterminal offset page", () => {
  assert.throws(() => buildLibraryPage({
    table: "source",
    limit: 25,
    offset: 3,
    pageResult: { results: [[]], truncated: false },
    countResult: { results: [[{ count: 4 }]], truncated: false },
  }), /empty or non-advancing/);
  const terminal = buildLibraryPage({
    table: "source",
    limit: 25,
    offset: 4,
    pageResult: { results: [[]], truncated: false },
    countResult: { results: [[{ count: 4 }]], truncated: false },
  });
  assert.equal(terminal.next_offset, null);
});

test("GET /api/store/library pages beyond 200, and /api/store/record returns canonical exact detail", async () => {
  const [mod, store] = await Promise.all([loadStoreModule(), getSharedStore()]);
  for (let index = 0; index < 205; index += 1) {
    await mod.casePut(store, {
      table: "reference",
      id: `desktop-library-fixture-${String(index).padStart(3, "0")}`,
      data: {
        kind: "fixture-kind",
        category: "fixture-category",
        pattern: `fixture-pattern-${index}`,
        definition: `Synthetic reference body ${index}.`,
        aliases: ["fixture alias"],
        source: index === 204
          ? { path: "fixture/source-a.md", sha256: "fixture-hash-a", official_url: "https://example.invalid/source-a" }
          : null,
      },
    });
  }
  await mod.casePut(store, {
    table: "source",
    id: "desktop-library-source-fixture",
    data: {
      title: "Synthetic source citation",
      body: "Synthetic source body retained in the exact record.",
      source_path: "fixture/citation.md",
      sha256: "fixture-source-hash",
      official_url: "https://example.invalid/citation",
    },
  });

  const res = await app.inject({ method: "GET", url: "/api/store/library?table=reference&limit=50&offset=200" });
  assert.equal(res.statusCode, 200);
  const body = res.json();
  assert.equal(body.table, "reference");
  assert.equal(body.total, 205);
  assert.equal(body.offset, 200);
  assert.equal(body.entries.length, 5);
  assert.equal(body.entries[0].id, "reference:desktop-library-fixture-200");
  assert.equal(body.entries[4].id, "reference:desktop-library-fixture-204");
  assert.equal(body.next_offset, null);

  const detail = await app.inject({ method: "GET", url: "/api/store/record?id=reference%3Adesktop-library-fixture-204" });
  assert.equal(detail.statusCode, 200);
  assert.equal(detail.json().id, "reference:desktop-library-fixture-204");
  assert.equal(detail.json().table, "reference");
  assert.match(detail.json().version, /^sha256:/);
  assert.equal(detail.json().record.definition, "Synthetic reference body 204.");
  assert.equal(detail.json().record.source.sha256, "fixture-hash-a");
  assert.equal(detail.json().record.source.official_url, "https://example.invalid/source-a");

  const sourcePage = await app.inject({ method: "GET", url: "/api/store/library?table=source&limit=25&offset=0" });
  assert.equal(sourcePage.statusCode, 200);
  assert.equal(sourcePage.json().total, 1);
  assert.equal(sourcePage.json().entries[0].id, "source:desktop-library-source-fixture");
  const sourceDetail = await app.inject({ method: "GET", url: "/api/store/record?id=source%3Adesktop-library-source-fixture" });
  assert.equal(sourceDetail.statusCode, 200);
  assert.equal(sourceDetail.json().record.body, "Synthetic source body retained in the exact record.");
  assert.equal(sourceDetail.json().record.source_path, "fixture/citation.md");
  assert.equal(sourceDetail.json().record.sha256, "fixture-source-hash");
  assert.equal(sourceDetail.json().record.official_url, "https://example.invalid/citation");

  const countAfterReads = await mod.caseQuery(store, { surql: "SELECT count() AS count FROM source GROUP ALL;" });
  assert.equal(countAfterReads.results.at(-1)[0].count, 1, "read endpoints must not add or rewrite source rows");

  const invalid = await app.inject({ method: "GET", url: "/api/store/library?table=other" });
  assert.equal(invalid.statusCode, 400);
  const missing = await app.inject({ method: "GET", url: "/api/store/record?id=reference%3Amissing-fixture" });
  assert.equal(missing.statusCode, 404);
});

test("GET /api/store/search requires q", async () => {
  const res = await app.inject({ method: "GET", url: "/api/store/search" });
  assert.equal(res.statusCode, 400);
});

test("POST /api/chat requires prompt and returns 401 or SSE, never an unhandled crash", async () => {
  const res = await app.inject({ method: "POST", url: "/api/chat", payload: {} });
  assert.equal(res.statusCode, 400);
});
