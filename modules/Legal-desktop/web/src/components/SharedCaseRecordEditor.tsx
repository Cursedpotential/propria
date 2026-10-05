// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04.
"use client";

import { useState, type ChangeEvent, type FormEvent } from "react";
import { legalApiBase } from "@/lib/api/client";
import {
  buildPersonalCasePutArguments,
  isPersonalCaseVersionConflict,
  personalRecordForEditing,
  personalCaseRecordRef,
} from "@/lib/shared-case-record.mjs";

export type PersonalTable =
  | "person" | "child" | "order" | "hearing" | "deadline" | "event" | "message"
  | "exhibit" | "factor" | "source" | "note" | "court" | "court_event" | "filing"
  | "draft" | "memo" | "evidence_log" | "eval" | "case_status";
type ToolDefinition = { connection_id: string; name: string; writes: boolean };
type ToolCatalog = { items: ToolDefinition[] };
type Invocation = { state: "ok" | "tool_error"; text: string[]; structured: unknown };
type ExactRecord = { id: string; table: string; version: string; record: Record<string, unknown> };
type CasePutResult = { available: boolean; table: string; id: string; record: Record<string, unknown> };

const styles = {
  panel: { border: "1px solid var(--border)", padding: 16, margin: "18px 0", background: "var(--surface)" },
  input: { display: "block", width: "100%", minHeight: 44, marginTop: 5, padding: "8px 10px", color: "var(--text-primary)", background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 4 },
  button: { minHeight: 44, padding: "8px 14px", border: "1px solid var(--border)", borderRadius: 4, background: "var(--surface)", color: "var(--text-primary)", cursor: "pointer" },
  mono: { fontFamily: "ui-monospace, monospace", overflowWrap: "anywhere" as const },
};

/** Renders a native editor for one private shared case-store record.
 * Inputs: a personal table plus optional exact record identity, version, and complete record.
 * Outputs: the current shared record snapshot and a form for explicit edits.
 * Effects: confirmed writes invoke case_put with an exact version and immediately reread case_record; no local database copy is made.
 * Choose for personal records, including source rows whose kind is case_document; source authorities/references use LibraryProposalEditor.
 */
export function SharedCaseRecordEditor({
  table,
  initialId,
  expectedVersion,
  initialRecord,
}: {
  table: PersonalTable;
  initialId?: string;
  expectedVersion?: string;
  initialRecord?: Record<string, unknown>;
}) {
  const isNewRecord = !initialId;
  const startingRecord = personalRecordForEditing(
    initialRecord ?? (table === "source" ? { kind: "case_document" } : {}),
  );
  const [recordId, setRecordId] = useState(() => initialId ?? `${table}:${crypto.randomUUID()}`);
  const [creating, setCreating] = useState(isNewRecord);
  const [version, setVersion] = useState(expectedVersion ?? (isNewRecord ? "absent" : ""));
  const [record, setRecord] = useState<Record<string, unknown>>(startingRecord);
  const [recordText, setRecordText] = useState(JSON.stringify(startingRecord, null, 2));
  const [tools, setTools] = useState<ToolDefinition[]>([]);
  const [confirmed, setConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [readbackPending, setReadbackPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  /** Resolves one exact hosted tool definition from the existing BFF catalog.
   * Inputs: canonical MCP tool name and whether write capability is required. Outputs: the discovered tool.
   * Effects: read-only /v1/mcp/tools request. Use this instead of a local toolkit connection or hard-coded credentials.
   */
  async function findTool(canonical: string, writeRequired: boolean): Promise<ToolDefinition> {
    let catalog = tools;
    if (catalog.length === 0) {
      const response = await fetch(`${legalApiBase()}/v1/mcp/tools`, { cache: "no-store" });
      const body = await response.json().catch(() => null) as ToolCatalog | null;
      if (!response.ok || !Array.isArray(body?.items)) throw new Error(`Could not load hosted toolkit tools (${response.status}).`);
      catalog = body.items;
      setTools(catalog);
    }
    const found = catalog.find((item) => (!writeRequired || item.writes) &&
      [canonical.replaceAll("_", "-"), `family-court-${canonical.replaceAll("_", "-")}`].includes(item.name.toLowerCase().replaceAll("_", "-")));
    if (!found) throw new Error(`Hosted ${canonical.replaceAll("_", "-")} tool is unavailable.`);
    return found;
  }

  /** Calls one discovered hosted MCP tool through the same-origin legal-api BFF.
   * Inputs: discovered definition and exact JSON arguments. Outputs: validated invocation envelope.
   * Effects: posts to /v1/mcp/invocations with the existing confirmation envelope; errors remain visible.
   * Use for case_put and case_record instead of writing through local SQLite or a parallel API.
   */
  async function invokeHostedTool(tool: ToolDefinition, args: Record<string, unknown>): Promise<Invocation> {
    const response = await fetch(`${legalApiBase()}/v1/mcp/invocations`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ connection_id: tool.connection_id, tool: tool.name, arguments: args, confirm_write: tool.writes }),
    });
    const body = await response.json().catch(() => null) as Invocation | null;
    if (!response.ok) throw new Error(`Hosted tool call refused (${response.status}): ${body?.text?.[0] ?? "no detail"}`);
    if (!body || (body.state !== "ok" && body.state !== "tool_error")) throw new Error("Hosted tool returned a malformed invocation envelope.");
    if (body.state === "tool_error") throw new Error(body.text?.find((value) => typeof value === "string" && value.trim()) ?? "Hosted tool returned an error.");
    return body;
  }

  /** Updates the full JSON draft after an owner edits personal record content.
   * Inputs: textarea change event. Outputs: void. Effects: updates local form state only and clears stale feedback.
   * Choose for record values; this does not write to either database.
   */
  function handleRecordTextChange(event: ChangeEvent<HTMLTextAreaElement>) {
    setRecordText(event.target.value);
    setMessage(null);
    setError(null);
    setConflict(false);
  }

  /** Records explicit owner confirmation for the exact visible personal-record form.
   * Inputs: confirmation checkbox event. Outputs: void. Effects: updates local confirmation state only.
   * Choose before saveRecord; no tool call occurs from this handler.
   */
  function handleConfirmChange(event: ChangeEvent<HTMLInputElement>) {
    setConfirmed(event.target.checked);
  }

  /** Extracts the structured result, accepting JSON text only when no structured object exists.
   * Inputs: successful MCP invocation. Outputs: object or null. Effects: none.
   * Use to validate each server response before updating the editor's shared snapshot.
   */
  function structuredResult(invocation: Invocation): Record<string, unknown> | null {
    if (invocation.structured && typeof invocation.structured === "object" && !Array.isArray(invocation.structured)) {
      return invocation.structured as Record<string, unknown>;
    }
    for (const item of invocation.text ?? []) {
      try {
        const parsed: unknown = JSON.parse(item);
        if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) return parsed as Record<string, unknown>;
      } catch {
        // Non-JSON tool text is not treated as a successful record receipt.
      }
    }
    return null;
  }

  /** Submits a changed personal record under an explicit owner confirmation and refreshes its exact snapshot.
   * Inputs: form submission, current complete record, and its shared version. Outputs: refreshed shared record state.
   * Effects: invokes guarded case_put once and then read-only case_record; a failed refresh never retries the write.
   * Use only for personal records; legal source/reference edits remain in the proposal workflow.
   */
  async function saveRecord(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!confirmed || busy) return;
    setBusy(true);
    setError(null);
    setConflict(false);
    setMessage(null);
    let putReceipt: CasePutResult | null = null;
    try {
      const edited: unknown = JSON.parse(recordText);
      const args = buildPersonalCasePutArguments({
        table,
        id: recordId,
        expectedVersion: version,
        edited,
        original: creating ? null : record,
      });
      const putTool = await findTool("case_put", true);
      const put = structuredResult(await invokeHostedTool(putTool, args));
      if (!put || put.available !== true || typeof put.table !== "string" || typeof put.id !== "string" ||
        !put.record || typeof put.record !== "object" || Array.isArray(put.record)) {
        throw new Error("case_put did not return a complete saved-record receipt.");
      }
      putReceipt = put as CasePutResult;
      if (putReceipt.table !== table) throw new Error("case_put returned a record from an unexpected table.");

      const exactId = personalCaseRecordRef(table, putReceipt);
      setRecordId(exactId);
      setReadbackPending(true);
      const readTool = await findTool("case_record", false);
      const refreshed = structuredResult(await invokeHostedTool(readTool, { id: exactId })) as unknown as ExactRecord | null;
      if (!refreshed || refreshed.id !== exactId || refreshed.table !== table ||
        !/^sha256:[0-9a-f]{64}$/i.test(refreshed.version) ||
        !refreshed.record || typeof refreshed.record !== "object" || Array.isArray(refreshed.record)) {
        throw new Error(`case_record did not return the exact updated record ${exactId}.`);
      }
      if (table === "source" && refreshed.record.kind !== "case_document") {
        throw new Error("The refreshed source record is not a personal case_document.");
      }
      setRecordId(refreshed.id);
      setCreating(false);
      setVersion(refreshed.version);
      setReadbackPending(false);
      const editableRecord = personalRecordForEditing(refreshed.record);
      setRecord(editableRecord);
      setRecordText(JSON.stringify(editableRecord, null, 2));
      setConfirmed(false);
      setMessage(`Saved and reread ${refreshed.id} at ${refreshed.version}.`);
    } catch (cause) {
      const detail = cause instanceof Error ? cause.message : "Personal record could not be saved.";
      setConflict(isPersonalCaseVersionConflict(detail));
      setError(putReceipt ? `The shared record was saved, but exact readback failed: ${detail}` : detail);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-label="Personal shared record editor" style={styles.panel}>
      <h2>{recordId ? `Edit personal ${table} record` : `Create personal ${table} record`}</h2>
      {creating ? <p style={styles.mono}>New record: {recordId} · expected version: absent</p> :
        <p style={styles.mono}>Exact record: {recordId} · version: {version}</p>}
      {table === "source" ? <p>This editor accepts only personal source rows with kind `case_document`.</p> : null}
      <form onSubmit={saveRecord}>
        <label htmlFor={`shared-record-${table}`}>
          Complete record JSON (personal values and unknown fields are retained)
        </label>
        <textarea
          id={`shared-record-${table}`}
          aria-label="Complete personal record JSON"
          rows={18}
          spellCheck={false}
          value={recordText}
          onChange={handleRecordTextChange}
          style={styles.input}
        />
        <label style={{ display: "flex", alignItems: "center", gap: 8, margin: "14px 0" }}>
          <input type="checkbox" checked={confirmed} onChange={handleConfirmChange} />
          I reviewed and confirm this personal shared-record change.
        </label>
        <button type="submit" disabled={!confirmed || busy || readbackPending} style={styles.button}>
          {busy ? "Saving and refreshing…" : readbackPending ? "Saved; exact refresh required" : "Save personal record"}
        </button>
      </form>
      {error ? <p role="alert">{conflict ? "Version conflict: " : "Save error: "}{error}</p> : null}
      {readbackPending ? <p role="status">
        The write returned {recordId}, but the current version was not reread. Reload the exact shared record before editing again: {" "}
        <a href={`/toolkit?table=${table}&id=${encodeURIComponent(recordId)}`}>reload {recordId}</a>
      </p> : null}
      {message ? <p role="status">{message}</p> : null}
    </section>
  );
}
