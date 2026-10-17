// Source-backed unit read contracts. No browser-side membership or success inference.
// Byline: ChatGPT, 2026-10-17.
import type { ImportedMessage, ImportedSource } from "./imported-client";

export interface UnitVersion {
  source_version_id: string;
  source_key: string;
  export_key: string;
  conv: string;
  version_ordinal: number;
  retained_versions: number;
  generation_id: string | null;
  generation_ordinal: number | null;
  acquired_at: string | null;
  published: boolean;
  raw_records: number;
  normalized_records: number;
  messages: number;
}
export interface UnitMember {
  source: ImportedSource;
  versions: UnitVersion[];
  messages: number;
  normalized_records: number;
}
export interface ImportedUnit {
  source_id: string;
  root: string;
  membership_basis: "registered_export" | "registered_facebook_folder";
  conversation: string | null;
  members: UnitMember[];
  extraction_targets: Array<{ thread_id: string; source_id: string; label: string; generation_ids: string[] }>;
  generation_signature: string;
  generation_policy: string;
  read_at: string;
  coverage: string;
  message_count: number;
  normalization_verified: false;
}
export interface UnitMessage extends ImportedMessage {
  source: { id: string; file_name: string; source_key: string; export_key: string; thread_id: string };
  projection_kind: string | null;
}
export interface UnitMessagePage {
  items: UnitMessage[];
  next_cursor: string | null;
  generation_signature: string;
  read_at: string;
  query: string;
  search_scope: "all_selected_pg_records";
}
export interface UnitRecord {
  id: string;
  source_version_id: string;
  generation_id: string;
  record_ordinal: number;
  record_type: string;
  occurred_at: string | null;
  normalized_payload: Record<string, unknown>;
  source_key: string;
  export_key: string;
  preview_handle: string | null;
  current_generation: boolean;
  read_at: string;
}
