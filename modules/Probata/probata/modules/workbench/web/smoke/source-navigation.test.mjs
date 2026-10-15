import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

const text = readFileSync(new URL("../src/components/sources/source-navigation.ts", import.meta.url), "utf8");
const storage = new Map();
const module = { exports: {} };
runInNewContext(ts.transpileModule(text, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText, {
  module, exports: module.exports, sessionStorage: { getItem: (key) => storage.get(key), setItem: (key, value) => storage.set(key, value) },
  URL,
});
const { saveSourceNavigation, restoreSourceNavigation, pinnedSourceRef } = module.exports;

test("Back restores exact object metadata, version, checked targets and scroll within one route scope", () => {
  const value = { selectedRef: "b2://fixture/source.md", selectedFolder: null, checkedRefs: ["b2://fixture/source.md"], providerVersion: "immutable-v1", query: "source", appliedFilter: "source", scrollRow: 42,
    selectedObject: { source_ref: "b2://fixture/source.md", key: "source.md", name: "source.md", byte_length: 90, etag: '"opaque-etag"' } };
  saveSourceNavigation("LIVE:fixture:folder", value);
  assert.deepEqual(JSON.parse(JSON.stringify(restoreSourceNavigation("LIVE:fixture:folder"))), value);
  assert.equal(restoreSourceNavigation("DEV:fixture:folder"), null);
  assert.equal(restoreSourceNavigation("LIVE:other:folder"), null);
  assert.equal(restoreSourceNavigation("LIVE:fixture:other-folder"), null);
});

test("Malformed session metadata cannot restore an unrelated selected object", () => {
  storage.set("sources-navigation-v1:malformed", JSON.stringify({ selectedRef: "b2://a", selectedFolder: null, checkedRefs: [], providerVersion: null, query: "", appliedFilter: "", scrollRow: 0,
    selectedObject: { source_ref: "b2://b", key: "b", name: "b", byte_length: 9, etag: null } }));
  assert.equal(restoreSourceNavigation("malformed"), null);
});

test("legacy start uses the supported encoded versionId locator without guessing an absent version", () => {
  const original = "b2://bucket/path/source.pdf";
  const pinned = pinnedSourceRef(original, "retained+/version");
  assert.equal(new URL(pinned).searchParams.get("versionId"), "retained+/version");
  assert.equal(new URL(pinned).pathname, "/path/source.pdf");
  assert.equal(pinnedSourceRef(original, null), original);
  assert.throws(() => pinnedSourceRef(`${original}?other=query`, "version"), /unambiguous/);
});
