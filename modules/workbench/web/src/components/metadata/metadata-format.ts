// Byline: Claude Code · Opus 5.5 · 2026-09-26 (metadata screen helpers; kept out of component files)
import type { MetadataCorrection } from "@/lib/review-overlays-client";

export interface FlatField {
  path: string;
  value: unknown;
}

/** Dotted-path fields of a native JSON document, in document order, bounded. */
export function flattenFields(value: unknown, limit = 2000): { fields: FlatField[]; truncated: boolean } {
  const fields: FlatField[] = [];
  const stack: Array<[string, unknown]> = [["", value]];
  while (stack.length) {
    const [path, item] = stack.pop()!;
    if (item && typeof item === "object" && !Array.isArray(item) && Object.keys(item).length) {
      const entries = Object.entries(item as Record<string, unknown>);
      for (let index = entries.length - 1; index >= 0; index -= 1) {
        const [key, child] = entries[index];
        stack.push([path ? `${path}.${key}` : key, child]);
      }
    } else if (Array.isArray(item) && item.length) {
      for (let index = item.length - 1; index >= 0; index -= 1) stack.push([`${path}[${index}]`, item[index]]);
    } else {
      if (fields.length >= limit) return { fields, truncated: true };
      fields.push({ path: path || "(document)", value: item });
    }
  }
  return { fields, truncated: false };
}

export function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value) && value.length === 0) return "[]";
  if (typeof value === "object" && Object.keys(value as object).length === 0) return "{}";
  return JSON.stringify(value);
}

export function formatBytes(value: number | null | undefined): string {
  if (value === null || value === undefined) return "size not recorded";
  if (value < 1024) return `${value.toLocaleString()} bytes`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  if (value < 1024 * 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  return `${(value / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function formatWhen(value: string | null | undefined): string {
  if (!value) return "not recorded";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

/** Every correction revision per field key, newest first (the engine sends them that way). */
export function correctionsByField(corrections: MetadataCorrection[]): Map<string, MetadataCorrection[]> {
  const byField = new Map<string, MetadataCorrection[]>();
  for (const correction of corrections) {
    const list = byField.get(correction.field_key) ?? [];
    list.push(correction);
    byField.set(correction.field_key, list);
  }
  for (const list of byField.values()) list.sort((left, right) => right.revision - left.revision);
  return byField;
}

/** Keep the observed value's JSON type when the owner's text clearly matches it. */
export function parseCorrectedValue(draft: string, observed: unknown): unknown {
  const text = draft.trim();
  if (typeof observed === "number" && text !== "" && Number.isFinite(Number(text))) return Number(text);
  if (typeof observed === "boolean" && (text === "true" || text === "false")) return text === "true";
  return draft;
}

export const METADATA_CLASS_LABEL: Record<string, string> = {
  filesystem: "Storage and file system",
  embedded: "Embedded in the file",
  container: "Container structure",
  media_tool: "Media tool reading",
  record_native: "Per-record native",
};

export const SIDECAR_KIND_LABEL: Record<string, string> = {
  takeout_json: "Google Takeout sidecar",
  json: "JSON sidecar",
  xmp: "XMP sidecar",
  apple_aae: "Apple edit record (.aae)",
  owner_sidecar_md: "Owner analysis (.sidecar.md)",
  owner_extraction_md: "Owner extraction (.EXTRACTION.md)",
  companion: "Same-name companion file",
};
