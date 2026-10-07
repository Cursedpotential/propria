// Byline: Codex · 2026-09-20. Pre-ingest catalog discovery contracts.
export interface DiscoveryItem {
  id: string;
  rel: string;
  parent: string;
  name: string;
  kind: "file" | "directory" | "content_hit";
  text?: string;
  score?: number;
  source_id?: string;
  document_id?: string;
  chunk_id?: string;
  size: number | null;
  modified_at: string | null;
  recorded_at: string | null;
  source: string | null;
  scope: string | null;
  vault_key: string | null;
  /** Intake's reported locator status; not proof of evidence eligibility. */
  resolution?: string | null;
}
export interface DiscoveryCapabilities {
  backend: string;
  catalog_configured: boolean;
  index_configured: boolean;
  availability_verified: boolean;
  modes: { filename_substring: boolean; filename_prefix: boolean; contents: boolean; hybrid: boolean };
  tree: boolean;
  graph: boolean;
  coverage: string;
  filters: { parent: boolean; file_type: boolean; atomic_unit: boolean };
  zip_contents: boolean;
  bulk_intake: boolean;
  atomic_unit_catalog: boolean;
  unit_types?: string[];
  limitations: string[];
  [key: string]: unknown;
}
export interface DiscoveryPage {
  backend: string;
  collection?: string | null;
  items: DiscoveryItem[];
  next_cursor: string | null;
  has_more: boolean;
  complete: boolean;
  query_scope: Record<string, unknown>;
  notice?: string;
  coverage?: string;
  capabilities?: DiscoveryCapabilities;
  freshness?: { catalog_snapshot: string; checked_at_is_source_update: boolean };
}

export interface DiscoveryUnit {
  unit_id: number;
  unit_type: string;
  source: string | null;
  export_root: string | null;
  service: string | null;
  unit_root: string;
  member_count: number;
  total_bytes: number;
  members_without_sha1: number;
  parent_unit_id: number | null;
}
export interface DiscoveryUnits {
  items: DiscoveryUnit[];
  next_after_id: number | null;
  coverage: string;
  source_links_verified: boolean;
  notice: string;
}
export interface DiscoveryUnitMembers {
  next_cursor: string | null;
  items: { unit_id: number; key: string }[];
  has_more: boolean;
  source_links_verified: boolean;
  notice: string;
}
