// Byline: Codex · GPT-6 · 2026-10-04
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { DEFAULT_PLUGIN_ROOT, resolveStoreConfig } from "../lib/store-client.mjs";

test("default desktop configuration reads the shared URL from toolkit secrets", () => {
  const config = resolveStoreConfig({
    env: {},
    secretsText: "export CUSTODY_CASE_DB=\"wss://shared-store.example.test/rpc\" # shared endpoint\nUNRELATED_TOKEN=private-value",
  });

  assert.equal(config.pluginRoot, "E:\\AI_Workspace\\plugins\\plugins\\family-court-toolkit");
  assert.equal(config.pluginRoot, DEFAULT_PLUGIN_ROOT);
  assert.equal(config.dbUrl, "wss://shared-store.example.test/rpc");
});

test("desktop configuration accepts an explicitly selected canonical plugin checkout", () => {
  const config = resolveStoreConfig({
    env: { FAMILY_COURT_PLUGIN_ROOT: "F:\\toolkit-checkout" },
    secretsText: "CUSTODY_CASE_DB=ws://shared-store.example.test:8471",
  });

  assert.equal(config.pluginRoot, "F:\\toolkit-checkout");
});

test("secrets parser accepts single-quoted URLs with trailing comments", () => {
  const config = resolveStoreConfig({
    env: {},
    secretsText: "CUSTODY_CASE_DB='ws://shared-store.example.test:8471' # managed endpoint",
  });

  assert.equal(config.dbUrl, "ws://shared-store.example.test:8471");
});

test("missing shared-store configuration fails instead of selecting embedded storage", () => {
  assert.throws(
    () => resolveStoreConfig({ env: {}, secretsText: "" }),
    /Missing shared Family Court store configuration/,
  );
});

test("an explicit mem:// override remains available for isolated tests", () => {
  const config = resolveStoreConfig({ env: { CUSTODY_CASE_DB: "mem://desktop-config-test" }, secretsText: "" });

  assert.equal(config.dbUrl, "mem://desktop-config-test");
});

test("store loader imports the configured built-module fixture without connecting", () => {
  const here = dirname(fileURLToPath(import.meta.url));
  const pluginRoot = join(here, "fixtures", "toolkit-plugin");
  const storeClientUrl = new URL("../lib/store-client.mjs", import.meta.url).href;
  const script = `
    const { loadStoreModule, PLUGIN_ROOT } = await import(${JSON.stringify(storeClientUrl)});
    const mod = await loadStoreModule();
    if (PLUGIN_ROOT !== process.env.FAMILY_COURT_PLUGIN_ROOT || mod.configFixture !== true) process.exitCode = 1;
  `;
  const result = spawnSync(process.execPath, ["--input-type=module", "-e", script], {
    encoding: "utf8",
    env: {
      ...process.env,
      CUSTODY_CASE_DB: "mem://desktop-config-fixture",
      FAMILY_COURT_PLUGIN_ROOT: pluginRoot,
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "");
});

test("an embedded file URL cannot be selected by production configuration", () => {
  assert.throws(
    () => resolveStoreConfig({ env: {}, secretsText: "CUSTODY_CASE_DB=rocksdb://local/case.db" }),
    /valid shared/,
  );
});

test("invalid URLs and URLs without a host fail without echoing the configured value", () => {
  for (const configuredUrl of ["ws://?token=fixture-secret", "file://local/case.db?token=fixture-secret", "not a URL?token=fixture-secret"]) {
    assert.throws(
      () => resolveStoreConfig({ env: {}, secretsText: `CUSTODY_CASE_DB=${configuredUrl}` }),
      (error) => error.message.includes("valid shared") && !error.message.includes("fixture-secret"),
    );
  }
});
