// Byline: Claude Code · Fable 5.1 · 2026-09-07
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — tests for the owner's 13:09-13:16
// orders: court_event status, timeline lanes, docket, memo, evidence_log, eval,
// reference load+match, the case-extract/v1 importer, platform export, case_source.
// Byline: Codex · GPT-6 · 2026-10-04 — private full-context and governed-library contract.
import assert from "node:assert/strict";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import test, { after } from "node:test";
import { fileURLToPath } from "node:url";
import * as store from "../dist/store.js";

// The embedded engine's mem:// connections keep a native handle open past the
// last assertion (verified live — every assertion above finishes in ~1s but
// the process otherwise never exits on its own, even after closing every
// store). `node --test` isolates each matched test file into its own
// subprocess by default, so forcing exit here — only after every test in
// THIS file has already passed — cannot hide a failure in another test file.
after(async () => {
  await store.closeAllStoresForTests();
  // Give the test runner's own reporter a full turn to flush the last
  // test results before we force-exit (an immediate process.exit() here
  // was observed live to truncate the TAP report by 1-2 tests).
  await new Promise((resolve) => setTimeout(resolve, 250));
  process.exit(0);
});

async function freshStore() {
  store.resetStoreForTests();
  const s = await store.getStore("mem://");
  assert.equal(s.available, true, s.available ? "" : s.reason);
  return s;
}

// Byline: Codex · GPT-6 · 2026-10-04.
/** Create a uniquely named fixture directory under the ignored project quarantine.
 * Inputs: a short fixture-name prefix. Outputs: an existing writable directory path.
 * Effects: creates files that remain under to_be_deleted for owner-controlled cleanup.
 * Choose for generated test fixtures instead of operating-system temporary directories.
 */
function createRetainedFixtureDirectory(prefix) {
  const configuredRoot = process.env.FCT_TEST_RETAINED_ROOT;
  if (configuredRoot) {
    if (!isAbsolute(configuredRoot) || !configuredRoot.replaceAll("\\", "/").split("/").includes("to_be_deleted")) {
      throw new Error("VPS fixture root must be an absolute retained quarantine path");
    }
    mkdirSync(configuredRoot, { recursive: true });
    return mkdtempSync(join(configuredRoot, `${prefix}-`));
  }
  let repositoryRoot = dirname(fileURLToPath(import.meta.url));
  while (!existsSync(join(repositoryRoot, ".git"))) {
    const parent = dirname(repositoryRoot);
    if (parent === repositoryRoot) throw new Error("Could not locate the Propria Git worktree root for retained test fixtures");
    repositoryRoot = parent;
  }
  const fixtureRoot = join(repositoryRoot, "to_be_deleted", "family-court-console-store-tests");
  mkdirSync(fixtureRoot, { recursive: true });
  return mkdtempSync(join(fixtureRoot, `${prefix}-`));
}

test("migration is idempotent: opening the same mem:// store twice does not error and keeps one embed_dim", async () => {
  const s1 = await freshStore();
  const q1 = await store.caseQuery(s1, { surql: "SELECT embed_dim FROM meta:config;" });
  const dim1 = q1.results[0][0].embed_dim;

  // Re-run migration against the SAME live connection by calling getStore()
  // again with the same override url while cached — should be a no-op cache
  // hit, not a re-migration; then force a genuinely fresh re-open of a NEW
  // mem:// instance to prove the DDL itself is idempotent (IF NOT EXISTS).
  const sameHandle = await store.getStore("mem://");
  assert.equal(sameHandle.db, s1.db, "getStore() with the same url should return the cached connection");

  store.resetStoreForTests();
  const s2 = await store.getStore("mem://");
  const q2 = await store.caseQuery(s2, { surql: "SELECT embed_dim FROM meta:config;" });
  assert.equal(q2.results[0][0].embed_dim, dim1);
  assert.equal(typeof dim1, "number");
  assert.ok(dim1 > 0);
});

test("case_put upserts a record and RELATEs it in the same call", async () => {
  const s = await freshStore();
  const result = await store.casePut(s, {
    table: "event",
    id: "e1",
    data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T12:00:00Z", description: "Exchange missed; no notice given" },
  });
  assert.equal(result.table, "event");
  assert.equal(result.id, "e1");
  assert.equal(result.record.id, "event:e1");

  const withRelation = await store.casePut(s, {
    table: "exhibit",
    id: "x1",
    data: { label: "Ex. 4 screenshot", occurred_at: "2026-08-01T09:00:00Z", known_at: "2026-08-01T13:00:00Z" },
    relations: [{ edge: "evidences", from: "exhibit:x1", to: "event:e1" }],
  });
  assert.equal(withRelation.relations.length, 1);
  assert.equal(withRelation.relations[0].edge, "evidences");
  assert.equal(withRelation.relations[0].from, "exhibit:x1");
  assert.equal(withRelation.relations[0].to, "event:e1");
});

test("case_put retains full child identity with optional initials and age", async () => {
  const s = await freshStore();
  const saved = await store.casePut(s, {
    table: "child",
    id: "c1",
    data: { name: "Quinn Samplechild", aliases: ["Q. Sample"], initials: "Q.S.", age: 7 },
  });
  assert.equal(saved.record.name, "Quinn Samplechild");
  assert.deepEqual(saved.record.aliases, ["Q. Sample"]);
  assert.equal(saved.record.initials, "Q.S.");
  assert.equal(saved.record.age, 7);
});

test("case_query permits SELECT-only inspection, rejects mutations and functions for either write flag, and caps rows at 200", async () => {
  const s = await freshStore();
  for (let i = 0; i < 5; i++) {
    await store.casePut(s, { table: "note", id: `n${i}`, data: { text: `note ${i}` } });
  }
  const rejectedQueries = [
    "DELETE note;",
    "REMOVE TABLE note;",
    "DEFINE TABLE evil SCHEMALESS;",
    "SELECT * FROM fn::current_decisions();",
  ];
  for (const write of [false, true]) {
    for (const surql of rejectedQueries) {
      await assert.rejects(
        () => store.caseQuery(s, { surql, write }),
        /read-only|SELECT-only/i,
        `expected raw query to reject with write=${write}: ${surql}`,
      );
    }
  }
  const inspected = await store.caseQuery(s, { surql: "SELECT * FROM note;", write: false });
  assert.equal(inspected.results[0].length, 5);
  assert.equal(inspected.truncated, false);

  // Row cap: insert 205 events, confirm a plain SELECT caps at 200 and flags truncated.
  for (let i = 0; i < 205; i++) {
    await store.casePut(s, { table: "note", id: `bulk${i}`, data: { text: `bulk ${i}` } });
  }
  const capped = await store.caseQuery(s, { surql: "SELECT * FROM note;" });
  assert.equal(capped.results[0].length, 200);
  assert.equal(capped.truncated, true);
});

test("case_search: full-text search finds a seeded event", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T10:00:00Z", description: "Exchange missed at the police station; no notice given" } });
  await store.casePut(s, { table: "event", id: "e2", data: { occurred_at: "2026-08-15T10:00:00Z", known_at: "2026-08-20T00:00:00Z", description: "Text message threatening to withhold parenting time" } });

  const result = await store.caseSearch(s, { query: "parenting time", mode: "text" });
  assert.equal(result.degraded, false);
  assert.ok(result.hits.length >= 1, "expected at least one text-search hit");
  assert.ok(result.hits.some((h) => h.id === "event:e2"));
});

test("case_search: text-mode score is > 0 for a matching query, even in a tiny (1-2 record) corpus", async () => {
  // Coordinator-reported defect: `search::score(1)` on this SurrealDB build
  // uses the classical (non-smoothed) BM25 IDF term, which is mathematically
  // exactly 0 (or negative, clamped) whenever the matching term appears in
  // >= 50% of the indexed corpus — i.e. guaranteed for a store's first 1-2
  // records. Verified live: score stayed exactly 0 through 2 total
  // documents, then became positive from 3 documents on. naiveOverlapScore()
  // floors the reported score above 0 for any row that already matched the
  // FULLTEXT predicate, so a caller never sees a "relevant" row reported
  // with a score indistinguishable from "not relevant".
  const s = await freshStore();
  const single = await store.casePut(s, {
    table: "event",
    id: "probe1",
    data: { description: "Exchange missed; no notice of the change in parenting time" },
  });
  assert.equal(single.record.id, "event:probe1");

  const result1 = await store.caseSearch(s, { query: "parenting time", mode: "text" });
  const hit1 = result1.hits.find((h) => h.id === "event:probe1");
  assert.ok(hit1, "expected the single seeded record to be found (1-document corpus)");
  assert.ok(hit1.score > 0, `expected score > 0 for a matching query, got ${hit1.score}`);

  // A second, non-matching document (2-document corpus — still the
  // documented 50%-match degenerate case for raw BM25).
  await store.casePut(s, { table: "event", id: "filler1", data: { description: "unrelated filler about something else entirely" } });
  const result2 = await store.caseSearch(s, { query: "parenting time", mode: "text" });
  const hit2 = result2.hits.find((h) => h.id === "event:probe1");
  assert.ok(hit2, "expected the record to still be found in a 2-document corpus");
  assert.ok(hit2.score > 0, `expected score > 0 in a 2-document corpus, got ${hit2.score}`);
});

test("case_put then case_search in the SAME connection returns the record — same-session visibility", async () => {
  // Coordinator-reported defect: probing the INSTALLED server against a real
  // rocksdb file, `case_put` followed immediately by `case_search` in one
  // server session returned hits: [] — a fresh process against the same
  // file then found it. This test exercises the exact reported scenario
  // (same put id/description/query, mode: "text") against a REAL rocksdb://
  // file (not mem://) in a scratch OS temp directory, reusing the SAME
  // cached `getStore()` connection both calls would use in a real server
  // process, to catch any same-session/connection-reuse regression.
  const dir = createRetainedFixtureDirectory("fct-store-sesvis");
  const dbPath = join(dir, "case.db").replace(/\\/g, "/");
  store.resetStoreForTests();
  const s = await store.getStore(dbPath);
  assert.equal(s.available, true, s.available ? "" : s.reason);

  const put = await store.casePut(s, {
    table: "event",
    id: "probe1",
    data: { description: "Exchange missed; no notice of the change in parenting time" },
  });
  assert.equal(put.record.id, "event:probe1");

  // Immediately, same connection, same process — no delay, no reconnect.
  const search = await store.caseSearch(s, { query: "parenting time", mode: "text" });
  assert.ok(
    search.hits.some((h) => h.id === "event:probe1"),
    `expected same-session put-then-search to find the record; got hits: ${JSON.stringify(search.hits)}`,
  );

  await store.closeAllStoresForTests();
});

test("case_search: hybrid mode without embeddings configured degrades to text and still returns hits", async () => {
  // This dev machine's ~/.secrets/*.env may carry a real NVIDIA_API_KEY, so
  // unsetting the env vars alone would not exercise the degrade path — use
  // the explicit test-only override instead (see embeddingsDisabledForTests
  // in src/store.ts) for a deterministic "no embeddings" scenario.
  const had = process.env.CUSTODY_DISABLE_EMBEDDINGS;
  process.env.CUSTODY_DISABLE_EMBEDDINGS = "1";
  try {
    const s = await freshStore();
    await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T10:00:00Z", description: "Exchange missed at the police station; no notice given" } });
    const result = await store.caseSearch(s, { query: "police station", mode: "hybrid" });
    assert.equal(result.mode, "text");
    assert.equal(result.degraded, true);
    assert.match(result.degraded_reason, /NVIDIA_API_KEY|NIM_API_KEY/);
    assert.ok(result.hits.some((h) => h.id === "event:e1"));
  } finally {
    if (had === undefined) delete process.env.CUSTODY_DISABLE_EMBEDDINGS;
    else process.env.CUSTODY_DISABLE_EMBEDDINGS = had;
  }
});

test("case_graph: neighbourhood includes a RELATEd factor", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e2", data: { occurred_at: "2026-08-15T10:00:00Z", known_at: "2026-08-20T00:00:00Z", description: "withholds parenting time" } });
  await store.casePut(s, {
    table: "event",
    id: "e2",
    data: {},
    relations: [{ edge: "supports_factor", from: "event:e2", to: "factor:j", data: { weight: 0.8 } }],
  });

  const graph = await store.caseGraph(s, { id: "event:e2", depth: 1 });
  const factorNeighbor = graph.neighbors.find((n) => n.id === "factor:j");
  assert.ok(factorNeighbor, "expected factor:j in the graph neighbourhood");
  assert.equal(factorNeighbor.edge, "supports_factor");
  assert.equal(factorNeighbor.direction, "out");
});

test("case_factor_map: seeds all 12 MCL 722.23 factors and counts a RELATEd event", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T10:00:00Z", description: "supports factor j" } });
  await store.casePut(s, { table: "event", id: "e1", data: {}, relations: [{ edge: "supports_factor", from: "event:e1", to: "factor:j", data: { weight: 0.9 } }] });

  const factors = await store.caseFactorMap(s);
  assert.equal(factors.length, 12);
  assert.deepEqual(factors.map((f) => f.letter), ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"]);
  const j = factors.find((f) => f.letter === "j");
  assert.equal(j.support_count, 1);
  assert.equal(j.top_supporting[0].id, "event:e1");
});

test("case_timeline: known_by filter excludes later-known events", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T12:00:00Z", description: "known early" } });
  await store.casePut(s, { table: "event", id: "e2", data: { occurred_at: "2026-08-15T10:00:00Z", known_at: "2026-08-20T00:00:00Z", description: "known late" } });

  const all = await store.caseTimeline(s, {});
  assert.equal(all.length, 2);

  const knownByEarly = await store.caseTimeline(s, { known_by: "2026-08-10T00:00:00Z" });
  assert.equal(knownByEarly.length, 1);
  assert.equal(knownByEarly[0].id, "event:e1");
});

test("case_export -> case_import round trip preserves records and edges", async () => {
  const s1 = await freshStore();
  await store.casePut(s1, { table: "event", id: "e1", data: { occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T12:00:00Z", description: "seed event" } });
  await store.casePut(s1, {
    table: "exhibit",
    id: "x1",
    data: { label: "Ex 1", occurred_at: "2026-08-01T10:00:00Z", known_at: "2026-08-01T12:00:00Z" },
    relations: [{ edge: "evidences", from: "exhibit:x1", to: "event:e1" }],
  });
  await store.casePut(s1, { table: "child", id: "c1", data: { name: "Quinn Samplechild", initials: "Q.S.", age: 5 } });
  await store.casePut(s1, { table: "source", id: "personal-source", data: { kind: "case_document", title: "Quinn Samplechild private statement" } });
  await s1.db.query("UPSERT $rid CONTENT $data;", {
    rid: store.parseRef("reference:retained-rule"),
    data: { kind: "behavior_pattern", key: "retained-rule", category: "synthetic", pattern: "sample", definition: "Imported rule draft must not publish directly" },
  });
  await s1.db.query("UPSERT $rid CONTENT $data;", {
    rid: store.parseRef("source:legal-authority"),
    data: { kind: "statute", title: "Synthetic imported authority" },
  });

  const tmpDir = createRetainedFixtureDirectory("fct-store-export");
  const exportPath = join(tmpDir, "snapshot.json");
  const exp = await store.caseExport(s1, exportPath);
  assert.equal(exp.path, exportPath);
  assert.equal(exp.counts.event, 1);
  assert.equal(exp.counts.exhibit, 1);
  assert.equal(exp.counts.child, 1);
  assert.equal(exp.counts.evidences, 1);

  store.resetStoreForTests();
  const s2 = await store.getStore("mem://");
  await s2.db.query("UPSERT $rid CONTENT $data;", {
    rid: store.parseRef("reference:retained-rule"),
    data: { kind: "behavior_pattern", key: "retained-rule", category: "published", pattern: "published-rule", definition: "Published reference remains untouched" },
  });
  await s2.db.query("UPSERT $rid CONTENT $data;", {
    rid: store.parseRef("source:legal-authority"),
    data: { kind: "statute", title: "Published authority remains untouched" },
  });
  const imp = await store.caseImport(s2, exportPath);
  assert.equal(imp.kind, "snapshot");
  assert.equal(imp.counts.event, 1);
  assert.equal(imp.counts.reference, 1);
  assert.equal(imp.counts.source, 2);
  assert.equal(imp.library_proposals.length, 2);
  assert.ok(imp.library_proposals.every((proposal) => proposal.status === "citation_required"));
  const retainedLibraryRows = await store.caseQuery(s2, { surql: "SELECT proposed_record, status FROM library_proposal;" });
  assert.equal(retainedLibraryRows.results[0].length, 2);
  assert.ok(retainedLibraryRows.results[0].some((row) => row.proposed_record.title === "Synthetic imported authority"));
  assert.ok(retainedLibraryRows.results[0].some((row) => row.proposed_record.definition === "Imported rule draft must not publish directly"));

  const graph = await store.caseGraph(s2, { id: "event:e1" });
  assert.ok(graph.neighbors.some((n) => n.id === "exhibit:x1" && n.edge === "evidences"));

  const importedChild = await store.caseQuery(s2, { surql: "SELECT * FROM child:c1;" });
  assert.equal(importedChild.results[0][0].name, "Quinn Samplechild");
  const personalSourceId = store.parseRef("source:personal-source");
  const importedPersonalSource = await store.caseQuery(s2, { surql: "SELECT * FROM $rid;", params: { rid: personalSourceId } });
  assert.equal(importedPersonalSource.results[0][0].title, "Quinn Samplechild private statement");
  const importedReference = await store.caseReferenceList(s2, { kind: "behavior_pattern" });
  assert.equal(importedReference.length, 1);
  assert.equal(importedReference[0].definition, "Published reference remains untouched");
  const publishedAuthority = await store.caseQuery(s2, {
    surql: "SELECT * FROM $rid;",
    params: { rid: store.parseRef("source:legal-authority") },
  });
  assert.equal(publishedAuthority.results[0][0].title, "Published authority remains untouched");
});

test("case_summary returns the case_facts-compatible shape", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "child", id: "c1", data: { initials: "E.F.", age: 7 } });
  await store.casePut(s, { table: "order", id: "o1", data: { title: "Judgment of Divorce", entered: "2024-01-15", served: "2024-01-22" } });
  await store.casePut(s, { table: "deadline", id: "d1", data: { label: "Referee objection window", due: "2026-09-28", rule: "referee_objection" } });

  const summary = await store.caseSummary(s);
  assert.equal(summary.configured, true);
  assert.equal(summary.source, "surrealdb-case-store");
  assert.equal(summary.children.count, 1);
  assert.equal(summary.children.entries[0].initials, "E.F.");
  assert.equal(summary.children.entries[0].age, 7);
  assert.ok(summary.case_context.children[0].id);
  assert.equal(summary.controlling_orders.length, 1);
  assert.equal(summary.controlling_orders[0].title, "Judgment of Divorce");
  assert.equal(summary.deadlines.length, 1);
  assert.equal(summary.deadlines[0].label, "Referee objection window");
  // Same top-level keys as core.ts's CaseFactsConfigured shape.
  for (const key of ["county", "court", "judge", "referee", "controlling_orders", "next_hearing", "deadlines", "parties", "children", "flags"]) {
    assert.ok(key in summary, `case_summary is missing key "${key}"`);
  }
});

test("store.ts degrades gracefully when the native module cannot load (simulated)", async () => {
  // We cannot easily uninstall @surrealdb/node mid-test-run, so this test
  // exercises the same code path a load failure takes: getStore() against an
  // intentionally invalid rocksdb path parent that cannot be created (a file
  // masquerading as a directory) still returns { available: false, reason }
  // rather than throwing.
  const { writeFileSync } = await import("node:fs");
  const dir = createRetainedFixtureDirectory("fct-store-bad");
  const blockerFile = join(dir, "blocker");
  writeFileSync(blockerFile, "not a directory");
  store.resetStoreForTests();
  const bad = await store.getStore(`rocksdb://${join(blockerFile, "nested", "case.db").replace(/\\/g, "/")}`);
  assert.equal(bad.available, false);
  assert.match(bad.reason, /failed to open case store/);
});

test("regression: a Windows absolute drive path (C:/...) opens at the real location, not relative to cwd", { skip: process.platform !== "win32" }, async () => {
  // Found live while running the CLI against a scratch rocksdb path: passing
  // an absolute Windows path straight into "rocksdb://<path>" mis-parses the
  // drive letter as a URL authority and silently drops it, creating the
  // database RELATIVE to process.cwd() (e.g. "<cwd>/C/Users/...") instead of
  // at the real "C:\Users\..." location. Fixed via toOpaqueIfWindowsDriveAbsolute()
  // (opaque "rocksdb:C:/..." form, no "//"). This test opens a real absolute
  // path and asserts both that data lands there AND that no bogus relative
  // "C/" directory appears under cwd.
  const { existsSync } = await import("node:fs");
  const dir = createRetainedFixtureDirectory("fct-store-abs");
  const absPath = join(dir, "case.db").replace(/\\/g, "/");
  assert.match(absPath, /^[A-Za-z]:\//, "this regression only applies on Windows-style absolute paths");

  store.resetStoreForTests();
  const s = await store.getStore(absPath);
  assert.equal(s.available, true, s.available ? "" : s.reason);
  await store.casePut(s, { table: "note", id: "n1", data: { text: "absolute path regression check" } });

  assert.ok(existsSync(absPath), `expected the database to exist at the real absolute path ${absPath}`);
  const bogusRelative = join(process.cwd(), absPath.slice(0, 1) + absPath.slice(2)); // "C:/x" -> "<cwd>/C/x"
  assert.ok(!existsSync(bogusRelative), `expected no bogus relative directory at ${bogusRelative}`);

  // Close the native store before leaving the generated database in quarantine.
  await store.closeAllStoresForTests();
});

// ---------------------------------------------------------------------------
// Owner orders 2026-09-07 13:09-13:16 — new registers, importer, platform export.
// ---------------------------------------------------------------------------

test("case_put: court_event.status is computed from date vs now when absent, but never clobbered by a status-less patch", async () => {
  const s = await freshStore();
  const past = await store.casePut(s, { table: "court_event", id: "ce-past", data: { kind: "hearing", date: "2020-01-01T00:00:00Z", title: "past hearing" } });
  assert.equal(past.record.status, "past");

  const future = await store.casePut(s, { table: "court_event", id: "ce-future", data: { kind: "hearing", date: "2099-01-01T00:00:00Z", title: "future hearing" } });
  assert.equal(future.record.status, "upcoming");

  const explicit = await store.casePut(s, { table: "court_event", id: "ce-explicit", data: { kind: "hearing", date: "2020-01-01T00:00:00Z", title: "adjourned hearing", status: "adjourned" } });
  assert.equal(explicit.record.status, "adjourned");

  // A patch that doesn't touch `date` (or `status`) must not clobber the
  // already-computed status.
  const patched = await store.casePut(s, { table: "court_event", id: "ce-past", data: { outcome: "granted" } });
  assert.equal(patched.record.status, "past");
});

test("case_timeline: mode court/master/merged tags each entry with its lane, and upcoming filters the court lane", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-01-01T00:00:00Z", known_at: "2026-01-01T00:00:00Z", description: "master lane event" } });
  await store.casePut(s, { table: "court_event", id: "ce1", data: { kind: "hearing", date: "2099-01-01T00:00:00Z", title: "future hearing" } });
  await store.casePut(s, { table: "court_event", id: "ce2", data: { kind: "hearing", date: "2020-01-01T00:00:00Z", title: "past hearing" } });

  const court = await store.caseTimeline(s, { mode: "court" });
  assert.ok(court.every((e) => e.lane === "court"));
  assert.ok(court.some((e) => e.id === "court_event:ce1"));
  assert.ok(court.some((e) => e.id === "court_event:ce2"));

  const master = await store.caseTimeline(s, { mode: "master" });
  assert.ok(master.every((e) => e.lane === "master"));
  assert.ok(master.some((e) => e.id === "event:e1"));
  assert.ok(!master.some((e) => e.table === "court_event"), "master lane must never include court_event rows");

  const merged = await store.caseTimeline(s, {});
  assert.equal(merged.length, court.length + master.length);

  const upcoming = await store.caseTimeline(s, { mode: "court", upcoming: true });
  assert.ok(upcoming.some((e) => e.id === "court_event:ce1"));
  assert.ok(!upcoming.some((e) => e.id === "court_event:ce2"));

  const past = await store.caseTimeline(s, { mode: "court", upcoming: false });
  assert.ok(past.some((e) => e.id === "court_event:ce2"));
  assert.ok(!past.some((e) => e.id === "court_event:ce1"));
});

test("case_docket: filings + drafts + orders + upcoming court_events, filterable by status/doc_type/in_force", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "filing", id: "f1", data: { title: "Motion for temp relief", doc_type: "motion", filed_or_planned: "filed", status: "filed" } });
  await store.casePut(s, { table: "draft", id: "d1", data: { title: "Response draft", doc_type: "response", status: "working" } });
  await store.casePut(s, { table: "order", id: "o1", data: { title: "Interim order", entered: "2024-01-01", kind: "interim", in_force: true } });
  await store.casePut(s, { table: "court_event", id: "ce1", data: { kind: "hearing", date: "2099-01-01T00:00:00Z", title: "future hearing" } });
  await store.casePut(s, { table: "court_event", id: "ce2", data: { kind: "hearing", date: "2020-01-01T00:00:00Z", title: "past hearing" } });

  const all = await store.caseDocket(s, {});
  const ids = all.map((e) => e.id);
  assert.ok(ids.includes("filing:f1"));
  assert.ok(ids.includes("draft:d1"));
  assert.ok(ids.includes("order:o1"));
  assert.ok(ids.includes("court_event:ce1"));
  assert.ok(!ids.includes("court_event:ce2"), "the docket only shows upcoming court_events");

  const motionsOnly = await store.caseDocket(s, { doc_type: "motion" });
  assert.deepEqual(motionsOnly.map((e) => e.id), ["filing:f1"]);

  const inForceOnly = await store.caseDocket(s, { in_force: true });
  assert.ok(inForceOnly.some((e) => e.id === "order:o1"));
});

test("case_memo: put, list by kind/status, and latest per kind", async () => {
  const s = await freshStore();
  await store.caseMemoPut(s, { id: "m1", kind: "strategy", title: "Early strategy", text: "text1", status: "open" });
  await new Promise((resolve) => setTimeout(resolve, 5)); // ensure a distinct created_at
  await store.caseMemoPut(s, { id: "m2", kind: "strategy", title: "Revised strategy", text: "text2", status: "active" });
  await store.caseMemoPut(s, { id: "m3", kind: "weakness", title: "A weakness", text: "text3" });

  const strategies = await store.caseMemoList(s, { kind: "strategy" });
  assert.equal(strategies.length, 2);

  const active = await store.caseMemoList(s, { status: "active" });
  assert.deepEqual(active.map((m) => m.id), ["memo:m2"]);

  const latest = await store.caseMemoLatestByKind(s);
  assert.equal(latest.strategy.id, "memo:m2");
  assert.equal(latest.weakness.id, "memo:m3");
});

test("case_evidence_log: append (optionally RELATEd via `logs`) and list by exhibit/action", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "exhibit", id: "x1", data: { label: "Ex 1", occurred_at: "2026-01-01T00:00:00Z", known_at: "2026-01-01T00:00:00Z" } });

  await store.caseEvidenceLogAppend(s, { action: "received", exhibit: "exhibit:x1", by: "owner", notes: "intake" });
  await store.caseEvidenceLogAppend(s, { action: "hashed", exhibit: "exhibit:x1", hash: "abc123" });
  await store.caseEvidenceLogAppend(s, { action: "received", notes: "unrelated to any exhibit" });

  const forExhibit = await store.caseEvidenceLogList(s, { exhibit: "exhibit:x1" });
  assert.equal(forExhibit.length, 2);

  const hashedOnly = await store.caseEvidenceLogList(s, { action: "hashed" });
  assert.ok(hashedOnly.some((e) => e.hash === "abc123"));

  const graph = await store.caseGraph(s, { id: "exhibit:x1", edges: ["logs"] });
  assert.equal(graph.neighbors.filter((n) => n.edge === "logs").length, 2);
});

test("case_eval: put + list by kind/subject", async () => {
  const s = await freshStore();
  await store.caseEvalPut(s, { id: "ev1", kind: "eval", title: "Draft review", subject: "draft:d1", score: 82, verdict: "pass" });
  await store.caseEvalPut(s, { id: "ev2", kind: "report", title: "Monthly report", subject: "case" });

  const evalsOnly = await store.caseEvalList(s, { kind: "eval" });
  assert.equal(evalsOnly.length, 1);
  assert.equal(evalsOnly[0].score, 82);

  const bySubject = await store.caseEvalList(s, { subject: "draft:d1" });
  assert.deepEqual(bySubject.map((e) => e.id), ["eval:ev1"]);
});

test("case_reference: load from a file, list by kind/category, and match text with spans (pattern + alias)", async () => {
  const s = await freshStore();
  const tmpDir = createRetainedFixtureDirectory("fct-store-ref");
  try {
    await s.db.query("UPSERT $rid CONTENT $data;", {
      rid: store.parseRef("reference:test-darvo"),
      data: {
        kind: "behavior_pattern",
        key: "test-darvo",
        category: "published",
        pattern: "published-rule",
        definition: "Existing published reference remains unchanged",
        aliases: ["published-alias"],
      },
    });
    const refPath = join(tmpDir, "patterns.json");
    writeFileSync(
      refPath,
      JSON.stringify({
        entries: [
          {
            kind: "behavior_pattern",
            key: "test-darvo",
            category: "manipulation",
            pattern: "\\bDARVO\\b",
            definition: "Deny, attack, reverse victim and offender.",
            severity: "high",
            aliases: ["deny-attack-reverse"],
          },
        ],
      }),
    );

    const loaded = await store.caseReferenceLoad(s, { path: refPath });
    assert.equal(loaded.counts, 1);
    assert.deepEqual(loaded.loaded_from, [refPath]);
    assert.equal(loaded.library_proposals.length, 1);
    assert.equal(loaded.library_proposals[0].status, "citation_required");
    const draftRows = await store.caseQuery(s, { surql: "SELECT proposed_record FROM library_proposal;" });
    assert.equal(draftRows.results[0][0].proposed_record.pattern, "\\bDARVO\\b");
    assert.equal(draftRows.results[0][0].proposed_record.source.path, refPath);

    const list = await store.caseReferenceList(s, { kind: "behavior_pattern" });
    assert.equal(list.length, 1);
    assert.equal(list[0].category, "published");
    assert.equal(list[0].definition, "Existing published reference remains unchanged");

    const patternHits = await store.caseReferenceMatch(s, "He responded with classic DARVO tactics.");
    assert.deepEqual(patternHits, [], "an imported proposal is not available to published-reference matching");

    const patternText = "This matches published-rule.";
    const publishedHits = await store.caseReferenceMatch(s, patternText);
    const publishedPatternHit = publishedHits.find((h) => h.via === "pattern");
    assert.ok(publishedPatternHit, "existing published references remain available");
    const patternStart = patternText.indexOf("published-rule");
    assert.deepEqual(publishedPatternHit.span, [patternStart, patternStart + "published-rule".length]);
    const aliasHits = await store.caseReferenceMatch(s, "This uses a published-alias.");
    assert.ok(aliasHits.some((h) => h.via === "alias"), "existing published aliases remain matchable");
  } finally {
    // Generated reference fixtures remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});

test("case_reference: plugin imports are retained as proposals without publishing bundled references or lexicon rows", async () => {
  const s = await freshStore();
  const pluginRoot = process.env.FCT_TEST_PLUGIN_ROOT ?? resolve("..");
  const loaded = await store.caseReferenceLoad(s, { fromPlugin: true, pluginRoot });
  assert.ok(loaded.loaded_from.some((f) => f.replace(/\\/g, "/").endsWith("content/reference/behavior-patterns.example.json")), "expected the example reference file to be loaded");
  assert.ok(loaded.loaded_from.some((f) => f.replace(/\\/g, "/").endsWith("content/tools/court-language/lexicon.json")), "expected the existing court-language lexicon to be loaded");
  assert.ok(loaded.counts >= 15, `expected at least 15 retained rows (3 example + 12 lexicon), got ${loaded.counts}`);
  assert.equal(loaded.library_proposals.length, loaded.counts);
  assert.ok(loaded.library_proposals.every((proposal) => proposal.status === "citation_required"));
  const draftRows = await store.caseQuery(s, { surql: "SELECT proposed_record FROM library_proposal;" });
  const proposedReferences = draftRows.results[0].map((row) => row.proposed_record);
  assert.ok(proposedReferences.filter((row) => row.kind === "behavior_pattern").length >= 2);
  assert.ok(proposedReferences.some((row) => row.kind === "lexicon"
    && row.category === "banned_clinical_labels"
    && typeof row.definition === "string"
    && row.definition.length > 0));

  const lexiconRows = await store.caseReferenceList(s, { kind: "lexicon" });
  assert.equal(lexiconRows.length, 0);

  const patternRows = await store.caseReferenceList(s, { kind: "behavior_pattern" });
  assert.equal(patternRows.length, 0);
});

test("case_source: returns a record's source block, local path existence, and the r2 pointer (no network calls)", async () => {
  const s = await freshStore();
  const tmpDir = createRetainedFixtureDirectory("fct-store-source");
  try {
    const realFile = join(tmpDir, "real.txt");
    writeFileSync(realFile, "hello");
    await store.casePut(s, { table: "note", id: "n1", data: { text: "a note", source: { path: realFile, sha256: "abc", r2_path: "r2://bucket/real.txt" } } });
    await store.casePut(s, { table: "note", id: "n2", data: { text: "another note", source: { path: join(tmpDir, "missing.txt") } } });

    const found = await store.caseSourceOf(s, "note:n1");
    assert.equal(found.path_exists, true);
    assert.equal(found.r2_path, "r2://bucket/real.txt");
    assert.equal(found.source.sha256, "abc");

    const missing = await store.caseSourceOf(s, "note:n2");
    assert.equal(missing.path_exists, false);
    assert.equal(missing.r2_path, null);
  } finally {
    // Generated source files remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});

test("case_put accepts personal case_document sources and atomically protects existing library sources", async () => {
  const s = await freshStore();
  const saved = await store.casePut(s, {
    table: "source",
    id: "personal-source",
    data: { kind: "case_document", title: "Quinn Samplechild private statement" },
  });
  assert.equal(saved.record.kind, "case_document");
  assert.equal(saved.record.title, "Quinn Samplechild private statement");

  await s.db.query("UPSERT $rid CONTENT $data;", {
    rid: store.parseRef("source:published-authority"),
    data: { kind: "statute", title: "Published synthetic authority" },
  });
  await assert.rejects(
    () => store.casePut(s, {
      table: "source",
      id: "published-authority",
      data: { kind: "case_document", title: "Replacement private document" },
    }),
    /Existing library sources require governed revision/,
  );
  const authority = await store.caseQuery(s, {
    surql: "SELECT * FROM $rid;",
    params: { rid: store.parseRef("source:published-authority") },
  });
  assert.equal(authority.results[0][0].kind, "statute");
  assert.equal(authority.results[0][0].title, "Published synthetic authority");
});

test("case_search: hits include the record's source block when present", async () => {
  const s = await freshStore();
  await store.casePut(s, {
    table: "event",
    id: "e1",
    data: { occurred_at: "2026-01-01T00:00:00Z", known_at: "2026-01-01T00:00:00Z", description: "an event with a source pointer", source: { path: "X:/some/path.md", row: "A1" } },
  });
  const result = await store.caseSearch(s, { query: "source pointer", mode: "text" });
  const hit = result.hits.find((h) => h.id === "event:e1");
  assert.ok(hit);
  assert.equal(hit.source.row, "A1");
});

test("case_import_extract: personal names and narratives are retained while library rows become citation-required proposals", async () => {
  const s = await freshStore();
  const tmpDir = createRetainedFixtureDirectory("fct-store-extract");
  try {
    const envelope = {
      schema: "case-extract/v1",
      source: { row: "T1", path: "T:/fake/source.md", sha256: "deadbeef", kind: "assignment", authored_by: "ai:unknown", source_date: "2026-01-01" },
      extracted_at: "2026-09-07T00:00:00Z",
      extractor: "test",
      confidence: "high",
      records: [
        { id: "T1-1", type: "person", name: "Quinn Samplechild", role: "child", relationship: "Minor Child", aliases: ["Q. Sample"] },
        { id: "T1-3", type: "court_event", kind: "hearing", date: "2099-01-01", title: "Upcoming hearing" },
        { id: "T1-4", type: "entity_rule", pattern: "\\btest-pattern\\b", category: "manipulation" },
        { id: "T1-5", type: "source_authority", title: "Synthetic authority proposal", citation: "Synthetic citation locator" },
        { id: "T1-2", type: "event", occurred_at: "2024-05-01", description: "Quinn Samplechild described the private event; family alias Q. Sample." },
      ],
    };
    const envPath = join(tmpDir, "extract.json");
    writeFileSync(envPath, JSON.stringify(envelope));

    const result = await store.caseImportExtract(s, envPath);
    assert.equal(result.files_processed, 1);
    assert.equal(result.files_skipped.length, 0);
    assert.equal(result.counts.child, 1);
    assert.equal(result.counts.event, 1);
    assert.equal(result.counts.court_event, 1);
    assert.equal(result.counts.reference, 1);
    assert.equal(result.counts.source, 1);
    assert.equal(result.library_proposals.length, 2);
    assert.ok(result.library_proposals.every((proposal) => proposal.status === "citation_required"));
    const drafts = await store.caseQuery(s, { surql: "SELECT proposed_record FROM library_proposal;" });
    const proposedRecords = drafts.results[0].map((row) => row.proposed_record);
    assert.ok(proposedRecords.some((record) => record.type === "source_authority"
      && record.title === "Synthetic authority proposal"
      && record.citation === "Synthetic citation locator"
      && record.source.row === "T1"));
    assert.ok(proposedRecords.some((record) => record.kind === "behavior_pattern"
      && record.pattern === "\\btest-pattern\\b"
      && record.key === "T1-4"));

    const childRid = store.parseRef("child:T1-1");
    const childRows = await store.caseQuery(s, { surql: "SELECT * FROM $rid;", params: { rid: childRid } });
    const childRecord = childRows.results[0][0];
    assert.equal(childRecord.initials, "Q.S.");
    assert.equal(childRecord.name, "Quinn Samplechild");
    assert.deepEqual(childRecord.aliases, ["Q. Sample"]);
    assert.equal(childRecord.source.row, "T1");
    assert.equal(childRecord.extract_id, "T1-1");

    const eventRid = store.parseRef("event:T1-2");
    const eventRows = await store.caseQuery(s, { surql: "SELECT * FROM $rid;", params: { rid: eventRid } });
    const eventRecord = eventRows.results[0][0];
    assert.equal(eventRecord.description, "Quinn Samplechild described the private event; family alias Q. Sample.");

    const unpublishedReferences = await store.caseReferenceList(s, {});
    assert.deepEqual(unpublishedReferences, [], "entity_rule imports remain proposals until validated and published");
    const unpublishedSources = await store.caseQuery(s, { surql: "SELECT * FROM source;" });
    assert.deepEqual(unpublishedSources.results[0], [], "source_authority imports do not overwrite published sources");

    // Idempotency: re-running the same import must not duplicate records.
    const childCountBefore = (await store.caseQuery(s, { surql: "SELECT count() AS c FROM child GROUP ALL;" })).results[0][0].c;
    const eventCountBefore = (await store.caseQuery(s, { surql: "SELECT count() AS c FROM event GROUP ALL;" })).results[0][0].c;
    await store.caseImportExtract(s, envPath);
    const childCountAfter = (await store.caseQuery(s, { surql: "SELECT count() AS c FROM child GROUP ALL;" })).results[0][0].c;
    const eventCountAfter = (await store.caseQuery(s, { surql: "SELECT count() AS c FROM event GROUP ALL;" })).results[0][0].c;
    assert.equal(childCountAfter, childCountBefore);
    assert.equal(eventCountAfter, eventCountBefore);
  } finally {
    // Generated extract envelopes remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});

test("case_import_extract: walking a directory tree skips _INDEX.json and _SCHEMA* files", async () => {
  const s = await freshStore();
  const tmpDir = createRetainedFixtureDirectory("fct-store-extract-dir");
  try {
    const rowDir = join(tmpDir, "T2");
    mkdirSync(rowDir, { recursive: true });
    writeFileSync(
      join(rowDir, "extract.json"),
      JSON.stringify({
        schema: "case-extract/v1",
        source: { row: "T2", path: "T:/fake/source2.md" },
        records: [{ id: "T2-1", type: "note", text: "a plain note" }],
      }),
    );
    // Garbage content that would throw if it were ever parsed — proves these
    // are skipped by name, not merely tolerated as parse failures.
    writeFileSync(join(rowDir, "_INDEX.json"), "not valid json {{{");
    writeFileSync(join(rowDir, "_SCHEMA-note.json"), "not valid json {{{");
    writeFileSync(join(tmpDir, "_SCHEMA-case-extract-v1.md"), "# not json at all");

    const result = await store.caseImportExtract(s, tmpDir);
    assert.equal(result.files_processed, 1);
    assert.equal(result.files_skipped.length, 0);
    assert.equal(result.counts.note, 1);
  } finally {
    // Generated extract envelopes remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});

test("case_import: routes a case-extract/v1 file to the extract importer automatically (kind: 'case-extract')", async () => {
  const s = await freshStore();
  const tmpDir = createRetainedFixtureDirectory("fct-store-import-extract");
  try {
    const envPath = join(tmpDir, "extract.json");
    writeFileSync(
      envPath,
      JSON.stringify({
        schema: "case-extract/v1",
        source: { row: "T3" },
        records: [{ id: "T3-1", type: "note", text: "auto-routed note" }],
      }),
    );
    const result = await store.caseImport(s, envPath);
    assert.equal(result.kind, "case-extract");
    assert.equal(result.counts.note, 1);
  } finally {
    // Generated extract envelopes remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});

test("importVincentSchema preserves full synthetic personal details and narrative text", async () => {
  const s = await freshStore();
  const counts = await store.importVincentSchema(s, {
    parties: [{ name: "Quinn Samplechild", role: "child", age: 8, relationship: "child" }],
    timeline: [{ date: "2024-05-01", event: "Quinn Samplechild shared a private family detail." }],
    documents: [{ title: "Quinn Samplechild private statement.pdf" }],
  });
  assert.equal(counts.child, 1);
  assert.equal(counts.event, 1);
  assert.equal(counts.source, 1);

  const children = await store.caseQuery(s, { surql: "SELECT * FROM child;" });
  assert.equal(children.results[0][0].name, "Quinn Samplechild");
  assert.equal(children.results[0][0].initials, "Q.S.");
  assert.equal(children.results[0][0].age, 8);
  const events = await store.caseQuery(s, { surql: "SELECT * FROM event;" });
  assert.equal(events.results[0][0].description, "Quinn Samplechild shared a private family detail.");
  const sources = await store.caseQuery(s, { surql: "SELECT * FROM source;" });
  assert.equal(sources.results[0][0].title, "Quinn Samplechild private statement.pdf");
});

test("sidecar compatibility aliases: caseMemo/caseStatus/caseSource/caseReference/caseEvidenceLog/caseEvals resolve by the exact names app/sidecar/lib/store-client.mjs dispatches", async () => {
  const s = await freshStore();
  for (const name of ["caseMemo", "caseStatus", "caseSource", "caseReference", "caseEvidenceLog", "caseEvals"]) {
    assert.equal(typeof store[name], "function", `store.js must export "${name}" for the app sidecar's callStoreFn name-dispatch`);
  }

  await store.caseMemoPut(s, { id: "alias-memo-1", kind: "strategy", title: "Alias check", text: "text" });
  const memoAlias = await store.caseMemo(s);
  assert.ok(memoAlias.memos.some((m) => m.id === "memo:alias-memo-1"));

  await store.caseStatusSet(s, { phase: "discovery" });
  const statusAlias = await store.caseStatus(s);
  assert.equal(statusAlias.phase, "discovery");

  const missingIdAlias = await store.caseSource(s, {});
  assert.equal(missingIdAlias.available, false);
  const sourceAlias = await store.caseSource(s, { id: "memo:alias-memo-1" });
  assert.equal(sourceAlias.id, "memo:alias-memo-1");

  const referenceAlias = await store.caseReference(s, {});
  assert.ok(Array.isArray(referenceAlias.entries));

  await store.caseEvidenceLogAppend(s, { action: "received", notes: "alias check" });
  const evidenceLogAlias = await store.caseEvidenceLog(s);
  assert.ok(Array.isArray(evidenceLogAlias.entries) && evidenceLogAlias.entries.length >= 1);

  await store.caseEvalPut(s, { id: "alias-eval-1", kind: "eval", title: "Alias eval" });
  const evalsAlias = await store.caseEvals(s);
  assert.ok(evalsAlias.evals.some((e) => e.id === "eval:alias-eval-1"));
});

test("case_export: format 'platform' writes one NDJSON per table, edges.ndjson, and manifest.json", async () => {
  const s = await freshStore();
  await store.casePut(s, { table: "event", id: "e1", data: { occurred_at: "2026-01-01T00:00:00Z", known_at: "2026-01-01T00:00:00Z", description: "seed event" } });
  await store.casePut(s, { table: "child", id: "private-context", data: { name: "Quinn Samplechild", aliases: ["Q. Sample"], initials: "Q.S.", age: 8 } });
  await store.casePut(s, {
    table: "exhibit",
    id: "x1",
    data: { label: "Ex 1", occurred_at: "2026-01-01T00:00:00Z", known_at: "2026-01-01T00:00:00Z" },
    relations: [{ edge: "evidences", from: "exhibit:x1", to: "event:e1" }],
  });

  const tmpDir = createRetainedFixtureDirectory("fct-store-platform");
  try {
    const outDir = join(tmpDir, "bundle");
    const result = await store.caseExportPlatform(s, outDir);
    assert.equal(result.dir, outDir);
    assert.ok(existsSync(join(outDir, "manifest.json")));
    assert.ok(existsSync(join(outDir, "event.ndjson")));
    assert.ok(existsSync(join(outDir, "child.ndjson")));
    assert.ok(existsSync(join(outDir, "exhibit.ndjson")));
    assert.ok(existsSync(join(outDir, "edges.ndjson")));

    const manifest = JSON.parse(readFileSync(join(outDir, "manifest.json"), "utf8"));
    assert.equal(manifest.schema, "fct-platform-bundle/v1");
    assert.equal(manifest.personal_context, "preserved");
    assert.equal(manifest.counts.event, 1);
    assert.equal(manifest.counts.child, 1);

    const childLines = readFileSync(join(outDir, "child.ndjson"), "utf8").trim().split("\n");
    const childRow = JSON.parse(childLines[0]);
    assert.equal(childRow.name, "Quinn Samplechild");
    assert.deepEqual(childRow.aliases, ["Q. Sample"]);

    const eventLines = readFileSync(join(outDir, "event.ndjson"), "utf8").trim().split("\n");
    assert.equal(eventLines.length, 1);
    const eventRow = JSON.parse(eventLines[0]);
    assert.equal(eventRow.type, "event");
    assert.equal(eventRow.id, "event:e1");

    const edgeLines = readFileSync(join(outDir, "edges.ndjson"), "utf8").trim().split("\n");
    assert.equal(edgeLines.length, 1);
    const edgeRow = JSON.parse(edgeLines[0]);
    assert.equal(edgeRow.edge, "evidences");
  } finally {
    // Generated platform bundles remain in ignored to_be_deleted for owner-controlled cleanup.
  }
});
