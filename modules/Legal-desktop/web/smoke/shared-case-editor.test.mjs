// Byline: OpenAI Codex · GPT-6-Luna · 2026-10-04.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {
  buildPersonalCasePutArguments,
  isPersonalCaseVersionConflict,
  personalRecordForEditing,
  personalCaseRecordRef,
} from "../src/lib/shared-case-record.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const version = `sha256:${"a".repeat(64)}`;

/** Verifies a person update preserves PII/unknown baseline values and sends only the changed field.
 * Inputs: synthetic full-name record values. Outputs: assertion result. Effects: no I/O or database mutation.
 * Choose as a focused behavior check for the guarded case_put argument builder.
 */
test("existing personal updates use the exact version and changed-field patch", () => {
  const original = {
    name: "Jordan Example",
    narrative: "A private synthetic family detail",
    extra_private_field: { source_note: "Keep this unchanged" },
  };
  const args = buildPersonalCasePutArguments({
    table: "person",
    id: "person:jordan-example",
    expectedVersion: version,
    original,
    edited: { ...original, narrative: "Updated synthetic family detail" },
  });

  assert.equal(args.expected_version, version);
  assert.equal(args.id, "jordan-example");
  assert.deepEqual(args.data, { narrative: "Updated synthetic family detail" });
  assert.equal(original.name, "Jordan Example");
  assert.deepEqual(original.extra_private_field, { source_note: "Keep this unchanged" });
});

/** Verifies personal case-document imports retain full synthetic PII and always carry the source-kind guard.
 * Inputs: new and existing synthetic source rows. Outputs: assertion result. Effects: no I/O or database mutation.
 * Choose for the source special case because authority sources must stay on the citation proposal path.
 */
test("case_document writes retain full values and include kind in the guarded patch", () => {
  const created = buildPersonalCasePutArguments({
    table: "source",
    id: "source:intake-example",
    expectedVersion: "absent",
    edited: { title: "Jordan Example intake", full_text: "Synthetic private narrative" },
    original: null,
  });
  assert.equal(created.expected_version, "absent");
  assert.equal(created.id, "intake-example");
  assert.deepEqual(created.data, {
    title: "Jordan Example intake",
    full_text: "Synthetic private narrative",
    kind: "case_document",
  });

  const existing = buildPersonalCasePutArguments({
    table: "source",
    id: "source:intake-1",
    expectedVersion: version,
    original: { kind: "case_document", title: "Jordan Example intake", full_text: "Synthetic private narrative" },
    edited: { kind: "case_document", title: "Jordan Example intake", full_text: "Updated synthetic narrative" },
  });
  assert.deepEqual(existing.data, { full_text: "Updated synthetic narrative", kind: "case_document" });
});

/** Verifies legal source rows and stale/malformed version requests fail before any hosted write.
 * Inputs: invalid synthetic editor states. Outputs: assertion result. Effects: no remote calls or mutations.
 * Choose to prove an existing authority cannot be converted into a personal case_document.
 */
test("editor refuses authority conversion and rejects unversioned existing writes", () => {
  assert.throws(() => buildPersonalCasePutArguments({
    table: "source", id: "source:authority-1", expectedVersion: version,
    original: { kind: "primary_authority", citation: "Synthetic authority" },
    edited: { kind: "case_document", citation: "Synthetic authority" },
  }), /Only personal source records/);
  assert.throws(() => buildPersonalCasePutArguments({
    table: "person", id: "person:jordan-example", expectedVersion: "absent",
    original: { name: "Jordan Example" }, edited: { name: "Jordan Example Two" },
  }), /absent-version create cannot be based/);
  assert.throws(() => buildPersonalCasePutArguments({
    table: "reference", expectedVersion: "absent", edited: { title: "Synthetic legal reference" }, original: null,
  }), /only writes personal case tables/);
  assert.throws(() => buildPersonalCasePutArguments({
    table: "person", id: "person:jordan-example", expectedVersion: version,
    original: { name: "Jordan Example" }, edited: { id: "person:other", name: "Jordan Example Two" },
  }), /Managed id and embedding fields/);
  assert.throws(() => buildPersonalCasePutArguments({
    table: "person", id: "person:jordan-example", expectedVersion: version.toUpperCase(),
    original: { name: "Jordan Example" }, edited: { name: "Jordan Example Two" },
  }), /exact current case_record version/);
});

/** Verifies the form hides only server-managed fields while retaining all synthetic personal/unknown values.
 * Inputs: a synthetic shared-record snapshot. Outputs: editable snapshot assertion. Effects: none.
 * Choose before rendering so a person’s content stays editable without exposing managed identity/vector fields.
 */
test("editable projection preserves personal and unknown fields but omits managed fields", () => {
  assert.deepEqual(personalRecordForEditing({
    id: "person:jordan-example",
    embedding: [0.01, 0.02],
    name: "Jordan Example",
    unknown_private_field: "Synthetic preserved value",
  }), {
    name: "Jordan Example",
    unknown_private_field: "Synthetic preserved value",
  });
});

/** Verifies case_put's bare record key becomes an exact full case_record reference and conflicts are recognized.
 * Inputs: a synthetic successful write receipt and conflict text. Outputs: assertion result. Effects: none.
 * Choose for post-write exact readback identity and stale-version UI behavior.
 */
test("write receipts are qualified for exact readback and version conflicts remain classifiable", () => {
  assert.equal(personalCaseRecordRef("child", { table: "child", id: "casey-example" }), "child:casey-example");
  assert.equal(isPersonalCaseVersionConflict("expected_version conflict: record changed"), true);
  assert.equal(isPersonalCaseVersionConflict("toolkit console unavailable"), false);
  assert.throws(() => personalCaseRecordRef("child", { table: "person", id: "casey-example" }), /expected table/);
});

/** Ensures the client component keeps the shared BFF and exact readback path as its only persistence route.
 * Inputs: component source. Outputs: assertion result. Effects: reads source text only.
 * Choose as a wiring smoke check alongside helper behavior tests; no browser or local database is started.
 */
test("editor writes through MCP invocation and rereads through case_record", async () => {
  const source = await readFile(path.join(root, "src/components/SharedCaseRecordEditor.tsx"), "utf8");
  assert.match(source, /\/v1\/mcp\/tools/);
  assert.match(source, /\/v1\/mcp\/invocations/);
  assert.match(source, /findTool\("case_put", true\)/);
  assert.match(source, /findTool\("case_record", false\)/);
  assert.match(source, /confirm_write:\s*tool\.writes/);
  assert.match(source, /expectedVersion:\s*version/);
});
