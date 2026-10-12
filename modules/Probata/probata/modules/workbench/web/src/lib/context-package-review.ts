// Byline: Codex · GPT-6 · 2026-10-12.
import type { ContextPackageItem, PackageEdge } from "@/lib/context-package-client";

/** Overlay the backend's current attributed correction while preserving the original item.
 * Input: source item plus actual correction head. Output: reviewed display copy.
 * Effects: none; raw fields stay on the caller's original object. Pick for grid,
 * editor, timeline and graph so reopening consistently shows saved review values.
 */
export function reviewedPackageItem(item: ContextPackageItem): ContextPackageItem {
  const correction = item.correction;
  if (!correction || typeof correction !== "object" || Array.isArray(correction)) return item;
  const current = correction as Record<string, unknown>;
  const fields = current.fields && typeof current.fields === "object" && !Array.isArray(current.fields) ? { ...item.fields, ...current.fields as Record<string, unknown> } : item.fields;
  const changedEdges = fields.edges ?? fields.relationships ?? fields.relations;
  return { ...item, original_fields: item.fields, fields,
    label: typeof fields.label === "string" ? fields.label : typeof fields.name === "string" ? fields.name : typeof fields.title === "string" ? fields.title : item.label,
    edges: Array.isArray(changedEdges) ? changedEdges.filter((edge): edge is PackageEdge => Boolean(edge) && typeof edge === "object" && !Array.isArray(edge)) : item.edges,
    source_refs: Array.isArray(current.source_refs) ? current.source_refs : item.source_refs,
    review_state: typeof current.review_state === "string" ? current.review_state : item.review_state,
  };
}
