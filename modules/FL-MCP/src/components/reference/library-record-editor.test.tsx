// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { storeApi } from "@/lib/api-client";
import { LibraryRecordEditor, readLibraryOriginalLinks } from "./library-record-editor";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

const targetVersion = `sha256:${"a".repeat(64)}`;
const sourceVersion = `sha256:${"b".repeat(64)}`;
const proposedHash = `sha256:${"c".repeat(64)}`;
const proposalId = "library_proposal:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4";

/** Renders the editor under the existing TanStack query provider with an isolated cache.
 * Inputs are exact record fixture fields; output is a Testing Library render result.
 * It performs no network calls because library tool requests are explicitly mocked per test.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
function renderRecordEditor(table: "reference" | "source", id: string, record: Record<string, unknown>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LibraryRecordEditor table={table} id={id} version={targetVersion} record={record} />
    </QueryClientProvider>,
  );
}

describe("native shared library editor", () => {
  /** Accepts the exact hosted PDF locator and rejects arbitrary or malformed destinations.
   * Inputs are synthetic case_record original_links; output is the bounded normalized link list.
   * It performs no network access and verifies that the opaque version survives URL validation.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-05.
   */
  it("validates original-file links and preserves opaque provider versions", () => {
    const bindingId = "library_file:" + "a".repeat(64);
    const versionId = "opaque:version/one + two";
    const params = new URLSearchParams({ binding_id: bindingId, version_id: versionId });
    const link = {
      binding_id: bindingId,
      version_id: versionId,
      sha256: "b".repeat(64),
      bytes: 1234,
      content_type: "application/pdf",
      href: "https://family-court.tilapia-skilift.ts.net/api/library/original?" + params,
    };

    expect(readLibraryOriginalLinks([link])).toEqual([link]);
    expect(readLibraryOriginalLinks([
      { ...link, href: "https://example.com/api/library/original?" + params },
      { ...link, href: "https://family-court.tilapia-skilift.ts.net/api/library/other?" + params },
      { ...link, bytes: 20 * 1024 * 1024 + 1 },
      { ...link, version_id: "bad\nlocator" },
    ])).toEqual([]);
  });

  /** Confirms reference proposals use the full proposal id and a distinct exact validation receipt gate.
   * Inputs are synthetic hosted MCP envelopes. Output is observable UI and invoker arguments only.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("submits a changed-only cited proposal and publishes only after its matching cleared receipt", async () => {
    const calls: Array<{ name: string; args: Record<string, unknown> }> = [];
    const invoke = vi.spyOn(storeApi, "libraryTool").mockImplementation(async (name, args) => {
      calls.push({ name, args });
      if (name === "case_record" && args.id === "source:authority") {
        return { state: "ok", structured: { id: args.id, table: "source", version: sourceVersion, contract: "fixture", record: { title: "Synthetic authority" } }, text: [] };
      }
      if (name === "library_propose") return { state: "ok", structured: { proposal_id: proposalId, status: "pending_validation", proposed_hash: proposedHash }, text: [] };
      if (name === "case_record" && args.id === proposalId) {
        return { state: "ok", structured: { id: proposalId, table: "library_proposal", version: targetVersion, contract: "fixture", record: { status: "pending_validation", proposed_hash: proposedHash, dispatch: { state: "queued", retryable: false } } }, text: [] };
      }
      if (name === "case_record" && args.id === "library_validation:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4") {
        return { state: "ok", structured: { id: args.id, table: "library_validation", version: sourceVersion, contract: "fixture", record: { proposal_id: proposalId, proposed_hash: proposedHash, status: "VERIFIED_PRIMARY", currency_status: "cleared" } }, text: [] };
      }
      if (name === "library_publish") return { state: "ok", structured: { status: "published", id: "reference:fixture" }, text: [] };
      throw new Error(`Unexpected fixture operation ${name}`);
    });

    renderRecordEditor("reference", "reference:fixture", { title: "Existing", body: "Original complete text", created_at: "2026-10-04T10:00:00.000Z", embedding: [1] });
    fireEvent.click(screen.getByText("Propose a citation-validated reference edit"));
    const bodyEditor = screen.getByRole("textbox", { name: "Shared library record JSON" });
    fireEvent.change(bodyEditor, { target: { value: JSON.stringify({ title: "Existing", body: "Revised complete text", created_at: "2026-10-04T10:00:00.000Z" }, null, 2) } });
    fireEvent.change(screen.getByLabelText("Source record id"), { target: { value: "source:authority" } });
    fireEvent.change(screen.getByLabelText("Pinpoint"), { target: { value: "page 4" } });
    fireEvent.change(screen.getByLabelText("Claim supported"), { target: { value: "Synthetic claim" } });
    fireEvent.click(screen.getByRole("button", { name: "Resolve exact source version" }));
    await waitFor(() => expect(screen.getByText(/Exact source version: sha256:/)).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("Rationale"), { target: { value: "Correct the cited text." } });
    fireEvent.click(screen.getByLabelText(/I reviewed the complete changed fields/));
    fireEvent.click(screen.getByRole("button", { name: "Save shared proposal" }));
    await waitFor(() => expect(screen.getByText(proposalId)).toBeInTheDocument());

    const proposalCall = calls.find((call) => call.name === "library_propose");
    expect(proposalCall?.args).toMatchObject({
      id: "reference:fixture",
      expected_version: targetVersion,
      patch: { body: "Revised complete text" },
      citations: [{ source_id: "source:authority", source_version: sourceVersion, pinpoint: "page 4", claim: "Synthetic claim" }],
    });
    expect(Object.keys(proposalCall?.args.patch as object)).toEqual(["body"]);

    fireEvent.click(screen.getByRole("button", { name: "Read proposal and validation status" }));
    await waitFor(() => expect(screen.getByText(/Separate receipt library_validation:/)).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Publish cleared proposal" })).toBeDisabled();
    fireEvent.click(screen.getByLabelText(/I reviewed the matching server validation receipt/));
    expect(screen.getByRole("button", { name: "Publish cleared proposal" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Publish cleared proposal" }));
    await waitFor(() => expect(calls.some((call) => call.name === "library_publish" && call.args.proposal_id === proposalId)).toBe(true));
    expect(calls.some((call) => call.name === "case_record" && call.args.id === "library_validation:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4")).toBe(true);
    expect(invoke).toHaveBeenCalled();
  });

  /** Confirms personal case_document writes use the exact-version hosted case_put path with full names intact.
   * Inputs are a synthetic source record and fake MCP result. Output is the call envelope; no shared record is changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("saves a changed-only personal case source with expected_version and preserves full personal text", async () => {
    const invoke = vi.spyOn(storeApi, "libraryTool").mockResolvedValue({
      state: "ok", structured: { available: true, table: "source", id: "family-document", record: { body: "Full name: Synthetic Person; revised details" } }, text: [],
    });
    renderRecordEditor("source", "source:family-document", {
      kind: "case_document",
      title: "Full name: Synthetic Person",
      body: "Full name: Synthetic Person; original private details",
      created_at: "2026-10-04T10:00:00.000Z",
      embedding: [1, 2],
    });
    fireEvent.click(screen.getByText("Edit personal case-document source"));
    fireEvent.change(screen.getByRole("textbox", { name: "Personal case source JSON" }), {
      target: { value: JSON.stringify({ kind: "case_document", title: "Full name: Synthetic Person", body: "Full name: Synthetic Person; revised details", created_at: "2026-10-04T10:00:00.000Z" }, null, 2) },
    });
    fireEvent.click(screen.getByLabelText(/I reviewed the complete personal record/));
    fireEvent.click(screen.getByRole("button", { name: "Save personal revision" }));
    await waitFor(() => expect(invoke).toHaveBeenCalledWith("case_put", expect.objectContaining({
      table: "source",
      id: "family-document",
      expected_version: targetVersion,
      data: { body: "Full name: Synthetic Person; revised details", kind: "case_document" },
    })));
  });

  /** Shows a stale personal version as an explicit conflict and keeps the edit form available.
   * Input is a synthetic version-conflict tool result. Output is visible per-record feedback; no record is changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("surfaces a stale personal source version conflict", async () => {
    vi.spyOn(storeApi, "libraryTool").mockResolvedValue({ state: "tool_error", structured: null, text: ["Personal record version conflict"] });
    renderRecordEditor("source", "source:family-document", {
      kind: "case_document",
      title: "Full name: Synthetic Person",
      body: "Original personal source body",
    });
    fireEvent.click(screen.getByText("Edit personal case-document source"));
    fireEvent.change(screen.getByRole("textbox", { name: "Personal case source JSON" }), {
      target: { value: JSON.stringify({ kind: "case_document", title: "Full name: Synthetic Person", body: "Changed personal source body" }) },
    });
    fireEvent.click(screen.getByLabelText(/I reviewed the complete personal record/));
    fireEvent.click(screen.getByRole("button", { name: "Save personal revision" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Version conflict: Personal record version conflict");
  });

  /** Prevents ordinary legal sources from showing the personal direct-write editor.
   * Input is a legal source row fixture. Output is one visibility assertion; it performs no API or database call.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("does not expose direct case_put editing for a non-personal source", () => {
    renderRecordEditor("source", "source:authority", { kind: "primary_authority", title: "Synthetic law" });
    expect(screen.queryByText("Edit personal case-document source")).not.toBeInTheDocument();
  });

  /** Keeps a removed reference key visible as an inline error and prevents proposal submission.
   * Inputs are a synthetic reference row and edited JSON. Output is alert and disabled button state only.
   * No hosted tool is called or shared record changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows a removed reference key inline and disables proposal submission", () => {
    renderRecordEditor("reference", "reference:fixture", { title: "Fixture", body: "Retained text" });
    fireEvent.click(screen.getByText("Propose a citation-validated reference edit"));
    fireEvent.change(screen.getByRole("textbox", { name: "Shared library record JSON" }), { target: { value: JSON.stringify({ title: "Fixture" }) } });
    expect(screen.getByRole("alert")).toHaveTextContent('Existing field "body" was removed; use explicit null to clear it.');
    expect(screen.getByRole("button", { name: "Save shared proposal" })).toBeDisabled();
  });

  /** Keeps a removed personal source key visible as an inline error and prevents direct writes.
   * Inputs are a synthetic case_document row and edited JSON. Output is alert and disabled button state only.
   * No hosted tool is called or personal source changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows a removed personal key inline and disables direct submission", () => {
    renderRecordEditor("source", "source:family-document", { kind: "case_document", body: "Private fixture", title: "Fixture" });
    fireEvent.click(screen.getByText("Edit personal case-document source"));
    fireEvent.change(screen.getByRole("textbox", { name: "Personal case source JSON" }), { target: { value: JSON.stringify({ title: "Fixture" }) } });
    expect(screen.getByRole("alert")).toHaveTextContent('Existing field "body" was removed; use explicit null to clear it.');
    expect(screen.getByRole("button", { name: "Save personal revision" })).toBeDisabled();
  });

  /** Shows malformed reference JSON inline and disables proposal submission without reaching render boundaries.
   * Inputs are a synthetic reference row and malformed textarea text. Output is alert and disabled button state.
   * No hosted tool is called or shared record changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows malformed reference JSON inline and disables proposal submission", () => {
    renderRecordEditor("reference", "reference:fixture", { title: "Fixture", body: "Retained text" });
    fireEvent.click(screen.getByText("Propose a citation-validated reference edit"));
    fireEvent.change(screen.getByRole("textbox", { name: "Shared library record JSON" }), { target: { value: "{" } });
    expect(screen.getByRole("alert")).toHaveTextContent("Record content is not valid JSON.");
    expect(screen.getByRole("button", { name: "Save shared proposal" })).toBeDisabled();
  });

  /** Shows valid non-object reference JSON inline and disables proposal submission.
   * Inputs are a synthetic reference row and JSON array text. Output is the object-shape alert only.
   * No hosted tool is called or shared record changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows non-object reference JSON inline and disables proposal submission", () => {
    renderRecordEditor("reference", "reference:fixture", { title: "Fixture", body: "Retained text" });
    fireEvent.click(screen.getByText("Propose a citation-validated reference edit"));
    fireEvent.change(screen.getByRole("textbox", { name: "Shared library record JSON" }), { target: { value: "[]" } });
    expect(screen.getByRole("alert")).toHaveTextContent("Record content must be a JSON object.");
    expect(screen.getByRole("button", { name: "Save shared proposal" })).toBeDisabled();
  });

  /** Shows malformed personal JSON inline and disables direct submission.
   * Inputs are a synthetic case_document row and malformed textarea text. Output is alert and disabled button state.
   * No hosted tool is called or personal source changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows malformed personal JSON inline and disables direct submission", () => {
    renderRecordEditor("source", "source:family-document", { kind: "case_document", title: "Fixture", body: "Private fixture" });
    fireEvent.click(screen.getByText("Edit personal case-document source"));
    fireEvent.change(screen.getByRole("textbox", { name: "Personal case source JSON" }), { target: { value: "{" } });
    expect(screen.getByRole("alert")).toHaveTextContent("Record content is not valid JSON.");
    expect(screen.getByRole("button", { name: "Save personal revision" })).toBeDisabled();
  });

  /** Shows valid non-object personal JSON inline and disables direct submission.
   * Inputs are a synthetic case_document row and JSON null text. Output is the object-shape alert only.
   * No hosted tool is called or personal source changed.
   * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
   */
  it("shows non-object personal JSON inline and disables direct submission", () => {
    renderRecordEditor("source", "source:family-document", { kind: "case_document", title: "Fixture", body: "Private fixture" });
    fireEvent.click(screen.getByText("Edit personal case-document source"));
    fireEvent.change(screen.getByRole("textbox", { name: "Personal case source JSON" }), { target: { value: "null" } });
    expect(screen.getByRole("alert")).toHaveTextContent("Record content must be a JSON object.");
    expect(screen.getByRole("button", { name: "Save personal revision" })).toBeDisabled();
  });
});
