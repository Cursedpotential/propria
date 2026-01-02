
export type VerificationStatus = "Pending" | "Verified" | "Flagged";

export interface EvidenceSource {
  id: string; // UUIDv7
  file_hash: string; // SHA-256
  filename: string;
  imported_at: Date;
}

export interface EvidenceChunk {
  id: string; // UUIDv7
  source_id: string; // FK to EvidenceSource
  raw_text: string;
  chunk_hash: string; // SHA-256 of raw_text
  sequence_index: number;
}

export interface AnalysisRecord {
  id: string; // UUIDv7
  chunk_id: string; // FK to EvidenceChunk
  short_id: string; // last 8 chars of ID
  clean_text: string;
  entities: Array<{ name: string; type: string; }>; // JSON
  timeline_date: Date | null;
  tags: string[]; // JSON
  verification_status: VerificationStatus;
}

export interface IngestionResult {
  success: boolean;
  message: string;
  sourceId?: string;
  chunkCount?: number;
}

export interface AppSettings {
  geminiApiKey: string;
  proxyUrl: string;
  useProxy: boolean;
  debugMode: boolean;
  contextKeywords: Record<string, string[]>;
}
