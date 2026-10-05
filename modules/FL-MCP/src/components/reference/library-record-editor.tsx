// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
import * as React from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { storeApi } from "@/lib/api-client";
import {
  canPublishLibraryProposal,
  canRetryLibraryValidation,
  changedRecordPatch,
  editableRecord,
  hostedStructuredResult,
  proposalDispatch,
  validationRecordId,
} from "@/lib/library-mutations";

type LibraryTable = "reference" | "source";
type CitationDraft = { source_id: string; pinpoint: string; claim: string; source_version: string | null; error: string | null };
type ExactRecord = { contract: string; id: string; table: string; version: string; record: Record<string, unknown> };
type ProposalResult = Record<string, unknown> & { proposal_id: string; status: string; proposed_hash: string };

const HASH_VERSION = /^sha256:[a-f0-9]{64}$/i;

/** Loads one exact hosted record and rejects wrong identities or malformed versions.
 * Input is a full record reference. Output is an exact case_record envelope.
 * Effects are a read-only hosted MCP call; use it for citation, proposal, and receipt snapshots.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
async function loadHostedRecord(id: string): Promise<ExactRecord> {
  const envelope = await storeApi.libraryTool("case_record", { id });
  const result = hostedStructuredResult(envelope);
  if (!result) throw new Error(envelope.text.find((text) => text.trim()) ?? "Hosted case_record returned no structured record.");
  if (result.id !== id || typeof result.version !== "string" || !HASH_VERSION.test(result.version) ||
    !result.record || typeof result.record !== "object" || Array.isArray(result.record)) {
    throw new Error("Hosted case_record did not return the exact record identity, version, and body.");
  }
  return result as unknown as ExactRecord;
}

/** Parses a JSON record editor buffer and returns data or a visible validation message.
 * Input is textarea content. Output is a plain record or an error; it performs no side effects.
 * Use this before forming changed-only mutations so malformed drafts remain local.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function parseEditedRecord(text: string): { record: Record<string, unknown> | null; error: string | null } {
  try {
    const parsed: unknown = JSON.parse(text);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? { record: parsed as Record<string, unknown>, error: null }
      : { record: null, error: "Record content must be a JSON object." };
  } catch {
    return { record: null, error: "Record content is not valid JSON." };
  }
}

/** Classifies stale-version and citation-validation failures for per-record feedback.
 * Input is a hosted MCP error message. Output is conflict, validation, or error.
 * It performs no I/O and never upgrades failure into success.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function failureKind(message: string): "conflict" | "validation" | "error" {
  const normalized = message.toLowerCase();
  if (normalized.includes("conflict") || normalized.includes("version mismatch") || normalized.includes("expected version")) return "conflict";
  if (normalized.includes("validation") || normalized.includes("citation") || normalized.includes("currency")) return "validation";
  return "error";
}

/** Renders shared library proposals and direct versioned edits for personal source records.
 * Inputs are the selected exact reference/source body and case_record version; output is a native editor.
 * Side effects occur only on explicit actions and go through the five allowlisted hosted MCP calls.
 * Use this alongside full record detail; library publication remains server-gated by its stored receipt.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
export function LibraryRecordEditor({
  table,
  id,
  version,
  record,
}: {
  table: LibraryTable;
  id: string;
  version: string;
  record: Record<string, unknown>;
}) {
  return (
    <section className="space-y-4 border-t border-border pt-4" aria-label="Versioned library editing">
      <LibraryProposalEditor table={table} id={id} version={version} record={record} />
      {table === "source" && record.kind === "case_document" && (
        <PersonalCaseSourceEditor id={id} version={version} record={record} />
      )}
    </section>
  );
}

/** Renders the citation-bound proposal workflow for one exact shared source or reference.
 * Inputs are its full id, current case_record version, and complete record body; output is an editor/status panel.
 * Effects are hosted proposal, validation retry, exact record reads, and confirmed gated publication.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function LibraryProposalEditor({ table, id, version, record }: { table: LibraryTable; id: string; version: string; record: Record<string, unknown> }) {
  const queryClient = useQueryClient();
  const base = React.useMemo(() => editableRecord(record, "library"), [record]);
  const [bodyText, setBodyText] = React.useState(() => JSON.stringify(base, null, 2));
  const [rationale, setRationale] = React.useState("");
  const [citations, setCitations] = React.useState<CitationDraft[]>([{ source_id: "", pinpoint: "", claim: "", source_version: null, error: null }]);
  const [proposal, setProposal] = React.useState<ProposalResult | null>(null);
  const [proposalRecord, setProposalRecord] = React.useState<ExactRecord | null>(null);
  const [validationRecord, setValidationRecord] = React.useState<ExactRecord | null>(null);
  const [reviewed, setReviewed] = React.useState(false);
  const [confirmed, setConfirmed] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [message, setMessage] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [errorType, setErrorType] = React.useState<"conflict" | "validation" | "error" | null>(null);
  const edited = React.useMemo(() => parseEditedRecord(bodyText), [bodyText]);
  const patchState = React.useMemo(() => {
    if (!edited.record) return { patch: {}, error: null as string | null };
    try {
      return { patch: changedRecordPatch(base, edited.record, "library"), error: null };
    } catch (caught) {
      return { patch: {}, error: caught instanceof Error ? caught.message : "Record fields could not be compared." };
    }
  }, [base, edited.record]);
  const patch = patchState.patch;
  const inputError = edited.error ?? patchState.error;
  const dispatch = proposalDispatch(proposal, proposalRecord);
  const publishAllowed = canPublishLibraryProposal(proposal, proposalRecord, validationRecord);

  /** Invalidates only the selected library page and exact detail after a confirmed shared write.
   * Input is none. Output is void; it triggers local query refreshes without issuing another mutation.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  function refreshLibraryQueries() {
    void queryClient.invalidateQueries({ queryKey: ["store", "reference-library"] });
    void queryClient.invalidateQueries({ queryKey: ["store", "case-record", id] });
  }

  /** Clears server-status evidence after any local proposal content or citation change.
   * Input is none. Output is void; current form values remain intact and must be resubmitted.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  function clearProposalEvidence() {
    setProposal(null);
    setProposalRecord(null);
    setValidationRecord(null);
    setConfirmed(false);
    setReviewed(false);
    setMessage(null);
    setError(null);
    setErrorType(null);
  }

  /** Runs one hosted action and extracts structured output or preserves the server error for display.
   * Inputs are an allowlisted operation and JSON object; output is structured data.
   * Effects are one authenticated hosted MCP request. Use instead of any local store mutation call.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function invoke(operation: "library_propose" | "library_validate" | "library_publish" | "case_record" | "case_put", args: Record<string, unknown>) {
    const envelope = await storeApi.libraryTool(operation, args);
    const result = hostedStructuredResult(envelope);
    if (!result) throw new Error(envelope.text.find((text) => text.trim()) ?? `Hosted ${operation} returned no structured result.`);
    return result;
  }

  /** Resolves one citation to the exact current source version through hosted case_record.
   * Input is a citation index. Output is void; it updates only this editor's citation snapshot.
   * Effects are a read-only hosted MCP request; no user-entered hash is accepted as evidence.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function resolveCitation(index: number) {
    const citation = citations[index];
    setBusy(true);
    setError(null);
    setErrorType(null);
    try {
      if (!citation.source_id.startsWith("source:")) throw new Error("Citation id must be a source:<id> record.");
      const exact = await loadHostedRecord(citation.source_id);
      clearProposalEvidence();
      setCitations((current) => current.map((entry, currentIndex) => currentIndex === index
        ? { ...entry, source_version: exact.version, error: null }
        : entry));
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "Source version lookup failed.";
      setCitations((current) => current.map((entry, currentIndex) => currentIndex === index
        ? { ...entry, source_version: null, error: detail }
        : entry));
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Saves only changed editable fields as a citation-bound pending proposal.
   * Input is the reviewed target draft, exact citations, and rationale; output is the full proposal id/status/hash.
   * Effects retain a shared proposal through hosted library_propose; published source/reference rows stay unchanged.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function saveProposal() {
    if (!edited.record || Object.keys(patch).length === 0) return;
    setBusy(true);
    setError(null);
    setErrorType(null);
    setMessage(null);
    try {
      if (!HASH_VERSION.test(version)) throw new Error("The selected record has no current exact case_record version.");
      if (id !== `${table}:${id.slice(table.length + 1)}` || !id.startsWith(`${table}:`)) throw new Error("The selected record identity does not match its library table.");
      const resolvedCitations = citations.map((citation) => {
        if (!citation.source_version || !citation.pinpoint.trim() || !citation.claim.trim()) {
          throw new Error("Resolve every source version and provide a pinpoint and supported claim.");
        }
        return { source_id: citation.source_id, source_version: citation.source_version, pinpoint: citation.pinpoint.trim(), claim: citation.claim.trim() };
      });
      if (!rationale.trim()) throw new Error("Add a rationale before saving the proposal.");
      const result = await invoke("library_propose", {
        id,
        expected_version: version,
        patch,
        citations: resolvedCitations,
        rationale: rationale.trim(),
      });
      if (typeof result.proposal_id !== "string" || !result.proposal_id.startsWith("library_proposal:") ||
        result.status !== "pending_validation" || typeof result.proposed_hash !== "string" || !HASH_VERSION.test(result.proposed_hash)) {
        throw new Error("library_propose did not return its full proposal id, pending status, and proposed hash.");
      }
      setProposal(result as ProposalResult);
      setProposalRecord(null);
      setValidationRecord(null);
      setConfirmed(false);
      setReviewed(false);
      setMessage(`Saved ${result.proposal_id}; the shared validator controls publication.`);
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "The shared proposal was not saved.";
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Reads exact proposal and separate validator records to render server-owned state.
   * Input is the full result proposal id. Output is void; both snapshots are checked before they can gate publish.
   * Effects are two read-only hosted case_record calls and clear prior publish confirmation.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function refreshProposalStatus() {
    if (!proposal) return;
    setBusy(true);
    setError(null);
    setErrorType(null);
    setValidationRecord(null);
    setConfirmed(false);
    try {
      const exactProposal = await loadHostedRecord(proposal.proposal_id);
      const proposalBody = exactProposal.record;
      if (proposalBody.proposed_hash !== proposal.proposed_hash || proposalBody.status === "published") {
        setProposalRecord(exactProposal);
        throw new Error(proposalBody.status === "published" ? "This proposal is already published." : "The shared proposal hash no longer matches the saved result.");
      }
      setProposalRecord(exactProposal);
      const receiptId = validationRecordId(proposal.proposal_id);
      const exactReceipt = await loadHostedRecord(receiptId);
      if (exactReceipt.record.proposal_id !== proposal.proposal_id || exactReceipt.record.proposed_hash !== proposal.proposed_hash) {
        throw new Error("The separate validation record does not match this full proposal id and proposed hash.");
      }
      setValidationRecord(exactReceipt);
      setMessage(`Read proposal and receipt records at ${exactProposal.version} and ${exactReceipt.version}.`);
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "Shared proposal status could not be read.";
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Retries validation only when the latest server-owned dispatch marks its queue failure retryable.
   * Input is the full proposal result. Output is void; it refreshes exact proposal and receipt snapshots afterward.
   * Effects call hosted library_validate but cannot set validation status or currency fields.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function retryValidation() {
    if (!proposal || !canRetryLibraryValidation(dispatch)) return;
    setBusy(true);
    setError(null);
    setErrorType(null);
    try {
      const result = await invoke("library_validate", { proposal_id: proposal.proposal_id });
      const nextDispatch = result.dispatch && typeof result.dispatch === "object" ? result.dispatch as Record<string, unknown> : result;
      if (typeof nextDispatch.state !== "string") throw new Error("library_validate returned no dispatch state.");
      setProposal((current) => current ? { ...current, dispatch: nextDispatch } : current);
      setMessage(`Validation retry returned ${nextDispatch.state}; reread shared status for the receipt.`);
      await refreshProposalStatus();
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "Validation retry was not accepted.";
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Requests guarded publication after exact proposal/receipt checks and explicit user confirmation.
   * Input is a full pending proposal id and a confirmation checkbox; output is the server's publication receipt.
   * Effects call hosted library_publish, which atomically rechecks the stored receipt and citation versions.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function publishProposal() {
    if (!proposal || !publishAllowed || !confirmed) return;
    setBusy(true);
    setError(null);
    setErrorType(null);
    try {
      const result = await invoke("library_publish", { proposal_id: proposal.proposal_id });
      if (result.status !== "published") throw new Error("The hosted publisher did not confirm publication.");
      setMessage(`Published ${proposal.proposal_id} as ${String(result.id ?? "the server-confirmed record")}.`);
      setProposal(null);
      setProposalRecord(null);
      setValidationRecord(null);
      setConfirmed(false);
      refreshLibraryQueries();
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "Publication was not completed.";
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  /** Updates one citation field and invalidates all validation evidence from the prior draft.
   * Inputs are its row index, field name, and text value. Output is void; the citation version resets on id edits.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  function editCitation(index: number, field: "source_id" | "pinpoint" | "claim", value: string) {
    setCitations((current) => current.map((entry, currentIndex) => currentIndex === index
      ? { ...entry, [field]: value, ...(field === "source_id" ? { source_version: null, error: null } : {}) }
      : entry));
    clearProposalEvidence();
  }

  /** Appends one empty source claim citation to the draft.
   * Input is none. Output is void; existing personal citation text is preserved.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  function addCitation() {
    setCitations((current) => [...current, { source_id: "", pinpoint: "", claim: "", source_version: null, error: null }]);
    clearProposalEvidence();
  }

  /** Removes one citation row without changing any other citation or stored record.
   * Input is its row index. Output is void; the changed proposal evidence is discarded locally.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  function removeCitation(index: number) {
    setCitations((current) => current.filter((_, currentIndex) => currentIndex !== index));
    clearProposalEvidence();
  }

  return (
    <details className="rounded-[var(--radius-sm)] border border-border bg-bg p-3">
      <summary className="cursor-pointer font-medium text-accent-text">Propose a citation-validated {table} edit</summary>
      <div className="mt-3 space-y-3">
        <p className="text-sm text-text-secondary">Expected version <code className="break-all">{version}</code>. Full personal fields remain in the shared pending proposal. Keep existing keys in the JSON; use explicit null to clear a field.</p>
        <label className="block text-sm text-text-secondary">Complete editable record fields
          <textarea className="mt-1 min-h-56 w-full rounded-[var(--radius-sm)] border border-border bg-surface p-3 font-mono text-xs text-text-primary" maxLength={512_000} spellCheck={false} aria-label="Shared library record JSON" value={bodyText} onChange={(event) => { setBodyText(event.target.value); clearProposalEvidence(); }} />
        </label>
        {inputError && <p role="alert" className="text-sm text-critical-text">{inputError}</p>}
        <p className="text-xs text-text-tertiary">Changed fields sent: {Object.keys(patch).length}. Identity, embedding, and reserved publication fields are never included.</p>
        <h4 className="font-medium">Claim citations</h4>
        {citations.map((citation, index) => (
          <fieldset key={index} className="space-y-2 rounded-[var(--radius-sm)] border border-border p-3">
            <legend className="px-1 text-sm">Citation {index + 1}</legend>
            <label className="block text-sm text-text-secondary">Source record id
              <input className="mt-1 min-h-11 w-full rounded-[var(--radius-sm)] border border-border bg-surface px-3 text-text-primary" value={citation.source_id} placeholder="source:…" onChange={(event) => editCitation(index, "source_id", event.target.value)} />
            </label>
            <label className="block text-sm text-text-secondary">Pinpoint
              <input className="mt-1 min-h-11 w-full rounded-[var(--radius-sm)] border border-border bg-surface px-3 text-text-primary" value={citation.pinpoint} onChange={(event) => editCitation(index, "pinpoint", event.target.value)} />
            </label>
            <label className="block text-sm text-text-secondary">Claim supported
              <textarea className="mt-1 min-h-20 w-full rounded-[var(--radius-sm)] border border-border bg-surface p-3 text-text-primary" value={citation.claim} onChange={(event) => editCitation(index, "claim", event.target.value)} />
            </label>
            <p className="break-all text-xs text-text-tertiary">Exact source version: {citation.source_version ?? "unresolved"}{citation.error ? ` · ${citation.error}` : ""}</p>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" disabled={busy || !citation.source_id} onClick={() => void resolveCitation(index)}>Resolve exact source version</Button>
              {citations.length > 1 && <Button variant="outline" disabled={busy} onClick={() => removeCitation(index)}>Remove citation</Button>}
            </div>
          </fieldset>
        ))}
        <Button variant="outline" disabled={busy} onClick={addCitation}>Add citation</Button>
        <label className="block text-sm text-text-secondary">Rationale
          <textarea className="mt-1 min-h-20 w-full rounded-[var(--radius-sm)] border border-border bg-surface p-3 text-text-primary" value={rationale} onChange={(event) => { setRationale(event.target.value); clearProposalEvidence(); }} />
        </label>
        <label className="flex min-h-11 items-center gap-2 text-sm text-text-secondary"><input type="checkbox" checked={reviewed} onChange={(event) => setReviewed(event.target.checked)} />I reviewed the complete changed fields, retained personal content, and every citation.</label>
        <Button variant="solid" disabled={busy || Boolean(inputError) || !reviewed || !edited.record || Object.keys(patch).length === 0 || !rationale.trim() || citations.length === 0 || citations.some((item) => !item.source_version || !item.pinpoint.trim() || !item.claim.trim())} onClick={() => void saveProposal()}>
          {busy ? "Working…" : "Save shared proposal"}
        </Button>

        {proposal && (
          <div className="space-y-2 border-t border-border pt-3" aria-live="polite">
            <p><strong>Shared proposal</strong> · <code className="break-all">{proposal.proposal_id}</code> · {proposal.status}</p>
            <p className="break-all text-xs text-text-tertiary">Proposed hash: {proposal.proposed_hash}</p>
            {dispatch && <div role="status" data-dispatch-state={String(dispatch.state)}>
              <p>Validation dispatch: <strong>{String(dispatch.state ?? "unknown")}</strong>{dispatch.reason ? ` · ${String(dispatch.reason)}` : ""}</p>
              {typeof dispatch.workflow_id === "string" && <p className="break-all text-xs text-text-tertiary">Workflow {dispatch.workflow_id}</p>}
              {typeof dispatch.run_id === "string" && <p className="break-all text-xs text-text-tertiary">Run {dispatch.run_id}</p>}
              {canRetryLibraryValidation(dispatch) && <Button variant="outline" disabled={busy} onClick={() => void retryValidation()}>Retry validation</Button>}
            </div>}
            <Button variant="outline" disabled={busy} onClick={() => void refreshProposalStatus()}>Read proposal and validation status</Button>
            {proposalRecord && <p role="status" className="break-all text-sm text-text-secondary">Proposal record {proposalRecord.id} · {String(proposalRecord.record.status ?? "unknown")}; exact record version {proposalRecord.version}.</p>}
            {validationRecord && <p role="status" className="break-all text-sm text-text-secondary">Separate receipt {validationRecord.id} · {String(validationRecord.record.status ?? "unknown")} · currency {String(validationRecord.record.currency_status ?? "unknown")}.</p>}
            {publishAllowed ? (
              <>
                <label className="flex min-h-11 items-center gap-2 text-sm text-text-secondary"><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} />I reviewed the matching server validation receipt and authorize publishing this proposal.</label>
                <Button variant="solid" disabled={busy || !confirmed} onClick={() => void publishProposal()}>Publish cleared proposal</Button>
              </>
            ) : <p className="text-sm text-text-tertiary">Publish is disabled until the exact proposal and separate receipt match, report VERIFIED_PRIMARY, and show cleared currency.</p>}
          </div>
        )}
        {message && <p role="status" className="text-sm text-text-secondary">{message}</p>}
        {error && <p role="alert" data-state={errorType ?? "error"} className="text-sm text-critical-text">{errorType === "conflict" ? "Version conflict: " : errorType === "validation" ? "Validation blocked: " : "Error: "}{error}</p>}
      </div>
    </details>
  );
}

/** Renders a direct exact-version editor only for personal case_document source rows.
 * Inputs are the full source reference, exact case_record version, and complete personal record.
 * Effects call hosted case_put with changed keys and expected_version; legal sources never enter this route.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function PersonalCaseSourceEditor({ id, version, record }: { id: string; version: string; record: Record<string, unknown> }) {
  const queryClient = useQueryClient();
  const base = React.useMemo(() => editableRecord(record, "personal"), [record]);
  const [bodyText, setBodyText] = React.useState(() => JSON.stringify(base, null, 2));
  const [confirmed, setConfirmed] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [message, setMessage] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [errorType, setErrorType] = React.useState<"conflict" | "validation" | "error" | null>(null);
  const edited = React.useMemo(() => parseEditedRecord(bodyText), [bodyText]);
  const patchState = React.useMemo(() => {
    if (!edited.record) return { patch: {}, error: null as string | null };
    try {
      return { patch: changedRecordPatch(base, edited.record, "personal"), error: null };
    } catch (caught) {
      return { patch: {}, error: caught instanceof Error ? caught.message : "Record fields could not be compared." };
    }
  }, [base, edited.record]);
  const patch = patchState.patch;
  const inputError = edited.error ?? patchState.error;

  /** Saves a personal source revision through the version-aware hosted case_put tool.
   * Input is the reviewed source draft and exact selected version. Output is a server-confirmed write envelope.
   * Effects retain a before/after revision in the shared backend; mismatches return a conflict and write nothing.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  async function savePersonalRevision() {
    if (!confirmed || !edited.record || !Object.keys(patch).length) return;
    setBusy(true);
    setError(null);
    setErrorType(null);
    setMessage(null);
    try {
      if (!id.startsWith("source:") || record.kind !== "case_document" || !HASH_VERSION.test(version)) {
        throw new Error("Direct versioned edits require an exact case_document source record and version.");
      }
      const result = await storeApi.libraryTool("case_put", {
        table: "source",
        id: id.slice("source:".length),
        expected_version: version,
        // The hosted backend requires the personal-source discriminator on every source write.
        data: { ...patch, kind: "case_document" },
      });
      const saved = hostedStructuredResult(result);
      if (!saved || saved.available !== true || saved.table !== "source" || !saved.record || typeof saved.record !== "object") {
        throw new Error(result.text.find((text) => text.trim()) ?? "case_put did not confirm a versioned personal source revision.");
      }
      setMessage("Personal source revision saved with its exact prior version retained.");
      setConfirmed(false);
      void queryClient.invalidateQueries({ queryKey: ["store", "reference-library"] });
      void queryClient.invalidateQueries({ queryKey: ["store", "case-record", id] });
    } catch (caught) {
      const detail = caught instanceof Error ? caught.message : "The personal source revision was not saved.";
      setError(detail);
      setErrorType(failureKind(detail));
    } finally {
      setBusy(false);
    }
  }

  return (
    <details className="rounded-[var(--radius-sm)] border border-border bg-bg p-3">
      <summary className="cursor-pointer font-medium text-accent-text">Edit personal case-document source</summary>
      <div className="mt-3 space-y-3">
        <p className="text-sm text-text-secondary">This direct edit is limited to <code>kind: case_document</code> and is guarded by the exact record version. Full names and personal text are preserved. Keep existing keys in the JSON; use explicit null to clear a field.</p>
        <p className="break-all text-xs text-text-tertiary">Expected version: {version}</p>
        <label className="block text-sm text-text-secondary">Complete editable personal fields
          <textarea className="mt-1 min-h-56 w-full rounded-[var(--radius-sm)] border border-border bg-surface p-3 font-mono text-xs text-text-primary" maxLength={512_000} spellCheck={false} aria-label="Personal case source JSON" value={bodyText} onChange={(event) => { setBodyText(event.target.value); setConfirmed(false); setMessage(null); setError(null); }} />
        </label>
        {inputError && <p role="alert" className="text-sm text-critical-text">{inputError}</p>}
        <p className="text-xs text-text-tertiary">Changed fields sent: {Object.keys(patch).length}. Record identity and embedding are not editable.</p>
        <label className="flex min-h-11 items-center gap-2 text-sm text-text-secondary"><input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} />I reviewed the complete personal record and authorize this versioned edit.</label>
        <Button variant="solid" disabled={busy || Boolean(inputError) || !confirmed || !edited.record || Object.keys(patch).length === 0 || !HASH_VERSION.test(version)} onClick={() => void savePersonalRevision()}>
          {busy ? "Saving…" : "Save personal revision"}
        </Button>
        {message && <p role="status" className="text-sm text-text-secondary">{message}</p>}
        {error && <p role="alert" data-state={errorType ?? "error"} className="text-sm text-critical-text">{errorType === "conflict" ? "Version conflict: " : "Error: "}{error}</p>}
      </div>
    </details>
  );
}
