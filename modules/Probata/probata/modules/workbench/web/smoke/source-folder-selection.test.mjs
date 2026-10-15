import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

const require = createRequire(import.meta.url);
/** Load the production folder unit without starting a browser or calling any API.
 * Inputs: source filename and pure dependency overrides. Output: production exports.
 * Effects: source read only; choose for callback behavior independent of browser rendering.
 */
function load(name) {
  const source = readFileSync(new URL(`../src/components/sources/${name}`, import.meta.url), "utf8");
  const module = { exports: {} };
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX } }).outputText, {
    module, exports: module.exports, require: (id) => id === "@/lib/utils" ? { cn: (...values) => values.filter(Boolean).join(" ") } : require(id),
  });
  return module.exports;
}
const { SourceTree } = load("source-tree.tsx");
const { folderSelectionPlan, markFolderUnits, sourceRootMarks, toggleFolderSelection, unitMarkIdentity } = load("folder-selection.ts");

/** Walk actual React elements to inspect and invoke their production control callbacks.
 * Input: rendered element. Output: flat element list. Effects: none; choose for
 * row/checkbox/arrow semantics without duplicating the component's handlers.
 */
function elements(node) {
  if (!node || typeof node !== "object") return [];
  if (Array.isArray(node)) return node.flatMap(elements);
  return [node, ...elements(node.props?.children)];
}

test("folder name inspects without navigating, checkbox includes a unit and compact arrow opens it", () => {
  const calls = [];
  const prefix = "SourceCorpus/_Incoming/";
  const phone = `${prefix}+18102801581 - Adam/`;
  const nodes = elements(SourceTree({ roots: [], activeRootId: "vault", prefix, prefixes: [{ name: "+18102801581 - Adam", prefix: phone }], loading: false,
    selectedFolder: null, selectedFolders: new Set(), unitRoots: new Map(), onRootChange() {},
    onSelectFolder: (folder) => calls.push(["inspect", folder]), onPrefixChange: (folder) => calls.push(["browse", folder]),
    onToggleFolder: (folder, checked) => calls.push(["include", folder, checked]),
  }));
  nodes.find((n) => n.props?.["aria-label"] === "Select folder +18102801581 - Adam").props.onClick();
  assert.deepEqual(calls, [["inspect", phone]]);
  nodes.find((n) => n.props?.["aria-label"] === "Include folder +18102801581 - Adam").props.onChange({ target: { checked: true } });
  nodes.find((n) => n.props?.["aria-label"] === "Open folder +18102801581 - Adam").props.onClick();
  assert.deepEqual(calls.slice(1), [["include", phone, true], ["browse", phone]]);
  assert.ok(!nodes.some((n) => n.type === "button" && n.props.children === "Select"));
});

test("the current folder remains selectable at nested depth even when there are no child folders", () => {
  const prefix = "SourceCorpus/_Incoming/+18102801581 - Adam/attachments/";
  let selected;
  const nodes = elements(SourceTree({ roots: [], activeRootId: "vault", prefix, prefixes: [], loading: false, selectedFolder: null,
    selectedFolders: new Set(), unitRoots: new Map(), onRootChange() {}, onPrefixChange() { throw Error("current folder must not navigate"); },
    onSelectFolder: (folder) => { selected = folder; }, onToggleFolder() {},
  }));
  nodes.find((n) => n.props?.["aria-label"] === "Select current folder attachments").props.onClick();
  assert.equal(selected, prefix);
});

test("three phone folders keep separate rooted identities, and overlapping roots are disclosed", () => {
  const folders = ["SourceCorpus/_Incoming/+1 - Adam/", "SourceCorpus/_Incoming/+2 - Beth/", "SourceCorpus/_Incoming/+3 - Chris/"];
  const plan = folderSelectionPlan("b2://bucket/vault/", [...folders, folders[0]]);
  assert.equal(plan.units.length, 3);
  assert.equal(plan.issue, null);
  assert.equal(plan.folderRef, "b2://bucket/vault/SourceCorpus/_Incoming/");
  assert.deepEqual(Array.from(plan.units, (unit) => unit.sourceRef), folders.map((folder) => `b2://bucket/vault/${folder}`));
  assert.match(folderSelectionPlan("b2://bucket/vault", [folders[0], `${folders[0]}attachments/`]).issue, /overlap/);
  assert.equal(folderSelectionPlan("", folders).units[0].sourceRef, null);
  assert.match(folderSelectionPlan("b2://bucket/vault", Array.from({ length: 33 }, (_, index) => `Incoming/${index}/`)).issue, /32/);
});

test("bulk marking invokes the existing mark API once per folder and retains successes on a partial failure", async () => {
  const calls = [];
  const folders = ["Incoming/Adam/", "Incoming/Beth/", "Incoming/Chris/"];
  const result = await markFolderUnits("vault", folders, "other", false, async (request) => {
    calls.push(request);
    if (request.label === "Beth") throw Error("fixture rejection");
    return { unit_root: request.unit_root.replace(/\/$/, ""), unit_type: request.unit_type, label: request.label };
  });
  assert.deepEqual(calls.map((call) => call.unit_root), folders);
  assert.equal(result.marks.length, 2);
  assert.equal(result.failures.length, 1);
  assert.match(result.failures[0], /Beth/);
  assert.ok(calls.every((call) => call.root_id === "vault" && call.unit_type === "other" && call.confirm === false));
});

test("screen folder selection and file selection never leave each other's Process targets armed", () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const handlers = {};
  /** Collect actual folder and file selection handlers from their component props.
   * Input: syntax node. Output: captured callbacks. Effects: none; choose to catch stale mixed targets.
   */
  function visit(node) {
    if (ts.isJsxSelfClosingElement(node) && ["SourceTree", "SourceRowsGrid"].includes(node.tagName.getText(ast))) {
      for (const attribute of node.attributes.properties) {
        const name = attribute.name?.getText(ast);
        if (["onSelectFolder", "onToggleFolder", "onSelectIndex", "onToggleChecked"].includes(name)) handlers[name] = attribute.initializer.expression;
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
  const state = { selectedFolders: new Set(), checkedRefs: new Set(["old-file"]), selectedRef: "old-file", handlerOverride: "smsbackuprestore_xml" };
  const context = { rows: [{ sourceRef: "new-file" }],
    get selectedFolders() { return state.selectedFolders; }, toggleFolderSelection,
    setSelectedFolders: (value) => { state.selectedFolders = typeof value === "function" ? value(state.selectedFolders) : value; },
    setCheckedRefs: (value) => { state.checkedRefs = value; }, setSelectedRef: (value) => { state.selectedRef = value; },
    setSelectedFolder(value) { state.inspected = value; }, setPerspectiveChoice() {}, setLocalSource() {}, setUnitMarkKind() {}, setShowContentSearch() {}, setRestored() {}, setHandlerOverride(value) { state.handlerOverride = value; },
  };
  runInNewContext(ts.transpileModule(Object.entries(handlers).map(([name, node]) => `globalThis.${name}=${node.getText(ast)};`).join("\n"), { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  for (const folder of ["Incoming/Adam/", "Incoming/Beth/", "Incoming/Chris/"]) context.onToggleFolder(folder, true);
  assert.equal(state.selectedFolders.size, 3);
  assert.equal(state.checkedRefs.size, 0);
  assert.equal(state.selectedRef, null);
  assert.equal(state.handlerOverride, "", "folder checkbox clears the previous file handler");
  state.handlerOverride = "smsbackuprestore_xml";
  context.onSelectFolder("Incoming/Beth/");
  assert.equal(state.handlerOverride, "", "folder name clears the previous file handler");
  assert.equal(state.selectedFolders.size, 3, "inspecting an included folder keeps the existing bulk selection");
  context.onSelectIndex(0);
  assert.equal(state.selectedFolders.size, 0);
  assert.equal(state.selectedRef, "new-file");
  context.onToggleFolder("Incoming/Adam/", true);
  context.onToggleChecked(["new-file"]);
  assert.equal(state.selectedFolders.size, 0);
  assert.deepEqual(Array.from(state.checkedRefs), ["new-file"]);
  context.onToggleFolder("Incoming/Adam/", true);
  context.onToggleFolder("Incoming/Beth/", true);
  context.onToggleFolder("Incoming/Beth/", false);
  assert.equal(state.inspected, "Incoming/Adam/", "inspection follows a remaining checked folder");
  context.onToggleFolder("Incoming/Adam/", false);
  assert.equal(state.selectedFolders.size, 0, "unchecking the final folder keeps zero included units");
});

test("mark display and merge identity separate storage roots, reconfigured roots and unscoped history", () => {
  const base = { unit_root: "Incoming/Phone", unit_type: "other", identity_scope: "root_scoped", root_id: "vault", source_root_ref: "b2://fixture/vault/" };
  const other = { ...base, root_id: "other", source_root_ref: "b2://fixture/other/" };
  const reconfigured = { ...base, source_root_ref: "b2://fixture/new-vault/" };
  const legacy = { ...base, identity_scope: "legacy_unscoped", root_id: null, source_root_ref: null };
  const result = sourceRootMarks([base, other, reconfigured, legacy], "vault", "b2://fixture/vault/");
  assert.deepEqual(Array.from(result.scoped), [base]);
  assert.deepEqual(Array.from(result.legacy), [legacy]);
  assert.equal(new Set([base, other, reconfigured, legacy].map(unitMarkIdentity)).size, 4);
});

test("actual mark callback cannot write an inspected but unchecked folder", async () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let callback;
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && node.name?.text === "markUnit") callback = node;
    ts.forEachChild(node, visit);
  }
  visit(ast);
  const calls = [];
  const context = { selectedFolders: new Set(), selectedFolder: "Incoming/Unchecked/", activeRootId: "vault", activeRootRef: "b2://fixture/vault/",
    markKind: "other", markFolderUnits, unitMarkIdentity, errorText: (error) => error.message,
    setMarkPending() {}, setMarkError() {}, setLocalMarks(update) { update([]); },
    recordSourceUnitMark: async (request) => { calls.push(request); return { ...request, root_id: "vault", source_root_ref: "b2://fixture/vault/", identity_scope: "root_scoped" }; },
  };
  runInNewContext(ts.transpileModule(`${callback.getText(ast)}; globalThis.mark=markUnit;`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  await context.mark(false);
  assert.equal(calls.length, 0);
  context.selectedFolders.add("Incoming/Checked/");
  await context.mark(false);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].unit_root, "Incoming/Checked/");
  assert.equal(calls[0].root_id, "vault");
});

test("actual mark control stays disabled with zero included folders or an unresolved source root", () => {
  const source = readFileSync(new URL("../src/components/sources/source-metadata-panel.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("source-metadata-panel.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let disabled;
  function visit(node) {
    if (ts.isJsxOpeningElement(node) && node.tagName.getText(ast) === "Button") {
      const attributes = node.attributes.properties;
      if (attributes.some((attribute) => attribute.name?.getText(ast) === "onClick" && attribute.getText(ast).includes("onMarkUnit(needsConfirm)"))) {
        disabled = attributes.find((attribute) => attribute.name?.getText(ast) === "disabled").initializer.expression;
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.ok(disabled);
  const screen = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const screenAst = ts.createSourceFile("sources-screen.tsx", screen, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let count;
  function screenVisit(node) {
    if (ts.isJsxAttribute(node) && node.name.getText(screenAst) === "selectedFolderCount") count = node.initializer.expression;
    ts.forEachChild(node, screenVisit);
  }
  screenVisit(screenAst);
  const context = { activeRootRef: "b2://fixture/vault/", selectedFolders: new Set(), selectedFolder: "Incoming/Unchecked/", markPending: false };
  const code = `globalThis.count = () => ${count.getText(screenAst)}; globalThis.disabled = () => ${disabled.getText(ast)};`;
  runInNewContext(code, context);
  context.selectedFolderCount = context.count();
  assert.equal(context.disabled(), true);
  context.selectedFolders.add("Incoming/Checked/");
  context.selectedFolderCount = context.count();
  assert.equal(context.disabled(), false);
  context.activeRootRef = "";
  context.selectedFolderCount = context.count();
  assert.equal(context.disabled(), true);
});

test("actual browse callback leaves departing route selection intact for Back restoration", () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let callback;
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && node.name?.text === "browse") callback = node;
    ts.forEachChild(node, visit);
  }
  visit(ast);
  const calls = [];
  const context = { mode: "LIVE", URLSearchParams, router: { navigate: (request) => calls.push(request.href) },
    clearSearch() { throw Error("must not clear departing route before Back"); } };
  runInNewContext(ts.transpileModule(`${callback.getText(ast)}; globalThis.open=browse;`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  context.open("vault", "Incoming/Phone/");
  context.open("other", "");
  assert.equal(calls[0], "/sources?root=vault&prefix=Incoming%2FPhone%2F&mode=LIVE");
  assert.equal(calls[1], "/sources?root=other&prefix=&mode=LIVE");
});

test("Process sends three selected units through one durable package batch and keeps retry identity", async () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let callback;
  /** Find the production Process function rather than copying its folder dispatch branch.
   * Input: syntax node. Output: callback. Effects: none; choose for request-shape and no-flat-fanout proof.
   */
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && node.name?.text === "process") callback = node;
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.ok(callback);
  const folders = ["Incoming/Ada/", "Incoming/Beth/", "Incoming/Chris/"];
  const calls = [];
  const errors = [];
  const context = {
    processBusy: { current: false }, showContentSearch: false, processTarget: "3 separate folder units",
    folderPlan: folderSelectionPlan("b2://fixture/vault/", folders), unsupportedMedia: false, aiProcessBlocked: false, hasAIProcessTarget: false,
    matter: { id: "actual-matter" }, primaryCourtCase: { id: "actual-case" }, localSource: null, selectedFiles: [], selectedObject: null,
    selectedFolders: new Set(folders), listing: {}, mode: "LIVE", handlerOverride: "", batchRequests: { current: new Map() },
    newBatchId: () => "batch-fixed-package-selection",
    setProcessing() {}, setProcessMessage() {}, setUploadProgress() {}, setBatchIdent() {}, setActivityHref() {},
    setProcessError: (error) => { if (error) errors.push(error); }, errorText: (error) => error.message,
    startProfferBatch: async (request) => { calls.push(request); return { batch_id: request.batch_id, matter_mode: "LIVE" }; },
    startProffer: () => { throw Error("flat file processing is forbidden for selected units"); }, URLSearchParams,
  };
  runInNewContext(ts.transpileModule(`${callback.getText(ast)}; globalThis.dispatch=process;`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);
  await context.dispatch();
  assert.equal(errors.length, 0);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].folder_ref, "b2://fixture/vault/Incoming/");
  assert.deepEqual(Array.from(calls[0].unit_roots), folders.map((folder) => `b2://fixture/vault/${folder}`));
  assert.equal(calls[0].declared_format, "unknown_binary", "folder names do not fabricate an export parser");
  await context.dispatch();
  assert.equal(calls.length, 2);
  assert.equal(calls[0].batch_id, calls[1].batch_id);
  context.folderPlan = { ...context.folderPlan, issue: "Selected folders overlap" };
  await context.dispatch();
  assert.equal(calls.length, 2);
});
