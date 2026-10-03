// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// casebible-b2-hasher: stream B2 objects through SHA-1 and SHA-256 inside Cloudflare and return the digests.
//
// Owner 2026-10-02 20:50 EDT approved the B2 hash backfill. Same pattern and hash types as the deployed sibling
// casebible-r2-hasher (casebible/tools/r2_hash_worker/): POST a batch of keys, get digests back, nothing is stored here.
// First target: the 37 `salem-data` objects whose catalog row has no SHA-1 (338.7 GB, raw_duck.cf_b2_hash_todo_20261002);
// a Go Temporal Activity posts them and writes the answers to raw_duck.cf_b2_hashes_20261002.
//
// Reads only. Request:  POST /hash  Authorization: Bearer <B2HASHER_TOKEN>   {bucket, keys:[string] (1..4)}
// Response: NDJSON (a hash of a 78 GB object takes minutes, so the headers leave at once and a progress line follows every
// 15 s), always ending with {type:"done"}; a body without it is a cut call: retry it.
//   {type:"progress", key, bytes}
//   {type:"result",   key, size, sha1, sha256, bytes_hashed, b2_content_sha1, b2_file_id, b2_sha1_mismatch, ms}
//   {type:"error",    key, error}
//   {type:"done",     hashed, errors}
import { authorized, hex, json, mapLimit, ndjsonResponse } from "../../shared/lib.js";
import { openStore } from "../../shared/store.js";

const MAX_KEYS = 4;
const PROGRESS_MS = 15000;

/**
 * Hash one B2 object: stream it once through SHA-1 and SHA-256.
 *
 * Emits progress lines while it runs and one result or error line at the end. B2's own `x-bz-content-sha1` header is
 * returned next to the computed SHA-1 (it is "none" for objects B2 never hashed, which are exactly the ones in the backfill).
 */
async function hashOne(store, key, emit) {
  const started = Date.now();
  try {
    const obj = await store.stream(key);
    if (!obj) throw new Error("not found");
    let bytes = 0;
    const counter = new TransformStream({
      transform(chunk, controller) {
        bytes += chunk.byteLength;
        controller.enqueue(chunk);
      },
    });
    const sha1 = new crypto.DigestStream("SHA-1");
    const sha256 = new crypto.DigestStream("SHA-256");
    const [a, b] = obj.body.pipeThrough(counter).tee();
    const ticker = setInterval(() => emit({ type: "progress", key, bytes }).catch(() => {}), PROGRESS_MS);
    try {
      await Promise.all([a.pipeTo(sha1), b.pipeTo(sha256)]);
    } finally {
      clearInterval(ticker);
    }
    if (bytes !== obj.size) throw new Error(`short read: ${bytes} of ${obj.size} bytes`);
    const computed = hex(await sha1.digest);
    const claimed = obj.sha1 && /^[0-9a-f]{40}$/.test(obj.sha1) ? obj.sha1 : null;
    await emit({
      type: "result",
      key,
      size: obj.size,
      sha1: computed,
      sha256: hex(await sha256.digest),
      bytes_hashed: bytes,
      b2_content_sha1: obj.sha1 || null,
      b2_file_id: obj.fileId || null,
      b2_sha1_mismatch: claimed ? claimed !== computed : false,
      ms: Date.now() - started,
    });
    return true;
  } catch (error) {
    await emit({ type: "error", key, error: String(error?.message || error) });
    return false;
  }
}

export default {
  /** HTTP entry point: POST /hash with a batch of B2 object keys. */
  async fetch(request, env, ctx) {
    if (!authorized(request, env.B2HASHER_TOKEN)) return new Response("unauthorized", { status: 401 });
    const url = new URL(request.url);
    if (url.pathname !== "/hash" || request.method !== "POST") return new Response("not found", { status: 404 });
    let store;
    let keys;
    try {
      const body = await request.json();
      keys = body.keys;
      if (!Array.isArray(keys) || keys.length === 0 || keys.length > MAX_KEYS || keys.some((k) => typeof k !== "string")) {
        return json({ error: `keys must be 1..${MAX_KEYS} strings` }, 400);
      }
      store = openStore(env, "b2", body.bucket);
    } catch (error) {
      return json({ error: String(error?.message || error) }, 400);
    }
    return ndjsonResponse(ctx, async (emit) => {
      const outcomes = await mapLimit(keys, Math.min(Number(env.CONCURRENCY || 2), 4), (key) => hashOne(store, key, emit));
      const hashed = outcomes.filter((ok) => ok === true).length;
      await emit({ type: "done", hashed, errors: keys.length - hashed });
    });
  },
};
