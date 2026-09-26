// Byline: Claude Code · Opus 5 · 2026-09-22
// Name search across the whole catalog and all of B2 (owner 2026-09-22 18:54 EDT: the magnifying
// glass must find a folder name anywhere, not only inside the open folder).
// Engine side: apps/intake-engine/src/name_search.rs.
import { transport } from '../transport';

/** The scope dropdown, in the order it is shown. Ids are the engine's (`Scope::id`). */
export const NAME_SEARCH_SCOPES = [
  { id: 'everything', labelKey: 'intakeNameSearch.scopeEverything' },
  { id: 'this_folder', labelKey: 'intakeNameSearch.scopeThisFolder' },
  { id: 'this_folder_and_subfolders', labelKey: 'intakeNameSearch.scopeThisFolderAndSubfolders' },
  { id: 'subfolders_only', labelKey: 'intakeNameSearch.scopeSubfoldersOnly' },
  { id: 'one_level_up', labelKey: 'intakeNameSearch.scopeOneLevelUp' },
  { id: 'catalog_only', labelKey: 'intakeNameSearch.scopeCatalogOnly' },
  { id: 'b2_only', labelKey: 'intakeNameSearch.scopeB2Only' },
] as const;

export type NameSearchScope = (typeof NAME_SEARCH_SCOPES)[number]['id'];

/** Scopes that mean nothing without knowing which folder is open (mirrors `Scope::needs_folder`). */
export const SCOPES_NEEDING_FOLDER: readonly NameSearchScope[] = [
  'this_folder',
  'this_folder_and_subfolders',
  'subfolders_only',
  'one_level_up',
];

export type NameSearchKinds = 'both' | 'folders' | 'files';
export type NameSearchSort = 'name' | 'path';

export interface NameSearchHit {
  kind: 'folder' | 'file';
  name: string;
  /** The full path of the hit itself (`<zip>!/<entry>` for a file inside an archive). */
  path: string;
  /** The folder the active pane opens when the row is clicked. */
  navigate_to: string;
  /** The file to select once that folder has loaded; null for a folder row. */
  select: string | null;
  /** Where the bytes are now. */
  now: string | null;
  /** Where the catalog recorded it originally. */
  was: string | null;
  source: 'catalog' | 'b2' | 'zip';
  matched: 'name' | 'path';
  member_of: string | null;
  size: number | null;
  modified: number | null;
  flag: string | null;
}

export interface NameSearchLeg {
  leg: string;
  rows: number;
  ms: number;
  capped: boolean;
  error: string | null;
}

export interface NameSearchResult {
  query: string;
  scope: NameSearchScope;
  kinds: NameSearchKinds;
  sort: NameSearchSort;
  folder: string | null;
  hits: NameSearchHit[];
  shown_folders: number;
  shown_files: number;
  total_folders: number;
  total_files: number;
  capped: boolean;
  scan_cap: number;
  elapsed_ms: number;
  legs: NameSearchLeg[];
  notes: string[];
}

export interface NameSearchParams {
  query: string;
  scope?: NameSearchScope;
  /** The active pane's folder; only read by the folder-relative scopes. */
  path?: string;
  kinds?: NameSearchKinds;
  sort?: NameSearchSort;
  limit?: number;
}

export const searchNames = async (params: NameSearchParams): Promise<NameSearchResult> =>
  await transport('intake_search_names', { limit: 200, ...params });
