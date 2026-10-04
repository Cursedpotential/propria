// Byline: Codex · GPT-6 · 2026-10-04 — isolated source tests; no database writes or full build.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";

const app = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(resolve(process.env.FCT_TEST_ESBUILD_ROOT ?? app, "package.json"));
const { build } = require("esbuild");
const artifacts = resolve(app, "to_be_deleted", "shared-excerpt-tests", `${Date.now()}-${process.pid}`);
mkdirSync(artifacts, { recursive: true });
const source = "content/custody-guide/draft/M12-pleadings-motions-service.md";
const ref = "reference:custody-guide-draft-m12-pleadings-motions-service-md";
const sha = "a".repeat(64);
let state;

/** Build a read-only projection adapter with source text retained only in test memory.
 * Inputs: source body and optional metadata. Outputs: mutable query/error/race observations.
 * Effects: exposes one test adapter, no live connection/file reads. Pick over getStore for bounded wire-contract fixtures.
 */
function setup(body = "# Guide\nintro\n## 12.5 Timing\r\nOriginal text.\r\n## 12.6 Other\nUnrelated.", overrides = {}) {
  state = { row: { source_path: source, sha256: sha, body, ...overrides }, reads: [], payloads: [], files: 0, afterMetadata: null };
  globalThis.__excerptTest = state;
  state.query = async (sql, params) => {
    state.reads.push({ sql, params });
    if (params.rid === "reference:referee-hearing") return [[{ data: state.context }]];
    const row = params.rid === ref ? state.row : null;
    let rows = [];
    if (row && sql.includes("AS body_chars")) {
      const regex = new RegExp(params.pattern.replace(/^\(\?i\)/, ""), "i");
      const headings = sql.includes("[] AS headings") ? [] : row.body.split("\n").filter(line => regex.test(line)).slice(0, 2).map(line => Array.from(line).slice(0, 512).join(""));
      rows = [{ source_path: row.source_path, source_sha256: row.sha256, body_chars: Array.from(row.body).length, headings }];
      state.afterMetadata?.();
    } else if (row && row.sha256 === params.sha && row.source_path === params.path) {
      const whole = sql.includes("SELECT 1 AS parts");
      const parts = whole ? [row.body] : row.body.split(params.marker);
      rows = [{ parts: whole ? 1 : parts.length, excerpt: Array.from(whole ? row.body : parts[1] ?? "").slice(0, 4097).join("") }];
    }
    state.payloads.push(Buffer.byteLength(JSON.stringify(rows)));
    return [rows];
  };
  return state;
}

/** Compile the two owned readers against isolated store/core/filesystem adapters.
 * Inputs: none. Outputs: imported current source functions.
 * Effects: retains a tiny bundle under to_be_deleted; no dist/native database/full build. Pick over application build tests.
 */
async function readers() {
  const bundle = await build({
    stdin: { contents: 'export * from "./src/content-store.ts"; export * from "./src/survival-guide.ts";', resolveDir: app },
    bundle: true, platform: "node", format: "esm", write: false,
    plugins: [{ name: "excerpt-adapters", setup(builder) {
      builder.onResolve({ filter: /\/store\.js$/ }, () => ({ path: "store", namespace: "fixture" }));
      builder.onResolve({ filter: /\/core\.js$/ }, () => ({ path: "core", namespace: "fixture" }));
      builder.onResolve({ filter: /^node:fs$/ }, () => ({ path: "fs", namespace: "fixture" }));
      builder.onLoad({ filter: /.*/, namespace: "fixture" }, ({ path }) => ({ contents: path === "store" ? `
        export async function getStore(){return {available:true,db:{query:(sql,p)=>globalThis.__excerptTest.query(sql,p)}};}
        export function normalize(v){return v;} export function parseRef(r){return r.table+':'+r.id;}
      ` : path === "core" ? 'export function getCaseFacts(){return {configured:false};}' : `
        export function readdirSync(){return ['referee-hearing.json'];}
        export function readFileSync(){globalThis.__excerptTest.files++;throw new Error('packaged excerpt forbidden');}
      ` }));
    } }],
  });
  const output = resolve(artifacts, "readers.mjs");
  writeFileSync(output, bundle.outputFiles[0].contents);
  return import(pathToFileURL(output).href);
}

setup();
const api = await readers();
test.after(() => { delete globalThis.__excerptTest; });

test("exact loader path and unique heading return untouched located text with hash-version provenance", async () => {
  setup(undefined, { source_path: source.replaceAll("/", "\\") });
  const before = state.row.body;
  const result = await api.getReferenceExcerpt(source, "12.5");
  assert.equal(result.resolution_status, "resolved");
  assert.equal(result.reference_id, ref);
  assert.equal(result.source_sha256, sha);
  assert.equal(result.record_version, sha);
  assert.equal(result.requested_pinpoint, "12.5");
  assert.equal(result.excerpt, "## 12.5 Timing\r\nOriginal text.\r\n");
  assert.equal(result.excerpt_truncated, false);
  assert.equal(state.row.body, before);
  assert.equal(state.files, 0);
});

test("unresolved compound/subparagraph/prose pinpoints and ambiguous headings are empty named gaps", async () => {
  for (const pinpoint of ["12.1-12.7", "12.2, 12.9", "Rule 3.207(B)", "temporary orders context"]) {
    setup();
    const result = await api.getReferenceExcerpt(source, pinpoint);
    assert.equal(result.resolution_status, "pinpoint_gap");
    assert.equal(result.excerpt, "");
    assert.equal(result.gap_reason, "pinpoint_not_resolved");
    assert.equal(state.reads.length, 1);
  }
  setup("## 12.5 One\nfirst\n## 12.5 Two\nsecond");
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).gap_reason, "ambiguous_heading");
  setup("## 12.5 One\nbody repeats ## 12.5 One");
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).gap_reason, "ambiguous_heading_occurrence");
});

test("exact prose headings resolve while numeric prefix collisions and heading budgets stay gaps", async () => {
  setup("## Motions and briefs\nSelected text\n## Motion timing\nOther text");
  const exact = await api.getReferenceExcerpt(source, "Motions and briefs");
  assert.equal(exact.resolution_status, "resolved");
  assert.equal(exact.excerpt, "## Motions and briefs\nSelected text\n");
  setup("## 12.50 Different section\nUnrelated start of file");
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).resolution_status, "pinpoint_gap");
  setup("## 12.5 " + "x".repeat(600) + "\ntext");
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).gap_reason, "heading_budget_exceeded");
});

test("whole-file and late-heading excerpts keep 2.5MB source bodies off the wire", async () => {
  for (const [body, pinpoint] of [["😀".repeat(625000), "whole file"], ["x".repeat(2500000) + "\n## 12.5 Timing\n" + "y".repeat(12000), "12.5"]]) {
    setup(body);
    const result = await api.getReferenceExcerpt(source, pinpoint);
    assert.equal(result.resolution_status, "resolved");
    assert.equal(Array.from(result.excerpt).length, 4096);
    assert.equal(result.excerpt_truncated, true);
    assert.ok(Buffer.byteLength(JSON.stringify(result)) < 18000);
    assert.ok(state.payloads.every(bytes => bytes < 18000));
    assert.ok(state.reads.every(({ sql }) => !/SELECT \*|SELECT body\b/.test(sql)));
    assert.match(state.reads[1].sql, /string::slice\([\s\S]*0, 4097\)/);
  }
});

test("each request observes updates; between-read hash changes or disappearance return an empty source_changed gap", async () => {
  setup();
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).record_version, sha);
  state.row.sha256 = "b".repeat(64);
  state.row.body = "## 12.5 Timing\nUpdated";
  const updated = await api.getReferenceExcerpt(source, "12.5");
  assert.equal(updated.record_version, "b".repeat(64));
  assert.match(updated.excerpt, /Updated/);
  for (const race of [() => { state.row.sha256 = "c".repeat(64); }, () => { state.row = null; }]) {
    setup(); state.afterMetadata = race;
    const result = await api.getReferenceExcerpt(source, "12.5");
    assert.equal(result.resolution_status, "source_changed");
    assert.equal(result.excerpt, "");
    assert.equal(result.record_version, sha);
  }
});

test("missing references, key collisions and unmapped skills are explicit gaps without packaged fallback", async () => {
  setup(); state.row = null;
  assert.equal((await api.getReferenceExcerpt(source, "whole file")).gap_reason, "reference_missing");
  setup(undefined, { source_path: "content/different.md" });
  assert.equal((await api.getReferenceExcerpt(source, "12.5")).gap_reason, "reference_path_mismatch");
  setup();
  const result = await api.getReferenceExcerpt("skills/family-court-toolkit/references/custody-evaluation-summary/SKILL.md", "whole member");
  assert.equal(result.resolution_status, "source_gap");
  assert.equal(result.reference_id, null);
  assert.equal(result.excerpt, "");
  assert.equal(state.reads.length, 0);
  assert.equal(state.files, 0);
});

test("read failures, malformed projection and invalid/budgeted requests remain visible", async () => {
  setup(); state.query = async () => { throw new Error("remote read failed"); };
  await assert.rejects(api.getReferenceExcerpt(source, "12.5"), /remote read failed/);
  setup(undefined, { sha256: "not-a-hash" });
  await assert.rejects(api.getReferenceExcerpt(source, "12.5"), /Malformed/);
  setup(); state.query = async () => [[{ source_path: source, source_sha256: sha, body_chars: 17000000, headings: [] }]];
  await assert.rejects(api.getReferenceExcerpt(source, "whole file"), /budget/);
  await assert.rejects(api.getReferenceExcerpt(source, ""), /Invalid/);
  await assert.rejects(api.getReferenceExcerpt("x".repeat(513), "whole file"), /Invalid/);
  setup();
  const result = await api.getReferenceExcerpt("content/../outside.md", "whole file");
  assert.equal(result.resolution_status, "source_gap");
  assert.equal(state.files, 0);
  setup();
  const original = state.query;
  state.query = (sql, params) => sql.includes("AS body_chars") ? original(sql, params) : [[{ parts: 2, excerpt: "x".repeat(4098) }]];
  await assert.rejects(api.getReferenceExcerpt(source, "12.5"), /Malformed bounded/);
});

test("survival JSON and Markdown expose per-selection provenance/gaps and propagate shared excerpt failures", async () => {
  setup();
  state.context = { id: "referee-hearing", title: "Fixture", sequence: [], prepare: [], applicable_rules: [], deadlines: [], traps: [], do_not: [], phrases: {}, safety_gates: [], exit_checklist: [], sources: [{ file: source, section: "12.5" }, { file: "skills/unmapped/SKILL.md", section: "whole member" }] };
  const result = await api.buildSurvivalGuide({ event: "referee-hearing", format: "json" });
  assert.deepEqual(result.source_excerpt_resolutions.map(r => r.resolution_status), ["resolved", "source_gap"]);
  assert.equal(result.source_excerpts["skills/unmapped/SKILL.md"], "");
  const markdown = api.renderSurvivalGuideMarkdown(result);
  assert.match(markdown, /unsupported_source_path/);
  assert.match(markdown, /source SHA \/ content version/);
  assert.doesNotMatch(markdown, /first ~40/);
  assert.equal(state.files, 0);
  const original = state.query;
  state.query = (sql, params) => params.rid === ref ? Promise.reject(new Error("excerpt unavailable")) : original(sql, params);
  await assert.rejects(api.buildSurvivalGuide({ event: "referee-hearing", format: "json" }), /excerpt unavailable/);
  state.context.sources = Array.from({ length: 33 }, () => ({ file: source, section: "12.5" }));
  await assert.rejects(api.buildSurvivalGuide({ event: "referee-hearing", format: "json" }), /32-source budget/);
});
