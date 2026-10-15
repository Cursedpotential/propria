// Byline: Codex · GPT-6 · 2026-10-11.
import { ApiError } from "@/lib/api-client";
import { analysisQuery, type AnalysisClaim, type AnalysisProjection, type AnalysisQuery } from "@/lib/analysis-client";

/** Keep the real immutable result locator; metadata only, never browser storage credentials. */
export interface WalkArtifact { uri: string; version_id: string; sha256: string; bytes: number }
/** Carry exact graph pins and explicitly ordered historical dates to the existing starter. */
export interface WalkRequest { scope: AnalysisQuery; cutoffs: string[] }
/** Describe a real Temporal execution and its reference-only result, without invented progress. */
export interface WalkStatus {
  workflow_id: string; outcome: string; step: number; error?: string;
  result?: { artifact: WalkArtifact; checkpoints: { artifact: WalkArtifact }[]; partial: boolean; review_status: string };
}
/** Preserve a source-grounded analytical finding and its exact native locator. */
export interface WalkFinding {
  kind: string; text: string; citations: { claim_id: string; quote: string; source_pin: AnalysisClaim }[];
}
/** Describe one lazily loaded historical checkpoint, separate from hindsight. */
export interface WalkCheckpoint {
  perspective: "as_lived" | "hindsight"; horizon: string | null; step_index: number;
  findings: WalkFinding[]; limitations: string[]; review_status: string;
  model: unknown; prompt_version: string; prompt_sha256: string;
}
/** Expose the actual compact comparison and references, without fetching every checkpoint. */
export interface WalkContent {
  schema: "approved-walk-result-v1"; cutoffs: string[]; partial: boolean; review_status: string;
  checkpoints: { artifact: WalkArtifact }[]; coverage: string; validation: string; family_independence: string;
  comparison: { hindsight_only_claim_ids: string[]; available_after_last_cutoff_claim_ids: string[];
    unknown_availability_claim_ids: string[]; hindsight_findings_not_identical_to_last_as_lived: WalkFinding[] };
}

/** Use the existing same-origin boundary for a bounded walk request.
 * Inputs: fixed API path/options. Output: typed response. Effects: API I/O only.
 * Choose instead of opening protected B2 locators from the browser.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${import.meta.env.VITE_API_URL || ""}/api/analysis${path}`, init);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new ApiError(typeof body.detail === "string" ? body.detail : `Analysis request failed (${response.status})`, response.status);
  }
  return response.json() as Promise<T>;
}

/** Preserve selected pins and explicit dates without silently sorting the owner's sequence.
 * Inputs: recorded projection and local date/time values. Output: Go-compatible body.
 * Effects: none; choose before a single explicit walk start.
 */
export function walkRequest(projection: AnalysisProjection, dates: string[]): WalkRequest {
  if (!dates.length || dates.length > 20 || dates.some((date) => !date || !Number.isFinite(new Date(date).getTime()))) throw new Error("Enter between one and twenty date/time cutoffs.");
  const cutoffs = dates.map((date) => new Date(date).toISOString());
  if (cutoffs.some((date, index) => index > 0 && date <= cutoffs[index - 1])) throw new Error("Enter the cutoffs in increasing order.");
  return { scope: analysisQuery(projection, "as_lived", cutoffs[cutoffs.length - 1]), cutoffs };
}

/** Start one real walk; inputs scope/dates/stable key, output execution IDs, effects Temporal start. */
export function startWalk(body: WalkRequest, key: string) {
  return request<{ workflow_id: string; run_id: string }>("/walks", { method: "POST",
    headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify(body) });
}
/** Read real walk progress; input execution ID/signal, output status, effects GET only. */
export function getWalkStatus(id: string, signal?: AbortSignal) {
  return request<WalkStatus>(`/walk-workflows/${encodeURIComponent(id)}`, { signal });
}
/** Read the verified compact result; input completed ID, output comparison/refs, effects bounded GET. */
export function getWalkContent(id: string, signal?: AbortSignal) {
  return request<WalkContent>(`/walk-workflows/${encodeURIComponent(id)}/content`, { signal });
}
/** Read one selected verified step; inputs ID/index, output cited checkpoint, effects bounded GET. */
export function getWalkStep(id: string, index: number, signal?: AbortSignal) {
  return request<{ artifact: WalkArtifact; checkpoint: WalkCheckpoint; has_more: boolean; claim_count: number }>(
    `/walk-workflows/${encodeURIComponent(id)}/steps/${index}/content`, { signal });
}
