// Byline: Claude Code · Opus 5.5 · 2026-10-01; editable identifiers 2026-10-02
// Case page client: the registry read (engine) plus what the Case Bible catalog
// holds per identifier, and the owner's edits. An identifier is added, fixed in
// place or deleted; every change is one registry.identity_change row (before,
// after, who, why). The browser never holds the authority.
import { ApiError } from "@/lib/api-client";
import type { MatterMode } from "@/lib/shared/types";

export interface CaseMatter {
  id: string;
  title: string;
  description: string | null;
  status: string;
  verification_state: string;
  created_by: string;
  updated_at: string;
}

export interface CaseCourtCase {
  id: string;
  matter_id: string;
  caption: string;
  docket_number: string | null;
  court_name: string | null;
  jurisdiction: string | null;
  case_type: string | null;
  presiding_judge: string | null;
  status: string;
  filed_on: string | null;
  closed_on: string | null;
  verification_state: string;
  updated_at: string;
}

export interface CaseIdentifier {
  id: string;
  entity_id: string;
  raw_value: string;
  kind: string;
  normalized: string;
  status: "confirmed" | "candidate" | "retired";
  period: string | null;
  basis: string | null;
  recorded_by: string;
  created_at: string;
}

export interface CasePerson {
  id: string;
  display_name: string;
  canonical_name: string | null;
  short_name: string | null;
  role_in_case: string | null;
  connection_to: string | null;
  relationship_type: string | null;
  is_minor: boolean;
  is_party: boolean;
  notes: string | null;
  verification_state: string;
  identifiers: CaseIdentifier[];
}

export interface CaseChange {
  id: string;
  subject_table: string;
  subject_id: string;
  before: Record<string, unknown>;
  after: Record<string, unknown>;
  change_reason: string;
  recorded_by: string;
  recorded_at: string;
}

export interface ProbataCount {
  key: string;
  source: string;
  events: number;
  first_at: string | null;
  last_at: string | null;
}

export interface CatalogCount {
  identifier: string;
  match_on: "counterparty_phone" | "sender";
  event_kind: string;
  events: number;
  conversations: number;
  first_at: string | null;
  last_at: string | null;
  sources: string[];
}

export interface CatalogUnknown {
  identifier: string;
  events: number;
  first_at: string | null;
  last_at: string | null;
  match_on: string[];
  kind: "phone" | "name";
}

export interface ProbataUnknown {
  normalized: string;
  raw_value: string;
  events: number;
  first_at: string | null;
  last_at: string | null;
}

export interface CaseIdentityView {
  mode: MatterMode;
  matter: CaseMatter | null;
  court_case: CaseCourtCase | null;
  people: CasePerson[];
  history: CaseChange[];
  probata_counts: ProbataCount[];
  probata_unknowns: ProbataUnknown[];
  dismissed: { normalized: string; raw_value: string; basis: string; recorded_by: string; recorded_at: string }[];
  count_store: string;
  catalog: {
    store: string;
    table: string;
    events: string;
    snapshot: string;
    available: boolean;
    error?: string;
    counts: CatalogCount[];
    unknowns: { phone: CatalogUnknown[]; name: CatalogUnknown[] };
  };
}

export interface CatalogEvent {
  dedup_key: string;
  event_ts_utc: string | null;
  event_kind: string;
  source_format: string;
  direction: string | null;
  sender: string | null;
  recipients: string | null;
  contact_name: string | null;
  counterparty_phone: string | null;
  conversation_title: string | null;
  body: string | null;
  n_sources: number | null;
}

export interface CatalogEventsPage {
  store: string;
  table: string;
  events: string;
  identifier: string;
  match_on: string;
  items: CatalogEvent[];
  has_more: boolean;
  next_before: string | null;
}

export interface CaseReceipt {
  ref: string;
  kind: string;
  recorded_at: string;
  replayed: boolean;
}

export interface IdentifierAdd {
  entity_id: string;
  raw_value: string;
  kind: string;
  status: string;
  period: string | null;
  basis: string;
  change_reason: string;
}

async function caseFetch<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, init);
  } catch {
    throw new ApiError("Network error — check your connection", 0);
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    const detail = typeof body.detail === "string" ? body.detail : `API error: ${response.status}`;
    throw new ApiError(detail, response.status, body);
  }
  return response.json();
}

function write<T>(path: string, body: unknown, idempotencyKey: string, mode: MatterMode): Promise<T> {
  return caseFetch<T>(`${path}${path.includes("?") ? "&" : "?"}mode=${encodeURIComponent(mode)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Idempotency-Key": idempotencyKey },
    body: JSON.stringify(body),
  });
}

export function getCaseIdentity(mode: MatterMode) {
  return caseFetch<CaseIdentityView>(`/api/case-identity?mode=${encodeURIComponent(mode)}`);
}

export function getCatalogEvents(identifier: string, matchOn: string, before?: string | null) {
  const query = new URLSearchParams({ identifier, match_on: matchOn, limit: "100" });
  if (before) query.set("before", before);
  return caseFetch<CatalogEventsPage>(`/api/case-identity/catalog-events?${query.toString()}`);
}

export function addIdentifier(body: IdentifierAdd, key: string, mode: MatterMode) {
  return write<CaseReceipt>("/api/case-identity/identifiers", body, key, mode);
}

export function editIdentifier(id: string, body: { fields: Record<string, string | null>; change_reason: string }, key: string, mode: MatterMode) {
  return write<CaseReceipt>(`/api/case-identity/identifiers/${encodeURIComponent(id)}`, body, key, mode);
}

export function deleteIdentifier(id: string, body: { change_reason: string }, key: string, mode: MatterMode) {
  return write<CaseReceipt>(`/api/case-identity/identifiers/${encodeURIComponent(id)}/delete`, body, key, mode);
}

export function editCaseHeader(
  mode: MatterMode,
  body: { target: "matter" | "court_case"; id: string; fields: Record<string, string | null>; change_reason: string; expected_updated_at?: string },
  key: string,
) {
  return write<CaseReceipt>("/api/case-identity/header", body, key, mode);
}

export function editPerson(personId: string, body: { fields: Record<string, string | null>; change_reason: string }, key: string, mode: MatterMode) {
  return write<CaseReceipt>(`/api/case-identity/people/${encodeURIComponent(personId)}`, body, key, mode);
}

export function addPerson(
  body: { display_name: string; short_name: string | null; role_in_case: string; connection_to: string; is_minor: boolean; notes: string | null; change_reason: string },
  key: string,
  mode: MatterMode,
) {
  return write<CaseReceipt>("/api/case-identity/people", body, key, mode);
}

export interface PlaceholderReceipt extends CaseReceipt {
  detail?: {
    created: number;
    already_carried: number;
    not_a_phone_number: number;
    linked: Record<string, number>;
    dry_run: boolean;
    entity_ids: Record<string, string>;
  };
}

/** One placeholder person per unidentified number; the engine links every NULL row for it. */
export function addPlaceholders(body: { numbers: string[]; change_reason: string; dry_run?: boolean }, key: string, mode: MatterMode) {
  return write<PlaceholderReceipt>("/api/case-identity/placeholders", body, key, mode);
}

/** Merge a placeholder into an existing person: identifiers and linked rows move, nothing is deleted. */
export function mergePerson(fromId: string, body: { into_id: string; change_reason: string }, key: string, mode: MatterMode) {
  return write<CaseReceipt>(`/api/case-identity/people/${encodeURIComponent(fromId)}/merge`, body, key, mode);
}

export function triageIdentifier(body: { raw_value: string; decision: "dismissed" | "reopened"; basis: string }, key: string, mode: MatterMode) {
  return write<CaseReceipt>("/api/case-identity/triage", body, key, mode);
}

export function newIdempotencyKey(prefix: string) {
  return `${prefix}-${crypto.randomUUID()}`;
}
