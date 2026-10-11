// Byline: Codex · GPT-6 · 2026-10-07 (source-pinned tool surface replaces absent monitor API)
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const intake = readFileSync(new URL("../src/components/intake/unified-intake.tsx", import.meta.url), "utf8");
const action = readFileSync(new URL("../src/components/tools/source-pinned-action.tsx", import.meta.url), "utf8");
const explorer = readFileSync(new URL("../src/components/tools/tool-explorer.tsx", import.meta.url), "utf8");
const toolsPage = readFileSync(new URL("../src/app/tools/page.tsx", import.meta.url), "utf8");
const client = readFileSync(new URL("../src/lib/api-client.ts", import.meta.url), "utf8");

test("Atomic Tools is a tab inside the unified intake window", () => {
  assert.match(intake, /type OperatorTab = "intake" \| "atomic_tools"/);
  assert.match(intake, />Atomic Tools<\/button>/);
  assert.match(intake, /<SourcePinnedAction/);
  assert.match(intake, /initialSourceRef=\{selectedSourceRef/);
  assert.match(intake, /initialSHA256=\{inspection\?\.sha256/);
  assert.match(intake, /<ToolExplorer/);
});

test("operator pages use the implemented source-pinned BFF, not the absent monitor API", () => {
  assert.doesNotMatch(client, /\/api\/tools\/call/);
  assert.match(client, /\/api\/atomic-tool-actions/);
  assert.match(action, /startSourcePinnedToolAction/);
  assert.match(action, /getSourcePinnedToolStatus/);
  assert.match(toolsPage, /<SourcePinnedAction/);
  assert.match(toolsPage, /<ToolExplorer/);
  assert.doesNotMatch(`${intake}\n${toolsPage}`, /<AtomicTools/);
});

test("the approved action requires an immutable source and digest", () => {
  for (const phrase of [
    "Immutable source locator",
    "Verified SHA-256",
    "Run read-only tool",
    "workflow_id",
    "result_ref",
    "audit_chain_head",
  ]) assert.match(action, new RegExp(phrase));
  assert.doesNotMatch(explorer, /<ToolForm/);
  assert.match(explorer, /Run an approved source action/);
});
