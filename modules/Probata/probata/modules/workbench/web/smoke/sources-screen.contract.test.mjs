// Byline: Claude Code · Opus 5 · 2026-09-22
// Contract for Sources — the screen that replaces the Intake page's front door
// (ratified 2026-09-22 09:09, option A).
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

function source(path) {
  return readFileSync(new URL(path, import.meta.url), "utf8");
}

const screen = source("../src/components/sources/sources-screen.tsx");
const tree = source("../src/components/sources/source-tree.tsx");
const grid = source("../src/components/sources/source-rows-grid.tsx");
const panel = source("../src/components/sources/source-metadata-panel.tsx");
const search = source("../src/components/sources/source-search.tsx");
const state = source("../src/components/sources/source-state.ts");
const page = source("../src/app/sources/page.tsx");
const router = source("../src/router.tsx");
const navigation = source("../src/surfaces/primary/navigation.ts");
const selector = source("../src/components/intake/matter-mode-selector.tsx");
const intake = source("../src/components/intake/unified-intake.tsx");
const client = source("../src/lib/api-client.ts");
const packageManifest = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));

test("Sources is a route and the navigation entry; Intake bookmarks redirect", () => {
  assert.match(router, /applicationRoute\("sources"/);
  assert.match(router, /legacyRoute\("intake"/);
  assert.match(navigation, /title: "Sources"/);
  assert.match(navigation, /href: "\/sources"/);
  assert.doesNotMatch(navigation, /href: "\/intake"/);
});

test("the landing view is one viewport with three resizable regions", () => {
  assert.match(page, /overflow-hidden/);
  assert.doesNotMatch(page, /overflow-auto/);
  assert.equal(packageManifest.dependencies["react-resizable-panels"], "^3.0.6");
  assert.match(screen, /PanelGroup[\s\S]{0,200}direction="horizontal"/);
  assert.equal((screen.match(/<Panel /g) ?? []).length, 3);
  assert.match(screen, /<SourceTree/);
  assert.match(screen, /<SourceRowsGrid/);
  assert.match(screen, /<SourceMetadataPanel/);
});

test("Glide stays confined to the one row adapter and keeps its pinned version", () => {
  assert.equal(packageManifest.dependencies["@glideapps/glide-data-grid"], "6.0.4-alpha24");
  assert.match(grid, /@glideapps\/glide-data-grid/);
  for (const [name, text] of Object.entries({ screen, tree, panel, search, state })) {
    assert.doesNotMatch(text, /@glideapps\/glide-data-grid/, `${name} must not import Glide directly`);
  }
});

test("state marks come from server facts, never from a file name", () => {
  assert.match(state, /"not_processed" \| "decoded" \| "in_context" \| "failed"/);
  assert.match(client, /\/api\/proffer\/decoded\/exists/);
  assert.match(screen, /getDecodedExists/);
  assert.match(screen, /listProfferProposalResources/);
  // sourceState reads run records and the decode-manifest set only.
  const body = state.slice(state.indexOf("export function sourceState"), state.indexOf("export interface HandlerChoice"));
  assert.doesNotMatch(body, /\.name|extension|endsWith/);
});

test("units are marked on folders and members, and the catalog stays read-only", () => {
  assert.match(client, /\/api\/intake\/discovery\/unit-lookup/);
  assert.match(tree, /unitRoots\.get/);
  assert.match(grid, /unitMark/);
  assert.match(panel, /catalog unit/);
  assert.match(panel, /part of/);
  assert.match(panel, /Mark as unit/);
  assert.match(panel, /catalog(&apos;|')s own unit tables stay read-only/);
  assert.match(client, /\/api\/sources\/unit-marks/);
});

test("Sources uses the shared content retrieval component beside its location filter", () => {
  assert.match(screen, /<ContextSearch/);
  assert.match(screen, /Search content and relationships/);
  assert.match(search, /File names and paths/);
  assert.doesNotMatch(screen, /searchDiscovery|modeAvailable/);
});

test("Process is one button and never auto-starts, and no sort or mark gates it", () => {
  const ast = ts.createSourceFile("sources-screen.tsx", screen, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const buttons = [];
  const processCalls = [];
  /** Inspect the actual Process button and every invocation rather than matching old source formatting.
   * Input: syntax node. Output: collected button/call nodes. Effects: none.
   * Pick to reject auto-starts or new sort/mark gates without replacing the UI predicate in tests.
   */
  function visit(node) {
    if (ts.isJsxElement(node) && node.openingElement.tagName.getText(ast) === "Button" && node.children.some((child) => ts.isJsxText(child) && child.text.trim() === "Process")) buttons.push(node);
    if (ts.isCallExpression(node) && node.expression.getText(ast) === "process") processCalls.push(node);
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.equal(buttons.length, 1, "exactly one Process button");
  const attributes = buttons[0].openingElement.attributes.properties;
  const disabled = attributes.find((attribute) => attribute.name?.getText(ast) === "disabled")?.initializer?.expression;
  const clicked = attributes.find((attribute) => attribute.name?.getText(ast) === "onClick")?.initializer?.expression;
  assert.ok(disabled);
  assert.ok(clicked);
  assert.equal(processCalls.length, 1, "Process is invoked only by its button, never automatically");
  assert.ok(processCalls[0].pos >= clicked.pos && processCalls[0].end <= clicked.end);
  const ready = { processing: false, processTarget: "selected-file", unsupportedMedia: false, hasNativeProcessTarget: false, matter: { id: "existing-matter" }, primaryCourtCase: { id: "existing-case" } };
  const isDisabled = (changes = {}) => runInNewContext(disabled.getText(ast), { ...ready, ...changes });
  assert.equal(isDisabled(), false);
  assert.equal(isDisabled({ processing: true }), true);
  assert.equal(isDisabled({ processTarget: "" }), true);
  assert.equal(isDisabled({ matter: null }), true, "legacy processing requires its actual matter");
  assert.equal(isDisabled({ primaryCourtCase: null }), true, "legacy processing requires its actual court case");
  assert.equal(isDisabled({ hasNativeProcessTarget: true, matter: null, primaryCourtCase: null }), false, "native context processing permits optional case references");
  // No sorting/marking state is supplied: any added dependency also fails evaluation.
  assert.match(screen, /processSelection\(\[entry\], startProffer/);
  assert.match(screen, /submitHumanSource\(/);
  assert.match(screen, /startProfferBatch\(/);
  assert.match(client, /\/api\/proffer\/start-batch/);
  assert.match(client, /\/api\/proffer\/batches\//);
});

test("the Test / Live switch is shown once, in the top bar", () => {
  assert.match(selector, /DEV: "Dev", LIVE: "Live"/);
  assert.doesNotMatch(screen, /MatterModeSelector/);
  assert.doesNotMatch(intake, /MatterModeSelector/);
});
