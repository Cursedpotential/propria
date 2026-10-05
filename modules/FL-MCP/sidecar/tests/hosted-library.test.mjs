// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04

import assert from "node:assert/strict";
import Fastify from "fastify";
import { test } from "node:test";
import {
  createHostedLibraryInvoker,
  resolveHostedLibraryToolName,
  toHostedInvocation,
} from "../lib/hosted-library-client.mjs";
import { isVersionedPersonalSourceEdit, registerHostedLibraryRoutes } from "../lib/hosted-library-routes.mjs";

/** Confirms prefix-aware discovery resolves one exact hosted function name.
 * Inputs are synthetic MCP catalog rows; output is an assertion. It has no I/O or shared-data effects.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("hosted catalog resolution accepts exact names or the exact family-court alias only", () => {
  assert.equal(resolveHostedLibraryToolName("library_propose", [{ name: "family-court-library-propose" }]), "family-court-library-propose");
  assert.equal(resolveHostedLibraryToolName("case_record", [{ name: "case_record" }]), "case_record");
  assert.throws(() => resolveHostedLibraryToolName("case_put", [{ name: "family-court-case-put" }, { name: "case-put" }]), /ambiguous/);
  assert.throws(() => resolveHostedLibraryToolName("case_put", [{ name: "family-court-unrelated-case-put" }]), /unavailable/);
  assert.throws(() => resolveHostedLibraryToolName("arbitrary_write", [{ name: "arbitrary-write" }]), /not allowlisted/);
  assert.throws(() => resolveHostedLibraryToolName("case_put", Array.from({ length: 513 }, (_, i) => ({ name: `tool-${i}` }))), /oversized/);
});

/** Keeps the permitted case_put operation narrower than the toolkit's generic personal-record writer.
 * Inputs are synthetic args. Output is a boolean; no hosted tool call or store write occurs.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("case_put dispatch accepts only existing versioned case_document sources", () => {
  const valid = { table: "source", id: "personal-doc-1", expected_version: `sha256:${"a".repeat(64)}`, data: { kind: "case_document", body: "Private fixture" } };
  assert.equal(isVersionedPersonalSourceEdit(valid), true);
  assert.equal(isVersionedPersonalSourceEdit({ ...valid, table: "event" }), false);
  assert.equal(isVersionedPersonalSourceEdit({ ...valid, expected_version: "absent" }), false);
  assert.equal(isVersionedPersonalSourceEdit({ ...valid, data: { body: "no discriminator" } }), false);
  assert.equal(isVersionedPersonalSourceEdit({ ...valid, relations: [] }), false);
});

/** Exercises hosted invocation using fake config and MCP clients without network or credentials.
 * Inputs are injected fixtures; output is the returned envelope and call counters. It never contacts ContextForge.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("hosted invoker sends only allowlisted tool calls, caches bounded name discovery, and omits credentials", async () => {
  let listCalls = 0;
  const toolCalls = [];
  const closed = { count: 0 };
  const invoke = createHostedLibraryInvoker({
    loadConfig: () => ({ url: "https://contextforge.example/mcp", headers: { Authorization: "Bearer fake-server-secret" } }),
    createClient: async (config) => {
      assert.equal(config.headers.Authorization, "Bearer fake-server-secret");
      return {
        async listTools() { listCalls += 1; return { tools: [{ name: "family-court-case-record" }] }; },
        async callTool(call) { toolCalls.push(call); return { structuredContent: { id: call.arguments.id }, content: [{ type: "text", text: "fixture" }] }; },
        async close() { closed.count += 1; },
      };
    },
  });

  const first = await invoke("case_record", { id: "library_proposal:fixture" });
  const second = await invoke("case_record", { id: "library_validation:fixture" });
  assert.equal(listCalls, 1);
  assert.equal(closed.count, 2);
  assert.equal(first.state, "ok");
  assert.deepEqual(second.structured, { id: "library_validation:fixture" });
  assert.deepEqual(toolCalls, [
    { name: "family-court-case-record", arguments: { id: "library_proposal:fixture" } },
    { name: "family-court-case-record", arguments: { id: "library_validation:fixture" } },
  ]);
  assert.equal(JSON.stringify([first, second]).includes("fake-server-secret"), false);
  await assert.rejects(() => invoke("case_query", {}), /not allowlisted/);
});

/** Keeps missing hosted credentials visible without returning or logging credential contents.
 * Input is a synthetic value-free config failure. Output is a rejection assertion with no network client created.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("hosted invoker reports missing server token before attempting a connection", async () => {
  let clientCreated = false;
  const invoke = createHostedLibraryInvoker({
    loadConfig() { throw new Error("CF_MCP_CLIENT_TOKEN is not configured in the environment or designated secrets file."); },
    async createClient() { clientCreated = true; throw new Error("must not connect"); },
  });
  await assert.rejects(() => invoke("case_record", { id: "reference:fixture" }), /CF_MCP_CLIENT_TOKEN is not configured/);
  assert.equal(clientCreated, false);
});

/** Ensures malformed and failed MCP outputs never become successful structured tool results.
 * Input is synthetic SDK output; output is a safe envelope. It performs no I/O.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("MCP error results remain errors and credentials or non-text blocks are not copied", () => {
  assert.deepEqual(toHostedInvocation({
    isError: true,
    structuredContent: { status: "published" },
    content: [{ type: "image", data: "secret-adjacent" }, { type: "text", text: "conflict" }],
  }), { state: "tool_error", structured: { status: "published" }, text: ["conflict"] });
});

/** Verifies the HTTP route rejects generic writes and forwards only allowlisted hosted tool names.
 * Inputs are synthetic HTTP requests and an injected fake caller; output is status and recorded calls.
 * It uses Fastify injection only and cannot access a database or production MCP endpoint.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
test("native sidecar hosted-tool route is allowlisted and forwards JSON args", async () => {
  const app = Fastify({ logger: false });
  const calls = [];
  await registerHostedLibraryRoutes(app, {
    async invokeTool(name, args) { calls.push({ name, args }); return { state: "ok", structured: { proposal_id: "library_proposal:fixture" }, text: [] }; },
  });
  try {
    const bad = await app.inject({ method: "POST", url: "/api/library/tools/case_query", payload: { args: { surql: "DELETE source" } } });
    assert.equal(bad.statusCode, 404);
    const malformed = await app.inject({ method: "POST", url: "/api/library/tools/library_propose", payload: { args: [] } });
    assert.equal(malformed.statusCode, 400);
    const genericPut = await app.inject({ method: "POST", url: "/api/library/tools/case_put", payload: { args: { table: "person", id: "person-1", expected_version: `sha256:${"a".repeat(64)}`, data: { name: "Synthetic" } } } });
    assert.equal(genericPut.statusCode, 400);
    const good = await app.inject({ method: "POST", url: "/api/library/tools/library_propose", payload: { args: { id: "reference:fixture" } } });
    assert.equal(good.statusCode, 200);
    assert.equal(good.json().structured.proposal_id, "library_proposal:fixture");
    assert.deepEqual(calls, [{ name: "library_propose", args: { id: "reference:fixture" } }]);
  } finally {
    await app.close();
  }
});
