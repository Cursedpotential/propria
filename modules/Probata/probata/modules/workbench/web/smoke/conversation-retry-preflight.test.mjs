import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { runInNewContext } from "node:vm";
import test from "node:test";
import ts from "typescript";

const require = createRequire(import.meta.url);
const source = readFileSync(new URL("../src/components/conversations/conversation-actions.tsx", import.meta.url), "utf8");

/** Render the real action sheet and run its actual click callbacks with deterministic hook seams.
 * Input: durable previous attempt. Output: real component tree, state and bounded transport calls.
 * Effects: memory only; choose to verify retry/preflight UI transitions without any workflow dispatch.
 */
function actionSheet(previous) {
  const states = [null, true, previous, null, null];
  let cursor = 0;
  const starts = [];
  const checks = [];
  const preflight = { data: { eligible: false, token: "exact-token", targets: [], bounds: [], blockers: ["Mode receipt missing"], cost_estimate: "Unknown", runtime_estimate: "Unknown" }, isFetching: false, isPending: false, isError: false };
  const start = { isPending: false, error: null, reject: false,
    async mutateAsync(input) { starts.push(input); if (start.reject) { start.error = new Error("Transport rejected new attempt"); throw start.error; } start.error = null; return { workflow_id: "accepted-retry", run_id: "accepted-run" }; } };
  const ui = { Button: "button", Badge: "span", Checkbox: "input", Skeleton: "div", Sheet: "aside", SheetContent: "div", SheetDescription: "p", SheetFooter: "footer", SheetHeader: "header", SheetTitle: "h2", WorkflowSteps: "div", ExtractionsView: "div" };
  const module = { exports: {} };
  const mocks = {
    react: {
      useMemo: (factory) => factory(),
      useState(initial) { const index = cursor++; if (!(index in states)) states[index] = typeof initial === "function" ? initial() : initial; return [states[index], (value) => { states[index] = typeof value === "function" ? value(states[index]) : value; }]; },
    },
    "@/hooks/use-conversation-actions": {
      useExtractors: () => ({ data: { extractors: [{ id: "default", label: "Default", default: true }] }, isPending: false, isError: false }),
      useExtractionPreflight: (ids, extractors, enabled) => { checks.push({ ids, extractors, enabled }); return preflight; },
      useStartExtraction: () => start,
      useStartSend: () => ({ isPending: false, error: null }),
      useActionStatus: () => ({}),
    },
    "@/hooks/use-operation-notifications": { useOperationNotifications() {} },
    "@/components/mobile/mobile-format": { errorText: (error) => error.message },
    "@/lib/conversation-action-progress": {},
    "@/lib/utils": { cn: (...values) => values.filter((value) => typeof value === "string").join(" ") },
  };
  runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX } }).outputText, {
    module, exports: module.exports, require: (name) => mocks[name] ?? (name.startsWith("@/components/") ? ui : require(name)),
  });
  const render = () => { cursor = 0; return module.exports.ConversationActionSheet({ kind: "extract", threadIds: ["current-selection"], open: true, onOpenChange() {} }); };
  return { render, states, starts, checks, preflight, start };
}

/** Find actual element props in a returned React tree without reimplementing sheet visibility rules.
 * Inputs: component tree and predicate. Output: all matching elements. Effects: none;
 * choose to assert old attempt visibility alongside the real preflight and confirmation controls.
 */
function elements(tree, matches) {
  if (Array.isArray(tree)) return tree.flatMap((node) => elements(node, matches));
  if (!tree || typeof tree !== "object") return [];
  return [...(matches(tree) ? [tree] : []), ...elements(tree.props?.children, matches)];
}

const settle = () => new Promise((resolve) => setImmediate(resolve));

test("failed attempt stays visible through retry preflight and dispatch rejection, then joins the accepted retry chain", async () => {
  const older = { workflowId: "older-attempt", runId: "older-run", threadIds: ["saved-thread"], extractors: ["default"] };
  const previous = { workflowId: "failed-attempt", runId: "failed-run", threadIds: ["saved-thread"], extractors: ["default"], previous: [older] };
  const fixture = actionSheet(previous);
  let tree = fixture.render();
  const progress = elements(tree, (node) => node.props?.workflowId === "failed-attempt" && node.props.onRetry)[0];
  assert.ok(progress);
  await progress.props.onRetry(["default"]);
  tree = fixture.render();
  const visiblePrevious = () => elements(fixture.render(), (node) => node.props?.["aria-label"] === "Previous extraction attempt retained during retry");
  assert.equal(visiblePrevious().length, 1);
  assert.equal(elements(tree, (node) => node.props?.workflowId === "failed-attempt").length, 1);
  assert.deepEqual(Array.from(fixture.checks.at(-1).ids), ["saved-thread"]);

  const confirm = () => elements(fixture.render(), (node) => node.type === "footer")[0].props.children;
  assert.equal(confirm().props.disabled, true);
  confirm().props.onClick();
  await settle();
  assert.equal(fixture.starts.length, 0);
  assert.match(fixture.states[4], /exact target preflight/);
  assert.equal(visiblePrevious().length, 1);

  fixture.preflight.data.eligible = true;
  fixture.start.reject = true;
  confirm().props.onClick();
  await settle();
  assert.equal(fixture.starts.length, 1);
  assert.equal(fixture.states[2], null);
  assert.equal(visiblePrevious().length, 1);

  fixture.start.reject = false;
  confirm().props.onClick();
  await settle();
  assert.equal(fixture.states[2].workflowId, "accepted-retry");
  assert.deepEqual(Array.from(fixture.states[2].previous, (attempt) => attempt.workflowId), ["older-attempt", "failed-attempt"]);
  assert.equal(fixture.states[3], null);
  assert.equal(visiblePrevious().length, 0);
  assert.equal(elements(fixture.render(), (node) => node.props?.workflowId === "failed-attempt").length, 1);
});
