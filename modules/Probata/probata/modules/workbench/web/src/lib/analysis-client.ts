// Byline: Codex · GPT-6 · 2026-10-08.
import { ApiError } from "@/lib/api-client";

export interface AnalysisProjection {
  matter_id: string;
  court_case_id: string;
  access_policy_id: string;
  approved_revision_id: string;
  approval_digest: string;
  projection_generation_id: string;
  projection_hash: string;
  claim_count: number;
  completed_at?: string;
  source_pins?: { source_id: string; source_version_id: string; source_object_id: string; source_object_sha256: string; source_object_uri: string }[];
}

export interface AnalysisQuery {
  access_policy_id: string;
  approved_revision_id: string;
  approval_digest: string;
  projection_generation_id: string;
  projection_hash: string;
  perspective: "as_lived" | "hindsight";
  horizon?: string;
  limit: number;
  cursor?: string;
}

export interface AnalysisClaim {
  id: string;
  kind: "entity_mention" | "statement" | "event_account";
  text: string;
  source_object_uri: string;
  source_version_id: string;
  source_sha256: string;
  record_id: string;
  record_sha256: string;
  candidate_id: string;
  candidate_sha256: string;
  predicate?: string | null;
  native_json_pointer?: string | null;
  native_span_start?: number | null;
  native_span_end?: number | null;
  native_span_unit?: "unicode_codepoint" | null;
  native_span_sha256?: string | null;
  occurred_at?: string;
  source_available_from: string | null;
  approved_at: string;
  approved_by: string;
}

export interface AnalysisStatus {
  workflow_id: string;
  outcome: string;
  error?: string;
  result?: AnalysisProjection & {
    perspective: AnalysisQuery["perspective"];
    artifact: { uri: string; version_id: string; sha256: string; bytes: number };
    has_more: boolean;
    next_cursor?: string;
  };
}

export interface AnalysisContent extends AnalysisQuery {
  matter_id: string;
  court_case_id: string;
  has_more: boolean;
  next_cursor?: string;
  claims: AnalysisClaim[];
}

const API_BASE = import.meta.env.VITE_API_URL || "";

/** List recorded completed graph snapshots for the server's existing personal case.
 * Input: optional page cursor. Output: real projection pins; effects: GET only.
 * Pick for snapshot selection; never manufacture pins from candidate rows.
 */
export function getAnalysisProjections(cursor?: string, signal?: AbortSignal) {
  return request<{ items: AnalysisProjection[]; next_cursor?: string }>(`/projections${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`, { signal });
}

/** Call the existing Workbench boundary without sending service credentials to the browser.
 * Inputs: fixed API path and fetch options. Output: bounded typed response.
 * Effects: API request only; use for analysis reads and Temporal query starts.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}/api/analysis${path}`, init);
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new ApiError(typeof payload.detail === "string" ? payload.detail : `Analysis request failed (${response.status})`, response.status);
  }
  return response.json() as Promise<T>;
}

/** Start a real query with stable projection pins and one click idempotency key.
 * Input: chosen perspective/cutoff and recorded generation. Output: workflow/run IDs.
 * Effects: bounded Temporal query; never ingestion, graph edits or evidence actions.
 */
export function startAnalysisQuery(query: AnalysisQuery, key: string) {
  return request<{ workflow_id: string; run_id: string }>("/queries", {
    method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify(query),
  });
}

/** Read actual Temporal status for a previously returned workflow.
 * Input: workflow ID/abort signal. Output: status and result reference.
 * Effects: GET only; use for bounded progress polling.
 */
export function getAnalysisStatus(workflowId: string, signal?: AbortSignal) {
  return request<AnalysisStatus>(`/workflows/${encodeURIComponent(workflowId)}`, { signal });
}

/** Read findings only after the BFF verifies artifact version, hash and source scope.
 * Input: completed workflow ID. Output: bounded findings and source citations.
 * Effects: GET only; choose over opening raw storage URLs in the browser.
 */
export function getAnalysisContent(workflowId: string, signal?: AbortSignal) {
  return request<AnalysisContent>(`/workflows/${encodeURIComponent(workflowId)}/content`, { signal });
}

/** Keep exact generation pins when changing historical perspective or paging.
 * Inputs: recorded projection and chosen read controls. Output: query contract.
 * Effects: none; never infer a revision from whichever candidate is newest.
 */
export function analysisQuery(projection: Pick<AnalysisProjection, "access_policy_id" | "approved_revision_id" | "approval_digest" | "projection_generation_id" | "projection_hash">, perspective: AnalysisQuery["perspective"], cutoff: string, cursor?: string): AnalysisQuery {
  if (perspective === "as_lived" && (!cutoff || !Number.isFinite(new Date(cutoff).getTime()))) throw new Error("Choose the as-lived cutoff date and time.");
  return {
    access_policy_id: projection.access_policy_id, approved_revision_id: projection.approved_revision_id,
    approval_digest: projection.approval_digest, projection_generation_id: projection.projection_generation_id,
    projection_hash: projection.projection_hash, perspective,
    ...(perspective === "as_lived" ? { horizon: new Date(cutoff).toISOString() } : {}), limit: 25,
    ...(cursor ? { cursor } : {}),
  };
}
