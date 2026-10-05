// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Byline: Claude Code · Fable 5.1 · 2026-09-07 — tool count 9 -> 11 (survival_guide, court_language_review) -> 20 (embedded SurrealDB case store)
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — tool count 20 -> 27 (case_status,
// case_docket, case_memo, case_evidence_log, case_eval, case_reference, case_source)
import assert from "node:assert/strict";
import { resolve } from "node:path";
import test from "node:test";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";

// VPS-only: the child uses its configured shared case store; all calls below are read-only.
test("bundled MCP server exposes the shared-catalog contract over MCP", async () => {
  const transport = new StdioClientTransport({ command: process.execPath, args: [resolve("dist/server.js")] });
  const client = new Client({ name: "custody-console-test", version: "1.0.0" });
  try {
    await client.connect(transport);
    const tools = await client.listTools();
    const STORE = [
      "case_export", "case_factor_map", "case_graph", "case_import", "case_put", "case_query", "case_search", "case_summary", "case_timeline",
      "case_status", "case_docket", "case_memo", "case_evidence_log", "case_eval", "case_reference", "case_source",
      "library_propose", "library_validate", "library_publish",
      "case_record", // shared legal-record contract (Claude Code · Opus 5.5 · 2026-09-27)
    ];
    assert.deepEqual(tools.tools.map((tool) => tool.name).sort(), [
      "audit_sources", "build_chronology", "calculate_planning_date", "case_facts", "court_language_review",
      "get_checklist", "get_packet_plan", "open_dashboard", "route_issue", "search_guide", "survival_guide", ...STORE,
    ].sort());
    // console tools are read-only; store tools may write (case_put/case_import/case_query write:true) — 2026-09-07
    assert.ok(tools.tools.filter((tool) => !STORE.includes(tool.name)).every((tool) => tool.annotations?.readOnlyHint === true));
    const survivalEvent = tools.tools.find((tool) => tool.name === "survival_guide").inputSchema.properties.event;
    assert.equal(survivalEvent.type, "string", "shared event IDs are validated at request time, not captured as a packaged startup enum");
    assert.equal(survivalEvent.enum, undefined);

    const routed = await client.callTool({ name: "route_issue", arguments: { description: "A PPO and an out of state order" } });
    assert.equal(routed.structuredContent.status, "STOP_AND_VERIFY");

    const resources = await client.listResources();
    assert.ok(resources.resources.some((resource) => resource.uri === "custody://release-status"));
    assert.ok(resources.resources.some((resource) => resource.uri === "ui://family-court/dashboard.html"));
    const release = await client.readResource({ uri: "custody://release-status" });
    assert.equal(typeof JSON.parse(release.contents[0].text).label, "string");
    const sources = await client.readResource({ uri: "custody://verified-sources" });
    assert.ok(Array.isArray(JSON.parse(sources.contents[0].text)), "source resource returns current shared records");

    const facts = await client.callTool({ name: "case_facts", arguments: {} });
    assert.equal(facts.structuredContent.source, "surrealdb-case-store");
    assert.equal(typeof facts.structuredContent.configured, "boolean");
    if (facts.structuredContent.configured) assert.equal(typeof facts.structuredContent.case_context, "object", "private shared context is retained");
    const deadline = await client.callTool({ name: "calculate_planning_date", arguments: { anchorDate: "2026-10-05", rule: "referee_objection" } });
    assert.equal(deadline.structuredContent.preset_provenance.configured_citation, "MCR 3.215(E)(4)");
    assert.equal(deadline.structuredContent.preset_provenance.shared_rule_traceability.claimed_citation, "MCR 3.215(E)(4)");
    const unknownEvent = await client.callTool({ name: "survival_guide", arguments: { event: "synthetic-event-absent-from-shared-store", format: "json" } });
    assert.equal(unknownEvent.isError, true, "shared event lookup failures remain visible to callers");
    assert.match(unknownEvent.content.map((part) => part.text ?? "").join("\n"), /Unknown survival_guide event/, "a healthy shared read reports the unknown event rather than a catalog outage");

    const prompts = await client.listPrompts();
    assert.deepEqual(prompts.prompts.map((prompt) => prompt.name).sort(), ["safe-case-intake", "verify-legal-claim"]);
  } finally {
    await client.close();
  }
});
