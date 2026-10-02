#!/usr/bin/env node
// Byline: Claude Code · Opus 5.5 · 2026-09-27
//
// Owner 2026-09-27 22:02 EDT: "load the reference materials from the tool kit". The earlier
// load-content-to-store.mjs loads only the ledger, the source directory, the lexicon, four templates and
// the 17 event packs. This loader adds every other text document under content/ (cheat sheets,
// checklists, guides, research, council notes, toolkit pages) to the `reference` table, in the same row
// shape: kind, key, title, body, source_path, sha256, loaded_at, plus category and format.
//
// Skipped on purpose: PDFs (the primary sources; their ledger rows already point at them), scripts,
// compiled Python, logs, and every path load-content-to-store.mjs already owns.
// Every write is an idempotent casePut UPSERT on a stable id derived from the path, so re-running is safe.
//
// Usage: node scripts/load-reference-materials.mjs [--dry-run]
// CUSTODY_CASE_DB and its credentials resolve exactly as for every other script in this plugin.

import { createHash } from "node:crypto";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, extname, join, relative, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
// FCT_PLUGIN_ROOT lets a checkout without the built dist/ and node_modules/ run against the installed plugin.
const pluginRoot = process.env.FCT_PLUGIN_ROOT || join(here, "..", "..");
const { getStore, casePut, closeAllStoresForTests } = await import(
  pathToFileURL(join(pluginRoot, "mcp-app", "dist", "store.js")).href
);
const contentRoot = join(pluginRoot, "content");

const TEXT_FORMATS = new Set([".md", ".html", ".json", ".txt", ".csv"]);
const ALREADY_LOADED = new Set([
  "custody-guide/master_source_directory.md",
  "toolkit/ledger.json",
  "tools/court-language/lexicon.json",
  "tools/survival-guide/TEMPLATE.md",
  "tools/survival-guide/CARD_TEMPLATE.md",
]);
const ALREADY_LOADED_DIRS = ["tools/survival-guide/events/"];
// The toolkit's own test inputs are not reference material (they loaded as two fake "references"
// on 2026-09-27 and were removed from the store 2026-10-01; Claude Code · Opus 5.5).
const NOT_REFERENCE_DIRS = ["toolkit/tests/"];

const KIND_BY_FOLDER = [
  ["toolkit/cheatsheet", "cheat_sheet"],
  ["toolkit/checklists", "checklist"],
  ["custody-guide/council", "council_note"],
  ["custody-guide/sources", "source_note"],
  ["custody-guide", "guide"],
  ["toolkit", "toolkit_page"],
  ["reference", "reference_note"],
  ["tools", "tool_note"],
];

function walk(dir) {
  const out = [];
  for (const name of readdirSync(dir)) {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) out.push(...walk(full));
    else out.push(full);
  }
  return out;
}

function kindFor(rel) {
  if (/cheat[-_ ]?sheet/i.test(rel)) return "cheat_sheet";
  for (const [prefix, kind] of KIND_BY_FOLDER) if (rel.startsWith(prefix + "/")) return kind;
  return "document";
}

function keyFor(rel) {
  // The extension stays in the key: some sources exist as both .html and .md under the same name.
  return rel.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 140);
}

function titleFor(rel, body, format) {
  if (format === ".md") {
    const h = body.match(/^#\s+(.+)$/m);
    if (h) return h[1].trim();
  }
  if (format === ".html") {
    const t = body.match(/<title>([^<]+)<\/title>/i);
    if (t) return t[1].trim();
  }
  return rel.split("/").pop();
}

async function main() {
  const dryRun = process.argv.includes("--dry-run");
  const files = walk(contentRoot)
    .map((full) => ({ full, rel: relative(contentRoot, full).split(sep).join("/") }))
    .filter(({ rel }) => TEXT_FORMATS.has(extname(rel).toLowerCase()))
    .filter(({ rel }) => !ALREADY_LOADED.has(rel) && !ALREADY_LOADED_DIRS.some((d) => rel.startsWith(d)))
    .filter(({ rel }) => !NOT_REFERENCE_DIRS.some((d) => rel.startsWith(d)))
    .sort((a, b) => a.rel.localeCompare(b.rel));

  const seen = new Map();
  for (const f of files) {
    const key = keyFor(f.rel);
    if (seen.has(key)) throw new Error(`key collision: ${key} <- ${seen.get(key)} and ${f.rel}`);
    seen.set(key, f.rel);
  }

  let store = null;
  if (!dryRun) {
    const opened = await getStore();
    if (!opened.available) throw new Error(`Failed to open case store: ${opened.reason}`);
    store = opened;
  }

  const counts = {};
  for (const { full, rel } of files) {
    const buf = readFileSync(full);
    const body = buf.toString("utf8");
    const format = extname(rel).toLowerCase();
    const kind = kindFor(rel);
    const key = keyFor(rel);
    const row = {
      kind,
      key,
      title: titleFor(rel, body, format),
      category: rel.includes("/") ? rel.slice(0, rel.lastIndexOf("/")) : "",
      format: format.slice(1),
      body,
      source_path: join("content", rel).split(sep).join("/"),
      sha256: createHash("sha256").update(buf).digest("hex"),
      loaded_at: new Date().toISOString(),
    };
    if (dryRun) console.log(`[dry-run] reference:${key}  (${kind}, ${buf.length} B)  <- ${rel}`);
    else await casePut(store, { table: "reference", id: key, data: row });
    counts[kind] = (counts[kind] ?? 0) + 1;
  }
  console.log(`load-reference-materials: ${dryRun ? "[dry-run] " : ""}${files.length} documents`);
  for (const [kind, n] of Object.entries(counts).sort()) console.log(`  ${kind.padEnd(14)} ${n}`);
}

main()
  .then(async () => { await closeAllStoresForTests(); process.exit(0); })
  .catch(async (error) => {
    console.error("load-reference-materials: failed:", error);
    await closeAllStoresForTests().catch(() => undefined);
    process.exit(1);
  });
