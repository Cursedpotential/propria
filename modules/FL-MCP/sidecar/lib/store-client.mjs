// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Thin loader for the plugin's SurrealDB case store — a SHARED SERVER
// (ws://127.0.0.1:8471, see below), not a file this sidecar owns alone.
// Per the coordinator's boundary: this app NEVER imports mcp-app/src/* directly and
// NEVER edits anything under mcp-app/src/ — that stays owned by whichever
// agent maintains store.ts. We import the BUILT module from
// mcp-app/dist/store.js only.
//
// The store-extension work that was concurrent with this app's own build
// (case_docket, case_memo, case_status, case_source, case_reference,
// case_evidence_log, case_eval, mode-aware case_timeline) landed the SAME
// DAY — including compatibility alias functions in store.ts matching this
// file's exact dispatch names (see store.ts's own "Sidecar compatibility
// aliases" comment). Owner order 2026-09-07 15:29: drop the `queued: true`
// stub machinery that existed to bridge that concurrent-work window — it is
// no longer needed. `callStoreFn` now calls straight through; a missing
// export is a real error (surfaced as a 500 by server.mjs), not a
// gracefully-degraded "coming soon" state.

import { existsSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

// The case store is a SHARED SurrealDB SERVER (owner order 2026-09-07,
// same session as the app move) — NOT a local embedded RocksDB file this
// sidecar owns exclusively. store.ts resolves the connection URL from
// `process.env.CUSTODY_CASE_DB` (see its `resolveDbUrl`), falling back to a
// local `rocksdb://` file path when that env var is unset. We must never
// let that fallback fire in normal operation: set the default here (only
// if the environment hasn't already set something else — an explicit
// `CUSTODY_CASE_DB` from the caller, e.g. a test run's `mem://` override,
// always wins). store.ts itself resolves the server's credentials from
// `~/.secrets/family-court-toolkit.env` — this file only points at the URL.
if (!process.env.CUSTODY_CASE_DB) {
  process.env.CUSTODY_CASE_DB = "ws://127.0.0.1:8471";
}

// ABSOLUTE path (owner order 2026-09-07 15:29, after this app moved out of
// the plugin directory into its own repo at
// E:\AI_Workspace\Projects\the-platform-workspace\family-court-workbench\):
// this app is no longer a sibling of mcp-app/, so the old relative
// `../../mcp-app` traversal from this file's own location would resolve
// into the WRONG tree entirely. The plugin itself lives at a fixed,
// known location on this machine (a personal, single-machine tool — see
// docs/ARCHITECTURE.md), so a hardcoded absolute path is the correct,
// documented choice here, not a fallback of convenience.
export const PLUGIN_ROOT = "C:\\Users\\matts\\.claude\\local-plugins\\plugins\\family-court-toolkit";
const MCP_APP_DIST = join(PLUGIN_ROOT, "mcp-app", "dist");
const STORE_MODULE_PATH = join(MCP_APP_DIST, "store.js");

let storeModulePromise = null;

/** Dynamically imports the BUILT store module (mcp-app/dist/store.js). */
export function loadStoreModule() {
  if (!storeModulePromise) {
    if (!existsSync(STORE_MODULE_PATH)) {
      storeModulePromise = Promise.reject(
        new Error(
          `mcp-app is not built: ${STORE_MODULE_PATH} does not exist. Run "npm run build" in mcp-app/ first.`,
        ),
      );
    } else {
      storeModulePromise = import(pathToFileURL(STORE_MODULE_PATH).href);
    }
  }
  return storeModulePromise;
}

let cachedStore = null;

/** Returns the shared `StoreResult` (StoreOk | StoreErr), opening it once and caching. */
export async function getSharedStore() {
  const mod = await loadStoreModule();
  if (!cachedStore) {
    cachedStore = await mod.getStore();
  }
  return cachedStore;
}

/**
 * Calls `storeModule[fnName](store, ...args)`. A missing export throws
 * (real error, not a queued stub — see file header); an unavailable store
 * (native module failed to load, etc.) still degrades to
 * `{ available: false, reason }` since that is a genuine runtime condition,
 * not a "not implemented yet" one.
 */
export async function callStoreFn(fnName, ...args) {
  const mod = await loadStoreModule();
  if (typeof mod[fnName] !== "function") {
    throw new Error(`store.ts does not export "${fnName}"`);
  }
  const store = await getSharedStore();
  if (store.available === false) {
    return { available: false, reason: store.reason };
  }
  return mod[fnName](store, ...args);
}

export const TIMELINE_MODES = ["court", "master", "merged"];
