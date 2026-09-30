#!/usr/bin/env node
// Byline: Claude Code · Fable 5.1 · 2026-09-07
//
// Reads a Vincent-style case schema (a markdown file with one fenced ```json
// code block: parties/roles, timeline events with evidence cross-refs,
// evidence matrix, legal issues, witnesses, goals, risk matrix, document
// refs, key legal authorities — or a plain .json file with that same shape),
// maps it onto the case-store model, and loads it. Children are ALWAYS
// reduced to { initials, age } — a child's name in the source file is never
// copied into the store (src/store.ts's importVincentSchema() enforces this
// via casePut()'s own child-name rejection).
//
// Usage:
//   node scripts/import-vincent-schema.mjs <path> [--dry-run] [--commit]
//
// Safety: by default this runs in --dry-run mode against a throwaway mem://
// store and only PRINTS the counts it would have written — it never touches
// CUSTODY_CASE_DB unless you pass --commit. This mirrors the owner's
// instruction that seeding the real case store is the owner's decision, not
// something a script does silently.
//
// Default source path (the owner's private intake file, outside the plugin):
//   ~/.config/family-court-toolkit/intake/vincent_case_schema_2025-12-15.md

import { existsSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import * as store from "../dist/store.js";

function defaultSourcePath() {
  return join(homedir(), ".config", "family-court-toolkit", "intake", "vincent_case_schema_2025-12-15.md");
}

function fail(message) {
  console.error(`import-vincent-schema: ${message}`);
  process.exit(1);
}

async function main() {
  const args = process.argv.slice(2);
  const dryRun = args.includes("--dry-run") || !args.includes("--commit");
  const path = args.find((a) => !a.startsWith("--")) ?? defaultSourcePath();

  if (!existsSync(path)) {
    fail(`No file at ${path}. Pass the path to the Vincent-style case schema (.md with a fenced json block, or .json).`);
  }

  const contents = readFileSync(path, "utf8");
  let schema;
  try {
    schema = store.extractVincentJson(contents);
  } catch (err) {
    fail(`Failed to extract/parse the schema JSON from ${path}: ${err.message}`);
  }

  if (dryRun) {
    console.log(`[dry-run] Parsed ${path} — would import into ${process.env.CUSTODY_CASE_DB ?? "the default case store"}.`);
    const s = await store.getStore("mem://");
    if (!s.available) fail(s.reason);
    const counts = await store.importVincentSchema(s, schema);
    console.log(JSON.stringify({ dry_run: true, counts }, null, 2));
    console.log("Re-run with --commit to write this into the real case store (CUSTODY_CASE_DB or the default path).");
    return;
  }

  const s = await store.getStore();
  if (!s.available) fail(s.reason);
  const counts = await store.importVincentSchema(s, schema);
  console.log(JSON.stringify({ dry_run: false, counts }, null, 2));
}

await main();
await store.closeStore();
process.exit(0);
