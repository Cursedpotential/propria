// Byline: Claude Code · Opus 5.5 · 2026-09-25 (entity + event extraction client: propose -> correct -> commit)
/**
 * Client for the Workbench BFF's entity/event extraction routes. Every owner
 * act sends an Idempotency-Key minted once per click, so a retried request never
 * writes twice. Proposals and events arrive exactly as the engine stages them.
 */
import { ApiError } from "@/lib/api-client";
import type { MatterMode } from "@/lib/shared/types";

const API_BASE = import.meta.env.VITE_API_URL || "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, init);
  } catch {
    throw new ApiError("Network error — check your connection", 0);
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    const detail = typeof body.detail === "string" ? body.detail : `API error: ${response.status}`;
    throw new ApiError(detail, response.status);
  }
  return response.json() as Promise<T>;
}

/** One key per owner action; reused only by an automatic retry of that action. */
export function newIdempotencyKey(action: string) {
  const random = typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  return `${action}-${random}`;
}

function post<T>(path: string, body: unknown, key?: string): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (key) headers["Idempotency-Key"] = key;
  return request<T>(path, { method: "POST", headers, body: JSON.stringify(body) });
}

export interface ExtractionFlag {
  code: string;
  detail: string;
}

export interface EntityAlias {
  text: string;
  kind: string;
  address_kind?: string;
  normalized: string;
  source: string;
  confidence: number;
  /** Set only for a source-relative identity ("self"); never a registry alias. */
  scope?: string;
}

export interface EntityMention {
  record_id: string;
  ordinal: number;
  occurred_at?: string;
  source_available_from?: string;
  kind: string;
  role: string;
  surface: string;
  start?: number;
  end?: number;
  snippet?: string;
  method: string;
  confidence: number;
}

export interface EntityMatch {
  entity_id: string;
  display_name: string;
  via: string;
  alias?: string;
  score: number;
}

export interface EntitySuggestion {
  entity_id?: string;
  display_name: string;
  reason: string;
  score: number;
}

export interface OwnerCorrection {
  op: string;
  actor: { subject_uid: string; username: string };
  at: string;
  note?: string;
}

export interface EntityProposal {
  candidate_id: string;
  name: string;
  registry_type: string;
  aliases: EntityAlias[];
  mention_count: number;
  mention_sample?: EntityMention[];
  model_mentions?: EntityMention[];
  model_mention_count: number;
  match?: EntityMatch | null;
  suggestions?: EntitySuggestion[];
  confidence: number;
  detected_by: "auto" | "owner";
  extractors: string[];
  source_owner?: boolean;
  flags?: ExtractionFlag[];
  correction?: OwnerCorrection | null;
  review_state: "pending" | "approved" | "rejected";
  promoted_to_id?: string;
  first_occurred_at?: string;
  last_occurred_at?: string;
  keys: string[];
}

export interface EventSourceRecord {
  record_id: string;
  ordinal: number;
  occurred_at?: string;
  source_available_from?: string;
  snippet?: string;
}

export interface EventProposal {
  candidate_id: string;
  title: string;
  description?: string;
  event_type: string;
  occurred_at?: string;
  temporal_precision: "point" | "interval" | "uncertain";
  temporal_confidence: number;
  when_stated?: string;
  source_records: EventSourceRecord[];
  entity_keys?: string[];
  entity_names?: string[];
  detected_by: "auto" | "owner";
  flags?: ExtractionFlag[];
  correction?: OwnerCorrection | null;
  review_state: "pending" | "approved" | "rejected";
  /** The latest availability of the event's sources: an as-lived reader never sees it earlier. */
  source_available_from?: string;
  resolved_entity_candidate_ids: string[];
  unresolved_entity_keys?: string[];
}

export interface ExtractionRunSummary {
  id: string;
  extractor: string;
  extractor_version: string;
  model_id?: string;
  status: string;
  error?: string;
  started_at: string;
  finished_at?: string;
  stats: Record<string, unknown>;
}

export interface ProposalsResponse {
  preview_handle: string;
  normalized_generation_id: string;
  matter_mode: MatterMode;
  entities: EntityProposal[];
  events: EventProposal[];
  extractions: ExtractionRunSummary[];
  entity_types: string[];
  event_types: string[];
  alias_kinds: string[];
}

export interface WorkflowStep {
  step: string;
  status: "pending" | "running" | "completed" | "skipped" | "failed";
  detail: string;
  counts: Record<string, number>;
  flags: ExtractionFlag[];
}

export interface WorkflowProgress {
  workflow_id: string;
  outcome: string;
  steps: WorkflowStep[];
}

export interface ValidationCheck {
  rule: string;
  status: "pass" | "fail";
  reason: string;
}

export interface ValidationReport {
  ok: boolean;
  checks: ValidationCheck[];
  digest: string;
  counts: Record<string, number>;
}

export interface RecordView {
  record_id: string;
  ordinal: number;
  occurred_at?: string | null;
  source_available_from?: string | null;
  body: string;
  participants: Array<{ role: string; identifier: string; display_name?: string | null }>;
}

export interface RegistryEntity {
  id: string;
  display_name: string;
  registry_type: string;
}

export type EntityCorrection =
  | { op: "reject" | "restore" | "clear_match"; candidate_ids: [string] }
  | { op: "rename"; candidate_ids: [string]; name: string }
  | { op: "retype"; candidate_ids: [string]; registry_type: string }
  | { op: "add_alias"; candidate_ids: [string]; alias_text: string; alias_kind?: string }
  | { op: "remove_alias"; candidate_ids: [string]; alias_text: string }
  | { op: "merge"; candidate_ids: string[]; name?: string }
  | { op: "split"; candidate_ids: [string]; name: string; split_aliases: string[] }
  | { op: "set_match"; candidate_ids: [string]; match: { entity_id: string; display_name?: string } };

export type EventCorrection =
  | { op: "reject" | "restore"; candidate_ids: [string] }
  | {
      op: "edit";
      candidate_ids: [string];
      title?: string;
      description?: string;
      event_type?: string;
      occurred_at?: string;
      set_entities?: boolean;
      entity_keys?: string[];
    }
  | { op: "merge"; candidate_ids: string[]; title?: string };

const query = (params: Record<string, string>) => new URLSearchParams(params).toString();

export function getEntityProposals(previewHandle: string, mode: MatterMode, signal?: AbortSignal) {
  return request<ProposalsResponse>(`/api/entities/proposals?${query({ preview_handle: previewHandle, mode })}`, { signal });
}

export function startEntityExtraction(previewHandle: string, mode: MatterMode, useModel: boolean, key: string) {
  return post<{ workflow_id: string; normalized_generation_id: string; use_model: boolean }>(
    `/api/entities/extract?${query({ mode })}`,
    { preview_handle: previewHandle, use_model: useModel },
    key,
  );
}

export function getWorkflowProgress(kind: "extraction" | "commit", workflowId: string, previewHandle: string, mode: MatterMode, signal?: AbortSignal) {
  const segment = kind === "extraction" ? "extractions" : "commits";
  return request<WorkflowProgress>(
    `/api/entities/${segment}/${encodeURIComponent(workflowId)}?${query({ preview_handle: previewHandle, mode })}`,
    { signal },
  );
}

export function correctEntity(previewHandle: string, mode: MatterMode, entity: EntityCorrection, key: string) {
  return post<{ ok: boolean }>(`/api/entities/corrections?${query({ mode })}`, { preview_handle: previewHandle, target: "entity", entity }, key);
}

export function correctEvent(previewHandle: string, mode: MatterMode, event: EventCorrection, key: string) {
  return post<{ ok: boolean }>(`/api/entities/corrections?${query({ mode })}`, { preview_handle: previewHandle, target: "event", event }, key);
}

export function markEventFromRecord(
  previewHandle: string,
  mode: MatterMode,
  recordId: string,
  details: { title?: string; description?: string; event_type?: string },
  key: string,
) {
  return post<{ event: EventProposal; source_available_from?: string | null }>(
    `/api/events/from-record?${query({ mode })}`,
    { preview_handle: previewHandle, record_id: recordId, ...details },
    key,
  );
}

export function getExtractionRecord(previewHandle: string, mode: MatterMode, recordId: string, signal?: AbortSignal) {
  return request<RecordView>(
    `/api/entities/records/${encodeURIComponent(recordId)}?${query({ preview_handle: previewHandle, mode })}`,
    { signal },
  );
}

export function searchCommittedEntities(q: string, signal?: AbortSignal) {
  return request<{ entities: RegistryEntity[] }>(`/api/entities/registry?${query({ q, limit: "20" })}`, { signal });
}

export function validateEntityCommit(previewHandle: string, mode: MatterMode) {
  return post<ValidationReport>(`/api/entities/validate?${query({ mode })}`, { preview_handle: previewHandle });
}

export function commitEntities(previewHandle: string, mode: MatterMode, digest: string, key: string) {
  return post<{ workflow_id: string; commit_id: string; counts: Record<string, number> }>(
    `/api/entities/commit?${query({ mode })}`,
    { preview_handle: previewHandle, digest },
    key,
  );
}
