// Byline: Codex · 2026-09-20. Source-browser search feedback regression coverage.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";

const require = createRequire(import.meta.url);
const source = readFileSync(new URL("../src/components/intake/source-explorer.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
const module = { exports: {} };
runInNewContext(compiled, {
  module, exports: module.exports,
  require: (name) => name === "@/components/ui/button"
    ? { Button: ({ variant, size, ...props }) => React.createElement("button", props) }
    : require(name),
});
const response = {
  prefixes: [], objects: [], available_roots: [], available_file_types: ["archive", "text"],
  active_root_id: "raw", filter: "invoice", search_complete: false, scanned_count: 500,
  scan_limit_reached: false, is_truncated: true, continuation_token: "next",
};
function render(overrides = {}) {
  return renderToStaticMarkup(React.createElement(module.exports.SourceExplorer, {
    response, loading: false, error: null, rootId: "raw", prefix: "", query: "invoice",
    fileTypes: ["archive"], appliedQuery: "invoice", appliedFileTypes: ["archive"],
    onSearch() {}, onRootChange() {}, onPrefixChange() {}, onQueryChange() {}, onFileTypesChange() {}, onSelect() {}, onLoadMore() {}, ...overrides,
  }));
}
test("search is a submit form with an explicit button and accurate scope", () => {
  const html = render();
  assert.match(html, /<form/);
  assert.match(html, /type="submit"[^>]*>[\s\S]*?Search<\/button>/);
  assert.match(html, /File contents and files inside ZIPs are not searched here/);
});
test("busy searches announce waiting and do not present old results as new matches", () => {
  const html = render({ loading: true });
  assert.match(html, /aria-busy="true"/);
  assert.match(html, /Searching file names and paths — please wait/);
  assert.doesNotMatch(html, /No sources matched/);
  assert.doesNotMatch(html, /Results for/);
});
test("draft terms and filters keep applied results labeled and prevent stale pagination", () => {
  const html = render({ query: "changed", fileTypes: ["text"] });
  assert.match(html, /Filters changed/);
  assert.match(html, /Results for .*invoice.*archive/);
  assert.match(html, /disabled="">Continue search/);
});
test("page continuation is distinct from reaching a scan limit", () => {
  assert.match(render(), /More results may be available/);
  assert.doesNotMatch(render(), /Search is incomplete/);
  assert.match(render({ response: { ...response, scan_limit_reached: true } }), /Scan limit reached/);
  assert.match(render({ response: { ...response, search_complete: true, is_truncated: false } }), /Search complete/);
});

test("Enter/form submission applies once, while busy submission cannot queue another scan", () => {
  const directModule = { exports: {} };
  runInNewContext(compiled, {
    module: directModule, exports: directModule.exports,
    require: (name) => name === "react" ? { ...React, useMemo: (factory) => factory(), useState: (initial) => [typeof initial === "function" ? initial() : initial, () => {}] }
      : name === "@/components/ui/button" ? { Button: "button" } : require(name),
  });
  let submissions = 0;
  let prevented = 0;
  for (const loading of [false, true]) {
    const tree = directModule.exports.SourceExplorer({ response, loading, query: "draft", fileTypes: ["text"], appliedQuery: "invoice", appliedFileTypes: ["archive"], rootId: "raw", prefix: "", onSearch: () => submissions++ });
    const form = tree.props.children.find((element) => element?.type === "form");
    form.props.onSubmit({ preventDefault: () => prevented++ });
  }
  assert.equal(submissions, 1);
  assert.equal(prevented, 2);
});
