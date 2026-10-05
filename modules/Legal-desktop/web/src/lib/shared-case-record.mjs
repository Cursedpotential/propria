// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04.

const PERSONAL_TABLES = new Set([
  "person", "child", "order", "hearing", "deadline", "event", "message",
  "exhibit", "factor", "source", "note", "court", "court_event", "filing",
  "draft", "memo", "evidence_log", "eval", "case_status",
]);
const SHA256_VERSION = /^sha256:[0-9a-f]{64}$/;
const SERVER_FIELDS = new Set(["id", "embedding"]);

/** Canonicalizes JSON values for changed-field comparison without treating key order as an edit.
 * Inputs: JSON-compatible value. Outputs: stable JSON text. Effects: none. Use before building a patch.
 */
function canonicalJson(value) {
  if (Array.isArray(value)) return `[${value.map(canonicalJson).join(",")}]`;
  if (value && typeof value === "object") {
    return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJson(value[key])}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

/** Builds the guarded case_put arguments from the full editor value and exact shared snapshot.
 * Inputs: table, optional full record ID, exact SHA-256 version or `absent`, edited object, and prior object.
 * Outputs: case_put arguments containing only changed data for updates. Effects: none; source documents retain their personal-write kind guard.
 * Choose for personal case data; source/reference legal authorities belong to LibraryProposalEditor.
 */
export function buildPersonalCasePutArguments({ table, id, expectedVersion, edited, original }) {
  if (!PERSONAL_TABLES.has(table)) throw new Error("This editor only writes personal case tables; legal references use the proposal workflow.");
  if (!edited || typeof edited !== "object" || Array.isArray(edited)) throw new Error("Record data must be a JSON object.");
  if ([...SERVER_FIELDS].some((key) => Object.hasOwn(edited, key))) {
    throw new Error("Managed id and embedding fields cannot be edited.");
  }
  const isCreate = expectedVersion === "absent";
  if (isCreate && !id) throw new Error("A new version-bound record requires an explicit record ID.");
  if (isCreate && original != null) throw new Error("An absent-version create cannot be based on an existing record snapshot.");
  if (isCreate ? expectedVersion !== "absent" : !SHA256_VERSION.test(expectedVersion ?? "")) {
    throw new Error("Use `absent` for a new record or the exact current case_record version for an update.");
  }
  if (!isCreate && (!original || typeof original !== "object" || Array.isArray(original))) {
    throw new Error("An existing record update requires its complete case_record snapshot.");
  }
  if (table === "source") {
    if (!isCreate && original.kind !== "case_document") {
      throw new Error("Only personal source records with kind `case_document` can use this editor.");
    }
    if (Object.hasOwn(edited, "kind") && edited.kind !== "case_document") {
      throw new Error("A personal source record must remain kind `case_document`.");
    }
  }

  const editable = Object.fromEntries(Object.entries(edited).filter(([key]) => !SERVER_FIELDS.has(key)));
  let data = editable;
  if (!isCreate) {
    const baseline = Object.fromEntries(Object.entries(original).filter(([key]) => !SERVER_FIELDS.has(key)));
    if (Object.keys(baseline).some((key) => !Object.hasOwn(editable, key))) {
      throw new Error("To clear a field, set its value to null; removing it leaves stored content unchanged.");
    }
    data = Object.fromEntries(Object.entries(editable).filter(([key, value]) =>
      !Object.hasOwn(baseline, key) || canonicalJson(value) !== canonicalJson(baseline[key])));
    if (Object.keys(data).length === 0) throw new Error("Change a field before saving this record.");
  }
  // The case store routes personal source writes through an atomic kind guard.
  if (table === "source") data = { ...data, kind: "case_document" };

  const recordId = id ? id.replace(new RegExp(`^${table}:`), "") : undefined;
  if (id && (!id.startsWith(`${table}:`) || !recordId)) throw new Error("Record ID must match the selected table.");
  return {
    table,
    ...(recordId ? { id: recordId } : {}),
    expected_version: expectedVersion,
    data,
  };
}

/** Removes only server-managed identity/vector fields before showing a record in the editable form.
 * Inputs: complete shared record object. Outputs: copy with id and embedding omitted; all personal and unknown fields remain.
 * Effects: none. Use for the form view so managed fields cannot be changed by manual JSON edits.
 */
export function personalRecordForEditing(record) {
  return Object.fromEntries(Object.entries(record ?? {}).filter(([key]) => !SERVER_FIELDS.has(key)));
}

/** Builds the exact case_record reference returned by case_put's bare-key result.
 * Inputs: selected table and a case_put success result. Outputs: table-qualified shared ID.
 * Effects: none. Use for immediate case_record readback after case_put.
 */
export function personalCaseRecordRef(table, result) {
  if (!PERSONAL_TABLES.has(table) || result?.table !== table || typeof result.id !== "string" || !result.id.trim()) {
    throw new Error("case_put did not return the expected table and record key.");
  }
  return `${table}:${result.id}`;
}

/** Identifies an explicit shared-record version conflict for visible UI handling.
 * Inputs: a tool error message. Outputs: boolean. Effects: none. Use to distinguish stale edits from transport failures.
 */
export function isPersonalCaseVersionConflict(message) {
  return /conflict|expected[_ ]version|version mismatch|record changed/i.test(String(message ?? ""));
}
