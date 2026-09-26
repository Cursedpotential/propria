// Byline: Codex · 2026-09-20. Indexed discovery rendering and selection boundary regressions.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
const require = createRequire(import.meta.url);
const source = readFileSync(new URL("../src/components/intake/discovery-explorer.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source + "\nexport { OccurrenceRow, AtomicUnits };", { compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS } }).outputText;
const caps = { backend: "catalog", modes: { filename_substring: true, filename_prefix: true, contents: false, hybrid: false }, tree: false, atomic_unit_catalog: true, unit_types: ["takeout_zip", "git_repo"], coverage: "unknown", limitations: [] };
function load(capabilities) {
  const module = { exports: {} }; let stateCall = 0;
  runInNewContext(compiled, { module, exports: module.exports, require: (name) => name === "@/lib/api-client" ? {} : name === "react" ? { ...React, useState: (initial) => { stateCall++; return React.useState(stateCall === 2 && capabilities ? capabilities : initial); } } : name === "@/components/ui/button" ? { Button: ({ variant, size, ...props }) => React.createElement("button", props) } : require(name) });
  return module.exports;
}
test("indexed catalog is the default and unsupported content modes are disabled", () => {
  const component = load(caps).DiscoveryExplorer;
  const html = renderToStaticMarkup(React.createElement(component, { directStorage: "Direct source fallback" }));
  assert.match(html, /aria-selected="true"[^>]*>Indexed catalog/);
  assert.match(html, /value="contents" disabled=""/);
  assert.match(html, /value="filename_substring" selected=""/);
  assert.match(html, /Bulk intake is unavailable/);
  assert.match(html, /Selecting or exporting does not start ingestion/);
});
test("atomic unit options come from reported real unit types", () => {
  const { AtomicUnits } = load();
  const html = renderToStaticMarkup(React.createElement(AtomicUnits, { unitTypes: ["takeout_zip", "git_repo"] }));
  assert.match(html, /value="takeout_zip"/);
  assert.match(html, /value="git_repo"/);
  assert.doesNotMatch(html, /value="message"/);
});
test("equal names with separate original occurrence IDs are not collapsed", () => {
  const { OccurrenceRow } = load();
  const selected = [];
  for (const id of ["source-a/report.pdf", "source-b/report.pdf"]) {
    const item = { id, rel: id, name: "report.pdf", kind: "file" };
    const element = OccurrenceRow({ item, selected: false, limitReached: false, onToggle: (value) => selected.push(value.id) });
    element.props.children.props.children[0].props.onChange();
  }
  assert.deepEqual(selected, ["source-a/report.pdf", "source-b/report.pdf"]);
});
test("content hits cannot silently become intake selections", () => {
  const { OccurrenceRow } = load();
  const html = renderToStaticMarkup(React.createElement(OccurrenceRow, { item: { id: "chunk1", rel: "path", name: "hit", kind: "content_hit", text: "Content match" }, selected: false, limitReached: false, onToggle() {} }));
  assert.doesNotMatch(html, /type="checkbox"/);
  assert.match(html, /Current source resolution unavailable/);
});
