import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

/** Load actual pure source helpers rather than reproducing routing or attribution in test doubles.
 * Input: module path. Output: real exports. Effects: bounded in-memory compilation only.
 * Pick for native and legacy Process contracts without starting live workflows.
 */

/** The source helper now imports ApiError from the runtime client.
 * Its operations remain real; only this unavailable browser import is substituted.
 * No network actions or source mutations occur in these VM tests.
 */
function stubApiClient(id) {
  if (id !== "@/lib/api-client") throw new Error(`Unexpected runtime dependency: ${id}`);
  return {
    ApiError: class ApiError extends Error {
      constructor(message, status, body) {
        super(message);
        this.status = status;
        this.body = body;
      }
    }
  };
}

function loadHelpers(path) {
  const module = { exports: {} };
  runInNewContext(ts.transpileModule(readFileSync(new URL(path, import.meta.url), "utf8"), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText, { module, exports: module.exports, Error, URL, require: stubApiClient });
  return module.exports;
}
const { confirmedCaseOwner, sourcePerspectiveScope, processingIdentity, requiresMessagePerspective } = loadHelpers("../src/components/sources/source-perspective.ts");
const { isSMSContextSource, humanContextAttribution } = loadHelpers("../src/lib/human-context.ts");
const { processSelection } = loadHelpers("../src/components/sources/process-selection.ts");
const { pinnedSourceRef } = loadHelpers("../src/components/sources/source-navigation.ts");
const people = [
  { id: "other-person", role_in_case: "co_parent", verification_state: "confirmed" },
  { id: "registry-owner", role_in_case: "user", verification_state: "confirmed" },
];

test("checked files expose the source perspective choice even while a folder remains selected", () => {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let condition;
  function visit(node) {
    if (ts.isJsxExpression(node) && node.expression && ts.isBinaryExpression(node.expression)
      && node.expression.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandToken
      && node.expression.right.getText(ast).includes('aria-label="Whose phone/export?"')) condition = node.expression.left.getText(ast);
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.ok(condition);
  assert.equal(Boolean(runInNewContext(condition, { needsPerspective: true, showContentSearch: false })), true);
  assert.equal(Boolean(runInNewContext(condition, { needsPerspective: false, showContentSearch: false })), false);
  assert.equal(Boolean(runInNewContext(condition, { needsPerspective: true, showContentSearch: true })), false);
});

/** Execute the real Sources Process callback with bounded transport stubs.
 * Inputs: explicit source selection/registry state. Output: callback and captured requests.
 * Effects: in-memory TS compilation only; never accesses a corpus or starts a live workflow.
 */
function processHarness(overrides = {}) {
  const source = readFileSync(new URL("../src/components/sources/sources-screen.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("sources-screen.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const callbacks = new Map();
  function visit(node) {
    if (ts.isFunctionDeclaration(node) && ["process", "submitHumanSource"].includes(node.name?.text)) callbacks.set(node.name.text, node.getText(ast));
    ts.forEachChild(node, visit);
  }
  visit(ast);
  assert.equal(callbacks.size, 2);
  const requests = [];
  const humanRequests = [];
  const humanKeys = [];
  const versions = [];
  const workflows = [];
  const errors = [];
  const file = { source_ref: "b2://bucket/sms-20251102230302.xml", name: "sms-20251102230302.xml", etag: "version-a", byte_length: 1000 };
  const scope = sourcePerspectiveScope("LIVE", [[file.source_ref, file.etag, file.byte_length]]);
  const context = {
    matter: { id: "matter-registry" }, primaryCourtCase: { id: "court-registry" }, mode: "LIVE",
    processBusy: { current: false }, showContentSearch: false, processTarget: "selected", unsupportedMedia: false, hasAIProcessTarget: false, aiProcessBlocked: false, addAIChatToContext: async () => { throw new Error("Unexpected AI start"); }, setProcessing() {}, setProcessError: (error) => errors.push(error), setProcessMessage() {}, setUploadProgress() {}, setActivityHref() {}, setAlreadyImported() {},
    registryPeople: people, perspectiveScope: scope, perspectiveChoice: { scope, personId: "registry-owner" },
    localSource: null, processFiles: [file], selectedFiles: [], selectedObject: file, selectedFolder: null,
    submissions: { current: new Map() }, handlerOverride: "", runsQuery: { refetch() {} },
    humanSubmissions: { current: new Map() }, activeRootId: "actual-root",
    inspectionQuery: { data: null }, restored: null,
    setHumanWorkflows: (update) => { const next = update(workflows); workflows.splice(0, workflows.length, ...next); },
    processingIdentity, requiresMessagePerspective, isSMSContextSource, humanContextAttribution, processSelection, pinnedSourceRef,
    detectedFormat: () => "xml", URLSearchParams,
    crypto: { randomUUID: () => `request-${requests.length + humanRequests.length}` }, errorText: (error) => error.message,
    getProfferSourceVersion: async (object, root, signal, expectedVersion) => { versions.push({ object, root, expectedVersion }); return { source_ref: object.source_ref, provider_version_id: object.etag, etag: object.etag, byte_length: object.byte_length }; },
    previewProfferSource: async (object) => ({ source_ref: object.source_ref, provider_version_id: null }),
    startHumanContextSource: async (request, key) => { humanRequests.push(request); humanKeys.push(key); return { workflow_id: `native-${humanRequests.length}`, run_id: `run-${humanRequests.length}` }; },
    startProffer: async (request) => { requests.push(request); return { preview_handle: "same-handle" }; },
    Error,
    ...overrides,
  };
  // Match the component's active explicit perspective after every owner/source change.
  // This getter is recomputed when the harness mutates the selected identity.
  Object.defineProperty(context, "effectivePerspective", {
    get: () => context.perspectiveChoice?.scope === context.perspectiveScope
      ? context.perspectiveChoice
      : null,
  });
  const js = ts.transpileModule([...callbacks.values()].join("\n") + "\nglobalThis.invokeProcess = process;", { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(js, context);
  return { context, invoke: context.invokeProcess, requests, humanRequests, humanKeys, versions, workflows, errors, file, scope };
}

test("owner comes from unique confirmed registry user, never first person or candidate", () => {
  assert.equal(confirmedCaseOwner(people).id, "registry-owner");
  assert.equal(confirmedCaseOwner([...people, { id: "candidate", role_in_case: "user", verification_state: "candidate" }]).id, "registry-owner");
  assert.equal(confirmedCaseOwner([...people, { id: "ambiguous", role_in_case: "user", verification_state: "confirmed" }]), null);
});

test("content search and unsupported media disarm processing before any source/version or starter call", async () => {
  for (const change of [{ showContentSearch: true }, { unsupportedMedia: true }, { processTarget: null }]) {
    const harness = processHarness(change);
    await harness.invoke();
    assert.equal(harness.requests.length, 0);
    assert.equal(harness.humanRequests.length, 0);
    assert.equal(harness.versions.length, 0);
  }
});

test("explicit AI Process uses only native context and never falls back to legacy normalization", async () => {
  for (const rejected of [false, true]) {
    let nativeCalls = 0;
    const harness = processHarness({ hasAIProcessTarget: true, matter: null, primaryCourtCase: null,
      addAIChatToContext: async () => { nativeCalls++; if (rejected) throw new Error("Native context rejected source version"); } });
    if (rejected) await assert.rejects(harness.invoke(), /Native context rejected/);
    else await harness.invoke();
    assert.equal(nativeCalls, 1);
    assert.equal(harness.requests.length, 0);
    assert.equal(harness.humanRequests.length, 0);
  }
  let nativeCalls = 0;
  const blocked = processHarness({ hasAIProcessTarget: true, aiProcessBlocked: true, addAIChatToContext: async () => { nativeCalls++; } });
  await blocked.invoke();
  assert.equal(nativeCalls, 0);
  assert.equal(blocked.requests.length, 0);
});

test("native start resolves the inspected immutable version and legacy start retains it in the supported locator", async () => {
  const native = processHarness({ inspectionQuery: { data: { provider_version_id: "inspected-version" } } });
  await native.invoke();
  assert.equal(native.versions[0].expectedVersion, "inspected-version");
  const legacy = processHarness({ detectedFormat: () => "pdf", inspectionQuery: { data: { source_ref: "b2://bucket/sms-20251102230302.xml", provider_version_id: "retained+/version" } } });
  await legacy.invoke();
  assert.equal(new URL(legacy.requests[0].source_ref).searchParams.get("versionId"), "retained+/version");
});

test("native SMS Process forwards exact source version, registry owner and explicit perspective without legacy admission", async () => {
  const harness = processHarness();
  await harness.invoke();
  assert.equal(harness.humanRequests.length, 1);
  assert.equal(harness.requests.length, 0, "native SMS must not enter the legacy starter");
  assert.equal(harness.humanRequests[0].owner_person_ref, "registry-owner");
  assert.equal(harness.humanRequests[0].perspective_person_ref, "registry-owner");
  assert.equal(harness.humanRequests[0].perspective, "source-device");
  assert.equal(harness.humanRequests[0].source_ref, harness.file.source_ref);
  assert.equal(harness.humanRequests[0].provider_version_id, harness.file.etag);
  assert.equal(harness.humanRequests[0].byte_length, harness.file.byte_length);
  assert.equal(harness.humanRequests[0].declared_format, "smsbackuprestore_xml");
  assert.equal(harness.humanRequests[0].contract_version, "context-v1");
  assert.equal(harness.versions[0].root, "actual-root");
  assert.equal(harness.workflows.length, 1);
  await harness.invoke();
  assert.equal(harness.humanRequests.length, 1, "same accepted native request is not submitted twice");
  assert.equal(harness.versions.length, 1, "accepted native source is not resolved again");
  harness.context.perspectiveChoice = { scope: harness.scope, personId: "other-person" };
  await harness.invoke();
  assert.equal(harness.humanRequests.length, 2, "changed perspective must not reuse an accepted request");
  assert.equal(harness.humanRequests[1].perspective_person_ref, "other-person");
  assert.notEqual(harness.humanKeys[0], harness.humanKeys[1]);
});

test("changing selected source, version, mode or membership invalidates a previous choice", async () => {
  const harness = processHarness();
  harness.context.perspectiveScope = sourcePerspectiveScope("LIVE", [[harness.file.source_ref, "version-b", 1000]]);
  await harness.invoke();
  assert.equal(harness.requests.length, 0);
  assert.equal(harness.humanRequests.length, 0);
  assert.equal(harness.versions.length, 0, "stale attribution is rejected before resolving the source");
  assert.match(harness.errors.at(-1), /Choose whose phone/);
  assert.notEqual(harness.scope, sourcePerspectiveScope("LIVE", [["another-file", "version-a", 1000]]));
  assert.notEqual(harness.scope, sourcePerspectiveScope("TEST", [[harness.file.source_ref, "version-a", 1000]]));
  assert.notEqual(harness.scope, sourcePerspectiveScope("LIVE", [[harness.file.source_ref, "version-a", 1000], ["second-file", null, 12]]));
});

test("legacy message processing still rejects missing or ambiguous confirmed owner before submission", async () => {
  for (const registryPeople of [[], [...people, { id: "another-owner", role_in_case: "user", verification_state: "confirmed" }]]) {
    const file = { source_ref: "b2://bucket/calls-backup.xml", name: "calls-backup.xml", etag: "calls-v1", byte_length: 1000 };
    const harness = processHarness({ registryPeople, selectedObject: file });
    await harness.invoke();
    assert.equal(harness.requests.length, 0);
    assert.equal(harness.humanRequests.length, 0);
    assert.match(harness.errors.at(-1), /one confirmed owner/);
    assert.equal(harness.context.processBusy.current, false);
  }
});

test("native SMS requires confirmed explicit perspective but never guesses an absent or ambiguous owner", async () => {
  for (const registryPeople of [people.filter((person) => person.role_in_case !== "user"), [...people, { id: "ambiguous-owner", role_in_case: "user", verification_state: "confirmed" }]]) {
    const harness = processHarness({ registryPeople, matter: null, primaryCourtCase: null });
    harness.context.perspectiveChoice = { scope: harness.scope, personId: "other-person" };
    await harness.invoke();
    assert.equal(harness.humanRequests.length, 1);
    assert.equal(harness.requests.length, 0);
    const request = harness.humanRequests[0];
    assert.equal(request.perspective_person_ref, "other-person");
    for (const field of ["owner_person_ref", "matter_id", "court_case_id"]) assert.equal(Object.hasOwn(request, field), false, field);
  }
  for (const choice of [null, { personId: "unconfirmed", scope: "current" }]) {
    const harness = processHarness({ perspectiveChoice: choice });
    await harness.invoke();
    assert.equal(harness.humanRequests.length, 0);
    assert.equal(harness.requests.length, 0);
    assert.match(harness.errors.at(-1), /Choose whose phone/);
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
  assert.deepEqual(harness.humanRequests.map((request) => request.source_ref), files.map((file) => file.source_ref));
  assert.equal(harness.requests.length, 0);
  assert.ok(harness.humanRequests.every((request) => request.perspective_person_ref === "other-person" && request.owner_person_ref === "registry-owner"));
});

test("non-message sources keep normal Process without a perspective requirement", () => {
  assert.equal(requiresMessagePerspective("report.xml", "xml"), false);
  assert.equal(requiresMessagePerspective("chatgpt.json", "message_export_json"), false);
  assert.equal(requiresMessagePerspective("sms-20251102230302.xml", "xml"), true);
  assert.deepEqual(JSON.parse(JSON.stringify(processingIdentity([], "selection", null, false))), {});
});

test("legacy selected messages retain exact attribution, accepted receipts and failed-request identity on retry", async () => {
  const files = [
    { source_ref: "b2://bucket/calls-one.xml", name: "calls-one.xml", etag: "one", byte_length: 10 },
    { source_ref: "b2://bucket/calls-two.xml", name: "calls-two.xml", etag: "two", byte_length: 20 },
  ];
  const scope = sourcePerspectiveScope("LIVE", files.map((file) => [file.source_ref, file.etag, file.byte_length]));
  let failSecond = true;
  const attempts = [];
  let serial = 0;
  const harness = processHarness({ selectedFiles: files, processFiles: files, perspectiveScope: scope, perspectiveChoice: { scope, personId: "other-person" },
    crypto: { randomUUID: () => `legacy-request-${++serial}` },
    startProffer: async (request) => { attempts.push(request); if (request.source_ref === files[1].source_ref && failSecond) throw new Error("Transport unavailable"); return { preview_handle: request.source_ref }; },
  });
  await harness.invoke();
  assert.equal(harness.humanRequests.length, 0);
  assert.equal(attempts.length, 2);
  assert.match(harness.errors.at(-1), /Transport unavailable/);
  for (const request of attempts) {
    assert.equal(request.owner_person_id, "registry-owner");
    assert.equal(request.perspective_person_id, "other-person");
    assert.equal(request.matter_id, "matter-registry");
    assert.equal(request.court_case_id, "court-registry");
    assert.equal(request.declared_format, "xml");
  }
  failSecond = false;
  await harness.invoke();
  assert.deepEqual(attempts.map((request) => request.source_ref), [files[0].source_ref, files[1].source_ref, files[1].source_ref]);
  assert.equal(attempts[2].request_id, attempts[1].request_id, "failed legacy request is retried with its original identity");
  assert.equal(harness.context.processBusy.current, false);
});

test("legacy Process requires a fixed case and rejects a stale explicit message perspective", async () => {
  const file = { source_ref: "b2://bucket/calls-backup.xml", name: "calls-backup.xml", etag: "calls-v1", byte_length: 1000 };
  for (const overrides of [{ matter: null }, { primaryCourtCase: null }, { perspectiveChoice: null }]) {
    const harness = processHarness({ selectedObject: file, ...overrides });
    await harness.invoke();
    assert.equal(harness.requests.length, 0);
    assert.equal(harness.humanRequests.length, 0);
    assert.match(harness.errors.at(-1), overrides.perspectiveChoice === null ? /Choose whose phone/ : /fixed case/);
  }
});

test("local upload retry reuses acquisition but starts a new request when perspective changes", async () => {
  let uploads = 0;
  const local = { requestId: "local-acquisition", file: { name: "sms-20251102230302.xml", size: 1000 } };
  const scope = sourcePerspectiveScope("LIVE", [[local.requestId, null, local.file.size]]);
  const harness = processHarness({ localSource: local, perspectiveScope: scope, perspectiveChoice: { scope, personId: "registry-owner" },
    localUploads: { current: new Map() }, uploadProfferSource: async () => { uploads++; return { acquisition_ref: "sealed-source", byte_length: 1000 }; } });
  await harness.invoke();
  await harness.invoke();
  harness.context.perspectiveChoice = { scope, personId: "other-person" };
  await harness.invoke();
  assert.equal(uploads, 1);
  assert.equal(harness.humanRequests.length, 2);
  assert.equal(harness.requests.length, 0);
  assert.notEqual(harness.humanKeys[0], harness.humanKeys[1]);
  assert.equal(harness.humanRequests[1].source_ref, "sealed-source");
  assert.equal(harness.humanRequests[1].byte_length, 1000);
});
