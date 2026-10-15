import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const checkpointSource = readFileSync(new URL("../src/lib/proffer-context-checkpoints.ts", import.meta.url), "utf8");
const compiled = ts.transpileModule(checkpointSource, { compilerOptions: { module: ts.ModuleKind.ESNext } }).outputText;
const { profferCheckpointStatuses } = await import(`data:text/javascript;base64,${Buffer.from(compiled).toString("base64")}`);

test("starting without lifecycle evidence does not invent a running checkpoint", () => {
  assert.deepEqual(Object.values(profferCheckpointStatuses({ started: true })), Array(6).fill("waiting"));
});

test("an explicit live checkpoint can run and a terminal lifecycle clears stale running state", () => {
  const checkpoints = [{ checkpoint: "parser_execution", status: "running" }];
  assert.equal(profferCheckpointStatuses({ started: true, lifecycle: "running", checkpoints }).parser_execution, "running");
  for (const lifecycle of ["failed", "cancelled", "completed", "rerun_required", "unavailable"]) {
    const statuses = profferCheckpointStatuses({ started: true, phase: "awaiting_handler_selection", lifecycle, checkpoints });
    assert.ok(!Object.values(statuses).includes("running"));
  }
});

test("upstream completion survives downstream failure and unavailable projection counts stay unknown", () => {
  const statuses = profferCheckpointStatuses({ started: true, lifecycle: "failed", receipts: [{ receipt_type: "storage", status: "completed" }] });
  assert.equal(statuses.storage, "completed");
  assert.equal(statuses.raw_source_verification, "waiting", "a downstream failure is not evidence that source verification failed");
  const surface = readFileSync(new URL("../src/components/sbv/proffer-operator-preview.tsx", import.meta.url), "utf8");
  for (const tab of ["entities", "relationships", "graph", "lineage"]) assert.match(surface, new RegExp(`${tab}: \\{ count: null \\}`));
  assert.doesNotMatch(surface, /content\?\.(?:records|attachments)\.length \?\? 0/);
});
