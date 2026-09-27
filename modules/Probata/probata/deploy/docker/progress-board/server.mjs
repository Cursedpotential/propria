// Byline: Codex · 2026-09-12 — read-only Propria progress projection.
// 2026-09-14 (Claude Code · Sonnet 5): added native-widget JSON endpoints
// (lane rollups, surfaces summary, health summary, to-do feed, timeline,
// health history ring buffer) and a locally vendored ApexCharts static
// route, for the Homepage portal native-widget rework. Read-only additions;
// no change to existing routes' behaviour.
import http from "node:http";
import { readFile, stat } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { resolve, extname, sep } from "node:path";
import { createHash } from "node:crypto";
import { cachedHealth } from "./health.mjs";
import {createProviderLimits} from './provider-limits.mjs';
import { createRepairQueue } from "./repair-queue.mjs";
import { createOpenListBridge, StorageError } from "./openlist-bridge.mjs";
import { createCatalog } from "./pg-catalog.mjs";

// Hosted Intake engine (Coolify app intake-engine on ovh-files, tailnet only).
const INTAKE_ENGINE_URL = process.env.INTAKE_ENGINE_URL || "http://100.91.190.107:8790";

// Origins allowed to read the JSON board endpoints cross-origin (Homepage's
// custom.js runs on the Homepage origin, not progress-board's, so its
// browser-side fetch() calls need CORS; customapi widgets are fetched
// server-side by Homepage's backend and do not need it, but the header is
// harmless there too). Read-only, unauthenticated, non-sensitive endpoints.
const CORS_ALLOWED_ORIGINS = new Set([
  "https://homepage.tilapia-skilift.ts.net",
  "https://homepage.int.mitechconsult.com",
  "http://100.72.169.40:3010",
  "http://100.72.169.40:3012",
]);
function applyCors(req, res) {
  const origin = req.headers.origin;
  if (origin && CORS_ALLOWED_ORIGINS.has(origin)) {
    res.setHeader("Access-Control-Allow-Origin", origin);
    res.setHeader("Vary", "Origin");
  }
}
const HEALTH_HISTORY_MAX = 120; // ~1h at a 30s sample interval
const healthHistory = [];
function summarizeHealthSample(h) {
  const endpoints = Array.isArray(h.endpoints) ? h.endpoints : [];
  const latencies = endpoints
    .map((e) => e.latency_ms)
    .filter((n) => Number.isFinite(n))
    .sort((a, b) => a - b);
  const pct = (p) =>
    latencies.length
      ? latencies[Math.min(latencies.length - 1, Math.floor(p * latencies.length))]
      : null;
  const up = endpoints.filter((e) =>
    ["responding", "TCP open"].includes(e.state),
  ).length;
  const warn = endpoints.filter((e) =>
    ["auth required", "redirect", "unexpected content"].includes(e.state),
  ).length;
  const fail = endpoints.length - up - warn;
  return {
    ts: h.generated_at,
    ok: up,
    warn,
    fail,
    total: endpoints.length,
    p50_ms: pct(0.5),
    p95_ms: pct(0.95),
  };
}
export function recordHealthHistory(h, store = healthHistory, max = HEALTH_HISTORY_MAX) {
  store.push(summarizeHealthSample(h));
  while (store.length > max) store.shift();
  return store;
}
function classifySurfaceState(state) {
  if (["responding", "TCP open"].includes(state)) return "up";
  if (state === 'monitor access denied') return 'monitor_denied';
  if (state === "auth required") return "auth_required";
  if (state === "not connected") return "unconfigured";
  if (["redirect", "unexpected content"].includes(state))
    return "degraded";
  return "down";
}
export async function readTodoFeed(path) {
  let text;
  try {
    text = await readFile(path, "utf8");
  } catch {
    return { status: "not-synced", message: "No synced copy of URGENT-TODO.md yet.", items: [] };
  }
  const lines = text.split(/\r?\n/);
  let sectionDate = null;
  const items = [];
  for (const line of lines) {
    // Headings may carry text after the date ("## 2026-09-18 — Intake v1 follow-ups").
    const header = line.match(/^##\s+(\d{4}-\d{2}-\d{2})\b/);
    if (header) {
      sectionDate = header[1];
      continue;
    }
    const open = line.match(/^-\s\[\s\]\s+(.*)$/);
    if (!open) continue;
    const title = plain(open[1].replace(/\*\*/g, "").trim(), 220);
    if (!title) continue;
    let ageDays = null;
    if (sectionDate) {
      const ms = Date.now() - Date.parse(`${sectionDate}T00:00:00Z`);
      if (Number.isFinite(ms)) ageDays = Math.max(0, Math.floor(ms / 86400000));
    }
    const age =
      ageDays === null ? "" : ageDays === 0 ? " (today)" : ` (${ageDays}d ago)`;
    const shortTitle = title.length > 90 ? `${title.slice(0, 87)}...` : title;
    items.push({
      title,
      area: sectionDate,
      age_days: ageDays,
      summary: `${shortTitle}${age}`,
    });
  }
  items.reverse(); // file is oldest-section-first; newest section last
  return { status: "ok", message: null, items: items.slice(0, 60) };
}

const root = fileURLToPath(new URL(".", import.meta.url));
export function safeAssetPath(base, requestPath) {
  let decoded;
  try {
    decoded = decodeURIComponent(requestPath);
  } catch {
    return null;
  }
  if (decoded.includes("\\") || decoded.includes("\0")) return null;
  const file = resolve(base, decoded || "index.html");
  return file.startsWith(resolve(base) + sep) ? file : null;
}
const assetTypes = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".mjs": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".svg": "image/svg+xml",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".wasm": "application/wasm",
  ".gif": "image/gif",
};
export const QUERY = `SELECT id, title, detail, project, owner, status, priority, due, source_doc, updated, closed_at FROM todo ORDER BY updated DESC LIMIT 501;
SELECT id, lane, kind, title, status, owner, detail, source_url, observed_at, updated_at, heartbeat_at, checked_at, upcoming FROM portal_observation ORDER BY observed_at DESC LIMIT 301;`;
const plain = (value, max = 2000) =>
  typeof value === "string" ? value.slice(0, max) : "";
const date = (value) =>
  typeof value === "string" && Number.isFinite(Date.parse(value))
    ? new Date(value).toISOString()
    : null;
export function safeUrl(value) {
  try {
    const url = new URL(value);
    return ["https:", "http:"].includes(url.protocol) &&
      !url.username &&
      !url.password
      ? url.href
      : null;
  } catch {
    return null;
  }
}
export function configFromEnv(env = process.env) {
  let endpoint = null;
  try {
    const u = new URL(env.SURREAL_DOCS_URL);
    u.protocol =
      u.protocol === "wss:"
        ? "https:"
        : u.protocol === "ws:"
          ? "http:"
          : u.protocol;
    if (!["http:", "https:"].includes(u.protocol) || u.username || u.password)
      throw Error();
    u.pathname =
      u.pathname.replace(/\/(rpc|sql)\/?$/, "").replace(/\/$/, "") + "/sql";
    u.search = "";
    u.hash = "";
    endpoint = u.href;
  } catch {}
  return {
    endpoint,
    user: env.SURREAL_DOCS_USER,
    pass: env.SURREAL_DOCS_PASS,
    namespace: env.SURREAL_DOCS_NS || "probata",
    database: env.SURREAL_DOCS_DB || "docs",
  };
}
export function projectTask(row) {
  return {
    id: plain(row.id, 160),
    title: plain(row.title, 300),
    detail: plain(row.detail),
    lane: plain(row.project, 100) || "Unassigned",
    owner: plain(row.owner, 160) || "Unassigned",
    status: plain(row.status, 40) || "unknown",
    priority: Number.isInteger(row.priority) ? row.priority : null,
    due: date(row.due),
    source_doc: plain(row.source_doc, 200),
    updated_at: date(row.updated),
    closed_at: date(row.closed_at),
  };
}
export function projectObservation(row) {
  return {
    id: plain(row.id, 160),
    origin: ['portal_observation:portal_task_coolify_mcp_connector','portal_observation:portal_task_openlist_intake_bridge'].includes(String(row.id))?{agent:'codex',session_id:'01a09620-155b-7b01-8ab3-da15cfd419a9',verified:true}:null,
    lane: plain(row.lane, 100) || "Unassigned",
    kind: plain(row.kind, 40),
    title: plain(row.title, 300),
    status: plain(row.status, 80),
    owner: plain(row.owner, 160),
    detail: plain(row.detail),
    source_url: safeUrl(row.source_url),
    observed_at: date(row.observed_at),
    updated_at: date(row.updated_at),
    heartbeat_at: date(row.heartbeat_at),
    checked_at: date(row.checked_at),
    upcoming: Array.isArray(row.upcoming)
      ? row.upcoming.slice(0, 20).map((v) => plain(v, 300))
      : [],
  };
}
export async function readBoard(
  config,
  fetcher = fetch,
  now = () => new Date().toISOString(),
) {
  const base = {
    fetched_at: now(),
    tasks: [],
    observations: [],
    sources: {},
    truncated: false,
  };
  if (!config.endpoint || !config.user || !config.pass)
    return {
      ...base,
      status: "unavailable",
      message: "SurrealDB connection is not configured. Task state is unknown.",
    };
  try {
    const response = await fetcher(config.endpoint, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "text/plain",
        "Surreal-NS": config.namespace,
        "Surreal-DB": config.database,
        "Surreal-Auth-NS": config.namespace,
        "Surreal-Auth-DB": config.database,
        Authorization: `Basic ${Buffer.from(`${config.user}:${config.pass}`).toString("base64")}`,
      },
      body: QUERY,
      signal: AbortSignal.timeout(8000),
      redirect: "error",
    });
    if (!response.ok) throw Error("database-response");
    const length = Number(response.headers?.get("content-length") || 0);
    if (length > 4000000) throw Error("database-response-size");
    const body = await response.json();
    if (!Array.isArray(body) || body.length !== 2)
      throw Error("database-shape");
    const todoOk = body[0]?.status === "OK" && Array.isArray(body[0].result);
    const observationsOk =
      body[1]?.status === "OK" && Array.isArray(body[1].result);
    return {
      ...base,
      status: todoOk && observationsOk ? "connected" : "unavailable",
      message:
        todoOk && observationsOk
          ? "SurrealDB read completed. Reports retain their own timestamps."
          : "One or more source queries failed. Missing state is unknown.",
      tasks: todoOk ? body[0].result.slice(0, 500).map(projectTask) : [],
      observations: observationsOk
        ? body[1].result.slice(0, 300).map(projectObservation)
        : [],
      sources: {
        todo: todoOk ? "available" : "unavailable",
        portal_observation: observationsOk ? "available" : "unavailable",
      },
      truncated:
        (todoOk && body[0].result.length > 500) ||
        (observationsOk && body[1].result.length > 300),
    };
  } catch {
    return {
      ...base,
      status: "unavailable",
      message:
        "SurrealDB could not be read. Check the server connection and reader permissions. Task state is unknown.",
    };
  }
}
export async function readSurfaces(path = resolve(root, "surfaces.json")) {
  try {
    const rows = JSON.parse(await readFile(path, "utf8"));
    return Array.isArray(rows)
      ? rows.slice(0, 80).map((s) => ({
          name: plain(s.name, 120),
          lane: plain(s.lane, 100),
          url: safeUrl(s.url),
          status: plain(s.status, 80) || "Not checked",
          updated_at: date(s.updated_at),
          checked_at: date(s.checked_at),
        }))
      : [];
  } catch {
    return [];
  }
}
export function createServer({
  config = configFromEnv(),
  fetcher = fetch,
  surfacesPath,
  todoPath = resolve(root, "data", "URGENT-TODO.md"),
  cacheMs = 10000,
} = {}) {
  const usageLimits=createProviderLimits();
  const repairQueue = createRepairQueue({readLimits:usageLimits.read,readTasks:async()=>{const value=await board();return [...value.tasks,...value.observations.filter(row=>row.kind==='task')];},readSurfaces: () => readSurfaces(surfacesPath), readHealth: cachedHealth});
  const storage = createOpenListBridge({
    token: process.env.OPENLIST_TOKEN,
    fetchImpl: fetcher,
  });
  const catalog = createCatalog();
  let cache,
    cacheAt = 0,
    pending;
  const board = async () => {
    if (cache && Date.now() - cacheAt < cacheMs) return cache;
    if (!pending)
      pending = readBoard(config, fetcher)
        .then((value) => {
          cache = value;
          cacheAt = Date.now();
          return value;
        })
        .finally(() => {
          pending = null;
        });
    return pending;
  };
  return http.createServer(async (req, res) => {
    applyCors(req, res);
    res.setHeader("Cache-Control", "no-store");
    res.setHeader("X-Content-Type-Options", "nosniff");
    res.setHeader("Referrer-Policy", "no-referrer");
    res.setHeader(
      "Content-Security-Policy",
      "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; form-action 'none'",
    );
    const incomingPath = new URL(req.url, "http://local").pathname;
    if (incomingPath === "/progress") {
      res.writeHead(308, { Location: "/progress/" });
      res.end();
      return;
    }
    const path = incomingPath.startsWith("/progress/")
      ? incomingPath.slice("/progress".length)
      : incomingPath;
    try {
      if (path === "/api/repair-jobs") { await repairQueue(req, res); return; }
      // Byline: Claude Code · Opus 5 · 2026-09-17 — hosted Intake engine cutover (owner decision 9):
      // /intake/storage/api/* is a streaming reverse proxy to the intake-engine Coolify app on
      // ovh-files (donor Rust handlers, web-mode contract: POST /api/<command>, GET /api/asset with
      // Range, SSE /api/events/<event>). The Node OpenList bridge is no longer the backend.
      // Rollback: restore server.mjs.bak-20260917-pre-intake-engine-cutover and restart the service.
      if (path.startsWith("/intake/storage/api/")) {
        const upstream = new URL(
          path.slice("/intake/storage".length) + new URL(req.url, "http://local").search,
          INTAKE_ENGINE_URL,
        );
        const headers = {};
        for (const h of ["content-type", "content-length", "range", "accept", "last-event-id", "if-none-match", "if-modified-since"]) {
          if (req.headers[h]) headers[h] = req.headers[h];
        }
        await new Promise((done) => {
          const up = http.request(
            upstream,
            { method: req.method, headers, timeout: 0 },
            (ur) => {
              const out = {};
              for (const [k, v] of Object.entries(ur.headers)) {
                if (!["connection", "keep-alive", "transfer-encoding"].includes(k)) out[k] = v;
              }
              res.removeHeader("Content-Security-Policy");
              res.removeHeader("Cache-Control");
              if (!out["content-security-policy"])
                out["content-security-policy"] = "default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'; sandbox allow-scripts";
              if (String(ur.headers["content-type"] || "").startsWith("text/event-stream")) {
                out["cache-control"] = "no-cache";
                out["x-accel-buffering"] = "no";
              }
              res.writeHead(ur.statusCode || 502, out);
              if (typeof res.flushHeaders === "function") res.flushHeaders();
              ur.pipe(res);
              ur.on("end", done);
              ur.on("error", done);
              res.on("close", () => {
                ur.destroy();
                done();
              });
            },
          );
          up.on("error", (e) => {
            if (!res.headersSent) {
              res.writeHead(502, { "Content-Type": "text/plain; charset=utf-8" });
              res.end(`Intake engine unreachable: ${e.message}`);
            } else res.destroy();
            done();
          });
          req.on("aborted", () => up.destroy());
          if (req.method === "GET" || req.method === "HEAD") up.end();
          else req.pipe(up);
        });
        return;
      }
      if(path==='/api/provider-limits'){await usageLimits.handle(req,res);return;}
      if (req.method !== "GET" && req.method !== "HEAD") {
        res.writeHead(405, { Allow: "GET, HEAD" });
        res.end();
        return;
      }
      if (path === "/healthz") {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ status: "ok", service: "progress-board" }));
        return;
      }
      if (path === "/api/portal-config-version") {
        // Byline: Claude Code · Sonnet 5 · 2026-09-15 — portal-editor
        // auto-refresh signal. Read-only: max mtime (ms) of the config
        // files that change the rendered page, per Homepage instance.
        // `dir` is validated against a fixed allowlist, never used to
        // build an arbitrary path (no traversal surface).
        const q = new URL(req.url, "http://local").searchParams;
        const ALLOWED_DIRS = {
          homepage: "/data/dashboards/homepage",
          "homepage-public": "/data/dashboards/homepage-public",
        };
        const dir = q.get("dir");
        const base = ALLOWED_DIRS[dir];
        if (!base) {
          res.writeHead(400, { "Content-Type": "application/json" });
          res.end(
            JSON.stringify({
              error: "dir must be one of: " + Object.keys(ALLOWED_DIRS).join(", "),
            }),
          );
          return;
        }
        const WATCHED_FILES = [
          "services.yaml",
          "widgets.yaml",
          "settings.yaml",
          "custom.css",
          "custom.js",
        ];
        let version = 0;
        for (const f of WATCHED_FILES) {
          try {
            const st = await stat(resolve(base, f));
            if (st.mtimeMs > version) version = st.mtimeMs;
          } catch {
            // missing file: ignore, doesn't block the version signal
          }
        }
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ dir, version: Math.round(version) }));
        return;
      }
      if (path === "/api/health-report") {
        const value = await cachedHealth();
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify(value));
        return;
      }
      if (path === "/health") {
        res.writeHead(308, { Location: incomingPath + "/" });
        res.end();
        return;
      }
      if (path === "/health/") {
        const content = await readFile(
          resolve(root, "outputs", "live-service-storage-health.html"),
          "utf8",
        );
        const scripts = [
          ...content.matchAll(/<script>([\s\S]*?)<\/script>/g),
        ].map(
          (m) =>
            `'sha256-${createHash("sha256").update(m[1]).digest("base64")}'`,
        );
        const styles = [...content.matchAll(/<style>([\s\S]*?)<\/style>/g)].map(
          (m) =>
            `'sha256-${createHash("sha256").update(m[1]).digest("base64")}'`,
        );
        res.setHeader(
          "Content-Security-Policy",
          `default-src 'self'; script-src ${scripts.join(" ")}; style-src ${styles.join(" ")}; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'`,
        );
        res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
        res.end(req.method === "HEAD" ? undefined : content);
        return;
      }
      if (path === "/api/board") {
        const value = await board();
        res.writeHead(value.status === "connected" ? 200 : 503, {
          "Content-Type": "application/json",
        });
        res.end(
          JSON.stringify({
            ...value,
            served_at: new Date().toISOString(),
            surfaces: await readSurfaces(surfacesPath),
          }),
        );
        return;
      }
      if (path === "/api/lane-rollups") {
        const value = await board();
        const observations = Array.isArray(value.observations)
          ? value.observations
          : [];
        const byLane = new Map();
        for (const o of observations) {
          const lane = o.lane || "Unassigned";
          if (!byLane.has(lane))
            byLane.set(lane, { lane, total: 0, statuses: new Map(), last_updated: null });
          const entry = byLane.get(lane);
          entry.total += 1;
          const status = (o.status || "unknown").toLowerCase().slice(0, 40);
          entry.statuses.set(status, (entry.statuses.get(status) || 0) + 1);
          // A probe or first observation is not an application update.
          const updated = o.updated_at && Number.isFinite(Date.parse(o.updated_at)) ? new Date(o.updated_at).toISOString() : null;
          if (updated && (!entry.last_updated || updated > entry.last_updated))
            entry.last_updated = updated;
        }
        // 2026-09-15 (Claude Code · Sonnet 5): owner rule 23:26 — native widgets
        // must be glanceable, not cramped text. `summary` used to join EVERY
        // distinct status found for a lane (observed live: "8 total · http 200 ·
        // reachability only 5 · http 401 · sign-in required 1 · done 1 · blocked
        // 1" — a run-on sentence for lanes with mixed vocab). Homepage's
        // dynamic-list widget only supports one name+label pair per row (no
        // chips — checked against gethomepage docs), so the fix is a shorter
        // string: top-3 statuses by count, "+N more" for the rest. Full detail
        // stays one click away via the widget's href to the live board.
        const LANE_SUMMARY_TOP_N = 3;
        const lanes = [...byLane.values()]
          .map((e) => {
            const sorted = [...e.statuses.entries()].sort((a, b) => b[1] - a[1]);
            const parts = sorted
              .slice(0, LANE_SUMMARY_TOP_N)
              .map(([status, n]) => `${status} ${n}`);
            const extra = sorted.length - LANE_SUMMARY_TOP_N;
            const compact =
              extra > 0 ? `${parts.join(" · ")} · +${extra} more` : parts.join(" · ");
            return {
              lane: e.lane,
              total: e.total,
              last_updated: e.last_updated,
              status_breakdown: sorted.map(([status, count]) => ({ status, count })),
              summary: `${e.total} · ${compact}`,
            };
          })
          .sort((a, b) => b.total - a.total);
        res.writeHead(value.status === "connected" ? 200 : 503, {
          "Content-Type": "application/json",
        });
        res.end(
          JSON.stringify({
            generated_at: value.fetched_at,
            status: value.status,
            lane_count: lanes.length,
            lanes,
          }),
        );
        return;
      }
      if (path === "/api/surfaces-summary") {
        const [health, surfaces] = await Promise.all([
          cachedHealth(),
          readSurfaces(surfacesPath),
        ]);
        const laneByName = new Map(surfaces.map((s) => [s.name, s.lane]));
        const endpoints = Array.isArray(health.endpoints) ? health.endpoints : [];
        const rows = endpoints
          .filter((e) => e.kind === "Browser surface")
          .map((e) => {
            const state = classifySurfaceState(e.state);
            return {
              name: e.service,
              lane: laneByName.get(e.service) || "Unassigned",
              url: safeUrl(surfaces.find(s => s.name === e.service)?.url),
              state,
              raw_state: e.state,
              checked_at: e.checked_at,
              // 2026-09-15 (Claude Code · Sonnet 5): dropped the leading
              // `${e.service} · ` — the dynamic-list widget already shows
              // `name` (== e.service) on the left of this row, so repeating
              // it in the label wrapped long service names to two lines
              // (observed live: "SurrealDB Studio / Surrealist · down · not
              // connected"). Same fix as lane-rollups: no data lost, just no
              // longer duplicated.
              summary: `${state} · ${e.state}`,
            };
          });
        const up = rows.filter((r) => r.state === "up").length;
        const degraded = rows.filter((r) => r.state === "degraded").length;
        const down = rows.filter((r) => r.state === "down").length;
        const auth_required = rows.filter(r => r.state === "auth_required").length;
        const unconfigured = rows.filter(r => r.state === "unconfigured").length;
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            generated_at: health.generated_at,
            up,
            degraded,
            down,
            auth_required,
            monitor_denied:rows.filter(r=>r.state==='monitor_denied').length,
            unconfigured,
            total: rows.length,
            down_list: rows
              .filter((r) => r.state !== "up")
              .sort((a, b) => (a.state === "down" ? -1 : 1)),
          }),
        );
        return;
      }
      if (path === "/api/health-summary") {
        const health = await cachedHealth();
        const endpoints = Array.isArray(health.endpoints) ? health.endpoints : [];
        const sample = summarizeHealthSample(health);
        const withLatency = endpoints.filter((e) => Number.isFinite(e.latency_ms));
        const slowest = [...withLatency]
          .sort((a, b) => b.latency_ms - a.latency_ms)
          .slice(0, 5)
          .map((e) => ({
            service: e.service,
            latency_ms: e.latency_ms,
            state: e.state,
            // 2026-09-15 (Claude Code · Sonnet 5): same fix as surfaces-summary
            // above — `name` (== e.service) is already the left side of this
            // dynamic-list row, so the label no longer repeats it.
            summary: `${e.latency_ms} ms · ${e.state}`,
          }));
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            generated_at: health.generated_at,
            ok: sample.ok,
            warn: sample.warn,
            fail: sample.fail,
            total: sample.total,
            p50_ms: sample.p50_ms,
            p95_ms: sample.p95_ms,
            slowest,
          }),
        );
        return;
      }
      if (path === "/api/health-history") {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            generated_at: new Date().toISOString(),
            poll_seconds: 30,
            samples: healthHistory,
          }),
        );
        return;
      }
      if (path === "/api/timeline") {
        const value = await board();
        const observations = Array.isArray(value.observations)
          ? value.observations
          : [];
        const byLane = new Map();
        for (const o of observations) {
          const start = o.observed_at ? Date.parse(o.observed_at) : NaN;
          if (!Number.isFinite(start)) continue;
          let end = o.updated_at
            ? Date.parse(o.updated_at)
            : o.checked_at
              ? Date.parse(o.checked_at)
              : NaN;
          if (!Number.isFinite(end) || end <= start) end = start + 30 * 60 * 1000;
          const lane = o.lane || "Unassigned";
          if (!byLane.has(lane)) byLane.set(lane, []);
          if (byLane.get(lane).length < 40)
            byLane.get(lane).push({
              x: plain(o.title, 80) || plain(o.id, 40) || "Observation",
              y: [start, end],
              status: o.status || "unknown",
            });
        }
        const series = [...byLane.entries()].map(([lane, data]) => ({
          name: lane,
          data,
        }));
        res.writeHead(value.status === "connected" ? 200 : 503, {
          "Content-Type": "application/json",
        });
        res.end(
          JSON.stringify({ generated_at: value.fetched_at, status: value.status, series }),
        );
        return;
      }
      if (path === "/api/todo-feed") {
        const value = await readTodoFeed(todoPath);
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            generated_at: new Date().toISOString(),
            ...value,
            item_count: value.items.length,
          }),
        );
        return;
      }
      if (path === "/vendor/apexcharts.min.js") {
        const content = await readFile(
          resolve(root, "public", "vendor", "apexcharts.min.js"),
        );
        res.writeHead(200, {
          "Content-Type": "text/javascript",
          "Cache-Control": "public, max-age=86400",
        });
        res.end(req.method === "HEAD" ? undefined : content);
        return;
      }
      if (path === "/intake/metadata/api/lookup") {
        const selected = new URL(req.url, "http://local").searchParams.get(
          "path",
        );
        const value = await catalog.lookup(selected);
        res.writeHead(value.error ? 400 : 200, {
          "Content-Type": "application/json",
        });
        res.end(JSON.stringify(value));
        return;
      }
      if (/^\/intake(?:\/(?:xplorer|metadata))?$/.test(path)) {
        res.writeHead(308, { Location: incomingPath + "/" });
        res.end();
        return;
      }
      const intakeFiles = {
        "/intake/": ["index.html", "text/html"],
        "/intake/style.css": ["style.css", "text/css"],
        "/intake/app.js": ["app.js", "text/javascript"],
        "/intake/view.mjs": ["view.mjs", "text/javascript"],
        "/intake/pr-tokens.css": ["pr-tokens.css", "text/css"],
        "/intake/pr-theme.css": ["pr-theme.css", "text/css"],
        "/intake/pr-intake-adapter.css": ["pr-intake-adapter.css", "text/css"],
      };
      if (intakeFiles[path]) {
        const [file, type] = intakeFiles[path];
        const content = await readFile(resolve(root, "intake-preview", file));
        res.writeHead(200, { "Content-Type": `${type}; charset=utf-8` });
        res.end(req.method === "HEAD" ? undefined : content);
        return;
      }
      const intakeAsset = path.match(/^\/intake\/(xplorer|metadata)\/(.*)$/);
      if (intakeAsset) {
        res.setHeader(
          "Content-Security-Policy",
          "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; img-src 'self' data: blob:; font-src 'self' data:; worker-src 'self' blob:; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'",
        );
        const manifest = JSON.parse(
          await readFile(resolve(root, "intake-build", "current.json"), "utf8"),
        );
        if (!/^[a-zA-Z0-9_-]{1,80}$/.test(manifest.release))
          throw Error("Invalid release");
        const base = resolve(
          root,
          "intake-build",
          "releases",
          manifest.release,
          intakeAsset[1],
        );
        const file = safeAssetPath(base, intakeAsset[2]);
        const type = file && assetTypes[extname(file)];
        if (!file || !type) {
          res.writeHead(404);
          res.end("Not found");
          return;
        }
        try {
          const content = await readFile(file);
          res.writeHead(200, { "Content-Type": type });
          res.end(req.method === "HEAD" ? undefined : content);
        } catch {
          res.writeHead(404);
          res.end("Preview asset not found");
        }
        return;
      }
      if (path === "/family-court") {
        res.writeHead(308, { Location: incomingPath + "/" });
        res.end();
        return;
      }
      const familyFiles = {
        "/family-court/": ["index.html", "text/html"],
        "/family-court/styles.css": ["styles.css", "text/css"],
        "/family-court/app.js": ["app.js", "text/javascript"],
      };
      if (familyFiles[path]) {
        const [file, type] = familyFiles[path];
        const content = await readFile(
          resolve(root, "family-court-preview", file),
        );
        res.writeHead(200, { "Content-Type": `${type}; charset=utf-8` });
        res.end(req.method === "HEAD" ? undefined : content);
        return;
      }
      const files = {
        "/": ["index.html", "text/html"],
        '/task-actions.mjs':['task-actions.mjs','text/javascript'],
        '/provider-settings.js':['provider-settings.js','text/javascript'],
        "/app.js": ["app.js", "text/javascript"],
        "/view.mjs": ["view.mjs", "text/javascript"],
        "/style.css": ["style.css", "text/css"],
      };
      if (!files[path]) {
        res.writeHead(404);
        res.end("Not found");
        return;
      }
      const [file, type] = files[path];
      const content = await readFile(resolve(root, "public", file));
      res.writeHead(200, { "Content-Type": `${type}; charset=utf-8` });
      res.end(req.method === "HEAD" ? undefined : content);
    } catch (error) {
      const status = error instanceof StorageError ? error.status : 500;
      res.writeHead(status, { "Content-Type": "text/plain" });
      res.end(
        error instanceof StorageError ? error.message : "Board request failed.",
      );
    }
  });
}
if (
  process.argv[1] &&
  resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  const port = Number(process.env.PORT || 3011);
  const host = process.env.HOST || "127.0.0.1";
  createServer().listen(port, host, () =>
    console.log(`Progress board listening on ${host}:${port}`),
  );
  // Sample health every poll interval into the in-memory ring buffer so the
  // Homepage latency-trend chart has history beyond a single request.
  // Independent of request traffic; guarded to the real running process only
  // (not started when createServer() is imported for tests).
  const sampleOnce = () =>
    cachedHealth()
      .then((h) => recordHealthHistory(h))
      .catch(() => {});
  sampleOnce();
  setInterval(sampleOnce, 30000).unref?.();
}
