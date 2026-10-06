// Byline: Codex · GPT-6 · 2026-10-06 (Activity durable Proffer contract)
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const activityPage = readFileSync(new URL("../src/app/activity/page.tsx", import.meta.url), "utf8");
const activity = readFileSync(new URL("../src/components/activity/activity-page.tsx", import.meta.url), "utf8");
const ledger = readFileSync(new URL("../src/components/activity/activity-operations-ledger.tsx", import.meta.url), "utf8");
const batch = readFileSync(new URL("../src/components/activity/batch-activity-panel.tsx", import.meta.url), "utf8");
const cancel = readFileSync(new URL("../src/components/activity/operation-cancel-control.tsx", import.meta.url), "utf8");
const stateSource = readFileSync(new URL("../src/components/activity/proffer-operations-state.ts", import.meta.url), "utf8");
const client = readFileSync(new URL("../src/lib/api-client.ts", import.meta.url), "utf8");
const types = readFileSync(new URL("../src/lib/shared/types.ts", import.meta.url), "utf8");
const compatibilityTable = readFileSync(new URL("../src/components/intake/proffer-operations-table.tsx", import.meta.url), "utf8");

const stateModule = await import(`data:text/javascript;base64,${Buffer.from(ts.transpileModule(stateSource, {
  compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
}).outputText).toString("base64")}`);

test("Activity presents the durable Proffer ledger rather than legacy Workbench runs", () => {
  assert.match(activityPage, /<ActivityPage \/>/);
  assert.match(activity, /<ActivityOperationsLedger \/>/);
  assert.match(compatibilityTable, /<ActivityOperationsLedger \/>/);
  assert.doesNotMatch(ledger, /listRuns|\/api\/runs/);
});

test("the ledger uses the engine-backed operation list and detail projections", () => {
  assert.match(client, /apiFetch<ProfferOperationListResponse>\(`\/api\/proffer\/operations\$\{suffix\}`/);
  assert.match(client, /apiFetch<ProfferOperationDetail>/);
  assert.match(ledger, /listProfferOperations\(/);
  assert.match(ledger, /getProfferOperation\(selectedHandle/);
  for (const lifecycle of ["running", "awaiting_repair_decision", "awaiting_preview_decision", "completed", "failed", "cancelled", "unavailable"]) {
    assert.match(types, new RegExp(`\\| "${lifecycle}"|= "${lifecycle}"`));
  }
});

test("refresh merges the latest first page without dropping older loaded rows", () => {
  const makeOperation = (preview_handle, reason) => ({ preview_handle, request_id: `request-${preview_handle}`, reason });
  const older = makeOperation("older", "unchanged");
  const first = makeOperation("first", "before refresh");
  const updatedFirst = makeOperation("first", "after refresh");
  const newest = makeOperation("newest", "new first page row");
  const merged = stateModule.mergeOperationRows([first, older], [newest, updatedFirst]);

  assert.deepEqual(merged, [newest, updatedFirst, older]);
  assert.equal(merged.find((row) => row.preview_handle === "first").reason, "after refresh");
  const withOlderPage = stateModule.mergeOperationRows([makeOperation("oldest", "older cursor page")], merged);
  assert.deepEqual(withOlderPage, [newest, updatedFirst, older, makeOperation("oldest", "older cursor page")]);
});

test("filters and selected operation stay in the Activity URL", () => {
  assert.match(ledger, /searchParams\.get\("status"\)/);
  assert.match(ledger, /searchParams\.get\("operation_status"\)/);
  assert.match(ledger, /searchParams\.get\("q"\)/);
  assert.match(ledger, /searchParams\.get\("operation_source"\)/);
  assert.match(ledger, /searchParams\.get\("preview_handle"\)/);
  assert.match(ledger, /searchParams\.get\("operation"\)/);
  assert.match(ledger, /operation_status: null/);
  assert.match(ledger, /operation_source: null/);
  assert.match(ledger, /navigate\.replace\(`\/activity\$\{query/);
  assert.match(activity, /searchParams\.get\("batch"\)/);
  assert.match(activity, /navigate\.replace\(`\/activity\$\{query/);
});

test("folder Activity reads durable server status and links each item to its exact attempt", () => {
  assert.match(batch, /getProfferBatch\(batchId, mode, signal\)/);
  assert.match(client, /`\/api\/proffer\/batches\/\$\{encodeURIComponent\(batchId\)\}/);
  assert.match(batch, /batch\.data\.counts/);
  assert.match(batch, /item\.preview_handle/);
  assert.match(batch, /listing_truncated \|\| batch\.data\.items_truncated/);
  assert.doesNotMatch(batch, /localStorage|sessionStorage|Math\.round\([^)]*\/[^)]*\)/);
});

test("cancellation appears only after the exact mode-scoped operator snapshot is confirmed", () => {
  assert.match(cancel, /getProfferOperatorSnapshot\(previewHandle, mode, signal\)/);
  assert.match(cancel, /exactSnapshot\.preview_handle !== previewHandle/);
  assert.match(cancel, /exactSnapshot\.matter_mode !== mode/);
  assert.match(cancel, /<CancelRunSection snapshot=\{exactSnapshot\} \/>/);
  assert.match(client, /\/previews\/\$\{encodeURIComponent\(previewHandle\)\}\/cancel/);
});

test("source labels and attempt links use filenames and canonical mode-scoped Read", () => {
  assert.equal(stateModule.sourceFilename("r2://vault/folder/Family%20messages.json"), "Family messages.json");
  assert.equal(stateModule.attemptReviewHref("attempt_012345678901234567890123456789", "DEV"), "/read?resource=attempt_012345678901234567890123456789&mode=DEV");
  assert.equal(stateModule.nextOperationAction("failed"), "Start a new import");
  assert.match(ledger, /Technical details/);
  assert.match(ledger, /OperationCancelControl previewHandle=\{detail\.preview_handle\} mode=\{mode\}/);
  assert.match(activity, /Dev mode/);
  assert.match(activity, /Live mode/);
  assert.doesNotMatch(activity, /Dev case|Live case/);
});

test("invalid selected handles exit loading and older-page requests abort on unmount", () => {
  assert.match(ledger, /loading=\{validSelectedHandle && detail\.isPending\}/);
  assert.match(ledger, /sourceContextLoading=\{validSelectedHandle && sourceContext\.isPending\}/);
  assert.match(ledger, /loadMoreAbortRef\.current\?\.abort\(\)/);
  assert.match(ledger, /\}, controller\.signal\)/);
  assert.match(ledger, /if \(controller\.signal\.aborted\) return/);
  assert.match(ledger, /mergeOperationRows\(response\.items, current\)/);
});

test("new exported Activity units carry bylines and interface documentation", () => {
  for (const source of [activityPage, activity, ledger, batch, cancel, stateSource, compatibilityTable]) {
    assert.match(source, /Byline: Codex · GPT-6 · 2026-10-06/);
    assert.match(source, /Inputs:/);
    assert.match(source, /Output:/);
    assert.match(source, /Side effects:/);
  }
});
