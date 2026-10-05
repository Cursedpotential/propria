// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04

import { loadConsoleMcpServerConfig } from "./console-mcp-config.mjs";

export const HOSTED_LIBRARY_TOOLS = Object.freeze([
  "library_propose",
  "library_validate",
  "library_publish",
  "case_record",
  "case_put",
]);

const MAX_CATALOG_TOOLS = 512;
const TOOL_CACHE_TTL_MS = 60_000;

/** Normalizes ContextForge tool names before exact allowlist matching, including one exact optional prefix.
 * Inputs: one MCP tool name. Output: normalized name or null when malformed.
 * Effects: none. Use this only for discovery; actual calls use the original exact hosted name.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function normalizeHostedLibraryToolName(name) {
  if (typeof name !== "string" || name.length > 256) return null;
  const normalized = name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  return normalized || null;
}

/** Finds one allowlisted canonical operation in a bounded ContextForge catalog.
 * Inputs: canonical operation and MCP listTools result. Output: exact remote name.
 * Effects: none. Rejects missing, ambiguous, or oversized catalogs; use before callTool.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function resolveHostedLibraryToolName(canonical, catalog) {
  if (!HOSTED_LIBRARY_TOOLS.includes(canonical)) throw new Error("Hosted library operation is not allowlisted.");
  if (!Array.isArray(catalog) || catalog.length > MAX_CATALOG_TOOLS) throw new Error("Hosted tool catalog is malformed or oversized.");
  const expected = canonical.replaceAll("_", "-");
  const matches = catalog.filter((tool) => {
    const normalized = normalizeHostedLibraryToolName(tool?.name);
    return normalized === expected || normalized === `family-court-${expected}`;
  });
  if (matches.length !== 1) throw new Error(matches.length ? "Hosted library tool name is ambiguous." : `Hosted ${canonical} tool is unavailable.`);
  return matches[0].name;
}

/** Converts one Streamable HTTP MCP result into a browser-safe invocation envelope.
 * Input: MCP CallToolResult. Output: status, text content, and structured data only.
 * Effects: none; credentials and transport internals are omitted. Use for sidecar responses.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function toHostedInvocation(result) {
  const structured = result?.structuredContent;
  return {
    state: result?.isError ? "tool_error" : "ok",
    structured: structured && typeof structured === "object" && !Array.isArray(structured) ? structured : null,
    text: Array.isArray(result?.content)
      ? result.content.filter((item) => item?.type === "text" && typeof item.text === "string").map((item) => item.text)
      : [],
  };
}

/** Connects to the configured ContextForge server and invokes one allowlisted library tool.
 * Inputs: optional config, SDK factory, and clock seams; output is a safe MCP envelope.
 * Effects: reads the server-only token, makes hosted MCP calls, and closes the client.
 * Prefer this hosted shared-store path over importing a local mutation adapter.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function createHostedLibraryInvoker({
  loadConfig = loadConsoleMcpServerConfig,
  createClient = createSdkClient,
  now = Date.now,
} = {}) {
  const resolvedNames = new Map();
  let catalogUrl = null;

  return async function invokeHostedLibraryTool(canonical, args) {
    if (!HOSTED_LIBRARY_TOOLS.includes(canonical)) throw new Error("Hosted library operation is not allowlisted.");
    if (!args || typeof args !== "object" || Array.isArray(args)) throw new Error("Hosted library tool arguments must be a JSON object.");

    const config = loadConfig();
    if (catalogUrl !== config.url) {
      resolvedNames.clear();
      catalogUrl = config.url;
    }
    let client;
    try {
      client = await createClient(config);
      let exactName = resolvedNames.get(canonical);
      if (!exactName || exactName.expiresAt <= now()) {
        const catalog = await client.listTools();
        const tools = catalog?.tools;
        exactName = { name: resolveHostedLibraryToolName(canonical, tools), expiresAt: now() + TOOL_CACHE_TTL_MS };
        if (resolvedNames.size >= HOSTED_LIBRARY_TOOLS.length) resolvedNames.clear();
        resolvedNames.set(canonical, exactName);
      }
      return toHostedInvocation(await client.callTool({ name: exactName.name, arguments: args }));
    } catch (error) {
      if (error?.message?.startsWith("Hosted ") || error?.message?.startsWith("Hosted library")) throw error;
      if (error?.message === "CF_MCP_CLIENT_TOKEN is not configured in the environment or designated secrets file." ||
        error?.message === "FAMILY_COURT_CONSOLE_MCP_URL must be a valid HTTP(S) URL." ||
        error?.message === "Unable to read the designated ContextForge secrets file.") throw error;
      throw new Error("Hosted ContextForge library request failed.");
    } finally {
      try {
        await client?.close?.();
      } catch {
        // Close failures do not replace a completed result or its original error.
      }
    }
  };
}

/** Opens an MCP SDK Streamable HTTP client with the server-only bearer credential.
 * Input: resolved hosted URL and headers. Output: connected MCP Client.
 * Effects: opens one authenticated network connection; caller must close it.
 * Choose this SDK transport instead of implementing the MCP HTTP protocol by hand.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
async function createSdkClient(config) {
  const [{ Client }, { StreamableHTTPClientTransport }] = await Promise.all([
    import("@modelcontextprotocol/sdk/client/index.js"),
    import("@modelcontextprotocol/sdk/client/streamableHttp.js"),
  ]);
  const client = new Client({ name: "family-court-workbench", version: "0.1.0" });
  const transport = new StreamableHTTPClientTransport(new URL(config.url), {
    requestInit: { headers: config.headers },
  });
  await client.connect(transport);
  return client;
}
