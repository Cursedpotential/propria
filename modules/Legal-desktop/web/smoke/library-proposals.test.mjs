// Byline: Codex · GPT-6-Luna · 2026-10-04
import assert from "node:assert/strict";
import test from "node:test";
import {
  buildLibraryProposalArguments,
  editableLibraryPatch,
  changedLibraryPatch,
  canRetryLibraryValidation,
  canPublishReviewedProposal,
  libraryProposalRecordId,
  proposalDispatchState,
  libraryProposalFailureKind,
  structuredToolResult,
  toolFailureMessage,
} from "../src/lib/library-proposals.mjs";

const sourceVersion = `sha256:${"a".repeat(64)}`;
const targetVersion = `sha256:${"b".repeat(64)}`;

// Byline: Codex · GPT-6 · 2026-10-04 — unchanged typed values and private context regression.
test("editor excludes only managed fields and preserves personal context in its merge base", () => {
  const record = { id: "reference:fixture", embedding: [1], validation_status: "old", published_at: "2026-10-04", title: "Original", party_name: "Fixture Full Name", private_details: { address: "Fixture private address", aliases: ["Fixture Alias"] }, dated_at: "2026-10-04T12:00:00Z" };
  const original = structuredClone(record);
  const editable = editableLibraryPatch(record);
  assert.deepEqual(editable.private_details, record.private_details);
  assert.equal(editable.party_name, record.party_name);
  assert.equal("id" in editable, false);
  assert.equal("validation_status" in editable, false);
  assert.deepEqual(changedLibraryPatch({ ...editable, title: "Reviewed" }, record), { title: "Reviewed" });
  assert.deepEqual(record, original);
});

test("editing clears only explicit nulls and rejects ambiguous removal or empty changes", () => {
  const record = { body: "Personal context retained", dated_at: "2026-10-04T12:00:00Z", private_note: "Fixture note" };
  assert.deepEqual(changedLibraryPatch({ ...record, private_note: null }, record), { private_note: null });
  assert.throws(() => changedLibraryPatch({ body: record.body }, record), /set its value to null/);
  assert.throws(() => changedLibraryPatch({ ...record }, record), /Change a field/);
  assert.deepEqual(changedLibraryPatch({ ...record }, null), record);
});

test("proposal arguments preserve full personal patch and exact source snapshot", () => {
  const patch = { title: "Fixture note", body: "Full personal-field fixture content", extra: { keep: [1, 2, 3] } };
  const args = buildLibraryProposalArguments({
    id: "reference:fixture-1",
    expectedVersion: targetVersion,
    patch,
    citations: [{ source_id: "source:order-1", source_version: sourceVersion, pinpoint: "p. 4", claim: "Fixture claim" }],
    rationale: "Preserve the reviewed source correction.",
  });
  assert.deepEqual(args.patch, patch);
  assert.equal(args.expected_version, targetVersion);
  assert.equal(args.citations[0].source_version, sourceVersion);
  assert.equal(args.citations[0].claim, "Fixture claim");
});

test("new source self-capture alone accepts absent citation version", () => {
  const args = buildLibraryProposalArguments({
    id: "source:new-capture",
    expectedVersion: "absent",
    patch: { official_url: "https://example.test/primary" },
    citations: [{ source_id: "source:new-capture", source_version: "absent", pinpoint: "official record", claim: "Primary-source capture" }],
    rationale: "Capture the primary record for validation.",
  });
  assert.equal(args.citations[0].source_version, "absent");
  assert.throws(() => buildLibraryProposalArguments({
    id: "source:other-capture",
    expectedVersion: "absent",
    patch: { official_url: "https://example.test/primary" },
    citations: args.citations,
    rationale: "Capture the primary record for validation.",
  }), /exact source case_record version/);
});

test("non-self citations require the exact current source version", () => {
  const input = {
    id: "reference:fixture-2",
    expectedVersion: targetVersion,
    patch: { body: "edited" },
    citations: [{ source_id: "source:order-1", source_version: "absent", pinpoint: "p. 4", claim: "Claim" }],
    rationale: "Update after checking the original.",
  };
  assert.throws(() => buildLibraryProposalArguments(input), /exact source case_record version/);
  assert.doesNotThrow(() => buildLibraryProposalArguments({ ...input, expectedVersion: "absent", citations: [{ ...input.citations[0], source_version: sourceVersion }] }));
});

test("publish requires the full proposal identity and a separately read matching validation receipt", () => {
  const id = "library_proposal:2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4";
  const hash = "sha256:" + "e".repeat(64);
  const proposal = { proposal_id: id, proposed_hash: hash, status: "pending_validation" };
  const row = { id, version: targetVersion, record: { proposed_hash: hash, status: "pending_validation" } };
  const receipt = { id: id.replace("library_proposal:", "library_validation:"), version: sourceVersion, record: { proposal_id: id, proposed_hash: hash, status: "VERIFIED_PRIMARY", currency_status: "cleared" } };
  assert.equal(libraryProposalRecordId(proposal.proposal_id), id);
  assert.throws(() => libraryProposalRecordId(id.split(":")[1]), /valid UUID/);
  assert.equal(canPublishReviewedProposal(proposal, row, receipt), true);
  assert.equal(canPublishReviewedProposal(proposal, row, null), false);
  assert.equal(canPublishReviewedProposal(proposal, row, { ...receipt, record: { ...receipt.record, currency_status: "provisional" } }), false);
  assert.equal(canPublishReviewedProposal(proposal, row, { ...receipt, record: { ...receipt.record, proposed_hash: "sha256:" + "f".repeat(64) } }), false);
  assert.equal(canPublishReviewedProposal(proposal, { ...row, record: { ...row.record, status: "published" } }, receipt), false);
});

test("queue state is surfaced and retry is enabled only for retryable failures", () => {
  const proposal = { proposal_id: "2eaebd0e-e1c2-4fd8-9d0a-7515ed54b8b4", dispatch: { state: "queue_failed", reason: "temporary", retryable: true } };
  const record = { id: `library_proposal:${proposal.proposal_id}`, version: targetVersion, record: { dispatch: { state: "queued", workflow_id: "wf-fixture", run_id: "run-fixture", retryable: false } } };
  assert.deepEqual(proposalDispatchState(proposal, record), record.record.dispatch);
  assert.deepEqual(proposalDispatchState(proposal, null), proposal.dispatch);
  assert.equal(canRetryLibraryValidation({ state: "queue_failed", retryable: true }), true);
  assert.equal(canRetryLibraryValidation({ state: "queue_failed", retryable: false }), false);
  assert.equal(canRetryLibraryValidation({ state: "queued", retryable: true }), false);
});

test("tool errors and malformed results cannot masquerade as validation", () => {
  assert.equal(structuredToolResult({ state: "tool_error", structured: { currency_status: "cleared" } }), null);
  assert.equal(structuredToolResult({ state: "ok", text: ["not json"], structured: null }), null);
  assert.deepEqual(structuredToolResult({ state: "ok", text: [], structured: { currency_status: "cleared" } }), { currency_status: "cleared" });
  assert.match(toolFailureMessage({ state: "tool_error", text: ["version conflict"] }), /version conflict/);
  assert.equal(libraryProposalFailureKind("record version conflict"), "conflict");
  assert.equal(libraryProposalFailureKind("citation source version failed validation"), "validation");
  assert.throws(() => libraryProposalRecordId("not-a-uuid"), /valid UUID/);
});
