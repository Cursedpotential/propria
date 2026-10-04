// Byline: Codex, 2026-10-04.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import test from "node:test";
import { transformSync } from "esbuild";

/** Run the production library view with bounded UI and transport seams.
 * Inputs: a page-list callback; outputs: appended nodes and observed table requests.
 * Side effects: reads and transforms one source function; no browser or datastore is opened.
 * Choose for verifying working-library routing independently of packaged originals.
 */
async function runLibraryView(load) {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("async function viewLibrary():");
  const end = source.indexOf("\n// Byline: Codex, 2026-10-04.\n/** Browse the files", start);
  assert.ok(start >= 0 && end > start, "production view has a bounded source boundary");
  const code = transformSync(source.slice(start, end), { loader: "ts", target: "es2022" }).code;
  const nodes = [];
  const requests = [];
  const host = { append: (...items) => nodes.push(...items) };
  const view = new Function("page", "el", "pagedRecordList", "errBox", "viewPackagedFiles",
    code + "; return viewLibrary;")(
    () => host,
    (tag, attrs, ...children) => ({ tag, attrs, children }),
    async (_host, tables, ...args) => { requests.push(tables); await load(...args); },
    (error) => ({ error: error.message }),
    () => { throw new Error("packaged view must require an explicit click"); },
  );
  await view();
  return { nodes, requests };
}

/** Verify that the Library reads shared source/reference pages instead of packaged files.
 * Inputs: synthetic page loader; outputs: assertion result.
 * Side effects: none outside source-function evaluation.
 * Choose to catch surface drift even when both stores have identical record titles.
 */
test("working Library loads shared reference and source records", async () => {
  const result = await runLibraryView(async () => {});
  assert.deepEqual(result.requests, [["reference", "source"]]);
  assert.equal(result.nodes.some((node) => node.error), false);
  assert.equal(result.nodes[0].children[0].children[0], "Packaged files");
});

/** Verify that a shared-store error remains visible without a file fallback.
 * Inputs: synthetic store failure; outputs: assertion result.
 * Side effects: none; packaged-file loader is never invoked.
 * Choose alongside successful routing to prevent an outage from displaying stale bundled content.
 */
test("working Library shows store failure without opening packaged content", async () => {
  const result = await runLibraryView(async () => { throw new Error("shared library unavailable"); });
  assert.deepEqual(result.requests, [["reference", "source"]]);
  assert.ok(result.nodes.some((node) => node.error === "shared library unavailable"));
});

/** Verify navigation uses the source catalog's official locator and rejects unsafe schemes.
 * Inputs: bounded synthetic source records; outputs: assertion result.
 * Side effects: transforms one pure production helper; no network requests occur.
 * Choose for the catalog field mismatch that previously left official sources without a link.
 */
test("source details prefer official URLs and reject non-web or credential-bearing locators", async () => {
  const source = (await readFile(resolve("web/host.ts"), "utf8")).replace(/\r\n/g, "\n");
  const start = source.indexOf("export function officialSourceUrl(");
  const end = source.indexOf("\nasync function openRecord(", start);
  const code = transformSync(source.slice(start, end), { loader: "ts", format: "cjs" }).code;
  const module = { exports: {} };
  new Function("module", "exports", code)(module, module.exports);
  const url = module.exports.officialSourceUrl;
  assert.equal(url({ official_url: "https://www.courts.michigan.gov/example", url: "https://mirror.example/" }), "https://www.courts.michigan.gov/example");
  assert.equal(url({ official_url: "javascript:alert(1)" }), null);
  assert.equal(url({ official_url: "https://user:secret@example.org/" }), null);
  assert.equal(url({ official_url: "bad URL", source_url: "https://example.org/source" }), "https://example.org/source");
});
