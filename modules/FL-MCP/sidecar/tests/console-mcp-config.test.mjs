// Byline: Codex · GPT-6 · 2026-10-04
import assert from "node:assert/strict";
import { test } from "node:test";
import { createConsoleMcpServerConfig, loadConsoleMcpServerConfig } from "../lib/console-mcp-config.mjs";

const TEST_TOKEN = "fixture-server-token";

test("uses the shared HTTP console with a server-only bearer header", () => {
  const config = createConsoleMcpServerConfig({
    env: { CF_MCP_CLIENT_TOKEN: TEST_TOKEN },
  });

  assert.deepEqual(config, {
    type: "http",
    url: "https://mcp.mitechconsult.com/servers/0b85f64905be468ba477d75cb2a80323/mcp",
    headers: { Authorization: `Bearer ${TEST_TOKEN}` },
  });
  assert.equal("command" in config, false);
  assert.equal("args" in config, false);
});

test("prefers the environment token and parses optional export, quotes, and comments", () => {
  const fromEnv = createConsoleMcpServerConfig({
    env: { CF_MCP_CLIENT_TOKEN: ` ${TEST_TOKEN} ` },
    secretsText: "export CF_MCP_CLIENT_TOKEN='file-token' # fixture\n",
  });
  const fromFile = createConsoleMcpServerConfig({
    env: {},
    secretsText: 'export CF_MCP_CLIENT_TOKEN="file-token" # fixture\n',
  });

  assert.equal(fromEnv.headers.Authorization, `Bearer ${TEST_TOKEN}`);
  assert.equal(fromFile.headers.Authorization, "Bearer file-token");
});

test("does not read the secrets file when the environment token is configured", () => {
  const config = loadConsoleMcpServerConfig({
    env: { CF_MCP_CLIENT_TOKEN: TEST_TOKEN },
    readSecrets: () => { throw new Error("secrets file should not be read"); },
  });

  assert.equal(config.headers.Authorization, `Bearer ${TEST_TOKEN}`);
});

test("fails visibly for missing token and invalid configured URL without exposing values", () => {
  assert.throws(
    () => createConsoleMcpServerConfig({ env: {} }),
    /CF_MCP_CLIENT_TOKEN is not configured/,
  );
  assert.throws(
    () => createConsoleMcpServerConfig({
      env: { CF_MCP_CLIENT_TOKEN: TEST_TOKEN, FAMILY_COURT_CONSOLE_MCP_URL: "javascript:alert(1)" },
    }),
    /FAMILY_COURT_CONSOLE_MCP_URL must be a valid HTTP\(S\) URL/,
  );
});
