// Byline: Codex · GPT-6 · 2026-10-12.
import { ApiError } from "@/lib/api-client";

/** Server-recorded placement; paths and classifications never come from filename guesses. */
export interface PackagePlacement {
  domain?: string | null;
  category?: string | null;
  path?: string | string[] | null;
  basis?: unknown;
  unresolved?: boolean;
  inherited_from?: string | null;
  [key: string]: unknown;
}

/** A real persisted Case Vault category returned by the category listing. */
export interface ContextCategory {
  id: string;
  label: string;
  domain?: string | null;
  category?: string | null;
  path?: string | string[] | null;
  classification_basis?: unknown;
  count: number;
  unresolved?: boolean;
}

/** Package identity stays intact inside its persisted domain/category. */
export interface ContextPackageSummary {
  handle: string | null;
  package_ref: string;
  source_ref: string;
  source_version_ref: string;
  source_type: string;
  label: string;
  revision: string | number;
  domain?: string | null;
  category?: string | null;
  path?: string | string[] | null;
  classification_basis?: unknown;
  placement?: PackagePlacement;
  available?: boolean;
  reason?: string;
}

/** Opaque cursor page; every fetch loads one bounded server page. */
export interface ContextPage<T> { items: T[]; next_cursor?: string | null }

/** Source coordinates are preserved as returned, including unrecognized native fields. */
export type PackageSourceRef = string | Record<string, unknown>;

/** Actual edge inventory, not a locally inferred relationship taxonomy. */
export interface PackageEdge {
  id?: string;
  from?: string;
  to?: string;
  relation?: string;
  target?: string;
  direction?: string;
  source_refs?: PackageSourceRef[];
  basis?: unknown;
  [key: string]: unknown;
}

/** Backend capability descriptors determine which writes the operator can invoke. */
export interface PackageCapability {
  operation: string;
  available?: boolean;
  reason?: string;
  label?: string;
  path?: string;
  method?: string;
  [key: string]: unknown;
}

/** One source-linked extraction, complete document, metadata record or reviewable finding. */
export interface ContextPackageItem {
  id: string;
  kind: string;
  label: string;
  fields: Record<string, unknown>;
  source_refs: PackageSourceRef[];
  edges: PackageEdge[];
  review_state: string;
  revision: string | number;
  capabilities: Array<string | PackageCapability>;
  placement?: PackagePlacement;
  [key: string]: unknown;
}

/** Dynamic section inventory includes unavailable sections and their actual reasons. */
export interface ContextPackageSection { id: string; label: string; count: number | null; available: boolean; reason?: string | null }

/** A bounded package snapshot, independent of legacy normalized-message identifiers. */
export interface ContextPackageSnapshot {
  package_ref: string;
  source_ref: string;
  source_version_ref: string;
  source_type: string;
  revision: string | number;
  metadata: Record<string, unknown>;
  placement?: PackagePlacement;
  domain?: string | null;
  category?: string | null;
  path?: string | string[] | null;
  classification_basis?: unknown;
  sections: ContextPackageSection[];
  items: ContextPackageItem[];
  next_cursor?: string | null;
  capabilities?: Array<string | PackageCapability>;
  [key: string]: unknown;
}

/** A revision-guarded human correction/decision; source bytes remain outside this request. */
export interface PackageReviewInput {
  operation: string;
  item_id?: string;
  revision: string | number;
  fields?: Record<string, unknown>;
  reason: string;
  [key: string]: unknown;
}

const API_BASE = import.meta.env.VITE_API_URL || "";

/** Fetch typed package data through the same-origin BFF.
 * Inputs: API path and abortable request. Output: server JSON. Effects: HTTP only.
 * Pick for context package routes; errors retain status and never imply a saved correction.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, { credentials: "same-origin", ...init });
  if (!response.ok) {
    const body = await response.json().catch(() => ({})) as { detail?: unknown };
    throw new ApiError(typeof body.detail === "string" ? body.detail : `Context review request failed (${response.status})`, response.status);
  }
  return response.json() as Promise<T>;
}

/** Read one persisted category page for the default category-header landing.
 * Inputs: opaque cursor and abort signal. Output: actual category identities and counts.
 * Effects: bounded GET. Pick before package selection; never infer a taxonomy locally.
 */
export function getContextCategories(cursor: string, signal?: AbortSignal): Promise<ContextPage<ContextCategory>> {
  const query = new URLSearchParams();
  if (cursor) query.set("cursor", cursor);
  return request(`/api/context/categories?${query}`, { signal });
}

/** Read one package page within a recorded category/domain.
 * Inputs: exact category identity, domain, cursor and signal. Output: whole-package summaries.
 * Effects: bounded GET. Pick beneath a category header; no physical source movement.
 */
export function getContextPackages(category: string, domain: string, cursor: string, signal?: AbortSignal): Promise<ContextPage<ContextPackageSummary>> {
  const query = new URLSearchParams({ category });
  if (domain) query.set("domain", domain);
  if (cursor) query.set("cursor", cursor);
  return request(`/api/context/packages?${query}`, { signal });
}

/** Read one section page of the exact selected extracted package.
 * Inputs: native handle, section, cursor and signal. Output: inventory and source-linked items.
 * Effects: bounded GET. Pick instead of normalizing a native source into legacy messages.
 */
export function getContextPackage(handle: string, section: string, cursor: string, signal?: AbortSignal): Promise<ContextPackageSnapshot> {
  const query = new URLSearchParams();
  if (section) query.set("section", section);
  if (cursor) query.set("cursor", cursor);
  return request(`/api/context/packages/${encodeURIComponent(handle)}?${query}`, { signal });
}

/** Find an actually advertised operation, excluding explicitly unavailable capabilities.
 * Inputs: server capabilities and operation name. Output: enabled descriptor or undefined.
 * Effects: none. Pick before rendering any write button; absence means no operation.
 */
export function packageCapability(capabilities: Array<string | PackageCapability> | undefined, operation: string): PackageCapability | undefined {
  const found = capabilities?.find((entry) => typeof entry === "string" ? entry === operation : entry.operation === operation && entry.available !== false);
  return typeof found === "string" ? { operation: found } : found;
}

/** Append a human review action using the package review writer and exact revision.
 * Inputs: native handle, human input and one stable idempotency key. Output: durable receipt.
 * Effects: explicit POST. Pick only for a backend-advertised operation; never mutate source bytes.
 */
export function reviewContextPackage(handle: string, input: PackageReviewInput, idempotencyKey: string): Promise<Record<string, unknown>> {
  return request(`/api/context/packages/${encodeURIComponent(handle)}/review`, {
    method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": idempotencyKey }, body: JSON.stringify(input),
  });
}

/** Invoke an explicitly advertised extraction retry at its existing same-origin API path.
 * Inputs: server capability, operator-selected fields and stable action key. Output: actual receipt.
 * Effects: clicked POST only. Pick for model retry when backend provides a working path/body;
 * unavailable or cross-origin capabilities never become executable controls.
 */
export function retryContextPackage(capability: PackageCapability, fields: Record<string, unknown>, key: string): Promise<Record<string, unknown>> {
  if (capability.available === false || capability.method?.toUpperCase() !== "POST" || !capability.path?.startsWith("/api/") || capability.path.includes("\\")) throw new Error("No usable extraction retry capability was returned.");
  const target = new URL(capability.path, "https://context.invalid");
  if (target.origin !== "https://context.invalid" || !target.pathname.startsWith("/api/")) throw new Error("Invalid extraction retry API path.");
  const defaults = capability.body && typeof capability.body === "object" && !Array.isArray(capability.body) ? capability.body : {};
  return request(`${target.pathname}${target.search}`, { method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key }, body: JSON.stringify({ ...defaults, ...fields }) });
}
