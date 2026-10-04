#!/usr/bin/env node
// Byline: Claude Code · Sonnet 5 · 2026-09-07
//
// Node sidecar for the Family Court Workbench Tauri app. Binds to 127.0.0.1
// only (loopback — never 0.0.0.0) on an ephemeral port by default (PORT=0),
// because both the Agent SDK and the plugin's SurrealDB case store client
// (a shared server at ws://127.0.0.1:8471, see lib/store-client.mjs) need
// a real Node runtime, which a pure webview cannot provide.
//
// On startup this prints exactly one machine-readable line to stdout:
//   SIDECAR_READY {"port":NNNN}
// so a Tauri Rust parent (see ../src-tauri/src/main.rs) can capture the
// OS-assigned port and hand it to the webview via a `sidecar_port` command,
// without the port (or any secret) ever being baked into the frontend
// bundle. When run standalone (`npm run dev:sidecar` / `npm run dev`), set
// PORT to a fixed value (see .env.example) for a stable dev URL.
//
// Security: loopback-only bind, no auth token ever leaves this process (the
// Agent SDK spawns its own CLI subprocess in-process; the webview only ever
// sees this HTTP API, never the token). See ../docs/ARCHITECTURE.md.

import { pathToFileURL } from "node:url";
import cors from "@fastify/cors";
import Fastify from "fastify";
import { describeAuth } from "./lib/auth.mjs";
import { checkAuthAvailable, streamChat } from "./lib/chat.mjs";
import { callStoreFn, getSharedStore, loadStoreModule, TIMELINE_MODES } from "./lib/store-client.mjs";

// Default to a fixed port (4177) for plain `npm run dev` / `npm run
// dev:sidecar`, where a browser tab needs a predictable URL to talk to
// (VITE_SIDECAR_PORT in .env / .env.example matches this default). The
// Tauri production path (src-tauri/src/main.rs) explicitly sets PORT=0 when
// spawning this process so the OS assigns a real ephemeral port, which it
// then reads back off this process's stdout and hands to the webview via
// the `sidecar_port` Tauri command — the port is never hardcoded into the
// frontend bundle for that path.
const PORT = Number(process.env.PORT ?? 4177);
const HOST = "127.0.0.1";

const app = Fastify({ logger: false });
await app.register(cors, { origin: [/^https?:\/\/127\.0\.0\.1(:\d+)?$/, /^tauri:\/\//, /^https:\/\/tauri\.localhost$/] });

app.get("/api/health", async () => ({ ok: true, pid: process.pid }));

app.get("/api/auth/status", async () => ({
  ...describeAuth(),
  chat: checkAuthAvailable(),
}));

// ---------------------------------------------------------------------------
// Store surface — thin JSON wrappers over mcp-app/dist/store.js
// ---------------------------------------------------------------------------

app.get("/api/store/summary", async (_req, reply) => {
  try {
    const store = await getSharedStore();
    if (store.available === false) return { available: false, reason: store.reason };
    const summary = await callStoreFn("caseSummary");
    return summary;
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/search", async (req, reply) => {
  const { q, mode, k, tables } = req.query;
  if (!q) {
    reply.code(400);
    return { error: "missing required query param: q" };
  }
  try {
    return await callStoreFn("caseSearch", {
      query: q,
      mode: mode ?? "hybrid",
      k: k ? Number(k) : undefined,
      tables: tables ? String(tables).split(",") : undefined,
    });
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/graph", async (req, reply) => {
  const { id, depth, edges } = req.query;
  if (!id) {
    reply.code(400);
    return { error: "missing required query param: id" };
  }
  try {
    return await callStoreFn("caseGraph", {
      id,
      depth: depth ? Number(depth) : undefined,
      edges: edges ? String(edges).split(",") : undefined,
    });
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

// case_timeline is mode-aware (court | master | merged).
app.get("/api/store/timeline", async (req, reply) => {
  const { from, to, known_by, tables, mode, upcoming } = req.query;
  const modeRequested = mode && TIMELINE_MODES.includes(mode) ? mode : "merged";
  try {
    const entries = await callStoreFn("caseTimeline", {
      mode: modeRequested,
      from,
      to,
      known_by,
      upcoming: upcoming === undefined ? undefined : upcoming === "true",
      tables: tables ? String(tables).split(",") : undefined,
    });
    return { entries, mode: modeRequested };
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/factor-map", async (_req, reply) => {
  try {
    return { entries: await callStoreFn("caseFactorMap") };
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

// case_docket, case_memo, case_status, case_source, case_reference,
// case_evidence_log, case_eval — store.ts exposes compatibility alias
// functions (caseMemo/caseStatus/caseSource/caseReference/caseEvidenceLog/
// caseEvals) matching these exact callStoreFn names (see store.ts's
// "Sidecar compatibility aliases" comment).

app.get("/api/store/docket", async (req, reply) => {
  const { status, doc_type, in_force } = req.query;
  try {
    const entries = await callStoreFn("caseDocket", {
      status,
      doc_type,
      in_force: in_force === undefined ? undefined : in_force === "true",
    });
    return { entries };
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/memos", async (_req, reply) => {
  try {
    return await callStoreFn("caseMemo");
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/status", async (_req, reply) => {
  try {
    return await callStoreFn("caseStatus");
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/source", async (req, reply) => {
  const { id } = req.query;
  if (!id) {
    reply.code(400);
    return { error: 'missing required query param: id (a "table:id" record ref)' };
  }
  try {
    return await callStoreFn("caseSource", { id });
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/reference", async (req, reply) => {
  const { match } = req.query;
  try {
    return await callStoreFn("caseReference", match ? { match } : {});
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

/**
 * Parses an optional decimal query parameter within an inclusive maximum.
 * Inputs are its raw value, a fallback, and the maximum; output is an integer
 * or null for invalid input. It has no side effects and is used by bounded
 * list routes instead of accepting untrusted paging values directly.
 * Byline: Codex · GPT-6 · 2026-10-04
 */
function parseBoundedQueryInteger(value, fallback, max) {
  if (value === undefined) return fallback;
  if (typeof value !== "string" || !/^\d+$/.test(value)) return null;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed <= max ? parsed : null;
}

/**
 * Returns a bounded page from the canonical shared reference list. Inputs are
 * optional q, limit, and offset query parameters; output is entries plus page
 * metadata. It reads existing Surreal rows without mutation and complements
 * /api/store/reference?match=, which serves ontology match hits.
 * Byline: Codex · GPT-6 · 2026-10-04
 */
app.get("/api/store/reference/library", async (req, reply) => {
  const rawQuery = typeof req.query?.q === "string" ? req.query.q.trim() : "";
  if (rawQuery.length > 120) {
    reply.code(400);
    return { error: "q must be 120 characters or fewer" };
  }
  const limit = parseBoundedQueryInteger(req.query?.limit, 25, 50);
  const offset = parseBoundedQueryInteger(req.query?.offset, 0, 200);
  if (!limit || offset === null) {
    reply.code(400);
    return { error: "limit must be 1–50 and offset must be 0–200" };
  }
  try {
    const result = await callStoreFn("caseReference");
    if (result?.available === false) return result;
    const entries = Array.isArray(result?.entries) ? result.entries : [];
    const needle = rawQuery.toLowerCase();
    const filtered = needle
      ? entries.filter((entry) => {
          const source = entry.source && typeof entry.source === "object" ? entry.source : {};
          const fields = [entry.id, entry.kind, entry.category, entry.pattern, entry.definition,
            ...(Array.isArray(entry.aliases) ? entry.aliases : []), source.path, source.source_path,
            source.sha256, source.hash, source.url, source.source_url, source.r2_path];
          return fields.some((field) => typeof field === "string" && field.toLowerCase().includes(needle));
        })
      : entries;
    return {
      entries: filtered.slice(offset, offset + limit),
      total: filtered.length,
      offset,
      limit,
      next_offset: offset + limit < filtered.length ? offset + limit : null,
    };
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

// Evidence: blends the live `exhibit` search results with the separate
// `evidence_log` register.
app.get("/api/store/evidence", async (_req, reply) => {
  try {
    const [exhibits, evidenceLog] = await Promise.all([
      callStoreFn("caseSearch", { query: "*", tables: ["exhibit"], k: 200, mode: "text" }).catch((e) => ({ error: String(e?.message ?? e) })),
      callStoreFn("caseEvidenceLog"),
    ]);
    return { exhibits, evidence_log: evidenceLog };
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

app.get("/api/store/evals", async (_req, reply) => {
  try {
    return await callStoreFn("caseEvals");
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

// case_export supports both "snapshot" (caseExport, default) and "platform"
// (caseExportPlatform) formats.
app.post("/api/store/export", async (req, reply) => {
  const format = req.body?.format ?? "snapshot";
  try {
    const store = await getSharedStore();
    if (store.available === false) return { available: false, reason: store.reason };
    const mod = await loadStoreModule();
    if (format === "platform") {
      return await mod.caseExportPlatform(store, req.body?.dir);
    }
    if (format !== "snapshot") {
      reply.code(400);
      return { error: `unknown export format "${format}" — expected "snapshot" or "platform"` };
    }
    return await mod.caseExport(store, req.body?.path);
  } catch (err) {
    reply.code(500);
    return { error: String(err?.message ?? err) };
  }
});

// ---------------------------------------------------------------------------
// Chat (SSE) — Agent SDK
// ---------------------------------------------------------------------------

app.post("/api/chat", async (req, reply) => {
  const { prompt, sessionId } = req.body ?? {};
  if (!prompt || typeof prompt !== "string") {
    reply.code(400);
    return { error: "missing required body field: prompt (string)" };
  }
  await streamChat(reply, { prompt, sessionId });
  return reply;
});

// Only bind a real socket when this file is run directly (`node
// sidecar/server.mjs`) — tests import `app` via `sidecar/tests/*.test.mjs`
// and drive it with Fastify's `.inject()`, which never touches the network.
const isMain = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;

if (isMain) {
  app.listen({ port: PORT, host: HOST }, (err, address) => {
    if (err) {
      console.error(err);
      process.exit(1);
    }
    const actualPort = Number(new URL(address).port);
    // Machine-readable line consumed by src-tauri/src/main.rs. Keep this the
    // ONLY thing this process prints to stdout on startup.
    console.log(`SIDECAR_READY ${JSON.stringify({ port: actualPort })}`);
  });

  process.on("SIGTERM", () => process.exit(0));
  process.on("SIGINT", () => process.exit(0));
}

export default app;
