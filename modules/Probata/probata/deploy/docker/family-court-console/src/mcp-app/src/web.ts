// Byline: Claude Code · Opus 5.5 · 2026-09-28
//
// The Family Law Toolkit web app, served by the same process as the MCP console
// (server.ts, HTTP mode). Owner, 2026-09-27 09:29-09:37 EDT: "this is my app ... I need
// to use shit", "port it over to the hosted application", "make it share the data with
// the legal work desk like we decided".
//
// One operation contract: the browser never talks to SurrealDB and never gets a second
// implementation of any tool. Every read and write goes through the same McpServer that
// /mcp serves (buildServer()), reached in-process over the SDK's InMemoryTransport:
//
//   GET  /api/health               open: liveness and store connection
//   GET  /                         the mobile-first host page (dist/web/)
//   GET  /api/whoami               the authenticated principal
//   GET  /api/tools                every registered tool with its JSON input schema
//   POST /api/tools/:name          call one tool; body { arguments }
//   GET  /api/widgets/:name        a tool's MCP App widget HTML (rendered by the host in an iframe)
//   GET  /api/library              the toolkit's shipped content files (cheat sheets, guide, law)
//   GET  /api/library/file?path=   one content file, with its sha256 version
//   GET  /api/records?table=&limit=&cursor= one bounded, id-ordered page and its continuation
//   GET  /api/records/:table:id    one record in the shared legal-record contract (case_record)
//
// Authentication (web-auth.ts, the Workbench's trusted-proxy model): the owner's Tailscale
// identity through svc:family-court, or Authentik through the public family-court.int route.
// Every other caller gets 403. /mcp keeps its bearer for ContextForge.

import { createHash } from "node:crypto";
import { readFile, readdir, stat } from "node:fs/promises";
import type { IncomingMessage, ServerResponse } from "node:http";
import { dirname, extname, join, normalize as normalizePath, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { InMemoryTransport } from "@modelcontextprotocol/sdk/inMemory.js";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { widgets } from "./generated-widgets.js";
import { DATA_TABLES, caseRecord, getStore, normalize, type StoreOk } from "./store.js";
import { authenticate, loadAuthConfig, type Principal } from "./web-auth.js";
import { proxyLibraryOriginal } from "./library-original-proxy.js";

const here = dirname(fileURLToPath(import.meta.url));
// dist/server.js sits at <root>/mcp-app/dist/, so content/ is two levels up and the
// host page is dist/web/ (copied there by build.mjs).
const WEB_DIR = join(here, "web");
const CONTENT_ROOTS = ["content", "skills"].map((name) => join(here, "..", "..", name));
const MAX_BODY_BYTES = 12 * 1024 * 1024;
const HEAVY_FIELDS = ["embedding", "file_b64", "body"];
const authConfig = loadAuthConfig();

const MIME: Record<string, string> = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".md": "text/markdown; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
  ".pdf": "application/pdf",
  ".svg": "image/svg+xml",
};

let clientPromise: Promise<Client> | null = null;

function toolClient(buildServer: () => McpServer): Promise<Client> {
  clientPromise ??= (async () => {
    const server = buildServer();
    const [clientSide, serverSide] = InMemoryTransport.createLinkedPair();
    await server.connect(serverSide);
    const client = new Client({ name: "family-law-toolkit-web", version: "1.0.0" });
    await client.connect(clientSide);
    return client;
  })().catch((err) => {
    clientPromise = null;
    throw err;
  });
  return clientPromise;
}

function send(res: ServerResponse, status: number, body: unknown, headers: Record<string, string> = {}): void {
  res.writeHead(status, { "content-type": "application/json; charset=utf-8", "cache-control": "no-store", ...headers });
  res.end(JSON.stringify(body));
}

async function readJsonBody(req: IncomingMessage): Promise<Record<string, unknown>> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of req) {
    size += (chunk as Buffer).length;
    if (size > MAX_BODY_BYTES) throw Object.assign(new Error("Request body is larger than 12 MB."), { status: 413 });
    chunks.push(chunk as Buffer);
  }
  if (size === 0) return {};
  const parsed = JSON.parse(Buffer.concat(chunks).toString("utf8"));
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw Object.assign(new Error("Body must be a JSON object."), { status: 400 });
  return parsed as Record<string, unknown>;
}

function sha256(bytes: Buffer): string {
  return createHash("sha256").update(bytes).digest("hex");
}

function recordTitle(row: Record<string, unknown>): string {
  for (const key of ["title", "citation", "label", "key", "description", "posture", "text"]) {
    const value = row[key];
    if (typeof value === "string" && value.trim()) return value.trim().slice(0, 200);
  }
  return String(row.id ?? "record");
}

async function openStore(): Promise<StoreOk> {
  const store = await getStore();
  if (!store.available) throw Object.assign(new Error(`Case store unavailable: ${store.reason}`), { status: 503 });
  return store as StoreOk;
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Parse a non-negative query integer within the caller's explicit bounds.
 * Inputs: URL query parameters, parameter name, default, minimum and maximum.
 * Outputs: the validated integer, or a 400 error for malformed, fractional, unsafe or out-of-range input.
 * Side effects: none.
 * Use this helper for bounded pagination parameters; do not use it for optional text filters.
 */
function parseInteger(url: URL, name: string, fallback: number, min: number, max: number): number {
  const raw = url.searchParams.get(name);
  if (raw === null) return fallback;
  if (!/^(0|[1-9]\d*)$/.test(raw)) throw Object.assign(new Error(`${name} must be a whole number between ${min} and ${max}.`), { status: 400 });
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < min || value > max) {
    throw Object.assign(new Error(`${name} must be a whole number between ${min} and ${max}.`), { status: 400 });
  }
  return value;
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Parse the bounded page size and numeric offset cursor for the records endpoint.
 * Inputs: the request URL with optional `limit` and `cursor` query values.
 * Outputs: a page size from 1 to 1,000 and an offset from 0 to 2,147,483,647; omitted values default to 200 and 0.
 * Side effects: none; invalid values become client-visible HTTP 400 errors.
 * Use this for `/api/records`; file-library paging is a separate in-memory presentation concern.
 */
function recordPageRequest(url: URL): { limit: number; offset: number } {
  return { limit: parseInteger(url, "limit", 200, 1, 1000), offset: parseInteger(url, "cursor", 0, 0, 2_147_483_647) };
}

// Byline: Codex · GPT-6 · 2026-10-04
/** Read one id-ordered table page and a separate total count without materializing the full table.
 * Inputs: an allowlisted table name, a bounded row limit and a numeric offset.
 * Outputs: normalized record envelopes plus the table count; the offset is not a snapshot token, so concurrent writes can shift later pages.
 * Side effects: performs bounded page and count reads against the shared Surreal case store.
 * Use this for phone-visible case-store tables; prefer the dedicated record lookup for one known ID.
 */
async function listRecords(table: string, limit: number, offset: number): Promise<{ records: Record<string, unknown>[]; total: number }> {
  if (!(DATA_TABLES as readonly string[]).includes(table)) throw Object.assign(new Error(`Unknown table "${table}".`), { status: 400 });
  const store = await openStore();
  const [pageResult, countResult] = await Promise.all([
    store.db.query<Array<Record<string, unknown>>>(
      `SELECT * OMIT ${HEAVY_FIELDS.join(", ")} FROM type::table($table) ORDER BY id ASC LIMIT ${limit} START ${offset};`,
      { table },
    ),
    store.db.query<Array<{ count: number }>>(`SELECT count() AS count FROM type::table($table) GROUP ALL;`, { table }),
  ]);
  const records = (pageResult.at(-1) ?? []).map((raw) => {
    const row = normalize(raw) as Record<string, unknown>;
    return { id: String(row.id), table, kind: typeof row.kind === "string" ? row.kind : table, title: recordTitle(row), record: row };
  });
  const total = Number(countResult.at(-1)?.[0]?.count ?? 0);
  return { records, total };
}

async function walk(dir: string, root: string, out: Array<Record<string, unknown>>): Promise<void> {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    if (entry.name.startsWith(".") || entry.name === "__pycache__" || entry.name === "node_modules") continue;
    const full = join(dir, entry.name);
    if (entry.isDirectory()) await walk(full, root, out);
    else if ([".md", ".json", ".txt", ".pdf", ".html"].includes(extname(entry.name).toLowerCase())) {
      const info = await stat(full);
      out.push({ path: relative(root, full).split(sep).join("/"), size: info.size });
    }
  }
}

let libraryCache: Array<Record<string, unknown>> | null = null;

async function library(): Promise<Array<Record<string, unknown>>> {
  if (libraryCache) return libraryCache;
  const out: Array<Record<string, unknown>> = [];
  for (const root of CONTENT_ROOTS) {
    try {
      await walk(root, join(root, ".."), out);
    } catch {
      // A missing content root (e.g. skills/ in a trimmed image) lists nothing.
    }
  }
  out.sort((a, b) => String(a.path).localeCompare(String(b.path)));
  libraryCache = out;
  return out;
}

function resolveContentPath(path: string): string | null {
  const full = normalizePath(join(here, "..", "..", path));
  if (!CONTENT_ROOTS.some((root) => full.startsWith(root + sep))) return null;
  return full;
}

async function serveStatic(res: ServerResponse, file: string): Promise<void> {
  const body = await readFile(join(WEB_DIR, file));
  res.writeHead(200, { "content-type": MIME[extname(file)] ?? "application/octet-stream", "cache-control": "no-cache" });
  res.end(body);
}

/** Returns true when the request was a web-app route and has been answered. */
export async function handleWebRequest(req: IncomingMessage, res: ServerResponse, url: URL, buildServer: () => McpServer): Promise<boolean> {
  const path = url.pathname;
  const method = req.method ?? "GET";
  const isWeb = path === "/" || path === "/index.html" || path === "/app.css" || path === "/tokens.css" || path === "/host.js" || path.startsWith("/api/");
  if (!isWeb) return false;
  try {
    if (method === "GET" && path === "/api/health") {
      const store = await getStore();
      send(res, 200, { ok: true, store: store.available ? "connected" : "unavailable" });
      return true;
    }

    const who: Principal | null = authenticate(req.socket.remoteAddress, req.headers, authConfig);
    if (!who) {
      console.error(`family-law-toolkit web: 403 for peer ${req.socket.remoteAddress ?? "unknown"} ${method} ${path}`);
      send(res, 403, { error: "forbidden" });
      return true;
    }

    if (method === "GET" && path === "/api/library/original") {
      await proxyLibraryOriginal(url, res);
      return true;
    }

    if (method === "GET" && (path === "/" || path === "/index.html")) {
      await serveStatic(res, "index.html");
      return true;
    }
    if (method === "GET" && (path === "/app.css" || path === "/tokens.css" || path === "/host.js")) {
      await serveStatic(res, path.slice(1));
      return true;
    }
    if (method === "GET" && path === "/api/whoami") {
      send(res, 200, { principal: who.principal, via: who.via });
      return true;
    }
    if (method === "GET" && path === "/api/tools") {
      const client = await toolClient(buildServer);
      const { tools } = await client.listTools();
      send(res, 200, { tools });
      return true;
    }
    const toolMatch = /^\/api\/tools\/([a-z0-9_]+)$/.exec(path);
    if (method === "POST" && toolMatch) {
      const body = await readJsonBody(req);
      const client = await toolClient(buildServer);
      const result = await client.callTool({ name: toolMatch[1], arguments: (body.arguments as Record<string, unknown>) ?? {} });
      send(res, 200, result);
      return true;
    }
    const widgetMatch = /^\/api\/widgets\/([a-z]+)$/.exec(path);
    if (method === "GET" && widgetMatch) {
      const html = (widgets as Record<string, string>)[widgetMatch[1]];
      if (!html) {
        send(res, 404, { error: "no such widget" });
        return true;
      }
      res.writeHead(200, { "content-type": "text/html; charset=utf-8", "cache-control": "no-cache" });
      res.end(html);
      return true;
    }
    if (method === "GET" && path === "/api/library") {
      send(res, 200, { files: await library() });
      return true;
    }
    if (method === "GET" && path === "/api/library/file") {
      const full = resolveContentPath(url.searchParams.get("path") ?? "");
      if (!full) {
        send(res, 400, { error: "path must be inside content/ or skills/" });
        return true;
      }
      const bytes = await readFile(full);
      const version = `sha256:${sha256(bytes)}`;
      if (url.searchParams.get("raw") === "1") {
        res.writeHead(200, { "content-type": MIME[extname(full)] ?? "application/octet-stream", "x-content-version": version });
        res.end(bytes);
        return true;
      }
      const text = extname(full) === ".pdf" ? null : bytes.toString("utf8");
      send(res, 200, { path: url.searchParams.get("path"), version, size: bytes.length, text });
      return true;
    }
    if (method === "GET" && path === "/api/records") {
      // Byline: Codex · GPT-6 · 2026-10-04
      /** Return one bounded page from an allowlisted case-store table.
       * Inputs: authenticated GET with `table`, optional `limit`, and numeric offset `cursor` query parameters.
       * Outputs: backward-compatible `records` plus `total` and nullable `nextCursor` metadata.
       * Side effects: reads one bounded page and a count from shared `fct/case`; offset paging is not snapshot-isolated across concurrent writes.
       * Use this collection route for browsing; `/api/records/:table:id` is the exact-record sibling.
       */
      const { limit, offset } = recordPageRequest(url);
      const page = await listRecords(url.searchParams.get("table") ?? "", limit, offset);
      const nextOffset = offset + page.records.length;
      send(res, 200, {
        ...page,
        nextCursor: nextOffset < page.total ? String(nextOffset) : null,
      });
      return true;
    }
    const recordMatch = /^\/api\/records\/(.+)$/.exec(path);
    if (method === "GET" && recordMatch) {
      const ref = decodeURIComponent(recordMatch[1]);
      if (!/^[a-z_]+:.+$/.test(ref)) {
        send(res, 400, { error: "expected /api/records/<table>:<id>" });
        return true;
      }
      const record = await caseRecord(await openStore(), ref);
      if (record) send(res, 200, { ...record, owner: "family-court-toolkit/surreal-case" });
      else send(res, 404, { error: "not found", id: ref });
      return true;
    }
    send(res, 404, { error: "not found" });
    return true;
  } catch (err) {
    const status = (err as { status?: number }).status ?? 500;
    if (!res.headersSent) send(res, status, { error: (err as Error).message ?? String(err) });
    else res.end();
    return true;
  }
}
