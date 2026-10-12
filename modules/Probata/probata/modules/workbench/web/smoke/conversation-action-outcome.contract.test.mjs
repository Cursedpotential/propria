// Byline: Codex · GPT-6 · 2026-10-12.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import test from "node:test";
import { runInNewContext } from "node:vm";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";

const require = createRequire(import.meta.url);
const cn = (...values) => values.filter(Boolean).join(" ");

/** Compile actual web modules with explicit transport/hooks for outcome regressions.
 * Inputs: source path and dependencies. Output: real exports. Effects: in-memory reads only.
 * Pick for business outcome and rendered action checks without live workflows or a desktop browser.
 */
function load(path, dependencies = {}, suffix = "") {
  const module = { exports: {} };
  const source = readFileSync(new URL(path, import.meta.url), "utf8").replaceAll("import.meta.env.VITE_API_URL", '""');
  const js = ts.transpileModule(source + suffix, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(js, { module, exports: module.exports, require: (name) => dependencies[name] ?? require(name), DOMException, ...dependencies.globals });
  return module.exports;
}
const { conversationActionProgress } = load("../src/lib/conversation-action-progress.ts");
const requested = ["fixture-one", "fixture-two", "fixture-three"];
const resolution = { step: "resolve", status: "completed", detail: "1 conversation(s), 1 run(s), 0 without runs", counts: { conversations: 1, runs: 1, skipped: 0 } };
const stage = (extractor, status, detail = "") => ({ step: `conversation|${extractor}`, conversation: "Actual conversation", extractor, status, detail });
const status = (outcome, steps) => ({ workflow_id: "actual-workflow", kind: "extraction", outcome, steps, receipts: [] });

test("HTTP 200 and Temporal COMPLETED preserve a failed business outcome and its exact stage errors", async () => {
  const payload = { ...status("failed", [resolution, ...requested.map((id) => stage(id, "failed", "missing verified operating_mode"))]), execution_status: "COMPLETED" };
  const { conversationApi } = load("../src/lib/conversation-actions-client.ts", {
    "@/lib/api-client": { ApiError: Error }, "@/lib/entity-extraction-client": { newIdempotencyKey: () => "unused" },
    globals: { fetch: async () => ({ ok: true, json: async () => payload }) },
  });
  const read = await conversationApi.status(payload.workflow_id);
  assert.equal(read.outcome, "failed");
  const result = conversationActionProgress(read, [], requested);
  assert.equal(result.progress.outcome, "failed");
  assert.equal(result.progress.steps.filter((step) => step.detail === "missing verified operating_mode").length, 3);
  assert.match(result.progress.steps[0].step, /preparation/);
});

test("resolution alone, all failed stages, all skipped stages and missing requested stages cannot complete extraction", () => {
  for (const steps of [[resolution], [resolution, ...requested.map((id) => stage(id, "failed"))], [resolution, ...requested.map((id) => stage(id, "skipped"))], [resolution, stage("unrequested", "completed")]]) {
    assert.equal(conversationActionProgress(status("completed", steps), [], requested).progress.outcome, "failed");
  }
});

test("partial requires actual requested successes, and completed requires every reported requested stage to succeed", () => {
  for (const nonSuccess of ["failed", "skipped", "pending", "running"]) {
    const result = conversationActionProgress(status("completed", [resolution, stage(requested[0], "completed"), stage(requested[1], nonSuccess), stage(requested[2], "completed")]), [], requested);
    assert.equal(result.progress.outcome, "partial");
  }
  assert.equal(conversationActionProgress(status("completed", [resolution, stage(requested[0], "completed")]), [], requested).progress.outcome, "partial", "missing chosen extractors cannot be treated as successful");
  const all = [resolution, ...requested.map((id) => ({ ...stage(id, "completed"), counts: { entities: 0, events: 0 } }))];
  assert.equal(conversationActionProgress(status("completed", all), [], requested).progress.outcome, "completed", "successful stages may legitimately find zero entities");
  assert.equal(conversationActionProgress(status("completed_with_flags", all), [], requested).progress.outcome, "completed_with_flags");
  assert.equal(conversationActionProgress(status("failed", all), [], requested).progress.outcome, "failed", "explicit business failure has priority");
  assert.equal(conversationActionProgress(status("running", [resolution]), [], requested).progress.outcome, "running", "pending children must not be prematurely failed");
  assert.equal(conversationActionProgress({ ...status("completed", [resolution]), kind: "send_to_surreal" }, [], requested).progress.outcome, "completed", "send business semantics are unchanged");
});

test("a failed extraction renders a failed headline, preserved errors and a working retry action beside successful resolution", async () => {
  const { WorkflowSteps } = load("../src/components/entities/workflow-steps.tsx", { "@/lib/utils": { cn } });
  const buttons = [];
  let failStatusRead = false;
  let readsRetried = 0;
  const Empty = ({ children }) => React.createElement("div", null, children);
  const { ActionProgress } = load("../src/components/conversations/conversation-actions.tsx", {
    "@/components/entities/workflow-steps": { WorkflowSteps },
    "@/components/conversations/extractions-view": { ExtractionsView: Empty },
    "@/components/mobile/mobile-format": { errorText: (error) => error.message },
    "@/components/ui/button": { Button: ({ children, variant, ...props }) => { buttons.push({ children, ...props }); return React.createElement("button", props, children); } },
    "@/components/ui/badge": { Badge: Empty }, "@/components/ui/checkbox": { Checkbox: Empty }, "@/components/ui/skeleton": { Skeleton: Empty },
    "@/components/ui/sheet": { Sheet: Empty, SheetContent: Empty, SheetDescription: Empty, SheetFooter: Empty, SheetHeader: Empty, SheetTitle: Empty },
    "@/hooks/use-conversation-actions": { useActionStatus: () => ({ data: status("completed", [resolution, ...requested.map((id) => stage(id, "failed", `${id}: missing verified operating_mode`))]), isError: failStatusRead, error: new Error("Status transport unavailable"), refetch: () => { readsRetried++; } }), useExtractors: () => ({ data: { extractors: [] } }) },
    "@/lib/conversation-action-progress": { conversationActionProgress }, "@/lib/utils": { cn },
  }, "\nexport { ActionProgress };\n");
  let retried = 0;
  const html = renderToStaticMarkup(React.createElement(ActionProgress, { workflowId: "actual-workflow", threadIds: ["actual-thread"], title: "Extraction", requested, retrying: false, onRetry: async () => { retried++; } }));
  assert.match(html, /data-outcome="failed"/);
  assert.match(html, /No requested extractor completed successfully/);
  assert.match(html, /Resolve source conversations \(preparation\)/);
  for (const id of requested) assert.ok(html.includes(`${id}: missing verified operating_mode`));
  const retry = buttons.find((button) => button.children === "Retry extraction");
  assert.ok(retry);
  await retry.onClick();
  assert.equal(retried, 1);
  failStatusRead = true;
  const retained = renderToStaticMarkup(React.createElement(ActionProgress, { workflowId: "actual-workflow", threadIds: ["actual-thread"], title: "Extraction", requested, retrying: false, onRetry: async () => {} }));
  assert.match(retained, /Status transport unavailable/);
  assert.match(retained, /data-outcome="failed"/);
  assert.match(retained, /fixture-one: missing verified operating_mode/, "a background status read failure must not erase known extraction errors");
  buttons.find((button) => button.children === "Retry status read").onClick();
  assert.equal(readsRetried, 1);
});

test("retry starts the saved conversations and chosen extractors after list selection was cleared", async () => {
  const text = readFileSync(new URL("../src/components/conversations/conversation-actions.tsx", import.meta.url), "utf8");
  const ast = ts.createSourceFile("actions.tsx", text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  let run;
  function visit(node) { if (ts.isFunctionDeclaration(node) && node.name?.text === "run") run = node.getText(ast); ts.forEachChild(node, visit); }
  visit(ast);
  assert.ok(run);
  const calls = [];
  let saved;
  const context = { kind: "extract", threadIds: [], checked: new Set(["different-current-choice"]), startExtraction: { mutateAsync: async (input) => { calls.push(input); return { workflow_id: "new-workflow" }; } }, setStarted: (value) => { saved = value; }, onStarted: undefined };
  const code = ts.transpileModule(`${run}\nglobalThis.retry = run`, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(code, context);
  await context.retry({ workflowId: "failed-workflow", threadIds: ["original-thread"], extractors: requested });
  assert.deepEqual(JSON.parse(JSON.stringify(calls[0])), { threadIds: ["original-thread"], extractors: requested });
  assert.equal(saved.workflowId, "new-workflow");
  assert.deepEqual(saved.threadIds, ["original-thread"]);
  saved = undefined;
  context.startExtraction.mutateAsync = async () => { throw new Error("Retry starter unavailable"); };
  await assert.rejects(context.retry({ workflowId: "failed-workflow", threadIds: ["original-thread"], extractors: requested }), /Retry starter unavailable/);
  assert.equal(saved, undefined, "failed retry must not replace the tracked failed workflow");
});
