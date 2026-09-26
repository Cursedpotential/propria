// Byline: Claude Code · Opus 5.5 · 2026-09-25 (Review Actions panel contract)
// Owner 2026-09-25 00:16: "there's no option to do any of it. Can't modify any metadata. I can't
// add any context, I can't choose a parser, I can't choose a repair. I can't do anything."
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

function source(path) {
  return readFileSync(new URL(path, import.meta.url), "utf8");
}

const surface = source("../src/components/sbv/proffer-operator-preview.tsx");
const review = source("../src/components/sbv/proffer-preview-client.tsx");
const panel = source("../src/components/sbv/review-actions-panel.tsx");
const contextSection = source("../src/components/sbv/review-context-section.tsx");
const rerunHook = source("../src/hooks/use-review-rerun.ts");
const parserPanel = source("../src/components/intake/parser-selection-panel.tsx");
const assertions = source("../src/lib/source-assertions.ts");
const intake = source("../src/components/intake/unified-intake.tsx");
const client = source("../src/lib/api-client.ts");
const dropdown = source("../src/components/ui/dropdown-menu.tsx");

test("the Actions panel renders for every selected run, not only at a paused gate", () => {
  assert.match(surface, /<ReviewActionsPanel/);
  assert.doesNotMatch(surface, /\{[^}]*&&\s*<ReviewActionsPanel/);
  for (const section of ["review-actions-context", "review-actions-parser", "review-actions-repair", "review-actions-process"]) {
    assert.match(`${panel}\n${contextSection}`, new RegExp(`data-testid="${section}"`));
  }
  assert.match(panel, /Re-run this source/);
});

test("context is read back per run and saved as an append-only supersession", () => {
  assert.match(client, /\/source-context\?/);
  assert.match(client, /run source context crossed its preview or TEST\/REAL boundary/);
  assert.match(contextSection, /supersedes_ref: current\?\.source_context_ref \?\? null/);
  assert.match(contextSection, /observed_source: observation/);
  assert.match(contextSection, /Saved as revision \{saved\.revision\}/);
  assert.match(contextSection, /Re-run with this context/);
  assert.match(intake, /<SourceAssertionsFields value=\{assertions\} onChange=\{updateAssertion\} \/>/);
  assert.match(contextSection, /<SourceAssertionsFields/);
  assert.match(assertions, /observationFromRegistration/);
});

test("parser and repair choices answer the gate when paused, else seed a fresh run", () => {
  assert.match(panel, /onRecordDecision=\{\(\) => selectedCandidate && onSelectHandler\(selectedCandidate\)\}/);
  assert.match(parserPanel, /Re-run with this parser/);
  assert.match(parserPanel, /Re-run and choose the parser/);
  assert.match(panel, /Retain sealed original and continue/);
  assert.match(panel, /Re-run with this choice/);
  assert.match(panel, /Repair check failed: \{failure\.text\}/);
  assert.match(rerunHook, /OPERATOR_HANDLER_SELECTION = "operator-handler-selection\/v1"/);
  assert.match(rerunHook, /decideProfferHandler\(previewHandle, mode/);
  assert.match(rerunHook, /decideProfferRepair\(previewHandle, mode, \{ approved: true, apply_repair: false \}\)/);
  assert.match(rerunHook, /startProffer\(\{/);
  assert.match(rerunHook, /request_id: requestId/);
  // AMENDED 2026-09-26 (Claude Code · Opus 5.5): the disabled "Apply the proposed repair" is
  // replaced by the repair builder (smoke/repair-builder.contract.test.mjs). It runs a separate
  // plan through /api/proffer/repair/*; this run's gate is never answered with apply_repair.
  assert.doesNotMatch(panel, /Apply the proposed repair/);
  assert.match(panel, /<RepairBuilder snapshot=\{snapshot\} onOpenRun=\{onOpenRun\} \/>/);
  assert.doesNotMatch(`${panel}\n${rerunHook}`, /apply_repair: true/);
});

test("the Test / Live switch is shown once; Review carries one small flag", () => {
  assert.doesNotMatch(review, /MatterModeSelector/);
  assert.doesNotMatch(review, /resource selected/);
  assert.doesNotMatch(surface, /operation destination/);
  assert.match(surface, /data-testid="review-mode-flag"/);
  assert.match(review, /APPROVAL_LOCK_REASON/);
});

test("tab markers come from real row counts and More is a portal dropdown", () => {
  assert.doesNotMatch(surface, /tabAvailability/);
  assert.match(surface, /function TabDot\(\{ stat \}: \{ stat: TabStat \}\)/);
  assert.match(surface, /if \(!stat\.count\) return null;/);
  assert.match(surface, /<DropdownMenuRadioItem/);
  assert.match(surface, /\(\{countLabel\(stats\[id\]\) \|\| "0"\}\)/);
  assert.match(dropdown, /DropdownMenuPrimitive\.Portal/);
});

test("a derive-only run reads its decoded conversation files on the Messages tab", () => {
  assert.match(surface, /snapshot\.parser_handler === "smsthreads_derive" \|\| snapshot\.parser_execution_path === "derive"/);
  assert.match(surface, /<DecodedSourceViewer sourceRef=\{snapshot\.source_ref\} \/>/);
});
