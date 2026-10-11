// Byline: Codex · 2026-10-06. Read-only combined-search browser contract.
export type RetrievalMode = "keyword" | "hybrid" | "vector";
export type RetrievalLeg = "intake" | "proffer";
export interface RetrievalHit {
  text: string;
  score: number;
  legs: string[];
  citation: {
    collection: string;
    object_id: string;
    source_id?: string | null;
    source_version_ids: string[];
    document_id?: string | null;
    chunk_id?: string | null;
    first_message_id?: string | null;
    locator: Record<string, string>;
  };
  location?: { name: string; href: string };
  navigation?: { status: "resolved" | "unresolved" | "unavailable"; sources: {
    source_version_id: string; source_uri: string; name: string; href: string;
  }[] };
}
export interface RetrievalResult {
  request_id: string;
  mode: RetrievalMode;
  status: "success" | "partial" | "failed";
  items: RetrievalHit[];
  legs: { name: string; status: "success" | "failed"; count: number; error?: string | null }[];
}

export interface GraphResolution {
  source_id: string;
  document_id: string;
  ambiguous: boolean;
  overflow: boolean;
  matches: { table: "occurrence"; key: string; record_id: string; snapshot_key: string;
    manifest_sha256: string; version_id: string | null }[];
}
