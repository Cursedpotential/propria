// Byline: Codex · GPT-6 · 2026-10-04

import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

const DEFAULT_CONSOLE_URL = "https://mcp.mitechconsult.com/servers/0b85f64905be468ba477d75cb2a80323/mcp";
const SECRETS_FILE = join(homedir(), ".secrets", "contextforge.env");

/** Reads dotenv text from the designated ContextForge secrets file.
 * Inputs: no arguments. Output: file contents, or an empty string if absent.
 * Effects: reads ~/.secrets/contextforge.env; never logs its contents.
 * Prefer this fixed-file reader over shell evaluation or scanning unrelated secret files. */
function readContextForgeSecrets() {
  try {
    return readFileSync(SECRETS_FILE, "utf8");
  } catch (error) {
    if (error?.code === "ENOENT") return "";
    throw new Error("Unable to read the designated ContextForge secrets file.");
  }
}

/** Extracts a configured key from dotenv text without evaluating it.
 * Inputs: dotenv text and an environment key. Output: the key's value or null.
 * Effects: none; does not log or execute input. Prefer this over shell sourcing. */
function parseEnvValue(text, key) {
  const keyPattern = key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const matcher = new RegExp(`^\\s*(?:export\\s+)?${keyPattern}\\s*=\\s*(.*?)\\s*$`);
  for (const line of String(text ?? "").split(/\r?\n/)) {
    const match = matcher.exec(line);
    if (!match || match[1].startsWith("#")) continue;
    const raw = match[1];
    const quoted = /^(?:"([^"]*)"|'([^']*)')(?:\s+#.*)?$/.exec(raw);
    if (quoted) return quoted[1] ?? quoted[2] ?? null;
    const bare = raw.replace(/\s+#.*$/, "").trim();
    if (bare) return bare;
  }
  return null;
}

/** Builds the SDK's hosted Streamable HTTP config for the shared toolkit console.
 * Inputs: optional env and secretsText overrides for tests. Output: SDK http config.
 * Effects: none. Throws a value-free configuration error for invalid URL or missing token.
 * Prefer this shared server config over launching a local stdio MCP process. */
export function createConsoleMcpServerConfig({ env = process.env, secretsText = "" } = {}) {
  const configuredUrl = env.FAMILY_COURT_CONSOLE_MCP_URL?.trim();
  const url = configuredUrl || DEFAULT_CONSOLE_URL;
  let parsedUrl;
  try {
    parsedUrl = new URL(url);
  } catch {
    throw new Error("FAMILY_COURT_CONSOLE_MCP_URL must be a valid HTTP(S) URL.");
  }
  if (!(["http:", "https:"].includes(parsedUrl.protocol) && parsedUrl.hostname && !parsedUrl.username && !parsedUrl.password)) {
    throw new Error("FAMILY_COURT_CONSOLE_MCP_URL must be a valid HTTP(S) URL.");
  }

  const token = env.CF_MCP_CLIENT_TOKEN?.trim() || parseEnvValue(secretsText, "CF_MCP_CLIENT_TOKEN");
  if (!token) throw new Error("CF_MCP_CLIENT_TOKEN is not configured in the environment or designated secrets file.");

  return { type: "http", url: parsedUrl.href, headers: { Authorization: `Bearer ${token}` } };
}

/** Resolves the hosted MCP config from environment and, when needed, the fixed secret file.
 * Inputs: optional environment and secret-reader overrides for deterministic tests.
 * Output: SDK http config for the shared console.
 * Effects: reads the designated file only when the environment token is absent; never logs the token.
 * Prefer this runtime entry point over local process spawning or broad secret scans. */
export function loadConsoleMcpServerConfig({ env = process.env, readSecrets = readContextForgeSecrets } = {}) {
  const envToken = env.CF_MCP_CLIENT_TOKEN?.trim();
  const secretsText = envToken ? "" : readSecrets();
  return createConsoleMcpServerConfig({ env, secretsText });
}
