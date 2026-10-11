/** Read small Go-produced SMS preview pages; never fetch or parse original XML.
 * Inputs: exact listing metadata/cursor and cancellation signal. Output: one page.
 * Effects: bounded BFF requests/polling only; choose before processing a backup.
 */
import type { ProfferSourcePreview } from "@/lib/shared/types";

export type PreviewSelection = Pick<ProfferSourcePreview, "root_id" | "source_ref" | "key" | "byte_length" | "etag">;
export interface RawPreviewMessage {
  id: string; kind: string; date_ms: number | null; direction: string;
  participants: string[]; body: string; body_truncated: boolean; metadata_truncated?: boolean;
  attachments: { content_type: string; name: string; encoded_bytes: number }[];
}
export interface RawPreviewPage {
  page_id: string; status: "ready";
  source: { source_ref: string; etag: string; version_id?: string | null; byte_length: number };
  messages: RawPreviewMessage[]; next_cursor?: string | null;
  scanned_bytes: number; truncation_reason?: string | null; preview_only: true;
}
interface PendingPage { page_id: string; status: "pending"; poll_url: string }
const BASE = `${import.meta.env.VITE_API_URL || ""}/api/proffer/source-preview`;
const MAX_BYTES = 512 * 1024;
const HANDLE = /^[A-Za-z0-9_-]{1,128}$/;

async function responsePage(response: Response): Promise<RawPreviewPage | PendingPage> {
  if (!response.body) throw new Error("Message preview returned no page");
  const reader = response.body.getReader();
  const parts: Uint8Array[] = [];
  let size = 0;
  try {
    for (;;) {
      const item = await reader.read();
      if (item.done) break;
      size += item.value.byteLength;
      if (size > MAX_BYTES) throw new Error("Message preview exceeded its page limit");
      parts.push(item.value);
    }
  } catch (error) { await reader.cancel(); throw error; }
  finally { reader.releaseLock(); }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const part of parts) { bytes.set(part, offset); offset += part.byteLength; }
  const data = JSON.parse(new TextDecoder().decode(bytes));
  if (!response.ok) throw new Error(typeof data.detail === "string" ? data.detail : "Message preview is unavailable");
  if (!data || !HANDLE.test(data.page_id) || !["ready", "pending"].includes(data.status)) throw new Error("Invalid message preview page");
  if (data.status === "ready" && (!Array.isArray(data.messages) || data.messages.length > 50 || data.preview_only !== true)) throw new Error("Invalid message preview page");
  return data;
}

/** Fetch one page with a sixty-second deadline, abortable polling and source correlation.
 * Input: listing pins and opaque continuation. Output: at most fifty messages.
 * Effects: read-preview only; no Process, ELT, indexing or media download.
 */
export async function getSourceMessagePreview(selection: PreviewSelection, cursor: string | null, signal: AbortSignal): Promise<RawPreviewPage> {
  const bounded = AbortSignal.any([signal, AbortSignal.timeout(60_000)]);
  let response = await fetch(BASE, { method: "POST", headers: { "Content-Type": "application/json" }, signal: bounded,
    body: JSON.stringify({ root_id: selection.root_id, key: selection.key, source_ref: selection.source_ref,
      expected_byte_length: selection.byte_length, expected_etag: selection.etag, cursor, limit: 25 }) });
  let page = await responsePage(response);
  for (let attempt = 0; page.status === "pending" && attempt < 60; attempt++) {
    await new Promise<void>((resolve, reject) => {
      const abort = () => { clearTimeout(timer); reject(bounded.reason); };
      const timer = setTimeout(() => { bounded.removeEventListener("abort", abort); resolve(); }, 1000);
      bounded.addEventListener("abort", abort, { once: true });
      if (bounded.aborted) abort();
    });
    response = await fetch(`${BASE}/${encodeURIComponent(page.page_id)}`, { signal: bounded });
    page = await responsePage(response);
  }
  if (page.status !== "ready") throw new Error("Message preview is taking longer than expected. Try again.");
  if (page.source.source_ref !== selection.source_ref || page.source.etag.replace(/^"|"$/g, "") !== selection.etag.replace(/^"|"$/g, "")
      || page.source.byte_length !== selection.byte_length) throw new Error("The source changed. Reopen the file.");
  return page;
}
