// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
import { describe, expect, it } from "vitest";
import {
  canPublishLibraryProposal,
  canRetryLibraryValidation,
  changedRecordPatch,
  editableRecord,
  hostedStructuredResult,
  proposalDispatch,
  validationRecordId,
} from "./library-mutations";

const targetVersion = `sha256:${"a".repeat(64)}`;
const proposedHash = `sha256:${"b".repeat(64)}`;
const proposalId = "library_proposal:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4";

/** Exercises partial edits, field preservation, and exact receipt gates using synthetic records only.
 * Inputs are in-memory fixtures; output is assertion status. This test makes no sidecar or database calls.
 * Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04
 */
describe("native library mutation helpers", () => {
  it("keeps full personal values in edited data while filtering server-owned fields", () => {
    const base = editableRecord({
      id: "source:family-document",
      kind: "case_document",
      title: "Full name retained",
      body: "Complete confidential synthetic narrative",
      created_at: "2026-10-04T10:00:00.000Z",
      embedding: [1, 2],
    }, "personal");
    expect(base).toEqual({ title: "Full name retained", body: "Complete confidential synthetic narrative", created_at: "2026-10-04T10:00:00.000Z" });
    expect(changedRecordPatch(base, { ...base, body: "Revised complete narrative", kind: "authority", id: "source:forged", embedding: [9] }, "personal"))
      .toEqual({ body: "Revised complete narrative" });
  });

  it("emits only changed library keys and preserves explicit null, nested values, and unchanged dates", () => {
    const base = editableRecord({ id: "reference:one", title: "Fixture", updated_at: "2026-10-04T10:00:00.000Z", body: { text: "unchanged" }, notes: "private value" }, "library");
    const edited = { ...base, updated_at: "2026-10-04T10:00:00.000Z", body: { text: "changed", pii: "complete value" }, notes: null };
    expect(changedRecordPatch(base, edited, "library")).toEqual({ body: { text: "changed", pii: "complete value" }, notes: null });
    expect(changedRecordPatch(base, { ...base }, "library")).toEqual({});
  });

  it("rejects removed existing keys and requires explicit null to clear a value", () => {
    const base = { title: "Fixture", notes: "retain this field" };
    expect(() => changedRecordPatch(base, { title: "Fixture" }, "library"))
      .toThrow('Existing field "notes" was removed; use explicit null to clear it.');
    expect(changedRecordPatch(base, { title: "Fixture", notes: null }, "library"))
      .toEqual({ notes: null });
    expect(() => changedRecordPatch({ body: "private fixture" }, {}, "personal"))
      .toThrow('Existing field "body" was removed; use explicit null to clear it.');
  });

  it("maps only a full proposal id to the separate receipt id", () => {
    expect(validationRecordId(proposalId)).toBe("library_validation:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4");
    expect(() => validationRecordId("2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4")).toThrow(/full library_proposal/);
    expect(() => validationRecordId("library_proposal:library_proposal:uuid")).toThrow(/full library_proposal/);
  });

  it("requires matching proposal and separate verified, cleared receipt records before enabling publish", () => {
    const proposal = { proposal_id: proposalId, proposed_hash: proposedHash, status: "pending_validation" };
    const proposalRecord = { id: proposalId, version: targetVersion, record: { proposed_hash: proposedHash, status: "pending_validation" } };
    const receiptId = validationRecordId(proposalId);
    const receipt = { id: receiptId, version: targetVersion, record: { proposal_id: proposalId, proposed_hash: proposedHash, status: "VERIFIED_PRIMARY", currency_status: "cleared" } };
    expect(canPublishLibraryProposal(proposal, proposalRecord, receipt)).toBe(true);
    expect(canPublishLibraryProposal({ ...proposal, status: "published" }, proposalRecord, receipt)).toBe(false);
    expect(canPublishLibraryProposal(proposal, proposalRecord, { ...receipt, id: "library_validation:other" })).toBe(false);
    expect(canPublishLibraryProposal(proposal, proposalRecord, { ...receipt, record: { ...receipt.record, proposed_hash: targetVersion } })).toBe(false);
    expect(canPublishLibraryProposal(proposal, proposalRecord, { ...receipt, record: { ...receipt.record, currency_status: "provisional" } })).toBe(false);
  });

  it("allows validation retry only for a server-marked queue failure", () => {
    expect(canRetryLibraryValidation({ state: "queue_failed", retryable: true })).toBe(true);
    expect(canRetryLibraryValidation({ state: "queue_failed", retryable: false })).toBe(false);
    expect(canRetryLibraryValidation({ state: "running", retryable: true })).toBe(false);
    expect(proposalDispatch({ dispatch: { state: "queue_failed" } }, null)).toEqual({ state: "queue_failed" });
  });

  it("ignores textual or failed envelopes as evidence of structured success", () => {
    expect(hostedStructuredResult({ state: "tool_error", structured: { status: "published" } })).toBeNull();
    expect(hostedStructuredResult({ state: "ok", structured: "published" })).toBeNull();
    expect(hostedStructuredResult({ state: "ok", structured: { status: "pending_validation" } })).toEqual({ status: "pending_validation" });
  });
});
