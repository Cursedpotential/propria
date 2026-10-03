// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// casebible-zip-lister: list the members of ZIP archives with ranged reads only (end-of-central-directory, ZIP64 record,
// central directory). No archive is downloaded and no member is decompressed.
//
// Owner 2026-10-02 20:50 EDT approved bulk reads on Cloudflare's side. First target: the AI-chat ZIPs in B2 `salem-data`.
// A Go Temporal Activity posts batches of archives taken from the catalog and writes the answers to
// raw_duck.cf_zip_archives_20261002 / cf_zip_members_20261002; nothing is stored here.
//
// Reads only. Request:  POST /list  Authorization: Bearer <ZIPLISTER_TOKEN>
//   {provider:"b2"|"r2", bucket, keys:[string | {key, size?}] (1..50), max_members?: 1..500000}
// Response: NDJSON, one object per line, always ending with {type:"done"} (a body without it is a cut call: retry it):
//   {type:"members", key, rows:[{index, name, comp_size, size, crc32, method, flags, encrypted, is_dir, local_header_offset, disk_start, mtime}]}
//   {type:"archive", key, size, entries_declared, entries_listed, cd_offset, cd_size, zip64, comment_len, prefix_bytes, truncated, requests, bytes_read}
//   {type:"error",   key, error}
//   {type:"done",    archives, errors}
import { authorized, json, mapLimit, ndjsonResponse, normalizeItems } from "../../shared/lib.js";
import { openStore } from "../../shared/store.js";
import { listZip } from "./zipcd.js";

const MAX_KEYS = 50;
const ROWS_PER_LINE = 1000;

/**
 * List one archive, emitting its member rows in chunks and then its summary line (or an error line).
 *
 * Returns true when the archive listed cleanly.
 */
async function listOne(store, item, emit, maxMembers) {
  try {
    const size = item.size ?? (await store.head(item.key))?.size;
    if (size == null) throw new Error("not found");
    const summary = await listZip(
      store,
      item.key,
      size,
      async (members) => {
        for (let i = 0; i < members.length; i += ROWS_PER_LINE) await emit({ type: "members", key: item.key, rows: members.slice(i, i + ROWS_PER_LINE) });
      },
      { maxMembers },
    );
    await emit({ type: "archive", key: item.key, size, ...summary });
    return true;
  } catch (error) {
    await emit({ type: "error", key: item.key, error: String(error?.message || error) });
    return false;
  }
}

export default {
  /** HTTP entry point: POST /list with a batch of archive keys. */
  async fetch(request, env, ctx) {
    if (!authorized(request, env.ZIPLISTER_TOKEN)) return new Response("unauthorized", { status: 401 });
    const url = new URL(request.url);
    if (url.pathname !== "/list" || request.method !== "POST") return new Response("not found", { status: 404 });
    let body;
    let store;
    try {
      body = await request.json();
      if (!Array.isArray(body.keys) || body.keys.length === 0 || body.keys.length > MAX_KEYS) return json({ error: `keys must be 1..${MAX_KEYS}` }, 400);
      store = openStore(env, body.provider, body.bucket);
    } catch (error) {
      const message = String(error?.message || error);
      return json({ error: message }, 400);
    }
    const items = normalizeItems(body.keys);
    const maxMembers = Math.max(1, Math.min(Number(body.max_members || 200000), 500000));
    return ndjsonResponse(ctx, async (emit) => {
      const outcomes = await mapLimit(items, Math.min(Number(env.CONCURRENCY || 4), 6), (item) => listOne(store, item, emit, maxMembers));
      const errors = outcomes.filter((ok) => ok !== true).length;
      await emit({ type: "done", archives: items.length, errors });
    });
  },
};
