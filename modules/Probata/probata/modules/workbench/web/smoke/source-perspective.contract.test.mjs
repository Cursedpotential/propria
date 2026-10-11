import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

const module = { exports: {} };
runInNewContext(ts.transpileModule(readFileSync(new URL("../src/components/sources/source-perspective.ts", import.meta.url), "utf8"), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
}).outputText, { module, exports: module.exports });
const { confirmedCaseOwner, sourcePerspectiveScope, processingIdentity, requiresMessagePerspective } = module.exports;
const people = [
  { id: "other-person", role_in_case: "co_parent", verification_state: "confirmed" },
  { id: "registry-owner", role_in_case: "user", verification_state: "confirmed" },
];

/** Execute the real Sources Process callback with bounded transport stubs.
 * Inputs: explicit source selection/registry state. Output: callback and captured requests.
 * Effects: in-memory TS compilation only; never accesses a corpus or starts a live workflow.
 */
function processHarness(overrides = {}) {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let callback;
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && node.name?.text === "process") callback = node.getText(ast);
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.ok(callback);
  const requests = [];
  const errors = [];
  const file = { source_ref: "b2://bucket/sms-20251102230302.xml", name: "sms-20251102230302.xml", etag: "version-a", byte_length: 1000 };
  const scope = sourcePerspectiveScope("LIVE", [[file.source_ref, file.etag, file.byte_length]]);
  const context = {
    matter: { id: "matter-registry" }, primaryCourtCase: { id: "court-registry" }, mode: "LIVE",
    processBusy: { current: false }, setProcessing() {}, setProcessError: (error) => errors.push(error), setProcessMessage() {}, setUploadProgress() {}, setActivityHref() {},
    registryPeople: people, perspectiveScope: scope, perspectiveChoice: { scope, personId: "registry-owner" },
    localSource: null, processFiles: [file], selectedFiles: [], selectedObject: file, selectedFolder: null,
    submissions: { current: new Map() }, handlerOverride: "", runsQuery: { refetch() {} },
    processingIdentity, requiresMessagePerspective, detectedFormat: () => "xml", URLSearchParams,
    crypto: { randomUUID: () => `request-${requests.length}` }, errorText: (error) => error.message,
    startProffer: async (request) => { requests.push(request); return { preview_handle: "same-handle" }; },
    processSelection: async (entries, start, accepted) => {
      for (const entry of entries) { if (!entry.response) entry.response = await start(entry.request); accepted(entry); }
      return { accepted: entries.length, error: null };
    },
    ...overrides,
  };
  const js = ts.transpileModule(callback + "\nglobalThis.invokeProcess = process;", { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(js, context);
  return { context, invoke: context.invokeProcess, requests, errors, file, scope };
}

test("owner comes from unique confirmed registry user, never first person or candidate", () => {
  assert.equal(confirmedCaseOwner(people).id, "registry-owner");
  assert.equal(confirmedCaseOwner([...people, { id: "candidate", role_in_case: "user", verification_state: "candidate" }]).id, "registry-owner");
  assert.equal(confirmedCaseOwner([...people, { id: "ambiguous", role_in_case: "user", verification_state: "confirmed" }]), null);
});

test("normal SMS Process forwards existing registry owner and explicit source perspective", async () => {
  const harness = processHarness();
  await harness.invoke();
  assert.equal(harness.requests.length, 1);
  assert.equal(harness.requests[0].owner_person_id, "registry-owner");
  assert.equal(harness.requests[0].perspective_person_id, "registry-owner");
  assert.equal(harness.requests[0].source_ref, harness.file.source_ref);
  assert.equal(harness.requests[0].declared_format, "xml");
  await harness.invoke();
  assert.equal(harness.requests.length, 1, "same admitted request is not submitted twice");
  harness.context.perspectiveChoice = { scope: harness.scope, personId: "other-person" };
  await harness.invoke();
  assert.equal(harness.requests.length, 2, "changed perspective must not reuse an accepted request");
  assert.equal(harness.requests[1].perspective_person_id, "other-person");
});

test("changing selected source, version, mode or membership invalidates a previous choice", async () => {
  const harness = processHarness();
  harness.context.perspectiveScope = sourcePerspectiveScope("LIVE", [[harness.file.source_ref, "version-b", 1000]]);
  await harness.invoke();
  assert.equal(harness.requests.length, 0);
  assert.match(harness.errors.at(-1), /Choose whose phone/);
  assert.notEqual(harness.scope, sourcePerspectiveScope("LIVE", [["another-file", "version-a", 1000]]));
  assert.notEqual(harness.scope, sourcePerspectiveScope("TEST", [[harness.file.source_ref, "version-a", 1000]]));
  assert.notEqual(harness.scope, sourcePerspectiveScope("LIVE", [[harness.file.source_ref, "version-a", 1000], ["second-file", null, 12]]));
});

test("missing or ambiguous confirmed owner prevents an incorrectly attributed SMS submission", async () => {
  for (const registryPeople of [[], [...people, { id: "another-owner", role_in_case: "user", verification_state: "confirmed" }]]) {
    const harness = processHarness({ registryPeople });
    await harness.invoke();
    assert.equal(harness.requests.length, 0);
    assert.match(harness.errors.at(-1), /one confirmed owner/);
    assert.equal(harness.context.processBusy.current, false);
  }
});

test("explicit selection applies perspective only to the selected files, not another source", async () => {
  const files = [
    { source_ref: "b2://bucket/sms-one.xml", name: "sms-one.xml", etag: "one", byte_length: 10 },
    { source_ref: "b2://bucket/sms-two.xml", name: "sms-two.xml", etag: "two", byte_length: 20 },
  ];
  const scope = sourcePerspectiveScope("LIVE", files.map((file) => [file.source_ref, file.etag, file.byte_length]));
  const harness = processHarness({ selectedFiles: files, processFiles: files, perspectiveScope: scope, perspectiveChoice: { scope, personId: "other-person" } });
  await harness.invoke();
  assert.deepEqual(harness.requests.map((request) => request.source_ref), files.map((file) => file.source_ref));
  assert.ok(harness.requests.every((request) => request.perspective_person_id === "other-person" && request.owner_person_id === "registry-owner"));
});

test("non-message sources keep normal Process without a perspective requirement", () => {
  assert.equal(requiresMessagePerspective("report.xml", "xml"), false);
  assert.equal(requiresMessagePerspective("chatgpt.json", "message_export_json"), false);
  assert.equal(requiresMessagePerspective("sms-20251102230302.xml", "xml"), true);
  assert.deepEqual(JSON.parse(JSON.stringify(processingIdentity([], "selection", null, false))), {});
});

test("local upload retry reuses acquisition but starts a new request when perspective changes", async () => {
  let uploads = 0;
  const local = { requestId: "local-acquisition", file: { name: "sms-20251102230302.xml", size: 1000 } };
  const scope = sourcePerspectiveScope("LIVE", [[local.requestId, null, local.file.size]]);
  const harness = processHarness({ localSource: local, perspectiveScope: scope, perspectiveChoice: { scope, personId: "registry-owner" },
    localUploads: { current: new Map() }, uploadProfferSource: async () => { uploads++; return { acquisition_ref: "sealed-source" }; } });
  await harness.invoke();
  await harness.invoke();
  harness.context.perspectiveChoice = { scope, personId: "other-person" };
  await harness.invoke();
  assert.equal(uploads, 1);
  assert.equal(harness.requests.length, 2);
  assert.notEqual(harness.requests[0].request_id, harness.requests[1].request_id);
  assert.equal(harness.requests[1].source_ref, "sealed-source");
});
