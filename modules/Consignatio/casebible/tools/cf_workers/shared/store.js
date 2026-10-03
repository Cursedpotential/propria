// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Object-store access for the bulk-read Workers: Backblaze B2 over its native API, and Cloudflare R2 over bindings.
// Everything here only reads; nothing is written, copied or deleted.

/** R2 bucket name to Worker binding name (same names as the deployed casebible-r2-hasher). */
export const R2_BINDINGS = {
  "casebible-hash-ledger": "B_HASH_LEDGER",
  "casebible-lakehouse": "B_LAKEHOUSE",
  "casebible-quarantine": "B_QUARANTINE",
  "casebible-raw": "B_RAW",
  "casebible-sorted": "B_SORTED",
  "milvus-memsearch": "B_MILVUS_MEMSEARCH",
  nexus: "B_NEXUS",
  photos: "B_PHOTOS",
  "r2-explorer-bucket": "B_R2_EXPLORER",
};

let b2Session = null; // {downloadUrl, token, at}: lives as long as the isolate; B2 tokens last 24 h

async function b2Authorize(env, force) {
  if (!force && b2Session && Date.now() - b2Session.at < 6 * 3600 * 1000) return b2Session;
  if (!env.B2_KEY_ID || !env.B2_APP_KEY) throw new Error("B2_KEY_ID / B2_APP_KEY secrets are not set");
  const basic = btoa(`${env.B2_KEY_ID}:${env.B2_APP_KEY}`);
  const authUrl = env.B2_AUTH_URL || "https://api.backblazeb2.com/b2api/v3/b2_authorize_account"; // B2_AUTH_URL only for a local stand-in server
  const res = await fetch(authUrl, { headers: { authorization: `Basic ${basic}` } });
  if (!res.ok) throw new Error(`b2_authorize_account failed: HTTP ${res.status}`);
  const body = await res.json();
  const downloadUrl = body.apiInfo?.storageApi?.downloadUrl || body.downloadUrl;
  if (!downloadUrl || !body.authorizationToken) throw new Error("b2_authorize_account returned no download URL or token");
  b2Session = { downloadUrl, token: body.authorizationToken, at: Date.now() };
  return b2Session;
}

/** Percent-encode an object key for a B2 download URL, keeping the slashes. */
export function encodeKey(key) {
  return key.split("/").map(encodeURIComponent).join("/");
}

/** B2 read-only store: head, ranged read and full stream of one bucket's objects. */
class B2Store {
  constructor(env, bucket) {
    this.env = env;
    this.bucket = bucket;
    this.provider = "b2";
  }

  async #get(key, method, range) {
    for (let attempt = 0; attempt < 2; attempt++) {
      const session = await b2Authorize(this.env, attempt > 0);
      const headers = { authorization: session.token };
      if (range) headers.range = range;
      const res = await fetch(`${session.downloadUrl}/file/${this.bucket}/${encodeKey(key)}`, { method, headers });
      if (res.status === 401 && attempt === 0) {
        await res.body?.cancel();
        continue;
      }
      return res;
    }
    throw new Error("B2 authorization rejected twice");
  }

  /** Size of an object, or null when it does not exist. */
  async head(key) {
    const res = await this.#get(key, "HEAD");
    if (res.status === 404) return null;
    if (!res.ok) throw new Error(`B2 HEAD HTTP ${res.status}`);
    return { size: Number(res.headers.get("content-length")), sha1: res.headers.get("x-bz-content-sha1"), fileId: res.headers.get("x-bz-file-id") };
  }

  /** `length` bytes from `offset` (fewer at the end of the object). Never reads more than asked, even if B2 ignores the range. */
  async range(key, offset, length) {
    const res = await this.#get(key, "GET", `bytes=${offset}-${offset + length - 1}`);
    if (res.status === 416) return new Uint8Array(0);
    if (res.status === 404) throw new Error("not found");
    if (!res.ok) throw new Error(`B2 GET HTTP ${res.status}`);
    if (res.status === 200) {
      // B2 ignored the range: the stream starts at byte 0. Fine for a head read, an error for anything else.
      if (offset !== 0) {
        await res.body?.cancel();
        throw new Error("B2 ignored the Range header");
      }
      return readAtMost(res.body, length);
    }
    return new Uint8Array(await res.arrayBuffer());
  }

  /** Whole object as a stream: `{body, size, sha1, fileId}`, or null when it does not exist. */
  async stream(key) {
    const res = await this.#get(key, "GET");
    if (res.status === 404) return null;
    if (!res.ok) throw new Error(`B2 GET HTTP ${res.status}`);
    return { body: res.body, size: Number(res.headers.get("content-length")), sha1: res.headers.get("x-bz-content-sha1"), fileId: res.headers.get("x-bz-file-id") };
  }
}

async function readAtMost(body, limit) {
  const reader = body.getReader();
  const parts = [];
  let total = 0;
  while (total < limit) {
    const { value, done } = await reader.read();
    if (done) break;
    parts.push(value);
    total += value.byteLength;
  }
  await reader.cancel().catch(() => {});
  const out = new Uint8Array(Math.min(total, limit));
  let at = 0;
  for (const part of parts) {
    const take = Math.min(part.byteLength, out.length - at);
    out.set(part.subarray(0, take), at);
    at += take;
  }
  return out;
}

/** R2 read-only store over a bucket binding. */
class R2Store {
  constructor(bucket, name) {
    this.bucket = name;
    this.r2 = bucket;
    this.provider = "r2";
  }

  async head(key) {
    const o = await this.r2.head(key);
    return o ? { size: o.size } : null;
  }

  async range(key, offset, length) {
    if (length <= 0) return new Uint8Array(0);
    const o = await this.r2.get(key, { range: { offset, length } });
    if (!o) throw new Error("not found");
    return new Uint8Array(await o.arrayBuffer());
  }

  async stream(key) {
    const o = await this.r2.get(key);
    return o && o.body ? { body: o.body, size: o.size } : null;
  }
}

/**
 * Open a read-only store for `provider` ("b2" or "r2") and `bucket`.
 *
 * B2 is limited to the bucket named by the `B2_BUCKET` variable; R2 to the buckets bound in wrangler.jsonc.
 * Throws an Error whose message starts with "bad request:" for anything else.
 */
export function openStore(env, provider, bucket) {
  if (provider === "b2") {
    if (!env.B2_BUCKET || bucket !== env.B2_BUCKET) throw new Error(`bad request: B2 bucket ${bucket} is not enabled on this Worker`);
    return new B2Store(env, bucket);
  }
  if (provider === "r2") {
    const binding = env[R2_BINDINGS[bucket]];
    if (!binding) throw new Error(`bad request: R2 bucket ${bucket} is not bound to this Worker`);
    return new R2Store(binding, bucket);
  }
  throw new Error(`bad request: provider must be "b2" or "r2", got ${provider}`);
}
