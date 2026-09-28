#!/usr/bin/env node
// Byline: Claude Code · Sonnet 5 · 2026-09-08
//
// "Content lives in the SurrealDB store" — loads the plugin's on-disk content
// (the 193-record verification ledger, the master source directory, the
// court-language lexicon + worked examples/templates, the two survival-guide
// writing templates, and all 17 survival-guide event context packs) into the
// `reference` and `source` tables so src/content-store.ts's getReference()/
// getSources() can serve them, with core.ts/court-language.ts/survival-guide.ts
// falling back to their original direct file reads whenever a row is absent.
//
// This script only WRITES. Reading back through the store happens through
// content-store.ts, exercised by tests/content_store.test.mjs.
//
// Table/key shape (see README-store.md "Content in the store"):
//   - ledger.json entries              -> source:<ledger id>            (all original fields + r2_path/sha256/loaded_at)
//   - master_source_directory.md       -> reference:master-source-directory   (kind "directory", body = whole file)
//   - court-language/lexicon.json      -> reference:court-language-lexicon    (kind "lexicon",   data = parsed JSON)
//   - skills/court-language/EXAMPLES.md  -> reference:court-language-examples   (kind "template",  body = whole file)
//   - skills/court-language/TEMPLATES.md -> reference:court-language-templates  (kind "template",  body = whole file)
//   - survival-guide/TEMPLATE.md       -> reference:survival-guide-template    (kind "template",  body = whole file)
//   - survival-guide/CARD_TEMPLATE.md  -> reference:survival-guide-card-template (kind "template", body = whole file)
//   - survival-guide/events/<id>.json  -> reference:<id>                 (kind "event_pack", data = parsed JSON)
//
// Every write is an idempotent UPSERT (casePut() MERGEs onto a stable id), so
// re-running this script is always safe.
//
// Usage:
//   node scripts/load-content-to-store.mjs [--dry-run]
//
// CUSTODY_CASE_DB (env) selects the target store exactly as every other
// script/tool in this plugin does (default: ~/.config/family-court-toolkit/case.db,
// prefixed rocksdb://; set CUSTODY_CASE_DB=mem:// for a throwaway store, which
// is what tests/content_store.test.mjs does).

import { createHash } from "node:crypto";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { getStore, casePut } from "../dist/store.js";

const here = dirname(fileURLToPath(import.meta.url));
const pluginRoot = join(here, "..", ".."); // scripts/ -> mcp-app/ -> <plugin root>/

function sha256File(path) {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

/**
 * Loads the plugin's on-disk content into the SurrealDB case store.
 *
 * @param {{ dryRun?: boolean, store?: import("../dist/store.js").StoreOk }} [options]
 * @returns {Promise<{ counts: Record<string, number> }>} per-kind row counts
 *   ("source" for the ledger, "directory"/"lexicon"/"template"/"event_pack"
 *   for the reference kinds).
 */
export async function loadContent(options = {}) {
  const dryRun = Boolean(options.dryRun);
  const counts = { source: 0, directory: 0, lexicon: 0, template: 0, event_pack: 0 };

  let store = options.store;
  if (!store) {
    const opened = await getStore();
    if (!opened.available) throw new Error(`Failed to open case store: ${opened.reason}`);
    store = opened;
  }

  async function put(table, id, data, kind) {
    if (dryRun) {
      console.log(`[dry-run] would upsert ${table}:${id}`);
    } else {
      await casePut(store, { table, id, data });
    }
    counts[kind] += 1;
  }

  // 1. ledger.json -> source:<id> (every original field, plus loader provenance).
  const ledgerPath = join(pluginRoot, "content", "toolkit", "ledger.json");
  const ledger = JSON.parse(readFileSync(ledgerPath, "utf8"));
  if (!Array.isArray(ledger)) throw new Error(`${ledgerPath} did not contain an array.`);
  for (const [i, entry] of ledger.entries()) {
    const id = String(entry.id ?? `ledger-${i}`);
    const { id: _drop, ...rest } = entry; // the record identity comes from casePut's `id`, never a duplicate `id` data field
    const relativeSourcePath = typeof entry.file_path === "string" ? entry.file_path : null;
    const localSourcePath = relativeSourcePath
      ? join(pluginRoot, "content", "custody-guide", "sources", "primary", relativeSourcePath)
      : null;
    const r2Path = relativeSourcePath ? `casebible-sorted/fct-sources/primary/${relativeSourcePath}` : null;
    const sha256 = localSourcePath && existsSync(localSourcePath) ? sha256File(localSourcePath) : null;
    await put("source", id, {
      ...rest,
      r2_path: r2Path,
      sha256,
      loaded_at: new Date().toISOString(),
    }, "source");
  }

  // 2. master_source_directory.md -> reference:master-source-directory (kind "directory").
  const masterDirRelPath = join("content", "custody-guide", "master_source_directory.md");
  const masterDirPath = join(pluginRoot, masterDirRelPath);
  const masterDirBody = readFileSync(masterDirPath, "utf8");
  await put("reference", "master-source-directory", {
    kind: "directory",
    key: "master-source-directory",
    body: masterDirBody,
    source_path: masterDirRelPath,
    sha256: sha256File(masterDirPath),
    loaded_at: new Date().toISOString(),
  }, "directory");

  // 3. court-language/lexicon.json -> reference:court-language-lexicon (kind "lexicon").
  const lexiconRelPath = join("content", "tools", "court-language", "lexicon.json");
  const lexiconPath = join(pluginRoot, lexiconRelPath);
  const lexiconData = JSON.parse(readFileSync(lexiconPath, "utf8"));
  await put("reference", "court-language-lexicon", {
    kind: "lexicon",
    key: "court-language-lexicon",
    data: lexiconData,
    source_path: lexiconRelPath,
    sha256: sha256File(lexiconPath),
    loaded_at: new Date().toISOString(),
  }, "lexicon");

  // 4. skills/court-language/EXAMPLES.md + TEMPLATES.md -> reference kind "template".
  const templateFiles = [
    { relPath: join("skills", "court-language", "EXAMPLES.md"), key: "court-language-examples" },
    { relPath: join("skills", "court-language", "TEMPLATES.md"), key: "court-language-templates" },
    { relPath: join("content", "tools", "survival-guide", "TEMPLATE.md"), key: "survival-guide-template" },
    { relPath: join("content", "tools", "survival-guide", "CARD_TEMPLATE.md"), key: "survival-guide-card-template" },
  ];
  for (const { relPath, key } of templateFiles) {
    const path = join(pluginRoot, relPath);
    const body = readFileSync(path, "utf8");
    await put("reference", key, {
      kind: "template",
      key,
      body,
      source_path: relPath,
      sha256: sha256File(path),
      loaded_at: new Date().toISOString(),
    }, "template");
  }

  // 5. survival-guide/events/*.json -> reference:<file stem> (kind "event_pack").
  const eventsRelDir = join("content", "tools", "survival-guide", "events");
  const eventsDir = join(pluginRoot, eventsRelDir);
  const eventFiles = readdirSync(eventsDir).filter((name) => name.endsWith(".json")).sort();
  for (const file of eventFiles) {
    const key = file.slice(0, -5);
    const path = join(eventsDir, file);
    const data = JSON.parse(readFileSync(path, "utf8"));
    await put("reference", key, {
      kind: "event_pack",
      key,
      data,
      source_path: join(eventsRelDir, file),
      sha256: sha256File(path),
      loaded_at: new Date().toISOString(),
    }, "event_pack");
  }

  return { counts };
}

async function main() {
  const dryRun = process.argv.slice(2).includes("--dry-run");
  const { counts } = await loadContent({ dryRun });
  console.log(`load-content-to-store: ${dryRun ? "[dry-run] " : ""}per-kind counts:`);
  for (const [kind, count] of Object.entries(counts)) {
    console.log(`  ${kind.padEnd(10)} ${count}`);
  }
}

// Only run as a CLI when invoked directly (not when imported by a test).
const invokedDirectly = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (invokedDirectly) {
  const { closeAllStoresForTests } = await import("../dist/store.js");
  main()
    .then(async () => {
      await closeAllStoresForTests();
      process.exit(0);
    })
    .catch(async (error) => {
      console.error("load-content-to-store: failed:", error);
      await closeAllStoresForTests().catch(() => undefined);
      process.exit(1);
    });
}
