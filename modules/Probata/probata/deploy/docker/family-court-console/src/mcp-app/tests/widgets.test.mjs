// Byline: OpenAI Codex / GPT-5.6, 2026-08-13
// Updated by: OpenAI Codex · GPT-5 · 2026-09-12 — host-theme and connection contract coverage.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import test from "node:test";

const required = {
  dashboard: ["release", "missingModules", "attorneyReviewRequired", "sourceSummary", "typeof x==='string'"],
  route: ["status", "flags", "nextActions", "warning", "STOP_AND_VERIFY"],
  deadline: ["anchorDate", "rawDate", "adjustedDate", "adjustment", "direction", "warning"],
  packet: ["releaseStatus", "sections", "stopConditions", "note", "aria-valuenow"],
  checklist: ["kind", "items", "warning", "aria-live"],
  sources: ["releaseStatus", "authority", "supports", "checkedOn", "verified_primary"],
  search: ["releaseStatus", "authority", "status", "supports", "checkedOn"],
  chronology: ["knowledgeDate", "Knowledge date not supplied", "warning", "dateTime"],
};

for (const [name, tokens] of Object.entries(required)) {
  test(`${name} widget carries its tool contract and safety states`, () => {
    const html = readFileSync(resolve("widgets", `${name}.html`), "utf8");
    assert.equal((html.match(/\/\*__EXT_APPS_BUNDLE__\*\//g) ?? []).length, 1);
    assert.equal(html.includes("innerHTML"), false);
    assert.match(html, /app\.connect\(\).*catch/s);
    assert.match(html, /aria-live/);
    for (const token of tokens) assert.ok(html.includes(token), `${name} missing ${token}`);
  });
}

test("generated widgets follow the MCP host theme and shared Workbench palette", () => {
  const generated = readFileSync(resolve("src", "generated-widgets.ts"), "utf8");
  assert.match(generated, /applyWorkbenchHostContext/);
  assert.match(generated, /onhostcontextchanged/);
  assert.match(generated, /getHostContext/);
  assert.match(generated, /applyDocumentTheme/);
  assert.match(generated, /applyHostStyleVariables/);
  assert.match(generated, /await app\.connect\(\);\\n/);
  assert.equal(generated.includes("await connectWorkbenchApp(app);\\n  applyWorkbenchHostContext"), false);
  assert.match(generated, /data-theme=\\\"dark\\\"/);
  assert.match(generated, /#1d252c/);
  assert.match(generated, /#242e36/);
  assert.equal(generated.includes("prefers-color-scheme:dark"), false);
  assert.equal(generated.includes("CanvasText"), false);
});
