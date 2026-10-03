// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Shared helpers of the three Case Bible bulk-read Workers (format sniffer, ZIP lister, B2 hasher).
// Same contract as the deployed sibling casebible-r2-hasher: every request carries
// `Authorization: Bearer <worker secret>`; request and response bodies are JSON (or NDJSON when a call can run long).

/** Compare two strings without leaking where they differ. */
function timingSafeEqual(a, b) {
  const x = new TextEncoder().encode(a);
  const y = new TextEncoder().encode(b);
  let diff = x.length ^ y.length;
  const n = Math.max(x.length, y.length);
  for (let i = 0; i < n; i++) diff |= (x[i] || 0) ^ (y[i] || 0);
  return diff === 0;
}

/**
 * Check the bearer token of a request against the Worker's secret.
 *
 * Returns true only when the secret is configured and the header matches it; an unset secret fails closed.
 */
export function authorized(request, secret) {
  if (!secret) return false;
  const header = request.headers.get("authorization") || "";
  if (!header.startsWith("Bearer ")) return false;
  return timingSafeEqual(header.slice(7), secret);
}

/** Build a JSON response. */
export function json(body, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
}

/** Lowercase hex of a byte buffer. */
export function hex(buffer) {
  return [...new Uint8Array(buffer)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Stream an NDJSON response whose producer may run for minutes.
 *
 * The response headers leave at once (so no proxy times out waiting for them); `producer(emit)` then writes one JSON
 * object per line with `await emit(obj)`. The producer must end with a `{type:"done"}` line: a body without it is a
 * truncated call and the caller must retry. An exception becomes a `{type:"fatal"}` line.
 */
export function ndjsonResponse(ctx, producer) {
  const { readable, writable } = new TransformStream();
  const writer = writable.getWriter();
  const encoder = new TextEncoder();
  const emit = (obj) => writer.write(encoder.encode(JSON.stringify(obj) + "\n"));
  ctx.waitUntil(
    (async () => {
      try {
        await producer(emit);
      } catch (error) {
        await emit({ type: "fatal", error: String(error?.message || error) }).catch(() => {});
      } finally {
        await writer.close().catch(() => {});
      }
    })(),
  );
  return new Response(readable, { headers: { "content-type": "application/x-ndjson" } });
}

/**
 * Run `fn(item, index)` over `items` with at most `concurrency` calls in flight (Workers allow 6 open connections).
 *
 * Each result lands at its item's index; a throwing call becomes `{error}` for that item so one bad key never fails a batch.
 */
export async function mapLimit(items, concurrency, fn) {
  const results = new Array(items.length);
  let next = 0;
  const lane = async () => {
    for (;;) {
      const i = next++;
      if (i >= items.length) return;
      try {
        results[i] = await fn(items[i], i);
      } catch (error) {
        results[i] = { error: String(error?.message || error) };
      }
    }
  };
  await Promise.all(Array.from({ length: Math.max(1, Math.min(concurrency, items.length)) }, lane));
  return results;
}

/** Normalise request keys: a string, or an object `{key, size?}`, into `{key, size}`. */
export function normalizeItems(keys) {
  return keys.map((k) => (typeof k === "string" ? { key: k, size: null } : { key: String(k.key), size: k.size == null ? null : Number(k.size) }));
}
