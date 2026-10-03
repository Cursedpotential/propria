// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// casebible-format-sniffer: range-read the first bytes of objects next to the data and return the detected format.
//
// Owner 2026-10-02 20:50 EDT approved bulk reads on Cloudflare's side. First target: the 30,048 .txt and 9,308 .html
// AI-chat candidates in B2 `salem-data` (raw_duck.ai_chat_probe_20260918). A Go Temporal Activity posts batches of keys
// taken from the catalog and writes the answers back to raw_duck.cf_format_sniff_20261002; nothing is stored here.
//
// Reads only. Request:  POST /sniff  Authorization: Bearer <SNIFFER_TOKEN>
//   {provider:"b2"|"r2", bucket, keys:[string | {key, size?}] (1..200), ruleset?:"proffer-v1"|"casebible-probe-v1", head_bytes?:8192..262144}
// Response: {ruleset, results:[{key, size, bytes_read, format, signature_kind, confidence, rule_source, error?}]}
import { authorized, json, mapLimit, normalizeItems } from "../../shared/lib.js";
import { openStore } from "../../shared/store.js";
import { RULESETS, sniff } from "./sniff.js";

const MAX_KEYS = 200;
const DEFAULT_HEAD = 64 * 1024;

/**
 * Sniff one object: ranged-read its head and detect the format.
 *
 * Returns a result row; a missing object or a read failure becomes the row's `error`, never an exception.
 */
async function sniffOne(store, item, ruleset, headBytes) {
  const row = { key: item.key, size: item.size, bytes_read: 0 };
  try {
    if (item.size === 0) return { ...row, ...sniff(new Uint8Array(0), { ruleset }) };
    const length = item.size == null ? headBytes : Math.min(headBytes, item.size);
    const head = await store.range(item.key, 0, length);
    row.bytes_read = head.length;
    const truncated = item.size == null ? head.length === headBytes : head.length < item.size;
    return { ...row, ...sniff(head, { ruleset, truncated }) };
  } catch (error) {
    return { ...row, error: String(error?.message || error) };
  }
}

export default {
  /** HTTP entry point: POST /sniff with a batch of keys. */
  async fetch(request, env) {
    if (!authorized(request, env.SNIFFER_TOKEN)) return new Response("unauthorized", { status: 401 });
    const url = new URL(request.url);
    if (url.pathname !== "/sniff" || request.method !== "POST") return new Response("not found", { status: 404 });
    let body;
    try {
      body = await request.json();
      const { provider, bucket, keys } = body;
      const ruleset = body.ruleset || "proffer-v1";
      if (!RULESETS.includes(ruleset)) return json({ error: `unknown ruleset ${ruleset}` }, 400);
      if (!Array.isArray(keys) || keys.length === 0 || keys.length > MAX_KEYS) return json({ error: `keys must be 1..${MAX_KEYS}` }, 400);
      const headBytes = Math.max(8192, Math.min(Number(body.head_bytes || DEFAULT_HEAD), 262144));
      const store = openStore(env, provider, bucket);
      const items = normalizeItems(keys);
      const concurrency = Math.max(1, Math.min(Number(env.CONCURRENCY || 6), 6));
      const results = await mapLimit(items, concurrency, (item) => sniffOne(store, item, ruleset, headBytes));
      return json({ ruleset, results });
    } catch (error) {
      const message = String(error?.message || error);
      return json({ error: message }, message.startsWith("bad request:") ? 400 : 502);
    }
  },
};
