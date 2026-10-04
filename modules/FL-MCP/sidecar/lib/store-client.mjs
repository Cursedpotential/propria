// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Thin loader for the plugin's shared SurrealDB case store, configured through
// the toolkit secrets file or an explicit environment override.
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

import { existsSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

// The plugin is a shared development dependency, so its root can be pointed at
// an explicit checkout while defaulting to the canonical workspace source.
export const DEFAULT_PLUGIN_ROOT = "E:\\AI_Workspace\\plugins\\plugins\\family-court-toolkit";
export const PLUGIN_ROOT = process.env.FAMILY_COURT_PLUGIN_ROOT?.trim() || DEFAULT_PLUGIN_ROOT;

/**
 * Reads only the CUSTODY_CASE_DB assignment from the toolkit's designated
 * secrets file. Input is the file text; output is that value or undefined.
 * It never evaluates the file or exposes unrelated secret values. Use this
 * targeted reader instead of the plugin's broader credential-file scan for
 * desktop URL configuration.
 */
// Byline: Codex · GPT-6 · 2026-10-04
export function readStoreUrlFromSecrets(text) {
  for (const line of text.split(/\r?\n/)) {
    const match = /^\s*(?:export\s+)?CUSTODY_CASE_DB\s*=\s*(.*?)\s*$/.exec(line);
    if (!match) continue;
    let value = match[1];
    if (value.startsWith("\"") || value.startsWith("'")) {
      const quote = value[0];
      const closingQuote = value.indexOf(quote, 1);
      if (closingQuote < 0) return undefined;
      const trailing = value.slice(closingQuote + 1).trim();
      if (trailing && !trailing.startsWith("#")) return undefined;
      value = value.slice(1, closingQuote);
    } else {
      value = value.replace(/\s+#.*$/, "").trim();
    }
    return value || undefined;
  }
  return undefined;
}

/**
 * Resolves the plugin root and shared-store URL before loading the canonical
 * store module. Inputs may provide an environment and secrets text for isolated
 * tests; output contains the root and URL. Only an explicit mem:// environment
 * override may select an embedded test store; production config must name a
 * shared remote endpoint. It never opens a database or logs config values. Use
 * this before loading the canonical store module, not as a database client.
 */
// Byline: Codex · GPT-6 · 2026-10-04
export function resolveStoreConfig({ env = process.env, secretsText } = {}) {
  const pluginRoot = env.FAMILY_COURT_PLUGIN_ROOT?.trim() || DEFAULT_PLUGIN_ROOT;
  const explicitUrl = env.CUSTODY_CASE_DB?.trim();
  if (explicitUrl?.startsWith("mem://")) return { pluginRoot, dbUrl: explicitUrl };

  let dbUrl = explicitUrl;
  if (!dbUrl) {
    if (secretsText === undefined) {
      try {
        secretsText = readFileSync(join(homedir(), ".secrets", "family-court-toolkit.env"), "utf8");
      } catch {
        secretsText = "";
      }
    }
    dbUrl = readStoreUrlFromSecrets(secretsText);
  }

  if (!dbUrl) {
    throw new Error("Missing shared Family Court store configuration. Set CUSTODY_CASE_DB or add it to ~/.secrets/family-court-toolkit.env.");
  }
  let parsedUrl;
  try {
    parsedUrl = new URL(dbUrl);
  } catch {
    // Keep the diagnostic generic so malformed configuration values stay private.
  }
  if (!parsedUrl || !["ws:", "wss:", "http:", "https:"].includes(parsedUrl.protocol) || !parsedUrl.hostname) {
    throw new Error("Desktop Family Court store configuration must be a valid shared ws://, wss://, http://, or https:// URL with a host.");
  }
  return { pluginRoot, dbUrl };
}

/**
 * Validates shared-store configuration and passes its URL to the canonical
 * store module. It reads the process environment and designated secrets file,
 * returns the resolved root and URL, and mutates only CUSTODY_CASE_DB. Use it
 * when loading the built toolkit module so store.ts cannot choose local storage.
 */
// Byline: Codex · GPT-6 · 2026-10-04
function configureSharedStore() {
  const config = resolveStoreConfig();
  process.env.CUSTODY_CASE_DB = config.dbUrl;
  return config;
}

let storeModulePromise = null;

/**
 * Imports the built store module from the configured canonical plugin root.
 * Input: none. Output: the imported module promise. Side effects: validates
 * shared-store configuration and sets CUSTODY_CASE_DB before import. Use this
 * instead of importing plugin source or resolving a second local store.
 */
// Byline: Codex · GPT-6 · 2026-10-04
export function loadStoreModule() {
  if (!storeModulePromise) {
    let storePath;
    try {
      const { pluginRoot } = configureSharedStore();
      storePath = join(pluginRoot, "mcp-app", "dist", "store.js");
    } catch (err) {
      storeModulePromise = Promise.reject(err);
      return storeModulePromise;
    }
    if (!existsSync(storePath)) {
      storeModulePromise = Promise.reject(
        new Error(
          `mcp-app is not built: ${storePath} does not exist. Run "npm run build" in mcp-app/ first.`,
        ),
      );
    } else {
      storeModulePromise = import(pathToFileURL(storePath).href);
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
