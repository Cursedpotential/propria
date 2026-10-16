// Byline: Codex · GPT-6 · 2026-10-11.
import { ApiError } from "@/lib/api-client";
import type { MatterMode } from "@/lib/shared/types";

export type DiscoveryBackend = "postgresql" | "surrealdb";
export interface DiscoveryScope { mode: MatterMode; matter_id: string; court_case_id: string }
export interface DiscoveryDataset {
  dataset_id: string; label: string; input_backend: DiscoveryBackend; selection_revision: string;
  generation_ids: string[]; scope: DiscoveryScope;
}
export interface DiscoveryFilters {
  perspective: "hindsight" | "as_lived"; cutoff?: string; start_at?: string; end_at?: string;
  participants?: string[]; platforms?: string[];
}
export interface DiscoveryRequest {
  run_id: string; scope: DiscoveryScope; input_backend: DiscoveryBackend; dataset_id: string;
  selection_revision: string; generation_ids: string[]; hypotheses: { hypothesis_id: string; text: string }[];
  filters: DiscoveryFilters; max_records: number; max_candidates: number;
}
export interface DiscoveryRun {
  request: DiscoveryRequest; revision: number; status: string; score_revision: number; feedback_revision: number;
  dispatch_pending: boolean; input_record_count?: number; input_truncated?: boolean; failure_stage?: string; workflow_id?: string; created_at: string; updated_at: string;
}
export interface DiscoveryCandidate {
  candidate_id: string; episode_id: string; title: string; summary: string; labels: string[]; score: number;
  speaker_ids: string[]; attributed_actor_ids: string[]; attribution_basis: string; rationale: string;
  uncertainty: string[]; citations: { record_id: string; locator_id: string; quote: string; start_char: number; end_char: number }[];
  current_revision?: number; review_state?: string;
}
export interface DiscoveryRecord {
  record_id: string; source_version_id: string; generation_id: string; ordinal: number; record_type: string;
  platform: string; source_kind: string; conversation_id: string; explicit_episode_id?: string;
  occurred_at?: string; source_available_from?: string; text: string;
  participants: { role: string; identifier: string; display_name: string }[];
  source_ref?: string; source_sha256?: string; source_object_id?: string; source_provider_version?: string; record_revision?: string; input_revision?: string;
  imported_thread_id?: string; imported_source_id?: string; imported_message_id?: string; working_message_id?: string;
  context_uncertainty?: string[];
  group_revision?: string; native_source_available_from?: string; group_knowledge_available_from?: string;
  locators: { locator_id: string; source_version_id: string; coordinate_system: string; range_start: number; range_end: number;
    projection: Record<string, unknown>; exact_slice_sha256?: string; verification_activity_receipt_id?: string }[];
}
export type DiscoveryFeedbackKind = "relevant" | "irrelevant" | "wrong_person" | "wrong_behavior" | "reported_speech" | "missing_context" | "partial_match" | "skip" | "free_text";
export interface DiscoveryFeedback {
  feedback_id: string; candidate_id: string; expected_revision: number; kind: DiscoveryFeedbackKind;
  labels: string[]; attributed_actor_ids: string[]; note: string;
}
export interface DiscoveryReceipt { feedback: DiscoveryFeedback & { revision: number; created_at: string }; run: DiscoveryRun }
export interface DiscoveryPage { items: DiscoveryCandidate[]; revision: number; next_cursor?: string }
export interface DiscoveryContext { candidate: DiscoveryCandidate; records: DiscoveryRecord[]; revision: number }

/** Bind browser query/cache identity to the admitted case and explicit policy.
 * Input: scope. Output: stable key. Effects: none; pick for every discovery request.
 */
export function discoveryScopeKey(scope: DiscoveryScope): string {
  return JSON.stringify([scope.mode, scope.matter_id, scope.court_case_id]);
}

/** Request one fixed same-origin discovery endpoint with abortable scope pins.
 * Inputs: fixed suffix, admitted scope/options. Output: real BFF JSON or ApiError.
 * Effects: HTTP only; never send actor credentials, SQL or browser-selected URLs.
 */
async function request<T>(suffix: string, scope: DiscoveryScope, init: RequestInit = {}, cursor?: string): Promise<T> {
  const query = new URLSearchParams({ ...scope });
  if (cursor) query.set("cursor", cursor);
  let response: Response;
  try {
    response = await fetch(`${import.meta.env.VITE_API_URL || ""}/api/discovery${suffix}?${query}`, init);
  } catch (error) {
    if (init.signal?.aborted) throw error;
    throw new ApiError("Discovery connection unavailable", 0);
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new ApiError(typeof body.detail === "string" ? body.detail : `Discovery request failed (${response.status})`, response.status);
  }
  return response.json() as Promise<T>;
}

/** List admitted datasets; input scope/signal, output choices, effects read-only GET. */
export function discoveryDatasets(scope: DiscoveryScope, signal?: AbortSignal) {
  return request<{ items: DiscoveryDataset[] }>("/datasets", scope, { signal });
}
/** Start bounded discovery; input config/key/signal, output durable run, effects clicked POST. */
export function startDiscovery(body: DiscoveryRequest, key: string, signal?: AbortSignal) {
  return request<DiscoveryRun>("/runs", body.scope, { method: "POST", signal, headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify(body) });
}
/** Resume a persisted run; input identity/scope/signal, output status, effects GET only. */
export function discoveryRun(id: string, scope: DiscoveryScope, signal?: AbortSignal) {
  return request<DiscoveryRun>(`/runs/${encodeURIComponent(id)}`, scope, { signal });
}
/** Read a bounded grouped queue; input run/scope/cursor, output page, effects GET only. */
export function discoveryCandidates(id: string, scope: DiscoveryScope, cursor: string, signal?: AbortSignal) {
  return request<DiscoveryPage>(`/runs/${encodeURIComponent(id)}/candidates`, scope, { signal }, cursor);
}
/** Read source-linked context; input genuine IDs/scope, output context, effects GET only. */
export function discoveryContext(run: string, candidate: string, scope: DiscoveryScope, signal?: AbortSignal) {
  return request<DiscoveryContext>(`/runs/${encodeURIComponent(run)}/context/${encodeURIComponent(candidate)}`, scope, { signal });
}
/** Save revision-checked feedback; input choice/key/signal, output acknowledged receipt, effects POST. */
export function saveDiscoveryFeedback(run: string, scope: DiscoveryScope, body: DiscoveryFeedback, key: string, signal?: AbortSignal) {
  return request<DiscoveryReceipt>(`/runs/${encodeURIComponent(run)}/feedback`, scope, { method: "POST", signal, headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify(body) });
}
