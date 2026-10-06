// Byline: Claude Code · Opus 5 · 2026-09-22
// Contract for Sources — the screen that replaces the Intake page's front door
// (ratified 2026-09-22 09:09, option A).
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

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

test("search ships with four modes and one small flag per unreachable mode", () => {
  for (const label of ["Names and paths", "Contents", "Meaning", "Relationships"]) {
    assert.ok(search.includes(label), `missing search mode ${label}`);
  }
  assert.match(search, /not available yet/);
  // One flag on the mode, never a caveat paragraph or a banner.
  assert.doesNotMatch(search, /role="alert"/);
});

test("Process is one button and never auto-starts, and no sort or mark gates it", () => {
  assert.match(screen, /Process\s*<\/Button>/);
  assert.match(screen, /processSelection\(entries, startProffer/);
  assert.match(screen, /startProfferBatch\(/);
  assert.match(client, /\/api\/proffer\/start-batch/);
  assert.match(client, /\/api\/proffer\/batches\//);
  // The only disable reasons are "nothing selected", "already running" and "no case".
  assert.match(screen, /disabled=\{processing \|\| !processTarget \|\| !matter \|\| !primaryCourtCase\}/);
  assert.doesNotMatch(screen, /useEffect\([^)]*\)\s*=>\s*\{\s*void process\(\)/);
});

test("the Test / Live switch is shown once, in the top bar", () => {
  assert.match(selector, /DEV: "Dev", LIVE: "Live"/);
  assert.doesNotMatch(screen, /MatterModeSelector/);
  assert.doesNotMatch(intake, /MatterModeSelector/);
});
