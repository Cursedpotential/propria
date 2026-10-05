// Byline: Codex · GPT-6 · 2026-10-05. Private transport for the shared library sync service.
import { timingSafeEqual } from "node:crypto";
import { readFileSync, statSync } from "node:fs";
import type { IncomingMessage, ServerResponse } from "node:http";
import { isAbsolute } from "node:path";

const PREFIX = "/api/internal/library-sync";
const METADATA_LIMIT = 2 * 1024 * 1024;
const PAYLOAD_LIMIT = 8 * 1024 * 1024;
const UUID = "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}";
const OUTBOX = new RegExp("^/outbox/(" + UUID + ")/(payload|write-intent|complete|failure)$");

export interface SyncHttpOperation {
  method: "GET" | "POST";
  path: string;
  query: URLSearchParams;
  body: Record<string, unknown>;
  leaseId?: string;
}
export interface SyncHttpResult {
  body: unknown;
  contentType?: string;
}
export type SyncHttpDispatcher = (request: SyncHttpOperation) => Promise<SyncHttpResult>;

/** Load only the dedicated bounded server-service credential from its mounted file.
 * Inputs: configured absolute token-file path. Outputs: literal secret or undefined when unavailable.
 * Effects: bounded file metadata/read only, never logging. Choose for private sync routes rather than the MCP bearer.
 */
export function loadLibrarySyncToken(file = process.env.TOOLKIT_LIBRARY_SYNC_TOKEN_FILE): string | undefined {
  if (!file || !isAbsolute(file)) return undefined;
  try {
    const stat = statSync(file);
    if (!stat.isFile() || stat.size > 8192) return undefined;
    const raw = readFileSync(file, "utf8").trim();
    const assignment = /^TOOLKIT_LIBRARY_SYNC_TOKEN\s*=\s*(.+)$/.exec(raw);
    const token = (assignment?.[1] ?? raw).replace(/^(['"])(.*)\1$/, "$2");
    return token.length >= 32 && token.length <= 4096 && !/[\r\n\0]/.test(token) ? token : undefined;
  } catch { return undefined; }
}

/** Compare one dedicated bearer credential without accepting ordinary MCP or browser identity headers.
 * Inputs: authorization header and configured token. Outputs: admission boolean. Effects: none.
 * Choose before dispatch or database access; unavailable configuration always fails closed.
 */
export function librarySyncAuthorized(header: string | undefined, token: string | undefined): boolean {
  if (!token || token.length < 32 || token.length > 4096 || !header || header.length > 4103) return false;
  const expected = Buffer.from("Bearer " + token, "utf8");
  const supplied = Buffer.from(header, "utf8");
  return expected.length === supplied.length && timingSafeEqual(expected, supplied);
}

/** Admit only the frozen sync contract routes and reject ambiguous query and lease inputs.
 * Inputs: method, request URL and optional lease header. Outputs: normalized private operation or a 400/404/405 error.
 * Effects: none. Choose before body reads; ordinary user original-download routes are separate.
 */
export function parseLibrarySyncHttpOperation(method: string | undefined, url: URL, leaseId?: string): SyncHttpOperation {
  const route = url.pathname.slice(PREFIX.length);
  let requiredMethod: "GET" | "POST";
  if (route === "/outbox/claim" || route === "/observations") requiredMethod = "POST";
  else if (/^\/observations\/[a-f0-9]{64}\/payload$/.test(route)) requiredMethod = "POST";
  else if (/^\/observations\/[a-f0-9]{64}\/status$/.test(route)) requiredMethod = "GET";
  else {
    const outbox = OUTBOX.exec(route);
    if (outbox) requiredMethod = outbox[2] === "payload" ? "GET" : "POST";
    else if (/^\/bindings\/(?:library_file:|library_file%3[Aa])[a-f0-9]{64}\/original$/.test(route)) requiredMethod = "GET";
    else throw Object.assign(new Error("Unknown private library sync route"), { status: 404 });
  }
  if (method !== requiredMethod) throw Object.assign(new Error("Method not permitted"), { status: 405 });
  const original = route.startsWith("/bindings/");
  if (original) {
    const versions = url.searchParams.getAll("version_id");
    if (versions.length !== 1 || !versions[0] || versions[0] === "null" || versions[0].length > 2048 ||
        /[\r\n\0]/.test(versions[0]) || [...url.searchParams.keys()].some(key => key !== "version_id")) {
      throw Object.assign(new Error("An exact retained original version is required"), { status: 400 });
    }
  } else if ([...url.searchParams.keys()].length) {
    throw Object.assign(new Error("Unexpected private sync query"), { status: 400 });
  }
  if (route.startsWith("/outbox/") && route.endsWith("/payload") && (!leaseId || leaseId.length > 200 || /[\r\n\0]/.test(leaseId))) {
    throw Object.assign(new Error("An active sync lease is required"), { status: 400 });
  }
  return { method: requiredMethod, path: decodeURIComponent(route), query: url.searchParams, body: {}, leaseId };
}

/** Read a bounded sync metadata object without truncating or exposing an over-budget request.
 * Inputs: authenticated request stream. Outputs: parsed plain object or a 400/413 error.
 * Effects: consumes at most the fixed body budget before refusing. Choose for private metadata, never original-file bytes.
 */
export async function readLibrarySyncMetadata(request: AsyncIterable<Uint8Array | string>): Promise<Record<string, unknown>> {
  const chunks: Buffer[] = [];
  let bytes = 0;
  for await (const value of request) {
    const chunk = Buffer.from(value);
    bytes += chunk.byteLength;
    if (bytes > METADATA_LIMIT) throw Object.assign(new Error("Sync metadata exceeds 2 MiB"), { status: 413 });
    chunks.push(chunk);
  }
  let body: unknown;
  try { body = JSON.parse(Buffer.concat(chunks).toString("utf8")); }
  catch { throw Object.assign(new Error("Sync metadata must be valid JSON"), { status: 400 }); }
  if (!body || typeof body !== "object" || Array.isArray(body)) {
    throw Object.assign(new Error("Sync metadata must be an object"), { status: 400 });
  }
  return body as Record<string, unknown>;
}

/** Retain a complete bounded incoming file payload with its byte identity and original-source identity.
 * Inputs: authenticated request stream and worker content/source SHA-256 headers. Outputs: raw bytes plus verified metadata.
 * Effects: consumes at most the 8 MiB payload budget; overflow or wrong hashes refuses without truncation.
 * Choose for observation hydration rather than a JSON/base64 wrapper that expands or loses file content.
 */
export async function readLibrarySyncIncomingPayload(req: IncomingMessage): Promise<Record<string, unknown>> {
  const sha256 = req.headers["x-toolkit-sync-payload-sha256"];
  const sourceSha256 = req.headers["x-toolkit-sync-source-sha256"];
  const contentType = req.headers["content-type"];
  if (typeof sha256 !== "string" || !/^[a-f0-9]{64}$/.test(sha256) || typeof sourceSha256 !== "string" || !/^[a-f0-9]{64}$/.test(sourceSha256)
    || typeof contentType !== "string" || !contentType || contentType.length > 200 || /[\r\n\0]/.test(contentType))
    throw Object.assign(new Error("Invalid incoming payload identity"), { status: 400 });
  const length = Number(req.headers["content-length"]);
  if (Number.isFinite(length) && length > PAYLOAD_LIMIT) throw Object.assign(new Error("Payload budget exceeded"), { status: 413 });
  const chunks: Buffer[] = []; let size = 0;
  for await (const value of req) {
    const chunk = Buffer.from(value); size += chunk.length;
    if (size > PAYLOAD_LIMIT) throw Object.assign(new Error("Payload budget exceeded"), { status: 413 });
    chunks.push(chunk);
  }
  const bytes = Buffer.concat(chunks, size);
  const { createHash } = await import("node:crypto");
  if (createHash("sha256").update(bytes).digest("hex") !== sha256) throw Object.assign(new Error("Payload hash mismatch"), { status: 422 });
  return { incoming_bytes: bytes, sha256, source_sha256: sourceSha256, content_type: contentType };
}

/** Authenticate and serve only the internal library sync transport using the existing console process.
 * Inputs: HTTP request/response, parsed URL, admitted backend dispatcher and dedicated mounted token.
 * Outputs: handled flag. Effects: bounded metadata/payload response and guarded backend calls after authentication.
 * Choose before public web/MCP routing; this does not expose a generic query or credential endpoint.
 */
export async function handleLibrarySyncHttpRequest(
  req: IncomingMessage, res: ServerResponse, url: URL, dispatch: SyncHttpDispatcher,
  token = loadLibrarySyncToken(),
): Promise<boolean> {
  if (url.pathname !== PREFIX && !url.pathname.startsWith(PREFIX + "/")) return false;
  const sendError = (status: number, code: string) => {
    res.writeHead(status, { "content-type": "application/json", "cache-control": "no-store" });
    res.end(JSON.stringify({ error: code }));
  };
  if (!token) { sendError(503, "sync_service_not_configured"); return true; }
  if (!librarySyncAuthorized(typeof req.headers.authorization === "string" ? req.headers.authorization : undefined, token)) {
    sendError(401, "sync_service_unauthorized"); return true;
  }
  try {
    const header = req.headers["x-toolkit-sync-lease"];
    const operation = parseLibrarySyncHttpOperation(req.method, url, typeof header === "string" ? header : undefined);
    if (operation.method === "POST" && /^\/observations\/[a-f0-9]{64}\/payload$/.test(operation.path)) {
      operation.body = await readLibrarySyncIncomingPayload(req);
    } else if (operation.method === "POST") {
      if ((req.headers["content-type"] ?? "").split(";")[0].trim().toLowerCase() !== "application/json") {
        throw Object.assign(new Error("JSON content type required"), { status: 415 });
      }
      const length = Number(req.headers["content-length"]);
      if (Number.isFinite(length) && length > METADATA_LIMIT) throw Object.assign(new Error("Metadata budget exceeded"), { status: 413 });
      operation.body = await readLibrarySyncMetadata(req);
    }
    const result = await dispatch(operation);
    const binary = result.body instanceof Uint8Array;
    const body = binary ? Buffer.from(result.body as Uint8Array) : Buffer.from(JSON.stringify(result.body), "utf8");
    if (body.byteLength > (binary ? PAYLOAD_LIMIT : METADATA_LIMIT)) throw Object.assign(new Error("Response budget exceeded"), { status: 413 });
    res.writeHead(200, { "content-type": binary ? "application/octet-stream" : "application/json",
      "content-length": body.byteLength, "cache-control": "private, no-store", "x-content-type-options": "nosniff" });
    res.end(body);
  } catch (error) {
    const status = Number((error as { status?: unknown })?.status);
    sendError([400, 404, 405, 409, 413, 415, 422, 503].includes(status) ? status : 503, "sync_operation_refused");
  }
  return true;
}
