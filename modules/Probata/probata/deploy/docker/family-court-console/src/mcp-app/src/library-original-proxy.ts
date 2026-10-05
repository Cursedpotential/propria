// Byline: Codex · GPT-6 · 2026-10-05. Human-authorized original downloads through the existing console.
import type { ServerResponse } from "node:http";
import { loadLibrarySyncToken } from "./library-sync-http.js";

const MAX_ORIGINAL = 20 * 1024 * 1024;
let active = 0;

/** Proxy one exact retained PDF after the caller's existing human authorization succeeds.
 * Inputs: binding/version query, response and configured private starter origin. Outputs: verified PDF or opaque error.
 * Effects: bounded private HTTP read with a server-held credential; never exposes B2 credentials or buffers above 20 MiB.
 * Choose inside the authenticated web handler; this function supplies no independent human authentication.
 */
export async function proxyLibraryOriginal(url: URL, res: ServerResponse): Promise<void> {
  const fail = (status: number, code: string) => {
    res.writeHead(status, { "Content-Type": "application/json", "Cache-Control": "private, no-store" });
    res.end(JSON.stringify({ error: code }));
  };
  const binding = url.searchParams.get("binding_id") ?? "";
  const version = url.searchParams.get("version_id") ?? "";
  if (url.searchParams.size !== 2 || !/^library_file:[a-f0-9]{64}$/.test(binding)
    || !version || version === "null" || Buffer.byteLength(version) > 2048 || /[\r\n\0]/.test(version)) {
    fail(400, "exact_original_reference_required"); return;
  }
  const token = loadLibrarySyncToken();
  if (!token || !process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL) { fail(503, "original_service_unavailable"); return; }
  if (active >= 2) { fail(429, "original_download_busy"); return; }
  active++;
  try {
    const endpoint = new URL(process.env.TOOLKIT_LIBRARY_SYNC_ORIGINAL_URL);
    if (endpoint.username || endpoint.password || endpoint.search || endpoint.hash || endpoint.pathname !== "/toolkit/library/files/"
      || (endpoint.protocol !== "https:" && !(endpoint.protocol === "http:" && endpoint.hostname === "100.91.190.107")))
      throw new Error("configuration");
    endpoint.pathname += encodeURIComponent(binding) + "/original";
    endpoint.searchParams.set("version_id", version);
    const response = await fetch(endpoint, { redirect: "error", signal: AbortSignal.timeout(95000), headers: { Authorization: "Bearer " + token } });
    if (!response.ok || !response.body || response.headers.get("content-type") !== "application/pdf") {
      await response.body?.cancel(); throw new Error("upstream");
    }
    const length = Number(response.headers.get("content-length"));
    if (!Number.isSafeInteger(length) || length <= 0 || length > MAX_ORIGINAL) { await response.body.cancel(); throw new Error("budget"); }
    const reader = response.body.getReader(); const chunks: Uint8Array[] = []; let size = 0;
    try {
      for (;;) {
        const next = await reader.read(); if (next.done) break;
        size += next.value.length;
        if (size > MAX_ORIGINAL || size > length) { await reader.cancel(); throw new Error("budget"); }
        chunks.push(next.value);
      }
    } finally { reader.releaseLock(); }
    const bytes = Buffer.concat(chunks, size);
    if (size !== length || bytes.subarray(0, 4).toString() !== "%PDF") throw new Error("integrity");
    res.writeHead(200, { "Content-Type": "application/pdf", "Content-Length": String(size),
      "Content-Disposition": "inline", "Cache-Control": "private, no-store", "X-Content-Type-Options": "nosniff" });
    res.end(bytes);
  } catch { if (!res.headersSent) fail(502, "original_unavailable_or_integrity_failed"); else res.end(); }
  finally { active--; }
}
