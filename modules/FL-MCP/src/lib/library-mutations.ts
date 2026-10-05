// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04

const SERVER_FIELDS = new Set(["id", "embedding"]);
const LIBRARY_FIELDS = new Set(["validation", "validation_status", "published_at"]);
const PERSONAL_FIELDS = new Set(["kind"]);
const SHA256_VERSION = /^sha256:[a-f0-9]{64}$/i;
const PROPOSAL_ID = /^library_proposal:([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$/i;

/** Returns a JSON-editable snapshot with only server-owned fields omitted.
 * Inputs are the exact shared record and a library/personal mode; output is a detached object.
 * It does not mutate or persist the source record; use it to initialize the editor safely.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function editableRecord(record: Record<string, unknown>, mode: "library" | "personal"): Record<string, unknown> {
  return Object.fromEntries(Object.entries(record).filter(([key]) =>
    !SERVER_FIELDS.has(key) && (mode !== "library" || !LIBRARY_FIELDS.has(key)) && (mode !== "personal" || !PERSONAL_FIELDS.has(key))));
}

/** Computes only changed top-level fields and rejects removed keys unless they are explicitly set to null.
 * Inputs are the original editable base, parsed edit, and record mode; output is a merge patch.
 * Omitted existing keys throw a validation error; nested values remain complete; it performs no I/O.
 * Prefer this over sending the whole displayed record to case_put or library_propose.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function changedRecordPatch(
  base: Record<string, unknown>,
  edited: Record<string, unknown>,
  mode: "library" | "personal",
): Record<string, unknown> {
  const patch: Record<string, unknown> = {};
  for (const key of Object.keys(base)) {
    if (SERVER_FIELDS.has(key) || mode === "library" && LIBRARY_FIELDS.has(key) || mode === "personal" && PERSONAL_FIELDS.has(key)) continue;
    if (!Object.hasOwn(edited, key)) throw new Error(`Existing field "${key}" was removed; use explicit null to clear it.`);
  }
  for (const [key, value] of Object.entries(edited)) {
    if (SERVER_FIELDS.has(key) || mode === "library" && LIBRARY_FIELDS.has(key) || mode === "personal" && PERSONAL_FIELDS.has(key)) continue;
    if (!Object.hasOwn(base, key) || stableJson(base[key]) !== stableJson(value)) patch[key] = value;
  }
  return patch;
}

/** Produces deterministic JSON for equality checks without changing user payloads.
 * Input is any JSON value; output is canonical JSON text. It has no side effects and only sorts object keys.
 * Use for detecting real edits despite whitespace or property-order changes in the textarea.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function stableJson(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stableJson).join(",")}]`;
  if (value && typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>).sort(([left], [right]) => left.localeCompare(right));
    return `{${entries.map(([key, item]) => `${JSON.stringify(key)}:${stableJson(item)}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

/** Builds the exact separate validation case_record id from a full proposal id.
 * Input is the full `library_proposal:UUID`; output is `library_validation:UUID`.
 * It performs no I/O and refuses bare/doubly-prefixed ids rather than guessing.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function validationRecordId(proposalId: string): string {
  const match = PROPOSAL_ID.exec(proposalId);
  if (!match) throw new Error("The proposal result must contain its full library_proposal:UUID id.");
  return `library_validation:${match[1]}`;
}

/** Allows publish only when the exact proposal and separate validator receipt agree.
 * Inputs are the returned proposal, its exact case_record, and its exact validation record; output is a boolean.
 * It performs no I/O and never treats proposal flags or user input as validation evidence.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function canPublishLibraryProposal(
  proposal: Record<string, unknown> | null,
  proposalRecord: Record<string, unknown> | null,
  validationRecord: Record<string, unknown> | null,
): boolean {
  if (!proposal || !proposalRecord || !validationRecord || typeof proposal.proposal_id !== "string") return false;
  let expectedReceiptId: string;
  try {
    expectedReceiptId = validationRecordId(proposal.proposal_id);
  } catch {
    return false;
  }
  const proposalBody = proposalRecord.record as Record<string, unknown> | undefined;
  const receipt = validationRecord.record as Record<string, unknown> | undefined;
  const proposedHash = proposalBody?.proposed_hash;
  return proposal.status === "pending_validation" && proposalRecord.id === proposal.proposal_id &&
    proposalBody?.status === "pending_validation" &&
    SHA256_VERSION.test(String(proposalRecord.version ?? "")) &&
    proposedHash === proposal.proposed_hash && SHA256_VERSION.test(String(proposedHash ?? "")) &&
    validationRecord.id === expectedReceiptId && SHA256_VERSION.test(String(validationRecord.version ?? "")) &&
    receipt?.proposal_id === proposal.proposal_id && receipt.proposed_hash === proposedHash &&
    receipt.status === "VERIFIED_PRIMARY" && receipt.currency_status === "cleared";
}

/** Enables validation retry only after a server-reported retryable dispatch failure.
 * Input is a server dispatch record; output is a boolean. It performs no writes and excludes pending/running states.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function canRetryLibraryValidation(dispatch: unknown): boolean {
  if (!dispatch || typeof dispatch !== "object") return false;
  const state = dispatch as Record<string, unknown>;
  return state.state === "queue_failed" && state.retryable === true;
}

/** Extracts queue status from the latest exact proposal record, falling back to the save response.
 * Inputs are proposal result and optional proposal record; output is a dispatch object or null.
 * It performs no I/O and does not infer validation completion from dispatch state.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function proposalDispatch(proposal: Record<string, unknown> | null, proposalRecord: Record<string, unknown> | null): Record<string, unknown> | null {
  const record = proposalRecord?.record as Record<string, unknown> | undefined;
  const candidate = record?.dispatch ?? proposal?.dispatch;
  return candidate && typeof candidate === "object" && !Array.isArray(candidate) ? candidate as Record<string, unknown> : null;
}

/** Reads structured data only from a successful sidecar tool envelope.
 * Input is the returned state/text/structured envelope; output is a record or null.
 * It has no side effects and ignores textual success claims when structured data is absent.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function hostedStructuredResult(envelope: unknown): Record<string, unknown> | null {
  if (!envelope || typeof envelope !== "object") return null;
  const result = envelope as Record<string, unknown>;
  const structured = result.structured;
  return result.state === "ok" && structured && typeof structured === "object" && !Array.isArray(structured)
    ? structured as Record<string, unknown>
    : null;
}
