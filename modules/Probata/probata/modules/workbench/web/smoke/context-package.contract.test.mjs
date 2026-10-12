// Byline: Codex · GPT-6 · 2026-10-12.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

/** Load the pure display adapter with no browser, network or private source data.
 * Input: source file. Output: real module exports. Effects: in-memory transpilation only.
 * Pick for preserving source/correction separation independently of UI snapshots.
 */
function reviewAdapter() {
  const module = { exports: {} };
  const source = readFileSync(new URL("../src/lib/context-package-review.ts", import.meta.url), "utf8");
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(code, { module, exports: module.exports });
  return module.exports;
}

test("reopened correction updates all display values while keeping the exact original fields and pins", () => {
  const { reviewedPackageItem } = reviewAdapter();
  const original = { id: "actual-item", kind: "entity", label: "Original", fields: { name: "Original", source_device: "d1", person_ref: null }, source_refs: [{ source_version_ref: "v1", locator: { ordinal: 7 } }], edges: [], revision: "saved-ref", capabilities: [], review_state: "accepted" };
  const item = { ...original, correction: { fields: { name: "Reviewed", person_ref: "p1" }, source_refs: original.source_refs, review_state: "accepted" } };
  const displayed = reviewedPackageItem(item);
  assert.equal(displayed.label, "Reviewed");
  assert.equal(displayed.fields.person_ref, "p1");
  assert.equal(displayed.fields.source_device, "d1");
  assert.equal(displayed.original_fields, original.fields);
  assert.equal(original.fields.person_ref, null);
  assert.equal(displayed.source_refs, original.source_refs);
  assert.equal(displayed.revision, "saved-ref");
});

test("an actual native edge row stays a visible edge without inventing identity relationships", () => {
  const { reviewedPackageItem } = reviewAdapter();
  const fields = { edge_id: "edge-1", kind: "reported_delivery_of", from_node_id: "provider-event", to_node_id: "message", source_pins: [{ locator: { record: 4 } }] };
  const displayed = reviewedPackageItem({ id: "edge-1", fields, edges: [], capabilities: [], source_refs: [], review_state: "unreviewed" });
  assert.equal(displayed.edges.length, 1);
  assert.equal(displayed.edges[0], fields);
  const isolated = reviewedPackageItem({ id: "entity-1", fields: { account_ref: "a1", person_ref: "p1" }, edges: [], capabilities: [], source_refs: [] });
  assert.equal(isolated.edges.length, 0);
});

test("complete nested document and native message text are not clipped to grid-preview length", () => {
  const { packageTextBlocks } = reviewAdapter();
  const complete = "Source original ".repeat(2000);
  const blocks = packageTextBlocks({ native: { body: complete, occurred_at: "approximately noon" }, metadata: { description: "Provider description" } });
  assert.equal(blocks[0].path, "native.body");
  assert.equal(blocks[0].text, complete);
  assert.equal(blocks[1].path, "metadata.description");
});

test("canonical candidate edits preserve operation guards and cannot patch source envelopes", () => {
  const { packageEditorFields, packageReviewInput } = reviewAdapter();
  const candidate = { name: "Observed", confidence: 0.7, statement: "Original assertion" };
  const item = { id: "display-item", revision: "display-revision", source_refs: ["display-pin"], fields: { id: "actual-working-uuid", attrs: { candidate }, source_text: "Retained original" } };
  const capability = { operation: "edit", scope: "canonical_working_candidate", editable_fields: ["name", "confidence", "statement", "occurred_at"], body: { item_id: "actual-working-uuid", revision: "authoritative-revision", candidate_content_sha256: "actual-hash", review_item_id: "stable-review-chain", source_refs: ["authoritative-pin"] } };
  assert.deepEqual(JSON.parse(JSON.stringify(packageEditorFields(item, capability))), { name: "Observed", confidence: 0.7, statement: "Original assertion", occurred_at: null });
  const request = packageReviewInput(item, capability, { name: "Reviewed", confidence: 0.8 }, "Owner correction", { revision: "stale", candidate_content_sha256: "wrong" });
  for (const [key, value] of Object.entries(capability.body)) assert.equal(request[key], value);
  assert.equal(request.fields.name, "Reviewed");
  assert.equal(request.fields.confidence, 0.8);
  assert.equal(candidate.name, "Observed");
  assert.throws(() => packageReviewInput(item, capability, { attrs: { candidate: { name: "Bypass" } } }, "Invalid"), /not an editable/);
  assert.throws(() => packageReviewInput(item, capability, { id: "replacement" }, "Invalid"), /not an editable/);
  assert.equal(packageEditorFields(item, { operation: "metadata" }), item.fields, "ordinary metadata keeps its full field panel");
});
