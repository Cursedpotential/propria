// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Drives @anthropic-ai/claude-agent-sdk's `query()` for the /api/chat SSE
// endpoint. Restricted to Read + this plugin's own MCP tool surface — see
// ALLOWED_TOOLS below and docs/AUTH.md for the auth-source discussion.
//
// IMPORTANT (see ../../docs/AUTH.md): using a Claude subscription's
// long-term OAuth token (`claude setup-token`) to drive traffic OUTSIDE the
// official Claude Code CLI/Agent SDK harness may be restricted by Anthropic's
// terms. This code uses the official `@anthropic-ai/claude-agent-sdk`
// exclusively (never a raw HTTP call with the token), which is the intended
// use of that token — but the root agent should still verify current terms
// before this ships beyond local/desktop use. Not asserted as permitted here.

import { homedir } from "node:os";
import { join } from "node:path";
import { resolveAuthToken } from "./auth.mjs";
import { PLUGIN_ROOT } from "./store-client.mjs";
import { loadConsoleMcpServerConfig } from "./console-mcp-config.mjs";

const PLUGIN_QUALIFIED_PREFIX = "mcp__plugin_family-court-toolkit_family-court-console__";
const EXPLICIT_ALIAS_PREFIX = "mcp__family-court-console__";

export const ALLOWED_TOOLS = [`${PLUGIN_QUALIFIED_PREFIX}*`, `${EXPLICIT_ALIAS_PREFIX}*`, "Read"];

/**
 * Best-effort local auth check used only to decide whether to return a fast,
 * clear 401 instead of spawning the CLI subprocess and waiting for it to
 * fail. This is a heuristic, not authoritative — the CLI's own credential
 * storage location/format is not guaranteed by any doc read for this task
 * (flagged as a TODO in docs/AUTH.md); an explicit CLAUDE_CODE_OAUTH_TOKEN
 * (env or ~/.secrets/*.env) is always the reliable path.
 */
function hasLikelyCliLogin() {
  const candidates = [
    join(homedir(), ".claude", ".credentials.json"),
    join(homedir(), ".claude.json"),
  ];
  return candidates.some((p) => existsSync(p));
}

export function checkAuthAvailable() {
  const { token, source } = resolveAuthToken();
  if (token) return { available: true, source };
  if (hasLikelyCliLogin()) return { available: true, source: "cli-login (heuristic — unverified)" };
  return {
    available: false,
    source: "none",
    fix:
      'No CLAUDE_CODE_OAUTH_TOKEN found (env or ~/.secrets/*.env) and no local CLI credentials detected. ' +
      'Run `claude setup-token` once, then either export CLAUDE_CODE_OAUTH_TOKEN=<token> before starting ' +
      "this app, or add a line `CLAUDE_CODE_OAUTH_TOKEN=<token>` to a file under ~/.secrets/*.env. " +
      "Alternatively, run `claude login` interactively once on this machine.",
  };
}

function sseWrite(reply, event, data) {
  reply.raw.write(`event: ${event}\n`);
  reply.raw.write(`data: ${JSON.stringify(data)}\n\n`);
}

function extractTextAndTools(message) {
  const blocks = message?.message?.content ?? [];
  const out = [];
  for (const block of blocks) {
    if (block.type === "text") out.push({ kind: "text", text: block.text });
    else if (block.type === "tool_use") out.push({ kind: "tool_use", id: block.id, name: block.name, input: block.input });
    else if (block.type === "tool_result") out.push({ kind: "tool_result", tool_use_id: block.tool_use_id, content: block.content });
  }
  return out;
}

/**
 * Streams one turn of the Agent SDK to `reply` as Server-Sent Events.
 * Assumes headers have not been sent yet; sends a 401 JSON instead of SSE
 * if `checkAuthAvailable()` says no credentials are available.
 */
export async function streamChat(reply, { prompt, sessionId }) {
  let consoleMcpServer;
  try {
    consoleMcpServer = loadConsoleMcpServerConfig();
  } catch (error) {
    reply.code(503).send({
      error: "mcp_configuration",
      detail: error instanceof Error ? error.message : "Shared toolkit MCP configuration is invalid.",
    });
    return;
  }

  const auth = checkAuthAvailable();
  if (!auth.available) {
    reply.code(401).send({ error: "unauthenticated", ...auth });
    return;
  }

  const { query } = await import("@anthropic-ai/claude-agent-sdk");

  reply.raw.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache, no-transform",
    Connection: "keep-alive",
    "X-Accel-Buffering": "no",
  });
  sseWrite(reply, "auth", { source: auth.source });

  const abortController = new AbortController();
  reply.raw.req.on("close", () => abortController.abort());

  const q = query({
    prompt,
    options: {
      abortController,
      cwd: PLUGIN_ROOT,
      settingSources: ["user"],
      allowedTools: ALLOWED_TOOLS,
      permissionMode: "default",
      mcpServers: {
        "family-court-console": consoleMcpServer,
      },
      ...(sessionId ? { resume: sessionId } : {}),
    },
  });

  try {
    for await (const message of q) {
      switch (message.type) {
        case "assistant": {
          for (const block of extractTextAndTools(message)) {
            sseWrite(reply, block.kind, block);
          }
          if (message.error) sseWrite(reply, "model_error", { error: message.error });
          break;
        }
        case "result": {
          sseWrite(reply, "result", {
            subtype: message.subtype,
            is_error: message.is_error,
            result: "result" in message ? message.result : undefined,
            session_id: message.session_id,
          });
          break;
        }
        case "system": {
          if (message.subtype === "init") sseWrite(reply, "session_start", { session_id: message.session_id, model: message.model });
          break;
        }
        default:
          // Other frame types (thinking tokens, hook events, etc.) are not
          // needed by the current chat pane — dropped rather than forwarded
          // to keep the SSE stream small.
          break;
      }
    }
    sseWrite(reply, "done", {});
  } catch (err) {
    sseWrite(reply, "error", { message: err instanceof Error ? err.message : String(err) });
  } finally {
    reply.raw.end();
  }
}
