import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

/** Load a pure TypeScript helper for focused runtime assertions.
 * Inputs: source path and explicit import replacements. Outputs: module exports.
 * Effects: source reads only; choose when a helper has no browser dependencies.
 */
async function load(path, replacements = {}) {
  let source = readFileSync(new URL(path, import.meta.url), "utf8");
  for (const [before, after] of Object.entries(replacements)) source = source.replace(before, after);
  const output = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText;
  return import(`data:text/javascript;base64,${Buffer.from(output).toString("base64")}`);
}
const human = await load("../src/lib/human-context.ts");
const humanURL = `data:text/javascript;base64,${Buffer.from(ts.transpileModule(readFileSync(new URL("../src/lib/human-context.ts", import.meta.url), "utf8"), { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText).toString("base64")}`;
const { nativeSearchTargets } = await load("../src/lib/search-read-target.ts", { '"@/lib/human-context"': JSON.stringify(humanURL) });
const version = "11111111-1111-4111-8111-111111111111";

test("native origin alone never invents a human reader", () => {
  assert.deepEqual(nativeSearchTargets({ origin_system: "probata-native-context", source_version_id: version }), { native: true, handles: [] });
});
test("authoritative human corpus and validated explicit handle retain the native reader", () => {
  assert.deepEqual(nativeSearchTargets({ origin_system: "probata-native-context", corpus: "native_human_context", source_version_id: version }).handles, [human.humanPreviewHandle(version)]);
  assert.deepEqual(nativeSearchTargets({ preview_handle: human.humanPreviewHandle(version), source_version_id: version }).handles, [human.humanPreviewHandle(version)]);
});
test("known AI refuses a conflicting human handle without guessing a different workflow", () => {
  for (const discriminator of [{ source_kind: "ai_chat" }, { record_kind: "ai_chat" }, { source_format: "chatgpt_official_json" }, { source_format: "claude_markdown" }]) {
    assert.deepEqual(nativeSearchTargets({ origin_system: "probata-native-context", corpus: "native_human_context", preview_handle: human.humanPreviewHandle(version), source_version_id: version, ...discriminator }), { native: true, handles: [] });
  }
});
