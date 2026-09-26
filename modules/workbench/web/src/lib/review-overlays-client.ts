// Byline: Claude Code · Opus 5.5 · 2026-09-26 (metadata screen + context review client)
/**
 * Client for the Review overlays: the file metadata screen with the owner's
 * append-only corrections, and per-message context review with the
 * hindsight-only foreshadowing flag.
 *
 * Observed values are never edited here: a correction is a new attributed
 * revision. The actor comes from the authenticated session on the server; the
 * BFF derives each write's idempotency key from the exact submission.
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

function post<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
}

function previewPath(previewHandle: string, suffix: string, mode: MatterMode, extra: Record<string, string> = {}) {
  const query = new URLSearchParams({ mode, ...extra });
  return `/api/proffer/previews/${encodeURIComponent(previewHandle)}${suffix}?${query.toString()}`;
}

// ---------------------------------------------------------------- metadata screen

export type MetadataClass = "filesystem" | "embedded" | "container" | "media_tool" | "record_native";

export interface ObjectFacts {
  object_ref: string;
  storage_class: string;
  sha256: string;
  byte_length: number;
  immutable_at: string;
}

export interface SourceFacts {
  source_version_ref: string;
  source_key: string;
  provenance_class: string;
  declared_format: string;
  original_filename: string | null;
  acquired_at: string | null;
  status: string;
  matter_id: string | null;
  court_case_id: string | null;
  source_context_ref: string | null;
  original: ObjectFacts | null;
}

export interface MetadataRow {
  metadata_ref: string;
  metadata_class: MetadataClass;
  extractor_id: string;
  extractor_version: string | null;
  generated_at: string;
  receipt_ref: string;
  /** Native JSON exactly as recorded; null only when larger than the screen reads. */
  fields: unknown;
  fields_bytes: number;
}

export interface RetainedMember {
  object_ref: string;
  role: string;
  parent_object_ref: string | null;
  sha256: string;
  byte_length: number;
  storage_class: string;
  member_locator: Record<string, unknown>;
}

export interface AttachmentFacts {
  sha256: string;
  filename: string | null;
  media_type: string | null;
  byte_length: number | null;
  source_locator_ref: string;
  message_ids: string[];
}

export interface HashReceipt {
  hash_kind: string;
  construction: string;
  digest: string;
  computed_at: string;
  computed_by: string;
}

export interface MetadataCorrection {
  correction_ref: string;
  subject_sha256: string;
  field_key: string;
  revision: number;
  supersedes_ref: string | null;
  action: "correct" | "retract";
  source_value: unknown;
  corrected_value: unknown;
  change_reason: string;
  actor_username: string;
  receipt_ref: string;
  recorded_at: string;
}

export type SidecarKind = "takeout_json" | "json" | "xmp" | "apple_aae" | "owner_sidecar_md" | "owner_extraction_md" | "companion";

export interface Sidecar {
  name: string;
  key: string;
  kind: SidecarKind;
  found_via: "beside_object" | "catalog_folder";
  byte_length: number | null;
  fields: Array<{ path: string; value: unknown }>;
  fields_truncated: boolean;
  text: string | null;
  text_truncated: boolean;
  unread_reason: string | null;
}

export interface SidecarConflict {
  topic: "capture_time" | "gps";
  sidecar_key: string;
  sidecar_path: string;
  sidecar_value: unknown;
  file_field: string;
  file_value: unknown;
  detail: string;
}

export type LookupState = "ok" | "unavailable" | "not_configured" | "not_applicable";

export interface MetadataScreen {
  preview_handle: string;
  request_id: string;
  source_ref: string;
  subject_kind: "source" | "member" | "attachment";
  subject_sha256: string;
  source: SourceFacts | null;
  metadata: MetadataRow[];
  metadata_truncated: boolean;
  record_metadata_count: number;
  members: RetainedMember[];
  members_truncated: boolean;
  attachment: AttachmentFacts | null;
  attachment_count: number;
  hashes: HashReceipt[];
  corrections: MetadataCorrection[];
  corrections_available: boolean;
  sidecars: Sidecar[];
  sidecar_conflicts: SidecarConflict[];
  sidecar_lookup: { beside_object: LookupState; catalog_folder: LookupState };
  matter_mode: MatterMode;
}

export interface MetadataCorrectionInput {
  subject_sha256: string;
  field_key: string;
  supersedes_ref: string;
  action: "correct" | "retract";
  source_value: unknown;
  corrected_value: unknown;
  change_reason: string;
}

export interface MetadataCorrectionReceipt {
  correction_ref: string;
  receipt_ref: string;
  content_digest: string;
  revision: number;
  recorded_at: string;
  matter_mode: MatterMode;
}

export function getMetadataScreen(previewHandle: string, mode: MatterMode, subjectSha256?: string | null) {
  const extra: Record<string, string> = subjectSha256 ? { subject_sha256: subjectSha256 } : {};
  return request<MetadataScreen>(previewPath(previewHandle, "/metadata", mode, extra));
}

export function postMetadataCorrection(previewHandle: string, mode: MatterMode, body: MetadataCorrectionInput) {
  return post<MetadataCorrectionReceipt>(previewPath(previewHandle, "/metadata/corrections", mode), body);
}

// ---------------------------------------------------------------- context review

export type Horizon = "as_lived" | "hindsight";
export type AboutChild = "yes" | "no" | "unsure";

export interface ReviewParty {
  label: string;
  entity_id: string | null;
}

export interface ReviewAssertions {
  addressed_to: ReviewParty[];
  about: ReviewParty[];
  about_child: AboutChild | null;
  relevant: boolean | null;
}

export interface ReviewRevision {
  review_ref: string;
  revision: number;
  assertions: ReviewAssertions;
  change_reason: string;
  actor_username: string;
  receipt_ref: string;
  recorded_at: string;
}

export interface ForeshadowingRevision {
  flag_ref: string;
  revision: number;
  foreshadowing: boolean;
  note: string;
  horizon: "hindsight";
  knowledge_time: string;
  change_reason: string;
  actor_username: string;
  receipt_ref: string;
}

export interface ContextReview {
  preview_handle: string;
  message_id: string;
  horizon: Horizon;
  reviews: ReviewRevision[];
  /** Present only on a hindsight read; never on an as-lived read. */
  foreshadowing?: ForeshadowingRevision[];
  matter_mode: MatterMode;
}

export interface ContextReviewInput extends ReviewAssertions {
  supersedes_ref: string;
  change_reason: string;
}

export interface ForeshadowingInput {
  supersedes_ref: string;
  foreshadowing: boolean;
  note: string;
  change_reason: string;
}

export interface ContextReviewReceipt {
  ref: string;
  receipt_ref: string;
  content_digest: string;
  revision: number;
  recorded_at: string;
  horizon: Horizon;
  matter_mode: MatterMode;
}

const messagePath = (messageId: string, suffix: string) => `/messages/${encodeURIComponent(messageId)}${suffix}`;

export function getContextReview(previewHandle: string, mode: MatterMode, messageId: string, horizon: Horizon) {
  return request<ContextReview>(previewPath(previewHandle, messagePath(messageId, "/context-review"), mode, { horizon }));
}

export function postContextReview(previewHandle: string, mode: MatterMode, messageId: string, body: ContextReviewInput) {
  return post<ContextReviewReceipt>(previewPath(previewHandle, messagePath(messageId, "/context-review"), mode), body);
}

export function postForeshadowing(previewHandle: string, mode: MatterMode, messageId: string, body: ForeshadowingInput) {
  return post<ContextReviewReceipt>(previewPath(previewHandle, messagePath(messageId, "/foreshadowing"), mode), body);
}
