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
const { saveSourceNavigation, restoreSourceNavigation, pinnedSourceRef, linkedSourceKey } = module.exports;

test("linked files resolve one recorded key rather than a duplicate basename in another folder", () => {
  assert.equal(linkedSourceKey("recorded/photos/", "same.jpg"), "recorded/photos/same.jpg");
  assert.equal(linkedSourceKey("recorded/photos", "same.jpg"), "recorded/photos/same.jpg");
  assert.equal(linkedSourceKey("recorded/photos/", "recorded/photos/same.jpg"), "recorded/photos/same.jpg");
  assert.equal(linkedSourceKey("recorded/photos/", "other/photos/same.jpg"), null);
});

test("fresh deep links browse the exact folder, while explicit Find file still searches the location", async () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const callbacks = {};
  /** Collect production route initialization and search callbacks for behavior verification.
   * Input: syntax node. Output: captured callbacks. Effects: none; choose to exercise
   * exact linked selection and explicit search together without replacing their predicates.
   */
  function visit(node) {
    if (ts.isVariableDeclaration(node)) {
      const name = node.name.getText(ast);
      if (name === "[appliedFilter, setAppliedFilter]") callbacks.initialFilter = node.initializer.arguments[0];
      if (name === "[selectedRef, setSelectedRef]") callbacks.initialSelection = node.initializer.arguments[0];
      if (name === "selectedObject" || name === "runSearch") callbacks[name] = node.initializer.arguments[0];
      if (name === "listingQuery") callbacks.list = node.initializer.arguments[0].properties.find((p) => p.name?.getText(ast) === "queryFn").initializer;
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.equal(Object.keys(callbacks).length, 5);
  const file = "same.jpg";
  const exact = { source_ref: "b2://fixture/recorded/photos/same.jpg", key: "recorded/photos/same.jpg", name: file, etag: "v2-etag", byte_length: 20 };
  const wrong = { ...exact, source_ref: "b2://fixture/other/photos/same.jpg", key: "other/photos/same.jpg", etag: "other-version-etag", byte_length: 90 };
  const requests = [];
  const context = {
    restored: null, prefix: "recorded/photos/", linkedScope: true, linkedFile: file, selectedRef: file, objects: [wrong, exact], linkedSourceKey,
    mode: "LIVE", rootId: "fixture", query: file,
    listProfferSources: async (request) => { requests.push(request); },
    setAppliedFilter: (value) => { context.appliedFilter = value; },
    setSelectedRef: (value) => { context.selectedRef = value; },
    setPerspectiveChoice() {}, setSelectedFolder() {}, setCheckedRefs() {}, setLocalSource() {}, setScrollRow() {}, setSearchSummary() {},
  };
  const code = Object.entries(callbacks).map(([name, node]) => `globalThis.${name} = ${node.getText(ast)};`).join("\n");
  runInNewContext(ts.transpileModule(code, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  context.appliedFilter = context.initialFilter;
  assert.equal(context.appliedFilter, "");
  await context.list({ pageParam: undefined, signal: undefined });
  assert.equal(requests.at(-1).prefix, "recorded/photos/");
  assert.equal(requests.at(-1).filter, undefined);
  assert.equal(context.selectedObject(), exact);
  context.objects = [wrong];
  assert.equal(context.selectedObject(), null);
  context.objects = [wrong, exact];
  const retained = { ...exact, etag: "retained-v1-etag", byte_length: 10 };
  context.restored = { selectedRef: exact.source_ref, selectedObject: retained, providerVersion: "retained-v1" };
  context.selectedRef = exact.source_ref;
  assert.equal(context.selectedObject(), retained);
  context.runSearch();
  assert.equal(context.selectedRef, null);
  await context.list({ pageParam: undefined, signal: undefined });
  assert.equal(requests.at(-1).prefix, "");
  assert.equal(requests.at(-1).filter, file);
  context.restored = null;
  context.prefix = "";
  context.linkedScope = false;
  runInNewContext(ts.transpileModule(code, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  context.appliedFilter = context.initialFilter;
  context.selectedRef = context.initialSelection;
  assert.equal(context.appliedFilter, file, "a legacy basename-only link keeps location-wide search");
  assert.equal(context.selectedRef, null, "an unscoped name cannot silently select a duplicate");
  await context.list({ pageParam: undefined, signal: undefined });
  assert.equal(requests.at(-1).filter, file);
  assert.equal(context.selectedObject(), null);
  context.linkedScope = true;
  context.objects = [wrong, { ...exact, key: file, source_ref: `b2://fixture/${file}` }];
  runInNewContext(ts.transpileModule(code, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  context.appliedFilter = context.initialFilter;
  context.selectedRef = context.initialSelection;
  assert.equal(context.appliedFilter, "", "an explicit empty prefix scopes to root");
  assert.equal(context.selectedObject().key, file);
});

test("Back restores exact object metadata, version, checked targets and scroll within one route scope", () => {
  const value = { selectedRef: "b2://fixture/source.md", selectedFolder: null, checkedRefs: ["b2://fixture/source.md"], providerVersion: "immutable-v1", query: "source", appliedFilter: "source", scrollRow: 42, aiExportSelection: "b2://fixture/source.md", aiProvider: "gemini_markdown",
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

test("Back keeps v1 pinned until an explicit listed-row selection requests v2 with fresh metadata", async () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let selectObject;
  let selectRow;
  let inspect;
  /** Locate the production selection and inspection callbacks rather than duplicating their conditions.
   * Input: AST node. Output: three callback nodes. Effects: none; choose to catch
   * a restored object or version pin surviving an explicit user selection.
   */
  function visit(node) {
    if (ts.isVariableDeclaration(node) && node.name.getText(ast) === "selectedObject") selectObject = node.initializer.arguments[0];
    if (ts.isVariableDeclaration(node) && node.name.getText(ast) === "inspectionQuery") inspect = node.initializer.arguments[0].properties.find((property) => property.name?.getText(ast) === "queryFn").initializer;
    if (ts.isJsxSelfClosingElement(node) && node.tagName.getText(ast) === "SourceRowsGrid") selectRow = node.attributes.properties.find((attribute) => attribute.name?.getText(ast) === "onSelectIndex").initializer.expression;
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.ok(selectObject && selectRow && inspect);
  const ref = "b2://fixture/source.json";
  const v1 = { source_ref: ref, key: "source.json", name: "source.json", byte_length: 10, etag: "etag-v1" };
  const v2 = { ...v1, byte_length: 20, etag: "etag-v2" };
  const calls = [];
  const context = {
    restored: { selectedRef: ref, selectedObject: v1, providerVersion: "version-v1" }, selectedRef: ref,
    objects: [v2], rows: [{ sourceRef: ref }], linkedFile: "", mode: "LIVE", activeRootId: "fixture",
    setRestored: (value) => { context.restored = value; },
    setSelectedRef: (value) => { context.selectedRef = value; },
    setPerspectiveChoice() {}, setLocalSource() {}, setSelectedFolder() {}, setHandlerOverride() {},
    previewProfferSource: async (object, mode, root, signal, version) => { calls.push({ object, version }); return { provider_version_id: version ?? "version-v2" }; },
  };
  const compiled = ts.transpileModule(`globalThis.selectObject = ${selectObject.getText(ast)}; globalThis.selectRow = ${selectRow.getText(ast)}; globalThis.inspect = ${inspect.getText(ast)};`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(compiled, context);
  context.selectedObject = context.selectObject();
  assert.equal(context.selectedObject, v1);
  await context.inspect({ signal: undefined });
  assert.equal(calls.at(-1).version, "version-v1");
  context.selectRow(0);
  context.selectedObject = context.selectObject();
  assert.equal(context.selectedObject, v2);
  await context.inspect({ signal: undefined });
  assert.equal(calls.at(-1).object, v2);
  assert.equal(calls.at(-1).version, null);
  assert.equal(context.restored, null);
});
