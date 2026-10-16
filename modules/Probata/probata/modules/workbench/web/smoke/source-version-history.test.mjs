import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

const module = { exports: {} };
runInNewContext(ts.transpileModule(readFileSync(new URL("../src/components/sources/source-state.ts", import.meta.url), "utf8"), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
}).outputText, { module, exports: module.exports, URL });
const { runsBySource, sourceRunHistory, sourceState } = module.exports;
const source = "b2://fixture/folder/source.json";
const run = (id, ref, lifecycle) => ({ preview_handle: id, source_ref: ref, lifecycle, created_at: `2026-10-15T01:00:0${id.length}Z` });

test("current status joins exactly the inspected provider version and keeps path-only history separate", () => {
  const facts = { runs: runsBySource([
    run("old", source, "completed"),
    run("first", `${source}?versionId=retained%2B%2Fversion`, "failed"),
    run("second", `${source}?versionId=other-version`, "completed"),
  ]), decoded: new Set([source]) };
  const history = sourceRunHistory(source, facts, "retained+/version");
  assert.deepEqual(Array.from(history.exact, (item) => item.preview_handle), ["first"]);
  assert.deepEqual(Array.from(history.unpinned, (item) => item.preview_handle), ["old"]);
  assert.equal(history.otherVersionCount, 1);
  assert.equal(sourceState(source, facts, "retained+/version"), "failed");
  assert.equal(sourceState(source, facts, "other-version"), "in_context");
  assert.equal(sourceState(source, facts, "third-version"), "version_unverified");
  assert.equal(sourceState(source, facts), "version_unverified");
});

test("bare-path completion and decode manifests never imply that the selected version was processed", () => {
  const facts = { runs: runsBySource([run("legacy", source, "completed")]), decoded: new Set([source]) };
  assert.equal(sourceState(source, facts, "current-version"), "version_unverified");
  assert.equal(sourceRunHistory(source, facts, "current-version").exact.length, 0);
  assert.equal(sourceRunHistory(source, facts, "current-version").unpinned.length, 1);
});

test("version history rejects ambiguous selectors and preserves other object boundaries", () => {
  const facts = { runs: runsBySource([
    run("bad", `${source}?versionId=selected&versionId=other`, "completed"),
    run("wrong", `${source}.other?versionId=selected`, "completed"),
    run("unknown", `${source}?unrelated=selected`, "completed"),
  ]), decoded: new Set() };
  assert.equal(sourceRunHistory(source, facts, "selected").exact.length, 0);
  assert.equal(sourceState(source, facts, "selected"), "not_processed");
});
