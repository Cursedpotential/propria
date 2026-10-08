// Byline: Codex · GPT-6 · 2026-10-08.
import { ApiError } from "@/lib/api-client";

/** Preserve the atomic staging pin without a normalized conversation identity.
 * Input: Go Stage result. Output: exact original source identity. Effects: none.
 */
export interface AIReviewSource {
  source_version_id: string;
  source_ref: string;
  version_id: string | null;
  source_object_id: "";
  source_sha256: string;
  prepared_ref: string;
}

export type AIDecision = "approved" | "rejected" | "needs_info";

/** Describe one staged proposal with its grounded native source locator.
 * Input: existing candidate API. Output: display data. Effects: none.
 */
export interface AICandidateRow {
  candidate_id: string;
  kind: string;
  reported_kind: string;
  review_domain: string;
  review_state: string;
  content_sha256: string;
  decision_id?: string;
  candidate: AIReviewSource & {
    name?: string;
    statement?: string;
    evidence_quote?: string;
    predicate?: string;
    entity_type?: string;
    event_type?: string;
    occurred_at: string | null;
    native_json_pointer?: string;
    span_unit: string;
    source_span: { start: number; end: number; sha256: string };
  };
}

/** Keep the committed choice separate from any later graph operation.
 * Input: Go decision reply. Output: real receipt. Effects: none.
 */
export interface AIDecisionReceipt {
  candidate_id: string;
  decision: AIDecision;
  decision_id: string;
  request_digest: string;
  matter_mode: "LIVE";
  decision_committed?: true;
  projection?: {
    status: "enqueued" | "pending" | "not_requested";
    workflow_id?: string;
    run_id?: string;
    retryable?: boolean;
    error?: string;
  };
}

/** Call the same-origin BFF without exposing service tokens or accepting invented pins.
 * Inputs: workflow ID, fixed suffix/options. Output: actual API response.
 * Effects: one request; use only for staged AI review.
 */
async function request<T>(workflowId: string, suffix: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${import.meta.env.VITE_API_URL || ""}/api/context/sources/workflows/${encodeURIComponent(workflowId)}/candidates${suffix}`, init);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new ApiError(typeof body.detail === "string" ? body.detail : `Candidate review failed (${response.status})`, response.status);
  }
  return response.json() as Promise<T>;
}

/** Read one real candidate page pinned server-side to its context workflow.
 * Inputs: returned workflow ID and cursor. Output: original pin and candidates.
 * Effects: GET only; no approvals or ingestion.
 */
export function getAICandidates(workflowId: string, afterId: string, signal?: AbortSignal) {
  return request<{ source: AIReviewSource; source_version_id: string; candidates: AICandidateRow[]; next_cursor: string }>(workflowId,
    afterId ? `?after_id=${encodeURIComponent(afterId)}` : "", { signal });
}

/** Submit exactly one user choice with its displayed digest and stable retry key.
 * Inputs: candidate row, explicit decision, workflow ID and key. Output: receipt.
 * Effects: existing semantic decision only; never assume graph publication.
 */
export function decideAICandidate(workflowId: string, row: AICandidateRow, decision: AIDecision, key: string) {
  return request<AIDecisionReceipt>(workflowId, "/decision", {
    method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key },
    body: JSON.stringify({ candidate_id: row.candidate_id, expected_content_sha256: row.content_sha256, decision }),
  });
}
