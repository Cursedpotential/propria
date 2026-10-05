// Byline: Codex · GPT-6 · 2026-10-04. Pure shared-store configuration checks; no database or source writes.
import assert from "node:assert/strict";
import test from "node:test";
import { resolveConfiguredDbUrl } from "../dist/store.js";
import { putVersionedPersonalRecord, validateLibraryProposal } from "../dist/case-library.js";

test("missing shared configuration fails visibly rather than opening a private database", () => {
  for (const value of [undefined, "", "   "]) assert.throws(() => resolveConfiguredDbUrl(value), /Shared case store is not configured/);
});
test("implicit embedded configuration cannot divide the shared case store", () => {
  for (const value of ["mem://", "rocksdb:///tmp/case.db", "C:/case.db"]) assert.throws(() => resolveConfiguredDbUrl(value), /shared hosted SurrealDB/);
});
test("shared service URLs retain their exact endpoint", () => {
  for (const value of ["ws://surreal-case:8000/rpc", "https://case.example/rpc", "wss://case.example/rpc"]) assert.equal(resolveConfiguredDbUrl(`  ${value}  `), value);
});
test("explicit isolated callers retain memory and Windows-path compatibility", () => {
  assert.equal(resolveConfiguredDbUrl("mem://", true), "mem://");
  assert.equal(resolveConfiguredDbUrl("C:/case.db", true), "rocksdb:C:/case.db");
  assert.equal(resolveConfiguredDbUrl("rocksdb:C:/case.db", true), "rocksdb:C:/case.db");
});

test("ordinary edits cannot redirect server-owned Case Bible file bindings", async () => {
  const store = { db: { query() { throw new Error("Rejected binding edit reached the database"); } } };
  await assert.rejects(putVersionedPersonalRecord(store, "person", "private-context",
    { library_file_id: "library_file:" + "f".repeat(64) }, "absent"), /server-owned fields/);
  assert.throws(() => validateLibraryProposal({
    id: "reference:guide", expected_version: "absent",
    patch: { library_file_id: "library_file:" + "f".repeat(64) },
    citations: [{ source_id: "source:primary", source_version: "sha256:" + "a".repeat(64), pinpoint: "1", claim: "claim" }],
    rationale: "change",
  }), /reserved publication field/);
});
