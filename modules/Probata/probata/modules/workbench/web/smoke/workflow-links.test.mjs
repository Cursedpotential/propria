// Byline: Codex · GPT-6 · 2026-10-06. Bookmark and partial-submission regression proof.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

function load(path) {
  const source = readFileSync(new URL(path, import.meta.url), "utf8");
  const module = { exports: {} };
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText,
    { module, exports: module.exports, URLSearchParams, Error });
  return module.exports;
}
const { canonicalWorkflowHref, attemptReadHref, previewSelectionHref } = load("../src/lib/workflow-links.ts");
const { processSelection } = load("../src/components/sources/process-selection.ts");

test("legacy bookmarks retain exact opaque IDs, policy, search and fragment", () => {
  for (const path of ["/review", "/evidence/preview"]) {
    const url = new URL(canonicalWorkflowHref(path, "?resource=a%2Fb%2Bc&mode=DEV&q=hello+world", "details"), "https://example.test");
    assert.equal(url.pathname, "/read");
    assert.equal(url.searchParams.get("resource"), "a/b+c");
    assert.equal(url.searchParams.get("mode"), "DEV");
    assert.equal(url.searchParams.get("q"), "hello world");
    assert.equal(url.hash, "#details");
  }
  assert.equal(canonicalWorkflowHref("/conversations", "?source=s&thread=t&around=m"), "/read?source=s&thread=t&around=m");
  assert.equal(canonicalWorkflowHref("/intake", "?preview_handle=h"), "/activity?preview_handle=h");
  assert.equal(canonicalWorkflowHref("/", "?root=b2&prefix=folder%2F"), "/sources?root=b2&prefix=folder%2F");
  assert.equal(canonicalWorkflowHref("/m/review", "?resource=h"), "/m/review?resource=h");
  assert.equal(attemptReadHref("a/b+c", "LIVE"), "/read?resource=a%2Fb%2Bc&mode=LIVE");
});

test("partial selection retry preserves accepted receipts and the failed request identity", async () => {
  const entries = ["first", "second", "third"].map((id) => ({ request: { request_id: id } }));
  const calls = [];
  let fail = true;
  const start = async (request) => {
    calls.push(request.request_id);
    if (request.request_id === "second" && fail) throw new Error("Temporary failure");
    return { preview_handle: `handle-${request.request_id}`, matter_mode: "LIVE" };
  };
  const first = await processSelection(entries, start, () => {});
  assert.equal(first.accepted, 1);
  assert.equal(first.error, "Temporary failure");
  fail = false;
  const second = await processSelection(entries, start, () => {});
  assert.equal(second.accepted, 3);
  assert.equal(second.error, null);
  assert.deepEqual(calls, ["first", "second", "second", "third"]);
  await processSelection(entries, start, () => {});
  assert.equal(calls.length, 4);
});

test("repair attempt selection keeps the source, message and query to return to", () => {
  const target = new URL(previewSelectionHref("?source=s%2F1&thread=t&around=m&q=term&attempt=old&mode=DEV", "new/a+b", "LIVE"), "https://example.test");
  assert.equal(target.pathname, "/read");
  assert.equal(target.searchParams.get("resource"), "new/a+b");
  assert.equal(target.searchParams.get("attempt"), null);
  assert.equal(target.searchParams.get("source"), "s/1");
  assert.equal(target.searchParams.get("thread"), "t");
  assert.equal(target.searchParams.get("around"), "m");
  assert.equal(target.searchParams.get("q"), "term");
  assert.equal(target.searchParams.get("mode"), "LIVE");
});
