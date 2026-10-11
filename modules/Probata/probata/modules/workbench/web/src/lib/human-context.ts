import type { MatterMode, ProfferPreviewMessagesResponse, ProfferPreviewParticipant } from "@/lib/shared/types";
import type { CasePerson } from "@/lib/case-identity-client";

/** Describe one context start using source coordinates and explicit phone/export attribution.
 * Inputs: catalog or acquisition reference and registry identities. Outputs: POST fields.
 * Effects: none. Choose for human SMS Process, before legacy handler selection.
 */
export interface HumanContextSourceRequest {
  contract_version: "context-v1";
  source_ref: string;
  provider_version_id?: string;
  package_ref?: string;
  source_kind: "sms";
  declared_format: string;
  matter_id?: string;
  court_case_id?: string;
  owner_person_ref?: string;
  perspective_person_ref?: string;
  perspective?: string;
  byte_length: number;
  etag?: string;
}

/** Resolve native export attribution from an explicit current choice without requiring a case.
 * Inputs: confirmed registry people, current source scope and selected person. Outputs: actual attribution references.
 * Effects: throws if no confirmed current choice exists. Choose for native SMS; optional owner is included only
 * when the registry identifies exactly one confirmed owner, with no synthetic or global default.
 */
export function humanContextAttribution(people: CasePerson[], scope: string, choice: { scope: string; personId: string } | null) {
  const person = choice?.scope === scope ? people.find((entry) => entry.id === choice.personId && entry.verification_state === "confirmed") : undefined;
  if (!person) throw new Error("Choose whose phone/export these selected messages came from.");
  const owners = people.filter((entry) => entry.role_in_case === "user" && entry.verification_state === "confirmed");
  return {
    ...(owners.length === 1 ? { owner_person_ref: owners[0].id } : {}),
    perspective_person_ref: person.id,
    perspective: "source-device",
  };
}

/** Carry real execution and registration references without source bodies.
 * Inputs: Go start/status responses. Outputs: bounded tracking state. Effects: none.
 * Choose for Process tracking; only source_version_ref can produce a human handle.
 */
export interface HumanContextProgress {
  workflow_id: string;
  run_id?: string;
  source_version_ref?: string;
  status?: string;
  reason?: string;
}

/** Preserve the native bundle envelope returned by the human content endpoint.
 * Inputs: DuckDB-native record fields. Outputs: unchanged native values and status.
 * Effects: none. Choose instead of AI normalization or PostgreSQL message rows.
 */
export interface HumanContextRecord {
  ordinal: number;
  status: string;
  native: Record<string, unknown>;
}

/** Carry a bounded native page and its server-owned continuation.
 * Inputs: content response. Outputs: records, offset and total. Effects: none.
 * Choose for explicit page navigation; never accumulate a whole export in memory.
 */
export interface HumanContextPage {
  records: HumanContextRecord[];
  next_offset?: number | null;
  total: number;
}

/** Describe actual human extraction/publication stages and their manifest references.
 * Inputs: metadata snapshot. Outputs: source identity and stage receipts. Effects: none.
 * Choose for the native reader without synthesizing legacy completeness receipts.
 */
export interface HumanContextSnapshot {
  preview_handle: string;
  source_version_ref: string;
  source_ref: string;
  workflow_id: string;
  manifest_ref?: string;
  stages: Array<{ stage: string; status: string; manifest_ref: string; records: number; pending: number; published: number; verified: number }>;
}

/** Recognize supported SMS selections using the existing format choice and backup filename convention.
 * Inputs: selected filename and declared format. Output: SMS routing decision. Effects: none.
 * Choose at Process dispatch; calls, AI and other legacy sources keep their existing route.
 */
export function isSMSContextSource(name: string, format: string): boolean {
  return format === "smsbackuprestore_xml" || (format === "xml" && /^sms(?:[-_ .]|backup)/i.test(name));
}

/** Construct a native reader handle only from a real returned source-version UUID.
 * Input: registered source_version_ref. Output: human handle or null. Effects: none.
 * Choose after registration; workflow IDs and client-generated IDs are never source handles.
 */
export function humanPreviewHandle(sourceVersionRef: string | undefined): string | null {
  return sourceVersionRef && /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i.test(sourceVersionRef)
    ? `human-${sourceVersionRef}` : null;
}

/** Recognize the native reader branch without reinterpreting legacy opaque handles.
 * Input: route handle. Output: native-handle flag. Effects: none.
 * Choose before the existing preview client starts legacy polling or event streams.
 */
export function isHumanPreviewHandle(handle: string): boolean {
  return humanPreviewHandle(handle.startsWith("human-") ? handle.slice(6) : undefined) === handle;
}

/** Present a native page through the existing SBV message-browser row contract.
 * Inputs: exact native page, real handle and display mode. Outputs: presentation rows with original envelopes attached.
 * Effects: none. Choose for human bundles; no AI normalization, identity resolution, hashing or retained-media lookup occurs.
 */
export function humanMessagePage(page: HumanContextPage, handle: string, mode: MatterMode): ProfferPreviewMessagesResponse {
  const directory = new Map<string, ProfferPreviewParticipant>();
  const messages = page.records.map((record) => {
    const native = record.native;
    const addresses = Array.isArray(native.participants) ? native.participants.flatMap((entry) => {
      const value = typeof entry === "string" ? entry : entry && typeof entry === "object" && "address" in entry ? entry.address : null;
      return typeof value === "string" && value ? [value] : [];
    }) : [];
    const sender = typeof native.sender_address === "string" ? native.sender_address : null;
    for (const address of [...addresses, ...(sender ? [sender] : [])]) {
      directory.set(address, { participant_id: address, display_name: address, canonical_address: address });
    }
    return {
      message_id: `${handle}:${record.ordinal}`,
      ordinal: record.ordinal,
      sent_at: typeof native.occurred_at === "string" ? native.occurred_at : null,
      sender_participant_id: sender,
      body: typeof native.body === "string" ? native.body : "",
      participant_ids: addresses,
      attachments: [],
      source_locator_ref: typeof native.source_locator === "string" ? native.source_locator : JSON.stringify(native.source_locator ?? null),
      native_record: record,
    };
  });
  return { preview_handle: handle, matter_mode: mode, participants: [...directory.values()], messages,
    next_cursor: (page.next_offset === null || page.next_offset === undefined) ? null : String(page.next_offset), total_messages: page.total, total_matches: page.total };
}
