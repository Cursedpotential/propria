// Byline: Claude Code · Sonnet 5 · 2026-09-07
// Exercises the MCP_TRANSPORT=http mode added to src/server.ts: starts
// dist/server.js in HTTP mode on an ephemeral port with CUSTODY_CASE_DB=mem://,
// then drives it with the SDK's own StreamableHTTPClientTransport (bearer
// header via requestInit), plus raw fetch() probes for the auth gate and the
// unauthenticated /healthz and /version endpoints. Mirrors the stdio round
// trip in tests/protocol.test.mjs (28 tools) but over HTTP.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createServer as createNetServer } from "node:net";
import { resolve } from "node:path";
import test from "node:test";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

const BEARER_TOKEN = "test-http-transport-bearer-9f3a1c";

async function findFreePort() {
  return new Promise((resolvePort, reject) => {
    const probe = createNetServer();
    probe.once("error", reject);
    probe.listen(0, "127.0.0.1", () => {
      const { port } = probe.address();
      probe.close(() => resolvePort(port));
    });
  });
}

async function waitForHealthz(baseUrl, timeoutMs = 15000) {
  const deadline = Date.now() + timeoutMs;
  let lastErr;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseUrl}/healthz`);
      if (res.status === 200) return;
      lastErr = new Error(`unexpected /healthz status ${res.status}`);
    } catch (err) {
      lastErr = err;
    }
    await new Promise((r) => setTimeout(r, 100));
  }
  throw lastErr ?? new Error("timed out waiting for /healthz");
}

async function withHttpServer(fn) {
  const port = await findFreePort();
  const baseUrl = `http://127.0.0.1:${port}`;
  const child = spawn(process.execPath, [resolve("dist/server.js")], {
    env: {
      ...process.env,
      MCP_TRANSPORT: "http",
      MCP_HTTP_HOST: "127.0.0.1",
      MCP_HTTP_PORT: String(port),
      MCP_BEARER_TOKEN: BEARER_TOKEN,
      CUSTODY_CASE_DB: "mem://",
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let stderr = "";
  child.stderr.on("data", (chunk) => {
    stderr += chunk.toString();
  });
  const exitPromise = new Promise((res) => child.once("exit", (code, signal) => res({ code, signal })));
  try {
    await waitForHealthz(baseUrl);
    await fn({ baseUrl, port });
  } finally {
    child.kill();
    await Promise.race([exitPromise, new Promise((r) => setTimeout(r, 3000))]);
    if (!child.killed && child.exitCode === null) child.kill("SIGKILL");
  }
  return { stderr };
}

test("HTTP transport: /healthz and /version are reachable without a bearer token", async () => {
  await withHttpServer(async ({ baseUrl }) => {
    const health = await fetch(`${baseUrl}/healthz`);
    assert.equal(health.status, 200);
    const healthBody = await health.json();
    assert.equal(healthBody.status, "ok");

    const version = await fetch(`${baseUrl}/version`);
    assert.equal(version.status, 200);
    const versionBody = await version.json();
    assert.equal(versionBody.name, "family-court-console");
    assert.equal(versionBody.transport, "streamable-http");
  });
});

test("HTTP transport: POST /mcp without a bearer token is rejected with 401", async () => {
  await withHttpServer(async ({ baseUrl }) => {
    const res = await fetch(`${baseUrl}/mcp`, {
      method: "POST",
      headers: { "content-type": "application/json", accept: "application/json, text/event-stream" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/list", params: {} }),
    });
    assert.equal(res.status, 401);
    const body = await res.json();
    assert.equal(body.error, "unauthorized");
  });
});

test("HTTP transport: POST /mcp with the wrong bearer token is rejected with 401", async () => {
  await withHttpServer(async ({ baseUrl }) => {
    const res = await fetch(`${baseUrl}/mcp`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        accept: "application/json, text/event-stream",
        authorization: "Bearer not-the-right-token",
      },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/list", params: {} }),
    });
    assert.equal(res.status, 401);
  });
});

test("HTTP transport: full MCP round trip over StreamableHTTPClientTransport with the bearer token", async () => {
  await withHttpServer(async ({ baseUrl }) => {
    const transport = new StreamableHTTPClientTransport(new URL(`${baseUrl}/mcp`), {
      requestInit: { headers: { Authorization: `Bearer ${BEARER_TOKEN}` } },
    });
    const client = new Client({ name: "http-transport-test", version: "1.0.0" });
    try {
      await client.connect(transport);
      const tools = await client.listTools();
      assert.equal(tools.tools.length, 28); // + case_record (Claude Code · Opus 5.5 · 2026-09-27)
      assert.ok(tools.tools.some((tool) => tool.name === "case_summary"));

      const summary = await client.callTool({ name: "case_summary", arguments: {} });
      assert.equal(summary.structuredContent.available, true);
    } finally {
      await client.close();
    }
  });
});
