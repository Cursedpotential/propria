// Byline: Codex · GPT-6-Luna · 2026-10-04
"use client";

import { useMemo, useState } from "react";
import { legalApiBase } from "@/lib/api/client";
import {
  buildLibraryProposalArguments,
  editableLibraryPatch,
  changedLibraryPatch,
  canRetryLibraryValidation,
  canPublishReviewedProposal,
  libraryProposalFailureKind,
  libraryProposalRecordId,
  proposalDispatchState,
  structuredToolResult,
  toolFailureMessage,
} from "@/lib/library-proposals.mjs";

type CitationDraft = { source_id: string; pinpoint: string; claim: string; source_version: string | null; error: string | null };
type ToolDefinition = { connection_id: string; name: string; writes: boolean };
type ToolCatalog = { items: ToolDefinition[] };
type Invocation = { state: "ok" | "tool_error"; text: string[]; structured: unknown };
type ExactRecord = { id: string; table?: string; version: string; record: Record<string, unknown> };
type Dispatch = { state: string; workflow_id?: string; run_id?: string; retryable?: boolean; reason?: string };
type ProposalResult = { proposal_id: string; status: string; proposed_hash?: string; dispatch?: Dispatch };

const styles = {
  panel: { border: "1px solid var(--border)", padding: 16, margin: "18px 0", background: "var(--surface)" },
  input: { display: "block", width: "100%", minHeight: 44, marginTop: 5, padding: "8px 10px", color: "var(--text-primary)", background: "var(--surface)", border: "1px solid var(--border)", borderRadius: 4 },
  button: { minHeight: 44, padding: "8px 14px", border: "1px solid var(--border)", borderRadius: 4, background: "var(--surface)", color: "var(--text-primary)", cursor: "pointer" },
  label: { display: "block", marginTop: 12, fontSize: 13, color: "var(--text-muted)" },
  mono: { fontFamily: "ui-monospace, monospace", overflowWrap: "anywhere" as const },
};

/** Calls one hosted Family Court MCP tool through the existing same-origin BFF.
 * Inputs are its discovered definition and JSON arguments; output is the MCP
 * invocation envelope. Effects are a confirmed remote call, never a local store
 * write. Use this adapter for proposal, proposal-read, and guarded publish calls.
 */
async function invokeHostedTool(tool: ToolDefinition, args: Record<string, unknown>): Promise<Invocation> {
  const response = await fetch(`${legalApiBase()}/v1/mcp/invocations`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ connection_id: tool.connection_id, tool: tool.name, arguments: args, confirm_write: tool.writes }),
  });
  const body = await response.json().catch(() => null);
  if (!response.ok) throw new Error(`Hosted tool call refused (${response.status}): ${body?.detail ?? "no detail"}`);
  if (!body || (body.state !== "ok" && body.state !== "tool_error")) throw new Error("Hosted tool returned a malformed invocation envelope.");
  if (body.state === "tool_error") throw new Error(toolFailureMessage(body as Invocation));
  return body as Invocation;
}

/** Renders a version-bound proposal editor for a shared source or reference.
 * Inputs are a record type, optional exact target and version, and full starting
 * patch. Output is a native workdesk form. Effects call the hosted proposal,
 * exact `case_record` read, and guarded publish tools only after explicit user
 * actions; sibling is the phone/desktop shared-library detail surface.
 */
export function LibraryProposalEditor({
  table,
  initialId,
  expectedVersion,
  initialPatch,
}: {
  table: "source" | "reference";
  initialId?: string;
  expectedVersion?: string;
  initialPatch?: Record<string, unknown>;
}) {
  const [id, setId] = useState(initialId ?? `${table}:${crypto.randomUUID()}`);
  const [version, setVersion] = useState(expectedVersion ?? "absent");
  const [patchText, setPatchText] = useState(() => JSON.stringify(editableLibraryPatch(initialPatch ?? {}), null, 2));
  const [rationale, setRationale] = useState("");
  const [citations, setCitations] = useState<CitationDraft[]>([{ source_id: "", pinpoint: "", claim: "", source_version: null, error: null }]);
  const [tools, setTools] = useState<ToolDefinition[]>([]);
  const [reviewed, setReviewed] = useState(false);
  const [publishConfirmed, setPublishConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errorKind, setErrorKind] = useState<"conflict" | "validation" | "error" | null>(null);
  const [proposal, setProposal] = useState<ProposalResult | null>(null);
  const [proposalRecord, setProposalRecord] = useState<ExactRecord | null>(null);
  const [validationRecord, setValidationRecord] = useState<ExactRecord | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const publishAllowed = canPublishReviewedProposal(proposal, proposalRecord, validationRecord);
  const dispatch = proposalDispatchState(proposal, proposalRecord) as Dispatch | null;
  const createMode = !initialId;
  const normalizedPatch = useMemo(() => {
    try {
      const parsed: unknown = JSON.parse(patchText);
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return { patch: null, error: "Record patch must be a JSON object." };
      return { patch: parsed as Record<string, unknown>, error: null };
    } catch {
      return { patch: null, error: "Record patch is not valid JSON." };
    }
  }, [patchText]);

  /** Clears proposal-state evidence whenever the proposed content changes.
   * Inputs are ordinary form events; output is void. It preserves typed content
   * while requiring a fresh proposal/review after every edit.
   */
  function contentChanged() {
    setProposal(null);
    setProposalRecord(null);
    setValidationRecord(null);
    setMessage(null);
    setError(null);
    setErrorKind(null);
    setReviewed(false);
    setPublishConfirmed(false);
  }

  /** Resolves an exact source version from the canonical record endpoint.
   * Input is one citation index; output is void. It invokes read-only case_record
   * and stores the version only when the exact source id and hash match.
   */
  async function resolveCitation(index: number) {
    const citation = citations[index];
    if (citation.source_id === id && createMode && table === "source") {
      contentChanged();
      setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index
        ? { ...row, source_version: "absent", error: null }
        : row));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const tool = await findTool("case_record", false);
      const detail = structuredToolResult(await invokeHostedTool(tool, { id: citation.source_id })) as ExactRecord | null;
      if (detail?.id !== citation.source_id || !citation.source_id.startsWith("source:") || !/^sha256:[0-9a-f]{64}$/i.test(detail.version)) {
        throw new Error("The source lookup did not return the exact source case_record version.");
      }
      contentChanged();
      setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index
        ? { ...row, source_version: detail.version, error: null }
        : row));
    } catch (exc) {
      const detail = exc instanceof Error ? exc.message : "Source version lookup failed.";
      contentChanged();
      setErrorKind(libraryProposalFailureKind(detail));
      setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, source_version: null, error: detail } : row));
    } finally {
      setBusy(false);
    }
  }

  /** Loads the hosted tool catalog and selects only the requested normalized name.
   * Input is the canonical underscore tool name; output is its hosted definition.
   * It performs read-only catalog I/O and never substitutes a local tool.
   */
  async function findTool(canonical: string, writeRequired = true): Promise<ToolDefinition> {
    let catalog = tools;
    if (catalog.length === 0) {
      const response = await fetch(`${legalApiBase()}/v1/mcp/tools`, { cache: "no-store" });
      const body = await response.json().catch(() => null) as ToolCatalog | null;
      if (!response.ok || !Array.isArray(body?.items)) throw new Error(`Could not load hosted library tools (${response.status}).`);
      catalog = body.items;
      setTools(catalog);
    }
    const tool = catalog.find((item) => (!writeRequired || item.writes) && [canonical.replaceAll("_", "-"), `family-court-${canonical.replaceAll("_", "-")}`].includes(item.name.toLowerCase().replaceAll("_", "-")));
    if (!tool) throw new Error(`Hosted ${canonical.replaceAll("_", "-")} tool is unavailable.`);
    return tool;
  }

  /** Submits the complete record patch and exact citation snapshots as a shared proposal.
   * Input comes from the explicit form review; output is a pending proposal result.
   * It changes only shared pending-proposal state through the hosted tool and keeps
   * the full draft visible when validation or version checks fail.
   */
  async function propose() {
    setBusy(true);
    setError(null);
    setErrorKind(null);
    setMessage(null);
    try {
      if (!normalizedPatch.patch) throw new Error(normalizedPatch.error ?? "Invalid record patch.");
      const args = buildLibraryProposalArguments({
        id,
        expectedVersion: version,
        patch: changedLibraryPatch(normalizedPatch.patch, initialPatch ?? null),
        citations: citations.map(({ source_id, source_version, pinpoint, claim }) => ({ source_id, source_version, pinpoint, claim })),
        rationale,
      });
      const tool = await findTool("library_propose");
      const invocation = await invokeHostedTool(tool, args);
      const result = structuredToolResult(invocation) as ProposalResult | null;
      if (!result?.proposal_id || result.status !== "pending_validation") throw new Error("Proposal tool did not return a pending proposal id/status.");
      libraryProposalRecordId(result.proposal_id);
      setProposal(result);
      setProposalRecord(null);
    setValidationRecord(null);
      setReviewed(false);
      setPublishConfirmed(false);
      setMessage(`Saved shared proposal ${result.proposal_id}; validation is pending.`);
    } catch (exc) {
      const detail = exc instanceof Error ? exc.message : "Proposal could not be saved.";
      setError(detail);
      setErrorKind(libraryProposalFailureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Reads a proposal by its existing exact case_record id.
   * Input is the actual proposal result; output is void. It performs a read-only
   * case_record call and displays only server-stored currency status for publish gating.
   */
  async function refreshProposal() {
    if (!proposal) return;
    setBusy(true);
    setError(null);
    setErrorKind(null);
    try {
      const proposalId = libraryProposalRecordId(proposal.proposal_id);
      const tool = await findTool("case_record", false);
      const record = structuredToolResult(await invokeHostedTool(tool, { id: proposalId })) as ExactRecord | null;
      if (record?.id !== proposalId || !proposalId.startsWith("library_proposal:") ||
        !/^sha256:[0-9a-f]{64}$/i.test(record.version) || !record.record || typeof record.record !== "object") {
        throw new Error("Proposal lookup did not return its exact shared case_record version and body.");
      }
      const validationId = proposalId.replace(/^library_proposal:/, "library_validation:");
      const validation = structuredToolResult(await invokeHostedTool(tool, { id: validationId }));
      if (validation?.found === false) setValidationRecord(null);
      else if (validation?.id === validationId && /^sha256:[0-9a-f]{64}$/i.test(String(validation.version)) && validation.record && typeof validation.record === "object")
        setValidationRecord(validation as ExactRecord);
      else throw new Error("Validation receipt lookup returned a malformed shared record.");
      setProposalRecord(record);
      setPublishConfirmed(false);
      setMessage(`Proposal status read from ${record.id} at ${record.version}.`);
    } catch (exc) {
      const detail = exc instanceof Error ? exc.message : "Proposal status could not be read.";
      setError(detail);
      setErrorKind(libraryProposalFailureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Retries only a server-marked retryable validation-queue failure.
   * Input is the current pending proposal id; output is the returned dispatch
   * state. It invokes library_validate through the hosted MCP transport and then
   * rereads the exact proposal record; it cannot set currency or validation flags.
   */
  async function retryValidation() {
    if (!proposal || !canRetryLibraryValidation(dispatch)) return;
    setBusy(true);
    setError(null);
    setErrorKind(null);
    setMessage(null);
    try {
      const tool = await findTool("library_validate");
      const result = structuredToolResult(await invokeHostedTool(tool, { proposal_id: proposal.proposal_id }));
      const nextDispatch = result?.dispatch && typeof result.dispatch === "object" ? result.dispatch as Dispatch : result as Dispatch | null;
      if (!nextDispatch || typeof nextDispatch.state !== "string") throw new Error("Validation retry did not return a dispatch state.");
      setProposal((current) => current ? { ...current, dispatch: nextDispatch } : current);
      setMessage(`Validation retry returned ${nextDispatch.state}; rereading the shared proposal record.`);
      await refreshProposal();
    } catch (exc) {
      const detail = exc instanceof Error ? exc.message : "Validation retry was not accepted.";
      setError(detail);
      setErrorKind(libraryProposalFailureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Requests server-guarded publication after the exact proposal case_record is cleared.
   * Input is the pending proposal id and the separate human confirmation; output is
   * the actual publish response. It writes only through library_publish, whose server
   * validates the stored receipt atomically with current citations.
   */
  async function publish() {
    if (!proposal || !publishAllowed || !publishConfirmed) return;
    setBusy(true);
    setError(null);
    setErrorKind(null);
    setMessage(null);
    try {
      const tool = await findTool("library_publish");
      const result = structuredToolResult(await invokeHostedTool(tool, { proposal_id: proposal.proposal_id }));
      if (!result || result.status !== "published") throw new Error("Publish tool did not confirm publication; the proposal remains available for review.");
      setMessage(`Published proposal ${proposal.proposal_id} as ${String(result.id ?? "the server-confirmed record")}.`);
      setProposal(null);
      setProposalRecord(null);
    setValidationRecord(null);
      setReviewed(false);
      setPublishConfirmed(false);
    } catch (exc) {
      const detail = exc instanceof Error ? exc.message : "Publish was not completed.";
      setError(detail);
      setErrorKind(libraryProposalFailureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section style={styles.panel} aria-label="Shared library proposal editor">
      <h3 style={{ marginTop: 0 }}>Propose shared {table} {createMode ? "creation" : "edit"}</h3>
      <p style={{ color: "var(--text-muted)" }}>
        Changes stay in the shared pending-proposal workflow until server validation is cleared and you explicitly publish.
      </p>
      <label style={styles.label}>
        Record id
        <input style={styles.input} value={id} onChange={(event) => { setId(event.target.value); contentChanged(); }} aria-label="Record id" />
      </label>
      <p style={styles.mono}>Expected version: {version}</p>
      <label style={styles.label}>
        Full record patch (JSON)
        <textarea
          style={{ ...styles.input, minHeight: 220, fontFamily: "ui-monospace, monospace", lineHeight: 1.45 }}
          value={patchText}
          onChange={(event) => { setPatchText(event.target.value); contentChanged(); }}
          aria-label="Full record patch JSON"
          spellCheck={false}
        />
      </label>
      {normalizedPatch.error && <p role="alert" style={{ color: "var(--danger, #b33a3a)" }}>{normalizedPatch.error}</p>}

      <h4>Citations to current shared sources</h4>
      {citations.map((citation, index) => (
        <fieldset key={index} style={{ border: "1px solid var(--border)", padding: 12, margin: "12px 0" }}>
          <legend>Citation {index + 1}</legend>
          <label style={styles.label}>Source record id<input style={styles.input} value={citation.source_id} onChange={(event) => {
            setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, source_id: event.target.value, source_version: null, error: null } : row));
            contentChanged();
          }} placeholder="source:…" /></label>
          <label style={styles.label}>Pinpoint<input style={styles.input} value={citation.pinpoint} onChange={(event) => {
            setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, pinpoint: event.target.value } : row)); contentChanged();
          }} /></label>
          <label style={styles.label}>Claim supported<textarea style={{ ...styles.input, minHeight: 84 }} value={citation.claim} onChange={(event) => {
            setCitations((rows) => rows.map((row, rowIndex) => rowIndex === index ? { ...row, claim: event.target.value } : row)); contentChanged();
          }} /></label>
          <p style={styles.mono}>
            Source version: {citation.source_version ?? "not resolved"}
            {citation.error ? <span role="alert"> · {citation.error}</span> : null}
          </p>
          <button type="button" style={styles.button} disabled={busy || !citation.source_id} onClick={() => void resolveCitation(index)}>
            {createMode && table === "source" && citation.source_id === id ? "Use absent version for self-capture" : "Resolve exact source version"}
          </button>
          {citations.length > 1 && <button type="button" style={{ ...styles.button, marginLeft: 8 }} onClick={() => {
            setCitations((rows) => rows.filter((_, rowIndex) => rowIndex !== index)); contentChanged();
          }}>Remove citation</button>}
        </fieldset>
      ))}
      <button type="button" style={styles.button} onClick={() => {
        setCitations((rows) => [...rows, { source_id: "", pinpoint: "", claim: "", source_version: null, error: null }]); contentChanged();
      }}>Add citation</button>
      {createMode && table === "source" && <p style={{ color: "var(--text-muted)" }}>A new primary-source capture may cite its own id at version absent; the hosted validator still has to fetch the proposed URL and clear currency.</p>}

      <label style={styles.label}>Rationale<textarea style={{ ...styles.input, minHeight: 84 }} value={rationale} onChange={(event) => { setRationale(event.target.value); contentChanged(); }} /></label>
      <label style={{ display: "flex", alignItems: "center", gap: 10, minHeight: 44, marginTop: 12 }}>
        <input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} />
        I reviewed this complete patch, personal content, and every citation.
      </label>
      <button type="button" style={styles.button} disabled={busy || !reviewed || !normalizedPatch.patch} onClick={() => void propose()}>
        {busy ? "Working…" : "Save shared proposal"}
      </button>

      {proposal && (
        <div style={{ borderTop: "1px solid var(--border)", marginTop: 16, paddingTop: 12 }} aria-live="polite">
          <p><strong>Pending proposal</strong> · <code>{proposal.proposal_id}</code> · {proposal.status}</p>
          {proposal.proposed_hash && <p style={styles.mono}>Proposed hash {proposal.proposed_hash}</p>}
          {dispatch && <div role="status" data-dispatch-state={dispatch.state}>
            <p>Validation dispatch: <strong>{dispatch.state}</strong>{dispatch.reason ? ` · ${dispatch.reason}` : ""}</p>
            {dispatch.workflow_id && <p style={styles.mono}>Workflow {dispatch.workflow_id}</p>}
            {dispatch.run_id && <p style={styles.mono}>Run {dispatch.run_id}</p>}
            {canRetryLibraryValidation(dispatch) && <button type="button" style={styles.button} disabled={busy} onClick={() => void retryValidation()}>
              {busy ? "Retrying…" : "Retry validation"}
            </button>}
          </div>}
          <button type="button" style={styles.button} disabled={busy} onClick={() => void refreshProposal()}>
            {busy ? "Reading…" : "Read shared validation status"}
          </button>
          {proposalRecord && (
            <p role="status" style={{ color: validationRecord?.record.currency_status === "cleared" ? "var(--accent, #5368d8)" : "var(--text-muted)" }}>
              Shared proposal {proposalRecord.id} · receipt currency: {String(validationRecord?.record.currency_status ?? "not cleared")}
            </p>
          )}
          {publishAllowed ? (
            <>
              <label style={{ display: "flex", alignItems: "center", gap: 10, minHeight: 44 }}>
                <input type="checkbox" checked={publishConfirmed} onChange={(event) => setPublishConfirmed(event.target.checked)} />
                I authorize publishing this server-cleared proposal.
              </label>
              <button type="button" style={styles.button} disabled={busy || !publishConfirmed} onClick={() => void publish()}>
                Publish cleared proposal
              </button>
            </>
          ) : <p style={{ color: "var(--text-muted)" }}>Publishing is unavailable until the exact shared validation receipt is verified and its currency is cleared.</p>}
        </div>
      )}
      {message && <p role="status">{message}</p>}
      {error && <p role="alert" data-state={errorKind ?? "error"} style={{ color: "var(--danger, #b33a3a)" }}>
        {errorKind === "conflict" ? "Version conflict: " : errorKind === "validation" ? "Validation blocked: " : "Error: "}{error}
      </p>}
    </section>
  );
}
