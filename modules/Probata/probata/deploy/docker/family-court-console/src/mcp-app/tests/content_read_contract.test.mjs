// Byline: Codex · GPT-6 · 2026-10-04
// Bounded source-bundle checks; no native database, full build, or remote writes.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";

const app = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(resolve(process.env.FCT_TEST_ESBUILD_ROOT ?? app, "package.json"));
const { build } = require("esbuild");
const artifacts = resolve(app, "to_be_deleted", "content-read-tests", `${Date.now()}-${process.pid}`);
mkdirSync(artifacts, { recursive: true });
const originalEnv = process.env.CUSTODY_CASE_DB;
let state;

/** Construct a deterministic read-only database stub for a single test.
 * Inputs: fixture mode and optional query implementation. Outputs: mutable test observations.
 * Effects: sets explicit test configuration only; no connection is opened. Pick over native store fixtures for failure checks.
 */
function setup(fixture = false, query = async () => [[]]) {
  process.env.CUSTODY_CASE_DB = fixture ? "mem://" : "ws://test.invalid:8000";
  state = { available: true, reads: [], files: 0, query, fixtureRoot: resolve(app, "..") };
  globalThis.__contentReadTest = state;
  return state;
}

/** Bundle the owned readers with isolated database and observed file adapters.
 * Inputs: none. Outputs: imported readers compiled directly from current TypeScript.
 * Effects: retains a tiny bundle in to_be_deleted; no source/dist or real database writes. Pick over the full app build.
 */
async function readers() {
  const result = await build({
    stdin: { contents: 'export * from "./src/content-store.ts"; export * from "./src/core.ts"; export * from "./src/court-language.ts"; export * from "./src/survival-guide.ts";', resolveDir: app },
    bundle: true, platform: "node", format: "esm", write: false,
    plugins: [{ name: "isolated-read-adapters", setup(builder) {
      builder.onResolve({ filter: /\/store\.js$/ }, () => ({ path: "store", namespace: "test" }));
      builder.onResolve({ filter: /^node:fs$/ }, () => ({ path: "fs", namespace: "test" }));
      builder.onLoad({ filter: /.*/, namespace: "test" }, ({ path }) => ({ contents: path === "store" ? `
        export async function getStore() {
          const s=globalThis.__contentReadTest;
          return s.available ? {available:true,db:{query:async(sql,params)=>{s.reads.push({sql,params}); return s.query(sql,params);}}} : {available:false,reason:'simulated outage'};
        }
        export function parseRef(r) { return r.table+':'+r.id; }
        export function normalize(v) { return v; }
      ` : `
        export function readFileSync(path, encoding) {
          const s=globalThis.__contentReadTest; s.files++;
          const text=String(path).replaceAll('\\\\','/');
          const start=text.indexOf('/content/'); const skill=text.indexOf('/skills/');
          if(start<0 && skill<0) throw new Error('unexpected fixture file');
          return globalThis.__fixtureRead(s.fixtureRoot+text.slice(start>=0?start:skill),encoding);
        }
        export function readdirSync() { return ['referee-hearing.json']; }
      ` }));
    } }],
  });
  const output = resolve(artifacts, "readers.mjs");
  writeFileSync(output, result.outputFiles[0].contents);
  globalThis.__fixtureRead = readFileSync;
  return import(pathToFileURL(output).href);
}

setup();
const api = await readers();
test.after(() => {
  if (originalEnv === undefined) delete process.env.CUSTODY_CASE_DB;
  else process.env.CUSTODY_CASE_DB = originalEnv;
  delete globalThis.__contentReadTest;
  delete globalThis.__fixtureRead;
});

test("remote missing, malformed and unavailable references fail without file fallback", async () => {
  for (const response of [[[]], [[{ body: 42 }]], {}, [[null]]]) {
    setup(false, async () => response);
    await assert.rejects(api.getReference("example"), /Shared reference|Malformed shared/);
    assert.equal(state.files, 0);
  }
  setup(); state.available = false;
  await assert.rejects(api.getReference("example"), /unavailable/);
});

test("only explicit mem fixture absence permits fallback; query errors still fail", async () => {
  setup(true);
  assert.equal(await api.getReference("absent"), null);
  assert.deepEqual(await api.getSources(), []);
  setup(true, async () => { throw new Error("query failed"); });
  await assert.rejects(api.getReference("absent"), /query failed/);
  setup(); delete process.env.CUSTODY_CASE_DB;
  await assert.rejects(api.getReference("absent"), /missing/);
});

test("references and lexicon callers observe each update without reset", async () => {
  let version = "first";
  setup(false, async () => [[{ body: version, data: { entries: [], doc_profiles: { version } } }]]);
  assert.equal((await api.getReference("example")).body, "first");
  assert.equal((await api.loadLexicon()).doc_profiles.version, "first");
  version = "second";
  assert.equal((await api.getReference("example")).body, "second");
  assert.equal((await api.loadLexicon()).doc_profiles.version, "second");
  assert.equal(state.reads.length, 4);
  state.available = false;
  await assert.rejects(api.loadLexicon(), /unavailable/);
});

test("audit caller observes source updates without reset or cached partial success", async () => {
  let title = "first";
  setup(false, async (sql) => sql.includes("FROM source")
    ? [[{ id: "source:example", content_cursor: "example", title }]]
    : [[{ body: "- [Official directory](https://example.invalid)" }]]);
  assert.equal((await api.auditSources()).sources.find(row => row.origin === "ledger").title, "first");
  title = "second";
  assert.equal((await api.auditSources()).sources.find(row => row.origin === "ledger").title, "second");
  state.query = async () => { throw new Error("later outage"); };
  await assert.rejects(api.auditSources(), /later outage/);
});

test("source traversal includes more than 1000 rows with bounded advancing pages", async () => {
  const rows = Array.from({ length: 1201 }, (_, i) => ({ id: `source:${String(i).padStart(5, "0")}`, content_cursor: String(i).padStart(5, "0") }));
  setup(false, async (_sql, params) => [rows.filter(row => !params.after || row.content_cursor > params.after).slice(0, 200).map(row => ({ ...row }))]);
  const result = await api.getSources();
  assert.equal(result.length, 1201);
  assert.equal(state.reads.length, 7);
  assert.ok(state.reads.every(({ sql }) => /LIMIT 200/.test(sql)));
  assert.ok(!("content_cursor" in result[0]));
});

test("source malformed, backward, duplicate, empty and over-budget listings fail visibly", async () => {
  for (const page of [[], [{ id: "source:a", content_cursor: "a" }, { id: "source:a", content_cursor: "a" }], [{ id: "source:z", content_cursor: "z" }, { id: "source:a", content_cursor: "a" }], [{ id: "wrong:a" }]]) {
    setup(false, async () => [page.map(row => ({ ...row }))]);
    await assert.rejects(api.getSources(), /empty|Malformed|nonadvancing/);
  }
  let n = 0;
  setup(false, async () => [Array.from({ length: 200 }, () => {
    const id = `source:${String(n++).padStart(6, "0")}`;
    return { id, content_cursor: id.slice(7) };
  })]);
  await assert.rejects(api.getSources(), /budget/);
  assert.equal(state.reads.length, 51);
  const repeat = Array.from({ length: 200 }, (_, i) => {
    const id = `source:${String(i).padStart(6, "0")}`;
    return { id, content_cursor: id.slice(7) };
  });
  setup(false, async () => [repeat.map(row => ({ ...row }))]);
  await assert.rejects(api.getSources(), /nonadvancing/);
  assert.equal(state.reads.length, 2);
});

test("malformed caller-specific content and missing templates never use packaged files", async () => {
  setup(false, async () => [[{ body: "valid text", data: {} }]]);
  await assert.rejects(api.loadLexicon(), /Malformed/);
  await assert.rejects(api.loadContextPack("referee-hearing"), /Malformed/);
  const context = JSON.parse(readFileSync(resolve(app, "../content/tools/survival-guide/events/referee-hearing.json"), "utf8"));
  context.sources = [];
  state.query = async (_sql, params) => params.rid === "reference:referee-hearing" ? [[{ data: context }]] : [[]];
  await assert.rejects(api.buildSurvivalGuide({ event: "referee-hearing" }), /missing/);
  assert.equal(state.files, 0);
});

test("all callers propagate shared failures without file fallback", async () => {
  setup(false, async () => { throw new Error("shared read failed"); });
  for (const call of [() => api.auditSources(), () => api.searchRecords("rule"), () => api.loadLexicon(), () => api.loadPhrasebanks(), () => api.loadContextPack("referee-hearing"), () => api.buildSurvivalGuide({ event: "referee-hearing" })]) {
    await assert.rejects(call(), /shared read failed/);
  }
  assert.equal(state.files, 0);
});

test("empty explicit mem fixtures preserve packaged caller behavior", async () => {
  setup(true);
  const audit = await api.auditSources();
  assert.ok(audit.source_stats.ledgerLoaded);
  assert.ok((await api.loadLexicon()).entries.length);
  assert.equal((await api.loadContextPack("referee-hearing")).id, "referee-hearing");
  assert.ok(state.files >= 4);
});
