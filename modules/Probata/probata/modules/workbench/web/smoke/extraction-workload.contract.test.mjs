import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import ts from "typescript";

// Load the actual pure presentation unit without APIs or a desktop browser.
const source = readFileSync(new URL("../src/components/conversations/conversation-actions.tsx", import.meta.url), "utf8");
const ast = ts.createSourceFile("actions.tsx", source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
let unit;
function visit(node) {
  if (ts.isFunctionDeclaration(node) && node.name?.text === "preflightWorkloadText") unit = node.getText(ast);
  ts.forEachChild(node, visit);
}
visit(ast);
assert.ok(unit);
const context = {};
runInNewContext(ts.transpileModule(unit + "\nglobalThis.describe = preflightWorkloadText;", { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, context);

test("unqueried or legacy unknown workload never renders numeric zero", () => {
  for (const workload_status of ["not_queried", undefined]) {
    const text = context.describe({ workload_status, messages: 0, characters: 0, utf8_bytes: 0 });
    assert.match(text, /Workload unknown.*not queried/);
    assert.doesNotMatch(text, /0 messages|0 characters|0 UTF-8/);
  }
});

test("partial workload labels observed sizes and unknown remainder; queried zero stays zero", () => {
  const target = { messages: 4, characters: 42, utf8_bytes: 45, max_message_characters: 20, max_message_utf8_bytes: 23, queried_generations: 1, resolved_generations: 2 };
  const partial = context.describe({ ...target, workload_status: "partial" });
  assert.match(partial, /Partial workload: observed 4 messages/);
  assert.match(partial, /Queried 1 of 2 resolved generations; remaining workload is unknown/);
  const available = context.describe({ ...target, messages: 0, workload_status: "available" });
  assert.match(available, /^0 messages/);
  assert.doesNotMatch(available, /unknown|Partial/);
});
