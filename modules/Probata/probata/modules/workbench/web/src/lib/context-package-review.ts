// Byline: Codex · GPT-6 · 2026-10-12.
import type { ContextPackageItem, PackageCapability, PackageEdge, PackageReviewInput } from "@/lib/context-package-client";

/** Select candidate assertion fields without allowing edits to its persisted envelope.
 * Inputs: item and actual edit descriptor. Output: editable assertion projection or all metadata.
 * Effects: none. Pick for canonical working candidates; full original fields remain inspectable.
 */
export function packageEditorFields(item: ContextPackageItem, capability?: PackageCapability): Record<string, unknown> {
  if (capability?.scope !== "canonical_working_candidate") return item.fields;
  const attrs = item.fields.attrs as Record<string, unknown> | undefined;
  const candidate = attrs?.candidate;
  const values = candidate && typeof candidate === "object" && !Array.isArray(candidate) ? candidate as Record<string, unknown> : {};
  const allowed = Array.isArray(capability.editable_fields) ? capability.editable_fields.filter((key): key is string => typeof key === "string") : [];
  return Object.fromEntries(allowed.map((key) => [key, values[key] ?? null]));
}

/** Build a review write preserving server-issued revision, candidate hash and stable review identity.
 * Inputs: exact item, advertised descriptor, patch, reason and operation extras. Output: guarded request.
 * Effects: none. Pick for all package review writes; canonical envelopes cannot be patched as metadata.
 */
export function packageReviewInput(item: ContextPackageItem, descriptor: PackageCapability, fields: Record<string, unknown> | undefined, reason: string, extras: Record<string, unknown> = {}): PackageReviewInput {
  if (descriptor.scope === "canonical_working_candidate" && fields) {
    const allowed = Array.isArray(descriptor.editable_fields) ? descriptor.editable_fields : [];
    const invalid = Object.keys(fields).find((key) => !allowed.includes(key));
    if (invalid) throw new Error(`${invalid} is not an editable candidate assertion field.`);
  }
  const defaults = descriptor.body && typeof descriptor.body === "object" && !Array.isArray(descriptor.body) ? descriptor.body : {};
  return { item_id: item.id, revision: item.revision, source_refs: item.source_refs, ...extras, ...defaults, operation: descriptor.operation, reason, ...(fields ? { fields } : {}) };
}

/** Allow only existing app/API links; inert source text never becomes an executable URL.
 * Input: recorded URL. Output: local source link or null. Effects: none.
 * Pick before rendering source links or embedded media from extracted fields.
 */
export function packageSourceHref(value: unknown): string | null {
  if (typeof value !== "string" || !/^\/(?:read(?:\?|$)|sources(?:\?|$)|api\/(?:context|proffer|imported)\/)/.test(value)) return null;
  const url = new URL(value, "https://context.invalid");
  return url.origin === "https://context.invalid" ? `${url.pathname}${url.search}${url.hash}` : null;
}

/** Read complete text blocks from native nested objects without truncating document content.
 * Input: bounded item fields. Output: exact text and original field paths. Effects: none.
 * Pick for source-native records/documents instead of requiring a normalized body column.
 */
export function packageTextBlocks(fields: Record<string, unknown>): Array<{ path: string; text: string }> {
  const blocks: Array<{ path: string; text: string }> = [];
  const stack: Array<{ path: string; value: unknown }> = [{ path: "", value: fields }];
  while (stack.length) {
    const { path, value } = stack.pop()!;
    if (typeof value === "string" && /(?:^|\.)(text|body|raw_body|message_body|full_text|content|markdown|document_text|snippet|description|statement|summary|extracted_text|ocr_text|transcript|transcription)$/.test(path)) blocks.push({ path, text: value });
    else if (Array.isArray(value)) for (let index = value.length - 1; index >= 0; index -= 1) stack.push({ path: `${path}[${index}]`, value: value[index] });
    else if (value && typeof value === "object") for (const [key, child] of Object.entries(value).reverse()) stack.push({ path: path ? `${path}.${key}` : key, value: child });
  }
  return blocks;
}

/** Overlay the backend's current attributed correction while preserving the original item.
 * Input: source item plus actual correction head. Output: reviewed display copy.
 * Effects: none; raw fields stay on the caller's original object. Pick for grid,
 * editor, timeline and graph so reopening consistently shows saved review values.
 */
export function reviewedPackageItem(item: ContextPackageItem): ContextPackageItem {
  const correction = item.correction;
  const current = correction && typeof correction === "object" && !Array.isArray(correction) ? correction as Record<string, unknown> : {};
  const fields = current.fields && typeof current.fields === "object" && !Array.isArray(current.fields) ? { ...item.fields, ...current.fields as Record<string, unknown> } : item.fields;
  const changedEdges = fields.edges ?? fields.relationships ?? fields.relations;
  const edges = Array.isArray(changedEdges) ? changedEdges.filter((edge): edge is PackageEdge => Boolean(edge) && typeof edge === "object" && !Array.isArray(edge)) : item.edges;
  const directEdge = typeof (fields.from_node_id ?? fields.from) === "string" && typeof (fields.to_node_id ?? fields.to) === "string" && typeof (fields.kind ?? fields.relation) === "string";
  return { ...item, ...(item.correction ? { original_fields: item.fields } : {}), fields,
    label: typeof fields.label === "string" ? fields.label : typeof fields.name === "string" ? fields.name : typeof fields.title === "string" ? fields.title : item.label,
    edges: directEdge && !edges.length ? [fields as PackageEdge] : edges,
    source_refs: Array.isArray(current.source_refs) ? current.source_refs : item.source_refs,
    review_state: typeof current.review_state === "string" ? current.review_state : item.review_state,
  };
}
