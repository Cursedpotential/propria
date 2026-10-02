// Byline: Claude Code · Sonnet · 2026-10-02
// Client for the mobile Imported view (/api/imported/*). Read-only: every call here is a GET.
// The Review decision on a preview stays on the existing Review client (decideProffer).
import { ApiError } from "@/lib/api-client";

const API_BASE = import.meta.env.VITE_API_URL || "";

export type SourceStatus = "committed" | "awaiting_review" | "parked" | "failed" | "running" | "not_finished";

export interface ImportedSource {
  id: string;
  file_name: string;
  casevault_key: string;
  format: "SMS" | "Calls" | "Facebook" | "Other";
  device: string | null;
  owner: string | null;
  owner_name: string | null;
  files: number;
  raw: number;
  normalized: number;
  committed: number;
  messages: number;
  calls: number;
  first_at: string | null;
  last_at: string | null;
  imported_at: string | null;
  status: SourceStatus;
  status_counts: Partial<Record<SourceStatus, number>>;
}

export interface SourcesPage {
  items: ImportedSource[];
  total: number;
  next_offset: number | null;
  totals: { files: number; raw: number; normalized: number; committed: number };
  run_state_available: boolean;
}

export interface Participant {
  id: string | null;
  label: string;
  mine: boolean;
  person: string | null;
}

export interface ImportedThread {
  id: string;
  title: string;
  participants: Participant[];
  files: number;
  records: number;
  messages: number;
  calls: number;
  first_at: string | null;
  last_at: string | null;
  party: "first_party" | "third_party" | "mixed" | "unclassified";
  first_party_messages: number;
  third_party_messages: number;
}

export interface ThreadsPage {
  source: ImportedSource;
  items: ImportedThread[];
  next_offset: number | null;
}

export interface ImportedMessage {
  id: string;
  at: string | null;
  body: string;
  sender: Participant;
  outgoing: boolean;
  party: "first_party" | "third_party" | null;
  attachments: number;
  certainty: string | null;
}

export interface MessagesPage {
  thread_id: string;
  source: Pick<ImportedSource, "id" | "file_name" | "format" | "device" | "owner" | "owner_name">;
  conversation: string;
  items: ImportedMessage[];
  older_cursor: string | null;
  newer_cursor: string | null;
  focus: string | null;
}

export interface ImportedCall {
  id: string;
  at: string | null;
  kind: string;
  direction: string | null;
  missed: boolean;
  duration_s: number | null;
  with: Participant;
  device: string | null;
}

export interface CallsPage {
  items: ImportedCall[];
  next_cursor: string | null;
  summary: { total: number; missed: number; incoming: number; outgoing: number; first_at: string | null; last_at: string | null } | null;
  read_from: string;
}

export interface SearchHit {
  id: string;
  body: string;
  sender: string;
  at: string | null;
  score: number;
  thread_id: string | null;
  source: string | null;
  format: string | null;
  device: string | null;
}

export interface SearchPage {
  query: string;
  mode: "keyword" | "hybrid";
  items: SearchHit[];
  next_offset: number | null;
  note: string;
}

export interface ImportedSummary {
  matter_id: string;
  totals: { sources: number; files: number; raw: number; normalized: number; committed: number; messages: number; calls: number };
  files_by_status: Partial<Record<SourceStatus, number>>;
  run_state_available: boolean;
  generated_at: string;
}

export interface ReviewQueueItem {
  preview_handle: string;
  title: string;
  file_name: string;
  format: string;
  device: string | null;
  owner: string | null;
  records: number;
  waiting_since: string | null;
}

async function getJson<T>(path: string, params: Record<string, string | number | undefined | null> = {}, signal?: AbortSignal): Promise<T> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") query.set(key, String(value));
  }
  const suffix = query.size ? `?${query.toString()}` : "";
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}${suffix}`, { signal });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw error;
    throw new ApiError("No connection. Check your signal and try again.", 0);
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new ApiError(typeof body.detail === "string" ? body.detail : `Request failed (${response.status})`, response.status, body);
  }
  return response.json() as Promise<T>;
}

export const importedApi = {
  summary: (signal?: AbortSignal) => getJson<ImportedSummary>("/api/imported/summary", {}, signal),
  sources: (offset: number, format?: string, signal?: AbortSignal) =>
    getJson<SourcesPage>("/api/imported/sources", { limit: 20, offset, format }, signal),
  threads: (sourceId: string, offset: number, signal?: AbortSignal) =>
    getJson<ThreadsPage>(`/api/imported/sources/${encodeURIComponent(sourceId)}/threads`, { limit: 25, offset }, signal),
  messages: (threadId: string, params: { cursor?: string | null; around?: string | null }, signal?: AbortSignal) =>
    getJson<MessagesPage>(`/api/imported/threads/${encodeURIComponent(threadId)}/messages`, {
      limit: 40, direction: "before", cursor: params.cursor, around: params.around,
    }, signal),
  calls: (cursor: string | null, signal?: AbortSignal) => getJson<CallsPage>("/api/imported/calls", { limit: 40, cursor }, signal),
  search: (q: string, offset: number, signal?: AbortSignal) => getJson<SearchPage>("/api/imported/search", { q, limit: 20, offset }, signal),
  reviewQueue: (signal?: AbortSignal) => getJson<{ items: ReviewQueueItem[]; total: number }>("/api/imported/review-queue", {}, signal),
};
