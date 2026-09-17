// Byline: Claude Code · Opus 5 · 2026-09-17
// Hosted Intake engine commands (engine-native, not in the donor): extracted metadata for the
// native Metadata panel and catalog history for a file. Paths may be B2 mount paths or catalog://.
import { transport } from '../transport';

export interface IntakeCatalogOccurrence {
  catalog_path: string | null;
  source: string;
  scope: string | null;
  path: string;
  source_id: string | null;
  size: number | null;
  modtime: string | null;
  md5: string | null;
  native_hash_kind: string | null;
  native_hash: string | null;
  disposition: string | null;
  b2_key_recorded: string | null;
  matched_origin: string | null;
  metadata: unknown;
  recorded_at: string | null;
  sha1: string | null;
}

export interface IntakeCatalogLookup {
  b2_key: string | null;
  vault_object?: { key: string; size: number | null; sha1: string | null } | null;
  occurrences: IntakeCatalogOccurrence[];
  count: number;
  engine_ops?: Array<Record<string, unknown>>;
  note?: string;
}

export type IntakeFileMetadata = Record<string, unknown>;

export const getIntakeFileMetadata = async (path: string): Promise<IntakeFileMetadata> =>
  await transport('intake_file_metadata', { path });

export const getIntakeCatalogLookup = async (path: string): Promise<IntakeCatalogLookup> =>
  await transport('intake_catalog_lookup', { path });
