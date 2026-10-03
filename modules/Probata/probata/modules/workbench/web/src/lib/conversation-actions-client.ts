// Byline: Claude Code · Sonnet 5.5 · 2026-10-02
// Client for Extract and Send to Surreal on the Imported view's conversations (/api/extractors,
// /api/imported/threads/{extract,send-to-surreal,<id>/extractions}, /api/imported/workflows/<id>).
// The two starts send an Idempotency-Key minted once per click, so a retried request joins the running workflow.
// Reads are GETs; the extraction view is read-only on every device.
import { ApiError } from "@/lib/api-client";
import { newIdempotencyKey } from "@/lib/entity-extraction-client";

const API_BASE = import.meta.env.VITE_API_URL || "";

export { newIdempotencyKey };

/** One selectable extractor, as the engine's registry lists it. Every one stays selectable. */
export interface ExtractorInfo {
  id: string;
  label: string;
  description: string;
  kind: "internal" | "external";
  default: boolean;
  /** Its output is kept for side-by-side viewing and stays out of the Review proposals until the owner picks it. */
  compare_only: boolean;
  run_extractors: string[];
}

export interface ExtractorListing {
  extractors: ExtractorInfo[];
  default: string;
}

export interface WorkflowStarted {
  workflow_id: string;
  run_id: string;
  kind: "extraction" | "send_to_surreal";
  conversations: number;
}

export interface ActionStep {
  step: string;
  status: "pending" | "running" | "completed" | "skipped" | "failed";
  detail?: string;
  counts?: Record<string, number>;
  flags?: { code: string; detail: string }[];
  /** The conversation this line is about, and the extractor (or "surreal"). */
  conversation?: string;
  extractor?: string;
}

export interface SendReceipt {
  conversation: { export_key: string; conv: string };
  thread_id: string;
  plan: { messages: number; source_files: number; participants: number };
  written: { threads: number; messages: number; entities: number; events: number; runs: number };
  verified: { messages: number; entities: number; events: number; match: boolean };
  sent_at: string;
}

export interface ActionStatus {
  workflow_id: string;
  kind: "extraction" | "send_to_surreal";
  outcome: string;
  steps: ActionStep[];
  receipts: SendReceipt[];
}

export interface ExtractedEntity {
  id: string;
  run_id: string;
  name: string;
  type: string;
  confidence: number;
  review_state: string;
  aliases: string[];
  mention_count: number;
  mentions: { record_id: string | null; surface: string | null; snippet: string | null }[];
}

export interface ExtractedEvent {
  id: string;
  run_id: string;
  title: string;
  type: string;
  occurred_at: string | null;
  precision: string | null;
  confidence: number;
  review_state: string;
  record_ids: string[];
  description: string;
  when_stated: string | null;
}

export interface ExtractionRun {
  id: string;
  extractor: string;
  version: string;
  model_id: string | null;
  status: string;
  error: string | null;
  started_at: string | null;
  finished_at: string | null;
  stats: Record<string, unknown>;
}

/** What one extractor found in one conversation. */
export interface ExtractionGroup {
  id: string;
  label: string;
  kind: string | null;
  compare_only: boolean;
  status: "running" | "completed" | "failed" | "completed_with_failures";
  runs: ExtractionRun[];
  entities: ExtractedEntity[];
  events: ExtractedEvent[];
  counts: { entities: number; events: number; runs: number };
}

export interface ThreadExtractions {
  thread_id: string;
  extractors: ExtractionGroup[];
  truncated: boolean;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, init);
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

function post<T>(path: string, body: unknown, key: string): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify(body) });
}

export const conversationApi = {
  extractors: (signal?: AbortSignal) => request<ExtractorListing>("/api/extractors", { signal }),
  startExtraction: (threadIds: string[], extractors: string[], key: string) =>
    post<WorkflowStarted>("/api/imported/threads/extract", { thread_ids: threadIds, extractors }, key),
  startSend: (threadIds: string[], includeExtractions: boolean, key: string) =>
    post<WorkflowStarted>("/api/imported/threads/send-to-surreal", { thread_ids: threadIds, include_extractions: includeExtractions }, key),
  status: async (workflowId: string, signal?: AbortSignal): Promise<ActionStatus> => {
    const status = await request<ActionStatus>(`/api/imported/workflows/${encodeURIComponent(workflowId)}`, { signal });
    // The engine omits empty lists; the view maps over them.
    return { ...status, steps: (status.steps ?? []).map((step) => ({ ...step, flags: step.flags ?? [] })), receipts: status.receipts ?? [] };
  },
  extractions: (threadId: string, signal?: AbortSignal) =>
    request<ThreadExtractions>(`/api/imported/threads/${encodeURIComponent(threadId)}/extractions`, { signal }),
};
