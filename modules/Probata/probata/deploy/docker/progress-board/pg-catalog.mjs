// Byline: Claude Code · Sonnet 5 · 2026-09-14
//
// Read-only Case Bible catalog lookup: OpenList path -> B2 key -> raw_duck.b2_content
// / raw_duck.source_occurrences on casebible-pg18 (Coolify database, ovh-files),
// reached over the tailnet at PGCATALOG_HOST:PGCATALOG_PORT (ovh-files tailscale serve tcp 5433,
// deploy/tailscale/catalog-serve.sh; Claude Code · Opus 5.5 · 2026-10-02). Credentials never
// reach the browser bundle -- this module runs server-side only, inside
// progress-board's Node process, and the password is read once at startup from
// a file (never an env var, never logged).
//
// The join key is exactly what was proven live in
// docs/RECEIPT-2026-09-14-XPLORE-CB-CO-WORKSPACE.md: an OpenList path under
// /b2/salem-data/ maps to a B2 object key by stripping that prefix, then both
// raw_duck.b2_content and raw_duck.source_occurrences carry that same b2_key
// column directly (verified live 2026-09-14: joining on b2_key, not just
// (md5,size), is required -- a name/size/mtime-matched OneDrive occurrence for
// the proven example file has md5 = NULL and would be silently dropped by an
// (md5,size) join).
import pg from "pg";
import { readFileSync } from "node:fs";

const B2_PREFIX = "/b2/salem-data/";
const MAX_OCCURRENCES = 200;

function loadPassword(path) {
  try {
    return readFileSync(path, "utf8").trim() || null;
  } catch {
    return null;
  }
}

export function b2KeyFromOpenListPath(openlistPath) {
  if (typeof openlistPath !== "string" || openlistPath.length === 0) return null;
  let decoded;
  try {
    decoded = decodeURIComponent(openlistPath);
  } catch {
    return null;
  }
  if (!decoded.startsWith(B2_PREFIX)) return null;
  const key = decoded.slice(B2_PREFIX.length);
  return key.length ? key : null;
}

export function createCatalog({
  host = process.env.PGCATALOG_HOST || "100.91.190.107",
  port = Number(process.env.PGCATALOG_PORT || 5433),
  user = process.env.PGCATALOG_USER || "metabase_ro",
  database = process.env.PGCATALOG_DATABASE || "casebible",
  passwordFile = process.env.PGCATALOG_PASSWORD_FILE ||
    "/data/probata/secrets/metabase/pg-readonly",
} = {}) {
  const password = loadPassword(passwordFile);
  if (!password) {
    return {
      configured: false,
      async lookup() {
        return {
          configured: false,
          error: "Catalog credential is not configured on this host.",
          b2_key: null,
          object: null,
          occurrences: [],
        };
      },
    };
  }

  const pool = new pg.Pool({
    host,
    port,
    user,
    password,
    database,
    max: 3,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 5000,
  });
  // An idle client in the pool can emit a background error (e.g. the peer closed
  // the connection); without a handler that crashes the whole Node process.
  pool.on("error", () => {});

  async function lookup(openlistPath) {
    const b2Key = b2KeyFromOpenListPath(openlistPath);
    if (!b2Key) {
      return {
        configured: true,
        error: "path must be an OpenList path under /b2/salem-data/",
        b2_key: null,
        object: null,
        occurrences: [],
      };
    }
    let client;
    try {
      client = await pool.connect();
      const objectResult = await client.query(
        `SELECT md5, size, origin FROM raw_duck.b2_content WHERE b2_key = $1 LIMIT 1`,
        [b2Key],
      );
      const occurrenceResult = await client.query(
        `SELECT source, scope, path, size, modtime, md5, native_hash_kind,
                native_hash, disposition, matched_origin, recorded_at
           FROM raw_duck.source_occurrences
          WHERE b2_key = $1
          ORDER BY source, path
          LIMIT ${MAX_OCCURRENCES}`,
        [b2Key],
      );
      return {
        configured: true,
        b2_key: b2Key,
        object: objectResult.rows[0] || null,
        occurrences: occurrenceResult.rows,
        count: occurrenceResult.rows.length,
        truncated: occurrenceResult.rows.length === MAX_OCCURRENCES,
      };
    } catch {
      return {
        configured: true,
        error: "Catalog query failed.",
        b2_key: b2Key,
        object: null,
        occurrences: [],
      };
    } finally {
      client?.release();
    }
  }

  return { configured: true, lookup };
}
