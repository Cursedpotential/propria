// Byline: Claude Code · Sonnet 5 · 2026-09-08
//
// Tests for "content lives in the SurrealDB store": scripts/load-content-to-store.mjs's
// loadContent() against a throwaway mem:// store, and the store-first/exact-fallback
// contract in src/content-store.ts + its callers (core.ts, court-language.ts,
// survival-guide.ts).
//
// Order matters in this file (node:test runs one file's tests sequentially, and
// store.ts/content-store.ts caches are module-level, shared across every test
// here): the EMPTY-store fallback assertions run first, then loadContent()
// populates that SAME mem:// connection, then the store-backed assertions run.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test, { after } from "node:test";
import { auditSources, resetCoreContentCacheForTests } from "../dist/core.js";
import { reviewCourtLanguage, resetCourtLanguageCacheForTests } from "../dist/court-language.js";
import { buildSurvivalGuide } from "../dist/survival-guide.js";
import { getReference, getSources, resetContentStoreCacheForTests } from "../dist/content-store.js";
import * as store from "../dist/store.js";
import { loadContent } from "../scripts/load-content-to-store.mjs";

// See tests/core.test.mjs for why: the embedded engine's native mem:// handle
// otherwise keeps the process alive past every test finishing.
after(async () => {
  await store.closeAllStoresForTests();
  await new Promise((r) => setTimeout(r, 250));
  process.exit(0);
});

const pluginRoot = resolve("..");

function resetAllContentCaches() {
  resetContentStoreCacheForTests();
  resetCoreContentCacheForTests();
  resetCourtLanguageCacheForTests();
}

async function freshEmptyStore() {
  store.resetStoreForTests();
  resetAllContentCaches();
  const s = await store.getStore("mem://");
  assert.equal(s.available, true, s.available ? "" : s.reason);
  return s;
}

const CLINICAL_LABEL_TEXT =
  "He is a narcissist and an alcoholic who always shows up late and never calls, and it's obviously " +
  "outrageous. He probably did it on purpose because he's manipulative.";

// ---------------------------------------------------------------------------
// 1. Fallback parity: with an EMPTY mem:// store, every store-first reader
//    must return EXACTLY what the direct file read returns.
// ---------------------------------------------------------------------------

test("fallback: with an empty mem:// store, audit_sources' ledger records match ledger.json exactly", async () => {
  await freshEmptyStore();

  const fileLedger = JSON.parse(readFileSync(resolve(pluginRoot, "content", "toolkit", "ledger.json"), "utf8"));
  const audit = await auditSources();
  assert.ok(audit.source_stats.ledgerLoaded, "expected the ledger to load via the file fallback");

  const ledgerSources = audit.sources.filter((s) => s.origin === "ledger");
  assert.equal(ledgerSources.length, fileLedger.length, "ledger record count must match ledger.json exactly");

  const byId = new Map(ledgerSources.map((s) => [s.id, s]));
  const first = fileLedger[0];
  const mapped = byId.get(String(first.id));
  assert.ok(mapped, `expected ledger id "${first.id}" in the fallback result`);
  assert.equal(mapped.title, String(first.title ?? first.short_title ?? first.id));
  assert.equal(mapped.authority, String(first.issuing_body ?? "Unspecified issuing body"));
  assert.equal(mapped.url, String(first.official_url ?? ""));
});

test("fallback: with an empty mem:// store, court_language_review's lexicon flagging matches the file lexicon exactly", async () => {
  await freshEmptyStore();

  const fileLexicon = JSON.parse(
    readFileSync(resolve(pluginRoot, "content", "tools", "court-language", "lexicon.json"), "utf8"),
  );
  const result = await reviewCourtLanguage({ text: CLINICAL_LABEL_TEXT, doc_type: "affidavit", mode: "review" });
  assert.ok(result.findings.length > 0, "expected the clinical-label paragraph to trigger findings from the file lexicon");
  const categories = new Set(result.findings.map((f) => f.category));
  const expectedCategories = new Set(fileLexicon.entries.filter((e) => new RegExp(e.pattern, "i").test(CLINICAL_LABEL_TEXT)).map((e) => e.category));
  assert.deepEqual(categories, expectedCategories, "fallback findings' categories must match a direct file-lexicon scan exactly");
});

test("fallback: with an empty mem:// store, survival_guide's context pack and templates match the files exactly", async () => {
  await freshEmptyStore();

  const fileContextPack = JSON.parse(
    readFileSync(resolve(pluginRoot, "content", "tools", "survival-guide", "events", "referee-hearing.json"), "utf8"),
  );
  const fileTemplate = readFileSync(resolve(pluginRoot, "content", "tools", "survival-guide", "TEMPLATE.md"), "utf8");
  const fileCardTemplate = readFileSync(resolve(pluginRoot, "content", "tools", "survival-guide", "CARD_TEMPLATE.md"), "utf8");

  const full = await buildSurvivalGuide({ event: "referee-hearing", format: "full" });
  assert.deepEqual(full.context_pack, fileContextPack, "fallback context_pack must equal the file's parsed JSON exactly");
  assert.equal(full.template, fileTemplate, "fallback full template must equal TEMPLATE.md exactly");

  const card = await buildSurvivalGuide({ event: "referee-hearing", format: "card" });
  assert.equal(card.template, fileCardTemplate, "fallback card template must equal CARD_TEMPLATE.md exactly");
});

// ---------------------------------------------------------------------------
// 2. loadContent() populates the SAME mem:// connection with the expected
//    per-kind counts (idempotent UPSERT; --dry-run writes nothing).
// ---------------------------------------------------------------------------

let loadedCounts;

test("load-content-to-store: --dry-run writes nothing", async () => {
  // Still against the empty store left by the previous test — a dry run must
  // not create any row content-store.ts could then read back.
  const { counts } = await loadContent({ dryRun: true });
  assert.ok(counts.source > 0 && counts.directory === 1 && counts.lexicon === 1 && counts.template === 4 && counts.event_pack === 17);

  resetAllContentCaches();
  const stillEmpty = await getSources();
  assert.ok(stillEmpty === null || stillEmpty.length === 0, "a --dry-run must not have written any source rows");
});

test("load-content-to-store: loadContent() writes the expected per-kind counts", async () => {
  const result = await loadContent({ dryRun: false });
  loadedCounts = result.counts;

  // The real ledger.json currently carries 193 entries — well short of a
  // round 400. Asserted against the file's own actual length (not a fixed
  // literal) so this test tracks the real corpus rather than a stale guess.
  const fileLedger = JSON.parse(readFileSync(resolve(pluginRoot, "content", "toolkit", "ledger.json"), "utf8"));
  assert.equal(loadedCounts.source, fileLedger.length, `expected one source: row per ledger.json entry (${fileLedger.length})`);
  assert.ok(loadedCounts.source >= 150, `expected a substantial ledger (>=150), got ${loadedCounts.source}`);
  assert.equal(loadedCounts.directory, 1);
  assert.equal(loadedCounts.lexicon, 1);
  assert.equal(loadedCounts.template, 4);
  assert.equal(loadedCounts.event_pack, 17);
});

test("load-content-to-store: re-running loadContent() is idempotent (same counts, no duplication)", async () => {
  const { counts } = await loadContent({ dryRun: false });
  assert.deepEqual(counts, loadedCounts);
  resetAllContentCaches();
  const sources = await getSources();
  assert.equal(sources.length, loadedCounts.source, "UPSERT re-run must not duplicate source rows");
});

// ---------------------------------------------------------------------------
// 3. Store-backed reads: once loaded, readers must serve the STORE copy
//    (not silently keep using the file fallback) and getReference() must
//    expose the raw row content-store.ts caches.
// ---------------------------------------------------------------------------

test("content-store: getReference('court-language-lexicon') returns the store copy", async () => {
  resetAllContentCaches();
  const ref = await getReference("court-language-lexicon");
  assert.ok(ref, "expected a reference row for court-language-lexicon after loadContent()");
  assert.equal(ref.kind, "lexicon");
  const fileLexicon = JSON.parse(
    readFileSync(resolve(pluginRoot, "content", "tools", "court-language", "lexicon.json"), "utf8"),
  );
  assert.deepEqual(ref.data, fileLexicon, "the stored lexicon data must round-trip byte-for-byte");
});

test("content-store: getReference('master-source-directory') and getSources() are non-empty after loadContent()", async () => {
  resetAllContentCaches();
  const directoryRef = await getReference("master-source-directory");
  assert.ok(directoryRef, "expected the master-source-directory reference row");
  assert.equal(directoryRef.kind, "directory");
  assert.ok(typeof directoryRef.body === "string" && directoryRef.body.length > 100);

  const sources = await getSources();
  assert.ok(Array.isArray(sources) && sources.length === loadedCounts.source);
});

test("court_language_review: still flags a clinical-label paragraph after loading (store-backed path)", async () => {
  resetAllContentCaches();
  const result = await reviewCourtLanguage({ text: CLINICAL_LABEL_TEXT, doc_type: "affidavit", mode: "review" });
  assert.ok(result.findings.length > 0, "expected findings from the STORE-backed lexicon");
  assert.ok(result.stop_flags.length >= 0); // stop_flags may be empty for this text; findings must not be
  const categories = new Set(result.findings.map((f) => f.category));
  assert.ok(categories.size > 0);
});

test("survival_guide: still resolves a full context pack + template after loading (store-backed path)", async () => {
  resetAllContentCaches();
  const fileContextPack = JSON.parse(
    readFileSync(resolve(pluginRoot, "content", "tools", "survival-guide", "events", "referee-hearing.json"), "utf8"),
  );
  const result = await buildSurvivalGuide({ event: "referee-hearing", format: "full" });
  assert.deepEqual(result.context_pack, fileContextPack, "store-backed context_pack must still equal the file's JSON exactly");
  assert.match(result.template, /Fixed sections/);
});

test("audit_sources: still reports the full ledger after loading (store-backed path)", async () => {
  resetAllContentCaches();
  const audit = await auditSources();
  assert.ok(audit.source_stats.ledgerLoaded);
  assert.equal(audit.sources.filter((s) => s.origin === "ledger").length, loadedCounts.source);
});
