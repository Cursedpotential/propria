// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — explicit host framing contract.
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — wire case_facts (9th tool) and
// deadline rule presets into the tool registry; widen audit_sources ids schema
// to accommodate longer ledger-derived ids.
// Byline: Claude Code · Sonnet 5 · 2026-09-07 — HTTP transport mode: the whole
// registration block moved into buildServer() so both stdio (default, one
// long-lived server) and Streamable HTTP (stateless, one fresh server per
// request per the SDK's own simpleStatelessStreamableHttp.ts example) share
// one registration path. See docs/2026-09-07-case-store-registers.md for the
// cloud-console + ContextForge wiring this enables.
// Byline: Claude Code · Sonnet 5 · 2026-09-08 — "content lives in the SurrealDB store":
// auditSources()/searchRecords()/reviewCourtLanguage()/buildSurvivalGuide() are now
// store-first and async; every call site here awaits them.
import { randomBytes, timingSafeEqual } from "node:crypto";
import { createServer as createHttpServer, type IncomingMessage, type ServerResponse } from "node:http";
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { registerAppResource, registerAppTool, RESOURCE_MIME_TYPE } from "@modelcontextprotocol/ext-apps/server";
import { z } from "zod";
import { widgets } from "./generated-widgets.js";
import { registerStoreTools } from "./store-tools.js";
import { handleWebRequest } from "./web.js";
import {
  DEADLINE_RULE_PRESETS,
  RELEASE_STATUS,
  SOURCES,
  auditSources,
  buildChronology,
  buildPacketPlan,
  calculateDirectionalDeadline,
  getCaseFacts,
  getChecklist,
  routeIssue,
  searchRecords,
} from "./core.js";
import { DOC_TYPES, reviewCourtLanguage, renderMarkdownReport, type DocType } from "./court-language.js";
import { buildSurvivalGuide, listSurvivalGuideEvents, renderSurvivalGuideMarkdown } from "./survival-guide.js";

const SERVER_NAME = "family-court-console";
const SERVER_VERSION = "3.0.0";

const readOnly = { readOnlyHint: true, destructiveHint: false, idempotentHint: true, openWorldHint: false } as const;
const uiUri = (name: keyof typeof widgets) => `ui://family-court/${name}.html`;
const result = (data: Record<string, unknown>) => ({
  structuredContent: data,
  content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }],
});

function buildServer(): McpServer {
  const server = new McpServer({ name: SERVER_NAME, version: SERVER_VERSION });
  registerStoreTools(server); // embedded SurrealDB case store (search + graph) — Claude Code · Fable 5.1 · 2026-09-07

  for (const [name, html] of Object.entries(widgets) as Array<[keyof typeof widgets, string]>) {
    registerAppResource(server, `Family Court ${name}`, uiUri(name), { description: `Interactive ${name} view for the Michigan Family Court Console.` }, async () => ({
      contents: [{ uri: uiUri(name), mimeType: RESOURCE_MIME_TYPE, text: html, _meta: { ui: { prefersBorder: false, csp: { connectDomains: [], resourceDomains: [] } } } }],
    }));
  }

  registerAppTool(server, "open_dashboard", {
  title: "Open Michigan Family Court Console",
  description: "Show release status, safety gates, source confidence, and the next planning steps. The underlying guide is publication-blocked.",
  inputSchema: {}, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("dashboard") } },
}, async () => result({ release: RELEASE_STATUS, sourceSummary: (await auditSources()).counts, actions: ["Route the issue", "Check a planning date", "Build a packet plan", "Audit sources"] }));

registerAppTool(server, "route_issue", {
  title: "Route a Michigan family-court issue",
  description: "Identify safety-critical route-outs before using draft material. Does not diagnose a case or give legal advice.",
  inputSchema: { description: z.string().min(3).max(4000) }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("route") } },
}, async ({ description }) => result(routeIssue(description)));

registerAppTool(server, "calculate_planning_date", {
  title: "Calculate a directional planning date",
  description: "Count calendar days before or after an anchor, adjusting unavailable dates in the safe direction (MCR 1.108-style: exclude the anchor day, roll forward over weekends/holidays). Pass either an explicit day count and direction, or a named `rule` preset (referee_objection, appeal_of_right, motion_response, mail_service_addon). This is a planning aid, not a filing deadline determination.",
  inputSchema: {
    anchorDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
    days: z.number().int().min(0).max(3650).optional(),
    direction: z.enum(["after", "before"]).optional(),
    includeAnchor: z.boolean().optional(),
    holidays: z.array(z.string().regex(/^\d{4}-\d{2}-\d{2}$/)).max(100).optional(),
    rule: z.enum(Object.keys(DEADLINE_RULE_PRESETS) as [string, ...string[]]).optional(),
  }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("deadline") } },
}, async (input) => result(calculateDirectionalDeadline(input as Parameters<typeof calculateDirectionalDeadline>[0])));

registerAppTool(server, "get_packet_plan", {
  title: "Build a guarded packet plan",
  description: "Create an organization plan with legal-review blocks and safety stop conditions.",
  inputSchema: { stage: z.string().min(1).max(200), goal: z.string().min(1).max(500) }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("packet") } },
}, async ({ stage, goal }) => result(buildPacketPlan(stage, goal)));

registerAppTool(server, "get_checklist", {
  title: "Open a family-court checklist",
  description: "Return a non-filing checklist for evidence, hearing preparation, or source review.",
  inputSchema: { kind: z.enum(["evidence", "hearing", "source-review"]) }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("checklist") } },
}, async ({ kind }) => result(getChecklist(kind)));

registerAppTool(server, "audit_sources", {
  title: "Audit verified source records",
  description: "Show normalized authority and currency status across the console's curated sources, the 191-record verification ledger, and the master source directory, with per-status/per-origin counts. Falls back to the 7 curated sources alone if the ledger or directory files are unavailable.",
  inputSchema: { ids: z.array(z.string().max(80)).max(200).optional() }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("sources") } },
}, async ({ ids }) => result(await auditSources(ids)));

registerAppTool(server, "search_guide", {
  title: "Search reviewed guide records",
  description: "Search only the console's curated source registry. Stale draft prose is intentionally excluded.",
  inputSchema: { query: z.string().min(2).max(300) }, annotations: readOnly, _meta: { ui: { resourceUri: uiUri("search") } },
}, async ({ query }) => result({ query, results: await searchRecords(query), releaseStatus: RELEASE_STATUS.label }));

registerAppTool(server, "build_chronology", {
  title: "Build a fact chronology",
  description: "Sort dated events while preserving source and knowledge-date fields. Does not infer truth or causation.",
  inputSchema: { events: z.array(z.object({ date: z.string(), title: z.string().min(1).max(1000), source: z.string().max(1000).optional(), knowledgeDate: z.string().optional() })).min(1).max(500) },
  annotations: readOnly, _meta: { ui: { resourceUri: uiUri("chronology") } },
}, async ({ events }) => result(buildChronology(events)));

registerAppTool(server, "case_facts", {
  title: "Read local case facts",
  description: "Return the county, court, judge, referee, controlling orders, next hearing, deadlines, party initials, and child count/ages from a local case-state JSON file (env CUSTODY_CASE_FILE, else ~/.config/family-court-toolkit/case.json). Never returns a child's name; any children[].name in the source file is reduced to initials. Returns { configured: false, example_path, hint } if no case file exists yet.",
  inputSchema: {}, annotations: readOnly, _meta: { ui: {} },
}, async () => result(getCaseFacts() as unknown as Record<string, unknown>));

const SURVIVAL_GUIDE_EVENTS = listSurvivalGuideEvents();

registerAppTool(server, "survival_guide", {
  title: "Generate a hearing/document survival guide context pack",
  description: "Owner-requested tool 1: resolves a lightweight, cited context pack (sequence, applicable rules, deadlines with rule presets, traps, do-not list, phrases, safety gates, source paths) plus a writing template (full guide or one-page card) for the requested event/document type — never authors the guide itself. Optionally merges case_facts (never a child's name). The calling model writes the actual guide from these inputs; every guide is PROVISIONAL until checked against the archived MCR/MCL.",
  inputSchema: {
    event: z.enum(SURVIVAL_GUIDE_EVENTS as [string, ...string[]]),
    format: z.enum(["full", "card", "json"]).optional(),
    include_case_facts: z.boolean().optional(),
  },
  annotations: readOnly,
  _meta: { ui: {} },
}, async ({ event, format, include_case_facts }) => {
  const guide = await buildSurvivalGuide({ event, format, include_case_facts });
  return { structuredContent: guide as unknown as Record<string, unknown>, content: [{ type: "text" as const, text: renderSurvivalGuideMarkdown(guide) }] };
});

registerAppTool(server, "court_language_review", {
  title: "Review or plan a court-safe language rewrite",
  description: "Owner-requested tool 2: deterministically runs every court-language lexicon pattern (banned clinical labels, absolutes, mind-reading/motive, characterizations, emotional intensifiers, profanity/insults, threats, child-as-witness, speculation, layperson legal conclusions, recording/surveillance admissions, minor PII) plus doc-type profile checks against the text, and returns a score, stop flags, findings, profile violations, an ordered rewrite plan, and a safe phrasebank drawn from references/court-language/EXAMPLES.md. It does not rewrite the text — the calling model rewrites following references/court-language/SKILL.md + TEMPLATES.md + EXAMPLES.md, then re-reviews (mode: \"review\") until score >= 90 and stop_flags is empty.",
  inputSchema: {
    text: z.string().min(1).max(20000),
    doc_type: z.enum(DOC_TYPES as [string, ...string[]]),
    mode: z.enum(["review", "rewrite_plan"]).optional(),
  },
  annotations: readOnly,
  _meta: { ui: {} },
}, async ({ text, doc_type, mode }) => {
  const review = await reviewCourtLanguage({ text, doc_type: doc_type as DocType, mode });
  return { structuredContent: review as unknown as Record<string, unknown>, content: [{ type: "text" as const, text: renderMarkdownReport(review) }] };
});

server.registerResource("release-status", "custody://release-status", { title: "Custody Guide Release Status", mimeType: "application/json" }, async (uri) => ({ contents: [{ uri: uri.href, mimeType: "application/json", text: JSON.stringify(RELEASE_STATUS, null, 2) }] }));
server.registerResource("verified-sources", "custody://verified-sources", { title: "Reviewed Official Sources", mimeType: "application/json" }, async (uri) => ({ contents: [{ uri: uri.href, mimeType: "application/json", text: JSON.stringify(SOURCES, null, 2) }] }));

server.registerPrompt("safe-case-intake", { title: "Safe Michigan family-court intake", description: "Gather minimum facts and route safety-critical issues before analysis.", argsSchema: { issue: z.string(), county: z.string().optional() } }, ({ issue, county }) => ({ messages: [{ role: "user", content: { type: "text", text: `Route this Michigan family-court issue before analysis. County: ${county ?? "unknown"}. Issue: ${issue}. Do not request child identifiers. Surface immediate-danger, PPO/DV/CPS, UCCJEA, appeal, recording, and criminal-overlap gates. Use current primary authority and label uncertainty.` } }] }));
  server.registerPrompt("verify-legal-claim", { title: "Verify a Michigan legal claim", description: "Verify one claim against current official primary authority.", argsSchema: { claim: z.string(), dateRelevant: z.string().optional() } }, ({ claim, dateRelevant }) => ({ messages: [{ role: "user", content: { type: "text", text: `Verify this Michigan family-law claim: ${claim}. Relevant date: ${dateRelevant ?? "current"}. Identify issuing body, authority level, pinpoint, currency, direct support, and conflicts. Do not treat a live link as substantive verification.` } }] }));

  return server;
}

// ---------------------------------------------------------------------------
// Transport selection. Default is stdio (desktop plugin behavior, unchanged:
// one long-lived McpServer connected over stdin/stdout). Setting
// MCP_TRANSPORT=http (or passing --http) instead serves MCP Streamable HTTP
// per the SDK's own "stateless" pattern (see node_modules/@modelcontextprotocol/
// sdk/dist/esm/examples/server/simpleStatelessStreamableHttp.js and the
// "Streamable HTTP server (stateless)" row of the SDK README): a fresh
// McpServer + StreamableHTTPServerTransport(sessionIdGenerator: undefined) per
// request. That is the SDK-recommended shape for "simple API-style servers"
// serving multiple concurrent clients, and it fits this server cleanly —
// registerStoreTools' SurrealDB connection is memoized at module scope in
// store.ts (getStore()'s cachedStore/cachedUrl), not on the McpServer
// instance, so rebuilding the McpServer per request is cheap and does not
// reopen the database connection.
// ---------------------------------------------------------------------------

const TRANSPORT_MODE = (process.env.MCP_TRANSPORT?.trim().toLowerCase() || (process.argv.includes("--http") ? "http" : "stdio"));

function timingSafeEqualString(a: string, b: string): boolean {
  const bufA = Buffer.from(a, "utf8");
  const bufB = Buffer.from(b, "utf8");
  // timingSafeEqual throws on length mismatch — compare against a
  // same-length random buffer first so a wrong-length token still takes the
  // constant-time path instead of short-circuiting on .length.
  if (bufA.length !== bufB.length) {
    timingSafeEqual(bufA, randomBytes(bufA.length));
    return false;
  }
  return timingSafeEqual(bufA, bufB);
}

function sendJson(res: ServerResponse, status: number, body: Record<string, unknown>, extraHeaders?: Record<string, string>): void {
  res.writeHead(status, { "content-type": "application/json", ...extraHeaders });
  res.end(JSON.stringify(body));
}

async function handleHttpRequest(req: IncomingMessage, res: ServerResponse, bearerToken: string): Promise<void> {
  const url = new URL(req.url ?? "/", "http://localhost");

  if (url.pathname === "/healthz") {
    sendJson(res, 200, { status: "ok", name: SERVER_NAME, version: SERVER_VERSION });
    return;
  }

  if (url.pathname === "/version") {
    sendJson(res, 200, { name: SERVER_NAME, version: SERVER_VERSION, protocol: "mcp", transport: "streamable-http" });
    return;
  }

  if (url.pathname !== "/mcp") {
    // The Family Law Toolkit web app (web.ts): Tailscale identity via svc:family-court or
    // Authentik via family-court.int, checked in web-auth.ts. Claude Code · Opus 5.5 · 2026-09-28.
    if (await handleWebRequest(req, res, url, buildServer)) return;
    sendJson(res, 404, { error: "not_found" });
    return;
  }

  const authHeader = req.headers.authorization ?? "";
  if (!timingSafeEqualString(authHeader, `Bearer ${bearerToken}`)) {
    sendJson(res, 401, { error: "unauthorized" }, { "www-authenticate": "Bearer" });
    return;
  }

  // Stateless: one McpServer + one transport per request, per the SDK's own
  // stateless example. The transport enforces this itself (it throws if a
  // stateless transport instance is reused across requests).
  const server = buildServer();
  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  res.on("close", () => {
    void transport.close();
    void server.close();
  });
  await server.connect(transport);
  await transport.handleRequest(req, res);
}

async function runHttpServer(): Promise<void> {
  const host = process.env.MCP_HTTP_HOST?.trim() || "0.0.0.0";
  const port = Number.parseInt(process.env.MCP_HTTP_PORT?.trim() || "8765", 10);
  const bearerToken = process.env.MCP_BEARER_TOKEN?.trim();
  if (!bearerToken) {
    console.error("MCP_TRANSPORT=http requires MCP_BEARER_TOKEN to be set (bearer auth on /mcp is mandatory).");
    process.exit(1);
  }

  const httpServer = createHttpServer((req, res) => {
    handleHttpRequest(req, res, bearerToken).catch((err: unknown) => {
      console.error("family-court-console HTTP transport error:", err);
      if (!res.headersSent) sendJson(res, 500, { error: "internal_server_error" });
      else res.end();
    });
  });

  await new Promise<void>((resolve, reject) => {
    httpServer.once("error", reject);
    httpServer.listen(port, host, () => resolve());
  });
  console.error(`family-court-console MCP Streamable HTTP transport listening on ${host}:${port} (POST /mcp, GET /healthz, GET /version, authenticated web app on / and /api/*)`);
}

if (TRANSPORT_MODE === "http") {
  await runHttpServer();
} else {
  const server = buildServer();
  await server.connect(new StdioServerTransport());
}
