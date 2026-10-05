// Byline: Codex · GPT-6-Luna · 2026-10-04
// Byline: Codex · GPT-6 · 2026-10-04 — exact shared identities, separate receipts and unchanged-field preservation.

/** Prepare editable fields while preserving every personal value and leaving server-owned fields unchanged.
 * Inputs: full shared record. Outputs: editable object. Effects: none; omitted fields remain in the server's merge base.
 * Choose for editor initialization; this does not redact, delete or rewrite stored context.
 */
export function editableLibraryPatch(record) {
  return Object.fromEntries(Object.entries(record).filter(([key]) => !["id", "embedding", "validation", "validation_status", "published_at"].includes(key)));
}

/** Submit only changed fields so normalized display dates do not overwrite unchanged typed database values.
 * Inputs: edited JSON and the initial normalized record, or null for creation. Outputs: a partial patch preserving all edited personal fields.
 * Effects: none. Choose before proposal submission; field removal requires an explicit null value.
 */
export function changedLibraryPatch(edited, original) {
  const patch = editableLibraryPatch(edited);
  if (!original) return patch;
  const baseline = editableLibraryPatch(original);
  if (Object.keys(baseline).some(key => !(key in patch))) throw new Error("To clear a field, set its value to null; removing it leaves stored content unchanged.");
  const changed = Object.fromEntries(Object.entries(patch).filter(([key, value]) => JSON.stringify(value) !== JSON.stringify(baseline[key])));
  if (!Object.keys(changed).length) throw new Error("Change a field before saving a proposal.");
  return changed;
}

/** Builds the server-owned proposal request without editing local store state.
 * Inputs are a reference/source id, exact expected version or `absent`, full
 * record patch, resolved citation snapshots, and rationale. Output is the
 * `library_propose` argument object. Effects are none; cite versions must come
 * from exact case_record reads, not fields typed by the user. Prefer this helper
 * to direct case_put calls for reference/source changes.
 */
export function buildLibraryProposalArguments({ id, expectedVersion, patch, citations, rationale }) {
  if (!/^(reference|source):\S+$/.test(id)) throw new Error("Target id must begin with reference: or source:.");
  if (expectedVersion !== "absent" && !/^sha256:[0-9a-f]{64}$/i.test(expectedVersion)) {
    throw new Error("The target must have an exact case_record sha256 version or be marked absent.");
  }
  if (!patch || typeof patch !== "object" || Array.isArray(patch)) throw new Error("Record patch must be a JSON object.");
  if (!Array.isArray(citations) || citations.length === 0) throw new Error("Add at least one version-resolved source citation.");
  for (const citation of citations) {
    const selfCapture = id.startsWith("source:") && expectedVersion === "absent" &&
      citation.source_id === id && citation.source_version === "absent";
    if (!/^source:\S+$/.test(citation.source_id) ||
      (!selfCapture && !/^sha256:[0-9a-f]{64}$/i.test(citation.source_version))) {
      throw new Error("Each citation needs an exact source case_record version; only a new source may cite its own absent version.");
    }
    if (!citation.pinpoint?.trim() || !citation.claim?.trim()) throw new Error("Each citation needs a pinpoint and claim.");
  }
  if (!rationale?.trim()) throw new Error("Rationale is required.");
  return {
    id,
    expected_version: expectedVersion,
    patch,
    citations: citations.map(({ source_id, source_version, pinpoint, claim }) => ({
      source_id,
      source_version,
      pinpoint: pinpoint.trim(),
      claim: claim.trim(),
    })),
    rationale: rationale.trim(),
  };
}

/** Reads a structured MCP result without guessing from a success HTTP status.
 * Input is one invocation envelope; output is a JSON object or null. It has no
 * side effects and preserves tool errors for the caller to display.
 */
export function structuredToolResult(invocation) {
  if (!invocation || invocation.state !== "ok") return null;
  if (invocation.structured && typeof invocation.structured === "object" && !Array.isArray(invocation.structured)) {
    return invocation.structured;
  }
  for (const text of invocation.text ?? []) {
    try {
      const parsed = JSON.parse(text);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) return parsed;
    } catch {
      // Text-only, non-JSON tool output is deliberately not treated as approval.
    }
  }
  return null;
}

/** Validate the complete shared proposal identity returned by the server.
 * Input is library_proposal:<uuid>; output is the same exact identity.
 * Effects: none. Choose before reading the proposal or its separate validation receipt.
 */
export function libraryProposalRecordId(proposalId) {
  if (typeof proposalId !== "string" || !/^library_proposal:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(proposalId)) {
    throw new Error("Proposal result did not contain a valid UUID.");
  }
  return proposalId;
}

/** Test whether exact proposal and separate validation snapshots allow a publish attempt.
 * Inputs are the proposal tool result and two loaded shared case_record envelopes;
 * output is a boolean. It has no side effects and accepts only server-read status.
 */
export function canPublishReviewedProposal(proposal, proposalRecord, validationRecord) {
  const validationId = proposal?.proposal_id?.replace(/^library_proposal:/, "library_validation:");
  return Boolean(
    proposal?.proposal_id && proposalRecord?.id === proposal.proposal_id &&
    /^sha256:[0-9a-f]{64}$/i.test(proposalRecord.version ?? "") &&
    validationRecord?.id === validationId && /^sha256:[0-9a-f]{64}$/i.test(validationRecord.version ?? "") &&
    validationRecord.record?.proposal_id === proposal.proposal_id &&
    validationRecord.record?.proposed_hash === proposalRecord.record?.proposed_hash &&
    /^sha256:[0-9a-f]{64}$/i.test(proposalRecord.record?.proposed_hash ?? "") &&
    validationRecord.record?.status === "VERIFIED_PRIMARY" &&
    validationRecord.record?.currency_status === "cleared" && proposalRecord.record?.status !== "published"
  );
}

/** Returns the latest dispatch envelope visible on a proposal or exact record read.
 * Inputs are the proposal tool result and its case_record response; output is a
 * dispatch object or null. It has no side effects and prefers shared-record state.
 */
export function proposalDispatchState(proposal, proposalRecord) {
  const fromRecord = proposalRecord?.record?.dispatch;
  if (fromRecord && typeof fromRecord === "object" && !Array.isArray(fromRecord)) return fromRecord;
  const fromResult = proposal?.dispatch;
  return fromResult && typeof fromResult === "object" && !Array.isArray(fromResult) ? fromResult : null;
}

/** Enables retry only for a server-reported retryable queue failure.
 * Input is the latest dispatch envelope; output is a boolean. It performs no
 * side effects and never treats pending, running, or validation-blocked states as retryable.
 */
export function canRetryLibraryValidation(dispatch) {
  return dispatch?.state === "queue_failed" && dispatch.retryable === true;
}

/** Extracts a useful failure message from a hosted MCP invocation envelope.
 * Input is a tool invocation response; output is its first error text or a safe
 * fallback. It has no side effects and does not infer validation status.
 */
export function toolFailureMessage(invocation) {
  const first = invocation?.text?.find((value) => typeof value === "string" && value.trim());
  return first ?? `The hosted tool returned ${invocation?.state ?? "no result"}.`;
}

/** Classifies a visible failure into per-record conflict, validation, or transport state.
 * Input is an error message; output is a compact UI state key. It has no side effects
 * and never converts a failure into approval or a saved-record claim.
 */
export function libraryProposalFailureKind(message) {
  const value = String(message ?? "").toLowerCase();
  if (value.includes("conflict") || value.includes("expected version") || value.includes("version mismatch")) return "conflict";
  if (value.includes("validation") || value.includes("citation") || value.includes("currency")) return "validation";
  return "error";
}
