// Byline: Codex · GPT-6 · 2026-10-05. Durable export recovery inside the existing console process.
import type { StoreOk } from "./store.js";
import { loadLibrarySyncToken } from "./library-sync-http.js";

const OPERATION = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/;
type Sealer = (operationId: string) => Promise<{ status: string }>;

/** Start or rejoin a durable immutable export without including record bodies in Temporal requests.
 * Inputs: connected store and captured operation UUID. Outputs: queued or visible retryable dispatch state.
 * Effects: bounded authenticated starter request and dispatch metadata only; payload, baseline and intent remain untouched.
 * Choose for timer recovery after a saved edit, including a lost starter response.
 */
export async function queueLibrarySync(store: Pick<StoreOk, "db">, operationId: string): Promise<Record<string, unknown>> {
  if (!OPERATION.test(operationId)) throw new Error("Invalid durable sync operation identity");
  let code = "SYNC_CONFIGURATION";
  let outcome: Record<string, unknown>;
  try {
    const token = loadLibrarySyncToken();
    const endpoint = new URL(process.env.TOOLKIT_LIBRARY_SYNC_START_URL ?? "");
    if (!token || endpoint.username || endpoint.password || endpoint.search || endpoint.hash || endpoint.pathname !== "/toolkit/library/sync"
      || (endpoint.protocol !== "https:" && !(endpoint.protocol === "http:" && endpoint.hostname === "100.91.190.107"))) throw new Error("configuration");
    code = "SYNC_NETWORK";
    const response = await fetch(endpoint, { method: "POST", redirect: "error", signal: AbortSignal.timeout(15000),
      headers: { Authorization: "Bearer " + token, "Content-Type": "application/json" }, body: JSON.stringify({ operation_id: operationId }) });
    if (!response.ok) { code = "SYNC_HTTP_" + response.status; await response.body?.cancel(); throw new Error("upstream"); }
    code = "SYNC_RESPONSE";
    if (!response.body) throw new Error("response");
    const reader = response.body.getReader(); const chunks: Uint8Array[] = []; let size = 0;
    try {
      for (;;) { const next = await reader.read(); if (next.done) break; size += next.value.length;
        if (size > 4096) { await reader.cancel(); throw new Error("response"); } chunks.push(next.value); }
    } finally { reader.releaseLock(); }
    const value = JSON.parse(Buffer.concat(chunks, size).toString("utf8")) as Record<string, unknown>;
    if (value.workflow_id !== "library-sync-write-" + operationId || typeof value.run_id !== "string" || !OPERATION.test(value.run_id)) throw new Error("response");
    outcome = { state: "queued", workflow_id: value.workflow_id, run_id: value.run_id, retryable: false };
  } catch { outcome = { state: "queue_failed", error_code: code, retryable: true }; }
  await store.db.query("UPDATE type::record('library_sync_outbox', $op) SET dispatch = $dispatch, dispatch_at = time::now();", { op: operationId, dispatch: outcome });
  return outcome;
}

/** Seal and dispatch a bounded page of durable operations; retain encoding and queue failures for later recovery.
 * Inputs: store and immutable-snapshot sealer. Outputs: attempted/queued counts. Effects: snapshot sealing and starter dispatch.
 * Choose after startup and periodically in the existing console; never encode from a later live record or claim a database lease here.
 */
export async function drainLibrarySyncOutbox(store: Pick<StoreOk, "db">, seal: Sealer): Promise<{ attempted: number; queued: number }> {
  const response = await store.db.query<unknown[]>("SELECT operation_id, status FROM library_sync_outbox WHERE status IN ['pending_encoding', 'pending', 'retry_wait', 'write_unknown'] ORDER BY created_at LIMIT 20;");
  const rows = response.at(-1);
  if (!Array.isArray(rows)) throw new Error("Malformed durable sync outbox page");
  let attempted = 0; let queued = 0;
  for (const row of rows) {
    if (!row || typeof row !== "object") continue;
    const value = row as { operation_id?: unknown; status?: unknown };
    if (typeof value.operation_id !== "string" || !OPERATION.test(value.operation_id)) throw new Error("Malformed durable sync operation");
    if (value.status === "pending_encoding" && (await seal(value.operation_id)).status !== "pending") continue;
    attempted++;
    if ((await queueLibrarySync(store, value.operation_id)).state === "queued") queued++;
  }
  return { attempted, queued };
}

/** Recover durable exports periodically without spawning processes or keeping a completed console alive.
 * Inputs: connected-store/sealer factory. Outputs: cancellation closure. Effects: one serialized unreferenced timer with visible safe error logs.
 * Choose once during HTTP server composition when dedicated sync configuration is enabled.
 */
export function startLibrarySyncOutboxRecovery(resolve: () => Promise<{ store: Pick<StoreOk, "db">; seal: Sealer }>): () => void {
  let busy = false; let closed = false;
  const tick = async () => {
    if (busy || closed) return;
    busy = true;
    try { const { store, seal } = await resolve(); await drainLibrarySyncOutbox(store, seal); }
    catch { console.error("family-court library export recovery failed; durable operations retained for retry"); }
    finally { busy = false; }
  };
  const timer = setInterval(() => { void tick(); }, 30000); timer.unref(); void tick();
  return () => { closed = true; clearInterval(timer); };
}
