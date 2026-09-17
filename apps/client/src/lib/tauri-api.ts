// Thin re-export — the real implementation lives in ./tauri-api/ domain modules.
export { TauriAPI } from './tauri-api/index';
export * from './tauri-api-types';
export type {
  IntakeCatalogLookup,
  IntakeCatalogOccurrence,
  IntakeFileMetadata,
} from './tauri-api/intake-engine';
