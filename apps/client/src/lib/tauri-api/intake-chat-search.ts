// Byline: Claude Code · Opus 5 · 2026-09-18
// Index-first Content Search (hosted Intake engine): the chats index (PG full-text + Weaviate hybrid)
// and the capped "Search this folder live" action. See apps/intake-engine/src/{chat_search,live_search}.rs.
import { transport } from '../transport';

export type ChatPerson = 'all' | 'katrina' | 'daughter';

export interface ChatIndexInfo {
  available: boolean;
  name: string;
  events: number;
  first: string | null;
  last: string | null;
  katrina_events: number;
  daughter_events: number;
  semantic_events: number | null;
  semantic_error: string | null;
  formats: { format: string | null; events: number }[];
}

export interface ChatTag {
  tag: 'katrina' | 'daughter';
  confidence: string;
  ref_type?: string | null;
}

export interface ChatSearchHit {
  dedup_key: string;
  date: string | null;
  sender: string | null;
  recipients: string | null;
  participants: string | null;
  source_format: string | null;
  event_kind: string | null;
  conversation_title: string | null;
  direction: string | null;
  n_sources: number | null;
  tags: ChatTag[];
  snippet: string;
  vault_key: string | null;
  source_path: string | null;
  catalog_path: string | null;
  score: number;
  matched_by: string[];
}

export interface ChatSearchResponse {
  query: string;
  scope: { name: string; events: number | null; semantic_events: number | null };
  hits: ChatSearchHit[];
  keyword_matches: number | null;
  semantic_mode: string;
  timings_ms: { keyword: number; semantic: number; total: number };
  notes: string[];
}

export interface ChatSearchParams {
  query: string;
  person?: ChatPerson;
  from?: string;
  to?: string;
  sourceFormat?: string;
  limit?: number;
}

export interface ChatEventDetail {
  dedup_key: string;
  date: string | null;
  tz_status: string | null;
  ts_original: string | null;
  source_format: string | null;
  event_kind: string | null;
  conversation_title: string | null;
  sender: string | null;
  recipients: string | null;
  participants: string | null;
  direction: string | null;
  counterparty_phone: string | null;
  contact_name: string | null;
  body: string | null;
  attachments: string | null;
  tags: ChatTag[];
  vault_key: string | null;
  catalog_rel: string | null;
  n_sources: number | null;
  provenance: Array<Record<string, string | number | null>>;
  table: string;
}

export interface LiveFolderMatch {
  file: string;
  line: number;
  content: string;
  filename: string;
}

export interface LiveFolderSearchResult {
  root: string;
  query: string;
  matches: LiveFolderMatch[];
  files_read: number;
  bytes_read: number;
  skipped_large: number;
  skipped_binary: number;
  entries_listed: number;
  stopped: string | null;
  caps: { max_files: number; max_bytes: number; max_file_bytes: number };
  elapsed_ms: number;
}

export interface LiveFolderProgress {
  searchId: string | null;
  root: string;
  current: string;
  done: boolean;
  files_read: number;
  bytes_read: number;
  skipped_large: number;
  matches: number;
  max_files: number;
  max_bytes: number;
}

export const LIVE_SEARCH_PROGRESS_EVENT = 'intake-live-search-progress';

export const getChatIndexInfo = async (): Promise<ChatIndexInfo> =>
  await transport('intake_chat_index_info');

export const searchChatIndex = async (params: ChatSearchParams): Promise<ChatSearchResponse> =>
  await transport('intake_chat_search', { ...params });

export const getChatEvent = async (dedupKey: string): Promise<ChatEventDetail> =>
  await transport('intake_chat_event', { dedupKey });

export const liveFolderSearch = async (
  path: string,
  query: string,
  searchId: string,
): Promise<LiveFolderSearchResult> =>
  await transport('intake_live_folder_search', { path, query, searchId, maxResults: 200 });
