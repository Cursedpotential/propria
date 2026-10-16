// Milestone 1: imported-library view helpers. This file never changes source data.
import type { ImportedSource, SourceStatus } from "@/lib/imported-client";

export type LibraryFormat = "all" | ImportedSource["format"];
export type LibraryOwnerFilter = "all" | "recorded" | "unrecorded";

export interface LibraryFilters {
  text: string;
  status: "all" | SourceStatus;
  owner: LibraryOwnerFilter;
}

/**
 * Parse a shareable library category without treating unknown formats as classified.
 * Input: an optional URL value. Output: one supported server-side format or all.
 * Side effects: none. Use with the existing /api/imported/sources format filter.
 */
export function libraryFormat(value: string | null): LibraryFormat {
  return value === "SMS" || value === "Calls" || value === "Facebook" || value === "Other" ? value : "all";
}

/**
 * Filter only the imported rows returned by the current server-side page window.
 * Input: recorded import summaries and user-selected display filters.
 * Output: visible rows; no changes to identity, classification, or source order.
 * Side effects: none. Never report this as an exhaustive corpus-wide count.
 */
export function filterImportedLibrary(rows: readonly ImportedSource[], filters: LibraryFilters): ImportedSource[] {
  const text = filters.text.trim().toLocaleLowerCase();
  return rows.filter((source) => {
    const hasOwner = Boolean(source.owner || source.owner_name);
    if (filters.owner === "recorded" && !hasOwner) return false;
    if (filters.owner === "unrecorded" && hasOwner) return false;
    if (filters.status !== "all" && source.status !== filters.status) return false;
    if (!text) return true;
    return [
      source.file_name, source.casevault_key, source.device, source.owner_name,
      source.owner, source.format, source.status,
    ].some((value) => (value ?? "").toLocaleLowerCase().includes(text));
  });
}

/**
 * Find equal filename labels among fetched imported summaries.
 * Input: a selected recorded import and only the currently loaded source rows.
 * Output: candidates for manual lineage review, never confirmed duplicates.
 * Side effects: none. A name match does not establish equal bytes or derivation.
 */
export function possibleSameNameImports(source: ImportedSource, rows: readonly ImportedSource[]): ImportedSource[] {
  const name = source.file_name.trim().toLocaleLowerCase();
  if (!name) return [];
  return rows.filter((row) => row.id !== source.id && row.file_name.trim().toLocaleLowerCase() === name);
}
