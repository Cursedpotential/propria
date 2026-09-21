// Byline: Claude Code · Fable 5.1 · 2026-09-21
// Client for GET /api/proffer/decoded/* — SBV's decoded output for one source,
// readable before any ingest run exists (see app/service/proffer_decoded.py).

const API_BASE = import.meta.env.VITE_API_URL || "";
const SHA256_HEX = /^[0-9a-f]{64}$/;

export interface DecodedThread {
  file: string;
  thread: string;
  chunk: number;
  participants: string[];
  records: number;
  first_occurred_at: string | null;
  last_occurred_at: string | null;
}

export interface DecodedManifest {
  source_ref: string;
  decoded_at: string | null;
  decoder: string | null;
  records: number;
  rejected: number;
  media_objects: number;
  media_bytes: number;
  threads: DecodedThread[];
}

export interface DecodedAttachment {
  ordinal: number;
  name: string | null;
  media_type: string | null;
  sha256: string | null;
  byte_length: number | null;
}

export interface DecodedMessage {
  ordinal: number;
  kind: string;
  status: string;
  occurred_at: string | null;
  sender: string | null;
  participants: string[];
  body: string;
  attachments: DecodedAttachment[];
  missing_attachments: number;
}

export interface DecodedThreadPage {
  file: string;
  offset: number;
  total_records: number;
  messages: DecodedMessage[];
  next_offset: number | null;
}

export class DecodedSourceError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}

async function getJson<T>(path: string, query: Record<string, string>, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${API_BASE}${path}?${new URLSearchParams(query).toString()}`, { signal });
  if (!response.ok) {
    let detail = response.statusText;
    try {
      const body = (await response.json()) as { detail?: unknown };
      if (typeof body.detail === "string") detail = body.detail;
    } catch {
      // keep the status text
    }
    throw new DecodedSourceError(detail, response.status);
  }
  return (await response.json()) as T;
}

export function getDecodedManifest(sourceRef: string, signal?: AbortSignal) {
  return getJson<DecodedManifest>("/api/proffer/decoded/manifest", { source_ref: sourceRef }, signal);
}

export function getDecodedThreadPage(sourceRef: string, file: string, offset: number, signal?: AbortSignal) {
  return getJson<DecodedThreadPage>(
    "/api/proffer/decoded/thread",
    { source_ref: sourceRef, file, offset: String(offset), limit: "200" },
    signal,
  );
}

/** Null for anything that is not a plain sha256, so a media element never gets an unvalidated path. */
export function getDecodedMediaUrl(sourceRef: string, sha256: string | null): string | null {
  if (!sha256 || !SHA256_HEX.test(sha256)) return null;
  return `${API_BASE}/api/proffer/decoded/media/${sha256}?${new URLSearchParams({ source_ref: sourceRef }).toString()}`;
}
