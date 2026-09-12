// Byline: Claude Code · Sonnet 5 · 2026-09-07
import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { after, test } from "node:test";
import { callStoreFn, loadStoreModule, PLUGIN_ROOT } from "../lib/store-client.mjs";

// Even a `mem://` test store holds a native handle that keeps the event
// loop alive — without an explicit close, `node --test` hangs after every
// test passes (observed 2026-09-07). `--test-force-exit` on the npm script
// is the safety net; this teardown is the polite version of the same fix.
// Tests always use an isolated `mem://` override (see package.json's
// test:sidecar script) — never the shared ws://127.0.0.1:8471 server.
after(async () => {
  const mod = await loadStoreModule();
  await mod.closeAllStoresForTests?.();
});

test("loadStoreModule resolves the BUILT mcp-app/dist/store.js, not src/", async () => {
  const mod = await loadStoreModule();
  assert.equal(typeof mod.caseSummary, "function");
  assert.equal(typeof mod.getStore, "function");
});

test("callStoreFn throws on a genuinely nonexistent export name (no more queued-stub degradation)", async () => {
  await assert.rejects(() => callStoreFn("thisFunctionWillNeverExist"), /does not export/);
});

test("callStoreFn dispatches to the real case_docket/case_memo/case_status/case_source/case_reference/case_evidence_log/case_eval functions", async () => {
  for (const fn of ["caseDocket", "caseMemo", "caseStatus", "caseSource", "caseReference", "caseEvidenceLog", "caseEvals"]) {
    const result = await callStoreFn(fn);
    assert.ok(result !== undefined, `${fn} should return a real result`);
  }
});

test("mcp-app is actually built at the plugin's absolute location (sanity check for the import path this sidecar relies on)", () => {
  // Not a functional assertion about store.ts's contents (out of scope —
  // another agent owns mcp-app/src/*) — just proves the build artifact this
  // sidecar imports from actually exists on disk. Absolute path, not a
  // relative traversal: this app is no longer a sibling of mcp-app/ (moved
  // 2026-09-07, see PLUGIN_ROOT in store-client.mjs).
  const dist = `${PLUGIN_ROOT}\\mcp-app\\dist\\store.js`;
  assert.ok(existsSync(dist), `expected ${dist} to exist — run "npm run build" in mcp-app/`);
});
