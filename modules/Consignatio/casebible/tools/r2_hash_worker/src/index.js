// Byline: Claude Code · Opus 5.5 · 2026-10-02
// casebible-r2-hasher: stream R2 objects through SHA-1 and SHA-256 inside Cloudflare and return the digests.
//
// Owner 2026-10-02 19:09 EDT: prove nothing is lost before R2 is released, comparing SHA-1 to SHA-1 against B2,
// on Cloudflare's compute. Successor of `casebible-sha256-backfill` (2026-08, SHA-256 of casebible-sorted only,
// self-listing with an R2 ledger). This one is driven by the catalog: the VPS driver
// (`r2_hash_driver_20261002.py`) posts batches of keys taken from the fresh R2 listing and stores the answers in
// the catalog, so nothing is written to R2 (which is being retired).
//
// Reads only: R2 `get` through the bucket bindings. Nothing is written, copied or deleted. Every request must carry
// `Authorization: Bearer <HASHER_TOKEN>` (a Worker secret).
//
// POST /hash  {"bucket": "<name>", "keys": ["k1", ...]}
//   -> {"results": [{"key", "size", "etag", "version", "uploaded", "sha1", "sha256", "error"}]}
// POST /list  {"bucket": "<name>", "cursor": null} -> {"objects": [...], "truncated", "cursor"}
// GET  /buckets -> the bucket names this Worker can read.

const BINDINGS = {
  "casebible-hash-ledger": "B_HASH_LEDGER",
  "casebible-lakehouse": "B_LAKEHOUSE",
  "casebible-quarantine": "B_QUARANTINE",
  "casebible-raw": "B_RAW",
  "casebible-sorted": "B_SORTED",
  "milvus-memsearch": "B_MILVUS_MEMSEARCH",
  "nexus": "B_NEXUS",
  "photos": "B_PHOTOS",
  "r2-explorer-bucket": "B_R2_EXPLORER",
};
const MAX_KEYS = 200;

function hex(buffer) {
  return [...new Uint8Array(buffer)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function hashObject(bucket, key) {
  const object = await bucket.get(key);
  if (!object || !object.body) return { key, error: "not found" };
  const sha1 = new crypto.DigestStream("SHA-1");
  const sha256 = new crypto.DigestStream("SHA-256");
  const [a, b] = object.body.tee();
  await Promise.all([a.pipeTo(sha1), b.pipeTo(sha256)]);
  return {
    key,
    size: object.size,
    etag: object.etag.replaceAll('"', ""),
    version: object.version,
    uploaded: object.uploaded.toISOString(),
    sha1: hex(await sha1.digest),
    sha256: hex(await sha256.digest),
  };
}

export default {
  async fetch(request, env) {
    const token = request.headers.get("authorization")?.slice(7);
    if (!env.HASHER_TOKEN || token !== env.HASHER_TOKEN) return new Response("unauthorized", { status: 401 });
    const url = new URL(request.url);

    if (url.pathname === "/buckets" && request.method === "GET") {
      return Response.json(Object.keys(BINDINGS).filter((name) => env[BINDINGS[name]]));
    }
    // POST /list {"bucket", "cursor"?} -> one page (up to 1000) of objects with size, ETag and stored checksums.
    // rclone's listing of these buckets needed a HEAD per multipart object (hours per bucket); the binding's list
    // returns the same metadata 1000 at a time (2026-10-02).
    if (url.pathname === "/list" && request.method === "POST") {
      const { bucket: name, cursor } = await request.json();
      const bucket = env[BINDINGS[name]];
      if (!bucket) return Response.json({ error: `unknown bucket ${name}` }, { status: 400 });
      const page = await bucket.list({ limit: 1000, cursor: cursor || undefined });
      return Response.json({
        truncated: page.truncated,
        cursor: page.truncated ? page.cursor : null,
        objects: page.objects.map((o) => ({
          key: o.key,
          size: o.size,
          etag: o.etag.replaceAll('"', ""),
          uploaded: o.uploaded.toISOString(),
          md5: o.checksums?.md5 ? hex(o.checksums.md5) : null,
          sha1: o.checksums?.sha1 ? hex(o.checksums.sha1) : null,
          sha256: o.checksums?.sha256 ? hex(o.checksums.sha256) : null,
        })),
      });
    }
    if (url.pathname !== "/hash" || request.method !== "POST") return new Response("not found", { status: 404 });

    const { bucket: name, keys } = await request.json();
    const bucket = env[BINDINGS[name]];
    if (!bucket) return Response.json({ error: `unknown bucket ${name}` }, { status: 400 });
    if (!Array.isArray(keys) || keys.length === 0 || keys.length > MAX_KEYS) {
      return Response.json({ error: `keys must be 1..${MAX_KEYS}` }, { status: 400 });
    }

    const concurrency = Math.max(1, Math.min(Number(env.CONCURRENCY || 6), 16));
    const results = new Array(keys.length);
    let next = 0;
    const lane = async () => {
      for (;;) {
        const i = next++;
        if (i >= keys.length) return;
        try {
          results[i] = await hashObject(bucket, keys[i]);
        } catch (error) {
          results[i] = { key: keys[i], error: String(error?.message || error) };
        }
      }
    };
    await Promise.all(Array.from({ length: Math.min(concurrency, keys.length) }, lane));
    return Response.json({ results });
  },
};
