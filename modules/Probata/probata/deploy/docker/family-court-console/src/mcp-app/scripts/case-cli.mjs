#!/usr/bin/env node
// Byline: Claude Code · Fable 5.1 · 2026-09-07
//
// Node CLI twin for the embedded SurrealDB case store (src/store.ts, built to
// dist/store.js). Python cannot open the embedded RocksDB store concurrently
// with the MCP server process, so this is a Node script, not a Python one.
//
// Usage:
//   node scripts/case-cli.mjs put <table> [id] <json-data> [--relate "edge|from|to[|json-data]"]...
//   node scripts/case-cli.mjs query <surql> [--params <json>] [--write]
//   node scripts/case-cli.mjs search <query> [--tables event,message,note,exhibit] [--k 10] [--mode text|vector|hybrid]
//   node scripts/case-cli.mjs graph <table:id> [--depth 1|2] [--edges e1,e2]
//   node scripts/case-cli.mjs timeline [--from ISO] [--to ISO] [--known-by ISO] [--tables event,message,exhibit]
//   node scripts/case-cli.mjs export [path]
//   node scripts/case-cli.mjs import <path>
//   node scripts/case-cli.mjs summary
//
// Every subcommand accepts CUSTODY_CASE_DB (env) to point at a specific
// rocksdb:// path (or a bare filesystem path, which is prefixed with
// rocksdb:// automatically) instead of the default
// ~/.config/family-court-toolkit/case.db.

import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import * as store from "../dist/store.js";

const here = dirname(fileURLToPath(import.meta.url));

function fail(message) {
  console.error(`case-cli: ${message}`);
  process.exit(1);
}

function parseFlags(args) {
  const positional = [];
  const flags = {};
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg.startsWith("--")) {
      const key = arg.slice(2);
      const next = args[i + 1];
      const value = next !== undefined && !next.startsWith("--") ? (i++, next) : true;
      // Repeated flags (e.g. multiple --relate) accumulate into an array.
      if (key in flags) {
        flags[key] = Array.isArray(flags[key]) ? [...flags[key], value] : [flags[key], value];
      } else {
        flags[key] = value;
      }
    } else {
      positional.push(arg);
    }
  }
  return { positional, flags };
}

async function requireStore() {
  const s = await store.getStore();
  if (!s.available) fail(s.reason);
  return s;
}

async function cmdPut(args) {
  const { positional, flags } = parseFlags(args);
  const [table, maybeIdOrData, maybeData] = positional;
  if (!table) fail("usage: case-cli.mjs put <table> [id] <json-data>");
  let id;
  let dataJson;
  if (maybeData !== undefined) {
    id = maybeIdOrData;
    dataJson = maybeData;
  } else {
    dataJson = maybeIdOrData;
  }
  if (!dataJson) fail("usage: case-cli.mjs put <table> [id] <json-data>");
  const data = JSON.parse(dataJson);

  const relations = [];
  const relateFlags = Array.isArray(flags.relate) ? flags.relate : flags.relate ? [flags.relate] : [];
  for (const spec of relateFlags) {
    // "|" separates the 3-4 parts because "table:id" refs already use ":".
    const [edge, from, to, relJson] = spec.split("|");
    if (!edge || !from || !to) fail(`invalid --relate "${spec}"; expected "edge|from-table:from-id|to-table:to-id[|json-data]"`);
    relations.push({ edge, from, to, data: relJson ? JSON.parse(relJson) : undefined });
  }

  const s = await requireStore();
  const result = await store.casePut(s, { table, id, data, relations: relations.length ? relations : undefined });
  console.log(JSON.stringify(result, null, 2));
}

async function cmdQuery(args) {
  const { positional, flags } = parseFlags(args);
  const [surql] = positional;
  if (!surql) fail("usage: case-cli.mjs query <surql> [--params <json>] [--write]");
  const params = flags.params ? JSON.parse(flags.params) : undefined;
  const s = await requireStore();
  const result = await store.caseQuery(s, { surql, params, write: Boolean(flags.write) });
  console.log(JSON.stringify(result, null, 2));
}

async function cmdSearch(args) {
  const { positional, flags } = parseFlags(args);
  const [query] = positional;
  if (!query) fail("usage: case-cli.mjs search <query> [--tables a,b] [--k N] [--mode text|vector|hybrid]");
  const tables = flags.tables ? String(flags.tables).split(",") : undefined;
  const k = flags.k ? Number.parseInt(String(flags.k), 10) : undefined;
  const mode = flags.mode ? String(flags.mode) : undefined;
  const s = await requireStore();
  const result = await store.caseSearch(s, { query, tables, k, mode });
  console.log(JSON.stringify(result, null, 2));
}

async function cmdGraph(args) {
  const { positional, flags } = parseFlags(args);
  const [id] = positional;
  if (!id) fail("usage: case-cli.mjs graph <table:id> [--depth 1|2] [--edges e1,e2]");
  const depth = flags.depth ? Number.parseInt(String(flags.depth), 10) : undefined;
  const edges = flags.edges ? String(flags.edges).split(",") : undefined;
  const s = await requireStore();
  const result = await store.caseGraph(s, { id, depth, edges });
  console.log(JSON.stringify(result, null, 2));
}

async function cmdFactorMap() {
  const s = await requireStore();
  const result = await store.caseFactorMap(s);
  console.log(JSON.stringify(result, null, 2));
}

async function cmdTimeline(args) {
  const { flags } = parseFlags(args);
  const tables = flags.tables ? String(flags.tables).split(",") : undefined;
  const s = await requireStore();
  const result = await store.caseTimeline(s, {
    from: flags.from,
    to: flags.to,
    known_by: flags["known-by"],
    tables,
  });
  console.log(JSON.stringify(result, null, 2));
}

async function cmdExport(args) {
  const { positional } = parseFlags(args);
  const s = await requireStore();
  const result = await store.caseExport(s, positional[0]);
  console.log(JSON.stringify(result, null, 2));
}

async function cmdImport(args) {
  const { positional } = parseFlags(args);
  const [path] = positional;
  if (!path) fail("usage: case-cli.mjs import <path>");
  const s = await requireStore();
  const result = await store.caseImport(s, path);
  console.log(JSON.stringify(result, null, 2));
}

async function cmdSummary() {
  const s = await requireStore();
  const result = await store.caseSummary(s);
  console.log(JSON.stringify(result, null, 2));
}

async function main() {
  const [, , command, ...rest] = process.argv;
  switch (command) {
    case "put":
      await cmdPut(rest);
      break;
    case "query":
      await cmdQuery(rest);
      break;
    case "search":
      await cmdSearch(rest);
      break;
    case "graph":
      await cmdGraph(rest);
      break;
    case "factor-map":
      await cmdFactorMap();
      break;
    case "timeline":
      await cmdTimeline(rest);
      break;
    case "export":
      await cmdExport(rest);
      break;
    case "import":
      await cmdImport(rest);
      break;
    case "summary":
      await cmdSummary();
      break;
    default:
      console.log(
        [
          "case-cli.mjs — CLI twin for the family-court-toolkit case store",
          "",
          "Subcommands:",
          '  put <table> [id] <json-data> [--relate "edge|from|to[|json-data]"]...',
          "  query <surql> [--params <json>] [--write]",
          "  search <query> [--tables a,b] [--k N] [--mode text|vector|hybrid]",
          "  graph <table:id> [--depth 1|2] [--edges e1,e2]",
          "  factor-map",
          "  timeline [--from ISO] [--to ISO] [--known-by ISO] [--tables event,message,exhibit]",
          "  export [path]",
          "  import <path>",
          "  summary",
          "",
          "Set CUSTODY_CASE_DB to point at a specific database path (bare paths are",
          "prefixed with rocksdb:// automatically); default is",
          "~/.config/family-court-toolkit/case.db.",
        ].join("\n"),
      );
      process.exit(command ? 1 : 0);
  }
}

await main();
// Close the connection before exiting — an abrupt process.exit() against a
// rocksdb:// path can leave its LOCK file held, hanging the NEXT open()
// against the same path (verified live).
await store.closeStore();
process.exit(0);
