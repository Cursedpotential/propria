// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04

import { HOSTED_LIBRARY_TOOLS } from "./hosted-library-client.mjs";

const MAX_ARGS_BYTES = 512_000;
const VERSION_HASH = /^sha256:[a-f0-9]{64}$/i;

/** Restricts case_put to an existing exact-version personal case_document source edit.
 * Input is the tool argument object. Output is a boolean; it performs no I/O.
 * This boundary prevents the allowlisted tool from becoming a generic personal-store write adapter.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function isVersionedPersonalSourceEdit(args) {
  return args && Object.keys(args).every((key) => ["table", "id", "expected_version", "data"].includes(key)) &&
    args.table === "source" && typeof args.id === "string" && args.id.length > 0 && args.id.length <= 200 &&
    !args.id.startsWith("source:") && VERSION_HASH.test(args.expected_version ?? "") &&
    args.data && typeof args.data === "object" && !Array.isArray(args.data) && args.data.kind === "case_document" &&
    !Object.hasOwn(args.data, "id") && !Object.hasOwn(args.data, "embedding") && !Object.hasOwn(args, "relations");
}

/** Registers the narrow native library-tool route backed only by hosted ContextForge.
 * Inputs: Fastify app and injected hosted invoker. Output: registered POST route.
 * Effects: receives five allowlisted MCP calls; rejects arbitrary tool names and malformed bodies.
 * Use this instead of exposing generic filesystem, database, or MCP dispatch routes.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export async function registerHostedLibraryRoutes(app, { invokeTool }) {
  app.post("/api/library/tools/:name", async (request, reply) => {
    const { name } = request.params ?? {};
    if (!HOSTED_LIBRARY_TOOLS.includes(name)) {
      reply.code(404);
      return { error: "Hosted library operation is not allowlisted." };
    }
    const args = request.body?.args;
    if (!args || typeof args !== "object" || Array.isArray(args) || Buffer.byteLength(JSON.stringify(args), "utf8") > MAX_ARGS_BYTES) {
      reply.code(400);
      return { error: "args must be a JSON object within the 512 KB request limit." };
    }
    if (name === "case_put" && !isVersionedPersonalSourceEdit(args)) {
      reply.code(400);
      return { error: "case_put is limited to existing source case_document edits with an exact sha256 expected_version." };
    }
    try {
      return await invokeTool(name, args);
    } catch (error) {
      reply.code(502);
      return { error: error instanceof Error ? error.message : "Hosted ContextForge library request failed." };
    }
  });
}
