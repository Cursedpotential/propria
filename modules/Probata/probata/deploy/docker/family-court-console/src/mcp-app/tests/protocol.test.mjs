// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Byline: Claude Code · Fable 5.1 · 2026-09-07 — tool count 9 -> 11 (survival_guide, court_language_review) -> 20 (embedded SurrealDB case store)
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — tool count 20 -> 27 (case_status,
// case_docket, case_memo, case_evidence_log, case_eval, case_reference, case_source)
import assert from "node:assert/strict";
import { resolve } from "node:path";
import test from "node:test";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

test("bundled MCP server completes the core protocol round trip", async () => {
  const transport = new StdioClientTransport({ command: process.execPath, args: [resolve("dist/server.js")] });
  const client = new Client({ name: "custody-console-test", version: "1.0.0" });
  try {
    await client.connect(transport);
    const tools = await client.listTools();
    const STORE = [
      "case_export", "case_factor_map", "case_graph", "case_import", "case_put", "case_query", "case_search", "case_summary", "case_timeline",
      "case_status", "case_docket", "case_memo", "case_evidence_log", "case_eval", "case_reference", "case_source",
      "case_record", // shared legal-record contract (Claude Code · Opus 5.5 · 2026-09-27)
    ];
    assert.deepEqual(tools.tools.map((tool) => tool.name).sort(), [
      "audit_sources", "build_chronology", "calculate_planning_date", "case_facts", "court_language_review",
      "get_checklist", "get_packet_plan", "open_dashboard", "route_issue", "search_guide", "survival_guide", ...STORE,
    ].sort());
    // console tools are read-only; store tools may write (case_put/case_import/case_query write:true) — 2026-09-07
    assert.ok(tools.tools.filter((tool) => !STORE.includes(tool.name)).every((tool) => tool.annotations?.readOnlyHint === true));

    const routed = await client.callTool({ name: "route_issue", arguments: { description: "A PPO and an out of state order" } });
    assert.equal(routed.structuredContent.status, "STOP_AND_VERIFY");

    const resources = await client.listResources();
    assert.ok(resources.resources.some((resource) => resource.uri === "custody://release-status"));
    assert.ok(resources.resources.some((resource) => resource.uri === "ui://family-court/dashboard.html"));
    const release = await client.readResource({ uri: "custody://release-status" });
    assert.match(release.contents[0].text, /PUBLICATION BLOCKED/);

    const prompts = await client.listPrompts();
    assert.deepEqual(prompts.prompts.map((prompt) => prompt.name).sort(), ["safe-case-intake", "verify-legal-claim"]);
  } finally {
    await client.close();
  }
});
