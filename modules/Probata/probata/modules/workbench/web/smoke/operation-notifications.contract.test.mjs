// Byline: Codex · GPT-6 · 2026-10-12.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

/** Run actual notification hook effects against durable operation observations.
 * Inputs: real hook module and controlled in-app Toaster. Output: state function and hook.
 * Effects: recorded notifications only. Pick for standalone/composed failure and recovery checks.
 */
function notifications() {
  const events = [];
  const current = { events: new Set(), failed: false };
  const module = { exports: {} };
  const source = readFileSync(new URL("../src/hooks/use-operation-notifications.ts", import.meta.url), "utf8");
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  runInNewContext(code, { module, exports: module.exports, require: (id) => id === "react" ? { useEffect: (effect) => effect(), useRef: () => ({ current }) } : { toast: Object.fromEntries(["error", "warning", "success"].map((kind) => [kind, (message, options) => events.push({ kind, message, options })])) } });
  return { ...module.exports, events };
}

test("standalone failure and retry recovery notify once per real attempt without inventing workflow success", () => {
  const { useOperationNotifications: observe, operationState, events } = notifications();
  for (const status of ["pending", "running", "retrying", "COMPLETED"]) observe("standalone-workflow", "run-one", status, "Standalone extraction");
  assert.equal(events.length, 0, "Temporal COMPLETED is not a business success value");
  observe("standalone-workflow", "run-one", "failed", "Standalone extraction", "Actual Activity failure");
  observe("standalone-workflow", "run-one", "failed", "Standalone extraction", "Actual Activity failure");
  assert.equal(events.length, 1);
  assert.equal(events[0].kind, "error");
  assert.match(events[0].options.description, /Actual Activity failure.*standalone-workflow.*run-one/);
  observe("retry-workflow", "run-two", "running", "Standalone extraction");
  observe("retry-workflow", "run-two", "complete", "Standalone extraction");
  assert.equal(events[1].kind, "success");
  assert.equal(events[1].message, "Standalone extraction recovered");
  assert.equal(operationState("accepted"), "pending");
  assert.equal(operationState("retrying"), "retrying");
  assert.equal(operationState("enriching"), "running");
});

test("composed partial failure remains partial and preserves failure detail until actual recovery", () => {
  const { useOperationNotifications: observe, operationState, events } = notifications();
  observe("parent-workflow", "parent-run", "partial", "Composed extraction", "Child two failed; child one output retained");
  observe("parent-workflow", "parent-run", "partial", "Composed extraction", "Child two failed; child one output retained");
  assert.equal(events.length, 1);
  assert.equal(events[0].kind, "warning");
  assert.match(events[0].options.description, /Child two failed; child one output retained/);
  assert.equal(operationState("partial_enrichment"), "partial");
  observe("targeted-child-retry", "retry-run", "failed", "Composed extraction", "Still failed");
  assert.equal(events[1].kind, "error");
  observe("targeted-child-retry-two", "retry-run-two", "completed", "Composed extraction");
  assert.equal(events[2].message, "Composed extraction recovered");
});
