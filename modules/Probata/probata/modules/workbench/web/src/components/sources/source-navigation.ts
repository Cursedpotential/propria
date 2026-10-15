import type { ProfferSourceObject } from "@/lib/shared/types";

export interface SourceNavigationState {
  selectedRef: string | null;
  selectedFolder: string | null;
  selectedFolders?: string[];
  checkedRefs: string[];
  selectedObject: ProfferSourceObject | null;
  providerVersion: string | null;
  query: string;
  appliedFilter: string;
  scrollRow: number;
  aiExportSelection?: string | null;
  aiProvider?: "" | "chatgpt" | "claude" | "gemini_markdown";
}

/** Resolve a linked filename to one exact object key inside its recorded folder.
 * Inputs: folder prefix and basename or full object key. Output: exact key or null.
 * Effects: none; choose for source links, never for explicit location-wide name search.
 */
export function linkedSourceKey(prefix: string, file: string): string | null {
  if (!file) return null;
  const folder = prefix && !prefix.endsWith("/") ? `${prefix}/` : prefix;
  return file.includes("/") ? (file.startsWith(folder) ? file : null) : `${folder}${file}`;
}

/** Restore Sources navigation metadata within this browser tab and exact route scope.
 * Input: mode/root/prefix route key. Output: validated metadata or null.
 * Effects: reads session storage; choose for Back navigation without storing source bodies.
 */
export function restoreSourceNavigation(key: string): SourceNavigationState | null {
  try {
    const value = JSON.parse(sessionStorage.getItem(`sources-navigation-v1:${key}`) ?? "null");
    if (!value || !Array.isArray(value.checkedRefs) || !value.checkedRefs.every((ref: unknown) => typeof ref === "string")
        || typeof value.query !== "string" || typeof value.appliedFilter !== "string"
        || !Number.isInteger(value.scrollRow) || value.scrollRow < 0
        || (value.selectedRef !== null && typeof value.selectedRef !== "string")
        || (value.selectedFolder !== null && typeof value.selectedFolder !== "string")
        || (value.providerVersion !== null && typeof value.providerVersion !== "string")) return null;
    if (value.selectedFolders !== undefined && (!Array.isArray(value.selectedFolders)
        || !value.selectedFolders.every((folder: unknown) => typeof folder === "string" && folder.length > 0))) return null;
    if ((value.aiExportSelection !== undefined && value.aiExportSelection !== null && typeof value.aiExportSelection !== "string")
        || (value.aiProvider !== undefined && !["", "chatgpt", "claude", "gemini_markdown"].includes(value.aiProvider))) return null;
    const object = value.selectedObject;
    if (object && (object.source_ref !== value.selectedRef || typeof object.key !== "string"
        || typeof object.name !== "string" || typeof object.byte_length !== "number" || object.byte_length < 0
        || (object.etag !== null && object.etag !== undefined && typeof object.etag !== "string"))) return null;
    return value as SourceNavigationState;
  } catch { return null; }
}

/** Save only route selection, identity pins and scroll position for tab-local recovery.
 * Inputs: scoped key and source metadata. Output: none. Effects: session storage;
 * choose on Sources navigation changes, never as durable evidence or processing authority.
 */
export function saveSourceNavigation(key: string, value: SourceNavigationState): void {
  try { sessionStorage.setItem(`sources-navigation-v1:${key}`, JSON.stringify(value)); } catch { /* Navigation remains usable when storage is unavailable. */ }
}

/** Carry the inspected provider version through the engine's existing source locator contract.
 * Inputs: original locator and optional provider version. Output: exact versionId URI.
 * Effects: none; choose for legacy starts so inspection cannot silently dispatch latest.
 */
export function pinnedSourceRef(sourceRef: string, providerVersion: string | null | undefined): string {
  if (!providerVersion) return sourceRef;
  const parsed = new URL(sourceRef);
  if (parsed.search || parsed.hash || parsed.username || parsed.password) throw new Error("The selected source locator cannot carry an unambiguous version pin.");
  parsed.searchParams.set("versionId", providerVersion);
  return parsed.toString();
}
