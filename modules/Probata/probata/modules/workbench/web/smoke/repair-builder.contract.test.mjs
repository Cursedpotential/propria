// Byline: Claude Code · Opus 5.5 · 2026-09-26 (repair workflow builder contract)
// Review Actions → Repair: propose, compose, validate, run, poll, re-entry link; the Review
// catalog's "N runs hidden: mode unknown" flag. Owner rules pinned here: "one flag, no
// disclaimers", and a UI never describes what he cannot do (nothing for agent_available).
//
// Three layers: the pure helpers run as-is (Node strips their types); the builder's own
// components are rendered to static markup with their real UI primitives (only the network
// client is replaced, and nothing here calls it); the wiring is pinned by source contract.
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { runInThisContext } from "node:vm";

import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

import {
  MAX_REPAIR_STEPS,
  addStep,
  draftFromProposal,
  moveStep,
  newPlanId,
  noRepairLine,
  noStepsProposed,
  numberParam,
  paramFields,
  planFromDraft,
  planKey,
  removeStep,
  repairReentryFromStages,
  setParam,
  summaryLine,
} from "../src/lib/repair-plan.ts";

const require = createRequire(import.meta.url);
const SRC = fileURLToPath(new URL("../src/", import.meta.url));

function source(path) {
  return readFileSync(new URL(path, import.meta.url), "utf8");
}

// --- a tiny TSX loader for the builder's components -------------------------------------

const neverCalled = () => {
  throw new Error("the network client is not called while rendering");
};
const apiClient = {
  RepairRunRefusedError: class RepairRunRefusedError extends Error {},
  getProfferBatch: neverCalled,
  getProfferRepairRun: neverCalled,
  listProfferRepairTools: neverCalled,
  proposeProfferRepairs: neverCalled,
  runProfferRepairPlan: neverCalled,
  validateProfferRepairPlan: neverCalled,
};
const loaded = new Map();

function resolveFile(base) {
  for (const candidate of [`${base}.tsx`, `${base}.ts`, resolve(base, "index.tsx"), resolve(base, "index.ts")]) {
    if (existsSync(candidate)) return candidate;
  }
  throw new Error(`cannot resolve ${base}`);
}

function load(file) {
  if (loaded.has(file)) return loaded.get(file).exports;
  const { outputText } = ts.transpileModule(readFileSync(file, "utf8"), {
    fileName: file,
    compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true },
  });
  const module = { exports: {} };
  loaded.set(file, module);
  const localRequire = (specifier) => {
    if (specifier === "@/lib/api-client") return apiClient;
    if (specifier.startsWith("@/")) return load(resolveFile(resolve(SRC, specifier.slice(2))));
    if (specifier.startsWith(".")) return load(resolveFile(resolve(dirname(file), specifier)));
    return require(specifier);
  };
  runInThisContext(`(function (exports, require, module) {${outputText}\n})`, { filename: file })(module.exports, localRequire, module);
  return module.exports;
}

const component = (path) => load(resolveFile(resolve(SRC, path)));
const { RepairProposalList } = component("components/sbv/repair-builder");
const { RepairStepEditor } = component("components/sbv/repair-plan-steps");
const { BatchRunList, RepairCheckList, RepairRunView, RepairedRunLine } = component("components/sbv/repair-run-view");
const { ReviewResourceList } = component("components/sbv/review-resource-list");
const { QueryClient, QueryClientProvider } = require("@tanstack/react-query");

function html(element) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderToStaticMarkup(React.createElement(QueryClientProvider, { client }, element));
}

const noop = () => {};

// --- fixtures shaped exactly like the engine's contract --------------------------------

const FIND = {
  id: "repair.find_other_version",
  description: "Find another copy of this file by name in the Case Bible catalog. Reads only; writes nothing.",
  input_types: ["any"],
  output_types: ["same-as-input"],
  params_schema: {
    type: "object",
    additionalProperties: false,
    properties: {
      max_candidates: { type: "integer", minimum: 1, maximum: 20, default: 5, description: "How many copies to report." },
      require_larger: { type: "boolean", default: false },
      include_quarantine: { type: "boolean", default: false },
    },
  },
  writes: "none",
  needs_n8n: false,
};
const SALVAGE = {
  id: "repair.salvage_truncated_xml",
  description: "Keep every record up to the last complete one and publish a new, hashed derived copy.",
  input_types: ["sms_backup_xml", "xml"],
  output_types: ["same-as-input"],
  params_schema: { type: "object", additionalProperties: false, properties: {} },
  writes: "derived",
  needs_n8n: false,
};
const LENIENT = { ...SALVAGE, id: "repair.lenient_decode", input_types: ["sms_backup_xml"], output_types: ["derived_ndjson_threads"] };
const TOOLS = [FIND, SALVAGE, LENIENT];
const HANDLE = "anchor_review_run_handle_0123456789ab";

const findProposal = {
  signature: "sms_backup_xml:truncated",
  rationale: "Look in the Case Bible catalog for another copy with the same file name.",
  steps: [{ step_id: "s1", activity: FIND.id, params: {} }],
  by: "rule",
  agent_available: false,
};
const waitProposal = {
  signature: "sms_backup_xml:truncated",
  rationale: "Wait: do not repair yet. Nothing runs from here.",
  steps: [],
  by: "rule",
  agent_available: false,
};

// --- pure helpers -----------------------------------------------------------------------

test("a picked proposal becomes an editable step list that the plan numbers in order", () => {
  let steps = draftFromProposal(findProposal);
  steps = addStep(steps, SALVAGE.id);
  steps = addStep(steps, LENIENT.id);
  assert.equal(new Set(steps.map((step) => step.key)).size, 3);
  steps = moveStep(steps, 2, -1);
  assert.deepEqual(steps.map((step) => step.activity), [FIND.id, LENIENT.id, SALVAGE.id]);
  assert.deepEqual(moveStep(steps, 0, -1).map((step) => step.activity), steps.map((step) => step.activity));
  steps = removeStep(steps, 1);
  const plan = planFromDraft({ planId: newPlanId(HANDLE), sourceRef: "b2://bucket/sms-1.xml", previewHandle: HANDLE, mode: "LIVE", steps });
  assert.deepEqual(plan.steps.map((step) => [step.step_id, step.activity]), [["s1", FIND.id], ["s2", SALVAGE.id]]);
  assert.equal(plan.preview_handle, HANDLE);
  assert.equal(plan.matter_mode, "LIVE");
  assert.match(plan.plan_id, /^[A-Za-z0-9_-]{8,96}$/);
});

test("a plan is bounded to the engine's twelve steps", () => {
  let steps = [];
  for (let index = 0; index < MAX_REPAIR_STEPS + 3; index += 1) steps = addStep(steps, SALVAGE.id);
  assert.equal(steps.length, 12);
});

test("an edit changes the plan key, so an old validation never vouches for a new plan", () => {
  const steps = draftFromProposal(findProposal);
  const base = { planId: "rp-abcdefgh", sourceRef: "b2://bucket/sms-1.xml", previewHandle: HANDLE, mode: "LIVE" };
  const before = planKey(planFromDraft({ ...base, steps }));
  const edited = [{ ...steps[0], params: setParam(steps[0].params, "max_candidates", 3) }];
  assert.notEqual(planKey(planFromDraft({ ...base, steps: edited })), before);
  assert.equal(planKey(planFromDraft({ ...base, steps })), before);
});

test("params are edited from params_schema and an emptied field falls back to the tool default", () => {
  const fields = paramFields(FIND.params_schema);
  assert.deepEqual(fields.map((field) => [field.name, field.kind, field.defaultValue]), [
    ["max_candidates", "integer", 5],
    ["require_larger", "boolean", false],
    ["include_quarantine", "boolean", false],
  ]);
  assert.equal(fields[0].maximum, 20);
  assert.deepEqual(paramFields({ type: "object", properties: { mode: { type: "string", enum: ["a", "b"] } } })[0].choices, ["a", "b"]);
  assert.deepEqual(setParam({ max_candidates: 3 }, "max_candidates", undefined), {});
  assert.equal(numberParam(""), undefined);
  assert.equal(numberParam("7"), 7);
  assert.equal(numberParam("x"), undefined);
});

test("no steps proposed opens the builder with one line; a clean file is not called unrepairable", () => {
  assert.equal(noStepsProposed([]), true);
  assert.equal(noStepsProposed([waitProposal]), true);
  assert.equal(noStepsProposed([findProposal, waitProposal]), false);
  assert.equal(noRepairLine("pdf:damaged"), "No known repair for this file type");
  assert.equal(noRepairLine("sms_backup_xml:clean"), "Nothing to repair: the repair check found this file clean.");
});

test("the repaired-run link is read from the run's own repair.reentry receipt", () => {
  const run = "repair-plan-rp-anchor-abc-0123456789ab-reentry-1a2b3c4d";
  const batch = `repair-${"a".repeat(40)}`;
  const stage = (ref, status = "success") => ({ stage: "repair.reentry", status, ref, receipt_ref: `receipt-${ref.length}` });
  assert.equal(repairReentryFromStages([]), null);
  assert.equal(repairReentryFromStages([{ stage: "repair.salvage_truncated_xml", status: "success", ref: run }]), null);
  assert.equal(repairReentryFromStages([stage(run, "failed")]), null);
  assert.deepEqual(repairReentryFromStages([stage(run)]), {
    kind: "run", repairWorkflowId: "repair-plan-rp-anchor-abc-0123456789ab", receiptRef: `receipt-${run.length}`,
  });
  // The newest successful re-entry wins.
  assert.deepEqual(repairReentryFromStages([stage(run), stage(batch)]), { kind: "batch", batchId: batch, receiptRef: "receipt-47" });
  assert.equal(repairReentryFromStages([stage("not-a-reentry-id")]), null);
});

test("a step summary is one short line of counts", () => {
  assert.equal(summaryLine({ kept_records: 120, dropped: 3, nested: { a: 1 } }), "kept records 120 · dropped 3");
  assert.equal(summaryLine(undefined), "");
});

// --- rendered components ------------------------------------------------------------------

test("proposals show their rationale and steps; only a proposal with steps can be used", () => {
  const markup = html(React.createElement(RepairProposalList, {
    response: { signature: "sms_backup_xml:truncated", proposals: [findProposal, waitProposal], agent_available: false, matter_mode: "LIVE" },
    onUse: noop,
  }));
  assert.match(markup, /Look in the Case Bible catalog/);
  assert.match(markup, /Wait: do not repair yet/);
  assert.match(markup, /Find another copy/);
  assert.equal(markup.match(/Use this plan/g)?.length, 1);
  assert.doesNotMatch(markup, /No known repair/);
  // Nothing is rendered for agent_available: no roadmap, no "coming soon", no placeholder.
  assert.doesNotMatch(markup, /agent|coming soon|roadmap|not yet/i);
});

test("when no proposal has steps the list says so in one line", () => {
  const onlyWait = html(React.createElement(RepairProposalList, {
    response: { signature: "xml:unassessed", proposals: [waitProposal], agent_available: false, matter_mode: "LIVE" },
    onUse: noop,
  }));
  assert.match(onlyWait, /No known repair for this file type/);
  assert.doesNotMatch(onlyWait, /Use this plan/);
  const clean = html(React.createElement(RepairProposalList, {
    response: { signature: "sms_backup_xml:clean", proposals: [], agent_available: false, matter_mode: "LIVE" },
    onUse: noop,
  }));
  assert.match(clean, /Nothing to repair: the repair check found this file clean\./);
});

test("an empty plan shows the whole tool list; a plan's steps edit their params from the schema", () => {
  const empty = html(React.createElement(RepairStepEditor, { steps: [], tools: TOOLS, toolsLoading: false, toolsError: null, disabled: false, onChange: noop }));
  assert.match(empty, /data-testid="repair-tool-list"/);
  for (const tool of TOOLS) assert.ok(empty.includes(tool.description), tool.id);
  assert.equal((empty.match(/ Add<\/button>/g) ?? []).length, TOOLS.length);

  const steps = [...draftFromProposal({ ...findProposal, steps: [{ step_id: "s1", activity: FIND.id, params: { max_candidates: 3 } }] })];
  const editor = html(React.createElement(RepairStepEditor, {
    steps: addStep(steps, SALVAGE.id), tools: TOOLS, toolsLoading: false, toolsError: null, disabled: false, onChange: noop,
  }));
  assert.match(editor, />s1</);
  assert.match(editor, />s2</);
  assert.match(editor, /type="number"[^>]*value="3"|value="3"[^>]*type="number"/);
  assert.match(editor, /placeholder="5"/);
  assert.equal(editor.match(/role="switch"/g)?.length, 2);
  assert.match(editor, /aria-label="Move s1 up"[^>]*disabled=""|disabled=""[^>]*aria-label="Move s1 up"/);
  assert.match(editor, /writes a derived copy/);
  assert.match(editor, /reads only/);
  assert.match(editor, /Add step/);
});

test("validation shows every check inline as pass or fail with the engine's reason", () => {
  const markup = html(React.createElement(RepairCheckList, {
    checks: [
      { rule: "anchor_resolves", status: "pass", reason: "Anchored to Review run abc." },
      { rule: "type_chain", status: "fail", reason: "Step s2 reads sms_backup_xml but receives pdf." },
    ],
  }));
  assert.match(markup, /data-status="pass"/);
  assert.match(markup, /data-status="fail"/);
  assert.match(markup, /Belongs to this run/);
  assert.match(markup, /Each step can read what it is given/);
  assert.match(markup, /Step s2 reads sms_backup_xml but receives pdf\./);
});

test("a run shows each step's status and receipt, then links its re-entered run", () => {
  const status = {
    workflow_id: "repair-plan-rp-anchor-abc-0123456789ab",
    plan_id: "rp-anchor-abc",
    preview_handle: HANDLE,
    matter_mode: "LIVE",
    status: "completed",
    reason: "",
    steps: [
      { step_id: "s1", activity: FIND.id, status: "succeeded", receipt_ref: "receipt-one", output_ref: "b2://bucket/other/sms-1.xml", reason: "", summary: { candidates: 2 } },
      { step_id: "s2", activity: SALVAGE.id, status: "succeeded", receipt_ref: "receipt-two", output_ref: "b2://bucket/sms-1.xml.derived/salvaged/sms-1.xml", output_sha256: "abcdef0123456789".padEnd(64, "0"), reason: "", summary: { kept_records: 120 } },
    ],
    reentry_preview_handle: "reentered_run_handle_0123456789abcdef",
    reentry_receipt_ref: "receipt-reentry",
    checks: [],
  };
  const markup = html(React.createElement(RepairRunView, { status, onOpenRun: noop }));
  assert.match(markup, /Repair run finished/);
  assert.equal(markup.match(/data-status="succeeded"/g)?.length, 2);
  assert.match(markup, /receipt receipt-one/);
  assert.match(markup, /receipt receipt-two/);
  assert.match(markup, /sha256 abcdef012345/);
  assert.match(markup, /kept records 120/);
  assert.match(markup, /Re-entered →/);
  assert.match(markup, /sms-1\.xml · run reentere/);
  assert.match(markup, /re-entry receipt receipt-reentry/);

  const failed = html(React.createElement(RepairRunView, {
    status: { ...status, status: "failed", reason: "step s2 failed", reentry_preview_handle: null, reentry_receipt_ref: null,
      steps: [{ ...status.steps[0] }, { ...status.steps[1], status: "failed", reason: "no complete record", receipt_ref: "receipt-fail" }] },
    onOpenRun: noop,
  }));
  assert.match(failed, /Repair run stopped/);
  assert.match(failed, /no complete record/);
  assert.doesNotMatch(failed, /Re-entered/);

  const batch = html(React.createElement(RepairRunView, {
    status: { ...status, reentry_preview_handle: null, reentry_batch_id: `repair-${"b".repeat(40)}` },
    onOpenRun: noop,
  }));
  assert.match(batch, new RegExp(`Re-entered →.*batch repair-${"b".repeat(40)}`));
});

test("a batch re-entry lists that batch's runs, each opening in Review once bound", () => {
  const markup = html(React.createElement(BatchRunList, {
    lead: "Repaired →",
    onOpenRun: noop,
    batch: {
      batch_id: `repair-${"c".repeat(40)}`, prefix: "x/", terminal: false, listing_truncated: false, items_truncated: false,
      counts: { total: 2, queued: 0, running: 1, waiting_on_gate: 1, done: 0, failed: 0, skipped: 0 },
      items: [
        { key: "threads/a.ndjson", source_ref: "b2://bucket/x/threads/a.ndjson", request_id: "r-a", preview_handle: "bound_batch_run_handle_0123456789abcd", status: "waiting_on_gate", reason: "" },
        { key: "threads/b.ndjson", source_ref: "b2://bucket/x/threads/b.ndjson", request_id: "r-b", preview_handle: "", status: "running", reason: "" },
      ],
      matter_mode: "LIVE",
    },
  }));
  assert.match(markup, /Repaired →/);
  assert.match(markup, /2 runs/);
  assert.match(markup, /<button[^>]*>a\.ndjson<\/button>/);
  assert.doesNotMatch(markup, /<button[^>]*>b\.ndjson<\/button>/);
  assert.match(markup, /waiting on gate/);
});

test("a repaired run reads 'Repaired → <new run>' as a link", () => {
  const markup = html(React.createElement(RepairedRunLine, { lead: "Repaired →", handle: "reentered_run_handle_0123456789abcdef", name: "sms-1.xml", onOpenRun: noop }));
  assert.match(markup, /Repaired →/);
  assert.match(markup, /<button[^>]*>sms-1\.xml · run reentere<\/button>/);
});

test("the Review catalog shows one small flag for runs hidden because their mode is unknown", () => {
  const resource = {
    preview_handle: HANDLE, request_id: "r", source_ref: "b2://bucket/sms-1.xml", created_at: "2026-09-26T00:00:00Z",
    lifecycle: "awaiting_preview_decision", terminal: false, reason: "", completed_stage_count: 6,
    representation_state: "committed_readback", representation_detail: "", content_status: "available", content_reason: "",
    record_preview_available: true, chunk_preview_available: false, open_path: "/", content_path: "/", operator_path: "/",
  };
  const props = { resources: [resource], loading: false, selectedHandle: HANDLE, onSelect: noop };
  assert.match(html(React.createElement(ReviewResourceList, { ...props, unboundCount: 2 })), /2 runs hidden: mode unknown/);
  assert.match(html(React.createElement(ReviewResourceList, { ...props, unboundCount: 1 })), /1 run hidden: mode unknown/);
  assert.doesNotMatch(html(React.createElement(ReviewResourceList, { ...props, unboundCount: 0 })), /hidden: mode unknown/);
  assert.match(html(React.createElement(ReviewResourceList, { ...props, resources: [], unboundCount: 3 })), /3 runs hidden: mode unknown/);
});

// --- wiring contract ------------------------------------------------------------------------

const client = source("../src/lib/api-client.ts");
const builder = source("../src/components/sbv/repair-builder.tsx");
const steps = source("../src/components/sbv/repair-plan-steps.tsx");
const runView = source("../src/components/sbv/repair-run-view.tsx");
const panel = source("../src/components/sbv/review-actions-panel.tsx");
const review = source("../src/components/sbv/proffer-preview-client.tsx");
const surface = source("../src/components/sbv/proffer-operator-preview.tsx");

test("every repair call goes through the BFF with the Dev/Live mode and checks its echo", () => {
  for (const route of ["tools", "propose", "validate", "run"]) {
    assert.match(client, new RegExp(`/api/proffer/repair/${route}\\?\\$\\{query\\.toString\\(\\)\\}`));
  }
  assert.match(client, /\/api\/proffer\/repair\/runs\/\$\{encodeURIComponent\(workflowId\)\}\?\$\{query\.toString\(\)\}/);
  const repairClient = client.slice(client.indexOf("class RepairRunRefusedError"), client.indexOf("export function getDecodedExists"));
  assert.equal((repairClient.match(/new URLSearchParams\(\{ mode(: plan\.matter_mode)? \}\)/g) ?? []).length, 5);
  assert.equal((repairClient.match(/matter_mode !== (mode|plan\.matter_mode)/g) ?? []).length, 5);
  assert.match(client, /response\.workflow_id !== workflowId \|\| response\.matter_mode !== mode/);
  // A refused run keeps the engine's checklist so it is shown inline.
  assert.match(client, /class RepairRunRefusedError extends ApiError/);
  assert.match(client, /error\.status === 422 && checks/);
});

test("the builder proposes, composes from the tool list, validates, runs and polls", () => {
  assert.match(builder, /proposeProfferRepairs\(mode, \{ source_ref: snapshot\.source_ref, preview_handle: snapshot\.preview_handle \}\)/);
  assert.match(builder, /if \(noStepsProposed\(response\.proposals\)\) openDraft\(\[\]\)/);
  assert.match(builder, /listProfferRepairTools\(mode, signal\)/);
  assert.match(builder, /validateProfferRepairPlan\(candidate\)/);
  assert.match(builder, /disabled=\{!plan \|\| !checked\?\.ok \|\| start\.isPending\}/);
  assert.match(builder, /getProfferRepairRun\(workflowId \?\? "", mode, signal\)/);
  assert.match(builder, /refetchInterval:/);
  assert.match(runView, /onOpenRun\(handle\)/);
  assert.match(runView, /status\.reentry_batch_id/);
  assert.match(runView, /getProfferBatch\(batchId, mode, signal\)/);
  // Controls are the shared primitives, never hand-rolled.
  for (const file of [builder, steps, runView]) assert.doesNotMatch(file, /<button|<select|<input/);
  assert.match(steps, /<Switch /);
  assert.match(steps, /<DropdownMenuItem /);
});

test("nothing renders for agent_available: no roadmap, no coming-soon, no placeholder", () => {
  for (const file of [builder, steps, runView]) {
    assert.doesNotMatch(file, /agent_available|coming soon|roadmap/i);
  }
});

test("the Actions panel's repair section hosts the builder and the repaired link, and keeps the gate", () => {
  assert.doesNotMatch(panel, /Apply the proposed repair/);
  assert.match(panel, /<RepairedLink stages=\{snapshot\.stages\} mode=\{snapshot\.matter_mode\} onOpenRun=\{onOpenRun\} \/>/);
  assert.match(panel, /<RepairBuilder snapshot=\{snapshot\} onOpenRun=\{onOpenRun\} \/>/);
  assert.match(panel, /Retain sealed original and continue/);
  assert.match(surface, /onOpenRun=\{onOpenRun\}/);
  assert.match(review, /onOpenRun=\{openStartedRun\}/);
  assert.match(review, /setUnboundCount\(response\.unbound_count \?\? 0\)/);
  assert.match(review, /unboundCount=\{unboundCount\}/);
});
