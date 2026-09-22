// Thin re-export — the real implementation lives in ./tauri-api/ domain modules.
export { TauriAPI } from './tauri-api/index';
export * from './tauri-api-types';
export type {
  IntakeCatalogLookup,
  IntakeCatalogOccurrence,
  IntakeFileMetadata,
} from './tauri-api/intake-engine';
// Index-first Content Search (Claude Code · Opus 5 · 2026-09-18).
export { LIVE_SEARCH_PROGRESS_EVENT } from './tauri-api/intake-chat-search';
export type {
  ChatEventDetail,
  ChatIndexInfo,
  ChatPerson,
  ChatSearchHit,
  ChatSearchParams,
  ChatSearchResponse,
  ChatTag,
  LiveFolderMatch,
  LiveFolderProgress,
  LiveFolderSearchResult,
  TimelineResult,
} from './tauri-api/intake-chat-search';
// Name search across the catalog and all of B2 (Claude Code · Opus 5 · 2026-09-22).
export { NAME_SEARCH_SCOPES, SCOPES_NEEDING_FOLDER } from './tauri-api/intake-name-search';
export type {
  NameSearchHit,
  NameSearchKinds,
  NameSearchLeg,
  NameSearchParams,
  NameSearchResult,
  NameSearchScope,
  NameSearchSort,
} from './tauri-api/intake-name-search';
